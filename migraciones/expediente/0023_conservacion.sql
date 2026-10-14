SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.seguimiento') IS NULL
 OR to_regclass('expediente.conservacion') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

CREATE TABLE expediente.conservacion (
 expediente_id uuid PRIMARY KEY REFERENCES expediente.expediente(id),
 episodio_id uuid NOT NULL REFERENCES expediente.episodio(id),
 cierre date NOT NULL CHECK(isfinite(cierre)),
 anios integer NOT NULL CHECK(anios BETWEEN 1 AND 200),
 regla_id uuid NOT NULL REFERENCES catalogo.version(id),
 fin_conservacion date NOT NULL CHECK(isfinite(fin_conservacion) AND fin_conservacion>cierre),
 version integer NOT NULL DEFAULT 1 CHECK(version>0),
 version_expediente integer NOT NULL,
 propuesta_por uuid REFERENCES identidad.usuario(id), propuesta_en timestamptz,
 confirmada_por uuid REFERENCES identidad.usuario(id), anonimizado_en timestamptz,
 CHECK((propuesta_por IS NULL)=(propuesta_en IS NULL)),
 CHECK((confirmada_por IS NULL)=(anonimizado_en IS NULL)),
 CHECK(confirmada_por IS NULL OR (propuesta_por IS NOT NULL AND confirmada_por<>propuesta_por AND anonimizado_en>=propuesta_en))
);
CREATE INDEX conservacion_pendiente ON expediente.conservacion(fin_conservacion,expediente_id) WHERE anonimizado_en IS NULL;
-- Credencial interna de una única transacción. Un GUC del cliente nunca basta
-- para omitir protecciones de cierre ni desencadenantes de historia.
CREATE TABLE expediente.conservacion_ejecucion (
 transaccion xid8 PRIMARY KEY, recurso uuid, anonimizar boolean NOT NULL
);
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['conservacion','conservacion_ejecucion'] LOOP
  EXECUTE format('ALTER TABLE expediente.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE expediente.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('REVOKE ALL ON expediente.%I FROM PUBLIC,etf_app,etf_copia',tabla);
  EXECUTE format('CREATE POLICY propietario ON expediente.%I TO etf_propietario USING(true) WITH CHECK(true)',tabla);
 END LOOP;
END $$;
GRANT SELECT ON expediente.conservacion TO etf_copia;
CREATE POLICY copia ON expediente.conservacion FOR SELECT TO etf_copia USING(true);

CREATE FUNCTION expediente.conservacion_interna(p_recurso uuid,p_anonimizar boolean) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
 SELECT EXISTS(SELECT 1 FROM expediente.conservacion_ejecucion
 WHERE transaccion=pg_current_xact_id_if_assigned() AND (NOT p_anonimizar OR (anonimizar AND recurso=p_recurso)))
$$;
REVOKE EXECUTE ON FUNCTION expediente.conservacion_interna(uuid,boolean) FROM PUBLIC,etf_app,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.conservacion_interna(uuid,boolean) TO etf_calculo_memoria;

-- El acceso de propietario sólo existe mientras ejecuta estas funciones. No
-- abre filas personales a Administración, Consulta ni al ejecutor estadístico.
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['expediente','episodio','derivacion_datos','adulto','menor','marca','intervencion_marca','finalizacion','salida_spm','spm_forzado','salida_servicio','seguimiento','seguimiento_historia','peticion_idempotente','peticion_spm','salida_servicio_peticion'] LOOP
  EXECUTE format('CREATE POLICY conservacion_interna ON expediente.%I TO etf_propietario USING(expediente.conservacion_interna(NULL,false)) WITH CHECK(expediente.conservacion_interna(NULL,false))',tabla);
 END LOOP;
END $$;

-- Mantiene intactas las funciones anteriores salvo la excepción autenticada
-- para retirar datos personales. Se aplica a sus desencadenantes de validación
-- e historia; no a permisos ni a auditoría.
DO $$ DECLARE f record; definicion text; BEGIN
 FOR f IN SELECT DISTINCT p.oid FROM pg_trigger t JOIN pg_proc p ON p.oid=t.tgfoid
 JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='expediente' AND NOT t.tgisinternal
 AND c.relname=ANY(ARRAY['expediente','episodio','derivacion_datos','adulto','menor','marca','intervencion_marca','finalizacion','salida_spm','spm_forzado','salida_servicio','seguimiento','seguimiento_historia'])
 LOOP
  definicion:=pg_get_functiondef(f.oid);
  IF definicion !~* '\mBEGIN\M' THEN RAISE EXCEPTION 'migracion.estado_previo'; END IF;
  definicion:=regexp_replace(definicion,'\mBEGIN\M',
   'BEGIN IF current_user=''etf_propietario'' THEN IF expediente.conservacion_interna(coalesce((to_jsonb(NEW)->>''expediente_id'')::uuid,(to_jsonb(NEW)->>''id'')::uuid),true) THEN IF TG_WHEN=''AFTER'' THEN RETURN NULL; END IF; RETURN NEW; END IF; END IF;','i');
  EXECUTE definicion;
 END LOOP;
END $$;

CREATE FUNCTION expediente.marcar_conservacion(p_id uuid,p_anios integer,p_regla uuid,p_ahora timestamptz) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE fila record; actor uuid:=nullif(current_setting('app.usuario_id',true),'')::uuid; perfiles text[]:=coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}');
BEGIN
 IF actor IS NULL OR NOT ('administracion'=ANY(perfiles) OR (p_id IS NOT NULL AND 'coordinacion'=ANY(perfiles))) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 IF p_anios IS NULL OR p_ahora IS NULL OR p_anios NOT BETWEEN 1 AND 200 OR p_regla IS NULL OR NOT isfinite(p_ahora) THEN RAISE EXCEPTION 'conservacion.no_permitida'; END IF;
 INSERT INTO expediente.conservacion_ejecucion VALUES(pg_current_xact_id(),NULL,false);
 -- Sólo se bloquean y escriben las marcas nuevas o desfasadas. La lectura
 -- repetida de Administración no recorre una consulta por cada expediente.
 FOR fila IN
  WITH candidatas AS MATERIALIZED (
   SELECT e.id,e.version,p.id AS episodio,p.cierre_fecha
   FROM expediente.expediente e JOIN expediente.episodio p ON p.expediente_id=e.id
   LEFT JOIN expediente.conservacion c ON c.expediente_id=e.id
   WHERE e.estado='cerrada' AND (p_id IS NULL OR e.id=p_id) AND p.cierre_fecha IS NOT NULL
   AND NOT EXISTS(SELECT 1 FROM expediente.episodio hijo WHERE hijo.reapertura_de_id=p.id)
   AND (c.expediente_id IS NULL OR (c.anonimizado_en IS NULL AND
    (c.episodio_id,c.cierre,c.anios,c.version_expediente) IS DISTINCT FROM (p.id,p.cierre_fecha,p_anios,e.version)))
   ORDER BY e.id FOR UPDATE OF e
  ), cambiadas AS (
   INSERT INTO expediente.conservacion(expediente_id,episodio_id,cierre,anios,regla_id,fin_conservacion,version_expediente)
   SELECT id,episodio,cierre_fecha,p_anios,p_regla,(cierre_fecha+make_interval(years=>p_anios))::date,version FROM candidatas
   ON CONFLICT(expediente_id) DO UPDATE SET episodio_id=excluded.episodio_id,cierre=excluded.cierre,anios=excluded.anios,regla_id=excluded.regla_id,fin_conservacion=excluded.fin_conservacion,version_expediente=excluded.version_expediente,
    version=conservacion.version+1,propuesta_por=NULL,propuesta_en=NULL
   WHERE conservacion.anonimizado_en IS NULL AND (conservacion.episodio_id,conservacion.cierre,conservacion.anios,conservacion.version_expediente) IS DISTINCT FROM (excluded.episodio_id,excluded.cierre,excluded.anios,excluded.version_expediente)
   RETURNING expediente_id
  ) SELECT expediente_id FROM cambiadas ORDER BY expediente_id
 LOOP
  PERFORM auditoria.registrar(p_ahora,'conservacion.marcar','expediente',fila.expediente_id,fila.expediente_id,'ok');
 END LOOP;
 DELETE FROM expediente.conservacion_ejecucion WHERE transaccion=pg_current_xact_id();
