"""Instala los recursos de la distribución sin tocar lo publicado.

La base de cada idioma y tema (todo lo que no cuelga de una carpeta
``.versiones``) es de la distribución: cada actualización la sustituye. Lo que
cuelga de ``.versiones``, a cualquier profundidad, lo publica Administración
y es inmutable: sólo se añade lo que falte y nunca se reescribe ni se borra.

Un fichero existente se sustituye si su huella figura, para esa ruta, en el
manifiesto de la entrega anterior (``.distribucion.json``) o en el inventario
de todas las versiones históricas generado al construir. Sólo se retira lo que
figura en el manifiesto anterior con la misma ruta y la misma huella. Lo
modificado a mano se conserva y se avisa una vez.

Cada fichero se sustituye de forma atómica (temporal en la misma carpeta con
``O_EXCL|O_NOFOLLOW``, ``fsync``, ``os.replace`` y ``fsync`` de la carpeta).
Las rutas se recorren con descriptores y ``O_NOFOLLOW``: un enlace simbólico
en el destino hace fallar la preparación y nunca se sigue.
"""
from collections import Counter
import hashlib
import json
import os
import secrets
import stat
import subprocess
import sys

VERSIONES = ".versiones"
MANIFIESTO = ".distribucion.json"
TEMPORAL = ".etf-nuevo-"
_DIRECTORIO = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC
_MAXIMO = 64 << 20


class EnlaceNoPermitido(OSError):
    pass


def huella(contenido):
    return hashlib.sha256(contenido).hexdigest()


def avisar(mensaje):
    print("aviso: " + mensaje, file=sys.stderr)


def _abrir_carpeta(padre, nombre, uid, gid, crear):
    """Abre (o crea) una subcarpeta real; rechaza enlaces y ficheros."""
    try:
        info = os.stat(nombre, dir_fd=padre, follow_symlinks=False)
    except FileNotFoundError:
        if not crear:
            raise
        os.mkdir(nombre, 0o755, dir_fd=padre)
        fd = os.open(nombre, _DIRECTORIO, dir_fd=padre)
        os.fchown(fd, uid, gid)
        os.fchmod(fd, 0o755)
        return fd
    if stat.S_ISLNK(info.st_mode):
        raise EnlaceNoPermitido(f"enlace simbólico en el destino: {nombre}")
    return os.open(nombre, _DIRECTORIO, dir_fd=padre)


def _abrir_ruta(raiz, partes, uid, gid, crear):
    fd = os.dup(raiz)
    try:
        for parte in partes:
            siguiente = _abrir_carpeta(fd, parte, uid, gid, crear)
            os.close(fd)
            fd = siguiente
        return fd
    except BaseException:
        os.close(fd)
        raise


def _leer(carpeta, nombre):
    """Contenido de un fichero regular, None si no existe; un enlace es un error."""
    try:
        info = os.stat(nombre, dir_fd=carpeta, follow_symlinks=False)
    except FileNotFoundError:
        return None
    if stat.S_ISLNK(info.st_mode):
        raise EnlaceNoPermitido(f"enlace simbólico en el destino: {nombre}")
    if not stat.S_ISREG(info.st_mode):
        raise IsADirectoryError(nombre)
    fd = os.open(nombre, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=carpeta)
    with os.fdopen(fd, "rb") as f:
        contenido = f.read(_MAXIMO + 1)
    if len(contenido) > _MAXIMO:
        raise ValueError(f"fichero demasiado grande: {nombre}")
    return contenido


def _escribir(carpeta, nombre, contenido, uid, gid, sustituir):
    """Escribe el fichero completo con un nombre temporal y lo coloca de golpe."""
    temporal = TEMPORAL + secrets.token_hex(8)
    fd = os.open(temporal, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                 0o600, dir_fd=carpeta)
    try:
        try:
            vista = memoryview(contenido)
            while vista:
                vista = vista[os.write(fd, vista):]
            os.fchown(fd, uid, gid)
            os.fchmod(fd, 0o644)
            os.fsync(fd)
        finally:
            os.close(fd)
        if sustituir:
            os.replace(temporal, nombre, src_dir_fd=carpeta, dst_dir_fd=carpeta)
        else:
            # Lo publicado no se reescribe: si apareció entretanto, se conserva.
            try:
                os.link(temporal, nombre, src_dir_fd=carpeta, dst_dir_fd=carpeta, follow_symlinks=False)
            except FileExistsError:
                pass
        os.fsync(carpeta)
    finally:
        try:
            os.unlink(temporal, dir_fd=carpeta)
        except FileNotFoundError:
            pass


