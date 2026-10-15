"""Construye un ZIP autocontenido, sin secretos ni herramientas de desarrollo."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import zipfile

RAIZ = Path(__file__).resolve().parents[2]
WINDOWS = RAIZ / "despliegue/windows"


def ejecutar(*args, **kwargs):
    return subprocess.run(list(args), cwd=RAIZ, check=True, **kwargs)


def configurar(config, version):
    config["name"] = "etf-windows"
    servicios = config["services"]
    # Windows incorpora los recursos mediante su servicio herramientas.
    servicios.pop("incorporar", None)
    for nombre, servicio in servicios.items():
        if isinstance(servicio.get("environment"), list):
            servicio["environment"] = dict(linea.split("=", 1) for linea in servicio["environment"])
        servicio.pop("build", None)
        servicio.pop("logging", None)
        servicio.pop("restart", None)
        if "tmpfs" in servicio:
            servicio["tmpfs"] = [re.sub(r"uid=\d+,|gid=\d+,", "", montaje).replace("mode=0700", "mode=1777")
                                for montaje in servicio["tmpfs"]]
        if nombre in ("app", "migrar", "ldap", "postgres", "clamav", "entrada-demo"):
            servicio["image"] = f"localhost/etf-windows-{nombre}:{version}"
        if nombre == "preparar-roles":
            servicio["image"] = f"localhost/etf-windows-postgres:{version}"
        # Los archivos operativos se incluyen en imágenes; no hay bind mounts.
        servicio["volumes"] = [v for v in servicio.get("volumes", []) if v["type"] == "volume"]
    servicios["ldap"]["volumes"] = [{"type": "volume", "source": "ldap", "target": "/fixture", "read_only": True}]
    # WSL conserva los procesadores del equipo: slapadd necesita más memoria
    # durante la primera carga que en el contenedor de la demo Linux.
    servicios["ldap"]["mem_limit"] = "512m"
    servicios["app"]["volumes"].append({"type": "volume", "source": "ca", "target": "/demo", "read_only": True})
    servicios["app"]["environment"]["ETF_HTTP_PROXIES"] = "10.254.241.0/28"
    servicios["app"]["environment"]["ETF_AD_CA_RUTA"] = "/demo/servidor.pem"
    servicios["entrada-demo"]["ports"] = [{"target": 8080, "published": "${ETF_WINDOWS_PUERTO:-18083}", "host_ip": "127.0.0.1", "protocol": "tcp"}]
    servicios["entrada-demo"]["volumes"] = ["nginx:/etc/nginx:ro"]
    # Volumen socket sin opciones del host: compatible con la máquina WSL de Podman.
    config["volumes"]["postgres_socket"] = {}
    config["volumes"].update({"estado": {}, "ldap": {}, "ca": {}, "nginx": {}})
    for volumen in config["volumes"].values():
        volumen.pop("name", None)
    for nombre, subred in (("interna", "10.254.241.0/28"), ("entrada", "10.254.241.16/28")):
        config["networks"][nombre] = {"ipam": {"config": [{"subnet": subred}]}}
        if nombre == "interna":
            config["networks"][nombre]["internal"] = True
    servicios["herramientas"] = {
        "image": f"localhost/etf-windows-preparar:{version}",
        "profiles": ["herramientas"], "read_only": True,
        "cap_drop": ["ALL"], "cap_add": ["CHOWN", "FOWNER", "DAC_OVERRIDE"],
        "security_opt": ["no-new-privileges:true"], "networks": ["entrada"],
        "environment": {"ETF_DEMO_PUERTO": "${ETF_WINDOWS_PUERTO:-18083}"},
        "tmpfs": ["/tmp:size=128m,mode=1777"], "mem_limit": "256m", "pids_limit": 64,
        "volumes": [f"{origen}:{destino}" for origen, destino in (
            ("estado", "/estado"), ("ldap", "/estado/ldap"),
            ("demo_idiomas", "/datos/idiomas"), ("demo_temas", "/datos/temas"), ("nginx", "/datos/nginx"), ("ca", "/datos/ca"))],
    }
    return config


def main():
    version = os.environ.get("VERSION", "") or ejecutar("git", "rev-parse", "--short=12", "HEAD", capture_output=True, text=True).stdout.strip()
    if not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}", version):
        raise SystemExit("VERSION debe ser una etiqueta válida de hasta 64 caracteres.")
    salida = RAIZ / "bin/paquete-windows"
    salida.mkdir(parents=True, exist_ok=True)
    spec = importlib.util.spec_from_file_location("demo", RAIZ / "despliegue/demo/demo.py")
    demo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(demo)
    demo.comprobar_docker_local()
    demo.LOCAL.mkdir(parents=True, exist_ok=True)
    demo.compilar()
    imagenes = []
    for nombre in ("app", "migrar", "ldap", "postgres", "clamav", "entrada-demo", "preparar"):
        imagen = f"localhost/etf-windows-{nombre}:{version}"
        orden = ["docker", "build", "--tag", imagen]
        if nombre in ("app", "migrar"):
            orden += ["--target", nombre, str(RAIZ / "despliegue/demo")]
        elif nombre == "ldap":
            orden += [str(RAIZ / "internal/adaptadores/ldap/testdata")]
        else:
            # Contexto permitido explícitamente: excluye .local y credenciales.
            with tempfile.TemporaryDirectory(prefix="win1-contexto-") as temporal:
                contexto = Path(temporal)
                for ruta in ("despliegue/windows", "despliegue/base", "despliegue/demo",
                             "internal/adaptadores/ldap/testdata", "internal/adaptadores/anexos/docx/testdata", "idiomas", "temas"):
                    shutil.copytree(RAIZ / ruta, contexto / ruta,
                                    ignore=shutil.ignore_patterns(".local", "*.local*", "__pycache__"))
                # Inventario de versiones históricas para sustituir sólo lo de la distribución.
                (contexto / "despliegue/windows/inventario.json").write_text(
                    json.dumps(demo.inventario_distribucion()) + "\n", encoding="utf-8")
                destino = "entrada" if nombre == "entrada-demo" else nombre
                ejecutar(*orden, "--target", destino, "--file", str(contexto / "despliegue/windows/Dockerfile"), str(contexto))
                imagenes.append(imagen)
                continue
        ejecutar(*orden)
        imagenes.append(imagen)
    config = json.loads(ejecutar("docker", "compose", "-f", "compose.yaml", "-f", "despliegue/demo/compose.demo.yaml",
                               "config", "--no-interpolate", "--format", "json", capture_output=True, text=True).stdout)
    with tempfile.TemporaryDirectory(prefix="win1-paquete-") as temporal:
        carpeta = Path(temporal)
        for patron in ("*.ps1", "*.cmd", "LEEME.md"):
            for ruta in WINDOWS.glob(patron):
                shutil.copyfile(ruta, carpeta / ruta.name)
        (carpeta / "LEEME.md").write_text((carpeta / "LEEME.md").read_text().replace("../../docs/capturas/", "capturas/"), encoding="utf-8")
        shutil.copyfile(RAIZ / "idiomas/es/textos/windows.json", carpeta / "mensajes.json")
        ejecutar("node", str(WINDOWS / "icono.mjs"), str(carpeta / "aplicacion.ico"))
        (carpeta / "capturas").mkdir()
        for nombre in set(re.findall(r"capturas/(windows-[\w-]+\.png)", (carpeta / "LEEME.md").read_text())):
            shutil.copyfile(RAIZ / "docs/capturas" / nombre, carpeta / "capturas" / nombre)
        shutil.copyfile(RAIZ / "internal/adaptadores/anexos/docx/testdata/anexo3.docx", carpeta / "Anexo-3-pruebas.docx")
        shutil.copyfile(RAIZ / "internal/adaptadores/anexos/odt/testdata/anexo3.odt", carpeta / "Anexo-3-pruebas.odt")
        shutil.copyfile(RAIZ / "internal/adaptadores/anexos/actas/testdata/aceptacion-2022.docx", carpeta / "Acta-aceptacion-pruebas.docx")
        (carpeta / "compose.json").write_text(json.dumps(configurar(config, version), indent=2), encoding="utf-8")
        (carpeta / "VERSION.txt").write_text(version + "\n", encoding="ascii")
        ejecutar("docker", "save", "--output", str(carpeta / "imagenes.tar"), *imagenes)
        sumas = {str(p.relative_to(carpeta)): hashlib.file_digest(p.open("rb"), "sha256").hexdigest()
                 for p in carpeta.rglob("*") if p.is_file()}
        (carpeta / "SUMAS.json").write_text(json.dumps(sumas, indent=2), encoding="utf-8")
        archivo = salida / f"expedientes-etf-windows-{version}.zip"
        with zipfile.ZipFile(archivo, "w", zipfile.ZIP_DEFLATED, compresslevel=1) as zip_:
            for p in sorted(carpeta.rglob("*")):
                if p.is_file():
                    zip_.write(p, "Expedientes-ETF/" + str(p.relative_to(carpeta)))
        huella = hashlib.file_digest(archivo.open("rb"), "sha256").hexdigest()
        archivo.with_suffix(".zip.sha256").write_text(f"{huella}  {archivo.name}\n", encoding="ascii")
        print(f"PAQUETE-OK: {archivo.name}; SHA-256 {huella}")


if __name__ == "__main__":
    main()
