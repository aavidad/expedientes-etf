"""Pruebas sin red: texto editorial, caché y publicación de manifiestos."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from comun import generar, legible, limpiar, secciones, sha_texto, VOZ
import construir


class VozTest(unittest.TestCase):
    def test_limpieza_no_lee_imagenes_ni_urls(self):
        texto = '# Título\n**Abre** [Ayuda](https://portal.invalid/ruta).\n![No debe leerse](foto.png)\nVisita https://portal.invalid/privado\n`Escuchar`\n'
        self.assertEqual(limpiar(texto), 'Título Abre Ayuda . Visita Escuchar')

    def test_secciones_no_confunde_titulos_dentro_de_codigo(self):
        partes = secciones('Introducción\n\n## Entrada\nAbre Ayuda.\n```\n## No es título\n```\n## Salida\nCierra.', 'pagina')
        self.assertEqual([s['id'] for s in partes], ['pagina-s0', 'pagina-s1', 'pagina-s2'])
        self.assertEqual(partes[1]['titulo'], 'Entrada')
        self.assertNotIn('No es título', partes[1]['texto'])

    def test_signo_suelto_no_se_graba(self):
        self.assertFalse(legible('?'))
        self.assertTrue(legible('¿Qué es?'))
        self.assertEqual(secciones('?', 'pagina'), [])

    def test_cache_deduplica_y_cambia_con_texto(self):
        llamadas = []
        def simular(texto, archivo):
            llamadas.append(texto); archivo.write_bytes(b'mp3-simulado')
        with tempfile.TemporaryDirectory() as temporal:
            destino = Path(temporal)
            lista = [{'id':'primera','texto':'Abre Ayuda.'}, {'id':'segunda','texto':'Abre Ayuda.'}]
            datos = generar(lista,destino,simular,lambda _:1.5)
            generar(lista,destino,simular,lambda _: self.fail('No debe medir otra vez'))
            self.assertEqual(llamadas,['Abre Ayuda.'])
            self.assertEqual(datos['primera']['sha'],sha_texto(VOZ+'Abre Ayuda.'))
            self.assertEqual(datos['primera']['fichero'],datos['segunda']['fichero'])
            generar([{'id':'primera','texto':'Cierra Ayuda.'}],destino,simular,lambda _:2)
            self.assertEqual(len(llamadas),2)
            publicado=json.loads((destino/'manifiesto.json').read_text())
            self.assertEqual(list(publicado),['primera'])
            self.assertEqual(publicado['primera']['duracion'],2)

    def test_fallo_no_publica_manifestado_incompleto(self):
        with tempfile.TemporaryDirectory() as temporal:
            destino=Path(temporal)
            generar([{'id':'uno','texto':'Abre.'}],destino,lambda _,p:p.write_bytes(b'mp3'),lambda _:1)
            previo=(destino/'manifiesto.json').read_bytes()
            def falla(_,archivo):
                archivo.write_bytes(b'incompleto'); raise RuntimeError('fallo-simulado')
            with self.assertRaises(RuntimeError):
                generar([{'id':'dos','texto':'Cierra.'}],destino,falla,lambda _:1)
            self.assertEqual((destino/'manifiesto.json').read_bytes(),previo)
            self.assertEqual(list(destino.glob('*.tmp.mp3')),[])

    def test_rechaza_id_ruta_y_duplicado(self):
        with tempfile.TemporaryDirectory() as temporal:
            for lista in ([{'id':'../fuera','texto':'Abre'}], [{'id':'uno','texto':'Abre'}]*2):
                with self.assertRaises(ValueError):
                    generar(lista,temporal,lambda _,p:p.write_bytes(b'mp3'),lambda _:1)

    def test_manuales_sin_html_activo_y_con_imagenes_locales(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz=Path(temporal); docs=raiz/'docs/manuales';docs.mkdir(parents=True)
            (docs/'captura.png').write_bytes(b'png-local')
            for nombre in (*construir.MANUALES.values(),'comun'):
                (docs/(nombre+'.md')).write_text('# Manual\n\n## Abrir\n<script>alert(1)</script>\n[enlace](javascript:alert(1))\n![Captura local](captura.png)\n')
            with patch.object(construir,'RAIZ',raiz):
                construir.manuales(docs/'web')
            datos=json.loads((docs/'web/coordinacion.json').read_text())
            contenido=''.join(s['html'] for s in datos['secciones'])
            self.assertNotIn('<script>',contenido)
            self.assertNotIn('href="javascript:',contenido)
            self.assertIn('/manuales/recursos/',contenido)
            (docs/'administracion.md').write_text('# Manual\n![Captura](../../../secreto.png)')
            with patch.object(construir,'RAIZ',raiz),self.assertRaises(ValueError):
                construir.manuales(docs/'web')

    def test_indices_y_referencias_entre_manuales(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz=Path(temporal);docs=raiz/'docs/manuales';docs.mkdir(parents=True)
            for nombre in (*construir.MANUALES.values(),'comun'):
                (docs/(nombre+'.md')).write_text('# Manual\n\n## Entrada\n[Entrada](#entrada)\n[Guía relacionada](profesionales.md#entrada)\n[Todos los manuales](usuario.md)\n')
                (docs/(nombre+'.pdf')).write_bytes(b'%PDF-editorial')
            with patch.object(construir,'RAIZ',raiz):construir.manuales(docs/'web')
            datos=json.loads((docs/'web/coordinacion.json').read_text())
            contenido=''.join(s['html'] for s in datos['secciones'])
            self.assertIn('href="#coordinacion-s1"',contenido)
            self.assertIn('href="/ayuda"',contenido)
            self.assertNotIn('<h3>Entrada</h3>',contenido)
            recursos=json.loads((docs/'web/recursos.json').read_text())
            pdf=next(r for r in recursos.values() if r['ruta']=='manuales/profesionales.pdf')
            self.assertIn('coordinacion',pdf['manuales'])


if __name__ == '__main__':
    unittest.main()
