# Probar Expedientes ETF en Windows

Usa sólo datos inventados. Nunca introduzcas datos reales de personas o expedientes.
No necesitas Docker ni herramientas de programación.

La guía completa está en
https://github.com/aavidad/expedientes-etf/blob/main/docs/windows/PROBAR-EN-CASA.md
La página de descargas es https://github.com/aavidad/expedientes-etf/releases.

## 1. Preparar Windows una vez

Necesitas Windows 10 22H2 o Windows 11 de 64 bits, 8 GB de RAM instalada
(16 GB recomendados) y al menos 15 GiB libres en C: para el instalador, además
del espacio para descargar y extraer el ZIP (unos 325 MiB y 627 MiB). Reserva
17 GiB libres antes de descargar. Reserva entre 20 y 60 minutos para
preparar tu PC y otros 5 a 15 para instalar la aplicación, según tu conexión y
su rendimiento. Necesitas Internet para descargar WSL, Podman Desktop, Compose
y su máquina Linux; después puedes usar la demostración sin conexión.

1. Pulsa Ctrl + Mayús + Esc → Más detalles → Rendimiento → CPU. Debe poner
   «Virtualización: Habilitado». Si no, busca en el soporte del fabricante de tu
   PC cómo activar Intel VT-x o AMD SVM en BIOS/UEFI. No cambies otros ajustes;
   si necesitas ayuda, pídela antes de continuar.
2. Abre PowerShell como administrador y ejecuta `wsl --install`. Reinicia.
   Si se abre Ubuntu, termina de crear su usuario. No es una cuenta de la app.
3. Abre PowerShell y ejecuta `wsl --status`: debe indicar versión predeterminada 2.
   Si falla, ejecuta `wsl --update` como administrador y reinicia. En el ensayo
   se usó `wsl --install --no-distribution`, WSL 3.0.3 y núcleo 6.18.54.1.
4. Descarga Podman Desktop para Windows x64 desde https://podman-desktop.io/downloads.
   Versiones ensayadas: Desktop 1.29.3, Podman 6.1.4 y Compose 5.1.4.
5. Abre su asistente Get started / Start Onboarding. Instala Podman con WSLv2,
   crea la máquina (en el ensayo: 2 procesadores y 4 GiB) e instala Compose.
   Espera a que la máquina indique Running.
6. Activa Rootful en los ajustes de esa máquina. Si no aparece, abre PowerShell
   normal y ejecuta, una orden cada vez:

   ```powershell
   podman machine stop
   podman machine set --rootful
   podman machine start
   podman machine list
   podman compose version
   ```

## 2. Descargar, comprobar e instalar

1. Descarga de la misma Release el ZIP v1.0.0 y su `.zip.sha256`.
2. En el Explorador abre esa carpeta, escribe `powershell` en la barra de
   dirección y pulsa Intro. Ejecuta:

   ```powershell
   Get-FileHash .\expedientes-etf-windows-v1.0.0.zip -Algorithm SHA256
   Get-Content .\expedientes-etf-windows-v1.0.0.zip.sha256
   ```

   Compara los 64 caracteres del Hash y del archivo; deben coincidir, sin importar
   mayúsculas. Si difieren, vuelve a descargar ambos y no instales esa copia.
3. Pulsa el ZIP con el botón derecho → Extraer todo. Usa una carpeta local,
   por ejemplo C:\ETF-Pruebas, fuera de OneDrive y unidades compartidas.
4. Abre Expedientes-ETF y haz doble clic en instalar.cmd desde el Explorador,
   sin elevar permisos. Si aparece SmartScreen y has comprobado procedencia y
   suma, pulsa Más información → Ejecutar de todas formas. El paquete no está
   firmado. Si esa opción no aparece, pide ayuda; no desactives SmartScreen.
5. Espera a la comprobación de requisitos, carga de imágenes y preparación de
   datos. El antivirus incluido puede tardar en arrancar. Al terminar se abre
   Edge y el Bloc de notas con las cuentas y contraseñas propias de tu PC.

Se crea «Expedientes ETF» en escritorio e Inicio. Abre Edge en una ventana propia
(o Chrome si falta Edge), en http://localhost:18083. La copia privada del paquete
permite mover la carpeta extraída. Las contraseñas están en USUARIOS.txt dentro
de `%LOCALAPPDATA%\Expedientes-ETF\pruebas`, accesibles sólo para tu cuenta.
No compartas ese archivo ni entorno.env.