END $$;
REVOKE EXECUTE ON FUNCTION expediente.marcar_conservacion(uuid,integer,uuid,timestamptz) FROM PUBLIC,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.marcar_conservacion(uuid,integer,uuid,timestamptz) TO etf_app;

CREATE FUNCTION expediente.listar_conservacion(p_ahora timestamptz) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE resultado jsonb; actor uuid:=nullif(current_setting('app.usuario_id',true),'')::uuid;
BEGIN
 IF actor IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 INSERT INTO expediente.conservacion_ejecucion VALUES(pg_current_xact_id(),NULL,false);
 SELECT coalesce(jsonb_agg(jsonb_build_object('expediente_id',c.expediente_id,'numero',e.numero,'cierre',c.cierre,'fin_conservacion',c.fin_conservacion,'anios',c.anios,'version',c.version,
 'estado',CASE WHEN c.anonimizado_en IS NOT NULL THEN 'anonimizado' WHEN c.propuesta_por IS NOT NULL THEN 'pendiente_segunda_persona' ELSE 'pendiente_propuesta' END,
 'puede_confirmar',c.propuesta_por IS NOT NULL AND c.propuesta_por<>actor AND c.anonimizado_en IS NULL) ORDER BY c.fin_conservacion,c.expediente_id),'[]'::jsonb)
 INTO resultado FROM expediente.conservacion c JOIN expediente.expediente e ON e.id=c.expediente_id
 WHERE c.fin_conservacion<=(p_ahora AT TIME ZONE 'Europe/Madrid')::date AND e.estado='cerrada'
 AND NOT EXISTS(SELECT 1 FROM expediente.seguimiento s WHERE s.expediente_id=e.id AND s.estado='activo');
 PERFORM auditoria.registrar(p_ahora,'conservacion.listar','expediente',NULL,NULL,'ok');
 DELETE FROM expediente.conservacion_ejecucion WHERE transaccion=pg_current_xact_id();
 RETURN resultado;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.listar_conservacion(timestamptz) FROM PUBLIC,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.listar_conservacion(timestamptz) TO etf_app;

