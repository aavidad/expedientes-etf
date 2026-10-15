# WIN2: prueba del paquete Windows 1.0.1 en la máquina virtual

Fecha: 15/10/2026. Máquina: `etf-win10-pruebas` (Windows 10 19045, WSL 2,
Podman 6.1.4, Compose 5.1.4). Paquete construido desde el commit `488e4db9`,
sin cambios de código. Todo el recorrido usa datos inventados.

## Paquetes

| Paquete | SHA-256 | Comprobación en la VM |
| --- | --- | --- |
| privada `45713f869b12` (14/10) | `cdfcc5369aebad91431436968e89cc7f9f5871287e00e04b6d9ee5e1f3fbccdc` | coincide |
| privada `1.0.1` | `daa0e7c3f61f204e2bdc35490611a704e4596844a3ff69dfba4c523e10c69e6b` | coincide; `VERSION.txt` = 1.0.1 |

Los ZIP se sirvieron desde el anfitrión por HTTP en la red de gestión.

## Entorno: Podman no levanta win-sshproxy

Después de arrancar Windows, Podman Desktop se abre solo al iniciar sesión y
enciende la máquina WSL, pero **no levanta `win-sshproxy`**:

- `podman machine start` avisa «API forwarding for Docker API clients is not
  available … win-sshproxy.exe failed to start, exit code: 0»;
- falta la tubería `\\.\pipe\podman-machine-default`, que necesita Compose;
- sin ella, el acceso, `actualizar.cmd` e `instalar.cmd` acaban con «No se ha
  podido completar la operación compose (código 1)».

Se arregló lanzando `win-sshproxy.exe` a mano con los argumentos que muestra
`podman --log-level=debug machine start`. Abrir Podman Desktop no lo arregla.
Pasó en los dos arranques de la VM de este día. La guía para casa no cubre
este caso y quizá haya que añadirlo.

## Primer intento, descartado: v1.0.0 pública → 1.0.1 privada

La VM tenía la v1.0.0 **pública**. `actualizar.cmd` 1.0.1 se paró en `migrar`
con `migracion.huella`. Hay 13 migraciones de `documento` con huellas de
plantilla distintas entre las ramas pública y privada. No es un camino de
actualización válido, porque son dos distribuciones distintas. Los datos quedaron
intactos.

## 1. Actualización privada 45713f869b12 → 1.0.1: correcta

1. La v1.0.0 pública se desinstaló con `desinstalar.cmd` (DESINSTALAR, se borran datos).
2. Se instaló la privada `45713f869b12`. Coordinación aceptó una derivación
   desde la pantalla y quedó como expediente **04-02-26**.
   - Antes de actualizar: 31 expedientes, huella `a891af3cd3b6aec0806e5b75574daebc`
     (calculada sobre id, número y estado) y 61 migraciones.
3. `actualizar.cmd` 1.0.1 terminó con «Actualización terminada. Se han
   conservado los datos y las cuentas de pruebas»
   ([captura](../capturas/windows-101-actualizar-ok.png)).
   - Después: los mismos 31 expedientes con la misma huella, el 04-02-26 sigue
     aceptado y hay 74 migraciones.
   - La app es la 1.0.1, `/salud` responde 200, la copia privada es la 1.0.1 y
     las contraseñas se conservaron: se entró con las mismas.
4. Se cerró Edge y se abrió desde el acceso del escritorio: la sesión estaba
   activa y apareció la bandeja de Coordinación.
5. Comprobaciones de la 1.0.1:
   - **Consulta (lucia.navarro)**: «Descargar la Memoria en Word» descarga
     `memoria-cifras-2026.docx` (45 033 bytes, ZIP válido con `word/document.xml`).
     Este era el fallo anterior y ya no ocurre
     ([captura](../capturas/windows-101-consulta-word.png)).
   - **Consulta**: aparece «Oculto» en las bases, los porcentajes y «sin dato»,
     con el aviso de cifras protegidas
     ([captura](../capturas/windows-101-consulta-oculto.png)).
   - **Coordinación**: al pulsar una cifra de la Memoria se abre «Expedientes
     que forman la cifra» ([captura](../capturas/windows-101-desglose-memoria.png)).
   - **Profesional (marta.cabrera)**: en el expediente 02-02-26 aparece
     «Derivar a otro servicio · Anexo 6» y su pantalla con el destino escrito
     ([captura](../capturas/windows-101-anexo6-profesional.png)).
   - **Aviso de sesión**: los textos `seg1.sesion_aviso` y `seg1.continuar` llegan
     en `/api/idiomas/es/textos`. No se esperó a que caducara la sesión para verlo
     en pantalla.

## 2. Instalación limpia 1.0.1: correcta

1. `desinstalar.cmd` (DESINSTALAR) e `instalar.cmd` 1.0.1. Quedan los 31
   expedientes de la demo y el 04-02-26 ya no existe.
2. Coordinación subió un **Anexo 3 en ODT** inventado. Se eligió el municipio,
   se creó la derivación, se completaron fecha, puesto y sexo de los dos menores
   y se aceptó: expediente **01-01-26**.
3. Se descargó el **Acta de Aceptación** en Word: 129 267 bytes, ZIP válido, con
   el número 01-01-26 y los dos menores
   ([captura](../capturas/windows-101-limpia-acta.png)).
4. `parar.cmd`: «Aplicación detenida» y `/salud` deja de responder. Al volver a
   abrir desde el acceso, la app arranca, 32 expedientes, y el 01-01-26 sigue
   aceptado.

**Observación:** la primera subida, unos 4 minutos después de instalar, avisó
«No se ha podido revisar el fichero con el antivirus. La derivación se ha
registrado y necesita revisión» (estado `no_revisado`), aunque ClamAV ya
figuraba como sano. Al repetir más tarde con el mismo ODT respondió «fichero
revisado, sin amenazas detectadas»; esa revisión se descartó. Parece que ClamAV
todavía cargaba sus firmas. No se ha cambiado nada.

## Cierre

VM apagada con `virsh shutdown`; la otra máquina virtual no se tocó. Se bajaron el
servidor HTTP temporal, el visor y la pantalla virtual. Las contraseñas de la
demo se pasaron al formulario mediante el portapapeles de la VM, que se vació
después. No se guardaron en ningún fichero ni en las capturas, y tampoco la
contraseña de la VM.
