#!/usr/bin/env python3
"""Ejecuta F3.06 y F3.07 contra la demostración aislada de este worktree."""
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


def comprobar_persistencia(demo, evidencia):
    fixture = evidencia / 'persistencia-fixture.json'
    if not fixture.exists():
        return
    ids = json.loads(fixture.read_text())
    # Sólo se interpolan identificadores UUID generados por la API de esta demo.
    ids = {clave: str(UUID(valor)) for clave, valor in ids.items()}
    a, d, b, e = (ids[clave] for clave in ['origen', 'destino', 'seguimiento', 'forzado'])
    consulta = f"""
BEGIN READ ONLY;
SELECT json_build_object(
 'episodios_origen',(SELECT count(*) FROM expediente.episodio WHERE expediente_id='{a}'),
 'cierres_origen',(SELECT count(*) FROM expediente.episodio WHERE expediente_id='{a}' AND cierre_fecha IS NOT NULL),
 'traslados',(SELECT count(*) FROM expediente.traslado WHERE origen_id='{a}'),
 'episodios_destino',(SELECT count(*) FROM expediente.episodio WHERE expediente_id='{d}'),
 'seguimientos',(SELECT count(*) FROM expediente.seguimiento WHERE expediente_id='{b}'),
 'versiones_seguimiento',(SELECT json_agg(version ORDER BY version) FROM expediente.seguimiento_historia WHERE expediente_id='{b}'),
 'acciones_seguimiento',(SELECT json_agg(accion ORDER BY version) FROM expediente.seguimiento_historia WHERE expediente_id='{b}'),
 'anexo11',(SELECT count(*) FROM documento.emision WHERE expediente_id='{a}' AND tipo_documento='finalizacion/anexo11'),
 'anexo12',(SELECT count(*) FROM documento.emision WHERE expediente_id='{b}' AND tipo_documento='finalizacion/anexo12'),
 'forzados',(SELECT count(*) FROM expediente.spm_forzado WHERE expediente_id='{e}'),
 'forzado_auditado',(SELECT count(*) FROM auditoria.evento a JOIN expediente.spm_forzado f ON f.id=a.entidad_id WHERE f.expediente_id='{e}' AND a.accion='expediente.spm_forzar' AND a.resultado='ok'),
 'cierres_misma_persona',(SELECT count(*) FROM expediente.episodio ep JOIN expediente.propuesta_cierre p ON p.id=ep.propuesta_cierre_id WHERE ep.expediente_id IN ('{a}','{b}','{d}','{e}') AND ep.cerrado_por=p.propuesta_por)
);
ROLLBACK;
"""
    resultado = demo.compose('exec', '-T', 'postgres', 'psql', '-X', '-U', 'etf_instalador', '-d', 'etf',
                             '-v', 'ON_ERROR_STOP=1', '-Atq', input=consulta,
                             capture_output=True, text=True)
    datos = json.loads(resultado.stdout.strip())
    esperado = {
        'episodios_origen': 2, 'cierres_origen': 2, 'traslados': 1, 'episodios_destino': 1,
        'seguimientos': 1, 'versiones_seguimiento': [1, 2, 3],
        'acciones_seguimiento': ['iniciar', 'terminar', 'reabrir'], 'anexo11': 1, 'anexo12': 1,
        'forzados': 1, 'forzado_auditado': 1, 'cierres_misma_persona': 0,
    }
    (evidencia / 'persistencia.json').write_text(json.dumps({'esperado': esperado, 'observado': datos}, indent=2) + '\n')
    if datos != esperado:
        raise AssertionError(f'Persistencia: {datos}; esperado: {esperado}')
    print('F3.06-PERSISTENCIA-OK: dos episodios, un traslado, tres versiones de seguimiento, emisiones únicas, forzado auditado y personas distintas.', flush=True)


def main():
    inicio = time.monotonic()
    temporal = Path(os.environ.get('TMPDIR', str(RAIZ / 'bin/temporales')))
    temporal.mkdir(parents=True, exist_ok=True)
    evidencia = Path(tempfile.mkdtemp(prefix='f306-f307-', dir=temporal))
    entorno = dict(os.environ)
    if not (RAIZ / 'despliegue/demo/.local/puerto').exists() and not entorno.get('ETF_DEMO_PUERTO'):
        with socket.socket() as s:
            s.bind(('127.0.0.1', 0))
            entorno['ETF_DEMO_PUERTO'] = str(s.getsockname()[1])
    os.environ.update({k: v for k, v in entorno.items() if k.startswith('ETF_DEMO_')})
    spec = importlib.util.spec_from_file_location('demo_f306', RAIZ / 'despliegue/demo/demo.py')
    demo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(demo)
    print(f'F3.06/F3.07-EVIDENCIA: {evidencia}', flush=True)
    # F0.11 puede agotar el límite de consultas de sesión antes de make demo.
    if (demo.LOCAL / 'entorno.env').exists():
        activos = demo.compose('ps', '--status', 'running', '-q', 'app', capture_output=True, text=True).stdout.strip()
        if activos:
            demo.compose('restart', '--no-deps', 'app', stdout=subprocess.DEVNULL)
    # La salida de make demo contiene las contraseñas de las cuentas inventadas.
    with (evidencia / 'demo.log').open('w') as salida:
        subprocess.run(['make', 'demo'], cwd=RAIZ, env=entorno, check=True,
                       timeout=600, stdout=salida, stderr=subprocess.STDOUT)
    # Reinicia sólo la app de este worktree para restablecer los límites de entrada.
    demo.compose('up', '--detach', '--force-recreate', '--no-deps', '--wait', 'app',
                 stdout=subprocess.DEVNULL)
    recorrido = subprocess.run(['node', 'scripts/recorridos/salidas-spaf.mjs', demo.URL, str(evidencia)],
                               cwd=RAIZ, env=entorno, timeout=600)
    comprobar_persistencia(demo, evidencia)
    if recorrido.returncode:
        print(f'F3.06/F3.07-FALLO: {time.monotonic() - inicio:.1f} s; evidencia en {evidencia}', flush=True)
        raise SystemExit(recorrido.returncode)
    print(f'F3.06/F3.07-OK: {time.monotonic() - inicio:.1f} s; evidencia en {evidencia}')


if __name__ == '__main__':
    main()
