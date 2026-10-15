"""Comprueba las capacidades reunidas de los manuales de Sistemas y usuarios."""

import importlib.util
from pathlib import Path
import re
import tempfile
import unittest
from xml.etree import ElementTree as ET
from zipfile import ZipFile

SPEC = importlib.util.spec_from_file_location("exportador_union", Path(__file__).with_name("exportar.py"))
EXPORTADOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(EXPORTADOR)
NS = {
    "text": "urn:oasis:names:tc:opendocument:xmlns:text:1.0",
    "draw": "urn:oasis:names:tc:opendocument:xmlns:drawing:1.0",
    "table": "urn:oasis:names:tc:opendocument:xmlns:table:1.0",
    "xlink": "http://www.w3.org/1999/xlink",
    "meta": "urn:oasis:names:tc:opendocument:xmlns:meta:1.0",
    "style": "urn:oasis:names:tc:opendocument:xmlns:style:1.0",
}


class ExportadorUnion(unittest.TestCase):
    def documento(self, nombre):
        with tempfile.TemporaryDirectory(prefix="MANU-test-") as temporal:
            destino = Path(temporal) / f"{nombre}.odt"
            EXPORTADOR.documento(nombre, destino)
            with ZipFile(destino) as archivo:
                contenido = ET.fromstring(archivo.read("content.xml"))
                meta = ET.fromstring(archivo.read("meta.xml")) if "meta.xml" in archivo.namelist() else None
                manifiesto = ET.fromstring(archivo.read("META-INF/manifest.xml"))
                rutas = [e.attrib for e in manifiesto]
                self.assertEqual(len(rutas), len({tuple(r.items()) for r in rutas}))
                figuras = [n for n in archivo.namelist() if n.startswith("Pictures/")]
        anclas = {b.get("{" + NS["text"] + "}name") for b in contenido.findall(".//text:bookmark", NS)}
        for enlace in contenido.findall(".//text:a", NS):
            destino = enlace.get("{" + NS["xlink"] + "}href")
            if destino.startswith("#"):
                self.assertIn(destino[1:], anclas, destino)
        for marco in contenido.findall(".//draw:frame", NS):
            self.assertTrue(marco.find("svg:title", {"svg": "urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0"}).text)
        return contenido, meta, figuras

    def test_sistemas_anexos_tablas_literales_negritas_y_pasos(self):
        contenido, meta, figuras = self.documento("sistemas")
        self.assertEqual(len(figuras), 3)
        self.assertEqual(meta.find(".//meta:initial-creator", NS).text, "Expedientes ETF")
        titulos = ["".join(h.itertext()) for h in contenido.findall(".//text:h", NS)]
        for nombre in ("LEEME.md", "CONFIGURACION.md", "OPERACION.md", "DIAGNOSTICO.md", "MANSIS.md"):
            fuente = (EXPORTADOR.BASE.parent / "sistemas" / nombre).read_text()
            titulo = next(re.sub(r"^#+ ", "", l) for l in fuente.splitlines() if re.match(r"^#{1,2} ", l))
            self.assertIn(titulo, titulos)
        self.assertGreater(len(contenido.findall(".//table:table", NS)), 0)
        estilos = {n.get("{" + NS["text"] + "}style-name") for n in contenido.findall(".//text:span", NS)}
        self.assertTrue({"Literal", "Negrita"} <= estilos)
        pasos = contenido.findall(".//text:list-item", NS)
        self.assertTrue(any(p.get("{" + NS["text"] + "}start-value") == "2" for p in pasos))
        self.assertNotIn("```", "".join(contenido.itertext()))

    def test_usuarios_ayuda_comun_capturas_y_autor(self):
        for nombre in ("profesionales", "coordinacion"):
            with self.subTest(manual=nombre):
                contenido, meta, figuras = self.documento(nombre)
                self.assertGreater(len(figuras), 0)
                self.assertEqual(meta.find(".//meta:initial-creator", NS).text.rstrip("."), "Expedientes ETF")
                titulos = ["".join(h.itertext()) for h in contenido.findall(".//text:h", NS)]
                self.assertIn("Entrada, preferencias y ayuda", titulos)
                self.assertIn("Glosario de la aplicación", titulos)

    def test_administracion_conserva_contenido_propio(self):
        contenido, _, figuras = self.documento("administracion")
        self.assertGreater(len(figuras), 0)
        self.assertNotIn("Entrada, preferencias y ayuda", "".join(contenido.itertext()))


if __name__ == "__main__":
    unittest.main()
