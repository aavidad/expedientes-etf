"""Añade capítulos del guion al MP4 sin recodificar imagen ni voz."""
from pathlib import Path
from herramientas import ejecutar, duracion


def insertar(carpeta, guion, plan, ffmpeg):
    capitulos = guion['config'].get('capitulos', [])
    if not capitulos:
        return
    carpeta = Path(carpeta)
    comienzos = {paso['id']: paso['sale_ini'] for paso in plan['plan']}
    marcas = []
    for capitulo in capitulos:
        if capitulo.get('paso') not in comienzos or not isinstance(capitulo.get('titulo'), str) or not capitulo['titulo'].strip():
            raise ValueError('Capítulo sin paso o título válido')
        marcas.append((round(comienzos[capitulo['paso']] * 1000), capitulo['titulo']))
    if any(a[0] >= b[0] for a, b in zip(marcas, marcas[1:])):
        raise ValueError('Los capítulos deben seguir el orden del recorrido')
    video = carpeta / 'video.mp4'
    final = round(duracion(video) * 1000)
    lineas = [';FFMETADATA1']
    for i, (inicio, titulo) in enumerate(marcas):
        fin = marcas[i + 1][0] if i + 1 < len(marcas) else final
        if fin <= inicio:
            raise ValueError('Capítulo fuera de la duración del vídeo')
        for caracter in ['\\', '=', ';', '#', '\n']:
            titulo = titulo.replace(caracter, '\\' + caracter)
        lineas.extend(['[CHAPTER]', 'TIMEBASE=1/1000', f'START={inicio}', f'END={fin}', f'title={titulo}'])
    metadatos = carpeta / 'capitulos.txt'
    metadatos.write_text('\n'.join(lineas) + '\n', encoding='utf-8')
    salida = carpeta / 'video-capitulos.mp4'
    ejecutar([ffmpeg, '-v', 'error', '-y', '-i', video, '-f', 'ffmetadata', '-i', metadatos,
              '-map', '0:v', '-map', '0:a', '-map_metadata', '1', '-map_chapters', '1',
              '-c', 'copy', '-movflags', '+faststart', salida])
    salida.replace(video)
