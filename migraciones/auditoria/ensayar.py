#!/usr/bin/env python3
"""Ensaya F0.08b con PostgreSQL de Compose, datos inventados y red interna."""

import os
import sys
from pathlib import Path
import re
import secrets
import shutil
import subprocess
import tempfile


RAIZ = Path(__file__).resolve().parents[2]


sys.path.insert(0, str(RAIZ / 'scripts' / 'ci'))
from ensayos import ejecutar as ejecutar_ensayo


def ejecutar(args, entorno, **opciones):
    resultado = ejecutar_ensayo(args, entorno=entorno, cwd=RAIZ,
                               timeout=opciones.get('timeout', 120), modulo='auditoria')
    if 'stdout' not in opciones and resultado.stdout:
        print(resultado.stdout.strip(), flush=True)
    return resultado


def main():
    # Se compila con dependencias ya presentes; no se descarga ni sube código.
    entorno = {k: os.environ[k] for k in ("PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE") if k in os.environ}
    entorno.update(GOPROXY="off", GOSUMDB="off", CGO_ENABLED="0")
    imagen = re.search(r"image: (postgres:[^\s]+)", (RAIZ / "compose.yaml").read_text()).group(1)
    ejecutar(["docker", "image", "inspect", imagen], entorno, stdout=subprocess.DEVNULL)

    with tempfile.TemporaryDirectory(prefix="etf-f008b-") as temporal:
        carpeta = Path(temporal)
        binarios = carpeta / "binarios"
        binarios.mkdir(mode=0o755)
        for nombre in ("migrar", "comprobar-auditoria"):
            ejecutar(["go", "build", "-trimpath", "-o", str(binarios / nombre), "./cmd/" + nombre], entorno)
        ejecutar(["go", "test", "-c", "-o", str(binarios / "postgres.test"), "./internal/adaptadores/postgres"], entorno)

        anterior = carpeta / "anterior"
        shutil.copytree(RAIZ / "migraciones/plataforma", anterior / "plataforma")
        credenciales = {k: secrets.token_hex(24) for k in (
            "ETF_INSTALADOR_CONTRASENA", "ETF_MIGRADOR_CONTRASENA", "ETF_APP_CONTRASENA", "ETF_COPIA_CONTRASENA")}
        credenciales.update(ETF_RED_SUBRED=f"10.252.{secrets.randbelow(250) + 1}.0/28")
        archivo_env = carpeta / "pruebas.env"
        archivo_env.write_text("".join(k + "=" + v + "\n" for k, v in credenciales.items()))
        archivo_env.chmod(0o600)
        proyecto = "etf-f008b-" + secrets.token_hex(4)
        override = carpeta / "override.yaml"
        override.write_text(f"""services:
  migrar:
    image: {imagen}
    user: "70:70"
    entrypoint: /opt/pruebas/migrar
    command: []
    environment:
      ETF_MIGRACIONES_RUTA: /opt/etf/migraciones
    volumes:
      - {binarios}:/opt/pruebas:ro
      - ${{ETF_ENSAYO_MIGRACIONES}}:/opt/etf/migraciones:ro
    tmpfs:
      - /tmp:uid=70,gid=70,mode=0700,size=16m
""")
        compose = ["docker", "compose", "--env-file", str(archivo_env), "-p", proyecto,
                   "-f", str(RAIZ / "compose.yaml"), "-f", str(override)]
        entorno["ETF_ENSAYO_MIGRACIONES"] = str(RAIZ / "migraciones")

        def migrar(clave):
            resultado = ejecutar(compose + ["run", "--rm", "--no-deps", "migrar"], entorno,
                                 stdout=subprocess.PIPE, text=True)
            if clave not in resultado.stdout:
                raise RuntimeError("migracion.resultado_inesperado")
            print(resultado.stdout.strip(), flush=True)

        def levantar():
            ejecutar(compose + ["up", "-d", "--wait", "--wait-timeout", "45", "postgres"], entorno)

        def destruir():
            ejecutar(compose + ["down", "--volumes", "--remove-orphans"], entorno)

        try:
            levantar()
            migrar("migracion.aplicada.auditoria.0001")
            migrar("migracion.existente.auditoria.0001")
            # El proceso de prueba sólo ve binarios de lectura y /tmp limitado.
            # La red interna de Compose no permite salir a servicios externos.
            entorno_pruebas = carpeta / "integracion.env"
            variables = {
                "ETF_POSTGRES_INTEGRACION": "1", "ETF_BD_HOST": "postgres", "ETF_BD_PUERTO": "5432",
                "ETF_BD_NOMBRE": "etf", "ETF_BD_CONTRASENA": credenciales["ETF_APP_CONTRASENA"],
                "ETF_MIGRADOR_CONTRASENA": credenciales["ETF_MIGRADOR_CONTRASENA"],
                "ETF_AUDITORIA_COMPROBADOR": "/opt/pruebas/comprobar-auditoria",
            }
            entorno_pruebas.write_text("".join(k + "=" + v + "\n" for k, v in variables.items()))
            entorno_pruebas.chmod(0o600)
            ejecutar(["docker", "run", "--rm", "--read-only", "--user", "70:70",
                      "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
                      "--memory", "128m", "--cpus", "1", "--pids-limit", "64",
                      "--network", proyecto + "_interna", "--env-file", str(entorno_pruebas),
                      "--mount", "type=bind,src=" + str(binarios) + ",dst=/opt/pruebas,readonly",
                      "--tmpfs", "/tmp:uid=70,gid=70,mode=0700,size=64m",
                      "--entrypoint", "/opt/pruebas/postgres.test", imagen,
                      "-test.v", "-test.run", "TestIntegracion", "-test.timeout", "60s"], entorno,
                     timeout=70)
            destruir()
            levantar()
            entorno["ETF_ENSAYO_MIGRACIONES"] = str(anterior)
            migrar("migracion.aplicada.plataforma.0001")
            entorno["ETF_ENSAYO_MIGRACIONES"] = str(RAIZ / "migraciones")
            migrar("migracion.aplicada.auditoria.0001")
            migrar("migracion.existente.auditoria.0001")
            print("ENSAYO-OK: limpia, repetida y actualización desde plataforma/0001", flush=True)
        finally:
            destruir()


if __name__ == "__main__":
    main()
