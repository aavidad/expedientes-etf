SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regclass('estadistica.memoria_revision') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='etf_calculo_memoria')
 OR to_regclass('estadistica.calculo_expedientes') IS NOT NULL THEN
 RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
-- El ejecutor técnico no pertenece a etf_app ni puede asumir otro rol.
GRANT USAGE ON SCHEMA estadistica,catalogo TO etf_calculo_memoria;
CREATE POLICY calculo_consulta ON expediente.expediente FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.episodio FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.hito FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.estado_cambio FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.dato_estadistico_vigencia FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.familia_revision FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.adultos_revision FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.adulto_revision FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.derivacion_datos FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND EXISTS(SELECT 1 FROM expediente.expediente e WHERE e.id=expediente_id AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR e.equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
CREATE VIEW estadistica.calculo_expedientes WITH(security_barrier=true) AS SELECT id,equipo_id,numero,fecha_recepcion,municipio_id,servicio FROM ( SELECT e.id,e.equipo_id,e.numero,e.fecha_recepcion,e.municipio_id,
 CASE d.contenido->'contenido'->>'tipo' WHEN 'anexo3' THEN 'SSSSCC' WHEN 'anexo4' THEN 'SPM' ELSE '' END AS servicio
 FROM expediente.expediente e LEFT JOIN expediente.derivacion_datos d ON d.expediente_id=e.id
 WHERE e.equipo_id IS NOT NULL) fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_expedientes FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_expedientes TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_episodios WITH(security_barrier=true) AS SELECT id,expediente_id,equipo_id,apertura_fecha,creado_en,reapertura_de_id,cierre_fecha,motivo_cierre,cerrado_en FROM ( SELECT p.id,p.expediente_id,p.equipo_id,p.apertura_fecha,p.creado_en,
 to_jsonb(p)->>'reapertura_de_id' AS reapertura_de_id,
 (to_jsonb(p)->>'cierre_fecha')::date AS cierre_fecha,
 to_jsonb(p)->>'motivo_cierre' AS motivo_cierre,
 (to_jsonb(p)->>'cerrado_en')::timestamptz AS cerrado_en
 FROM expediente.episodio p) fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_episodios FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_episodios TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_hitos WITH(security_barrier=true) AS SELECT id,expediente_id,equipo_id,episodio_id,tipo,fecha,confirmado_en FROM ( SELECT id,expediente_id,equipo_id,episodio_id,tipo,fecha,confirmado_en FROM expediente.hito) fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_hitos FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_hitos TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_estados WITH(security_barrier=true) AS SELECT id,expediente_id,equipo_id,estado_anterior,estado_nuevo,cambiado_en FROM ( SELECT id,expediente_id,equipo_id,estado_anterior,estado_nuevo,cambiado_en FROM expediente.estado_cambio) fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_estados FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_estados TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_vigencias WITH(security_barrier=true) AS SELECT id,expediente_id,equipo_id,atributo,valor,vigente_desde,registrado_en FROM ( SELECT id,expediente_id,equipo_id,atributo,valor,vigente_desde,registrado_en
 FROM expediente.dato_estadistico_vigencia WHERE entidad_tipo='expediente' AND atributo='subprograma') fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_vigencias FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_vigencias TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_familias WITH(security_barrier=true) AS SELECT id,expediente_id,equipo_id,version,vigente_desde,registrado_en,valor FROM ( SELECT id,expediente_id,equipo_id,version,vigente_desde,registrado_en,
 jsonb_build_object('composicion_familiar',valor->'composicion_familiar',
 'familia_numerosa',valor->'familia_numerosa','situacion_economica',valor->'situacion_economica',
 'ingresos_anuales_centimos',valor->'ingresos_anuales_centimos','tramo_smi',valor->'tramo_smi') AS valor
 FROM expediente.familia_revision) fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_familias FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_familias TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_adultos WITH(security_barrier=true) AS SELECT id,expediente_id,equipo_id,version,vigente_desde,registrado_en,valor FROM ( SELECT r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en,
 coalesce(jsonb_agg(jsonb_build_object('papel',a.papel,'sexo',a.sexo,
 'fecha_nacimiento',a.fecha_nacimiento,'nivel_educativo',a.nivel_educativo,
 'situacion_laboral',a.situacion_laboral) ORDER BY a.orden) FILTER(WHERE a.revision_id IS NOT NULL),'[]') AS valor
 FROM expediente.adultos_revision r LEFT JOIN expediente.adulto_revision a ON a.revision_id=r.id
 GROUP BY r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en) fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_adultos FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_adultos TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_comision_tecnica WITH(security_barrier=true) AS SELECT equipo_id,fecha,retirada FROM actividad.comision_tecnica WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_comision_tecnica FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_comision_tecnica TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_informe_tecnico WITH(security_barrier=true) AS SELECT equipo_id,fecha,retirada,organismo FROM actividad.informe_tecnico WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_informe_tecnico FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_informe_tecnico TO etf_calculo_memoria;
GRANT SELECT(anio,importe_mensual_centimos,importe_anual_centimos,version) ON catalogo.smi TO etf_calculo_memoria;
CREATE POLICY calculo_memoria ON catalogo.smi FOR SELECT TO etf_calculo_memoria USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')));

GRANT SELECT(id,codigo) ON catalogo.equipo TO etf_calculo_memoria;
CREATE POLICY calculo_memoria ON catalogo.equipo FOR SELECT TO etf_calculo_memoria USING(nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));

