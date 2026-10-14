#!/usr/bin/env python3
"""Un comando para validar, grabar, poner voz y montar recorridos locales."""
import argparse
import json
import os
import re
import subprocess
from datetime import datetime
from pathlib import Path
from guion import leer
from herramientas import RAIZ, destino, ejecutar, ffmpeg
from poner_voz import montar, preparar
from hoja_control import generar
from alinear import alinear
from capitulos import insertar



def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--recorrido", action="append")
    ap.add_argument("--sintetico", action="store_true")
    ap.add_argument("--sin-voz", action="store_true")
    ap.add_argument("--validar", action="store_true")
    args = ap.parse_args()
    if args.sin_voz and not args.sintetico:
        ap.error("--sin-voz sólo se permite en la prueba sintética")
    carpeta = RAIZ / "docs/recorridos/guiones"
    if args.recorrido:
        if any(not re.fullmatch(r"[0-9]{2}-[a-z0-9-]+", nombre) for nombre in args.recorrido):
            ap.error("Identificador de recorrido no válido")
        if len(set(args.recorrido)) != len(args.recorrido):
            ap.error("No repitas un recorrido en la misma ejecución")
        ficheros = [carpeta / f"{nombre}.md" for nombre in args.recorrido]
    else:
        ficheros = sorted(carpeta.glob("[0-9][0-9]-*.md"))
    if not ficheros:
        raise ValueError("No hay guiones")
    guiones = [(f, leer(f)) for f in ficheros]
    if args.validar:
        print(f"GUIONES-OK {len(guiones)}: introducción, cierre, rótulos y siglas")
        return
    destino()
    herramienta = ffmpeg()
    reloj_activo = False
    try:
        if not args.sintetico:
            comprobacion = subprocess.run(["make", "--no-print-directory", "-n", "demo"], cwd=RAIZ, capture_output=True)
            if comprobacion.returncode == 0:
                if len(guiones) == 1 and guiones[0][1]['config'].get('automatizacion') == 'ciclo':
                    os.environ['ETF_VIDEO_DEMO'] = '1'
                    reloj_activo = True
                    ejecutar(['python3', '-B', RAIZ / 'scripts/recorridos/reloj_video.py', 'iniciar'], cwd=RAIZ, capture_output=True, timeout=3600)
                else:
                    ejecutar(["make", "--no-print-directory", "demo"], cwd=RAIZ, capture_output=True, timeout=3600)
                local = RAIZ / 'despliegue/demo/.local'
                puerto = (local / 'puerto').read_text().strip()
                claves = json.loads((local / 'usuarios.json').read_text())
                url_demo = f'http://127.0.0.1:{puerto}'
                if any(g['config'].get('automatizacion') == 'ciclo' for _, g in guiones) and os.environ.get('ETF_VIDEO_URL', url_demo) != url_demo:
                    raise ValueError('El ciclo exige la demo de este worktree')
                os.environ.setdefault('ETF_VIDEO_URL', url_demo)
                os.environ.setdefault('ETF_VIDEO_DEMO', '1')
                for _, g in guiones:
                    c = g['config']
                    for perfil in c.get('perfiles', ['coordinacion'] if c.get('automatizacion') == 'coordinacion' else ['profesional', 'administracion']):
                        cuenta = os.environ.setdefault(c[f'{perfil}_env'], c[f'{perfil}_demo'])
                        if c[f'clave_{perfil}_env'] not in os.environ:
                            if cuenta not in claves:
                                raise ValueError('Falta la contraseña inventada de la cuenta elegida')
                            os.environ[c[f'clave_{perfil}_env']] = claves[cuenta]
            elif any(g['config'].get('automatizacion') == 'ciclo' for _, g in guiones):
                raise ValueError('El ciclo exige make demo de este worktree; no admite arranque manual')
            elif not os.environ.get("ETF_VIDEO_URL"):
                raise ValueError("No existe make demo. Arranque manual: consulta docs/recorridos/LEEME.md")
        for fichero, guion in guiones:
            prefijo = "sintetico-" if args.sintetico else ""
            carpeta_salida = destino() / f"{prefijo}{fichero.stem}-{datetime.now():%Y%m%d-%H%M%S-%f}"
            carpeta_salida.mkdir()
            (carpeta_salida / "guion.md").write_text(fichero.read_text(), encoding="utf-8")
            (carpeta_salida / "guion.json").write_text(json.dumps(guion, ensure_ascii=False, indent=2), encoding="utf-8")
            if not args.sintetico and guion['config'].get('automatizacion') == 'ciclo' and len(guiones) != 1:
                reloj_activo = True
                ejecutar(['python3', '-B', RAIZ / 'scripts/recorridos/reloj_video.py', 'iniciar'], cwd=RAIZ, capture_output=True, timeout=3600)
            try:
                preparar(carpeta_salida, guion, herramienta, args.sin_voz)
                modo = "sintetico" if args.sintetico else "demo"
                ejecutar(["node", RAIZ / "scripts/recorridos/grabar.mjs", carpeta_salida / "guion.json", carpeta_salida, modo])
                escenas = alinear(carpeta_salida, herramienta)
                ejecutar(["node", RAIZ / "scripts/recorridos/cartelas.mjs", carpeta_salida])
                plan = montar(carpeta_salida, guion, escenas, herramienta, args.sin_voz)
                insertar(carpeta_salida, guion, plan, herramienta)
                generar(carpeta_salida, guion, plan, herramienta)
            finally:
                if reloj_activo:
                    ejecutar(['python3', '-B', RAIZ / 'scripts/recorridos/reloj_video.py', 'restaurar'], cwd=RAIZ, capture_output=True, timeout=3600)
                    reloj_activo = False
            print(f"VIDEO-OK {carpeta_salida / 'video.mp4'}; intro={plan['intro_s']:.1f}s; voz={'silencio de prueba' if args.sin_voz else 'AlvaroNeural'}; hoja-control.png")
    finally:
        if reloj_activo:
            ejecutar(['python3', '-B', RAIZ / 'scripts/recorridos/reloj_video.py', 'restaurar'], cwd=RAIZ, capture_output=True, timeout=3600)


if __name__ == "__main__":
    try:
        main()
    except (ValueError, RuntimeError, OSError, subprocess.SubprocessError) as error:
        # Los detalles de edge-tts y credenciales no se imprimen.
        raise SystemExit(f"VIDEOS-ERROR {error}") from None
