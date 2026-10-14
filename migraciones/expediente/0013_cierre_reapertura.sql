SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.propuesta_cierre') IS NULL
 OR to_regprocedure('expediente.proteger_hito_confirmado()') IS NULL OR to_regclass('expediente.adulto') IS NULL
 OR EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='expediente' AND table_name='episodio' AND column_name='cierre_fecha') THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
-- Se amplía sólo el conjunto de estados aceptados en las restricciones existentes.
DO $$ DECLARE c record; definicion text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) AS def FROM pg_constraint
 WHERE conrelid='expediente.expediente'::regclass AND conname IN ('expediente_estado_check','expediente_aceptacion_completa','expediente_decision_completa') LOOP
  definicion:=replace(c.def,'''en_intervencion''::text','''en_intervencion''::text, ''derivada_spm''::text, ''cerrada''::text');
  EXECUTE format('ALTER TABLE expediente.expediente DROP CONSTRAINT %I, ADD CONSTRAINT %I %s',c.conname,c.conname,definicion);
 END LOOP;
END $$;
ALTER TABLE expediente.episodio
 ADD COLUMN cierre_fecha date CHECK(isfinite(cierre_fecha)),
 ADD COLUMN motivo_cierre text,
 ADD COLUMN motivo_acta text,
 ADD COLUMN motivo_acta_texto text,
 ADD COLUMN motivo_spaf text,
 ADD COLUMN cerrado_por uuid REFERENCES identidad.usuario(id),
 ADD COLUMN cerrado_en timestamptz,
 ADD COLUMN propuesta_cierre_id uuid REFERENCES expediente.propuesta_cierre(id),
 ADD COLUMN forzado_id uuid REFERENCES expediente.spm_forzado(id),
 ADD COLUMN reapertura_de_id uuid,
 ADD COLUMN motivo_reapertura text,
 ADD FOREIGN KEY(reapertura_de_id,expediente_id,equipo_id) REFERENCES expediente.episodio(id,expediente_id,equipo_id),
 ADD CHECK ((cierre_fecha IS NULL AND motivo_cierre IS NULL AND motivo_acta IS NULL AND motivo_acta_texto IS NULL AND motivo_spaf IS NULL AND cerrado_por IS NULL AND cerrado_en IS NULL AND propuesta_cierre_id IS NULL AND forzado_id IS NULL)
 OR (cierre_fecha>=apertura_fecha AND motivo_cierre IS NOT NULL AND motivo_acta IS NOT NULL AND motivo_acta_texto IS NOT NULL AND motivo_spaf IS NOT NULL AND cerrado_por IS NOT NULL AND cerrado_en IS NOT NULL AND propuesta_cierre_id IS NOT NULL)),
 ADD CHECK ((reapertura_de_id IS NULL AND motivo_reapertura IS NULL) OR (reapertura_de_id IS NOT NULL AND length(btrim(motivo_reapertura)) BETWEEN 1 AND 4000));
CREATE UNIQUE INDEX episodio_origen_unico ON expediente.episodio(expediente_id) WHERE reapertura_de_id IS NULL;
CREATE INDEX episodio_historia ON expediente.episodio(expediente_id,id);
CREATE INDEX propuesta_cierre_historia ON expediente.propuesta_cierre(expediente_id,propuesta_en DESC,id);
CREATE UNIQUE INDEX episodio_activo ON expediente.episodio(expediente_id) WHERE cierre_fecha IS NULL;
CREATE UNIQUE INDEX episodio_reapertura_unica ON expediente.episodio(reapertura_de_id) WHERE reapertura_de_id IS NOT NULL;
GRANT UPDATE(cierre_fecha,motivo_cierre,motivo_acta,motivo_acta_texto,motivo_spaf,cerrado_por,cerrado_en,propuesta_cierre_id,forzado_id) ON expediente.episodio TO etf_app;
CREATE POLICY coordinacion_confirmar ON expediente.propuesta_cierre FOR UPDATE TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')))
 WITH CHECK(resuelta_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
CREATE OR REPLACE FUNCTION expediente.proteger_propuesta_cierre() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.estado<>'pendiente' OR NEW.propuesta_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 ELSE
  IF OLD.estado<>'pendiente' OR NEW.resuelta_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid
   OR NOT (NEW.estado='retirada' OR (NEW.estado='confirmada' AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND NEW.resuelta_por<>OLD.propuesta_por)) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_propuesta_cierre() FROM PUBLIC,etf_app,etf_copia;
ALTER TABLE expediente.estado_cambio DROP CONSTRAINT estado_cambio_transicion, DROP CONSTRAINT estado_cambio_asignacion,
 ADD CONSTRAINT estado_cambio_transicion CHECK (
 (estado_anterior='derivacion_recibida' AND estado_nuevo IN ('lista_espera','no_aceptada','aceptada'))
 OR (estado_anterior='lista_espera' AND estado_nuevo IN ('no_aceptada','aceptada'))
 OR (estado_anterior='aceptada' AND estado_nuevo='en_intervencion')
 OR (estado_anterior IN ('aceptada','en_intervencion','derivada_spm') AND estado_nuevo='cerrada')
 OR (estado_anterior='cerrada' AND estado_nuevo IN ('aceptada','en_intervencion','derivada_spm'))),
 ADD CONSTRAINT estado_cambio_asignacion CHECK ((estado_nuevo IN ('aceptada','en_intervencion','derivada_spm','cerrada') AND equipo_id IS NOT NULL) OR (estado_nuevo IN ('lista_espera','no_aceptada') AND equipo_id IS NULL));
CREATE FUNCTION expediente.proteger_episodio_cierre() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ DECLARE previo expediente.episodio; BEGIN
 IF TG_OP='UPDATE' THEN
  IF OLD.cierre_fecha IS NOT NULL OR NEW.cierre_fecha IS NULL OR NEW.cierre_fecha>(NEW.cerrado_en AT TIME ZONE 'Europe/Madrid')::date OR NEW.cerrado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid
   OR NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
   OR NOT EXISTS(SELECT 1 FROM expediente.propuesta_cierre p WHERE p.id=NEW.propuesta_cierre_id AND p.expediente_id=NEW.expediente_id AND p.estado='confirmada' AND p.resuelta_por=NEW.cerrado_por AND p.motivo_finalizacion=NEW.motivo_cierre AND p.motivo_acta=NEW.motivo_acta AND p.motivo_spaf=NEW.motivo_spaf) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
  IF EXISTS(SELECT 1 FROM expediente.salida_spm sp WHERE sp.expediente_id=NEW.expediente_id AND sp.respuesta='a_la_espera' AND NOT EXISTS(
   SELECT 1 FROM expediente.spm_forzado f WHERE f.id=NEW.forzado_id AND f.salida_id=sp.id AND f.version_salida=sp.version)) THEN
   RAISE EXCEPTION 'expediente.cierre_bloqueado_anexo5';
  END IF;
  IF NEW.forzado_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM expediente.salida_spm sp JOIN expediente.spm_forzado f ON f.salida_id=sp.id AND f.version_salida=sp.version
   WHERE sp.expediente_id=NEW.expediente_id AND sp.respuesta='a_la_espera' AND f.id=NEW.forzado_id) THEN
   RAISE EXCEPTION 'expediente.cierre_invalido';
  END IF;
  IF (to_jsonb(NEW)-ARRAY['cierre_fecha','motivo_cierre','motivo_acta','motivo_acta_texto','motivo_spaf','cerrado_por','cerrado_en','propuesta_cierre_id','forzado_id']) IS DISTINCT FROM
     (to_jsonb(OLD)-ARRAY['cierre_fecha','motivo_cierre','motivo_acta','motivo_acta_texto','motivo_spaf','cerrado_por','cerrado_en','propuesta_cierre_id','forzado_id']) THEN
   RAISE EXCEPTION 'expediente.asignacion_inmutable';
  END IF;
 ELSE
  IF NEW.cierre_fecha IS NOT NULL OR NOT EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=NEW.expediente_id AND e.estado IN ('aceptada','en_intervencion','derivada_spm')) THEN
   RAISE EXCEPTION 'error.estado_no_permite';
  END IF;
  IF NEW.reapertura_de_id IS NOT NULL THEN
  SELECT * INTO previo FROM expediente.episodio WHERE id=NEW.reapertura_de_id;
  IF previo.cierre_fecha IS NULL OR NEW.apertura_fecha<previo.cierre_fecha OR NEW.cierre_fecha IS NOT NULL
   OR NEW.creado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid
   OR NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_episodio_cierre() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_episodio_cierre BEFORE INSERT OR UPDATE ON expediente.episodio FOR EACH ROW EXECUTE FUNCTION expediente.proteger_episodio_cierre();
