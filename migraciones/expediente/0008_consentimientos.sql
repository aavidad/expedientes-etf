SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.hito') IS NULL
 OR to_regprocedure('expediente.proteger_propuesta_version()') IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='expediente.expediente'::regclass AND conname='expediente_decision_completa')
 OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='expediente'::regnamespace AND proname='proteger_hito_confirmado') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Se añade la intervención conservando las condiciones previas de las decisiones.
ALTER TABLE expediente.expediente
 DROP CONSTRAINT expediente_estado_check,
 DROP CONSTRAINT expediente_aceptacion_completa,
 DROP CONSTRAINT expediente_decision_completa,
 ADD CONSTRAINT expediente_estado_check CHECK (estado IN ('derivacion_recibida','lista_espera','no_aceptada','aceptada','en_intervencion')),
 ADD CONSTRAINT expediente_aceptacion_completa CHECK (
  (estado IN ('derivacion_recibida','lista_espera','no_aceptada') AND numero IS NULL AND equipo_id IS NULL AND municipio_id IS NULL
   AND fecha_aceptacion IS NULL AND anio_numero IS NULL AND correlativo IS NULL AND regla_numeracion_id IS NULL AND centro_ss_id IS NULL
   AND profesional_referencia_puesto IS NULL AND profesional_referencia_id IS NULL AND descripcion_situacion IS NULL AND fase_intervencion IS NULL)
  OR (estado IN ('aceptada','en_intervencion') AND numero IS NOT NULL AND equipo_id IS NOT NULL AND municipio_id IS NOT NULL AND centro_ss_id IS NOT NULL
   AND fecha_aceptacion IS NOT NULL AND anio_numero IS NOT NULL AND anio_numero=extract(year FROM fecha_aceptacion)
   AND correlativo IS NOT NULL AND regla_numeracion_id IS NOT NULL AND subprograma IS NOT NULL
   AND profesional_referencia_puesto IS NOT NULL AND descripcion_situacion IS NOT NULL AND fase_intervencion IS NOT NULL
   AND modificado_por IS NOT NULL AND modificado_en IS NOT NULL)
 ),
 ADD CONSTRAINT expediente_decision_completa CHECK (
  (estado='no_aceptada' AND fecha_no_aceptacion IS NOT NULL AND causa_no_aceptacion IS NOT NULL AND subprograma IS NOT NULL
   AND modificado_por IS NOT NULL AND modificado_en IS NOT NULL)
  OR (estado<>'no_aceptada' AND fecha_no_aceptacion IS NULL AND causa_no_aceptacion IS NULL
   AND (estado IN ('aceptada','en_intervencion') OR subprograma IS NULL))
 );
ALTER TABLE expediente.hito
 DROP CONSTRAINT hito_tipo_check,
 DROP CONSTRAINT hito_asignacion,
 ADD CONSTRAINT hito_tipo_check CHECK(tipo IN ('aceptacion','no_aceptacion','lista_espera','consentimiento','anexo7','anexo8')),
 ADD CONSTRAINT hito_asignacion CHECK (
  (tipo IN ('aceptacion','consentimiento','anexo7','anexo8') AND equipo_id IS NOT NULL AND episodio_id IS NOT NULL)
  OR (tipo IN ('no_aceptacion','lista_espera') AND equipo_id IS NULL AND episodio_id IS NULL)
 );
ALTER TABLE expediente.estado_cambio
 DROP CONSTRAINT estado_cambio_transicion,
 DROP CONSTRAINT estado_cambio_asignacion,
 ADD CONSTRAINT estado_cambio_transicion CHECK (
  (estado_anterior='derivacion_recibida' AND estado_nuevo IN ('lista_espera','no_aceptada','aceptada'))
  OR (estado_anterior='lista_espera' AND estado_nuevo IN ('no_aceptada','aceptada'))
  OR (estado_anterior='aceptada' AND estado_nuevo='en_intervencion')
 ),
 ADD CONSTRAINT estado_cambio_asignacion CHECK ((estado_nuevo IN ('aceptada','en_intervencion') AND equipo_id IS NOT NULL)
  OR (estado_nuevo IN ('lista_espera','no_aceptada') AND equipo_id IS NULL));

