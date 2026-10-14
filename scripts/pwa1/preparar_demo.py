#!/usr/bin/env python3
"""Renueva sólo los recursos públicos de PWA1 en la demo de este worktree."""
import importlib.util
from pathlib import Path
import sys
from urllib.parse import urlsplit

RAIZ = Path(__file__).resolve().parents[2]


def preparar(base):
    spec = importlib.util.spec_from_file_location('demo_pwa1', RAIZ / 'despliegue/demo/demo.py')
    demo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(demo)
    actual, esperado = urlsplit(base), urlsplit(demo.URL)
    if (actual.scheme != esperado.scheme or actual.hostname not in ('127.0.0.1', 'localhost')
            or actual.port != esperado.port or actual.username or actual.password
            or actual.path not in ('', '/') or actual.query or actual.fragment):
        raise SystemExit('pwa1.demo_distinta')
    # Reconstruir la imagen no renueva los volúmenes. No se borran datos,
    # versiones publicadas ni preferencias; se actualiza sólo la reserva PWA1.
    archivos = list((RAIZ / 'idiomas').glob('*/textos/pwa1.json'))
    archivos += list((RAIZ / 'temas').glob('*/iconos/etf-*.png'))
    for archivo in sorted(archivos):
        destino = '/datos/' + archivo.relative_to(RAIZ).as_posix()
        demo.compose('exec', '-T', 'app', 'mkdir', '-p', str(Path(destino).parent),
                     capture_output=True, text=True)
        demo.compose('cp', str(archivo), 'app:' + destino, capture_output=True, text=True)
    print(f'PWA1-DEMO: {len(archivos)} recursos públicos renovados en la demo local.')


if __name__ == '__main__':
    preparar(sys.argv[1])
