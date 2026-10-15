#!/usr/bin/env python3
"""Construye el curso desde Markdown y YAML usando la CLI de eXeLearning 4.0.5."""
from __future__ import annotations

import argparse
import hashlib
import html
import json
import os
from pathlib import Path
import re
import shutil
import sys
import subprocess
from urllib.parse import unquote, urlsplit
import xml.etree.ElementTree as ET
import zipfile

import yaml
from markdown_it import MarkdownIt

RAIZ = Path(__file__).resolve().parents[1]
CURSO = RAIZ / 'curso'
SALIDA = RAIZ / 'bin/curso'
MD = MarkdownIt('commonmark', {'html': True}).enable('table')
MAX_RECURSO = 30 * 1024 * 1024


def cargar_yaml(ruta):
    datos = yaml.safe_load(ruta.read_text(encoding='utf-8'))
    if not isinstance(datos, dict):
        raise ValueError(f'{ruta}: se esperaba un objeto YAML')
    return datos


def identificador(texto):
    return 'etf-' + hashlib.sha256(texto.encode()).hexdigest()[:20]


def elemento(padre, nombre, valor=None):
    nodo = ET.SubElement(padre, nombre)
    if valor is not None:
        nodo.text = str(valor)
    return nodo


def propiedades(padre, contenedor, entrada, valores):
    grupo = elemento(padre, contenedor)
    for clave, valor in valores.items():
        nodo = elemento(grupo, entrada)
        elemento(nodo, 'key', clave)
        elemento(nodo, 'value', valor)


