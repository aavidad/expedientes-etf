#!/usr/bin/env python3
"""Construye manuales saneados y, sólo bajo petición, locuciones editoriales."""
from __future__ import annotations
import argparse
import hashlib
import html
import json
from pathlib import Path
import re
import sys
import yaml
from urllib.parse import unquote, urlsplit

RAIZ = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(Path(__file__).parent))
from comun import MD, generar, legible, limpiar, medir, secciones, sha_texto, sintetizar_edge, sintetizar_simulado

MANUALES = {'profesional_etf': 'profesionales', 'coordinacion': 'coordinacion', 'administracion': 'administracion'}


def pagina_curso(fuente):
    return 'etf-' + hashlib.sha256(str(fuente.relative_to(RAIZ / 'curso')).encode()).hexdigest()[:20]


def entradas():
    grupos = {'curso': [], 'ayuda': [], 'manuales': []}
    interfaz = yaml.safe_load((RAIZ / 'curso/interfaz.yaml').read_text())
    config = yaml.safe_load((RAIZ / 'curso/estructura.yaml').read_text())
    for modulo in config['modulos']:
        prefijo = 'etf-' + hashlib.sha256(modulo['carpeta'].encode()).hexdigest()[:20]
        grupos['curso'].extend(secciones(modulo['titulo'] + '\n\n' + interfaz['entrada_modulo'], prefijo))
        carpeta = RAIZ / 'curso/contenido' / modulo['carpeta']
        for fuente in sorted(carpeta.glob('*.md')):
            texto = fuente.read_text()
            cabecera = re.match(r'\A---\s*\n(.*?)\n---\s*\n', texto, re.S)
            datos = yaml.safe_load(cabecera[1]) if cabecera else {}
            cuerpo = texto[cabecera.end():] if cabecera else texto
            objetivos = datos.get('objetivos', [])
            if objetivos:
                cuerpo = '## ' + interfaz['objetivos'] + '\n\n' + '\n'.join('- ' + x for x in objetivos) + '\n\n' + cuerpo
            grupos['curso'].extend(secciones(cuerpo, pagina_curso(fuente)))
        for fuente in sorted(carpeta.glob('*-autoevaluacion.yaml')):
            datos = yaml.safe_load(fuente.read_text())
            partes = [datos['titulo'], interfaz['instrucciones']]
            for pregunta in datos['preguntas']:
                partes += [pregunta.get('enunciado', ''), *pregunta.get('opciones', [])]
                if pregunta.get('texto'):
                    partes.append(re.sub(r'\{[^}]+\}', '…', pregunta['texto']))
            grupos['curso'].extend(secciones('\n\n'.join(partes), pagina_curso(fuente)))
    for fuente in sorted((RAIZ / 'idiomas/es/textos').glob('*.json')):
        for clave, valor in json.loads(fuente.read_text()).items():
            if 'ayuda' in clave and isinstance(valor, str) and legible(limpiar(valor)):
                grupos['ayuda'].append({'id': clave, 'texto': limpiar(valor), 'original': valor})
    for nombre in MANUALES.values():
        fuente = RAIZ / 'docs/manuales' / (nombre + '.md')
        texto = fuente.read_text()
        if nombre != 'administracion':
            texto += '\n' + (RAIZ / 'docs/manuales/comun.md').read_text()
        grupos['manuales'].extend(secciones(texto, nombre))
    return grupos


