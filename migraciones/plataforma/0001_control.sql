-- El bootstrap crea los roles; el ejecutor conecta como etf_migrador y hace
-- SET ROLE etf_propietario antes de esta transacción.
DO $$
BEGIN
    IF current_user <> 'etf_propietario'
       OR session_user <> 'etf_migrador'
       OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'plataforma') THEN
        RAISE EXCEPTION 'migracion.estado_previo';
    END IF;
END $$;

CREATE SCHEMA plataforma AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA plataforma FROM PUBLIC;
GRANT USAGE ON SCHEMA plataforma TO etf_app, etf_copia;

ALTER DEFAULT PRIVILEGES FOR ROLE etf_propietario
    REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE etf_propietario
    REVOKE ALL ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE etf_propietario
    REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

CREATE TABLE plataforma.migracion_aplicada (
    modulo text NOT NULL CHECK (modulo ~ '^[a-z][a-z_]*$'),
    numero text NOT NULL CHECK (numero ~ '^[0-9]{4}$'),
    nombre text NOT NULL,
    huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
    aplicada_en timestamptz NOT NULL DEFAULT clock_timestamp(),
    aplicada_por text NOT NULL CHECK (aplicada_por = 'etf_migrador'),
    PRIMARY KEY (modulo, numero)
);
ALTER TABLE plataforma.migracion_aplicada ENABLE ROW LEVEL SECURITY;
ALTER TABLE plataforma.migracion_aplicada FORCE ROW LEVEL SECURITY;
CREATE POLICY migracion_propietario ON plataforma.migracion_aplicada
    TO etf_propietario USING (true) WITH CHECK (true);
CREATE POLICY migracion_compatibilidad ON plataforma.migracion_aplicada
    FOR SELECT TO etf_app USING (true);
REVOKE ALL ON plataforma.migracion_aplicada FROM PUBLIC, etf_app, etf_copia;
GRANT SELECT ON plataforma.migracion_aplicada TO etf_app, etf_copia;
