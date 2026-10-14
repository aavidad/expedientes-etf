# Evidencia F0.03 · 13/10/2026

Rama `f0.03-compose`. Todos los datos y secretos del ensayo son inventados.
`plataforma/0001` se reservó previamente en `ETF_RESERVAS_MIGRACIONES.md`.
Salidas reales de esta ejecución, sin valores secretos; se han quitado espacios al final de línea:

## docker compose up -d --build (extracto final; tras borrar sólo el volumen de ensayo local)

```text
 Container etf-f003-postgres-1 Created
 Container etf-f003-migrar-1 Creating
 Container etf-f003-clamav-1 Created
 Container etf-f003-migrar-1 Created
 Container etf-f003-app-1 Creating
 Container etf-f003-app-1 Created
 Container etf-f003-postgres-1 Starting
 Container etf-f003-clamav-1 Starting
 Container etf-f003-clamav-1 Started
 Container etf-f003-postgres-1 Started
 Container etf-f003-postgres-1 Waiting
 Container etf-f003-postgres-1 Healthy
 Container etf-f003-migrar-1 Starting
 Container etf-f003-migrar-1 Started
 Container etf-f003-migrar-1 Waiting
 Container etf-f003-migrar-1 Exited
 Container etf-f003-app-1 Starting
 Container etf-f003-app-1 Started
```

## docker compose ps -a

```text
NAME                  IMAGE                                                                                          COMMAND                  SERVICE    CREATED         STATUS                     PORTS
etf-f003-app-1        etf-f003-app                                                                                   "/usr/local/bin/arra…"   app        2 minutes ago   Up 2 minutes (healthy)     127.0.0.1:18083->8080/tcp
etf-f003-clamav-1     clamav/clamav:1.4.3@sha256:75fb5fd95fcbe1d7e6d240c369c1572b686ee2c95949d1042b5148de8eddebb4    "/usr/sbin/clamd --c…"   clamav     2 minutes ago   Up 2 minutes (healthy)     3310/tcp, 7357/tcp
etf-f003-migrar-1     etf-f003-migrar                                                                                "/usr/local/bin/migr…"   migrar     2 minutes ago   Exited (0) 2 minutes ago
etf-f003-postgres-1   postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15   "docker-entrypoint.s…"   postgres   2 minutes ago   Up 2 minutes (healthy)     5432/tcp
```

## curl --fail --silent http://127.0.0.1:18083/salud

```text
{"estado":"ok"}
```

## docker compose logs --no-color migrar (primera aplicación)

```text
migrar-1  | migracion.aplicada.plataforma.0001
migrar-1  | {"time":"2026-10-13T10:55:47.983981895Z","level":"INFO","msg":"migracion.fin"}
```

## docker compose run --rm --no-deps migrar

```text
 Container etf-f003-migrar-run-10530c628cfd Creating
 Container etf-f003-migrar-run-10530c628cfd Created
migracion.existente.plataforma.0001
{"time":"2026-10-13T10:56:15.45622888Z","level":"INFO","msg":"migracion.fin"}
```

## python3 -u despliegue/base/comprobar-migraciones.py

```text
ENSAYO-OK limpia y repetida: 1 migración, etf_migrador
ENSAYO-OK propietario, RLS forzada, app sin CREATE/escritura y contraseña incorrecta rechazada
ENSAYO-OK contraseña del instalador retirada y acceso TCP rechazado
ENSAYO-OK huella alterada rechazada antes de escribir
ENSAYO-OK fallo SQL revierte DDL y control en la misma transacción
ENSAYO-OK dependencias desordenadas y dos ejecutores concurrentes: 3 filas, sin duplicados
ENSAYO-OK historial con retroceso rechazado
ENSAYO-OK base desechable eliminada
```

## Comprobación por docker inspect: salud, usuarios, raíz, capacidades, red y nombres de variables (sin imprimir valores)

```text
app: healthy, usuario=10001:10001, raíz=ro, capacidades=vacías, no-new-privileges
postgres: healthy, usuario=70:70, raíz=ro, capacidades=vacías, no-new-privileges
clamav: healthy, usuario=100:101, raíz=ro, capacidades=vacías, no-new-privileges
APP-SIN-CREDENCIALES-MIGRADOR: OK
```

## App aislada: docker run con --network none --read-only --cap-drop ALL --security-opt no-new-privileges, sin entorno añadido ni montajes; eliminada al terminar

```text
{"estado":"ok"}
APP-SIN-SECRETOS-NI-VOLUMENES: OK
idiomas.reserva
temas.reserva
{"time":"2026-10-13T10:58:29.522359578Z","level":"INFO","msg":"servidor.inicio"}
{"time":"2026-10-13T10:58:29.588549927Z","level":"INFO","msg":"http.peticion","ruta":"/salud","estado":200,"duracion_ms":0}
```

## go test ./... (salida completa)

