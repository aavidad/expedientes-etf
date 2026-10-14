import unittest
from comprobar import infracciones


class ColoresTest(unittest.TestCase):
    def test_rechaza_formatos_y_nombres(self):
        for dato in ['color: #abc;', '"#abcdef12"', 'background: rgb(0 0 0);', 'color: hsl(0 0% 0%);', 'color: rebeccapurple;', '"white"', 'color: oklch(20% 0 0);']:
            self.assertTrue(infracciones(dato, css=True), dato)

    def test_permite_tokens_y_comentarios(self):
        for dato in ['color: var(--color-texto);', '// color: red', '/* #abcdef */', 'var color = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)']:
            self.assertFalse(infracciones(dato, css=True), dato)


if __name__ == '__main__':
    unittest.main()
