-- Ensayo local: fixtures inventadas, SET ROLE etf_app y ROLLBACK obligatorio.
BEGIN;
SET LOCAL statement_timeout='30s';
SELECT set_config('app.usuario_id',(SELECT id::text FROM identidad.usuario WHERE cuenta_ad='pablo.ruiz'),true);
SELECT set_config('app.perfiles','profesional_etf',true);
SELECT set_config('app.equipos',ARRAY[(SELECT id FROM catalogo.equipo WHERE codigo='04')]::text,true);
INSERT INTO expediente.finalizacion(expediente_id,equipo_id,datos,modificado_por,modificado_en)
 SELECT id,equipo_id,'{}',current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM expediente.expediente WHERE estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.propuesta_cierre(expediente_id,equipo_id,propuesta_por,propuesta_en,
 motivo_finalizacion,catalogo_version_id,motivo_acta,motivo_spaf)
 SELECT e.id,e.equipo_id,current_setting('app.usuario_id')::uuid,clock_timestamp(),'SIN_CONSENTIMIENTO',
 (SELECT id FROM catalogo.version LIMIT 1),'SIN_CONSENTIMIENTO','SIN_CONSENTIMIENTO'
 FROM expediente.expediente e WHERE e.estado='aceptada'
 AND NOT EXISTS(SELECT 1 FROM expediente.propuesta_cierre p WHERE p.expediente_id=e.id AND p.estado='pendiente');
INSERT INTO expediente.carga_inicial(id,equipo_id,cargada_por,cargada_en,recuentos)
 SELECT gen_random_uuid(),(SELECT id FROM catalogo.equipo WHERE codigo='04'),
 current_setting('app.usuario_id')::uuid,clock_timestamp(),'{"familias":0}';
INSERT INTO expediente.adulto(id,expediente_id,equipo_id,papel,sexo,datos)
 SELECT gen_random_uuid(),e.id,e.equipo_id,'MADRE','M','{"nombre":"Teresa","apellidos":"Serrano Vílchez"}'
 FROM expediente.expediente e WHERE e.estado='aceptada'
 AND NOT EXISTS(SELECT 1 FROM expediente.adulto a WHERE a.expediente_id=e.id);
INSERT INTO expediente.familia_revision(expediente_id,equipo_id,version,vigente_desde,valor,registrado_por,registrado_en)
 SELECT e.id,e.equipo_id,1,'2026-10-14','{}',current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM expediente.expediente e WHERE e.estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.adultos_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en)
 SELECT e.id,e.equipo_id,1,'2026-10-14',current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM expediente.expediente e WHERE e.estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.adulto_revision(revision_id,adulto_id,expediente_id,equipo_id,orden,papel,sexo)
 SELECT r.id,a.id,a.expediente_id,a.equipo_id,1,a.papel,a.sexo FROM expediente.adulto a
 JOIN expediente.adultos_revision r ON r.expediente_id=a.expediente_id AND r.equipo_id=a.equipo_id
 JOIN expediente.expediente e ON e.id=a.expediente_id AND e.estado='aceptada'
 WHERE NOT EXISTS(SELECT 1 FROM expediente.adulto_revision x WHERE x.revision_id=r.id);
INSERT INTO expediente.menores_revision(expediente_id,equipo_id,version,vigente_desde,registrado_por,registrado_en)
 SELECT e.id,e.equipo_id,1,'2026-10-14',current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM expediente.expediente e WHERE e.estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.menor_revision(revision_id,menor_id,expediente_id,equipo_id,orden,sexo,subprograma)
 SELECT r.id,m.id,m.expediente_id,m.equipo_id,m.orden,m.sexo,'RIESGO_SOCIAL' FROM expediente.menor m
 JOIN expediente.menores_revision r ON r.expediente_id=m.expediente_id AND r.equipo_id=m.equipo_id
 -- Los históricos cerrados conservan sus revisiones; la prueba prepara sólo expedientes abiertos.
 JOIN expediente.expediente e ON e.id=m.expediente_id AND e.estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.marca(expediente_id,equipo_id,lista,codigo)
 SELECT e.id,e.equipo_id,'factor_menor','RIESGO' FROM expediente.expediente e WHERE e.estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.marca_menor_excluido(marca_id,menor_id,expediente_id,equipo_id)
 SELECT c.id,m.id,m.expediente_id,m.equipo_id FROM expediente.marca c
 JOIN expediente.menor m ON m.expediente_id=c.expediente_id AND m.equipo_id=c.equipo_id AND m.orden=1 ON CONFLICT DO NOTHING;
INSERT INTO expediente.intervencion_revision(expediente_id,equipo_id,version,vigente_desde,listas_conocidas,registrado_por,registrado_en)
 SELECT e.id,e.equipo_id,1,'2026-10-14',ARRAY['factor_menor'],current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM expediente.expediente e WHERE e.estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.intervencion_marca(revision_id,marca_id,expediente_id,equipo_id)
 SELECT r.id,c.id,c.expediente_id,c.equipo_id FROM expediente.marca c
 JOIN expediente.intervencion_revision r ON r.expediente_id=c.expediente_id AND r.equipo_id=c.equipo_id ON CONFLICT DO NOTHING;
