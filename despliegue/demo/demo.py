#!/usr/bin/env python3
"""Prepara y comprueba únicamente la demostración local con datos inventados."""

import base64
import contextlib
import datetime
import fcntl
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import sys
import tempfile
import uuid


DEMO = Path(__file__).resolve().parent
RAIZ = DEMO.parent.parent
LOCAL = DEMO / ".local"
PUERTO_GUARDADO = LOCAL / "puerto"
PUERTO = os.environ.get("ETF_DEMO_PUERTO", "") or (
    PUERTO_GUARDADO.read_text().strip() if PUERTO_GUARDADO.exists() else "18083"
)
if not PUERTO.isascii() or not PUERTO.isdecimal() or not 1024 <= int(PUERTO) <= 65535:
    raise SystemExit("ETF_DEMO_PUERTO debe ser un número entre 1024 y 65535.")
PUERTO = str(int(PUERTO))
if PUERTO_GUARDADO.exists() and PUERTO_GUARDADO.read_text().strip() != PUERTO:
    raise SystemExit("La demostración ya tiene otro puerto; usa demo-limpiar antes de cambiarlo.")
URL = "http://127.0.0.1:" + PUERTO
PROYECTO = "etf-demo-" + hashlib.sha256(str(RAIZ).encode()).hexdigest()[:8]
RED_GUARDADA = LOCAL / "red"
RED = os.environ.get("ETF_DEMO_RED", "") or (
    RED_GUARDADA.read_text().strip() if RED_GUARDADA.exists()
    else ""
)
try:
    if RED and ipaddress.IPv4Address(RED + ".0") not in ipaddress.IPv4Network("10.0.0.0/8"):
        raise ValueError()
except ValueError:
    raise SystemExit("ETF_DEMO_RED debe ser un prefijo privado como 10.253.11.")
if RED_GUARDADA.exists() and RED_GUARDADA.read_text().strip() != RED:
    raise SystemExit("La demostración ya tiene otra red; usa demo-limpiar antes de cambiarla.")
PERSONAS = [
    ("elena.salas", "Elena Salas Márquez", "administracion", None),
    ("ines.moreno", "Inés Moreno Castillo", "coordinacion", None),
    ("javier.medina", "Javier Medina Robles", "coordinacion", None),
    ("marta.cabrera", "Marta Cabrera Molina", "profesional_etf", "02"),
    ("pablo.ruiz", "Pablo Ruiz Vílchez", "profesional_etf", "04"),
    ("lucia.navarro", "Lucía Navarro Benítez", "consulta", None),
    ("sergio.campos", "Sergio Campos Roldán", None, None),
]


def ejecutar(orden, **opciones):
    opciones.setdefault("cwd", RAIZ)
    return subprocess.run(orden, check=True, **opciones)


def comprobar_docker_local():
    destino = os.environ.get("DOCKER_HOST", "")
    contexto = ejecutar(["docker", "context", "inspect", "--format",
                         "{{.Endpoints.docker.Host}}"], capture_output=True, text=True).stdout.strip()
    if (destino and not destino.startswith("unix://")) or not contexto.startswith("unix://"):
        raise RuntimeError("La demostración sólo permite Docker en este equipo.")


def compose(*argumentos, **opciones):
    comprobar_docker_local()
    # El entorno del operador no puede sustituir los secretos ni el proyecto.
    entorno = {k: v for k, v in os.environ.items()
               if not k.startswith(("ETF_", "COMPOSE_"))}
    entorno["ETF_DEMO_PUERTO"] = PUERTO
    # Conserva las subredes por worktree y el prefijo opcional ya guardado.
    tercero, bloque = divmod(int(hashlib.sha256(str(RAIZ).encode()).hexdigest()[:4], 16) % 2048, 8)
    subredes = {
        "ETF_DEMO_RED_SUBRED": f"{RED}.0/28" if RED else f"10.253.{tercero}.{bloque * 32}/28",
        "ETF_DEMO_ENTRADA_SUBRED": f"{RED}.16/28" if RED else f"10.253.{tercero}.{bloque * 32 + 16}/28",
    }
    # Las dos opciones de red del entorno local no son secretos ni comandos.
    for linea in (LOCAL / "entorno.env").read_text().splitlines():
        clave, separador, valor = linea.partition("=")
        if separador and clave in subredes:
            red = ipaddress.IPv4Network(valor.strip())
            if not red.is_private or red.prefixlen != 28:
                raise RuntimeError("Las subredes de demostración deben ser privadas y /28.")
            subredes[clave] = str(red)
    entorno.update(subredes)
    return ejecutar([
        "docker", "compose", "--project-directory", str(RAIZ),
        "--env-file", str(LOCAL / "entorno.env"), "-p", PROYECTO,
        "-f", str(RAIZ / "compose.yaml"), "-f", str(DEMO / "compose.demo.yaml"),
        *argumentos,
    ], env=entorno, **opciones)


