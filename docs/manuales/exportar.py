#!/usr/bin/env python3
"""Exporta los manuales a PDF con LibreOffice local, sin conexión externa."""

from __future__ import annotations

import argparse
from io import BytesIO
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
from xml.sax.saxutils import escape, quoteattr
from zipfile import ZIP_STORED, ZIP_DEFLATED, ZipFile
from urllib.parse import unquote, urlsplit

from PIL import Image


BASE = Path(__file__).resolve().parent
MANUALES = ("profesionales", "coordinacion", "administracion")
NS = (
    'xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" '
    'xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" '
    'xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" '
    'xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" '
    'xmlns:draw="urn:oasis:names:tc:opendocument:xmlns:drawing:1.0" '
    'xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0" '
    'xmlns:xlink="http://www.w3.org/1999/xlink"'
)


def ancla(titulo: str) -> str:
    return re.sub(r"[^\w -]", "", titulo.lower()).replace(" ", "-")


def texto(linea: str, incluir_comun: bool) -> str:
    """Conserva los enlaces; el glosario común queda dentro del propio PDF."""
    salida = []
    inicio = 0
    for enlace in re.finditer(r"\[([^\]]+)\]\(([^)]+)\)", linea):
        salida.append(escape(linea[inicio:enlace.start()]))
        destino = enlace[2]
        if incluir_comun and destino.startswith("usuario.md#"):
            destino = destino[len("usuario.md"):]
        elif not destino.startswith("#") and not urlsplit(destino).scheme:
            ruta, separador, fragmento = destino.partition("#")
            destino = (BASE / ruta).resolve().as_uri()
            if separador:
                destino += "#" + fragmento
        salida.append(
            f'<text:a xlink:href={quoteattr(destino)}>{escape(enlace[1])}</text:a>'
        )
        inicio = enlace.end()
    salida.append(escape(linea[inicio:]))
    return "".join(salida)


def enlaces_relativos(pdf: Path, carpeta: Path) -> Path:
    """Convierte las acciones de fichero a rutas junto al manual distribuido."""
    resultado = subprocess.run(
        ["qpdf", "--json-output=2", "--json-stream-data=none", str(pdf)],
        check=True, capture_output=True, text=True,
    )
    modelo = json.loads(resultado.stdout)
    cambios = {}

    def corregir(valor) -> bool:
        cambiado = False
        if isinstance(valor, dict):
            uri = valor.get("/URI", "")
            if isinstance(uri, str) and uri.startswith("u:file:"):
                url = urlsplit(uri[2:])
                ruta = Path(unquote(url.path)).resolve()
                if url.netloc or not ruta.is_relative_to(BASE.parents[1]):
                    raise ValueError("Enlace PDF fuera del repositorio.")
                relativo = Path(os.path.relpath(ruta, BASE)).as_posix()
                if url.fragment:
                    relativo += "#" + url.fragment
                valor["/URI"] = "u:" + relativo
                cambiado = True
            for hijo in valor.values():
                cambiado = corregir(hijo) or cambiado
        elif isinstance(valor, list):
            for hijo in valor:
                cambiado = corregir(hijo) or cambiado
        return cambiado

    for identificador, objeto in modelo["qpdf"][1].items():
        if corregir(objeto):
            cambios[identificador] = objeto
    destino = carpeta / "enlaces.json"
    destino.write_text(json.dumps({"qpdf": [modelo["qpdf"][0], cambios]}), encoding="utf-8")
    return destino


