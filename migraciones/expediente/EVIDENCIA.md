# Evidencia de F1.01 servidor

Rama: `f1.01-subida-anexo`. Fecha: 13 de octubre de 2026.

## CI completa

Comando: `make ci`. Código de salida: 0. Incluye `go vet ./...`,
`go test ./...`, compilación de servidor y migrador, comprobación de arquitectura,
textos, temas, enlaces, secretos y las pruebas del frontend existente.

Última línea real:

```text
CI mínima completada.
```

Resultados Go pertinentes de esa ejecución:

```text
ok   github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/lectura 0.162s
ok   github.com/aavidad/expedientes-etf/internal/adaptadores/http 0.056s
ok   github.com/aavidad/expedientes-etf/internal/aplicacion/expediente 0.199s
```

Las pruebas cubren DOCX real del corpus, PDF de ambos anexos y PDF sin texto,
macros por extensión, parte y tipo de contenido, bomba ZIP, rutas inseguras,
entidades XML, tamaño, máximo de menores, propuestas con variantes reales del
catálogo, autorización por perfil, sesión ausente o caducada, CSRF, multipart,
descarte del buffer, auditoría y cancelación. Inotify comprueba que no se crean
temporales en el recorrido de aplicación, incluido el proceso PDF.

Las pruebas de integración PostgreSQL se omiten en `go test ./...` cuando no
hay servicio explícito; sí se ejecutaron en el ensayo siguiente.

## PostgreSQL 18 y ClamAV reales

Comando: `python3 -B migraciones/expediente/ensayar.py`. Código de salida: 0.
Base desechable, red interna propia, datos inventados, tmpfs y contenedores sin
intercambio a disco. Se destruyeron los contenedores y la red al terminar.

Salida pertinente:

```text
ENSAYO-OK versión anterior, migración nueva y repetición
--- PASS: TestIntegracionGuardarRecuperarRLSYAuditoria (0.05s)
--- PASS: TestIntegracionFalloAuditoriaRevierteTresTablas (0.04s)
--- PASS: TestIntegracionClamAVRealEICARLimpioYCaido (0.13s)
PASS
ENSAYO-OK base limpia, RLS, auditoría, EICAR y ClamAV caído
```

La firma de ensayo EICAR se introdujo en memoria dentro de un DOCX válido.
ClamAV la rechazó antes de leer o guardar datos. Su caída se ensayó contra un
puerto local cerrado. La recuperación confirma la misma derivación y el aviso;
la exportación de auditoría verifica los eventos de subida, aviso y lectura.
Un fallo de auditoría revierte las tres tablas. Administración, Consulta,
Profesional ETF y una identidad sin perfiles no ven sus filas. Se ejecutó
`EXPLAIN (FORMAT JSON)` sobre la recuperación por identificador.

## Seguridad y revisión independiente

Semgrep ejecutado localmente con métricas y comprobación de versión desactivadas,
usando `internal/adaptadores/anexos/lectura/semgrep.yml` sobre los ficheros de
producción del cambio y los enganches. Código de salida: 0.

```text
Ran 3 rules on 19 files: 0 findings.
```

El agente independiente `revision_f101` revisó el código staged, SQL/RLS,
auditoría, identidad, HTTP y lectores mediante `revisar-cambios-etf`,
`revisar-sql-etf` y revisión focal de `security-audit`. Resultado final:
«GO de revisión del código staged». No ejecutó de nuevo el ensayo ni Semgrep;
esas ejecuciones corresponden al autor de la entrega.

## Límites de la entrega

La pantalla queda pendiente del armazón. La concesión persistente de perfiles
F0.10 no está integrada; la sesión existente y el punto único de autorización
deniegan por defecto. VALÓRAME no se guarda en expediente: se conserva su aviso
de evaluación pendiente, también en PDF. La ausencia de volcado de cuerpos en
el proxy y de intercambio a disco en producción requiere comprobación de
despliegue; este ensayo local no la acredita.
