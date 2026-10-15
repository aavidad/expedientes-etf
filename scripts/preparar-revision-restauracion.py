#!/usr/bin/env python3
"""Genera el paso SQL obligatorio a partir de la lista custodiada por sistemas.

La salida se conecta a psql con ON_ERROR_STOP; nunca se escribe en un registro.
La aplicación y su entrada deben permanecer paradas durante el procedimiento.
"""
import argparse
import base64
import datetime
import json
import os
from pathlib import Path
import stat


def sentencia(referencia, fecha_copia):
    fecha = datetime.datetime.fromisoformat(fecha_copia.replace("Z", "+00:00"))
    if fecha.tzinfo is None:
        raise ValueError("La fecha de la copia debe incluir zona horaria.")
    datos = json.loads(referencia)
    if not isinstance(datos, list) or not datos:
        raise ValueError("La referencia debe ser una lista completa de personas.")
    carga = base64.b64encode(json.dumps(datos).encode()).decode("ascii")
    return ("BEGIN;\nSELECT identidad.iniciar_revision_restauracion('" + fecha.isoformat()
            + "'::timestamptz, convert_from(decode('" + carga
            + "','base64'),'UTF8')::jsonb);\nCOMMIT;\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--referencia", required=True, type=Path)
    parser.add_argument("--fecha-copia", required=True)
    args = parser.parse_args()
    info = args.referencia.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise ValueError("La referencia debe ser un fichero propio, sin permisos para otras personas.")
    if info.st_size > 2 * 1024 * 1024:
        raise ValueError("La referencia supera el tamaño permitido.")
    print(sentencia(args.referencia.read_text(), args.fecha_copia), end="")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError):
        raise SystemExit("No se ha preparado la revisión. Comprueba la referencia y la fecha de la copia; mantén la aplicación parada.")
