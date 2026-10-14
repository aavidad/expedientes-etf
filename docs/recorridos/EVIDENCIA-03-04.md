# Vídeos de actividad y publicación de plantillas

Grabación final del 14 de octubre de 2026 en `vid3-videos-actividad-plantillas`. `make videos` terminó con código 0 para los cuatro recorridos, incluidas sus hojas de control. Se usaron Chrome del sistema, el idioma castellano y las cuentas inventadas de la demo local del puerto 18503.

## MP4 entregados

| Recorrido | Archivo fuera de Git | Duración | Título | Introducción | Cierre |
| --- | --- | --- | --- | --- | --- |
| 03 · Registrar una comisión técnica y un informe | `videos/03-actividad-20261014-011749-573675/video.mp4` | 96,96 s | 3,50 s | 14,24 s | 5,20 s |
| 04 · Publicar una plantilla Word nueva | `videos/04-plantillas-20261014-011934-224889/video.mp4` | 99,80 s | 3,50 s | 13,32 s | 5,72 s |

La misma tanda conserva `01-entrada-20261014-011517-686287/video.mp4` y `02-coordinacion-20261014-011623-481576/video.mp4` dentro de `videos/`. No se incorporan vídeos, capturas ni ficheros de trabajo a Git.

## Qué se comprobó

03 entra con Marta Cabrera Molina, del ETF 02, registra una comisión del 8 de octubre con observaciones inventadas y un informe para Juzgados del 9 de octubre. Corrige sólo ese informe al 10 de octubre y comprueba la versión 2 en el listado. Inés Moreno Castillo consulta la comisión y el informe con «Todos los equipos». Las respuestas de creación y corrección y los identificadores consultados quedan en `estado.json`.

04 entra con Elena Salas Márquez y consulta la versión activa y el historial del Acta de Aceptación en castellano. El Word con `intruso` recibe HTTP 422 y muestra el aviso que pide retirar o corregir ese marcador. Conserva todos los marcadores válidos; no se publica la versión rechazada ni cambia la activa. El Word correcto recibe HTTP 201, supera el antivirus y queda pendiente. La activación se confirma en el diálogo, recibe HTTP 200 y registra el nombre y la fecha. La versión `3.0.0` queda activa; se conservan las versiones anteriores.

Cada ejecución crea registros propios y elige una versión de plantilla libre. No reintenta escrituras sin confirmación. Los ensayos previos comprobaron la corrección y consulta con registros anteriores presentes; la tanda final se grabó en una demo recreada para la línea actual de migraciones. No se presenta la copia antigua como el estado de esta demo.

## Fotogramas y montaje

Se extrajeron directamente con ffmpeg y se revisaron individualmente los segundos 1, 3, 6 y 10, la mitad de cada paso y el cierre: once imágenes de 03 y doce de 04. Están junto a cada MP4 como `revision-*.png`; `revision.json` conserva sus instantes. También se conservan `hoja-control.png`, `control.json`, las capturas originales y `ultima.png`.

El título ocupa 3,5 segundos. La introducción continúa sobre capturas reales con movimiento y transiciones; el fotograma del segundo 10 de 04 muestra una transición entre dos capturas. Las frases describen la acción o el resultado de su fase. El ratón permanece visible y el rótulo resume el paso con una frase distinta. Se centró Observaciones antes de escribir para dejar el campo entero por encima del rótulo. El cierre usa la última pantalla real difuminada.

`escenas.json`, `alineacion.json`, `escenas-alineadas.json` y `plan.json` conservan el anclaje y los intervalos de cada frase. El montaje retira sólo esperas con fotogramas idénticos; no acelera escritura ni ratón. Ambos MP4 tienen vídeo H.264 de 1440 × 900 y audio AAC con señal. El barrido a 25 fotogramas por segundo encontró cero fotogramas uniformes: 2424 revisados en 03 y 2495 en 04.

La voz es `es-ES-AlvaroNeural`, con velocidad `+4%`. Sólo se envió el texto inventado de la locución al servicio autorizado de edge-tts. Las contraseñas aparecen enmascaradas.

## Integración y entorno

Los vídeos finales se grabaron tras integrar `main` en `f381b53`. Después se volvió a integrar `main` en `46fc142` antes de comprobar la entrega completa. El código propio de grabación mantiene el cambio revisado en `a52e484`; las integraciones conservan las capacidades de las otras tareas.

Al incorporar las migraciones de Memoria, el ejecutor detectó `migracion.retroceso` en la demo anterior: ya tenía migraciones posteriores a los números recién incorporados. Antes de recrearla se guardaron el volcado de PostgreSQL, los roles, la configuración y las plantillas en `bin/temporales`, con acceso privado. Es una copia temporal en memoria, no una copia operativa. No se cambió SQL ni se reservó una migración en VID3. Los vídeos y las capturas anteriores se conservaron fuera de Git.

La guía indica que vídeos, demo y CI se ejecutan en secuencia porque reinstalan las mismas dependencias de la interfaz. La tanda final terminó sin esa interferencia. Los ensayos fallidos de montaje, acceso limitado y dependencias no son los MP4 entregados.

## Pruebas y revisión

Las catorce pruebas del montaje pasaron. La revisión independiente con `revisar-cambios-etf` y `security-audit` dio GO de fuente a `a52e484`; no sustituye la revisión de los fotogramas. Semgrep se ejecutó localmente, sin métricas ni consulta de versiones:

```text
Ran 3 rules on 5 files: 0 findings.
no leaks found
```

Gitleaks no encontró secretos. Se actualizan los guiones, la guía, el README, el índice de documentación y `PLAN_IMPLEMENTACION.md`. No se modifican pantallas, backend ni SQL del módulo para estos recorridos.

Resultado real de `npm_config_cache=bin/temporales make ci`, después de la última integración de `main`:

```text
Ran 14 tests in 0.244s
OK
CI mínima completada.
```

La comprobación completa incluye `go vet ./...`, `go test ./...`, frontend, textos, temas y los ensayos de PostgreSQL presentes en la rama integrada. La demo se paró conservando sus volúmenes:

```text
DEMO-PARADA
```

No se hace push ni se fusiona esta rama en `main`.
