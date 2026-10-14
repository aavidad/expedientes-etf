#!/usr/bin/env python3
"""Comprueba la Memoria integrada contra la demo local del worktree (F4.09)."""
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time
from zipfile import ZipFile
from xml.etree import ElementTree

RAIZ = Path(__file__).resolve().parents[2]


def comprobar_word(evidencia):
    casos = json.loads((evidencia / 'casos.json').read_text())
    for nombre in ('memoria-equipo.docx', 'memoria-consulta.docx'):
        with ZipFile(evidencia / nombre) as documento:
            assert documento.testzip() is None, nombre
            contenido = ElementTree.fromstring(documento.read('word/document.xml'))
            texto = ''.join(contenido.itertext())
            assert texto.strip(), nombre
            narrativa = 'Se revisan las rutinas familiares y la asistencia escolar en reuniones semanales.'
            if nombre == 'memoria-equipo.docx':
                assert narrativa in texto, nombre
            else:
                assert narrativa not in texto, nombre
            for identificador in (casos['a'], casos['b'], casos['c']):
                assert identificador not in texto, nombre
            if nombre == 'memoria-consulta.docx':
                for numero in casos['numeros']:
                    assert numero not in texto, f'{nombre}: número de expediente'
            # Los documentos de Memoria no deben incluir identidades del juego.
            for identidad in ('Torres Montes', 'Daniel', 'Lucía'):
                assert identidad not in texto, f'{nombre}: identidad {identidad}'
    print('F4.09-WORD-OK: dos DOCX válidos; Consulta sin narrativa ni identificadores; sin nombres de menores.', flush=True)


def main():
    inicio = time.monotonic()
    temporal = Path(os.environ.get('TMPDIR', str(RAIZ / 'bin/temporales')))
    temporal.mkdir(parents=True, exist_ok=True)
    evidencia = Path(tempfile.mkdtemp(prefix='f409-', dir=temporal))
    entorno = dict(os.environ)
    if not (RAIZ / 'despliegue/demo/.local/puerto').exists() and not entorno.get('ETF_DEMO_PUERTO'):
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            entorno['ETF_DEMO_PUERTO'] = str(sock.getsockname()[1])
    os.environ.update({k: v for k, v in entorno.items() if k.startswith('ETF_DEMO_')})
    spec = importlib.util.spec_from_file_location('demo_f409', RAIZ / 'despliegue/demo/demo.py')
    demo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(demo)
    print(f'F4.09-EVIDENCIA: {evidencia}', flush=True)
    if (demo.LOCAL / 'entorno.env').exists():
        activos = demo.compose('ps', '--status', 'running', '-q', 'app', capture_output=True, text=True).stdout.strip()
        if activos:
            demo.compose('restart', '--no-deps', 'app', stdout=subprocess.DEVNULL)
    # El registro de arranque contiene claves de cuentas inventadas: queda fuera de Git.
    log = evidencia / 'demo.log'
    with log.open('w') as salida:
        resultado = subprocess.run(['make', 'demo'], cwd=RAIZ, env=entorno, timeout=600,
                                   stdout=salida, stderr=subprocess.STDOUT)
    if resultado.returncode and 'migracion.estado_previo' in log.read_text():
        with log.open('a') as salida:
            subprocess.run(['make', 'demo-limpiar'], cwd=RAIZ, env=entorno, check=True,
                           timeout=120, stdout=salida, stderr=subprocess.STDOUT)
            resultado = subprocess.run(['make', 'demo'], cwd=RAIZ, env=entorno, timeout=600,
                                       stdout=salida, stderr=subprocess.STDOUT)
    resultado.check_returncode()
    demo.compose('up', '--detach', '--force-recreate', '--no-deps', '--wait', 'app', stdout=subprocess.DEVNULL)
    recorrido = subprocess.run(['node', 'scripts/recorridos/memoria-integrada.mjs', demo.URL, str(evidencia)],
                               cwd=RAIZ, env=entorno, timeout=600)
    if all((evidencia / nombre).exists() for nombre in ('memoria-equipo.docx', 'memoria-consulta.docx')):
        comprobar_word(evidencia)
    if recorrido.returncode:
        print(f'F4.09-FALLO: {time.monotonic() - inicio:.1f} s; evidencia en {evidencia}', flush=True)
        raise SystemExit(recorrido.returncode)
    print(f'F4.09-OK: {time.monotonic() - inicio:.1f} s; evidencia en {evidencia}', flush=True)


if __name__ == '__main__':
    main()
