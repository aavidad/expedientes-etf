"""Prepara el Anexo 5 oficial con marcadores y filas repetibles.

Conserva rótulos y estilos. Los apartados 5 a 8 quedan disponibles para que
una persona los complete después de descargar el Word.
"""
from copy import deepcopy
from pathlib import Path
import os
from xml.etree import ElementTree as E
from zipfile import ZipFile
import re

RAIZ = Path(__file__).resolve().parents[4]
W = '{http://schemas.openxmlformats.org/wordprocessingml/2006/main}'
with ZipFile(Path(os.environ.get('ETF_PLANTILLAS_FUENTE', str(RAIZ / 'bin/plantillas-fuente'))) / 'versiones-30-09/Anexo 5 (De ETF a SPM).docx') as z:
    xml = z.read('word/document.xml')
    for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', re.search(rb'<w:document\b[^>]*>', xml).group()):
        E.register_namespace(prefijo.decode(), uri.decode())
    raiz = E.fromstring(xml)
    # Se conservan los contenidos y se retiran controles del formulario.
    for padre in list(raiz.iter()):
        for hijo in list(padre):
            if hijo.tag == W + 'sdt':
                indice = list(padre).index(hijo)
                contenido = hijo.find(W + 'sdtContent')
                padre.remove(hijo)
                if contenido is not None:
                    for nodo in list(contenido):
                        padre.insert(indice, nodo)
                        indice += 1
    ps = list(raiz.iter(W + 'p'))
    def poner(i, texto):
        p = ps[i]
        for hijo in list(p):
            if hijo.tag != W + 'pPr':
                p.remove(hijo)
        r = E.SubElement(p, W + 'r')
        E.SubElement(r, W + 't').text = texto
    campos = {4:'numero',6:'fecha_documento',8:'fecha_entrada_sscc',10:'entidad_derivante',12:'equipo_menores',
              15:'equipo',16:'profesional',17:'profesional_telefono',18:'profesional_email',34:'direccion',
              60:'estructura_familiar',61:'convivencia_otras',64:'vivienda',66:'situacion_laboral',65:'ingresos',67:'situacion_economica',68:'tramo_ingresos'}
    for i,campo in campos.items():
        original=''.join(t.text or '' for t in ps[i].iter(W+'t'))
        prefijo=''
        if i in [15,18,64,65,66,67,68]:
            prefijo=original.split(':',1)[0]+': '
        elif i==16:
            prefijo=original.split('nombre',1)[0]
        elif i==17:
            prefijo=original.split('teléfono',1)[0]
        poner(i,prefijo+'{{'+campo+'}}')
    def fila(indices, coleccion, nombres, quitar):
        for i,nombre in zip(indices,nombres):
            poner(i,'{{'+nombre+'}}')
        p = ps[indices[0]]
        padres = {h:padre for padre in raiz.iter() for h in padre}
        while p.tag != W+'tr':
            p = padres[p]
        textos = list(p.iter(W+'t'))
        textos[0].text = '{{#'+coleccion+'}}'+textos[0].text
        textos[-1].text += '{{/'+coleccion+'}}'
        for indice in quitar:
            nodo=ps[indice]
            while nodo.tag != W+'tr':
                nodo=padres[nodo]
            if nodo in list(padres[nodo]):
                padres[nodo].remove(nodo)
    fila([25,26,27,28],'spm_contactos',['nombre','apellidos','telefono','email'],[29])
    fila([41,42,43,44],'spm_menores',['nombre','nacimiento','escolarizacion','centro_educativo'],[45,49,53])
    poner(78,'{{orden}}')
    fila([79,80,81,82,83],'spm_adultos',['nombre','parentesco','edad','estudios','otros'],[84,90,96,102])
    fila([117,118,119,120],'spm_significativas',['nombre','apellidos','parentesco','telefono'],[121,125,129])
    # Rótulos oficiales conservados; sólo se vacían las instrucciones de entrada.
    vaciar=[138,146,148,150,152,166,169,172,175,178,181,184,187,190,193,198,202,206,209,212,214,224,228,230,232,234,236,238,242,244,246,248,250,252,257,264]
    for i in vaciar:
        poner(i,'')
    for i,texto in {136:'Tiempo de convivencia:',141:'Descripción de la situación que motiva la derivación:',142:'Duración de la problemática:',195:'Descripción de los resultados:'}.items():
        poner(i,texto)
    nuevo=E.tostring(raiz,encoding='utf-8',xml_declaration=True)
    for idioma in ['es','en']:
        destino=RAIZ/'idiomas'/idioma/'plantillas/salida/spm/anexo5/v1.0.0.docx'
        destino.parent.mkdir(parents=True,exist_ok=True)
        with ZipFile(destino,'w') as salida:
            for parte in z.infolist():
                if parte.filename.startswith(('word/activeX/','word/embeddings/')):
                    continue
                salida.writestr(deepcopy(parte),nuevo if parte.filename=='word/document.xml' else z.read(parte.filename))
