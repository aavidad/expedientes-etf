SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('catalogo.smi') IS NULL
 OR EXISTS (SELECT 1 FROM pg_policies
            WHERE schemaname='catalogo' AND tablename='smi'
              AND policyname='lectura_profesional_etf') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
-- El SMI no contiene datos de expedientes. Sólo se amplía su lectura al
-- profesional con equipo de sesión; la función de escritura sigue reservada
-- a Coordinación por catalogo/0005.
CREATE POLICY lectura_profesional_etf ON catalogo.smi
 FOR SELECT TO etf_app
 USING (
  nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL
  AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
  AND cardinality(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[]))=1
 );
