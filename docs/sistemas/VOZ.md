# Grabaciones del curso y de la ayuda

El responsable del producto autoriza edge-tts (versión 7.3.1; la 7.2.8 recibe 403 de Microsoft desde el 13/10/2026) con Microsoft para grabar los textos redactados por el proyecto. La autorización, ampliada el 14/10/2026, comprende curso, ayuda contextual y manuales de personas usuarias. Se usa la voz castellana `es-ES-AlvaroNeural`.

Genera los audios en el equipo de construcción con `make voz`, antes de `make curso`, `make build` o `make paquete-windows`. El comando requiere conexión a Microsoft. Envía únicamente texto editorial limpio; nunca datos de expedientes, información de PostgreSQL ni contenido introducido por las personas usuarias. La aplicación y el curso reproducen ficheros estáticos y no llaman al servicio de Microsoft.

Los MP3 mono, 24 kHz y 48 kbps quedan en `bin/voz/`, fuera de Git. La caché reutiliza cada texto ya grabado mediante SHA-256 de voz y texto. Los manifiestos enlazan secciones, ficheros, duración y huella. Conserva esta carpeta entre construcciones para evitar reenvíos. Un fallo conserva el manifiesto anterior; vuelve a ejecutar `make voz` cuando el servicio responda.

La imagen de aplicación y el paquete Windows incluyen la carpeta si existe. Los recursos se leen desde `/opt/etf/bin/voz`. No se necesita una credencial de síntesis en producción ni durante la instalación en Windows. La política CSP admite únicamente medios del propio servidor. Los recursos de ayuda exigen sesión activa y la pantalla muestra sólo los manuales de los perfiles autorizados.

Si faltan audios, la aplicación y el curso conservan el contenido y ocultan los controles de escucha. Una sección modificada necesita una nueva grabación; hasta entonces se puede leer. El lector queda oculto al seleccionar inglés. El contenido de los manuales se convierte al construir, con HTML crudo desactivado y una lista cerrada de imágenes y PDF locales.

Las pruebas de CI generan silencio con FFmpeg en `bin/voz-simulada/`, nunca llaman a Microsoft y no sustituyen los audios reales. La imagen de producción no copia esa carpeta. Procedimiento y pruebas en [scripts/voz/LEEME.md](../../scripts/voz/LEEME.md).

En la comprobación del 14/10/2026, `make voz` y una petición editorial breve con el ejecutable autorizado devolvieron HTTP 403 de Microsoft. No se generaron grabaciones reales. Las capturas del recorrido prueban los controles con silencio local; la generación real queda pendiente de recuperar el servicio. No se ha cambiado la versión ni la voz decididas por el director.

Las grabaciones de manual se filtran por los perfiles de la sesión, tanto en el manifiesto como al pedir el MP3. Las imágenes y PDF sólo se sirven cuando los referencia uno de sus manuales. Los enlaces a otro manual de personas usuarias conservan su referencia mediante el PDF editorial; esa referencia concede su lectura desde el manual que lo enlaza. Consulta no recibe recursos de manuales. Este contenido contiene datos inventados y no amplía el acceso a expedientes.
