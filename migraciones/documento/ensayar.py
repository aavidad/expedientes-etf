#!/usr/bin/env python3
"""Ensaya documento en PostgreSQL 18 desechable con datos inventados."""
import os
import sys
from functools import partial
import json
import pathlib
import secrets
import shutil
import subprocess
import tempfile

from sembrar_consentimientos import (sembrar_consentimientos,
                                      activar_catalogos_consentimientos)

RAIZ = pathlib.Path(__file__).resolve().parents[2]
IMAGEN = 'postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15'


sys.path.insert(0, str(RAIZ / 'scripts' / 'ci'))
from ensayos import orden as ejecutar_orden, esperar_postgres, recortar_dependientes

orden = partial(ejecutar_orden, modulo='documento')


def sembrar_entradas(sql):
    nombres = ['Nerea', 'Álvaro', 'Celia', 'Mateo', 'Irene', 'Hugo', 'Paula', 'Bruno',
               'Aitana', 'Leo', 'Claudia', 'Mario', 'Vera', 'Lucas', 'Eva']
    sentencias = ["BEGIN;", "SELECT set_config('app.usuario_id','11111111-1111-4111-8111-111111111111',true); SELECT set_config('app.perfiles','coordinacion',true);", """INSERT INTO catalogo.firmante(id,version,tipo,rol,nombre,cargo_codigo,organismo_codigo,direccion,vigente_desde,creada_por)
VALUES('33333333-3333-4333-8333-333333333334',1,'destinatario','jefe_spm','Clara Serrano Rivas','JEFATURA','SPM','',DATE '2026-01-01','11111111-1111-4111-8111-111111111111');"""]
    for origen in ('anexo3', 'anexo4'):
        for total in (1, 6, 15):
            indice = (0 if origen == 'anexo3' else 100) + total
            ident = f'77777777-7777-4777-8777-{indice:012d}'
            campos = []
            def campo(codigo, valor):
                campos.append({'codigo': codigo, 'valor': valor, 'pagina': 1, 'dudoso': False})
            for codigo, valor in [('servicio.nombre', 'Centro de Servicios Sociales Comunitarios de Alhama'),
                                   ('familia.direccion', 'Calle del Laurel 18, Alhama de Granada'),
                                   ('familia.contacto.nombre', 'Lucía'), ('familia.contacto.apellidos', 'Ortega Salas')]:
                campo(codigo, valor)
            for papel, nombre in [('Madre','Lucía Ortega Salas'), ('Padre','Daniel Pérez Molina'),
                                  ('Abuela','Rosa Salas Ávila'), ('Tío','Pablo Ortega Vega'),
                                  ('Tía','Marta Gil Ruiz'), ('Abuelo','Luis Salas Ríos'), ('Tía','Elena Pérez Gil')]:
                campo('sociodemografia.adulto.nombre', nombre)
                campo('sociodemografia.adulto.parentesco', papel)
            for i in range(total):
                campo('menores.persona.nombre', nombres[i]+' Ortega Salas')
                campo('menores.persona.nacimiento', f'{i+1:02d}/05/2016')
            datos = json.dumps({'esquema_version': '1', 'contenido': {'campos': campos, 'tipo': origen,
                              'version': 'cliente-2026-v1', 'plantilla_reconocida': True}}, ensure_ascii=False)
            sentencias.append(f"""
INSERT INTO expediente.expediente(id,numero,equipo_id,municipio_id,estado,fecha_recepcion,version,centro_ss_id,anio_numero,correlativo,regla_numeracion_id,fecha_aceptacion,subprograma,profesional_referencia_puesto,descripcion_situacion,fase_intervencion,modificado_por,modificado_en)
 SELECT '{ident}','06-{indice:02d}-26',e.id,m.id,'aceptada',clock_timestamp(),2,c.id,2026,{indice},r.version_id,DATE '2026-10-13','RIESGO_SOCIAL','TS','Requiere apoyo familiar y coordinación educativa','recepcion','11111111-1111-4111-8111-111111111111',clock_timestamp()
 FROM catalogo.equipo e,catalogo.centro_ss c,catalogo.municipio m,catalogo.regla_numeracion r
 WHERE e.codigo='06' AND c.codigo='ALHAMA' AND m.ine='18013' LIMIT 1;
INSERT INTO expediente.lectura_anexo(id,expediente_id,formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en)
 VALUES(gen_random_uuid(),'{ident}','docx',1000,decode(repeat('a',64),'hex'),'limpio','','cliente-2026-v1','{{}}','11111111-1111-4111-8111-111111111111',clock_timestamp());
INSERT INTO expediente.derivacion_datos(expediente_id,esquema_version,contenido) VALUES('{ident}','1',$datos${datos}$datos$);
INSERT INTO expediente.menor(expediente_id,equipo_id,orden,sexo)
 SELECT '{ident}',equipo_id,s,'M' FROM expediente.expediente e CROSS JOIN generate_series(1,{total}) s WHERE e.id='{ident}';
""")
            sentencias.append(f"""
INSERT INTO expediente.episodio(expediente_id,equipo_id,apertura_fecha,creado_por,creado_en)
 SELECT id,equipo_id,fecha_aceptacion,modificado_por,modificado_en FROM expediente.expediente WHERE id='{ident}';
INSERT INTO expediente.hito(expediente_id,equipo_id,episodio_id,tipo,fecha,origen,confirmado_por,confirmado_en)
 SELECT e.id,e.equipo_id,p.id,'aceptacion',e.fecha_aceptacion,'documento_externo',e.modificado_por,e.modificado_en
 FROM expediente.expediente e JOIN expediente.episodio p ON p.expediente_id=e.id WHERE e.id='{ident}';
""")
    # Las decisiones de F1.05b no asignan equipo, municipio ni número.
    for estado, base in [('no_aceptada', 200), ('lista_espera', 300)]:
        for total in (1, 6, 15):
            origenes = ('anexo3', 'anexo4') if estado == 'no_aceptada' else ('anexo3',)
            for origen in origenes:
                indice = base + total + (100 if origen == 'anexo4' else 0)
                prefijo = '88888888' if estado == 'no_aceptada' else '99999999'
                ident = f'{prefijo}-7777-4777-8777-{indice:012d}'
                fuente = f'77777777-7777-4777-8777-{total+(100 if origen == "anexo4" else 0):012d}'
                subprograma = 'REUNIFICACION' if origen == 'anexo4' else 'RIESGO_SOCIAL'
                decision = f"fecha_no_aceptacion=DATE '2026-10-13',causa_no_aceptacion='La familia recibe atención especializada de otro recurso.',subprograma='{subprograma}'" if estado == 'no_aceptada' else "fecha_lista_espera=DATE '2026-10-13'"
                sentencias.append(f"""
INSERT INTO expediente.expediente(id,fecha_recepcion)
 VALUES('{ident}',TIMESTAMPTZ '2026-10-06 22:30:00+00');
INSERT INTO expediente.lectura_anexo(id,expediente_id,formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en)
 SELECT gen_random_uuid(),'{ident}',formato,tamano_bytes,huella_sha256,antivirus,antivirus_detalle,plantilla_detectada,campos_dudosos,leida_por,leida_en FROM expediente.lectura_anexo WHERE expediente_id='{fuente}';
INSERT INTO expediente.derivacion_datos(expediente_id,esquema_version,contenido)
 SELECT '{ident}',esquema_version,contenido FROM expediente.derivacion_datos WHERE expediente_id='{fuente}';
UPDATE expediente.expediente SET estado='{estado}',version=2,{decision},modificado_por='11111111-1111-4111-8111-111111111111',modificado_en=clock_timestamp() WHERE id='{ident}';
""")
    sentencias.append('COMMIT;')
    orden(sql, '\n'.join(sentencias))

