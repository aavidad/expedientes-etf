# Vídeos de recorridos con voz

Esta herramienta graba recorridos de demostración con Chrome, añade una introducción, explica los pasos con rótulos y voz y termina con una cartela de cierre. Los guiones están en [guiones/](guiones/); incluyen [01-entrada](guiones/01-entrada.md) y [02-coordinacion](guiones/02-coordinacion.md). Para preparar otro, usa la [plantilla](PLANTILLA.md).

Los vídeos y todos los ficheros de trabajo se guardan fuera del repositorio, en la carpeta `ETF_VIDEOS_SALIDA` (por defecto `~/Videos/Expedientes-ETF`). Cada ejecución crea una carpeta nueva, por lo que conserva las grabaciones anteriores. `ETF_VIDEOS_SALIDA` permite elegir otra carpeta, siempre fuera del repositorio. `.gitignore` excluye también los formatos de vídeo y las carpetas locales de herramientas.

## Preparar las herramientas

Necesitas Python 3 con su módulo `venv`, Node y npm de las versiones del proyecto, Chrome del sistema, `ffprobe` y un `ffmpeg` que tenga los codificadores H.264 (`libx264`) y AAC. Instala las dependencias fijadas de la interfaz con `npm ci --ignore-scripts --prefix web`; se usa su `playwright-core`, sin descargar otro navegador.

Para grabar, Playwright usa su ejecutable de ffmpeg de `~/.cache/ms-playwright/`. Si falta, instala sólo ese componente con `web/node_modules/.bin/playwright-core install ffmpeg`. Para montar el MP4 se busca primero el ffmpeg de esa caché. Algunas versiones sólo codifican VP8: en ese caso se usa el ffmpeg del sistema. Puedes elegir uno con `ETF_VIDEO_FFMPEG`. `ETF_VIDEO_CHROME` cambia la ruta del Chrome del sistema; por defecto es `/usr/bin/google-chrome`.

La voz se instala en un entorno separado dentro de la carpeta de salida:

```sh
bash scripts/recorridos/preparar_voz.sh
```

Se fija el arreglo del repositorio oficial de edge-tts, commit `0b1c544d020b01338ae491d7f4547729afdc39d3`. La versión 7.2.8 publicada en PyPI daba un error 403 en el kit de referencia. No se altera ese kit ni el Python del sistema. Si ya tienes la biblioteca corregida en un kit local, puedes reutilizarla en lectura con `ETF_EDGE_LIB=/ruta/al/kit/voz/lib`; se comprueba su commit. La voz es `es-ES-AlvaroNeural`, con velocidad `+4%`. Las frases se guardan en una caché local por texto, voz, velocidad y commit; una repetición no vuelve a solicitar las mismas frases.

Sólo se envía a Microsoft el texto inventado de la locución, si la entidad autoriza ese servicio. No se envían vídeos, capturas, cuentas, contraseñas, cookies ni expedientes. Usa siempre guiones sin datos personales reales. No ejecutes `make videos`, `make video` ni `make demo` a la vez que `make ci`: reinstalan las mismas dependencias de la interfaz. Espera a que termine también la hoja de control antes de arrancar otro vídeo. No ejecutes a la vez vídeos y comprobaciones de acceso: comparten el límite de entradas de la demostración (cinco por cuenta en quince minutos). Si aparece HTTP 429, espera a que termine esa ventana antes de volver a grabar. El navegador sólo admite peticiones al origen local de la demostración y no guarda HAR ni sesiones.

## Arrancar y grabar

```sh
make videos
make videos RECORRIDOS="05-ficha 06-spaf 08-traslado"
make videos RECORRIDOS="09-memoria 10-carga-2026"
make video RECORRIDO=01-entrada
```

`make videos RECORRIDOS="05-ficha 06-spaf 08-traslado"` selecciona esos tres recorridos, en ese orden, con un solo arranque de la demo. Sin `RECORRIDOS`, graba todos. No repitas un identificador en la misma ejecución.

