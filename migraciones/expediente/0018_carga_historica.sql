SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.menores_revision') IS NULL
 OR to_regclass('expediente.carga_inicial') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- El lote sólo guarda procedencia y recuentos. Familia, menores, decisiones e
-- historia siguen perteneciendo a sus tablas y servicios existentes.
CREATE TABLE expediente.carga_inicial (
 id uuid PRIMARY KEY,equipo_id uuid NOT NULL REFERENCES catalogo.equipo(id),
 anio integer NOT NULL DEFAULT 2026 CHECK(anio=2026),
 origen text NOT NULL DEFAULT 'manual' CHECK(origen IN ('manual','hoja')),
 estado text NOT NULL DEFAULT 'cargado_para_revision' CHECK(estado='cargado_para_revision'),
 cargada_por uuid NOT NULL REFERENCES identidad.usuario(id),
 cargada_en timestamptz NOT NULL CHECK(isfinite(cargada_en)),
 recuentos jsonb NOT NULL CHECK(jsonb_typeof(recuentos)='object'),
 transaccion xid8 NOT NULL DEFAULT pg_current_xact_id(),
 UNIQUE(id,equipo_id)
);
ALTER TABLE expediente.carga_inicial ENABLE ROW LEVEL SECURITY;
ALTER TABLE expediente.carga_inicial FORCE ROW LEVEL SECURITY;
REVOKE ALL ON expediente.carga_inicial FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.carga_inicial TO etf_app,etf_copia;
GRANT INSERT(id,equipo_id,cargada_por,cargada_en,recuentos) ON expediente.carga_inicial TO etf_app;
CREATE POLICY lectura ON expediente.carga_inicial FOR SELECT TO etf_app USING (
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))));
CREATE POLICY escritura ON expediente.carga_inicial FOR INSERT TO etf_app WITH CHECK (
 cargada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 OR ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))));

ALTER TABLE expediente.expediente
 DROP CONSTRAINT expediente_origen_check,
 ADD CONSTRAINT expediente_origen_check CHECK(origen IN ('derivacion','carga_inicial')),
 ADD COLUMN carga_id uuid,
 ADD COLUMN equipo_carga_id uuid,
 ADD COLUMN cargado_por uuid REFERENCES identidad.usuario(id),
 ADD FOREIGN KEY(carga_id,equipo_carga_id) REFERENCES expediente.carga_inicial(id,equipo_id),
 ADD CONSTRAINT carga_procedencia CHECK (
 (origen='derivacion' AND carga_id IS NULL AND equipo_carga_id IS NULL AND cargado_por IS NULL)
 OR (origen='carga_inicial' AND carga_id IS NOT NULL AND equipo_carga_id IS NOT NULL AND cargado_por IS NOT NULL
  AND (equipo_id IS NULL OR equipo_id=equipo_carga_id)));
CREATE INDEX expediente_carga ON expediente.expediente(carga_id);
-- Un número desconocido permanece nulo, sólo en un caso histórico. No se
-- atribuye un correlativo ni una regla de numeración que nunca existieron.
DO $$ DECLARE definicion text; condicion text; BEGIN
 SELECT pg_get_constraintdef(oid) INTO STRICT definicion FROM pg_constraint
 WHERE conrelid='expediente.expediente'::regclass AND conname='expediente_aceptacion_completa';
 FOREACH condicion IN ARRAY ARRAY['numero IS NOT NULL','anio_numero IS NOT NULL','correlativo IS NOT NULL','regla_numeracion_id IS NOT NULL'] LOOP
  IF condicion='regla_numeracion_id IS NOT NULL' THEN
   definicion:=replace(definicion,'('||condicion||')','(origen=''carga_inicial'' OR '||condicion||')');
  ELSE
   definicion:=replace(definicion,'('||condicion||')','((origen=''carga_inicial'' AND numero IS NULL) OR '||condicion||')');
  END IF;
 END LOOP;
 ALTER TABLE expediente.expediente DROP CONSTRAINT expediente_aceptacion_completa;
 EXECUTE 'ALTER TABLE expediente.expediente ADD CONSTRAINT expediente_aceptacion_completa '||definicion;