def manuales(destino):
    destino.mkdir(parents=True, exist_ok=True)
    recursos = {}
    def slug(titulo):
        return re.sub(r'[^\w -]', '', titulo.lower()).replace(' ', '-')
    for nombre in MANUALES.values():
        fuente = RAIZ / 'docs/manuales' / (nombre + '.md')
        texto = fuente.read_text()
        if nombre != 'administracion':
            texto += '\n' + (RAIZ / 'docs/manuales/comun.md').read_text()
        titulo = next(t.content for t in MD.parse(texto) if t.type == 'inline')
        partes = secciones(texto, nombre)
        anclas = {}
        for seccion in partes:
            tokens = MD.parse(seccion['markdown'])
            for i, token in enumerate(tokens):
                if token.type == 'heading_open':
                    etiqueta = limpiar(tokens[i+1].content)
                    anclas.setdefault(slug(etiqueta), seccion['id'] if token.tag in {'h1', 'h2'} else seccion['id']+'-h'+str(i))

        def recurso(referencia):
            partes_url = urlsplit(referencia)
            if partes_url.scheme or partes_url.netloc:
                raise ValueError('voz.recurso_invalido')
            ruta = (fuente.parent / unquote(partes_url.path)).resolve()
            if not ruta.is_relative_to(RAIZ / 'docs') or not ruta.is_file() or ruta.suffix.lower() not in {'.png', '.jpg', '.jpeg', '.webp', '.pdf'}:
                raise ValueError('voz.recurso_invalido')
            identificador = sha_texto(str(ruta.relative_to(RAIZ / 'docs'))) + ruta.suffix.lower()
            entrada = recursos.setdefault(identificador, {'ruta': str(ruta.relative_to(RAIZ / 'docs')), 'manuales': []})
            if nombre not in entrada['manuales']:
                entrada['manuales'].append(nombre)
            return '/manuales/recursos/' + identificador

        datos = {'idioma': 'es', 'titulo': titulo, 'secciones': []}
        for seccion in partes:
            tokens = MD.parse(seccion['markdown'])
            quitar_titulo = bool(tokens and tokens[0].type == 'heading_open' and tokens[0].tag in {'h1', 'h2'})
            for i, token in enumerate(tokens):
                if token.type == 'heading_open':
                    etiqueta = limpiar(tokens[i+1].content)
                    destino_ancla = anclas.get(slug(etiqueta))
                    if token.tag not in {'h1','h2'}:
                        token.attrSet('id',seccion['id']+'-h'+str(i))
                if token.type in {'heading_open','heading_close'}:
                    token.tag = 'h' + str(min(6,int(token.tag[1:])+1))
                enlace_omitido = False
                for hijo in token.children or []:
                    if hijo.type == 'image':
                        src = hijo.attrGet('src') or ''
                        if Path(urlsplit(src).path).suffix.lower() == '.pdf':
                            raise ValueError('voz.imagen_invalida')
                        hijo.attrSet('src', recurso(src))
                        hijo.attrSet('loading','lazy')
                    elif hijo.type == 'link_open':
                        href = hijo.attrGet('href') or ''
                        url = urlsplit(href)
                        destino_enlace = None
                        if url.scheme in {'https','http'} and url.netloc and not url.username and not url.password:
                            destino_enlace = href
                            hijo.attrSet('rel','noreferrer')
                        elif not url.scheme and not url.netloc:
                            if url.path in {'',nombre+'.md','comun.md'} and url.fragment:
                                ancla = anclas.get(unquote(url.fragment))
                                if ancla:
                                    destino_enlace = '#' + ancla
                            elif Path(url.path).suffix.lower() in {'.png','.jpg','.jpeg','.webp','.pdf'}:
                                destino_enlace = recurso(href)
                            elif url.path in {x+'.md' for x in MANUALES.values()} and url.path != nombre+'.md':
                                destino_enlace = recurso(url.path[:-3]+'.pdf')
                            elif url.path == nombre+'.md':
                                destino_enlace = '#manual-'+nombre
                            elif url.path == 'usuario.md':
                                destino_enlace = '/ayuda'
                        if destino_enlace:
                            hijo.attrSet('href',destino_enlace)
                        else:
                            # Un enlace sin destino disponible conserva su texto, sin una acción engañosa.
                            hijo.type = 'html_inline'; hijo.content = ''; enlace_omitido = True
                    elif hijo.type == 'link_close' and enlace_omitido:
                        hijo.type = 'html_inline'; hijo.content = ''; enlace_omitido = False
            # El título del manual ya aparece en el encabezado y en el índice.
            if quitar_titulo:
                tokens = tokens[3:]
            contenido = MD.renderer.render(tokens,MD.options,{})
            datos['secciones'].append({'id':seccion['id'], 'titulo':seccion['titulo'] or titulo, 'html':contenido, 'sha':sha_texto('es-ES-AlvaroNeural'+seccion['texto'])})
        (destino/(nombre+'.json')).write_text(json.dumps(datos,ensure_ascii=False,indent=2)+'\n')
    (destino/'recursos.json').write_text(json.dumps(recursos,ensure_ascii=False,indent=2)+'\n')

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--audios', action='store_true')
    parser.add_argument('--simulado', action='store_true')
    parser.add_argument('--destino', type=Path, default=RAIZ / 'bin/voz')
    args = parser.parse_args()
    if args.simulado and args.destino.resolve() == (RAIZ / 'bin/voz').resolve():
        parser.error('La prueba simulada necesita un destino distinto de bin/voz.')
    manuales(RAIZ / 'docs/manuales/web')
    if args.audios:
        grupos = entradas()
        for grupo, lista in grupos.items():
            generar(lista, args.destino / grupo, sintetizar_simulado if args.simulado else sintetizar_edge, medir)
            print(f'VOZ {grupo}: {len(lista)} secciones', flush=True)
    else:
        print('Manuales web construidos sin llamadas a la red.')


if __name__ == '__main__':
    main()
