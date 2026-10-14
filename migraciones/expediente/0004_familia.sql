SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.peticion_idempotente') IS NULL
 OR to_regclass('expediente.adulto') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
ALTER TABLE expediente.expediente ADD COLUMN familia jsonb NOT NULL DEFAULT '{}'::jsonb
 CHECK(jsonb_typeof(familia)='object');
GRANT UPDATE(familia) ON expediente.expediente TO etf_app;
-- La política sólo abre expedientes activos del equipo de la sesión.
CREATE POLICY equipo_ficha ON expediente.expediente FOR UPDATE TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
  AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])) AND estado IN ('aceptada','en_intervencion'))
 WITH CHECK(nullif(current_setting('app.usuario_id',true),'')::uuid=modificado_por AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
  AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])) AND estado IN ('aceptada','en_intervencion'));
-- Los permisos de columna previos sirven también a Coordinación. Este filtro
-- impide que la política nueva permita alterar aceptación o estado al equipo.
CREATE FUNCTION expediente.proteger_ficha() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
BEGIN
 -- La restricción afecta a las escrituras del rol de aplicación. El migrador
 -- y el instalador ya tienen permisos propios para cargas y mantenimiento.
 IF current_user <> 'etf_app' THEN
  RETURN NEW;
 END IF;
 IF NOT coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
 AND ((to_jsonb(NEW)-ARRAY['familia','version','modificado_por','modificado_en'])
 IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['familia','version','modificado_por','modificado_en'])) THEN
  RETURN NULL;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_ficha() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_ficha BEFORE UPDATE ON expediente.expediente FOR EACH ROW EXECUTE FUNCTION expediente.proteger_ficha();
CREATE TABLE expediente.adulto (
 id uuid PRIMARY KEY, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 papel text NOT NULL CHECK(length(papel) BETWEEN 1 AND 64),
 sexo text NOT NULL CHECK(length(sexo) BETWEEN 1 AND 64),
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 UNIQUE(id,expediente_id,equipo_id),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE INDEX adulto_expediente ON expediente.adulto(expediente_id);
-- Revisiones de sólo adición. Los intervalos se derivan de fechas efectivas
-- distintas; corregir la misma fecha añade revisión y conserva la sustituida.
CREATE TABLE expediente.familia_revision (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 version integer NOT NULL CHECK(version>0), vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 valor jsonb NOT NULL CHECK(jsonb_typeof(valor)='object'
  AND valor-ARRAY['composicion_familiar','familia_numerosa','vivienda','situacion_economica','ingresos_anuales_centimos','tramo_smi']='{}'::jsonb
  AND (NOT valor ? 'familia_numerosa' OR jsonb_typeof(valor->'familia_numerosa') IN ('boolean','null'))
  AND (NOT valor ? 'ingresos_anuales_centimos' OR jsonb_typeof(valor->'ingresos_anuales_centimos')='null'
    OR (jsonb_typeof(valor->'ingresos_anuales_centimos')='number' AND (valor->>'ingresos_anuales_centimos')::numeric>=0))
  AND (coalesce(jsonb_typeof(valor->'ingresos_anuales_centimos'),'null')='null'
    OR coalesce(jsonb_typeof(valor->'tramo_smi'),'null')='null')),
 registrado_por uuid NOT NULL REFERENCES identidad.usuario(id), registrado_en timestamptz NOT NULL,
 UNIQUE(expediente_id,version), FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE TABLE expediente.adultos_revision (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 version integer NOT NULL CHECK(version>0), vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 registrado_por uuid NOT NULL REFERENCES identidad.usuario(id), registrado_en timestamptz NOT NULL,
 UNIQUE(id,expediente_id,equipo_id), UNIQUE(expediente_id,version),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE TABLE expediente.adulto_revision (
 revision_id uuid NOT NULL, adulto_id uuid NOT NULL, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 orden integer NOT NULL CHECK(orden BETWEEN 1 AND 100), papel text NOT NULL, sexo text NOT NULL,
 fecha_nacimiento date CHECK(isfinite(fecha_nacimiento)), edad_declarada integer CHECK(edad_declarada BETWEEN 18 AND 130),
 nivel_educativo text, situacion_laboral text, es_tutor boolean,
 CHECK(fecha_nacimiento IS NULL OR edad_declarada IS NULL),
 PRIMARY KEY(revision_id,adulto_id), UNIQUE(revision_id,orden),
 FOREIGN KEY(revision_id,expediente_id,equipo_id) REFERENCES expediente.adultos_revision(id,expediente_id,equipo_id),
 FOREIGN KEY(adulto_id,expediente_id,equipo_id) REFERENCES expediente.adulto(id,expediente_id,equipo_id)
);
CREATE INDEX familia_historia ON expediente.familia_revision(expediente_id,vigente_desde DESC,version DESC);
CREATE INDEX adultos_historia ON expediente.adultos_revision(expediente_id,vigente_desde DESC,version DESC);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['adulto','familia_revision','adultos_revision','adulto_revision'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT,INSERT ON expediente.%I TO etf_app',tabla);
  EXECUTE format('GRANT SELECT ON expediente.%I TO etf_copia',tabla);
  EXECUTE format('CREATE POLICY lectura ON expediente.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND (''coordinacion''=ANY(string_to_array(current_setting(''app.perfiles'',true),'',''))
   OR (''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),'''')::uuid[],''{}''::uuid[])))))',tabla);
  EXECUTE format('CREATE POLICY escritura ON expediente.%I FOR INSERT TO etf_app WITH CHECK (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND ''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),'''')::uuid[],''{}''::uuid[]))
   AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=%1$I.expediente_id AND e.equipo_id=%1$I.equipo_id
   AND e.estado IN (''aceptada'',''en_intervencion'')))',tabla);
 END LOOP;
END $$;
GRANT UPDATE(papel,sexo,datos) ON expediente.adulto TO etf_app;
CREATE POLICY modificar ON expediente.adulto FOR UPDATE TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))
 WITH CHECK(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[]))
 AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=adulto.expediente_id AND e.equipo_id=adulto.equipo_id AND e.estado IN ('aceptada','en_intervencion')));