INSERT INTO expediente.intervencion_exclusion(revision_id,marca_id,menor_id,expediente_id,equipo_id)
 SELECT r.revision_id,r.marca_id,m.id,m.expediente_id,m.equipo_id FROM expediente.intervencion_marca r
 JOIN expediente.menor m ON m.expediente_id=r.expediente_id AND m.equipo_id=r.equipo_id AND m.orden=1 ON CONFLICT DO NOTHING;
INSERT INTO evaluacion.instrumento_aplicado(id,expediente_id,equipo_id,datos)
 SELECT x.id,e.id,e.equipo_id,jsonb_build_object('id',x.id,'expediente_id',e.id,'equipo_id',e.equipo_id)
 FROM expediente.expediente e CROSS JOIN LATERAL (SELECT gen_random_uuid() id WHERE e.id IS NOT NULL) x WHERE e.estado='aceptada';
INSERT INTO evaluacion.aplicacion_escala(id,expediente_id,equipo_id,instrumento,momento,sujeto,datos,version)
 SELECT x.id,e.id,e.equipo_id,'APOYO_FAMILIAR','inicial','{"tipo":"familia"}',
 jsonb_build_object('id',x.id,'expediente_id',e.id,'equipo_id',e.equipo_id,'instrumento','APOYO_FAMILIAR',
 'momento','inicial','sujeto',jsonb_build_object('tipo','familia'),'version',1),1
 FROM expediente.expediente e CROSS JOIN LATERAL (SELECT gen_random_uuid() id WHERE e.id IS NOT NULL) x WHERE e.estado='aceptada';
INSERT INTO evaluacion.respuesta(aplicacion_id,expediente_id,equipo_id,item,valor_numerico)
 SELECT id,expediente_id,equipo_id,'APOYO',1 FROM evaluacion.aplicacion_escala ON CONFLICT DO NOTHING;
INSERT INTO evaluacion.peticion(usuario_id,expediente_id,equipo_id,clave,huella,resultado)
 SELECT current_setting('app.usuario_id')::uuid,e.id,e.equipo_id,'rls',repeat('a',64),'{}'
 FROM expediente.expediente e WHERE e.estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.salida_servicio(id,expediente_id,equipo_id,servicio_destino,catalogo_id,motivo,creada_por,creada_en)
 SELECT gen_random_uuid(),e.id,e.equipo_id,'SALUD',(SELECT id FROM catalogo.version LIMIT 1),
 'La familia necesita apoyo del centro de salud.',current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM expediente.expediente e WHERE e.estado='aceptada';
INSERT INTO expediente.salida_servicio_hito(salida_id,expediente_id,equipo_id,tipo,fecha,origen,confirmado_por,confirmado_en)
 SELECT s.id,s.expediente_id,s.equipo_id,'derivacion_otro_servicio','2026-10-14','documento_externo',
 current_setting('app.usuario_id')::uuid,clock_timestamp() FROM expediente.salida_servicio s ON CONFLICT DO NOTHING;
INSERT INTO expediente.salida_servicio_peticion(usuario_id,clave,equipo_id,expediente_id,ruta,huella,respuesta,creada_en)
 SELECT current_setting('app.usuario_id')::uuid,'rls_'||replace(s.id::text,'-',''),s.equipo_id,s.expediente_id,
 '/api/expedientes/'||s.expediente_id::text||'/derivaciones-salida',repeat('a',64),'{}',clock_timestamp()
 FROM expediente.salida_servicio s ON CONFLICT DO NOTHING;
INSERT INTO expediente.salida_spm(id,expediente_id,equipo_id,version,motivo,propuesta_medida,respuesta,
 creado_por,creado_en,modificado_por,modificado_en)
 SELECT gen_random_uuid(),e.id,e.equipo_id,1,'Se requiere valoración de protección.',
 'Valorar el apoyo familiar y las medidas de protección.','a_la_espera',
 current_setting('app.usuario_id')::uuid,clock_timestamp(),current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM expediente.expediente e WHERE e.estado='aceptada'
 AND NOT EXISTS(SELECT 1 FROM expediente.salida_spm s WHERE s.expediente_id=e.id);
SELECT set_config('app.usuario_id',(SELECT id::text FROM identidad.usuario WHERE cuenta_ad='ines.moreno'),true);
SELECT set_config('app.perfiles','coordinacion',true);
INSERT INTO expediente.spm_forzado(id,salida_id,expediente_id,equipo_id,version_salida,motivo,registrado_por,registrado_en)
 SELECT gen_random_uuid(),s.id,s.expediente_id,s.equipo_id,s.version,
 'La coordinación ha confirmado la atención por otro recurso.',current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM expediente.salida_spm s JOIN expediente.expediente e ON e.id=s.expediente_id AND e.estado='aceptada' ON CONFLICT DO NOTHING;
