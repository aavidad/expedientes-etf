"""Sustituye los cuadros ActiveX vacíos de las candidatas SPAF por tablas Word.

Sólo modifica v1.0.1 en es/en. Mantiene los textos, los marcadores, la fuente,
el tamaño y las dimensiones de los cuadros; elimina partes y relaciones activas.
"""
from pathlib import Path
from zipfile import ZipFile
from lxml import etree as E

RAIZ = Path(__file__).resolve().parents[4]
W = 'http://schemas.openxmlformats.org/wordprocessingml/2006/main'
NS = {'w': W}


def elemento(nombre, **atributos):
    n = E.Element('{' + W + '}' + nombre)
    for clave, valor in atributos.items():
        n.set('{' + W + '}' + clave, str(valor))
    return n


def tabla(celdas):
    t = elemento('tbl')
    pr = E.SubElement(t, '{' + W + '}tblPr')
    pr.extend([elemento('tblW', w=sum(ancho for _, ancho, _ in celdas), type='dxa'),
               elemento('jc', val='center'), elemento('tblLayout', type='fixed')])
    margenes = E.SubElement(pr, '{' + W + '}tblCellMar')
    for lado in ('top', 'left', 'bottom', 'right'):
        margenes.append(elemento(lado, w=0, type='dxa'))
    grid = E.SubElement(t, '{' + W + '}tblGrid')
    grid.extend(elemento('gridCol', w=ancho) for _, ancho, _ in celdas)
    fila = E.SubElement(t, '{' + W + '}tr')
    altura = E.SubElement(fila, '{' + W + '}trPr')
    altura.append(elemento('trHeight', val=349, hRule='atLeast'))
    for texto, ancho, borde in celdas:
        c = E.SubElement(fila, '{' + W + '}tc')
        cp = E.SubElement(c, '{' + W + '}tcPr')
        cp.extend([elemento('tcW', w=ancho, type='dxa'), elemento('vAlign', val='center')])
        if borde:
            bordes = E.SubElement(cp, '{' + W + '}tcBorders')
            for lado in ('bottom', 'right'):
                bordes.append(elemento(lado, val='single', sz=4))
        p = E.SubElement(c, '{' + W + '}p')
        pp = E.SubElement(p, '{' + W + '}pPr')
        pp.extend([elemento('pStyle', val='NoSpacing'), elemento('jc', val='center')])
        r = E.SubElement(p, '{' + W + '}r')
        rp = E.SubElement(r, '{' + W + '}rPr')
        rp.extend([elemento('rFonts', ascii='Times New Roman', hAnsi='Times New Roman'),
                   elemento('b'), elemento('sz', val=28), elemento('szCs', val=28)])
        E.SubElement(r, '{' + W + '}t').text = texto
    return t


def sanear(ruta):
    with ZipFile(ruta) as z:
        partes = {n: z.read(n) for n in z.namelist()}
        entradas = z.infolist()
    doc = E.fromstring(partes['word/document.xml'])
    parrafos = doc.xpath('//w:p[w:r/w:object]', namespaces=NS)
    if not parrafos:
        return
    for p in parrafos:
        if '{{equipo}}' in ''.join(p.itertext()):
            nueva = tabla([('ETF nº', 1200, False), ('{{equipo}}', 2749, True)])
        else:
            nueva = tabla([('Mes', 700, False), ('{{mes}}', 2749, True),
                           ('Año', 1000, False), ('{{anio}}', 2749, True)])
        p.getparent().replace(p, nueva)
    partes['word/document.xml'] = E.tostring(doc, xml_declaration=True, encoding='UTF-8')
    for n in list(partes):
        if n.startswith(('word/activeX/', 'word/embeddings/')):
            del partes[n]
            continue
        if n.endswith('.rels') or n == '[Content_Types].xml':
            raiz = E.fromstring(partes[n])
            for hijo in list(raiz):
                valores = ' '.join(hijo.attrib.values()).lower()
                if 'activex' in valores or '/control' in valores or 'oleobject' in valores:
                    raiz.remove(hijo)
            partes[n] = E.tostring(raiz, xml_declaration=True, encoding='UTF-8')
    with ZipFile(ruta, 'w') as z:
        for entrada in entradas:
            if entrada.filename in partes:
                z.writestr(entrada, partes[entrada.filename])


if __name__ == '__main__':
    for idioma in ('es', 'en'):
        sanear(RAIZ / 'idiomas' / idioma / 'plantillas/spaf/v1.0.1.docx')
