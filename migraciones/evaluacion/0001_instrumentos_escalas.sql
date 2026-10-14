SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.adulto') IS NULL OR to_regnamespace('evaluacion') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
CREATE SCHEMA evaluacion AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA evaluacion FROM PUBLIC;
GRANT USAGE ON SCHEMA evaluacion TO etf_app,etf_copia;
CREATE TABLE evaluacion.instrumento_aplicado (
 id uuid PRIMARY KEY, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id),
 CHECK((datos->>'id'=id::text AND datos->>'expediente_id'=expediente_id::text AND datos->>'equipo_id'=equipo_id::text) IS TRUE)
);
CREATE INDEX instrumento_expediente ON evaluacion.instrumento_aplicado(expediente_id,equipo_id);
CREATE TABLE evaluacion.aplicacion_escala (
 id uuid PRIMARY KEY, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 adulto_id uuid, instrumento text NOT NULL, momento text NOT NULL CHECK(momento IN ('derivacion','inicial','intermedia','final')),
 sujeto jsonb NOT NULL, datos jsonb NOT NULL CHECK(jsonb_typeof(datos)='object'),
 version integer NOT NULL CHECK(version>0),
 UNIQUE(id,expediente_id,equipo_id),
 UNIQUE(expediente_id,equipo_id,instrumento,momento,sujeto),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id),
 FOREIGN KEY(adulto_id,expediente_id,equipo_id) REFERENCES expediente.adulto(id,expediente_id,equipo_id),
 CHECK(jsonb_typeof(sujeto)='object'),
 CHECK(((sujeto->>'tipo'='figura_parental' AND adulto_id IS NOT NULL AND sujeto->>'adulto_id'=adulto_id::text AND NOT sujeto ? 'menor_id' AND NOT sujeto ? 'profesional_id')
 OR (sujeto='{"tipo":"familia"}'::jsonb AND adulto_id IS NULL)
 OR (sujeto->>'tipo'='profesional' AND adulto_id IS NULL AND NOT sujeto ? 'menor_id' AND NOT sujeto ? 'adulto_id' AND (sujeto->>'profesional_id')::uuid IS NOT NULL)) IS TRUE),
 CHECK((datos->>'id'=id::text AND datos->>'expediente_id'=expediente_id::text AND datos->>'equipo_id'=equipo_id::text
 AND datos->>'instrumento'=instrumento AND datos->>'momento'=momento AND datos->'sujeto'=sujeto AND (datos->>'version')::integer=version) IS TRUE)
);
CREATE TABLE evaluacion.respuesta (
 aplicacion_id uuid NOT NULL, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 item text NOT NULL CHECK(length(item) BETWEEN 1 AND 128), valor_numerico double precision, valor_codigo text,
 PRIMARY KEY(aplicacion_id,item),
 FOREIGN KEY(aplicacion_id,expediente_id,equipo_id) REFERENCES evaluacion.aplicacion_escala(id,expediente_id,equipo_id),
 CHECK((valor_numerico IS NULL)<>(valor_codigo IS NULL)),
 CHECK(valor_numerico IS NULL OR (valor_numerico > '-Infinity'::float8 AND valor_numerico < 'Infinity'::float8)),
 CHECK(valor_codigo IS NULL OR length(valor_codigo) BETWEEN 1 AND 128)
);
CREATE INDEX respuesta_expediente ON evaluacion.respuesta(expediente_id,equipo_id);
CREATE TABLE evaluacion.peticion (
 usuario_id uuid NOT NULL REFERENCES identidad.usuario(id), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 clave text NOT NULL CHECK(length(clave) BETWEEN 1 AND 128), huella text NOT NULL CHECK(huella ~ '^[a-f0-9]{64}$'), resultado jsonb NOT NULL,
 PRIMARY KEY(usuario_id,expediente_id,clave),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE FUNCTION evaluacion.proteger_aplicacion() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,evaluacion AS $$
BEGIN
 IF NEW.id<>OLD.id OR NEW.expediente_id<>OLD.expediente_id OR NEW.equipo_id<>OLD.equipo_id OR NEW.instrumento<>OLD.instrumento
 OR NEW.momento<>OLD.momento OR NEW.sujeto<>OLD.sujeto OR NEW.adulto_id IS DISTINCT FROM OLD.adulto_id OR NEW.version<>OLD.version+1 THEN
  RAISE EXCEPTION 'evaluacion.peticion_invalida' USING ERRCODE='23514';
 END IF; RETURN NEW;
END $$;
REVOKE ALL ON FUNCTION evaluacion.proteger_aplicacion() FROM PUBLIC;
CREATE TRIGGER proteger_aplicacion BEFORE UPDATE ON evaluacion.aplicacion_escala FOR EACH ROW EXECUTE FUNCTION evaluacion.proteger_aplicacion();
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['instrumento_aplicado','aplicacion_escala','respuesta','peticion'] LOOP
  EXECUTE format('ALTER TABLE evaluacion.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE evaluacion.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON evaluacion.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT ON evaluacion.%I TO etf_app,etf_copia',tabla);
  EXECUTE format('CREATE POLICY propietario ON evaluacion.%I TO etf_propietario USING(true) WITH CHECK(true)',tabla);
  EXECUTE format($p$CREATE POLICY lectura ON evaluacion.%I FOR SELECT TO etf_app USING (
   nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND
   ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR
    ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))))$p$,tabla);
  EXECUTE format($p$CREATE POLICY escritura ON evaluacion.%I FOR ALL TO etf_app USING (
   EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=evaluacion.%I.expediente_id AND e.equipo_id=evaluacion.%I.equipo_id AND e.estado IN ('aceptada','en_intervencion')) AND
   nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND
   ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR
    ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))))
   WITH CHECK (
   EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=evaluacion.%I.expediente_id AND e.equipo_id=evaluacion.%I.equipo_id AND e.estado IN ('aceptada','en_intervencion')) AND
   nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND
   ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR
    ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))))$p$,tabla,tabla,tabla,tabla,tabla);
 END LOOP;
END $$;
-- Las respuestas son el único detalle sustituible; instrumentos e idempotencia
-- sólo admiten altas. No se conceden borrados del agregado.
GRANT INSERT ON evaluacion.instrumento_aplicado,evaluacion.aplicacion_escala,evaluacion.respuesta,evaluacion.peticion TO etf_app;
GRANT UPDATE(datos,version) ON evaluacion.aplicacion_escala TO etf_app;
GRANT DELETE ON evaluacion.respuesta TO etf_app;
CREATE POLICY peticion_propia ON evaluacion.peticion AS RESTRICTIVE TO etf_app
 USING(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid)
 WITH CHECK(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid);
