"""Completa la ficha de los expedientes inventados del ETF 02 por sus API."""
from menores import personas
import json
import tempfile
import uuid


def sembrar_fi1(demo):
    marca = demo.LOCAL / "ui3-fi1-sembrado"
    if marca.exists():
        return
    claves = json.loads((demo.LOCAL / "usuarios.json").read_text())
    with tempfile.TemporaryDirectory(dir=demo.LOCAL) as temporal:
        from pathlib import Path
        for cuenta in ("marta.cabrera", "pablo.ruiz"):
            profesional = demo.Sesion(Path(temporal), cuenta)
            profesional.entrar(cuenta, claves)
            expedientes = profesional.pedir("GET", "/api/equipo/expedientes")["filas"]
            for expediente in expedientes:
                id_exp = expediente["id"]
                ficha = profesional.pedir("GET", f"/api/expedientes/{id_exp}/menores")
                if ficha["estado"] == "cerrada":
                    continue
                menores = ficha["menores"]
                nombres = set()
                for menor in menores:
                    disponibles = [p for p in personas() if p["nombre"] not in nombres]
                    persona = next((p for p in disponibles if p["nombre"] == menor["nombre"]), disponibles[0])
                    nombres.add(persona["nombre"])
                    menor.update(persona)
                    menor.update({"apellidos": menor["apellidos"] or "Romero Vega", "propuesto_atencion_temprana": False,
                                  "centro_educativo": "" if persona["nivel_escolarizacion"] == "SIN_ESCOLARIZAR" else ("Instituto Vega de Granada" if persona["nivel_escolarizacion"] in ("ESO", "BACHILLERATO", "FP") else "Colegio Sierra de Granada")})
                if menores:
                    profesional.pedir("PUT", f"/api/expedientes/{id_exp}/menores", datos={"version": ficha["version"], "vigente_desde": ficha["fecha_aceptacion"][:10], "menores": menores}, clave=str(uuid.uuid4()))
                intervencion = profesional.pedir("GET", f"/api/expedientes/{id_exp}/intervencion")
                profesional.pedir("PUT", f"/api/expedientes/{id_exp}/intervencion", datos={"version": intervencion["version"], "vigente_desde": ficha["fecha_aceptacion"][:10],
                    "intervencion": {"fase": "EVALUACION", "listas": {"coordinacion": [{"codigo": "EDUCACION", "menores_excluidos": []}], "factor_familia": [{"codigo": "ECONOMIA", "menores_excluidos": []}], "factor_menor": [{"codigo": "ABSENTISMO", "menores_excluidos": [m["id"] for m in menores if m["nivel_escolarizacion"] == "SIN_ESCOLARIZAR"]}], "tipologia": [{"codigo": "NEGLIGENCIA", "menores_excluidos": []}]}}}, clave=str(uuid.uuid4()))
            profesional.pedir("DELETE", "/api/sesion")
    demo.guardar(marca, "Datos inventados de Menores e Intervención confirmados por las API.\n")
