#!/usr/bin/env python3
"""Comprueba que el conjunto de prueba no incluye texto o identificadores reales."""

import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[1]
PATRON_IDENTIDAD = re.compile(r"(?<!\d)(?:\d{8}[ -]?[A-HJ-NP-TV-Z]|[XYZ]\d{7}[A-HJ-NP-TV-Z])(?![A-Z0-9])", re.IGNORECASE)
TERMINO_BLOQUEADO = "eje" + "mplo"


def main():
    errores = []
    for requerido in (RAIZ / "expedientes.json", RAIZ / "LEEME.md"):
        if not requerido.is_file():
            errores.append(f"Falta el fichero requerido: {requerido.name}")
    for archivo in sorted(path for path in RAIZ.rglob("*") if path.is_file()):
        texto = archivo.read_text(encoding="utf-8")
        if TERMINO_BLOQUEADO in texto.casefold():
            errores.append(f"Aparece el término prohibido en {archivo.name}")
        if PATRON_IDENTIDAD.search(texto):
            errores.append(f"Aparece un patrón de DNI/NIE en {archivo.name}")
    if errores:
        print("\n".join(errores), file=sys.stderr)
        raise SystemExit(1)
    print("Privacidad de datos sintéticos: sin término prohibido ni patrones DNI/NIE.")


if __name__ == "__main__":
    main()