def _limpiar_temporales(carpeta):
    # Restos de una actualización cortada: nunca son ficheros de la distribución.
    for nombre in os.listdir(carpeta):
        if nombre.startswith(TEMPORAL) or nombre.startswith(".win1-"):
            try:
                if stat.S_ISREG(os.stat(nombre, dir_fd=carpeta, follow_symlinks=False).st_mode):
                    os.unlink(nombre, dir_fd=carpeta)
            except FileNotFoundError:
                pass


def _relativa_valida(relativa):
    if not isinstance(relativa, str) or not relativa or relativa.startswith("/"):
        return False
    partes = relativa.split("/")
    return (all(p not in ("", ".", "..") for p in partes) and VERSIONES not in partes
            and relativa != MANIFIESTO and not partes[-1].startswith(TEMPORAL))


def _manifiesto_anterior(raiz):
    try:
        datos = json.loads(_leer(raiz, MANIFIESTO) or b"null")
    except ValueError:
        avisar("manifiesto de distribución ilegible; se usa sólo el inventario")
        return {}
    ficheros = datos.get("ficheros") if isinstance(datos, dict) else None
    if not isinstance(ficheros, dict):
        return {}
    return {r: h for r, h in ficheros.items() if _relativa_valida(r) and isinstance(h, str)}


def _base_existente(raiz):
    """Ficheros de la base del destino, sin entrar en .versiones ni seguir enlaces."""
    encontrados = []

    def recorrer(fd, prefijo):
        for nombre in sorted(os.listdir(fd)):
            relativa = prefijo + nombre
            if nombre == VERSIONES or nombre.startswith(TEMPORAL) or relativa == MANIFIESTO:
                continue
            info = os.stat(nombre, dir_fd=fd, follow_symlinks=False)
            if stat.S_ISLNK(info.st_mode):
                raise EnlaceNoPermitido(f"enlace simbólico en el destino: {relativa}")
            if stat.S_ISDIR(info.st_mode):
                hijo = os.open(nombre, _DIRECTORIO, dir_fd=fd)
                try:
                    recorrer(hijo, relativa + "/")
                finally:
                    os.close(hijo)
            elif stat.S_ISREG(info.st_mode):
                encontrados.append(relativa)

    recorrer(raiz, "")
    return encontrados


def incorporar(origen, destino, uid, gid, inventario=None):
    """Instala ``origen`` en ``destino``; ``inventario`` es el de generar_inventario."""
    inventario = inventario or {}
    historicas = set(inventario.get("huellas", []))
    entrega = str(inventario.get("entrega", ""))
    destino.mkdir(parents=True, exist_ok=True)
    raiz = os.open(destino, _DIRECTORIO)
    try:
        os.fchown(raiz, uid, gid)
        os.fchmod(raiz, 0o755)
        anterior = _manifiesto_anterior(raiz)

        def de_distribucion(relativa, contenido):
            h = huella(contenido)
            return h == anterior.get(relativa) or h in historicas

        distribucion = {}
        conservados = set()
        for actual, carpetas, ficheros in os.walk(origen):
            carpetas[:] = sorted(c for c in carpetas if not os.path.islink(os.path.join(actual, c)))
            relativa = os.path.relpath(actual, origen)
            partes = [] if relativa == "." else relativa.split(os.sep)
            publicada = VERSIONES in partes
            carpeta = _abrir_ruta(raiz, partes, uid, gid, True)
            try:
                _limpiar_temporales(carpeta)
                for nombre in carpetas:
                    os.close(_abrir_carpeta(carpeta, nombre, uid, gid, True))
                for nombre in sorted(ficheros):
                    ruta = os.path.join(actual, nombre)
                    ruta_destino = "/".join([*partes, nombre])
                    if os.path.islink(ruta) or not os.path.isfile(ruta) or ruta_destino == MANIFIESTO:
                        continue
                    with open(ruta, "rb") as f:
                        contenido = f.read()
                    existente = _leer(carpeta, nombre)
                    if publicada:
                        if existente is None:
                            _escribir(carpeta, nombre, contenido, uid, gid, False)
                        continue
                    if existente is None or (existente != contenido and de_distribucion(ruta_destino, existente)):
                        _escribir(carpeta, nombre, contenido, uid, gid, True)
                    elif existente != contenido:
                        avisar(f"se conserva {ruta_destino}: cambiado fuera de la distribución")
                        conservados.add(ruta_destino)
                        continue
                    distribucion[ruta_destino] = huella(contenido)
            finally:
                os.close(carpeta)
        # Se retiran sólo los ficheros que instaló la entrega anterior (misma ruta
        # y misma huella en su manifiesto) y la nueva ya no trae. El inventario
        # histórico sólo autoriza a sustituir en la misma ruta, nunca a retirar:
        # una copia manual con contenido histórico se conserva, como lo publicado.
        for relativa in _base_existente(raiz):
            if relativa in distribucion or relativa in conservados:
                continue
            partes = relativa.split("/")
            carpeta = _abrir_ruta(raiz, partes[:-1], uid, gid, False)
            try:
                contenido = _leer(carpeta, partes[-1])
                if contenido is not None and anterior.get(relativa) == huella(contenido):
                    os.unlink(partes[-1], dir_fd=carpeta)
                    os.fsync(carpeta)
                elif contenido is not None:
                    avisar(f"se conserva {relativa}: no pertenece a la distribución")
            finally:
                os.close(carpeta)
        manifiesto = {"entrega": entrega, "ficheros": dict(sorted(distribucion.items()))}
        _escribir(raiz, MANIFIESTO, json.dumps(manifiesto, ensure_ascii=False, indent=1).encode() + b"\n",
                  uid, gid, True)
    finally:
        os.close(raiz)


