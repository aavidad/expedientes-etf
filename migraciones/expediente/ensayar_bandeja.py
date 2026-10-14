#!/usr/bin/env python3
"""Ensaya la bandeja y su historial en PostgreSQL 18 con datos inventados."""
import os
import secrets
import shutil
import tempfile
from pathlib import Path
from ensayar_aceptacion import orden, esperar_postgres, POSTGRES, retirar_desde

RAIZ = Path(__file__).resolve().parents[2]


def main():
    with tempfile.TemporaryDirectory(prefix='etf-f206-') as carpeta:
        tmp = Path(carpeta)
        tmp.chmod(0o755)
        nombre = 'etf-f206-' + secrets.token_hex(4)
        claves = {k: secrets.token_hex(24) for k in ('ETF_MIGRADOR_CONTRASENA', 'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA')}
        env = tmp / 'base.env'
        env.write_text('POSTGRES_USER=etf_instalador\nPOSTGRES_DB=etf\nPOSTGRES_PASSWORD=' + secrets.token_hex(24) + '\n' + ''.join(k + '=' + v + '\n' for k, v in claves.items()))
        env.chmod(0o600)
        runner = tmp / 'runner.env'
        runner.write_text('ETF_BD_HOST=127.0.0.1\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\nETF_BD_USUARIO=etf_app\nETF_POSTGRES_INTEGRACION=1\nETF_BANDEJA_POSTGRES_INTEGRACION=1\nETF_BD_CONTRASENA=' + claves['ETF_APP_CONTRASENA'] + '\nETF_MIGRADOR_CONTRASENA=' + claves['ETF_MIGRADOR_CONTRASENA'] + '\n')
        runner.chmod(0o600)
        compilacion = dict(os.environ, CGO_ENABLED='0')
        orden(['go', 'build', '-o', str(tmp / 'migrar'), './cmd/migrar'], entorno=compilacion)
        orden(['go', 'test', '-c', '-o', str(tmp / 'expediente.test'), './internal/adaptadores/postgres/expediente'], entorno=compilacion)
        previo = tmp / 'previo'
        shutil.copytree(RAIZ / 'migraciones', previo, ignore=shutil.ignore_patterns('__pycache__'))
        # La versión anterior excluye también las entregas posteriores al corte.
        retirar_desde(previo, [('auditoria', '0002'), ('expediente', '0020'), ('estadistica', '0005')])
        try:
            orden(['docker', 'run', '-d', '--name', nombre, '--network', 'none', '--env-file', str(env), '--tmpfs', '/var/lib/postgresql:rw,size=512m', '--mount', f'type=bind,source={tmp},target=/pruebas,readonly', '--mount', f'type=bind,source={RAIZ},target=/repo,readonly', '--mount', f'type=bind,source={RAIZ}/despliegue/base/01_roles.sh,target=/docker-entrypoint-initdb.d/01_roles.sh,readonly', POSTGRES])
            sql = ['docker', 'exec', nombre, 'psql', '-U', 'etf_instalador', '-d', 'etf', '-v', 'ON_ERROR_STOP=1', '-Atc']
            esperar_postgres(nombre, runner, modulo='expediente')
            ejecutar = ['docker', 'exec', '--env-file', str(runner), nombre]
            for ruta in ['/pruebas/previo', '/repo/migraciones', '/repo/migraciones']:
                print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=' + ruta, '/pruebas/migrar']))
            prueba = ['/pruebas/expediente.test', '-test.v', '-test.run', '^TestBandejaEquipo']
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir', '/repo/internal/adaptadores/postgres/expediente', nombre] + prueba))
            orden(sql + ['CREATE DATABASE etf_limpia OWNER etf_propietario'])
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_limpia', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir', '/repo/internal/adaptadores/postgres/expediente', nombre, 'env', 'ETF_BD_NOMBRE=etf_limpia'] + prueba))
            print('ENSAYO-OK F2.06: instalación y actualización, iniciales, RLS, perfil doble, historial y fallo de auditoría')
        finally:
            import subprocess
            subprocess.run(['docker', 'rm', '-fv', nombre], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


if __name__ == '__main__':
    main()
