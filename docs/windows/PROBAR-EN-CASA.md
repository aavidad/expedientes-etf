# Probar Expedientes ETF en casa

Esta guía te ayuda a instalar una demostración en tu PC. Usa sólo datos inventados.
Nunca introduzcas datos reales de personas o expedientes, aunque trabajes desde casa.
No necesitas Docker ni herramientas de programación.

## Novedades de la versión 1.0.3

- Instalación: ya no tienes que preparar Windows antes. instalar.cmd activa
  WSL2 si falta, instala Podman para tu usuario, crea su máquina y usa el
  Compose que viene en el paquete. Si Windows tiene que reiniciarse, la
  instalación sigue sola cuando vuelves a entrar.
- Arranque tras reiniciar: si Podman todavía se está encendiendo, la ventana
  espera en lugar de fallar. Si algo sale mal, el aviso aparece delante de las
  demás ventanas y queda un registro sin contraseñas.
- Parar: parar.cmd apaga también la máquina de Podman, para que Windows pueda
  apagarse o reiniciarse sin quedarse esperando.
- Desinstalar: quita sólo lo que puso el paquete y pregunta antes de quitar
  Podman. WSL2 se queda en el PC.
- Plantillas: los documentos Word del paquete inglés están traducidos por
  completo.

La versión 1.0.3 conserva lo que trajeron las anteriores:

- Ayuda: el menú tiene un apartado con el manual de tu perfil, su índice, las
  capturas y un PDF para imprimir. Puedes escuchar en castellano la ayuda de
  cada pantalla y los manuales; las grabaciones vienen dentro del paquete.
- Antivirus: la ventana espera a que esté listo antes de abrir la aplicación,
  así el primer anexo que subas ya se revisa.
- Anexo 6: desde la ficha o desde Salidas y cierre puedes preparar la derivación
  a otro servicio, descargar el Word y anotar después la fecha real de envío.
- Memoria: al pulsar una cifra ves qué expedientes la forman, con su número,
  equipo y aportación, sin nombres.
- Consulta: las cifras por debajo de 3 casos aparecen como «Oculto», en pantalla
  y en Word, para que nadie pueda reconocer a una familia.
- Sesión: se cierra tras 30 minutos sin actividad, con un aviso dos minutos
  antes. Lo que ya guardaste se conserva.
- Carga histórica: Administración fija hasta qué día se pueden añadir expedientes
  antiguos. La aplicación avisa de posibles duplicados y Coordinación puede anular
  un lote completo dentro del plazo.
- Restaurar una copia: después de recuperarla, Administración revisa cuentas y
  permisos antes de que nadie vuelva a entrar.
- Actualizar: actualizar.cmd cambia los textos, plantillas y temas que trae el
  paquete y conserva los que hayas publicado o cambiado tú.

## 1. Comprueba tu PC

Necesitas Windows 10 de **64 bits** (versión 2004 o posterior) o Windows 11 y
**8 GB de RAM instalada** como mínimo. Con 16 GB tendrás más margen para usar
otros programas. Para comprobar tu versión, abre Inicio, escribe `winver` y pulsa
Intro. Puedes ver la RAM en Configuración → Sistema → Acerca de.

