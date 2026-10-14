"""Prepara el Acta de Cierre oficial por idioma; sólo se ejecuta en desarrollo.

Conserva estilos, cabeceras y pies. Los textos y sus traducciones se leen del
paquete de idioma. La plantilla publicada no se sobrescribe.
"""
from copy import deepcopy
import hashlib
import json
from pathlib import Path
import os
import re
from xml.etree import ElementTree as E
from zipfile import ZipFile

RAIZ = Path(__file__).resolve().parents[4]
W = '{http://schemas.openxmlformats.org/wordprocessingml/2006/main}'
X = '{http://www.w3.org/XML/1998/namespace}space'


def convertir(idioma):
    carpeta = RAIZ / 'idiomas' / idioma / 'plantillas/cierre/acta_cierre'
    textos = json.loads((carpeta / 'textos.json').read_text())
    destino = carpeta / 'v1.0.0.docx'
    if destino.exists():
        raise RuntimeError('documento.version_inmutable')
    with ZipFile(Path(os.environ.get('ETF_PLANTILLAS_FUENTE', str(RAIZ / 'bin/plantillas-fuente'))) / 'Acta Cierre.docx') as entrada:
        xml = entrada.read('word/document.xml')
        apertura = re.search(rb'<w:document\b[^>]*>', xml).group()
        for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', apertura):
            E.register_namespace(prefijo.decode(), uri.decode())
        raiz = E.fromstring(xml)
        cuerpo = raiz.find(W + 'body')
        ps = list(raiz.iter(W + 'p'))
        for p in ps:
            for r in list(p):
                if r.tag == W + 'r' and r.find(W + 'object') is not None:
                    p.remove(r)
        def poner(i, valor):
            p = ps[i]
            estilo = p.find('.//' + W + 'rPr')
            for h in list(p):
                if h.tag != W + 'pPr':
                    p.remove(h)
            r = E.SubElement(p, W + 'r')
            if estilo is not None:
                r.append(deepcopy(estilo))
            E.SubElement(r, W + 't', {X: 'preserve'}).text = valor
        for i, valor in textos.items():
            poner(int(i), valor)
        # Tabla repetible: una fila por menor, hasta el límite del contrato.
        tabla = E.Element(W + 'tbl')
        props = E.SubElement(tabla, W + 'tblPr')
        E.SubElement(props, W + 'tblW', {W + 'w': '0', W + 'type': 'auto'})
        grid = E.SubElement(tabla, W + 'tblGrid')
        E.SubElement(grid, W + 'gridCol', {W + 'w': '9000'})
        fila = E.SubElement(tabla, W + 'tr')
        celda = E.SubElement(fila, W + 'tc')
        cuerpo.insert(list(cuerpo).index(ps[75]), tabla)
        for i in (75, 76, 77):
            cuerpo.remove(ps[i])
            celda.append(ps[i])
        for i in [*range(15, 23), *range(26, 40), 79, *range(84, 89)]:
            cuerpo.remove(ps[i])
        contenido = E.tostring(raiz, encoding='utf-8', xml_declaration=True)
        contenido = re.sub(rb'<w:document\b[^>]*>', lambda _: apertura, contenido, count=1)
        with ZipFile(destino, 'w') as salida:
            for parte in entrada.infolist():
                if parte.filename.startswith('word/activeX/'):
                    continue
                datos = entrada.read(parte.filename)
                if parte.filename == 'word/document.xml':
                    datos = contenido
                elif parte.filename == 'word/_rels/document.xml.rels':
                    datos = re.sub(rb'<Relationship\b[^>]*Type="[^"]*/control"[^>]*/>', b'', datos)
                elif parte.filename == '[Content_Types].xml':
                    datos = re.sub(rb'<Override\b[^>]*PartName="/word/activeX/[^>]*?/>', b'', datos)
                salida.writestr(parte, datos)
    return hashlib.sha256(destino.read_bytes()).hexdigest()


if __name__ == '__main__':
    for codigo in ('es', 'en'):
        print(codigo, convertir(codigo))
