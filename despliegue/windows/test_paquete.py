"""Protecciones del paquete frente a credenciales y montajes del servidor."""
import copy
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
    def test_conserva_publicados_y_incorpora_versiones_nuevas(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            origen, destino = raiz / "origen", raiz / "destino"
            origen.mkdir()
            destino.mkdir()
            (origen / "texto.json").write_text("reserva")
            (destino / "texto.json").write_text("publicado")
            (origen / "plantillas/acta/v2").mkdir(parents=True)
            (origen / "plantillas/acta/v2/acta.docx").write_bytes(b"plantilla nueva")
            archivos.incorporar(origen, destino, os.getuid(), os.getgid())
            self.assertEqual((destino / "texto.json").read_text(), "publicado")
            self.assertEqual((destino / "plantillas/acta/v2/acta.docx").read_bytes(), b"plantilla nueva")

    def test_fallo_de_copia_no_publica_archivo_parcial_y_permite_repetir(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            origen, destino = raiz / "origen", raiz / "destino"
            origen.mkdir()
            (origen / "archivo.json").write_bytes(b"contenido completo")

            def interrumpir(entrada, salida):
                Path(salida).write_bytes(b"parcial")
                raise OSError("copia interrumpida")

            with patch.object(archivos.shutil, "copyfile", side_effect=interrumpir):
                with self.assertRaises(OSError):
                    archivos.incorporar(origen, destino, os.getuid(), os.getgid())
            self.assertFalse((destino / "archivo.json").exists())
            self.assertEqual(list(destino.iterdir()), [])
            archivos.incorporar(origen, destino, os.getuid(), os.getgid())
            self.assertEqual((destino / "archivo.json").read_bytes(), b"contenido completo")


if __name__ == "__main__":
    unittest.main()
