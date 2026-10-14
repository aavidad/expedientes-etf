"""Añade marcadores a la tabla oficial SPAF sin cambiar sus textos fijos."""
from copy import copy
from pathlib import Path
import os
from xml.etree import ElementTree as E
from zipfile import ZipFile
import re

RAIZ = Path(__file__).resolve().parents[4]
W = '{http://schemas.openxmlformats.org/wordprocessingml/2006/main}'
with ZipFile(Path(os.environ.get('ETF_PLANTILLAS_FUENTE', str(RAIZ / 'bin/plantillas-fuente'))) / 'Tabla Mensual para el SPAF.docx') as z:
    xml = z.read('word/document.xml')
    for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', xml):
        E.register_namespace(prefijo.decode(), uri.decode())
    r = E.fromstring(xml)
    ps = list(r.iter(W+'p'))
    def agregar(p, texto):
        E.SubElement(E.SubElement(p, W+'r'), W+'t').text = texto
    agregar(ps[2], '{{equipo}}')
    agregar(ps[3], '{{mes}} / {{anio}}')
    tablas = list(r.iter(W+'tbl'))
    filas = tablas[0].findall(W+'tr')
    pr = filas[0].find(W+'trPr')
    if pr is None:
        pr = E.SubElement(filas[0],W+'trPr')
    E.SubElement(pr,W+'tblHeader')
    campos = ['numero','menores','numero_menores','guardadores','municipio_zts',
              'caracteristicas','derivacion','servicio','fechas','fase',
              'temporalizacion','finalidad_pronostico','instrumentos','cierres']
    for i,(celda,campo) in enumerate(zip(filas[1].findall(W+'tc'),campos)):
        p=celda.find(W+'p')
        agregar(p, ('{{#casos_spaf}}' if i==0 else '')+'{{'+campo+'}}'+
                ('{{/casos_spaf}}' if i==13 else ''))
    for fila in filas[2:]:
        tablas[0].remove(fila)
    for celda,campo in zip(tablas[1].findall(W+'tr')[1].findall(W+'tc'),
                           ['altas','bajas','casos','menores']):
        agregar(celda.find(W+'p'),'{{'+campo+'}}')
    for idioma in ('es','en'):
        ruta=RAIZ/f'idiomas/{idioma}/plantillas/spaf/v1.0.0.docx'
        ruta.parent.mkdir(parents=True,exist_ok=True)
        with ZipFile(ruta,'w') as salida:
            for info in z.infolist():
                salida.writestr(copy(info),E.tostring(r,encoding='utf-8',xml_declaration=True)
                                if info.filename=='word/document.xml' else z.read(info.filename))
