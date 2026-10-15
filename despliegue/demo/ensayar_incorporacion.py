#!/usr/bin/env python3
"""Ensaya una actualización de recursos en una demo desechable, sin limpiar."""
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import sys
import tempfile
import uuid
import zipfile

RAIZ = Path(__file__).resolve().parents[2]


def ensayar():
    with tempfile.TemporaryDirectory(prefix="fx7-") as temporal:
        copia = Path(temporal) / "arbol"
        copia.mkdir()
        # Incluye los cambios de la candidata, sin secretos ni artefactos locales.
        nombres = subprocess.run(["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"],
                                 cwd=RAIZ, check=True, capture_output=True).stdout.split(b"\0")
        for nombre in filter(None, nombres):
            relativa = Path(os.fsdecode(nombre))
            fuente = RAIZ / relativa
            if fuente.is_file():
                destino = copia / relativa
                destino.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(fuente, destino)
        # La copia no tiene Git: el inventario de versiones se toma del repositorio.
        os.environ["ETF_REPOSITORIO_GIT"] = str(RAIZ)
        with socket.socket() as puerto:
            puerto.bind(("127.0.0.1", 0))
            os.environ["ETF_DEMO_PUERTO"] = str(puerto.getsockname()[1])
        spec = importlib.util.spec_from_file_location("demo_fx7", copia / "despliegue/demo/demo.py")
        demo = importlib.util.module_from_spec(spec)
        sys.modules[spec.name] = demo
        spec.loader.exec_module(demo)
        sys.path.insert(0, str(demo.DEMO))
        log = Path(temporal) / "demo.log"

        def arrancar():
            with log.open("ab") as salida:
                subprocess.run(["python3", "-B", str(demo.DEMO / "demo.py"), "iniciar"],
                               cwd=copia, check=True, stdout=salida, stderr=subprocess.STDOUT, timeout=600)

        def sql(texto):
            return demo.compose("exec", "-T", "postgres", "psql", "-X", "-U", "etf_instalador",
                                "-d", "etf", "-v", "ON_ERROR_STOP=1", "-Atq", input=texto,
                                capture_output=True, text=True).stdout.strip()

        def emitir(sesion, expediente, esperado):
            datos = json.dumps({"tipo": "entrada/aceptacion", "expediente_id": expediente, "idioma": "es"})
            config = {"url": demo.URL + "/api/documentos/emisiones", "request": "POST",
                      "cookie": str(sesion.cookies), "header": "Origin: " + demo.URL}
            lineas = [f"{k} = {json.dumps(v)}" for k, v in config.items()]
            lineas += ["header = " + json.dumps("X-CSRF-Token: " + sesion.csrf),
                       "header = " + json.dumps("Idempotency-Key: " + str(uuid.uuid4())),
                       'header = "Content-Type: application/json"', "data = " + json.dumps(datos)]
            respuesta = demo.ejecutar(["curl", "--silent", "--show-error", "--max-time", "30",
                                       "--noproxy", "*", "--config", "-", "--write-out", "\n%{http_code}"],
                                      input="\n".join(lineas).encode(), capture_output=True).stdout
            cuerpo, estado = respuesta.rsplit(b"\n", 1)
            assert int(estado) == esperado, f"Emisión HTTP {estado.decode()}; esperado {esperado}"
            if esperado == 422:
                assert json.loads(cuerpo)["error"]["clave"] == "documento.plantilla_no_disponible", cuerpo
            else:
                with zipfile.ZipFile(io.BytesIO(cuerpo)) as documento:
                    assert documento.testzip() is None
                    assert b"{{" not in documento.read("word/document.xml")

        def app(*orden, entrada=None):
            return demo.compose("exec", "-T", "app", *orden, input=entrada, capture_output=True).stdout

        def word_consulta(sesion, anio):
            config = {"url": demo.URL + f"/api/consulta/memoria/{anio}/word", "request": "POST",
                      "cookie": str(sesion.cookies), "header": "Origin: " + demo.URL}
            lineas = [f"{k} = {json.dumps(v)}" for k, v in config.items()]
            lineas += ["header = " + json.dumps("X-CSRF-Token: " + sesion.csrf),
                       "header = " + json.dumps("Idempotency-Key: " + str(uuid.uuid4())),
                       'header = "Content-Type: application/json"',
                       "data = " + json.dumps(json.dumps({"idioma": "es"}))]
            respuesta = demo.ejecutar(["curl", "--silent", "--show-error", "--max-time", "30",
                                       "--noproxy", "*", "--config", "-", "--write-out", "\n%{http_code}"],
                                      input="\n".join(lineas).encode(), capture_output=True).stdout
            cuerpo, estado = respuesta.rsplit(b"\n", 1)
            return int(estado), cuerpo

        def huellas_publicadas():
            # Todo lo publicado por Administración cuelga de alguna carpeta .versiones.
            salida = app("sh", "-c", "cd /datos && find idiomas temas -path '*/.versiones/*' -type f "
                                     "-exec sha256sum {} + | sort")
            return salida.decode()

        try:
            arrancar()
            claves = json.loads((demo.LOCAL / "usuarios.json").read_text())
            coordinacion = demo.Sesion(Path(temporal), "ines.moreno")
            coordinacion.entrar("ines.moreno", claves)
            expediente = sql("SELECT id FROM expediente.expediente WHERE estado='aceptada' LIMIT 1;")
            ruta = sql("SELECT p.ruta_relativa FROM documento.plantilla p JOIN documento.referencia_activa r "
                       "ON r.plantilla_id=p.id WHERE p.tipo_documento='entrada/aceptacion' AND p.idioma='es';")
            origen = copia / "idiomas/es" / ruta
            nueva = origen.with_name("v99.7.0.docx")
            shutil.copy2(origen, nueva)
            relativa = nueva.relative_to(copia / "idiomas/es").as_posix()
            # Simula la referencia que una actualización instala antes de emitir.
            sql("BEGIN; INSERT INTO documento.plantilla "
                "(id,tipo_documento,idioma,version,ruta_relativa,huella_sha256,rol_firmante,vigente_desde,comprobacion) "
                "SELECT '99999999-9999-4999-8999-999999999997',p.tipo_documento,p.idioma,'99.7.0','" + relativa + "',"
                "p.huella_sha256,p.rol_firmante,p.vigente_desde,p.comprobacion "
                "FROM documento.plantilla p JOIN documento.referencia_activa r ON r.plantilla_id=p.id "
                "WHERE p.tipo_documento='entrada/aceptacion' AND p.idioma='es'; "
                "UPDATE documento.referencia_activa SET plantilla_id='99999999-9999-4999-8999-999999999997',version=version+1 "
                "WHERE tipo_documento='entrada/aceptacion' AND idioma='es'; COMMIT;")
            emitir(coordinacion, expediente, 422)
            # Demo instalada antes de SEG1 con el incorporador antiguo: sin
            # manifiesto y con el fichero de textos de entonces, sin la clave nueva.
            consulta = demo.Sesion(Path(temporal), "lucia.navarro")
            consulta.entrar("lucia.navarro", claves)
            estado, cuerpo = word_consulta(consulta, 2026)
            assert estado == 200, f"Word de Consulta HTTP {estado} en instalación limpia"
            textos = "/datos/idiomas/es/textos/consulta_memoria.json"
            actuales = json.loads(app("cat", textos))
            nueva_clave = "estadistica.consulta_oculto"
            assert actuales.get(nueva_clave), "la distribución no trae la clave nueva"
            git = ["git", "-C", str(RAIZ)]
            seg1 = subprocess.run([*git, "log", "--format=%H", "-S", nueva_clave, "--reverse", "--",
                                   "idiomas/es/textos/consulta_memoria.json"],
                                  check=True, capture_output=True, text=True).stdout.split()[0]
            viejos = subprocess.run([*git, "show", seg1 + "^:idiomas/es/textos/consulta_memoria.json"],
                                    check=True, capture_output=True).stdout
            assert nueva_clave.encode() not in viejos
            app("sh", "-c", "cat > " + textos + " && rm -f /datos/idiomas/.distribucion.json", entrada=viejos)
            # Versión de idioma publicada y activa, anterior a la clave nueva y
            # a la plantilla nueva, y un catálogo publicado: no se pueden tocar.
            publicada = "/datos/idiomas/.versiones/es/9.0.0/" + "a" * 64 + "/" + "b" * 32 + "/es"
            app("sh", "-c", f"mkdir -p {publicada} && cp -R /datos/idiomas/es/. {publicada}/ "
                            "&& mkdir -p /datos/idiomas/es/catalogos/.versiones "
                            "&& printf '{\"centro.ALHAMA\":\"Centro comarcal\"}' > "
                            "/datos/idiomas/es/catalogos/.versiones/" + "c" * 64 + ".json")
            fecha_publicada = "Consulta publicada 9.0.0"
            textos_publicados = json.loads(viejos)
            textos_publicados["estadistica.consulta_fecha"] = fecha_publicada
            app("sh", "-c", f"cat > {publicada}/textos/consulta_memoria.json",
                entrada=json.dumps(textos_publicados, ensure_ascii=False).encode())
            sql("BEGIN; INSERT INTO paquete.idioma_instalado (id,codigo,version,ruta_relativa,huella_sha256,registrado_por) "
                "VALUES ('99999999-9999-4999-8999-999999999990','es','9.0.0','" + publicada[len("/datos/idiomas/"):]
                + "','" + "a" * 64 + "',gen_random_uuid()); "
                "INSERT INTO paquete.referencia_activa (tipo,codigo,version_contenido,version,idioma_id,activada_por) "
                "VALUES ('idioma','es','9.0.0',1,'99999999-9999-4999-8999-999999999990',gen_random_uuid()); COMMIT;")
            antes = huellas_publicadas()
            assert publicada[len("/datos/"):] + "/textos/consulta_memoria.json" in antes
            estado, cuerpo = word_consulta(consulta, 2026)
            assert estado == 503, f"Word de Consulta HTTP {estado}; se esperaba 503 con textos viejos"
            demo.compose("exec", "-T", "app", "sh", "-c", "printf conservado > /datos/temas/nota-local")
            credenciales = (demo.LOCAL / "entorno.env").read_bytes()
            arrancar()
            assert (demo.LOCAL / "entorno.env").read_bytes() == credenciales
            assert json.loads(app("cat", textos)) == actuales, "la preparación no actualizó los textos"
            assert huellas_publicadas() == antes, "la preparación tocó versiones publicadas"
            assert app("cat", "/datos/temas/nota-local") == b"conservado"
            assert app("cat", "/datos/idiomas/es/" + relativa) == nueva.read_bytes()
            servidos = json.loads(demo.ejecutar(["curl", "--silent", "--show-error", "--max-time", "15",
                                                 "--noproxy", "*", demo.URL + "/api/idiomas/es/textos"],
                                                capture_output=True).stdout)
            servidos = servidos.get("textos", servidos)
            assert servidos.get(nueva_clave) == actuales[nueva_clave], "texto nuevo no servido a la web"
            consulta.entrar("lucia.navarro", claves)
            estado, cuerpo = word_consulta(consulta, 2026)
            assert estado == 200, f"Word de Consulta HTTP {estado} tras actualizar"
            with zipfile.ZipFile(io.BytesIO(cuerpo)) as documento:
                assert documento.testzip() is None
                xml = documento.read("word/document.xml")
                assert b"{{" not in xml
                # Manda la versión activa; la clave que le falta llega de la distribución.
                assert fecha_publicada.encode() in xml, "no se usó el texto de la versión activa"
                assert actuales[nueva_clave].encode() in xml, "falta el texto nuevo de la distribución"
            coordinacion.entrar("ines.moreno", claves)
            emitir(coordinacion, expediente, 201)
            print("FX7-OK: demo creada; plantilla nueva; 422 antes; textos viejos: Word de Consulta 503; "
                  "reinicio sin limpiar; textos nuevos servidos; Word de Consulta 200; emisión 201 Word válido; "
                  "versiones publicadas, notas y credenciales intactas.")
        except Exception:
            detalle = log.read_text(errors="replace") if log.exists() else ""
            if (demo.LOCAL / "entorno.env").exists():
                for linea in (demo.LOCAL / "entorno.env").read_text().splitlines():
                    clave, _, valor = linea.partition("=")
                    if "CONTRASENA" in clave and valor:
                        detalle = detalle.replace(valor, "[oculto]")
            if (demo.LOCAL / "usuarios.json").exists():
                for valor in json.loads((demo.LOCAL / "usuarios.json").read_text()).values():
                    detalle = detalle.replace(valor, "[oculto]")
            print("\n".join(detalle.splitlines()[-20:]), file=sys.stderr)
            raise
        finally:
            if (demo.LOCAL / "entorno.env").exists():
                demo.compose("down", "--volumes", "--remove-orphans", stdout=subprocess.DEVNULL,
                             stderr=subprocess.DEVNULL)
            sys.path.remove(str(demo.DEMO))


if __name__ == "__main__":
    ensayar()
