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
DESARROLLO = BASE.parent / "desarrollo"
CAPITULOS = (
    "LEEME.md", "01-arquitectura.md", "02-postgresql.md", "03-api.md",
    "04-interfaz.md", "05-paquetes.md", "06-lectores.md",
    "07-calculos-auditoria.md", "08-pruebas-entrega.md", "09-recetas.md",
)
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


def texto(linea: str, incluir_comun: bool, base: Path = BASE) -> str:
    """Conserva los enlaces; el glosario común queda dentro del propio PDF."""
    salida = []
    inicio = 0
    for enlace in re.finditer(r"\[([^\]]+)\]\(([^)]+)\)", linea):
        salida.append(escape(linea[inicio:enlace.start()]))
        destino = enlace[2]
        if incluir_comun and destino.startswith(("usuario.md#", "comun.md#")):
            destino = "#" + destino.split("#", 1)[1]
        elif not destino.startswith("#") and not urlsplit(destino).scheme:
            ruta, separador, fragmento = destino.partition("#")
            destino = (base / ruta).resolve().as_uri()
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


def fuente_programadores() -> str:
    """Reúne capítulos y convierte sus enlaces cruzados en anclas del PDF."""
    fuentes = {nombre: (DESARROLLO / nombre).read_text(encoding="utf-8") for nombre in CAPITULOS}
    titulos = {nombre: ancla(re.search(r"(?m)^# (.+)$", fuente)[1]) for nombre, fuente in fuentes.items()}

    def destino(enlace: re.Match) -> str:
        ruta, _, fragmento = enlace[1].partition("#")
        if ruta in titulos:
            return "(#" + (fragmento or titulos[ruta]) + ")"
        return enlace[0]

    return "\n\n".join(re.sub(r"\(([^)]+)\)", destino, fuente) for fuente in fuentes.values())


def diagrama_mermaid(lineas: list[str]) -> bytes:
    """Dibuja localmente el subconjunto flowchart usado por este manual."""
    direccion = re.fullmatch(r"flowchart (LR|TB)", lineas[0].strip())
    if not direccion:
        raise ValueError("El PDF admite diagramas flowchart LR o TB.")
    dot = ["digraph manual {", f"rankdir={direccion[1]};", 'node [shape=box, fontname="DejaVu Sans", fontsize=12];']
    for linea in lineas[1:]:
        linea = linea.strip()
        if not linea or linea.startswith("%%"):
            continue
        nodo = re.fullmatch(r'(\w+)\["([^"\n]+)"\]', linea)
        flecha = re.fullmatch(r"(\w+) --> (\w+)", linea)
        if nodo:
            dot.append(f"{json.dumps(nodo[1])} [label={json.dumps(nodo[2], ensure_ascii=False)}];")
        elif flecha:
            dot.append(f"{json.dumps(flecha[1])} -> {json.dumps(flecha[2])};")
        else:
            raise ValueError(f"Línea Mermaid no admitida en el PDF: {linea}")
    dot.append("}")
    return subprocess.run(["dot", "-Tpng", "-Gdpi=150"], input="\n".join(dot).encode(),
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True, timeout=30).stdout


