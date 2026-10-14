"""Compara por página el original oficial y el Word rellenado por el motor.

Antes: ETF_DOCX_EVIDENCIA_DIR=<temporal>/generado go test
./internal/adaptadores/docx -run TestRevisionTodasLasPlantillas -count=1.
Después: python3 comparar_plantillas.py <temporal> <directorio_de_capturas>.
LibreOffice sólo se ejecuta en desarrollo con un perfil temporal aislado.
Requiere las herramientas locales soffice, pdftoppm, pdfinfo y Pillow.
"""
import argparse
import json
from pathlib import Path
import os
import subprocess
from PIL import Image, ImageDraw

RAIZ = Path(__file__).resolve().parents[4]


def ejecutar(*argumentos):
    return subprocess.run(argumentos, check=True, capture_output=True, text=True).stdout


def pdf(ruta, perfil):
    ejecutar('env', '-u', 'LD_LIBRARY_PATH', 'soffice',
             '-env:UserInstallation=' + perfil.as_uri(), '--headless',
             '--convert-to', 'pdf', '--outdir', str(ruta.parent), str(ruta))
    resultado = ruta.with_suffix('.pdf')
    if not resultado.exists():
        raise RuntimeError('documento.pdf_ausente')
    return resultado


def paginas(ruta):
    for anterior in ruta.parent.glob('pagina-*.png'):
        anterior.unlink()
    ejecutar('pdftoppm', '-scale-to', '1000', '-png', str(ruta), str(ruta.parent / 'pagina'))
    return sorted(ruta.parent.glob('pagina-*.png'))


def comparar(temporal, destino):
    fuentes = json.loads(Path(__file__).with_name('revision_fuentes.json').read_text())
    registro = []
    for ruta in sorted((temporal / 'generado').rglob('*.docx')):
        relativo = ruta.relative_to(temporal / 'generado')
        idioma = relativo.parts[0]
        clave = relativo.parent.relative_to(Path(idioma) / 'plantillas').as_posix()
        carpeta = temporal / 'original' / clave
        carpeta.mkdir(parents=True, exist_ok=True)
        fuente = Path(os.environ.get('ETF_PLANTILLAS_FUENTE', str(RAIZ / 'bin/plantillas-fuente'))) / fuentes[clave]
        original = carpeta / ('original' + fuente.suffix)
        if not original.exists():
            original.write_bytes(fuente.read_bytes())
        opdf = original.with_suffix('.pdf')
        if not opdf.exists():
            opdf = pdf(original, temporal / 'perfil')
        gpdf = pdf(ruta, temporal / 'perfil-generado')
        antes = paginas(opdf)
        despues = paginas(gpdf)
        prefijo = idioma + '-' + clave.replace('/', '-') + '-' + ruta.stem
        for anterior in destino.glob(prefijo + '-p*.png'):
            anterior.unlink()
        archivos = []
        for numero in range(max(len(antes), len(despues))):
            par = Image.new('RGB', (1440, 1050), 'white')
            lapiz = ImageDraw.Draw(par)
            lapiz.text((10, 8), 'Original: ' + fuentes[clave], fill='black')
            lapiz.text((730, 8), str(relativo), fill='black')
            for x, lista in ((0, antes), (720, despues)):
                if numero < len(lista):
                    imagen = Image.open(lista[numero]).convert('RGB')
                    imagen.thumbnail((700, 1000))
                    par.paste(imagen, (x + 10, 30))
                else:
                    lapiz.text((x + 10, 50), 'Sin pagina equivalente', fill='black')
            nombre = prefijo + '-p' + str(numero + 1).zfill(2) + '.png'
            par.save(destino / nombre, optimize=True)
            archivos.append(nombre)
        registro.append({'plantilla': str(relativo), 'fuente': fuentes[clave],
                         'paginas_original': len(antes), 'paginas_generado': len(despues),
                         'comparaciones': archivos})
        print(relativo, str(len(antes)) + ' -> ' + str(len(despues)), flush=True)
    (destino / 'inventario.json').write_text(json.dumps(registro, ensure_ascii=False, indent=2) + '\n')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('temporal', type=Path)
    parser.add_argument('destino', type=Path)
    args = parser.parse_args()
    args.destino.mkdir(parents=True, exist_ok=True)
    comparar(args.temporal.resolve(), args.destino.resolve())
