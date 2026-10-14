"""Escenas narradas: acciones por frase y vista previa de pantallas reales."""
import hashlib
import json
import math
import os
import subprocess
import sys
import wave
import re
from pathlib import Path
from herramientas import SALIDA, duracion, ejecutar

VOZ = "es-ES-AlvaroNeural"
RATE = "+4%"
COMMIT = "0b1c544d020b01338ae491d7f4547729afdc39d3"


def sintetizar(texto, cache):
    cache.mkdir(parents=True, exist_ok=True)
    clave = hashlib.sha256(f"{COMMIT}|{VOZ}|{RATE}|{texto}".encode()).hexdigest()
    fichero = cache / f"{clave}.mp3"
    if fichero.is_file() and fichero.stat().st_size > 1000:
        return fichero
    python = SALIDA / ".herramientas/voz/bin/python"
    env = {"PATH": os.defpath, "LANG": "C.UTF-8"}
    lib = os.environ.get("ETF_EDGE_LIB")
    if lib:
        # Reutilización de un kit local ya instalado, sólo del arreglo autorizado.
        metadatos = Path(lib) / "edge_tts-7.2.8.dist-info/direct_url.json"
        datos = json.loads(metadatos.read_text())
        if datos.get("vcs_info", {}).get("commit_id") != COMMIT:
            raise ValueError("ETF_EDGE_LIB no contiene el commit autorizado de edge-tts")
        env["PYTHONPATH"] = str(Path(lib).resolve())
        python = Path(sys.executable)
    elif not python.is_file():
        raise ValueError("Falta el entorno de voz. Ejecuta scripts/recorridos/preparar_voz.sh")
    temporal = fichero.with_suffix(".part.mp3")
    # Sólo el texto de la locución se entrega a edge-tts; no recibe capturas ni datos de sesión.
    try:
        ejecutar([python, "-m", "edge_tts", "--voice", VOZ, f"--rate={RATE}", "--text", texto,
                  "--write-media", temporal], env=env, capture_output=True, timeout=90)
        if temporal.stat().st_size < 1000:
            raise ValueError("Voz incompleta")
        temporal.replace(fichero)
    except subprocess.CalledProcessError as error:
        raise RuntimeError("edge-tts no pudo generar la voz; comprueba instalación y conexión") from error
    finally:
        temporal.unlink(missing_ok=True)
    return fichero


def preparar(dir, guion, ffmpeg, sin_voz=False):
    dir = Path(dir)
    for indice, bloque in enumerate([guion['intro'], *guion['pasos'], guion['cierre']]):
        frases = re.split(r'(?<=[.!?])\s+', bloque['Locución']) if 'id' in bloque else [bloque['Locución']]
        bloque['frases'] = []
        for n, texto in enumerate(frases):
            wav = dir / f'voz-{indice:02}-{n:02}.wav'
            if sin_voz:
                with wave.open(str(wav), 'wb') as w:
                    w.setnchannels(1); w.setsampwidth(2); w.setframerate(24000)
                    w.writeframes(bytes(int(len(texto.split()) / 2.7 * 24000) * 2))
            else:
                mp3 = sintetizar(texto, SALIDA / '.cache-voz')
                ejecutar([ffmpeg, '-v', 'error', '-y', '-i', mp3, '-ac', '1', '-ar', '24000', wav])
            bloque['frases'].append({'texto': texto, 'duracion_s': duracion(wav), 'archivo': wav.name})
    intro = math.ceil(max(12, guion['intro']['frases'][0]['duracion_s'] + 0.6) * 25) / 25
    cierre = math.ceil(max(4, guion['cierre']['frases'][0]['duracion_s'] + 0.6) * 25) / 25
    if intro > 18 or cierre > 6:
        raise ValueError(f'Acorta la voz: introducción={intro:.2f}s (12–18); cierre={cierre:.2f}s (4–6)')
    if guion['config'].get('automatizacion') == 'ciclo':
        total = intro + cierre + sum(math.ceil((f['duracion_s'] + 0.3) * 25) / 25 for p in guion['pasos'] for f in p['frases'])
        if not 480 <= total <= 720:
            raise ValueError(f'El ciclo exige 8–12 minutos antes de grabar: {total:.2f}s')
    (dir / 'guion.json').write_text(json.dumps(guion, ensure_ascii=False, indent=2), encoding='utf-8')


