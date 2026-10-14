SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$
BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('expediente.expediente') IS NULL
       OR to_regclass('catalogo.firmante') IS NULL
       OR to_regclass('paquete.referencia_activa') IS NULL
       OR to_regclass('auditoria.evento') IS NULL
       OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='documento') THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;

CREATE SCHEMA documento AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA documento FROM PUBLIC;
GRANT USAGE ON SCHEMA documento TO etf_app,etf_copia;

CREATE TABLE documento.plantilla (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo_documento text NOT NULL CHECK (tipo_documento='entrada/aceptacion'),
    idioma text NOT NULL CHECK (length(idioma)<=63 AND idioma ~ '^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$'),
    version text NOT NULL CHECK (length(version)<=32 AND version ~ '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'),
    ruta_relativa text NOT NULL,
    huella_sha256 bytea NOT NULL CHECK (octet_length(huella_sha256)=32),
    rol_firmante text NOT NULL CHECK (rol_firmante ~ '^[a-z][a-z0-9_]{0,63}$'),
    estado text NOT NULL DEFAULT 'verificada' CHECK (estado='verificada'),
    vigente_desde date NOT NULL CHECK (isfinite(vigente_desde)),
    subida_por uuid REFERENCES identidad.usuario(id),
    subida_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    comprobacion text NOT NULL CHECK (comprobacion='motor_docx_1'),
    UNIQUE(tipo_documento,idioma,version),
    UNIQUE(id,tipo_documento,idioma),
    CHECK (ruta_relativa='plantillas/'||tipo_documento||'/v'||version||'.docx')
);
CREATE TABLE documento.referencia_activa (
    tipo_documento text NOT NULL,
    idioma text NOT NULL,
    plantilla_id uuid NOT NULL,
    version bigint NOT NULL CHECK (version>0),
    PRIMARY KEY(tipo_documento,idioma),
    FOREIGN KEY(plantilla_id,tipo_documento,idioma) REFERENCES documento.plantilla(id,tipo_documento,idioma)
);

CREATE TABLE documento.emision (
    id uuid PRIMARY KEY,
    tipo_documento text NOT NULL,
    expediente_id uuid NOT NULL REFERENCES expediente.expediente(id),
    equipo_id uuid REFERENCES catalogo.equipo(id),
    plantilla_id uuid NOT NULL,
    idioma text NOT NULL,
    fecha_documento date NOT NULL CHECK (isfinite(fecha_documento)),
    generada_por uuid NOT NULL REFERENCES identidad.usuario(id),
    generada_en timestamptz NOT NULL CHECK (isfinite(generada_en)),
    huella_sha256 bytea NOT NULL CHECK (octet_length(huella_sha256)=32),
    huella_datos bytea NOT NULL CHECK (octet_length(huella_datos)=32),
    versiones_datos jsonb NOT NULL CHECK (jsonb_typeof(versiones_datos)='object'),
    version_motor text NOT NULL CHECK (length(version_motor) BETWEEN 1 AND 64),
    ruta_idioma text NOT NULL,
    version_idioma text NOT NULL,
    aviso_clave text NOT NULL CHECK (aviso_clave IN ('','documento.regenerado')),
    FOREIGN KEY(plantilla_id,tipo_documento,idioma) REFERENCES documento.plantilla(id,tipo_documento,idioma)
);
CREATE INDEX emision_expediente_tipo ON documento.emision(expediente_id,tipo_documento,idioma,generada_en DESC,id DESC);
CREATE INDEX emision_equipo ON documento.emision(equipo_id,generada_en DESC);
CREATE INDEX emision_plantilla ON documento.emision(plantilla_id);
CREATE TABLE documento.peticion_idempotente (
    clave text NOT NULL CHECK (clave ~ '^[A-Za-z0-9_-]{1,128}$'),
    usuario_id uuid NOT NULL REFERENCES identidad.usuario(id),
    ruta text NOT NULL CHECK (ruta='/api/documentos/emisiones'),
    huella_peticion bytea NOT NULL CHECK (octet_length(huella_peticion)=32),
    emision_id uuid NOT NULL REFERENCES documento.emision(id),
    respuesta jsonb NOT NULL CHECK (respuesta=jsonb_build_object('emision_id',emision_id::text)),
    creada_en timestamptz NOT NULL CHECK (isfinite(creada_en)),
    PRIMARY KEY(usuario_id,ruta,clave)
);
CREATE INDEX peticion_caducidad ON documento.peticion_idempotente(creada_en);
CREATE INDEX peticion_emision ON documento.peticion_idempotente(emision_id);

