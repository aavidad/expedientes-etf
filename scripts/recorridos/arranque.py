"""Retira los fotogramas uniformes anteriores a la primera pantalla de Chrome."""
from pathlib import Path
from herramientas import ejecutar

TAMANO = 144 * 90
FPS = 25


def inicio_con_contenido(datos):
    for indice in range(len(datos) // TAMANO):
        frame = datos[indice * TAMANO:(indice + 1) * TAMANO]
        if max(frame) - min(frame) > 24:
            return indice / FPS
    raise ValueError("La grabación empieza sin una pantalla visible durante tres segundos")


def recortar(dir, ffmpeg):
    dir = Path(dir)
    original = dir / 'original.webm'
    datos = ejecutar([ffmpeg, '-v', 'error', '-i', original, '-t', '3',
                      '-vf', f'fps={FPS},scale=144:90,format=gray',
                      '-f', 'rawvideo', '-'], capture_output=True).stdout
    inicio = inicio_con_contenido(datos)
    if inicio == 0:
        return
    salida = dir / 'sin-arranque.webm'
    try:
        # Un margen de un fotograma evita conservar una transición muestreada.
        ejecutar([ffmpeg, '-v', 'error', '-y', '-ss', f'{inicio + 1 / FPS:.3f}',
                  '-i', original, '-an', '-c:v', 'libvpx', '-crf', '4', '-b:v', '4M', salida])
        salida.replace(original)
    finally:
        salida.unlink(missing_ok=True)
