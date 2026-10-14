# Revisión de los vídeos 09 y 10

14 de octubre de 2026. Chrome del sistema, 1440 × 900, voz castellana `es-ES-AlvaroNeural`. Cuentas, familias, números y fechas pertenecen a la demostración inventada.

## Archivos definitivos

| Recorrido | MP4 | Duración | Título | Introducción | Cierre |
| --- | --- | --- | --- | --- | --- |
| 09 | `videos/09-memoria-20261014-050829-601093/video.mp4` | 103.16 s | 3.50 s | 13.40 s | 5.48 s |
| 10 | `videos/10-carga-2026-20261014-051605-068465/video.mp4` | 128.52 s | 3.50 s | 15.32 s | 5.48 s |

Los dos MP4 tienen imagen H.264 y audio AAC. Cada carpeta conserva `hoja-control.png`, `estado.json`, el WebM original, las escenas, la alineación, el plan de montaje y `revision-vid6/` con los fotogramas y sus tiempos. La de 09 incluye `memoria-equipo.docx`; la de 10, `hoja-original.csv`, `carga-pendiente.csv` y `carga-2026.csv`.

## Fotogramas revisados

Se extrajeron con ffmpeg los segundos 1, 3, 6 y 10, la mitad de cada paso y la mitad del cierre. Se revisaron también el apartado con datos pendientes, el equipo cerrado y la provincial cargada en 09, y el alta manual confirmada y los datos guardados del lote en 10: dieciséis imágenes en 09 y quince en 10.

| Vídeo | Mitad de cada paso, en segundos |
| --- | --- |
| 09 | abrir: 18.06; avisos: 28.32; narrativa: 39.40; word: 47.78; equipo: 55.74; provincial: 66.50; cerrar: 77.36; reabrir: 90.10 |
| 10 | abrir: 20.08; origen: 32.44; familia: 48.74; manual: 62.48; hoja: 71.88; incidencia: 84.18; corregida: 99.78; confirmar: 115.24 |

El título permanece 3,5 segundos. La introducción continúa sobre capturas reales con movimiento y transiciones. Los pasos muestran ratón y rótulos distintos de la voz, con los controles usados visibles. El cierre se superpone a la última pantalla real difuminada.

En 09 se ven los avisos, la columna «Sin dato», la metodología escrita, el aviso de descarga, los diálogos de cierre, la consulta provincial y el motivo de reapertura. A mitad del cambio de perfil aparece la carga de la Memoria; el fotograma suplementario acredita la consulta provincial terminada. La pausa entre perfiles queda fuera del MP4.

En 10 se ven las fechas históricas, la revisión previa, la descarga de la hoja inventada, la incidencia de fila 4 sin confirmación disponible, los tres casos corregidos y el resultado guardado. El último tramo consulta el número y las fechas del primer caso del lote.

## Comprobaciones reales

Comandos de grabación: `make videos RECORRIDOS="09-memoria 10-carga-2026"` y, tras corregir la lectura de respuestas de Chrome, `make videos RECORRIDOS="10-carga-2026"`. El segundo terminó con código 0. Se conservaron las grabaciones anteriores y sus actos confirmados.

```text
MEMORIA-VIDEO-OK: 8 pasos; Word=true; equipo cerrado; provincial cerrada y reabierta con motivo
CARGA-VIDEO-OK: 8 pasos; alta manual y lote de tres; incidencias por fila; cuatro casos nuevos
Ran 15 tests in 0.248s
OK
RENUMERACION-OK: 91→92 y 92→93 sin colisiones
Ran 2 rules on 3 files: 0 findings.
F0.11-RLS-OK: 7 actores; 53 tablas pobladas; 371 comprobaciones de lectura; 795 escrituras denegadas.
F0.11-VISTAS-OK: 9 vistas; 63 lecturas con RLS.
```

El Word pasa la comprobación de ZIP y contiene la narrativa guardada. Las escrituras de Memoria tienen respuesta confirmada, versión e instantánea. El equipo conserva la revisión 3 cerrada y la provincial conserva la revisión 2 tras reabrirse. Las repeticiones anteriores siguen en la historia.

En 10, el alta manual conserva `02-14-26`, derivación del 08/01/2026 y aceptación del 15/01/2026. La hoja guarda `02-15-26`, `02-16-26` y un no aceptado sin número de aceptación. La copia inválida cambia sólo la fecha de derivación de fila 4 por `fecha-pendiente`. El informe la rechaza con HTTP 422; la corregida devuelve 200 y confirma tres identificadores. Se comprueban los estados y la marca de revisión al recuperar cada caso.

Chrome descartaba del inspector el cuerpo de algunas respuestas de subida. La herramienta conserva en memoria una copia JSON de la respuesta real del mismo origen y devuelve la original a la pantalla. Compara su estado con el observado por Playwright; no simula respuestas ni repite peticiones. Las contraseñas, cookies y sesiones no se copian a los artefactos.

La prueba RLS de F0.11 asumía que Marta no tenía actos ni revisiones de Memoria. Se adaptaron sus expectativas al equipo 02, sus recibos al actor y las pruebas negativas a una fila del equipo 04. Las discrepancias siguen siendo exclusivas de Coordinación. No se cambian migraciones, roles ni políticas. El ensayo completo usa PostgreSQL 18 local con datos inventados y termina en ROLLBACK. Revisión independiente focal de SQL y seguridad: GO estático; grabaciones y ensayos reales comprobados aparte.

## Integración y alcance

Se conservan las aportaciones de VID5 al integrar `main`, incluidos sus guiones 05, 06 y 08 y la selección de recorridos. Los cambios propios son los guiones 09/10, `scripts/recorridos/{memoria-video,carga-video,grabar}.mjs`, el ensayo `entrada_permisos_rls.sql`, esta evidencia, la guía de recorridos, ambos README y `PLAN_IMPLEMENTACION.md` con su índice regenerado.

Los avisos de Memoria no se resuelven al grabar ni se presentan las cifras como validadas por el Programa. La carga usa únicamente casos inventados; estos vídeos no autorizan introducir datos reales ni completan la formación de F5.07.

Se integró `main` hasta `7828247`, conservando ambos trabajos. `make ci` terminó con código 0 sobre la candidata `3f6d33c`, con `go vet ./...`, `go test ./...`, frontend, idiomas, arquitectura, ensayos PostgreSQL, matriz de permisos y carga real con Chrome. Su última línea fue:

```text
CI mínima completada.
```

La revisión focal independiente dio GO estático para `3f6d33c`. Semgrep local ejecutó dos reglas sobre los tres scripts cambiados, sin hallazgos. No se ejecutó una auditoría completa. Las grabaciones definitivas quedan fuera de Git; los guiones, automatizaciones y evidencias están en esta rama.