La herramienta ejecuta `make demo` antes de grabar y espera a que los servicios estén listos. Lee el puerto y las contraseñas inventadas de los ficheros locales que prepara la demostración, sin copiarlos al guion ni al vídeo. El recorrido usa a Sergio Campos, pendiente de alta, y a Elena Salas, que administra los accesos, y asigna el equipo 2 y el puesto de trabajo social. No necesitas indicar esas cuentas para el recorrido incluido. Si el puerto habitual está ocupado, elige otro en el primer arranque con `ETF_DEMO_PUERTO=18491 make video RECORRIDO=01-entrada`; la demo conserva ese puerto.

Si trabajas con otra instalación local de demostración y no existe `make demo`, el arranque es manual. Sigue [la guía de desarrollo](../../despliegue/base/LEEME.md) para una instalación local aislada, aplica las migraciones con el ejecutor separado y configura un directorio de usuarios de prueba y cuentas inventadas. El Compose base por sí solo no aporta ese directorio ni las cuentas necesarias para este recorrido. Si no están disponibles, usa la prueba sintética del apartado siguiente.

En una demo preparada, indica `ETF_VIDEO_URL` con su origen local, sin ruta ni credenciales (por ejemplo, `http://127.0.0.1:18083`), y `ETF_VIDEO_DEMO=1`. Da las cuentas existentes mediante `ETF_VIDEO_PROFESIONAL` y `ETF_VIDEO_ADMINISTRACION`, sus contraseñas inventadas mediante `ETF_VIDEO_CLAVE_PROFESIONAL` y `ETF_VIDEO_CLAVE_ADMINISTRACION`, y el identificador de un equipo de la demo mediante `ETF_VIDEO_EQUIPO_ID`. Carga estas variables desde un fichero local fuera de Git o introdúcelas en la terminal. No pongas contraseñas en el guion ni en parámetros de línea de comandos.

Las variables anteriores permiten sustituir las cuentas del guion por otras cuentas inventadas de la misma demostración. Si actualizas la rama y aparecen botones o cabeceras vacíos, los volúmenes de idioma de la demo pueden conservar la versión anterior. Sigue la guía de demostración para recrear únicamente ese entorno con `make demo-limpiar`: borra sus datos inventados y sus contraseñas. La herramienta de vídeos nunca lo hace automáticamente.

La cuenta profesional debe existir, estar pendiente de alta o activa sin perfiles y no tener equipo. La herramienta lo comprueba antes de grabar; no crea usuarios. Administración debe tener permiso para gestionar usuarios. El recorrido concede el perfil y muestra la confirmación. Después retira la concesión para permitir repetir el vídeo: la cuenta queda activa sin perfil ni equipo y su próximo acceso queda pendiente. Se conservan los apuntes de auditoría y aumenta su versión. Sólo retira el perfil si el servidor ha confirmado que guardó nuestra asignación. Si guardar devuelve un error o no llega esa confirmación, conserva los permisos y se detiene para revisar la demostración. Si la cuenta cambia mientras se graba, la herramienta falla y conserva los permisos para que una persona los revise. Si interrumpes el proceso de forma forzada, comprueba esa cuenta antes de repetirlo. `estado.json` informa de si llegó a escribir y si restauró la demo.

Los textos de pantalla que busca la automatización proceden de `idiomas/es/textos/`. Graba con el idioma castellano de la demo. Los rótulos y cartelas toman los tokens y las fuentes del tema activo de la página; no imponen un tema ni colores propios.

## Recibir una derivación y aceptarla

El [recorrido 02](guiones/02-coordinacion.md) entra con Inés Moreno Castillo, muestra las derivaciones pendientes, sube el Anexo 3 inventado de pruebas y revisa la lectura y la ubicación propuesta. Confirma la derivación, completa los datos de aceptación, obtiene el número de expediente y descarga el Acta en Word. Termina buscando ese número en «Aceptados recientes».

```sh
make videos
make video RECORRIDO=02-coordinacion
```

Cada grabación usa una derivación nueva. Conserva el expediente aceptado, su número y la emisión del Acta; repetir avanza el correlativo sin modificar los expedientes anteriores. El Acta inventada queda junto al vídeo como `acta-aceptacion.docx`. `estado.json` identifica la derivación creada, el número y la descarga confirmada. Si una escritura no se confirma, el recorrido se detiene sin repetirla.

