SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
DO $$ BEGIN
 IF current_user<>'etf_propietario' OR session_user<>'etf_migrador'
 OR to_regprocedure('estadistica.codigo_cifras(text,text)') IS NULL
 OR to_regclass('expediente.espera_memoria') IS NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Conserva todos los códigos históricos y la validación cerrada de las cifras.
CREATE OR REPLACE FUNCTION estadistica.codigo_cifras(campo text,codigo text) RETURNS boolean
 LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog,estadistica,catalogo AS $$
 SELECT codigo='' OR CASE campo
  WHEN 'apartado.codigo' THEN codigo=ANY(ARRAY['1','2','3','4','5','6','7','8','9','10','11.1','11.2','12.1','12.2','13.1','13.2','13.3','14','14.1','15','16','17','18','19','20','21','22','23','24','25','26','27','28'])
  WHEN 'poblacion' THEN codigo=ANY(ARRAY['AT_ME','ME','AT','CI','CO','NC','ACT','PE'])
  WHEN 'contrato.denominador' THEN codigo=ANY(ARRAY['pendiente_programa','atendidos_del_apartado_1','no_aceptados_con_fecha_en_el_anio','aceptados_sin_consentimiento_mas_de_30_dias','lista_espera_o_aceptados_sin_consentimiento_mas_de_30_dias','cerrados_en_el_anio','atendidos_riesgo_del_apartado_1','propuestas_del_apartado_12_1','reunificacion_del_apartado_1'])
  WHEN 'contrato.desconocidos' THEN codigo=ANY(ARRAY['sin_dato_separado'])
  WHEN 'contrato.fecha_datos' THEN codigo=ANY(ARRAY['corte','cierre_episodio','fecha','fecha_no_aceptacion','cese_medida'])
  WHEN 'contrato.division_cero' THEN codigo=ANY(ARRAY['estadistica.divisor_cero'])
  WHEN 'celda.unidad' THEN codigo=ANY(ARRAY['familias','menores','adultos','comisiones','informes'])
  WHEN 'grupo.subprograma' THEN codigo=ANY(ARRAY['riesgo','reunificacion'])
  WHEN 'grupo.origen' THEN codigo=ANY(ARRAY['anterior','incorporado'])
  WHEN 'grupo.sexo' THEN codigo=ANY(ARRAY['nino','nina'])
  WHEN 'grupo.motivo' THEN codigo=ANY(ARRAY['objetivos_cumplidos','objetivos_parciales','rechazo','no_adherencia','traslado','mayoria_edad','propuesta_dsr','medida_protectora','ingreso_centro','otras_causas'])
  WHEN 'grupo.estado:3' THEN codigo=ANY(ARRAY['si','no'])
  WHEN 'grupo.estado:10' THEN codigo=ANY(ARRAY['si','no'])
  WHEN 'grupo.estado:18' THEN codigo=ANY(ARRAY['si','no'])
  WHEN 'grupo.estado:11.1' THEN codigo=ANY(ARRAY['si','no'])
  WHEN 'grupo.estado:11.2' THEN codigo=ANY(ARRAY['si','no'])
  WHEN 'grupo.estado:12.1' THEN codigo=ANY(ARRAY['si','no'])
  WHEN 'grupo.estado:12.2' THEN codigo=ANY(ARRAY['inicio_desamparo_o_medida','a_la_espera','no_necesaria'])
  WHEN 'grupo.estado:14' THEN codigo=ANY(ARRAY['SALUD_MENTAL','SALUD','ATENCION_TEMPRANA','ADICCIONES','EDUCACION','AYUDAS_FAMILIARES','VIOLENCIA_SEXUAL','SPM','ONG','MUJER','DISCAPACIDAD','DEPENDENCIA','FISCALIA','OTROS'])
  WHEN 'grupo.estado:14.1' THEN codigo=ANY(ARRAY['si','no'])
  WHEN 'grupo.estado:15' THEN codigo=ANY(ARRAY['comision'])
  WHEN 'grupo.estado:17' THEN codigo=ANY(ARRAY['BIPARENTAL','MONOPARENTAL_PADRE','MONOPARENTAL_MADRE','CUSTODIA_COMPARTIDA','SIN_CUSTODIA_COMPARTIDA','RECONSTITUIDA','EXTENSA','OTRAS'])
  WHEN 'grupo.estado:20' THEN codigo=ANY(ARRAY['ANALFABETISMO','SIN_OBLIGATORIOS','OBLIGATORIOS','SECUNDARIOS','UNIVERSITARIOS'])
  WHEN 'grupo.estado:21' THEN codigo=ANY(ARRAY['AUTONOMO','EMPLEO_FIJO','EMPLEO_TEMPORAL','ECONOMIA_SUMERGIDA','SIN_EMPLEO','PENSIONISTA','VARIAS','OTRAS'])
  WHEN 'grupo.estado:22' THEN codigo=ANY(ARRAY['SIN_DIFICULTADES','DIFICULTADES_SIN_NECESIDADES_BASICAS','DIFICULTADES_NECESIDADES_BASICAS'])
  WHEN 'grupo.estado:24' THEN codigo=ANY(ARRAY['FAMILIA_01','FAMILIA_02','FAMILIA_03','FAMILIA_04','FAMILIA_05','FAMILIA_06','FAMILIA_07','FAMILIA_08','FAMILIA_09','FAMILIA_10','FAMILIA_11','FAMILIA_12','FAMILIA_13','FAMILIA_14','FAMILIA_15','FAMILIA_16','FAMILIA_17','FAMILIA_18','FAMILIA_19','FAMILIA_20','FAMILIA_21','FAMILIA_22','FAMILIA_23'])
  WHEN 'grupo.estado:26' THEN codigo=ANY(ARRAY['MENOR_01','MENOR_02','MENOR_03','MENOR_04','MENOR_05','MENOR_06','MENOR_07','MENOR_08','MENOR_09','MENOR_10','MENOR_11'])
  WHEN 'grupo.estado:27' THEN codigo=ANY(ARRAY['DESPROTECCION_01','DESPROTECCION_02','DESPROTECCION_03','DESPROTECCION_04','DESPROTECCION_05','DESPROTECCION_06','DESPROTECCION_07','DESPROTECCION_08','DESPROTECCION_09','DESPROTECCION_10','DESPROTECCION_11','DESPROTECCION_12'])
  WHEN 'grupo.estado:28' THEN codigo=ANY(ARRAY['PADRE_MADRE','PADRE','MADRE','PAREJA','TUTOR','ABUELO','TIO','HERMANO','PRIMO','FAMILIAR','ALLEGADO'])
  WHEN 'grupo.tramo:13.1' THEN codigo=ANY(ARRAY['0','1','2','3'])
  WHEN 'grupo.tramo:13.2' THEN codigo=ANY(ARRAY['0','1','2','3'])
  WHEN 'grupo.tramo:19' THEN codigo=ANY(ARRAY['edad_menos_20','edad_20_29','edad_30_39','edad_40_49','edad_50_mas'])
  WHEN 'grupo.tramo:23' THEN codigo=ANY(ARRAY['MENOS_50','DE_50_A_100','DE_100_A_150','DE_150_A_200','MAS_200'])
  WHEN 'grupo.tramo:25' THEN codigo=ANY(ARRAY['edad_0_5','edad_6_11','edad_12_17'])
  WHEN 'grupo.organismo:16' THEN codigo=ANY(ARRAY['JUZGADOS','FISCALIA','SPAF','SPM','CENTROS_EDUCATIVOS','CENTROS_SANITARIOS','SERVICIOS_SOCIALES','OTROS'])
  WHEN 'porcentaje.aviso' THEN codigo=ANY(ARRAY['estadistica.denominador_pendiente','estadistica.divisor_cero'])
  WHEN 'aviso' THEN codigo=ANY(ARRAY['estadistica.no_aceptados_sin_equipo','estadistica.poblacion_parental_sin_dato','estadistica.catalogo_dimension_sin_dato','estadistica.denominador_pendiente','estadistica.divisor_cero','estadistica.recuento_actividad_sin_dato','estadistica.seguimiento_subprograma_sin_dato','estadistica.varios_cierres_pendiente','estadistica.motivo_sin_dato','estadistica.subprograma_sin_dato','estadistica.aceptacion_sin_dato'])
  WHEN 'version_regla' THEN codigo ~ '^[0-9]+(\.[0-9]+)*(:[0-9]+)?$'
  ELSE false END
 OR EXISTS(SELECT 1 FROM catalogo.lista_opcion o JOIN catalogo.version v ON v.id=o.version_id
  WHERE v.estado='publicada' AND o.codigo=$2 AND o.lista=CASE campo
   WHEN 'grupo.papel' THEN 'papel_parental'
   WHEN 'grupo.estado:4' THEN 'servicio_derivante'
   WHEN 'grupo.estado:5' THEN 'servicio_derivante'
   WHEN 'grupo.causa:5' THEN 'causa_no_aceptacion'
   WHEN 'grupo.estado:17' THEN 'composicion_familiar'
   WHEN 'grupo.estado:20' THEN 'nivel_educativo'
   WHEN 'grupo.estado:21' THEN 'situacion_laboral'
   WHEN 'grupo.estado:22' THEN 'situacion_economica'
   WHEN 'grupo.tramo:23' THEN 'tramo_ingresos_smi'
   WHEN 'grupo.organismo:16' THEN 'organismo_informe'
   ELSE NULL END);
$$;
REVOKE EXECUTE ON FUNCTION estadistica.codigo_cifras(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION estadistica.codigo_cifras(text,text) TO etf_app;

-- El ejecutor de Consulta recibe sólo la proyección estadística de la espera.
CREATE VIEW estadistica.calculo_espera_memoria WITH(security_barrier=true) AS
 SELECT expediente_id,equipo_id,recepcion,desde,hasta,registrado_en,terminado_en
 FROM expediente.espera_memoria
 WHERE desde IS NOT NULL AND session_user='etf_calculo_memoria'
 AND nullif(current_setting('app.usuario_id',true),'') IS NOT NULL
 AND 'consulta'=ANY(coalesce(string_to_array(current_setting('app.perfiles',true),','),'{}'::text[]))
 AND (coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]='{}'::uuid[]
 OR equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]));
REVOKE ALL ON estadistica.calculo_espera_memoria FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.calculo_espera_memoria TO etf_calculo_memoria;