-- FX1: fuentes históricas de menores e intervención, sin nombres ni texto_otros.
CREATE POLICY calculo_consulta ON expediente.menores_revision FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.menor_revision FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.intervencion_revision FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.intervencion_marca FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.intervencion_exclusion FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE POLICY calculo_consulta ON expediente.marca FOR SELECT TO etf_propietario USING (session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[])));
CREATE VIEW estadistica.calculo_menores WITH(security_barrier=true) AS SELECT id,expediente_id,equipo_id,version,vigente_desde,registrado_en,valor FROM ( SELECT r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en,
 coalesce(jsonb_agg(jsonb_build_object('id',m.menor_id,'sexo',m.sexo,
 'fecha_nacimiento',m.fecha_nacimiento,'subprograma',m.subprograma,
 'centro_proteccion_codigo',m.centro_proteccion_codigo,
 'propuesto_atencion_temprana',m.propuesto_atencion_temprana) ORDER BY m.orden)
 FILTER(WHERE m.revision_id IS NOT NULL),'[]'::jsonb) AS valor
 FROM expediente.menores_revision r LEFT JOIN expediente.menor_revision m ON m.revision_id=r.id
 GROUP BY r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en) fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_menores FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_menores TO etf_calculo_memoria;
CREATE VIEW estadistica.calculo_intervencion WITH(security_barrier=true) AS SELECT id,expediente_id,equipo_id,version,vigente_desde,registrado_en,valor FROM ( SELECT r.id,r.expediente_id,r.equipo_id,r.version,r.vigente_desde,r.registrado_en,
 jsonb_build_object('fase',r.fase,'listas_conocidas',r.listas_conocidas,'marcas',coalesce((
 SELECT jsonb_agg(jsonb_build_object('lista',m.lista,'codigo',m.codigo,'menores_excluidos',coalesce((
 SELECT jsonb_agg(x.menor_id ORDER BY x.menor_id) FROM expediente.intervencion_exclusion x
 WHERE x.revision_id=r.id AND x.marca_id=m.id),'[]'::jsonb)) ORDER BY m.lista,m.codigo)
 FROM expediente.intervencion_marca rm JOIN expediente.marca m ON m.id=rm.marca_id
 WHERE rm.revision_id=r.id),'[]'::jsonb)) AS valor
 FROM expediente.intervencion_revision r) fuentes WHERE session_user='etf_calculo_memoria' AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[] OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_intervencion FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_intervencion TO etf_calculo_memoria;

CREATE POLICY revision_cifras_interna ON estadistica.memoria_revision FOR SELECT TO etf_propietario
 USING('consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')));