def sembrar_catalogo_anexo6(sql):
    orden(sql, """
SET ROLE etf_propietario;
INSERT INTO catalogo.version(id,ambito,numero,vigente_desde,motivo,creada_por)
SELECT '66666666-6666-4666-8666-666666666666','listas',coalesce(max(numero),0)+1,DATE '2026-10-13','catalogo.edicion','00000000-0000-4000-8000-000000000001' FROM catalogo.version WHERE ambito='listas';
INSERT INTO catalogo.lista_opcion(version_id,lista,codigo,orden,admite_texto,activa)
SELECT '66666666-6666-4666-8666-666666666666',lista,codigo,orden,admite_texto,activa FROM catalogo.lista_opcion o JOIN catalogo.version v ON v.id=o.version_id WHERE v.ambito='listas' AND v.estado='publicada' AND v.vigente_hasta IS NULL;
INSERT INTO catalogo.lista_opcion VALUES('66666666-6666-4666-8666-666666666666','servicio_destino','SALUD',1,false,true);
SELECT catalogo.publicar('66666666-6666-4666-8666-666666666666','00000000-0000-4000-8000-000000000001');
RESET ROLE;
""")


def sembrar_anexo6(sql):
    sentencias = []
    for total in (1, 6, 15):
        destino=f'aaaaaaaa-7777-4777-8777-{total:012d}'
        origen=f'77777777-7777-4777-8777-{total:012d}'
        sentencias.append(f"""
INSERT INTO expediente.expediente SELECT (jsonb_populate_record(NULL::expediente.expediente,to_jsonb(e)||jsonb_build_object('id','{destino}','numero','06-{500+total}-26','correlativo',{500+total}))).* FROM expediente.expediente e WHERE id='{origen}';
INSERT INTO expediente.lectura_anexo SELECT (jsonb_populate_record(NULL::expediente.lectura_anexo,to_jsonb(l)||jsonb_build_object('id',gen_random_uuid(),'expediente_id','{destino}'))).* FROM expediente.lectura_anexo l WHERE expediente_id='{origen}';
INSERT INTO expediente.derivacion_datos SELECT (jsonb_populate_record(NULL::expediente.derivacion_datos,to_jsonb(d)||jsonb_build_object('expediente_id','{destino}'))).* FROM expediente.derivacion_datos d WHERE expediente_id='{origen}';
INSERT INTO expediente.menor SELECT (jsonb_populate_record(NULL::expediente.menor,to_jsonb(m)||jsonb_build_object('id',gen_random_uuid(),'expediente_id','{destino}'))).* FROM expediente.menor m WHERE expediente_id='{origen}';
""")
    orden(sql, '\n'.join(sentencias))