-- La RLS permite únicamente filas del equipo de la sesión. No concede permisos nuevos
-- sobre columnas: se reutiliza el UPDATE existente del rol de aplicación.
CREATE POLICY equipo_confirmar_hito ON expediente.expediente FOR UPDATE TO etf_app
 USING (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
  AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]) AND estado IN ('aceptada','en_intervencion'))
 WITH CHECK (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
  AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]) AND estado IN ('aceptada','en_intervencion') AND modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid);
CREATE POLICY equipo_confirmar_hito ON expediente.hito FOR INSERT TO etf_app
 WITH CHECK (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
  AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]) AND tipo IN ('consentimiento','anexo7','anexo8') AND confirmado_por=nullif(current_setting('app.usuario_id',true),'')::uuid);
CREATE POLICY equipo_confirmar_hito ON expediente.estado_cambio FOR INSERT TO etf_app
 WITH CHECK (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
  AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]) AND estado_anterior='aceptada' AND estado_nuevo='en_intervencion' AND cambiado_por=nullif(current_setting('app.usuario_id',true),'')::uuid);
DROP POLICY propia ON expediente.peticion_idempotente;
CREATE POLICY propia ON expediente.peticion_idempotente TO etf_app
 USING (usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]) && ARRAY['coordinacion','profesional_etf']::text[])
 WITH CHECK (usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]) && ARRAY['coordinacion','profesional_etf']::text[]);

-- Segunda barrera para fecha, subprograma y estado. No se aceptan firmas futuras.
CREATE FUNCTION expediente.proteger_hito_confirmado() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
DECLARE e expediente.expediente;
BEGIN
 IF NEW.tipo IN ('consentimiento','anexo7','anexo8') THEN
  SELECT * INTO e FROM expediente.expediente WHERE id=NEW.expediente_id FOR UPDATE;
  IF e.id IS NULL OR e.estado NOT IN ('aceptada','en_intervencion')
   OR (NEW.tipo='consentimiento' AND e.estado<>'aceptada') THEN
   RAISE EXCEPTION 'error.estado_no_permite';
  END IF;
  IF NEW.fecha<e.fecha_aceptacion OR NEW.fecha>(NEW.confirmado_en AT TIME ZONE 'Europe/Madrid')::date THEN
   RAISE EXCEPTION 'expediente.hito_fecha_invalida';
  END IF;
  IF (NEW.tipo='anexo7' AND e.subprograma<>'RIESGO_SOCIAL') OR (NEW.tipo='anexo8' AND e.subprograma<>'REUNIFICACION') THEN
   RAISE EXCEPTION 'expediente.subprograma_invalido';
  END IF;
  IF NEW.confirmado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid THEN
   RAISE EXCEPTION 'error.sin_permiso';
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_hito_confirmado() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_hito_confirmado BEFORE INSERT ON expediente.hito
 FOR EACH ROW EXECUTE FUNCTION expediente.proteger_hito_confirmado();

-- La protección de propuesta de cierre sigue permitiendo sólo la versión y las
-- marcas técnicas. La única excepción nueva es iniciar la intervención con firma.
CREATE OR REPLACE FUNCTION expediente.proteger_propuesta_version() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))) THEN
  IF (to_jsonb(NEW)-ARRAY['estado','version','modificado_por','modificado_en']::text[])
   IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['estado','version','modificado_por','modificado_en']::text[])
   OR (NEW.estado IS DISTINCT FROM OLD.estado AND NOT (
    OLD.estado='aceptada' AND NEW.estado='en_intervencion'
    AND EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=NEW.id AND tipo='consentimiento'))) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_propuesta_version() FROM PUBLIC,etf_app,etf_copia;

CREATE FUNCTION expediente.proteger_inicio_intervencion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
BEGIN
 -- El permiso previo de UPDATE incluye datos de aceptación. Confirmar un hito
 -- no autoriza a un profesional a corregirlos ni a perder su historia.
 IF 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
  AND NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))) THEN
  IF (to_jsonb(NEW)-ARRAY['estado','version','modificado_por','modificado_en']::text[])
   IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['estado','version','modificado_por','modificado_en']::text[]) THEN
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
REVOKE EXECUTE ON FUNCTION expediente.proteger_inicio_intervencion() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_inicio_intervencion BEFORE UPDATE ON expediente.expediente
 FOR EACH ROW EXECUTE FUNCTION expediente.proteger_inicio_intervencion();
