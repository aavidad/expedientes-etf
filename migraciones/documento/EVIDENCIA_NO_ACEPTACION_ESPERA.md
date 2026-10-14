# Evidencia de F1.07b: No Aceptación y Lista de Espera

Se activan `entrada/no_aceptacion` y `entrada/no_aceptacion/lista_espera` por
`POST /api/documentos/emisiones`, con Coordinación, sesión, CSRF e idempotencia.
La migración documento/0003 conserva las plantillas oficiales 1.0.0 y activa sus
cuatro referencias es/en; depende de documento/0002 y expediente/0003. El número
está reservado fuera de Git en el registro de migraciones de ETF.

El puerto de expediente entrega la derivación, su decisión y la recepción dentro
de la transacción recibida, con `FOR SHARE`. No Aceptación usa motivo, subprograma
y fecha confirmados; Lista de Espera usa su fecha de decisión y la recepción para
el oficio. F1.05b deja número, equipo y municipio nulos antes de aceptar: los huecos
correspondientes quedan vacíos, igual que el sexo no confirmado. No se confirma
una propuesta automática ni se modifica un hito al emitir.

Pruebas locales con datos inventados:

```sh
python3 -B migraciones/documento/ensayar.py
ETF_DOCUMENTO_PROBAR_LIBREOFFICE=1 ETF_DOCUMENTO_EVIDENCIA=/tmp/etf-f107b-evidencia go test -count=1 -v ./internal/aplicacion/documento -run TestNoAceptacionEsperaEmisionesUnoSeisQuinceMenores
go vet ./...
go test ./...
go test -race ./internal/aplicacion/documento ./internal/adaptadores/http ./internal/adaptadores/postgres/documento ./internal/adaptadores/postgres/expediente
```

El ensayo usa PostgreSQL 18 desechable sin red externa. Aplica documento/0003
sobre la versión anterior, repite el migrador y verifica una base limpia. La matriz
HTTP real cubre No Aceptación de Anexo 3 y 4 y Lista de Espera de Anexo 3, con
1, 6 y 15 menores, en es/en: dieciocho emisiones y sus reproducciones binarias.
Comprueba cada menor exactamente una vez, motivo y subprograma, fecha de recepción,
registro pendiente sólo del oficio y decisión conservada. La recepción a las
22:30 UTC del 6 de octubre se presenta como 7 de octubre, fecha civil de Madrid. Pedir estos documentos
en estados incompatibles devuelve 409 con las claves es/en correspondientes.
Equipo, Administración y Consulta no leen ni la emisión ni la decisión por RLS.
La lectura sin transacción se rechaza.

Salida real conservada en `/tmp/etf-f107b-postgres-fecha.log`:

```text
--- PASS: TestDecisionesDocumentosPostgresHTTPPuertoReal (0.46s)
ENSAYO-OK base limpia, RLS, auditoría, concurrencia e integridad
```

LibreOffice abrió los seis documentos en castellano y produjo PDF válido.
La matriz de emisiones en memoria cubre ambos idiomas, los tres tamaños,
repetición idéntica y perfiles denegados. Resultado real:

```text
PASS
ok  github.com/aavidad/expedientes-etf/internal/aplicacion/documento 4.658s
```

PDF y DOCX de quince menores quedan en `/tmp/etf-f107b-evidencia`, fuera del
repositorio; salida en `/tmp/etf-f107b-libreoffice.log`. La conversión es activable
y no forma parte de producción. No se ha hecho una revisión visual de las páginas.

`go vet ./...` y `go test ./...` terminaron con código 0. Las pruebas con detector
de carreras de los cuatro paquetes indicados terminaron correctamente. Una
ampliación inicial de la prueba con carreras a todo el paquete de aplicación de
expediente falló en dos pruebas de su lector PDF con `anexo.limite_recursos`.
El lector PDF no cambia en esta entrega; sus pruebas normales pasan en la puerta
completa. La salida de esa ampliación está en `/tmp/etf-f107b-race.log`.

Textos: ocho pruebas aprobadas, cero fallidas; 214 claves usadas en dos idiomas,
sin literales visibles. Semgrep se ejecutó sólo en local, con reglas de
`semgrep-local.yaml`, métricas y consulta de versiones desactivadas:

```text
Ran 5 rules on 16 files: 0 findings.
```

Revisión independiente focal de código, SQL, permisos y datos personales:
**GO sobre 30762d3**, sin hallazgos bloqueantes; confirmado sobre la candidata
integrada **9e9a331**, cuyo código, SQL e idiomas son idénticos. La revisión
confirmó después **GO sobre 799d466** tras incorporar DEMO y **GO sobre 378d5eb**
para la corrección de fecha civil de Madrid. El revisor leyó la candidata y sus
dependencias, comprobó las cuatro huellas de plantillas y leyó las evidencias;
no repitió los ensayos. La regla de autorización HTTP existente ya reserva el
POST a Coordinación; no ha sido necesario cambiar la tabla de rutas.

La CI completa se ejecuta con `npm_config_offline=true make ci`; su salida se
conserva en `/tmp/etf-f107b-ci-integrada.log`. La ejecución final, con DEMO
integrado y la corrección de fecha, terminó con código 0; última línea real:

```text
CI mínima completada.
```

Incluye instalación offline con versiones fijadas, comprobaciones de frontend,
textos, temas, secretos, arquitectura, `go vet ./...`, `go test ./...`, compilación
y ambos ensayos completos de PostgreSQL. La máquina sufrió saturación de E/S
durante el ensayo final; éste terminó correctamente sin borrar archivos ajenos.

Sin pantallas, sin push y sin integración en main. Se ha ejecutado `git merge main`
en la rama de trabajo: primero `Already up to date.` y, tras avanzar main,
merge 9e9a331 sin conflictos, que incorpora una actualización de instrucciones.
Después se integró el entorno DEMO de main en 799d466, también sin conflictos,
y se regeneró el índice de producto. La CI completa pasó sobre esta integración
y la corrección de fecha ya terminada.

Ficheros de F1.07b (los cambios heredados de main constan en los commits de integración):

```text
docs/producto/ESQUEMA_BACKEND.md
docs/producto/FLUJO_APP.md
docs/producto/INDICE.md
docs/producto/PLAN_IMPLEMENTACION.md
idiomas/en/textos/documento.json
idiomas/es/textos/documento.json
internal/adaptadores/http/documentos.go
internal/adaptadores/http/documentos_test.go
internal/adaptadores/postgres/documento/decisiones_integracion_test.go
internal/adaptadores/postgres/expediente/documentos.go
internal/aplicacion/documento/datos_entrada.go
internal/aplicacion/documento/entrada_test.go
internal/aplicacion/documento/puertos/entrada/emision.go
internal/aplicacion/expediente/documentos.go
internal/aplicacion/expediente/puertos/entrada/documentos.go
internal/dominio/documento/emision.go
internal/dominio/expediente/documentos.go
internal/puertos/salida/expediente/documentos.go
migraciones/documento/0003_no_aceptacion_espera.sql
migraciones/documento/EVIDENCIA_NO_ACEPTACION_ESPERA.md
migraciones/documento/LEEME.md
migraciones/documento/ensayar.py
migraciones/documento/manifiesto.json
```
