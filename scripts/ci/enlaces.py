#!/usr/bin/env python3
"""Comprueba rutas y anclas Markdown del repositorio, sin acceder a la red."""

import re
import subprocess
import sys
from pathlib import Path
from urllib.parse import unquote, urlsplit


def sin_codigo(texto, inline=True):
    texto = re.sub(r"(?ms)^ {0,3}(`{3,}|~{3,})[^\n]*\n.*?^ {0,3}\1[^\n]*$",
                   lambda m: "\n" * m[0].count("\n"), texto)
    if inline:
        texto = re.sub(r"(`+)[^\n]*?\1", lambda m: " " * len(m[0]), texto)
    return texto


def destino(texto):
    texto = texto.strip()
    if texto.startswith("<"):
        return texto[1:texto.index(">")]
    # Admitir espacios en rutas existentes; quitar únicamente títulos opcionales.
    return re.sub(r'\s+[\'"].*?[\'"]\s*$', "", texto)


def enlaces(texto):
    texto = sin_codigo(texto)
    referencias = {}
    for m in re.finditer(r"(?m)^ {0,3}\[([^]\n]+)\]:\s*(.+)$", texto):
        referencias[" ".join(m[1].lower().split())] = destino(m[2])
        yield texto.count("\n", 0, m.start()) + 1, destino(m[2])
    for m in re.finditer(r"!?\[([^]\n]*)\]\(", texto):
        inicio = m.end()
        nivel, angular, escapado = 1, False, False
        for i in range(inicio, len(texto)):
            c = texto[i]
            if escapado:
                escapado = False
                continue
            if c == "\\":
                escapado = True
            elif c == "<":
                angular = True
            elif c == ">":
                angular = False
            elif not angular and c == "(":
                nivel += 1
            elif not angular and c == ")":
                nivel -= 1
                if nivel == 0:
                    yield texto.count("\n", 0, m.start()) + 1, destino(texto[inicio:i])
                    break
    for m in re.finditer(r"!?\[([^]\n]+)\](?:\[([^]\n]*)\])?(?![\w(\[]|:)", texto):
        clave = " ".join((m[2] or m[1]).lower().split())
        if clave in referencias:
            yield texto.count("\n", 0, m.start()) + 1, referencias[clave]
        elif m[2]:
            # Una referencia explícita sin definición es un enlace roto.
            yield texto.count("\n", 0, m.start()) + 1, None


def anclas(texto):
    texto = sin_codigo(texto, inline=False)
    titulos = re.findall(r"(?m)^ {0,3}#{1,6}\s+(.+?)(?:\s+#+)?\s*$", texto)
    titulos += re.findall(r"(?m)^([^\n]+)\n {0,3}(?:=+|-+)\s*$", texto)
    usadas = set(re.findall(r'\b(?:id|name)=[\'"]([^\'"]+)[\'"]', texto))
    for titulo in titulos:
        titulo = re.sub(r"!?\[([^]]+)\]\([^)]*\)", r"\1", titulo)
        titulo = re.sub(r"<[^>]+>", "", titulo)
        base = re.sub(r"[^\w\- ]", "", titulo.lower()).replace(" ", "-")
        candidata, indice = base, 0
        while candidata in usadas:
            indice += 1
            candidata = f"{base}-{indice}"
        usadas.add(candidata)
    return usadas


def comprobar(raiz, documentos):
    errores, locales, externos = [], 0, 0
    for documento in documentos:
        for linea, enlace in enlaces(documento.read_text(encoding="utf-8")):
            ubicacion = f"{documento.relative_to(raiz)}:{linea}"
            if enlace is None:
                errores.append(f"{ubicacion}: referencia sin definir")
                continue
            url = urlsplit(enlace)
            if url.scheme or url.netloc:
                externos += 1
                continue
            locales += 1
            ruta = unquote(url.path)
            ruta = re.sub(r"\\([() ])", r"\1", ruta)
            # El constructor incorpora estos vídeos desde ETF_CURSO_VIDEOS.
            # No viven en Git; make curso comprueba su existencia al exportar.
            if (documento.is_relative_to(raiz / 'curso/contenido')
                    and re.fullmatch(r'videos/\d{2}-[\w.-]+\.mp4', ruta)
                    and not url.query and not url.fragment):
                continue
            archivo = ((raiz / ruta.lstrip("/")) if ruta.startswith("/")
                       else (documento.parent / ruta if ruta else documento)).resolve()
            if not archivo.is_relative_to(raiz):
                errores.append(f"{ubicacion}: destino fuera del repositorio: {enlace}")
            elif not archivo.exists():
                errores.append(f"{ubicacion}: destino inexistente: {enlace}")
            elif url.fragment and archivo.suffix.lower() == ".md":
                if unquote(url.fragment) not in anclas(archivo.read_text(encoding="utf-8")):
                    errores.append(f"{ubicacion}: ancla inexistente: {enlace}")
    return errores, locales, externos


def main():
    raiz = Path(__file__).resolve().parents[2]
    nombres = subprocess.check_output(
        ["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.md"],
        cwd=raiz,
    ).decode().split("\0")
    documentos = [raiz / nombre for nombre in nombres if nombre]
    errores, locales, externos = comprobar(raiz, documentos)
    for error in errores:
        print(error, file=sys.stderr)
    print(f"Markdown: {len(documentos)} documentos, {locales} enlaces locales, "
          f"{externos} enlaces externos sin consulta de red, {len(errores)} errores.")
    return bool(errores)


if __name__ == "__main__":
    sys.exit(main())