```text
?   	github.com/aavidad/expedientes-etf/cmd/etf	[no test files]
?   	github.com/aavidad/expedientes-etf/cmd/migrar	[no test files]
ok  	github.com/aavidad/expedientes-etf/internal	0.004s
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/anexos	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/clamav	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/docx	[no test files]
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros	0.004s
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/http	0.007s
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/ldap	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/postgres	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/registro_manual	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/actividad	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/actividad/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/documento	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/evaluacion	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/evaluacion/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/expediente	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/identidad	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/paquete	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/plataforma	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/plataforma/puertos/entrada	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/actividad	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/auditoria	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/catalogo	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/documento	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/estadistica	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/evaluacion	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/expediente	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/identidad	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/paquete	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/plataforma	[no test files]
ok  	github.com/aavidad/expedientes-etf/internal/plataforma	(cached)
ok  	github.com/aavidad/expedientes-etf/internal/plataforma/migracion	0.003s
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/actividad	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/auditoria	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/catalogo	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/documento	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/estadistica	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/evaluacion	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/expediente	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/identidad	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/paquete	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma	[no test files]
?   	github.com/aavidad/expedientes-etf/web	[no test files]
```

## npm test --prefix web (tras npm ci --ignore-scripts --no-audit --no-fund --prefix web)

```text

> etf-expedientes-web@0.1.0 test
> node --test tests/*.test.mjs

TAP version 13
# Subtest: todas las variables CSS usadas existen en el tema
ok 1 - todas las variables CSS usadas existen en el tema
  ---
  duration_ms: 0.834331
  ...
# Subtest: los textos propios de pantalla salen del idioma
ok 2 - los textos propios de pantalla salen del idioma
  ---
  duration_ms: 7.065795
  ...
# Subtest: la estructura no contiene colores, fuentes ni dimensiones de tema
ok 3 - la estructura no contiene colores, fuentes ni dimensiones de tema
  ---
  duration_ms: 0.244251
  ...
1..3
# tests 3
# suites 0
# pass 3
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 313.5212
```

## Otras comprobaciones

- `go vet ./...`: salida vacía, salida del proceso 0.
- `bin/gopls version`: `golang.org/x/tools/gopls v0.20.0`; comprobación CLI de los cuatro Go cambiados: salida vacía, salida del proceso 0.
- Semgrep 1.178.0, métricas y comprobación remota de versión desactivadas:
  `semgrep --metrics=off --disable-version-check --config despliegue/base/semgrep.yaml --error cmd/migrar internal/plataforma/migracion compose.yaml Dockerfile`.
  Reglas locales focales, sin subida de código; no equivalen a una auditoría completa.
- Revisión independiente con `revisar-cambios-etf`, `revisar-sql-etf` y
  `security-audit` focal: GO de código tras retirar la contraseña del instalador
  y rechazar dependencias a números iguales/posteriores del mismo módulo.
  Las comprobaciones runtime pendientes del revisor están acreditadas arriba.

## Incidencias resueltas y límites

El primer arranque encontró agotados los rangos automáticos de Docker; se
configuraron subredes libres. Docker 29.5.2 no publicó el puerto de app en una
red exclusivamente interna: la red de entrada sólo para app resolvió el acceso
HTTP. `npm test` falló inicialmente por falta de `node_modules`; después de
`npm ci` con el lock existente pasan las tres pruebas. No se cambiaron versiones
ni dependencias del proyecto. `gopls` faltaba y se instaló la versión fijada
0.20.0 en `bin/` (ignorado), sin cambiar `go.mod`.

Los logs de ClamAV advierten de firmas incluidas antiguas. Salud acredita el
daemon, no la actualidad de firmas para producción. Su suministro local y el
mantenimiento periódico siguen pendientes de TRD §13. El servidor F0.04 aún
no abre PostgreSQL ni comprueba compatibilidad; F0.03 no cambia `cmd/etf/`.
Idiomas y temas quedan en sólo lectura hasta la publicación administrada F0.14c.
No se implementaron copias, HTTPS, despliegue de producción ni capacidades de negocio.

No hubo push ni fusión. Se conserva el entorno local con los tres servicios sanos;
la base auxiliar y el contenedor de reserva se eliminaron.

## Salida de Semgrep local

```text
┌─────────────┐
│ Scan Status │
└─────────────┘
  Scanning 5 files tracked by git with 4 Code rules:

  Language      Rules   Files          Origin   Rules
 ─────────────────────────────        ────────────────
  <multilang>       2       5          Custom       4
  go                2       3



┌──────────────┐
│ Scan Summary │
└──────────────┘
✅ Scan completed successfully.
 • Findings: 0 (0 blocking)
 • Rules run: 4
 • Targets scanned: 5
 • Parsed lines: ~100.0%
 • Scan skipped:
   ◦ Files matching .semgrepignore patterns: 1
 • Scan was limited to files tracked by git
 • For a detailed list of skipped files and lines, run semgrep with the --verbose flag
Ran 4 rules on 5 files: 0 findings.
```

## Detección de secretos del diff

Orden: `git diff --cached | gitleaks stdin --no-banner --redact`.

```text
12:59PM INF scanned ~62363 bytes (62.36 KB) in 54ms
12:59PM INF no leaks found
```
