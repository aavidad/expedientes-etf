# Salida al SPM · F3.01a

Reservas del director: expediente/0005 y documento/0006. Cada migración escribe
sólo las tablas de su módulo; documento depende del puerto de bloqueo publicado
por expediente/0005 y de la lectura profesional de catálogo/0007. No depende de expediente/0004 ni de consentimiento.

La salida tiene versión propia. Crear compara la versión de la ficha; confirmar
fecha, responder o registrar un forzado comparan la de la salida. La ficha
conserva estado y versión al preparar. FX1 (`expediente/0022`) conecta la
fecha confirmada desde `en_intervencion` con `derivada_spm` y la respuesta
habilitante fechada con el regreso a `en_intervencion`. Estas transiciones
incrementan la versión de la ficha y añaden historia en la transacción auditada;
repetir el recibo no las duplica. Desde `aceptada` se conserva el contrato
previo, sin inventar consentimiento. Una salida pendiente bloquea la consulta ordinaria;
las dos respuestas habilitantes requieren fecha. La historia conserva todas las
versiones, aunque caduque el recibo idempotente. El forzado no cambia la respuesta.

F3.02b debe consultar el puerto dentro de su transacción y tomar los bloqueos en
este orden: salida SPM, ficha, cierre. La auditoría se confirma antes de entregar
resultados. El motivo del forzado permanece en su registro protegido y el evento
de auditoría enlaza el identificador de esa acción.

Pruebas: `python3 -B migraciones/expediente/ensayar_aceptacion.py` y
`python3 -B migraciones/documento/ensayar.py`. Ambos crean bases PostgreSQL 18
desechables, aplican migraciones y ejecutan pruebas con datos inventados.
El ensayo de documento incluye descarga HTTP del Anexo 5, apertura del ZIP DOCX,
comprobación de marcadores y repetición después de medianoche en Madrid.
Las plantillas conservan el formato oficial castellano en es/en.
