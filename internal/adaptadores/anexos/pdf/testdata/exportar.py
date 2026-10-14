#!/usr/bin/env python3
"""Regenera el corpus PDF desde los Word inventados y sin membretes."""

import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

from PIL import Image


BASE = Path(__file__).resolve().parent
RAIZ = BASE.parents[4]


def ejecutar(*args):
    subprocess.run(args, check=True, capture_output=True, timeout=120)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--salida", type=Path, default=BASE)
    args = parser.parse_args()
    for herramienta in ("soffice", "pdftoppm", "exiftool", "qpdf"):
        if not shutil.which(herramienta):
            parser.error(f"Necesitas {herramienta} local.")
    temporales = Path(os.environ.get("TMPDIR", str(RAIZ / "bin/temporales")))
    temporales.mkdir(parents=True, exist_ok=True)
    args.salida.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="pdf-corpus-", dir=temporales) as temporal:
        carpeta = Path(temporal)
        ejecutar("env", "-u", "LD_LIBRARY_PATH", "soffice",
                 f"-env:UserInstallation={(carpeta / 'perfil').as_uri()}",
                 "--headless", "--convert-to", "pdf", "--outdir", str(carpeta),
                 str(BASE / "anexo3.docx"), str(BASE / "anexo4.docx"))
        # Una página rasterizada conserva el caso de un escaneado sin texto.
        ejecutar("pdftoppm", "-f", "1", "-singlefile", "-scale-to", "1000",
                 "-png", str(carpeta / "anexo3.pdf"), str(carpeta / "pagina"))
        with Image.open(carpeta / "pagina.png") as imagen:
            imagen.convert("RGB").save(carpeta / "sin_texto.pdf", resolution=100)
        for nombre in ("anexo3", "anexo4", "sin_texto"):
            pdf = carpeta / f"{nombre}.pdf"
            ejecutar("exiftool", "-overwrite_original", "-all=",
                     "-Author=Expedientes ETF", "-Creator=Expedientes ETF",
                     "-Producer=Expedientes ETF", str(pdf))
            # El lector usa tablas clásicas; no generar objetos comprimidos.
            ejecutar("qpdf", "--object-streams=disable", str(pdf),
                     str(args.salida / pdf.name))
            print(f"PDF-CORPUS-OK: {pdf.name}")


if __name__ == "__main__":
    main()
