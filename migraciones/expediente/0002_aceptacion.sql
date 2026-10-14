SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('expediente.expediente') IS NULL OR to_regclass('catalogo.regla_numeracion') IS NULL
 OR to_regclass('expediente.contador_expediente') IS NOT NULL
 OR EXISTS (SELECT 1 FROM expediente.expediente WHERE estado<>'derivacion_recibida' OR numero IS NOT NULL OR equipo_id IS NOT NULL OR municipio_id IS NOT NULL OR version<>1) THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;
ALTER TABLE expediente.expediente
 DROP CONSTRAINT expediente_numero_check, DROP CONSTRAINT expediente_equipo_id_check,
 DROP CONSTRAINT expediente_municipio_id_check, DROP CONSTRAINT expediente_estado_check,
 DROP CONSTRAINT expediente_version_check,
 ADD COLUMN centro_ss_id uuid REFERENCES catalogo.centro_ss(id),
 ADD COLUMN anio_numero integer CHECK (anio_numero BETWEEN 1 AND 9999),
 ADD COLUMN correlativo integer CHECK (correlativo > 0),
 ADD COLUMN regla_numeracion_id uuid REFERENCES catalogo.regla_numeracion(version_id),
 ADD COLUMN fecha_aceptacion date CHECK (isfinite(fecha_aceptacion)),
 ADD COLUMN subprograma text CHECK (subprograma IN ('RIESGO_SOCIAL','REUNIFICACION')),
 ADD COLUMN profesional_referencia_puesto text CHECK (profesional_referencia_puesto IN ('TS','ES','PS')),
 ADD COLUMN profesional_referencia_id uuid REFERENCES identidad.usuario(id),
 ADD COLUMN descripcion_situacion text CHECK (length(btrim(descripcion_situacion))>0 AND octet_length(descripcion_situacion)<=20000),
 ADD COLUMN fase_intervencion text CHECK (fase_intervencion='recepcion'),
 ADD COLUMN modificado_por uuid REFERENCES identidad.usuario(id),
 ADD COLUMN modificado_en timestamptz,
 ADD CONSTRAINT expediente_estado_check CHECK (estado IN ('derivacion_recibida','aceptada')),
 ADD CONSTRAINT expediente_version_check CHECK (version>0),
 ADD CONSTRAINT expediente_pertenencia UNIQUE (id,equipo_id),
 ADD CONSTRAINT expediente_correlativo UNIQUE (equipo_id,anio_numero,correlativo),
 ADD CONSTRAINT expediente_aceptacion_completa CHECK (
  (estado='derivacion_recibida' AND numero IS NULL AND equipo_id IS NULL AND municipio_id IS NULL AND fecha_aceptacion IS NULL
   AND anio_numero IS NULL AND correlativo IS NULL AND regla_numeracion_id IS NULL AND centro_ss_id IS NULL
   AND subprograma IS NULL AND profesional_referencia_puesto IS NULL AND profesional_referencia_id IS NULL
   AND descripcion_situacion IS NULL AND fase_intervencion IS NULL)
  OR (estado='aceptada' AND numero IS NOT NULL AND equipo_id IS NOT NULL AND municipio_id IS NOT NULL AND centro_ss_id IS NOT NULL
   AND fecha_aceptacion IS NOT NULL AND anio_numero IS NOT NULL AND anio_numero=extract(year FROM fecha_aceptacion)
   AND correlativo IS NOT NULL AND regla_numeracion_id IS NOT NULL AND subprograma IS NOT NULL
   AND profesional_referencia_puesto IS NOT NULL AND descripcion_situacion IS NOT NULL AND fase_intervencion IS NOT NULL
   AND modificado_por IS NOT NULL AND modificado_en IS NOT NULL)
 );
CREATE INDEX expediente_equipo_estado ON expediente.expediente(equipo_id,estado);

