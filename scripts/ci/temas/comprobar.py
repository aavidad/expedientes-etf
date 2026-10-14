#!/usr/bin/env python3
"""Colores de tema sólo en sus paquetes de datos; excluye pruebas y comentarios."""
import pathlib
import hashlib
import json
import re
import sys

RAIZ = pathlib.Path(__file__).resolve().parents[3]
# Lista de nombres CSS (también transparent); currentColor hereda un token y sí se admite.
NOMBRES = set('''aliceblue antiquewhite aqua aquamarine azure beige bisque black blanchedalmond blue blueviolet brown burlywood cadetblue chartreuse chocolate coral cornflowerblue cornsilk crimson cyan darkblue darkcyan darkgoldenrod darkgray darkgrey darkgreen darkkhaki darkmagenta darkolivegreen darkorange darkorchid darkred darksalmon darkseagreen darkslateblue darkslategray darkslategrey darkturquoise darkviolet deeppink deepskyblue dimgray dimgrey dodgerblue firebrick floralwhite forestgreen fuchsia gainsboro ghostwhite gold goldenrod gray grey green greenyellow honeydew hotpink indianred indigo ivory khaki lavender lavenderblush lawngreen lemonchiffon lightblue lightcoral lightcyan lightgoldenrodyellow lightgray lightgrey lightgreen lightpink lightsalmon lightseagreen lightskyblue lightslategray lightslategrey lightsteelblue lightyellow lime limegreen linen magenta maroon mediumaquamarine mediumblue mediumorchid mediumpurple mediumseagreen mediumslateblue mediumspringgreen mediumturquoise mediumvioletred midnightblue mintcream mistyrose moccasin navajowhite navy oldlace olive olivedrab orange orangered orchid palegoldenrod palegreen paleturquoise palevioletred papayawhip peachpuff peru pink plum powderblue purple rebeccapurple red rosybrown royalblue saddlebrown salmon sandybrown seagreen seashell sienna silver skyblue slateblue slategray slategrey snow springgreen steelblue tan teal thistle tomato transparent turquoise violet wheat white whitesmoke yellow yellowgreen'''.split())
COLOR = re.compile(r'#[\da-f]{3}(?:[\da-f]{1}|[\da-f]{3}(?:[\da-f]{2})?)?\b|\b(?:rgba?|hsla?|hwb|lab|lch|oklab|oklch|color|color-mix)\s*\(', re.I)
# Tokeniza primero: no elimina URL ni // dentro de cadenas.
LEXICO = re.compile(r'//[^\n]*|/\*[\s\S]*?\*/|"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'|`[^`]*`', re.M)


def infracciones(contenido, css=False):
    limpio = LEXICO.sub(lambda m: '' if m[0].startswith(('//', '/*')) else m[0], contenido)
    hallazgos = [m[0] for m in COLOR.finditer(limpio)]
    cadenas = [m[0][1:-1].lower() for m in LEXICO.finditer(limpio) if not m[0].startswith(('//', '/*'))]
    hallazgos += [valor for valor in cadenas if valor in NOMBRES]
    if css:
        for declaracion in re.finditer(r':([^;{}]+)[;}]', limpio):
            hallazgos += [p for p in re.findall(r'\b[a-z]+\b', declaracion[1].lower()) if p in NOMBRES]
    return hallazgos


def comprobar(raiz=RAIZ):
    fallos = []
    rutas = list((raiz / 'web/src').rglob('*')) + list(raiz.rglob('*.go'))
    for ruta in sorted(set(rutas)):
        relativo = ruta.relative_to(raiz)
        if not ruta.is_file() or ruta.suffix not in {'.go', '.ts', '.tsx', '.css'}:
            continue
        if 'node_modules' in relativo.parts or relativo.parts[0] == 'scripts' or ruta.name.endswith('_test.go') or re.search(r'\.(test|spec)\.', ruta.name):
            continue
        for color in infracciones(ruta.read_text(), ruta.suffix == '.css'):
            fallos.append(f'{relativo}: color fijo {color}')
    for fuente in json.loads((raiz / 'temas/fuentes.lock.json').read_text()):
        for nombre in ['expedientes-etf', 'alto-contraste']:
            ruta = raiz / 'temas' / nombre / fuente['ruta']
            if hashlib.sha256(ruta.read_bytes()).hexdigest() != fuente['sha256']:
                fallos.append(f'{ruta}: suma de fuente distinta de fuentes.lock.json')
    return fallos


if __name__ == '__main__':
    fallos = comprobar()
    if fallos:
        print('\n'.join(fallos), file=sys.stderr)
        sys.exit(1)
    print('Temas: sin colores fijos en Go ni web/src.')
