"""Revisa la accesibilidad y corrige metadatos sin cambiar el formato visual.

Herramienta de desarrollo. Requiere lxml del sistema; no usa servicios externos.
--corregir crea candidatas nuevas; nunca sobrescribe versiones existentes.
"""
from copy import deepcopy
import argparse
import hashlib
import json
from pathlib import Path
import posixpath
from zipfile import ZipFile
from lxml import etree as E

RAIZ = Path(__file__).resolve().parents[4]
W = 'http://schemas.openxmlformats.org/wordprocessingml/2006/main'
R = 'http://schemas.openxmlformats.org/officeDocument/2006/relationships'
A = 'http://schemas.openxmlformats.org/drawingml/2006/main'
NS = {'w': W, 'a': A}


def tag(n):
    return '{' + W + '}' + n


def hijo(p, n):
    v = p.find(tag(n))
    if v is None:
        v = E.Element(tag(n))
        p.insert(0, v)
    return v


def titulo(documento, estilos, datos):
    # Un estilo por estilo base conserva fuente, tamaño, espaciado y márgenes.
    candidatos = []
    for p in documento.findall('.//w:body/w:p', NS):
        texto = ''.join(p.xpath('.//w:t/text()', namespaces=NS)).strip()
        if texto and '{{' not in texto and len(texto) > 12:
            candidatos.append((p, texto))
    if not candidatos:
        raise ValueError('documento.titulo_ausente')
    primero = next((p for p, texto in candidatos if texto.startswith('ASUNTO:')), candidatos[0][0])
    for p, texto in candidatos:
        nivel = 0 if p is primero else 1
        if p is not primero and (not texto.isupper() or len(texto) > 110 or texto.startswith(('18071', 'CONSEJERÍA', 'CENTRO DE SERVICIOS'))):
            continue
        pr = hijo(p, 'pPr')
        previo = pr.find(tag('pStyle'))
        base = previo.get(tag('val')) if previo is not None else 'Normal'
        if base.startswith('ETFRevision'):
            continue
        ident = 'ETFRevision' + str(nivel) + base
        if not estilos.xpath('w:style[@w:styleId=$id]', namespaces=NS, id=ident):
            s = E.SubElement(estilos, tag('style'), {tag('type'): 'paragraph', tag('styleId'): ident})
            E.SubElement(s, tag('name'), {tag('val'): datos['estilo_titulo' if nivel == 0 else 'estilo_seccion'] + ' (' + base + ')'})
            E.SubElement(s, tag('basedOn'), {tag('val'): base})
            E.SubElement(s, tag('qFormat'))
            sp = E.SubElement(s, tag('pPr'))
            E.SubElement(sp, tag('outlineLvl'), {tag('val'): str(nivel)})
        hijo(pr, 'pStyle').set(tag('val'), ident)
        # Sobrescribe niveles heredados sin alterar la presentación.
        hijo(pr, 'outlineLvl').set(tag('val'), str(nivel))


def cabeceras(documento):
    for tabla in documento.findall('.//w:tbl', NS):
        filas = tabla.findall('w:tr', NS)
        if len(filas) < 2:
            continue  # Tablas de disposición o fichas, sin cabecera de columnas.
        textos = [''.join(r.xpath('.//w:t/text()', namespaces=NS)) for r in filas[:2]]
        celdas = filas[0].findall('w:tc', NS)
        # Cabeceras de personas/firmas, resumen y evaluaciones de columnas.
        es_cabecera = (len(celdas) >= 3 and '{{' not in textos[0])
        normalizado = ''.join(textos[1].split())
        evaluacion = 'EVALUACIÓNINICIAL' in normalizado or 'INITIALASSESSMENT' in normalizado
        if es_cabecera or evaluacion:
            for fila in filas[:2 if evaluacion else 1]:
                hijo(hijo(fila, 'trPr'), 'tblHeader')


