#!/usr/bin/env python3
"""Compara los PDF reunidos con los publicados en las ramas de origen, sin red."""

from pathlib import Path
import re
import subprocess
import tempfile
from urllib.parse import unquote, urlsplit

BASE = Path(__file__).resolve().parents[2]
ORIGENES = (
    ("manual-sistemas", "docs/sistemas/sistemas.pdf"),
    ("manual-programadores", "docs/desarrollo/programadores.pdf"),
    ("manual-usuario-etf", "docs/manuales/profesionales.pdf"),
    ("manual-usuario-etf", "docs/manuales/coordinacion.pdf"),
    ("manual-usuario-etf", "docs/manuales/administracion.pdf"),
)


def comprobar(ruta: Path, enlaces_locales: bool = False) -> tuple:
    datos = ruta.read_bytes()
    info = subprocess.check_output(["pdfinfo", str(ruta)], text=True)
    paginas = int(re.search(r"Pages:\s+(\d+)", info)[1])
    texto = subprocess.check_output(["pdftotext", str(ruta), "-"], text=True)
    texto = " ".join(re.sub(r"Expedientes ETF\s*·\s*\d+", "", texto).split())
    imagenes = subprocess.check_output(["pdfimages", "-list", str(ruta)], text=True)
    # Las dimensiones también detectan capturas recortadas o diagramas perdidos.
    imagenes = [tuple(l.split()[2:7]) for l in imagenes.splitlines()[2:] if l.strip()]
    # LibreOffice escribe las anotaciones y los destinos como objetos no comprimidos.
    objetos = {int(m[1]): m[2] for m in re.finditer(rb"(?m)^(\d+) 0 obj\s*(.*?)\s*endobj", datos, re.S)}
    destinos = re.findall(rb"/Dest\[(\d+) 0 R", datos)
    assert destinos, f"{ruta.name}: faltan capítulos y destinos internos"
    for destino in destinos:
        assert re.search(rb"/Type\s*/Page\b", objetos[int(destino)]), f"Destino inválido: {destino}"
    enlaces = sum(b"/Subtype/Link" in o and b"/Dest[" in o for o in objetos.values())
    assert enlaces, f"{ruta.name}: faltan enlaces internos"
    if enlaces_locales:
        urls = subprocess.check_output(["pdfinfo", "-url", str(ruta)], text=True)
        for linea in urls.splitlines()[1:]:
            destino = linea.split(maxsplit=2)[2]
            uri = urlsplit(destino)
            if not uri.scheme or uri.scheme == "file":
                local = Path(unquote(uri.path))
                if not local.is_absolute():
                    local = ruta.parent / local
                assert local.exists(), f"Enlace local roto: {destino}"
    return paginas, imagenes, enlaces, len(destinos), texto


def main() -> None:
    with tempfile.TemporaryDirectory(prefix="MANU-pdf-") as temporal:
        for rama, nombre in ORIGENES:
            anterior = Path(temporal) / Path(nombre).name
            anterior.write_bytes(subprocess.check_output(["git", "-C", str(BASE), "show", f"{rama}:{nombre}"]))
            publicado = BASE / nombre
            origen = comprobar(anterior)
            unido = comprobar(publicado, enlaces_locales=True)
            assert unido == origen, f"{publicado.name}: difieren páginas, imágenes, enlaces, capítulos o texto"
            imagenes = sum(i[0] == "image" for i in unido[1])
            print(f"MANU-PDF-OK: {publicado.name}; {unido[0]} páginas, {imagenes} imágenes, {unido[2]} enlaces internos; texto y capítulos iguales a {rama}.")


if __name__ == "__main__":
    main()
