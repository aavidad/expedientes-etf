"""Conversión local y reproducible de la copia del Acta para F0.17b.

Ejecutar desde la raíz: python3 internal/adaptadores/docx/testdata/convertir_acta.py
Sólo usa la biblioteca estándar. Nunca se ejecuta en producción.
"""
from copy import deepcopy
from pathlib import Path
import os
import re
from xml.etree import ElementTree as E
from zipfile import ZipFile

W = "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}"
carpeta = Path(__file__).resolve().parent
origen = Path(os.environ.get("ETF_PLANTILLAS_FUENTE", str(carpeta.parents[3] / "bin/plantillas-fuente"))) / "Acta Aceptación.docx"
destino = carpeta / "acta_aceptacion.docx"
# La versión publicada se conserva inmutable; esta salida sólo sirve para
# preparar y revisar una conversión nueva antes de publicarla.

with ZipFile(origen) as entrada:
    xml = entrada.read("word/document.xml")
    apertura = re.search(rb"<w:document\b[^>]*>", xml).group()
    for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', apertura):
        E.register_namespace(prefijo.decode(), uri.decode())
    raiz = E.fromstring(xml)
    cuerpo = raiz.find(W + "body")
    parrafos = list(raiz.iter(W + "p"))

    # Quita los objetos ActiveX del documento de prueba. Las opciones pasan
    # a ser datos y los textos fijos restantes conservan sus estilos.
    for p in parrafos:
        for r in list(p):
            if r.tag == W + "r" and r.find(W + "object") is not None:
                p.remove(r)

    def poner(indice, texto):
        p = parrafos[indice]
        estilo = p.find(".//" + W + "rPr")
        for hijo in list(p):
            if hijo.tag != W + "pPr":
                p.remove(hijo)
        r = E.SubElement(p, W + "r")
        if estilo is not None:
            r.append(deepcopy(estilo))
        t = E.SubElement(r, W + "t", {"{http://www.w3.org/XML/1998/namespace}space": "preserve"})
        t.text = texto

    def agregar(indice, marcador):
        r = E.SubElement(parrafos[indice], W + "r")
        E.SubElement(r, W + "t").text = "{{" + marcador + "}}"

    for indice, campo in {8: "numero", 14: "equipo", 25: "centro", 55: "descripcion", 59: "familia", 60: "madre", 61: "madre_nacimiento", 63: "padre", 64: "padre_nacimiento", 95: "profesional"}.items():
        agregar(indice, campo)
    for n, indice in enumerate([66, 70, 74, 78, 82], 1):
        agregar(indice, f"adulto{n}")
        agregar(indice + 1, f"adulto{n}_parentesco")
        agregar(indice + 2, f"adulto{n}_nacimiento")
    poner(42, "MUNICIPIO: {{municipio}}")
    poner(46, "Caso procedente de: {{servicio_derivante}}")
    poner(106, "Fdo. {{firmante}}")
    poner(88, "{{#menores}}Nombre: {{nombre}}")
    poner(89, "Sexo: {{sexo}}")
    poner(90, "Fecha de Nacimiento: {{nacimiento}}{{/menores}}")

    tabla = E.Element(W + "tbl")
    propiedades = E.SubElement(tabla, W + "tblPr")
    E.SubElement(propiedades, W + "tblW", {W + "w": "0", W + "type": "auto"})
    grid = E.SubElement(tabla, W + "tblGrid")
    E.SubElement(grid, W + "gridCol", {W + "w": "9000"})
    fila = E.SubElement(tabla, W + "tr")
    celda = E.SubElement(fila, W + "tc")
    pr = E.SubElement(celda, W + "tcPr")
    E.SubElement(pr, W + "tcW", {W + "w": "9000", W + "type": "dxa"})
    cuerpo.insert(list(cuerpo).index(parrafos[88]), tabla)
    for indice in [88, 89, 90]:
        cuerpo.remove(parrafos[indice])
        celda.append(parrafos[indice])
    # Opciones que ahora recibe el motor y nota editorial del bloque de menores.
    for indice in [*range(15, 23), *range(26, 40), 47, 48, 49, 92, 96, 97, 98]:
        cuerpo.remove(parrafos[indice])

    contenido = E.tostring(raiz, encoding="utf-8", xml_declaration=True)
    # Conserva las declaraciones originales, incluidas las usadas por VML.
    contenido = re.sub(rb"<w:document\b[^>]*>", lambda _: apertura, contenido, count=1)
    with ZipFile(destino, "w") as salida:
        for parte in entrada.infolist():
            if parte.filename.startswith("word/activeX/"):
                continue
            datos = entrada.read(parte.filename)
            if parte.filename == "word/document.xml":
                datos = contenido
            elif parte.filename == "word/_rels/document.xml.rels":
                datos = re.sub(rb'<Relationship\b[^>]*Type="[^"]*/control"[^>]*/>', b"", datos)
            elif parte.filename == "[Content_Types].xml":
                datos = re.sub(rb'<Override\b[^>]*PartName="/word/activeX/[^>]*?/>', b"", datos)
            salida.writestr(parte, datos)

print(destino.relative_to(carpeta.parents[3]))
