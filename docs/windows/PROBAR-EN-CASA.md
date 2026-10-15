# Probar Expedientes ETF en casa

Esta guía te ayuda a instalar una demostración en tu PC. Usa sólo datos inventados.
Nunca introduzcas datos reales de personas o expedientes, aunque trabajes desde casa.
No necesitas Docker ni herramientas de programación.

## Novedades de la versión 1.0.2

- Ayuda: el menú tiene un apartado nuevo con el manual de tu perfil, su índice,
  las capturas y un PDF para imprimir.
- Escuchar: puedes oír en castellano la ayuda de cada pantalla y los manuales.
  Las grabaciones vienen dentro del paquete y la aplicación no se conecta a
  ningún servicio para leerlas.
- Arranque con Podman: a veces, después de reiniciar Windows, Podman enciende su
  máquina pero no abre la conexión. El acceso, instalar.cmd y actualizar.cmd lo
  detectan y lo arreglan solos.
- Antivirus: la ventana espera a que el antivirus esté listo antes de abrir la
  aplicación. Así el primer anexo que subas ya se revisa.

La versión 1.0.2 incluye también lo que trajo la 1.0.1:

- Anexo 6: desde la ficha o desde Salidas y cierre puedes preparar la derivación
  a otro servicio, descargar el Word y anotar después la fecha real de envío.
- Memoria: al pulsar una cifra ves qué expedientes la forman, con su número,
  equipo y aportación, sin nombres. Lo usan Coordinación y cada equipo con lo suyo.
- Consulta: las cifras por debajo de 3 casos aparecen como «Oculto», en pantalla
  y en Word, para que nadie pueda reconocer a una familia.
- Sesión: se cierra tras 30 minutos sin actividad. Dos minutos antes aparece un
  aviso para seguir trabajando. Lo que ya guardaste se conserva.
- Carga histórica: Administración fija hasta qué día se pueden añadir expedientes
  antiguos. La aplicación avisa de posibles duplicados y Coordinación puede anular
  un lote completo dentro del plazo.
- Restaurar una copia: después de recuperar una copia, Administración revisa
  cuentas y permisos antes de que nadie vuelva a entrar.
- Actualizar: actualizar.cmd cambia los textos, plantillas y temas que trae el
  paquete y conserva los que hayas publicado o cambiado tú.

## 1. Comprueba tu PC

Necesitas Windows 10 **22H2** o Windows 11 de **64 bits**, una cuenta con permiso
para instalar programas y **8 GB de RAM instalada** como mínimo. Con 16 GB tendrás
más margen para usar otros programas. Para comprobar tu versión, abre Inicio,
escribe `winver` y pulsa Intro. Puedes ver la RAM en Configuración → Sistema → Acerca de.

