# Revisión de los vídeos 06 y 07

14 de octubre de 2026. Chrome del sistema, pantalla de 1440 × 900, voz `es-ES-AlvaroNeural` y textos en castellano. Todas las cuentas, familias y fechas pertenecen a la demostración inventada.

`make videos` terminó y generó los seis recorridos disponibles. Después de revisar los fotogramas se regeneraron 06 y 07 con `make video RECORRIDO=06-spaf` y `make video RECORRIDO=07-cierre`. Se separó el retorno a la tabla de la descarga y se recreó únicamente la demo de este worktree para cargar los idiomas de `main` actual. Las grabaciones anteriores se conservan fuera del repositorio.

## Archivos definitivos

| Recorrido | MP4 | Duración | Título | Introducción | Cierre |
| --- | --- | --- | --- | --- | --- |
| 06 · Preparar la Tabla SPAF del mes | `videos/06-spaf-20261014-030314-619796/video.mp4` | 60.40 s | 3.50 s | 13.76 s | 4.56 s |
| 07 · Cerrar un expediente | `videos/07-cierre-20261014-030439-963288/video.mp4` | 116.96 s | 3.50 s | 13.84 s | 5.28 s |

Ambos MP4 tienen imagen H.264 y audio AAC. Cada carpeta contiene `hoja-control.png`, los Word descargados, `estado.json`, los intervalos de las escenas y `revision-vid4/` con los fotogramas extraídos del MP4 y su relación de tiempos.

## Fotogramas revisados

Se extrajeron y revisaron los segundos 1, 3, 6 y 10, la mitad de cada paso y la mitad del cierre. La extracción usa `ffmpeg -ss INSTANTE -i video.mp4 -frames:v 1`; `ffprobe` mide las duraciones. El título permanece 3,5 segundos; después se ven pantallas reales con movimiento. Los pasos tienen ratón visible y rótulos distintos de la voz. El cierre se superpone a la última pantalla difuminada.

| Vídeo | Mitad de cada paso, en segundos |
| --- | --- |
| 06 | abrir: 16.52; periodo: 23.24; movimientos: 32.10; alta: 41.72; volver: 48.78; word: 53.48 |
| 07 | ficha: 18.28; motivo: 39.80; anexo: 59.74; proponer: 66.28; ines: 75.62; javier: 87.52; cerrar: 100.08; acta: 109.04 |

En 06, el retorno conserva equipo y mes y muestra las cifras antes de pasar a la descarga. El siguiente paso muestra el aviso de descarga, con el control y el resultado visibles. En 07, se ven los datos de finalización, la emisión del Anexo 11, la propuesta, las cuentas de Inés y Javier, el diálogo de cierre y el aviso de Acta descargada. Las pestañas y los avisos de la ficha tienen sus textos completos.

## Comprobaciones reales

```text
SPAF-VIDEO-OK: 6 pasos; alta y baja nuevas; Word=true
CIERRE-VIDEO-OK: 8 pasos; expediente nuevo; cerrado=true; Acta=true
Ran 14 tests in 0.259s
OK
Ran 2 rules on 4 files: 0 findings.
```

`go vet ./...` y `go test ./...` terminaron con código 0. `make ci` se ejecutó completo sobre `main` integrado, incluidos frontend, textos, arquitectura, ensayos PostgreSQL y recorrido de permisos con Chrome. Su última línea fue:

```text
CI mínima completada.
```

La preparación de 06 añade dos casos del último mes terminado y compara las cifras antes y después: dos altas, una baja y dos casos más. La ejecución definitiva muestra 4 altas, 3 bajas y 5 casos en septiembre de 2026; abre `02-03-26`. Se probaron tanto el aviso de primera emisión como el aviso de datos cambiados en grabaciones anteriores. Cada ejecución busca números libres y conserva los registros anteriores.

07 acepta un expediente nuevo del ETF 02, `02-05-26` en la ejecución definitiva. Marta guarda y propone; Inés consulta la propuesta y Javier confirma el cierre. La automatización compara los identificadores de quien propuso y quien confirma para comprobar que son personas distintas. Conserva el cierre y las emisiones; otra ejecución crea otro expediente.

Los tres DOCX pasan la comprobación del ZIP y del XML de `word/document.xml`: `tabla-spaf.docx` contiene el número consultado, y `anexo11.docx` y `acta-cierre.docx` contienen el número del expediente cerrado. Las descargas comprueban el recibo de emisión y que no haya error. Las escrituras sin confirmación se detienen y no se reintentan.

## Alcance y límite

La ficha todavía no ofrece un botón para generar los Anexos 11/12. En 07, el Anexo 11 se genera por la API local desde la sesión profesional, después de guardar la finalización, y se consulta su emisión real en Documentos. El vídeo acredita esa consulta. La propuesta, el cierre y la descarga del Acta sí se realizan mediante los controles de la aplicación.

Los guiones y la herramienta cambian en `docs/recorridos/guiones/06-spaf.md`, `07-cierre.md` y `scripts/recorridos/{grabar,datos-video,tabla-video,cierre-video}.mjs`. Se actualizan `docs/recorridos/LEEME.md`, `README.md`, `docs/README.md` y `docs/producto/PLAN_IMPLEMENTACION.md`. La revisión local de seguridad y Semgrep se limita a esos scripts; no se ha ejecutado una auditoría completa.
