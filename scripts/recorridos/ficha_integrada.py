#!/usr/bin/env python3
"""Prepara la demo de este worktree y ejecuta F2.07 con Chrome del sistema."""
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time
from uuid import UUID

RAIZ = Path(__file__).resolve().parents[2]


def comprobar_rls(demo, evidencia):
    resultado = json.loads((evidencia / 'resultado.json').read_text())
    expediente = str(UUID(resultado['expediente']))
    tablas = ['adulto', 'familia_revision', 'adultos_revision', 'adulto_revision',
              'menores_revision', 'menor_revision', 'intervencion_revision']
    proyeccion = ','.join(
        f"'{tabla}',(SELECT count(*) FROM expediente.{tabla} WHERE expediente_id='{expediente}')"
        for tabla in tablas)
    observado = {}
    for cuenta in ['marta.cabrera', 'pablo.ruiz']:
        consulta = f"""
BEGIN READ ONLY;
SET LOCAL statement_timeout='10s';
SELECT set_config('app.usuario_id',(SELECT id::text FROM identidad.usuario WHERE cuenta_ad='{cuenta}'),true);
SELECT set_config('app.perfiles','profesional_etf',true);
SELECT set_config('app.equipos',ARRAY[(SELECT e.equipo_id FROM identidad.usuario_equipo e JOIN identidad.usuario u ON u.id=e.usuario_id WHERE u.cuenta_ad='{cuenta}' AND e.desde <= clock_timestamp() AND (e.hasta IS NULL OR e.hasta > clock_timestamp()))]::text,true);
SET LOCAL ROLE etf_app;
SELECT json_build_object('rol',current_user,'expediente',
 (SELECT count(*) FROM expediente.expediente WHERE id='{expediente}'),{proyeccion});
ROLLBACK;
"""
        r = demo.compose('exec', '-T', 'postgres', 'psql', '-X', '-U', 'etf_instalador', '-d', 'etf',
                         '-v', 'ON_ERROR_STOP=1', '-Atq', input=consulta, capture_output=True, text=True)
        observado[cuenta] = json.loads(next(linea for linea in r.stdout.splitlines() if linea.startswith('{"rol"')))
        assert observado[cuenta]['rol'] == 'etf_app'
        for tabla in ['expediente', *tablas]:
            cantidad = observado[cuenta][tabla]
            assert cantidad > 0 if cuenta == 'marta.cabrera' else cantidad == 0, (cuenta, tabla, cantidad)
    (evidencia / 'rls.json').write_text(json.dumps(observado, indent=2) + '\n')
    print('F2.07-RLS-OK: etf_app; expediente y siete tablas pobladas, propietario > 0, otro equipo = 0.', flush=True)


def main():
    inicio = time.monotonic()
    temporal = Path(os.environ.get('TMPDIR', str(RAIZ / 'bin/temporales')))
    temporal.mkdir(parents=True, exist_ok=True)
    evidencia = Path(tempfile.mkdtemp(prefix='f207-', dir=temporal))
    if not (RAIZ / 'despliegue/demo/.local/puerto').exists() and not os.environ.get('ETF_DEMO_PUERTO'):
        with socket.socket() as s:
            s.bind(('127.0.0.1', 0))
            os.environ['ETF_DEMO_PUERTO'] = str(s.getsockname()[1])
    spec = importlib.util.spec_from_file_location('demo_f207', RAIZ / 'despliegue/demo/demo.py')
    demo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(demo)
    print(f'F2.07-EVIDENCIA: {evidencia}', flush=True)
    if (demo.LOCAL / 'entorno.env').exists():
        activos = demo.compose('ps', '--status', 'running', '-q', 'app', capture_output=True, text=True).stdout.strip()
        if activos:
            demo.compose('restart', '--no-deps', 'app', stdout=subprocess.DEVNULL)
    # La demo imprime contraseñas sintéticas; el registro permanece fuera de Git.
    with (evidencia / 'demo.log').open('w') as salida:
        subprocess.run(['make', 'demo'], cwd=RAIZ, check=True, timeout=600,
                       stdout=salida, stderr=subprocess.STDOUT)
    demo.compose('up', '--detach', '--force-recreate', '--no-deps', '--wait', 'app', stdout=subprocess.DEVNULL)
    resultado = subprocess.run(['node', 'scripts/recorridos/ficha-integrada.mjs', demo.URL, str(evidencia)],
                               cwd=RAIZ, timeout=600)
    comprobar_rls(demo, evidencia)
    print(f'F2.07-{"FALLO" if resultado.returncode else "OK"}: {time.monotonic() - inicio:.1f} s; evidencia en {evidencia}', flush=True)
    raise SystemExit(resultado.returncode)


if __name__ == '__main__':
    main()
