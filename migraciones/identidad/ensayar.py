#!/usr/bin/env python3
"""Ensaya SEG2 desde la versión anterior y desde cero con PostgreSQL 18 local."""
import json
import os
from pathlib import Path
import secrets
import shutil
import subprocess
import sys
import tempfile

RAIZ = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(RAIZ / 'scripts/ci'))
from ensayos import orden, esperar_postgres

POSTGRES = 'postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15'


def main():
    with tempfile.TemporaryDirectory(prefix='etf-seg2-') as carpeta:
        tmp = Path(carpeta)
        tmp.chmod(0o755)
        nombre = 'etf-seg2-' + secrets.token_hex(4)
        red = nombre + '-red'
        claves = {k: secrets.token_hex(24) for k in ('ETF_MIGRADOR_CONTRASENA', 'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA')}
        env = tmp / 'base.env'
        env.write_text('POSTGRES_USER=etf_instalador\nPOSTGRES_DB=etf\nPOSTGRES_PASSWORD=' + secrets.token_hex(24) + '\n' + ''.join(k + '=' + v + '\n' for k, v in claves.items()))
        env.chmod(0o600)
        runner = tmp / 'runner.env'
        runner.write_text('ETF_BD_HOST=127.0.0.1\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\nETF_MIGRADOR_CONTRASENA=' + claves['ETF_MIGRADOR_CONTRASENA'] + '\n')
        runner.chmod(0o600)
        orden(['go', 'build', '-o', str(tmp / 'migrar'), './cmd/migrar'], entorno=dict(os.environ, CGO_ENABLED='0'))
        previo = tmp / 'previo'
        shutil.copytree(RAIZ / 'migraciones', previo, ignore=shutil.ignore_patterns('__pycache__'))
        manifiesto = previo / 'identidad/manifiesto.json'
        entradas = json.loads(manifiesto.read_text())
        for e in entradas:
            if e['numero'] >= '0003':
                (manifiesto.parent / e['archivo']).unlink()
        manifiesto.write_text(json.dumps([e for e in entradas if e['numero'] < '0003']))
        try:
            orden(['docker', 'network', 'create', '--internal', '--subnet', f'10.252.{32 + secrets.randbelow(192)}.0/28', red])
            orden(['docker', 'run', '-d', '--name', nombre, '--network', red, '--read-only', '--memory', '512m', '--memory-swap', '512m', '--cpus', '1', '--pids-limit', '128', '--env-file', str(env), '--tmpfs', '/var/lib/postgresql:rw,size=512m', '--tmpfs', '/var/run/postgresql:rw,size=16m', '--tmpfs', '/tmp:rw,size=16m', '--mount', f'type=bind,source={tmp},target=/pruebas,readonly', '--mount', f'type=bind,source={RAIZ},target=/repo,readonly', '--mount', f'type=bind,source={RAIZ}/despliegue/base/01_roles.sh,target=/docker-entrypoint-initdb.d/01_roles.sh,readonly', POSTGRES])
            esperar_postgres(nombre, runner, modulo='seg2')
            ejecutar = ['docker', 'exec', '--env-file', str(runner), nombre]
            for ruta in ['/pruebas/previo', '/repo/migraciones', '/repo/migraciones']:
                print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=' + ruta, '/pruebas/migrar']))
            direccion = orden(['docker', 'inspect', '--format', '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}', nombre])
            entorno = dict(os.environ, ETF_BD_HOST=direccion, ETF_BD_PUERTO='5432', ETF_BD_NOMBRE='etf', ETF_BD_USUARIO='etf_app', ETF_BD_SSLMODE='disable', ETF_BD_CONTRASENA=claves['ETF_APP_CONTRASENA'], ETF_MIGRADOR_CONTRASENA=claves['ETF_MIGRADOR_CONTRASENA'], ETF_POSTGRES_CONTENEDOR=nombre, ETF_POSTGRES_INTEGRACION='1')
            for k in list(entorno):
                if k.startswith('PG'):
                    del entorno[k]
            def probar():
                resultado = orden(['go', 'test', '-count=1', '-run', '^TestIntegracionPerfiles', '-v', './internal/adaptadores/postgres'], entorno=entorno)
                if '--- SKIP:' in resultado or 'SEG2-RLS-OK:' not in resultado:
                    raise RuntimeError('seg2.prueba_no_ejecutada')
                print(resultado)
            probar()
            orden(['docker', 'exec', nombre, 'psql', '-U', 'etf_instalador', '-d', 'etf', '-v', 'ON_ERROR_STOP=1', '-c', 'CREATE DATABASE etf_limpia OWNER etf_propietario'])
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_limpia', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            entorno['ETF_BD_NOMBRE'] = 'etf_limpia'
            probar()
            print('ENSAYO-OK SEG2: actualización, repetición, instalación limpia, autocambios, RLS, auditoría y alta inicial')
        finally:
            subprocess.run(['docker', 'rm', '-fv', nombre], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            subprocess.run(['docker', 'network', 'rm', red], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


if __name__ == '__main__':
    main()
