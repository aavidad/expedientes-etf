#!/usr/bin/env python3
"""Comprueba accesibilidad sólo en la demostración local de este worktree."""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import urllib.request

RAIZ = Path(__file__).resolve().parents[2]


class SinRedireccion(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise RuntimeError('La consulta de textos debe permanecer en la demo local')


def preparar_reserva(demo):
    # Renueva sólo ficheros de reserva que no aportan los textos del árbol actual.
    # No modifica versiones publicadas de paquetes ni sus referencias activas.
    cliente = urllib.request.build_opener(urllib.request.ProxyHandler({}), SinRedireccion())
    copiados = 0
    for idioma in ('es', 'en'):
        with cliente.open(f'{demo.URL}/api/idiomas/{idioma}/textos', timeout=10) as respuesta:
            actuales = json.load(respuesta)
        for reserva in sorted((RAIZ / 'idiomas' / idioma / 'textos').glob('*.json')):
            esperados = json.loads(reserva.read_text())
            faltan = any(isinstance(v, str) and v.strip() and
                         actuales.get(k) != v
                         for k, v in esperados.items())
            if faltan:
                demo.compose('cp', str(reserva), f'app:/datos/idiomas/{idioma}/textos/{reserva.name}',
                             capture_output=True, text=True)
                copiados += 1
    print(f'F5.03-TEXTOS: {copiados} reservas locales renovadas.')


def main():
    temporal = Path(os.environ.get('TMPDIR', str(RAIZ / 'bin/temporales')))
    temporal.mkdir(parents=True, exist_ok=True)
    evidencia = Path(tempfile.mkdtemp(prefix='f503-', dir=temporal))
    entorno = {k: v for k, v in os.environ.items() if k not in ('ETF_AXE_PERFIL', 'ETF_AXE_RUTA', 'ETF_AXE_DEBUG', 'ETF_AXE_CAPTURAS')}
    if not (RAIZ / 'despliegue/demo/.local/puerto').exists() and not entorno.get('ETF_DEMO_PUERTO'):
        with socket.socket() as s:
            s.bind(('127.0.0.1', 0))
            entorno['ETF_DEMO_PUERTO'] = str(s.getsockname()[1])
    os.environ.update({k: v for k, v in entorno.items() if k.startswith('ETF_DEMO_')})
    spec = importlib.util.spec_from_file_location('demo_f503', RAIZ / 'despliegue/demo/demo.py')
    demo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(demo)
    fallo = False
    try:
        with (evidencia / 'demo.log').open('w') as salida:
            subprocess.run(['make', 'demo'], cwd=RAIZ, env=entorno, stdout=salida,
                           stderr=subprocess.STDOUT, check=True, timeout=600)
        preparar_reserva(demo)
        # Reinicio acotado: la auditoría no hereda el contador de intentos de otros recorridos.
        demo.compose('restart', '--no-deps', 'app', stdout=subprocess.DEVNULL)
        subprocess.run(['node', 'scripts/recorridos/accesibilidad.mjs', demo.URL, str(evidencia)],
                       cwd=RAIZ, check=True, timeout=1200)
        print('F5.03-OK: axe sin infracciones serias ni críticas.')
    except Exception:
        fallo = True
        print(f'F5.03-ERROR: evidencia local en {evidencia}')
        raise
    finally:
        subprocess.run(['make', 'demo-parar'], cwd=RAIZ, env=entorno, check=True, timeout=120)
        if not fallo:
            shutil.rmtree(evidencia)


if __name__ == '__main__':
    main()
