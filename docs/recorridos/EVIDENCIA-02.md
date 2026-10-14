# Comprobación del recorrido 02 de Coordinación

Esta comprobación corresponde al montaje inicial. Los vídeos revisados tras el cambio pedido por el responsable del producto están en [EVIDENCIA-MONTAJE.md](EVIDENCIA-MONTAJE.md).

Grabación local del 13 de octubre de 2026, en `vid2-coordinacion`, con Chrome del sistema y la demostración de este worktree en el puerto 18502. Sólo se usaron personas y datos inventados. El navegador quedó limitado al origen local; edge-tts recibió únicamente las frases inventadas de la locución.

## Vídeo y revisión

`make videos` generó los recorridos 01 y 02. El vídeo 02 se regeneró para retirar un fotograma blanco inicial de Chrome. La herramienta incorpora esa corrección antes de detectar los rótulos y montar la voz.

- Vídeo: `videos/02-coordinacion-20261013-234729-271078/video.mp4`.
- Hoja de control: `videos/02-coordinacion-20261013-234729-271078/hoja-control.png`.
- Duración final, medida con ffprobe: **119,96 segundos**.
- Introducción: **20,42 segundos**, con título, cartela y voz `es-ES-AlvaroNeural`.
- Formato: H.264 y AAC. Las nueve frases tienen audio no silencioso: introducción, siete pasos y cierre.

Se revisaron los nueve fotogramas de la hoja de control y la transición de la introducción al navegador. Los rótulos son legibles, las cartelas están completas y el aviso de descarga queda visible. Tras retirar el arranque uniforme, la transición muestra la pantalla de entrada. Un barrido del MP4 completo a 25 fotogramas por segundo encontró cero fotogramas uniformes. La lectura propone Alhama de Granada, centro Alhama y equipo 6, según el Anexo 3 de pruebas. La familia, los menores, teléfonos y correos proceden de ese fichero inventado; no se usaron originales del cliente.

El recorrido confirma una lectura con antivirus limpio, crea una derivación desde la pantalla, completa los datos necesarios para aceptar, obtiene `06-01-26`, descarga el Acta y busca ese número en «Aceptados recientes». El Word descargado queda junto al vídeo como `acta-aceptacion.docx`; se verificaron su recepción, su extensión y su cabecera ZIP.

Salida real de la grabación:

```text
COORDINACION-OK: 7 pasos; número=06-01-26; Acta descargada=true
```

## Repetición

Se repitió la misma automatización Playwright sin borrar datos ni reiniciar la demo. La segunda ejecución creó otro expediente, descargó su Acta y encontró su propia fila. Sus capturas y WebM están en `videos/02-coordinacion-repeticion-20261013-234941/`; es una comprobación de repetición, sin un segundo montaje narrado.

```text
COORDINACION-OK: 7 pasos; número=06-02-26; Acta descargada=true
```

Los dos `estado.json` contienen identificadores distintos. Se conservaron los expedientes anteriores, la numeración y las emisiones. La demo se paró al terminar:

```text
DEMO-PARADA
```

## Código y pruebas

La siembra de la bandeja necesitó una adaptación al contrato ya integrado de P2: subida con HTTP 200 devuelve una lectura; confirmar esa lectura con HTTP 201 crea la derivación. El cambio se limita a la preparación de datos de la demo y usa sus mismas rutas, sin alterar el módulo de derivación.

Las diez pruebas de `scripts/recorridos/` pasaron, incluidas las regresiones que retiran un arranque uniforme y rechazan una grabación sin contenido. Semgrep local, con métricas y consulta de versiones desactivadas, comprobó seis ficheros:

```text
Ran 3 rules on 6 files: 0 findings.
```

La revisión independiente con `revisar-cambios-etf` y `security-audit` fue estática, sobre los commits de esta tarea; no equivale a una auditoría completa. Gitleaks no encontró secretos. Se integró el `main` vigente antes de la comprobación final. La grabación se realizó antes de esa integración; las incorporaciones de otros módulos no cambian los pasos ni sus rutas.

La comprobación final `make ci` incluye `go vet ./...`, `go test ./...`, frontend, idiomas, temas y ensayos locales de PostgreSQL. Resultado final:

```text
Ran 10 tests in 0.250s
OK
CI mínima completada.
```

La primera salida corresponde a las pruebas directas del recorrido; la última, a `make ci` completo tras integrar `main`. No se hizo push ni se fusionó esta rama en `main`.
