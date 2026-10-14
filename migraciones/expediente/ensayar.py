#!/usr/bin/env python3
"""Ensaya F1.01 en PostgreSQL 18 y ClamAV desechables, sin salida a Internet.

Sólo se crean recursos propios. La base y los temporales de ClamAV viven en
tmpfs sin intercambio a disco. Las firmas de este ensayo son sintéticas.
"""
import hashlib
import json
import os
import sys
from functools import partial
import pathlib
import secrets
import shutil
import subprocess
import tempfile

RAIZ = pathlib.Path(__file__).resolve().parents[2]
POSTGRES = 'postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15'
CLAMAV = 'clamav/clamav:1.4.3@sha256:75fb5fd95fcbe1d7e6d240c369c1572b686ee2c95949d1042b5148de8eddebb4'


sys.path.insert(0, str(RAIZ / 'scripts' / 'ci'))
from ensayos import orden as ejecutar_orden, esperar_postgres, esperar_orden

orden = partial(ejecutar_orden, modulo='expediente')


esperar = partial(esperar_orden, modulo='expediente')


def main():
    with tempfile.TemporaryDirectory(prefix='etf-f101-') as carpeta:
        tmp = pathlib.Path(carpeta)
        tmp.chmod(0o755)
        prefijo = 'etf-f101-' + secrets.token_hex(4)
        red, bd, antivirus = prefijo + '-red', prefijo + '-bd', prefijo + '-av'
        claves = {k: secrets.token_hex(24) for k in ('ETF_MIGRADOR_CONTRASENA',
                  'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA')}
        env = tmp / 'base.env'
        env.write_text('POSTGRES_USER=etf_instalador\nPOSTGRES_DB=etf\nPOSTGRES_PASSWORD=' +
                       secrets.token_hex(24) + '\n' + ''.join(k+'='+v+'\n' for k,v in claves.items()))
        env.chmod(0o600)
        runner = tmp / 'runner.env'
        runner.write_text('ETF_BD_HOST=127.0.0.1\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\n'
                          'ETF_BD_USUARIO=etf_app\nETF_POSTGRES_INTEGRACION=1\n'
                          'ETF_PRUEBA_CLAMAV_DIRECCION=clamav:3310\n'
                          'ETF_BD_CONTRASENA='+claves['ETF_APP_CONTRASENA']+'\n'
                          'ETF_MIGRADOR_CONTRASENA='+claves['ETF_MIGRADOR_CONTRASENA']+'\n')
        runner.chmod(0o600)
        firmas = tmp / 'firmas'
        firmas.mkdir()
        eicar = b'X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*'
        (firmas / 'local.hdb').write_text(hashlib.md5(eicar).hexdigest()+':68:ETF.Eicar.Test\n')
        (tmp / 'clamd.conf').write_text('Foreground yes\nTCPSocket 3310\nTCPAddr 0.0.0.0\n'
            'DatabaseDirectory /pruebas/firmas\nTemporaryDirectory /tmp\n'
            'StreamMaxLength 10M\nMaxScanSize 100M\nMaxFileSize 10M\nMaxRecursion 8\n'
            'MaxFiles 1000\nLogTime no\nLogClean no\nLogFile /dev/null\n')
        compilacion = dict(os.environ, CGO_ENABLED='0')
        orden(['go', 'build', '-o', str(tmp / 'migrar'), './cmd/migrar'], entorno=compilacion)
        orden(['go', 'test', '-c', '-o', str(tmp / 'expediente.test'),
               './internal/adaptadores/postgres/expediente'], entorno=compilacion)
        previo = tmp / 'previo'
        previo.mkdir()
        for modulo in ('plataforma', 'catalogo', 'auditoria', 'identidad', 'paquete'):
            shutil.copytree(RAIZ / 'migraciones' / modulo, previo / modulo)
        # El corte anterior a F1.01 no incluye la auditoría de ficha posterior.
        manifiesto = previo / 'auditoria/manifiesto.json'
        entradas = json.loads(manifiesto.read_text())
        for entrada in entradas:
            if entrada['numero'] >= '0002':
                (previo / 'auditoria' / entrada['archivo']).unlink()
        manifiesto.write_text(json.dumps([e for e in entradas if e['numero'] < '0002']))
        # RESP4 ensaya la actualización desde el esquema inmediatamente anterior,
        # con una derivación Word ya confirmada, sin editar migraciones aplicadas.
        anterior_odt = tmp / 'anterior_odt'
        shutil.copytree(RAIZ / 'migraciones', anterior_odt)
        manifiesto_odt = anterior_odt / 'expediente/manifiesto.json'
        entradas_odt = json.loads(manifiesto_odt.read_text())
        for entrada in entradas_odt:
            if entrada['numero'] >= '0026':
                (anterior_odt / 'expediente' / entrada['archivo']).unlink()
        manifiesto_odt.write_text(json.dumps([e for e in entradas_odt if e['numero'] < '0026']))
        montar = ['--mount', f'type=bind,source={tmp},target=/pruebas,readonly',
                  '--mount', f'type=bind,source={RAIZ},target=/repo,readonly']
        try:
            orden(['docker', 'network', 'create', '--internal', '--subnet',
                   f'10.252.{secrets.randbelow(240)}.0/28', red])
            orden(['docker', 'run', '-d', '--name', bd, '--network', red,
                   '--read-only', '--memory', '512m', '--memory-swap', '512m', '--pids-limit', '128',
                   '--env-file', str(env), '--tmpfs', '/var/lib/postgresql:rw,size=512m',
                   '--tmpfs', '/var/run/postgresql:rw,size=16m',
                   '--tmpfs', '/tmp:rw,size=16m', *montar,
                   '--mount', f'type=bind,source={RAIZ}/despliegue/base/01_roles.sh,target=/docker-entrypoint-initdb.d/01_roles.sh,readonly',
                   POSTGRES])
            esperar_postgres(bd, runner, modulo='expediente')
            ejecutar = ['docker', 'exec', '--env-file', str(runner), bd]
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/previo', '/pruebas/migrar']))
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/anterior_odt', '/pruebas/migrar']))
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir', '/repo/internal/adaptadores/postgres/expediente', bd,
                                   '/pruebas/expediente.test', '-test.v',
                                   '-test.run', '^TestIntegracionConfirmacionConcurrenteSinDuplicados$']))
            sql_anterior = "SELECT count(*)>0 FROM expediente.lectura_anexo"
            estado_anterior = orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador',
                                    '-d', 'etf', '-At', '-c', sql_anterior])
            if estado_anterior != 't':
                raise RuntimeError('expediente.resp4_sin_datos_previos')
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            print('ENSAYO-OK RESP4: versión anterior con derivación DOCX, expediente/0026 y repetición')
            orden(['docker', 'run', '-d', '--name', antivirus, '--network', red,
                   '--network-alias', 'clamav', '--read-only', '--memory', '512m',
                   '--memory-swap', '512m', '--pids-limit', '64', '--cap-drop', 'ALL',
                   '--security-opt', 'no-new-privileges:true', '--tmpfs', '/tmp:rw,size=64m',
                   *montar, '--entrypoint', '/usr/sbin/clamd', CLAMAV,
                   '--config-file=/pruebas/clamd.conf'])
            esperar(['docker', 'exec', antivirus, 'sh', '-c',
                     "printf 'nPING\\n' | nc -w 1 127.0.0.1 3310 | grep -qx PONG"])
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir',
                         '/repo/internal/adaptadores/postgres/expediente', bd,
                         '/pruebas/expediente.test', '-test.v']))
            # Otra base, desde cero, acredita también el camino de instalación.
            orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador', '-d', 'etf',
                   '-v', 'ON_ERROR_STOP=1', '-c', 'CREATE DATABASE etf_limpia OWNER etf_propietario'])
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_limpia',
                        'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            print('ENSAYO-OK base limpia, RLS, auditoría, EICAR y ClamAV caído')
        finally:
            for nombre in (antivirus, bd):
                subprocess.run(['docker', 'rm', '-fv', nombre], stdout=subprocess.DEVNULL,
                               stderr=subprocess.DEVNULL)
            subprocess.run(['docker', 'network', 'rm', red], stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL)


if __name__ == '__main__':
    main()
