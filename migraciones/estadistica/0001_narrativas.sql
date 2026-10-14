SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('catalogo.equipo') IS NULL OR to_regclass('estadistica.memoria') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
CREATE SCHEMA IF NOT EXISTS estadistica AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA estadistica FROM PUBLIC;
GRANT USAGE ON SCHEMA estadistica TO etf_app,etf_copia;
CREATE TABLE estadistica.memoria (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),anio integer NOT NULL CHECK(anio BETWEEN 1 AND 9998),
 equipo_id uuid REFERENCES catalogo.equipo(id),estado text NOT NULL DEFAULT 'borrador' CHECK(estado IN ('borrador','cerrada')),
 revision_actual integer,version integer NOT NULL CHECK(version>0),textos jsonb NOT NULL CHECK(jsonb_typeof(textos)='object'),
 modificado_por uuid NOT NULL REFERENCES identidad.usuario(id),modificado_en timestamptz NOT NULL,
 UNIQUE NULLS NOT DISTINCT(anio,equipo_id)
);
CREATE TABLE estadistica.memoria_narrativa_historia (
 memoria_id uuid NOT NULL REFERENCES estadistica.memoria(id),version integer NOT NULL CHECK(version>0),
 textos jsonb NOT NULL CHECK(jsonb_typeof(textos)='object'),
 registrado_por uuid NOT NULL REFERENCES identidad.usuario(id),registrado_en timestamptz NOT NULL,
 PRIMARY KEY(memoria_id,version)
);
ALTER TABLE estadistica.memoria ENABLE ROW LEVEL SECURITY;
ALTER TABLE estadistica.memoria FORCE ROW LEVEL SECURITY;
ALTER TABLE estadistica.memoria_narrativa_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE estadistica.memoria_narrativa_historia FORCE ROW LEVEL SECURITY;
REVOKE ALL ON estadistica.memoria,estadistica.memoria_narrativa_historia FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT,INSERT ON estadistica.memoria,estadistica.memoria_narrativa_historia TO etf_app;
GRANT UPDATE(version,textos,modificado_por,modificado_en) ON estadistica.memoria TO etf_app;
GRANT SELECT ON estadistica.memoria,estadistica.memoria_narrativa_historia TO etf_copia;
CREATE POLICY memoria_acceso ON estadistica.memoria TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))))
 WITH CHECK(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
CREATE POLICY memoria_historia_acceso ON estadistica.memoria_narrativa_historia TO etf_app
 USING(EXISTS(SELECT 1 FROM estadistica.memoria m WHERE m.id=memoria_id))
 WITH CHECK(registrado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND EXISTS(SELECT 1 FROM estadistica.memoria m WHERE m.id=memoria_id AND m.version=memoria_narrativa_historia.version));
