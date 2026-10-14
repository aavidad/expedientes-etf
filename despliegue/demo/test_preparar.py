"""Comprueba que actualizar una demo conserva su entorno privado."""
import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

RUTA = Path(__file__).with_name("demo.py")
SPEC = importlib.util.spec_from_file_location("demo_preparar", RUTA)
DEMO = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(DEMO)


class PrepararDemoTest(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory(prefix="fx2-entorno-")
        self.addCleanup(self.temporal.cleanup)
        local = Path(self.temporal.name)
        self.parche = patch.object(DEMO, "LOCAL", local)
        self.parche.start()
        self.addCleanup(self.parche.stop)
        self.ruta = local / "entorno.env"
        self.anterior = "ETF_APP_CONTRASENA=clave_de_prueba\nETF_DEMO_SUBRED_INTERNA=10.253.20.0/28\n"
        self.ruta.write_text(self.anterior)

    def test_amplia_entorno_antiguo_sin_cambiar_credenciales(self):
        DEMO.preparar()
        contenido = self.ruta.read_text()
        self.assertTrue(contenido.startswith(self.anterior))
        self.assertEqual(contenido.count("ETF_CALCULO_MEMORIA_CONTRASENA="), 1)
        self.assertTrue(contenido.split("ETF_CALCULO_MEMORIA_CONTRASENA=")[1].strip())
        self.assertEqual(self.ruta.stat().st_mode & 0o777, 0o600)
        DEMO.preparar()
        self.assertEqual(self.ruta.read_text(), contenido)

    def test_fallo_de_publicacion_conserva_archivo_anterior(self):
        with patch.object(DEMO.os, "replace", side_effect=OSError("fallo_de_prueba")):
            with self.assertRaises(OSError):
                DEMO.preparar()
        self.assertEqual(self.ruta.read_text(), self.anterior)
        self.assertEqual(list(self.ruta.parent.iterdir()), [self.ruta])

    def test_conserva_credencial_configurada_incluso_vacia(self):
        for clave in ("", "clave_configurada"):
            contenido = self.anterior + "ETF_CALCULO_MEMORIA_CONTRASENA=" + clave + "\n"
            self.ruta.write_text(contenido)
            DEMO.preparar()
            self.assertEqual(self.ruta.read_text(), contenido)