INSERT INTO expediente.peticion_spm(usuario_id,clave,ruta,huella_peticion,respuesta,creada_en)
 SELECT current_setting('app.usuario_id')::uuid,'rls_'||replace(s.id::text,'-',''),
 '/api/expedientes/'||s.expediente_id::text||'/salida-spm',repeat('a',64),'{}',clock_timestamp()
 FROM expediente.salida_spm s ON CONFLICT DO NOTHING;
INSERT INTO documento.emision
 SELECT (jsonb_populate_record(NULL::documento.emision,to_jsonb(e)||jsonb_build_object(
  'id',gen_random_uuid(),'tipo_documento',p.tipo_documento,'plantilla_id',p.id))).*
 FROM documento.emision e JOIN documento.plantilla p ON p.idioma=e.idioma
 JOIN documento.tipo_documento t ON t.codigo=p.tipo_documento AND t.requiere_registro LIMIT 1;
INSERT INTO documento.emision
 SELECT (jsonb_populate_record(NULL::documento.emision,to_jsonb(e)||jsonb_build_object(
  'id',gen_random_uuid(),'expediente_id',NULL,'tipo_documento','spaf','plantilla_id',p.id,'fecha_documento','2026-10-31'))).*
 FROM documento.emision e JOIN documento.plantilla p ON p.idioma=e.idioma AND p.tipo_documento='spaf' LIMIT 1;
INSERT INTO documento.peticion_spaf(usuario_id,clave,huella_peticion,emision_id,creada_en)
 SELECT current_setting('app.usuario_id')::uuid,'rls_'||replace(e.id::text,'-',''),e.huella_sha256,e.id,clock_timestamp()
 FROM documento.emision e WHERE tipo_documento='spaf' ON CONFLICT DO NOTHING;
INSERT INTO documento.registro_salida(emision_id,adaptador,estado,version)
 SELECT e.id,'manual','pendiente',1 FROM documento.emision e
 JOIN documento.tipo_documento t ON t.codigo=e.tipo_documento AND t.requiere_registro
 ON CONFLICT DO NOTHING;
INSERT INTO documento.registro_peticion(usuario_id,clave,emision_id,ruta,huella_peticion,respuesta,creada_en)
 SELECT generada_por,'rls_'||replace(id::text,'-',''),id,
 '/api/documentos/emisiones/'||id::text||'/registro-salida',huella_sha256,'{}',clock_timestamp()
 FROM documento.emision ON CONFLICT DO NOTHING;

INSERT INTO estadistica.memoria(anio,equipo_id,version,textos,modificado_por,modificado_en)
 SELECT 2026,equipo,1,'{}',current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM (VALUES ((SELECT id FROM catalogo.equipo WHERE codigo='04')),(NULL::uuid)) equipos(equipo)
 ON CONFLICT DO NOTHING;
INSERT INTO estadistica.memoria_narrativa_historia(memoria_id,version,textos,registrado_por,registrado_en)
 SELECT id,version,textos,modificado_por,modificado_en FROM estadistica.memoria ON CONFLICT DO NOTHING;
INSERT INTO documento.emision_memoria(id,anio,equipo_id,idioma,clave,plantilla_id,generada_por,generada_en,
 huella_peticion,huella_datos,huella_resultado,version_motor,ruta_idioma,version_idioma)
 SELECT gen_random_uuid(),2026,equipo,'es',replace(gen_random_uuid()::text,'-',''),
 (SELECT id FROM documento.plantilla WHERE tipo_documento='memoria' AND idioma='es' LIMIT 1),
 current_setting('app.usuario_id')::uuid,clock_timestamp(),decode(repeat('a',64),'hex'),
 decode(repeat('b',64),'hex'),decode(repeat('c',64),'hex'),'motor_docx_1','idiomas/es','1'
 FROM (VALUES ((SELECT id FROM catalogo.equipo WHERE codigo='04')),(NULL::uuid)) equipos(equipo);

-- Fotos y avisos inventados para las tablas de cierre de Memoria.
SELECT set_config('app.usuario_id',(SELECT id::text FROM identidad.usuario WHERE cuenta_ad='ines.moreno'),true);
SELECT set_config('app.perfiles','coordinacion',true);
INSERT INTO estadistica.memoria_acto(memoria_id,version,accion,usuario_id,ocurrido_en)
 SELECT id,version+1,'cerrar',current_setting('app.usuario_id')::uuid,clock_timestamp()
 FROM estadistica.memoria WHERE estado='borrador';
-- Conserva la secuencia si un recorrido anterior cerró y reabrió la Memoria.
UPDATE estadistica.memoria SET estado='cerrada',revision_actual=coalesce(revision_actual,0)+1,version=version+1,
 modificado_por=current_setting('app.usuario_id')::uuid,modificado_en=clock_timestamp() WHERE estado='borrador';
