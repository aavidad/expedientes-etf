"""Datos inventados y aislados para el recorrido HTTP de consentimientos."""

def sembrar_consentimientos(sql, orden):
    sentencias = ["BEGIN;"]
    for indice, origen, subprograma in ((1, 1, 'RIESGO_SOCIAL'),
                                      (2, 101, 'REUNIFICACION'),
                                      (3, 1, 'RIESGO_SOCIAL'),
                                      (4, 101, 'REUNIFICACION')):
        ident = f'a2030000-0000-4000-8000-{indice:012d}'
        fuente = f'77777777-7777-4777-8777-{origen:012d}'
        sentencias.append(f"""
INSERT INTO expediente.expediente(id,numero,equipo_id,municipio_id,estado,fecha_recepcion,version,centro_ss_id,anio_numero,correlativo,regla_numeracion_id,fecha_aceptacion,subprograma,profesional_referencia_puesto,descripcion_situacion,fase_intervencion,modificado_por,modificado_en)
 SELECT '{ident}','06-{50+indice}-26',equipo_id,municipio_id,'aceptada',fecha_recepcion,2,centro_ss_id,anio_numero,{50+indice},regla_numeracion_id,fecha_aceptacion,'{subprograma}',profesional_referencia_puesto,descripcion_situacion,fase_intervencion,modificado_por,modificado_en
 FROM expediente.expediente WHERE id='{fuente}';
INSERT INTO expediente.lectura_anexo(id,expediente_id,formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en)
 SELECT gen_random_uuid(),'{ident}',formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en
 FROM expediente.lectura_anexo WHERE expediente_id='{fuente}';
INSERT INTO expediente.derivacion_datos(expediente_id,esquema_version,contenido)
 SELECT '{ident}',esquema_version,contenido FROM expediente.derivacion_datos WHERE expediente_id='{fuente}';
INSERT INTO expediente.menor(expediente_id,equipo_id,orden,sexo)
 SELECT '{ident}',equipo_id,orden,sexo FROM expediente.menor WHERE expediente_id='{fuente}';
INSERT INTO expediente.episodio(expediente_id,equipo_id,apertura_fecha,creado_por,creado_en)
 SELECT id,equipo_id,fecha_aceptacion,modificado_por,modificado_en FROM expediente.expediente WHERE id='{ident}';
INSERT INTO expediente.hito(expediente_id,equipo_id,episodio_id,tipo,fecha,origen,confirmado_por,confirmado_en)
 SELECT e.id,e.equipo_id,p.id,'aceptacion',e.fecha_aceptacion,'documento_externo',e.modificado_por,TIMESTAMPTZ '2026-10-13 09:00:00+00'
 FROM expediente.expediente e JOIN expediente.episodio p ON p.expediente_id=e.id WHERE e.id='{ident}';
""")
    sentencias.append('COMMIT;')
    orden(sql, '\n'.join(sentencias))


def activar_catalogos_consentimientos(sql, orden, raiz, destino):
    import hashlib
    import json
    import shutil

    shutil.copytree(raiz / 'idiomas', destino)
    for idioma in ('es', 'en'):
        nombres = {}
        for archivo in sorted((destino / idioma / 'catalogos').glob('*.json')):
            nombres.update(json.loads(archivo.read_text()))
        nombres['municipio.18013'] = 'Alhama de Granada (Granada)'
        contenido = json.dumps(nombres, ensure_ascii=False, sort_keys=True).encode()
        huella = hashlib.sha256(contenido).hexdigest()
        carpeta = destino / idioma / 'catalogos' / '.versiones'
        carpeta.mkdir(exist_ok=True)
        (carpeta / (huella + '.json')).write_bytes(contenido)
        ruta = f'{idioma}/catalogos/.versiones/{huella}.json'
        orden(sql, f"""BEGIN;
SELECT set_config('app.usuario_id','11111111-1111-4111-8111-111111111111',true);
SELECT set_config('app.perfiles','administracion',true);
SELECT paquete.activar_catalogo('{idioma}','{huella}','{ruta}','{huella}',0);
COMMIT;""")
    # PostgreSQL ejecuta las pruebas sin privilegios: sólo lectura del fixture.
    for carpeta in [destino, *destino.rglob('*')]:
        carpeta.chmod(0o755 if carpeta.is_dir() else 0o644)