def ensayar_auditoria_plantillas(sql):
    orden(sql, """BEGIN;
SET LOCAL ROLE etf_propietario;
SELECT set_config('app.usuario_id','11111111-1111-4111-8111-111111111111',true);
SELECT set_config('app.perfiles','administracion',true);
SELECT * FROM documento.guardar_version_plantilla(
 '99999999-9999-4999-8999-999999999991','entrada/aceptacion','es','99.98.97',
 'plantillas/entrada/aceptacion/.versiones/99999999-9999-4999-8999-999999999991/v99.98.97.docx',
 repeat('a',64),(SELECT version FROM documento.referencia_activa WHERE tipo_documento='entrada/aceptacion' AND idioma='es'));
CREATE FUNCTION pg_temp.pl1_fallo_auditoria() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN IF NEW.accion='documento.plantilla_activar' THEN RAISE EXCEPTION 'prueba.auditoria' USING ERRCODE='P7771'; END IF; RETURN NEW; END $$;
REVOKE EXECUTE ON FUNCTION pg_temp.pl1_fallo_auditoria() FROM PUBLIC;
CREATE TRIGGER pl1_fallo_auditoria BEFORE INSERT ON auditoria.evento FOR EACH ROW EXECUTE FUNCTION pg_temp.pl1_fallo_auditoria();
SET LOCAL ROLE etf_app;
DO $$ BEGIN
 PERFORM documento.activar_version_plantilla('99999999-9999-4999-8999-999999999991',
 (SELECT version FROM documento.referencia_activa WHERE tipo_documento='entrada/aceptacion' AND idioma='es'));
 RAISE EXCEPTION 'prueba.no_rechazada' USING ERRCODE='P7772';
EXCEPTION WHEN SQLSTATE 'P7771' THEN NULL; END $$;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM documento.referencia_activa WHERE plantilla_id='99999999-9999-4999-8999-999999999991')
 OR EXISTS(SELECT 1 FROM documento.activacion_plantilla WHERE plantilla_id='99999999-9999-4999-8999-999999999991') THEN
 RAISE EXCEPTION 'prueba.escritura_parcial'; END IF;
END $$;
ROLLBACK;""")
    print('PL1-AUDITORIA-OK: fallo de auditoría revierte referencia e historial', flush=True)