-- La base anterior sólo admite equipo nulo y no publica aún la clave
-- (id,equipo_id). Este disparador valida la relación, incluso con nulos,
-- sin ampliar permisos ni modificar tablas de expediente.
CREATE FUNCTION documento.comprobar_pertenencia() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,documento AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM expediente.expediente e
                   WHERE e.id=NEW.expediente_id AND e.equipo_id IS NOT DISTINCT FROM NEW.equipo_id) THEN
        RAISE EXCEPTION 'documento.acceso_denegado' USING ERRCODE='42501';
    END IF;
    RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION documento.comprobar_pertenencia() FROM PUBLIC,etf_app;
CREATE TRIGGER pertenencia BEFORE INSERT ON documento.emision FOR EACH ROW EXECUTE FUNCTION documento.comprobar_pertenencia();
CREATE FUNCTION documento.inmutable() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,documento AS $$
BEGIN RAISE EXCEPTION 'documento.version_inmutable' USING ERRCODE='42501'; END $$;
REVOKE EXECUTE ON FUNCTION documento.inmutable() FROM PUBLIC,etf_app;
CREATE TRIGGER plantilla_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON documento.plantilla FOR EACH STATEMENT EXECUTE FUNCTION documento.inmutable();
CREATE TRIGGER emision_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON documento.emision FOR EACH STATEMENT EXECUTE FUNCTION documento.inmutable();

DO $$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY['plantilla','referencia_activa','emision','peticion_idempotente'] LOOP
        EXECUTE format('ALTER TABLE documento.%I ENABLE ROW LEVEL SECURITY',tabla);
        EXECUTE format('ALTER TABLE documento.%I FORCE ROW LEVEL SECURITY',tabla);
        EXECUTE format('REVOKE ALL ON documento.%I FROM PUBLIC,etf_app,etf_copia',tabla);
        EXECUTE format('GRANT SELECT ON documento.%I TO etf_copia',tabla);
    END LOOP;
END $$;
CREATE POLICY plantilla_propietario ON documento.plantilla TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY referencia_propietario ON documento.referencia_activa TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY plantilla_lectura ON documento.plantilla FOR SELECT TO etf_app USING (
    nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
    AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','equipo','consulta','administracion']::text[]);
CREATE POLICY referencia_lectura ON documento.referencia_activa FOR SELECT TO etf_app USING (
    nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
    AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','equipo','consulta','administracion']::text[]);
CREATE POLICY emision_lectura ON documento.emision FOR SELECT TO etf_app USING (
    nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND (
        'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
        OR ('equipo'=ANY(string_to_array(current_setting('app.perfiles',true),','))
            AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))));
CREATE POLICY emision_escritura ON documento.emision FOR INSERT TO etf_app WITH CHECK (
    generada_por::text=nullif(current_setting('app.usuario_id',true),'') AND (
        'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
        OR ('equipo'=ANY(string_to_array(current_setting('app.perfiles',true),','))
            AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))));
CREATE POLICY peticion_actor ON documento.peticion_idempotente TO etf_app
USING (usuario_id::text=nullif(current_setting('app.usuario_id',true),'')
       AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','equipo']::text[])
WITH CHECK (usuario_id::text=nullif(current_setting('app.usuario_id',true),'')
       AND string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','equipo']::text[]);
GRANT SELECT ON documento.plantilla,documento.referencia_activa TO etf_app;
GRANT SELECT,INSERT ON documento.emision TO etf_app;
GRANT SELECT,INSERT,UPDATE ON documento.peticion_idempotente TO etf_app;

-- Plantillas oficiales de distribución, verificadas por el motor. La carga
-- no contiene expedientes, personas ni firmantes sintéticos.
INSERT INTO documento.plantilla(tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion)
SELECT 'entrada/aceptacion',codigo,'1.0.0','plantillas/entrada/aceptacion/v1.0.0.docx',
       decode('fee157922aff0c066460162851e10b81325a83a0814ee26f3ad5fc9943dcba5a','hex'),
       'aceptacion',DATE '2026-10-13','motor_docx_1' FROM unnest(ARRAY['es','en']) codigo;
INSERT INTO documento.referencia_activa SELECT tipo_documento,idioma,id,1 FROM documento.plantilla;
