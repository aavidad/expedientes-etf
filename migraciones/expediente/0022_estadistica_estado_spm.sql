SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('expediente.intervencion_revision') IS NULL
 OR to_regclass('expediente.api_estadistica_familias') IS NULL
 OR to_regclass('expediente.api_estadistica_menores') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- No se calcula aquí la vigencia final: depende de las revisiones conocidas
-- por el lector. Una corrección futura no puede acortar una foto anterior.
CREATE VIEW expediente.api_estadistica_menores WITH(security_invoker=true,security_barrier=true) AS
 SELECT r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en,
 coalesce(jsonb_agg(jsonb_build_object('id',m.menor_id,'sexo',m.sexo,
 'fecha_nacimiento',m.fecha_nacimiento,'subprograma',m.subprograma,
 'centro_proteccion_codigo',m.centro_proteccion_codigo,
 'propuesto_atencion_temprana',m.propuesto_atencion_temprana) ORDER BY m.orden)
 FILTER(WHERE m.revision_id IS NOT NULL),'[]'::jsonb) AS valor
 FROM expediente.menores_revision r LEFT JOIN expediente.menor_revision m ON m.revision_id=r.id
 GROUP BY r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en;

CREATE VIEW expediente.api_estadistica_intervencion WITH(security_invoker=true,security_barrier=true) AS
 SELECT r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en,
 jsonb_build_object('fase',r.fase,'listas_conocidas',r.listas_conocidas,'marcas',coalesce((
 SELECT jsonb_agg(jsonb_build_object('lista',m.lista,'codigo',m.codigo,'menores_excluidos',coalesce((
 SELECT jsonb_agg(x.menor_id ORDER BY x.menor_id) FROM expediente.intervencion_exclusion x
 WHERE x.revision_id=r.id AND x.marca_id=m.id),'[]'::jsonb)) ORDER BY m.lista,m.codigo)
 FROM expediente.intervencion_marca rm JOIN expediente.marca m ON m.id=rm.marca_id
 WHERE rm.revision_id=r.id),'[]'::jsonb)) AS valor
 FROM expediente.intervencion_revision r;
REVOKE ALL ON expediente.api_estadistica_menores,expediente.api_estadistica_intervencion FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON expediente.api_estadistica_menores,expediente.api_estadistica_intervencion TO etf_app,etf_copia;

-- El puerto sólo sincroniza una salida ya confirmada, nunca una emisión Word.
CREATE FUNCTION expediente.transicion_spm(recurso uuid, anterior text, nuevo text) RETURNS boolean
 LANGUAGE sql STABLE SET search_path=pg_catalog,expediente AS $$
 SELECT EXISTS(SELECT 1 FROM expediente.salida_spm s WHERE s.expediente_id=recurso AND s.fecha_derivacion IS NOT NULL
 AND ((anterior='en_intervencion' AND nuevo='derivada_spm' AND s.respuesta='a_la_espera')
 OR (anterior='derivada_spm' AND nuevo='en_intervencion' AND s.respuesta IN ('inicio_desamparo_medida','medida_no_necesaria') AND s.fecha_respuesta IS NOT NULL)))
$$;
REVOKE EXECUTE ON FUNCTION expediente.transicion_spm(uuid,text,text) FROM PUBLIC,etf_app,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.transicion_spm(uuid,text,text) TO etf_app;

CREATE FUNCTION expediente.sincronizar_estado_spm(recurso uuid) RETURNS void
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,expediente AS $$
DECLARE e expediente.expediente; s expediente.salida_spm; nuevo text; actor uuid;
BEGIN
 actor:=nullif(current_setting('app.usuario_id',true),'')::uuid;
 SELECT * INTO e FROM expediente.expediente WHERE id=recurso FOR UPDATE;
 IF actor IS NULL OR e.id IS NULL OR NOT (
 coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
 OR (coalesce('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)
 AND e.equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),'')::uuid[],'{}'::uuid[])))) THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO s FROM expediente.salida_spm WHERE expediente_id=recurso;
 IF s.modificado_por IS DISTINCT FROM actor THEN
  RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
 END IF;
 nuevo:=CASE WHEN s.respuesta='a_la_espera' THEN 'derivada_spm' ELSE 'en_intervencion' END;
 IF NOT expediente.transicion_spm(recurso,e.estado,nuevo) THEN RETURN; END IF;
 UPDATE expediente.expediente SET estado=nuevo,version=version+1,modificado_por=actor,modificado_en=s.modificado_en WHERE id=recurso;
 INSERT INTO expediente.estado_cambio(expediente_id,equipo_id,estado_anterior,estado_nuevo,cambiado_por,cambiado_en)
 VALUES(recurso,e.equipo_id,e.estado,nuevo,actor,s.modificado_en);
