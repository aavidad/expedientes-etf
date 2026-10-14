"""Datos inventados exclusivos del recorrido de cierre y Acta."""

def sembrar_cierre(sql, orden):
    orden(sql, """
BEGIN;
INSERT INTO identidad.usuario(id,object_guid_ad,cuenta_ad,nombre_mostrado,estado,primer_acceso_en,ultimo_acceso_en)
VALUES('11111111-1111-4111-8111-111111111113','11111111-1111-4111-8111-111111111114','elena.nunez','Elena Núñez Romero','activo',clock_timestamp(),clock_timestamp());
INSERT INTO expediente.expediente(id,numero,equipo_id,municipio_id,estado,fecha_recepcion,version,centro_ss_id,anio_numero,correlativo,regla_numeracion_id,fecha_aceptacion,subprograma,profesional_referencia_puesto,descripcion_situacion,fase_intervencion,modificado_por,modificado_en)
 SELECT '77777777-7777-4777-8777-000000000501','06-1501-26',equipo_id,municipio_id,estado,fecha_recepcion,version,centro_ss_id,anio_numero,1501,regla_numeracion_id,fecha_aceptacion,subprograma,profesional_referencia_puesto,descripcion_situacion,fase_intervencion,modificado_por,modificado_en
 FROM expediente.expediente WHERE id='77777777-7777-4777-8777-000000000015';
INSERT INTO expediente.lectura_anexo(id,expediente_id,formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en)
 SELECT gen_random_uuid(),'77777777-7777-4777-8777-000000000501',formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en FROM expediente.lectura_anexo WHERE expediente_id='77777777-7777-4777-8777-000000000015';
INSERT INTO expediente.derivacion_datos(expediente_id,esquema_version,contenido)
 SELECT '77777777-7777-4777-8777-000000000501',esquema_version,contenido FROM expediente.derivacion_datos WHERE expediente_id='77777777-7777-4777-8777-000000000015';
INSERT INTO expediente.menor(expediente_id,equipo_id,orden,sexo)
 SELECT '77777777-7777-4777-8777-000000000501',equipo_id,orden,sexo FROM expediente.menor WHERE expediente_id='77777777-7777-4777-8777-000000000015';
INSERT INTO expediente.episodio(expediente_id,equipo_id,apertura_fecha,creado_por,creado_en)
 SELECT id,equipo_id,fecha_aceptacion,modificado_por,modificado_en FROM expediente.expediente WHERE id='77777777-7777-4777-8777-000000000501';
COMMIT;
""")
