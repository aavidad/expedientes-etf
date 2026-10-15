"""Publica traducciones inglesas conservando tablas, estilos y relaciones."""
import json
import re
from pathlib import Path
from zipfile import ZipFile

from lxml import etree as E

RAIZ = Path(__file__).resolve().parents[4]
BASE = RAIZ / 'idiomas/en/plantillas'
W = 'http://schemas.openxmlformats.org/wordprocessingml/2006/main'
NS = {'w': W}
MARCA = re.compile(r'(\{\{[^{}]+\}\})')


def repartir(original, traduccion, nodos):
    """Conserva los runs y ancla cada marcador en su posición original."""
    origen = MARCA.split(original)
    destino = MARCA.split(traduccion)
    if origen[1::2] != destino[1::2]:
        raise ValueError('documento.traduccion_marcadores')
    limites = []
    acumulado = 0
    for n in nodos:
        acumulado += len(n.text or '')
        limites.append(acumulado)
    segmentos = [''] * len(nodos)
    posicion = 0
    for indice, (antes, despues) in enumerate(zip(origen, destino, strict=True)):
        if not antes and not despues:
            continue
        fin = posicion + len(antes)
        cortes = [0]
        for limite in sorted(set(limites[:-1])):
            if posicion < limite < fin:
                relativo = limite - posicion
                if indice % 2:
                    corte = relativo
                else:
                    ideal = round(relativo * len(despues) / len(antes))
                    # El formato de un run no debe cortar una palabra traducida.
                    candidatos = [m.end() for m in re.finditer(r'\s+', despues)]
                    corte = min([0, len(despues), *candidatos], key=lambda n: abs(n - ideal))
                cortes.append(max(cortes[-1], corte))
        cortes.append(len(despues))
        afectados = [i for i, limite in enumerate(limites)
                     if nodos[i].text and limite > posicion and limite - len(nodos[i].text) < fin]
        if not antes and despues:
            afectados = [next((i for i, limite in enumerate(limites) if limite >= posicion), 0)]
        for i, a, b in zip(afectados, cortes[:-1], cortes[1:], strict=True):
            segmentos[i] += despues[a:b]
        posicion = fin
    for n, texto in zip(nodos, segmentos, strict=True):
        n.text = texto
        if texto != texto.strip():
            n.set('{http://www.w3.org/XML/1998/namespace}space', 'preserve')


def publicar():
    for datos in sorted(BASE.glob('traduccion-*.json')):
        for tipo, cambio in json.loads(datos.read_text()).items():
            previa = BASE / tipo / ('v' + cambio['previa'] + '.docx')
            nueva = BASE / tipo / ('v' + cambio['nueva'] + '.docx')
            encontrados = set()
            with ZipFile(previa) as z, nueva.open('xb') as f, ZipFile(f, 'w') as salida:
                for info in z.infolist():
                    contenido = z.read(info.filename)
                    if info.filename.startswith('word/') and info.filename.endswith('.xml'):
                        raiz = E.fromstring(contenido)
                        modificado = False
                        # Los párrafos de cuadros de texto se procesan antes que sus contenedores.
                        for p in reversed(raiz.findall('.//w:p', NS)):
                            nodos = p.findall('.//w:t', NS)
                            texto = ''.join(n.text or '' for n in nodos)
                            if texto in cambio['sustituciones']:
                                repartir(texto, cambio['sustituciones'][texto], nodos)
                                encontrados.add(texto)
                                modificado = True
                        for opcion in raiz.findall('.//w:listItem', NS):
                            atributo = '{' + W + '}displayText'
                            texto = opcion.get(atributo)
                            if texto in cambio.get('opciones', {}):
                                opcion.set(atributo, cambio['opciones'][texto])
                                modificado = True
                        for idioma in raiz.findall('.//w:lang', NS):
                            for clave in ('val', 'eastAsia', 'bidi'):
                                atributo = '{' + W + '}' + clave
                                if atributo in idioma.attrib and idioma.get(atributo) != 'en-GB':
                                    idioma.set(atributo, 'en-GB')
                                    modificado = True
                        if modificado:
                            contenido = E.tostring(raiz, xml_declaration=True, encoding='UTF-8', standalone=True)
                    salida.writestr(info, contenido)
            faltan = cambio['sustituciones'].keys() - encontrados
            if faltan:
                nueva.unlink()
                raise ValueError(f'documento.traduccion_sin_aplicar: {tipo}: {faltan}')
            print(nueva.relative_to(RAIZ))


if __name__ == '__main__':
    publicar()