END $$;
ALTER TABLE expediente.expediente ADD CONSTRAINT numero_historico_coherente CHECK (
 origen<>'carga_inicial' OR
 (numero IS NULL AND anio_numero IS NULL AND correlativo IS NULL AND regla_numeracion_id IS NULL)
 OR (numero ~ '^[0-9]{2}-[0-9]{2}-[0-9]{2}$' AND correlativo=split_part(numero,'-',2)::integer
  AND correlativo>0 AND split_part(numero,'-',3)=to_char(fecha_aceptacion,'YY')));

-- Permite sólo la carga de filas nuevas en esta transacción, con el actor y
-- ámbito originales. No es una variable de sesión que pueda reactivar un caso.
CREATE FUNCTION expediente.carga_en_curso(recurso uuid) RETURNS boolean
LANGUAGE sql STABLE SET search_path=pg_catalog,expediente AS $$
 SELECT EXISTS(SELECT 1 FROM expediente.expediente e JOIN expediente.carga_inicial c ON c.id=e.carga_id
  WHERE e.id=recurso AND e.origen='carga_inicial' AND c.transaccion=pg_current_xact_id()
  AND c.cargada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND e.cargado_por=c.cargada_por
  AND ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) OR
   ('profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND c.equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))))
$$;
REVOKE EXECUTE ON FUNCTION expediente.carga_en_curso(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION expediente.carga_en_curso(uuid) TO etf_app;
CREATE POLICY carga_manual_insertar ON expediente.expediente FOR INSERT TO etf_app WITH CHECK (
 origen='carga_inicial' AND cargado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_carga_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])));
CREATE POLICY carga_manual_leer ON expediente.expediente FOR SELECT TO etf_app USING (
 origen='carga_inicial' AND nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND
 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_carga_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])));
CREATE POLICY carga_manual_actualizar ON expediente.expediente FOR UPDATE TO etf_app
 USING (origen='carga_inicial' AND nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND
 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_carga_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))
 WITH CHECK (origen='carga_inicial' AND modificado_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND
 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])) AND equipo_carga_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])));
CREATE POLICY carga_manual_datos ON expediente.derivacion_datos FOR INSERT TO etf_app
 WITH CHECK(expediente.carga_en_curso(expediente_id));
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['episodio','hito','estado_cambio','dato_estadistico_vigencia','adulto','familia_revision','adultos_revision','adulto_revision','menor','menores_revision','menor_revision'] LOOP
  EXECUTE format('CREATE POLICY carga_manual ON expediente.%I TO etf_app USING (expediente.carga_en_curso(expediente_id)) WITH CHECK (expediente.carga_en_curso(expediente_id))',tabla);
 END LOOP;
END $$;
ALTER TABLE expediente.episodio ADD COLUMN cierre_historico boolean NOT NULL DEFAULT false;
DO $$ DECLARE c record; definicion text; BEGIN
 FOR c IN SELECT conname,pg_get_constraintdef(oid) AS definicion FROM pg_constraint
  WHERE conrelid='expediente.episodio'::regclass AND contype='c' LOOP
  IF c.definicion LIKE '%propuesta_cierre_id IS NOT NULL%' THEN
   definicion:=replace(c.definicion,'propuesta_cierre_id IS NOT NULL','(propuesta_cierre_id IS NOT NULL OR cierre_historico)');
   EXECUTE format('ALTER TABLE expediente.episodio DROP CONSTRAINT %I',c.conname);
   EXECUTE format('ALTER TABLE expediente.episodio ADD CONSTRAINT %I %s',c.conname,definicion);
  END IF;
 END LOOP;
END $$;
ALTER TABLE expediente.episodio ADD CHECK(NOT cierre_historico OR (cierre_fecha IS NOT NULL AND propuesta_cierre_id IS NULL AND forzado_id IS NULL));
GRANT UPDATE(cierre_historico) ON expediente.episodio TO etf_app;

