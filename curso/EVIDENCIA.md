# CUR2: curso actualizado a la aplicación del 14 de octubre de 2026

Comprobado en `/dev/shm/etf/wt-CUR`, rama `curso-exelearning`, con eXeLearning 4.0.5 del contenedor `etf-curso-exelearning-1` y Chrome del sistema. El contenedor se ha mantenido en marcha.

## Fuentes y alcance

Se fusionó `integracion` en el commit `c6fb7c4b`, con el corte `870ff2ae`. La resolución de la fusión conserva el objetivo de construcción del curso y el empaquetado Windows en `Makefile`, y regenera el índice de producto. Esos dos ficheros sólo se modificaron para resolver la fusión solicitada.

Los cambios propios del curso se limitan a `curso/contenido/`, `curso/PENDIENTE_VALIDAR.md` y esta evidencia. No se modifica el constructor ni se implementa el lector de voz.

M4, M5 y M9 incorporan lectura de anexos DOCX, ODT y PDF firmado; actas históricas de aceptación y cierre; plazo, coincidencias y anulación de lotes; escalas opcionales; reglas de espera, municipio y SMI; numeración vigente; Anexo 6; desglose de Memoria; privacidad de Consulta; caducidad de sesión y tareas de Administración. Las prácticas adicionales utilizan expedientes inventados separados del caso principal.

Fuentes contrastadas por apartado: manuales de profesionales, Coordinación y Administración; PRD; `docs/dudas.md`; evidencias RESP1, RESP3, RESP5, SEG1 y SEG3; recorrido RESP4; apartado de restauración del manual de sistemas para la mención de SEG4. Se mantiene una diferencia documental: el PRD menciona D23 pendiente, mientras `docs/dudas.md` no contiene su apartado. Se señala en el inventario pendiente sin alterar la regla vigente.

## Páginas y preguntas

| Módulo | Páginas exportadas, incluida su portada | Preguntas |
| --- | --- | --- |
| M0: presentación | 5 | 5 |
| M4: aplicación por perfiles | 14 | 14 |
| M5: caso práctico | 10 | 14 |
| M9: glosario y preguntas frecuentes | 4 | 14 |
| Total | 33 | 47 |

Se conserva el formato común de `curso/LEEME.md`: cabecera YAML, secciones Markdown, soluciones desplegables y autoevaluaciones nativas. Las capturas enlazadas ya existían y llevan texto alternativo. Se mantienen los doce archivos de vídeo existentes, incluido el recorrido completo en sus dos versiones.

## Comprobaciones ejecutadas

`TMPDIR=/dev/shm/etf/tmp make curso`, código 0:

```text
Curso: 4 módulos, 33 páginas, 47 preguntas.
Web: bin/curso/web/index.html; vídeos incluidos: 12
SCORM 1.2: bin/curso/scorm12.zip
Validaciones pendientes: 3 (bin/curso/resumen.json)
```

Se generan también `bin/curso/expedientes-etf.elpx` y el inventario `bin/curso/resumen.json`. Los resultados quedan fuera de Git.

`TMPDIR=/dev/shm/etf/tmp python3 -B curso/comprobar.py`, código 0:

```text
SCORM_METADATA_OK: autor Alberto Avidad; derechos CC BY-SA 4.0
OFFLINE_OK file://: 33 páginas; 4 autoevaluaciones; 47 respuestas correctas; 0 errores de consola
VIDEOS_OK: 12 vídeos con metadatos cargados sin conexión
CAPTURAS_OK: curso-1440.png; curso-390.png; sin desbordamiento horizontal
SCORM_NOTA_OK: cmi.core.score.raw=100 (LMS simulado)
```

El comprobador genera automáticamente dos capturas del curso. Se restauraron sus versiones de Git: no se entregan capturas nuevas. No se ejecutó la opción de importar al editor ni se repitieron ensayos del constructor ajenos al contenido.

`TMPDIR=/dev/shm/etf/tmp python3 -B scripts/ci/enlaces.py`, código 0:

```text
Markdown: 222 documentos, 1286 enlaces locales, 21 enlaces externos sin consulta de red, 0 errores.
```

`TMPDIR=/dev/shm/etf/tmp python3 scripts/ci/indice/generar.py --comprobar`, código 0:

```text
INDICE_OK
```

`git diff --check`, código 0, sin salida. No se ejecuta `make ci` completo, conforme al encargo.

## Revisión y pendientes

Revisión editorial independiente, sólo lectura, con humanizer, usabilidad-vec y revisar-cambios-vec: GO sobre los contenidos y los tres cuestionarios de 14 preguntas. No acredita una revisión del código de la aplicación ni sustituye la validación del Programa.

El inventario generado contiene tres marcas: D25/D26 en documentos de M4, D24 y apartado G en Memoria de M4, y D24 en Memoria de M5. `PENDIENTE_VALIDAR.md` conserva además D5, la referencia numérica de las actas D18, la diferencia sobre D23 y la revisión de curso y vídeos por el Programa. Las antiguas marcas de ZTS y hoja provisional se retiraron por estar resueltas.

No se modifica documentación de producto fuera de la resolución del índice durante la fusión. No se publica el curso, no se hace push ni se crea o fusiona una PR. Queda pendiente la validación del Programa y la prueba del paquete SCORM en la plataforma de formación de destino; la nota se ha probado con un LMS simulado local.
