#!/usr/bin/env python3
"""Ensaya estadistica en PostgreSQL 18 desechable, sin red ni datos reales."""
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
from ensayos import orden as ejecutar_orden, esperar_postgres

orden = partial(ejecutar_orden, modulo='estadistica')


def main():
    with tempfile.TemporaryDirectory(prefix='etf-f406-') as carpeta:
        tmp = pathlib.Path(carpeta)
        tmp.chmod(0o755)
        nombre = 'etf-f406-' + secrets.token_hex(4)
        claves = {k: secrets.token_hex(24) for k in ('ETF_MIGRADOR_CONTRASENA',
                  'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA', 'ETF_CALCULO_MEMORIA_CONTRASENA')}
        env = tmp / 'postgres.env'
        env.write_text('POSTGRES_USER=etf_instalador\nPOSTGRES_DB=etf\nPOSTGRES_PASSWORD=' +
                       secrets.token_hex(24) + '\n' + ''.join(k+'='+v+'\n' for k,v in claves.items()))
        env.chmod(0o600)
        runner = tmp / 'runner.env'
        runner.write_text('ETF_BD_HOST=127.0.0.1\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\n'
                          'ETF_BD_USUARIO=etf_app\nETF_POSTGRES_INTEGRACION=1\n'
                          'ETF_BD_CONTRASENA='+claves['ETF_APP_CONTRASENA']+'\n'
                          'ETF_CALCULO_MEMORIA_CONTRASENA='+claves['ETF_CALCULO_MEMORIA_CONTRASENA']+'\n'
                          'ETF_MIGRADOR_CONTRASENA='+claves['ETF_MIGRADOR_CONTRASENA']+'\n')
        runner.chmod(0o600)
        compilacion = dict(os.environ, CGO_ENABLED='0')
        orden(['go', 'build', '-o', str(tmp / 'migrar'), './cmd/migrar'], entorno=compilacion)
        orden(['go', 'test', '-c', '-o', str(tmp / 'estadistica.test'),
               './internal/adaptadores/postgres/estadistica'], entorno=compilacion)
        previo = tmp / 'previo'
        shutil.copytree(RAIZ / 'migraciones', previo, ignore=shutil.ignore_patterns('__pycache__'))
        shutil.rmtree(previo / 'estadistica')
        # Esta instantánea precede a expediente/0018, dependencia de carga/0001.
        shutil.rmtree(previo / 'carga')
        for modulo,numero in [('expediente','0010'),('documento','0010'),('auditoria','0002')]:
            manifest = previo / modulo / 'manifiesto.json'
            entradas = json.loads(manifest.read_text())
            for entry in entradas:
                if entry['numero']>=numero:
                    (previo / modulo / entry['archivo']).unlink()
            manifest.write_text(json.dumps([e for e in entradas if e['numero']<numero]))
        # La base previa al cierre conserva todos los módulos de main;
        # sólo excluye la nueva migración de cierre de Memoria.
        anterior_cierre = tmp / 'anterior_cierre'
        shutil.copytree(RAIZ / 'migraciones', anterior_cierre, ignore=shutil.ignore_patterns('__pycache__'))
        # La conservación depende del ejecutor de Consulta, posterior a este corte.
        manifest_exp = anterior_cierre / 'expediente' / 'manifiesto.json'
        entradas_exp = json.loads(manifest_exp.read_text())
        for entry in entradas_exp:
            if entry['numero'] >= '0023':
                (anterior_cierre / 'expediente' / entry['archivo']).unlink()
        manifest_exp.write_text(json.dumps([e for e in entradas_exp if e['numero'] < '0023']))
        manifest = anterior_cierre / 'estadistica' / 'manifiesto.json'
        entradas = json.loads(manifest.read_text())
        for entry in entradas:
            if entry['numero'] >= '0002':
                (anterior_cierre / 'estadistica' / entry['archivo']).unlink()
        manifest.write_text(json.dumps([e for e in entradas if e['numero'] < '0002']))
        # FX4 se ensaya también sobre la versión inmediatamente anterior.
        anterior_consulta = tmp / 'anterior_consulta'
        shutil.copytree(RAIZ / 'migraciones', anterior_consulta, ignore=shutil.ignore_patterns('__pycache__'))
        manifest = anterior_consulta / 'estadistica' / 'manifiesto.json'
        entradas = json.loads(manifest.read_text())
        for entry in entradas:
            if entry['numero'] >= '0006':
                (anterior_consulta / 'estadistica' / entry['archivo']).unlink()
        manifest.write_text(json.dumps([e for e in entradas if e['numero'] < '0006']))
        orden(['go', 'test', '-c', '-o', str(tmp / 'expediente.test'),
               './internal/adaptadores/postgres/expediente'], entorno=compilacion)


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
            esperar_postgres(nombre, runner, modulo='estadistica')
            ejecutar = ['docker', 'exec', '--env-file', str(runner), nombre]
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/previo', '/pruebas/migrar']), flush=True)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/anterior_cierre', '/pruebas/migrar']), flush=True)
            # Simular una instalación anterior, cuyo initdb aún no creó el rol.
            orden(sql,"DROP OWNED BY etf_calculo_memoria; DROP ROLE etf_calculo_memoria;")
            habilitar=['/bin/sh','/repo/despliegue/base/habilitar_calculo_memoria.sh']
            orden(ejecutar+habilitar)
            orden(ejecutar+habilitar)
            print('ROL-OK actualización e idempotencia del ejecutor técnico',flush=True)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/anterior_consulta', '/pruebas/migrar']), flush=True)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print('ENSAYO-OK base previa, actualización estadistica/0005 a 0006 y repetición', flush=True)
            pruebas_expediente = ['docker','exec','--env-file',str(runner),'--workdir','/repo/internal/adaptadores/postgres/expediente',nombre]
            pruebas = ['docker','exec','--env-file',str(runner),'--workdir','/repo/internal/adaptadores/postgres/estadistica',nombre]
            print(orden(pruebas_expediente + ['/pruebas/expediente.test', '-test.v', '-test.run=^TestMemoriaConsultaSEG3', '-test.timeout=60s']), flush=True)
            print(orden(pruebas + ['/pruebas/estadistica.test', '-test.v', '-test.timeout=60s']), flush=True)
            print(orden(pruebas_expediente + ['/pruebas/expediente.test', '-test.v', '-test.run=^(TestMemoria(Cierre|Consulta|Desglose)|TestLecturaEstadisticaColecciones|TestSPAFJulio|TestRESP1)', '-test.timeout=60s']), flush=True)
            print(orden(sql + ['-Atc', "SELECT 'AUDITORIA-OK ' || count(*) FROM auditoria.evento WHERE accion LIKE 'estadistica.%'"]), flush=True)
            mantenimiento = list(sql)
            mantenimiento[mantenimiento.index("-d") + 1] = "postgres"
            orden(mantenimiento, "DROP DATABASE etf WITH (FORCE); CREATE DATABASE etf OWNER etf_propietario;")
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']), flush=True)
            print('ENSAYO-OK instalación limpia', flush=True)
        finally:
            subprocess.run(['docker', 'rm', '-f', '-v', nombre], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=30)


if __name__ == '__main__':
    main()