def planificar(escenas, pasos, intro):
    plan, cursor = [], intro
    for escena, paso in zip(escenas, pasos, strict=True):
        if escena['id'] != paso['id'] or len(escena['fases']) != len(paso['frases']):
            raise ValueError('Las acciones no coinciden con las frases preparadas')
        fases = []
        for fase, frase in zip(escena['fases'], paso['frases'], strict=True):
            if fase['fin_s'] <= fase['inicio_s']:
                raise ValueError('Una acción no tiene duración válida')
            largo = math.ceil((frase['duracion_s'] + 0.3) * 25) / 25
            fases.append({**fase, 'sale_ini': cursor, 'sale_fin': cursor + largo,
                          'voz_ini': cursor + 0.1, 'archivo': frase['archivo']})
            cursor += largo
        plan.append({'id': paso['id'], 'sale_ini': fases[0]['sale_ini'], 'sale_fin': cursor, 'fases': fases})
    return plan, cursor


def codificar(ffmpeg, entradas, filtro, salida, largo=None):
    orden = [ffmpeg, '-v', 'error', '-y', '-threads', '2', '-filter_complex_threads', '2', *entradas,
             '-filter_complex', filtro, '-map', '[v]', '-an', '-c:v', 'libx264', '-preset', 'fast', '-crf', '20']
    if largo is not None:
        orden += ['-t', f'{largo:.3f}']
    ejecutar([*orden, salida])


