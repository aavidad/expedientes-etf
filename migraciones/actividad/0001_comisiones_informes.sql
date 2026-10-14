DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('catalogo.equipo') IS NULL OR to_regprocedure('auditoria.registrar(timestamp with time zone,text,text,uuid,uuid,text)') IS NULL
 OR to_regnamespace('actividad') IS NOT NULL THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
SET LOCAL lock_timeout = '3s';
SET LOCAL statement_timeout = '30s';
CREATE SCHEMA actividad AUTHORIZATION etf_propietario;
REVOKE ALL ON SCHEMA actividad FROM PUBLIC;
GRANT USAGE ON SCHEMA actividad TO etf_app, etf_copia;

-- Catálogo inicial inmutable: sólo códigos y claves, las etiquetas viven en idiomas.
CREATE TABLE actividad.organismo (
 codigo text PRIMARY KEY,
 clave_texto text NOT NULL UNIQUE,
 admite_texto boolean NOT NULL,
 orden smallint NOT NULL UNIQUE,
 version integer NOT NULL CHECK (version = 1)
);
INSERT INTO actividad.organismo VALUES
 ('JUZGADOS','actividad.organismo.JUZGADOS',false,1,1),
 ('FISCALIA','actividad.organismo.FISCALIA',false,2,1),
 ('SPAF','actividad.organismo.SPAF',false,3,1),
 ('SPM','actividad.organismo.SPM',false,4,1),
 ('CENTROS_EDUCATIVOS','actividad.organismo.CENTROS_EDUCATIVOS',false,5,1),
 ('CENTROS_SANITARIOS','actividad.organismo.CENTROS_SANITARIOS',false,6,1),
 ('SERVICIOS_SOCIALES','actividad.organismo.SERVICIOS_SOCIALES',false,7,1),
 ('OTROS','actividad.organismo.OTROS',true,8,1);

CREATE TABLE actividad.comision_tecnica (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 equipo_id uuid NOT NULL REFERENCES catalogo.equipo(id),
 fecha date NOT NULL CHECK (isfinite(fecha) AND fecha >= DATE '0001-01-01' AND fecha < DATE '9999-01-01'),
 observaciones text NOT NULL DEFAULT '' CHECK (char_length(observaciones) <= 4000),
 version integer NOT NULL DEFAULT 1 CHECK (version > 0),
 retirada boolean NOT NULL DEFAULT false
);
CREATE TABLE actividad.informe_tecnico (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 equipo_id uuid NOT NULL REFERENCES catalogo.equipo(id),
 fecha date NOT NULL CHECK (isfinite(fecha) AND fecha >= DATE '0001-01-01' AND fecha < DATE '9999-01-01'),
 organismo text NOT NULL REFERENCES actividad.organismo(codigo),
 organismo_otros text NOT NULL DEFAULT '' CHECK (char_length(organismo_otros) <= 500),
 version integer NOT NULL DEFAULT 1 CHECK (version > 0),
 retirada boolean NOT NULL DEFAULT false,
 CHECK ((organismo = 'OTROS' AND organismo_otros ~ '[^[:space:]]') OR (organismo <> 'OTROS' AND organismo_otros = ''))
);
CREATE INDEX comision_equipo_fecha ON actividad.comision_tecnica(equipo_id,fecha) WHERE NOT retirada;
CREATE INDEX informe_equipo_fecha ON actividad.informe_tecnico(equipo_id,fecha) WHERE NOT retirada;
CREATE INDEX comision_fecha ON actividad.comision_tecnica(fecha) WHERE NOT retirada;
CREATE INDEX informe_fecha ON actividad.informe_tecnico(fecha) WHERE NOT retirada;

CREATE FUNCTION actividad.proteger_registro() RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog,actividad AS $$
BEGIN
 IF TG_OP = 'UPDATE' THEN
  IF NEW.id <> OLD.id OR NEW.equipo_id <> OLD.equipo_id OR NEW.version <> OLD.version + 1 OR OLD.retirada THEN
   RAISE EXCEPTION 'actividad.datos_invalidos' USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
 END IF;
 RAISE EXCEPTION 'actividad.datos_invalidos' USING ERRCODE = '23514';
END $$;
CREATE FUNCTION actividad.proteger_organismos() RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog,actividad AS $$
BEGIN RAISE EXCEPTION 'actividad.catalogo_inmutable' USING ERRCODE = '23514'; END $$;
REVOKE ALL ON FUNCTION actividad.proteger_registro(), actividad.proteger_organismos() FROM PUBLIC;
CREATE TRIGGER organismos_inmutables BEFORE INSERT OR UPDATE OR DELETE OR TRUNCATE ON actividad.organismo
 FOR EACH STATEMENT EXECUTE FUNCTION actividad.proteger_organismos();

DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['organismo','comision_tecnica','informe_tecnico'] LOOP
  EXECUTE format('ALTER TABLE actividad.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE actividad.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON actividad.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('CREATE POLICY propietario ON actividad.%I TO etf_propietario USING (true) WITH CHECK (true)',tabla);
  EXECUTE format('GRANT SELECT ON actividad.%I TO etf_app,etf_copia',tabla);
 END LOOP;
 FOREACH tabla IN ARRAY ARRAY['comision_tecnica','informe_tecnico'] LOOP
  EXECUTE format($politica$
   CREATE POLICY lectura ON actividad.%I FOR SELECT TO etf_app USING (
    nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND
    ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR
     'sistema'=ANY(string_to_array(current_setting('app.perfiles',true),',')) OR
     ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
      equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))))
   $politica$,tabla);
  EXECUTE format($politica$
   CREATE POLICY alta ON actividad.%I FOR INSERT TO etf_app WITH CHECK (
    nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND
    'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
    equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])) AND version=1 AND NOT retirada)
   $politica$,tabla);
  EXECUTE format($politica$
   CREATE POLICY correccion ON actividad.%I FOR UPDATE TO etf_app USING (
    nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND
    'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
    equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[]))) WITH CHECK (
    nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND
    'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND
    equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))
   $politica$,tabla);
  EXECUTE format('GRANT INSERT, UPDATE ON actividad.%I TO etf_app',tabla);
  EXECUTE format('CREATE TRIGGER registro_inmutable BEFORE UPDATE OR DELETE ON actividad.%I FOR EACH ROW EXECUTE FUNCTION actividad.proteger_registro()',tabla);
  EXECUTE format('CREATE TRIGGER impedir_truncado BEFORE TRUNCATE ON actividad.%I FOR EACH STATEMENT EXECUTE FUNCTION actividad.proteger_registro()',tabla);
 END LOOP;
END $$;
CREATE POLICY lectura ON actividad.organismo FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND
 string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','profesional_etf','sistema']);
