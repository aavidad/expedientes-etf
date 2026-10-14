SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
DO $$ BEGIN
 IF current_user <> 'etf_propietario' OR session_user <> 'etf_migrador'
 OR to_regclass('estadistica.memoria') IS NULL OR to_regclass('catalogo.equipo') IS NULL
 OR to_regclass('estadistica.foto_totales') IS NOT NULL THEN
  RAISE EXCEPTION 'migracion.estado_previo';
 END IF;
END $$;

-- Valores aprobados del dominio, catálogos y parámetros de Memoria v2026.1.
-- Una versión futura que introduzca códigos nuevos requiere ampliar este contrato.
CREATE FUNCTION estadistica.codigo_cifras(campo text,codigo text) RETURNS boolean
 LANGUAGE sql STABLE SECURITY INVOKER SET search_path=pg_catalog,estadistica,catalogo AS $$
 SELECT codigo='' OR CASE campo
  WHEN 'apartado.codigo' THEN codigo=ANY(ARRAY['1','2','3','4','5','6','7','8','9','10','11.1','11.2','12.1','12.2','13.1','13.2','13.3','14','14.1','15','16','17','18','19','20','21','22','23','24','25','26','27','28'])
  WHEN 'poblacion' THEN codigo=ANY(ARRAY['AT_ME','ME','AT','CI','CO','NC','ACT'])
  WHEN 'contrato.denominador' THEN codigo=ANY(ARRAY['pendiente_programa','atendidos_del_apartado_1','no_aceptados_con_fecha_en_el_anio','aceptados_sin_consentimiento_mas_de_30_dias','cerrados_en_el_anio','atendidos_riesgo_del_apartado_1','propuestas_del_apartado_12_1','reunificacion_del_apartado_1'])
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
REVOKE ALL ON FUNCTION estadistica.codigo_cifras(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION estadistica.codigo_cifras(text,text) TO etf_app;

-- La forma cerrada impide introducir narrativas, composición o identificadores
-- de expedientes, incluso si se intenta escribir directamente desde SQL.
CREATE FUNCTION estadistica.validar_nodo_cifras(nodo jsonb, tipo text, apartado text DEFAULT '') RETURNS boolean
 LANGUAGE plpgsql STABLE SECURITY INVOKER SET search_path=pg_catalog,estadistica,catalogo AS $$
DECLARE
 claves text[]; clave text; valor jsonb; hijo text; texto text; campo text;
BEGIN
 IF nodo IS NULL THEN RETURN false; END IF;
 IF tipo='entero' THEN
  RETURN jsonb_typeof(nodo)='number' AND nodo::text ~ '^[0-9]{1,10}$'
   AND nodo::numeric<=2147483647;
 ELSIF tipo='codigo' THEN
  texto:=nodo#>>'{}';
  RETURN jsonb_typeof(nodo)='string' AND texto ~ '^[A-Za-z0-9_.:-]{0,128}$'
   AND texto !~* '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'
   AND texto !~ '[0-9]{1,4}-[0-9]{1,6}-[0-9]{2,4}';
 ELSIF tipo='fecha' THEN
  texto:=nodo#>>'{}';
  IF jsonb_typeof(nodo)<>'string' OR texto !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$' THEN RETURN false; END IF;
  PERFORM texto::timestamptz;
  RETURN true;
 ELSIF tipo='apartados' OR tipo='celdas' THEN
  IF jsonb_typeof(nodo)<>'array' OR jsonb_array_length(nodo)>10000 THEN RETURN false; END IF;
  hijo:=CASE WHEN tipo='apartados' THEN 'apartado' ELSE 'celda' END;
  FOR valor IN SELECT v FROM jsonb_array_elements(nodo) AS a(v) LOOP
   IF NOT estadistica.validar_nodo_cifras(valor,hijo,apartado) THEN RETURN false; END IF;
  END LOOP;
  RETURN true;
 ELSIF tipo='porcentaje' AND nodo='null'::jsonb THEN RETURN true;
 END IF;

 IF jsonb_typeof(nodo)<>'object' THEN RETURN false; END IF;
 claves:=CASE tipo
  WHEN 'raiz' THEN ARRAY['anio','equipo_id','equipo_codigo','corte','conocido_en','apartados']
  WHEN 'apartado' THEN ARRAY['codigo','contrato','total','celdas','avisos']
  WHEN 'contrato' THEN ARRAY['poblacion','denominador','desconocidos','fecha_datos','corte','fecha_edad','version_regla','division_cero']
  WHEN 'total' THEN ARRAY['familias','menores','adultos','comisiones','informes','familias_sin_relacion_menores']
  WHEN 'celda' THEN ARRAY['grupo','unidad','numerador','denominador','porcentaje','sin_dato','poblacion','version_regla']
  WHEN 'grupo' THEN ARRAY['subprograma','origen','sexo','tramo','motivo','papel','estado','causa','organismo']
  WHEN 'denominador' THEN ARRAY['conocido','valor']
  WHEN 'porcentaje' THEN ARRAY['centesimas','aviso']
  WHEN 'avisos' THEN NULL
  ELSE ARRAY[]::text[] END;
 IF tipo='raiz' THEN
  IF nodo->>'equipo_id'='' THEN
   IF nodo->>'equipo_codigo'<>'' THEN RETURN false; END IF;
  ELSIF NOT EXISTS(SELECT 1 FROM catalogo.equipo e
   WHERE e.id=(nodo->>'equipo_id')::uuid AND e.codigo=nodo->>'equipo_codigo') THEN RETURN false;
  END IF;
 ELSIF tipo='apartado' THEN apartado:=nodo->>'codigo';
 END IF;
 IF tipo<>'avisos' AND (cardinality(claves)=0 OR NOT nodo ?& claves
   OR (SELECT count(*) FROM jsonb_object_keys(nodo))<>cardinality(claves)) THEN RETURN false; END IF;
 FOR clave,valor IN SELECT k,v FROM jsonb_each(nodo) AS a(k,v) LOOP
  IF tipo='avisos' THEN
   IF NOT estadistica.codigo_cifras('aviso',clave) OR NOT estadistica.validar_nodo_cifras(valor,'entero') THEN RETURN false; END IF;
   CONTINUE;
  ELSIF tipo='raiz' AND clave='equipo_id' THEN
   IF jsonb_typeof(valor)<>'string' OR (valor#>>'{}'<>'' AND valor#>>'{}' !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$') THEN RETURN false; END IF;
   CONTINUE;
  ELSIF tipo='denominador' AND clave='conocido' THEN
   IF jsonb_typeof(valor)<>'boolean' THEN RETURN false; END IF;
   CONTINUE;
  END IF;
  hijo:=CASE
   WHEN tipo='raiz' AND clave='anio' THEN 'entero'
   WHEN tipo='total' OR (tipo='celda' AND clave IN ('numerador','sin_dato'))
    OR (tipo='denominador' AND clave='valor') OR (tipo='porcentaje' AND clave='centesimas') THEN 'entero'
   WHEN (tipo='raiz' AND clave IN ('corte','conocido_en')) OR (tipo='contrato' AND clave IN ('corte','fecha_edad')) THEN 'fecha'
   WHEN tipo='raiz' AND clave='apartados' THEN 'apartados'
   WHEN tipo='apartado' AND clave IN ('contrato','total','celdas','avisos') THEN clave
   WHEN tipo='celda' AND clave IN ('grupo','denominador','porcentaje') THEN clave
   ELSE 'codigo' END;
  IF hijo='codigo' THEN
   IF NOT estadistica.validar_nodo_cifras(valor,'codigo') THEN RETURN false; END IF;
   campo:=CASE
    WHEN tipo='raiz' AND clave='equipo_codigo' THEN 'equipo_codigo'
    WHEN clave='poblacion' AND tipo IN ('contrato','celda') THEN 'poblacion'
    WHEN clave='version_regla' AND tipo IN ('contrato','celda') THEN 'version_regla'
    WHEN tipo='grupo' AND clave IN ('estado','tramo','causa','organismo') THEN 'grupo.'||clave||':'||apartado
    ELSE tipo||'.'||clave END;
   IF campo<>'equipo_codigo' AND NOT estadistica.codigo_cifras(campo,valor#>>'{}') THEN RETURN false; END IF;
  ELSIF NOT estadistica.validar_nodo_cifras(valor,hijo,apartado) THEN RETURN false;
  END IF;
 END LOOP;
 RETURN true;
EXCEPTION WHEN OTHERS THEN RETURN false;
END $$;
REVOKE ALL ON FUNCTION estadistica.validar_nodo_cifras(jsonb,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION estadistica.validar_nodo_cifras(jsonb,text,text) TO etf_app;

CREATE TABLE estadistica.foto_totales (
 anio integer NOT NULL CHECK(anio BETWEEN 1 AND 9998),
 equipo_id uuid REFERENCES catalogo.equipo(id),
 fecha timestamptz NOT NULL,
 cifras jsonb NOT NULL,
 publicado_por uuid NOT NULL REFERENCES identidad.usuario(id),
 UNIQUE NULLS NOT DISTINCT(anio,equipo_id),
 CONSTRAINT cifras_forma CHECK(octet_length(cifras::text)<=8388608
  AND estadistica.validar_nodo_cifras(cifras,'raiz')),
 CONSTRAINT cifras_ambito CHECK((cifras->>'anio')::integer=anio
  AND cifras->>'equipo_id'=coalesce(equipo_id::text,'')),
 CONSTRAINT cifras_fecha CHECK((cifras->>'conocido_en')::timestamptz=fecha)
);
ALTER TABLE estadistica.foto_totales ENABLE ROW LEVEL SECURITY;
ALTER TABLE estadistica.foto_totales FORCE ROW LEVEL SECURITY;
REVOKE ALL ON estadistica.foto_totales FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT,INSERT ON estadistica.foto_totales TO etf_app;
GRANT UPDATE(fecha,cifras,publicado_por) ON estadistica.foto_totales TO etf_app;
GRANT SELECT ON estadistica.foto_totales TO etf_copia;

CREATE POLICY foto_lectura_app ON estadistica.foto_totales FOR SELECT TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL
 AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
CREATE POLICY foto_insertar ON estadistica.foto_totales FOR INSERT TO etf_app
 WITH CHECK(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL
 AND publicado_por=nullif(current_setting('app.usuario_id',true),'')::uuid
 AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));
CREATE POLICY foto_actualizar ON estadistica.foto_totales FOR UPDATE TO etf_app
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL
 AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))))
 WITH CHECK(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL
 AND publicado_por=nullif(current_setting('app.usuario_id',true),'')::uuid
 AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));

-- El propietario de la vista lee sólo agregados ya validados. El contexto
-- Consulta no obtiene una política de lectura sobre la tabla para etf_app.
CREATE POLICY foto_vista_propietario ON estadistica.foto_totales FOR SELECT TO etf_propietario
 USING(nullif(current_setting('app.usuario_id',true),'')::uuid IS NOT NULL
 AND ('coordinacion'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 OR 'consulta'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 OR ('profesional_etf'=ANY(string_to_array(current_setting('app.perfiles',true),','))
 AND equipo_id=ANY(coalesce(nullif(current_setting('app.equipos',true),''),'{}')::uuid[]))));

CREATE VIEW estadistica.api_cifras_consulta WITH(security_barrier=true) AS
 SELECT anio,equipo_id,fecha,cifras FROM estadistica.foto_totales;
REVOKE ALL ON estadistica.api_cifras_consulta FROM PUBLIC,etf_app,etf_copia;
GRANT SELECT ON estadistica.api_cifras_consulta TO etf_app;
