"""Ancla el reloj de las acciones al primer rótulo que aparece en el WebM."""
import json
from pathlib import Path
from herramientas import ejecutar

FILTRO = 'scale=1440:900,crop=1440:180:0:720,scale=144:18,format=gray'
TAMANO = 144 * 18
FPS = 25


def buscar_ancla(datos, referencia):
    if len(referencia) != TAMANO:
        raise ValueError('Imagen de anclaje incompleta')
    for indice in range(len(datos) // TAMANO):
        frame = datos[indice*TAMANO:(indice+1)*TAMANO]
        distancia = sum(abs(a-b) for a, b in zip(frame, referencia)) / TAMANO
        if distancia < 3:
            return indice / FPS
    raise ValueError('El vídeo no contiene el anclaje visual de sus acciones')


def alinear(dir, ffmpeg):
    dir = Path(dir)
    referencia = ejecutar([ffmpeg, '-v', 'error', '-i', dir / 'ancla.png', '-vf', FILTRO,
                           '-frames:v', '1', '-f', 'rawvideo', '-'], capture_output=True).stdout
    datos = ejecutar([ffmpeg, '-v', 'error', '-i', dir / 'original.webm', '-t', '15',
                      '-vf', f'fps={FPS},{FILTRO}', '-f', 'rawvideo', '-'], capture_output=True).stdout
    pts = buscar_ancla(datos, referencia)
    reloj = json.loads((dir / 'ancla.json').read_text())['reloj_s']
    ajuste = pts - reloj
    escenas = json.loads((dir / 'escenas.json').read_text())
    for escena in escenas:
        for tramo in [escena, *escena['fases']]:
            tramo['inicio_s'] += ajuste
            tramo['fin_s'] += ajuste
            if tramo['inicio_s'] < 0 or tramo['fin_s'] <= tramo['inicio_s']:
                raise ValueError('Anclaje temporal inválido')
    (dir / 'alineacion.json').write_text(json.dumps({'pts_s': pts, 'reloj_s': reloj, 'ajuste_s': ajuste}, indent=2))
    (dir / 'escenas-alineadas.json').write_text(json.dumps(escenas, ensure_ascii=False, indent=2))
    return escenas
