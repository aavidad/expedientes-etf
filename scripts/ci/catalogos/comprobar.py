#!/usr/bin/env python3
"""Valida la integridad de los catálogos iniciales de ubicación."""

from __future__ import annotations

import json
import re
import sys
import unicodedata
from pathlib import Path


RAIZ = Path(__file__).resolve().parents[3]
DATOS = RAIZ / "datos" / "catalogos"
TEXTOS = RAIZ / "idiomas" / "es" / "catalogos"


def leer(nombre: str):
    return json.loads((DATOS / nombre).read_text(encoding="utf-8"))


def normalizar(texto: str) -> str:
    texto = unicodedata.normalize("NFD", texto.casefold())
    texto = "".join(c for c in texto if unicodedata.category(c) != "Mn")
    return re.sub(r"[^a-z0-9]+", " ", texto).strip()


def exigir(condicion: bool, codigo: str) -> None:
    if not condicion:
        raise ValueError(codigo)


def comprobar() -> None:
    equipos = leer("equipos.json")
    centros = leer("centros.json")
    municipios = leer("municipios.json")
    textos = json.loads((TEXTOS / "ubicaciones.json").read_text(encoding="utf-8"))

    codigos_equipos = [fila["codigo"] for fila in equipos["equipos"]]
    exigir(len(codigos_equipos) == len(set(codigos_equipos)), "EQUIPOS_CODIGO_REPETIDO")
    exigir(codigos_equipos == [f"{n:02}" for n in range(1, 9)], "EQUIPOS_CODIGOS_INVALIDOS")
    exigir(equipos["capacidad"] == 12, "EQUIPOS_CAPACIDAD_INVALIDA")

    codigos_centros = [fila["codigo"] for fila in centros["centros"]]
    exigir(len(codigos_centros) == len(set(codigos_centros)), "CENTROS_CODIGO_REPETIDO")
    exigir(len(codigos_centros) == 14, "CENTROS_CANTIDAD_INVALIDA")

    codigos_municipios = [fila["codigo"] for fila in municipios["municipios"]]
    exigir(len(codigos_municipios) == len(set(codigos_municipios)), "MUNICIPIOS_CODIGO_REPETIDO")
    filas_municipio = municipios["municipios"]
    exigir(sum(fila["tipo"] == "municipio" for fila in filas_municipio) == 174,
           "MUNICIPIOS_CANTIDAD_INVALIDA")
    exigir(sum(fila["tipo"] == "ela" for fila in filas_municipio) == 5, "ELAS_CANTIDAD_INVALIDA")
    exigir({"18087", "18140", "18017", "18021", "18905", "18127", "18022", "18023", "18193", "18122"}
           <= {fila["codigo"] for fila in filas_municipio if not fila["ambito_diputacion"]},
           "MUNICIPIOS_D10_SIN_MARCA")
    illora = next(fila for fila in filas_municipio if fila["codigo"] == "18102")
    exigir(illora["centro_codigo"] == "PINOS_PUENTE_MOCLIN" and illora["equipo_codigo"] == "06",
           "ILLORA_ASIGNACION_INVALIDA")
    ids_equipo = set(codigos_equipos)
    ids_centro = set(codigos_centros)
    todos_codigos = codigos_equipos + codigos_centros + codigos_municipios
    exigir(len(todos_codigos) == len(set(todos_codigos)), "CODIGO_REPETIDO_ENTRE_CATALOGOS")
    asignaciones: dict[str, str] = {}
    claves_texto: set[str] = set()
    variantes: dict[str, str] = {}
    for fila in filas_municipio:
        exigir(fila["tipo"] in {"municipio", "ela"}, "MUNICIPIO_TIPO_INVALIDO")
        codigo = fila["codigo"]
        clave = fila["clave_texto"]
        exigir(clave not in claves_texto, "CLAVE_TEXTO_REPETIDA")
        claves_texto.add(clave)
        if fila["ambito_diputacion"]:
            exigir(fila.get("centro_codigo") in ids_centro, "MUNICIPIO_CENTRO_FALTANTE")
            exigir(fila.get("equipo_codigo") in ids_equipo, "MUNICIPIO_EQUIPO_FALTANTE")
        if fila["tipo"] == "ela":
            exigir(bool(fila.get("matriz_codigo")), "ELA_SIN_MATRIZ")
            exigir(fila["matriz_codigo"] in set(codigos_municipios), "ELA_MATRIZ_DESCONOCIDA")
        else:
            exigir(fila.get("matriz_codigo") is None, "MUNICIPIO_CON_MATRIZ")
        for variante in fila["variantes"]:
            exigida = normalizar(variante)
            exigir(variante == exigida and bool(exigida), "VARIANTE_NO_NORMALIZADA")
            anterior = variantes.setdefault(exigida, codigo)
            exigir(anterior == codigo, "VARIANTE_EN_COLISION")
        exigir(len(fila["variantes"]) == len(set(fila["variantes"])), "VARIANTE_REPETIDA")
        if fila["ambito_diputacion"]:
            asignaciones[codigo] = fila["equipo_codigo"]

    claves_esperadas: set[str] = set()
    for nombre, prefijo, catalogo in (("equipos", "equipo", equipos),
                                      ("centros", "centro", centros),
                                      ("municipios", "municipio", municipios)):
        for fila in catalogo[nombre]:
            clave = fila["clave_texto"]
            exigir(clave == f"{prefijo}.{fila['codigo']}", "CLAVE_TEXTO_INVALIDA")
            exigir(clave not in claves_esperadas, "CLAVE_TEXTO_REPETIDA")
            claves_esperadas.add(clave)
            etiqueta = textos.get(clave)
            exigir(isinstance(etiqueta, str) and bool(etiqueta.strip()),
                   "TEXTO_CATALOGO_INVALIDO")
    exigir(set(textos) == claves_esperadas, "TEXTOS_CATALOGO_INCOMPLETOS")


if __name__ == "__main__":
    try:
        comprobar()
    except (KeyError, OSError, json.JSONDecodeError, ValueError) as error:
        print(f"CATALOGOS_INVALIDOS:{error}", file=sys.stderr)
        raise SystemExit(1)
    print("CATALOGOS_OK")
