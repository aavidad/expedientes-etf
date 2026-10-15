# Mantener y exportar los manuales

Los manuales para las personas usuarias empiezan en [usuario.md](usuario.md). Esta guía explica cómo mantenerlos y generar los PDF en el equipo local.

## Fuentes y alcance

La edición F5.06 describe las pantallas disponibles y las reglas de los documentos de producto. Incluye CA1 de alta histórica y carga por hoja y F4.02 de consulta, textos, Word, cierre y reapertura de Memoria, integradas antes de cerrar esta entrega. GAP1 completa la edición de Familia y la descarga separada de los Anexos 11/12 desde Cierre. El Anexo 6 aún no tiene formulario de generación en la ficha. MEMDES incorpora el desglose de las cifras de Memoria por expedientes. La consulta de profesionales por nombre y la ZTS siguen pendientes.

Las capturas reutilizadas de [docs/capturas](../capturas/) contienen datos inventados y tienen sus recorridos de origen junto a ellas. Algunas corresponden a una versión anterior del menú o del cálculo. Las capturas de [Administración funcional](administracion.md) están en [administracion/](administracion/) y muestran la demo local a 1440 px y el editor de permisos a 390 px, con Elena Salas y personal inventado. Las capturas nuevas de [capturas/](capturas/) se toman en la demostración de este worktree, con Playwright y `/usr/bin/google-chrome`, sin modificar la página para representar funciones ausentes. Se usan Marta Cabrera e Inés Moreno; los nombres, fechas y datos de los casos son inventados. Las contraseñas quedan fuera de las imágenes y de Git.

