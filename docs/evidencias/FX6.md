# FX6 · Recuperación de los ensayos antiguos de migraciones

Rama: `fx6-ensayos-antiguos`. Fecha: 14 de octubre de 2026.

## Alcance

Se recuperan `migraciones/catalogo/ensayar.py` y
`migraciones/expediente/ensayar.py`, cuyos fallos previos quedaron documentados
en [FX5](FX5.md#comprobaciones-auxiliares-y-límites).

Catálogo comprueba ahora 25 opciones en el ámbito `listas`, incluido
`fin_seguimiento.CAMBIO_SITUACION`, y su versión inicial vigente 2. Las pruebas
de edición calculan las versiones nuevas desde la publicación previa y
conservan las comprobaciones de conflicto, historia y reversión auditada.
La prueba de inmutabilidad publica en 2030 para respetar la vigencia actual
del seguimiento; comprueba los dos lados de la frontera y la lectura anual.
Las instantáneas anteriores conservan sólo las migraciones correspondientes
al corte: Auditoría/0001 en la actualización antigua y Catálogo anterior a
0012 en la prueba de avisos, sin seguimiento ni módulos consumidores.

El ensayo inicial de Expediente incluye Paquete, requerido por el Catálogo
actual. Retira de la copia temporal Auditoría desde 0002, cuya dependencia
Expediente/0020 aún no existe en ese corte anterior a F1.01. Después instala
todas las migraciones actuales, repite la ejecución y prueba una base limpia.
Conserva las pruebas de RLS, auditoría, EICAR y antivirus caído.

No se cambian catálogos, migraciones SQL ni manifiestos del repositorio.
Sólo se recortan las copias temporales de los ensayos. Los contenedores usan
PostgreSQL 18 y ClamAV con las imágenes fijadas, credenciales efímeras y datos
inventados; al terminar se eliminan sus recursos.

## Tiempo e incorporación a CI

Medición inicial con `/usr/bin/time -f` sobre cada comando completo:

```text
TIEMPO catalogo 18.69 s
TIEMPO expediente 17.42 s
```

Ambos quedan por debajo de un minuto y se añaden a `make ci` mediante
`scripts/ci/comprobar.sh`, después de la prueba del arranque de PostgreSQL.

## Validación

Se ejecutó cada ensayo diez veces consecutivas, sin filtros y deteniendo la
serie ante el primer fallo, con estos comandos:

```sh
python3 -B migraciones/catalogo/ensayar.py
python3 -B migraciones/expediente/ensayar.py
```

Salida real del ejecutor de repeticiones:

```text
catalogo: 10/10 verde; mínimo=17.46 s; máximo=18.94 s; media=18.08 s
expediente: 10/10 verde; mínimo=16.25 s; máximo=17.87 s; media=17.18 s
```

Cada ejecución de Catálogo terminó con:

```text
ENSAYO-OK actualización desde catalogo/0002 + paquete/0001; UUID y versión semver conservados
ENSAYO-OK avisos: plazos pasados activos hoy y conservados; hoy/futuros rechazados sin cambios
ENSAYO-OK contenedor y volumen destruidos
```

Cada ejecución de Expediente terminó con:

```text
ENSAYO-OK base limpia, RLS, auditoría, EICAR y ClamAV caído
```

Revisión independiente estática con `revisar-cambios-etf`, `revisar-sql-etf`
y `security-audit`: GO sobre el cambio, sin hallazgos bloqueantes. Semgrep
local, con métricas y consulta de versión desactivadas, usando
`scripts/ci/equipo/semgrep-local.yaml` sobre los archivos cambiados: una regla,
dos archivos Go analizados, cero hallazgos. Esa regla no analiza Python ni Bash.

`git merge main`: `Already up to date.`. No se hizo push ni se integró la rama en main.

`make ci` completo terminó con código 0. Incluyó `go vet ./...`,
`go test ./...`, textos es/en, temas, frontend, Gitleaks, todos los ensayos
incorporados y los recorridos de entrada, permisos y carga de 2026 con Chrome.
Última línea real:

```text
CI mínima completada.
```

## Archivos modificados

- `migraciones/catalogo/ensayar.py` y `migraciones/expediente/ensayar.py`:
  cortes históricos y dependencias de las copias temporales.
- `internal/adaptadores/postgres/catalogo/integracion_test.go` y
  `zz_edicion_integracion_test.go`: expectativas actuales de opciones y versiones.
- `scripts/ci/comprobar.sh`: enganche común autorizado para ambos ensayos.
- `scripts/ci/README.md`, `README.md`, `docs/README.md`: alcance de CI y evidencia.
- `docs/producto/TRD.md`, `PLAN_IMPLEMENTACION.md` e `INDICE.md`: pruebas y puerta
  de integración, con índice regenerado.
- Este documento: evidencia y tiempos.

No hay migraciones nuevas, cambios de producto ni dudas pendientes.
