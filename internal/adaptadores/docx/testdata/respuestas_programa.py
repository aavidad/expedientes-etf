"""Publica candidatas Word con las respuestas del Programa, sin cambiar originales."""
import json
from pathlib import Path
from zipfile import ZipFile
from lxml import etree as E

RAIZ = Path(__file__).resolve().parents[4]
NS = {'w': 'http://schemas.openxmlformats.org/wordprocessingml/2006/main'}


def sustituir(parrafo, anterior, nuevo):
    nodos = parrafo.findall('.//w:t', NS)
    texto = ''.join(n.text or '' for n in nodos)
    while anterior in texto:
        inicio = texto.index(anterior)
        fin = inicio + len(anterior)
        posicion = 0
        primero = True
        for n in nodos:
            t = n.text or ''
            hasta = posicion + len(t)
            if posicion < fin and hasta > inicio:
                n.text = t[:max(0, inicio-posicion)] + (nuevo if primero else '') + t[max(0, fin-posicion):]
                primero = False
            posicion = hasta
        texto = ''.join(n.text or '' for n in nodos)


def publicar():
    for idioma in ('es', 'en'):
        base = RAIZ / 'idiomas' / idioma / 'plantillas'
        cambios = json.loads((base / 'respuestas-2026-10-14.json').read_text())
        for tipo, cambio in cambios.items():
            previa = base / tipo / ('v' + cambio['previa'] + '.docx')
            nueva = base / tipo / ('v' + cambio['nueva'] + '.docx')
            with ZipFile(previa) as z, nueva.open('xb') as f, ZipFile(f, 'w') as salida:
                for info in z.infolist():
                    contenido = z.read(info.filename)
                    if info.filename.startswith('word/') and info.filename.endswith('.xml'):
                        raiz = E.fromstring(contenido)
                        for p in raiz.findall('.//w:p', NS):
                            for anterior, nuevo in cambio['sustituciones'].items():
                                sustituir(p, anterior, nuevo)
                        contenido = E.tostring(raiz, xml_declaration=True, encoding='UTF-8', standalone=True)
                    salida.writestr(info, contenido)
            print(str(nueva.relative_to(RAIZ)))


if __name__ == '__main__':
    publicar()
