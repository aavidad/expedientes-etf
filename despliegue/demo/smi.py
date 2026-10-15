"""Prepara los importes oficiales del SMI en la demostración por su API."""
import json
import tempfile
from pathlib import Path


def sembrar_smi(demo):
    claves = json.loads((demo.LOCAL / "usuarios.json").read_text())
    datos = json.loads((demo.RAIZ / "datos/catalogos/smi.json").read_text())
    with tempfile.TemporaryDirectory() as temporal:
        sesion = demo.Sesion(Path(temporal), "smi-coordinacion")
        sesion.entrar("ines.moreno", claves)
        filas = sesion.pedir("GET", "/api/coordinacion/smi")
        for importe in datos["importes"]:
            fila = next(f for f in filas if f["anio"] == importe["anio"])
            # Una modificación de Coordinación conserva siempre su importe.
            if fila["importe_mensual_centimos"] is None and fila["version"] == 0:
                sesion.pedir("PUT", "/api/coordinacion/smi", {
                    "anio": importe["anio"],
                    "version_esperada": fila["version"],
                    "importe_mensual": format(importe["importe_mensual_centimos"] / 100, ".2f"),
                })
        sesion.pedir("DELETE", "/api/sesion")
