#!/usr/bin/env python3
"""Ensayo destructivo limitado a la demostración del worktree actual."""
import fcntl
import hashlib
import io
import json
import tarfile
import importlib.util
import os
from pathlib import Path
import subprocess
import shutil
import tempfile

RAIZ = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("demo", RAIZ / "despliegue/demo/demo.py")
demo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(demo)


def ejecutar(*args, entrada=None):
    return demo.compose(*args, input=entrada, stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE, timeout=180).stdout


def bd(*args, entrada=None):
    return ejecutar("exec", "-T", "postgres", *args, entrada=entrada)


def huella_datos():
    salida = bd("pg_dump", "-U", "etf_instalador", "-d", "etf", "--data-only", "--inserts")
    # pg_dump 18 añade testigos aleatorios de restricción; no son datos.
    lineas = sorted(l for l in salida.splitlines() if l and not l.startswith((b"--", b"\\restrict", b"\\unrestrict")))
    return hashlib.sha256(b"\n".join(lineas)).hexdigest()


def huella_paquetes(datos):
    with tarfile.open(fileobj=io.BytesIO(datos)) as archivo:
        miembros = []
        for miembro in archivo:
            contenido = archivo.extractfile(miembro).read() if miembro.isfile() else b""
            miembros.append((miembro.name, miembro.type.decode(), miembro.mode,
                             miembro.uid, miembro.gid, miembro.linkname,
                             hashlib.sha256(contenido).hexdigest()))
    return hashlib.sha256(json.dumps(sorted(miembros)).encode()).hexdigest()


def main():
    os.umask(0o077)
    demo.comprobar_docker_local()
    if not (demo.LOCAL / "entorno.env").is_file():
        raise RuntimeError("Arranca primero make demo en este worktree.")
    with (demo.DEMO / ".operacion.local").open("w") as cerrojo:
        fcntl.flock(cerrojo, fcntl.LOCK_EX)
        # El nombre se deriva del worktree; no se admite un destino del operador.
        if not demo.PROYECTO.startswith("etf-demo-"):
            raise RuntimeError("Sólo se permite el proyecto de demostración.")
        print(f"DESTINO: {demo.PROYECTO}; sólo datos inventados", flush=True)
        # Exigir un estado funcional antes de respaldarlo y reiniciar el
        # limitador de la demo tras la siembra o comprobaciones anteriores.
        ejecutar("restart", "app")
        ejecutar("up", "-d", "--no-deps", "--wait", "--wait-timeout", "90", "app", "entrada-demo")
        demo.ejecutar(["node", str(demo.DEMO / "recorrido.mjs"), demo.URL], timeout=120)
        ejecutar("stop", "entrada-demo", "app")
        carpeta = Path(tempfile.mkdtemp(prefix="f505-", dir=os.environ["TMPDIR"]))
        print(f"COPIA TEMPORAL: {carpeta}; se conserva si falla el ensayo", flush=True)
        copia = bd("pg_dump", "-U", "etf_instalador", "-d", "etf", "-Fc")
        (carpeta / "etf.dump").write_bytes(copia)
        # El operador local incluye todos los equipos; etf_copia aún carece de permisos en algunas tablas.
        paquetes = ejecutar("run", "--rm", "--no-deps", "-T", "--entrypoint", "tar", "app",
                            "-C", "/datos", "-cf", "-", "idiomas", "temas")
        (carpeta / "paquetes.tar").write_bytes(paquetes)
        antes = huella_datos()
        print(f"COPIA-OK: PostgreSQL y paquetes; datos SHA256 {antes}", flush=True)
        bd("dropdb", "-U", "etf_instalador", "etf")
        existe = bd("psql", "-XAt", "-U", "etf_instalador", "-d", "postgres", "-c",
                    "SELECT count(*) FROM pg_database WHERE datname='etf'").strip()
        if existe != b"0":
            raise RuntimeError("No se ha borrado la base del ensayo.")
        # Borrar, comprobar vacío y extraer en el mismo contenedor: Docker
        # repuebla automáticamente un volumen vacío al crear otro contenedor.
        borrado = ejecutar("run", "--rm", "--no-deps", "-T", "--entrypoint", "sh", "app", "-c",
                           'set -eu; find /datos/idiomas /datos/temas -mindepth 1 -delete; '
                           'test -z "$(find /datos/idiomas /datos/temas -mindepth 1 -print)"; '
                           'printf BORRADO; tar -C /datos -xpf -', entrada=paquetes)
        if borrado != b"BORRADO":
            raise RuntimeError("No se han borrado los paquetes del ensayo.")
        print("BORRADO-OK: base inexistente; idiomas y temas vacíos antes de extraer la copia", flush=True)
        bd("createdb", "-U", "etf_instalador", "-O", "etf_propietario", "etf")
        bd("pg_restore", "-U", "etf_instalador", "-d", "etf", "--exit-on-error", entrada=copia)
        # Los permisos de la base no forman parte de pg_dump sin --create.
        bd("psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", "etf_instalador", "-d", "postgres", "-c",
           "REVOKE ALL ON DATABASE etf FROM PUBLIC; GRANT CONNECT ON DATABASE etf TO etf_app, etf_migrador, etf_copia, etf_calculo_memoria;")
        despues = huella_datos()
        recuperados = ejecutar("run", "--rm", "--no-deps", "-T", "--entrypoint", "tar", "app",
                               "-C", "/datos", "-cf", "-", "idiomas", "temas")
        if antes != despues or huella_paquetes(paquetes) != huella_paquetes(recuperados):
            raise RuntimeError("La restauración no coincide con la copia.")
        print("RESTAURACION-OK: datos, secuencias y paquetes idénticos", flush=True)
        # Las sesiones de la copia no deben servir tras recuperar un estado anterior.
        bd("psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", "etf_instalador", "-d", "etf", "-c",
           "DELETE FROM identidad.sesion;")
        ejecutar("up", "-d", "--no-deps", "--wait", "--wait-timeout", "90", "app", "entrada-demo")
        demo.comprobar()
        demo.ejecutar(["node", str(demo.DEMO / "recorrido.mjs"), demo.URL], timeout=120)
        shutil.rmtree(carpeta)
        print("F5.05-OK: copia, borrado, restauración, siete accesos y recorrido Chrome", flush=True)


if __name__ == "__main__":
    try:
        main()
    except RuntimeError as error:
        print(str(error))
        raise SystemExit(1)
    except (OSError, subprocess.CalledProcessError, subprocess.TimeoutExpired):
        # No volcar argumentos, entorno ni respuestas de PostgreSQL con datos.
        print("F5.05-FALLO: ensayo detenido. Si se anunció una ruta de copia, consérvala; revisa la demo local.")
        raise SystemExit(1)