CREATE FUNCTION expediente.confirmar_conservacion(p_id uuid,p_version integer,p_anonimizar boolean,p_ahora timestamptz) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE c expediente.conservacion; e expediente.expediente; actor uuid:=nullif(current_setting('app.usuario_id',true),'')::uuid; resultado jsonb;
BEGIN
 IF actor IS NULL OR NOT coalesce('administracion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 IF p_id IS NULL OR p_version IS NULL OR p_version<1 OR p_anonimizar IS NULL OR p_ahora IS NULL OR NOT isfinite(p_ahora) THEN RAISE EXCEPTION 'conservacion.no_permitida'; END IF;
 INSERT INTO expediente.conservacion_ejecucion VALUES(pg_current_xact_id(),p_id,false);
 SELECT * INTO e FROM expediente.expediente WHERE id=p_id FOR UPDATE;
 SELECT * INTO c FROM expediente.conservacion WHERE expediente_id=p_id FOR UPDATE;
 IF c.expediente_id IS NULL OR e.estado<>'cerrada' OR c.version_expediente<>e.version OR c.fin_conservacion>(p_ahora AT TIME ZONE 'Europe/Madrid')::date
 OR EXISTS(SELECT 1 FROM expediente.seguimiento WHERE expediente_id=p_id AND estado='activo') THEN RAISE EXCEPTION 'conservacion.no_permitida'; END IF;
 -- Las repeticiones de una confirmación ya hecha no vuelven a retirar datos.
 IF NOT ((NOT p_anonimizar AND c.propuesta_por IS NOT DISTINCT FROM actor) OR (p_anonimizar AND c.confirmada_por IS NOT DISTINCT FROM actor)) THEN
  IF c.version<>p_version THEN RAISE EXCEPTION 'conservacion.conflicto' USING DETAIL=c.version::text; END IF;
  IF c.anonimizado_en IS NOT NULL THEN RAISE EXCEPTION 'conservacion.no_permitida'; END IF;
  IF NOT p_anonimizar THEN
   IF c.propuesta_por IS NOT NULL THEN RAISE EXCEPTION 'conservacion.no_permitida'; END IF;
   UPDATE expediente.conservacion SET propuesta_por=actor,propuesta_en=p_ahora,version=version+1 WHERE expediente_id=p_id RETURNING * INTO c;
  ELSE
   IF c.propuesta_por IS NULL OR c.propuesta_por=actor OR p_ahora<c.propuesta_en THEN RAISE EXCEPTION 'conservacion.no_permitida'; END IF;
   UPDATE expediente.conservacion_ejecucion SET anonimizar=true WHERE transaccion=pg_current_xact_id();
   -- Listas blancas: ningún campo libre nuevo se conserva accidentalmente.
   UPDATE expediente.expediente SET descripcion_situacion=CASE WHEN descripcion_situacion IS NULL THEN NULL ELSE '—' END,causa_no_aceptacion=CASE WHEN causa_no_aceptacion IS NULL THEN NULL ELSE '—' END,familia=(SELECT coalesce(jsonb_object_agg(key,value),'{}') FROM jsonb_each(familia) WHERE key=ANY(ARRAY['composicion_familiar','familia_numerosa','vivienda','situacion_economica','ingresos_anuales_centimos','tramo_smi'])),copia_traslado=CASE WHEN copia_traslado IS NULL THEN NULL ELSE '{}'::jsonb END WHERE id=p_id;
   UPDATE expediente.derivacion_datos SET contenido=jsonb_build_object('esquema_version','1','contenido',jsonb_build_object('tipo',contenido->'contenido'->'tipo'),'anonimizado',true) WHERE expediente_id=p_id;
   UPDATE expediente.adulto SET datos=(SELECT coalesce(jsonb_object_agg(key,value),'{}') FROM jsonb_each(datos) WHERE key=ANY(ARRAY['id','orden','papel','sexo','fecha_nacimiento','edad_declarada','nivel_educativo','situacion_laboral','es_tutor']))||jsonb_build_object('nombre','—','apellidos','—') WHERE expediente_id=p_id;
   UPDATE expediente.menor SET datos=(SELECT coalesce(jsonb_object_agg(key,value),'{}') FROM jsonb_each(datos) WHERE key=ANY(ARRAY['id','orden','sexo','fecha_nacimiento','nivel_escolarizacion','subprograma','centro_proteccion_codigo','propuesto_atencion_temprana']))||jsonb_build_object('nombre','—','apellidos','—') WHERE expediente_id=p_id;
   UPDATE expediente.marca SET texto_otros=NULL WHERE expediente_id=p_id;
   UPDATE expediente.intervencion_marca SET texto_otros='' WHERE expediente_id=p_id;
   UPDATE expediente.finalizacion SET datos=(SELECT coalesce(jsonb_object_agg(key,value),'{}') FROM jsonb_each(datos) WHERE key=ANY(ARRAY['version','motivo','sin_consentimiento','catalogo_version_id','motivo_acta','motivo_spaf'])) WHERE expediente_id=p_id;
   UPDATE expediente.episodio SET motivo_acta_texto=CASE WHEN motivo_acta_texto IS NULL THEN NULL ELSE '—' END,motivo_reapertura=CASE WHEN motivo_reapertura IS NULL THEN NULL ELSE '—' END WHERE expediente_id=p_id;
   UPDATE expediente.salida_spm SET motivo='—',propuesta_medida='—' WHERE expediente_id=p_id;
   UPDATE expediente.spm_forzado SET motivo='—' WHERE expediente_id=p_id;
   UPDATE expediente.salida_servicio SET motivo='—' WHERE expediente_id=p_id;
   UPDATE expediente.seguimiento SET motivo_reapertura=CASE WHEN motivo_reapertura IS NULL THEN NULL ELSE '—' END WHERE expediente_id=p_id;
   UPDATE expediente.seguimiento_historia SET datos=datos-'motivo_reapertura' WHERE expediente_id=p_id;
   -- Los recibos pueden contener fotos de nombres y textos. Se retiran sólo los
   -- del expediente; nunca eventos ni eslabones de auditoría.
   DELETE FROM expediente.peticion_idempotente WHERE ruta LIKE '%/'||p_id::text||'/%' OR strpos(respuesta::text,p_id::text)>0;
   DELETE FROM expediente.peticion_spm WHERE ruta LIKE '%/'||p_id::text||'/%' OR strpos(respuesta::text,p_id::text)>0;
   DELETE FROM expediente.salida_servicio_peticion WHERE expediente_id=p_id;
   PERFORM evaluacion.api_anonimizar_expediente(p_id,p_ahora);
   PERFORM documento.api_anonimizar_expediente(p_id,p_ahora);
   UPDATE expediente.conservacion SET confirmada_por=actor,anonimizado_en=p_ahora,version=version+1 WHERE expediente_id=p_id RETURNING * INTO c;
   -- Las restricciones diferidas deben terminar bajo el propietario y mientras
   -- sigue vigente la credencial. Administración no puede leer la ficha al commit.
   SET CONSTRAINTS expediente.comprobar_carga_confirmada,expediente.comprobar_episodio_actual,expediente.comprobar_enlace_traslado IMMEDIATE;
   SET CONSTRAINTS expediente.comprobar_carga_confirmada,expediente.comprobar_episodio_actual,expediente.comprobar_enlace_traslado DEFERRED;
  END IF;
 END IF;
 PERFORM auditoria.registrar(p_ahora,CASE WHEN p_anonimizar THEN 'conservacion.anonimizar' ELSE 'conservacion.proponer' END,'expediente',p_id,p_id,'ok');
 resultado:=jsonb_build_object('expediente_id',p_id,'numero',e.numero,'cierre',c.cierre,'fin_conservacion',c.fin_conservacion,'anios',c.anios,'version',c.version,'estado',CASE WHEN c.anonimizado_en IS NOT NULL THEN 'anonimizado' ELSE 'pendiente_segunda_persona' END,'puede_confirmar',c.propuesta_por<>actor AND c.anonimizado_en IS NULL);
 DELETE FROM expediente.conservacion_ejecucion WHERE transaccion=pg_current_xact_id();
 RETURN resultado;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.confirmar_conservacion(uuid,integer,boolean,timestamptz) FROM PUBLIC,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.confirmar_conservacion(uuid,integer,boolean,timestamptz) TO etf_app;

CREATE FUNCTION expediente.impedir_edicion_anonimizado() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE recurso uuid:=coalesce((to_jsonb(NEW)->>'expediente_id')::uuid,(to_jsonb(NEW)->>'id')::uuid,(to_jsonb(OLD)->>'expediente_id')::uuid);
BEGIN
 IF EXISTS(SELECT 1 FROM expediente.conservacion WHERE expediente_id=recurso AND anonimizado_en IS NOT NULL) THEN RAISE EXCEPTION 'conservacion.no_permitida'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.impedir_edicion_anonimizado() FROM PUBLIC,etf_app,etf_copia;
DO $$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['expediente','episodio','derivacion_datos','adulto','menor','marca','intervencion_marca','finalizacion','salida_spm','spm_forzado','salida_servicio','seguimiento','familia_revision','adultos_revision','adulto_revision','menores_revision','menor_revision','intervencion_revision','hito','dato_estadistico_vigencia'] LOOP
  EXECUTE format('CREATE TRIGGER impedir_edicion_anonimizado BEFORE INSERT OR UPDATE OR DELETE ON expediente.%I FOR EACH ROW EXECUTE FUNCTION expediente.impedir_edicion_anonimizado()',tabla);
 END LOOP;
END $$;