-- Las protecciones ordinarias siguen intactas después de confirmar el lote.
DROP TRIGGER proteger_propuesta_version ON expediente.expediente;
CREATE TRIGGER proteger_propuesta_version BEFORE UPDATE ON expediente.expediente FOR EACH ROW
 WHEN (NOT expediente.carga_en_curso(OLD.id)) EXECUTE FUNCTION expediente.proteger_propuesta_version();
DROP TRIGGER proteger_ficha ON expediente.expediente;
CREATE TRIGGER proteger_ficha BEFORE UPDATE ON expediente.expediente FOR EACH ROW
 WHEN (NOT expediente.carga_en_curso(OLD.id)) EXECUTE FUNCTION expediente.proteger_ficha();
DROP TRIGGER proteger_inicio_intervencion ON expediente.expediente;
CREATE TRIGGER proteger_inicio_intervencion BEFORE UPDATE ON expediente.expediente FOR EACH ROW
 WHEN (NOT expediente.carga_en_curso(OLD.id)) EXECUTE FUNCTION expediente.proteger_inicio_intervencion();
DROP TRIGGER proteger_expediente_cerrado ON expediente.expediente;
CREATE TRIGGER proteger_expediente_cerrado BEFORE UPDATE ON expediente.expediente FOR EACH ROW
 WHEN (NOT expediente.carga_en_curso(OLD.id)) EXECUTE FUNCTION expediente.proteger_expediente_cerrado();
DROP TRIGGER proteger_episodio_cierre ON expediente.episodio;
CREATE TRIGGER proteger_episodio_cierre BEFORE INSERT OR UPDATE ON expediente.episodio FOR EACH ROW
 WHEN (NOT expediente.carga_en_curso(NEW.expediente_id)) EXECUTE FUNCTION expediente.proteger_episodio_cierre();

CREATE FUNCTION expediente.proteger_carga_historica() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF TG_OP='UPDATE' AND ROW(NEW.origen,NEW.carga_id,NEW.equipo_carga_id,NEW.cargado_por,NEW.fecha_recepcion)
 IS DISTINCT FROM ROW(OLD.origen,OLD.carga_id,OLD.equipo_carga_id,OLD.cargado_por,OLD.fecha_recepcion) THEN
  RAISE EXCEPTION 'expediente.asignacion_inmutable';
 END IF;
 IF NEW.origen='carga_inicial' THEN
  IF TG_OP='INSERT' AND NOT EXISTS(SELECT 1 FROM expediente.carga_inicial c WHERE c.id=NEW.carga_id
    AND c.transaccion=pg_current_xact_id() AND c.cargada_por=nullif(current_setting('app.usuario_id',true),'')::uuid AND NEW.cargado_por=c.cargada_por) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
  IF NEW.numero IS NOT NULL AND NOT EXISTS(SELECT 1 FROM catalogo.equipo WHERE id=NEW.equipo_id AND codigo=split_part(NEW.numero,'-',1)) THEN
   RAISE EXCEPTION 'expediente.numero_historico_invalido';
  END IF;
  IF NEW.fecha_aceptacion<(NEW.fecha_recepcion AT TIME ZONE 'UTC')::date
   OR NEW.fecha_no_aceptacion<(NEW.fecha_recepcion AT TIME ZONE 'UTC')::date THEN
   RAISE EXCEPTION 'expediente.carga_historica_invalida';
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_carga_historica() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_carga_historica BEFORE INSERT OR UPDATE ON expediente.expediente FOR EACH ROW EXECUTE FUNCTION expediente.proteger_carga_historica();

