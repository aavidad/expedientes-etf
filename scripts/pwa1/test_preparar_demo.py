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
                                          compose=lambda *args, **kwargs: self.llamadas.append(args))
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

    def test_copia_solo_reserva_pwa_sin_versiones_publicadas(self):
        recursos = ['idiomas/es/textos/pwa1.json', 'temas/alto-contraste/iconos/etf-192-any.png']
        ajenos = ['idiomas/es/textos/comun.json', 'temas/alto-contraste/tokens.json',
                  'idiomas/.versiones/1/es/textos/pwa1.json']
        for nombre in recursos + ajenos:
            destino = self.raiz / nombre
            destino.parent.mkdir(parents=True, exist_ok=True)
            destino.write_text('{}')
        with contextlib.redirect_stdout(io.StringIO()):
            PREPARAR.preparar(self.demo.URL)
        copias = [args for args in self.llamadas if args[0] == 'cp']
        self.assertEqual({args[2] for args in copias}, {'app:/datos/' + nombre for nombre in recursos})
        self.assertEqual(len(copias), len(recursos))
        self.assertTrue(all(args[0] in ('cp', 'exec') for args in self.llamadas))


if __name__ == '__main__':
    unittest.main()