def ensayar_actualizacion_plv(sql):
    # El ensayo se revierte completo: no crea usuarios ni decisiones en la base
    # que después usa las pruebas de instalación limpia.
    preparacion = """SET SESSION AUTHORIZATION etf_migrador;
BEGIN;
SET LOCAL ROLE etf_propietario;
INSERT INTO identidad.usuario(id,object_guid_ad,cuenta_ad,nombre_mostrado,estado,primer_acceso_en,ultimo_acceso_en)
VALUES('99999999-9999-4999-8999-999999999990','99999999-9999-4999-8999-999999999989','carmen.navarro','Carmen Navarro Ruiz','activo',clock_timestamp(),clock_timestamp());
SELECT set_config('app.usuario_id','99999999-9999-4999-8999-999999999990',true);
SELECT set_config('app.perfiles','administracion',true);
SELECT documento.activar_version_plantilla(p.id,r.version)
FROM documento.referencia_activa r JOIN documento.plantilla p
 ON p.tipo_documento=r.tipo_documento AND p.idioma=r.idioma AND p.version='1.0.0'
WHERE r.tipo_documento='entrada/aceptacion' AND r.idioma='en';
SELECT documento.guardar_version_plantilla(
 '99999999-9999-4999-8999-999999999988','consentimiento/anexo2','en','1.0.1',
 'plantillas/consentimiento/anexo2/.versiones/99999999-9999-4999-8999-999999999988/v1.0.1.docx',
 repeat('b',64),1);
"""
    comprobacion = """
DO $$ BEGIN
 IF (SELECT p.version FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id
     WHERE r.tipo_documento='entrada/aceptacion' AND r.idioma='es') <> '1.1.2'
 OR (SELECT p.version FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id
     WHERE r.tipo_documento='entrada/aceptacion' AND r.idioma='en') <> '1.0.0'
 OR (SELECT version FROM documento.referencia_activa WHERE tipo_documento='entrada/aceptacion' AND idioma='en') <> 3
 OR (SELECT encode(huella_sha256,'hex') FROM documento.plantilla
     WHERE id='99999999-9999-4999-8999-999999999988') <> repeat('b',64)
 OR (SELECT count(*) FROM documento.activacion_plantilla) <> 1 THEN
  RAISE EXCEPTION 'prueba.plv_decision_administrativa';
 END IF;
END $$;
ROLLBACK;
RESET SESSION AUTHORIZATION;"""
    orden(sql, preparacion + (RAIZ / 'migraciones/documento/0013_plantillas_corregidas.sql').read_text() + (RAIZ / 'migraciones/documento/0016_respuestas_programa.sql').read_text() + comprobacion)
    print('PLV-RESP1-UPGRADE-OK: nuevas referencias, restauración y versión administrativa conservadas', flush=True)