class Constructor:
    def __init__(self, config, videos):
        self.config = config
        self.interfaz = cargar_yaml(CURSO / 'interfaz.yaml')
        self.videos = videos
        self.recursos = {}
        self.videos_usados = set()
        self.paginas = []
        self.pendientes = []
        self.preguntas = 0
        self.enlaces = []

    def recurso(self, referencia, fuente):
        ruta = (fuente.parent / unquote(referencia)).resolve()
        if not ruta.is_relative_to(RAIZ) or ruta.suffix.lower() not in {'.png', '.jpg', '.jpeg', '.svg', '.webp', '.gif', '.pdf'}:
            raise ValueError(f'{fuente}: recurso fuera del repositorio o formato no permitido: {referencia}')
        if not ruta.is_file() or ruta.stat().st_size > MAX_RECURSO:
            raise ValueError(f'{fuente}: recurso ausente o demasiado grande: {referencia}')
        nombre = hashlib.sha256(ruta.read_bytes()).hexdigest()[:24] + ruta.suffix.lower()
        self.recursos['resources/' + nombre] = ruta
        return 'resources/' + nombre

    def contenido(self, fuente):
        texto = fuente.read_text(encoding='utf-8')
        cabecera = re.match(r'\A---\s*\n(.*?)\n---\s*\n', texto, re.S)
        if not cabecera:
            raise ValueError(f'{fuente}: falta la cabecera YAML')
        datos = yaml.safe_load(cabecera[1])
        if not isinstance(datos, dict) or not isinstance(datos.get('titulo'), str):
            raise ValueError(f'{fuente}: falta titulo')
        cuerpo = texto[cabecera.end():]
        # Las fuentes son editoriales. Se permiten details y tablas, nunca código activo.
        if re.search(r'<\s*(script|iframe|object|embed|style)\b|\son\w+\s*=|javascript\s*:', cuerpo, re.I):
            raise ValueError(f'{fuente}: HTML activo no permitido')
        for marca in re.findall(r'\[Validar con el Programa:[^\]]+\]', cuerpo):
            self.pendientes.append({'fuente': str(fuente.relative_to(RAIZ)), 'marca': marca})
        tokens = MD.parse(cuerpo)
        for token in tokens:
            if token.type in {'html_block', 'html_inline'}:
                token.content = self.html_reimportado(token.content, fuente)
            if token.type == 'heading_open' and token.tag == 'h1':
                raise ValueError(f'{fuente}: utiliza títulos ## o ###; el título de página lo aporta eXeLearning')
            for hijo in token.children or []:
                if hijo.type == 'image':
                    alt = hijo.content.strip()
                    if not alt:
                        raise ValueError(f'{fuente}: imagen sin texto alternativo')
                    src = hijo.attrGet('src') or ''
                    if src.startswith('videos/'):
                        if not re.fullmatch(r'videos/[\w.-]+\.mp4', src):
                            raise ValueError(f'{fuente}: ruta de vídeo no válida')
                        self.videos_usados.add(src[7:])
                        hijo.type = 'html_inline'
                        hijo.content = (f'<figure><video controls preload="metadata" aria-label="{html.escape(alt, quote=True)}" '
                                        f'src="{src}" width="100%"></video><figcaption>{html.escape(alt.removeprefix("vídeo: "))}</figcaption></figure>')
                    else:
                        hijo.attrSet('src', self.recurso(src, fuente))
                if hijo.type == 'link_open':
                    href = hijo.attrGet('href') or ''
                    esquema = urlsplit(href).scheme
                    if esquema not in {'', 'https', 'http', 'mailto'}:
                        raise ValueError(f'{fuente}: esquema de enlace no permitido')
                    if not esquema and not href.startswith('#'):
                        self.enlaces.append((fuente, href))
        contenido = MD.renderer.render(tokens, MD.options, {})
        objetivos = datos.get('objetivos', [])
        if not isinstance(objetivos, list) or any(not isinstance(x, str) for x in objetivos):
            raise ValueError(f'{fuente}: objetivos debe ser una lista de frases')
        if objetivos:
            contenido = '<h2>' + html.escape(self.interfaz['objetivos']) + '</h2><ul>' + ''.join('<li>' + html.escape(x) + '</li>' for x in objetivos) + '</ul>' + contenido
        return datos['titulo'], contenido

    def html_reimportado(self, texto, fuente):
        def imagen(m):
            etiqueta = m[0]
            alt = re.search(r'\balt=["\']([^"\']+)["\']', etiqueta)
            src = re.search(r'\bsrc=(["\'])(.*?)\1', etiqueta)
            if not alt or not src:
                raise ValueError(f'{fuente}: imagen HTML sin alt o src')
            return etiqueta.replace(src[0], 'src="' + self.recurso(html.unescape(src[2]), fuente) + '"')
        texto = re.sub(r'<img\b[^>]*>', imagen, texto, flags=re.I)
        for src in re.findall(r'<video\b[^>]*\bsrc=["\']([^"\']+)', texto, re.I):
            if not re.fullmatch(r'videos/\d{2}-[\w.-]+\.mp4', src):
                raise ValueError(f'{fuente}: ruta de vídeo HTML no válida')
            self.videos_usados.add(src[7:])
        for href in re.findall(r'<a\b[^>]*\bhref=["\']([^"\']+)', texto, re.I):
            href = html.unescape(href)
            if urlsplit(href).scheme not in {'', 'https', 'http', 'mailto'}:
                raise ValueError(f'{fuente}: esquema de enlace HTML no permitido')
            if not urlsplit(href).scheme and not href.startswith('#'):
                self.enlaces.append((fuente, href))
        return texto

    def cuestionario(self, fuente):
        datos = cargar_yaml(fuente)
        if not isinstance(datos.get('titulo'), str) or not isinstance(datos.get('preguntas'), list) or not datos['preguntas']:
            raise ValueError(f'{fuente}: faltan título o preguntas')
        preguntas = []
        for n, pregunta in enumerate(datos['preguntas']):
            if not isinstance(pregunta, dict):
                raise ValueError(f'{fuente}: pregunta {n + 1} no válida')
            tipo = pregunta.get('tipo')
            enunciado = pregunta.get('enunciado')
            explicacion = pregunta.get('explicacion')
            if not isinstance(enunciado, str) or not isinstance(explicacion, str):
                raise ValueError(f'{fuente}: pregunta {n + 1} sin enunciado o explicación')
            if re.search(r'<\s*(script|iframe|object|embed|style)\b|\son\w+\s*=|javascript\s*:', enunciado + explicacion, re.I):
                raise ValueError(f'{fuente}: HTML activo no permitido en pregunta {n + 1}')
            q = {'id': identificador(f'{fuente.name}-{n}'), 'activityType': {'test': 'selection', 'vf': 'true-false', 'huecos': 'fill'}.get(tipo),
                 'baseText': MD.render(enunciado), 'feedbackRight': explicacion, 'feedbackWrong': explicacion,
                 'suggestion': '', 'wrongAnswersValue': '', 'selectionType': 'single', 'answers': [], 'answer': '1',
                 'capitalization': False, 'strict': False, 'order': n, 'customScore': 1, 'time': 0}
            correcta = pregunta.get('correcta')
            if tipo == 'test':
                opciones = pregunta.get('opciones')
                if not isinstance(opciones, list) or len(opciones) < 2 or any(not isinstance(o, str) for o in opciones) or type(correcta) is not int or not 0 <= correcta < len(opciones):
                    raise ValueError(f'{fuente}: opciones o índice incorrectos en pregunta {n + 1}')
                q['answers'] = [[i == correcta, o] for i, o in enumerate(opciones)]
            elif tipo == 'vf':
                if type(correcta) is not bool:
                    raise ValueError(f'{fuente}: correcta debe ser true o false')
                q['answer'] = '1' if correcta else '0'
            elif tipo == 'huecos':
                texto = pregunta.get('texto', '')
                if not isinstance(texto, str) or not re.search(r'\{[^{}]+\}', texto):
                    raise ValueError(f'{fuente}: faltan respuestas entre llaves')
                q['baseText'] = MD.render(enunciado) + '<p>' + re.sub(r'\{([^{}]+)\}', r'<u>\1</u>', html.escape(texto)) + '</p>'
            else:
                raise ValueError(f'{fuente}: tipo de pregunta desconocido: {tipo}')
            preguntas.append(q)
        self.preguntas += len(preguntas)
        # Los mensajes se mantienen como datos de idioma, fuera del constructor.
        msgs = cargar_yaml(CURSO / 'interfaz.yaml')
        return datos['titulo'], {'questionsData': preguntas, 'msgs': msgs, 'eXeFormInstructions': '<p>' + msgs['instrucciones'] + '</p>',
                                  'eXeIdeviceTextAfter': '', 'isScorm': 1, 'textButtonScorm': msgs['msgCheck'], 'repeatActivity': True,
                                  'scorm': {'buttonTextSave': msgs['msgCheck']},
                                  'weighted': 1, 'evaluation': False, 'evaluationID': '', 'questionsRandom': False,
                                  'percentageQuestions': 100, 'time': 0, 'passRate': 5, 'addBtnAnswers': True, 'showSlider': False}

    def leer(self):
        for orden, modulo in enumerate(self.config['modulos']):
            carpeta = CURSO / 'contenido' / modulo['carpeta']
            if not carpeta.resolve().is_relative_to(CURSO / 'contenido'):
                raise ValueError('Carpeta de módulo fuera de contenido')
            padre = identificador(modulo['carpeta'])
            self.paginas.append({'id': padre, 'padre': '', 'orden': orden, 'titulo': modulo['titulo'],
                                 'html': '<p>' + html.escape(self.interfaz['entrada_modulo']) + '</p>', 'tipo': 'text'})
            fuentes = sorted([*carpeta.glob('*.md'), *carpeta.glob('*-autoevaluacion.yaml')], key=lambda p: (p.name[:2], p.suffix == '.yaml', p.name))
            if not fuentes:
                self.paginas[-1]['html'] = '<p>' + html.escape(self.interfaz['modulo_pendiente'].format(ruta=str(carpeta.relative_to(RAIZ)))) + '</p>'
            for n, fuente in enumerate(fuentes):
                titulo, contenido = self.contenido(fuente) if fuente.suffix == '.md' else self.cuestionario(fuente)
                pagina = {'id': identificador(str(fuente.relative_to(CURSO))), 'padre': padre, 'orden': n,
                          'titulo': titulo, 'fuente': str(fuente.relative_to(RAIZ)), 'tipo': 'text' if fuente.suffix == '.md' else 'form'}
                pagina['html' if pagina['tipo'] == 'text' else 'json'] = contenido
                self.paginas.append(pagina)
        por_fuente = {p['fuente']: p['id'] for p in self.paginas if 'fuente' in p}
        for fuente, href in self.enlaces:
            partes = urlsplit(href)
            destino = (fuente.parent / unquote(partes.path)).resolve()
            relativo = str(destino.relative_to(RAIZ)) if destino.is_relative_to(RAIZ) else ''
            if relativo in por_fuente:
                viejo = html.escape(href, quote=True)
                nuevo = 'exe-node:' + por_fuente[relativo] + ('#' + partes.fragment if partes.fragment else '')
                for p in self.paginas:
                    if p.get('fuente') == str(fuente.relative_to(RAIZ)):
                        p['html'] = p['html'].replace(f'href="{viejo}"', f'href="{nuevo}"')
            elif partes.path.startswith('videos/'):
                self.videos_usados.add(partes.path[7:])
            else:
                raise ValueError(f'{fuente}: enlace local no incorporado al curso: {href}')

    def elpx(self, ruta):
        raiz = ET.Element('ode', {'version': '4.0', 'xmlns': 'http://www.intef.es/xsd/ode'})
        propiedades(raiz, 'userPreferences', 'userPreference', {'theme': 'base'})
        c = self.config
        pie = self.interfaz['pie'].format(**{k: html.escape(str(v), quote=True) for k, v in c.items()})
        propiedades(raiz, 'odeProperties', 'odeProperty', {'pp_title': c['titulo'], 'pp_author': c['autor'], 'pp_lang': 'es',
                   'pp_license': c['licencia_exe'], 'pp_addPagination': 'true', 'pp_addExeLink': 'false', 'exportSource': 'false', 'footer': pie,
                   'pp_extraHeadContent': '<style>' + (CURSO / 'ajustes.css').read_text(encoding='utf-8') + '</style>'})
        navegacion = elemento(raiz, 'odeNavStructures')
        for p in self.paginas:
            nav = elemento(navegacion, 'odeNavStructure')
            for nombre, valor in [('odePageId', p['id']), ('odeParentPageId', p['padre']), ('pageName', p['titulo']), ('odeNavStructureOrder', p['orden'])]:
                elemento(nav, nombre, valor)
            grupos = elemento(nav, 'odePagStructures')
            bloque = elemento(grupos, 'odePagStructure')
            bid, iid = p['id'] + '-b', p['id'] + '-i'
            for nombre, valor in [('odePageId', p['id']), ('odeBlockId', bid), ('blockName', p['titulo']), ('odePagStructureOrder', 0)]:
                elemento(bloque, nombre, valor)
            componentes = elemento(bloque, 'odeComponents')
            componente = elemento(componentes, 'odeComponent')
            for nombre, valor in [('odePageId', p['id']), ('odeBlockId', bid), ('odeIdeviceId', iid), ('odeIdeviceTypeName', p['tipo']),
                                  ('htmlView', p.get('html', '<div class="form-IDevice"></div>')), ('odeComponentsOrder', 0)]:
                elemento(componente, nombre, valor)
            if 'json' in p:
                elemento(componente, 'jsonProperties', json.dumps({**p['json'], 'id': iid, 'ideviceId': iid}, ensure_ascii=False))
        with zipfile.ZipFile(ruta, 'w', zipfile.ZIP_DEFLATED) as paquete:
            xml = ET.tostring(raiz, encoding='unicode')
            # CDATA evita una segunda decodificación de entidades al importar HTML.
            xml = re.sub(r'<(htmlView|jsonProperties)>(.*?)</\1>',
                         lambda m: '<' + m[1] + '><![CDATA[' + html.unescape(m[2]).replace(']]>', ']]]]><![CDATA[>') + ']]></' + m[1] + '>',
                         xml, flags=re.S)
            paquete.writestr('content.xml', '<?xml version="1.0" encoding="UTF-8"?>\n' + xml)
            for nombre, fuente in sorted(self.recursos.items()):
                paquete.write(fuente, nombre)


