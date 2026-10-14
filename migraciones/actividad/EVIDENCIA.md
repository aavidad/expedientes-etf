# Evidencia de F2.04 · servidor de actividad

Validación local del 13/10/2026 sobre la candidata `9b247b1` de
`f2.04-actividad`. Incluye `main` hasta `5bb5039`. El único conflicto de la
integración fue `docs/producto/INDICE.md`, resuelto con su generador.

Migración `actividad/0001` reservada en el registro externo de ETF antes de
crearla. Ensayo con PostgreSQL 18.4 de la imagen fijada, contenedor desechable
sin red y datos inventados. Se aplicaron la base anterior, la ampliación y una
repetición del ejecutor; después se comprobó una instalación limpia.

Comandos ejecutados:

```sh
go test -count=1 ./internal/dominio/actividad ./internal/aplicacion/actividad/... ./internal/adaptadores/http ./internal/adaptadores/postgres/actividad
python3 -B migraciones/actividad/ensayar.py
make ci
```

Las pruebas focales de Go pasaron. Las pruebas HTTP cubren actor desde sesión,
rechazo de equipo en el cuerpo, CSRF, perfiles, equipo ausente, método prohibido,
ruta desconocida, JSON concatenado y año repetido. La prueba PostgreSQL real,
ejecutada también dentro de `make ci`, produjo:

```text
ACTIVIDAD-OK RLS dos equipos, años limítrofes, recuentos 15/16, Otros, conflictos, retirada y reversión de auditoría
--- PASS: TestIntegracionActividadRLSRecuentosAuditoria (0.04s)
PASS
AUDITORIA-OK 25
ENSAYO-OK instalación limpia
```

La RLS se comprobó directamente como `etf_app`, además de las comprobaciones
de la aplicación. Se rechazaron lectura y escritura de otro equipo, traslado
de equipo, recuperación de un retirado, borrado físico y alteración de la
auditoría. Se comprobaron recuentos provinciales y de equipo antes y después
de retirar, excluyendo registros de años vecinos. El fallo de auditoría
revirtió la escritura y bloqueó la entrega de una lectura. Un conflicto
conservó el UUID del registro en el evento de error.

La puerta completa incluyó `go vet ./...`, `go test ./...`, frontend, textos,
temas, arquitectura, secretos y ensayos de expedientes, documentos y actividad.
Última línea:

```text
CI mínima completada.
```

Semgrep local con `migraciones/actividad/semgrep-local.yaml`, métricas y
comprobación de versiones desactivadas: tres reglas sobre los 13 ficheros Go
de producción cambiados; `Ran 3 rules on 13 files: 0 findings.` Gitleaks, dentro
de CI: `no leaks found`. `gopls references` sobre `actividad.Nuevo` terminó con
código 0 y localizó composición y pruebas.

Revisión independiente de fuente con `revisar-cambios-etf`, `revisar-sql-etf`
y `security-audit` en modo focal: GO para `9b247b1`. Se corrigió el hallazgo
de UUID omitido al auditar conflictos. La revisión de enganches tras integrar
`main` confirmó que montaje, permisos y CSRF conservaban el comportamiento.
El revisor no ejecutó pruebas; las salidas anteriores corresponden al autor.

No se añadieron pantallas ni se conectó un motor de Memoria: se entrega su
puerto neutro de cifras. No se hizo push ni se fusionó esta rama en `main`.
