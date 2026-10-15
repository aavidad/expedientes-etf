#!/usr/bin/env python3
"""Instala los recursos de distribución en la demo de este worktree (FX8)."""
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
    demo.incorporar_recursos()
    print('PWA1-DEMO: recursos de distribución instalados; versiones publicadas conservadas.')


if __name__ == '__main__':
    preparar(sys.argv[1])