CREATE FUNCTION estadistica.api_cifras_consulta(p_anio integer,p_equipo uuid) RETURNS jsonb
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,estadistica AS $$
 DECLARE i jsonb; r jsonb;
 BEGIN
 IF nullif(current_setting('app.usuario_id',true),'') IS NULL
 OR NOT 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) THEN
 RAISE EXCEPTION 'estadistica.memoria_acceso_denegado'; END IF;
 IF p_anio NOT BETWEEN 1 AND 9998 THEN RAISE EXCEPTION 'estadistica.entrada_invalida'; END IF;
 PERFORM pg_advisory_xact_lock(4606,1);
 SELECT rev.informe INTO i FROM estadistica.memoria m JOIN estadistica.memoria_revision rev
 ON rev.memoria_id=m.id AND rev.revision=m.revision_actual
 WHERE m.anio=p_anio AND m.equipo_id IS NOT DISTINCT FROM p_equipo AND m.estado='cerrada';
 IF i IS NULL THEN RETURN NULL; END IF;
 SELECT jsonb_build_object('anio',p_anio,'equipo_id',coalesce(p_equipo::text,''),
 'equipo_codigo',i->'EquipoCodigo','fecha_cifras',i->'ConocidoEn','corte',i->'Corte','definitivas',true,
 'apartados',coalesce(jsonb_agg(jsonb_build_object('codigo',a->'Codigo',
 'total',jsonb_build_object('familias',a->'Total'->'Familias','menores',a->'Total'->'Menores','adultos',a->'Total'->'Adultos','comisiones',a->'Total'->'Comisiones','informes',a->'Total'->'Informes','familias_sin_relacion_menores',a->'Total'->'FamiliasSinRelacionMenores'),
 'avisos',coalesce((SELECT jsonb_object_agg(key,value) FROM jsonb_each(a->'Avisos') WHERE key IN ('estadistica.aceptacion_sin_dato','estadistica.motivo_sin_dato','estadistica.recuento_actividad_sin_dato','estadistica.seguimiento_subprograma_sin_dato','estadistica.subprograma_sin_dato','estadistica.varios_cierres_pendiente','estadistica.denominador_pendiente','estadistica.divisor_cero','estadistica.no_aceptados_sin_equipo','estadistica.poblacion_parental_sin_dato','estadistica.catalogo_dimension_sin_dato')),'{}'::jsonb),'celdas',coalesce((SELECT jsonb_agg(jsonb_build_object(
 'grupo',jsonb_build_object('subprograma',c->'Grupo'->'Subprograma','origen',c->'Grupo'->'Origen','sexo',c->'Grupo'->'Sexo','tramo',c->'Grupo'->'Tramo','motivo',c->'Grupo'->'Motivo','papel',c->'Grupo'->'Papel','estado',c->'Grupo'->'Estado','causa',c->'Grupo'->'Causa','organismo',c->'Grupo'->'Organismo'),'unidad',c->'Unidad',
 'numerador',c->'Celda'->'Numerador','sin_dato',c->'Celda'->'SinDato',
 'denominador',CASE WHEN (c->'Celda'->'Denominador'->>'Conocido')::boolean THEN c->'Celda'->'Denominador'->'Valor' ELSE 'null'::jsonb END,
 'aviso_porcentaje',CASE WHEN c->'Celda'->'Porcentaje'->>'Aviso' IN ('estadistica.divisor_cero','estadistica.denominador_pendiente') THEN c->'Celda'->'Porcentaje'->>'Aviso' ELSE '' END,'porcentaje_centesimas',CASE WHEN c->'Celda'->'Porcentaje'->>'Aviso' IN ('','estadistica.divisor_cero') THEN c->'Celda'->'Porcentaje'->'Centesimas' ELSE 'null'::jsonb END)) FROM jsonb_array_elements(a->'Celdas') c),'[]'::jsonb)) ORDER BY a->>'Codigo'),'[]'::jsonb))
 INTO r FROM jsonb_array_elements(i->'Apartados') a;
 RETURN r;
 END $$;
REVOKE EXECUTE ON FUNCTION estadistica.api_cifras_consulta(integer,uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION estadistica.api_cifras_consulta(integer,uuid) TO etf_app;
