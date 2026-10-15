# Expedientes ETF: curso de la aplicación

El curso explica cómo usar la aplicación. Incluye presentación, tareas por perfil, un caso completo y un glosario de términos de la aplicación. Los módulos M4 y M5 contienen la aplicación por perfiles y el caso práctico completo.

Autor: Alberto Avidad. Contenido con licencia [CC BY-SA 4.0](LICENCIA.md). La aplicación es software libre con licencia EUPL-1.2.

## Construir y exportar

Necesitas Docker con el contenedor `etf-curso-exelearning-1` en ejecución, eXeLearning 4.0.5, Python 3, PyYAML 6.0.3 y markdown-it-py 3.0.0. Las versiones de Python están fijadas en `requirements.txt`; el entorno local ya las incluye. La construcción verifica la versión instalada de eXeLearning.

Desde la raíz del repositorio:

```sh
TMPDIR=/dev/shm/etf/tmp make curso
```

Se generan estos archivos, excluidos de Git:

| Resultado | Ruta |
| --- | --- |
| Proyecto editable | `bin/curso/expedientes-etf.elpx` |
| Sitio web completo | `bin/curso/web/` |
| Paquete SCORM 1.2 | `bin/curso/scorm12.zip` |
| Inventario de páginas, preguntas y dudas | `bin/curso/resumen.json` |

Para consultar el sitio sin conexión, abre `bin/curso/web/index.html` con Chrome. Copia siempre la carpeta completa. Las páginas, actividades e imágenes funcionan con `file://`; los enlaces a otros sitios necesitan Internet. Para servirlo junto a la aplicación, publica el contenido de `bin/curso/web/` bajo `/curso/`. Los enlaces son relativos. Este encargo prepara el sitio; no modifica el despliegue de la aplicación.

Para SCORM, importa `scorm12.zip` en tu plataforma de formación, sin descomprimirlo. Los formularios envían la puntuación al LMS al comprobar las respuestas y permiten repetir. Los vídeos se incluyen sólo en la edición web; SCORM muestra un aviso en su lugar.

## Escuchar páginas y secciones

Ejecuta `make voz` antes de `make curso` para incluir las grabaciones disponibles en la edición web y en SCORM 1.2. El lector ofrece escucha de página en orden y de cada sección `##`, con controles nativos, teclado y sin reproducción automática. El curso reproduce los MP3 incluidos y no llama a Microsoft. Si falta una grabación o ha cambiado su texto, se conserva la lectura sin ese control. [Construcción, caché y pruebas](../scripts/voz/LEEME.md).

## Vídeos externos

El constructor copia únicamente los vídeos enlazados por las páginas. Los toma de `ETF_CURSO_VIDEOS`, por defecto `~/Videos/Expedientes-ETF`. Si un vídeo enlazado falta, la construcción falla e indica su ruta. Los MP4 no se guardan en Git ni en el proyecto editable.

```sh
ETF_CURSO_VIDEOS=/ruta/a/los/videos TMPDIR=/dev/shm/etf/tmp make curso
```

Puedes seleccionar otro contenedor de la misma versión con `ETF_CURSO_CONTENEDOR`. La construcción usa su CLI, sin entrar en la interfaz y sin leer credenciales.

## Editar las fuentes

`estructura.yaml` fija el título, la autoría, las licencias y el orden de los módulos. Cada carpeta de `contenido/` contiene páginas Markdown con cabecera YAML y autoevaluaciones YAML. El constructor incorpora todos los ficheros presentes, ordenados por su prefijo numérico; con el mismo número, coloca la página antes de la autoevaluación. Si una carpeta está vacía o falta, muestra un marcador.

Carpetas del curso:

- `m0-presentacion`: cómo usar el curso y créditos.
- `m4-aplicacion`: Profesional de ETF, Coordinación, Consulta y Administración funcional.
- `m5-caso-practico`: uso de la aplicación de principio a fin.
- `m9-glosario`: términos y preguntas frecuentes de la aplicación.

Una página tiene este formato:

```markdown
---
titulo: Revisar la ficha
objetivos:
  - Comprobar los datos antes de generar un documento.
---

## Antes de descargar

Revisa las fechas que has registrado.

![Descripción de la pantalla](../../../docs/capturas/ficha-integrada.png)
![vídeo: La ficha del expediente](videos/05-ficha.mp4)

<details><summary>Ver solución</summary>

Comprueba el hito y su fecha antes de continuar.

</details>
```

Utiliza `##` y `###` para las secciones. Cada imagen necesita texto alternativo. Se admiten tablas, enlaces entre páginas Markdown, recuadros con citas y soluciones con `details`. Las imágenes y PDF se incorporan desde el repositorio; el constructor rechaza rutas que salen de él. Conserva los nombres de los vídeos externos. No añadas scripts ni recursos que se carguen desde servicios externos.

Una autoevaluación se llama `NN-autoevaluacion.yaml`:

