# Construir el lector de voz

El lector sirve grabaciones de los textos del proyecto. La aplicación y el curso no envían texto a Microsoft al abrir una página ni al pulsar Escuchar.

## Generar las grabaciones

Necesitas Python 3 con PyYAML 6.0.3 y markdown-it-py 3.0.0, las dependencias locales del curso, FFmpeg y FFprobe. La síntesis utiliza `~/.local/bin/edge-tts` 7.3.1 (la 7.2.8 dejó de funcionar el 13/10/2026: Microsoft responde 403) y la voz `es-ES-AlvaroNeural`, autorizada por el responsable del producto el 14/10/2026.

Desde la raíz del worktree:

```sh
make voz
make curso
make build
```

`make voz` envía a Microsoft sólo el texto limpio de `curso/contenido/`, los tres manuales de personas usuarias y su ayuda común, y las claves que contienen `ayuda` de `idiomas/es/textos/*.json`. Elimina Markdown, URLs y alternativas de imágenes. No consulta PostgreSQL ni admite textos de expedientes o entradas de personas usuarias. Las autoevaluaciones leen preguntas y opciones, sin revelar las soluciones.

Las grabaciones quedan fuera de Git, en `bin/voz/{curso,ayuda,manuales}/`. FFmpeg convierte cada resultado a MP3 mono, 24 kHz y 48 kbps. La caché usa SHA-256 de voz y texto. Cada carpeta publica `manifiesto.json`: identificador de sección, fichero, duración en segundos y SHA. Una segunda ejecución reutiliza las grabaciones existentes. Si falla una generación, conserva el manifiesto anterior y elimina el fichero incompleto. Puedes repetir el comando cuando vuelva el servicio.

La ayuda contextual comprueba también que el texto activo coincide con el texto grabado. Los manuales y el curso comparan la huella de cada sección; una edición sin nueva grabación se puede leer y oculta ese control. Si no hay grabaciones, toda la lectura sigue disponible. El lector se ofrece sólo con el idioma castellano.

## Manuales y exportaciones

`python3 -B scripts/voz/construir.py` convierte los manuales a datos HTML estáticos en `docs/manuales/web/`, sin llamar a la red. `make web`, `make build` y la preparación de la demo ejecutan esta conversión. Se conservan esos datos en Git para que la imagen Docker pueda construirse sin Python ni síntesis. Al cambiar un manual, incluye también sus datos web en el commit. El contenido Markdown se renderiza con HTML crudo desactivado. Sólo se publican recursos locales de una lista construida; no se ofrece acceso libre a `docs/`.

El menú Ayuda muestra los manuales de Profesional de ETF, Coordinación o Administración funcional que correspondan a los perfiles de la sesión. Incluye índice, capturas y PDF. Consulta tiene la pantalla y un aviso de manual no disponible. Los manuales de Sistemas y de programación quedan fuera de la aplicación.

`make curso` incluye los audios disponibles tanto en la web como en SCORM 1.2, junto con un script local de reproducción. Cada página tiene un control para escuchar las secciones en orden. Las secciones con encabezado `##` tienen su propio control. Ambos formatos funcionan sin síntesis, sin conexión a Microsoft y sin reproducción automática.

La imagen Docker copia `bin/voz/` si existe. El paquete Windows conserva esos recursos dentro de la imagen de aplicación exportada. Ninguno genera audio durante la instalación.

## Pruebas sin red de síntesis

```sh
python3 -B -m unittest discover -s scripts/voz -p 'test_*.py'
make voz-simulada
make demo
node scripts/recorridos/voz.mjs http://127.0.0.1:18614
make demo-parar
```

Ajusta el puerto a tu demostración. `make voz-simulada` genera silencio local en `bin/voz-simulada/`, separado de la caché real. La demo utiliza ese silencio cuando faltan grabaciones reales completas. Docker de producción no incorpora la carpeta simulada. No copies esos ficheros a `bin/voz/`.

`make ci` incluye las pruebas unitarias y el recorrido con Chrome del sistema. Nunca ejecuta edge-tts. En un equipo sin exportación eXeLearning, el recorrido prepara páginas de prueba desde las fuentes editoriales y aplica el mismo lector; no acredita la exportación eXeLearning. El recorrido local utiliza la exportación web real cuando está construida.

Para guardar las capturas solicitadas, añade `ETF_VOZ_CAPTURAS=1` al comando del recorrido. Comprueba lectura por perfil, teclado, ayuda contextual, capturas de manual, ausencia de audio, fallo con reintento, inglés sin lector y anchos 1440, 390, 360 y 1024. Ejecuta axe sobre la pantalla Ayuda y revisa el tema de alto contraste. Estas comprobaciones no sustituyen una revisión con un lector de pantalla del puesto.

Las grabaciones de manual se filtran por los perfiles de la sesión, tanto en el manifiesto como al pedir el MP3. Las imágenes y PDF sólo se sirven cuando los referencia uno de sus manuales. Los enlaces a otro manual de personas usuarias conservan su referencia mediante el PDF editorial; esa referencia concede su lectura desde el manual que lo enlaza. Consulta no recibe recursos de manuales. Este contenido contiene datos inventados y no amplía el acceso a expedientes.
