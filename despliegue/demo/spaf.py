"""Carga casos inventados para comprobar altas y bajas de septiembre de 2026."""

import json
import tempfile
import uuid
from pathlib import Path


def sembrar_spaf(raiz, local, url, sesion):
    marca = local / "spaf-sembrado"
    if marca.exists():
        return
    claves = json.loads((local / "usuarios.json").read_text())
    with tempfile.TemporaryDirectory() as temporal:
        carpeta = Path(temporal)
        admin = sesion(carpeta, "spaf-admin")
        admin.entrar("elena.salas", claves)
        catalogo = admin.pedir("GET", "/api/admin/catalogos/asignacion_municipios/datos?fecha=2026-09-01")
        admin.pedir("DELETE", "/api/sesion")
        equipos = {e["codigo"]: e["id"] for e in catalogo["equipos"]}
        coordinacion = sesion(carpeta, "spaf-coordinacion")
        coordinacion.entrar("ines.moreno", claves)
        casos = [
            ("02", "81", "Nerea", "Romero Cano", "2026-09-10", ""),
            ("02", "82", "Mateo", "Vargas Molina", "2026-08-12", "2026-09-18"),
            ("02", "83", "Adriana", "López Rueda", "2026-09-21", "2026-09-30"),
            ("01", "81", "Daniel", "Castro Nieto", "2026-09-07", ""),
        ]
        for codigo, correlativo, nombre, apellidos, apertura, cierre in casos:
            equipo = equipos[codigo]
            municipio = next(m for m in catalogo["municipios"] if m["equipo_id"] == equipo and m["centro_ss_id"])
            numero = f"{codigo}-{correlativo}-26"
            episodio = {"apertura_fecha": apertura}
            if cierre:
                episodio.update(cierre_fecha=cierre, motivo_cierre="OBJ_TOTAL")
            sexo = "H" if nombre in ("Mateo", "Daniel") else "M"
            menor = {"id": str(uuid.uuid5(uuid.NAMESPACE_URL, url + "/spaf/menor/" + numero)), "orden": 1,
                     "nombre": nombre, "apellidos": apellidos, "sexo": sexo, "fecha_nacimiento": "2016-04-12",
                     "nivel_escolarizacion": "PRIMARIA", "centro_educativo": "Colegio Sierra de Granada",
                     "subprograma": "RIESGO_SOCIAL", "propuesto_atencion_temprana": False}
            caso = {
                "numero": numero, "fecha_derivacion": apertura, "estado": "cerrada" if cierre else "aceptada",
                "derivacion_datos": {"esquema_version": "1", "contenido": {"tipo": "anexo3", "campos": [
                    {"codigo": "menores.persona.nombre", "valor": nombre},
                    {"codigo": "familia.contacto.apellidos", "valor": apellidos},
                ]}},
                "aceptacion": {"version": 1, "fecha_acta": apertura, "municipio_id": municipio["id"],
                               "centro_ss_id": municipio["centro_ss_id"], "equipo_id": equipo,
                               "subprograma": "RIESGO_SOCIAL", "profesional_referencia_puesto": "TS",
                               "profesional_referencia_id": None, "menores": [{"orden": 1, "sexo": sexo}],
                               "descripcion_situacion": "La familia necesita apoyo para organizar los cuidados y la asistencia escolar."},
                "episodios": [episodio],
                "menores": [{"version": 1, "vigente_desde": apertura, "menores": [menor]}],
            }
            # La misma clave recupera el alta si se interrumpió la siembra.
            clave = str(uuid.uuid5(uuid.NAMESPACE_URL, url + "/spaf/" + numero))
            coordinacion.pedir("POST", "/api/expedientes/historicos", {"equipo_id": equipo, "casos": [caso]}, clave=clave)
        coordinacion.pedir("DELETE", "/api/sesion")
    marca.write_text("SPAF septiembre de 2026: ETF 02, 2 altas, 2 bajas y 3 casos; ETF 01, 1 alta.\n")
