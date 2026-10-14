import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from ensayos import LIMITE_SALIDA, esperar_postgres, orden


class OrdenTest(unittest.TestCase):
    def test_fallo_muestra_ambos_canales_incluso_stderr_vacio(self):
        resultado = subprocess.CompletedProcess([], 1, '{"msg":"migracion.error"}', '')
        with patch('ensayos.subprocess.run', return_value=resultado):
            with self.assertRaisesRegex(RuntimeError, 'migracion.error') as fallo:
                orden(['migrar'], modulo='documento')
        self.assertIn('codigo=1', str(fallo.exception))
        self.assertIn('stderr:\n[vacío]', str(fallo.exception))

    def test_oculta_secretos_antes_de_acotar_y_omite_detalles_sql(self):
        with tempfile.TemporaryDirectory(prefix='fx5-diagnostico-') as carpeta:
            archivo = Path(carpeta) / 'runner.env'
            archivo.write_text('ETF_MIGRADOR_CONTRASENA=credencial-inventada\n')
            salida = ('credencial-inventada\nDETAIL: fila sensible\n'
                      'CONTEXT: SQL sensible\n' + 'x' * 10000 + '\nfin-del-fallo')
            resultado = subprocess.CompletedProcess([], 1, salida, 'clave-inventada')
            with patch('ensayos.subprocess.run', return_value=resultado):
                with self.assertRaises(RuntimeError) as fallo:
                    orden(['docker', '--env-file', str(archivo), 'TOKEN=clave-inventada'])
        mensaje = str(fallo.exception)
        for omitido in ('credencial-inventada', 'clave-inventada', 'fila sensible',
                        'SQL sensible', str(archivo), 'TOKEN='):
            self.assertNotIn(omitido, mensaje)
        self.assertIn('[secreto]', mensaje)
        self.assertIn('[salida truncada]', mensaje)
        self.assertIn('fin-del-fallo', mensaje)
        self.assertLess(len(mensaje), 2 * LIMITE_SALIDA + 100)

    def test_timeout_muestra_salidas_parciales_sin_argumentos(self):
        fallo = subprocess.TimeoutExpired(['orden', 'secreto-en-argumento'], 1,
                                          output=b'salida parcial', stderr=b'fallo parcial')
        with patch('ensayos.subprocess.run', side_effect=fallo):
            with self.assertRaises(RuntimeError) as error:
                orden(['orden'], timeout=1)
        self.assertIn('salida parcial', str(error.exception))
        self.assertIn('fallo parcial', str(error.exception))
        self.assertNotIn('secreto-en-argumento', str(error.exception))
        self.assertIsNone(error.exception.__cause__)

    def test_exito_conserva_salida_para_comprobaciones_del_ensayo(self):
        with patch('ensayos.subprocess.run', return_value=subprocess.CompletedProcess([], 0, '  OK\n', '')):
            self.assertEqual(orden(['orden']), 'OK')


class EsperarPostgresTest(unittest.TestCase):
    def test_espera_tcp_autenticado_y_no_ejecuta_migraciones(self):
        resultados = [subprocess.CompletedProcess([], 2, '', 'connection refused'),
                      subprocess.CompletedProcess([], 0, '1\n', '')]
        with patch('ensayos.subprocess.run', side_effect=resultados) as ejecutar:
            with patch('ensayos.time.sleep'):
                esperar_postgres('base-propia', '/ruta/inexistente')
        self.assertEqual(ejecutar.call_count, 2)
        comando = ejecutar.call_args.args[0]
        self.assertIn('base-propia', comando)
        self.assertIn('-h 127.0.0.1', comando[-1])
        self.assertIn('-U etf_migrador', comando[-1])
        self.assertIn('SELECT 1', comando[-1])
        self.assertNotIn('/pruebas/migrar', comando)

    def test_agota_plazo_con_diagnostico_del_ultimo_sondeo(self):
        resultado = subprocess.CompletedProcess([], 2, '', 'connection refused')
        with patch('ensayos.time.monotonic', side_effect=[0, 0, 0, 2, 2]):
            with patch('ensayos.subprocess.run', return_value=resultado) as ejecutar:
                with self.assertRaisesRegex(RuntimeError, 'connection refused') as fallo:
                    esperar_postgres('base-propia', '/ruta/inexistente', timeout=1)
        self.assertIn('servicio_no_disponible', str(fallo.exception))
        self.assertEqual(ejecutar.call_count, 1)