def ensayar_respuestas_programa(sql):
    # Datos anteriores a RESP1. El ensayo se revierte antes de continuar la
    # instalación, por lo que no cambia los permisos ni los datos de otras pruebas.
    preparacion = """BEGIN;
INSERT INTO identidad.usuario(id,object_guid_ad,cuenta_ad,nombre_mostrado,estado,primer_acceso_en,ultimo_acceso_en)
VALUES('99999999-9999-4999-8999-999999999980','99999999-9999-4999-8999-999999999981','andrea.molina','Andrea Molina Serrano','activo',clock_timestamp(),clock_timestamp());
SELECT set_config('app.usuario_id','99999999-9999-4999-8999-999999999980',true);
SELECT set_config('app.perfiles','coordinacion',true);
SELECT catalogo.guardar_smi(2026::smallint,130000::bigint,0);
SELECT set_config('app.perfiles','administracion',true);
SELECT documento.guardar_version_plantilla('99999999-9999-4999-8999-999999999982',
 'spaf','es','1.0.2','plantillas/spaf/.versiones/99999999-9999-4999-8999-999999999982/v1.0.2.docx',
 repeat('b',64),(SELECT version FROM documento.referencia_activa WHERE tipo_documento='spaf' AND idioma='es'));
SELECT documento.activar_version_plantilla('99999999-9999-4999-8999-999999999982',
 (SELECT version FROM documento.referencia_activa WHERE tipo_documento='spaf' AND idioma='es'));
SELECT set_config('app.perfiles','coordinacion',true);
INSERT INTO expediente.expediente(id,fecha_recepcion)
VALUES('99999999-9999-4999-8999-999999999983',TIMESTAMPTZ '2026-10-01 10:00:00Z'),
 ('99999999-9999-4999-8999-999999999984',TIMESTAMPTZ '2026-10-01 10:00:00Z');
INSERT INTO expediente.derivacion_datos(expediente_id,esquema_version,contenido)
SELECT e.id,'1',jsonb_build_object('contenido',jsonb_build_object('tipo','anexo3','campos','[]'::jsonb),
 'ubicacion_confirmada',jsonb_build_object('equipo_id',q.id))
FROM expediente.expediente e CROSS JOIN catalogo.equipo q
WHERE e.id IN ('99999999-9999-4999-8999-999999999983','99999999-9999-4999-8999-999999999984') AND q.codigo='01';
UPDATE expediente.expediente SET estado='lista_espera',fecha_lista_espera=DATE '2026-10-05',version=2,
 modificado_por='99999999-9999-4999-8999-999999999980',modificado_en=TIMESTAMPTZ '2026-10-05 10:00:00Z'
WHERE id IN ('99999999-9999-4999-8999-999999999983','99999999-9999-4999-8999-999999999984');
INSERT INTO expediente.estado_cambio(expediente_id,estado_anterior,estado_nuevo,cambiado_por,cambiado_en)
SELECT id,'derivacion_recibida','lista_espera','99999999-9999-4999-8999-999999999980',TIMESTAMPTZ '2026-10-05 10:00:00Z'
FROM expediente.expediente WHERE id IN ('99999999-9999-4999-8999-999999999983','99999999-9999-4999-8999-999999999984');
UPDATE expediente.expediente SET estado='no_aceptada',fecha_no_aceptacion=DATE '2026-10-10',
 causa_no_aceptacion='La familia recibe atención de otro recurso.',subprograma='RIESGO_SOCIAL',version=3,
 modificado_por='99999999-9999-4999-8999-999999999980',modificado_en=TIMESTAMPTZ '2026-10-10 10:00:00Z'
WHERE id='99999999-9999-4999-8999-999999999984';
INSERT INTO expediente.estado_cambio(expediente_id,estado_anterior,estado_nuevo,cambiado_por,cambiado_en)
VALUES('99999999-9999-4999-8999-999999999984','lista_espera','no_aceptada','99999999-9999-4999-8999-999999999980',TIMESTAMPTZ '2026-10-10 10:00:00Z');
SET SESSION AUTHORIZATION etf_migrador;
SET LOCAL ROLE etf_propietario;
"""
    comprobacion = """
DO $$ BEGIN
 IF (SELECT importe_mensual_centimos FROM catalogo.smi WHERE anio=2026)<>130000
 OR (SELECT version FROM catalogo.smi WHERE anio=2026)<>1
 OR (SELECT plantilla_id FROM documento.referencia_activa WHERE tipo_documento='spaf' AND idioma='es')<>'99999999-9999-4999-8999-999999999982'::uuid
 OR (SELECT encode(huella_sha256,'hex') FROM documento.plantilla WHERE id='99999999-9999-4999-8999-999999999982')<>repeat('b',64)
 OR (SELECT count(*) FROM expediente.espera_memoria WHERE expediente_id IN ('99999999-9999-4999-8999-999999999983','99999999-9999-4999-8999-999999999984') AND desde=DATE '2026-10-05')<>2
 OR (SELECT hasta FROM expediente.espera_memoria WHERE expediente_id='99999999-9999-4999-8999-999999999984')<>DATE '2026-10-10'
 OR NOT estadistica.codigo_cifras('poblacion','PE')
 OR NOT estadistica.codigo_cifras('contrato.denominador','lista_espera_o_aceptados_sin_consentimiento_mas_de_30_dias')
 OR EXISTS(SELECT 1 FROM pg_policies WHERE policyname='resp1_migrador') THEN
  RAISE EXCEPTION 'prueba.resp1_actualizacion';
 END IF;
END $$;
EXPLAIN (COSTS OFF) SELECT expediente_id FROM expediente.espera_memoria
 WHERE equipo_id=(SELECT id FROM catalogo.equipo WHERE codigo='01') AND desde<=DATE '2026-10-11';
ROLLBACK;
RESET SESSION AUTHORIZATION;
"""
    archivos = ['documento/0013_plantillas_corregidas.sql', 'catalogo/0014_respuestas_programa.sql',
                'expediente/0027_espera_memoria.sql', 'estadistica/0007_pendientes_lista_espera.sql',
                'documento/0016_respuestas_programa.sql']
    cambios = ''.join((RAIZ / 'migraciones' / archivo).read_text() for archivo in archivos)
    print(orden(sql, preparacion + cambios + comprobacion), flush=True)
    print('RESP1-UPGRADE-OK: espera previa reconstruida, SMI editado y plantilla administrativa conservados', flush=True)