INSERT INTO estadistica.memoria_revision(memoria_id,revision,informe,huella,motor,cerrada_por,cerrada_en)
 SELECT id,revision_actual,'{}',repeat('a',64),'F4.06.1',modificado_por,modificado_en
 FROM estadistica.memoria ON CONFLICT DO NOTHING;
INSERT INTO estadistica.memoria_base_equipo(memoria_id,revision,equipo_id,informe)
 SELECT id,revision_actual,coalesce(equipo_id,(SELECT id FROM catalogo.equipo WHERE codigo='04')),'{}'
 FROM estadistica.memoria ON CONFLICT DO NOTHING;
INSERT INTO estadistica.memoria_discrepancia(memoria_id,revision,equipo_id,apartados,detectada_en)
 SELECT id,revision_actual,coalesce(equipo_id,(SELECT id FROM catalogo.equipo WHERE codigo='04')),ARRAY['17'],clock_timestamp()
 FROM estadistica.memoria ON CONFLICT DO NOTHING;
INSERT INTO estadistica.peticion_idempotente(usuario_id,clave,ruta,huella_peticion,respuesta,creada_en)
 SELECT id,'rls_memoria','/api/estadistica/memoria/cerrar',repeat('b',64),'{}',clock_timestamp()
 FROM identidad.usuario WHERE cuenta_ad IN ('ines.moreno','pablo.ruiz');
-- Valida las relaciones antes de alterar disparadores en este ensayo reversible.
-- Prepara un seguimiento con cierre confirmado por una persona distinta.
-- Los disparadores de negocio siguen activos durante esta preparación.
DO $$ DECLARE recurso uuid; equipo uuid; propuesta expediente.propuesta_cierre;
 episodio uuid; regla uuid; hoy date:=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date; BEGIN
 SELECT p.* INTO STRICT propuesta FROM expediente.propuesta_cierre p
 JOIN expediente.expediente e ON e.id=p.expediente_id
 WHERE e.estado='aceptada' AND e.equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo='04')
 AND p.estado='pendiente' AND p.propuesta_por<>current_setting('app.usuario_id')::uuid LIMIT 1;
 recurso:=propuesta.expediente_id; equipo:=propuesta.equipo_id;
 UPDATE expediente.expediente SET subprograma='REUNIFICACION' WHERE id=recurso;
 UPDATE expediente.propuesta_cierre SET estado='confirmada',resuelta_por=current_setting('app.usuario_id')::uuid,
 resuelta_en=clock_timestamp() WHERE id=propuesta.id;
 UPDATE expediente.episodio SET cierre_fecha=hoy,motivo_cierre=propuesta.motivo_finalizacion,
 motivo_acta=propuesta.motivo_acta,motivo_acta_texto='',motivo_spaf=propuesta.motivo_spaf,
 cerrado_por=current_setting('app.usuario_id')::uuid,cerrado_en=clock_timestamp(),propuesta_cierre_id=propuesta.id,
 forzado_id=(SELECT f.id FROM expediente.spm_forzado f JOIN expediente.salida_spm s ON s.id=f.salida_id
  WHERE s.expediente_id=recurso AND s.respuesta='a_la_espera' AND f.version_salida=s.version LIMIT 1)
 WHERE expediente_id=recurso AND cierre_fecha IS NULL RETURNING id INTO STRICT episodio;
 UPDATE expediente.expediente SET estado='cerrada',version=version+1,
 modificado_por=current_setting('app.usuario_id')::uuid,modificado_en=clock_timestamp() WHERE id=recurso;
 SELECT id INTO STRICT regla FROM catalogo.version WHERE ambito='plazos' AND estado='publicada'
 AND daterange(vigente_desde,vigente_hasta,'[)') @> hoy;
 INSERT INTO expediente.seguimiento(expediente_id,equipo_id,episodio_id,version,estado,fecha_cese,tramo_desde,
 fecha_fin,regla_id,creado_por,creado_en,modificado_por,modificado_en)
 VALUES(recurso,equipo,episodio,1,'activo',hoy,hoy,catalogo.fin_seguimiento(hoy,regla,hoy),regla,
 current_setting('app.usuario_id')::uuid,clock_timestamp(),current_setting('app.usuario_id')::uuid,clock_timestamp());
END $$;
SET CONSTRAINTS ALL IMMEDIATE;

-- Se descubren todas las tablas con datos de expediente/equipo o recibos de escritura.
CREATE TEMP TABLE tablas_f011 AS
 SELECT n.nspname esquema,c.relname tabla,c.oid,
 (SELECT count(*) FROM pg_attribute a WHERE a.attrelid=c.oid AND a.attname='equipo_id' AND NOT a.attisdropped)>0 tiene_equipo,
 0::bigint total, 0::bigint coordinacion_total, 0::bigint profesional_total, 0::bigint marta_total, NULL::jsonb fila, NULL::jsonb fila_marta
 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE c.relkind='r' AND n.nspname IN ('expediente','documento','actividad','evaluacion','estadistica')
 -- Las marcas privadas se comprueban por denegación de acceso directo, como sesión y auditoría.
 AND n.nspname||'.'||c.relname NOT IN ('expediente.conservacion','expediente.conservacion_ejecucion','documento.retirada_personal','evaluacion.retirada_personal')
 AND (EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid=c.oid AND NOT a.attisdropped
             AND a.attname IN ('equipo_id','expediente_id','emision_id','memoria_id'))
      OR c.relname LIKE 'peticion_%');