END $$;
REVOKE EXECUTE ON FUNCTION expediente.sincronizar_estado_spm(uuid) FROM PUBLIC,etf_app,etf_copia;
GRANT EXECUTE ON FUNCTION expediente.sincronizar_estado_spm(uuid) TO etf_app;
CREATE POLICY propietario_spm ON expediente.estado_cambio TO etf_propietario USING(true) WITH CHECK(true);

ALTER TABLE expediente.estado_cambio DROP CONSTRAINT estado_cambio_transicion;
ALTER TABLE expediente.estado_cambio ADD CONSTRAINT estado_cambio_transicion CHECK (
 (estado_anterior='derivacion_recibida' AND estado_nuevo IN ('lista_espera','no_aceptada','aceptada'))
 OR (estado_anterior='lista_espera' AND estado_nuevo IN ('no_aceptada','aceptada'))
 OR (estado_anterior='aceptada' AND estado_nuevo='en_intervencion')
 OR (estado_anterior='en_intervencion' AND estado_nuevo='derivada_spm')
 OR (estado_anterior='derivada_spm' AND estado_nuevo='en_intervencion')
 OR (estado_anterior IN ('aceptada','en_intervencion','derivada_spm') AND estado_nuevo='cerrada')
 OR (estado_anterior='cerrada' AND estado_nuevo IN ('aceptada','en_intervencion','derivada_spm')));

CREATE OR REPLACE FUNCTION expediente.proteger_propuesta_version() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$ BEGIN
 IF current_user='etf_app' AND NEW.familia IS DISTINCT FROM OLD.familia
 AND OLD.estado NOT IN ('aceptada','en_intervencion') THEN
  RAISE EXCEPTION 'error.estado_no_permite' USING ERRCODE='42501';
 END IF;
 IF 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))) THEN
  IF (to_jsonb(NEW)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[])
   IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[])
   OR (NEW.estado IS DISTINCT FROM OLD.estado AND NOT ((
    OLD.estado='aceptada' AND NEW.estado='en_intervencion'
    AND EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=NEW.id AND tipo='consentimiento')) OR expediente.transicion_spm(NEW.id,OLD.estado,NEW.estado))) THEN
   RAISE EXCEPTION 'expediente.acceso_denegado' USING ERRCODE='42501';
  END IF;
 END IF;
 RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION expediente.proteger_inicio_intervencion() RETURNS trigger
LANGUAGE plpgsql SET search_path=pg_catalog,expediente AS $$
BEGIN
 IF 'profesional_etf'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND NOT ('coordinacion'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[])))
 AND (to_jsonb(NEW)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[])
 IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['familia','estado','version','modificado_por','modificado_en']::text[]) THEN
  RAISE EXCEPTION 'error.sin_permiso';
 END IF;
 IF NEW.estado='en_intervencion' AND OLD.estado IS DISTINCT FROM NEW.estado AND NOT expediente.transicion_spm(NEW.id,OLD.estado,NEW.estado) THEN
  IF NOT (OLD.estado='aceptada' OR (OLD.estado IN ('cerrada','derivada_spm') AND
   coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)))
   OR NOT EXISTS(SELECT 1 FROM expediente.hito WHERE expediente_id=NEW.id AND tipo='consentimiento') THEN
   RAISE EXCEPTION 'error.estado_no_permite';
  END IF;
 END IF;
 IF OLD.estado='en_intervencion' AND NEW.estado<>OLD.estado AND NOT expediente.transicion_spm(NEW.id,OLD.estado,NEW.estado) AND NOT (NEW.estado='cerrada' AND
  coalesce('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),',')),false)) THEN
  RAISE EXCEPTION 'error.estado_no_permite';
 END IF;
 RETURN NEW;
END $$;
REVOKE EXECUTE ON FUNCTION expediente.proteger_inicio_intervencion() FROM PUBLIC,etf_app,etf_copia;

