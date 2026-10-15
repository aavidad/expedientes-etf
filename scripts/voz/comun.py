"""Secciones editoriales, limpieza y caché del lector de voz."""
from __future__ import annotations
import hashlib
import html
import json
from pathlib import Path
import re
import subprocess
from markdown_it import MarkdownIt

VOZ = 'es-ES-AlvaroNeural'
MD = MarkdownIt('commonmark', {'html': False}).enable('table')


def sha_texto(texto):
    return hashlib.sha256(texto.encode('utf-8')).hexdigest()


def limpiar(texto):
    """Extrae sólo texto: imágenes, HTML crudo y destinos de enlaces quedan fuera."""
    partes = []
    for token in MD.parse(texto):
        if token.type in {'fence', 'code_block'}:
            continue
        for hijo in token.children or []:
            if hijo.type in {'text', 'code_inline'}:
                partes.append(hijo.content)
            elif hijo.type in {'softbreak', 'hardbreak'}:
                partes.append(' ')
    plano = html.unescape(' '.join(partes))
    plano = re.sub(r'<[^>]*>', ' ', plano)
    plano = re.sub(r'(?:https?://|www\.)[^\s<>]+', '', plano)
    return re.sub(r'\s+', ' ', plano).strip()


def legible(texto):
    """Hay algo que leer: una letra o una cifra. Un signo suelto no tiene audio."""
    return bool(re.search(r'\w', texto))


def secciones(texto, prefijo):
    texto = re.sub(r'\A---\s*\n.*?\n---\s*\n', '', texto, flags=re.S)
    tokens = MD.parse(texto)
    cortes = [t.map[0] for t in tokens if t.type == 'heading_open' and t.tag == 'h2']
    lineas = texto.splitlines(keepends=True)
    limites = sorted(set([0, *cortes, len(lineas)]))
    resultado = []
    for inicio, fin in zip(limites, limites[1:]):
        cuerpo = ''.join(lineas[inicio:fin])
        plano = limpiar(cuerpo)
        if legible(plano):
            titulo = re.search(r'^##\s+(.+)$', cuerpo, re.M)
            resultado.append({'id': f'{prefijo}-s{len(resultado)}', 'titulo': limpiar(titulo[1]) if titulo else '', 'texto': plano, 'markdown': cuerpo})
    return resultado


def generar(secciones_entrada, destino, sintetizar, duracion):
    destino = Path(destino)
    destino.mkdir(parents=True, exist_ok=True)
    manifiesto = {}
    for seccion in secciones_entrada:
        identificador, texto = seccion['id'], seccion['texto']
        if not re.fullmatch(r'[\w.-]+', identificador) or not isinstance(texto, str) or not texto.strip():
            raise ValueError('voz.seccion_invalida')
        if identificador in manifiesto:
            raise ValueError('voz.id_duplicado')
        sha = sha_texto(VOZ + texto)
        nombre = sha + '.mp3'
        archivo = destino / nombre
        metadatos = destino / (sha + '.json')
        if not archivo.is_file() or not archivo.stat().st_size or not metadatos.is_file():
            temporal = destino / (sha + '.tmp.mp3')
            try:
                sintetizar(texto, temporal)
                segundos = duracion(temporal)
                if segundos <= 0 or not temporal.stat().st_size:
                    raise ValueError('voz.audio_invalido')
                temporal.replace(archivo)
                metadatos.write_text(json.dumps({'duracion': segundos}), encoding='utf-8')
            finally:
                temporal.unlink(missing_ok=True)
        segundos = json.loads(metadatos.read_text())['duracion']
        manifiesto[identificador] = {'fichero': nombre, 'duracion': segundos, 'sha': sha}
        if 'original' in seccion:
            manifiesto[identificador]['original_sha'] = sha_texto(seccion['original'])
    temporal = destino / 'manifiesto.tmp.json'
    temporal.write_text(json.dumps(manifiesto, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    temporal.replace(destino / 'manifiesto.json')
    return manifiesto


def sintetizar_edge(texto, destino):
    # El texto es editorial y llega por fichero; nunca por parámetros del proceso.
    edge = Path.home() / '.local/bin/edge-tts'
    version = subprocess.run([str(edge), '--version'], check=True, capture_output=True, text=True).stdout.strip()
    if version != 'edge-tts 7.3.1':
        raise ValueError('voz.version_invalida')
    fuente = destino.with_suffix('.txt')
    bruto = destino.with_suffix('.edge.mp3')
    try:
        fuente.write_text(texto, encoding='utf-8')
        try:
            subprocess.run([str(edge), '--voice', VOZ, '--file', str(fuente), '--write-media', str(bruto)], check=True, capture_output=True, timeout=180)
        except subprocess.CalledProcessError as error:
            estado = '403' if b'403' in error.stderr else 'no_disponible'
            raise RuntimeError('voz.microsoft_' + estado) from None
        subprocess.run(['ffmpeg', '-nostdin', '-v', 'error', '-y', '-i', str(bruto), '-ar', '24000', '-ac', '1', '-b:a', '48k', str(destino)], check=True, capture_output=True, timeout=120)
    finally:
        fuente.unlink(missing_ok=True)
        bruto.unlink(missing_ok=True)


def medir(archivo):
    return float(subprocess.run(['ffprobe', '-v', 'error', '-show_entries', 'format=duration', '-of', 'default=nw=1:nk=1', str(archivo)], check=True, capture_output=True, text=True).stdout)


def sintetizar_simulado(texto, destino):
    subprocess.run(['ffmpeg', '-nostdin', '-v', 'error', '-y', '-f', 'lavfi', '-i', 'anullsrc=r=24000:cl=mono', '-t', '0.3', '-b:a', '48k', str(destino)], check=True, capture_output=True)