DO $$ DECLARE r record; cantidad bigint; BEGIN
 FOR r IN SELECT * FROM tablas_f011 LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=r.oid AND relrowsecurity AND relforcerowsecurity) THEN
   RAISE EXCEPTION 'RLS sin activar o forzar: %.%',r.esquema,r.tabla;
  END IF;
  EXECUTE format('SELECT count(*) FROM %I.%I',r.esquema,r.tabla) INTO cantidad;
  IF cantidad=0 THEN RAISE EXCEPTION 'Fixture vacía: %.%',r.esquema,r.tabla; END IF;
  EXECUTE format('SELECT to_jsonb(t) FROM %I.%I t LIMIT 1',r.esquema,r.tabla) INTO r.fila;
  -- Aísla las políticas RLS de los disparadores de negocio; ROLLBACK restaura todo.
  EXECUTE format('ALTER TABLE %I.%I DISABLE TRIGGER USER',r.esquema,r.tabla);
  UPDATE tablas_f011 SET total=cantidad,coordinacion_total=cantidad,fila=r.fila WHERE oid=r.oid;
  IF r.esquema||'.'||r.tabla IN ('expediente.peticion_idempotente','documento.peticion_idempotente','documento.registro_peticion','documento.peticion_spaf','expediente.peticion_spm','expediente.salida_servicio_peticion','evaluacion.peticion','estadistica.peticion_idempotente') THEN

   EXECUTE format('SELECT count(*) FROM %I.%I WHERE usuario_id=(SELECT id FROM identidad.usuario WHERE cuenta_ad=''ines.moreno'')',r.esquema,r.tabla) INTO cantidad;
   UPDATE tablas_f011 SET coordinacion_total=cantidad WHERE oid=r.oid;
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE usuario_id=(SELECT id FROM identidad.usuario WHERE cuenta_ad=''pablo.ruiz'')',r.esquema,r.tabla) INTO r.profesional_total;
  ELSIF r.esquema||'.'||r.tabla='estadistica.memoria_discrepancia' THEN
   r.profesional_total:=0;
  ELSIF r.esquema||'.'||r.tabla IN ('estadistica.memoria_revision','estadistica.memoria_base_equipo','estadistica.memoria_acto') THEN
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE memoria_id IN (SELECT id FROM estadistica.memoria WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04''))',r.esquema,r.tabla) INTO r.profesional_total;
  ELSIF r.esquema||'.'||r.tabla='expediente.expediente' THEN
   -- Un histórico sin número conserva su ámbito en el equipo de carga.
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'') OR (origen=''carga_inicial'' AND equipo_carga_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04''))',r.esquema,r.tabla) INTO r.profesional_total;
  ELSIF r.tiene_equipo AND r.tabla<>'contador_expediente' THEN
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'')',r.esquema,r.tabla) INTO r.profesional_total;
  ELSIF r.tabla IN ('lectura_anexo','derivacion_datos') THEN
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE expediente_id IN (SELECT id FROM expediente.expediente WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'') OR (origen=''carga_inicial'' AND equipo_carga_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'')))',r.esquema,r.tabla) INTO r.profesional_total;
  ELSIF r.esquema||'.'||r.tabla='estadistica.memoria_narrativa_historia' THEN
   SELECT count(*) INTO r.profesional_total FROM estadistica.memoria_narrativa_historia
    WHERE memoria_id IN (SELECT id FROM estadistica.memoria WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo='04'));
  END IF;
  IF r.tabla='salida_servicio_peticion' OR r.esquema||'.'||r.tabla='evaluacion.peticion' THEN
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE usuario_id=(SELECT id FROM identidad.usuario WHERE cuenta_ad=''pablo.ruiz'') AND equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'')',r.esquema,r.tabla) INTO r.profesional_total;
  END IF;
  IF r.esquema||'.'||r.tabla IN ('expediente.peticion_idempotente','documento.peticion_idempotente','documento.registro_peticion','documento.peticion_spaf','expediente.peticion_spm','expediente.salida_servicio_peticion','evaluacion.peticion','estadistica.peticion_idempotente') THEN
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE usuario_id=(SELECT id FROM identidad.usuario WHERE cuenta_ad=''ines.moreno'')',r.esquema,r.tabla) INTO cantidad;
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE usuario_id=(SELECT id FROM identidad.usuario WHERE cuenta_ad=''marta.cabrera'')',r.esquema,r.tabla) INTO r.marta_total;
  ELSIF r.esquema||'.'||r.tabla='estadistica.memoria_discrepancia' THEN
   r.marta_total:=0;
  ELSIF r.esquema||'.'||r.tabla IN ('estadistica.memoria_revision','estadistica.memoria_base_equipo','estadistica.memoria_acto') THEN
   -- Un recorrido de Memoria puede haber creado revisiones del equipo de Marta.
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE memoria_id IN (SELECT id FROM estadistica.memoria WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''02''))',r.esquema,r.tabla) INTO r.marta_total;
  ELSIF r.esquema||'.'||r.tabla='expediente.expediente' THEN
   -- Un histórico sin número conserva su ámbito en el equipo de carga.
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''02'') OR (origen=''carga_inicial'' AND equipo_carga_id=(SELECT id FROM catalogo.equipo WHERE codigo=''02''))',r.esquema,r.tabla) INTO r.marta_total;
  ELSIF r.tiene_equipo AND r.tabla<>'contador_expediente' THEN
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''02'')',r.esquema,r.tabla) INTO r.marta_total;
  ELSIF r.tabla IN ('lectura_anexo','derivacion_datos') THEN
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE expediente_id IN (SELECT id FROM expediente.expediente WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''02'') OR (origen=''carga_inicial'' AND equipo_carga_id=(SELECT id FROM catalogo.equipo WHERE codigo=''02'')))',r.esquema,r.tabla) INTO r.marta_total;
  ELSIF r.esquema||'.'||r.tabla='estadistica.memoria_narrativa_historia' THEN
   SELECT count(*) INTO r.marta_total FROM estadistica.memoria_narrativa_historia
    WHERE memoria_id IN (SELECT id FROM estadistica.memoria WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo='02'));
  END IF;
  IF r.tabla='salida_servicio_peticion' OR r.esquema||'.'||r.tabla='evaluacion.peticion' THEN
   EXECUTE format('SELECT count(*) FROM %I.%I WHERE usuario_id=(SELECT id FROM identidad.usuario WHERE cuenta_ad=''marta.cabrera'') AND equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''02'')',r.esquema,r.tabla) INTO r.marta_total;
  END IF;
  -- Los recibos del registro de salida son exclusivos de Coordinación.
  IF r.esquema||'.'||r.tabla='documento.registro_peticion' THEN
   r.profesional_total:=0;
   r.marta_total:=0;
  END IF;
  UPDATE tablas_f011 SET marta_total=r.marta_total WHERE oid=r.oid;
  UPDATE tablas_f011 SET profesional_total=r.profesional_total WHERE oid=r.oid;
  IF r.esquema||'.'||r.tabla IN ('estadistica.memoria_revision','estadistica.memoria_base_equipo','estadistica.memoria_acto','estadistica.memoria_narrativa_historia') THEN
   -- La escritura negativa usa una Memoria ajena, incluso tras grabar el ETF 02.
   EXECUTE format('SELECT to_jsonb(t) FROM %I.%I t WHERE memoria_id IN (SELECT id FROM estadistica.memoria WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'')) LIMIT 1',r.esquema,r.tabla) INTO r.fila_marta;
  ELSIF r.tiene_equipo THEN
   EXECUTE format('SELECT to_jsonb(t) FROM %I.%I t WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'') LIMIT 1',r.esquema,r.tabla) INTO r.fila_marta;
  ELSIF r.esquema||'.'||r.tabla IN ('expediente.peticion_idempotente','documento.peticion_idempotente','documento.registro_peticion','documento.peticion_spaf','expediente.peticion_spm','expediente.salida_servicio_peticion','evaluacion.peticion','estadistica.peticion_idempotente') THEN
   EXECUTE format('SELECT to_jsonb(t) FROM %I.%I t WHERE usuario_id<>(SELECT id FROM identidad.usuario WHERE cuenta_ad=''marta.cabrera'') LIMIT 1',r.esquema,r.tabla) INTO r.fila_marta;
  ELSIF r.tabla IN ('lectura_anexo','derivacion_datos') THEN
   EXECUTE format('SELECT to_jsonb(t) FROM %I.%I t WHERE expediente_id IN (SELECT id FROM expediente.expediente WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'')) LIMIT 1',r.esquema,r.tabla) INTO r.fila_marta;
  ELSE r.fila_marta:=r.fila;
  END IF;
  IF r.fila_marta IS NULL THEN RAISE EXCEPTION 'Falta fila ajena para Marta: %.%',r.esquema,r.tabla; END IF;
  UPDATE tablas_f011 SET fila_marta=r.fila_marta WHERE oid=r.oid;

 END LOOP;
