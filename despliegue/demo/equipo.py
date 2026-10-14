"""Añade expedientes inventados de equipo mediante las API de la demostración."""
from menores import completar_campos, personas
import json
import tempfile
import datetime
from zoneinfo import ZoneInfo
import uuid
import zipfile
from pathlib import Path


def sembrar_equipo(demo):
    if (demo.LOCAL / "equipo-sembrado").exists():
        return
    hoy = datetime.datetime.now(ZoneInfo("Europe/Madrid")).date().isoformat()
    claves = json.loads((demo.LOCAL / "usuarios.json").read_text())
    with tempfile.TemporaryDirectory(dir=demo.LOCAL) as temporal:
        carpeta = Path(temporal)
        admin = demo.Sesion(carpeta, "elena.salas")
        admin.entrar("elena.salas", claves)
        catalogo = admin.pedir("GET", "/api/admin/catalogos/asignacion_municipios/datos?fecha=" + hoy)
        usuarios = admin.pedir("GET", "/api/admin/usuarios")
        admin.pedir("DELETE", "/api/sesion")
        coordinacion = demo.Sesion(carpeta, "ines.moreno")
        coordinacion.entrar("ines.moreno", claves)
        profesional = demo.Sesion(carpeta, "marta.cabrera")
        profesional.entrar("marta.cabrera", claves)
        existentes = profesional.pedir("GET", "/api/equipo/expedientes")
        if existentes["filas"]:
            demo.guardar(demo.LOCAL / "equipo-sembrado", "El equipo ya tiene expedientes aceptados.\n")
            return
        for indice, (codigo_equipo, apellidos) in enumerate((("02", "Romero Vega"), ("02", "Molina Torres"), ("04", "Serrano Rivas"))):
            equipo = next(e for e in catalogo["equipos"] if e["codigo"] == codigo_equipo)
            municipio = next(m for m in catalogo["municipios"] if m["equipo_id"] == equipo["id"] and m["centro_ss_id"])
            referente = next(u for u in usuarios if (u.get("equipo") or {}).get("equipo_id") == equipo["id"])
            anexo = carpeta / "anexo3.docx"
            with zipfile.ZipFile(demo.RAIZ / "internal/adaptadores/anexos/docx/testdata/anexo3.docx") as original, zipfile.ZipFile(anexo, "w") as destino:
                for info in original.infolist():
                    contenido = original.read(info.filename)
                    if info.filename == "word/document.xml":
                        primero, segundo = apellidos.split()
                        contenido = contenido.replace(b"Ortega", primero.encode()).replace(b"Salas", segundo.encode()).replace(b"Salcedo", segundo.encode())
                    destino.writestr(info, contenido)
            derivacion = coordinacion.pedir("POST", "/api/coordinacion/derivaciones", fichero=anexo)
            completar_campos(derivacion["derivacion_datos"]["contenido"]["campos"], apellidos, indice * 3)
            recibo = coordinacion.pedir("POST", "/api/coordinacion/derivaciones/confirmar", {
                "lectura_id": derivacion["lectura_anexo"]["id"], "tipo": "anexo3",
                "campos": derivacion["derivacion_datos"]["contenido"]["campos"],
                "ubicacion": {"municipio_id": municipio["id"], "centro_id": municipio["centro_ss_id"], "equipo_id": equipo["id"]},
            }, clave=str(uuid.uuid4()), esperado=201)
            derivacion["id"] = recibo["id"]
            derivacion["version"] = recibo["version"]
            menores = [c for c in derivacion["derivacion_datos"]["contenido"]["campos"] if c["codigo"] == "menores.persona.nombre"]
            aceptada = coordinacion.pedir("POST", f"/api/coordinacion/derivaciones/{derivacion['id']}/aceptar", clave=str(uuid.uuid4()), datos={
                "version": derivacion["version"], "fecha_acta": hoy, "municipio_id": municipio["id"],
                "centro_ss_id": municipio["centro_ss_id"], "equipo_id": equipo["id"], "subprograma": "RIESGO_SOCIAL",
                "profesional_referencia_puesto": "TS", "profesional_referencia_id": None,
                "menores": [{"orden": i + 1, "sexo": personas()[(i + indice * 3) % len(personas())]["sexo"]} for i in range(len(menores))],
                "descripcion_situacion": "La familia necesita apoyo para organizar los cuidados y mantener la asistencia escolar.",
            })
            if indice == 0:
                final = profesional.pedir("PATCH", f"/api/expedientes/{derivacion['id']}/finalizacion", clave=str(uuid.uuid4()), datos={
                    "version": aceptada["version"], "motivo": "SIN_CONSENTIMIENTO", "motivo_texto": "", "sin_consentimiento": True,
                    "derivacion_tras_cierre": "Seguimiento por Servicios Sociales Comunitarios.", "coordinacion_recursos": "Reunión con el servicio derivante.",
                    "objetivos_abordados": "Contacto inicial con la familia.", "objetivos_pendientes": "La familia no ha consentido la intervención.",
                })
                profesional.pedir("POST", f"/api/expedientes/{derivacion['id']}/proponer-cierre", clave=str(uuid.uuid4()), datos={"version": final["version"]})
        profesional.pedir("DELETE", "/api/sesion")
        coordinacion.pedir("DELETE", "/api/sesion")
    demo.guardar(demo.LOCAL / "equipo-sembrado", "Expedientes inventados confirmados por las API.\n")