def main():
    with tempfile.TemporaryDirectory(prefix='etf-f106-') as carpeta:
        tmp = pathlib.Path(carpeta)
        tmp.chmod(0o755)
        bd = 'etf-f106-' + secrets.token_hex(4)
        claves = {k: secrets.token_hex(24) for k in ('ETF_MIGRADOR_CONTRASENA',
                  'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA')}
        env = tmp / 'base.env'
        env.write_text('POSTGRES_USER=etf_instalador\nPOSTGRES_DB=etf\nPOSTGRES_PASSWORD=' +
                       secrets.token_hex(24) + '\n' + ''.join(k+'='+v+'\n' for k,v in claves.items()))
        env.chmod(0o600)
        runner = tmp / 'runner.env'
        runner.write_text('ETF_BD_HOST=127.0.0.1\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\n'
                          'ETF_BD_USUARIO=etf_app\nETF_DOCUMENTO_POSTGRES_INTEGRACION=1\n'
                          'ETF_BD_CONTRASENA='+claves['ETF_APP_CONTRASENA']+'\n'
                          'ETF_MIGRADOR_CONTRASENA='+claves['ETF_MIGRADOR_CONTRASENA']+'\n')
        runner.chmod(0o600)
        compilacion = dict(os.environ, CGO_ENABLED='0')
        orden(['go', 'build', '-o', str(tmp / 'migrar'), './cmd/migrar'], entorno=compilacion)
        orden(['go', 'test', '-c', '-o', str(tmp / 'documento.test'),
               './internal/adaptadores/postgres/documento'], entorno=compilacion)
        previo = tmp / 'previo'
        previo_plv = tmp / 'previo-plv'
        shutil.copytree(RAIZ / 'migraciones', previo_plv,
                        ignore=shutil.ignore_patterns('__pycache__'))
        manifest_plv = previo_plv / 'documento/manifiesto.json'
        entradas_plv = json.loads(manifest_plv.read_text())
        for entrada in entradas_plv:
            if entrada['numero'] >= '0013':
                (previo_plv / 'documento' / entrada['archivo']).unlink()
        manifest_plv.write_text(json.dumps([e for e in entradas_plv if e['numero'] < '0013']))
        # Conservación depende de Documento14 y se instala después del corte PLV.
        manifest_exp = previo_plv / 'expediente' / 'manifiesto.json'
        entradas_exp = json.loads(manifest_exp.read_text())
        for entrada in entradas_exp:
            if entrada['numero'] >= '0023':
                (previo_plv / 'expediente' / entrada['archivo']).unlink()
        manifest_exp.write_text(json.dumps([e for e in entradas_exp if e['numero'] < '0023']))
        recortar_dependientes(previo_plv)
        shutil.copytree(RAIZ / 'migraciones', previo,
                        ignore=shutil.ignore_patterns('__pycache__'))
        # Se prueban todas las entregas sobre una base anterior común.
        # Cada corte retira sólo SQL y manifiestos de la copia temporal.
        for modulo, corte in [('documento','0003'),('catalogo','0007'),('expediente','0007'),('paquete','0003'),('auditoria','0002')]:
            carpeta_modulo = previo / modulo
            manifest = carpeta_modulo / 'manifiesto.json'
            entradas = json.loads(manifest.read_text())
            for entrada in entradas:
                if entrada['numero'] >= corte:
                    (carpeta_modulo / entrada['archivo']).unlink()
            manifest.write_text(json.dumps([e for e in entradas if e['numero'] < corte]))
        shutil.rmtree(previo / 'estadistica')
        # Esta instantánea precede a expediente/0018, dependencia de carga/0001.
        shutil.rmtree(previo / 'carga')
        previo_resp3 = tmp / 'previo-resp3'
        shutil.copytree(RAIZ / 'migraciones', previo_resp3)
        manifest_resp3 = previo_resp3 / 'documento' / 'manifiesto.json'
        entradas_resp3 = json.loads(manifest_resp3.read_text())
        for entrada in entradas_resp3:
            if entrada['numero'] >= '0015':
                (previo_resp3 / 'documento' / entrada['archivo']).unlink()
        manifest_resp3.write_text(json.dumps([e for e in entradas_resp3 if e['numero'] < '0015']))
        montar = ['--mount' , f'type=bind,source={tmp},target=/pruebas,readonly',
                  '--mount', f'type=bind,source={RAIZ},target=/repo,readonly']
        try:
            orden(['docker', 'run', '-d', '--name', bd, '--network', 'none',
                   '--read-only', '--memory', '512m', '--memory-swap', '512m',
                   '--cpus', '1', '--pids-limit', '128', '--env-file', str(env),
                   '--tmpfs', '/var/lib/postgresql:rw,size=512m',
                   '--tmpfs', '/var/run/postgresql:rw,size=16m',
                   '--tmpfs', '/tmp:rw,size=16m', *montar,
                   '--mount', f'type=bind,source={RAIZ}/despliegue/base/01_roles.sh,target=/docker-entrypoint-initdb.d/01_roles.sh,readonly',
                   IMAGEN])
            sql = ['docker', 'exec', '-i', bd, 'psql', '-U', 'etf_instalador',
                   '-d', 'etf', '-v', 'ON_ERROR_STOP=1']
            esperar_postgres(bd, runner, modulo='documento')
            ejecutar = ['docker', 'exec', '--env-file', str(runner), bd]
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/previo', '/pruebas/migrar']), flush=True)
            # El catálogo del ensayo existe antes de instalar las versiones nuevas.
            sembrar_catalogo_anexo6(sql)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/previo-plv', '/pruebas/migrar']), flush=True)
            ensayar_actualizacion_plv(sql)
            ensayar_respuestas_programa(sql)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print('ENSAYO-OK versión anterior, migración nueva y repetición', flush=True)
            orden(sql, """
INSERT INTO identidad.usuario(id,object_guid_ad,cuenta_ad,nombre_mostrado,estado,primer_acceso_en,ultimo_acceso_en)
VALUES('11111111-1111-4111-8111-111111111111','11111111-1111-4111-8111-111111111112','ines.moreno','Inés Moreno Castillo','activo',clock_timestamp(),clock_timestamp());
INSERT INTO expediente.expediente(id,fecha_recepcion) VALUES('22222222-2222-4222-8222-222222222222',clock_timestamp());
INSERT INTO catalogo.firmante(id,version,tipo,rol,nombre,cargo_codigo,organismo_codigo,direccion,vigente_desde,creada_por)
VALUES('33333333-3333-4333-8333-333333333333',1,'firmante','aceptacion','Inés Moreno Castillo','DIRECCION','DIPUTACION','',DATE '2026-01-01','11111111-1111-4111-8111-111111111111');
""")
            sembrar_entradas(sql)
            print(orden(['docker','exec','--env-file',str(runner),
                         '--user','70:70','--workdir','/repo/internal/adaptadores/postgres/documento',bd,
                         '/pruebas/documento.test','-test.v','-test.run=TestPlantillasInicialesCorregidasPostgres|TestRESP1ReferenciasNuevasYOriginalesInmutables','-test.timeout=30s']),flush=True)
            from sembrar_cierre import sembrar_cierre
            sembrar_cierre(sql, orden)
            sembrar_consentimientos(sql, orden)
            sembrar_anexo6(sql)
            print(orden(['docker', 'exec', '--env-file', str(runner), '--user' , '70:70',
                         '--workdir', '/repo/internal/adaptadores/postgres/documento', bd,
                         '/pruebas/documento.test', '-test.v', '-test.skip=TestConsentimientos|TestPlantillasInicialesCorregidasPostgres', '-test.timeout=120s']), flush=True)
            print(orden(['docker','exec','--env-file',str(runner),'--user','70:70',
                         '--workdir','/repo/internal/adaptadores/postgres/documento',bd,
                         '/pruebas/documento.test','-test.v','-test.run=TestPlantillasPublicacionPostgres','-test.timeout=30s']),flush=True)
            ensayar_auditoria_plantillas(sql)
            # Cambio de fixture por el propietario: conserva la versión usada.
            orden(sql, """SET ROLE etf_propietario;
INSERT INTO catalogo.firmante(id,version,tipo,rol,nombre,cargo_codigo,organismo_codigo,direccion,vigente_desde,creada_por)
SELECT id,2,tipo,rol,'Elena Núñez Romero',cargo_codigo,organismo_codigo,direccion,DATE '2026-10-13',creada_por FROM catalogo.firmante
WHERE id='33333333-3333-4333-8333-333333333333' AND version=1;
RESET ROLE;""")
            print(orden(['docker','exec','--env-file',str(runner),'-e','ETF_DOCUMENTO_FIRMANTE_CAMBIADO=1',
                         '--user','70:70','--workdir','/repo/internal/adaptadores/postgres/documento',bd,
                         '/pruebas/documento.test','-test.v','-test.run=TestFirmanteCatalogoCambiadoPostgres','-test.timeout=30s']),flush=True)
            activar_catalogos_consentimientos(sql, orden, RAIZ, tmp / 'idiomas')
            print(orden(['docker', 'exec', '--env-file', str(runner),
                         '-e', 'ETF_DOCUMENTO_IDIOMAS_RUTA=/pruebas/idiomas',
                         '--user', '70:70', '--workdir', '/repo/internal/adaptadores/postgres/documento',
                         bd, '/pruebas/documento.test', '-test.v',
                         '-test.run=TestConsentimientos', '-test.timeout=120s']), flush=True)
            orden(sql, 'CREATE DATABASE etf_limpia OWNER etf_propietario')
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_limpia',
                        'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print('ENSAYO-OK base limpia, RLS, auditoría, concurrencia e integridad', flush=True)
            orden(sql, 'CREATE DATABASE etf_resp3 OWNER etf_propietario')
            sql_resp3 = [*sql]
            sql_resp3[sql_resp3.index('-d') + 1] = 'etf_resp3'
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_resp3',
                        'ETF_MIGRACIONES_RUTA=/pruebas/previo-resp3', '/pruebas/migrar']), flush=True)
            orden(sql_resp3, """
SET ROLE etf_propietario;
INSERT INTO identidad.usuario(id,object_guid_ad,cuenta_ad,nombre_mostrado,estado,primer_acceso_en,ultimo_acceso_en)
VALUES('11111111-1111-4111-8111-111111111111','11111111-1111-4111-8111-111111111111',
 'elena.serrano','Elena Serrano Molina','activo',clock_timestamp(),clock_timestamp());
INSERT INTO documento.activacion_plantilla(tipo_documento,idioma,revision,plantilla_id,activada_por)
SELECT tipo_documento,idioma,1,id,(SELECT id FROM identidad.usuario ORDER BY id LIMIT 1)
FROM documento.plantilla WHERE tipo_documento='finalizacion/anexo12' AND idioma='es' AND version='1.0.1';
RESET ROLE;
""")
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_resp3',
                        'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            orden(sql_resp3, """
