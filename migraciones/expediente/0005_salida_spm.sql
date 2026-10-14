SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.expediente') IS NULL OR to_regclass('expediente.salida_spm') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- La salida tiene versión propia. No modifica estado ni versión de la ficha.
CREATE TABLE expediente.salida_spm (
 id uuid PRIMARY KEY, expediente_id uuid NOT NULL UNIQUE, equipo_id uuid NOT NULL,
 version integer NOT NULL CHECK(version>0),
 motivo text NOT NULL CHECK(length(btrim(motivo)) BETWEEN 1 AND 4000),
 propuesta_medida text NOT NULL CHECK(length(btrim(propuesta_medida)) BETWEEN 1 AND 4000),
 fecha_derivacion date CHECK(isfinite(fecha_derivacion)),
 respuesta text NOT NULL CHECK(respuesta IN ('a_la_espera','inicio_desamparo_medida','medida_no_necesaria')),
 fecha_respuesta date CHECK(isfinite(fecha_respuesta)),
 creado_por uuid NOT NULL REFERENCES identidad.usuario(id), creado_en timestamptz NOT NULL,
 modificado_por uuid NOT NULL REFERENCES identidad.usuario(id), modificado_en timestamptz NOT NULL,
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id),
 UNIQUE(id,expediente_id,equipo_id),
 CHECK ((respuesta='a_la_espera' AND fecha_respuesta IS NULL)
 OR (respuesta<>'a_la_espera' AND fecha_respuesta IS NOT NULL AND fecha_derivacion IS NOT NULL AND fecha_respuesta>=fecha_derivacion))
);
CREATE TABLE expediente.spm_forzado (
 id uuid PRIMARY KEY, salida_id uuid NOT NULL, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 version_salida integer NOT NULL CHECK(version_salida>0),
 motivo text NOT NULL CHECK(length(btrim(motivo)) BETWEEN 1 AND 4000),
 registrado_por uuid NOT NULL REFERENCES identidad.usuario(id), registrado_en timestamptz NOT NULL,
 FOREIGN KEY(salida_id,expediente_id,equipo_id) REFERENCES expediente.salida_spm(id,expediente_id,equipo_id),
 UNIQUE(salida_id,version_salida)
);
CREATE INDEX spm_forzado_salida ON expediente.spm_forzado(salida_id,version_salida);
CREATE TABLE expediente.peticion_spm (
 usuario_id uuid NOT NULL REFERENCES identidad.usuario(id), clave text NOT NULL CHECK(clave ~ '^[A-Za-z0-9._:-]{1,128}$'),
 ruta text NOT NULL, huella_peticion text NOT NULL CHECK(huella_peticion ~ '^[0-9a-f]{64}$'),
 respuesta jsonb NOT NULL, creada_en timestamptz NOT NULL, PRIMARY KEY(usuario_id,clave)
);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['salida_spm','spm_forzado','peticion_spm'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT,INSERT ON expediente.%I TO etf_app',tabla);
  EXECUTE format('GRANT SELECT ON expediente.%I TO etf_copia',tabla);
  EXECUTE format('CREATE POLICY propietario ON expediente.%I TO etf_propietario USING(true) WITH CHECK(true)',tabla);
 END LOOP;
 FOREACH tabla IN ARRAY ARRAY['salida_spm','spm_forzado'] LOOP
  EXECUTE format('CREATE POLICY lectura ON expediente.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND (
   ''coordinacion''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) OR (
   ''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND
   equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),''''),''{}'')::uuid[]))))',tabla);
 END LOOP;
END $$;
CREATE POLICY escritura ON expediente.salida_spm TO etf_app
 USING (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND (
 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR (
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))))
 WITH CHECK (modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND (
 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR (
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
GRANT UPDATE(version,fecha_derivacion,respuesta,fecha_respuesta,modificado_por,modificado_en) ON expediente.salida_spm TO etf_app;
CREATE POLICY registrar ON expediente.spm_forzado FOR INSERT TO etf_app WITH CHECK (
 registrado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND EXISTS (
 SELECT 1 FROM expediente.salida_spm s WHERE s.id=salida_id
 AND s.respuesta='a_la_espera' AND s.version=version_salida));
CREATE POLICY propia ON expediente.peticion_spm TO etf_app
 USING(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','profesional_etf'])
 WITH CHECK(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','profesional_etf']);
GRANT DELETE ON expediente.peticion_spm TO etf_app;
CREATE FUNCTION expediente.proteger_salida_spm() RETURNS trigger
 LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.version<>1 OR NEW.respuesta<>'a_la_espera' OR NEW.fecha_derivacion IS NOT NULL
  OR NEW.fecha_respuesta IS NOT NULL OR NEW.creado_por<>NEW.modificado_por THEN
   RAISE EXCEPTION 'expediente.spm_peticion_invalida'; END IF;
 ELSE
  IF NEW.id<>OLD.id OR NEW.expediente_id<>OLD.expediente_id OR NEW.equipo_id<>OLD.equipo_id
  OR NEW.motivo<>OLD.motivo OR NEW.propuesta_medida<>OLD.propuesta_medida
  OR NEW.creado_por<>OLD.creado_por OR NEW.creado_en<>OLD.creado_en OR NEW.version<>OLD.version+1
  OR (OLD.fecha_derivacion IS NOT NULL AND NEW.fecha_derivacion IS DISTINCT FROM OLD.fecha_derivacion) THEN
   RAISE EXCEPTION 'expediente.spm_peticion_invalida'; END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_salida_spm() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_salida_spm BEFORE INSERT OR UPDATE ON expediente.salida_spm FOR EACH ROW EXECUTE FUNCTION expediente.proteger_salida_spm();

-- SELECT FOR SHARE aplica también políticas de UPDATE. Este puerto permite
-- bloquear una ficha propia sin conceder al profesional edición de aceptación.
CREATE POLICY propietario_spm ON expediente.expediente TO etf_propietario USING(true) WITH CHECK(true);
CREATE FUNCTION expediente.bloquear_lectura_spm(recurso uuid) RETURNS integer
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
 DECLARE v integer;
 BEGIN
 IF nullif(current_setting('app.usuario_id',true),'')::uuid IS NULL THEN
 RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501'; END IF;
 SELECT version INTO v FROM expediente.expediente e WHERE e.id=recurso AND (
 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR (
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 e.equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))) FOR SHARE;
 RETURN v;
 END $$;
REVOKE EXECUTE ON FUNCTION expediente.bloquear_lectura_spm(uuid) FROM PUBLIC,etf_app,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.bloquear_lectura_spm(uuid) TO etf_app;

CREATE TABLE expediente.spm_historia (
 salida_id uuid NOT NULL, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 version integer NOT NULL CHECK(version>0), fecha_derivacion date, respuesta text NOT NULL,
 fecha_respuesta date, registrado_por uuid NOT NULL REFERENCES identidad.usuario(id), registrado_en timestamptz NOT NULL,
 PRIMARY KEY(salida_id,version),
 FOREIGN KEY(salida_id,expediente_id,equipo_id) REFERENCES expediente.salida_spm(id,expediente_id,equipo_id),
 CHECK(respuesta IN ('a_la_espera','inicio_desamparo_medida','medida_no_necesaria'))
);
ALTER TABLE expediente.spm_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE expediente.spm_historia FORCE ROW LEVEL SECURITY;
REVOKE ALL ON expediente.spm_historia FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.spm_historia TO etf_app,etf_copia;
CREATE POLICY propietario ON expediente.spm_historia TO etf_propietario USING(true) WITH CHECK(true);
CREATE POLICY lectura ON expediente.spm_historia FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND (
 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR (
 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
 equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
-- Sólo el disparador añade historia desde la fila cuya escritura ya pasó RLS.
CREATE FUNCTION expediente.historiar_spm() RETURNS trigger
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF session_user='etf_app' THEN
  IF NEW.modificado_por IS DISTINCT FROM nullif(current_setting('app.usuario_id',true),'')::uuid
  OR NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR (
   'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
   NEW.equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])),false) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 END IF;

 INSERT INTO expediente.spm_historia VALUES(NEW.id,NEW.expediente_id,NEW.equipo_id,NEW.version,
 NEW.fecha_derivacion,NEW.respuesta,NEW.fecha_respuesta,NEW.modificado_por,NEW.modificado_en);
 RETURN NEW;
 END $$;
REVOKE EXECUTE ON FUNCTION expediente.historiar_spm() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER historiar_spm AFTER INSERT OR UPDATE ON expediente.salida_spm FOR EACH ROW EXECUTE FUNCTION expediente.historiar_spm();