def _objeto_sin_repetidas(pares):
    repetidas = sorted(k for k, n in Counter(k for k, _ in pares).items() if n > 1)
    if repetidas:
        raise ValueError("claves repetidas: " + ", ".join(repetidas))
    return dict(pares)


def validar_textos(destino):
    """Carga los textos de la base de cada idioma; falla con el diagnóstico completo."""
    problemas = []
    raiz = os.open(destino, _DIRECTORIO)
    try:
        for codigo in sorted(os.listdir(raiz)):
            if codigo.startswith("."):
                continue
            try:
                textos = _abrir_ruta(raiz, [codigo, "textos"], 0, 0, False)
            except (FileNotFoundError, NotADirectoryError):
                continue
            try:
                vistas = {}
                for nombre in sorted(os.listdir(textos)):
                    if not nombre.endswith(".json"):
                        continue
                    ruta = f"{codigo}/textos/{nombre}"
                    try:
                        datos = json.loads(_leer(textos, nombre), object_pairs_hook=_objeto_sin_repetidas)
                        if not isinstance(datos, dict) or not all(isinstance(v, str) for v in datos.values()):
                            raise ValueError("no es un objeto de textos")
                    except ValueError as error:
                        problemas.append(f"{ruta}: {error}")
                        continue
                    for clave in datos:
                        if clave in vistas:
                            problemas.append(f"{ruta}: clave {clave} repetida en {vistas[clave]}")
                        vistas[clave] = ruta
            finally:
                os.close(textos)
    finally:
        os.close(raiz)
    if problemas:
        raise RuntimeError("textos no válidos tras la actualización:\n" + "\n".join(problemas))


def generar_inventario(repositorio, nombres=("idiomas", "temas")):
    """Huellas de todas las versiones que han existido en Git de esos recursos."""
    def git(*argumentos, entrada=None):
        return subprocess.run(["git", "-C", str(repositorio), *argumentos], check=True,
                              capture_output=True, input=entrada).stdout

    entrega = git("rev-parse", "HEAD").decode().strip()
    objetos = set()
    for linea in git("log", "--format=", "--raw", "--no-abbrev", "--no-renames", "--full-history", "-m", "HEAD", "--", *nombres).splitlines():
        if linea.startswith(b":"):
            campos = linea.split(b"\t")[0].split()
            objetos.update(o for o in campos[2:4] if o.strip(b"0"))
    for linea in git("ls-tree", "-r", "HEAD", "--", *nombres).splitlines():
        objetos.add(linea.split(b"\t")[0].split()[2])
    huellas = set()
    salida = git("cat-file", "--batch", entrada=b"\n".join(sorted(objetos)) + b"\n")
    posicion = 0
    while posicion < len(salida):
        fin = salida.index(b"\n", posicion)
        cabecera = salida[posicion:fin].split()
        if len(cabecera) < 3:
            # Objeto ausente (clon superficial): no se puede inventariar.
            posicion = fin + 1
            continue
        tamano = int(cabecera[2])
        contenido = salida[fin + 1:fin + 1 + tamano]
        if cabecera[1] == b"blob":
            huellas.add(huella(contenido))
        posicion = fin + 1 + tamano + 1
    return {"entrega": entrega, "huellas": sorted(huellas)}
