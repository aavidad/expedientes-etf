"""Datos inventados coherentes para la demostración, sin sustituir fichas reales."""
import json
from pathlib import Path


def personas():
    return json.loads(Path(__file__).with_suffix(".json").read_text())


def completar_campos(campos, apellidos, desplazamiento=0):
    menores = personas()
    indices = {}
    correspondencias = {"nombre": "nombre", "nacimiento": "fecha_nacimiento", "escolarizacion": "nivel_escolarizacion"}
    for campo in campos:
        prefijo = "menores.persona."
        if not campo["codigo"].startswith(prefijo):
            continue
        tipo = campo["codigo"][len(prefijo):]
        indice = indices.get(tipo, 0)
        indices[tipo] = indice + 1
        persona = menores[(indice + desplazamiento) % len(menores)]
        if tipo in correspondencias:
            campo["valor"] = persona[correspondencias[tipo]]
        elif tipo == "apellidos":
            campo["valor"] = apellidos
    return campos
