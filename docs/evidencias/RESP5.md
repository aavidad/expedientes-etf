# RESP5 · Carga histórica desde actas ODT

La carga histórica admite Actas de Aceptación y de Cierre en LibreOffice ODT,
además de DOCX y PDF. Reutiliza el lector ODT de RESP4 y el perfil de actas de
RESP2. La lectura devuelve propuestas para revisar; sólo la confirmación
existente guarda el expediente. No se conservan los ficheros ni se añaden tablas
o migraciones.

## Corpus y reproducción

Las tres muestras de `internal/adaptadores/anexos/actas/testdata/` contienen datos
inventados: aceptación de 2022, aceptación de 2023 y cierre de 2026. Proceden de
los DOCX generados desde las plantillas oficiales por `scripts/actas/generar.go`
y fechados por `scripts/actas/generar.py`. Para convertirlos:

```sh
python3 -B scripts/actas/generar_odt.py
```

LibreOffice se ejecuta localmente con un perfil temporal. El generador vacía
`meta.xml`, fija las fechas ZIP y elimina comentarios del contenedor. Las pruebas
comprueban que los metadatos no conservan autor ni fechas de edición.

## Pruebas y revisión

```sh
go test ./internal/adaptadores/anexos/lectura ./internal/adaptadores/anexos/odt ./internal/aplicacion/expediente ./internal/adaptadores/http
```

Los cuatro paquetes pasan. Se comprueban los campos esperados de ambas actas,
la obligación de revisión, el aislamiento respecto de las derivaciones y el
rechazo de contenedores hostiles también en modo actas: rutas, macros, objetos,
expansión, entidades XML, profundidad, atributos y repeticiones.

La revisión independiente con `revisar-cambios-etf`, `programar-interfaz-etf`
y `security-audit` focal no encuentra defectos confirmados en seguridad o
usabilidad estática. Su alcance es este cambio; no constituye una auditoría
completa del proyecto.

Semgrep se ejecuta localmente, sin métricas ni comprobación de versiones:

```sh
SEMGREP_SEND_METRICS=off semgrep scan --config internal/adaptadores/anexos/odt/semgrep.yml --config scripts/recorridos/seguridad.yml --metrics=off --disable-version-check --error --no-git-ignore internal/adaptadores/anexos/odt/acta.go internal/adaptadores/anexos/lectura/lector.go web/src/carga/actas.tsx scripts/actas/generar_odt.py scripts/recorridos/carga-actas.mjs
```

Salida: `Ran 6 rules on 5 files: 0 findings.`

Los textos mantienen paridad es/en y no añaden literales visibles al código.
Los manuales de profesionales y Coordinación sólo cambian en la sección de
carga; sus PDF se regeneran con el exportador local.

## Chrome y puerta completa

`make ci` termina con código 0 y esta última línea:

```text
CI mínima completada.
```

Incluye `go vet ./...`, `go test ./...`, frontend, textos, secretos, arquitectura,
ensayos PostgreSQL y recorridos de Chrome del sistema. La carga añade dos casos:
aceptación ODT de 2022 y aceptación ODT de 2023 con cierre ODT de 2026. El recorrido
comprueba antivirus limpio, lectura HTTP 200, propuestas, revisión, confirmación
HTTP 200 y recuperación de número, estado y adultos en ficha. Mantiene los casos
PDF firmado y DOCX/PDF de RESP2, permisos por equipo, ausencia de almacenamiento
en el navegador, teclado, inglés y alto contraste.

Salida del recorrido:

```text
RESP2_RESP5_CHROME_OK: actas ODT de aceptación y cierre confirmadas; aceptación firmada 2022, aceptación Word 2023 + cierre PDF 2026, datos confirmados y adultos en ficha; permisos propio equipo, sin almacenamiento, 1440/1024/390/360 y errores sin datos.
```

Las capturas de carga se actualizan en `docs/capturas/carga-actas-*`. No cambia
la disposición de la pantalla. La revisión independiente es estática; la evidencia
de navegador y ejecución corresponde a las pruebas anteriores.
