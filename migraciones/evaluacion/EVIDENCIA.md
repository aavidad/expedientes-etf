# Evidencia local de F2.05

Comando ejecutado el 14/10/2026 con PostgreSQL 18 desechable y datos inventados:

```sh
TMPDIR=bin/temporales python3 migraciones/evaluacion/ensayar.py
```

Salida del ensayo corregido, con la prueba de integración ejecutada:

```text
ENSAYO-OK base anterior, ampliación y repetición
--- PASS: TestEvaluacionPersistenciaHTTPRLSAuditoria (0.23s)
PASS
AUDITORIA-OK 16 0
ENSAYO-OK instalación limpia
```

El ensayo rechaza una prueba omitida, exige el resultado PASS de la prueba seleccionada y falla si no hay eventos de evaluación. Una primera ejecución omitió la prueba por una variable de activación incorrecta; esa ejecución sólo acreditaba la aplicación de migraciones y se sustituyó por el resultado anterior.

Se comprobaron conservación de cada respuesta, adulto vigente, recuperación de la versión, conflicto 409, repetición idéntica, rechazo de clave con contenido distinto, aislamiento por equipo, perfiles sin permiso, CSRF y método no permitido. La caída de auditoría revierte la escritura y evita entregar lecturas. Las pruebas SQL directas rechazan metadatos incompletos, sujeto sin tipo, adulto de otro expediente y respuesta vinculada a otro expediente. Se rechaza editar la derivación y usar sujetos menores o profesionales sin sus puertos pendientes.

La revisión independiente fue de código y SQL en modo de sólo lectura, con `revisar-cambios-etf`, `revisar-sql-etf` y `security-audit` focal. Detectó y verificó la corrección de restricciones CHECK que aceptaban NULL y de la lectura partida entre metadatos y respuestas. Semgrep local: tres reglas, doce archivos, cero hallazgos; Gitleaks sin secretos. El revisor no repitió la ejecución por las restricciones de su entorno de auditoría. La repetición independiente del ensayo antes de fusionar sigue correspondiendo al director.

Esta evidencia acredita la infraestructura, con una escala técnica inventada para pruebas. Las escalas oficiales siguen pendientes de sus cuestionarios y respuestas admitidas por ítem (D17); no se calcularon totales ni se inventaron baremos. No hay pantallas.