END $$;
CREATE TEMP TABLE vistas_f011 AS
 SELECT c.relname nombre,c.oid,0::bigint total,0::bigint profesional_total,0::bigint marta_total
 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='expediente' AND c.relkind='v' AND c.relname LIKE 'api_estadistica_%';
DO $$ DECLARE r record; cantidad bigint; BEGIN
 IF NOT EXISTS(SELECT 1 FROM vistas_f011) THEN RAISE EXCEPTION 'Faltan vistas estadísticas'; END IF;
 FOR r IN SELECT * FROM vistas_f011 LOOP
  IF NOT (SELECT coalesce(reloptions,'{}') @> ARRAY['security_invoker=true','security_barrier=true'] FROM pg_class WHERE oid=r.oid) THEN
   RAISE EXCEPTION 'Vista sin barreras: %',r.nombre;
  END IF;
  EXECUTE format('SELECT count(*) FROM expediente.%I',r.nombre) INTO cantidad;
  IF cantidad=0 THEN RAISE EXCEPTION 'Vista sin filas: %',r.nombre; END IF;
  UPDATE vistas_f011 SET total=cantidad WHERE oid=r.oid;
  EXECUTE format('SELECT count(*) FROM expediente.%I WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''04'')',r.nombre) INTO cantidad;
  UPDATE vistas_f011 SET profesional_total=cantidad WHERE oid=r.oid;
  EXECUTE format('SELECT count(*) FROM expediente.%I WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo=''02'')',r.nombre) INTO cantidad;
  UPDATE vistas_f011 SET marta_total=cantidad WHERE oid=r.oid;
 END LOOP;