DO $$ BEGIN
 IF (SELECT count(*) FROM documento.plantilla WHERE tipo_documento IN
 ('finalizacion/anexo11','finalizacion/anexo12') AND idioma IN ('es','en') AND version='1.0.2') <> 4
 OR (SELECT count(*) FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id
 WHERE r.tipo_documento IN ('finalizacion/anexo11','finalizacion/anexo12') AND p.version='1.0.2') <> 3
 OR (SELECT p.version FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id
 WHERE r.tipo_documento='finalizacion/anexo12' AND r.idioma='es') <> '1.0.1'
 THEN RAISE EXCEPTION 'documento.ensayo_escalas_opcionales'; END IF;
END $$;
""")
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_resp3',
                        'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print('RESP3-SQL-OK: actualización desde main, repetición y activación administrativa conservada', flush=True)
            # Una publicación administrativa puede ocupar v1.0.1 antes de 0013.
            # Se conserva y no impide crear la nueva versión del paquete inicial.
            orden(sql, 'CREATE DATABASE etf_resp3_publicacion OWNER etf_propietario')
            sql_pub = [*sql]
            sql_pub[sql_pub.index('-d') + 1] = 'etf_resp3_publicacion'
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_resp3_publicacion',
                        'ETF_MIGRACIONES_RUTA=/pruebas/previo-plv', '/pruebas/migrar']), flush=True)
            orden(sql_pub, """
