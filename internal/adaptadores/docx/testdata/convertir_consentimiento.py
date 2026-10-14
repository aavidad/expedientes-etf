"""Genera las versiones locales de plantillas DOCX desde fuentes oficiales."""
from copy import deepcopy
import json
from pathlib import Path
import re
from xml.etree import ElementTree as E
from zipfile import ZipFile

RAIZ = Path(__file__).resolve().parents[4]
CARPETA = Path(__file__).resolve().parent
W = "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}"
X = "{http://www.w3.org/XML/1998/namespace}space"
ORIGENES = {
    "anexo1": "Anexo 1 - (Consentimiento Informado - SubPrograma Riesgo).docx",
    "anexo2": "Anexo 2 - (Consentimiento Informado - SubPrograma Reintegración).docx",
}


def leer_datos(idioma):
    ruta = RAIZ / "idiomas" / idioma / "plantillas" / "consentimiento" / "conversion.json"
    return json.loads(ruta.read_text(encoding="utf-8"))


def texto(parrafo):
    return "".join(n.text or "" for n in parrafo.iter(W + "t"))


def poner(parrafo, valor):
    estilo = parrafo.find(".//" + W + "rPr")
    for hijo in list(parrafo):
        if hijo.tag != W + "pPr":
            parrafo.remove(hijo)
    corrida = E.SubElement(parrafo, W + "r")
    if estilo is not None:
        corrida.append(deepcopy(estilo))
    nodo_texto = E.SubElement(corrida, W + "t", {X: "preserve"})
    nodo_texto.text = valor


def preparar(raiz, idioma, datos):
    reemplazos = {k.strip(): v for k, v in datos.get("reemplazos", datos.get("traducciones", {})).items()}
    repetidos = {k.strip(): v for k, v in datos.get("repetidos", {}).items()}
    usados = {clave: 0 for clave in repetidos}
    for parrafo in raiz.iter(W + "p"):
        clave = texto(parrafo).strip()
        if not clave:
            continue
        if clave in repetidos:
            opciones = repetidos[clave]
            posicion = usados[clave]
            if posicion >= len(opciones):
                raise ValueError("Hay más apariciones de la prevista en conversion.json")
            poner(parrafo, opciones[posicion])
            usados[clave] += 1
        elif clave in reemplazos:
            poner(parrafo, reemplazos[clave])
        elif idioma == "en":
            raise ValueError("Falta una clave en conversion.json para la traducción inglesa")
    for clave, opciones in repetidos.items():
        if usados[clave] != len(opciones):
            raise ValueError("El número de apariciones no coincide con conversion.json")


def convertir(nombre, archivo_origen, idioma, datos):
    with ZipFile(RAIZ / "docs" / "cliente" / archivo_origen) as entrada:
        original = entrada.read("word/document.xml")
        apertura = re.search(rb"<w:document\b[^>]*>", original).group()
        for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', apertura):
            E.register_namespace(prefijo.decode(), uri.decode())
        raiz = E.fromstring(original)
        preparar(raiz, idioma, datos)
        contenido = E.tostring(raiz, encoding="utf-8", xml_declaration=True)
        contenido = re.sub(rb"<w:document\b[^>]*>", lambda _: apertura, contenido, count=1)
        destino = RAIZ / "idiomas" / idioma / "plantillas" / "consentimiento" / nombre / "v1.0.0.docx"
        destino.parent.mkdir(parents=True, exist_ok=True)
        with ZipFile(destino, "w") as salida:
            for parte in entrada.infolist():
                datos_parte = contenido if parte.filename == "word/document.xml" else entrada.read(parte.filename)
                salida.writestr(deepcopy(parte), datos_parte)


for idioma in ("es", "en"):
    datos = leer_datos(idioma)
    for nombre, origen in ORIGENES.items():
        convertir(nombre, origen, idioma, datos)
