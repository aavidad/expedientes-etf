"""Crea la versión inmutable de los Anexos 11 y 12 con casillas de evaluación vacías."""
from pathlib import Path
import re
from xml.etree import ElementTree as E
from zipfile import ZipFile

RAIZ = Path(__file__).resolve().parents[4]
W = '{http://schemas.openxmlformats.org/wordprocessingml/2006/main}'
MOMENTOS = ('inicial', 'intermedia', 'final')


def texto(nodo):
    return ''.join(t.text or '' for t in nodo.iter(W + 't'))


def poner(celda, valor):
    # Se conservan las propiedades de la celda; se eliminan controles de lista
    # y sus valores de muestra para que el resultado sea editable en Word.
    for hijo in list(celda):
        if hijo.tag != W + 'tcPr':
            celda.remove(hijo)
    parrafo = E.SubElement(celda, W + 'p')
    run = E.SubElement(parrafo, W + 'r')
    E.SubElement(run, W + 't').text = valor


def preparar(idioma, anexo):
    carpeta = RAIZ / 'idiomas' / idioma / 'plantillas' / 'finalizacion' / anexo
    destino = carpeta / 'v1.0.2.docx'
    tablas = ('valorame', 'ecp12' if anexo == 'anexo11' else 'aapi25',
              'score15', 'sdq_autoinformada', 'sdq_parental',
              'soatif_familias', 'soatif_profesionales', 'satisfaccion')
    with ZipFile(carpeta / 'v1.0.1.docx') as original:
        xml = original.read('word/document.xml')
        apertura = re.search(rb'<w:document\b[^>]*>', xml).group()
        for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', apertura):
            E.register_namespace(prefijo.decode(), uri.decode())
        raiz = E.fromstring(xml)
        candidatas = [tabla for tabla in raiz.iter(W + 'tbl')
                      if any(token in texto(tabla) for token in
                             ('VALÓRAME', 'ECP 12', 'AAPI-2.5', 'SCORE 15', 'SDQ', 'SOATIF',
                              'Satisfacción con la intervención', 'Satisfaction with the intervention'))]
        if len(candidatas) != len(tablas):
            raise ValueError('No coinciden las tablas de evaluación con la plantilla de origen')
        for codigo, tabla in zip(tablas, candidatas):
            indice = 0
            for fila in tabla.findall(W + 'tr'):
                celdas = fila.findall(W + 'tc')
                if len(celdas) == 4 and any(texto(c) in ('Puntuación.', 'Score.', 'Elija un elemento.', 'Select an option.') for c in celdas[1:]):
                    indice += 1
                    for momento, celda in zip(MOMENTOS, celdas[1:]):
                        if texto(celda):
                            poner(celda, '{{escala_' + codigo + '_' + str(indice) + '_' + momento + '}}')
                elif len(celdas) == 1 and any(token in texto(fila).lower() for token in ('descripción cualitativa', 'qualitative description')):
                    etiqueta = texto(celdas[0]).split(':', 1)[0]
                    poner(celdas[0], etiqueta + ': {{escala_' + codigo + '_descripcion}}')
        salida = E.tostring(raiz, encoding='utf-8', xml_declaration=True)
        salida = re.sub(rb'<w:document\b[^>]*>', lambda _: apertura, salida, count=1)
        ajustes_xml = original.read('word/settings.xml')
        apertura_ajustes = re.search(rb'<w:settings\b[^>]*>', ajustes_xml).group()
        for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', apertura_ajustes):
            E.register_namespace(prefijo.decode(), uri.decode())
        ajustes = E.fromstring(ajustes_xml)
        for proteccion in ajustes.findall(W + 'documentProtection'):
            ajustes.remove(proteccion)
        salida_ajustes = E.tostring(ajustes, encoding='utf-8', xml_declaration=True)
        salida_ajustes = re.sub(rb'<w:settings\b[^>]*>', lambda _: apertura_ajustes, salida_ajustes, count=1)
        with ZipFile(destino, 'w') as nuevo:
            for entrada in original.infolist():
                contenido = (salida if entrada.filename == 'word/document.xml'
                             else salida_ajustes if entrada.filename == 'word/settings.xml'
                             else original.read(entrada.filename))
                nuevo.writestr(entrada, contenido)


if __name__ == '__main__':
    for idioma in ('es', 'en'):
        for anexo in ('anexo11', 'anexo12'):
            preparar(idioma, anexo)