END $$;
CREATE TEMP TABLE resultado_vistas_f011(perfil text,vista text,lectura bigint);
CREATE TEMP TABLE actores_f011 AS
 SELECT cuenta_ad AS cuenta,id::text usuario,
 CASE cuenta_ad WHEN 'ines.moreno' THEN 'coordinacion' WHEN 'elena.salas' THEN 'administracion'
 WHEN 'lucia.navarro' THEN 'consulta' WHEN 'sergio.campos' THEN '' ELSE 'profesional_etf' END perfil,
 CASE cuenta_ad WHEN 'marta.cabrera' THEN ARRAY[(SELECT id FROM catalogo.equipo WHERE codigo='02')]::text
 WHEN 'pablo.ruiz' THEN ARRAY[(SELECT id FROM catalogo.equipo WHERE codigo='04')]::text ELSE '{}' END equipos
 FROM identidad.usuario WHERE cuenta_ad IN ('ines.moreno','elena.salas','lucia.navarro','sergio.campos','marta.cabrera','pablo.ruiz')
 UNION ALL SELECT 'anonimo','','','{}';
CREATE TEMP TABLE resultado_f011(perfil text, tabla text, lectura bigint, escrituras_denegadas integer);
GRANT SELECT ON tablas_f011,vistas_f011,actores_f011 TO etf_app;
GRANT INSERT ON resultado_vistas_f011 TO etf_app;
GRANT INSERT,SELECT ON resultado_f011 TO etf_app;
SET LOCAL ROLE etf_app;
DO $$ BEGIN
 IF current_user<>'etf_app' OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND (rolsuper OR rolbypassrls)) THEN
  RAISE EXCEPTION 'El ensayo no ejecuta RLS como etf_app';
 END IF;