| Usuario inventado | Perfil |
| --- | --- |
| ines.moreno, javier.medina | Coordinación: derivaciones, decisiones y actas de todos los equipos. |
| marta.cabrera | Profesional del ETF 2: ficha, documentos y Memoria de su equipo. |
| pablo.ruiz | Profesional del ETF 4: mismo trabajo en su equipo. |
| lucia.navarro | Consulta: sólo cifras, sin fichas personales. |
| elena.salas | Administración funcional: cuentas, catálogos, idiomas, temas y plantillas. |
| sergio.campos | Sin perfil: acceso pendiente. |

## 3. Del Anexo al Acta en cinco pasos

1. Entra como ines.moreno con la contraseña del Bloc de notas. En el menú Derivaciones,
   pulsa Subir Anexo 3 o 4.
2. Elige Anexo-3-pruebas.docx, incluido en esta carpeta, y pulsa Subir y leer anexo.
3. Revisa los datos, completa los que falten y confirma la derivación.
4. Ábrela. Completa fecha, equipo y los datos pendientes de los seis menores.
   Pulsa Aceptar derivación y después Aceptar y asignar número.
5. Descarga el Acta de aceptación y comprueba número, familia, seis menores y
   firmante. Puedes abrirla con Word o LibreOffice si tienes alguno instalado.

Prueba también Anexo-3-pruebas.odt desde Nueva derivación. En Expedientes
→ Añadir un expediente histórico, usa Acta-aceptacion-pruebas.docx y revisa los datos antes de
confirmar. Si el número ya existe, utiliza otro número inventado.

## 4. Parar, volver, reiniciar o desinstalar

- parar.cmd conserva datos y contraseñas. Después puedes cerrar Podman Desktop.
- Para volver, abre Podman Desktop y el acceso Expedientes ETF: prepara el
  arranque y abre la aplicación aunque estuviera parada.
- reiniciar-datos.cmd exige escribir BORRAR. Elimina todos tus cambios y genera
  contraseñas nuevas. Intro sin escribir nada cancela.
- desinstalar.cmd exige DESINSTALAR. Elimina datos, contraseñas y accesos propios;
  conserva archivos añadidos por ti y los programas Podman Desktop y WSL.
- Quitar Podman Desktop y WSL es opcional. Hazlo desde Configuración → Aplicaciones
  sólo si no los usa otro programa. Eliminar una máquina o ejecutar
  `wsl --unregister NOMBRE` borra todos sus datos. Comprueba los nombres con
  `wsl --list --verbose` y guarda lo que necesites antes.

## Si algo falla

| Mensaje o problema | Qué hacer |
| --- | --- |
| Virtualización deshabilitada | Sigue el paso 1 y el soporte de tu fabricante. |
| WSL2 no está listo | `wsl --update` como administrador, reiniciar y comprobar `wsl --status`. |
| Menos de 8 GB | Usa un PC con al menos 8 GB instalados. Cerrar programas no cambia ese mínimo. |
| Menos de 15 GB libres | Libera espacio en la unidad del perfil y repite. |
| Máquina Podman parada | Abre Podman Desktop, pulsa Start y espera a Running. |
| Rootful requerido | Para la máquina, activa rootful y vuelve a arrancarla. |
| Puerto 18083 ocupado | Cierra otra copia de la demo si la reconoces. En la primera instalación puedes usar `instalar.cmd -Puerto 18381`. |
| Antivirus lento o aplicación sin arrancar en cinco minutos | Revisa etf-windows en Podman Desktop; deja terminar ClamAV, libera RAM y repite instalar.cmd. |
| Archivo cambiado | Descarga y extrae de nuevo el ZIP completo. Conserva la carpeta privada. |
| Más de 20 minutos | Comprueba que la máquina esté encendida y revisa sus contenedores. |

Para actualizar, extrae el paquete nuevo en otra carpeta y ejecuta su actualizar.cmd.
Conserva datos y contraseñas. Si falla, conserva ambas carpetas y pide ayuda;
no uses reiniciar-datos para reparar una actualización. Indica VERSION.txt y el
mensaje de error; nunca envíes credenciales ni capturas del Bloc de notas.
