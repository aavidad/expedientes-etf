"""Añade archivos de reserva sin sustituir los ya publicados."""
import os
from pathlib import Path
import shutil
import tempfile


def incorporar(origen, destino, uid, gid):
    destino.mkdir(parents=True, exist_ok=True)
    os.chown(destino, uid, gid)
    destino.chmod(0o755)
    for ruta in sorted(origen.rglob("*")):
        nueva = destino / ruta.relative_to(origen)
        if nueva.exists():
            continue
        if ruta.is_dir():
            nueva.mkdir()
            os.chown(nueva, uid, gid)
            nueva.chmod(0o755)
        else:
            # Un corte durante la copia no debe dejar una versión parcial
            # que la siguiente actualización confunda con una publicada.
            with tempfile.NamedTemporaryFile(dir=nueva.parent, prefix=".win1-", delete=False) as temporal:
                try:
                    shutil.copyfile(ruta, temporal.name)
                    os.chown(temporal.name, uid, gid)
                    os.chmod(temporal.name, 0o644)
                    os.fsync(temporal.fileno())
                    try:
                        os.link(temporal.name, nueva)
                    except FileExistsError:
                        pass
                finally:
                    Path(temporal.name).unlink(missing_ok=True)
