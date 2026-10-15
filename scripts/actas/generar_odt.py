#!/usr/bin/env python3
"""Convierte las actas inventadas de las plantillas oficiales a ODT sanitizado."""
import os
from pathlib import Path
import subprocess
import tempfile
from zipfile import ZIP_DEFLATED, ZIP_STORED, ZipFile, ZipInfo

RAIZ = Path(__file__).resolve().parents[2]
DESTINO = RAIZ / "internal/adaptadores/anexos/actas/testdata"


def main():
    nombres = ["aceptacion-2022", "aceptacion-2023", "cierre-2026"]
    with tempfile.TemporaryDirectory(prefix="resp5-actas-") as temporal:
        tmp = Path(temporal)
        entorno = dict(os.environ)
        entorno.pop("LD_LIBRARY_PATH", None)
        subprocess.run([
            "libreoffice", "-env:UserInstallation=" + (tmp / "lo").as_uri(),
            "--headless", "--convert-to", "odt", "--outdir", str(tmp),
            *[str(DESTINO / (nombre + ".docx")) for nombre in nombres],
        ], check=True, env=entorno, timeout=90)
        for nombre in nombres:
            with ZipFile(tmp / (nombre + ".odt")) as origen:
                partes = [(i.filename, origen.read(i)) for i in origen.infolist()]
            with ZipFile(DESTINO / (nombre + ".odt"), "w") as salida:
                for ruta, contenido in partes:
                    if ruta == "meta.xml":
                        contenido = (
                            b'<?xml version="1.0" encoding="UTF-8"?>'
                            b'<office:document-meta xmlns:office="urn:oasis:names:tc:'
                            b'opendocument:xmlns:office:1.0" office:version="1.3">'
                            b'<office:meta/></office:document-meta>'
                        )
                    info = ZipInfo(ruta, (2026, 1, 1, 0, 0, 0))
                    info.compress_type = ZIP_STORED if ruta == "mimetype" else ZIP_DEFLATED
                    info.external_attr = 0o600 << 16
                    salida.writestr(info, contenido)
            print(f"RESP5_FIXTURE: {nombre}.odt; metadatos vacíos y fechas ZIP fijas.")


if __name__ == "__main__":
    main()
