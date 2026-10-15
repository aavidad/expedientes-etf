# Comprobación del lector de voz

La comprobación terminó el 15/10/2026 en la rama `lector-voz`, con Chrome del sistema y la demostración propia del worktree en el puerto 18614. Se usaron Inés Moreno, Marta Cabrera, Elena Salas y Lucía Navarro, con datos inventados. Las contraseñas no aparecen en las capturas ni en Git.

## Alcance comprobado

- Curso web exportado por eXeLearning 4.0.5: control de página en la portada, escucha de secciones y paso al siguiente audio. El paquete SCORM incluye controles, script, MP3 y sus entradas de manifiesto; lo verifican las pruebas de construcción. No se ha probado una plataforma LMS de destino.
- Ayuda: manual de los perfiles de cada sesión, índice por secciones, capturas, PDF y controles de escucha. Consulta recibe el estado de manual no disponible. Las imágenes y PDF se limitan a los recursos referenciados por el manual autorizado; los MP3 y su manifiesto se filtran por perfil.
- Ayuda contextual: diálogo de la Tabla SPAF y ayuda desplegada de Memoria; cierre con Escape y detención del audio. Sólo se comparan textos del catálogo, sin enviar ni grabar contenido de las personas.
- Teclado, anchos 1440, 1024, 390 y 360, y tema de alto contraste. Al elegir inglés desaparece el lector; el contenido editorial del manual conserva su atributo de idioma castellano.
- Ausencia de grabaciones y fallo de carga del manual con reintento. Sin reproducción automática ni peticiones externas desde Chrome.

El recorrido se ejecuta con `node scripts/recorridos/voz.mjs http://127.0.0.1:18614`. `ETF_VOZ_CAPTURAS=1` guarda las imágenes. La comprobación axe de Ayuda no detectó infracciones serias ni críticas. No se declara una certificación de accesibilidad ni una revisión con NVDA.

## Resultados

Las nueve pruebas de `scripts/voz/` pasaron: limpieza de Markdown y URLs, exclusión de alternativas de imágenes, división de secciones, caché sin reenvío, publicación íntegra del manifiesto, rutas e identificadores, HTML saneado, enlaces y recursos de curso web/SCORM. Los tests HTTP de VOZ pasaron, incluido acceso negativo a PDF, MP3 y manifiesto de un manual ajeno, sesión y Range. `make ci` ejecutó también `go vet ./...`, `go test ./...`, las 37 pruebas frontend, ensayos SQL y recorridos completos. Última línea real:

```text
CI mínima completada.
```

La matriz independiente F0.11 comprobó 1896 combinaciones HTTP, 378 lecturas y 810 escrituras denegadas con RLS. La entrada espera cuando recibe HTTP 429 y repite sólo el GET previo al acceso; los límites del servidor se conservan. Main incorporó la corrección de fecha de la prueba de Traslado desde espera, por lo que no queda ningún overlay de reloj de VOZ.

Semgrep se ejecutó localmente, con reglas locales y métricas desactivadas: cero hallazgos. La revisión independiente de usabilidad y seguridad focal corrigió enlaces, portada, títulos repetidos, tamaño de botones y permisos de recursos; su última pasada no encontró un defecto nuevo confirmado.

## Estado de las grabaciones

`make voz` se intentó con el ejecutable autorizado edge-tts 7.2.8 y `es-ES-AlvaroNeural`. Microsoft devolvió HTTP 403, también en el último intento. No hay grabaciones reales generadas por esta entrega. La síntesis real queda pendiente de recuperar el servicio, manteniendo la voz y versión decididas por el director.

Las capturas y la CI usan silencio generado localmente, separado en `bin/voz-simulada/`. FFprobe comprobó 24.000 Hz, un canal y 48.000 bits por segundo. El silencio permite verificar el reproductor, su secuencia y los permisos; no acredita la voz ni su pronunciación. La carpeta simulada no se copia a la imagen de producción. Sin audios reales, aplicación y curso conservan la lectura y ocultan los controles.

## Capturas

- Ayuda: [1440](voz-ayuda-1440.png), [1024](voz-ayuda-1024.png), [390](voz-ayuda-390.png), [360](voz-ayuda-360.png) y [alto contraste a 390](voz-contraste-390.png).
- Ayuda contextual: [1440](voz-contextual-1440.png) y [390](voz-contextual-390.png).
- Curso web: [1440](voz-curso-1440.png) y [390](voz-curso-390.png).

La demostración se detuvo al terminar. Las capturas de otros módulos que regeneró CI se restauraron a las versiones de main; esta entrega conserva sólo las capturas VOZ.
