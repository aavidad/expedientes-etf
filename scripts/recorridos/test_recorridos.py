"""Regresiones de los límites del guion y de la sincronización del montaje."""
import tempfile
import io
from contextlib import redirect_stdout, redirect_stderr
from unittest.mock import patch
from videos import main
import unittest
from pathlib import Path
from guion import leer
from herramientas import RAIZ
from poner_voz import planificar, retirar_pausas
from comun_test import validar_base, validar_restauracion
from arranque import inicio_con_contenido, TAMANO
from alinear import buscar_ancla, TAMANO as TAMANO_ANCLA

GUIÓN = RAIZ / 'docs/recorridos/guiones/01-entrada.md'


class GuionTest(unittest.TestCase):
    def cambiar(self, origen, destino):
        with tempfile.TemporaryDirectory() as d:
            f = Path(d) / 'guion.md'
            f.write_text(GUIÓN.read_text().replace(origen, destino))
            return leer(f)

    def test_guion_completo(self):
        g = leer(GUIÓN)
        self.assertEqual(len(g['pasos']), 7)
        self.assertIn('Equipos de Tratamiento Familiar', g['intro']['Locución'])

    def test_selecciona_recorridos_sin_repetirlos(self):
        salida = io.StringIO()
        with patch('sys.argv', ['videos.py', '--validar', '--recorrido', '05-ficha', '--recorrido', '08-traslado']), redirect_stdout(salida):
            main()
        self.assertIn('GUIONES-OK 2:', salida.getvalue())
        with patch('sys.argv', ['videos.py', '--validar', '--recorrido', '05-ficha', '--recorrido', '05-ficha']), redirect_stderr(io.StringIO()):
            with self.assertRaises(SystemExit) as error:
                main()
        self.assertEqual(error.exception.code, 2)

    def test_introduccion_obligatoria(self):
        with self.assertRaisesRegex(ValueError, 'Falta Introducción'):
            self.cambiar('## Introducción', '## Otra cabecera')

    def test_voz_obligatoria(self):
        with self.assertRaisesRegex(ValueError, 'Falta Introducción'):
            self.cambiar('Locución: Este recorrido', 'Voz ausente: Este recorrido')

    def test_siglas_en_voz(self):
        with self.assertRaisesRegex(ValueError, 'siglas'):
            self.cambiar('profesional de los Equipos de Tratamiento Familiar', 'profesional de los ETF')

    def test_otras_siglas_en_voz(self):
        with self.assertRaisesRegex(ValueError, 'siglas'):
            self.cambiar('prepara su cuenta', 'usa SQL y prepara su cuenta')

    def test_acciones_y_voz_comparten_fases(self):
        escenas = [{'id': 'entrar', 'fases': [{'inicio_s': 2, 'fin_s': 7}, {'inicio_s': 7, 'fin_s': 15}]}]
        pasos = [{'id': 'entrar', 'frases': [{'duracion_s': 4, 'archivo': 'uno.wav'}, {'duracion_s': 3, 'archivo': 'dos.wav'}]}]
        plan, fin = planificar(escenas, pasos, 14)
        self.assertAlmostEqual(plan[0]['fases'][0]['voz_ini'], 14.1)
        self.assertAlmostEqual(plan[0]['fases'][1]['sale_ini'], 18.32)
        self.assertAlmostEqual(fin, 21.64)
        for fase in plan[0]['fases']:
            self.assertGreater(fase['sale_fin'], fase['voz_ini'])

    def test_rechaza_fases_que_no_corresponden_a_la_voz(self):
        with self.assertRaisesRegex(ValueError, 'no coinciden'):
            planificar([{'id': 'entrar', 'fases': []}], [{'id': 'entrar', 'frases': [{'duracion_s': 4}]}], 14)

    def test_ancla_visual_corrige_inicio_de_la_captura(self):
        fondo = bytes([255]) * TAMANO_ANCLA
        marca = bytes([0, 255]) * (TAMANO_ANCLA // 2)
        self.assertAlmostEqual(buscar_ancla(fondo * 10 + marca * 2, marca), 0.4)
        with self.assertRaisesRegex(ValueError, 'no contiene el anclaje'):
            buscar_ancla(fondo * 10, marca)

    def test_recorta_esperas_sin_eliminar_el_cambio_de_pantalla(self):
        quieto = bytes([255]) * 4
        cambio = bytes([0]) * 4
        datos = quieto * 20 + cambio * 20
        tramos = retirar_pausas(datos, 0, 1.6, 1, tamano=4)
        self.assertTrue(any(a <= 0.8 <= b for a, b in tramos))
        self.assertAlmostEqual(sum(b-a for a, b in tramos), 1)

    def test_no_acelera_una_accion_sin_pausas(self):
        with self.assertRaisesRegex(ValueError, 'sin tiempo muerto'):
            retirar_pausas(bytes([200, 201]) * 40, 0, 1.6, 1, tamano=1)

    def test_retira_pantalla_uniforme_inicial(self):
        blanco = bytes([255]) * TAMANO
        pantalla = bytes([0, 255]) * (TAMANO // 2)
        self.assertAlmostEqual(inicio_con_contenido(blanco + pantalla), 0.04)
        self.assertEqual(inicio_con_contenido(pantalla), 0)

    def test_rechaza_arranque_sin_pantalla(self):
        with self.assertRaisesRegex(ValueError, 'sin una pantalla visible'):
            inicio_con_contenido(bytes([255]) * TAMANO * 75)

    def test_no_retira_concesion_ajena_tras_conflicto(self):
        validar_restauracion(self)

    def test_no_acepta_red_externa(self):
        validar_base(self)


if __name__ == '__main__':
    unittest.main()
