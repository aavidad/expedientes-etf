"""Convierte las cinco plantillas oficiales conservando textos y estilos.

Los botones ActiveX se sustituyen por marcas de selección. Los bloques de
personas pasan a filas repetibles. Ejecutar desde la raíz del repositorio.
Las versiones anteriores permanecen inmutables.
"""
from copy import deepcopy
from pathlib import Path
import os
import re
from xml.etree import ElementTree as E
from zipfile import ZipFile

RAIZ = Path(__file__).resolve().parents[4]
W = '{http://schemas.openxmlformats.org/wordprocessingml/2006/main}'
X = '{http://www.w3.org/XML/1998/namespace}space'
CENTROS = ['VEGAS_ALTAS', 'ALFACAR', 'VEGA_TEMPLE', 'SANTA_FE', 'RIO_FARDES',
           'VALLE_LECRIN_OGIJARES', 'SIERRA_NEVADA', 'MONTES_ORIENTALES',
           'PINOS_PUENTE_MOCLIN', 'HUETOR_TAJAR_MONTEFRIO', 'ALHAMA', 'COSTA', 'ALPUJARRA', 'NORTE']


def convertir(nombre, ruta, clase):
    with ZipFile(Path(os.environ.get('ETF_PLANTILLAS_FUENTE', str(RAIZ / 'bin/plantillas-fuente'))) / (nombre + '.docx')) as z:
        xml = z.read('word/document.xml')
        apertura = re.search(rb'<w:document\b[^>]*>', xml).group()
        for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', apertura):
            E.register_namespace(prefijo.decode(), uri.decode())
        raiz = E.fromstring(xml)
        cuerpo = raiz.find(W + 'body')
        ps = list(raiz.iter(W + 'p'))

        def texto(i):
            return ''.join(t.text or '' for t in ps[i].iter(W + 't'))

        def poner(i, valor):
            p = ps[i]
            estilo = p.find('.//' + W + 'rPr')
            for hijo in list(p):
                if hijo.tag != W + 'pPr':
                    p.remove(hijo)
            r = E.SubElement(p, W + 'r')
            if estilo is not None:
                estilo = deepcopy(estilo)
                for h in estilo.findall(W + 'highlight'):
                    estilo.remove(h)
                r.append(estilo)
            E.SubElement(r, W + 't', {X: 'preserve'}).text = valor

        def agregar(i, campo):
            r = E.SubElement(ps[i], W + 'r')
            E.SubElement(r, W + 't', {X: 'preserve'}).text = '{{' + campo + '}}'

        def bloque(indices, coleccion):
            tabla = E.Element(W + 'tbl')
            props = E.SubElement(tabla, W + 'tblPr')
            E.SubElement(props, W + 'tblW', {W + 'w': '0', W + 'type': 'auto'})
            grid = E.SubElement(tabla, W + 'tblGrid')
            E.SubElement(grid, W + 'gridCol', {W + 'w': '9000'})
            fila = E.SubElement(tabla, W + 'tr')
            celda = E.SubElement(fila, W + 'tc')
            pr = E.SubElement(celda, W + 'tcPr')
            E.SubElement(pr, W + 'tcW', {W + 'w': '9000', W + 'type': 'dxa'})
            cuerpo.insert(list(cuerpo).index(ps[indices[0]]), tabla)
            for i in indices:
                cuerpo.remove(ps[i]); celda.append(ps[i])
            poner(indices[0], '{{#' + coleccion + '}}' + texto(indices[0]))
            agregar(indices[-1], '/' + coleccion)

        # Se quitan sólo los objetos ejecutables, nunca sus etiquetas.
        for p in ps:
            for r in list(p):
                if r.tag == W + 'r' and r.find(W + 'object') is not None:
                    p.remove(r)

        if clase in ('acta', 'no_aceptacion'):
            es_acta = clase == 'acta'
            for i, campo in ({8: 'numero', 14: 'equipo', 25: 'centro', 55: 'descripcion',
                             59: 'familia', 60: 'madre', 61: 'madre_nacimiento',
                             63: 'padre', 64: 'padre_nacimiento', 95: 'profesional'} if es_acta else
                            {17: 'familia', 18: 'madre', 19: 'madre_nacimiento',
                             21: 'padre', 22: 'padre_nacimiento', 59: 'motivo'}).items():
                agregar(i, campo)
            m, servicio, firma, menores, otros, nota = ((42, 46, 106, 88, 66, 92) if es_acta else (45, 10, 66, 50, 24, 54))
            poner(m, re.sub(r'(MUNICIPIOS DE LA PROVINCIA\.|LISTADO DE LOS MUNICIPIOS DE LA PROVINCIA\.)', '{{municipio}}', texto(m)))
            agregar(servicio, 'servicio_derivante')
            poner(firma, texto(firma).replace('FIRMANTE_PENDIENTE', '{{firmante}}'))
            agregar(menores, 'nombre'); agregar(menores+2, 'nacimiento')
            poner(menores+1, texto(menores+1).replace('Varón', '{{marca_h}}Varón').replace('Femenino', '{{marca_m}}Femenino'))
            bloque([menores, menores+1, menores+2], 'menores')
            poner(otros, texto(otros).replace('1:', '{{orden}}:') + '{{nombre}}')
            agregar(otros+1, 'parentesco'); agregar(otros+2, 'nacimiento')
            bloque([otros, otros+1, otros+2], 'adultos')
            for n in range(1, 5):
                for j in range(4):
                    cuerpo.remove(ps[otros+4*n+j])
            # Esta nota del cliente describe la mecánica, no forma parte del acta.
            cuerpo.remove(ps[nota])
            if es_acta:
                for i in range(15, 23):
                    poner(i, '{{equipo_' + f'{i-14:02d}' + '}} ' + texto(i))
                for i, codigo in enumerate(CENTROS, 26):
                    poner(i, '{{centro_' + codigo.lower() + '}} ' + texto(i))
                for i, codigo in enumerate(['TS', 'ES', 'PS'], 96):
                    poner(i, '{{profesional_' + codigo.lower() + '}} ' + texto(i))
            for i, codigo in enumerate(['anexo3', 'anexo4', 'etf'], servicio+1):
                poner(i, '{{origen_' + codigo + '}} ' + texto(i))
        elif clase in ('sscc', 'spm'):
            for i, p in enumerate(ps):
                t = texto(i)
                if 'FIRMANTE_PENDIENTE' in t:
                    poner(i, t.replace('FIRMANTE_PENDIENTE', '{{firmante}}'))
                elif 'DESTINATARIO_PENDIENTE' in t:
                    poner(i, t.replace('DESTINATARIO_PENDIENTE', '{{jefe_spm}}'))
                elif 'XX–XX–XX' in t:
                    poner(i, t.replace('XX–XX–XX', '{{fecha_acta}}'))
                elif 'XXXXXXXXX, con domicilio' in t:
                    poner(i, t.replace('XXXXXXXXX', '{{familia}}').replace('XXXXXX', '{{direccion}}'))
                elif 'PERSONAS MENORES' in t:
                    poner(i, t.replace('XXXXXXXXX', '{{#menores}}{{nombre}}{{/menores}}'))
                    bloque([i], 'lista_menores')
                    # Una fila por menor; el rótulo permanece en la primera fila.
                    poner(i, '{{#lista_menores}}{{rotulo_menor}}{{nombre}}{{punto_final}}{{/lista_menores}}')
                elif 'XXXXXXXXXXXXXXXXXXXXX' in t:
                    poner(i, t.replace('XXXXXXXXXXXXXXXXXXXXX', '{{destinatario_sscc}}'))
                elif 'XXXXXXXXX' in t:
                    poner(i, t.replace('XXXXXXXXX', '{{equipo}}'))
        else:
            agregar(1, 'destinatario_sscc')
            poner(11, texto(11).replace('con fecha ,', 'con fecha {{fecha_derivacion}},')
                  .replace('los menores ,', 'los menores {{menores_lista}},')
                  .replace('municipio de,', 'municipio de{{municipio}},'))
            poner(23, texto(23).replace('Expediente: .', 'Expediente: {{numero}}.'))
            poner(29, texto(29).replace('FIRMANTE_PENDIENTE', '{{firmante}}'))

        salida_xml = E.tostring(raiz, encoding='utf-8', xml_declaration=True)
        salida_xml = re.sub(rb'<w:document\b[^>]*>', lambda _: apertura, salida_xml, count=1)
        for idioma in ('es', 'en'):
            destino = RAIZ / 'idiomas' / idioma / 'plantillas' / ruta
            destino.parent.mkdir(parents=True, exist_ok=True)
            with ZipFile(destino, 'w') as out:
                for parte in z.infolist():
                    if parte.filename.startswith('word/activeX/'):
                        continue
                    b = z.read(parte.filename)
                    if parte.filename == 'word/document.xml':
                        b = salida_xml
                    elif parte.filename == 'word/_rels/document.xml.rels':
                        b = re.sub(rb'<Relationship\b[^>]*Type="[^"]*/control"[^>]*/>', b'', b)
                    elif parte.filename == '[Content_Types].xml':
                        b = re.sub(rb'<Override\b[^>]*PartName="/word/activeX/[^>]*?/>', b'', b)
                    out.writestr(deepcopy(parte), b)


convertir('Acta Aceptación', 'entrada/aceptacion/v1.1.0.docx', 'acta')
convertir('Oficio de Acta de Aceptación para SSCC', 'entrada/aceptacion/oficio_sscc/v1.0.0.docx', 'sscc')
convertir('Oficio de Acta de Aceptación para SPM', 'entrada/aceptacion/oficio_spm/v1.0.0.docx', 'spm')
convertir('Acta NO Aceptación', 'entrada/no_aceptacion/v1.0.0.docx', 'no_aceptacion')
convertir('Oficio Lista de Espera', 'entrada/no_aceptacion/lista_espera/v1.0.0.docx', 'espera')
