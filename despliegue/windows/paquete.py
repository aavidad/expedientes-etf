"""Construye un ZIP autocontenido y su instalador .exe, sin secretos ni herramientas de desarrollo."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import urllib.request
import zipfile

RAIZ = Path(__file__).resolve().parents[2]
WINDOWS = RAIZ / "despliegue/windows"
DESCARGAS = RAIZ / "bin/descargas"

# Instaladores oficiales que instalar.cmd usa si faltan en el PC (WREQ).
# Versión y SHA-256 fijados; se descargan al construir y nunca en Windows.
# Podman 6 sólo publica el instalador MSI, que admite instalación por usuario.
REQUISITOS = (
    ("requisitos/podman-installer-windows-amd64.msi",
     "https://github.com/containers/podman/releases/download/v6.1.3/podman-installer-windows-amd64.msi",
     "8eb4b84b9130f915a83d2b659caa430270f0de4fe566f97c32fc1c919d28e22b"),
    ("requisitos/docker-compose.exe",
     "https://github.com/docker/compose/releases/download/v5.5.1/docker-compose-windows-x86_64.exe",
     "a3c0c73033eaede90210345d0cc2233edf4fab8fe0282a91dad8fd8436809d2f"),
)


# Idiomas del asistente: código del catálogo y nombre del idioma en NSIS.
IDIOMAS_NSIS = (("es", "SPANISH"), ("en", "ENGLISH"))
PREFIJO_INSTALADOR = "windows.instalador_"
# Firma opcional con el certificado de la entidad: rutas fuera del repositorio.
FIRMA = ("ETF_FIRMA_CERTIFICADO", "ETF_FIRMA_CLAVE")
LANZADOR = "Expedientes ETF.exe"


def ejecutar(*args, **kwargs):
    return subprocess.run(list(args), cwd=RAIZ, check=True, **kwargs)


def suma_fichero(ruta):
    with open(ruta, "rb") as fichero:
        return hashlib.file_digest(fichero, "sha256").hexdigest()


def descargar_requisito(url, suma, cache=DESCARGAS, abrir=urllib.request.urlopen):
    """Devuelve la copia local verificada; descarga sólo si falta o no cuadra."""
    if not url.startswith("https://github.com/") or not re.fullmatch(r"[0-9a-f]{64}", suma):
        raise SystemExit(f"Requisito sin origen o suma fijados: {url}")
    cache.mkdir(parents=True, exist_ok=True)
    destino = cache / suma
    if destino.is_file() and suma_fichero(destino) == suma:
        return destino
    temporal = cache / (suma + ".descarga")
    try:
        with abrir(url, timeout=120) as respuesta, open(temporal, "wb") as salida:
            shutil.copyfileobj(respuesta, salida)
        obtenida = suma_fichero(temporal)
        if obtenida != suma:
            raise SystemExit(f"SHA-256 inesperado para {url}: {obtenida}")
        temporal.replace(destino)
    finally:
        temporal.unlink(missing_ok=True)
    return destino


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


def escapar_nsis(texto):
    """Cadena literal de NSIS entre comillas dobles."""
    return (texto.replace("$", "$$").replace('"', '$\\"').replace("\r", "")
            .replace("\n", "$\\r$\\n").replace("\t", "$\\t"))


def idiomas_nsis(raiz=RAIZ):
    """LangString de NSIS generados desde idiomas/<código>/textos/windows.json."""
    lineas, claves = [], None
    for codigo, nombre in IDIOMAS_NSIS:
        textos = json.loads((raiz / "idiomas" / codigo / "textos/windows.json").read_text(encoding="utf-8"))
        propios = {k[len("windows."):]: v for k, v in textos.items() if k.startswith(PREFIJO_INSTALADOR)}
        if claves is None:
            claves = set(propios)
        if not propios or set(propios) != claves:
            raise SystemExit(f"Textos del instalador incompletos en {codigo}.")
        for clave in sorted(propios):
            lineas.append(f'LangString {clave} ${{LANG_{nombre}}} "{escapar_nsis(propios[clave])}"')
    return "\n".join(lineas) + "\n"


def accesos_nsis(raiz=RAIZ):
    """Nombre y descripción de los accesos para accesos.nsh, del mismo catálogo que mensajes.json."""
    textos = json.loads((raiz / "idiomas/es/textos/windows.json").read_text(encoding="utf-8"))
    return [f"-DACCESO_{clave.upper()}={escapar_nsis(textos['windows.acceso_' + clave])}"
            for clave in ("nombre", "descripcion")]


def archivos_nsis(carpeta):
    """Órdenes del desinstalador: sólo los archivos y carpetas copiados."""
    lineas, carpetas = [], set()
    for ruta in sorted(p for p in carpeta.rglob("*") if p.is_file()):
        relativa = ruta.relative_to(carpeta)
        if not re.fullmatch(r"[\w.-]+( [\w.-]+)*(/[\w.-]+)*", relativa.as_posix()):
            raise SystemExit(f"Nombre no admitido en el instalador: {relativa}")
        lineas.append(f'Delete "$INSTDIR\\{relativa.as_posix().replace("/", chr(92))}"')
        carpetas.update(relativa.parents)
    for subcarpeta in sorted(carpetas - {Path(".")}, key=lambda c: len(c.parts), reverse=True):
        lineas.append(f'RMDir "$INSTDIR\\{subcarpeta.as_posix().replace("/", chr(92))}"')
    return "\n".join(lineas) + "\n"


def version_numerica(version):
    """Versión X.X.X.X para las propiedades del .exe, si la etiqueta es numérica."""
    partes = re.fullmatch(r"v?(\d{1,5})(?:\.(\d{1,5}))?(?:\.(\d{1,5}))?(?:\.(\d{1,5}))?", version)
    if not partes or any(int(p) > 65535 for p in partes.groups() if p):
        return None
    return ".".join(p or "0" for p in partes.groups())


def escribir_suma(archivo):
    huella = suma_fichero(archivo)
    archivo.with_name(archivo.name + ".sha256").write_text(f"{huella}  {archivo.name}\n", encoding="ascii")
    return huella


def firmar(archivo, entorno=os.environ, ejecutar_orden=subprocess.run):
    """Firma con osslsigncode si se dan certificado y clave; si no, lo deja sin firmar."""
    certificado, clave = (entorno.get(nombre, "") for nombre in FIRMA)
    if not certificado and not clave:
        return False
    if not certificado or not clave:
        raise SystemExit("Para firmar hacen falta ETF_FIRMA_CERTIFICADO y ETF_FIRMA_CLAVE.")
    for ruta in (certificado, clave):
        if not Path(ruta).is_file() or Path(ruta).resolve().is_relative_to(RAIZ):
            raise SystemExit("El certificado y la clave deben existir fuera del repositorio.")
    firmado = archivo.with_name(archivo.name + ".firmado")
    orden = [shutil.which("osslsigncode") or "osslsigncode", "sign", "-h", "sha256",
             "-certs", certificado, "-key", clave, "-n", "Expedientes ETF", "-in", str(archivo), "-out", str(firmado)]
    if entorno.get("ETF_FIRMA_CONTRASENA_FICHERO"):
        orden += ["-readpass", entorno["ETF_FIRMA_CONTRASENA_FICHERO"]]
    if entorno.get("ETF_FIRMA_SELLO_URL"):
        orden += ["-ts", entorno["ETF_FIRMA_SELLO_URL"]]
    try:
        ejecutar_orden(orden, check=True)
        firmado.replace(archivo)
    finally:
        firmado.unlink(missing_ok=True)
    return True


def buscar_makensis(makensis=None):
    makensis = makensis or shutil.which("makensis")
    if not makensis:
        raise SystemExit("Falta makensis (NSIS 3.10 o posterior) para generar el lanzador y el instalador .exe.")
    return makensis


def construir_lanzador(carpeta, makensis=None):
    """«Expedientes ETF.exe»: los accesos lo abren en lugar de PowerShell."""
    destino = carpeta / LANZADOR
    subprocess.run([buscar_makensis(makensis), "-V2", "-NOCD", f"-DICONO={carpeta / 'aplicacion.ico'}",
                    f"-DSALIDA={destino}", *accesos_nsis(), str(WINDOWS / "lanzador.nsi")], cwd=carpeta, check=True)
    return destino


def construir_instalador(carpeta, version, salida, makensis=None, raiz=RAIZ):
    """Compila Expedientes-ETF-<versión>-instalador.exe con NSIS desde la carpeta del paquete."""
    makensis = buscar_makensis(makensis)
    archivo = salida / f"Expedientes-ETF-{version}-instalador.exe"
    with tempfile.TemporaryDirectory(prefix="win1-nsis-") as temporal:
        temporal = Path(temporal)
        # Con BOM, makensis lee los textos como UTF-8.
        (temporal / "idiomas.nsh").write_text(idiomas_nsis(raiz), encoding="utf-8-sig")
        (temporal / "archivos.nsh").write_text(archivos_nsis(carpeta), encoding="utf-8-sig")
        orden = [makensis, "-V2", "-NOCD", f"-DCARPETA={carpeta}", f"-DSALIDA={archivo}", f"-DVERSION={version}",
                 f"-DIDIOMAS={temporal / 'idiomas.nsh'}", f"-DARCHIVOS={temporal / 'archivos.nsh'}", *accesos_nsis(raiz)]
        if version_numerica(version):
            orden.append(f"-DVERSION_NUMERICA={version_numerica(version)}")
        if (raiz / "LICENSE").is_file():
            orden.append(f"-DLICENCIA={raiz / 'LICENSE'}")
        orden.append(str(WINDOWS / "instalador.nsi"))
        subprocess.run(orden, cwd=temporal, check=True)
    firmar(archivo)
    huella = escribir_suma(archivo)
    print(f"INSTALADOR-OK: {archivo.name}; {archivo.stat().st_size} bytes; SHA-256 {huella}")
    return archivo


def leer_version():
    version = os.environ.get("VERSION", "") or ejecutar("git", "rev-parse", "--short=12", "HEAD", capture_output=True, text=True).stdout.strip()
    if not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}", version):
        raise SystemExit("VERSION debe ser una etiqueta válida de hasta 64 caracteres.")
    return version


def instalador_desde_zip(version, salida=RAIZ / "bin/paquete-windows"):
    """Genera el .exe a partir de un ZIP ya construido y comprobado con su suma."""
    archivo = salida / f"expedientes-etf-windows-{version}.zip"
    suma = archivo.with_suffix(".zip.sha256")
    if not archivo.is_file() or not suma.is_file():
        raise SystemExit(f"Falta {archivo.name} o su suma: ejecuta antes make paquete-windows.")
    if suma.read_text(encoding="ascii").split()[0] != suma_fichero(archivo):
        raise SystemExit(f"La suma SHA-256 de {archivo.name} no coincide.")
    with tempfile.TemporaryDirectory(prefix="win1-zip-") as temporal, zipfile.ZipFile(archivo) as zip_:
        for nombre in zip_.namelist():
            if not nombre.startswith("Expedientes-ETF/") or ".." in Path(nombre).parts:
                raise SystemExit(f"Entrada no admitida en el ZIP: {nombre}")
        zip_.extractall(temporal)
        return construir_instalador(Path(temporal) / "Expedientes-ETF", version, salida)


def main():
    argumentos = argparse.ArgumentParser(description=__doc__)
    argumentos.add_argument("--solo-instalador", action="store_true",
                            help="genera sólo el .exe a partir del ZIP de bin/paquete-windows")
    if argumentos.parse_args().solo_instalador:
        instalador_desde_zip(leer_version())
        return
    version = leer_version()
    # Primero los instaladores: si una suma no cuadra, se para antes de compilar.
    requisitos = {nombre: descargar_requisito(url, suma) for nombre, url, suma in REQUISITOS}
    buscar_makensis()
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
        firmar(construir_lanzador(carpeta))
        (carpeta / "capturas").mkdir()
        for nombre in set(re.findall(r"capturas/(windows-[\w-]+\.png)", (carpeta / "LEEME.md").read_text())):
            shutil.copyfile(RAIZ / "docs/capturas" / nombre, carpeta / "capturas" / nombre)
        shutil.copyfile(RAIZ / "internal/adaptadores/anexos/docx/testdata/anexo3.docx", carpeta / "Anexo-3-pruebas.docx")
        shutil.copyfile(RAIZ / "internal/adaptadores/anexos/odt/testdata/anexo3.odt", carpeta / "Anexo-3-pruebas.odt")
        shutil.copyfile(RAIZ / "internal/adaptadores/anexos/actas/testdata/aceptacion-2022.docx", carpeta / "Acta-aceptacion-pruebas.docx")
        for nombre, origen in requisitos.items():
            (carpeta / nombre).parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(origen, carpeta / nombre)
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
        huella = escribir_suma(archivo)
        print(f"PAQUETE-OK: {archivo.name}; SHA-256 {huella}")
        # El mismo contenido, como asistente de instalación para Windows.
        construir_instalador(carpeta, version, salida)


if __name__ == "__main__":
    main()