CREATE TABLE expediente.contador_expediente (
 equipo_id uuid NOT NULL REFERENCES catalogo.equipo(id), anio integer NOT NULL CHECK (anio BETWEEN 1 AND 9999),
 ultimo integer NOT NULL CHECK (ultimo>0), PRIMARY KEY (equipo_id,anio)
);
CREATE TABLE expediente.episodio (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 apertura_fecha date NOT NULL CHECK (isfinite(apertura_fecha)), creado_por uuid NOT NULL REFERENCES identidad.usuario(id),
 creado_en timestamptz NOT NULL, UNIQUE(id,expediente_id,equipo_id),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE TABLE expediente.hito (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL, episodio_id uuid NOT NULL,
 tipo text NOT NULL CHECK(tipo='aceptacion'), fecha date NOT NULL CHECK(isfinite(fecha)),
 origen text NOT NULL CHECK(origen='documento_externo'),
 confirmado_por uuid NOT NULL REFERENCES identidad.usuario(id), confirmado_en timestamptz NOT NULL,
 UNIQUE(expediente_id,tipo), FOREIGN KEY(episodio_id,expediente_id,equipo_id) REFERENCES expediente.episodio(id,expediente_id,equipo_id)
);
CREATE TABLE expediente.estado_cambio (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 estado_anterior text NOT NULL CHECK(estado_anterior='derivacion_recibida'), estado_nuevo text NOT NULL CHECK(estado_nuevo='aceptada'),
 cambiado_por uuid NOT NULL REFERENCES identidad.usuario(id), cambiado_en timestamptz NOT NULL,
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
-- Primera parte del menor: identidad estable por orden; F2 ampliará su ficha.
CREATE TABLE expediente.menor (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 orden integer NOT NULL CHECK(orden BETWEEN 1 AND 15), sexo text NOT NULL CHECK(sexo IN ('H','M')),
 UNIQUE(expediente_id,orden), UNIQUE(id,expediente_id,equipo_id),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE TABLE expediente.dato_estadistico_vigencia (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 entidad_tipo text NOT NULL CHECK(entidad_tipo IN ('expediente','menor')), entidad_id uuid NOT NULL,
 menor_id uuid, atributo text NOT NULL CHECK(atributo IN ('sexo','subprograma')), valor jsonb NOT NULL,
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)), registrado_en timestamptz NOT NULL,
 registrado_por uuid NOT NULL REFERENCES identidad.usuario(id),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id),
 FOREIGN KEY(menor_id,expediente_id,equipo_id) REFERENCES expediente.menor(id,expediente_id,equipo_id),
 CHECK ((entidad_tipo='menor' AND menor_id IS NOT NULL AND entidad_id=menor_id AND atributo='sexo' AND valor IN ('"H"'::jsonb,'"M"'::jsonb))
  OR (entidad_tipo='expediente' AND menor_id IS NULL AND entidad_id=expediente_id AND atributo='subprograma' AND valor IN ('"RIESGO_SOCIAL"'::jsonb,'"REUNIFICACION"'::jsonb)))
);
CREATE INDEX dato_vigencia_expediente ON expediente.dato_estadistico_vigencia(expediente_id,vigente_desde);
CREATE TABLE expediente.peticion_idempotente (
 usuario_id uuid NOT NULL REFERENCES identidad.usuario(id), clave text NOT NULL CHECK(clave ~ '^[A-Za-z0-9._:-]{1,128}$'),
 ruta text NOT NULL, huella_peticion text NOT NULL CHECK(huella_peticion ~ '^[a-f0-9]{64}$'),
 respuesta jsonb NOT NULL CHECK(jsonb_typeof(respuesta)='object'), creada_en timestamptz NOT NULL,
 PRIMARY KEY(usuario_id,clave)
);
CREATE INDEX peticion_caducidad ON expediente.peticion_idempotente(creada_en);

-- La asignación confirmada y el número no se pueden mover ni reutilizar.
CREATE FUNCTION expediente.proteger_asignacion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
BEGIN
 IF OLD.equipo_id IS NOT NULL AND (NEW.equipo_id IS DISTINCT FROM OLD.equipo_id
  OR NEW.numero IS DISTINCT FROM OLD.numero OR NEW.anio_numero IS DISTINCT FROM OLD.anio_numero
  OR NEW.correlativo IS DISTINCT FROM OLD.correlativo OR NEW.regla_numeracion_id IS DISTINCT FROM OLD.regla_numeracion_id) THEN
  RAISE EXCEPTION 'expediente.asignacion_inmutable';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_asignacion() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_asignacion BEFORE UPDATE ON expediente.expediente FOR EACH ROW EXECUTE FUNCTION expediente.proteger_asignacion();
GRANT UPDATE (numero,equipo_id,municipio_id,centro_ss_id,anio_numero,correlativo,regla_numeracion_id,fecha_aceptacion,
 subprograma,profesional_referencia_puesto,profesional_referencia_id,descripcion_situacion,fase_intervencion,estado,version,modificado_por,modificado_en)
 ON expediente.expediente TO etf_app;
CREATE POLICY aceptar ON expediente.expediente FOR UPDATE TO etf_app
 USING (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')))
 WITH CHECK (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid);
CREATE POLICY equipo_lectura ON expediente.expediente FOR SELECT TO etf_app
 USING (nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
CREATE POLICY equipo_lectura ON expediente.lectura_anexo FOR SELECT TO etf_app
 USING (EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=expediente_id));
CREATE POLICY equipo_lectura ON expediente.derivacion_datos FOR SELECT TO etf_app
 USING (EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=expediente_id));
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['contador_expediente','episodio','hito','estado_cambio','menor','dato_estadistico_vigencia','peticion_idempotente'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT,INSERT ON expediente.%I TO etf_app',tabla);
  EXECUTE format('GRANT SELECT ON expediente.%I TO etf_copia',tabla);
  EXECUTE format('CREATE POLICY coordinacion ON expediente.%I TO etf_app USING (nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND ''coordinacion''=ANY(string_to_array(current_setting(''app.perfiles'',true),'',''))) WITH CHECK (nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND ''coordinacion''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')))',tabla);
 END LOOP;
 FOREACH tabla IN ARRAY ARRAY['episodio','hito','estado_cambio','menor','dato_estadistico_vigencia'] LOOP
  EXECUTE format('CREATE POLICY equipo_lectura ON expediente.%I FOR SELECT TO etf_app USING (nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND ''profesional_etf''=ANY(string_to_array(current_setting(''app.perfiles'',true),'','')) AND equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),''''),''{}'')::uuid[]))',tabla);
 END LOOP;
END $$;
GRANT UPDATE(ultimo) ON expediente.contador_expediente TO etf_app;
GRANT DELETE ON expediente.peticion_idempotente TO etf_app;
DROP POLICY coordinacion ON expediente.peticion_idempotente;
CREATE POLICY propia ON expediente.peticion_idempotente TO etf_app
 USING(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')))
 WITH CHECK(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND 'coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')));
