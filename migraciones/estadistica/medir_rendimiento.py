#!/usr/bin/env python3
"""Mide SQL real sobre PostgreSQL 18 desechable con datos inventados."""
import argparse
import json
import os
import pathlib
import re
import secrets
import subprocess
import tempfile
import time

RAIZ = pathlib.Path(__file__).resolve().parents[2]
IMAGEN = 'postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15'


def orden(args, entrada=None, timeout=180, entorno=None):
    r = subprocess.run(args, input=entrada, text=True, capture_output=True, timeout=timeout, env=entorno)
    if r.returncode:
        if '/pruebas/migrar' in args or '/pruebas/rendimiento.test' in args:
            print(r.stdout, flush=True)
        raise RuntimeError('rendimiento.orden_fallida: ' + r.stderr[:2000])
    return r.stdout.strip()


def consultas():
    zona = RAIZ / 'internal/adaptadores/postgres/expediente'
    memoria = re.search(r'`(SELECT e.id::text,e.equipo_id::text,coalesce\(e.numero.*?ORDER BY e.id)`',
                        (zona / 'estadistica.go').read_text(), re.S).group(1)
    bandeja = re.search(r'`(WITH base AS MATERIALIZED.*?)`',
                        (zona / 'bandeja.go').read_text(), re.S).group(1)
    codigo = (zona / 'bandeja_equipo.go').read_text()
    equipo = re.search(r'const proyeccionEquipo = `(.*?)`', codigo, re.S).group(1)
    equipo += re.search(r'proyeccionEquipo\+`(.*?)`', codigo, re.S).group(1)
    return memoria, bandeja, equipo