```yaml
titulo: Autoevaluación de la ficha
preguntas:
  - tipo: test
    enunciado: ¿Qué debes revisar antes de descargar el documento?
    opciones: [Los datos y las fechas, Sólo el título, Nada]
    correcta: 0
    explicacion: El documento se genera con los datos registrados.
  - tipo: vf
    enunciado: Descargar el Word confirma que ya está firmado.
    correcta: false
    explicacion: La firma se realiza por el cauce de la organización.
  - tipo: huecos
    enunciado: Completa el término.
    texto: El documento se genera con una {plantilla}.
    explicacion: La plantilla determina el formato del documento.
```

`correcta` empieza en cero para test. En verdadero/falso, utiliza los valores YAML `true` y `false`. En huecos, encierra cada respuesta entre llaves. El constructor genera un iDevice nativo «Formulario» con corrección y explicación. Los textos de sus controles están en `interfaz.yaml`.

Mantén los datos de prácticas inventados. Las dudas marcadas como «Validar con el Programa» se recogen en `bin/curso/resumen.json`; el constructor no modifica los contenidos de los módulos.

## Editar en eXeLearning y recuperar los cambios

1. Abre eXeLearning 4.0.5 en `http://127.0.0.1:18090`. Inicia sesión con las credenciales que mantiene la organización fuera de Git.
2. Usa **Archivo → Abrir** y selecciona `expedientes-etf.elpx`. Conserva los nombres y el orden de los cuatro módulos principales. El editor muestra una raíz con el título del curso, además de las páginas exportadas.
3. Edita el texto o las preguntas del iDevice «Formulario». Guarda cada iDevice antes de cambiar de página.
4. Usa **Archivo → Descargar** para conservar el proyecto `.elpx` fuera del repositorio. Puedes volver a abrir ese proyecto en eXeLearning y seguir editando.
5. Para llevar los cambios a las fuentes, recupera el proyecto en una carpeta nueva:

```sh
python3 -B curso/construir.py --reimportar /ruta/al/curso-editado.elpx --destino bin/curso/reimportado
```

La recuperación admite texto y una autoevaluación por página. Conserva el texto como HTML dentro de Markdown, que CommonMark admite, y las preguntas como YAML. Recupera los recursos disponibles en el proyecto. Si encuentra un tipo que no puede recuperar, falla antes de escribir; conserva el `.elpx` original para exportarlo desde eXeLearning.

Revisa `bin/curso/reimportado/contenido/` y compara los cambios con las fuentes. Lleva cada cambio a su módulo con el responsable de esa carpeta. Si recuperas imágenes, copia también `recursos/` junto a `contenido/`, conservando las rutas. No copies en bloque los ficheros recuperados sobre las fuentes sin revisarlos. Ejecuta `make curso` después de integrar los cambios. La construcción ordinaria siempre parte de las fuentes y reemplaza los resultados anteriores de `bin/curso/`.

También puedes exportar desde eXeLearning con sus opciones de sitio web y SCORM 1.2. Para la entrega offline con vídeos y las correcciones de exportación, utiliza el constructor después de recuperar e integrar las fuentes.

## Formato y vía de generación

Se utiliza un ZIP `.elpx` con `content.xml` ODE y recursos, leído por el importador de eXeLearning 4.0.5. El XML describe metadatos, páginas, bloques e iDevices. Los textos y propiedades JSON se guardan en CDATA. No necesita un estado Yjs precalculado: el importador lo crea al abrir el proyecto.

El constructor genera el proyecto directamente y llama, dentro del contenedor, a la CLI oficial `bun /app/dist/cli.js elp:export … html5` y `… scorm12`. Después incluye los vídeos en web, incorpora la plantilla del formulario al HTML para evitar peticiones bloqueadas por `file://`, desactiva el envío de notas en web y completa autor y derechos en el manifiesto SCORM. `ajustes.css` corrige el desbordamiento del texto oculto de navegación en móvil; se conserva el estilo base de eXeLearning, sin cambiar colores ni fuentes.

El contrato se comprobó leyendo el importador, el esquema ODE, el iDevice Formulario y la CLI de la imagen `exelearning/exelearning` v4.0.5. Su código, AGPL, permanece en el contenedor; no se incorpora al repositorio.

## Comprobar la entrega

```sh
TMPDIR=/dev/shm/etf/tmp python3 -B curso/comprobar.py --importar
make ci
```

`comprobar.py` usa Playwright con `/usr/bin/google-chrome`. Abre todas las páginas mediante `file://`, responde correctamente a cada autoevaluación, comprueba vídeos e imágenes, verifica el pie y guarda capturas a 390 y 1440 en `docs/capturas/curso-*.png`. Comprueba el envío de una nota SCORM contra un LMS simulado local; no sustituye la prueba en la plataforma de formación de destino.

Con `--importar`, comprueba también el número de páginas, autor, licencia y una autoevaluación dentro del editor. Lee las credenciales sólo para iniciar sesión, desde el archivo externo de administración. No guarda cookies, contraseñas, capturas del acceso ni trazas de sesión. Sin esa opción, comprueba las exportaciones sin leer credenciales.
