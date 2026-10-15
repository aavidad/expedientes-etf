"""Protecciones del paquete frente a credenciales y montajes del servidor."""
import copy
from contextlib import redirect_stderr
import io
import json
import subprocess
import importlib.util
import unittest
from unittest.mock import patch
import os
import re
import tempfile
from pathlib import Path

spec = importlib.util.spec_from_file_location("paquete", Path(__file__).with_name("paquete.py"))
paquete = importlib.util.module_from_spec(spec)
spec.loader.exec_module(paquete)
spec_archivos = importlib.util.spec_from_file_location("archivos", Path(__file__).with_name("archivos.py"))
archivos = importlib.util.module_from_spec(spec_archivos)
spec_archivos.loader.exec_module(archivos)


class ConfiguracionWindows(unittest.TestCase):
    def setUp(self):
        self.base = {
            "services": {nombre: {
                "build": {"context": "/ruta/repositorio"},
                "volumes": [{"type": "bind", "source": "/secretos", "target": "/privado"}],
                "tmpfs": ["/tmp:uid=70,gid=70,mode=0700,size=16m"],
            } for nombre in ("app", "migrar", "ldap", "postgres", "clamav", "entrada-demo", "preparar-roles")},
            "volumes": {"postgres_socket": {"name": "otra-demo_socket", "driver_opts": {"device": "/host"}}},
            "networks": {},
        }
        self.base["services"]["app"]["environment"] = ["ETF_BD_CONTRASENA=${ETF_APP_CONTRASENA:?}"]

    def test_aisla_archivos_secretos_y_volumenes_ajenos(self):
        config = paquete.configurar(copy.deepcopy(self.base), "2026.10.14")
        self.assertEqual(config["name"], "etf-windows")
        self.assertFalse(any("name" in v for v in config["volumes"].values()))
        self.assertEqual(config["volumes"]["postgres_socket"], {})
        for servicio in config["services"].values():
            self.assertNotIn("build", servicio)
            for volumen in servicio["volumes"]:
                if isinstance(volumen, dict):
                    self.assertEqual(volumen["type"], "volume")
        app = config["services"]["app"]
        self.assertEqual(app["volumes"], [{"type": "volume", "source": "ca", "target": "/demo", "read_only": True}])
        self.assertEqual(app["environment"]["ETF_BD_CONTRASENA"], "${ETF_APP_CONTRASENA:?}")
        self.assertNotIn("ETF_MIGRADOR_CONTRASENA", app["environment"])
        self.assertTrue(config["networks"]["interna"]["internal"])

    def test_preparador_linux_no_se_exporta_al_paquete(self):
        self.base["services"]["incorporar"] = {"build": {"target": "incorporar"}, "profiles": ["preparacion"]}
        config = paquete.configurar(copy.deepcopy(self.base), "2026.10.14")
        self.assertNotIn("incorporar", config["services"])
        self.assertTrue(all("image" in servicio for servicio in config["services"].values()))

    def test_motor_compatible_y_puerto_solo_local(self):
        config = paquete.configurar(copy.deepcopy(self.base), "2026.10.14")
        for servicio in config["services"].values():
            for temporal in servicio.get("tmpfs", []):
                self.assertNotIn("uid=", temporal)
                self.assertNotIn("gid=", temporal)
                self.assertIn("mode=1777", temporal)
        puerto = config["services"]["entrada-demo"]["ports"][0]
        self.assertEqual(puerto["host_ip"], "127.0.0.1")
        self.assertEqual(puerto["published"], "${ETF_WINDOWS_PUERTO:-18083}")
        self.assertEqual(config["services"]["preparar-roles"]["image"], config["services"]["postgres"]["image"])
        self.assertEqual(config["services"]["ldap"]["mem_limit"], "512m")


