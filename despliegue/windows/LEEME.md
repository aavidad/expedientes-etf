# Probar Expedientes ETF en Windows

Usa sólo datos inventados. Nunca introduzcas datos reales de personas o expedientes.
No necesitas Docker ni herramientas de programación.

La guía completa está en
https://github.com/aavidad/expedientes-etf/blob/main/docs/windows/PROBAR-EN-CASA.md
La página de descargas es https://github.com/aavidad/expedientes-etf/releases.

## Novedades de la versión 1.0.4

- Instalador: la descarga principal es ahora el asistente
  `Expedientes-ETF-1.0.4-instalador.exe`. Instala la aplicación sólo para tu
  cuenta, sin permiso de administrador, y aparece en Configuración >
  Aplicaciones para desinstalarla desde ahí. El ZIP con instalar.cmd sigue
  disponible como alternativa.
- Accesos: el acceso del escritorio y de Inicio abre un pequeño programa,
  `Expedientes ETF.exe`, en lugar de un script, para que los antivirus que
  bloquean scripts lanzados desde accesos directos no corten la aplicación. Al
  actualizar, los accesos antiguos se cambian solos.

La versión 1.0.4 conserva lo de las anteriores: la instalación sin preparar
Windows (WSL2, Podman y Compose), la espera a Podman tras reiniciar, la parada
que apaga su máquina, la desinstalación de lo instalado, las plantillas Word
inglesas, la pantalla Ayuda con el manual de cada perfil y su lectura en voz
alta, la espera al antivirus antes de abrir la aplicación, el Anexo 6, el
desglose de la Memoria, las cifras pequeñas ocultas en Consulta, el cierre de
sesión por inactividad, el control de la carga histórica, la revisión tras
restaurar una copia y la actualización de textos, plantillas y temas.

Necesitas Windows 10 de 64 bits (versión 2004 o posterior) o Windows 11,
8 GB de memoria y 15 GB libres para las imágenes y los datos. Recomendamos
16 GB de memoria. No necesitas instalar nada antes: instalar.cmd instala lo que
falte. La primera instalación necesita Internet, porque WSL2 se descarga de
Microsoft y Podman descarga el sistema de su máquina (unos 250 MB). El resto va
dentro del paquete.

## 1. Instalar con el asistente

La forma más sencilla es el instalador **Expedientes-ETF-VERSIÓN-instalador.exe**.
Instala la aplicación sólo para tu cuenta y no pide permiso de administrador.

1. Descarga el `.exe` y su archivo `.exe.sha256` de la misma entrega. Para
   comprobarlo, abre PowerShell en la carpeta de descargas y ejecuta
   `Get-FileHash .\Expedientes-ETF-VERSIÓN-instalador.exe -Algorithm SHA256`.
   El resultado debe coincidir con el contenido del archivo `.sha256`.
2. Haz doble clic en el `.exe`. El instalador no está firmado, así que Windows
   puede mostrar **Windows protegió su PC** (SmartScreen). Si el archivo
   procede de la página de descargas y la suma coincide, pulsa **Más información** y
   después **Ejecutar de todas formas**.
3. Sigue el asistente, en castellano o en inglés según el idioma de Windows.
   Copia la aplicación en `%LOCALAPPDATA%\Expedientes-ETF\pruebas\paquete`.
4. Al terminar, se abre una ventana negra que hace lo mismo que instalar.cmd
   (pasos 4 a 6 del apartado siguiente): instala lo que falte, prepara los datos
   y abre la aplicación. No la cierres hasta que termine.

Si ya tenías la aplicación instalada, el mismo `.exe` de una versión nueva la
actualiza: copia los archivos nuevos y ejecuta actualizar.cmd, que conserva tus
cuentas y datos. Para quitarla, ve a **Configuración > Aplicaciones**, busca
**Expedientes ETF (pruebas)** y pulsa **Desinstalar**: se abre la ventana de
desinstalar.cmd, que te pide confirmación, y después se borran los archivos
copiados y la entrada de la lista de aplicaciones.

## 1 bis. Alternativa: descomprimir el ZIP y ejecutar instalar.cmd

Si tu equipo no permite ejecutar el instalador, usa el ZIP de la misma entrega.

