#!/usr/bin/env python3
"""Reproduce la carrera de initdb sin depender de la carga del equipo.

Requiere bin/migrar compilado. La inicialización queda detenida después de
crear los roles, en un contenedor propio sin red y con datos desechables.
"""
from concurrent.futures import ThreadPoolExecutor
import json
import pathlib
import secrets
import shutil
import subprocess
import tempfile
import time

from ensayos import diagnostico, esperar_postgres, orden

RAIZ = pathlib.Path(__file__).resolve().parents[2]
IMAGEN = 'postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15'


def main():
    with tempfile.TemporaryDirectory(prefix='etf-fx5-arranque-') as carpeta:
        tmp = pathlib.Path(carpeta)
        tmp.chmod(0o755)
        env = tmp / 'base.env'
        valores = {'POSTGRES_USER': 'etf_instalador', 'POSTGRES_DB': 'etf',
                   'ETF_BD_HOST': '127.0.0.1', 'ETF_BD_PUERTO': '5432',
                   'ETF_BD_NOMBRE': 'etf', 'ETF_MIGRACIONES_RUTA': '/pruebas/previo'}
        for clave in ('POSTGRES_PASSWORD', 'ETF_MIGRADOR_CONTRASENA',
                      'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA'):
            valores[clave] = secrets.token_hex(24)
        env.write_text(''.join(k + '=' + v + '\n' for k, v in valores.items()))
        env.chmod(0o600)
        gate = tmp / '02_esperar.sh'
        gate.write_text('#!/bin/sh\nset -eu\ntouch /tmp/esperando\n'
                        'while [ ! -e /tmp/continuar ]; do sleep 0.05; done\n')
        gate.chmod(0o755)
        previo = tmp / 'previo'
        shutil.copytree(RAIZ / 'migraciones', previo,
                        ignore=shutil.ignore_patterns('__pycache__'))
        shutil.rmtree(previo / 'evaluacion')
        # Conservación es posterior a Evaluación y no pertenece a este corte.
        manifest_exp = previo / 'expediente' / 'manifiesto.json'
        entradas_exp = json.loads(manifest_exp.read_text())
        for entry in entradas_exp:
            if entry['numero'] >= '0023':
                (previo / 'expediente' / entry['archivo']).unlink()
        manifest_exp.write_text(json.dumps([e for e in entradas_exp if e['numero'] < '0023']))
        nombre = 'etf-fx5-arranque-' + secrets.token_hex(4)
        try:
            orden(['docker', 'run', '-d', '--name', nombre, '--network', 'none',
                   '--read-only', '--memory', '512m', '--memory-swap', '512m',
                   '--cpus', '1', '--pids-limit', '128', '--env-file', str(env),
                   '--tmpfs', '/var/lib/postgresql:rw,size=512m',
                   '--tmpfs', '/var/run/postgresql:rw,size=16m',
                   '--tmpfs', '/tmp:rw,size=16m',
                   '--mount', f'type=bind,source={tmp},target=/pruebas,readonly',
                   '--mount', f'type=bind,source={RAIZ},target=/repo,readonly',
                   '--mount', f'type=bind,source={RAIZ}/despliegue/base/01_roles.sh,target=/docker-entrypoint-initdb.d/01_roles.sh,readonly',
                   '--mount', f'type=bind,source={gate},target=/docker-entrypoint-initdb.d/02_esperar.sh,readonly',
                   IMAGEN], modulo='fx5')
            # El criterio antiguo da positivo mientras initdb sigue detenido.
            sondeo = ['docker', 'exec', nombre, 'sh', '-c',
                      'test -e /tmp/esperando && exec psql -X -U etf_instalador '
                      '-d etf -Atc "SELECT 1 / count(*) FROM pg_roles '
                      "WHERE rolname='etf_migrador'\""]
            limite = time.monotonic() + 60
            while True:
                resultado = subprocess.run(sondeo, text=True, capture_output=True, timeout=5)
                if resultado.returncode == 0 and resultado.stdout.strip() == '1':
                    break
                if time.monotonic() >= limite:
                    raise RuntimeError(diagnostico('fx5', 'initdb_no_disponible',
                                                  resultado.stdout, resultado.stderr))
                time.sleep(0.1)
            migrar = ['docker', 'exec', '--env-file', str(env), nombre, '/repo/bin/migrar']
            prematuro = subprocess.run(migrar, text=True, capture_output=True, timeout=15)
            if prematuro.returncode != 1 or '"msg":"migracion.error"' not in prematuro.stdout or prematuro.stderr:
                raise RuntimeError('fx5.carrera_no_reproducida')
            print('ARRANQUE-OK fallo reproducido: socket y rol listos; TCP cerrado; '
                  'migracion.error en stdout; stderr vacío', flush=True)
            with ThreadPoolExecutor(max_workers=1) as executor:
                espera = executor.submit(esperar_postgres, nombre, env, modulo='fx5')
                try:
                    time.sleep(0.3)
                    if espera.done():
                        espera.result()
                        raise RuntimeError('fx5.servidor_provisional_aceptado')
                finally:
                    orden(['docker', 'exec', nombre, 'touch', '/tmp/continuar'], modulo='fx5')
                espera.result(timeout=65)
            orden(migrar, modulo='fx5')
            orden(migrar[:-1] + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones',
                                '/repo/bin/migrar'], modulo='fx5')
            print('ARRANQUE-OK espera TCP autenticada; versión anterior y actualización', flush=True)
        finally:
            subprocess.run(['docker', 'rm', '-fv', nombre], stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL, timeout=30)


if __name__ == '__main__':
    main()
