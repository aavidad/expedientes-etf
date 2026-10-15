"""Reutiliza la preparación y la siembra de la demo dentro de un contenedor local."""
import json
import os
from pathlib import Path
import sys

sys.path.insert(0, "/opt/etf/despliegue/demo")
import demo

demo.LOCAL = Path("/estado")
demo.PUERTO_GUARDADO = demo.LOCAL / "puerto"
guardar_original = demo.guardar


def guardar(ruta, contenido, modo=0o600):
    if ruta.name == "USUARIOS.local.md":
        ruta = demo.LOCAL / ruta.name
    guardar_original(ruta, contenido, modo)


demo.guardar = guardar
os.umask(0o077)
accion = sys.argv[1]
if accion == "preparar":
    entorno = sys.stdin.read()
    demo.preparar()
    guardar(Path("/datos/ca/servidor.pem"), (demo.LOCAL / "ldap/servidor.pem").read_text(), 0o644)
    Path("/datos/ca").chmod(0o755)
    guardar(demo.LOCAL / "entorno.env", entorno)
    configuracion = (demo.DEMO / "nginx.conf").read_text().replace(
        '"http://127.0.0.1:18083" "https://etf.demo.interno";',
        '"http://127.0.0.1:18083" "https://etf.demo.interno";\n        "http://localhost:18083" "https://etf.demo.interno";')
    guardar(Path("/datos/nginx/nginx.conf"), configuracion.replace(
        "http://127.0.0.1:18083", "http://127.0.0.1:" + demo.PUERTO).replace(
        "http://localhost:18083", "http://localhost:" + demo.PUERTO), 0o644)
    Path("/datos/nginx").chmod(0o755)
    # La base de la distribución se sustituye; lo publicado y lo cambiado a
    # mano se conservan. Si los textos no cargan, actualizar no arranca la app.
    from archivos import incorporar, validar_textos
    inventario = json.loads(Path("/opt/etf/inventario.json").read_text(encoding="utf-8"))
    for nombre in ("idiomas", "temas"):
        incorporar(demo.RAIZ / nombre, Path("/datos") / nombre, 10001, 10001, inventario)
    validar_textos(Path("/datos/idiomas"))
    print((demo.LOCAL / "usuarios.json").read_text())
elif accion == "sembrar":
    ejecutar_original = demo.ejecutar

    def ejecutar(orden, **opciones):
        if orden[0] == "curl":
            # Conserva URL, origen y cookies Secure de localhost. Sólo cambia
            # la conexión TCP al proxy dentro de la red de contenedores.
            orden = [*orden, "--connect-to", f"127.0.0.1:{demo.PUERTO}:entrada-demo:8080"]
        return ejecutar_original(orden, **opciones)

    demo.ejecutar = ejecutar
    demo.sembrar()
    from bandeja import sembrar_bandeja
    from equipo import sembrar_equipo
    from spaf import sembrar_spaf
    from fi1 import sembrar_fi1
    sembrar_bandeja(demo.RAIZ, demo.LOCAL, demo.URL, demo.Sesion, demo.ejecutar)
    sembrar_equipo(demo)
    sembrar_spaf(demo.RAIZ, demo.LOCAL, demo.URL, demo.Sesion)
    sembrar_fi1(demo)
    demo.comprobar()
else:
    raise SystemExit(2)
