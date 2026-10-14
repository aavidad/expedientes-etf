SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.peticion_idempotente') IS NULL
 OR to_regclass('expediente.propuesta_cierre') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
CREATE TABLE expediente.finalizacion (
 expediente_id uuid PRIMARY KEY, equipo_id uuid NOT NULL,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 modificado_por uuid NOT NULL REFERENCES identidad.usuario(id),modificado_en timestamptz NOT NULL,
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE TABLE expediente.propuesta_cierre (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 propuesta_por uuid NOT NULL REFERENCES identidad.usuario(id), propuesta_en timestamptz NOT NULL,
 motivo_finalizacion text NOT NULL, catalogo_version_id uuid NOT NULL,
 motivo_acta text NOT NULL,motivo_spaf text NOT NULL,
 estado text NOT NULL DEFAULT 'pendiente' CHECK(estado IN ('pendiente','retirada','confirmada')),
 resuelta_por uuid REFERENCES identidad.usuario(id),resuelta_en timestamptz,
 CHECK((estado='pendiente' AND resuelta_por IS NULL AND resuelta_en IS NULL) OR (estado<>'pendiente' AND resuelta_por IS NOT NULL AND resuelta_en IS NOT NULL)),
 CHECK(estado<>'confirmada' OR resuelta_por<>propuesta_por),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE UNIQUE INDEX propuesta_cierre_pendiente ON expediente.propuesta_cierre(expediente_id) WHERE estado='pendiente';
CREATE INDEX propuesta_cierre_bandeja ON expediente.propuesta_cierre(estado,propuesta_en,id);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['finalizacion','propuesta_cierre'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT ON expediente.%I TO etf_app,etf_copia',tabla);
  EXECUTE format('CREATE POLICY lectura ON expediente.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND (
   ''coordinacion''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) OR
   (''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND
   equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),''''),''{}'')::uuid[]))))',tabla);
  EXECUTE format('CREATE POLICY escritura ON expediente.%I FOR ALL TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND
   ''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND
   equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),''''),''{}'')::uuid[])) WITH CHECK (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND
   ''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND
   equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),''''),''{}'')::uuid[]))',tabla);
 END LOOP;
END $$;
GRANT INSERT,UPDATE ON expediente.finalizacion TO etf_app;
GRANT INSERT ON expediente.propuesta_cierre TO etf_app;
GRANT UPDATE(estado,resuelta_por,resuelta_en) ON expediente.propuesta_cierre TO etf_app;
CREATE POLICY propuesta_version ON expediente.expediente FOR UPDATE TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))
 WITH CHECK(modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY propia_propuesta ON expediente.peticion_idempotente TO etf_app
 USING(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')))
 WITH CHECK(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
-- La política de UPDATE requiere protección de las columnas ya concedidas para aceptar.
-- Un profesional sólo mueve la versión y la marca técnica de modificación.
CREATE FUNCTION expediente.proteger_propuesta_version() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
 AND (to_jsonb(NEW)-ARRAY['version','modificado_por','modificado_en']) IS DISTINCT FROM
     (to_jsonb(OLD)-ARRAY['version','modificado_por','modificado_en']) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_propuesta_version() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_propuesta_version BEFORE UPDATE ON expediente.expediente FOR EACH ROW EXECUTE FUNCTION expediente.proteger_propuesta_version();
CREATE FUNCTION expediente.proteger_propuesta_cierre() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.estado<>'pendiente' OR NEW.propuesta_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 ELSE
  IF OLD.estado<>'pendiente' OR NEW.estado<>'retirada' OR NEW.resuelta_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_propuesta_cierre() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_propuesta_cierre BEFORE INSERT OR UPDATE ON expediente.propuesta_cierre FOR EACH ROW EXECUTE FUNCTION expediente.proteger_propuesta_cierre();

CREATE FUNCTION expediente.proteger_finalizacion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF NEW.modificado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid
 OR NOT EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=NEW.expediente_id AND e.equipo_id=NEW.equipo_id
   AND e.estado IN ('aceptada','en_intervencion')) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 IF TG_OP='UPDATE' AND NEW.datos IS DISTINCT FROM OLD.datos AND EXISTS(
 SELECT 1 FROM expediente.propuesta_cierre p WHERE p.expediente_id=NEW.expediente_id AND p.estado='pendiente') THEN
  RAISE EXCEPTION 'error.estado_no_permite';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_finalizacion() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_finalizacion BEFORE INSERT OR UPDATE ON expediente.finalizacion FOR EACH ROW EXECUTE FUNCTION expediente.proteger_finalizacion();