-- La segunda barrera bloquea también escritura directa con el rol de la aplicación.
CREATE FUNCTION expediente.proteger_expediente_cerrado() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF OLD.estado='cerrada' THEN
  IF NEW.estado NOT IN ('aceptada','en_intervencion','derivada_spm')
   OR NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
   OR (to_jsonb(NEW)-ARRAY['estado','version','modificado_por','modificado_en']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['estado','version','modificado_por','modificado_en']) THEN
   RAISE EXCEPTION 'error.estado_no_permite';
  END IF;
 END IF;
 IF NEW.estado='cerrada' AND OLD.estado<>'cerrada' THEN
  IF NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
   OR NOT EXISTS(SELECT 1 FROM expediente.episodio p WHERE p.expediente_id=NEW.id AND p.cierre_fecha IS NOT NULL AND p.cerrado_por=nullif(current_setting('app.usuario_id',true),'')::uuid)
   OR EXISTS(SELECT 1 FROM expediente.episodio p WHERE p.expediente_id=NEW.id AND p.cierre_fecha IS NULL) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_expediente_cerrado() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_expediente_cerrado BEFORE UPDATE ON expediente.expediente FOR EACH ROW EXECUTE FUNCTION expediente.proteger_expediente_cerrado();
CREATE FUNCTION expediente.exigir_expediente_abierto() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ DECLARE recurso uuid; estado_actual text; BEGIN
 recurso:=coalesce((to_jsonb(NEW)->>'expediente_id')::uuid,(to_jsonb(OLD)->>'expediente_id')::uuid);
 SELECT estado INTO estado_actual FROM expediente.expediente WHERE id=recurso FOR UPDATE;
 IF estado_actual='cerrada' THEN RAISE EXCEPTION 'error.estado_no_permite'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.exigir_expediente_abierto() FROM PUBLIC,etf_app,etf_copia;
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['finalizacion','propuesta_cierre','hito','menor','dato_estadistico_vigencia','salida_spm','spm_historia','spm_forzado','salida_servicio','salida_servicio_hito','derivacion_datos','adulto','familia_revision','adultos_revision','adulto_revision'] LOOP
  EXECUTE format('CREATE TRIGGER exigir_expediente_abierto BEFORE INSERT OR UPDATE OR DELETE ON expediente.%I FOR EACH ROW EXECUTE FUNCTION expediente.exigir_expediente_abierto()',tabla);
 END LOOP;
