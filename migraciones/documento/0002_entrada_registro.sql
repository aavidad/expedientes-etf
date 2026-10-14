SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('documento.emision') IS NULL OR to_regclass('documento.registro_salida') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='expediente' AND table_name='expediente' AND column_name='fecha_aceptacion') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

CREATE TABLE documento.tipo_documento (
 codigo text PRIMARY KEY,
 requiere_registro boolean NOT NULL
);
INSERT INTO documento.tipo_documento VALUES
 ('entrada/aceptacion',false),('entrada/aceptacion/oficio_sscc',true),('entrada/aceptacion/oficio_spm',true);
ALTER TABLE documento.plantilla DROP CONSTRAINT plantilla_tipo_documento_check;
ALTER TABLE documento.plantilla ADD CONSTRAINT plantilla_tipo_documento_fkey FOREIGN KEY(tipo_documento) REFERENCES documento.tipo_documento(codigo);

CREATE TABLE documento.registro_salida (
 emision_id uuid PRIMARY KEY REFERENCES documento.emision(id),
 adaptador text NOT NULL CHECK(adaptador IN ('manual','moad')),
 estado text NOT NULL CHECK(estado IN ('pendiente','enviado','anotado')),
 numero_registro text CHECK(length(numero_registro) BETWEEN 1 AND 128 AND numero_registro=btrim(numero_registro) AND numero_registro !~ '[[:cntrl:]]'),
 fecha_registro date CHECK(isfinite(fecha_registro)),
 referencia_externa text CHECK(length(referencia_externa) BETWEEN 1 AND 512),
 anotado_por uuid REFERENCES identidad.usuario(id),
 anotado_en timestamptz CHECK(isfinite(anotado_en)),
 error_clave text CHECK(error_clave ~ '^[a-z][a-z0-9_.]{1,127}$'),
 version integer NOT NULL CHECK(version>0),
 CHECK((estado='pendiente' AND numero_registro IS NULL AND fecha_registro IS NULL AND anotado_por IS NULL AND anotado_en IS NULL)
 OR (estado='enviado' AND adaptador='moad' AND referencia_externa IS NOT NULL)
 OR (estado='anotado' AND numero_registro IS NOT NULL AND fecha_registro IS NOT NULL AND anotado_por IS NOT NULL AND anotado_en IS NOT NULL))
);
CREATE INDEX registro_salida_estado ON documento.registro_salida(estado,fecha_registro);

CREATE FUNCTION documento.comprobar_registro_salida() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,documento AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM documento.emision e JOIN documento.tipo_documento t ON t.codigo=e.tipo_documento WHERE e.id=NEW.emision_id AND t.requiere_registro) THEN
  RAISE EXCEPTION 'documento.registro_solo_oficios' USING ERRCODE='42501';
 END IF;
 IF TG_OP='INSERT' AND (NEW.estado<>'pendiente' OR NEW.version<>1) THEN
  RAISE EXCEPTION 'documento.peticion_invalida' USING ERRCODE='23514';
 END IF;
 IF TG_OP='UPDATE' AND (NEW.emision_id<>OLD.emision_id OR NEW.version<>OLD.version+1 OR NEW.estado<>'anotado'
 OR NEW.anotado_por::text IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')) THEN
  RAISE EXCEPTION 'documento.peticion_invalida' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION documento.comprobar_registro_salida() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER comprobar_registro BEFORE INSERT OR UPDATE ON documento.registro_salida FOR EACH ROW EXECUTE FUNCTION documento.comprobar_registro_salida();

ALTER TABLE documento.tipo_documento ENABLE ROW LEVEL SECURITY;
ALTER TABLE documento.tipo_documento FORCE ROW LEVEL SECURITY;
ALTER TABLE documento.registro_salida ENABLE ROW LEVEL SECURITY;
ALTER TABLE documento.registro_salida FORCE ROW LEVEL SECURITY;
REVOKE ALL ON documento.tipo_documento,documento.registro_salida FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON documento.tipo_documento,documento.registro_salida TO etf_copia;
GRANT SELECT ON documento.tipo_documento TO etf_app;
GRANT SELECT,INSERT ON documento.registro_salida TO etf_app;
GRANT UPDATE(adaptador,estado,numero_registro,fecha_registro,referencia_externa,anotado_por,anotado_en,error_clave,version) ON documento.registro_salida TO etf_app;
CREATE POLICY tipos_lectura ON documento.tipo_documento FOR SELECT TO etf_app USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL);
CREATE POLICY tipos_propietario ON documento.tipo_documento TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY registros_lectura ON documento.registro_salida FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND EXISTS(SELECT 1 FROM documento.emision e WHERE e.id=emision_id));
CREATE POLICY registros_insertar ON documento.registro_salida FOR INSERT TO etf_app WITH CHECK(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND estado='pendiente' AND version=1 AND EXISTS(SELECT 1 FROM documento.emision e WHERE e.id=emision_id AND e.generada_por::text=current_setting('app.usuario_id',true)));
CREATE POLICY registros_anotar ON documento.registro_salida FOR UPDATE TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))) WITH CHECK(
 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND anotado_por::text=nullif(current_setting('app.usuario_id',true),''));

