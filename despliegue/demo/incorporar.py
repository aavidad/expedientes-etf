"""Instala los recursos de la distribución sin tocar versiones publicadas."""
import json
from pathlib import Path
from archivos import incorporar, validar_textos


def preparar(origen=Path("/reserva"), destino=Path("/datos"), uid=10001, gid=10001,
             inventario=Path("/opt/etf/inventario.json")):
    datos = json.loads(inventario.read_text(encoding="utf-8"))
    for nombre in ("idiomas", "temas"):
        incorporar(origen / nombre, destino / nombre, uid, gid, datos)
    # Si los textos resultantes no cargan, la demo no arranca.
    validar_textos(destino / "idiomas")


if __name__ == "__main__":
    preparar()
