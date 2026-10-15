#!/usr/bin/env python3
"""Ensaya actividad en PostgreSQL 18 desechable, sin red ni datos reales."""
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
IMAGEN = 'postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15'


sys.path.insert(0, str(RAIZ / 'scripts' / 'ci'))
from ensayos import orden as ejecutar_orden, esperar_postgres, recortar_dependientes

orden = partial(ejecutar_orden, modulo='actividad')


def main():
    with tempfile.TemporaryDirectory(prefix='etf-f204-') as carpeta:
        tmp = pathlib.Path(carpeta)
        tmp.chmod(0o755)
        nombre = 'etf-f204-' + secrets.token_hex(4)
        claves = {k: secrets.token_hex(24) for k in ('ETF_MIGRADOR_CONTRASENA',
                  'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA')}
        env = tmp / 'postgres.env'
        env.write_text('POSTGRES_USER=etf_instalador\nPOSTGRES_DB=etf\nPOSTGRES_PASSWORD=' +
                       secrets.token_hex(24) + '\n' + ''.join(k+'='+v+'\n' for k,v in claves.items()))
        env.chmod(0o600)
        runner = tmp / 'runner.env'
        runner.write_text('ETF_BD_HOST=127.0.0.1\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\n'
                          'ETF_BD_USUARIO=etf_app\nETF_ACTIVIDAD_POSTGRES_INTEGRACION=1\n'
                          'ETF_BD_CONTRASENA='+claves['ETF_APP_CONTRASENA']+'\n'
                          'ETF_MIGRADOR_CONTRASENA='+claves['ETF_MIGRADOR_CONTRASENA']+'\n')
        runner.chmod(0o600)
        compilacion = dict(os.environ, CGO_ENABLED='0')
        orden(['go', 'build', '-o', str(tmp / 'migrar'), './cmd/migrar'], entorno=compilacion)
        orden(['go', 'test', '-c', '-o', str(tmp / 'actividad.test'),
               './internal/adaptadores/postgres/actividad'], entorno=compilacion)
        previo = tmp / 'previo'
        shutil.copytree(RAIZ / 'migraciones', previo, ignore=shutil.ignore_patterns('__pycache__'))
        shutil.rmtree(previo / 'actividad')
        # La base anterior a Actividad tampoco puede contener sus consumidores.
        manifiesto = previo / 'estadistica' / 'manifiesto.json'
        entradas = json.loads(manifiesto.read_text())
        excluidas = [e for e in entradas if any(d['modulo'] == 'actividad' for d in e['depende_de'])]
        for entrada in excluidas:
            (previo / 'estadistica' / entrada['archivo']).unlink()
        manifiesto.write_text(json.dumps([e for e in entradas if e not in excluidas]))
        # Conservación consume el ejecutor de Consulta retirado en este corte.
        manifiesto_exp = previo / 'expediente' / 'manifiesto.json'
        entradas_exp = json.loads(manifiesto_exp.read_text())
        for entrada in entradas_exp:
            if entrada['numero'] >= '0023':
                (previo / 'expediente' / entrada['archivo']).unlink()
        manifiesto_exp.write_text(json.dumps([e for e in entradas_exp if e['numero'] < '0023']))
        recortar_dependientes(previo)
        montar = ['--mount', f'type=bind,source={tmp},target=/pruebas,readonly',
                  '--mount', f'type=bind,source={RAIZ},target=/repo,readonly']
        try:
            orden(['docker', 'run', '-d', '--name', nombre, '--network', 'none',
                   '--read-only', '--memory', '512m', '--memory-swap', '512m',
                   '--cpus', '1', '--pids-limit', '128', '--env-file', str(env),
                   '--tmpfs', '/var/lib/postgresql:rw,size=512m',
                   '--tmpfs', '/var/run/postgresql:rw,size=16m',
                   '--tmpfs', '/tmp:rw,size=16m', *montar,
                   '--mount', f'type=bind,source={RAIZ}/despliegue/base/01_roles.sh,target=/docker-entrypoint-initdb.d/01_roles.sh,readonly',
                   IMAGEN])
            sql = ['docker', 'exec', '-i', nombre, 'psql', '-U', 'etf_instalador',
                   '-d', 'etf', '-v', 'ON_ERROR_STOP=1']
            esperar_postgres(nombre, runner, modulo='actividad')
            ejecutar = ['docker', 'exec', '--env-file', str(runner), nombre]
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/previo', '/pruebas/migrar']), flush=True)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print('ENSAYO-OK base anterior, ampliación y repetición', flush=True)
            print(orden(ejecutar + ['/pruebas/actividad.test', '-test.v', '-test.timeout=60s']), flush=True)
            print(orden(sql + ['-Atc', "SELECT 'AUDITORIA-OK ' || count(*) FROM auditoria.evento WHERE accion LIKE 'actividad.%'"]), flush=True)
            mantenimiento = list(sql)
            mantenimiento[mantenimiento.index("-d") + 1] = "postgres"
            orden(mantenimiento, "DROP DATABASE etf WITH (FORCE); CREATE DATABASE etf OWNER etf_propietario;")
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print('ENSAYO-OK instalación limpia', flush=True)
        finally:
            subprocess.run(['docker', 'rm', '-f', '-v', nombre], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30)


if __name__ == '__main__':
    main()