CREATE POLICY propia_ficha ON expediente.peticion_idempotente TO etf_app
 USING(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND cardinality(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[]))=1)
 WITH CHECK(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND cardinality(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[]))=1);
-- Todas las vistas conservan RLS del invocador y exponen únicamente estadística.
CREATE VIEW expediente.consulta_familia_vigencia WITH(security_invoker=true) AS
 SELECT id,expediente_id,equipo_id,version,valor,vigente_desde,
 lead(vigente_desde) OVER(PARTITION BY expediente_id ORDER BY vigente_desde) AS vigente_hasta
 FROM (SELECT DISTINCT ON(expediente_id,vigente_desde) * FROM expediente.familia_revision
 ORDER BY expediente_id,vigente_desde,version DESC) r;
CREATE VIEW expediente.consulta_adultos_vigencia WITH(security_invoker=true) AS
 SELECT id,expediente_id,equipo_id,version,vigente_desde,
 lead(vigente_desde) OVER(PARTITION BY expediente_id ORDER BY vigente_desde) AS vigente_hasta
 FROM (SELECT DISTINCT ON(expediente_id,vigente_desde) * FROM expediente.adultos_revision
 ORDER BY expediente_id,vigente_desde,version DESC) r;
CREATE VIEW expediente.consulta_adulto WITH(security_invoker=true) AS
 SELECT a.adulto_id AS id,a.expediente_id,a.equipo_id,a.orden,a.papel,a.sexo,a.fecha_nacimiento,a.edad_declarada,
 a.nivel_educativo,a.situacion_laboral,a.es_tutor,r.vigente_desde,r.vigente_hasta,r.version
 FROM expediente.adulto_revision a JOIN expediente.consulta_adultos_vigencia r ON r.id=a.revision_id;
REVOKE ALL ON expediente.consulta_familia_vigencia,expediente.consulta_adultos_vigencia,expediente.consulta_adulto FROM PUBLIC;
GRANT SELECT ON expediente.consulta_familia_vigencia,expediente.consulta_adultos_vigencia,expediente.consulta_adulto TO etf_app,etf_copia;
