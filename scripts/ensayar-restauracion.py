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
        carpeta = Path(tempfile.mkdtemp(prefix="f505-", dir=os.environ["TMPDIR"]))
        print(f"COPIA TEMPORAL: {carpeta}; se conserva si falla el ensayo", flush=True)
        claves = json.loads((demo.LOCAL / "usuarios.json").read_text())
        anterior_sesion = demo.Sesion(carpeta, "sesion-antes-copia")
        anterior_sesion.entrar("ines.moreno", claves)
        ejecutar("stop", "entrada-demo", "app")
        fecha_copia = bd("psql", "-XAt", "-U", "etf_instalador", "-d", "etf", "-c", "SELECT clock_timestamp()").decode().strip()
        referencia_copia = bd("psql", "-XAt", "-U", "etf_instalador", "-d", "etf", "-c", "SELECT identidad.exportar_identidad()")
        copia = bd("pg_dump", "-U", "etf_instalador", "-d", "etf", "-Fc")
        (carpeta / "etf.dump").write_bytes(copia)
        # El operador local incluye todos los equipos; etf_copia aún carece de permisos en algunas tablas.
        paquetes = ejecutar("run", "--rm", "--no-deps", "-T", "--entrypoint", "tar", "app",
                            "-C", "/datos", "-cf", "-", "idiomas", "temas")
        (carpeta / "paquetes.tar").write_bytes(paquetes)
        antes = huella_datos()
        print(f"COPIA-OK: PostgreSQL y paquetes; datos SHA256 {antes}", flush=True)
        bd("psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", "etf_instalador", "-d", "etf",
           entrada=(RAIZ / "scripts/seg4/cambios.sql").read_bytes())
        referencia = carpeta / "identidad-vigente.json"
        referencia.write_bytes(bd("psql", "-XAt", "-U", "etf_instalador", "-d", "etf", "-c", "SELECT identidad.exportar_identidad()"))
        def iniciar_revision(ruta):
            paso = demo.ejecutar(["python3", "-B", str(RAIZ / "scripts/preparar-revision-restauracion.py"),
                                 "--referencia", str(ruta), "--fecha-copia", fecha_copia], capture_output=True).stdout
            bd("psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", "etf_instalador", "-d", "etf", entrada=paso)
        bd("dropdb", "-U", "etf_instalador", "etf")
        existe = bd("psql", "-XAt", "-U", "etf_instalador", "-d", "postgres", "-c",
                    "SELECT count(*) FROM pg_database WHERE datname='etf'").strip()
        if existe != b"0":
            raise RuntimeError("No se ha borrado la base del ensayo.")
        # Borrar, comprobar vacío y extraer en el mismo contenedor: Docker
        # repuebla automáticamente un volumen vacío al crear otro contenedor.
        # BusyBox tar aplica la umask incluso con -p. El archivo es la copia
        # privada recién creada: conservar también los permisos de sus miembros.
        # El preparador aislado puede reponer los UID/GID originales. La app
        # sigue sin privilegios y no podría recuperar ficheros de otro UID.
        borrado = ejecutar("run", "--rm", "--no-deps", "-T", "--entrypoint", "sh", "incorporar", "-c",
                           'set -eu; find /datos/idiomas /datos/temas -mindepth 1 -delete; '
                           'test -z "$(find /datos/idiomas /datos/temas -mindepth 1 -print)"; '
                           'printf BORRADO; umask 000; tar -C /datos -xpf -', entrada=paquetes)
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
        iniciar_revision(referencia)
        bd("psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", "etf_instalador", "-d", "etf",
           entrada=(RAIZ / "scripts/seg4/comprobar.sql").read_bytes())
        # Prueba Go sobre los adaptadores reales y la lectura auditada, sin secretos en argumentos.
        entorno_prueba = dict(os.environ)
        for linea in (demo.LOCAL / "entorno.env").read_text().splitlines():
            if "=" in linea:
                clave, valor = linea.split("=", 1)
                entorno_prueba[clave] = valor
        entorno_prueba.update(ETF_POSTGRES_INTEGRACION="1", ETF_BD_SSLMODE="disable",
                             ETF_BD_HOST=bd("hostname", "-i").decode().strip(),
                             ETF_BD_CONTRASENA=entorno_prueba["ETF_APP_CONTRASENA"],
                             ETF_SEG4_ADMIN=bd("psql", "-XAt", "-U", "etf_instalador", "-d", "etf", "-c",
                                               "SELECT id FROM identidad.usuario WHERE cuenta_ad='elena.salas'").decode().strip())
        demo.ejecutar(["go", "test", "-count=1", "-run", "TestIntegracionRevisionRestauracionLecturaAuditada", "-v",
                       "./internal/adaptadores/postgres"], env=entorno_prueba, timeout=60)
        print("SEG4-SQL-OK: sesiones revocadas, bloqueo no administrador, RLS, permisos, pendientes y revisión repetida", flush=True)
        ejecutar("up", "-d", "--no-deps", "--wait", "--wait-timeout", "90", "app", "entrada-demo")
        anterior_sesion.pedir("GET", "/api/sesion", esperado=401)
        revision_actual = bd("psql", "-XAt", "-U", "etf_instalador", "-d", "etf", "-c",
                             "SELECT id FROM identidad.revision_restauracion WHERE confirmada_en IS NULL AND anulada_en IS NULL").decode().strip()
        cambios_actuales = int(bd("psql", "-XAt", "-U", "etf_instalador", "-d", "etf", "-c",
                                  "SELECT count(*) FROM identidad.cambio_restauracion WHERE revision_id='" + revision_actual + "'").strip())
        demo.ejecutar(["node", str(RAIZ / "scripts/ensayar-revision-restauracion.mjs"), demo.URL], timeout=120)
        eventos = bd("psql", "-XAt", "-U", "etf_instalador", "-d", "etf", "-c",
                     "SELECT count(*) FROM auditoria.evento WHERE resultado='ok' AND ((accion='identidad.restauracion_confirmar' AND entidad_id='" + revision_actual + "') OR (accion IN ('identidad.restauracion_aplicar','identidad.restauracion_descartar') AND entidad_id IN (SELECT id FROM identidad.cambio_restauracion WHERE revision_id='" + revision_actual + "')))").strip()
        if int(eventos) != cambios_actuales + 1:
            raise RuntimeError("Faltan decisiones o confirmación auditadas.")
        print("SEG4-AUDITORIA-OK: decisiones y confirmación registradas", flush=True)
        # Segunda restauración real, sin diferencias: no hereda confirmaciones ni sesiones.
        ejecutar("stop", "entrada-demo", "app")
        bd("dropdb", "-U", "etf_instalador", "etf")
        bd("createdb", "-U", "etf_instalador", "-O", "etf_propietario", "etf")
        bd("pg_restore", "-U", "etf_instalador", "-d", "etf", "--exit-on-error", entrada=copia)
        bd("psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", "etf_instalador", "-d", "postgres", "-c",
           "REVOKE ALL ON DATABASE etf FROM PUBLIC; GRANT CONNECT ON DATABASE etf TO etf_app, etf_migrador, etf_copia, etf_calculo_memoria;")
        referencia.write_bytes(referencia_copia)
        iniciar_revision(referencia)
        ejecutar("up", "-d", "--no-deps", "--wait", "--wait-timeout", "90", "app", "entrada-demo")
        admin = demo.Sesion(carpeta, "admin-segunda-restauracion")
        admin.entrar("elena.salas", claves)
        revision = admin.pedir("GET", "/api/admin/restauracion")
        if revision["cambios"]:
            raise RuntimeError("La segunda restauración conserva diferencias anteriores.")
        demo.ejecutar(["node", str(RAIZ / "scripts/ensayar-revision-restauracion.mjs"), demo.URL, "vacia"], timeout=60)
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