CREATE FUNCTION expediente.proteger_episodio_historico() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ DECLARE previo expediente.episodio; BEGIN
 IF NOT expediente.carga_en_curso(NEW.expediente_id) THEN
  IF NEW.cierre_historico IS DISTINCT FROM (CASE WHEN TG_OP='UPDATE' THEN OLD.cierre_historico ELSE false END) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
  RETURN NEW;
 END IF;
 IF NEW.creado_por<>nullif(current_setting('app.usuario_id',true),'')::uuid OR NEW.apertura_fecha>(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date
  OR NEW.cierre_fecha>(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date
  OR NEW.apertura_fecha>DATE '2026-12-31' OR NEW.cierre_fecha>DATE '2026-12-31'
  OR (NEW.cierre_fecha IS NOT NULL AND (NEW.cerrado_por<>nullif(current_setting('app.usuario_id',true),'')::uuid OR NOT NEW.cierre_historico)) THEN
  RAISE EXCEPTION 'expediente.carga_historica_invalida';
 END IF;
 IF TG_OP='UPDATE' AND (OLD.cierre_fecha IS NOT NULL OR
  (to_jsonb(NEW)-ARRAY['cierre_fecha','motivo_cierre','motivo_acta','motivo_acta_texto','motivo_spaf','cerrado_por','cerrado_en','cierre_historico']) IS DISTINCT FROM
  (to_jsonb(OLD)-ARRAY['cierre_fecha','motivo_cierre','motivo_acta','motivo_acta_texto','motivo_spaf','cerrado_por','cerrado_en','cierre_historico'])) THEN
  RAISE EXCEPTION 'expediente.asignacion_inmutable';
 END IF;
 IF NEW.reapertura_de_id IS NOT NULL THEN
  SELECT * INTO STRICT previo FROM expediente.episodio WHERE id=NEW.reapertura_de_id;
  IF previo.expediente_id<>NEW.expediente_id OR previo.equipo_id<>NEW.equipo_id
   OR previo.cierre_fecha IS NULL OR NEW.apertura_fecha<previo.cierre_fecha THEN
   RAISE EXCEPTION 'expediente.carga_historica_invalida';
  END IF;
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_episodio_historico() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER proteger_episodio_historico BEFORE INSERT OR UPDATE ON expediente.episodio FOR EACH ROW EXECUTE FUNCTION expediente.proteger_episodio_historico();

-- El rol de aplicación tampoco puede confirmar un histórico incompleto por
-- SQL directo. Se comprueba el estado final, después de incorporar episodios.
CREATE FUNCTION expediente.comprobar_carga_confirmada() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ DECLARE e expediente.expediente; abiertos integer; BEGIN
 IF NEW.origen<>'carga_inicial' THEN RETURN NULL; END IF;
 SELECT * INTO STRICT e FROM expediente.expediente WHERE id=NEW.id;
 IF e.estado NOT IN ('aceptada','en_intervencion','cerrada','no_aceptada')
  OR (e.fecha_recepcion AT TIME ZONE 'UTC')::date>LEAST(DATE '2026-12-31',(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date)
  OR e.fecha_aceptacion>LEAST(DATE '2026-12-31',(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date) THEN
  RAISE EXCEPTION 'expediente.carga_historica_invalida';
 END IF;
 IF e.estado='no_aceptada' THEN
  IF e.fecha_no_aceptacion<DATE '2026-01-01' OR e.fecha_no_aceptacion>LEAST(DATE '2026-12-31',(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date) THEN
   RAISE EXCEPTION 'expediente.carga_historica_invalida';
  END IF;
 ELSE
  SELECT count(*) INTO abiertos FROM expediente.episodio WHERE expediente_id=e.id AND cierre_fecha IS NULL;
  IF (e.estado='cerrada' AND abiertos<>0) OR (e.estado<>'cerrada' AND abiertos<>1)
   OR NOT EXISTS(SELECT 1 FROM expediente.episodio WHERE expediente_id=e.id AND reapertura_de_id IS NULL AND apertura_fecha=e.fecha_aceptacion)
   OR NOT EXISTS(SELECT 1 FROM expediente.episodio WHERE expediente_id=e.id AND apertura_fecha<=DATE '2026-12-31' AND (cierre_fecha IS NULL OR cierre_fecha>=DATE '2026-01-01')) THEN
   RAISE EXCEPTION 'expediente.carga_historica_invalida';
  END IF;
 END IF;
 RETURN NULL;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.comprobar_carga_confirmada() FROM PUBLIC,etf_app,etf_copia;
CREATE CONSTRAINT TRIGGER comprobar_carga_confirmada AFTER INSERT OR UPDATE ON expediente.expediente
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION expediente.comprobar_carga_confirmada();