El instalador exige **15 GB libres en la unidad de tu perfil**, normalmente C:.
El ZIP ocupa unos **466 MiB** y la carpeta extraída, **801 MiB**, medidos en
esta entrega. Reserva **17 GiB libres antes de descargar** para mantener
los 15 GB que comprueba el instalador después de extraer. La copia privada del
paquete ocupa otros 801 MiB dentro de esa reserva, además de las imágenes y los
datos. Las medidas exactas se anotan en la [evidencia de la entrega](../sistemas/WIN1.md#paquete-público-103).
Comprueba el espacio desde Explorador → Este equipo → Disco local (C:).

La primera instalación necesita Internet: WSL2 se descarga de Microsoft y Podman
descarga el sistema de su máquina (unos 250 MB). El resto viene en el ZIP.
Después puedes usar la demostración sin conexión. Reserva entre 15 y 45 minutos
para la primera instalación, según tu conexión y tu PC, y algo más si Windows
tiene que reiniciarse. Son orientaciones, no tiempos garantizados.

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

## 2. Descarga y comprueba el paquete

1. Abre las [Releases de Expedientes ETF](https://github.com/aavidad/expedientes-etf/releases).
   En la entrega 1.0.3 descarga **expedientes-etf-windows-1.0.3.zip** y
   **expedientes-etf-windows-1.0.3.zip.sha256**. Guarda ambos en la misma carpeta.
2. En el Explorador, abre esa carpeta. Pulsa la barra de dirección, escribe
   `powershell` y pulsa Intro. Copia:

   ```powershell
   Get-FileHash .\expedientes-etf-windows-1.0.3.zip -Algorithm SHA256
   Get-Content .\expedientes-etf-windows-1.0.3.zip.sha256
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

## 3. Instala la demostración

1. Abre la carpeta extraída **Expedientes-ETF** y haz doble clic en
   **instalar.cmd** desde el Explorador. No lo abras dentro del ZIP ni como administrador.
2. Puede aparecer SmartScreen porque este paquete **no está firmado** y aún no
   tiene reputación de descarga. Si has comprobado su procedencia y su suma,
   pulsa **Más información → Ejecutar de todas formas**. Si tu PC no ofrece esa
   opción, consulta a quien lo administre; no desactives la protección de Windows.
   También puede salir «Editor desconocido» con un botón **Ejecutar**, como en
   [el aviso real del ensayo](capturas/aviso-seguridad.png). Comprueba antes
   procedencia y suma, y pulsa ese botón para continuar.
3. Deja abierta la consola y lee lo que indica. Si falta algo, lo instala en
   este orden:
   - **WSL2**, el componente de Windows que ejecuta los programas Linux de la
     demostración. Windows pide permiso de administrador: pulsa **Sí**. Si
     después pide reiniciar, guarda tu trabajo y reinicia. Al volver a entrar
     con tu usuario, la instalación sigue sola en una ventana nueva.
   - **Podman**, el programa que abre los componentes de la aplicación. Se
     instala sólo para tu usuario, sin permiso de administrador. Es gratuito.
   - **La máquina de Podman**, con 4 GB de memoria y 2 procesadores. Se
     enciende sola.
   - **Compose**, que arranca juntos todos los componentes. Va dentro del paquete.
4. Después comprueba las sumas de los archivos y verás «Cargando las imágenes
   incluidas» y «Preparando la base de datos y arrancando la aplicación». La
   primera carga tarda varios minutos. El antivirus incluido también necesita
   arrancar; no cierres la ventana mientras trabaja. Si se corta, vuelve a
   ejecutar instalar.cmd: sigue donde lo dejó.

Si Windows indica que la virtualización está deshabilitada, sigue los pasos del
apartado 1 para activarla y vuelve a ejecutar instalar.cmd. Si tu cuenta no
puede dar permiso de administrador, quien mantenga tu PC puede prepararlo con el
plan B del LEEME.md que viene en la carpeta.

5. Al terminar aparecen el acceso **Expedientes ETF** en el escritorio y en
   Inicio, la aplicación en una ventana propia de **Microsoft Edge** y un Bloc
   de notas con las cuentas y sus contraseñas. Si no tienes Edge, usa Chrome.
6. Conserva ese archivo de contraseñas en privado. Cada instalación genera
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

## 4. Tu primer recorrido: del Anexo al Acta

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

## 5. Parar, volver y borrar las pruebas

- Para terminar por hoy, abre **parar.cmd**. Conserva tus datos y contraseñas y
  apaga la máquina de Podman si no la usa otra aplicación.
- **Antes de apagar o reiniciar Windows, cierra la aplicación con parar.cmd.**
  Con la máquina de Podman encendida, Windows puede tardar mucho en reiniciarse.
- Para volver, pulsa **Expedientes ETF** en el escritorio o en Inicio. El acceso
  enciende la máquina de Podman si hace falta, arranca la demostración si está
  parada y abre su ventana de Edge. Si Podman ya se estaba encendiendo, espera a
  que termine. La preparación puede tardar unos minutos.
- Para empezar de cero, abre **reiniciar-datos.cmd**. Escribe **BORRAR** y pulsa
  Intro sólo si quieres eliminar todos tus cambios. Se recrean las cuentas con
  contraseñas nuevas. Pulsa Intro sin escribir nada para cancelar.
- Para quitar la aplicación, abre **desinstalar.cmd** y escribe **DESINSTALAR**.
  Elimina sus datos, contraseñas y accesos; conserva archivos que hayas añadido.
  Si la instalación puso Podman o su máquina, te pregunta si también quieres
  quitarlos: escribe **QUITAR** para hacerlo. Si la máquina tiene contenedores
  de otras aplicaciones, no la toca.
- WSL2 se queda en el PC, porque otros programas pueden usarlo. Si nadie más lo
  usa, puedes quitarlo desde Configuración → Aplicaciones (Subsistema de Windows
  para Linux) y desactivar «Subsistema de Windows para Linux» y «Plataforma de
  máquina virtual» en «Activar o desactivar las características de Windows».
  Después reinicia. No desactives «Plataforma de máquina virtual» si la
  necesita otro programa.

## Si algo falla

| Lo que ves | Qué puedes hacer |
| --- | --- |
| Virtualización deshabilitada | Revisa Administrador de tareas → Rendimiento → CPU y el apartado 1. Actívala siguiendo el soporte del fabricante. |
| Rechazaste el permiso de administrador para WSL2 | Vuelve a ejecutar instalar.cmd y pulsa **Sí**. Si tu cuenta no puede darlo, usa el plan B del LEEME.md. |
| Windows pide reiniciar | Guarda tu trabajo y reinicia. Al entrar de nuevo, la instalación sigue sola. |
| «El PC necesita al menos 8 GB» | Comprueba la RAM instalada. Cerrar programas no cambia ese mínimo: si tienes menos de 8 GB, usa otro PC. |
| «Necesitas al menos 15 GB» | Libera espacio en C: y vuelve a intentarlo. Vaciar sólo la carpeta de descarga no libera el disco Linux de Podman. |
| «No hay una máquina WSL de Podman encendida» | Ejecuta instalar.cmd otra vez: la crea o la enciende. |
| Mensaje sobre rootful | La máquina de Podman que ya tenías no usa ese modo. Sigue el plan B del LEEME.md para activarlo. |
| Error al arrancar y puerto 18083 ocupado | Cierra otra copia de esta demostración si sabes que la tienes abierta. No detengas procesos desconocidos. En la primera instalación puedes abrir PowerShell en la carpeta extraída y ejecutar `.\instalar.cmd -Puerto 18381`; se guardará ese puerto. |
| «La aplicación no ha arrancado en cinco minutos» o ClamAV iniciándose | Deja que el antivirus termine de arrancar, cierra programas que consuman RAM y repite instalar.cmd. No desactives el antivirus. |
| «Un archivo del paquete falta o ha cambiado» | Descarga y extrae otra vez el ZIP completo y comprueba su SHA-256. Conserva la carpeta privada. |
| La ventana de arranque avisa de un error | Lee el aviso. Lo que hizo el acceso queda en `%LOCALAPPDATA%\Expedientes-ETF\pruebas\registro-abrir.log`, sin contraseñas; es el único archivo que puedes enviar al pedir ayuda. |
| «La operación ha tardado más de 20 minutos» | Comprueba que la máquina sigue encendida antes de repetir. Si actualizabas, conserva ambas carpetas y pide ayuda. |

Si necesitas ayuda, indica la versión de **VERSION.txt** y el mensaje que aparece.
No compartas contraseñas, capturas del Bloc de notas ni datos de tu PC.
