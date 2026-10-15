# SEG3: control de la carga histórica

Decisión R-21 del responsable del producto del 14/10/2026. Rama `seg-carga-lotes`.

Administración fija una fecha de cierre, que puede ampliar con versión, historia y auditoría. Sin fecha configurada, la carga está cerrada. La demostración inventada configura el 31 de diciembre de 2026; esa fecha no se impone en producción. Los controles inmediatos y diferidos de PostgreSQL impiden confirmar cargas o anulaciones que terminen fuera del plazo.

Coordinación puede anular un lote completo dentro del plazo si no tiene actividad posterior. Se retiran sus fichas y reservas históricas, se conserva la auditoría y el recibo técnico del lote. Las consultas no bloquean; los cambios, evaluaciones y documentos sí. El marcador de cambios es monotónico y el rol de aplicación no puede resetearlo. Los lotes anteriores a SEG3 se protegen si no hay evidencia suficiente de que estén intactos. La exportación administrativa permite consultar la auditoría conservada; no se añade acceso a fichas para Administración.

La carga compara familia, personas adultas y menores conocidos, también entre filas de un lote, con una ventana inicial de 30 días configurable de 0 a 366. Un aviso exige confirmar expresamente que son expedientes distintos. El número ocupado se avisa y sigue bloqueado por unicidad. Un lote que agotaría los correlativos ordinarios disponibles se rechaza; la anulación nunca reduce el contador ordinario.

## Evidencia local

- `python3 -B migraciones/expediente/ensayar_aceptacion.py --pruebas 'TestSEG3|TestCargaHistorica|TestHoja|TestTraslado|TestSeguimiento'`: `ENSAYO-OK expediente: base limpia y anteriores con datos; selección de pruebas TestSEG3|TestCargaHistorica|TestHoja|TestTraslado|TestSeguimiento`. PostgreSQL 18 desechable; actualización y repetición del migrador, RLS, plazo cerrado y ampliado, duplicados sin y con confirmación, anulación de histórico cerrado, cuatro anulaciones simultáneas, numeración protegida, rechazo por evaluación que conserva todas las fichas y emisión concurrente con anulación.
- `python3 -B migraciones/estadistica/ensayar.py`: `ENSAYO-OK instalación limpia`; incluye la regresión de rol técnico y las lecturas concurrentes de Memoria.
- `node scripts/recorridos/seg3.mjs http://127.0.0.1:18423`: Chrome del sistema, sesión real de Administración y `ines.moreno`; cierre, propuesta conservada ante conflicto, coincidencias, confirmación por teclado y respuesta perdida tras una anulación ya confirmada. La pantalla bloquea otra escritura hasta actualizar el listado. Comprobaciones a 1440, 1024, 390 y 360 px, sin desplazamiento horizontal; alto contraste e inglés; campos con etiqueta y altura mínima; sin almacenamiento de datos en el navegador.
- Semgrep local con `web/src/carga/seguridad.yml` y `scripts/ci/equipo/semgrep-local.yaml`, métricas y comprobación de versión desactivadas: cero hallazgos. No se envió código ni datos a un servicio de análisis.
- Revisión independiente de backend, SQL y seguridad focal con `revisar-cambios-etf`, `revisar-sql-etf` y `security-audit`: GO de fuente. La revisión independiente no ejecutó el ensayo; la evidencia de ejecución corresponde al integrador.

Revisión independiente de usabilidad y visual: GO tras corregir plazo visible, errores de campo, propuesta conservada, incertidumbre de respuesta, foco y referencia estable del lote. Se inspeccionaron las capturas a 1440/390, incluidos inglés y alto contraste. El recorrido prueba también la paginación con 101 metadatos simulados por Chrome, fallo recuperado y cambio de idioma; las escrituras y la anulación descritas arriba usan el servidor real. La matriz independiente de permisos diferencia el plazo común, su edición administrativa, la lista de lotes por ámbito y la anulación exclusiva de Coordinación.

Migración reservada con `siguiente.sh expediente`: `0028`, anotada en `ETF_RESERVAS_MIGRACIONES.md`. No se modifica una migración anterior. Enganche nuevo: `cmd/etf/control_cargas.go` en `cmd/etf/modulos.go`; las pantallas pertenecen al registro existente de `web/src/carga/pantallas.ts`. La CI incorpora el recorrido SEG3 antes de las regresiones manual, hoja y actas.

La CI integrada detectó un permiso omitido en los comprobadores internos usados por las vistas del rol técnico de Memoria. La regresión reprodujo `42501: permission denied for function anulacion_carga_interna`. Se concede sólo la ejecución de los comprobadores booleanos, sin acceso a fichas ni a las tablas de credenciales o a la función de anulación. El marcador de cambios abre únicamente su recurso y lote mediante una credencial temporal de la propia transacción, en vez de una política general de propietario. La prueba verifica proyecciones, denegación de fichas y vacío sin contexto; el ensayo estadístico la ejecuta antes de las suites concurrentes.

La CI completa sobre `main` integrado terminó con `CI mínima completada.`. Incluye `go vet ./...`, `go test ./...`, TypeScript, pruebas del frontend, textos, migraciones, PostgreSQL/RLS y Chrome. La matriz de navegador comprobó 1.758 combinaciones de perfiles, rutas y métodos; el ensayo RLS verificó 378 lecturas, 810 escrituras denegadas y 63 lecturas de nueve vistas. Los recorridos de carga manual, hoja y Actas también pasaron.

Enganches y ajustes comunes: registro de módulos en `cmd/etf/modulos.go` e `internal/adaptadores/http/modulos.go`; expectativas de menú y sesión; matriz independiente de perfiles; `scripts/ci/comprobar.sh`; ensayo estadístico; inicialización del plazo sólo en la demostración inventada. La copia de prueba RLS omite identidades y columnas generadas, manteniendo las denegaciones esperadas. No cambian pantallas de otros módulos.

## Límites

La coincidencia es un aviso sobre identidades conocidas, no una identificación automática. Si los nombres faltan o las fechas exceden la ventana configurada, no se deduce que sea la misma familia. El número ocupado nunca se puede asignar dos veces aunque se marque la confirmación. La anulación conserva los eventos de auditoría y elimina las fichas del lote intacto; no permite retirar datos de un expediente que ya tenga actividad posterior.

No se ejecutó un lector de pantalla interactivo. Las comprobaciones automáticas verifican etiquetas, roles, foco y teclado.

Las pruebas y capturas usan datos inventados. Esta entrega no autoriza cargar datos reales ni sustituye la EIPD o las puertas operativas de producción.
