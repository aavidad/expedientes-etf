"""La preparación PWA1 sólo puede escribir sus recursos en su propia demo."""
import contextlib
import importlib.util
import io
from pathlib import Path
import tempfile
import types
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('preparar_pwa1', Path(__file__).with_name('preparar_demo.py'))
PREPARAR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PREPARAR)


class PreparacionPWA(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory(prefix='pwa1-fix-prueba-')
        self.addCleanup(self.temporal.cleanup)
        self.raiz = Path(self.temporal.name)
        self.llamadas = []
        self.demo = types.SimpleNamespace(URL='http://127.0.0.1:18541',
                                          incorporar_recursos=lambda: self.llamadas.append("incorporar"))
        spec = types.SimpleNamespace(loader=types.SimpleNamespace(exec_module=lambda modulo: None))
        for parche in [patch.object(PREPARAR, 'RAIZ', self.raiz),
                       patch.object(PREPARAR.importlib.util, 'spec_from_file_location', return_value=spec),
                       patch.object(PREPARAR.importlib.util, 'module_from_spec', return_value=self.demo)]:
            parche.start()
            self.addCleanup(parche.stop)

    def test_rechaza_otra_demo_antes_de_escribir(self):
        for base in ['http://127.0.0.1:18542', 'https://127.0.0.1:18541',
                     'http://servidor.interno:18541', 'http://127.0.0.1:18541?destino=otro']:
            with self.subTest(base=base), self.assertRaises(SystemExit):
                PREPARAR.preparar(base)
        self.assertEqual(self.llamadas, [])

    def test_utiliza_preparacion_comun_sin_copias_directas(self):
        with contextlib.redirect_stdout(io.StringIO()):
            PREPARAR.preparar(self.demo.URL)
        self.assertEqual(self.llamadas, ["incorporar"])



if __name__ == '__main__':
    unittest.main()
