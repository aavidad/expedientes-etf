"""Comprueba portada, secciones y recursos del lector en web y SCORM."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile
import curso
from comun import VOZ, sha_texto


class CursoVozTest(unittest.TestCase):
    def test_portada_y_secciones_en_ambas_exportaciones(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz=Path(temporal);salida=raiz/'curso';web=salida/'web';web.mkdir(parents=True)
            id='etf-'+('a'*20)
            secciones=[{'id':id+'-s0','titulo':'Entrada','texto':'Entrada Abre Ayuda.'}, {'id':id+'-s1','titulo':'Salida','texto':'Salida Cierra Ayuda.'}]
            pagina=f'<html id="exe-index"><body><div id="page-content-{id}"><h2>Entrada</h2><p>Abre Ayuda.</p><h2>Salida</h2><p>Cierra Ayuda.</p></div></body></html>'
            (web/'index.html').write_text(pagina)
            with zipfile.ZipFile(salida/'scorm12.zip','w') as paquete:
                paquete.writestr('index.html',pagina)
                paquete.writestr('imsmanifest.xml','<manifest><resources><resource></resource></resources></manifest>')
            audios=raiz/'voz/curso';audios.mkdir(parents=True)
            datos={}
            for s in secciones:
                sha=sha_texto(VOZ+s['texto']);nombre=sha+'.mp3'
                (audios/nombre).write_bytes(b'audio-simulado')
                datos[s['id']]={'fichero':nombre,'sha':sha,'duracion':1}
            (audios/'manifiesto.json').write_text(json.dumps(datos))
            with patch.object(curso,'entradas',lambda:{'curso':secciones}):
                curso.integrar(None,salida,raiz/'voz')
            html=(web/'index.html').read_text()
            self.assertIn('data-voz-pagina',html)
            self.assertEqual(html.count('data-voz-seccion='),2)
            self.assertNotIn('autoplay',html)
            with zipfile.ZipFile(salida/'scorm12.zip') as paquete:
                self.assertIn('data-voz-pagina',paquete.read('index.html').decode())
                self.assertIn('voz/lector.js',paquete.namelist())
                manifiesto=paquete.read('imsmanifest.xml').decode()
                for audio in datos.values():
                    self.assertIn('voz/'+audio['fichero'],paquete.namelist())
                    self.assertIn(audio['fichero'],manifiesto)

    def test_sin_audio_no_cambia_el_curso(self):
        with tempfile.TemporaryDirectory() as temporal:
            salida=Path(temporal)
            curso.integrar(None,salida,salida/'ausente')
            self.assertEqual(list(salida.iterdir()),[])


if __name__=='__main__':unittest.main()
