SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.propuesta_cierre') IS NULL
 OR to_regprocedure('expediente.proteger_propuesta_version()') IS NULL
 OR to_regprocedure('expediente.proteger_ficha()') IS NULL
 OR to_regprocedure('expediente.proteger_inicio_intervencion()') IS NULL
 OR NOT EXISTS(SELECT 1 FROM information_schema.columns
               WHERE table_schema='expediente' AND table_name='expediente' AND column_name='familia') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
-- Familia, propuesta e hitos comparten la versión. Se conserva la única
-- transición de estado del profesional: iniciar intervención con consentimiento
-- confirmado. Añadir familia a las columnas permitidas no autoriza otros campos.
CREATE OR REPLACE FUNCTION expediente.proteger_ficha() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF current_user <> 'etf_app' THEN RETURN NEW; END IF;
 IF NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
  IF (to_jsonb(NEW)-ARRAY['familia','estado','version','modificado_por','modificado_en']) IS DISTINCT FROM
     (to_jsonb(OLD)-ARRAY['familia','estado','version','modificado_por','modificado_en']) THEN
   RAISE EXCEPTION 'error.sin_permiso';
  END IF;
  IF NEW.estado IS DISTINCT FROM OLD.estado AND NOT (
   OLD.estado='aceptada' AND NEW.estado='en_intervencion'
   AND EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=NEW.id AND tipo='consentimiento')) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION expediente.proteger_propuesta_version() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF current_user='etf_app' AND NEW.familia IS DISTINCT FROM OLD.familia
 AND OLD.estado NOT IN ('aceptada','en_intervencion') THEN
  RAISE EXCEPTION 'error.estado_no_permite' USING ERRCODE='42501';
 END IF;
 IF 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))) THEN
  IF (to_jsonb(NEW)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[])
   IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[])
   OR (NEW.estado IS DISTINCT FROM OLD.estado AND NOT (
    OLD.estado='aceptada' AND NEW.estado='en_intervencion'
    AND EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=NEW.id AND tipo='consentimiento'))) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION expediente.proteger_inicio_intervencion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
BEGIN
 -- El permiso previo de UPDATE incluye datos de aceptación. Confirmar un hito
 -- no autoriza a un profesional a corregirlos ni a perder su historia.
 IF 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
  AND NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))) THEN
  IF (to_jsonb(NEW)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[])
   IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[]) THEN
   RAISE EXCEPTION 'error.sin_permiso';
  END IF;
 END IF;
 IF NEW.estado='en_intervencion' AND OLD.estado IS DISTINCT FROM NEW.estado THEN
  IF OLD.estado<>'aceptada' OR NOT EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=NEW.id AND tipo='consentimiento') THEN
   RAISE EXCEPTION 'error.estado_no_permite';
  END IF;
 END IF;
 IF OLD.estado='en_intervencion' AND NEW.estado<>OLD.estado THEN
  RAISE EXCEPTION 'error.estado_no_permite';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_ficha(),expediente.proteger_propuesta_version(),expediente.proteger_inicio_intervencion() FROM PUBLIC,etf_app,etf_copia;