En un arranque manual, usa `ETF_VIDEO_COORDINACION` y `ETF_VIDEO_CLAVE_COORDINACION` con una cuenta inventada de Coordinación. `make videos` obtiene la cuenta y su contraseña del entorno local de demostración. El Anexo está limitado al fichero versionado `internal/adaptadores/anexos/docx/testdata/anexo3.docx`; no carga originales del cliente. El municipio, centro y equipo que se ven son los propuestos por los catálogos vigentes.

La grabación comprueba una lectura con antivirus limpio, las respuestas de creación y aceptación, el formato del número, la descarga del Word y la fila aceptada que corresponde a ese expediente. Revisa la hoja de control antes de darla por terminada. Al acabar, para la demo con `make demo-parar`.

## Registrar comisiones e informes

El [recorrido 03](guiones/03-actividad.md) entra con Marta Cabrera, profesional del ETF 02. Registra una comisión con fecha y observaciones inventadas y un informe técnico emitido para Juzgados. Corrige la fecha de ese informe y comprueba el listado. Después entra Inés Moreno Castillo, de Coordinación, y consulta la comisión y el informe con el filtro «Todos los equipos».

```sh
make video RECORRIDO=03-actividad
```

Cada grabación conserva una comisión y un informe nuevos. La corrección se aplica sólo al identificador creado durante esa ejecución. Las respuestas de escritura, la versión corregida y la consulta provincial quedan comprobadas en `estado.json`. Una escritura sin confirmación detiene el recorrido sin repetirla. Para un arranque manual, las variables son `ETF_VIDEO_ACTIVIDAD_PROFESIONAL`, `ETF_VIDEO_CLAVE_ACTIVIDAD_PROFESIONAL`, `ETF_VIDEO_COORDINACION` y `ETF_VIDEO_CLAVE_COORDINACION`.

## Publicar una plantilla Word

El [recorrido 04](guiones/04-plantillas.md) entra con Elena Salas, de Administración. Consulta la versión activa y el historial del Acta de Aceptación en castellano. Sube un Word con el marcador añadido `intruso`, comprueba el aviso que pide retirarlo o corregirlo y sube el Word correcto. Activa la nueva versión mediante el diálogo de confirmación y muestra su autoría y fecha.

```sh
make video RECORRIDO=04-plantillas
```

Los dos Word se preparan en la carpeta del vídeo a partir de la plantilla inventada versionada en el proyecto. El archivo rechazado conserva los marcadores válidos y añade sólo el desconocido. La grabación comprueba que el rechazo no crea una versión ni cambia la activa, que el Word correcto supera el antivirus y queda pendiente, y que la activación registra la persona y la fecha. Conserva cada versión y su historial; al repetir elige un número `3.n.0` libre. La nueva versión queda activa al terminar. No se reintentan escrituras sin confirmación ni conflictos. En arranque manual se usan `ETF_VIDEO_ADMINISTRACION` y `ETF_VIDEO_CLAVE_ADMINISTRACION`.

`make videos` graba todos los guiones disponibles en `guiones/`. La variable opcional `RECORRIDOS` limita la tanda a los identificadores indicados, separados por espacios, sin cambiar el comportamiento por defecto. Las cuentas profesionales de 01 y 03 tienen variables distintas: el primero usa una cuenta pendiente de alta; el tercero, una profesional activa del equipo dos. Las contraseñas siguen fuera de los guiones y las grabaciones.

## Preparar la Tabla SPAF del mes

El [recorrido 06](guiones/06-spaf.md) entra con Inés Moreno, elige el ETF 02 y el último mes terminado, revisa el resumen y los movimientos, abre un expediente que cuenta como alta y descarga la tabla en Word.

```sh
make video RECORRIDO=06-spaf
```

Antes de grabar las escenas, prepara dos casos históricos nuevos por la API de la demo: ambos cuentan como alta y uno también como baja. Busca números libres y comprueba que las cifras aumentan en dos altas, una baja y dos casos y dos menores. Cada caso incorpora la ficha histórica de un menor desde la apertura. Comprueba también el incremento del total de menores y que los casos nuevos no añaden cifras incompletas. Conserva los registros y la emisión; repetir no modifica casos anteriores. Guarda `tabla-spaf.docx` y las cifras e identificadores en `estado.json`. En arranque manual usa las variables de Coordinación.

