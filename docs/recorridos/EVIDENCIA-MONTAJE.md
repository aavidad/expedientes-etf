# Vídeos con acciones y voz sincronizadas

Revisión del 14 de octubre de 2026 en `vid2-coordinacion`, tras el cambio pedido por el responsable del producto. `make videos` terminó correctamente para los dos recorridos. Se grabaron con Chrome del sistema, en la demo local del puerto 18502, y con datos inventados.

## Resultados finales

| Recorrido | MP4 fuera de Git | Duración | Introducción | Cierre |
| --- | --- | --- | --- | --- |
| 01 · Entrada y acceso | `videos/01-entrada-20261014-002541-966595/video.mp4` | 68,04 s | 12,00 s | 5,00 s |
| 02 · Coordinación | `videos/02-coordinacion-20261014-002647-823972/video.mp4` | 85,88 s | 12,80 s | 5,48 s |

Cada carpeta contiene `hoja-control.png`, `revision.json` y los once fotogramas `revision-*.png` extraídos directamente del MP4 con ffmpeg: segundos 2, 6 y 12, mitad de los siete pasos y cierre. Se revisaron individualmente, además de la hoja de control.

El título aparece durante 3,5 segundos. El resto de la introducción muestra capturas reales del recorrido con zoom, desplazamiento y transiciones suaves. Las locuciones explican el propósito y quién realiza el recorrido, sin enumerar pasos. Los rótulos resumen cada paso en 4–8 palabras.

Las acciones de escritura, selección, clic y carga se ven durante sus frases. El ratón permanece visible y sólo el elemento actual queda resaltado. Si sobra tiempo de locución, se mantiene el resultado. Si sobra tiempo de grabación, únicamente se pueden retirar fotogramas repetidos e idénticos a resolución completa; nunca se acelera la escritura ni el ratón. Si no hay pausas suficientes, la herramienta se detiene para ampliar la frase.

El cierre superpone su cartela sobre la última pantalla real difuminada. Ambas pistas usan `es-ES-AlvaroNeural`, con las frases medidas antes de grabar. Las contraseñas aparecen enmascaradas; no se usaron personas ni expedientes reales. Sólo el texto inventado de la voz se envió a edge-tts.

## Sincronización y repetición

`ancla.png` y `ancla.json` identifican un marcador inicial. La herramienta lo encuentra en el WebM y guarda el ajuste entre el reloj de las acciones y el vídeo en `alineacion.json`. `escenas-alineadas.json` y `plan.json` permiten comprobar los intervalos de cada frase y acción.

La grabación 01 terminó con siete pasos y restauró los permisos de la cuenta inventada. La grabación 02 creó y aceptó `06-05-26`, descargó su Acta y buscó su propio número en «Aceptados recientes». Se conservan la numeración, los expedientes previos y las emisiones. Los ensayos anteriores no se presentan como vídeos finales.

Se comprobaron los límites de introducción y cierre, el audio de todas las frases y el movimiento de la vista previa. Un barrido de ambos MP4 a 25 fotogramas por segundo encontró cero fotogramas uniformes.

## Pruebas y revisión

Las catorce pruebas locales pasaron. Incluyen el anclaje temporal, la correspondencia entre frases y acciones, la conservación de cambios de píxel y la retirada de esperas. Semgrep se ejecutó localmente, sin métricas ni consulta de versiones. La revisión independiente de fuente con `revisar-cambios-etf` y `security-audit` dio GO a `b0a37b7`; Gitleaks no encontró secretos. Esa revisión fue estática y no sustituye la revisión visual realizada aquí.

Se actualizan los guiones, la guía, la plantilla, el README y `PLAN_IMPLEMENTACION.md`. Se integra `main` antes de la comprobación completa de CI. La grabación usa el código de aplicación integrado antes de esa última comprobación; los ficheros de montaje pertenecen a esta tarea. La demo se para al acabar. No se hace push ni se fusiona esta rama en `main`.

Resultado real de la comprobación final, después de integrar `main`:

```text
CI mínima completada.
DEMO-PARADA
```

`make ci` incluye `go vet ./...`, `go test ./...`, frontend, textos, temas y los ensayos locales de PostgreSQL presentes. La rama queda limpia, con los vídeos y sus capturas fuera de Git.
