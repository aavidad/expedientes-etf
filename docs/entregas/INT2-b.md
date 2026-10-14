# Entrega INT2-b: integración de catálogos y paquetes

Rama `int2-f014`. Continúa la integración de F0.14a/b, ya incorporada en
`b45f4eb`, y termina la fusión pendiente de `f0.14c2-activar-paquetes` según
la decisión del director.

Se conserva sin cambios `paquete/0001_versiones_activas.sql` de F0.14c-2.
Se retira `0001_referencias_catalogo.sql` y se reserva `paquete 0002` en
`ETF_RESERVAS_MIGRACIONES.md`, fuera de Git. La nueva migración depende de
`paquete/0001` y `catalogo/0002`: amplía restricciones mediante `ALTER`,
admite versiones semver y huellas SHA-256, y activa catálogos enlazados por
UUID con auditoría en la misma transacción. Mantiene tablas, RLS, permisos
e inmutabilidad. El adaptador consulta ese UUID; la limpieza de catálogos
excluye los idiomas instalados con semver.

La prueba de edición comprueba la convivencia con un paquete semver de
otro idioma y el estado `verificada` del catálogo enlazado. El arnés
`migraciones/catalogo/ensayar.py` ejecuta las pruebas que necesitan Go o
Docker desde el host, contra su PostgreSQL aislado, y comprueba la
actualización desde c2 con una versión existente y el mismo UUID.

## Validación ejecutada

- `make ci`: `CI mínima completada.` Incluye `go vet ./...`, `go test ./...`,
  pruebas y compilación web, textos, temas, arquitectura y secretos.
- `python3 -B migraciones/catalogo/ensayar.py`: catálogo, paquetes, sesión,
  identidad y auditoría pasan en PostgreSQL 18; ninguna prueba de integración
  se omite. Migración limpia y repetida, actualización desde
  `catalogo/0002 + paquete/0001` con UUID y semver conservados, y destrucción
  del contenedor y volumen: `ENSAYO-OK`.
- `python3 /tmp/etf-int2-b-recorrido.py`: fixture temporal de HTTP, LDAPS y
  PostgreSQL reales; Playwright usa `/usr/bin/google-chrome`. Entrada con
  teclado, catálogos y paquetes responden 200 a 1024 px con tema predeterminado
  y a 390 px con alto contraste. `RECORRIDO-OK` en ambos tamaños, sin errores
  de JavaScript ni desbordamiento horizontal. Fixture retirado; contenedor
  y volumen destruidos. Capturas y trazas en `/tmp/etf-int2-b-evidencia`.
- Revisión independiente con `revisar-sql-etf`, `revisar-cambios-etf` y
  `security-audit` focal: GO, sin hallazgos bloqueantes. Semgrep usa reglas
  locales de `despliegue/base/semgrep.yaml`, sin métricas ni envío de código:
  `Ran 4 rules on 24 files: 0 findings.`

## Alcance y documentos

La sesión del recorrido queda pendiente de alta. Las pantallas de
administración se prueban con la barrera provisional de desarrollo que ya
tenían F0.14a/b/c. La autorización administrativa por sesión depende de
F0.10; esta integración no la habilita en producción. El recorrido comprueba
entrada y consulta de las pantallas; instalación, activación, concurrencia
y reversión se comprueban en las pruebas PostgreSQL.

Se actualizan `ESQUEMA_BACKEND.md` y `PLAN_IMPLEMENTACION.md` por esta decisión
y se regenera `INDICE.md`. La fusión incorpora además los cambios documentales
de c2 en `PRD.md` y `FLUJO_APP.md`. La entrega histórica F0.14a/b describía otra
migración base y una auditoría provisional; esta integración usa la base c2
y la auditoría persistente de F0.08b.

Sin cambios adicionales en los enganches comunes; se conservan los de c2
en `cmd/etf/main.go`, servidor HTTP y `web/src/main.tsx`. Sin push.