-- Las versiones anteriores permanecen referenciables para reproducir emisiones.
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 SELECT 'entrada/aceptacion',codigo,'1.1.0','plantillas/entrada/aceptacion/v1.1.0.docx',decode('1e323080f14f70a0379f11a04d08576cba18209eba5eae3bf7a468b2187bf1ac','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1' FROM unnest(ARRAY['es','en']) codigo;
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 SELECT 'entrada/aceptacion/oficio_sscc',codigo,'1.0.0','plantillas/entrada/aceptacion/oficio_sscc/v1.0.0.docx',decode('f10b451c01d5faad6ae1679cc73217432502a28602ad64431bab850e4bba57e0','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1' FROM unnest(ARRAY['es','en']) codigo;
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
 SELECT 'entrada/aceptacion/oficio_spm',codigo,'1.0.0','plantillas/entrada/aceptacion/oficio_spm/v1.0.0.docx',decode('fb08a2e575c21f551d11c62f0b22f9c6721e51aaeb6f62e3b8fc6fac1f819f73','hex'),'aceptacion',DATE '2026-10-13','motor_docx_1' FROM unnest(ARRAY['es','en']) codigo;
INSERT INTO documento.referencia_activa SELECT tipo_documento,idioma,id,1 FROM documento.plantilla WHERE version='1.1.0' OR tipo_documento<>'entrada/aceptacion'
 ON CONFLICT(tipo_documento,idioma) DO UPDATE SET plantilla_id=EXCLUDED.plantilla_id,version=documento.referencia_activa.version+1;

CREATE TABLE documento.registro_peticion (
 usuario_id uuid NOT NULL REFERENCES identidad.usuario(id),
 clave text NOT NULL CHECK(clave ~ '^[A-Za-z0-9_-]{1,128}$'),
 emision_id uuid NOT NULL REFERENCES documento.emision(id),
 ruta text NOT NULL CHECK(ruta='/api/documentos/emisiones/'||emision_id::text||'/registro-salida'),
 huella_peticion bytea NOT NULL CHECK(octet_length(huella_peticion)=32),
 respuesta jsonb NOT NULL CHECK(jsonb_typeof(respuesta)='object'),
 creada_en timestamptz NOT NULL CHECK(isfinite(creada_en)),
 PRIMARY KEY(usuario_id,clave)
);
CREATE INDEX registro_peticion_caducidad ON documento.registro_peticion(creada_en);
CREATE INDEX registro_peticion_emision ON documento.registro_peticion(emision_id);
ALTER TABLE documento.registro_peticion ENABLE ROW LEVEL SECURITY;
ALTER TABLE documento.registro_peticion FORCE ROW LEVEL SECURITY;
REVOKE ALL ON documento.registro_peticion FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON documento.registro_peticion TO etf_copia;
GRANT SELECT,INSERT,UPDATE ON documento.registro_peticion TO etf_app;
CREATE POLICY registro_peticion_actor ON documento.registro_peticion TO etf_app
 USING(usuario_id::text=nullif(current_setting('app.usuario_id',true),'') AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')))
 WITH CHECK(usuario_id::text=nullif(current_setting('app.usuario_id',true),'') AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND EXISTS(SELECT 1 FROM documento.emision e WHERE e.id=emision_id));