def ejecutar(*args):
    resultado = subprocess.run(args, capture_output=True, text=True, timeout=300)
    if resultado.returncode:
        raise RuntimeError(f'Falló {args[0]}: {resultado.stdout[-3000:]} {resultado.stderr[-3000:]}')
    return resultado.stdout


def extraer_seguro(archivo, destino):
    with zipfile.ZipFile(archivo) as paquete:
        if len(paquete.infolist()) > 10000 or sum(p.file_size for p in paquete.infolist()) > 256 * 1024 * 1024:
            raise ValueError('Exportación demasiado grande')
        for entrada in paquete.infolist():
            ruta = (destino / entrada.filename).resolve()
            if not ruta.is_relative_to(destino.resolve()) or (entrada.external_attr >> 16) & 0o170000 == 0o120000:
                raise ValueError('Exportación con ruta no válida')
        paquete.extractall(destino)


def exportar(constructor, contenedor):
    version = json.loads(ejecutar('docker', 'exec', contenedor, 'cat', '/app/package.json'))['version']
    if version != '4.0.5':
        raise ValueError(f'Se requiere eXeLearning 4.0.5, disponible {version}')
    trabajo = ejecutar('docker', 'exec', contenedor, 'mktemp', '-d', '/tmp/etf-curso-XXXXXX').strip()
    try:
        ejecutar('docker', 'cp', str(SALIDA / 'expedientes-etf.elpx'), contenedor + ':' + trabajo + '/curso.elpx')
        for formato, nombre in [('html5', 'web'), ('scorm12', 'scorm12')]:
            ejecutar('docker', 'exec', contenedor, 'bun', '/app/dist/cli.js', 'elp:export', trabajo + '/curso.elpx', trabajo + '/' + nombre + '.zip', formato)
            ejecutar('docker', 'cp', contenedor + ':' + trabajo + '/' + nombre + '.zip', str(SALIDA / (nombre + '.zip')))
        web = SALIDA / 'web'
        if web.exists():
            shutil.rmtree(web)
        extraer_seguro(SALIDA / 'web.zip', web)
        for nombre in sorted(constructor.videos_usados):
            fuente = constructor.videos / nombre
            if fuente.is_file():
                (web / 'videos').mkdir(exist_ok=True)
                shutil.copyfile(fuente, web / 'videos' / nombre)
            else:
                raise ValueError(f'Vídeo enlazado ausente: {fuente}. Configura ETF_CURSO_VIDEOS.')
        # eXeLearning coloca las páginas secundarias en html/. Ajustar enlaces a vídeos externos.
        for pagina in web.rglob('*.html'):
            prefijo = os.path.relpath(web, pagina.parent).replace(os.sep, '/')
            prefijo = '' if prefijo == '.' else prefijo + '/'
            texto = pagina.read_text(encoding='utf-8')
            texto = re.sub(r'(src|href)="videos/', lambda m: m[1] + '="' + prefijo + 'videos/', texto)
            # El exportador admite la plantilla completa en el atributo. Así no hace fetch bajo file://.
            texto = texto.replace('data-idevice-template="form.html"',
                                  'data-idevice-template="&lt;div class=&quot;exe-form-container&quot;&gt;{content}&lt;/div&gt;"')
            # La edición web es de práctica; no intenta registrar notas en un LMS.
            texto = texto.replace('&quot;isScorm&quot;:1', '&quot;isScorm&quot;:0')
            pagina.write_text(texto, encoding='utf-8')
        # El SCORM no incorpora vídeos: conserva un aviso y retira referencias inexistentes.
        with zipfile.ZipFile(SALIDA / 'scorm12.zip') as original:
            entradas = {p.filename: original.read(p) for p in original.infolist() if not p.is_dir()}
        for nombre, contenido in list(entradas.items()):
            if nombre.endswith('.html'):
                texto = contenido.decode('utf-8')
                texto = re.sub(r'<video\b[^>]*src="(?:\.\./)*videos/[^"<>]+"[^>]*>.*?</video>',
                               '<p>' + html.escape(constructor.interfaz['video_scorm']) + '</p>', texto, flags=re.S)
                entradas[nombre] = texto.encode('utf-8')
        c = constructor.config
        autor = html.escape(c['autor'])
        derechos = html.escape('© 2026 ' + c['autor'] + ' · CC BY-SA 4.0 · ' + c['licencia_url'])
        lom = ('<imsmd:lom><imsmd:lifecycle><imsmd:contribute>'
               '<imsmd:role><imsmd:source>LOMv1.0</imsmd:source><imsmd:value>Author</imsmd:value></imsmd:role>'
               '<imsmd:centity><imsmd:vcard>BEGIN:VCARD\nVERSION:3.0\nFN:' + autor + '\nEND:VCARD</imsmd:vcard></imsmd:centity>'
               '</imsmd:contribute></imsmd:lifecycle><imsmd:rights>'
               '<imsmd:cost><imsmd:source>LOMv1.0</imsmd:source><imsmd:value>no</imsmd:value></imsmd:cost>'
               '<imsmd:copyrightandotherrestrictions><imsmd:source>LOMv1.0</imsmd:source><imsmd:value>yes</imsmd:value>'
               '</imsmd:copyrightandotherrestrictions><imsmd:description><imsmd:langstring xml:lang="es">' + derechos +
               '</imsmd:langstring></imsmd:description></imsmd:rights></imsmd:lom>')
        manifiesto = entradas['imsmanifest.xml'].decode('utf-8')
        entradas['imsmanifest.xml'] = manifiesto.replace('</metadata>', lom + '</metadata>', 1).encode('utf-8')
        with zipfile.ZipFile(SALIDA / 'scorm12.zip', 'w', zipfile.ZIP_DEFLATED) as paquete:
            for nombre, contenido in entradas.items():
                paquete.writestr(nombre, contenido)
        sys.path.insert(0, str(RAIZ / 'scripts/voz'))
        from curso import integrar
        integrar(constructor, SALIDA, Path(os.environ.get('ETF_CURSO_VOZ', str(RAIZ / 'bin/voz'))))
        (SALIDA / 'web.zip').unlink()
    finally:
        ejecutar('docker', 'exec', contenedor, 'rm', '-rf', trabajo)


