# Evidencia de F0.08b

Rama: `f0.08b-auditoria-bd`. PostgreSQL 18.4 de `compose.yaml`, con su imagen
fijada por digest. Todos los datos y las credenciales del ensayo son inventados.

## Migración y transacciones reales

Orden ejecutada:

```sh
python3 -B migraciones/auditoria/ensayar.py
```

Salida del ensayo fresco:

```text
migracion.aplicada.auditoria.0001
migracion.existente.auditoria.0001
32 inserciones concurrentes: cadena SQL comprobada por el dominio Go
etf_app: INSERT/SELECT/UPDATE/DELETE/TRUNCATE y funciones privadas rechazados; exportación sólo Administración
rollback elimina evento; commit conjunto conserva evento; exportación auditada antes de entregar
{"clave":"auditoria.sello_externo_coincide"}
{"clave":"auditoria.huella_invalida","indice":0,"evento_id":1}
propietario: UPDATE/DELETE/TRUNCATE rechazados; sello repetible; alteración SQL y frontera diaria detectadas
--- PASS: TestIntegracionAuditoria (0.16s)
PASS
ENSAYO-OK: limpia, repetida y actualización desde plataforma/0001
```

El ensayo aplica desde cero, repite sin cambios, ejecuta las pruebas reales de
auditoría y contexto RLS, destruye la base, instala sólo `plataforma/0001` y
actualiza al plan actual. La segunda aplicación vuelve a reconocer las huellas.
Cada contenedor y volumen se destruye al terminar. El proceso de integración
usa binarios de sólo lectura, red interna sin salida y espacio temporal limitado.
Sin `ETF_POSTGRES_INTEGRACION=1`, las pruebas de integración se omiten; con la
variable activada, un fallo de conexión falla la prueba.

## Comprobaciones y revisión

`GOPROXY=off GOSUMDB=off go vet ./...` y `go test ./...`: salida correcta,
incluida la prueba de arquitectura. Semgrep local, sin métricas ni comprobación
de versiones, con `despliegue/base/semgrep.yaml` sobre lo añadido: `Ran 4 rules
on 7 files: 0 findings.` No se enviaron fuentes ni datos a servicios externos.
`./scripts/ci/textos/comprobar.sh`: ocho pruebas correctas y `Textos: 2 idiomas,
52 claves usadas; sin literales visibles.` Se instalaron las dependencias
fijadas del comprobador con `npm ci --offline --ignore-scripts --no-audit
--no-fund`. Índice de producto: `INDICE_OK`; enlaces Markdown: `0 errores`.

Revisión independiente estática con `revisar-sql-etf`, `revisar-cambios-etf` y
`security-audit`: GO de código después de corregir la comprobación del máximo ID
anterior al cierre diario. El ensayo fresco prueba esa corrección moviendo la
fecha de inserción del evento sellado; la simulación completa se revierte.

## Decisiones y límites

El actor `sistema` usa UUID nulo y `actor_sistema = true`, con restricción que
impide confundirlo con una persona. Evita inventar un usuario y conserva la
representación `sistema` de la forma canónica v1. Actor y perfiles proceden del
contexto del servidor. La conexión `etf_app` y ese contexto son de confianza;
ningún endpoint debe aceptar variables RLS enviadas por el navegador.

`registrado_en` delimita los días UTC terminados. Su hora exacta queda fuera de
los bytes v1 firmados; el verificador sí contrasta la frontera de cada sello.
El sello retornado está confirmado. Programación nocturna, escritura en registro
técnico, custodia de copia independiente, HTTP y pantalla quedan para integración
posterior. Sin puntos custodiados fuera del servidor no se demuestra resistencia
frente a un propietario que reescriba toda la cadena y los sellos.

`detalle` continúa pendiente de una versión canónica nueva. No hay dependencias
nuevas ni cambios en otros módulos. `gopls` no está instalado en este equipo;
compilación, vet y pruebas verifican el adaptador añadido.
