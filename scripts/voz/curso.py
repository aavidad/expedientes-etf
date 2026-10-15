"""Incorpora el lector estático a las exportaciones web y SCORM."""
import html
import json
from pathlib import Path
import re
import shutil
import zipfile
from comun import VOZ, sha_texto
from construir import entradas


def integrar(constructor, salida, audios):
    ruta = audios / 'curso/manifiesto.json'
    if not ruta.is_file():
        return
    datos = json.loads(ruta.read_text())
    secciones = entradas()['curso']
    actuales = {s['id']: s for s in secciones}
    datos = {id: a for id, a in datos.items() if id in actuales and a['sha'] == sha_texto(VOZ + actuales[id]['texto']) and re.fullmatch(r'[a-f0-9]{64}\.mp3', a['fichero']) and (ruta.parent / a['fichero']).is_file()}
    if not datos:
        return
    textos = json.loads((Path(__file__).resolve().parents[2] / 'idiomas/es/textos/voz.json').read_text())
    js = (Path(__file__).parent / 'lector.js').read_bytes()
    archivos = {'voz/lector.js': js}
    for audio in datos.values():
        archivos['voz/' + audio['fichero']] = (ruta.parent / audio['fichero']).read_bytes()

    def pagina(texto, prefijo):
        match = re.search(r'\bid="page-content-(etf-[a-f0-9]+)"', texto)
        if not match:
            return texto
        id = match[1]
        lista = [(s, datos[s['id']]) for s in secciones if s['id'].startswith(id+'-s') and s['id'] in datos]
        if not lista:
            return texto
        escapar = lambda valor: html.escape(valor, quote=True)
        rutas = [prefijo+'voz/'+audio['fichero'] for _, audio in lista]
        control = (f'<div class="etf-voz-pagina"><button type="button" data-voz-pagina="{escapar(json.dumps(rutas))}" '
                   f'data-voz-escuchar="{escapar(textos["voz.escuchar_pagina"])}" data-voz-pausar="{escapar(textos["voz.pausar"])}">'
                   f'{escapar(textos["voz.escuchar_pagina"])}</button><audio controls hidden preload="none" aria-label="{escapar(textos["voz.reproductor"])}"></audio></div>')
        texto = re.sub(r'(<div\b[^>]*\bid="page-content-'+id+r'"[^>]*>)', lambda m: m[1]+control, texto, count=1)
        for seccion, audio in lista:
            if not seccion['titulo']:
                continue
            etiqueta = textos['voz.escuchar_seccion_aria'].replace('{titulo}',seccion['titulo'])
            bloque = (f'<div class="etf-voz-seccion"><button type="button" data-voz-seccion="{escapar(prefijo+"voz/"+audio["fichero"])}" aria-label="{escapar(etiqueta)}">'
                      f'{escapar(textos["voz.escuchar_seccion"])}</button><audio controls hidden preload="none" aria-label="{escapar(textos["voz.reproductor"])}"></audio></div>')
            texto = re.sub(r'(<h2>'+re.escape(html.escape(seccion['titulo']))+r'</h2>)', lambda m: m[1]+bloque, texto, count=1)
        return texto.replace('</body>', '<script src="'+prefijo+'voz/lector.js"></script></body>')

    web = salida / 'web'
    for nombre, contenido in archivos.items():
        (web / nombre).parent.mkdir(parents=True,exist_ok=True)
        (web / nombre).write_bytes(contenido)
    for archivo in web.rglob('*.html'):
        archivo.write_text(pagina(archivo.read_text(), '../' if archivo.parent != web else ''))
    with zipfile.ZipFile(salida / 'scorm12.zip') as original:
        contenido = {p.filename: original.read(p) for p in original.infolist() if not p.is_dir()}
    for nombre in list(contenido):
        if nombre.endswith('.html'):
            contenido[nombre] = pagina(contenido[nombre].decode(), '../' if '/' in nombre else '').encode()
    contenido.update(archivos)
    manifiesto = contenido['imsmanifest.xml'].decode()
    ficheros = ''.join('<file href="'+nombre+'" />' for nombre in archivos)
    contenido['imsmanifest.xml'] = manifiesto.replace('</resource>', ficheros+'</resource>',1).encode()
    with zipfile.ZipFile(salida / 'scorm12.zip','w',zipfile.ZIP_DEFLATED) as paquete:
        for nombre, dato in contenido.items():
            paquete.writestr(nombre,dato)