SET ROLE etf_propietario;
INSERT INTO identidad.usuario(id,object_guid_ad,cuenta_ad,nombre_mostrado,estado,primer_acceso_en,ultimo_acceso_en)
VALUES('11111111-1111-4111-8111-111111111111','11111111-1111-4111-8111-111111111111',
 'elena.serrano','Elena Serrano Molina','activo',clock_timestamp(),clock_timestamp());
INSERT INTO documento.plantilla(id,tipo_documento,idioma,version,ruta_relativa,huella_sha256,
 rol_firmante,vigente_desde,subida_por,comprobacion,antivirus)
SELECT '55555555-5555-4555-8555-555555555555',tipo_documento,idioma,'1.0.1',
 'plantillas/finalizacion/anexo11/.versiones/55555555-5555-4555-8555-555555555555/v1.0.1.docx',
 huella_sha256,rol_firmante,vigente_desde,'11111111-1111-4111-8111-111111111111',comprobacion,'limpio'
FROM documento.plantilla WHERE tipo_documento='finalizacion/anexo11' AND idioma='es' AND version='1.0.0';
RESET ROLE;
""")
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_resp3_publicacion',
                        'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            orden(sql_pub, """
DO $$ BEGIN
 IF (SELECT count(*) FROM documento.plantilla WHERE id='55555555-5555-4555-8555-555555555555'
 AND subida_por='11111111-1111-4111-8111-111111111111' AND version='1.0.1') <> 1
 OR (SELECT count(*) FROM documento.referencia_activa r JOIN documento.plantilla p ON p.id=r.plantilla_id
 WHERE r.tipo_documento IN ('finalizacion/anexo11','finalizacion/anexo12') AND p.version='1.0.2') <> 4
 THEN RAISE EXCEPTION 'documento.ensayo_publicacion_conservada'; END IF;
END $$;
""")
            print('RESP3-PUBLICACION-OK: colisión administrativa v1.0.1 conservada; cuatro referencias iniciales actualizadas', flush=True)
        finally:
            subprocess.run(['docker', 'rm', '-fv', bd], stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL)


if __name__ == '__main__':
    main()