## Cerrar un expediente

El [recorrido 07](guiones/07-cierre.md) muestra a Marta Cabrera en la pestaña Cierre de un expediente nuevo: selecciona la falta de consentimiento, guarda la finalización y propone el cierre. En Documentos consulta la emisión del Anexo 11. Inés Moreno localiza la propuesta en Cierres propuestos. Javier Medina entra con su propia cuenta de Coordinación, confirma la fecha y el cierre y descarga el Acta.

```sh
make video RECORRIDO=07-cierre
```

La preparación acepta un expediente nuevo del ETF 02 con el Anexo 3 inventado versionado. El caso sin consentimiento permite proponer desde ACEPTADA sin registrar una intervención. Conserva el expediente, su número y sus emisiones; no reabre ni altera cierres anteriores. Guarda `anexo11.docx`, `acta-cierre.docx` y los recibos confirmados en `estado.json`.

La ficha todavía no tiene botón para generar los Anexos 11/12. La herramienta genera el Anexo 11 por la API local después de guardar la finalización y muestra su emisión real en Documentos. El vídeo acredita esa consulta; no representa una generación mediante un botón. La propuesta, el cierre y la descarga del Acta sí se realizan con los controles de la aplicación.

En arranque manual se usan las variables de profesional de actividad y de Coordinación, más `ETF_VIDEO_SEGUNDA_COORDINACION` y `ETF_VIDEO_CLAVE_SEGUNDA_COORDINACION` para Javier. Las contraseñas siguen fuera del guion y de las grabaciones. Cada escritura se envía una vez y se detiene el recorrido si no hay confirmación.

La [revisión de los vídeos 06 y 07](EVIDENCIA-06-07.md) recoge los archivos definitivos, las duraciones, los fotogramas y el límite de generación del Anexo.

## Probar el montaje sin la demo

```sh
python3 -B scripts/recorridos/videos.py --validar
python3 -B scripts/recorridos/videos.py --recorrido 01-entrada --sintetico
```

Aunque la demo esté disponible, la segunda orden permite probar las herramientas de forma aislada. Graba una cartela de prueba y todos los rótulos del guion, añade voz real y monta el MP4. Estas imágenes prueban las herramientas; no acreditan un recorrido realizado en la aplicación. La carpeta lleva el prefijo `sintetico-`. Usa los tokens de `temas/expedientes-etf/`; prueba otro tema con `ETF_VIDEO_TEMA=alto-contraste`.

Para comprobar el montaje sin conexión ni servicio de voz, añade `--sin-voz`. En ese modo la pista es silencio de prueba: el resultado no es un vídeo narrado. La introducción, las fases de acción y los tiempos de las frases se prueban con duraciones simuladas.

```sh
python3 -B -m unittest discover -s scripts/recorridos -p 'test_*.py'
```

## Revisar el resultado

Abre `video.mp4` y escucha la introducción, cada paso y el cierre. Revisa `hoja-control.png`: muestra la portada, un fotograma durante cada frase y el cierre, con el texto y su instante. Comprueba que la frase describe lo que aparece y que el rótulo no tapa el control que se usa.

Cada carpeta conserva el guion, el WebM original, una captura real por paso y `ultima.png`, sin rótulo, para el cierre. `escenas.json` contiene los intervalos grabados de cada acción; un marcador visual inicial fija su correspondencia con el WebM en `alineacion.json`, y `escenas-alineadas.json` conserva los tiempos corregidos; `plan.json` relaciona esos intervalos con las frases y sus instantes en el MP4. La herramienta prepara y mide la voz antes de grabar. Durante la grabación realiza una acción por oración, mantiene el resultado hasta terminar esa frase y pasa a la siguiente. El montaje ajusta la duración de cada fase, sin adelantar la voz respecto a la acción ni conservar tiempos ajenos al paso. El ratón queda visible y el elemento pulsado se resalta con los tokens del tema.

La introducción ocupa 12–18 segundos: título y cartela durante 3,5 segundos, seguidos de capturas reales del recorrido con zoom y desplazamiento suaves y transiciones. La locución cuenta de qué trata, quién lo hace y para qué sirve; no enumera los pasos. El cierre dura 4–6 segundos y superpone la cartela a la última pantalla real difuminada. El montaje rechaza una voz que exceda esos límites.

