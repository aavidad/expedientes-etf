"""Casa rótulos con sus capturas locales, sin OCR ni colores de un tema concreto."""
import json
import subprocess
from pathlib import Path
from herramientas import ejecutar

FILTRO = "scale=1440:900,crop=1440:180:0:720,scale=144:18"
TAMANO = 144 * 18
FPS = 10


def detectar(video, referencias, ffmpeg):
    modelos = [ejecutar([ffmpeg, "-v", "error", "-i", r["imagen"], "-vf", FILTRO,
                         "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "gray", "-"], capture_output=True).stdout
               for r in referencias]
    if any(len(m) != TAMANO for m in modelos):
        raise ValueError("Captura de rótulo no válida")
    # Submuestreo espacial suficiente para distinguir texto y caja a bajo coste.
    modelos = [m[::3] for m in modelos]
    tramos = [[] for _ in modelos]
    activo, inicio, indice = None, 0, 0
    proceso = subprocess.Popen([ffmpeg, "-v", "error", "-i", str(video), "-vf", f"fps={FPS},{FILTRO}",
                                "-f", "rawvideo", "-pix_fmt", "gray", "-"], stdout=subprocess.PIPE)
    try:
        while True:
            frame = proceso.stdout.read(TAMANO)
            if not frame:
                break
            if len(frame) != TAMANO:
                raise ValueError("Fotograma incompleto")
            muestra = frame[::3]
            distancias = [sum(abs(a - b) for a, b in zip(muestra, m)) / len(m) for m in modelos]
            mejor = min(range(len(modelos)), key=distancias.__getitem__)
            elegido = mejor if distancias[mejor] < 6 else None
            if elegido != activo:
                if activo is not None:
                    tramos[activo].append((inicio / FPS, indice / FPS))
                activo, inicio = elegido, indice
            indice += 1
        if activo is not None:
            tramos[activo].append((inicio / FPS, indice / FPS))
    except BaseException:
        proceso.terminate()
        raise
    finally:
        proceso.stdout.close()
        resultado = proceso.wait(timeout=30)
    if resultado:
        raise RuntimeError(f"ffmpeg detector: {resultado}")
    salida, anterior = [], 0
    for ref, candidatos in zip(referencias, tramos):
        validos = [(a, b) for a, b in candidatos if b - a >= 4.4 and a >= anterior]
        if not validos:
            raise ValueError(f"No se detectó el rótulo estable de cinco segundos: {ref['id']}")
        a, b = max(validos, key=lambda par: par[1] - par[0])
        salida.append({"id": ref["id"], "texto": ref["texto"], "inicio_s": a, "fin_s": b})
        anterior = b
    return salida


def guardar(dir, ffmpeg):
    referencias = json.loads((Path(dir) / "referencias.json").read_text())
    rotulos = detectar(Path(dir) / "original.webm", referencias, ffmpeg)
    (Path(dir) / "rotulos.json").write_text(json.dumps(rotulos, ensure_ascii=False, indent=2), encoding="utf-8")
    return rotulos
