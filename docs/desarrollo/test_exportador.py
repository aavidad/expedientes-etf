"""Comprueba los contratos de la edición PDF del manual de programadores."""

import importlib.util
from pathlib import Path
import tempfile
import unittest
from xml.etree import ElementTree as ET
from zipfile import ZipFile


RUTA = Path(__file__).resolve().parents[1] / "manuales" / "exportar.py"
SPEC = importlib.util.spec_from_file_location("exportador_manual", RUTA)
EXPORTADOR = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(EXPORTADOR)
NS = {
    "text": "urn:oasis:names:tc:opendocument:xmlns:text:1.0",
    "draw": "urn:oasis:names:tc:opendocument:xmlns:drawing:1.0",
    "table": "urn:oasis:names:tc:opendocument:xmlns:table:1.0",
    "xlink": "http://www.w3.org/1999/xlink",
    "meta": "urn:oasis:names:tc:opendocument:xmlns:meta:1.0",
}


class ExportadorProgramadores(unittest.TestCase):
    def test_capitulos_codigo_tablas_figuras_y_autor(self):
        with tempfile.TemporaryDirectory(prefix="MANDEV-test-") as temporal:
            destino = Path(temporal) / "programadores.odt"
            EXPORTADOR.documento("programadores", destino)
            with ZipFile(destino) as archivo:
                contenido = ET.fromstring(archivo.read("content.xml"))
                meta = ET.fromstring(archivo.read("meta.xml"))
                figuras = [n for n in archivo.namelist() if n.startswith("Pictures/")]
            self.assertEqual(meta.find(".//meta:initial-creator", NS).text, "Expedientes ETF")
            titulos = ["".join(h.itertext()) for h in contenido.findall(".//text:h", NS)]
            for nombre in EXPORTADOR.CAPITULOS:
                fuente = (EXPORTADOR.DESARROLLO / nombre).read_text()
                titulo = next(l[2:] for l in fuente.splitlines() if l.startswith("# "))
                self.assertIn(titulo, titulos)
            self.assertEqual(len(figuras), 7)
            self.assertEqual(len(contenido.findall(".//draw:frame", NS)), 7)
            self.assertGreaterEqual(len(contenido.findall(".//table:table", NS)), 6)
            codigos = [p for p in contenido.findall(".//text:p", NS)
                       if p.get("{" + NS["text"] + "}style-name") == "Codigo"]
            self.assertTrue(any("set_config" in "".join(p.itertext()) for p in codigos))
            self.assertTrue(any(p.find("text:line-break", NS) is not None for p in codigos))
            texto = "".join(contenido.itertext())
            self.assertNotIn("```", texto)
            self.assertNotIn("Glosario del Programa", texto)
            enlaces = [a.get("{" + NS["xlink"] + "}href") for a in contenido.findall(".//text:a", NS)]
            self.assertIn("#1-arquitectura-y-módulos", enlaces)
            self.assertTrue(any(e.endswith("cmd/etf/modulos.go") for e in enlaces))

    def test_mermaid_rechaza_sintaxis_que_no_puede_dibujar(self):
        with self.assertRaises(ValueError):
            EXPORTADOR.diagrama_mermaid(["sequenceDiagram", "A->>B: mensaje"])
        with self.assertRaises(ValueError):
            EXPORTADOR.diagrama_mermaid(["flowchart LR", "A -- texto --> B"])

    def test_enlaces_anteriores_conservan_su_base_y_glosario(self):
        self.assertIn('xlink:href="#lectura"', EXPORTADOR.texto("[ver](usuario.md#lectura)", True))
        esperado = (EXPORTADOR.BASE / "coordinacion.md").as_uri()
        self.assertIn(esperado, EXPORTADOR.texto("[ver](coordinacion.md)", False))


if __name__ == "__main__":
    unittest.main()