Los rótulos inferiores resumen cada paso en 4–8 palabras. La voz describe lo que se ve en ese momento. Revisa también los fotogramas extraídos con ffmpeg en los segundos 1, 3, 6 y 10 y a mitad de cada paso; las imágenes de la hoja de control por sí solas no prueban que las acciones estén sincronizadas.

El kit de recorridos usa únicamente cuentas y datos inventados de la demostración local.

La revisión de los vídeos 03 y 04, sus rutas y sus duraciones están en [EVIDENCIA-03-04.md](EVIDENCIA-03-04.md).

## Trabajar la ficha del expediente

El [recorrido 05](guiones/05-ficha.md) muestra a Marta Cabrera en la bandeja del ETF 02. Abre un expediente nuevo, revisa sus avisos, completa el nacimiento del primer menor y comprueba que desaparece ese aviso. Guarda la fase de evaluación en Intervención, descarga el Anexo 1 en Consentimiento y confirma por separado la fecha de firma. Termina registrando un instrumento inicial para la madre en Evaluación.

```sh
make video RECORRIDO=05-ficha
```

La preparación acepta un expediente nuevo y completa datos inventados de menores, dejando pendiente sólo el nacimiento del primero entre esos datos. Si falta una figura parental, la añade para poder registrar el instrumento. Conserva el expediente y sus cambios; no altera casos anteriores. Guarda `anexo1.docx` y las confirmaciones en `estado.json`. El registro del instrumento no publica el cuestionario oficial ni calcula puntuaciones o baremos. Usa las variables de profesional de actividad y Coordinación.

## Trasladar un expediente a otro equipo

El [recorrido 08](guiones/08-traslado.md) muestra el traslado de un expediente de Reunificación del ETF 02 al ETF 04. Javier acepta el caso nuevo y Marta prepara la propuesta de cierre antes de grabar las escenas. Javier confirma el traslado y acepta la derivación enlazada para asignarle número nuevo. Pablo Ruiz, profesional del destino, consulta ese número en su bandeja y ficha. Marta consulta el origen e inicia el seguimiento tras cese, con su fecha prevista de fin.

```sh
make video RECORRIDO=08-traslado
```

Cada ejecución acepta un expediente nuevo con el Anexo 4 inventado versionado y conserva el origen cerrado, el destino aceptado y el seguimiento. Sólo Reunificación admite seguimiento tras cese. El consentimiento y la propuesta se preparan por API local; el traslado, la aceptación y el seguimiento se realizan con los controles reales. No se reintenta ninguna escritura sin confirmación. Usa las variables de profesional de actividad, `ETF_VIDEO_TRASLADO_COORDINACION` y `ETF_VIDEO_CLAVE_TRASLADO_COORDINACION` para Javier y `ETF_VIDEO_DESTINO` y `ETF_VIDEO_CLAVE_DESTINO` para Pablo. Las contraseñas proceden del entorno de demostración.

La [revisión de los vídeos 05, 06 y 08](EVIDENCIA-VID5.md) recoge las rutas definitivas, las duraciones, los fotogramas y los resultados comprobados.

## Revisar y cerrar la Memoria anual

El [recorrido 09](guiones/09-memoria.md) muestra a Marta Cabrera en la Memoria de 2026 del ETF 02: consulta los avisos, escribe la metodología, guarda, descarga el Word y cierra. Inés Moreno consulta la provincial, la cierra y la reabre indicando el motivo.

```sh
make video RECORRIDO=09-memoria
```

Repetir conserva las revisiones cerradas. Entre el cierre del equipo y la entrada de Coordinación hay una pausa de un minuto para renovar el presupuesto de peticiones por IP; queda fuera de las escenas y del MP4. Durante la preparación, Coordinación reabre las Memorias que sigan cerradas con el motivo del guion; el vídeo termina con la del equipo cerrada y la provincial abierta. Los avisos no se resuelven mediante esta grabación ni se presentan las cifras como validadas por el Programa. `estado.json` comprueba guardado, versiones, instantáneas y reapertura; `memoria-equipo.docx` conserva la descarga. Usa las variables de profesional de actividad y de Coordinación; las contraseñas siguen fuera de los guiones.