def reimportar(proyecto, destino):
    """Recupera texto y formularios en una carpeta nueva, sin sobrescribir fuentes."""
    if destino.exists():
        raise ValueError('El destino de reimportación debe ser una carpeta nueva')
    config = cargar_yaml(CURSO / 'estructura.yaml')
    with zipfile.ZipFile(proyecto) as paquete:
        if paquete.getinfo('content.xml').file_size > MAX_RECURSO:
            raise ValueError('XML demasiado grande')
        contenido = paquete.read('content.xml')
        if len(contenido) > MAX_RECURSO or b'<!ENTITY' in contenido:
            raise ValueError('XML demasiado grande o con entidades no permitidas')
        raiz = ET.fromstring(contenido)
        navs = raiz.findall('.//{*}odeNavStructure')
        def valor(nodo, nombre):
            return nodo.findtext('{*}' + nombre, '')
        padres = [n for n in navs if not valor(n, 'odeParentPageId')]
        if len(padres) != len(config['modulos']):
            raise ValueError('Conserva los cuatro módulos del curso al editar en eXeLearning')
        padres.sort(key=lambda n: int(valor(n, 'odeNavStructureOrder')))
        if [valor(n, 'pageName') for n in padres] != [m['titulo'] for m in config['modulos']]:
            raise ValueError('Conserva los nombres y el orden de los cuatro módulos al editar')
        carpetas = {valor(n, 'odePageId'): m['carpeta'] for n, m in zip(padres, config['modulos'])}
        rutas = {}
        hijos = []
        for padre in padres:
            paginas = [n for n in navs if valor(n, 'odeParentPageId') == valor(padre, 'odePageId')]
            paginas.sort(key=lambda n: int(valor(n, 'odeNavStructureOrder')))
            for numero, pagina in enumerate(paginas, 1):
                carpeta = carpetas[valor(padre, 'odePageId')]
                componentes = pagina.findall('.//{*}odeComponent')
                tipos = [valor(c, 'odeIdeviceTypeName') for c in componentes]
                if any(t not in {'text', 'form', 'FreeTextIdevice'} for t in tipos) or ('form' in tipos and len(componentes) != 1):
                    raise ValueError('La reimportación admite texto y una autoevaluación por página; conserva otros iDevices en el proyecto original')
                extension = 'autoevaluacion.yaml' if tipos == ['form'] else 'pagina.md'
                rutas[valor(pagina, 'odePageId')] = Path('contenido') / carpeta / f'{numero:02d}-{extension}'
                hijos.append((pagina, componentes))
        if len(hijos) + len(padres) != len(navs):
            raise ValueError('La reimportación admite páginas directamente dentro de los cuatro módulos')
        # Validar todo antes de crear la carpeta de destino.
        recuperados = {}
        recursos = {}
        for pagina, componentes in hijos:
            ruta = rutas[valor(pagina, 'odePageId')]
            titulo = valor(pagina, 'pageName')
            if ruta.suffix == '.md':
                cuerpo = '\n\n'.join(valor(c, 'htmlView') for c in componentes)
                for idpagina, destino_enlace in rutas.items():
                    relativo = os.path.relpath(destino_enlace, ruta.parent).replace(os.sep, '/')
                    cuerpo = cuerpo.replace('exe-node:' + idpagina, relativo)
                for referencia in re.findall(r'(?:src|href)=["\']([^"\']+)', cuerpo):
                    referencia = html.unescape(referencia)
                    if referencia.startswith(('resources/', 'asset://', 'content/resources/')):
                        nombre = referencia.removeprefix('asset://').removeprefix('content/').removeprefix('resources/')
                        candidatas = [n for n in paquete.namelist() if n == 'resources/' + nombre or n.endswith('/' + nombre)]
                        if len(candidatas) != 1:
                            raise ValueError('Recurso no recuperable: ' + referencia)
                        archivo = paquete.getinfo(candidatas[0])
                        if archivo.file_size > MAX_RECURSO:
                            raise ValueError('Recurso demasiado grande')
                        nombre_salida = hashlib.sha256(candidatas[0].encode()).hexdigest()[:24] + Path(nombre).suffix
                        recursos[Path('recursos') / nombre_salida] = paquete.read(archivo)
                        cuerpo = cuerpo.replace(referencia, '../../recursos/' + nombre_salida)
                recuperados[ruta] = ('---\n' + yaml.safe_dump({'titulo': titulo}, allow_unicode=True) + '---\n\n' + cuerpo + '\n').encode()
            else:
                datos = json.loads(valor(componentes[0], 'jsonProperties'))
                preguntas = []
                for q in datos['questionsData']:
                    tipo = {'selection': 'test', 'true-false': 'vf', 'fill': 'huecos'}.get(q['activityType'])
                    if not tipo:
                        raise ValueError('Tipo de pregunta no recuperable')
                    pregunta = {'tipo': tipo, 'enunciado': q['baseText'], 'explicacion': q.get('feedbackRight', '')}
                    if tipo == 'test':
                        respuestas = q['answers']
                        get = lambda a, i: a[i] if isinstance(a, list) else a[str(i)]
                        pregunta['opciones'] = [get(a, 1) for a in respuestas]
                        pregunta['correcta'] = next(i for i, a in enumerate(respuestas) if get(a, 0))
                    elif tipo == 'vf':
                        pregunta['correcta'] = str(q['answer']) == '1'
                    else:
                        texto = re.sub(r'<u>(.*?)</u>', r'{\1}', q['baseText'], flags=re.S)
                        pregunta['texto'] = html.unescape(re.sub(r'<[^>]+>', '', texto)).strip()
                        pregunta['enunciado'] = cargar_yaml(CURSO / 'interfaz.yaml')['enunciado_huecos']
                    preguntas.append(pregunta)
                recuperados[ruta] = yaml.safe_dump({'titulo': titulo, 'preguntas': preguntas}, allow_unicode=True, sort_keys=False).encode()
        destino.mkdir(parents=True)
        for ruta, datos in {**recuperados, **recursos}.items():
            (destino / ruta.parent).mkdir(parents=True, exist_ok=True)
            (destino / ruta).write_bytes(datos)
        for nombre in ('estructura.yaml', 'interfaz.yaml', 'ajustes.css'):
            shutil.copyfile(CURSO / nombre, destino / nombre)
        print(f'REIMPORTACION_OK: {len(recuperados)} páginas recuperadas en {destino}; revisa antes de integrar')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--solo-proyecto', action='store_true')
    parser.add_argument('--reimportar', type=Path)
    parser.add_argument('--destino', type=Path)
    parser.add_argument('--contenedor', default=os.environ.get('ETF_CURSO_CONTENEDOR', 'etf-curso-exelearning-1'))
    args = parser.parse_args()
    if args.reimportar:
        if not args.destino:
            parser.error('--reimportar requiere --destino (carpeta nueva)')
        reimportar(args.reimportar, args.destino)
        return
    config = cargar_yaml(CURSO / 'estructura.yaml')
    videos = Path(os.environ.get('ETF_CURSO_VIDEOS', str(Path.home() / 'Videos/Expedientes-ETF'))).resolve()
    constructor = Constructor(config, videos)
    constructor.leer()
    SALIDA.mkdir(parents=True, exist_ok=True)
    constructor.elpx(SALIDA / 'expedientes-etf.elpx')
    if not args.solo_proyecto:
        exportar(constructor, args.contenedor)
    resumen = {'paginas': len(constructor.paginas), 'preguntas': constructor.preguntas, 'videos': sorted(constructor.videos_usados),
               'estructura': [{k: p[k] for k in ('titulo', 'fuente', 'tipo') if k in p} for p in constructor.paginas],
               'pendientes_validar': constructor.pendientes}
    (SALIDA / 'resumen.json').write_text(json.dumps(resumen, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(f'Curso: {len(config["modulos"])} módulos, {len(constructor.paginas)} páginas, {constructor.preguntas} preguntas.')
    print('Proyecto: bin/curso/expedientes-etf.elpx')
    if not args.solo_proyecto:
        print(f'Web: bin/curso/web/index.html; vídeos incluidos: {len(constructor.videos_usados)}')
        print('SCORM 1.2: bin/curso/scorm12.zip')
    print(f'Validaciones pendientes: {len(constructor.pendientes)} (bin/curso/resumen.json)')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, RuntimeError, OSError, yaml.YAMLError) as error:
        raise SystemExit(str(error)) from None
