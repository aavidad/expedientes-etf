"""Herramientas locales y destinos fuera de Git."""
import os
import shutil
import subprocess
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
SALIDA = Path(os.environ.get("ETF_VIDEOS_SALIDA", str(Path.home() / "Videos" / "Expedientes-ETF"))).expanduser().resolve()


def destino():
    if SALIDA == RAIZ or RAIZ in SALIDA.parents:
        raise ValueError("ETF_VIDEOS_SALIDA debe estar fuera del repositorio")
    SALIDA.mkdir(parents=True, exist_ok=True)
    return SALIDA


def ejecutar(args, **opciones):
    return subprocess.run([str(a) for a in args], check=True, **opciones)


def ffmpeg():
    candidatos = ([os.environ["ETF_VIDEO_FFMPEG"]] if os.environ.get("ETF_VIDEO_FFMPEG") else
                  sorted((Path.home() / ".cache/ms-playwright").glob("ffmpeg-*/ffmpeg-linux"), reverse=True)
                  + [shutil.which("ffmpeg")])
    for candidato in candidatos:
        if candidato and Path(candidato).is_file():
            encoders = ejecutar([candidato, "-encoders"], capture_output=True, text=True).stdout
            if "libx264" in encoders and "aac" in encoders:
                return str(candidato)
    raise RuntimeError("Hace falta ffmpeg con libx264 y AAC (el de Playwright puede ser sólo VP8)")


def duracion(fichero):
    r = ejecutar(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", fichero],
                 capture_output=True, text=True)
    return float(r.stdout)