## Cargar los casos de 2026

El [recorrido 10](guiones/10-carga-2026.md) muestra a Inés Moreno cargando un caso a mano con su número y fechas históricos. Después descarga la hoja inventada, sube una copia con una fecha incorrecta, revisa la incidencia por fila, sube la corregida y confirma los tres casos del lote. Termina consultando los datos guardados del primero.

```sh
make video RECORRIDO=10-carga-2026
```

Cada ejecución busca tres números libres del ETF 02: uno para el alta manual y dos para los casos aceptados de la hoja. El no aceptado no recibe número de aceptación. Conserva cuatro casos nuevos y no modifica los anteriores. Junto al vídeo quedan `hoja-original.csv`, `carga-pendiente.csv` y `carga-2026.csv`. La copia preparada cambia sólo los números para repetir; la incorrecta añade una fecha pendiente. `estado.json` comprueba la incidencia, el bloqueo de confirmación, los recibos y la lectura posterior. Usa las variables de Coordinación y se detiene ante una escritura sin confirmación.

Las rutas, duraciones, fotogramas y comprobaciones de ambos recorridos se recogen en [EVIDENCIA-09-10.md](EVIDENCIA-09-10.md).

## Del Anexo a la Memoria

El [recorrido 11](guiones/11-ciclo-completo.md) sigue un único caso inventado de la familia Ortega Salcedo, con seis menores. Javier recibe el Anexo 3, revisa, acepta y descarga el Acta. Marta completa Familia, Menores e Intervención, descarga el Anexo 1, confirma consentimiento y registra un instrumento; anota comisión e informe del equipo y consulta la Tabla SPAF de septiembre. Prepara la finalización, descarga el Anexo 11 y propone; Javier revisa, confirma cierre y descarga el Acta. Al avanzar a noviembre, el caso aparece como alta y baja de octubre. Termina con apartados, narrativa, Word y cierre de Memorias: Marta cierra la del equipo e Inés, de Coordinación, la provincial.

```sh
make video RECORRIDO=11-ciclo-completo
```

Usa el calendario de formación desde el 14 de octubre hasta el 1 de noviembre de 2026. La compilación local mediante overlay simula los relojes de negocio; las sesiones siguen en tiempo real y los ficheros Go de producción conservan su código. Consulta septiembre antes del cierre y octubre después, con el alta y la baja del caso. Al terminar o fallar, el comando restaura la compilación ordinaria mediante `finally`; si el proceso se termina por la fuerza o falla la restauración, ejecuta `ETF_VIDEO_DEMO=1 python3 -B scripts/recorridos/reloj_video.py restaurar`. Conserva el número, los datos y las emisiones de cada ejecución. Si alguna Memoria sigue cerrada, Coordinación la reabre con motivo antes de las escenas, conservando su instantánea. Las tres cuentas deben ser las del guion y distintas. Javier recibe y confirma el cierre del caso; Inés cierra la provincial. Así se respetan también las cinco entradas por cuenta en quince minutos cuando el arranque acaba de preparar los usuarios. El MP4 incluye seis capítulos navegables con los tiempos reales del montaje. La pausa inicial renueva el presupuesto HTTP tras las comprobaciones del arranque y queda fuera del montaje. Los avisos pendientes permanecen visibles; las cifras no se presentan como validadas por el Programa.

Duración, capítulos, resultados y fotogramas en [EVIDENCIA-11.md](EVIDENCIA-11.md). Para impartir la sesión, usa [formación](../formacion/LEEME.md), que incluye ejercicios, soluciones, preparación y reinicio de la demo local.

Para practicar las fechas inventadas en el aula, inicia el calendario según [preparar el aula](../formacion/preparar-aula.md) y abre `ETF_VIDEO_DEMO=1 node scripts/recorridos/aula.mjs`. Sus tres ventanas mantienen sesiones separadas. Equipos y Coordinación usan el calendario del negocio; Administración conserva el reloj ordinario para gestionar permisos. Los temporizadores siguen en tiempo real. `--comprobar` verifica la fecha, el mes permitido en la pantalla y la consulta de la API local. Cierra las ventanas tras comprobar los Word y restaura la compilación ordinaria.