def imagenes(nombre, raiz, partes, datos):
    ruta = posixpath.join(posixpath.dirname(nombre), '_rels', posixpath.basename(nombre) + '.rels')
    if ruta not in partes:
        return
    rels = E.fromstring(partes[ruta])
    destinos = {r.get('Id'): posixpath.normpath(posixpath.join(posixpath.dirname(nombre), r.get('Target'))) for r in rels}
    for dibujo in raiz.xpath('//*[local-name()="inline" or local-name()="anchor"]'):
        blips = dibujo.findall('.//a:blip', NS)
        if not blips:
            continue
        contenido = partes[destinos[blips[0].get('{' + R + '}embed')]]
        huella = hashlib.sha256(contenido).hexdigest()
        descripcion = datos['imagenes'].get(huella)
        if descripcion is None:
            raise ValueError('documento.imagen_desconocida:' + huella)
        for prop in dibujo.xpath('.//*[local-name()="docPr" or local-name()="cNvPr"]'):
            prop.set('descr', descripcion)
            if not descripcion:
                # OOXML de Office 2019: la imagen es sólo un fondo decorativo.
                uri = 'http://schemas.microsoft.com/office/drawing/2017/decorative'
                exturi = '{C183D7F6-B498-43B3-948B-1728B52AA6E4}'
                extlst = prop.find('{' + A + '}extLst')
                if extlst is None:
                    extlst = E.SubElement(prop, '{' + A + '}extLst')
                if not extlst.xpath('*[@uri=$uri]', uri=exturi):
                    ext = E.SubElement(extlst, '{' + A + '}ext', {'uri': exturi})
                    E.SubElement(ext, '{' + uri + '}decorative', {'val': '1'})


def formato(documento, relativo):
    if str(relativo).startswith('finalizacion/anexo11/'):
        cuerpo = documento.find('w:body', NS)
        siguiente = cuerpo.findall('w:tbl', NS)[-1].getnext()
        # La conversión añadió un párrafo vacío antes del salto de sección.
        # Si la tabla llena la página, ese párrafo crea otra página vacía.
        if (siguiente is not None and siguiente.tag == tag('p')
                and not siguiente.xpath('.//w:t|.//w:sectPr', namespaces=NS)):
            cuerpo.remove(siguiente)
    if str(relativo).startswith('consentimiento/anexo2/'):
        # La conversión sustituyó todo el párrafo 22, incluido su salto final.
        p = documento.findall('.//w:p', NS)[22]
        if not p.xpath('.//w:br[@w:type="page"]', namespaces=NS):
            r = E.SubElement(p, tag('r'))
            E.SubElement(r, tag('br'), {tag('type'): 'page'})
    if str(relativo).startswith('salida/spm/anexo5/'):
        for p in documento.findall('.//w:p', NS):
            estilo = p.find('w:pPr/w:rPr', NS)
            if estilo is not None:
                for r in p.findall('w:r', NS):
                    if r.find(tag('rPr')) is None and r.find(tag('t')) is not None:
                        r.insert(0, deepcopy(estilo))
            # Un salto en un párrafo vacío ocupa una página adicional cuando
            # el formulario anterior llena su página. pageBreakBefore no duplica.
            saltos = p.xpath('.//w:br[@w:type="page"]', namespaces=NS)
            if saltos and not ''.join(p.xpath('.//w:t/text()', namespaces=NS)).strip():
                for salto in saltos:
                    salto.getparent().remove(salto)
                hijo(hijo(p, 'pPr'), 'pageBreakBefore')
                hijo(hijo(p, 'pPr'), 'keepNext')


def formato_traduccion(documento, ruta, relativo):
    if ruta.relative_to(RAIZ / 'idiomas').parts[0] != 'en' or not str(relativo).startswith('salida/anexo6/'):
        return
    pareja = RAIZ / 'idiomas/es/plantillas' / relativo
    if not pareja.exists():
        pareja = pareja.with_name('v1.0.0.docx')
    with ZipFile(pareja) as z:
        original = E.fromstring(z.read('word/document.xml')).findall('.//w:p', NS)
    traducido = documento.findall('.//w:p', NS)
    if len(original) != len(traducido):
        raise ValueError('documento.estructura_traduccion')
    for antes, despues in zip(original, traducido):
        estilo = antes.find('w:r/w:rPr', NS)
        if estilo is None or len(estilo) == 0:
            continue
        for r in despues.findall('w:r', NS):
            pr = r.find(tag('rPr'))
            if r.find(tag('t')) is None or (pr is not None and len(pr) > 0):
                continue
            if pr is not None:
                r.remove(pr)
            r.insert(0, deepcopy(estilo))