def vista_previa(dir, guion, intro, ffmpeg):
    # Tras 3,5 segundos de título, tres capturas reales se mueven y se funden.
    titulo = 3.5
    dur = (intro - titulo + 0.8) / 3
    indices = [0, len(guion['pasos']) // 2, len(guion['pasos']) - 1]
    entradas = ['-loop', '1', '-t', str(titulo), '-i', dir / 'intro.png']
    fc = f'[0:v]fps=25,scale=1440:900,setsar=1,format=yuv420p,trim=duration={titulo},setpts=PTS-STARTPTS[t];'
    for n, indice in enumerate(indices):
        entradas += ['-i', dir / f"{guion['pasos'][indice]['id']}.png"]
        frames = math.ceil(dur * 25)
        fc += (f"[{n+1}:v]scale=1440:900,zoompan=z='1.01+0.025*on/{frames}':"
               f"x='iw/2-iw/zoom/2+on/{frames}*5':y='ih/2-ih/zoom/2':d={frames}:s=1440x900:fps=25,"
               f'trim=duration={dur:.3f},setsar=1,format=yuv420p,setpts=PTS-STARTPTS[p{n}];')
    fc += f'[p0][p1]xfade=transition=fade:duration=0.4:offset={dur-0.4:.3f}[u];'
    fc += f'[u][p2]xfade=transition=fade:duration=0.4:offset={2*dur-0.8:.3f}[previa];'
    fc += '[t][previa]concat=n=2:v=1:a=0[v]'
    codificar(ffmpeg, entradas, fc, dir / 'introduccion.mp4', intro)


def retirar_pausas(datos, inicio, fin, objetivo, tamano=1440*900*3):
    # Sólo se retiran repeticiones de imagen: la escritura y el ratón se conservan.
    exceso = fin - inicio - objetivo
    if exceso <= 0.06:
        return [(inicio, fin)]
    frames = [datos[n:n+tamano] for n in range(0, len(datos)-tamano+1, tamano)]
    pausas = []
    primero = None
    for i in range(1, len(frames)):
        quieto = frames[i-1] == frames[i]
        if quieto and primero is None:
            primero = i-1
        if primero is not None and (not quieto or i == len(frames)-1):
            hasta = i if not quieto else i+1
            a, b = inicio + (primero+3)/25, min(fin, inicio + hasta/25)
            if b > a:
                pausas.append((a, b))
            primero = None
    cortes = []
    for a, b in reversed(pausas):
        cantidad = min(exceso, b-a)
        if cantidad > 0:
            cortes.append((b-cantidad, b)); exceso -= cantidad
    if exceso > 0.06:
        raise ValueError('La acción supera su frase sin tiempo muerto suficiente; amplía la locución')
    cursor, tramos = inicio, []
    for a, b in sorted(cortes):
        if a > cursor:
            tramos.append((cursor, a))
        cursor = b
    if cursor < fin:
        tramos.append((cursor, fin))
    return tramos


def montar(dir, guion, escenas, ffmpeg, sin_voz=False):
    dir = Path(dir)
    intro = math.ceil(max(12, guion['intro']['frases'][0]['duracion_s'] + 0.6) * 25) / 25
    cierre = math.ceil(max(4, guion['cierre']['frases'][0]['duracion_s'] + 0.6) * 25) / 25
    plan, inicio_cierre = planificar(escenas, guion['pasos'], intro)
    vista_previa(dir, guion, intro, ffmpeg)
    partes = [dir / 'introduccion.mp4']
    for n, paso in enumerate(plan):
        for k, fase in enumerate(paso['fases']):
            archivo = dir / f'escena-{n:02}-{k:02}.mp4'
            origen = fase['fin_s'] - fase['inicio_s']
            objetivo = fase['sale_fin'] - fase['sale_ini']
            datos = b''
            if origen - objetivo > 0.06:
                datos = ejecutar([ffmpeg, '-v', 'error', '-ss', str(fase['inicio_s']),
                                  '-i', dir / 'original.webm', '-t', str(origen),
                                  '-vf', 'fps=25,format=rgb24', '-f', 'rawvideo', '-'], capture_output=True).stdout
            tramos = retirar_pausas(datos, fase['inicio_s'], fase['fin_s'], objetivo)
            fase['tramos_conservados'] = tramos
            filtro = f'[0:v]split={len(tramos)}' + ''.join(f'[r{j}]' for j in range(len(tramos))) + ';'
            for j, (a, b) in enumerate(tramos):
                filtro += f'[r{j}]trim=start={a-fase["inicio_s"]:.5f}:end={b-fase["inicio_s"]:.5f},setpts=PTS-STARTPTS[q{j}];'
            filtro += ''.join(f'[q{j}]' for j in range(len(tramos)))
            # Si se amplía la voz al revisar, conserva el último fotograma hasta su fin.
            # La búsqueda por fase evita decodificar todo el recorrido para cada frase.
            prolongacion = max(0.12, objetivo - sum(b-a for a, b in tramos) + 0.12)
            fase['prolongacion_s'] = max(0, objetivo - sum(b-a for a, b in tramos))
            filtro += (f'concat=n={len(tramos)}:v=1:a=0,fps=25,scale=1440:900,setsar=1,format=yuv420p,'
                       f'tpad=stop_mode=clone:stop_duration={prolongacion:.5f},trim=duration={objetivo:.3f}[v]')
            codificar(ffmpeg, ['-ss', str(fase['inicio_s']), '-i', dir / 'original.webm'], filtro, archivo, objetivo)
            partes.append(archivo)
    codificar(ffmpeg, ['-loop', '1', '-i', dir / 'cierre.png'],
              '[0:v]fps=25,scale=1440:900,setsar=1,format=yuv420p[v]', dir / 'despedida.mp4', cierre)
    partes.append(dir / 'despedida.mp4')
    largo = inicio_cierre + cierre
    pista = bytearray(math.ceil(largo * 24000) * 2)
    locuciones = [(guion['intro']['frases'][0]['archivo'], 0.3),
                 *[(f['archivo'], f['voz_ini']) for p in plan for f in p['fases']],
                 (guion['cierre']['frases'][0]['archivo'], inicio_cierre + 0.3)]
    for archivo, instante in locuciones:
        with wave.open(str(dir / archivo), 'rb') as w:
            datos = w.readframes(w.getnframes())
        a = round(instante * 24000) * 2
        if a + len(datos) > len(pista):
            raise ValueError('La voz desborda su escena')
        pista[a:a+len(datos)] = datos
    with wave.open(str(dir / 'voz.wav'), 'wb') as w:
        w.setnchannels(1); w.setsampwidth(2); w.setframerate(24000); w.writeframes(pista)
    lista = dir / 'partes.txt'
    # Nombres internos fijados por la herramienta; no se interpolan rutas del guion.
    lista.write_text(''.join(f"file '{p.name}'\n" for p in partes), encoding='utf-8')
    ejecutar([ffmpeg, '-v', 'error', '-y', '-f', 'concat', '-safe', '1', '-i', lista,
              '-i', dir / 'voz.wav', '-map', '0:v', '-map', '1:a', '-c:v', 'copy',
              '-af', 'loudnorm=I=-16:TP=-1.5:LRA=11', '-c:a', 'aac', '-b:a', '160k',
              '-shortest', '-movflags', '+faststart', dir / 'video.mp4'])
    datos = {'intro_s': intro, 'titulo_s': 3.5, 'cierre_s': cierre, 'inicio_cierre': inicio_cierre,
             'sin_voz': sin_voz, 'plan': plan, 'congelados': []}
    (dir / 'plan.json').write_text(json.dumps(datos, ensure_ascii=False, indent=2), encoding='utf-8')
    return datos
