# Evidencia de F0.08a

Rama `f0.08a-auditoria-dominio`, 13/10/2026. Sin SQL, PostgreSQL ni conexiones.
Contrato y uso del formato: [FORMATO.md](../../internal/dominio/auditoria/FORMATO.md).

## Pruebas y compilación

Las pruebas ejecutan cambios de todos los campos en cada posición, cambios de
cada uno de los 256 bits de las dos huellas, borrado en cada posición y de toda
la cadena, inserción, duplicado, inversión y recálculo frente al sello original.
Incluyen tramos vacíos y parciales, JSON ambiguo, límites de fichero y
confirmación antes de entregar; un registro fallido impide iniciar la entrega.
Hay 64 confirmaciones concurrentes contra el doble en memoria y procesos reales
del comprobador con los tres códigos de salida.

La validación usa Bubblewrap sin red, repositorio y herramientas de sólo lectura,
entorno vacío con la lista de variables que figura en el comando y temporales
en tmpfs limitados. Los límites finales son 100 segundos de CPU por proceso,
3 GiB de espacio de direcciones, 128 procesos, 256 descriptores y 128 MiB por
fichero. El tiempo total está limitado a 120 segundos; el disco temporal, a
1,5 GiB más 64 MiB.

Comando exacto del último vet, test completo, build y cobertura:

```sh
timeout 120s bwrap --unshare-all --die-with-parent --new-session \
  --ro-bind /usr /usr --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
  --symlink usr/bin /bin --proc /proc --dev /dev \
  --dir /workspace --ro-bind /ruta/al/proyecto /workspace \
  --size 1610612736 --tmpfs /scratch --size 67108864 --tmpfs /tmp \
  --chdir /workspace --clearenv \
  --setenv PATH /usr/local/go/bin:/usr/bin:/bin --setenv HOME /scratch \
  --setenv TMPDIR /scratch --setenv GOCACHE /scratch/go-cache \
  --setenv GOPATH /scratch/go --setenv GOPROXY off --setenv GOSUMDB off \
  --setenv GOMAXPROCS 2 \
  -- prlimit --cpu=100 --as=3221225472 --nproc=128 --nofile=256 --fsize=134217728 \
  -- sh -c 'go vet ./... && go test ./... && go build -trimpath -o /scratch/comprobar-auditoria ./cmd/comprobar-auditoria && go test -cover ./internal/dominio/auditoria ./internal/aplicacion/auditoria ./cmd/comprobar-auditoria'
```

Código de salida 0. `go vet` y `go build` no producen salida. Salida real:

```text
ok  	github.com/aavidad/expedientes-etf/cmd/comprobar-auditoria	0.008s
?   	github.com/aavidad/expedientes-etf/cmd/etf	[no test files]
?   	github.com/aavidad/expedientes-etf/cmd/migrar	[no test files]
ok  	github.com/aavidad/expedientes-etf/internal	0.004s
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/anexos	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/clamav	[no test files]
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/docx	0.252s
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros	0.003s
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/http	0.003s
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/ldap	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/postgres	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/adaptadores/registro_manual	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/actividad	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/aplicacion/actividad/puertos/entrada	[no test files]
ok  	github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria	0.002s
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
ok  	github.com/aavidad/expedientes-etf/internal/dominio/auditoria	0.009s
?   	github.com/aavidad/expedientes-etf/internal/dominio/catalogo	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/documento	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/estadistica	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/evaluacion	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/expediente	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/identidad	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/paquete	[no test files]
?   	github.com/aavidad/expedientes-etf/internal/dominio/plataforma	[no test files]
ok  	github.com/aavidad/expedientes-etf/internal/plataforma	0.002s
ok  	github.com/aavidad/expedientes-etf/internal/plataforma/migracion	0.002s
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
ok  	github.com/aavidad/expedientes-etf/internal/dominio/auditoria	0.010s	coverage: 95.6% of statements
ok  	github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria	0.003s	coverage: 100.0% of statements
ok  	github.com/aavidad/expedientes-etf/cmd/comprobar-auditoria	0.008s	coverage: 97.8% of statements
```

Después se añadió una semilla JSON válida al fuzzing, junto a las tres inválidas.
Se ejecutó en el mismo aislamiento, con 768 MiB en /scratch, 90 segundos de
tiempo total y 75 segundos de CPU por proceso. Comandos internos:

```sh
go test ./internal/dominio/auditoria
go test ./internal/dominio/auditoria -run=^$ -fuzz=FuzzDecodificarExportacion -fuzztime=5s -parallel=2
```

Salida real, código 0:

```text
ok  	github.com/aavidad/expedientes-etf/internal/dominio/auditoria	0.010s
fuzz: elapsed: 0s, gathering baseline coverage: 0/4 completed
fuzz: elapsed: 0s, gathering baseline coverage: 4/4 completed, now fuzzing with 2 workers
fuzz: elapsed: 3s, execs: 907 (302/sec), new interesting: 3 (total: 7)
fuzz: elapsed: 6s, execs: 907 (0/sec), new interesting: 3 (total: 7)
fuzz: elapsed: 6s, execs: 907 (0/sec), new interesting: 3 (total: 7)
PASS
ok  	github.com/aavidad/expedientes-etf/internal/dominio/auditoria	6.016s
```

Una ejecución anterior con tres semillas inválidas completó 222555 casos.
Los primeros intentos de validación fallaron por límites del entorno: la cuota
de procesos aplicada antes de crear el namespace, el temporal de compilación de
64 MiB y el límite por fichero de 64 MiB, menor que la prueba de fichero
sobredimensionado. Se ajustaron esos límites y la validación final pasó.
No se ejecutó `go test -race`. No hay frontend ni adaptador persistente que
probar en esta mitad.

## Revisión local

Revisión independiente con `revisar-cambios-etf` y `security-audit`: GO estático,
incluido el último contrato de entrada. Se corrigieron dos defectos con
regresiones: borrado de prefijo cambiando el punto anterior y evento sin
`perfiles` que rompía la ida y vuelta JSON. La revisión no ejecutó código.

Semgrep sobre las zonas del encargo, con cuatro reglas locales del repositorio:

```sh
SEMGREP_SEND_METRICS=off SEMGREP_ENABLE_VERSION_CHECK=0 semgrep --metrics=off --config despliegue/base/semgrep.yaml --quiet internal/dominio/auditoria internal/aplicacion/auditoria internal/puertos/salida/auditoria cmd/comprobar-auditoria internal/arquitectura_test.go
```

Salida vacía, código 0. El delta del puerto de entrada se volvió a analizar.
Gitleaks se ejecutó con `gitleaks dir --no-banner --redact --log-level error`
sobre cada una de esas cinco rutas y después sobre el puerto de entrada:
salida vacía, código 0. No se enviaron fuentes ni datos a servicios externos.

`gopls check` sobre evento, cadena, exportación, caso de uso y comando:
salida vacía. `git diff --check` y `gofmt -l`: salida vacía.

## Alcance pendiente

`docs/producto/` no cambia: el encargo divide F0.08 y aún no está completa.
F0.08b debe resolver la representación persistente de `sistema` frente al
campo UUID del diagrama, el detalle ampliado, transacciones, sellos diarios,
custodia de copias y autorización de exportación. La interfaz HTTP y la pantalla
no se implementan en F0.08a. No se hace push ni se fusiona.

