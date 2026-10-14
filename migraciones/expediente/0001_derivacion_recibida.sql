SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$
BEGIN
    IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
       OR to_regclass('catalogo.municipio') IS NULL
       OR to_regclass('auditoria.evento') IS NULL
       OR to_regclass('identidad.usuario') IS NULL
       OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'expediente') THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;

CREATE SCHEMA expediente AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA expediente FROM PUBLIC;
GRANT USAGE ON SCHEMA expediente TO etf_app, etf_copia;

-- La recepción no asigna número ni equipo; se confirman al aceptar.
-- Las fases posteriores ampliarán este modelo mediante nuevas migraciones.
CREATE TABLE expediente.expediente (
    id uuid PRIMARY KEY,
    numero text UNIQUE CHECK (numero IS NULL),
    equipo_id uuid REFERENCES catalogo.equipo(id) CHECK (equipo_id IS NULL),
    municipio_id uuid REFERENCES catalogo.municipio(id) CHECK (municipio_id IS NULL),
    estado text NOT NULL DEFAULT 'derivacion_recibida' CHECK (estado = 'derivacion_recibida'),
    origen text NOT NULL DEFAULT 'derivacion' CHECK (origen = 'derivacion'),
    version integer NOT NULL DEFAULT 1 CHECK (version = 1),
    fecha_recepcion timestamptz NOT NULL CHECK (isfinite(fecha_recepcion))
);
CREATE INDEX expediente_recepcion ON expediente.expediente(fecha_recepcion, id);

CREATE TABLE expediente.lectura_anexo (
    id uuid PRIMARY KEY,
    expediente_id uuid NOT NULL UNIQUE REFERENCES expediente.expediente(id),
    formato text NOT NULL CHECK (formato IN ('docx', 'pdf')),
    tamano_bytes integer NOT NULL CHECK (tamano_bytes BETWEEN 1 AND 10000000),
    huella_sha256 bytea NOT NULL CHECK (octet_length(huella_sha256) = 32),
    antivirus text NOT NULL CHECK (antivirus IN ('limpio', 'no_revisado')),
    antivirus_detalle text NOT NULL DEFAULT '',
    plantilla_detectada text NOT NULL,
    campos_dudosos text[] NOT NULL,
    leida_por uuid NOT NULL REFERENCES identidad.usuario(id),
    leida_en timestamptz NOT NULL CHECK (isfinite(leida_en)),
    CHECK ((antivirus = 'limpio' AND antivirus_detalle = '')
        OR (antivirus = 'no_revisado' AND antivirus_detalle = 'expediente.antivirus_no_revisado'))
);
CREATE TABLE expediente.derivacion_datos (
    expediente_id uuid PRIMARY KEY REFERENCES expediente.expediente(id),
    esquema_version text NOT NULL CHECK (esquema_version = '1'),
    contenido jsonb NOT NULL CHECK (jsonb_typeof(contenido) = 'object')
);

-- Sólo Coordinación recibe derivaciones sin equipo. No se amplía el acceso
-- a otros perfiles hasta que exista una asignación confirmada y su flujo.
DO $$
DECLARE tabla text;
BEGIN
    FOREACH tabla IN ARRAY ARRAY['expediente', 'lectura_anexo', 'derivacion_datos'] LOOP
        EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY', tabla);
        EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY', tabla);
        EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC, etf_app, etf_copia', tabla);
        EXECUTE format('GRANT SELECT, INSERT ON expediente.%I TO etf_app', tabla);
        EXECUTE format('GRANT SELECT ON expediente.%I TO etf_copia', tabla);
        EXECUTE format('CREATE POLICY lectura ON expediente.%I FOR SELECT TO etf_app USING (
            nullif(current_setting(''app.usuario_id'', true), '''') IS NOT NULL
            AND ''coordinacion'' = ANY(string_to_array(current_setting(''app.perfiles'', true), '','')))', tabla);
        EXECUTE format('CREATE POLICY recepcion ON expediente.%I FOR INSERT TO etf_app WITH CHECK (
            nullif(current_setting(''app.usuario_id'', true), '''') IS NOT NULL
            AND ''coordinacion'' = ANY(string_to_array(current_setting(''app.perfiles'', true), '','')))', tabla);
    END LOOP;
END $$;

-- El actor de la lectura se obtiene del contexto del servidor.
DROP POLICY recepcion ON expediente.lectura_anexo;
CREATE POLICY recepcion ON expediente.lectura_anexo FOR INSERT TO etf_app WITH CHECK (
    leida_por::text = nullif(current_setting('app.usuario_id', true), '')
    AND 'coordinacion' = ANY(string_to_array(current_setting('app.perfiles', true), ','))
);