END $$;
-- La coherencia se comprueba al confirmar la transacción, después de crear
-- el episodio de aceptación o reapertura. No admite una reapertura sin episodio.
CREATE FUNCTION expediente.comprobar_episodio_actual() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ DECLARE abiertos integer; estado_actual text; recurso uuid; BEGIN
 IF TG_TABLE_NAME='episodio' THEN recurso:=NEW.expediente_id;
 ELSE
  IF TG_OP='INSERT' AND NEW.estado<>'cerrada' THEN RETURN NULL; END IF;
  IF TG_OP='UPDATE' AND OLD.estado<>'cerrada' AND NEW.estado<>'cerrada' THEN RETURN NULL; END IF;
  recurso:=NEW.id;
 END IF;
 SELECT estado INTO estado_actual FROM expediente.expediente WHERE id=recurso;
 SELECT count(*) INTO abiertos FROM expediente.episodio WHERE expediente_id=recurso AND cierre_fecha IS NULL;
 IF (estado_actual='cerrada' AND abiertos<>0) OR (estado_actual IN ('aceptada','en_intervencion','derivada_spm') AND abiertos<>1) THEN
  RAISE EXCEPTION 'error.estado_no_permite';
 END IF;
 RETURN NULL;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.comprobar_episodio_actual() FROM PUBLIC,etf_app,etf_copia;
CREATE CONSTRAINT TRIGGER comprobar_episodio_actual AFTER INSERT OR UPDATE ON expediente.expediente
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION expediente.comprobar_episodio_actual();
-- Se conserva la protección de consentimiento de F2.03 y se habilitan sólo
-- cierre y restauración por Coordinación, sin inventar una firma.
CREATE OR REPLACE FUNCTION expediente.proteger_inicio_intervencion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
BEGIN
 IF 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])))
 AND (to_jsonb(NEW)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[])
 IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[]) THEN
  RAISE EXCEPTION 'error.sin_permiso';
 END IF;
 IF NEW.estado='en_intervencion' AND OLD.estado IS DISTINCT FROM NEW.estado THEN
  IF NOT (OLD.estado='aceptada' OR (OLD.estado IN ('cerrada','derivada_spm') AND
   coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)))
   OR NOT EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=NEW.id AND tipo='consentimiento') THEN
   RAISE EXCEPTION 'error.estado_no_permite';
  END IF;
 END IF;
 IF OLD.estado='en_intervencion' AND NEW.estado<>OLD.estado AND NOT (NEW.estado='cerrada' AND
  coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)) THEN
  RAISE EXCEPTION 'error.estado_no_permite';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_inicio_intervencion() FROM PUBLIC,etf_app,etf_copia;

CREATE CONSTRAINT TRIGGER comprobar_episodio_actual AFTER INSERT OR UPDATE ON expediente.episodio
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION expediente.comprobar_episodio_actual();
CREATE OR REPLACE FUNCTION expediente.proteger_finalizacion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF NEW.modificado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid
 OR NOT EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=NEW.expediente_id AND e.equipo_id=NEW.equipo_id
   AND e.estado IN ('aceptada','en_intervencion','derivada_spm')) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 IF TG_OP='UPDATE' AND NEW.datos IS DISTINCT FROM OLD.datos AND EXISTS(
 SELECT 1 FROM expediente.propuesta_cierre p WHERE p.expediente_id=NEW.expediente_id AND p.estado='pendiente') THEN
  RAISE EXCEPTION 'error.estado_no_permite';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_finalizacion() FROM PUBLIC,etf_app,etf_copia;