El instalador exige **15 GiB libres en la unidad de tu perfil**, normalmente C:.
El ZIP ocupa unos **426 MiB** y la carpeta extraída, **741 MiB**, medidos en esta
entrega. Reserva **17 GiB libres antes de descargar** para mantener los 15 GiB
que comprueba el instalador después de extraer. La copia privada del paquete
ocupa otros 741 MiB dentro de esa reserva, además de las imágenes y los datos.
Las medidas exactas se anotan en la [evidencia de la entrega](../sistemas/WIN1.md#paquete-público).
Comprueba el espacio desde Explorador → Este equipo → Disco local (C:).

Necesitas Internet para descargar el paquete y preparar WSL y Podman Desktop,
incluida su máquina Linux y Compose. Después puedes usar esta demostración sin
conexión: las imágenes de la aplicación vienen en el ZIP. La preparación inicial
puede ocupar entre 20 y 60 minutos, según tu conexión y tu PC; reserva otros
5 a 15 minutos para instalar la aplicación. Son orientaciones, no tiempos garantizados.

Antes de instalar:

1. Pulsa Ctrl + Mayús + Esc para abrir el **Administrador de tareas**.
2. Si aparece una ventana pequeña, pulsa **Más detalles**.
3. Entra en **Rendimiento → CPU**. Busca **Virtualización: Habilitado**.
4. Si dice «Deshabilitado», tendrás que activarla en BIOS/UEFI. Guarda tu trabajo.
   En la web de soporte del fabricante de tu PC busca su modelo y «activar
   virtualización». Los menús varían: Intel puede llamarlo **Intel Virtualization
   Technology / VT-x**; AMD, **SVM / AMD-V**. Dell, HP, Lenovo, ASUS y Acer tienen
   instrucciones propias de acceso y menús para cada modelo. Sigue las de tu
   fabricante; no cambies otros ajustes. Si no encuentras la opción o te pide una
   clave que desconoces, pide ayuda a quien mantenga tu PC. Reinicia y vuelve a
   comprobar el Administrador de tareas.

## 2. Instala WSL2

WSL permite que Windows ejecute los componentes Linux de la demostración.
Sólo tienes que prepararlo una vez.

1. Guarda tu trabajo. Abre Inicio, escribe **PowerShell**, pulsa con el botón
   derecho y elige **Ejecutar como administrador**. Acepta el aviso de Windows.
2. Copia esta orden y pulsa Intro:

   ```powershell
   wsl --install
   ```

3. Espera y **reinicia Windows**. Si se abre el asistente de Ubuntu, completa la
   creación de su usuario local; esa cuenta es distinta de las de Expedientes ETF.
   No necesitas abrir Ubuntu para usar esta aplicación.
4. Abre PowerShell y comprueba:

   ```powershell
   wsl --status
   ```

   Debe indicar que la versión predeterminada es **2**. Si indica 1, ejecuta
   `wsl --set-default-version 2`. Si pide actualizar el núcleo, abre PowerShell
   como administrador, ejecuta `wsl --update` y reinicia.

En el ensayo previo se instaló WSL sin Ubuntu con
`wsl --install --no-distribution`; también sirve. La versión comprobada es
**WSL 3.0.3**, con núcleo **6.18.54.1**. Puedes verla con `wsl --version`.
Si aparece un error, consulta la [instalación oficial de WSL](https://learn.microsoft.com/es-es/windows/wsl/install).

## 3. Instala y prepara Podman Desktop

Podman abre los componentes de la aplicación. Mantén su máquina encendida
mientras uses Expedientes ETF.

1. Abre [podman-desktop.io](https://podman-desktop.io/downloads), descarga el
   instalador para **Windows x64** y ejecútalo. La versión ensayada es
   **Podman Desktop 1.29.3**, con **Podman 6.1.4** y **Compose 5.1.4**.
   Para repetir el entorno, busca esa versión en las descargas oficiales.
2. Abre Podman Desktop. En **Get started / Start Onboarding**, acepta instalar
   Podman y elige **WSL / WSLv2** como proveedor.
3. Cuando ofrezca crear una máquina, pulsa **Create**. Conserva el nombre
   propuesto. En la prueba se usaron **2 procesadores y 4 GiB** para la máquina.
   Espera a que figure como **Running**. El PC necesita los 8 GB indicados antes.
4. En los ajustes de esa máquina activa **Rootful**. Este paquete necesita ese
   modo. Si no aparece la opción, cierra otras aplicaciones que usen Podman,
   abre PowerShell normal y ejecuta estas órdenes, una cada vez:

   ```powershell
   podman machine stop
   podman machine set --rootful
   podman machine start
   ```

5. Instala **Compose** desde el asistente o desde **Settings → Resources →
   Compose**. Comprueba en PowerShell:

   ```powershell
   podman machine list
   podman compose version
   ```

   La máquina debe estar encendida y Compose debe mostrar su versión. Cierra
   PowerShell. Si falla la conexión después de instalar Podman, cierra y vuelve
   a abrir Podman Desktop para que recoja los programas recién instalados.

Puedes consultar los [pasos oficiales para Windows](https://podman-desktop.io/docs/installation/windows-install)
y la [preparación de Compose](https://podman-desktop.io/docs/compose/setting-up-compose).

## 4. Descarga y comprueba el paquete

1. Abre las [Releases de Expedientes ETF](https://github.com/aavidad/expedientes-etf/releases).
   En la entrega 1.0.2 descarga **expedientes-etf-windows-1.0.2.zip** y
   **expedientes-etf-windows-1.0.2.zip.sha256**. Guarda ambos en la misma carpeta.
2. En el Explorador, abre esa carpeta. Pulsa la barra de dirección, escribe
   `powershell` y pulsa Intro. Copia:

   ```powershell
   Get-FileHash .\expedientes-etf-windows-1.0.2.zip -Algorithm SHA256
   Get-Content .\expedientes-etf-windows-1.0.2.zip.sha256
   ```

3. Compara los 64 caracteres de **Hash** con los del archivo `.sha256`.
   Deben coincidir; las mayúsculas no importan. Si son distintos, borra esa
   descarga y vuelve a obtener ambos archivos de la misma Release. No instales
   un ZIP cuya suma no coincide. La suma detecta cambios; descarga siempre del
   repositorio público indicado.
4. Cierra PowerShell. Pulsa el ZIP con el botón derecho y elige **Extraer todo**.
   Usa una carpeta local corta, por ejemplo `C:\ETF-Pruebas`, fuera de OneDrive,
   unidades de red y carpetas compartidas. Si no puedes crearla, usa una carpeta
   en Documentos que no esté sincronizada. Espera a que termine la extracción.

[Comprobación real de la suma en Windows](capturas/comprobar-sha256.png).

[Asistente real «Extraer todo» de Windows](capturas/extraer-todo.png).

## 5. Instala la demostración

1. Abre la carpeta extraída **Expedientes-ETF** y haz doble clic en
   **instalar.cmd** desde el Explorador. No lo abras dentro del ZIP ni como administrador.
2. Puede aparecer SmartScreen porque este paquete **no está firmado** y aún no
   tiene reputación de descarga. Si has comprobado su procedencia y su suma,
   pulsa **Más información → Ejecutar de todas formas**. Si tu PC no ofrece esa
   opción, consulta a quien lo administre; no desactives la protección de Windows.
   También puede salir «Editor desconocido» con un botón **Ejecutar**, como en
   [el aviso real del ensayo](capturas/aviso-seguridad.png). Comprueba antes
   procedencia y suma, y pulsa ese botón para continuar.
3. Deja abierta la consola. Primero comprueba requisitos y sumas; después verás
   «Cargando las imágenes incluidas» y «Preparando la base de datos y arrancando
   la aplicación». La primera carga tarda varios minutos. El antivirus incluido
   también necesita arrancar; no cierres la ventana mientras trabaja.
4. Al terminar aparecen el acceso **Expedientes ETF** en el escritorio y en
   Inicio, la aplicación en una ventana propia de **Microsoft Edge** y un Bloc
   de notas con las cuentas y sus contraseñas. Si no tienes Edge, usa Chrome.
5. Conserva ese archivo de contraseñas en privado. Cada instalación genera
   contraseñas distintas. Puedes volver a abrir **USUARIOS.txt** desde
   `%LOCALAPPDATA%\Expedientes-ETF\pruebas`: pega esa ruta en el Explorador.
   No envíes su contenido ni el archivo `entorno.env` al pedir ayuda.

[Acceso real creado en el escritorio](capturas/acceso-directo.png).

El acceso conserva una copia privada del paquete. Puedes mover la carpeta
extraída; guarda los ejemplos para seguir practicando. La dirección local es
`http://localhost:18083`; sólo responde en tu PC.

Estas siete cuentas son inventadas:

| Usuario | Perfil y práctica disponible |
| --- | --- |
| `ines.moreno` | Coordinación: subir anexos, aceptar, asignar equipo y descargar actas; consultar todos los equipos. |
| `javier.medina` | Coordinación: mismo acceso que la cuenta anterior. |
| `marta.cabrera` | Profesional del ETF 2: trabajar en la ficha, documentos, actividad y Memoria de su equipo. |
| `pablo.ruiz` | Profesional del ETF 4: mismo trabajo dentro de su equipo. |
| `lucia.navarro` | Consulta: ver cifras, sin nombres ni fichas personales. |
| `elena.salas` | Administración funcional: usuarios, catálogos, plantillas, idiomas y temas; no ve expedientes. |
| `sergio.campos` | Sin perfil asignado: sirve para comprobar el aviso de acceso pendiente. |

## 6. Tu primer recorrido: del Anexo al Acta

1. Entra como **ines.moreno** con su contraseña del Bloc de notas. En el menú
   **Derivaciones**, pulsa **Subir Anexo 3 o 4**.
2. Elige **Anexo-3-pruebas.docx**, incluido junto a instalar.cmd, y pulsa
   **Subir y leer anexo**.
3. Revisa los datos propuestos. Completa lo que pida la pantalla y pulsa **Confirmar y crear derivación**.
4. Abre esa derivación y completa fecha, equipo y asignación;
   si faltan datos de los seis menores, complétalos. Pulsa **Aceptar derivación**
   y después **Aceptar y asignar número** en la confirmación. Anota el número.
5. Descarga el **Acta de aceptación**. Guárdala y ábrela con Word o LibreOffice
   si dispones de uno. Comprueba número, familia, seis menores y firma prevista.
   Para la aplicación no necesitas instalar un procesador de textos.

También se incluyen **Anexo-3-pruebas.odt** y **Acta-aceptacion-pruebas.docx**.
Puedes probar el ODT desde Nueva derivación. Para el Acta entra en **Expedientes →
Añadir un expediente histórico**, en el apartado **Leer las actas (opcional)**: sube el archivo, revisa las fechas y completa
los campos que falten antes de confirmar. Usa otro caso o cambia su número de
prueba si ya existe. La carga histórica conserva los números que confirmas.

## 7. Parar, volver y borrar las pruebas

- Para terminar por hoy, abre **parar.cmd**. Conserva tus datos y contraseñas.
  Después puedes cerrar Podman Desktop.
- Para volver, abre Podman Desktop y pulsa **Expedientes ETF** en el escritorio.
  El acceso comprueba requisitos, arranca la demostración si está parada y abre
  su ventana de Edge. La preparación puede tardar unos minutos.
- Para empezar de cero, abre **reiniciar-datos.cmd**. Escribe **BORRAR** y pulsa
  Intro sólo si quieres eliminar todos tus cambios. Se recrean las cuentas con
  contraseñas nuevas. Pulsa Intro sin escribir nada para cancelar.
- Para quitar la aplicación, abre **desinstalar.cmd** y escribe **DESINSTALAR**.
  Elimina sus datos, contraseñas y accesos; conserva archivos que hayas añadido.
  Para reinstalarla tendrás que ejecutar instalar.cmd otra vez.
- Si también quieres quitar Podman Desktop, desinstálalo desde Configuración →
  Aplicaciones. Antes, elimina su máquina desde Podman Desktop sólo si no la
  usa ninguna otra aplicación: perderás todo lo que contenga.
- Quitar WSL es opcional. Si lo usas para otros programas, consérvalo. Para quitar
  una distribución usa Configuración → Aplicaciones. `wsl --unregister NOMBRE`
  borra definitivamente todos los datos de esa distribución; comprueba antes
  los nombres con `wsl --list --verbose`. Sólo cuando ya no lo uses, desactiva
  «Subsistema de Windows para Linux» en «Activar o desactivar las características
  de Windows» y reinicia. No desactives «Plataforma de máquina virtual» si la
  necesita otro programa.

## Si algo falla

| Lo que ves | Qué puedes hacer |
| --- | --- |
| Virtualización deshabilitada o error al crear la máquina | Revisa Administrador de tareas → Rendimiento → CPU y el paso 1. Actívala siguiendo el soporte del fabricante. |
| «WSL2 no está listo» | Ejecuta `wsl --update` como administrador, reinicia y comprueba `wsl --status`. Crea después la máquina Podman. |
| «El PC necesita al menos 8 GB» | Comprueba la RAM instalada. Cerrar programas libera memoria para trabajar, pero no cambia ese mínimo: si tienes menos de 8 GB, usa otro PC. |
| «Necesitas al menos 15 GB» | Libera espacio en C: y vuelve a intentarlo. Vaciar sólo la carpeta de descarga no libera el disco Linux de Podman. |
| «No hay una máquina WSL de Podman encendida» | Abre Podman Desktop → Resources y pulsa Start en su máquina. Espera a Running y repite instalar.cmd. |
| Mensaje sobre rootful | Sigue el paso 3: para la máquina, activa rootful y vuelve a arrancarla. |
| Error al arrancar y puerto 18083 ocupado | Cierra otra copia de esta demostración si sabes que la tienes abierta. No detengas procesos desconocidos. En la primera instalación puedes abrir PowerShell en la carpeta extraída y ejecutar `.\instalar.cmd -Puerto 18381`; se guardará ese puerto. |
| «La aplicación no ha arrancado en cinco minutos» o ClamAV iniciándose | Revisa en Podman Desktop el grupo etf-windows. Deja que el antivirus termine de arrancar, cierra programas que consuman RAM y repite instalar.cmd. No desactives el antivirus. |
| «Un archivo del paquete falta o ha cambiado» | Descarga y extrae otra vez el ZIP completo y comprueba su SHA-256. Conserva la carpeta privada. |
| «La operación ha tardado más de 20 minutos» | Comprueba que la máquina sigue encendida. Revisa los contenedores de etf-windows antes de repetir. Si actualizabas, conserva ambas carpetas y pide ayuda. |

Si necesitas ayuda, indica la versión de **VERSION.txt** y el mensaje que aparece.
No compartas contraseñas, capturas del Bloc de notas ni datos de tu PC.
