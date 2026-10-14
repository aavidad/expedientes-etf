#!/usr/bin/env python3
"""Ensaya catálogo en PostgreSQL 18 desechable con la configuración de Compose.

No usa credenciales previas ni una base existente. Destruye sólo los recursos
creados por esta ejecución. Requiere Docker Compose y la herramienta Go local.
"""
import os
import sys
from functools import partial
import pathlib
import secrets
import shutil
import subprocess
import tempfile

RAIZ = pathlib.Path(__file__).resolve().parents[2]
IMAGEN = 'postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15'


sys.path.insert(0, str(RAIZ / 'scripts' / 'ci'))
from ensayos import orden as ejecutar_orden

orden = partial(ejecutar_orden, modulo='catalogo')


def main():
    with tempfile.TemporaryDirectory(prefix='etf-catalogo-') as nombre:
        temporal = pathlib.Path(nombre)
        temporal.chmod(0o755)
        proyecto = 'etf-f009b-int2-' + secrets.token_hex(4)
        entorno = {k: secrets.token_hex(24) for k in ['ETF_INSTALADOR_CONTRASENA',
                    'ETF_MIGRADOR_CONTRASENA', 'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA']}
        subred = 32 + secrets.randbelow(192)
        entorno.update(ETF_RED_SUBRED=f'10.253.{subred}.0/28',
                       ETF_ENTRADA_SUBRED=f'10.253.{subred}.16/28')
        archivo_env = temporal / 'compose.env'
        archivo_env.write_text(''.join(k + '=' + v + '\n' for k, v in entorno.items()))
        archivo_env.chmod(0o600)
        compose = ['docker', 'compose', '--project-directory', str(RAIZ), '-f', str(RAIZ / 'compose.yaml'),
                   '--env-file', str(archivo_env), '-p', proyecto]
        # Ejecutar el mismo cmd/migrar del árbol; psql procede de la imagen fijada.
        compilacion = dict(os.environ, CGO_ENABLED='0')
        orden(['go', 'build', '-o', str(temporal / 'migrar'), './cmd/migrar'], entorno=compilacion)
        orden(['go', 'test', '-c', '-o', str(temporal / 'catalogo.test'),
               './internal/adaptadores/postgres/catalogo'], entorno=compilacion)
        entorno_runner = temporal / 'runner.env'
        entorno_runner.write_text('ETF_BD_HOST=postgres\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\n'
                                 'ETF_BD_USUARIO=etf_app\nETF_POSTGRES_INTEGRACION=1\n'
                                 'ETF_BD_CONTRASENA=' + entorno['ETF_APP_CONTRASENA'] + '\n'
                                 'ETF_MIGRADOR_CONTRASENA=' + entorno['ETF_MIGRADOR_CONTRASENA'] + '\n'
                                 'ETF_MIGRACIONES_RUTA=/repo/migraciones\n')
        entorno_runner.chmod(0o600)
        runner = ['docker', 'run', '--rm', '--read-only', '--cap-drop', 'ALL', '--security-opt',
                  'no-new-privileges:true', '--memory', '256m', '--cpus', '1', '--pids-limit', '64',
                  '--tmpfs', '/tmp:rw,noexec,nosuid,size=32m',
                  '--network', proyecto + '_interna', '--env-file', str(entorno_runner),
                  '--mount', f'type=bind,source={temporal},target=/pruebas,readonly',
                  '--mount', f'type=bind,source={RAIZ},target=/repo,readonly', '--user', '70:70']
        try:
            orden(compose + ['up', '-d', '--wait', 'postgres'])
            print(orden(runner + ['--entrypoint', '/pruebas/migrar', IMAGEN]))
            print(orden(runner + ['--entrypoint', '/pruebas/migrar', IMAGEN]))
            print('ENSAYO-OK migración limpia y repetida')
            print(orden(runner + ['--workdir', '/repo/internal/adaptadores/postgres/catalogo',
                                 '--entrypoint', '/pruebas/catalogo.test', IMAGEN, '-test.v', '-test.timeout=60s']))
            # Auditoría compila su comprobador y sesión inyecta un fallo con
            # Docker. Se ejecutan desde el host contra este contenedor aislado.
            contenedor = orden(compose + ['ps', '-q', 'postgres'])
            direccion = orden(['docker', 'inspect', '--format',
                               '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}', contenedor])
            nombre_contenedor = orden(['docker', 'inspect', '--format', '{{.Name}}', contenedor]).lstrip('/')
            entorno_pruebas = dict(os.environ, ETF_POSTGRES_INTEGRACION='1',
                                   ETF_BD_HOST=direccion, ETF_BD_PUERTO='5432', ETF_BD_NOMBRE='etf',
                                   ETF_BD_USUARIO='etf_app', ETF_BD_SSLMODE='disable',
                                   ETF_BD_CONTRASENA=entorno['ETF_APP_CONTRASENA'],
                                   ETF_MIGRADOR_CONTRASENA=entorno['ETF_MIGRADOR_CONTRASENA'],
                                   ETF_POSTGRES_CONTENEDOR=nombre_contenedor,
                                   ETF_AUDITORIA_REF=orden(['git', 'rev-parse', 'HEAD']))
            for clave in list(entorno_pruebas):
                if clave.startswith('PG'):
                    del entorno_pruebas[clave]
            print(orden(['go', 'test', '-count=1', '-run', 'TestIntegracion', '-v',
                         './internal/adaptadores/postgres'], entorno=entorno_pruebas))
            # Base anterior: catálogo F0.12 y paquete F0.14c-2, con un idioma
            # semver ya activado. Después se aplica sólo la ampliación pendiente.
            anterior = temporal / 'anterior' / 'plataforma'
            anterior.mkdir(parents=True)
            shutil.copyfile(RAIZ / 'migraciones/plataforma/0001_control.sql', anterior / '0001_control.sql')
            shutil.copyfile(RAIZ / 'migraciones/plataforma/manifiesto.json', anterior / 'manifiesto.json')
            catalogo_anterior=temporal / 'anterior' / 'catalogo'
            catalogo_anterior.mkdir()
            for nombre in ['0001_estructura.sql','0002_carga_inicial.sql']:
                shutil.copyfile(RAIZ / 'migraciones/catalogo' / nombre,catalogo_anterior / nombre)
            import json
            manifiesto=json.loads((RAIZ / 'migraciones/catalogo/manifiesto.json').read_text())
            (catalogo_anterior / 'manifiesto.json').write_text(json.dumps([m for m in manifiesto if m['numero']<='0002']))
            auditoria_anterior = temporal / 'anterior/auditoria'
            auditoria_anterior.mkdir()
            shutil.copyfile(RAIZ / 'migraciones/auditoria/0001_cadena.sql',
                            auditoria_anterior / '0001_cadena.sql')
            manifiesto_auditoria = json.loads((RAIZ / 'migraciones/auditoria/manifiesto.json').read_text())
            (auditoria_anterior / 'manifiesto.json').write_text(json.dumps(
                [m for m in manifiesto_auditoria if m['numero'] == '0001']))
            paquete_anterior = temporal / 'anterior/paquete'
            paquete_anterior.mkdir()
            shutil.copyfile(RAIZ / 'migraciones/paquete/0001_versiones_activas.sql',
                            paquete_anterior / '0001_versiones_activas.sql')
            manifiesto_paquete = json.loads((RAIZ / 'migraciones/paquete/manifiesto.json').read_text())
            (paquete_anterior / 'manifiesto.json').write_text(json.dumps([m for m in manifiesto_paquete if m['numero']=='0001']))
            sql = compose + ['exec', '-T', 'postgres', 'psql', '-X', '-qAt', '-v',
                             'ON_ERROR_STOP=1', '-U', 'etf_instalador', '-d', 'etf']
            orden(sql, 'CREATE DATABASE etf_anterior OWNER etf_propietario TEMPLATE template0;')
            print(orden(runner + ['-e', 'ETF_BD_NOMBRE=etf_anterior', '-e',
                                 'ETF_MIGRACIONES_RUTA=/pruebas/anterior', '--entrypoint', '/pruebas/migrar', IMAGEN]))
            sql_anterior = sql[:-1] + ['etf_anterior']
            huella = 'a' * 64
            ruta = '.versiones/pt/1.0.0/' + huella + '/' + 'b' * 32 + '/pt'
            orden(sql_anterior, "BEGIN; SET LOCAL ROLE etf_propietario; "
                  "SELECT set_config('app.usuario_id','00000000-0000-4000-8000-000000000012',true),"
                  "set_config('app.perfiles','administracion',true); "
                  f"SELECT paquete.activar('idioma','pt','1.0.0','{huella}','{ruta}',0); COMMIT;")
            previa = orden(sql_anterior, "SET ROLE etf_propietario; SELECT id FROM paquete.idioma_instalado WHERE codigo='pt';")
            print(orden(runner + ['-e', 'ETF_BD_NOMBRE=etf_anterior', '--entrypoint', '/pruebas/migrar', IMAGEN]))
            assert orden(sql_anterior, "SET ROLE etf_propietario; SELECT id FROM paquete.idioma_instalado WHERE codigo='pt';") == previa
            print(orden(runner + ['-e', 'ETF_BD_NOMBRE=etf_anterior', '--entrypoint', '/pruebas/migrar', IMAGEN]))
            print('ENSAYO-OK actualización desde catalogo/0002 + paquete/0001; UUID y versión semver conservados')
            # Ensayar la ampliación de avisos sobre plazos ya publicados.
            previo_avisos = temporal / 'previo_avisos'
            previo_avisos.mkdir()
            # El corte de avisos precede al seguimiento y a los módulos consumidores.
            for modulo in ('plataforma', 'catalogo', 'auditoria', 'paquete'):
                shutil.copytree(RAIZ / 'migraciones' / modulo, previo_avisos / modulo)
            for modulo, corte in [('catalogo', '0012'), ('auditoria', '0002')]:
                archivo = previo_avisos / modulo / 'manifiesto.json'
                entradas = json.loads(archivo.read_text())
                for entrada in entradas:
                    if entrada['numero'] >= corte:
                        (previo_avisos / modulo / entrada['archivo']).unlink()
                archivo.write_text(json.dumps([e for e in entradas if e['numero'] < corte]))
            orden(sql, 'CREATE DATABASE etf_previo_avisos OWNER etf_propietario TEMPLATE template0;')
            orden(runner + ['-e', 'ETF_BD_NOMBRE=etf_previo_avisos', '-e',
                            'ETF_MIGRACIONES_RUTA=/pruebas/previo_avisos',
                            '--entrypoint', '/pruebas/migrar', IMAGEN])
            for caso, dias in [('pasada', -1), ('hoy', 0), ('futura', 7)]:
                base_avisos = 'etf_avisos_' + caso
                orden(sql, f'CREATE DATABASE {base_avisos} OWNER etf_propietario TEMPLATE etf_previo_avisos;')
                sql_avisos = sql[:-1] + [base_avisos]
                orden(sql_avisos, "BEGIN; SET LOCAL ROLE etf_propietario; "
                      "INSERT INTO catalogo.version(id,ambito,numero,vigente_desde,motivo,creada_por) "
                      "VALUES('77777777-7777-4777-8777-777777777777','plazos',1,"
                      f"current_date+({dias}),'catalogo.edicion','00000000-0000-4000-8000-000000000012'); "
                      "INSERT INTO catalogo.parametro VALUES('77777777-7777-4777-8777-777777777777',"
                      "'plazo_consentimiento',45,'dias_naturales'); "
                      "SELECT catalogo.publicar('77777777-7777-4777-8777-777777777777',"
                      "'00000000-0000-4000-8000-000000000012'); COMMIT;")
                # La sesión y el rol son los mismos que usa el migrador.
                sentencia = ("SET SESSION AUTHORIZATION etf_migrador; BEGIN; "
                             "SET LOCAL ROLE etf_propietario; " +
                             (RAIZ / 'migraciones/catalogo/0012_avisos.sql').read_text() + " COMMIT;")
                resultado = subprocess.run(sql_avisos, input=sentencia, text=True,
                                           capture_output=True, timeout=60)
                if dias < 0:
                    assert resultado.returncode == 0, resultado.stderr
                    assert orden(sql_avisos, "SET ROLE etf_propietario; SELECT count(*) "
                                 "FROM catalogo.version v JOIN catalogo.parametro p ON p.version_id=v.id "
                                 "WHERE v.ambito='plazos' AND v.numero=2 AND v.vigente_desde=current_date "
                                 "AND p.clave='plazo_consentimiento' AND p.valor_entero=45;") == '1'
                else:
                    assert resultado.returncode != 0 and 'migracion.estado_previo' in resultado.stderr
                    assert orden(sql_avisos, "SELECT to_regclass('catalogo.fase_duracion') IS NULL;") == 't'
                    assert orden(sql_avisos, "SET ROLE etf_propietario; SELECT count(*) "
                                 "FROM catalogo.version WHERE ambito='plazos';") == '1'
            print('ENSAYO-OK avisos: plazos pasados activos hoy y conservados; hoy/futuros rechazados sin cambios')
            # Plan de lectura real para el índice de exclusión.
            print(orden(sql, "SET ROLE etf_propietario; EXPLAIN SELECT id FROM catalogo.version "
                        "WHERE ambito='listas' AND estado='publicada' "
                        "AND daterange(vigente_desde,vigente_hasta,'[)') @> DATE '2026-12-31';"))
        finally:
            orden(compose + ['down', '-v', '--remove-orphans'])
            print('ENSAYO-OK contenedor y volumen destruidos')


if __name__ == '__main__':
    os.chdir(RAIZ)
    main()