class PublicacionArchivos(unittest.TestCase):
    def preparar(self, temporal):
        raiz = Path(temporal)
        origen, destino = raiz / "origen", raiz / "destino"
        origen.mkdir()
        destino.mkdir()
        return origen, destino

    def incorporar(self, origen, destino, historicas=()):
        inventario = {"entrega": "prueba", "huellas": [archivos.huella(h) for h in historicas]}
        with redirect_stderr(io.StringIO()) as avisos:
            archivos.incorporar(origen, destino, os.getuid(), os.getgid(), inventario)
        return avisos.getvalue()

    def test_sustituye_la_base_y_conserva_lo_publicado(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            (origen / "es/textos").mkdir(parents=True)
            (origen / "es/textos/consulta.json").write_text('{"clave.vieja":"a"}')
            (origen / "es/textos/retirado.json").write_text("{}")
            self.incorporar(origen, destino)
            publicado = destino / "es/catalogos/.versiones/abc.json"
            publicado.parent.mkdir(parents=True)
            publicado.write_text("publicado")
            plantilla = destino / "es/plantillas/acta/.versiones/id/v1.0.0.docx"
            plantilla.parent.mkdir(parents=True)
            plantilla.write_bytes(b"plantilla publicada")
            version = destino / ".versiones/es/1.0.0/x/es/textos/consulta.json"
            version.parent.mkdir(parents=True)
            version.write_text("versión")
            (destino / "es/nota-local").write_text("conservado")
            (origen / "es/textos/consulta.json").write_text('{"clave.vieja":"a","clave.nueva":"b"}')
            (origen / "es/textos/retirado.json").unlink()
            (origen / "es/plantillas/acta").mkdir(parents=True)
            (origen / "es/plantillas/acta/v2.docx").write_bytes(b"plantilla nueva")
            avisos = self.incorporar(origen, destino)
            self.assertEqual((destino / "es/textos/consulta.json").read_text(), '{"clave.vieja":"a","clave.nueva":"b"}')
            self.assertFalse((destino / "es/textos/retirado.json").exists())
            self.assertEqual((destino / "es/plantillas/acta/v2.docx").read_bytes(), b"plantilla nueva")
            self.assertEqual(publicado.read_text(), "publicado")
            self.assertEqual(plantilla.read_bytes(), b"plantilla publicada")
            self.assertEqual(version.read_text(), "versión")
            self.assertEqual((destino / "es/nota-local").read_text(), "conservado")
            self.assertIn("es/nota-local", avisos)
            self.assertEqual([p.name for p in (destino / "es/textos").iterdir()], ["consulta.json"])
            manifiesto = json.loads((destino / archivos.MANIFIESTO).read_text())
            self.assertEqual(manifiesto["entrega"], "prueba")
            self.assertEqual(manifiesto["ficheros"]["es/textos/consulta.json"],
                             archivos.huella(b'{"clave.vieja":"a","clave.nueva":"b"}'))

    def test_conserva_lo_cambiado_a_mano(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            (origen / "a.json").write_text("v1")
            (origen / "b.json").write_text("v1")
            self.incorporar(origen, destino)
            (destino / "a.json").write_text("cambiado a mano")
            (destino / "b.json").write_text("también a mano")
            (origen / "a.json").write_text("v2")
            (origen / "b.json").unlink()
            avisos = self.incorporar(origen, destino)
            self.assertEqual((destino / "a.json").read_text(), "cambiado a mano")
            self.assertEqual((destino / "b.json").read_text(), "también a mano")
            self.assertIn("a.json", avisos)
            self.assertIn("b.json", avisos)

    def test_primera_actualizacion_sin_manifiesto_usa_el_inventario(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            (destino / "es/textos").mkdir(parents=True)
            (destino / "es/textos/consulta.json").write_text('{"k":"viejo"}')
            (destino / "es/textos/obsoleto.json").write_text('{"o":"x"}')
            (destino / "es/textos/propio.json").write_text('{"p":"x"}')
            (origen / "es/textos").mkdir(parents=True)
            (origen / "es/textos/consulta.json").write_text('{"k":"nuevo"}')
            avisos = self.incorporar(origen, destino, [b'{"k":"viejo"}', b'{"o":"x"}'])
            self.assertEqual((destino / "es/textos/consulta.json").read_text(), '{"k":"nuevo"}')
            # Sin manifiesto nada se retira: el inventario sólo autoriza a sustituir.
            self.assertEqual((destino / "es/textos/obsoleto.json").read_text(), '{"o":"x"}')
            self.assertEqual((destino / "es/textos/propio.json").read_text(), '{"p":"x"}')
            self.assertIn("propio.json", avisos)
            self.assertIn("obsoleto.json", avisos)

    def test_copia_manual_con_contenido_historico_se_conserva(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            (origen / "es/textos").mkdir(parents=True)
            (origen / "es/textos/comun.json").write_text('{"k":"v1"}')
            self.incorporar(origen, destino, [b'{"k":"v1"}'])
            (destino / "es/textos/copia.json.bak").write_text('{"k":"v1"}')
            (origen / "es/textos/comun.json").write_text('{"k":"v2"}')
            avisos = self.incorporar(origen, destino, [b'{"k":"v1"}', b'{"k":"v2"}'])
            self.assertEqual((destino / "es/textos/comun.json").read_text(), '{"k":"v2"}')
            self.assertEqual((destino / "es/textos/copia.json.bak").read_text(), '{"k":"v1"}')
            self.assertEqual(avisos.count("copia.json.bak"), 1)

    def test_un_solo_aviso_por_fichero_cambiado_a_mano(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            (origen / "a.json").write_text("v1")
            self.incorporar(origen, destino)
            (destino / "a.json").write_text("a mano")
            (origen / "a.json").write_text("v2")
            self.assertEqual(self.incorporar(origen, destino).count("a.json"), 1)

    def test_clave_movida_de_fichero_sin_repetidas(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            (origen / "es/textos").mkdir(parents=True)
            (origen / "es/textos/a.json").write_text('{"k1":"1","k2":"2"}')
            (origen / "es/textos/b.json").write_text('{"k3":"3"}')
            self.incorporar(origen, destino)
            (origen / "es/textos/a.json").write_text('{"k1":"1"}')
            (origen / "es/textos/b.json").write_text('{"k2":"2","k3":"3"}')
            self.incorporar(origen, destino)
            archivos.validar_textos(destino)
            # Un fichero cambiado a mano que conserva la clave movida se detecta.
            (destino / "es/textos/a.json").write_text('{"k1":"1","k2":"mano"}')
            with self.assertRaisesRegex(RuntimeError, "k2"):
                archivos.validar_textos(destino)

    def test_validacion_rechaza_json_corrupto_y_repetidas(self):
        with tempfile.TemporaryDirectory() as temporal:
            _, destino = self.preparar(temporal)
            (destino / "es/textos").mkdir(parents=True)
            (destino / "es/textos/a.json").write_text('{"k":"1","k":"2"}')
            (destino / "es/textos/b.json").write_text('{"k":')
            with self.assertRaises(RuntimeError) as error:
                archivos.validar_textos(destino)
            self.assertIn("a.json", str(error.exception))
            self.assertIn("b.json", str(error.exception))

    def test_no_reescribe_versiones_publicadas_de_la_distribucion(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            (origen / "es/catalogos/.versiones").mkdir(parents=True)
            (origen / "es/catalogos/.versiones/h.json").write_text("reserva")
            (origen / "es/catalogos/.versiones/nueva.json").write_text("añadida")
            (destino / "es/catalogos/.versiones").mkdir(parents=True)
            (destino / "es/catalogos/.versiones/h.json").write_text("publicado")
            self.incorporar(origen, destino, [b"publicado"])
            self.assertEqual((destino / "es/catalogos/.versiones/h.json").read_text(), "publicado")
            self.assertEqual((destino / "es/catalogos/.versiones/nueva.json").read_text(), "añadida")

    def test_no_sigue_enlaces_del_volumen(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            fuera = Path(temporal) / "fuera"
            fuera.mkdir()
            (origen / "es/textos").mkdir(parents=True)
            (origen / "es/textos/comun.json").write_text("{}")
            (destino / "es").mkdir()
            (destino / "es/textos").symlink_to(fuera)
            with self.assertRaises(OSError):
                self.incorporar(origen, destino)
            self.assertEqual(list(fuera.iterdir()), [])
            (destino / "es/textos").unlink()
            (destino / "es/textos").mkdir()
            (destino / "es/textos/comun.json").symlink_to(fuera / "objetivo")
            with self.assertRaises(OSError):
                self.incorporar(origen, destino)
            self.assertFalse((fuera / "objetivo").exists())
            (destino / "es/textos/comun.json").unlink()
            (destino / "es/otro").symlink_to(fuera)
            with self.assertRaises(OSError):
                self.incorporar(origen, destino)

    def test_manifiesto_manipulado_no_retira_fuera_de_la_base(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            (origen / "texto.json").write_text("{}")
            (destino / ".versiones").mkdir()
            (destino / ".versiones/v.json").write_text("publicado")
            (Path(temporal) / "ajeno").write_text("ajeno")
            h = archivos.huella(b"publicado")
            (destino / archivos.MANIFIESTO).write_text(json.dumps(
                {"entrega": "x", "ficheros": {"../ajeno": archivos.huella(b"ajeno"), ".versiones/v.json": h}}))
            self.incorporar(origen, destino)
            self.assertTrue((Path(temporal) / "ajeno").exists())
            self.assertTrue((destino / ".versiones/v.json").exists())

    def test_interrupcion_no_deja_archivo_parcial_y_permite_repetir(self):
        with tempfile.TemporaryDirectory() as temporal:
            origen, destino = self.preparar(temporal)
            for nombre in ("a.json", "b.json"):
                (origen / nombre).write_bytes(b"v1")
            self.incorporar(origen, destino)
            for nombre in ("a.json", "b.json"):
                (origen / nombre).write_bytes(b"v2 completo")
            original = archivos.os.fsync
            llamadas = []

            def cortar(fd):
                llamadas.append(fd)
                if len(llamadas) == 3:
                    raise OSError("corte de energía")
                return original(fd)

            with patch.object(archivos.os, "fsync", side_effect=cortar):
                with self.assertRaises(OSError):
                    self.incorporar(origen, destino)
            self.assertEqual((destino / "a.json").read_bytes(), b"v2 completo")
            self.assertEqual((destino / "b.json").read_bytes(), b"v1")
            self.assertEqual(sorted(p.name for p in destino.iterdir()), [archivos.MANIFIESTO, "a.json", "b.json"])
            self.incorporar(origen, destino)
            self.assertEqual((destino / "b.json").read_bytes(), b"v2 completo")
            ficheros = json.loads((destino / archivos.MANIFIESTO).read_text())["ficheros"]
            self.assertEqual(set(ficheros), {"a.json", "b.json"})

    def test_inventario_de_git_incluye_versiones_historicas(self):
        with tempfile.TemporaryDirectory() as temporal:
            repo = Path(temporal)
            git = lambda *a: subprocess.run(["git", "-C", str(repo), *a], check=True, capture_output=True)
            git("init", "-q")
            git("config", "user.email", "prueba@etf.invalid")
            git("config", "user.name", "Prueba")
            (repo / "idiomas").mkdir()
            for contenido in ("v1", "v2"):
                (repo / "idiomas/a.json").write_text(contenido)
                git("add", ".")
                git("commit", "-q", "-m", contenido)
            (repo / "fuera.txt").write_text("v9")
            git("add", ".")
            git("commit", "-q", "-m", "fuera")
            # Una fusión con conflicto resuelto crea un contenido que no está en ningún padre.
            principal = git("rev-parse", "--abbrev-ref", "HEAD").stdout.decode().strip()
            git("checkout", "-q", "-b", "rama")
            (repo / "idiomas/a.json").write_text("rama")
            git("commit", "-q", "-am", "rama")
            git("checkout", "-q", principal)
            (repo / "idiomas/a.json").write_text("principal")
            git("commit", "-q", "-am", "principal")
            subprocess.run(["git", "-C", str(repo), "merge", "-q", "rama"], capture_output=True)
            (repo / "idiomas/a.json").write_text("resuelto")
            git("add", ".")
            git("commit", "-q", "-m", "fusión")
            # Una segunda fusión sustituye el contenido resuelto: sólo queda en las fusiones.
            git("checkout", "-q", "rama")
            (repo / "idiomas/a.json").write_text("segunda")
            git("commit", "-q", "-am", "segunda")
            git("checkout", "-q", principal)
            subprocess.run(["git", "-C", str(repo), "merge", "-q", "rama"], capture_output=True)
            (repo / "idiomas/a.json").write_text("final")
            git("add", ".")
            git("commit", "-q", "-m", "segunda fusión")
            inventario = archivos.generar_inventario(repo)
            self.assertEqual(set(inventario["huellas"]), {archivos.huella(c) for c in (
                b"v1", b"v2", b"rama", b"principal", b"resuelto", b"segunda", b"final")})


class RequisitosFijados(unittest.TestCase):
    def test_versiones_y_sumas_fijadas(self):
        nombres = {nombre for nombre, _, _ in paquete.REQUISITOS}
        self.assertEqual(nombres, {"requisitos/podman-installer-windows-amd64.msi", "requisitos/docker-compose.exe"})
        for nombre, url, suma in paquete.REQUISITOS:
            self.assertRegex(url, r"^https://github\.com/(containers/podman|docker/compose)/releases/download/v\d+\.\d+\.\d+/")
            self.assertRegex(suma, r"^[0-9a-f]{64}$")
        # Los scripts de Windows usan exactamente estas rutas del paquete.
        comun = Path(__file__).with_name("comun.ps1").read_text(encoding="utf-8-sig")
        for nombre in nombres:
            self.assertIn("'" + nombre + "'", comun)

    def respuesta(self, contenido):
        return lambda url, timeout: io.BytesIO(contenido)

    def test_descarga_verificada_y_cache(self):
        import hashlib
        contenido = b"instalador oficial"
        suma = hashlib.sha256(contenido).hexdigest()
        url = "https://github.com/containers/podman/releases/download/v1.0.0/x.msi"
        with tempfile.TemporaryDirectory() as temporal:
            cache = Path(temporal)
            ruta = paquete.descargar_requisito(url, suma, cache, self.respuesta(contenido))
            self.assertEqual(ruta.read_bytes(), contenido)
            # Con la copia buena en caché no vuelve a descargar.
            def sin_red(url, timeout):
                raise AssertionError("descarga repetida")
            self.assertEqual(paquete.descargar_requisito(url, suma, cache, sin_red), ruta)
            # Una copia alterada en caché se descarta y se vuelve a bajar.
            ruta.write_bytes(b"alterado")
            self.assertEqual(paquete.descargar_requisito(url, suma, cache, self.respuesta(contenido)).read_bytes(), contenido)

    def test_suma_distinta_para_y_no_deja_restos(self):
        url = "https://github.com/docker/compose/releases/download/v1.0.0/c.exe"
        with tempfile.TemporaryDirectory() as temporal:
            cache = Path(temporal)
            with self.assertRaisesRegex(SystemExit, "SHA-256 inesperado"):
                paquete.descargar_requisito(url, "a" * 64, cache, self.respuesta(b"otro contenido"))
            self.assertEqual(list(cache.iterdir()), [])
            with self.assertRaises(SystemExit):
                paquete.descargar_requisito("http://ejemplo.invalid/x.exe", "a" * 64, cache, self.respuesta(b""))


def icono_minimo():
    """ICO de 16x16 y 32 bits, suficiente para que makensis compile."""
    import struct
    cabecera = struct.pack("<IiiHHIIiiII", 40, 16, 32, 1, 32, 0, 0, 0, 0, 0, 0)
    imagen = cabecera + b"\x40\x20\x10\xff" * 256 + b"\x00" * 64
    return struct.pack("<HHH", 0, 1, 1) + struct.pack("<BBBBHHII", 16, 16, 0, 0, 1, 32, len(imagen), 22) + imagen


class InstaladorWindows(unittest.TestCase):
    NSI = (Path(__file__).with_name("instalador.nsi")).read_text(encoding="utf-8")
    LANZADOR = (Path(__file__).with_name("lanzador.nsi")).read_text(encoding="utf-8")

    def test_instala_por_usuario_sin_administrador(self):
        for guion in (self.NSI, self.LANZADOR):
            self.assertIn("RequestExecutionLevel user", guion)
            self.assertNotIn("RequestExecutionLevel admin", guion)
            self.assertIn("Target amd64-unicode", guion)
        self.assertIn('InstallDir "$LOCALAPPDATA\\Expedientes-ETF\\pruebas\\paquete"', self.NSI)
        self.assertNotIn("HKLM", self.NSI)

    def test_registra_entrada_de_desinstalacion_del_usuario(self):
        self.assertIn('"Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Expedientes-ETF"', self.NSI)
        for valor in ("DisplayName", "DisplayVersion", "Publisher", "DisplayIcon", "UninstallString"):
            self.assertIn(f'WriteRegStr HKCU "${{CLAVE_DESINSTALAR}}" "{valor}"', self.NSI)
        self.assertIn('DeleteRegKey HKCU "${CLAVE_DESINSTALAR}"', self.NSI)
        self.assertIn("WriteUninstaller", self.NSI)
        # El desinstalador no borra carpetas enteras: sólo lo que copió.
        self.assertNotIn("RMDir /r", self.NSI)

    def test_instala_o_actualiza_segun_los_datos(self):
        self.assertIn('${If} ${FileExists} "$Privado\\entorno.env"\n        StrCpy $Accion "actualizar"', self.NSI)
        self.assertIn('StrCpy $Accion "instalar"', self.NSI)
        self.assertIn("""Exec '"${CONSOLA}" /d /c ""$INSTDIR\\$Accion.cmd""'""", self.NSI)
        self.assertIn('-File "$INSTDIR\\desinstalar.ps1"', self.NSI)

    def test_lanzador_abre_abrir_ps1_con_ruta_absoluta(self):
        self.assertIn('"$WINDIR\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"', self.LANZADOR)
        self.assertIn('-File "$EXEDIR\\abrir.ps1"', self.LANZADOR)
        self.assertIn("SilentInstall silent", self.LANZADOR)

    def test_textos_del_asistente_en_cada_idioma(self):
        textos = paquete.idiomas_nsis()
        for clave in ("instalador_nombre", "instalador_bienvenida_texto", "instalador_ajeno"):
            for idioma in ("SPANISH", "ENGLISH"):
                self.assertIn(f"LangString {clave} ${{LANG_{idioma}}} ", textos)
        usadas = set(re.findall(r"\$\((instalador_\w+)\)", self.NSI))
        definidas = set(re.findall(r"LangString (\w+) ", textos))
        self.assertEqual(usadas, definidas)
        self.assertEqual(paquete.escapar_nsis('a$b "c"\nd'), 'a$$b $\\"c$\\"$\\r$\\nd')

    def test_desinstalador_solo_borra_lo_copiado(self):
        with tempfile.TemporaryDirectory() as temporal:
            carpeta = Path(temporal)
            (carpeta / "requisitos").mkdir()
            (carpeta / "requisitos/podman.msi").write_bytes(b"x")
            (carpeta / "Expedientes ETF.exe").write_bytes(b"x")
            ordenes = paquete.archivos_nsis(carpeta).splitlines()
            self.assertEqual(ordenes, ['Delete "$INSTDIR\\Expedientes ETF.exe"', 'Delete "$INSTDIR\\requisitos\\podman.msi"',
                                       'RMDir "$INSTDIR\\requisitos"'])
            (carpeta / 'mal"$nombre').write_bytes(b"x")
            with self.assertRaises(SystemExit):
                paquete.archivos_nsis(carpeta)

    def test_version_numerica_para_propiedades(self):
        self.assertEqual(paquete.version_numerica("1.0.3"), "1.0.3.0")
        self.assertEqual(paquete.version_numerica("v2.1"), "2.1.0.0")
        self.assertIsNone(paquete.version_numerica("a1b2c3d4e5f6"))
        self.assertIsNone(paquete.version_numerica("1.99999"))

    def test_firma_opcional_sin_secretos_en_el_repositorio(self):
        llamadas = []
        def simular(orden, check):
            llamadas.append(orden)
            Path(orden[orden.index("-out") + 1]).write_bytes(b"firmado")
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            exe = raiz / "a.exe"
            exe.write_bytes(b"MZ")
            self.assertFalse(paquete.firmar(exe, {}, simular))
            with self.assertRaises(SystemExit):
                paquete.firmar(exe, {"ETF_FIRMA_CERTIFICADO": str(exe)}, simular)
            with self.assertRaises(SystemExit):
                paquete.firmar(exe, {"ETF_FIRMA_CERTIFICADO": str(paquete.RAIZ / "README.md"),
                                     "ETF_FIRMA_CLAVE": str(paquete.RAIZ / "README.md")}, simular)
            (raiz / "c.pem").write_text("c")
            (raiz / "k.pem").write_text("k")
            self.assertTrue(paquete.firmar(exe, {"ETF_FIRMA_CERTIFICADO": str(raiz / "c.pem"),
                                                 "ETF_FIRMA_CLAVE": str(raiz / "k.pem")}, simular))
            self.assertEqual(exe.read_bytes(), b"firmado")
            self.assertIn("osslsigncode", llamadas[0][0])
            self.assertEqual(sorted(p.name for p in raiz.iterdir()), ["a.exe", "c.pem", "k.pem"])

    @unittest.skipUnless(paquete.shutil.which("makensis"), "NSIS no está instalado en este puesto")
    def test_genera_lanzador_e_instalador_exe(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            carpeta, salida = raiz / "Expedientes-ETF", raiz / "salida"
            carpeta.mkdir()
            salida.mkdir()
            (carpeta / "aplicacion.ico").write_bytes(icono_minimo())
            for nombre in ("instalar.cmd", "actualizar.cmd", "desinstalar.ps1", "abrir.ps1", "SUMAS.json"):
                (carpeta / nombre).write_text("prueba", encoding="utf-8")
            with redirect_stderr(io.StringIO()), patch("builtins.print"):
                lanzador = paquete.construir_lanzador(carpeta)
                archivo = paquete.construir_instalador(carpeta, "1.2.3", salida)
            self.assertEqual(lanzador.read_bytes()[:2], b"MZ")
            self.assertEqual(archivo.name, "Expedientes-ETF-1.2.3-instalador.exe")
            self.assertEqual(archivo.read_bytes()[:2], b"MZ")
            suma = archivo.with_name(archivo.name + ".sha256").read_text(encoding="ascii")
            self.assertEqual(suma, f"{paquete.suma_fichero(archivo)}  {archivo.name}\n")


if __name__ == "__main__":
    unittest.main()
