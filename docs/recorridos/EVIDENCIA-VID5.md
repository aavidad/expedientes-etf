# Revisión de los vídeos 05, 06 y 08

14 de octubre de 2026. Chrome del sistema, pantalla de 1440 × 900, voz `es-ES-AlvaroNeural` y castellano. Las cuentas, familias y fechas pertenecen a la demostración inventada.

05 y 06 se grabaron con `make videos RECORRIDOS="05-ficha 06-spaf 08-traslado"`. El montaje rechazó las frases de 08 que no daban tiempo a escribir o entrar. Se ampliaron esas locuciones y se regrabó con `make videos RECORRIDOS=08-traslado`, que terminó correctamente. Se mantuvo el control del montaje que impide acelerar una acción. Las grabaciones anteriores se conservan fuera de Git.

## Archivos definitivos

| Recorrido | MP4 | Duración | Título | Introducción | Cierre |
| --- | --- | --- | --- | --- | --- |
| 05 | `videos/05-ficha-20261014-041347-239820/video.mp4` | 74.48 s | 3.50 s | 13.28 s | 4.80 s |
| 06 | `videos/06-spaf-20261014-041514-806447/video.mp4` | 61.68 s | 3.50 s | 13.76 s | 4.56 s |
| 08 | `videos/08-traslado-20261014-042408-212481/video.mp4` | 101.80 s | 3.50 s | 14.84 s | 4.88 s |

Los tres MP4 contienen imagen H.264 y audio AAC. Cada carpeta conserva el guion, el WebM original, los tiempos de las escenas, `plan.json`, `estado.json`, `hoja-control.png` y `revision-vid5/` con los fotogramas extraídos del MP4 y su relación de tiempos. 05 incluye `anexo1.docx`; 06, `tabla-spaf.docx`.

## Fotogramas y revisión independiente

Se extrajeron los segundos 1, 3, 6 y 10, la mitad de cada paso y la mitad del cierre mediante `ffmpeg -ss INSTANTE -i video.mp4 -frames:v 1`. Las duraciones se midieron con `ffprobe`. La revisión principal y la revisión independiente confirmaron el título durante 3,5 segundos, la introducción sobre pantallas reales, el ratón y el foco visibles, los rótulos breves distintos de la voz y el cierre sobre la última pantalla difuminada. Los controles narrados quedan visibles.

| Vídeo | Mitad de cada paso, en segundos |
| --- | --- |
| 05 | bandeja: 15.18; ficha: 21.68; menores: 30.30; fase: 37.98; anexo: 45.66; firma: 53.28; evaluacion: 63.28 |
| 06 | abrir: 16.52; periodo: 23.24; movimientos: 32.74; alta: 43.00; volver: 50.06; word: 54.76 |
| 08 | abrir: 17.38; destino: 24.60; trasladar: 33.72; aceptacion: 50.08; aceptar: 66.12; equipo: 76.28; origen: 89.62 |

08 incluye tres fotogramas adicionales de resultado: número aceptado, a los 67,92 segundos; ficha de destino, a los 80,20; y seguimiento activo, a los 94,82. La concordancia de la voz se contrastó con las frases y los tiempos del plan y las pantallas. No se hizo una escucha manual completa del audio.

## Resultados comprobados

05 crea `02-06-26`. Marta abre la bandeja del ETF 02 y el expediente, revisa el aviso del nacimiento pendiente, completa ese dato y comprueba que desaparece el aviso. Guarda la fase de evaluación, descarga el Anexo 1 y confirma por separado la firma del consentimiento. Registra un instrumento inicial para la madre; el listado lo muestra sin puntuaciones ni baremos oficiales. Las API y los recibos confirman cada escritura.

06 muestra septiembre de 2026 para el ETF 02: 4 altas, 3 bajas, 5 casos y 5 menores. Hay cero casos sin dato de menores. La preparación añade dos casos históricos, uno cerrado, con un menor en cada uno desde la apertura; compara con las cifras anteriores y exige dos altas, una baja, dos casos y dos menores más. Consulta `02-07-26` y descarga el Word.

08 traslada `02-15-26` del ETF 02 al ETF 04. Javier prepara el caso nuevo de Reunificación y confirma el traslado; acepta la derivación enlazada con los datos necesarios y conserva el sexo de los menores del origen. Pablo ve `04-06-26` en la bandeja y la ficha del destino. Marta consulta el origen cerrado e inicia el seguimiento tras cese, que queda activo hasta `2027-10-14`.


Cada ejecución crea expedientes propios y conserva los registros anteriores. Los números históricos se eligen entre los libres; el traslado asigna el nuevo número al aceptar en destino. Las escrituras sin confirmación detienen el recorrido y no se reintentan. En las pruebas de preparación se conservaron también los expedientes de las grabaciones interrumpidas; no se usaron para repetir una escritura incierta.

Los Word pasan la comprobación del ZIP y del XML de `word/document.xml`. El de SPAF contiene los números de alta y baja preparados. Las descargas comprueban la emisión y la ausencia de error. La firma del consentimiento sólo se confirma mediante su fecha, después de descargar el Anexo 1.

## Pruebas y entrega

```text
FICHA-VIDEO-OK: 7 pasos; aviso desaparecido; Anexo 1, firma e instrumento confirmados
SPAF-VIDEO-OK: 6 pasos; alta y baja nuevas; Word=true
TRASLADO-VIDEO-OK: 7 pasos; número nuevo visto en destino; seguimiento activo en origen
Ran 15 tests in 0.248s
OK
Ran 3 rules on 7 files: 0 findings.
```

`go vet ./...` y `go test ./...` pasaron. La revisión independiente de los scripts y guiones no dejó hallazgos pendientes. Semgrep se ejecutó con reglas locales y métricas desactivadas. No se hizo una auditoría completa.

`make ci` se ejecutó completo sobre `main` integrado (`f3bb312`), con Go, frontend, textos, arquitectura, ensayos PostgreSQL, matriz de permisos con Chrome y recorrido de carga de 2026. Su última línea fue:

```text
CI mínima completada.
```

Las capturas de carga de 2026 que regeneró la CI se descartaron de esta entrega; se conservaron las versiones integradas desde `main`. Los temporales propios de VID5 y sus ejecuciones de CI se eliminaron de `bin/temporales`.

## Alcance y ficheros

05 registra un instrumento; no publica un cuestionario oficial ni calcula un baremo. En 08, la aceptación inicial, el consentimiento y la propuesta se preparan por API local; el traslado, la aceptación en destino y el seguimiento se realizan con los controles de la aplicación. El seguimiento tras cese se muestra en Reunificación, conforme al PRD.

La selección `RECORRIDOS` de `make videos` permite grabar un lote con un solo arranque de demo; sin ese argumento sigue grabando todos los guiones. Se mantiene el límite de accesos de la demo: no deben ejecutarse a la vez vídeos y CI, y un HTTP 429 exige esperar la ventana indicada en la guía.

Cambios propios: `scripts/recorridos/{ficha-video,traslado-video,tabla-video,datos-video,grabar}.mjs`, `videos.py`, `test_recorridos.py`, `Makefile`, los guiones 05/06/08, esta evidencia, `docs/recorridos/LEEME.md`, `README.md`, `docs/README.md` y `docs/producto/PLAN_IMPLEMENTACION.md`. No se añadieron migraciones ni se cambiaron pantallas o permisos. Los cambios de `main` se integraron conservando ambos lados.