def guardar(ruta, contenido, modo=0o600):
    ruta.write_text(contenido, encoding="utf-8")
    ruta.chmod(modo)


def preparar():
    if (LOCAL / "entorno.env").exists():
        # Las demos anteriores conservan sus datos y todas sus credenciales.
        ruta = LOCAL / "entorno.env"
        contenido = ruta.read_text()
        if not any(linea.startswith("ETF_CALCULO_MEMORIA_CONTRASENA=") for linea in contenido.splitlines()):
            # Publicar el archivo completo evita perder las claves si se interrumpe.
            with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", dir=LOCAL,
                                             prefix=".entorno-", delete=False) as nuevo:
                temporal = Path(nuevo.name)
                try:
                    temporal.chmod(0o600)
                    nuevo.write(contenido.rstrip("\n") + "\nETF_CALCULO_MEMORIA_CONTRASENA="
                                + secrets.token_urlsafe(32) + "\n")
                    nuevo.flush()
                    os.fsync(nuevo.fileno())
                    os.replace(temporal, ruta)
                finally:
                    temporal.unlink(missing_ok=True)
        return
    LOCAL.mkdir(mode=0o700, exist_ok=True)
    ldap = LOCAL / "ldap"
    ldap.mkdir(mode=0o755, exist_ok=True)
    ldap.chmod(0o755)
    claves = {cuenta: secrets.token_urlsafe(24) for cuenta, *_ in PERSONAS}
    guardar(LOCAL / "usuarios.json", json.dumps(claves))
    guardar(ldap / "identidad.schema", (
        RAIZ / "internal/adaptadores/ldap/testdata/identidad.schema"
    ).read_text() + """
attributetype ( 1.3.6.1.4.1.4203.666.99.3 NAME 'sAMAccountName'
  EQUALITY caseIgnoreMatch SYNTAX 1.3.6.1.4.1.1466.115.121.1.15 SINGLE-VALUE )
objectclass ( 1.3.6.1.4.1.4203.666.99.4 NAME 'cuentaETF'
  SUP top AUXILIARY MAY sAMAccountName )
""", 0o644)
    shutil.copyfile(RAIZ / "internal/adaptadores/ldap/testdata/slapd.conf", ldap / "slapd.conf")
    (ldap / "slapd.conf").chmod(0o644)
    ldif = """dn: dc=etf,dc=interno
objectClass: top
objectClass: domain
dc: etf

dn: ou=personas,dc=etf,dc=interno
objectClass: organizationalUnit
ou: personas

"""
    for cuenta, nombre, *_ in PERSONAS:
        sal = secrets.token_bytes(16)
        clave = "{SSHA}" + base64.b64encode(
            hashlib.sha1(claves[cuenta].encode() + sal).digest() + sal
        ).decode()
        guid = uuid.uuid5(uuid.NAMESPACE_URL, "etf.demo/" + cuenta)
        nombre64 = base64.b64encode(nombre.encode()).decode()
        apellido64 = base64.b64encode(nombre.split(" ", 1)[1].encode()).decode()
        ldif += f"""dn: uid={cuenta},ou=personas,dc=etf,dc=interno
objectClass: inetOrgPerson
objectClass: identidadETF
objectClass: cuentaETF
uid: {cuenta}
sAMAccountName: {cuenta}
cn:: {nombre64}
sn:: {apellido64}
displayName:: {nombre64}
userPassword: {clave}
objectGUID:: {base64.b64encode(guid.bytes_le).decode()}

"""
    guardar(ldap / "datos.ldif", ldif, 0o644)
    ejecutar([
        "openssl", "req", "-x509", "-newkey", "rsa:3072", "-sha256", "-nodes",
        "-days", "365", "-subj", "/CN=ldap", "-addext", "subjectAltName=DNS:ldap",
        "-keyout", str(ldap / "servidor.key"), "-out", str(ldap / "servidor.pem"),
    ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    for fichero in ("servidor.key", "servidor.pem"):
        (ldap / fichero).chmod(0o644)
    textos = {
        "administracion": "Administración funcional y primer administrador",
        "coordinacion": "Coordinación", "profesional_etf": "Profesional",
        "consulta": "Consulta", None: "Pendiente de alta",
    }
    filas = ["# Usuarios de demostración\n\nTodas las personas son inventadas.\n",
             "| Persona | Usuario | Contraseña | Perfil |\n| --- | --- | --- | --- |"]
    for cuenta, nombre, perfil, equipo in PERSONAS:
        descripcion = textos[perfil] + (f" del ETF {int(equipo)}" if equipo else "")
        filas.append(f"| {nombre} | `{cuenta}` | `{claves[cuenta]}` | {descripcion} |")
    guardar(DEMO / "USUARIOS.local.md", "\n".join(filas) + "\n")
    guardar(LOCAL / "entorno.env", "".join(
        f"ETF_{rol}_CONTRASENA={secrets.token_urlsafe(32)}\n"
        for rol in ("INSTALADOR", "MIGRADOR", "APP", "COPIA", "CALCULO_MEMORIA")
    ) + f"ETF_DEMO_SUBRED_INTERNA=10.253.{int(PROYECTO[-8:-6], 16)}.{int(PROYECTO[-6:-4], 16) % 8 * 32}/28\n"
      + f"ETF_DEMO_SUBRED_ENTRADA=10.253.{int(PROYECTO[-8:-6], 16)}.{int(PROYECTO[-6:-4], 16) % 8 * 32 + 16}/28\n")


def preparar_entrada():
    guardar(PUERTO_GUARDADO, PUERTO + "\n")
    if RED:
        guardar(RED_GUARDADA, RED + "\n")
    guardar(LOCAL / "nginx.conf", (DEMO / "nginx.conf").read_text().replace(
        "http://127.0.0.1:18083", URL
    ), 0o644)


def compilar():
    ejecutar(["npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"], cwd=RAIZ / "web")
    ejecutar(["npm", "run", "build"], cwd=RAIZ / "web")
    binarios = LOCAL / "bin"
    binarios.mkdir(exist_ok=True)
    # go.mod fija la versión exacta de Go; si el puesto tiene otra, Go usa la
    # cadena de herramientas de esa versión, verificada con su suma pública.
    entorno = dict(os.environ, CGO_ENABLED="0", GOOS="linux", GOTOOLCHAIN="auto")
    for nombre in ("etf", "migrar"):
        ejecutar(["go", "build", "-mod=readonly", "-trimpath", "-o",
                  str(binarios / nombre), f"./cmd/{nombre}"], env=entorno)
        (binarios / nombre).chmod(0o755)
    shutil.copytree(RAIZ / "migraciones", LOCAL / "migraciones", dirs_exist_ok=True)
    shutil.copytree(RAIZ / "datos" / "catalogos", LOCAL / "datos" / "catalogos", dirs_exist_ok=True)
    for nombre in ("idiomas", "temas"):
        destino = LOCAL / nombre
        if destino.exists():
            shutil.rmtree(destino)
        shutil.copytree(RAIZ / nombre, destino)


class Sesion:
    """Usa curl real; los secretos viajan por stdin, nunca por los argumentos."""

    def __init__(self, carpeta, cuenta):
        self.cookies = carpeta / cuenta
        self.csrf = ""

    def pedir(self, metodo, ruta, datos=None, esperado=None, origen=URL, fichero=None, clave=None):
        if esperado is None:
            esperado = 204 if metodo == "DELETE" else 200
        config = {
            "url": URL + ruta, "request": metodo,
            "cookie": str(self.cookies), "cookie-jar": str(self.cookies),
            "header": "Origin: " + origen,
        }
        lineas = [f"{k} = {json.dumps(v)}" for k, v in config.items()]
        if metodo != "GET":
            lineas += [f"header = {json.dumps('X-CSRF-Token: ' + self.csrf)}"]
        if clave:
            lineas += ["header = " + json.dumps("Idempotency-Key: " + clave)]
        if datos is not None:
            lineas += ['header = "Content-Type: application/json"',
                       "data = " + json.dumps(json.dumps(datos))]
        if fichero is not None:
            lineas += ['form = ' + json.dumps('anexo=@' + str(fichero))]
        respuesta = ejecutar([
            "curl", "--silent", "--show-error", "--max-time", "15", "--noproxy", "*",
            "--config", "-", "--write-out", "\n%{http_code}",
        ], input="\n".join(lineas), capture_output=True, text=True).stdout
        cuerpo, estado = respuesta.rsplit("\n", 1)
        if int(estado) != esperado:
            raise RuntimeError(f"{metodo} {ruta}: HTTP {estado}; se esperaba {esperado}.")
        contenido = json.loads(cuerpo) if cuerpo else {}
        self.csrf = contenido.get("csrf", self.csrf) if isinstance(contenido, dict) else self.csrf
        return contenido

    def entrar(self, cuenta, claves):
        self.pedir("GET", "/api/sesion")
        return self.pedir("POST", "/api/sesion", {"cuenta": cuenta, "contrasena": claves[cuenta]})


def sembrar():
    if (LOCAL / "sembrado").exists():
        return
    claves = json.loads((LOCAL / "usuarios.json").read_text())
    registro = LOCAL / "asignaciones.json"
    intentadas = json.loads(registro.read_text()) if registro.exists() else []
    with tempfile.TemporaryDirectory() as temporal:
        carpeta = Path(temporal)
        admin = Sesion(carpeta, PERSONAS[0][0])
        acceso = admin.entrar(PERSONAS[0][0], claves)
        if acceso.get("perfiles") != ["administracion"]:
            raise RuntimeError("No se concedió el primer administrador; no se forzarán sus permisos.")
        catalogo = admin.pedir("GET", "/api/admin/catalogos/asignacion_municipios/datos?fecha=2026-07-01")
        equipos = {e["codigo"]: e["id"] for e in catalogo["equipos"]}
        for cuenta, _, perfil, equipo in PERSONAS[1:]:
            persona = Sesion(carpeta, cuenta)
            persona.entrar(cuenta, claves)
            if perfil:
                listado = admin.pedir("GET", "/api/admin/usuarios")
                usuario = next(u for u in listado if u["cuenta"] == cuenta)
                if usuario["perfiles"] or usuario["equipo"]:
                    if ([p["perfil"] for p in usuario["perfiles"]] != [perfil]
                            or (usuario["equipo"] or {}).get("equipo_id") != equipos.get(equipo)):
                        raise RuntimeError("La siembra incompleta encontró otros permisos; revisa los usuarios.")
                else:
                    if cuenta in intentadas:
                        raise RuntimeError("Hay una concesión anterior o sin confirmar; no se repondrán permisos retirados.")
                    desde = datetime.datetime.now(datetime.timezone.utc).isoformat()
                    adscripcion = {"equipo_id": equipos[equipo], "puesto": "ts",
                                   "desde": desde, "hasta": None} if equipo else None
                    # Se registra antes de escribir: un corte de conexión nunca
                    # autoriza a reponer automáticamente una concesión retirada.
                    intentadas.append(cuenta)
                    guardar(registro, json.dumps(intentadas))
                    admin.pedir("PUT", f"/api/admin/usuarios/{usuario['id']}/perfiles", {
                        "version": usuario["version"], "estado": "activo", "equipo": adscripcion,
                        "perfiles": [{"perfil": perfil, "desde": desde, "hasta": None}],
                    })
            persona.pedir("DELETE", "/api/sesion")
        firmantes = admin.pedir("GET", "/api/admin/firmantes")
        if not any(f["rol"] == "aceptacion" for f in firmantes["firmantes"]):
            if "firmante" in intentadas:
                raise RuntimeError("Hay una concesión de firmante sin confirmar; revisa la siembra.")
            intentadas.append("firmante")
            guardar(registro, json.dumps(intentadas))
            textos = {
                idioma: json.loads((RAIZ / "idiomas" / idioma / "textos" / "demo.json").read_text())["demo.cargo_aceptacion"]
                for idioma in ("es", "en")
            }
            admin.pedir("POST", "/api/admin/firmantes", {
                "version_esperada": 0, "referencias_esperadas": firmantes["referencias"],
                "vigente_desde": "2026-01-01T00:00:00Z", "textos": textos,
                "firmante": {"tipo": "firmante", "rol": "aceptacion", "nombre": PERSONAS[1][1],
                             "cargo_codigo": "COORDINACION", "organismo_codigo": "ENTIDAD_TITULAR"},
            }, esperado=201)
        admin.pedir("DELETE", "/api/sesion")
    guardar(LOCAL / "sembrado", "Siembra confirmada por la API después de migrar.\n")


def comprobar():
    claves = json.loads((LOCAL / "usuarios.json").read_text())
    with tempfile.TemporaryDirectory() as temporal:
        carpeta = Path(temporal)
        admin = Sesion(carpeta, "verificacion-admin")
        admin.entrar(PERSONAS[0][0], claves)
        catalogo = admin.pedir("GET", "/api/admin/catalogos/asignacion_municipios/datos?fecha=2026-07-01")
        equipos = {e["codigo"]: e["id"] for e in catalogo["equipos"]}
        listado = admin.pedir("GET", "/api/admin/usuarios")
        if len(listado) != len(PERSONAS) or len(equipos) != 8:
            raise RuntimeError("La demostración tiene usuarios duplicados o catálogos incompletos.")
        for cuenta, _, perfil, equipo in PERSONAS:
            persona = Sesion(carpeta, cuenta)
            acceso = persona.entrar(cuenta, claves)
            estado = "activo" if perfil else "pendiente_alta"
            if acceso["estado"] != estado:
                raise RuntimeError(f"Estado inesperado para {cuenta}.")
            if perfil:
                if acceso.get("perfiles") != [perfil] or acceso.get("equipo_id", "") != equipos.get(equipo, ""):
                    raise RuntimeError(f"Perfil o equipo incorrecto para {cuenta}.")
            elif set(acceso) != {"estado", "csrf"}:
                raise RuntimeError("La sesión pendiente recibe datos de identidad.")
            persona.pedir("GET", "/api/sesion")
            if perfil != "administracion":
                persona.pedir("GET", "/api/admin/usuarios", esperado=403)
            persona.pedir("DELETE", "/api/sesion", esperado=403, origen="http://ajeno.interno")
            persona.pedir("DELETE", "/api/sesion")
            print(f"SESION-OK {cuenta}: {estado}; {perfil or 'sin perfil'}" + (f"; ETF {int(equipo)}" if equipo else ""))
        admin.pedir("DELETE", "/api/sesion")
    print("DEMO-OK: 7 usuarios únicos, 8 equipos, perfiles, equipos, privacidad y rechazo CSRF comprobados con curl.")


def iniciar():
    # Nunca detiene otra instalación que esté ocupando el puerto solicitado.
    comprobar_docker_local()
    with contextlib.closing(socket.socket()) as prueba:
        try:
            prueba.bind(("127.0.0.1", int(PUERTO)))
        except OSError as error:
            puertos = ejecutar([
                "docker", "ps", "--filter", f"label=com.docker.compose.project={PROYECTO}",
                "--filter", "label=com.docker.compose.service=entrada-demo", "--format", "{{.Ports}}",
            ], capture_output=True, text=True).stdout
            if f"127.0.0.1:{PUERTO}->" not in puertos:
                raise RuntimeError(f"El puerto {PUERTO} está ocupado. Elige otro con ETF_DEMO_PUERTO.") from error
    preparar()
    preparar_entrada()
    compilar()
    compose("config", "--quiet")
    compose("up", "--detach", "--build", "--wait", "--wait-timeout", "180")
    # El proxy vuelve a resolver la dirección del contenedor recompilado.
    compose("restart", "entrada-demo")
    sembrar()
    from bandeja import sembrar_bandeja
    sembrar_bandeja(RAIZ, LOCAL, URL, Sesion, ejecutar)
    from equipo import sembrar_equipo
    sembrar_equipo(sys.modules[__name__])
    from spaf import sembrar_spaf
    sembrar_spaf(RAIZ, LOCAL, URL, Sesion)
    from fi1 import sembrar_fi1
    sembrar_fi1(sys.modules[__name__])
    print((DEMO / "USUARIOS.local.md").read_text())
    print(f"Demostración disponible en {URL}. Para pararla: make demo-parar.")


def main():
    os.umask(0o077)
    if len(sys.argv) != 2 or sys.argv[1] not in ("iniciar", "parar", "comprobar", "recorrer", "limpiar"):
        raise RuntimeError("Usa make demo, demo-parar, demo-comprobar, demo-recorrido o demo-limpiar.")
    # Serializa arranque, parada y comprobación para no mezclar credenciales.
    with (DEMO / ".operacion.local").open("w") as cerrojo:
        fcntl.flock(cerrojo, fcntl.LOCK_EX)
        accion = sys.argv[1]
        if accion == "iniciar":
            iniciar()
        elif accion == "comprobar":
            comprobar()
        elif accion == "recorrer":
            ejecutar(["node", str(DEMO / "recorrido.mjs"), URL], timeout=120)
        elif accion == "limpiar":
            if (LOCAL / "entorno.env").exists():
                compose("down", "--remove-orphans", "--volumes")
            if LOCAL.exists():
                shutil.rmtree(LOCAL)
            (DEMO / "USUARIOS.local.md").unlink(missing_ok=True)
            print("DEMO-LIMPIA")
        elif (LOCAL / "entorno.env").exists():
            compose("down", "--remove-orphans")
            print("DEMO-PARADA")
        else:
            print("DEMO-PARADA: no hay demostración preparada.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, subprocess.CalledProcessError, OSError) as error:
        # No imprime argumentos de procesos ni valores de entorno con secretos.
        print(str(error) if isinstance(error, RuntimeError) else "La operación local falló; revisa la salida anterior.", file=sys.stderr)
        sys.exit(1)
