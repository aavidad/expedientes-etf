SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.episodio') IS NULL
 OR to_regclass('expediente.salida_servicio') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Especialización del Anexo 6, independiente de las tablas futuras del SPM.
CREATE TABLE expediente.salida_servicio (
 id uuid PRIMARY KEY, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 tipo text NOT NULL DEFAULT 'otro_servicio' CHECK(tipo='otro_servicio'),
 servicio_destino text NOT NULL CHECK(servicio_destino ~ '^[A-Z][A-Z0-9_]{0,63}$'),
 catalogo_id uuid NOT NULL REFERENCES catalogo.version(id),
 motivo text NOT NULL CHECK(length(btrim(motivo))>0 AND octet_length(motivo)<=20000),
 fecha date CHECK(isfinite(fecha)), estado text NOT NULL DEFAULT 'borrador' CHECK(estado IN ('borrador','emitida')),
 version integer NOT NULL DEFAULT 1 CHECK(version>0),
 creada_por uuid NOT NULL REFERENCES identidad.usuario(id), creada_en timestamptz NOT NULL,
 confirmada_por uuid REFERENCES identidad.usuario(id), confirmada_en timestamptz,
 UNIQUE(id,expediente_id,equipo_id),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id),
 CHECK((estado='borrador' AND fecha IS NULL AND confirmada_por IS NULL AND confirmada_en IS NULL AND version=1)
 OR (estado='emitida' AND fecha IS NOT NULL AND confirmada_por IS NOT NULL AND confirmada_en IS NOT NULL AND version=2))
);
CREATE INDEX salida_servicio_expediente ON expediente.salida_servicio(expediente_id,id);

-- Un hito por salida: admite varias derivaciones del mismo expediente.
CREATE TABLE expediente.salida_servicio_hito (
 salida_id uuid PRIMARY KEY, expediente_id uuid NOT NULL, equipo_id uuid NOT NULL,
 tipo text NOT NULL CHECK(tipo='derivacion_otro_servicio'), fecha date NOT NULL CHECK(isfinite(fecha)),
 origen text NOT NULL CHECK(origen='documento_externo'),
 confirmado_por uuid NOT NULL REFERENCES identidad.usuario(id), confirmado_en timestamptz NOT NULL,
 FOREIGN KEY(salida_id,expediente_id,equipo_id) REFERENCES expediente.salida_servicio(id,expediente_id,equipo_id)
);
CREATE TABLE expediente.salida_servicio_peticion (
 usuario_id uuid NOT NULL REFERENCES identidad.usuario(id), clave text NOT NULL CHECK(clave ~ '^[A-Za-z0-9._:-]{1,128}$'),
 equipo_id uuid NOT NULL, expediente_id uuid NOT NULL, ruta text NOT NULL,
 huella text NOT NULL CHECK(huella ~ '^[a-f0-9]{64}$'), respuesta jsonb NOT NULL,
 creada_en timestamptz NOT NULL, PRIMARY KEY(usuario_id,clave),
 FOREIGN KEY(expediente_id,equipo_id) REFERENCES expediente.expediente(id,equipo_id)
);
CREATE INDEX salida_servicio_peticion_caducidad ON expediente.salida_servicio_peticion(creada_en);

DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['salida_servicio','salida_servicio_hito','salida_servicio_peticion'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('GRANT SELECT ON expediente.%I TO etf_app,etf_copia',tabla);
  EXECUTE format('CREATE POLICY lectura ON expediente.%I FOR SELECT TO etf_app USING (
   nullif(current_setting(''app.usuario_id'',true),'''')::uuid IS NOT NULL AND (''coordinacion''=ANY(coalesce(string_to_array(current_setting(''app.perfiles'',true),'',''),''{}''::text[]))
   OR (''profesional_etf''=ANY(coalesce(string_to_array(current_setting(''app.perfiles'',true),'',''),''{}''::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting(''app.equipos'',true),''''),''{}'')::uuid[]))))',tabla);
 END LOOP;
END $$;
GRANT INSERT ON expediente.salida_servicio,expediente.salida_servicio_hito,expediente.salida_servicio_peticion TO etf_app;
GRANT UPDATE(fecha,estado,version,confirmada_por,confirmada_en) ON expediente.salida_servicio TO etf_app;
GRANT DELETE ON expediente.salida_servicio_peticion TO etf_app;
CREATE POLICY crear ON expediente.salida_servicio FOR INSERT TO etf_app WITH CHECK(
 creada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND estado='borrador' AND
 ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
CREATE POLICY confirmar ON expediente.salida_servicio FOR UPDATE TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND estado='borrador' AND
 ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))))
 WITH CHECK(confirmada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND estado='emitida' AND
 ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
CREATE POLICY confirmar ON expediente.salida_servicio_hito FOR INSERT TO etf_app WITH CHECK(
 confirmado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND EXISTS(SELECT 1 FROM expediente.salida_servicio s
 WHERE s.id=salida_id AND s.estado='emitida' AND s.fecha=salida_servicio_hito.fecha
 AND s.confirmada_por=salida_servicio_hito.confirmado_por AND s.confirmada_en=salida_servicio_hito.confirmado_en));
CREATE POLICY propia ON expediente.salida_servicio_peticion FOR INSERT TO etf_app WITH CHECK(
 usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
CREATE POLICY propia_borrar ON expediente.salida_servicio_peticion FOR DELETE TO etf_app USING(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid);
CREATE POLICY propia_lectura ON expediente.salida_servicio_peticion AS RESTRICTIVE FOR SELECT TO etf_app USING(usuario_id=nullif(current_setting('app.usuario_id',true),'')::uuid);

-- Sólo esta función puede incrementar la versión para el profesional: no
-- se amplían sus permisos para editar el estado u otras columnas del expediente.
CREATE POLICY salida_propietario ON expediente.expediente TO etf_propietario
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))))
 WITH CHECK(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
CREATE FUNCTION expediente.version_salida_servicio(p_id uuid,p_version integer,p_incrementar boolean)
 RETURNS TABLE(actual integer,coincide boolean)
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
 DECLARE v integer; estado_actual text;
 BEGIN
  IF nullif(current_setting('app.usuario_id',true),'')::uuid IS NULL OR NOT (coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]) && ARRAY['coordinacion','profesional_etf']::text[]) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado';
  END IF;
  SELECT e.version,e.estado INTO v,estado_actual FROM expediente.expediente e
  WHERE e.id=p_id AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
   OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
    AND e.equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))) FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION 'expediente.derivacion_no_encontrada'; END IF;
  IF NOT p_incrementar THEN RETURN QUERY SELECT v,true; RETURN; END IF;
  IF p_version IS NULL OR v<>p_version THEN RETURN QUERY SELECT v,false; RETURN; END IF;
  IF estado_actual NOT IN ('aceptada','intervencion','en_intervencion','propuesta_cierre') THEN RAISE EXCEPTION 'error.estado_no_permite'; END IF;
  UPDATE expediente.expediente SET version=v+1,modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid,modificado_en=clock_timestamp() WHERE id=p_id;
  RETURN QUERY SELECT v+1,true;
 END $$;
REVOKE EXECUTE ON FUNCTION expediente.version_salida_servicio(uuid,integer,boolean) FROM PUBLIC,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.version_salida_servicio(uuid,integer,boolean) TO etf_app;
