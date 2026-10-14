SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('catalogo.version') IS NULL OR to_regclass('catalogo.regla_numeracion') IS NOT NULL
 OR EXISTS (SELECT 1 FROM catalogo.version WHERE ambito='numeracion') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
CREATE TABLE catalogo.regla_numeracion (
 version_id uuid PRIMARY KEY REFERENCES catalogo.version(id),
 componentes text[] NOT NULL CHECK (cardinality(componentes)=3 AND componentes @> ARRAY['equipo','correlativo','anio']),
 cifras_equipo integer NOT NULL CHECK (cifras_equipo BETWEEN 1 AND 4),
 cifras_correlativo integer NOT NULL CHECK (cifras_correlativo BETWEEN 1 AND 9),
 cifras_anio integer NOT NULL CHECK (cifras_anio IN (2,4)),
 separador text NOT NULL CHECK (separador IN ('-','/','.'))
);
ALTER TABLE catalogo.regla_numeracion ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalogo.regla_numeracion FORCE ROW LEVEL SECURITY;
REVOKE ALL ON catalogo.regla_numeracion FROM PUBLIC, etf_app, etf_copia;
GRANT SELECT ON catalogo.regla_numeracion TO etf_app, etf_copia;
CREATE POLICY propietario ON catalogo.regla_numeracion TO etf_propietario USING (true) WITH CHECK (true);
CREATE POLICY lectura ON catalogo.regla_numeracion FOR SELECT TO etf_app USING (
 EXISTS (SELECT 1 FROM catalogo.version v WHERE v.id=version_id)
);
CREATE TRIGGER proteger_numeracion BEFORE INSERT OR UPDATE OR DELETE ON catalogo.regla_numeracion
 FOR EACH ROW EXECUTE FUNCTION catalogo.proteger_regla('numeracion');
CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON catalogo.regla_numeracion
 FOR EACH STATEMENT EXECUTE FUNCTION catalogo.impedir_cambio();
-- Decisión provisional D1. La publicación conserva los límites y el contenido.
WITH nueva AS (
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
 VALUES ('numeracion',1,'2026-01-01','numeracion.d1','00000000-0000-4000-8000-000000000001') RETURNING id
)
INSERT INTO catalogo.regla_numeracion
 SELECT id,ARRAY['equipo','correlativo','anio'],2,2,2,'-' FROM nueva;
SELECT catalogo.publicar(id,'00000000-0000-4000-8000-000000000001') FROM catalogo.version WHERE ambito='numeracion';