def revisar(ruta, corregir):
    idioma = ruta.relative_to(RAIZ / 'idiomas').parts[0]
    datos = json.loads((RAIZ / 'idiomas' / idioma / 'plantillas/revision.json').read_text())
    # Estos paquetes en inglés contienen todavía el texto oficial castellano.
    relativo = ruta.relative_to(RAIZ / 'idiomas' / idioma / 'plantillas')
    idioma_real = 'es-ES' if str(relativo).startswith(('entrada/', 'salida/spm/', 'spaf/')) else datos['idioma']
    with ZipFile(ruta) as z:
        partes = {i.filename: z.read(i.filename) for i in z.infolist()}
        infos = z.infolist()
    documento = E.fromstring(partes['word/document.xml'])
    estilos = E.fromstring(partes['word/styles.xml'])
    formato_traduccion(documento, ruta, relativo)
    formato(documento, relativo)
    titulo(documento, estilos, datos)
    cabeceras(documento)
    defecto = hijo(hijo(hijo(estilos, 'docDefaults'), 'rPrDefault'), 'rPr')
    hijo(defecto, 'lang').set(tag('val'), idioma_real)
    partes['word/document.xml'] = E.tostring(documento, encoding='UTF-8', xml_declaration=True)
    partes['word/styles.xml'] = E.tostring(estilos, encoding='UTF-8', xml_declaration=True)
    for nombre in list(partes):
        if nombre.startswith('word/') and nombre.endswith('.xml'):
            r = E.fromstring(partes[nombre])
            for lang in r.findall('.//w:lang', NS):
                for atributo in ('val', 'eastAsia', 'bidi'):
                    if atributo == 'val' or tag(atributo) in lang.attrib:
                        lang.set(tag(atributo), idioma_real)
            imagenes(nombre, r, partes, datos)
            partes[nombre] = E.tostring(r, encoding='UTF-8', xml_declaration=True)
    cambios = []
    with ZipFile(ruta) as z:
        for info in infos:
            if info.filename.endswith('.xml') and E.tostring(E.fromstring(z.read(info.filename)), method='c14n') != E.tostring(E.fromstring(partes[info.filename]), method='c14n'):
                cambios.append(info.filename)
    if corregir and cambios:
        temporal = ruta.with_suffix('.revision')
        with ZipFile(temporal, 'w') as z:
            for info in infos:
                z.writestr(info, partes[info.filename])
        temporal.replace(ruta)
    return cambios


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--corregir', action='store_true')
    args = parser.parse_args()
    pendientes = 0
    for origen in sorted((RAIZ / 'idiomas').glob('*/plantillas/**/*.docx')):
        if origen.name not in ('v1.0.0.docx', 'v1.1.0.docx'):
            continue
        ruta = origen.with_name('v1.1.1.docx' if origen.name == 'v1.1.0.docx' else 'v1.0.1.docx')
        creada = False
        if not ruta.exists():
            if not args.corregir:
                pendientes += 1
                print(ruta.relative_to(RAIZ), 'PENDIENTE')
                continue
            ruta.write_bytes(origen.read_bytes())
            creada = True
        cambios = revisar(ruta, args.corregir and creada)
        pendientes += bool(cambios)
        print(ruta.relative_to(RAIZ), 'CORREGIDA' if cambios and args.corregir and creada else 'PENDIENTE' if cambios else 'OK')
    print('PLANTILLAS:', pendientes, 'corregidas' if args.corregir else 'pendientes')
    if pendientes and not args.corregir:
        raise SystemExit(1)
