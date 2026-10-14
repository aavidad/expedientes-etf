"""Carga derivaciones inventadas mediante los mismos puertos HTTP que la app."""

from menores import completar_campos, personas as menores_demo
import datetime
import json
import tempfile
import uuid
import zipfile
from zoneinfo import ZoneInfo
from pathlib import Path
from xml.sax.saxutils import escape


def sembrar_bandeja(raiz, local, url, sesion, ejecutar):
    claves = json.loads((local / "usuarios.json").read_text())
    with tempfile.TemporaryDirectory() as temporal:
        carpeta = Path(temporal)
        coordinacion = sesion(carpeta, "ines.moreno")
        coordinacion.entrar("ines.moreno", claves)
        bandeja = coordinacion.pedir("GET", "/api/coordinacion/bandeja?limite=1")
        if bandeja["total"]:
            coordinacion.pedir("DELETE", "/api/sesion")
            return
        admin = sesion(carpeta, "elena.salas")
        admin.entrar("elena.salas", claves)
        hoy = datetime.datetime.now(ZoneInfo("Europe/Madrid")).date().isoformat()
        catalogo = admin.pedir("GET", "/api/admin/catalogos/asignacion_municipios/datos?fecha=" + hoy)
        admin.pedir("DELETE", "/api/sesion")
        textos = json.loads((raiz / "idiomas/es/catalogos/ubicaciones.json").read_text())
        centros = {c["id"]: c for c in catalogo["centros"]}
        municipios_por_equipo = {}
        for municipio in sorted(catalogo["municipios"], key=lambda m: m["ine"]):
            if municipio["tipo"] == "municipio" and municipio["centro_ss_id"] and municipio["equipo_id"]:
                municipios_por_equipo.setdefault(municipio["equipo_id"], municipio)
        municipios = list(municipios_por_equipo.values())
        anexo = raiz / "internal/adaptadores/anexos/docx/testdata/anexo3.docx"
        personas = json.loads((raiz / "despliegue/demo/bandeja.json").read_text())
        for indice, persona in enumerate(personas):
            municipio = municipios[indice % len(municipios)]
            nombre_municipio = escape(textos[municipio["clave_texto"]])
            nombre_centro = escape(textos[centros[municipio["centro_ss_id"]]["clave_texto"]])
            fichero = carpeta / (str(indice) + ".docx")
            with zipfile.ZipFile(anexo) as origen, zipfile.ZipFile(fichero, "w", zipfile.ZIP_DEFLATED) as destino:
                for info in origen.infolist():
                    contenido = origen.read(info.filename)
                    if info.filename == "word/document.xml":
                        contenido = (contenido.decode()
                                     .replace("Lucía", escape(persona["nombre"]))
                                     .replace("Ortega Salcedo", escape(persona["apellidos"]))
                                     .replace("Servicios Sociales de Alhama", "Servicios Sociales de " + nombre_centro)
                                     .replace("18120 Alhama de Granada", nombre_municipio)
                                     .replace("Alhama de Granada", nombre_municipio)
                                     .replace("Alhama", nombre_municipio).encode())
                    destino.writestr(info.filename, contenido)
            config = ["url = " + json.dumps(url + "/api/coordinacion/derivaciones"),
                      "request = \"POST\"", "cookie = " + json.dumps(str(coordinacion.cookies)),
                      "header = " + json.dumps("Origin: " + url),
                      "header = " + json.dumps("X-CSRF-Token: " + coordinacion.csrf)]
            respuesta = ejecutar(["curl", "--silent", "--show-error", "--max-time", "30", "--noproxy", "*",
                                  "--config", "-", "--form", "anexo=@" + str(fichero),
                                  "--write-out", "\n%{http_code}"], input="\n".join(config), capture_output=True, text=True).stdout
            cuerpo, estado = respuesta.rsplit("\n", 1)
            if estado != "200":
                raise RuntimeError(f"El análisis de derivaciones de demostración ha fallado (HTTP {estado}).")
            analisis = json.loads(cuerpo)
            campos = [{**c, "dudoso": False, "motivos": []}
                      for c in analisis["derivacion_datos"]["contenido"]["campos"]]
            completar_campos(campos, persona["apellidos"], indice % 10)
            derivacion = coordinacion.pedir("POST", "/api/coordinacion/derivaciones/confirmar", {
                "lectura_id": analisis["lectura_anexo"]["id"],
                "tipo": analisis["derivacion_datos"]["contenido"]["tipo"],
                "campos": campos,
                "ubicacion": {"municipio_id": municipio["id"], "centro_id": municipio["centro_ss_id"],
                              "equipo_id": municipio["equipo_id"]},
            }, esperado=201, clave=str(uuid.uuid4()))
            derivacion = coordinacion.pedir("GET", "/api/coordinacion/derivaciones/" + derivacion["id"])

            ruta = "/api/coordinacion/derivaciones/" + derivacion["id"]
            if persona["estado"] == "lista_espera":
                coordinacion.pedir("POST", ruta + "/lista-espera", {"version": 1}, clave=str(uuid.uuid4()))
            elif persona["estado"] == "no_aceptada":
                coordinacion.pedir("POST", ruta + "/no-aceptar", {"version": 1, "fecha": hoy, "motivo": persona["motivo"], "subprograma": "RIESGO_SOCIAL"}, clave=str(uuid.uuid4()))
            elif persona["estado"] == "aceptada":
                menores = [c for c in campos if c["codigo"] == "menores.persona.nombre"]
                coordinacion.pedir("POST", ruta + "/aceptar", {
                    "version": 1, "fecha_acta": hoy, "municipio_id": municipio["id"],
                    "centro_ss_id": municipio["centro_ss_id"], "equipo_id": municipio["equipo_id"],
                    "subprograma": "RIESGO_SOCIAL", "profesional_referencia_puesto": "TS", "profesional_referencia_id": None,
                    "menores": [{"orden": i + 1, "sexo": menores_demo()[(i + indice % 10) % len(menores_demo())]["sexo"]} for i in range(len(menores))],
                    "descripcion_situacion": next(c["valor"] for c in campos if c["codigo"] == "valoracion.descripcion"),
                }, clave=str(uuid.uuid4()))
        coordinacion.pedir("DELETE", "/api/sesion")