def documento(nombre: str, destino: Path) -> None:
    desarrollo = nombre == "programadores"
    sistemas = nombre == "sistemas"
    base = DESARROLLO if desarrollo else BASE.parent / "sistemas" if sistemas else BASE
    incluir_comun = nombre in ("profesionales", "coordinacion")
    fuente = fuente_programadores() if desarrollo else (base / ("LEEME.md" if sistemas else f"{nombre}.md")).read_text(encoding="utf-8")
    if sistemas:
        for anexo in ("CONFIGURACION.md", "OPERACION.md", "DIAGNOSTICO.md", "MANSIS.md"):
            fuente += "\n\n" + (base / anexo).read_text(encoding="utf-8")
    if incluir_comun:
        if "(comun.md#" in fuente:
            comun = (BASE / "comun.md").read_text(encoding="utf-8")
            fuente += "\n\n" + comun[comun.index("## Entrada, preferencias y ayuda"):]
        else:
            comun = (BASE / "usuario.md").read_text(encoding="utf-8")
            fuente += "\n\n## Glosario del Programa" + comun.split("## Glosario del Programa", 1)[1].split(
                "## Preparar e imprimir los PDF", 1
            )[0]

    bloques: list[str] = []
    imagenes: dict[str, bytes] = {}
    lista = False
    parrafo: list[str] = []
    codigo: list[str] | None = None
    lenguaje = ""
    tabla: list[list[str]] = []

    def inline(linea: str) -> str:
        if sistemas:
            incluidos = {
                "LEEME.md": "manual-de-sistemas-de-expedientes-etf",
                "CONFIGURACION.md": "anexo-a-configuración-de-producción",
                "OPERACION.md": "anexo-b-copias-auditoría-y-conservación",
                "DIAGNOSTICO.md": "anexo-c-resolución-de-problemas",
                "MANSIS.md": "anexo-d-evidencia-del-manual-de-sistemas",
            }
            linea = re.sub(
                r"\]\((LEEME.md|CONFIGURACION.md|OPERACION.md|DIAGNOSTICO.md|MANSIS.md)(#[^)]*)?\)",
                lambda m: "](" + (m[2] or "#" + incluidos[m[1]]) + ")", linea,
            )
        if not (desarrollo or sistemas):
            return texto(linea, incluir_comun, base)
        # El código literal no se interpreta como enlaces ni como énfasis.
        resultado = "".join(
            '<text:span text:style-name="Literal">' + escape(p[1:-1]) + '</text:span>'
            if p.startswith("`") else texto(p, False, destino.parent if sistemas else base)
            for p in re.split(r"(`[^`]+`)", linea)
        )
        if sistemas:
            resultado = re.sub(r"\*\*([^*]+)\*\*", r'<text:span text:style-name="Negrita">\1</text:span>', resultado)
        return resultado

    def insertar_imagen(datos: bytes, descripcion: str) -> None:
        referencia = f"Pictures/{len(imagenes) + 1}.png"
        imagenes[referencia] = datos
        with Image.open(BytesIO(datos)) as original:
            ancho, alto = original.size
        ancho_cm = min(8.0 if ancho <= 600 else 17.6, 18.0 * ancho / alto)
        bloques.append(
            '<text:p text:style-name="Imagen">'
            f'<draw:frame draw:name="Figura{len(imagenes)}" text:anchor-type="as-char" '
            f'svg:width="{ancho_cm:.3f}cm" svg:height="{ancho_cm * alto / ancho:.3f}cm">'
            f'<svg:title>{escape(descripcion)}</svg:title>'
            f'<draw:image xlink:href="{referencia}" xlink:type="simple" '
            'xlink:show="embed" xlink:actuate="onLoad"/></draw:frame></text:p>'
        )
        bloques.append('<text:p text:style-name="Pie">' + escape(descripcion) + '</text:p>')

    def cerrar_tabla() -> None:
        if not tabla:
            return
        bloques.append('<table:table xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" table:name="Tabla' + str(len(bloques)) + '">')
        if sistemas:
            for indice in range(len(tabla[0])):
                bloques.append(f'<table:table-column table:style-name="Col{len(tabla[0])}_{indice}"/>')
        else:
            bloques.append(f'<table:table-column table:number-columns-repeated="{len(tabla[0])}"/>')
        for i, fila in enumerate(tabla):
            if i == 0 and desarrollo:
                bloques.append('<table:table-header-rows>')
            bloques.append('<table:table-row>')
            for celda in fila:
                estilo = ("CabeceraTabla" if i == 0 else "CeldaTabla") if sistemas else "Cuerpo"
                estilo_celda = ' table:style-name="Celda"' if desarrollo else ""
                bloques.append('<table:table-cell' + estilo_celda + ' office:value-type="string"><text:p text:style-name="' + estilo + '">' + inline(celda) + '</text:p></table:table-cell>')
            bloques.append('</table:table-row>')
            if i == 0 and desarrollo:
                bloques.append('</table:table-header-rows>')
        bloques.append('</table:table>')
        tabla.clear()

    def vaciar() -> None:
        if parrafo:
            bloques.append('<text:p text:style-name="Cuerpo">' + inline(" ".join(parrafo)) + '</text:p>')
            parrafo.clear()

    def cerrar_lista() -> None:
        nonlocal lista
        if lista:
            bloques.append("</text:list>")
            lista = False

    for linea in fuente.splitlines():
        if (desarrollo or sistemas) and linea.startswith("```"):
            vaciar()
            cerrar_lista()
            cerrar_tabla()
            if codigo is None:
                codigo, lenguaje = [], linea[3:].strip()
            else:
                if lenguaje == "mermaid":
                    descripcion = next((l.strip()[2:].strip() for l in codigo if l.strip().startswith("%%")), "Diagrama de flujo del apartado.")
                    insertar_imagen(diagrama_mermaid(codigo), descripcion)
                elif sistemas:
                    for literal in codigo:
                        literal = re.sub(r" +", lambda m: f'<text:s text:c="{len(m[0])}"/>', escape(literal))
                        bloques.append('<text:p text:style-name="Codigo">' + literal + '</text:p>')
                else:
                    contenido_codigo = '<text:line-break/>'.join(
                        re.sub(r" {2,}", lambda m: f'<text:s text:c="{len(m[0])}"/>', escape(l)) for l in codigo
                    )
                    bloques.append('<text:p text:style-name="Codigo">' + contenido_codigo + '</text:p>')
                codigo = None
            continue
        if codigo is not None:
            codigo.append(linea)
            continue
        if (desarrollo or sistemas) and not linea.startswith("| "):
            cerrar_tabla()
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
            estilo = "Anexo" if sistemas and titulo.startswith("Anexo ") else "Titulo" if nivel == 1 else f"H{nivel}"
            bloques.append(
                f'<text:h text:style-name="{estilo}" text:outline-level="{nivel}">'
                f'<text:bookmark text:name={quoteattr(ancla(titulo))}/>{escape(titulo)}</text:h>'
            )
        elif imagen:
            vaciar()
            cerrar_lista()
            ruta = (base / imagen[2]).resolve()
            if desarrollo:
                insertar_imagen(ruta.read_bytes(), imagen[1])
                continue
            with Image.open(ruta) as original:
                ancho, alto = original.size
                # Las capturas largas se dividen sin reducir el tamaño de sus letras.
                movil_administracion = nombre == "administracion" and ancho < 600
                tramo = alto if movil_administracion else int(ancho * (3 if ancho < 600 else 0.92))
                partes = (alto + tramo - 1) // tramo
                for parte, y in enumerate(range(0, alto, tramo), 1):
                    recorte = original.crop((0, y, ancho, min(y + tramo, alto)))
                    datos = BytesIO()
                    recorte.save(datos, format="PNG")
                    referencia = f"Pictures/{len(imagenes) + 1}.png"
                    imagenes[referencia] = datos.getvalue()
                    ancho_cm = min(7.5, 22 * ancho / alto) if movil_administracion else (7.5 if ancho < 600 else 17.6)
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
            inicio_paso = f'<text:list-item text:start-value="{int(linea.split(".", 1)[0])}">' if sistemas else '<text:list-item>'
            bloques.append(inicio_paso + '<text:p text:style-name="Paso">' + inline(paso[1]) + '</text:p></text:list-item>')
        elif linea.startswith("| "):
            vaciar()
            cerrar_lista()
            celdas = [c.strip() for c in linea.strip("| ").split("|")]
            if desarrollo or sistemas:
                if not all(re.fullmatch(r":?-+:?", c) for c in celdas):
                    tabla.append(celdas)
                continue
            if celdas[0] not in ("Término", "---"):
                bloques.append('<text:p text:style-name="Cuerpo">' + texto(": ".join(celdas), nombre != "administracion") + '</text:p>')
        elif sistemas and linea.startswith("- "):
            vaciar()
            cerrar_lista()
            bloques.append('<text:p text:style-name="Cuerpo">• ' + inline(linea[2:]) + '</text:p>')
        else:
            cerrar_lista()
            parrafo.append(linea)
    vaciar()
    cerrar_lista()
    cerrar_tabla()
    if codigo is not None:
        raise ValueError("Bloque de código sin cerrar.")

    tamanio = "11pt"
    titulo_pt = "20pt"
    apartado_pt = "15pt"
    interlineado = "115%"
    mantener_imagen = ' fo:keep-together="always"' if nombre == "administracion" else ""
    automaticos = f'''
      <style:style style:name="Cuerpo" style:family="paragraph"><style:paragraph-properties fo:margin-bottom="0.15cm" fo:orphans="2" fo:widows="2"/><style:text-properties fo:font-size="{tamanio}"/></style:style>
      <style:style style:name="Paso" style:family="paragraph" style:parent-style-name="Cuerpo"><style:paragraph-properties fo:keep-together="always"/></style:style>
      <style:style style:name="Titulo" style:family="paragraph"><style:paragraph-properties fo:keep-with-next="always" fo:margin-bottom="0.4cm"/><style:text-properties fo:font-size="{titulo_pt}" fo:font-weight="bold"/></style:style>
      <style:style style:name="H2" style:family="paragraph"><style:paragraph-properties fo:keep-with-next="always" fo:margin-top="0.4cm" fo:margin-bottom="0.2cm"/><style:text-properties fo:font-size="{apartado_pt}" fo:font-weight="bold"/></style:style>
      <style:style style:name="H3" style:family="paragraph"><style:paragraph-properties fo:keep-with-next="always" fo:margin-top="0.25cm"/><style:text-properties fo:font-size="12pt" fo:font-weight="bold"/></style:style>
      <style:style style:name="Imagen" style:family="paragraph"><style:paragraph-properties fo:keep-with-next="always"{mantener_imagen} fo:text-align="center"/></style:style>
      <style:style style:name="Pie" style:family="paragraph"><style:paragraph-properties fo:margin-bottom="0.25cm"/><style:text-properties fo:font-size="9pt"/></style:style>
      <text:list-style style:name="Pasos"><text:list-level-style-number text:level="1" style:num-format="1" style:num-suffix="."><style:list-level-properties text:space-before="0cm" text:min-label-width="0.65cm"/></text:list-level-style-number></text:list-style>
    '''
    if desarrollo:
        automaticos += '''
          <style:style style:name="Literal" style:family="text"><style:text-properties fo:font-family="DejaVu Sans Mono" fo:font-size="9pt"/></style:style>
          <style:style style:name="Codigo" style:family="paragraph"><style:paragraph-properties fo:margin-top="0.15cm" fo:margin-bottom="0.25cm" fo:keep-together="always"/><style:text-properties fo:font-family="DejaVu Sans Mono" fo:font-size="8pt"/></style:style>
          <style:style style:name="Celda" style:family="table-cell"><style:table-cell-properties fo:padding="0.12cm" fo:border="0.5pt solid #808080"/></style:style>
        '''
    if sistemas:
        automaticos += '<style:style style:name="Anexo" style:family="paragraph" style:parent-style-name="H2"><style:paragraph-properties fo:break-before="page"/></style:style>'
        automaticos += '<style:style style:name="Codigo" style:family="paragraph"><style:paragraph-properties fo:line-height="100%" fo:margin-bottom="0cm"/><style:text-properties fo:font-family="Liberation Mono" fo:font-size="8pt"/></style:style>'
        automaticos += '<style:style style:name="Literal" style:family="text"><style:text-properties fo:font-family="Liberation Mono" fo:font-size="9pt"/></style:style>'
        automaticos += '<style:style style:name="Negrita" style:family="text"><style:text-properties fo:font-weight="bold"/></style:style>'
        automaticos += '<style:style style:name="CeldaTabla" style:family="paragraph"><style:paragraph-properties fo:margin-top="0.1cm" fo:margin-bottom="0.15cm"/><style:text-properties fo:font-size="9pt"/></style:style>'
        automaticos += '<style:style style:name="CabeceraTabla" style:family="paragraph" style:parent-style-name="CeldaTabla"><style:text-properties fo:font-weight="bold"/></style:style>'
        for numero, anchos in ((2, (6.0, 11.6)), (3, (4.0, 6.8, 6.8))):
            for indice, ancho in enumerate(anchos):
                automaticos += f'<style:style style:name="Col{numero}_{indice}" style:family="table-column"><style:table-column-properties style:column-width="{ancho}cm"/></style:style>'
    contenido = f'<?xml version="1.0" encoding="UTF-8"?><office:document-content {NS} office:version="1.2"><office:automatic-styles>{automaticos}</office:automatic-styles><office:body><office:text>{"".join(bloques)}</office:text></office:body></office:document-content>'
    estilos = f'''<?xml version="1.0" encoding="UTF-8"?><office:document-styles {NS} office:version="1.2">
      <office:styles><style:default-style style:family="paragraph"><style:paragraph-properties fo:line-height="{interlineado}"/></style:default-style></office:styles>
      <office:automatic-styles><style:page-layout style:name="A4"><style:page-layout-properties fo:page-width="21cm" fo:page-height="29.7cm" fo:margin-top="1.5cm" fo:margin-bottom="1.5cm" fo:margin-left="1.7cm" fo:margin-right="1.7cm"/></style:page-layout></office:automatic-styles>
      <office:master-styles><style:master-page style:name="Standard" style:page-layout-name="A4"><style:footer><text:p>Expedientes ETF · <text:page-number/></text:p></style:footer></style:master-page></office:master-styles>
    </office:document-styles>'''
    manifiesto = ['<manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.text"/>', '<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>', '<manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>']
    autor = re.search(r"(?m)^Autor: (.+)$", fuente)
    metadatos = ('<?xml version="1.0" encoding="UTF-8"?><office:document-meta '
                 'xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" '
                 'xmlns:dc="http://purl.org/dc/elements/1.1/" '
                 'xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" office:version="1.2">'
                 '<office:meta><meta:initial-creator>' + escape(autor[1]) + '</meta:initial-creator>'
                 '<dc:creator>' + escape(autor[1]) + '</dc:creator></office:meta></office:document-meta>') if autor else None
    if desarrollo or sistemas:
        metadatos = f'<?xml version="1.0" encoding="UTF-8"?><office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" office:version="1.2"><office:meta><dc:title>Manual de {nombre} de Expedientes ETF</dc:title><meta:initial-creator>Expedientes ETF</meta:initial-creator><dc:creator>Expedientes ETF</dc:creator></office:meta></office:document-meta>'
    if metadatos:
        manifiesto.append('<manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>')
    manifiesto.extend(f'<manifest:file-entry manifest:full-path="{ref}" manifest:media-type="image/png"/>' for ref in imagenes)
    with ZipFile(destino, "w", compression=ZIP_DEFLATED) as archivo:
        archivo.writestr("mimetype", "application/vnd.oasis.opendocument.text", compress_type=ZIP_STORED)
        archivo.writestr("content.xml", contenido)
        archivo.writestr("styles.xml", estilos)
        if metadatos:
            archivo.writestr("meta.xml", metadatos)
        archivo.writestr("META-INF/manifest.xml", '<?xml version="1.0"?><manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">' + "".join(manifiesto) + '</manifest:manifest>')
        for referencia, datos in imagenes.items():
            archivo.writestr(referencia, datos)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manual", action="append", choices=(*MANUALES, "sistemas", "programadores"),
                        help="Exporta sólo el manual indicado; se puede repetir.")
    parser.add_argument("--salida", type=Path)
    args = parser.parse_args()
    manuales = tuple(dict.fromkeys(args.manual or MANUALES))
    if not shutil.which("soffice"):
        parser.error("Necesitas LibreOffice local (soffice).")
    temporales = Path(os.environ.get("TMPDIR", str(BASE.parents[1] / "bin/temporales")))
    temporales.mkdir(parents=True, exist_ok=True)
    for herramienta in ("exiftool", "qpdf"):
        if not shutil.which(herramienta):
            parser.error(f"Necesitas {herramienta} local para limpiar metadatos PDF.")
    if "programadores" in manuales and not shutil.which("dot"):
        parser.error("El manual de programadores necesita Graphviz local (dot).")
    with tempfile.TemporaryDirectory(prefix="f506-pdf-", dir=temporales) as temporal:
        carpeta = Path(temporal)
        for nombre in manuales:
            sistemas = nombre == "sistemas"
            salida = args.salida or (DESARROLLO if nombre == "programadores" else BASE.parent / "sistemas" if sistemas else BASE)
            salida = salida.resolve()
            salida.mkdir(parents=True, exist_ok=True)
            documento(nombre, carpeta / f"{nombre}.odt")
            opciones_pdf = (
                'pdf:writer_pdf_Export:{"UseLosslessCompression":{"type":"boolean","value":"true"},'
                '"ReduceImageResolution":{"type":"boolean","value":"false"},'
                '"ExportBookmarks":{"type":"boolean","value":"true"}'
                + (',"UseTaggedPDF":{"type":"boolean","value":"true"}' if sistemas or nombre == "programadores" else '')
                + (',"ExportLinksRelativeFsys":{"type":"boolean","value":"true"}' if sistemas else ',"ExportLinksRelative":{"type":"boolean","value":"true"}')
                + '}'
            )
            # Sistemas conserva los enlaces relativos al árbol publicado.
            # LibreOffice calcula esas rutas desde el destino de conversión.
            salida_pdf = salida if sistemas else carpeta
            subprocess.run([
                "env", "-u", "LD_LIBRARY_PATH", "soffice",
                f"-env:UserInstallation={(carpeta / 'perfil').as_uri()}",
                "--headless", "--convert-to", opciones_pdf,
                "--outdir", str(salida_pdf), str(carpeta / f"{nombre}.odt"),
            ], check=True, timeout=180, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            pdf = salida_pdf / f"{nombre}.pdf"
            if not pdf.is_file() or pdf.stat().st_size == 0:
                raise RuntimeError(f"No se ha generado {pdf.name}.")
            subprocess.run(["exiftool", "-overwrite_original", "-all=",
                            "-Author=Expedientes ETF", "-Creator=Expedientes ETF",
                            "-Producer=Expedientes ETF", str(pdf)],
                           check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            if not sistemas:
                limpio = carpeta / f"{nombre}-publico.pdf"
                enlaces = enlaces_relativos(pdf, carpeta)
                subprocess.run(["qpdf", "--object-streams=disable",
                                f"--update-from-json={enlaces}", str(pdf), str(limpio)],
                               check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                shutil.copyfile(limpio, salida / pdf.name)
            print(f"PDF-OK: {pdf.name}")


if __name__ == "__main__":
    main()
