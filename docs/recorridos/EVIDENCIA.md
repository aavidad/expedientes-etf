# Comprobación de VID1

Esta comprobación corresponde al montaje inicial. Los vídeos revisados tras el cambio pedido por el responsable del producto están en [EVIDENCIA-MONTAJE.md](EVIDENCIA-MONTAJE.md).

Comprobación local del 13 de octubre de 2026 en la rama `vid1-videos`, con `main` integrado hasta `5bb5039` (armazón UI1). El código de la retirada segura de perfiles se revisó en `6096492`.

## Recorrido real con voz

Se ejecutaron `make videos` y `make video RECORRIDO=01-entrada` contra la demostración de este worktree, en el puerto 18491. El recorrido entra sin perfil, muestra el acceso pendiente de alta, sale mediante el menú de la persona y entra como Administración. Después selecciona a Sergio Campos, asigna el perfil profesional y el equipo 2, guarda y comprueba la confirmación.

El vídeo final, con la corrección de concurrencia, está fuera de Git:

```text
videos/01-entrada-20261013-195543-513597/video.mp4
videos/01-entrada-20261013-195543-513597/hoja-control.png
```

Salida real:

```text
GRABACION-OK demo: 7 rótulos; demo restaurada=true
VIDEO-OK videos/01-entrada-20261013-195543-513597/video.mp4; intro=23.4s; voz=AlvaroNeural; hoja-control.png
MONTAJE-FINAL-OK intro=23.38s; frases=7; congelados=6; audio no silencioso=True
VOZ-OK 9 frases con audio no silencioso
```

`ffprobe` confirmó H.264, AAC y 89,72 segundos. La introducción dura 23,38 segundos. Las nueve frases corresponden a la introducción, los siete pasos y el cierre. La hoja de control permite comprobar los fotogramas durante la locución de cada paso.

## Repetición y temas

Los recorridos de `01-entrada-20261013-193636-239645/` y `01-entrada-20261013-194156-611910/` pasaron con UI1. Entre ambos se paró y reanudó la demostración conservando sus datos. Ambos terminaron con siete rótulos y `restaurado=true`. La comprobación posterior de los accesos confirmó:

```text
SESION-OK sergio.campos: pendiente_alta; sin perfil
DEMO-OK: 7 usuarios únicos, 8 equipos, perfiles, equipos, privacidad y rechazo CSRF comprobados con curl.
```

El montaje sintético pasó con `expedientes-etf` y con `alto-contraste`. El segundo está en `sintetico-01-entrada-20261013-193728-460630/`. Sus siete rótulos se detectaron correctamente. Estas dos pruebas usan silencio de prueba y no acreditan un recorrido en la aplicación ni una voz narrada.

## Pruebas y revisión

```text
Ran 8 tests in 0.226s
OK
Ran 3 rules on 12 files: 0 findings.
CI mínima completada.
```

La última línea corresponde a `make ci` completo después de corregir la retirada de perfiles. Incluye `go vet ./...`, `go test ./...`, frontend, textos, temas y los ensayos de PostgreSQL, concurrencia, auditoría y seguridad por filas. Semgrep se ejecutó con reglas locales de `scripts/recorridos/seguridad.yml`, sin métricas ni consulta de versiones.

La revisión independiente con `revisar-cambios-etf` y `security-audit`, centrada en el código cambiado, encontró que una respuesta fallida podía llevar a retirar una concesión ajena de la misma versión. Se corrigió: sin respuesta propia confirmada, conserva los permisos. La regresión comprueba ese conflicto, la respuesta propia válida y un cambio posterior. El revisor dio GO al código de `6096492`; su revisión fue estática, no una auditoría completa.

## Incidencias del entorno

El disco de Trabajo retuvo operaciones de lectura y escritura. Para completar la comprobación se usó un auxiliar temporal de lectura `O_NOATIME` y se instalaron las dependencias npm en `videos/.herramientas/npm/`, enlazadas desde `web/node_modules`. El `package-lock.json` coincide con el del repositorio; `npm ci` se ejecutó íntegro en esa carpeta, sin omitir la instalación ni otras pruebas. Estas herramientas locales no forman parte del kit ni de Git.

Una comprobación de accesos lanzada a la vez que una grabación agotó el límite de entradas y produjo HTTP 429. Se separaron las pruebas. Al integrar UI1, los volúmenes de idiomas todavía contenían los textos anteriores y los botones nuevos salían vacíos. Se recreó sólo esta demostración inventada conforme a su guía; las comprobaciones de repetición posteriores conservaron sus datos. Los intentos fallidos no se cuentan como vídeos terminados.

No se hizo push. No se modificó código propio de `web/src` ni de `internal`; los cambios de esas carpetas llegaron únicamente al integrar `main` según el encargo.
