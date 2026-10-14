# Evidencia de F1.05a

Rama: `f1.05a-aceptar`. Datos inventados. Ninguna conexión a una base existente.

## Pruebas ejecutadas

`make ci` completo, con puerta de migraciones y aceptación sobre PostgreSQL 18:

```text
ENSAYO-OK F1.05a: base limpia y anterior, 20 concurrentes, idempotencia, RLS, auditoría y numeración vigente
CI mínima completada.
```

La puerta ejecuta `go vet ./...`, `go test ./...`, comprobación de arquitectura,
textos e idiomas, pruebas y compilación del frontend, detección de secretos y
compilación de los ejecutables. El ensayo instala desde cero y sobre la
versión anterior, repite la instalación y ejecuta las pruebas reales dos veces.

Las pruebas demuestran 20 aceptaciones simultáneas con números consecutivos,
20 repeticiones con un solo número y hito, rechazo de una clave con contenido
o destino distinto, reversión del contador si falla auditoría, obligatoriedad
del puesto profesional y sexo de todos los menores, RLS por perfil y equipo,
límite de 99, conflicto de versión y formato leído del catálogo con vigencia.
Incluyen regresiones de subida de anexo, ClamAV y descarte del fichero.
El recorrido HTTP usa aplicación, catálogo y PostgreSQL reales; sustituye
únicamente la sesión, obtiene CSRF, acepta, repite el recibo idéntico y
recupera la ficha auditada.

`go test -race` pasó en dominio de catálogo, dominio de expediente y HTTP.
El intento adicional sobre todo el paquete de aplicación de expediente falló
en las pruebas previas de PDF: el proceso hijo devolvió
`anexo.limite_recursos` bajo la instrumentación de carreras. Esas pruebas
pasaron con `go test ./...` y en el ensayo. No se amplió el límite del lector.

Semgrep se ejecutó localmente sobre todos los Go de producción cambiados,
con cuatro reglas locales, métricas desactivadas y sin consultar el registro.
Resultado: `Ran 4 rules on 17 files: 0 findings.`
No se enviaron código ni datos a servicios de análisis.

## Revisión

La revisión focal propia comprobó permisos de Coordinación, sesión y CSRF,
consultas parametrizadas, lectura auditada, RLS de hijos por equipo, claves
compuestas, identidad profesional mediante puerto y atomicidad del contador.
Revisión independiente en solo lectura: **GO para `f3db8c2`**, sin
bloqueantes confirmados en SQL, permisos, HTTP ni integración. No repitió
pruebas ni levantó servicios: comprobó fuente y evidencia ejecutada. Señaló
una discrepancia menor entre el nombre de la tabla de numeración en el
esquema y la migración; se corrigió en la documentación.

## Pendiente

Pantalla de aceptación y generación de documentos. La selección de una persona
requiere F0.10: el adaptador actual no devuelve perfiles ni adscripción. El
puesto profesional se puede confirmar; un identificador de persona sin esa
acreditación se rechaza.

## Ficheros modificados

- `cmd/etf/aceptaciones.go`
- `cmd/etf/main.go`
- `docs/producto/ESQUEMA_BACKEND.md`
- `docs/producto/FLUJO_APP.md`
- `docs/producto/INDICE.md`
- `docs/producto/PLAN_IMPLEMENTACION.md`
- `idiomas/en/textos/aceptacion.json`
- `idiomas/en/textos/expediente.json`
- `idiomas/es/textos/aceptacion.json`
- `idiomas/es/textos/expediente.json`
- `internal/adaptadores/http/aceptaciones.go`
- `internal/adaptadores/http/aceptaciones_test.go`
- `internal/adaptadores/http/servidor.go`
- `internal/adaptadores/postgres/catalogo/consulta.go`
- `internal/adaptadores/postgres/catalogo/numeracion.go`
- `internal/adaptadores/postgres/expediente/aceptacion.go`
- `internal/adaptadores/postgres/expediente/aceptacion_http_integracion_test.go`
- `internal/adaptadores/postgres/expediente/aceptacion_integracion_test.go`
- `internal/adaptadores/postgres/expediente/derivaciones.go`
- `internal/aplicacion/expediente/aceptacion.go`
- `internal/aplicacion/expediente/derivaciones.go`
- `internal/aplicacion/expediente/puertos/entrada/aceptacion.go`
- `internal/dominio/catalogo/datos.go`
- `internal/dominio/catalogo/numeracion.go`
- `internal/dominio/catalogo/numeracion_test.go`
- `internal/dominio/expediente/aceptacion.go`
- `internal/dominio/expediente/derivacion.go`
- `internal/puertos/salida/catalogo/numeracion.go`
- `internal/puertos/salida/expediente/aceptacion.go`
- `migraciones/catalogo/0004_numeracion.sql`
- `migraciones/catalogo/manifiesto.json`
- `migraciones/expediente/0002_aceptacion.sql`
- `migraciones/expediente/ACEPTACION.md`
- `migraciones/expediente/EVIDENCIA_ACEPTACION.md`
- `migraciones/expediente/ensayar_aceptacion.py`
- `migraciones/expediente/manifiesto.json`
- `migraciones/expediente/seguridad_aceptacion.yml`
- `scripts/ci/comprobar.sh`
