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


if __name__ == "__main__":
    unittest.main()