END $$;
DO $$ DECLARE a record; r record; cantidad bigint; afectadas bigint; columna text; denegadas integer; esperado bigint; fila_ajena jsonb; BEGIN
 FOR a IN SELECT * FROM actores_f011 LOOP
  PERFORM set_config('app.usuario_id',a.usuario,true);
  PERFORM set_config('app.perfiles',a.perfil,true);
  PERFORM set_config('app.equipos',a.equipos,true);
  SELECT count(*) INTO cantidad FROM identidad.usuario WHERE id::text<>a.usuario;
  IF (a.perfil='administracion' AND cantidad<>6) OR (a.perfil<>'administracion' AND cantidad<>0) THEN
   RAISE EXCEPTION 'Identidad ajena visible: % filas %',a.cuenta,cantidad;
  END IF;
  FOREACH columna IN ARRAY ARRAY['identidad.sesion','auditoria.evento','auditoria.sello_diario','expediente.conservacion','expediente.conservacion_ejecucion','documento.retirada_personal','evaluacion.retirada_personal'] LOOP
   BEGIN
    EXECUTE format('SELECT count(*) FROM %s',columna) INTO cantidad;
    RAISE EXCEPTION 'Lectura directa protegida permitida: % %',a.cuenta,columna;
   EXCEPTION WHEN insufficient_privilege THEN NULL;
   END;
   IF columna IN ('expediente.conservacion','expediente.conservacion_ejecucion','documento.retirada_personal','evaluacion.retirada_personal') THEN
    IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=columna::regclass AND relrowsecurity AND relforcerowsecurity)
    OR has_table_privilege(columna,'INSERT,UPDATE,DELETE,TRUNCATE') OR has_any_column_privilege(columna,'INSERT,UPDATE') THEN
     RAISE EXCEPTION 'Permisos o RLS de marca privada: % %',a.cuenta,columna;
    END IF;
   END IF;
  END LOOP;
  FOR r IN SELECT * FROM tablas_f011 LOOP
   EXECUTE format('SELECT count(*) FROM %I.%I',r.esquema,r.tabla) INTO cantidad;
   esperado := 0;
   IF a.perfil='coordinacion' THEN esperado:=r.coordinacion_total; END IF;
   IF a.cuenta='pablo.ruiz' THEN esperado:=r.profesional_total; END IF;
   IF a.cuenta='marta.cabrera' THEN esperado:=r.marta_total; END IF;
   IF cantidad<>esperado THEN
    RAISE EXCEPTION 'Lectura RLS: % %.% filas %, esperadas %',a.cuenta,r.esquema,r.tabla,cantidad,esperado;
   END IF;
   denegadas:=0;
   IF a.cuenta NOT IN ('ines.moreno','pablo.ruiz') THEN
    -- La fila procede del propietario, antes de aplicar RLS; no del SELECT filtrado.
    fila_ajena:=CASE WHEN a.cuenta='marta.cabrera' THEN r.fila_marta ELSE r.fila END;
    BEGIN
     EXECUTE format('INSERT INTO %I.%I SELECT * FROM jsonb_populate_record(NULL::%I.%I,$1)',r.esquema,r.tabla,r.esquema,r.tabla) USING fila_ajena;
     RAISE EXCEPTION 'INSERT permitido: % %.%',a.cuenta,r.esquema,r.tabla;
    EXCEPTION WHEN insufficient_privilege THEN denegadas:=denegadas+1;
    END;
    SELECT attname INTO columna FROM pg_attribute WHERE attrelid=r.oid AND attnum>0 AND NOT attisdropped
     AND has_column_privilege(r.oid,attname,'UPDATE') ORDER BY attnum LIMIT 1;
    IF columna IS NOT NULL THEN
     BEGIN
      IF a.cuenta='marta.cabrera' THEN
       EXECUTE format('UPDATE %I.%I t SET %I=t.%I WHERE to_jsonb(t)=$1',r.esquema,r.tabla,columna,columna) USING fila_ajena;
      ELSE
       EXECUTE format('UPDATE %I.%I SET %I=%I',r.esquema,r.tabla,columna,columna);
      END IF;
      GET DIAGNOSTICS afectadas=ROW_COUNT;
      IF afectadas<>0 THEN RAISE EXCEPTION 'Escritura RLS: % %.%',a.cuenta,r.esquema,r.tabla; END IF;
      denegadas:=denegadas+1;
     EXCEPTION WHEN insufficient_privilege THEN denegadas:=denegadas+1;
     END;
    ELSE denegadas:=denegadas+1;
    END IF;
    BEGIN
     IF a.cuenta='marta.cabrera' THEN
      EXECUTE format('DELETE FROM %I.%I t WHERE to_jsonb(t)=$1',r.esquema,r.tabla) USING fila_ajena;
     ELSE
      EXECUTE format('DELETE FROM %I.%I',r.esquema,r.tabla);
     END IF;
     GET DIAGNOSTICS afectadas=ROW_COUNT;
     IF afectadas<>0 THEN RAISE EXCEPTION 'Borrado RLS: % %.%',a.cuenta,r.esquema,r.tabla; END IF;
     denegadas:=denegadas+1;
    EXCEPTION WHEN insufficient_privilege THEN denegadas:=denegadas+1;
    END;
   END IF;
   INSERT INTO resultado_f011 VALUES(a.cuenta,r.esquema||'.'||r.tabla,cantidad,denegadas);
  END LOOP;
  FOR r IN SELECT * FROM vistas_f011 LOOP
   EXECUTE format('SELECT count(*) FROM expediente.%I',r.nombre) INTO cantidad;
   esperado:=0;
   IF a.perfil='coordinacion' THEN esperado:=r.total; END IF;
   IF a.cuenta='pablo.ruiz' THEN esperado:=r.profesional_total; END IF;
   IF a.cuenta='marta.cabrera' THEN esperado:=r.marta_total; END IF;
   IF cantidad<>esperado THEN RAISE EXCEPTION 'Lectura de vista RLS: % % filas %, esperadas %',a.cuenta,r.nombre,cantidad,esperado; END IF;
   INSERT INTO resultado_vistas_f011 VALUES(a.cuenta,r.nombre,cantidad);
  END LOOP;
 END LOOP;
END $$;
RESET ROLE;
SELECT 'F0.11-RLS-OK: '||count(DISTINCT perfil)||' actores; '||count(DISTINCT tabla)||
 ' tablas pobladas; '||count(*)||' comprobaciones de lectura; '||sum(escrituras_denegadas)||' escrituras denegadas.' FROM resultado_f011;
SELECT 'F0.11-VISTAS-OK: '||count(DISTINCT vista)||' vistas; '||count(*)||' lecturas con RLS.' FROM resultado_vistas_f011;
ROLLBACK;
