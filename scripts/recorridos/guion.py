"""Lee guiones Markdown. Los textos del vídeo nunca se generan desde el código."""
import json
import re
from pathlib import Path


def leer(ruta):
    texto = Path(ruta).read_text(encoding="utf-8")
    secciones = {}
    actual = None
    for linea in texto.splitlines():
        if linea.startswith("## "):
            actual = linea[3:].strip()
            if actual in secciones:
                raise ValueError(f"Apartado repetido: {actual}")
            secciones[actual] = []
        elif actual:
            secciones[actual].append(linea)

    def campos(nombre):
        resultado = {}
        for linea in secciones.get(nombre, []):
            m = re.fullmatch(r"(Título|Cartela|Rótulo|Locución):\s*(.+)", linea)
            if m:
                if m[1] in resultado:
                    raise ValueError(f"Campo repetido: {nombre}/{m[1]}")
                resultado[m[1]] = m[2].strip()
        return resultado

    def bloque(nombre, claves):
        c = campos(nombre)
        if any(not c.get(k) for k in claves):
            raise ValueError(f"Falta {nombre}: exige {', '.join(claves)}")
        if re.search(r"\b[A-ZÁÉÍÓÚÜÑ]{2,}\b", c["Locución"]):
            raise ValueError(f"Desarrolla las siglas en la locución de {nombre}")
        return c

    intro = bloque("Introducción", ["Título", "Cartela", "Locución"])
    palabras = len(intro["Locución"].split())
    if not 28 <= palabras <= 48:
        raise ValueError("La introducción debe tener 28–48 palabras; duración comprobada al montar")
    cierre = bloque("Cierre", ["Título", "Cartela", "Locución"])
    pasos = []
    for nombre in secciones:
        if nombre.startswith("Paso "):
            if not re.fullmatch(r"Paso [a-z0-9-]+", nombre):
                raise ValueError(f"Identificador de paso no válido: {nombre}")
            pasos.append({"id": nombre[5:], **bloque(nombre, ["Rótulo", "Locución"])})
    if not pasos or len({p["Rótulo"] for p in pasos}) != len(pasos):
        raise ValueError("Faltan pasos o hay rótulos duplicados")
    for paso in pasos:
        if not 4 <= len(paso['Rótulo'].split()) <= 8:
            raise ValueError('Cada rótulo debe tener 4–8 palabras')
        if paso['Rótulo'] == paso['Locución']:
            raise ValueError('El rótulo resume la locución, sin repetirla')
    preparacion = "\n".join(secciones.get("Preparación", []))
    m = re.search(r"```json\s*\n(.*?)\n```", preparacion, re.S)
    if not m:
        raise ValueError("Falta el bloque JSON de Preparación")
    config = json.loads(m[1])
    return {"intro": intro, "cierre": cierre, "pasos": pasos, "config": config}