Los vídeos disponibles son los seis que aparecen en [usuario.md](usuario.md#vídeos-de-demostración). Se enlazan sus guiones y se cita el título para localizar el MP4 fuera del repositorio. Esta entrega no crea ni copia vídeos.

## Leer en la aplicación y escuchar

El menú Ayuda muestra el manual de tus perfiles con índice y capturas. La conversión web usa `python3 -B scripts/voz/construir.py` y se ejecuta al construir la interfaz. Incluye sus datos generados de `docs/manuales/web/` cuando cambies las fuentes. No admite HTML crudo. `make voz` añade las grabaciones opcionales; si faltan, se conserva la lectura. [Procedimiento](../../scripts/voz/LEEME.md).

## Exportar a PDF

Necesitas Python 3, Pillow instalado en el equipo, LibreOffice local (`soffice`), ExifTool y qpdf. La exportación no instala dependencias ni usa servicios externos.

Desde la raíz del worktree:

```sh
mkdir -p bin/temporales
TMPDIR="$PWD/bin/temporales" python3 -B docs/manuales/exportar.py
```

El script prepara documentos ODT temporales y ejecuta `env -u LD_LIBRARY_PATH soffice --headless --convert-to pdf:writer_pdf_Export` con un perfil aislado de LibreOffice. Copia los tres PDF a esta carpeta y elimina sus temporales. Puedes usar `--salida bin/temporales/f506-revision` para revisar una exportación antes de sustituir los PDF publicados. Para actualizar únicamente Administración, ejecuta `python3 -B docs/manuales/exportar.py --manual administracion`; la opción `--manual` se puede repetir.

Cada PDF contiene las imágenes: no depende de su carga al abrirlo. Las capturas largas se dividen en tramos consecutivos, sin omitir contenido. Los manuales de profesionales y Coordinación incluyen el glosario y las preguntas frecuentes comunes. El manual completo de Administración funcional contiene sus propios términos, incidencias y límites de gestión, sin añadir el glosario del Programa. Los enlaces internos llevan a los apartados del propio PDF. Los enlaces a otros archivos apuntan a este worktree; si compartes el PDF o cambias la carpeta del repositorio, consulta esos enlaces en Markdown o localiza el vídeo por su título.

El exportador admite los encabezados, párrafos, listas numeradas, enlaces, imágenes y tabla de glosario que usan estos manuales. Si añades otro formato, revisa su representación en PDF antes de entregar.

## Comprobación de F5.06

Se han revisado 29 tareas: 12 para profesionales, 13 para Coordinación y 4 para Administración. Todas llevan pasos numerados y una captura real. Los tres PDF conservan el texto completo de sus manuales. Profesionales y Coordinación ocupan 18 páginas cada uno; Administración, dos páginas A4. Las imágenes se exportan sin compresión con pérdida y los encabezados permiten navegar por tareas en el PDF.

Las diez capturas nuevas se tomaron el 14 de octubre de 2026, en la demostración local del worktree, puerto 18506. Menores e Intervención muestran la cuenta del profesional. El conflicto de Menores se provocó con un guardado concurrente real y se comprobó que el formulario seguía disponible. Para Cierre, Marta guardó y propuso la finalización; Inés confirmó el cierre con su propia cuenta y se consultó la reapertura disponible. El registro de salida se capturó tras descargar el oficio. La demostración se detuvo al terminar.

Estas comprobaciones acreditan las pantallas capturadas y los documentos. Las capturas de carga se reutilizan del [recorrido CA1](../capturas/carga.md), con datos inventados. Memoria reutiliza el [recorrido F4.02](../capturas/memoria.md): sus dos vistas superiores conservan los píxeles de las capturas originales en el rectángulo de 1440 por 1100 píxeles desde su esquina superior izquierda. Los manuales enlazan también las imágenes completas. No se declara una aprobación del cálculo por el Programa ni de los resultados pendientes de D22.

Se revisaron el texto y las imágenes de los PDF, incluidos cierre y registro de salida. El tono se repasó con `humanizer`, manteniendo las reglas y los límites del producto.

La primera ejecución de `make ci` reutilizó la demostración de las capturas y falló en el ensayo de permisos al intentar modificar un expediente cerrado (`error.estado_no_permite`). Después de integrar `main` con CA1, se limpió sólo la demostración local de este worktree con `make demo-limpiar` y se ejecutó de nuevo `make ci` completo. Pasaron `go vet ./...`, `go test ./...`, las comprobaciones de frontend, textos, temas, migraciones y los recorridos F0.11 y CA1. La última línea fue:

```text
CI mínima completada.
```

La comprobación adicional de los manuales verificó el texto íntegro de cada PDF, sus capturas, los enlaces internos y los destinos de los enlaces al worktree. No quedan enlaces hacia los ODT temporales.

Al avanzar `main` con F4.02, se incorporó la pantalla de Memoria a los manuales y se exportaron de nuevo los tres PDF. Se repitió `make ci` completo con una demostración limpia de este worktree y terminó de nuevo con `CI mínima completada.` Los manuales finales incluyen 29 tareas y las acciones de consulta, descarga, cierre y reapertura de Memoria.

## Revisar un cambio

1. Comprueba cada nombre de acción contra la pantalla disponible y su catálogo de textos. Una acción pendiente debe indicar su límite y no describir un botón inexistente.
2. Revisa el tono con `humanizer`: castellano claro, de tú, instrucciones directas y términos del Programa explicados.
3. Comprueba enlaces e índice de producto con `python3 -B scripts/ci/enlaces.py` y `python3 -B scripts/ci/indice/generar.py --comprobar`.
4. Exporta los tres PDF. Revisa el texto extraído con `pdftotext` y las páginas con `pdftoppm`; comprueba pasos, imágenes, pies y saltos de página.
5. Ejecuta `make ci` completo antes de entregar y actualiza los enlaces de los README y la fila F5.06 si cambia el alcance.

## Edición pública

Los manuales usan la cuenta corporativa de la entidad. Las capturas de acceso y las comparaciones con originales privados se retiraron; las restantes contienen datos inventados. El generador exporta enlaces relativos, identifica a Expedientes ETF como autor y creador del PDF y reescribe el archivo con qpdf para eliminar metadatos antiguos.
