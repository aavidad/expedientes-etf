SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.conservacion') IS NULL
 OR to_regclass('expediente.plazo_carga') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

CREATE TABLE expediente.plazo_carga (
 id boolean PRIMARY KEY DEFAULT true CHECK(id), fecha_cierre date,
 version integer NOT NULL DEFAULT 1 CHECK(version>0),
 dias_duplicados integer NOT NULL DEFAULT 30 CHECK(dias_duplicados BETWEEN 0 AND 366),
 CHECK(fecha_cierre IS NULL OR isfinite(fecha_cierre))
);
INSERT INTO expediente.plazo_carga(id) VALUES(true);
ALTER TABLE expediente.plazo_carga ENABLE ROW LEVEL SECURITY;
ALTER TABLE expediente.plazo_carga FORCE ROW LEVEL SECURITY;
REVOKE ALL ON expediente.plazo_carga FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.plazo_carga TO etf_app,etf_copia;
CREATE POLICY lectura ON expediente.plazo_carga FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]) && ARRAY['administracion','coordinacion','profesional_etf']);
CREATE POLICY propietario ON expediente.plazo_carga TO etf_propietario USING(true) WITH CHECK(true);
GRANT UPDATE(fecha_cierre,version,dias_duplicados) ON expediente.plazo_carga TO etf_app;
CREATE POLICY administrar ON expediente.plazo_carga FOR UPDATE TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'administracion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])))
 WITH CHECK(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'administracion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])));

CREATE FUNCTION expediente.bloquear_plazo_carga() RETURNS expediente.plazo_carga
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE p expediente.plazo_carga; BEGIN
 IF nullif(current_setting('app.usuario_id',true),'')::uuid IS NULL OR NOT coalesce(string_to_array(current_setting('app.perfiles',true),',') && ARRAY['coordinacion','profesional_etf','administracion'],false) THEN RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM expediente.plazo_carga WHERE id FOR SHARE;
 RETURN p;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.bloquear_plazo_carga() FROM PUBLIC,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.bloquear_plazo_carga() TO etf_app;

CREATE TABLE expediente.plazo_carga_historia (
 version integer PRIMARY KEY,fecha_cierre date NOT NULL,dias_duplicados integer NOT NULL,
 modificada_por uuid NOT NULL REFERENCES identidad.usuario(id),modificada_en timestamptz NOT NULL
);
ALTER TABLE expediente.plazo_carga_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE expediente.plazo_carga_historia FORCE ROW LEVEL SECURITY;
REVOKE ALL ON expediente.plazo_carga_historia FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.plazo_carga_historia TO etf_app,etf_copia;
CREATE POLICY lectura ON expediente.plazo_carga_historia FOR SELECT TO etf_app USING(
 nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL AND 'administracion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])));
CREATE POLICY propietario ON expediente.plazo_carga_historia TO etf_propietario USING(true) WITH CHECK(true);
CREATE FUNCTION expediente.registrar_plazo_carga() RETURNS trigger
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF nullif(current_setting('app.usuario_id',true),'')::uuid IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
 OR NEW.version<>OLD.version+1 THEN RAISE EXCEPTION 'carga.control_invalido'; END IF;
 INSERT INTO expediente.plazo_carga_historia VALUES(NEW.version,NEW.fecha_cierre,NEW.dias_duplicados,nullif(current_setting('app.usuario_id',true),'')::uuid,clock_timestamp());
 PERFORM auditoria.registrar(clock_timestamp(),'carga.fijar_plazo','carga',NULL,NULL,'ok');
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.registrar_plazo_carga() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER registrar_plazo_carga AFTER UPDATE ON expediente.plazo_carga FOR EACH ROW EXECUTE FUNCTION expediente.registrar_plazo_carga();

ALTER TABLE expediente.carga_inicial
 ADD COLUMN referencia bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
 DROP CONSTRAINT carga_inicial_estado_check,
 ADD CONSTRAINT carga_inicial_estado_check CHECK(estado IN ('cargado_para_revision','anulado')),
 ADD COLUMN actividad_posterior boolean NOT NULL DEFAULT true,
 ADD COLUMN anulada_por uuid REFERENCES identidad.usuario(id),
 ADD COLUMN anulada_en timestamptz,
 ADD COLUMN casos_anulados jsonb,
 ADD CHECK((estado='anulado')=(anulada_por IS NOT NULL AND anulada_en IS NOT NULL AND casos_anulados IS NOT NULL));
-- Las cargas anteriores quedan protegidas: no conocemos todos sus cambios.
ALTER TABLE expediente.carga_inicial ALTER COLUMN actividad_posterior SET DEFAULT false;
CREATE TABLE expediente.anulacion_carga_ejecucion (
 transaccion xid8 PRIMARY KEY,lote_id uuid NOT NULL
);
ALTER TABLE expediente.anulacion_carga_ejecucion ENABLE ROW LEVEL SECURITY;
ALTER TABLE expediente.anulacion_carga_ejecucion FORCE ROW LEVEL SECURITY;
REVOKE ALL ON expediente.anulacion_carga_ejecucion FROM PUBLIC,etf_app,etf_copia;
CREATE POLICY propietario ON expediente.anulacion_carga_ejecucion TO etf_propietario USING(true) WITH CHECK(true);
CREATE FUNCTION expediente.anulacion_carga_interna() RETURNS boolean
 LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
 SELECT EXISTS(SELECT 1 FROM expediente.anulacion_carga_ejecucion WHERE transaccion=pg_current_xact_id_if_assigned())
$$;
REVOKE EXECUTE ON FUNCTION expediente.anulacion_carga_interna() FROM PUBLIC,etf_app,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.anulacion_carga_interna() TO etf_calculo_memoria;
-- El propietario sólo abre las filas durante la operación interna. No concede
-- lectura de fichas a Administración ni permisos generales de borrado a la app.
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['expediente','carga_inicial','espera_memoria','derivacion_datos','episodio','hito','estado_cambio','dato_estadistico_vigencia','adulto','familia_revision','adultos_revision','adulto_revision','menor','menores_revision','menor_revision','contador_expediente','peticion_idempotente'] LOOP
  EXECUTE format('CREATE POLICY anulacion_interna ON expediente.%I TO etf_propietario USING(expediente.anulacion_carga_interna()) WITH CHECK(expediente.anulacion_carga_interna())',tabla);
 END LOOP;
END $$;

CREATE FUNCTION expediente.exigir_plazo_carga() RETURNS trigger
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE fin date; BEGIN
 SELECT fecha_cierre INTO fin FROM expediente.plazo_carga WHERE id FOR SHARE;
 IF fin IS NULL OR fin<(clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date THEN
  RAISE EXCEPTION 'carga.plazo_cerrado';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.exigir_plazo_carga() FROM PUBLIC,etf_app,etf_copia;
CREATE TRIGGER exigir_plazo_carga BEFORE INSERT ON expediente.carga_inicial
 FOR EACH ROW EXECUTE FUNCTION expediente.exigir_plazo_carga();
CREATE CONSTRAINT TRIGGER confirmar_plazo_carga AFTER INSERT ON expediente.carga_inicial
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION expediente.exigir_plazo_carga();
CREATE CONSTRAINT TRIGGER confirmar_plazo_anulacion AFTER UPDATE ON expediente.carga_inicial
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN(NEW.estado='anulado' AND OLD.estado<>'anulado') EXECUTE FUNCTION expediente.exigir_plazo_carga();

CREATE TABLE expediente.actividad_carga_ejecucion (
 transaccion xid8 PRIMARY KEY,recurso uuid NOT NULL,lote uuid
);
ALTER TABLE expediente.actividad_carga_ejecucion ENABLE ROW LEVEL SECURITY;
ALTER TABLE expediente.actividad_carga_ejecucion FORCE ROW LEVEL SECURITY;
REVOKE ALL ON expediente.actividad_carga_ejecucion FROM PUBLIC,etf_app,etf_copia;
CREATE POLICY propietario ON expediente.actividad_carga_ejecucion TO etf_propietario USING(true) WITH CHECK(true);
CREATE FUNCTION expediente.actividad_carga_interna(p_id uuid) RETURNS boolean
 LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
 SELECT EXISTS(SELECT 1 FROM expediente.actividad_carga_ejecucion WHERE transaccion=pg_current_xact_id_if_assigned() AND (recurso=p_id OR lote=p_id))
$$;
REVOKE EXECUTE ON FUNCTION expediente.actividad_carga_interna(uuid) FROM PUBLIC,etf_app,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.actividad_carga_interna(uuid) TO etf_calculo_memoria;

-- El marcador sólo pasa de falso a verdadero. El rol de aplicación no puede
-- escribirlo. Las filas creadas por la carga pertenecen a su misma transacción.
CREATE FUNCTION expediente.registrar_actividad_carga() RETURNS trigger
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE recurso uuid; lote_de_carga uuid; BEGIN
 IF expediente.anulacion_carga_interna() THEN IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW; END IF;
 IF TG_TABLE_NAME='traslado' THEN
  recurso:=coalesce((to_jsonb(NEW)->>'origen_id')::uuid,(to_jsonb(OLD)->>'origen_id')::uuid);
 ELSE
 recurso:=coalesce((to_jsonb(NEW)->>'expediente_id')::uuid,(to_jsonb(OLD)->>'expediente_id')::uuid,
  (to_jsonb(NEW)->>'id')::uuid,(to_jsonb(OLD)->>'id')::uuid);
 END IF;
 -- Esta función necesita leer sólo el padre del cambio; las políticas de
 -- propietario específicas de abajo no afectan a etf_app.
 INSERT INTO expediente.actividad_carga_ejecucion VALUES(pg_current_xact_id(),recurso,NULL);
 SELECT carga_id INTO lote_de_carga FROM expediente.expediente WHERE id=recurso FOR UPDATE;
 UPDATE expediente.actividad_carga_ejecucion SET lote=lote_de_carga WHERE transaccion=pg_current_xact_id();
 IF lote_de_carga IS NOT NULL THEN
  UPDATE expediente.carga_inicial SET actividad_posterior=true
   WHERE id=lote_de_carga AND transaccion<>pg_current_xact_id();
 END IF;
 DELETE FROM expediente.actividad_carga_ejecucion WHERE transaccion=pg_current_xact_id();
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.registrar_actividad_carga() FROM PUBLIC,etf_app,etf_copia;
CREATE POLICY actividad_propietario ON expediente.expediente TO etf_propietario USING(expediente.actividad_carga_interna(id)) WITH CHECK(expediente.actividad_carga_interna(id));
CREATE POLICY actividad_propietario ON expediente.carga_inicial TO etf_propietario USING(expediente.actividad_carga_interna(id)) WITH CHECK(expediente.actividad_carga_interna(id));
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['expediente','derivacion_datos','episodio','hito','estado_cambio','dato_estadistico_vigencia','adulto','familia_revision','adultos_revision','adulto_revision','menor','menores_revision','menor_revision','marca','marca_menor_excluido','intervencion_revision','intervencion_marca','intervencion_exclusion','salida_spm','spm_historia','spm_forzado','salida_servicio','salida_servicio_hito','propuesta_cierre','finalizacion','seguimiento','seguimiento_historia','traslado'] LOOP
  IF to_regclass('expediente.'||tabla) IS NOT NULL THEN
   EXECUTE format('CREATE TRIGGER zz_registrar_actividad_carga BEFORE UPDATE OR DELETE ON expediente.%I FOR EACH ROW EXECUTE FUNCTION expediente.registrar_actividad_carga()',tabla);
   IF tabla<>'expediente' THEN
    EXECUTE format('CREATE TRIGGER zz_registrar_actividad_carga_insertar BEFORE INSERT ON expediente.%I FOR EACH ROW EXECUTE FUNCTION expediente.registrar_actividad_carga()',tabla);
   END IF;
  END IF;
 END LOOP;
END $$;

-- Permite borrar exclusivamente dentro de la función autorizada; los controles
-- de expedientes cerrados siguen vigentes en todas las demás operaciones.
DO $$ DECLARE f record; definicion text; BEGIN
 FOR f IN SELECT DISTINCT p.oid FROM pg_trigger t JOIN pg_proc p ON p.oid=t.tgfoid
 JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='expediente' AND NOT t.tgisinternal AND (t.tgtype & 8)<>0
 AND c.relname=ANY(ARRAY['expediente','derivacion_datos','episodio','hito','estado_cambio','dato_estadistico_vigencia','adulto','familia_revision','adultos_revision','adulto_revision','menor','menores_revision','menor_revision'])
 AND p.proname<>'registrar_actividad_carga'
 LOOP
  definicion:=pg_get_functiondef(f.oid);
  definicion:=regexp_replace(definicion,'\mBEGIN\M',
   'BEGIN IF TG_OP=''DELETE'' THEN IF current_user=''etf_propietario'' THEN IF expediente.anulacion_carga_interna() THEN RETURN OLD; END IF; END IF; END IF;','i');
  EXECUTE definicion;
 END LOOP;
END $$;

CREATE FUNCTION expediente.anular_lote_carga(p_lote uuid,p_ahora timestamptz) RETURNS void
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE lote expediente.carga_inicial; fin date; ids uuid[]; tabla text; BEGIN
 IF nullif(current_setting('app.usuario_id',true),'')::uuid IS NULL OR NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 SELECT fecha_cierre INTO fin FROM expediente.plazo_carga WHERE id FOR SHARE;
 IF fin IS NULL OR fin<(clock_timestamp() AT TIME ZONE 'Europe/Madrid')::date THEN RAISE EXCEPTION 'carga.plazo_cerrado'; END IF;
 IF p_lote IS NULL OR p_ahora IS NULL OR NOT isfinite(p_ahora) THEN RAISE EXCEPTION 'carga.control_invalido'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended(p_lote::text,697053));
 INSERT INTO expediente.anulacion_carga_ejecucion VALUES(pg_current_xact_id(),p_lote);
 -- Orden común de padres antes del lote: los escritores del marcador bloquean
 -- primero el expediente y luego su lote, evitando una inversión de bloqueos.
 PERFORM id FROM expediente.expediente WHERE carga_id=p_lote ORDER BY id FOR UPDATE;
 SELECT * INTO lote FROM expediente.carga_inicial WHERE id=p_lote FOR UPDATE;
 IF lote.id IS NULL THEN RAISE EXCEPTION 'expediente.no_encontrado'; END IF;
 IF lote.estado='anulado' THEN
  DELETE FROM expediente.anulacion_carga_ejecucion WHERE transaccion=pg_current_xact_id(); RETURN;
 END IF;
 IF lote.actividad_posterior THEN RAISE EXCEPTION 'carga.lote_con_actividad'; END IF;
 SELECT array_agg(id ORDER BY id) INTO ids FROM expediente.expediente WHERE carga_id=p_lote;
 IF cardinality(ids) IS DISTINCT FROM (lote.recuentos->>'casos')::integer THEN RAISE EXCEPTION 'carga.lote_con_actividad'; END IF;
 UPDATE expediente.carga_inicial SET casos_anulados=(SELECT jsonb_agg(jsonb_build_object('id',id,'numero',numero)) FROM expediente.expediente WHERE carga_id=p_lote),
  estado='anulado',anulada_por=nullif(current_setting('app.usuario_id',true),'')::uuid,anulada_en=p_ahora WHERE id=p_lote;
 -- Lista de tablas propias. Cualquier vínculo de otro módulo impide el borrado
 -- por su FK: ni documentos ni evaluaciones se borran para permitir anular.
 FOREACH tabla IN ARRAY ARRAY['espera_memoria','menor_revision','menores_revision','adulto_revision','adultos_revision','familia_revision','hito','estado_cambio','dato_estadistico_vigencia','adulto','menor','episodio','derivacion_datos'] LOOP
  EXECUTE format('DELETE FROM expediente.%I WHERE expediente_id=ANY($1)',tabla) USING ids;
 END LOOP;
 DELETE FROM expediente.peticion_idempotente WHERE strpos(respuesta::text,p_lote::text)>0
  OR EXISTS(SELECT 1 FROM unnest(ids) i WHERE strpos(respuesta::text,i::text)>0);
 DELETE FROM expediente.expediente WHERE id=ANY(ids);
 PERFORM auditoria.registrar(p_ahora,'carga.anular_lote','carga',p_lote,NULL,'ok');
 DELETE FROM expediente.anulacion_carga_ejecucion WHERE transaccion=pg_current_xact_id();
EXCEPTION WHEN foreign_key_violation THEN RAISE EXCEPTION 'carga.lote_con_actividad';
END $$;
REVOKE EXECUTE ON FUNCTION expediente.anular_lote_carga(uuid,timestamptz) FROM PUBLIC,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.anular_lote_carga(uuid,timestamptz) TO etf_app;
