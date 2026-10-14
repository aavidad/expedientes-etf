"""Rellena las tablas originales; la conversión PDF se hace con LibreOffice local."""
import copy
import json
import pathlib
import sys
import zipfile
from lxml import etree as E

W = "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}"
NS = {"w": W[1:-1]}


def metadatos(nombre, contenido):
    if nombre.startswith("docProps/") and nombre.endswith(".xml"):
        root = E.fromstring(contenido)
        for hijo in list(root):
            root.remove(hijo)
        return E.tostring(root, xml_declaration=True, encoding="UTF-8", standalone=True)
    return contenido


destino = pathlib.Path(sys.argv[1])
fuentes = pathlib.Path(sys.argv[2])
datos = json.loads(pathlib.Path(__file__).with_name("relleno.json").read_text())
destino.mkdir(parents=True, exist_ok=True)
if len(sys.argv) == 4:
    # Una página rasterizada, sin ningún nodo de texto en el cuerpo del Word.
    with zipfile.ZipFile(destino / "anexo3.docx") as archivo:
        root = E.fromstring(archivo.read("word/document.xml"))
        cuerpo = root.find("w:body", NS)
        dibujo = copy.deepcopy(root.find(".//w:drawing", NS))
        seccion = copy.deepcopy(cuerpo.find("w:sectPr", NS))
        for hijo in list(cuerpo):
            cuerpo.remove(hijo)
        p = E.SubElement(cuerpo, W + "p")
        E.SubElement(p, W + "r").append(dibujo)
        cuerpo.append(seccion)
        for elemento in dibujo.iter():
            if E.QName(elemento).localname in ("extent", "ext"):
                elemento.set("cx", "4800000")
                elemento.set("cy", "6788571")
            elemento.attrib.pop("descr", None)
        blip = dibujo.find(".//{http://schemas.openxmlformats.org/drawingml/2006/main}blip")
        id_imagen = blip.get("{http://schemas.openxmlformats.org/officeDocument/2006/relationships}embed")
        relaciones = E.fromstring(archivo.read("word/_rels/document.xml.rels"))
        ruta = next(x.get("Target") for x in relaciones if x.get("Id") == id_imagen)
        imagen = pathlib.Path(sys.argv[3]).read_bytes()
        with zipfile.ZipFile(destino / "sin_texto.docx", "w", zipfile.ZIP_DEFLATED) as salida:
            for info in archivo.infolist():
                contenido = archivo.read(info.filename)
                if info.filename == "word/document.xml":
                    contenido = E.tostring(root, xml_declaration=True, encoding="UTF-8", standalone=True)
                if info.filename == "word/" + ruta:
                    contenido = imagen
                salida.writestr(info, metadatos(info.filename, contenido))
    sys.exit(0)
for numero, anexo in datos["anexos"].items():
    origen = next(fuentes.glob(f"Anexo {numero} (*.docx"))
    with zipfile.ZipFile(origen) as archivo:
        root = E.fromstring(archivo.read("word/document.xml"))
        tablas = root.findall(".//w:tbl", NS)
        for indice, filas in anexo["tables"].items():
            tabla = tablas[int(indice)]
            originales = tabla.findall("w:tr", NS)
            for fila, valores in zip(originales, filas):
                for celda, valor in zip(fila.findall("w:tc", NS), valores):
                    parrafo = celda.find(".//w:p", NS)
                    estilo = copy.deepcopy(parrafo.find("w:pPr", NS))
                    for hijo in list(celda):
                        if hijo.tag != W + "tcPr":
                            celda.remove(hijo)
                    for linea in valor.split("\n"):
                        p = E.SubElement(celda, W + "p")
                        if estilo is not None:
                            p.append(copy.deepcopy(estilo))
                        E.SubElement(E.SubElement(p, W + "r"), W + "t").text = linea
            for fila in originales[len(filas):]:
                tabla.remove(fila)
        for texto in root.findall(".//w:t", NS):
            for antes, despues in datos["sustituciones"].items():
                texto.text = (texto.text or "").replace(antes, despues)
        with zipfile.ZipFile(destino / f"anexo{numero}.docx", "w", zipfile.ZIP_DEFLATED) as salida:
            for info in archivo.infolist():
                contenido = E.tostring(root, xml_declaration=True, encoding="UTF-8", standalone=True) if info.filename == "word/document.xml" else archivo.read(info.filename)
                salida.writestr(info, metadatos(info.filename, contenido))
