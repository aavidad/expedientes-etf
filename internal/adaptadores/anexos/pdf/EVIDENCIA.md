# CIPDF: estabilidad del lector en integración continua

## Fallo y reproducción

Las ejecuciones de GitHub Actions `38453701706` y `38449096352` fallaron en la lectura de anexos válidos, con `anexo.limite_recursos` en unos 0,01 s. El segundo mensaje, «rechazo guardó una derivación», comparaba el total con tres altas esperadas aunque alguna alta anterior hubiera fallado. No demostraba que un rechazo guardara datos.

En Linux local, Go 1.25.5, 50 recorridos con `taskset -c 0` y `GOMAXPROCS=2` pasaron con los límites anteriores. También pasaron 100 hijos directos con tuberías y el entorno del trabajador. No se reprodujo el fallo espontáneo del runner. Un binario con `-race` falló con `fatal error: runtime: cannot allocate memory`; sus reservas virtuales adicionales no representan el binario de producción.

La regresión `TestTrabajadorConPresionDeMemoria` conserva una reserva sintética de 80 MiB antes de instalar los límites y procesa el mismo Anexo 4 inventado. Simula mayor consumo inicial del runtime; no mide el consumo real de GitHub. Se compilaron tres variantes del mismo código y se ejecutó la misma prueba:

| DATA | AS | Resultado |
| --- | --- | --- |
| 96 MiB | 2 GiB | `exit status 2: fatal error: runtime: cannot allocate memory` |
| 96 MiB | 4 GiB | Mismo fallo |
| 256 MiB | 2 GiB | `PASS` (0,09 s) |

La comparación aísla DATA como límite insuficiente bajo esa presión. El runtime Go utiliza mmap de lectura/escritura además del heap; DATA limita también esas reservas. Sigue pendiente atribuir el fallo espontáneo del runner a un consumo medido allí.

## Protección y diagnóstico

Se mantiene DATA como límite duro y se eleva a 256 MiB. AS pasa a 4 GiB para permitir las reservas virtuales de Go. `GOMEMLIMIT=64MiB` es un objetivo blando del recolector, no sustituye al límite duro. CPU sigue en 3 s, tiempo en 5 s, fichero y core en cero; se respetan tanto el límite blando como el duro heredados si son menores. Permanecen una plaza por lector, 10 MB de entrada, 20 páginas y cuotas de objetos, lecturas, glifos y salida. La prohibición de intercambio a disco sigue siendo responsabilidad del despliegue.

El padre drena stderr y retiene como máximo 4 KiB en memoria. Descarta pilas, excepciones y contenido desconocido; sólo conserva líneas exactas de errores de memoria del runtime. `FalloTrabajador.Error()` entrega únicamente `anexo.limite_recursos`; `errors.Is` reconoce esa clave. El diagnóstico técnico se consulta mediante el tipo de error o `%+v`. No se añade a HTTP ni a auditoría, y no se registra el contenido del PDF.

Cada rechazo compara ahora el número de derivaciones antes y después de su propia petición, incluido EICAR cuando se activa ClamAV. Las altas válidas siguen exigiendo HTTP 201 y consulta posterior con HTTP 200.

## Comprobaciones locales

- `go test ./internal/adaptadores/anexos/pdf/ ./internal/adaptadores/http/ -count=30`: ambos paquetes verdes, con las regresiones de presión, límites heredados y diagnóstico sin contenido personal.
- Semgrep local con `internal/adaptadores/anexos/pdf/semgrep.yml`, métricas y comprobación de versión desactivadas: sin hallazgos en las reglas del lector. No se envió código al servicio de análisis.
- La revisión focal cubre los límites del hijo, las tuberías y la privacidad del diagnóstico. No es una auditoría completa ni acredita el aislamiento del despliegue.

La prueba ClamAV sigue siendo optativa mediante `ETF_PRUEBA_CLAMAV_DIRECCION`; esta corrección no modifica su política. La evidencia definitiva de integración continua se entrega con el enlace a la ejecución del commit publicado en esta rama.
