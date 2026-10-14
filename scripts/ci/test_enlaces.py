import tempfile
import unittest
from pathlib import Path

from enlaces import comprobar


class EnlacesTest(unittest.TestCase):
    def comprobar_texto(self, texto):
        # Todo el material temporal permanece en la zona de esta tarea.
        with tempfile.TemporaryDirectory(dir=Path(__file__).parent) as directorio:
            raiz = Path(directorio).resolve()
            documento = raiz / "inicio.md"
            documento.write_text(texto, encoding="utf-8")
            (raiz / "Acta (2026).md").write_text("# Revisión\n# Revisión\n", encoding="utf-8")
            return comprobar(raiz, [documento])

    def test_rutas_con_espacios_parentesis_y_anclas(self):
        errores, locales, _ = self.comprobar_texto(
            "[Acta](<Acta (2026).md#revisión-1>)\n"
            "[Acta](Acta%20(2026).md#revisi%C3%B3n)\n"
            "[Acta](Acta (2026).md)\n"
        )
        self.assertEqual((errores, locales), ([], 3))

    def test_destino_ancla_y_referencia_rotos_fallan(self):
        errores, _, _ = self.comprobar_texto(
            "[Acta](ausente.md)\n[Acta](<Acta (2026).md#ausente>)\n[Acta][ausente]\n"
        )
        self.assertEqual(len(errores), 3)
        self.assertIn(":1: destino inexistente", errores[0])
        self.assertIn(":2: ancla inexistente", errores[1])
        self.assertIn(":3: referencia sin definir", errores[2])

    def test_referencias_y_codigos_sin_falsos_enlaces(self):
        errores, locales, externos = self.comprobar_texto(
            "[Acta][ruta]\n[ruta]: <Acta (2026).md>\n"
            "`[Acta](ausente.md)`\n```md\n[Acta](ausente.md)\n```\n"
            "[D1]\n[Portal](https://invalid.invalid/acta)\n"
        )
        self.assertEqual((errores, locales, externos), ([], 2, 1))

    def test_destinos_fuera_del_repositorio_fallan(self):
        errores, _, _ = self.comprobar_texto("[Acta](../ausente.md)\n")
        self.assertEqual(len(errores), 1)
        self.assertIn("destino fuera del repositorio", errores[0])