def documento(nombre: str, destino: Path) -> None:
    fuente = (BASE / f"{nombre}.md").read_text(encoding="utf-8")
    if nombre != "administracion":
        comun = (BASE / "usuario.md").read_text(encoding="utf-8")
        fuente += "\n\n## Glosario del Programa" + comun.split("## Glosario del Programa", 1)[1].split(
            "## Preparar e imprimir los PDF", 1
        )[0]

    bloques: list[str] = []
    imagenes: dict[str, bytes] = {}
    lista = False
    parrafo: list[str] = []

    def vaciar() -> None:
        if parrafo:
            bloques.append('<text:p text:style-name="Cuerpo">' + texto(" ".join(parrafo), nombre != "administracion") + '</text:p>')
            parrafo.clear()

    def cerrar_lista() -> None:
        nonlocal lista
        if lista:
            bloques.append("</text:list>")
            lista = False

    for linea in fuente.splitlines():
        imagen = re.fullmatch(r"!\[([^\]]*)\]\(([^)]+)\)", linea)
        encabezado = re.match(r"^(#{1,3}) (.+)$", linea)
        paso = re.match(r"^\d+\. (.+)$", linea)
        if not linea.strip():
            vaciar()
            cerrar_lista()
        elif encabezado:
            vaciar()
            cerrar_lista()
            nivel, titulo = len(encabezado[1]), encabezado[2]
            estilo = "Titulo" if nivel == 1 else f"H{nivel}"
            bloques.append(
                f'<text:h text:style-name="{estilo}" text:outline-level="{nivel}">'
                f'<text:bookmark text:name={quoteattr(ancla(titulo))}/>{escape(titulo)}</text:h>'
            )
        elif imagen:
            vaciar()
            cerrar_lista()
            ruta = (BASE / imagen[2]).resolve()
            with Image.open(ruta) as original:
                ancho, alto = original.size
                # Las capturas largas se dividen sin reducir el tamaño de sus letras.
                tramo = int(ancho * 0.92)
                partes = (alto + tramo - 1) // tramo
                for parte, y in enumerate(range(0, alto, tramo), 1):
                    recorte = original.crop((0, y, ancho, min(y + tramo, alto)))
                    datos = BytesIO()
                    recorte.save(datos, format="PNG")
                    referencia = f"Pictures/{len(imagenes) + 1}.png"
                    imagenes[referencia] = datos.getvalue()
                    ancho_cm = 7.5 if nombre == "administracion" else 17.6
                    alto_cm = ancho_cm * recorte.height / ancho
                    bloques.append(
                        '<text:p text:style-name="Imagen">'
                        f'<draw:frame draw:name="Captura{len(imagenes)}" text:anchor-type="as-char" '
                        f'svg:width="{ancho_cm:.3f}cm" svg:height="{alto_cm:.3f}cm">'
                        f'<svg:title>{escape(imagen[1])}</svg:title>'
                        f'<draw:image xlink:href="{referencia}" xlink:type="simple" '
                        'xlink:show="embed" xlink:actuate="onLoad"/></draw:frame></text:p>'
                    )
                    continuacion = f" (parte {parte} de {partes})" if partes > 1 else ""
                    bloques.append('<text:p text:style-name="Pie">' + escape(imagen[1] + continuacion) + '</text:p>')
        elif paso:
            vaciar()
            if not lista:
                bloques.append('<text:list text:style-name="Pasos">')
                lista = True
            bloques.append('<text:list-item><text:p text:style-name="Paso">' + texto(paso[1], nombre != "administracion") + '</text:p></text:list-item>')
        elif linea.startswith("| "):
            vaciar()
            cerrar_lista()
            celdas = [c.strip() for c in linea.strip("| ").split("|")]
            if celdas[0] not in ("Término", "---"):
                bloques.append('<text:p text:style-name="Cuerpo">' + texto(": ".join(celdas), nombre != "administracion") + '</text:p>')
        else:
            cerrar_lista()
            parrafo.append(linea)
    vaciar()
    cerrar_lista()

    tamanio = "10pt" if nombre == "administracion" else "11pt"
    titulo_pt = "18pt" if nombre == "administracion" else "20pt"
    apartado_pt = "13pt" if nombre == "administracion" else "15pt"
    interlineado = "105%" if nombre == "administracion" else "115%"
    automaticos = f'''
      <style:style style:name="Cuerpo" style:family="paragraph"><style:paragraph-properties fo:margin-bottom="0.15cm" fo:orphans="2" fo:widows="2"/><style:text-properties fo:font-size="{tamanio}"/></style:style>
      <style:style style:name="Paso" style:family="paragraph" style:parent-style-name="Cuerpo"><style:paragraph-properties fo:keep-together="always"/></style:style>
      <style:style style:name="Titulo" style:family="paragraph"><style:paragraph-properties fo:keep-with-next="always" fo:margin-bottom="0.4cm"/><style:text-properties fo:font-size="{titulo_pt}" fo:font-weight="bold"/></style:style>
      <style:style style:name="H2" style:family="paragraph"><style:paragraph-properties fo:keep-with-next="always" fo:margin-top="0.4cm" fo:margin-bottom="0.2cm"/><style:text-properties fo:font-size="{apartado_pt}" fo:font-weight="bold"/></style:style>
      <style:style style:name="H3" style:family="paragraph"><style:paragraph-properties fo:keep-with-next="always" fo:margin-top="0.25cm"/><style:text-properties fo:font-size="12pt" fo:font-weight="bold"/></style:style>
      <style:style style:name="Imagen" style:family="paragraph"><style:paragraph-properties fo:keep-with-next="always" fo:text-align="center"/></style:style>
      <style:style style:name="Pie" style:family="paragraph"><style:paragraph-properties fo:margin-bottom="0.25cm"/><style:text-properties fo:font-size="9pt"/></style:style>
      <text:list-style style:name="Pasos"><text:list-level-style-number text:level="1" style:num-format="1" style:num-suffix="."><style:list-level-properties text:space-before="0cm" text:min-label-width="0.65cm"/></text:list-level-style-number></text:list-style>
    '''
    contenido = f'<?xml version="1.0" encoding="UTF-8"?><office:document-content {NS} office:version="1.2"><office:automatic-styles>{automaticos}</office:automatic-styles><office:body><office:text>{"".join(bloques)}</office:text></office:body></office:document-content>'
    estilos = f'''<?xml version="1.0" encoding="UTF-8"?><office:document-styles {NS} office:version="1.2">
      <office:styles><style:default-style style:family="paragraph"><style:paragraph-properties fo:line-height="{interlineado}"/></style:default-style></office:styles>
      <office:automatic-styles><style:page-layout style:name="A4"><style:page-layout-properties fo:page-width="21cm" fo:page-height="29.7cm" fo:margin-top="1.5cm" fo:margin-bottom="1.5cm" fo:margin-left="1.7cm" fo:margin-right="1.7cm"/></style:page-layout></office:automatic-styles>
      <office:master-styles><style:master-page style:name="Standard" style:page-layout-name="A4"><style:footer><text:p>Expedientes ETF · <text:page-number/></text:p></style:footer></style:master-page></office:master-styles>
    </office:document-styles>'''
    manifiesto = ['<manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.text"/>', '<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>', '<manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>']
    manifiesto.extend(f'<manifest:file-entry manifest:full-path="{ref}" manifest:media-type="image/png"/>' for ref in imagenes)
    with ZipFile(destino, "w", compression=ZIP_DEFLATED) as archivo:
        archivo.writestr("mimetype", "application/vnd.oasis.opendocument.text", compress_type=ZIP_STORED)
        archivo.writestr("content.xml", contenido)
        archivo.writestr("styles.xml", estilos)
        archivo.writestr("META-INF/manifest.xml", '<?xml version="1.0"?><manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">' + "".join(manifiesto) + '</manifest:manifest>')
        for referencia, datos in imagenes.items():
            archivo.writestr(referencia, datos)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--salida", type=Path, default=BASE)
    args = parser.parse_args()
    if not shutil.which("soffice"):
        parser.error("Necesitas LibreOffice local (soffice).")
    temporales = Path(os.environ.get("TMPDIR", str(BASE.parents[1] / "bin/temporales")))
    temporales.mkdir(parents=True, exist_ok=True)
    for herramienta in ("exiftool", "qpdf"):
        if not shutil.which(herramienta):
            parser.error(f"Necesitas {herramienta} local para limpiar metadatos PDF.")
    args.salida.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="f506-pdf-", dir=temporales) as temporal:
        carpeta = Path(temporal)
        for nombre in MANUALES:
            documento(nombre, carpeta / f"{nombre}.odt")
        subprocess.run([
            "env", "-u", "LD_LIBRARY_PATH", "soffice",
            f"-env:UserInstallation={(carpeta / 'perfil').as_uri()}",
            "--headless", "--convert-to",
            'pdf:writer_pdf_Export:{"UseLosslessCompression":{"type":"boolean","value":"true"},'
            '"ReduceImageResolution":{"type":"boolean","value":"false"},'
            '"ExportBookmarks":{"type":"boolean","value":"true"},"ExportLinksRelative":{"type":"boolean","value":"true"}}',
            "--outdir", str(carpeta),
            *(str(carpeta / f"{nombre}.odt") for nombre in MANUALES),
        ], check=True, timeout=180, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        for nombre in MANUALES:
            pdf = carpeta / f"{nombre}.pdf"
            if not pdf.is_file() or pdf.stat().st_size == 0:
                raise RuntimeError(f"No se ha generado {pdf.name}.")
            subprocess.run(["exiftool", "-overwrite_original", "-all=",
                            "-Author=Expedientes ETF", "-Creator=Expedientes ETF",
                            "-Producer=Expedientes ETF", str(pdf)],
                           check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            limpio = carpeta / f"{nombre}-publico.pdf"
            enlaces = enlaces_relativos(pdf, carpeta)
            subprocess.run(["qpdf", "--object-streams=disable",
                            f"--update-from-json={enlaces}", str(pdf), str(limpio)],
                           check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            shutil.copyfile(limpio, args.salida / pdf.name)
            print(f"PDF-OK: {pdf.name}")


if __name__ == "__main__":
    main()
