SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regprocedure('estadistica.api_cifras_consulta(integer,uuid)') IS NULL THEN
 RAISE EXCEPTION 'migracion.estado_previo'; END IF;
END $$;
-- La revisión cerrada es inmutable. Estado y revisión se leen en una sola
-- sentencia MVCC, sin esperar ni impedir el cierre o la reapertura.
CREATE OR REPLACE FUNCTION estadistica.api_cifras_consulta(p_anio integer,p_equipo uuid) RETURNS jsonb
 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,estadistica AS $$
 DECLARE i jsonb; r jsonb;
 BEGIN
 IF nullif(current_setting('app.usuario_id',true),'') IS NULL
 OR NOT 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}')) THEN
 RAISE EXCEPTION 'estadistica.memoria_acceso_denegado'; END IF;
 IF p_anio NOT BETWEEN 1 AND 9998 THEN RAISE EXCEPTION 'estadistica.entrada_invalida'; END IF;
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
