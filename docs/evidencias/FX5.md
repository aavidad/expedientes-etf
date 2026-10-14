# FX5: ensayos intermitentes de migraciones

Fecha: 14 de octubre de 2026. Rama: `fx5-ensayos-intermitentes`.

## Causa y reproducción

La ejecución de GitHub Actions
[38516520542](https://github.com/aavidad/expedientes-etf/actions/runs/38516520542)
falló en `evaluacion/ensayar.py` al aplicar `/pruebas/previo` sobre `1171a7c`.
El mensaje estaba vacío porque el ensayo sólo recogía stderr en el error y
`cmd/migrar` escribe claves JSON en stdout.

Los ensayos esperaban la existencia de `etf_migrador` consultando por socket
como `etf_instalador`. La imagen oficial de PostgreSQL usa un servidor
provisional sin TCP durante initdb. Tras crear los roles, esa consulta ya
responde, pero el migrador conecta por TCP a `127.0.0.1:5432`. Puede encontrar
el servidor provisional o su parada anterior al arranque definitivo.

La reproducción local controlada dejó tres segundos entre la creación de
roles y el fin de initdb:

```text
SOCKET Y ROL: 0 1
TCP: 2 127.0.0.1:5432 - no response
MIGRADOR PREMATURO: 1 stdout={"time":"2026-10-14T05:37:02.873867857Z","level":"ERROR","msg":"migracion.error"} stderr=''
ESPERA TCP AUTENTICADA: 3.03 s
MIGRADOR SERVIDOR DEFINITIVO: 0 {"time":"2026-10-14T05:37:06.717200405Z","level":"INFO","msg":"migracion.fin"}
```

La prueba permanente `scripts/ci/ensayar-arranque-postgres.py` mantiene initdb
detenido con una señal en el tmpfs del contenedor. Aplica la misma instantánea
anterior de Evaluación. Verifica que el criterio antiguo acepta el servidor
provisional, que el migrador falla y que la nueva espera permanece bloqueada.
Después libera initdb y comprueba versión anterior y actualización.

```sh
go build -o bin/migrar ./cmd/migrar
python3 -B scripts/ci/ensayar-arranque-postgres.py
python3 -B -m unittest discover -s scripts/ci -p test_ensayos.py -v
```

```text
ARRANQUE-OK fallo reproducido: socket y rol listos; TCP cerrado; migracion.error en stdout; stderr vacío
ARRANQUE-OK espera TCP autenticada; versión anterior y actualización
Ran 6 tests in 0.002s
OK
```

## Corrección y alcance

`scripts/ci/ensayos.py` reúne ejecución, diagnósticos y espera TCP autenticada
con `etf_migrador`. El plazo máximo de la espera es de 60 segundos. Sólo se
sondea `SELECT 1`; no se reintenta una migración. Los ensayos con Compose
conservan su espera de salud, que ya usa TCP.

Los fallos publican ambos canales, con un límite de 4.096 caracteres por
canal, ocultando credenciales y detalles SQL. Los ensayos sólo trabajan con
datos inventados. No cambian SQL, manifiestos, migrador, permisos ni datos de
la aplicación. Los nombres de contenedor ya eran aleatorios y los cortes
de manifiestos se conservan.

## Repeticiones

Antes de cambiar la espera se ejecutaron 30 ensayos completos de Evaluación
y 30 de Documento, intercalados con tres procesos concurrentes. Resultado:

```text
ANTES evaluacion: 30/30 OK, paralelos=3
ANTES documento: 30/30 OK, paralelos=3
DURACION 147.1s
```

La carrera no apareció espontáneamente en esa muestra; se reprodujo con la
inicialización detenida descrita arriba. La prueba controlada queda en
`make ci` para que la regresión no dependa de la velocidad del equipo.

Tras corregir la espera se repitió la misma muestra:

```text
DESPUES evaluacion: 30/30 OK, paralelos=3
DESPUES documento: 30/30 OK, paralelos=3
DURACION 147.3s
```

Para repetir la muestra con cada versión de los scripts:

```sh
python3 -B - <<'PY'
from concurrent.futures import ThreadPoolExecutor
import subprocess
import tempfile

def ensayar(modulo):
    with tempfile.TemporaryFile(mode='w+') as log:
        r = subprocess.run(['python3', '-B', f'migraciones/{modulo}/ensayar.py'],
                           stdout=log, stderr=subprocess.STDOUT, timeout=240)
        if r.returncode:
            log.seek(0)
            print(log.read()[-9000:])
        return modulo, r.returncode

with ThreadPoolExecutor(max_workers=3) as pool:
    resultados = list(pool.map(ensayar, ['evaluacion', 'documento'] * 30))
for modulo in ('evaluacion', 'documento'):
    codigos = [codigo for nombre, codigo in resultados if nombre == modulo]
    print(f'{modulo}: {codigos.count(0)}/{len(codigos)} OK, paralelos=3')
assert all(codigo == 0 for _, codigo in resultados)
PY
```

Tres ejecuciones consecutivas de `make ci` terminaron con código 0, incluidos
`go vet ./...`, `go test ./...`, los ensayos de migraciones, los textos y los
recorridos Chrome de entrada/permisos y carga de 2026:

```text
CI mínima completada.
```

Las dos últimas duraron 368,9 y 372,0 segundos. Cada una terminó con la línea
anterior, sin omitir puertas.

## Presupuesto de GitHub Actions

La ejecución
[38518714514](https://github.com/aavidad/expedientes-etf/actions/runs/38518714514),
sobre `6206840`, pasó la reproducción controlada del arranque y todos los
ensayos de migraciones. Evaluación terminó su instalación limpia a las
05:57:15 UTC y Bandeja a las 05:57:22 UTC. El recorrido de entrada pasó a
las 06:00:50 UTC, tras preparar la demo en el runner con cachés frías.

La matriz HTTP esperó sus ventanas por IP a las 06:01:04, 06:02:05 y
06:03:09 UTC. GitHub canceló el trabajo a las 06:04:45 UTC, al agotar los
15 minutos asignados, antes de completar esa matriz y el recorrido de carga:

```text
The job has exceeded the maximum execution time of 15m0s
```

El workflow dispone ahora de 25 minutos para completar la puerta en frío.
Se conservan las mismas pruebas, sus límites internos y las restricciones
de peticiones HTTP. Esta corrección del presupuesto se valida con un nuevo
commit y una nueva ejecución; no se repite el trabajo cancelado sin cambios.

## Comprobaciones auxiliares y límites

Se ejecutaron también los ensayos completos de Catálogo, Auditoría y el antiguo
ensayo de Expediente, que no forman parte de `make ci`. Auditoría pasó:

```text
ENSAYO-OK: limpia, repetida y actualización desde plataforma/0001
```

Para Auditoría se puso primero en `PATH` el Go ya descargado que devuelve
`go env GOROOT`. Ese ensayo desactiva descargas; el lanzador Go del sistema
intentaba verificar la selección automática de herramienta con GOSUMDB
desactivado. Se conservó esa restricción de red.

Catálogo falla por expectativas de versiones y opciones iniciales anteriores
a las ampliaciones de listas; el ensayo antiguo de Expediente copia una
instantánea sin todas las dependencias actuales y devuelve
`migracion.dependencia_ausente`. Se repitieron ambos con sus scripts originales
de `1171a7c`: también fallan. FX5 conserva sus datos y cortes, mejora sus
diagnósticos y no modifica esas pruebas de producto. Los ensayos de Expediente
activos en CI son Aceptación y Bandeja.

La revisión focal de seguridad se limita a la ejecución y a los diagnósticos
de estos ensayos: argumentos sin shell en el host, credenciales de prueba por
ficheros privados, contenedores propios, montajes de lectura y ausencia de
consultas a datos reales. Semgrep se ejecutó con cuatro reglas Python locales,
sin métricas ni consulta al registro de reglas:

```text
Ran 4 rules on 12 files: 0 findings.
```