1. Descarga el ZIP y su archivo `.zip.sha256` de la misma entrega. Para
   comprobarlo, abre PowerShell en la carpeta descargada y ejecuta
   `Get-FileHash .\expedientes-etf-windows-VERSION.zip -Algorithm SHA256`,
   sustituyendo VERSION por la versión del nombre. El resultado debe coincidir
   con el contenido del archivo `.sha256`.
2. Pulsa con el botón derecho sobre el ZIP y elige **Extraer todo**.
   Guarda la carpeta en Documentos, fuera de OneDrive y de carpetas compartidas.
   No ejecutes los archivos desde dentro del ZIP.
3. Abre la carpeta **Expedientes-ETF** y haz doble clic en **instalar.cmd**.
   No hace falta abrirlo como administrador. Puede aparecer SmartScreen porque el
   paquete no está firmado. Si has descargado el ZIP de la página de descargas
   y su suma coincide, pulsa **Más información → Ejecutar de todas formas**.
   No desactives la protección de Windows.
4. Espera y lee lo que indica la ventana. Si falta algo, lo instala en este orden:
   - **WSL2**, el componente de Windows que ejecuta los contenedores Linux.
     Windows pide permiso de administrador: pulsa **Sí**. Si después pide
     reiniciar, guarda tu trabajo y reinicia. Al volver a entrar con tu usuario,
     la instalación sigue sola en una ventana nueva.
   - **Podman**, el programa que ejecuta los contenedores. Se instala sólo para
     tu usuario, sin permiso de administrador. Es gratuito. Docker Desktop exige
     una suscripción de pago en organizaciones grandes y administraciones
     públicas, según sus [condiciones](https://docs.docker.com/subscription-billing/desktop-license/).
   - **La máquina de Podman**, con 4 GB de memoria y 2 procesadores, en modo
     con permisos de administrador (rootful). Se enciende sola.
   - **Compose**, que arranca juntos todos los componentes de la aplicación.
5. Después la ventana comprueba las sumas de los archivos, carga las imágenes y
   prepara los datos. Todo es repetible: si se corta, vuelve a ejecutar
   instalar.cmd y seguirá donde lo dejó.
6. Se abrirán el navegador y un archivo con los usuarios de pruebas. Para
   Coordinación utiliza **ines.moreno** y copia su contraseña del archivo.
   Cada PC genera contraseñas distintas. No compartas ese archivo.

Si Windows indica que la virtualización está deshabilitada, pide a Informática
que la active en la configuración del PC. No cambies esa configuración por tu
cuenta. Si tu cuenta no puede conceder el permiso de administrador, Informática
puede preparar el PC con el plan B de más abajo.

El instalador crea **Expedientes ETF** en el escritorio y en Inicio. Ese acceso
abre el programa **Expedientes ETF.exe** de la copia del paquete, que a su vez
abre una ventana propia en Microsoft Edge. Los accesos usan este lanzador .exe
para no activar las reglas de los antivirus que bloquean scripts lanzados desde
accesos directos; al actualizar, los accesos antiguos se cambian solos. El navegador es Edge; si no está instalado, utiliza Chrome.
Puedes mover o borrar la carpeta extraída: el acceso conserva una copia privada
del paquete. La dirección es **http://localhost:18083**. Sólo se publica en este PC.
Las contraseñas se guardan en
`%LOCALAPPDATA%\Expedientes-ETF\pruebas`, con acceso limitado a tu cuenta.
No se guardan en el ZIP ni en la carpeta extraída.

## 2. Del Anexo al Acta en cinco pasos

1. Entra como **ines.moreno** con la contraseña del archivo de usuarios. En el
   menú **Derivaciones**, pulsa **Subir Anexo 3 o 4**.
2. Elige **Anexo-3-pruebas.docx**, incluido junto a instalar.cmd, y pulsa
   **Subir y leer anexo**.
3. Revisa los datos, completa los que falten y confirma la derivación.
4. Ábrela. Completa fecha, equipo y los datos pendientes de los seis menores.
   Pulsa **Aceptar derivación** y después **Aceptar y asignar número**.
5. Descarga el **Acta de aceptación** y comprueba número, familia, seis menores
   y firmante. Se abre con Word o LibreOffice. La aplicación no conserva el
   documento descargado.

Prueba también **Anexo-3-pruebas.odt** desde Nueva derivación. En **Expedientes
→ Añadir un expediente histórico**, usa **Acta-aceptacion-pruebas.docx** y revisa
los datos antes de confirmar. Si el número ya existe, utiliza otro número inventado.

## Parar, volver a abrir y actualizar

- **parar.cmd** detiene esta demostración, cierra su ventana y conserva los datos.
  También apaga la máquina de Podman si no tiene contenedores de otras
  aplicaciones. Si has borrado la carpeta extraída, está en
  `%LOCALAPPDATA%\Expedientes-ETF\pruebas\paquete\parar.cmd`. Para volver, abre **Expedientes ETF** desde el escritorio o Inicio: el acceso
  muestra la preparación, enciende la máquina de Podman si hace falta y arranca
  esta demostración si está parada. También puedes ejecutar instalar.cmd.
- Para actualizar, descarga y extrae el paquete nuevo en otra carpeta. Haz
  doble clic en **actualizar.cmd del paquete nuevo**. Conserva tus cuentas y
  datos; prepara los roles técnicos y aplica las migraciones antes de arrancar.
  No abras al mismo tiempo dos instaladores. Una versión anterior no sirve
  para deshacer migraciones ni devuelve los textos y plantillas anteriores. Si falla, conserva ambas carpetas y pide ayuda;
  no ejecutes reiniciar-datos para reparar una actualización.
- **Antes de apagar o reiniciar Windows, cierra la aplicación con parar.cmd.**
  En las pruebas, Windows 10 se quedó más de media hora en «Reiniciando» con
  la máquina de Podman encendida. Si te pasa, espera unos minutos; si no
  avanza, mantén pulsado el botón de encendido hasta que el PC se apague.
  No hay una forma segura de pararla sola al apagar sin permisos de
  administrador.
- **reiniciar-datos.cmd** borra toda esta demostración y la vuelve a crear.
  Escribe **BORRAR** sólo si quieres perder sus cambios. Las contraseñas también
  cambian. Pulsar Intro sin escribir nada cancela la operación.

- Si instalaste con el `.exe`, desinstala desde **Configuración > Aplicaciones**
  (ver el apartado 1). Hace lo mismo que desinstalar.cmd.
- **desinstalar.cmd** elimina los datos de esta demostración y sus accesos.
  Escribe **DESINSTALAR** para confirmarlo. Si la instalación puso Podman o
  su máquina, pregunta si también quieres quitarlos (escribe **QUITAR**).
  Conserva WSL, lo que ya había en el PC y los archivos que hayas añadido por
  tu cuenta. Para usar la aplicación después, vuelve a ejecutar instalar.cmd.

## Qué instala y cómo quitarlo

instalar.cmd anota en `%LOCALAPPDATA%\Expedientes-ETF\pruebas\instalado.json`
lo que instala, y desinstalar.cmd sólo ofrece quitar lo anotado ahí.

| Qué | Dónde | Cómo se quita |
| --- | --- | --- |
| WSL2, si faltaba | Windows, con tu permiso de administrador | No se quita solo. Ver más abajo |
| Podman 6.1.3, si faltaba | Sólo para tu usuario; su instalador añade su carpeta a tu PATH y la retira al desinstalarse | desinstalar.cmd, si escribes QUITAR |
| La máquina de Podman, si faltaba | La distribución de WSL de Podman | desinstalar.cmd, si escribes QUITAR |
| Compose, si Podman no tenía | `pruebas\herramientas` | desinstalar.cmd |
| Datos, contraseñas, registro y copia del paquete | `%LOCALAPPDATA%\Expedientes-ETF\pruebas` | desinstalar.cmd |
| Accesos **Expedientes ETF** | Escritorio e Inicio | desinstalar.cmd |
| Reanudación tras reiniciar | Entrada `ExpedientesETF` de RunOnce de tu usuario | Se borra al usarse o con desinstalar.cmd |
| Entrada **Expedientes ETF (pruebas)** de la lista de aplicaciones, si usaste el `.exe` | `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Expedientes-ETF` | Desinstalar desde **Configuración > Aplicaciones** |

No cambia otros ajustes de Windows ni crea tareas programadas o servicios.
Si la máquina de Podman tiene contenedores de otras aplicaciones, desinstalar.cmd
no quita ni la máquina ni Podman. Puedes hacerlo después: abre PowerShell y
ejecuta `podman machine rm`, y quita Podman desde **Configuración >
Aplicaciones**.

WSL2 se conserva porque Windows y otras aplicaciones pueden usarlo. Si nadie
más lo usa en este PC, Informática puede quitarlo desde **Configuración >
Aplicaciones** (Subsistema de Windows para Linux) y desactivar las
características **Subsistema de Windows para Linux** y **Plataforma de máquina
virtual** en **Activar o desactivar las características de Windows**. Después hay que reiniciar Windows.

## Plan B: preparar Windows a mano

Sirve si la instalación automática no puede terminar, por ejemplo porque tu
cuenta no tiene permiso de administrador. Hazlo una sola vez y después
ejecuta instalar.cmd: detecta lo instalado y no repite nada.

### WSL2

1. Guarda tu trabajo. Abre Inicio, escribe **PowerShell**, pulsa con el botón
   derecho y elige **Ejecutar como administrador**.
2. Copia esta orden, pulsa Intro y espera a que termine:

   ```powershell
   wsl --install --no-distribution
   ```

3. Reinicia Windows cuando lo pida. WSL2 permite ejecutar los contenedores Linux
   de la aplicación. Si la orden no existe en tu versión de Windows 10, sigue
   la [instalación manual de Microsoft](https://learn.microsoft.com/es-es/windows/wsl/install-manual).
4. Si Windows indica que la virtualización está deshabilitada, pide a
   Informática que la active en la configuración del PC. No cambies esa
   configuración por tu cuenta.

### Podman Desktop

1. Descarga **Podman Desktop para Windows** de su
   [página oficial](https://podman-desktop.io/downloads) e instálalo.
   Es gratuito.
2. Abre Podman Desktop y sigue **Get started**. Instala Podman y la herramienta
   **Compose** cuando lo ofrezca. Compose permite abrir todos los componentes
   de la aplicación juntos.
3. Crea una máquina Podman con el proveedor **WSL**, al menos 4 GB de memoria
   y 2 procesadores. Enciéndela desde Podman Desktop.
4. Activa el modo **Rootful** en los ajustes de esa máquina. Si no aparece,
   abre una PowerShell normal y ejecuta, sin otras aplicaciones Podman abiertas:

   ```powershell
   podman machine stop
   podman machine set --rootful
   podman machine start
   ```

5. Cierra la ventana PowerShell. La guía oficial de
   [Windows](https://podman-desktop.io/docs/installation/windows-install) y la
   de [Compose](https://podman-desktop.io/docs/compose/setting-up-compose)
   explican estos pasos con sus pantallas actuales.

## Alternativa: Rancher Desktop

Si Informática prefiere Rancher Desktop, usa su
[instalador oficial](https://docs.rancherdesktop.io/getting-started/installation/).
También necesita WSL2. En sus ajustes elige **Moby (dockerd)** y desactiva
Kubernetes. Abre PowerShell en la carpeta del paquete y ejecuta
`powershell -NoProfile -ExecutionPolicy Bypass -File .\instalar.ps1 -Motor Rancher`.
Para las otras operaciones utiliza el mismo parámetro con su archivo `.ps1`.
No alternes Podman y Rancher: cada motor conserva sus propios datos.

## Si algo falla

Lee el mensaje de la ventana: indica el requisito que falta. Después pulsa una
tecla para cerrarla y repite instalar.cmd cuando lo hayas resuelto. Si has
instalado Podman a mano, comprueba que su máquina está encendida y que Compose
está instalado. Si el puerto 18083 está ocupado, cierra la otra demostración;
no detengas otras aplicaciones sin saber cuáles son. Informática puede elegir
otro puerto la primera vez con `instalar.cmd -Puerto 18381`; se conserva en
las siguientes aperturas y actualizaciones.

**Podman enciende la máquina pero Compose no conecta.** A veces, después de
reiniciar Windows, Podman enciende su máquina sin abrir la conexión
que usa Compose (la tubería `\\.\pipe\podman-machine-default`). Antes se veía
«No se ha podido completar la operación compose (código 1)». Ahora el acceso,
instalar.cmd y actualizar.cmd lo detectan solos: reinician una vez la máquina
de Podman y, si no basta, abren la conexión con `win-sshproxy.exe`, el programa
que Podman instala para eso. La ventana avisa mientras lo hace. Si aun así
aparece el mensaje de que Podman no ha abierto la conexión, abre PowerShell y
ejecuta `podman machine stop` y después `podman machine start`. Luego vuelve a
abrir la aplicación. Si sigue igual, reinicia Windows o pide ayuda con ese
mensaje.

**Podman Desktop se abre solo al entrar en Windows.** Si Podman Desktop se
abre al iniciar sesión, enciende la máquina de Podman por su cuenta. El acceso
**Expedientes ETF** espera hasta tres minutos a que termine, sin pararla ni
encenderla encima, y la ventana avisa mientras espera. Si la máquina no
termina de encenderse, aparece un aviso por encima de las demás ventanas.

**La ventana de arranque se cierra.** Cualquier error al abrir se muestra en un
aviso por encima de las demás ventanas, y la ventana negra no se cierra hasta
que lo aceptas. Lo que hace el acceso queda en
`%LOCALAPPDATA%\Expedientes-ETF\pruebas\registro-abrir.log`, sin
contraseñas. Puedes enviar ese archivo a Informática.

**El antivirus tarda en estar listo.** Al instalar, actualizar o volver a abrir,
la ventana espera hasta cinco minutos a que el antivirus analice de verdad un
texto de prueba. Así la primera subida ya se revisa. Si pasan los cinco minutos,
revisa el contenedor `clamav` de **etf-windows** (en PowerShell,
`podman ps --all`) y vuelve a abrir la aplicación.

Para pedir ayuda, indica la versión de VERSION.txt y el mensaje de error.
No envíes el archivo de usuarios, entorno.env ni el resto de la carpeta
privada; sólo registro-abrir.log. Los contenedores se llaman **etf-windows-…**; si usas Podman Desktop,
aparecen en el grupo **etf-windows**.

## Para quien prepara la entrega

Desde Linux con las herramientas de la demo y Docker Engine local:
`make paquete-windows` (o `VERSION=2026.10.14 make paquete-windows`).
El resultado está en `bin/paquete-windows/`, con versión y suma SHA-256.
El ZIP incluye imágenes Linux para PC de 64 bits, archivos operativos y un anexo
inventado. También incluye los instaladores oficiales de Podman 6.1.3 (MSI para
Windows de 64 bits) y de Compose 5.5.1 en `requisitos/`. Sus direcciones y sumas
SHA-256 están fijadas en `paquete.py` (`REQUISITOS`): se descargan al construir,
se guardan en `bin/descargas/` y el paquete se detiene si una suma no coincide.
Para cambiar de versión, actualiza la dirección y la suma con las publicadas en
GitHub y elige una versión estable de hace al menos dos semanas.
No necesitas descargar el repositorio. El paquete no incorpora credenciales
ni datos guardados en otras demostraciones.
SHA-256 detecta cambios; comprueba además la procedencia del ZIP.

La misma orden genera también `Expedientes-ETF-VERSIÓN-instalador.exe` y su
suma con NSIS (`makensis` 3.10 o posterior; en Debian y Ubuntu, el paquete
`nsis`). Usa la misma carpeta que el ZIP, comprimida con LZMA sólida: con la
1.0.3 pasaba de 543 MB en ZIP a 412 MB y tarda unos cinco minutos. Para
rehacer sólo el `.exe` a partir de un ZIP ya construido:
`VERSION=1.0.4 make instalador-windows`. El guion del asistente es
`instalador.nsi` y el del lanzador de los accesos, `lanzador.nsi`; sus textos
están en `idiomas/es/textos/windows.json` y `idiomas/en/textos/windows.json`
(claves `windows.instalador_*`).

El `.exe` y el lanzador no van firmados, por eso Windows muestra el aviso de
SmartScreen. Con el certificado de firma de código de la entidad se firman
con `osslsigncode` al construir: define `ETF_FIRMA_CERTIFICADO` y
`ETF_FIRMA_CLAVE` con las rutas al certificado y a la clave, fuera del
repositorio. Opcionalmente, `ETF_FIRMA_CONTRASENA_FICHERO` (fichero con la
contraseña de la clave) y `ETF_FIRMA_SELLO_URL` (servidor de sello de tiempo).
Nunca guardes el certificado, la clave ni su contraseña en el repositorio.
