SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.peticion_idempotente') IS NULL
 OR to_regclass('documento.peticion_spaf') IS NOT NULL
 OR EXISTS(SELECT 1 FROM documento.tipo_documento WHERE codigo='spaf') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

INSERT INTO documento.tipo_documento(codigo,requiere_registro) VALUES('spaf',false);
ALTER TABLE documento.emision ALTER COLUMN expediente_id DROP NOT NULL;
ALTER TABLE documento.emision ADD CONSTRAINT emision_alcance_spaf CHECK (
 (tipo_documento='spaf' AND expediente_id IS NULL AND equipo_id IS NOT NULL
  AND fecha_documento=(date_trunc('month',fecha_documento)::date+interval '1 month - 1 day')::date)
 OR (tipo_documento<>'spaf' AND expediente_id IS NOT NULL));

-- La emisión mensual sólo depende del equipo de su alcance. El control de
-- pertenencia de los documentos de expediente conserva su comportamiento.
CREATE OR REPLACE FUNCTION documento.comprobar_pertenencia() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,documento AS $$
BEGIN
 IF NEW.tipo_documento='spaf' THEN
  IF NEW.expediente_id IS NOT NULL OR NEW.equipo_id IS NULL THEN
   RAISE EXCEPTION 'documento.acceso_denegado' USING ERRCODE='42501';
  END IF;
 ELSIF NOT EXISTS(SELECT 1 FROM expediente.expediente e
  WHERE e.id=NEW.expediente_id AND e.equipo_id IS NOT DISTINCT FROM NEW.equipo_id) THEN
  RAISE EXCEPTION 'documento.acceso_denegado' USING ERRCODE='42501';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION documento.comprobar_pertenencia() FROM PUBLIC,etf_app,etf_copia;

-- Políticas adicionales exclusivamente para SPAF; las de entrada no se amplían.
CREATE POLICY emision_spaf_lectura ON documento.emision FOR SELECT TO etf_app USING (
 tipo_documento='spaf' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])));
CREATE POLICY emision_spaf_escritura ON documento.emision FOR INSERT TO etf_app WITH CHECK (
 tipo_documento='spaf' AND generada_por::text=nullif(current_setting('app.usuario_id',true),'')
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])));
CREATE POLICY plantilla_spaf_lectura ON documento.plantilla FOR SELECT TO etf_app USING (
 tipo_documento='spaf' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE POLICY referencia_spaf_lectura ON documento.referencia_activa FOR SELECT TO etf_app USING (
 tipo_documento='spaf' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE INDEX emision_spaf_mes ON documento.emision(equipo_id,fecha_documento,idioma,generada_en DESC,id DESC) WHERE tipo_documento='spaf';

CREATE TABLE documento.peticion_spaf (
 usuario_id uuid NOT NULL REFERENCES identidad.usuario(id),
 clave text NOT NULL CHECK(clave ~ '^[A-Za-z0-9_-]{1,128}$'),
 huella_peticion bytea NOT NULL CHECK(octet_length(huella_peticion)=32),
 emision_id uuid NOT NULL REFERENCES documento.emision(id),
 creada_en timestamptz NOT NULL CHECK(isfinite(creada_en)),
 PRIMARY KEY(usuario_id,clave)
);
CREATE INDEX peticion_spaf_emision ON documento.peticion_spaf(emision_id);
CREATE INDEX peticion_spaf_caducidad ON documento.peticion_spaf(creada_en);
ALTER TABLE documento.peticion_spaf ENABLE ROW LEVEL SECURITY;
ALTER TABLE documento.peticion_spaf FORCE ROW LEVEL SECURITY;
REVOKE ALL ON documento.peticion_spaf FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT,INSERT,UPDATE ON documento.peticion_spaf TO etf_app;
GRANT SELECT ON documento.peticion_spaf TO etf_copia;
CREATE POLICY peticion_spaf_actor ON documento.peticion_spaf TO etf_app
 USING (usuario_id::text=nullif(current_setting('app.usuario_id',true),'')
 AND EXISTS(SELECT 1 FROM documento.emision e WHERE e.id=emision_id AND e.tipo_documento='spaf')
 AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','profesional_etf']::text[])
 WITH CHECK (usuario_id::text=nullif(current_setting('app.usuario_id',true),'')
 AND EXISTS(SELECT 1 FROM documento.emision e WHERE e.id=emision_id AND e.tipo_documento='spaf')
 AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','profesional_etf']::text[]);

INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 SELECT 'spaf',codigo,'1.0.0','plantillas/spaf/v1.0.0.docx',decode('5085dfb7d7ea7918cb0b4e1c794835a9b4dc26d6c7c5e465bba350479b1dec0d','hex'),'sin_firmante',DATE '2026-10-13','motor_docx_1' FROM unnest(ARRAY['es','en']) codigo;
INSERT INTO documento.referencia_activa(tipo_documento,idioma,plantilla_id,version)
 SELECT tipo_documento,idioma,id,1 FROM documento.plantilla WHERE tipo_documento='spaf';