def medir(sql, nombre, consulta, tipos, parametros, perfil, equipo, destino):
    identidad = "SELECT set_config('app.usuario_id','40404040-4040-4040-8040-404040404040',true)," \
                f"set_config('app.perfiles','{perfil}',true),set_config('app.equipos','{{{equipo}}}',true);"
    sentencia = f"BEGIN; SET LOCAL ROLE etf_app; {identidad} PREPARE medida({tipos}) AS {consulta};\n" \
                f"EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) EXECUTE medida({parametros}); ROLLBACK;"
    salida = orden(sql + ['-qAt'], sentencia)
    inicio = salida.index('[\n')
    plan = json.JSONDecoder().raw_decode(salida[inicio:])[0]
    (destino / (nombre + '.json')).write_text(json.dumps(plan, indent=2) + '\n')
    p = plan[0]
    return p['Execution Time']


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--salida', type=pathlib.Path, required=True)
    args = parser.parse_args()
    args.salida.mkdir(parents=True, exist_ok=True)
    memoria, bandeja, equipo = consultas()
    with tempfile.TemporaryDirectory(prefix='etf-f404-') as carpeta:
        tmp = pathlib.Path(carpeta)
        tmp.chmod(0o755)
        nombre = 'etf-f404-' + secrets.token_hex(4)
        claves = {k: secrets.token_hex(24) for k in ('ETF_MIGRADOR_CONTRASENA', 'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA')}
        env = tmp / 'postgres.env'
        env.write_text('POSTGRES_USER=etf_instalador\nPOSTGRES_DB=etf\nPOSTGRES_PASSWORD=' + secrets.token_hex(24) + '\n' +
                       ''.join(k + '=' + v + '\n' for k, v in claves.items()))
        env.chmod(0o600)
        runner = tmp / 'runner.env'
        runner.write_text('ETF_BD_HOST=127.0.0.1\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\nETF_BD_USUARIO=etf_app\n'
                          'ETF_RENDIMIENTO=1\n'
                          'ETF_BD_CONTRASENA=' + claves['ETF_APP_CONTRASENA'] + '\n'
                          'ETF_MIGRADOR_CONTRASENA=' + claves['ETF_MIGRADOR_CONTRASENA'] + '\n')
        runner.chmod(0o600)
        compilacion = dict(os.environ, CGO_ENABLED='0', GOCACHE=str(tmp / 'go-cache'))
        orden(['go', 'build', '-o', str(tmp / 'migrar'), './cmd/migrar'], entorno=compilacion)
        orden(['go', 'test', '-c', '-o', str(tmp / 'rendimiento.test'), './internal/adaptadores/postgres/expediente'], entorno=compilacion)
        try:
            orden(['docker', 'run', '-d', '--name', nombre, '--network', 'none', '--read-only',
                   '--memory', '768m', '--memory-swap', '768m', '--cpus', '1', '--pids-limit', '128',
                   '--env-file', str(env), '--tmpfs', '/var/lib/postgresql:rw,size=512m',
                   '--tmpfs', '/var/run/postgresql:rw,size=16m', '--tmpfs', '/tmp:rw,size=16m',
                   '--mount', f'type=bind,source={tmp},target=/pruebas,readonly',
                   '--mount', f'type=bind,source={RAIZ},target=/repo,readonly',
                   '--mount', f'type=bind,source={RAIZ}/despliegue/base/01_roles.sh,target=/docker-entrypoint-initdb.d/01_roles.sh,readonly', IMAGEN])
            sql = ['docker', 'exec', '-i', nombre, 'psql', '-U', 'etf_instalador', '-d', 'etf', '-v', 'ON_ERROR_STOP=1']
            for _ in range(100):
                r = subprocess.run(sql + ['-Atc', "SELECT 1 / count(*) FROM pg_roles WHERE rolname='etf_migrador'"], capture_output=True)
                if r.returncode == 0:
                    break
                time.sleep(0.2)
            else:
                raise RuntimeError('rendimiento.postgres_no_disponible')
            orden(['docker', 'exec', '--env-file', str(runner), nombre, 'env',
                   'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar'])
            orden(sql, (RAIZ / 'migraciones/estadistica/sembrar_rendimiento.fixture').read_text())
            eid = orden(sql + ['-Atc', "SELECT id FROM catalogo.equipo WHERE codigo='01'"])
            print(orden(sql + ['-Atc', "SELECT 'expedientes='||count(*)||'; equipos='||count(DISTINCT equipo_id)||'; años='||count(DISTINCT anio_numero) FROM expediente.expediente"]), flush=True)
            resultados = {}
            for nombre_medida, consulta, tipos, parametros, perfil, equipo_id in [
                ('memoria_provincial', memoria, 'text,timestamptz', "'','2026-12-31T23:59:59Z'", 'coordinacion', ''),
                ('spaf_equipo', memoria, 'text,timestamptz', f"'{eid}','2026-12-31T23:59:59Z'", 'profesional_etf', eid),
                ('bandeja_provincial', bandeja, 'text,text,text,text,text,text,text,int,int', "'','','','','','','',25,0", 'coordinacion', ''),
                ('bandeja_equipo', equipo, 'text,text,text,text,int,int', "'','','','',25,0", 'profesional_etf', eid),
            ]:
                muestras = [medir(sql, f'{nombre_medida}.{i+1:02}', consulta, tipos, parametros, perfil, equipo_id, args.salida) for i in range(20)]
                resultados[nombre_medida] = {'muestras_ms': muestras, 'p95_ms': sorted(muestras)[18]}
                print(f"{nombre_medida}: SQL p95={resultados[nombre_medida]['p95_ms']:.3f} ms; muestras=20", flush=True)
            (args.salida / 'tiempos.json').write_text(json.dumps(resultados, indent=2) + '\n')
            if resultados['memoria_provincial']['p95_ms'] >= 2000 or any(resultados[k]['p95_ms'] >= 300 for k in ('spaf_equipo', 'bandeja_provincial', 'bandeja_equipo')):
                raise RuntimeError('rendimiento.objetivo_incumplido')
            motor = orden(['docker', 'exec', '--env-file', str(runner), '--workdir',
                           '/repo/internal/adaptadores/postgres/expediente', nombre,
                           '/pruebas/rendimiento.test', '-test.v', '-test.run', '^TestRendimientoMemoria3000$', '-test.timeout=60s'])
            (args.salida / 'motor.txt').write_text(motor + '\n')
            print(motor, flush=True)
        finally:
            subprocess.run(['docker', 'rm', '-fv', nombre], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)


if __name__ == '__main__':
    main()
