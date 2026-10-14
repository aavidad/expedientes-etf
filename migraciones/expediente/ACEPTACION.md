# F1.05a: aceptación en el servidor

Coordinación confirma la fecha real del Acta, municipio, centro, equipo,
subprograma, puesto profesional TS/ES/PS, sexo H/M de cada menor por su orden
y descripción de la situación. La petición incluye la versión leída y una
cabecera `Idempotency-Key` de entre 1 y 128 caracteres alfanuméricos, con
puntos, guiones, guiones bajos o dos puntos. El cuerpo es JSON.

La persona profesional es opcional. Si se indica su identificador, el puerto
de identidad comprueba que está activa, tiene perfil profesional y pertenece
al equipo elegido. El adaptador actual no ofrece adscripción ni perfiles;
esa selección requiere integrar F0.10. Se deniega mientras falten esos datos.

Rutas montadas mediante `Opciones.Expedientes`:

- `POST /api/coordinacion/derivaciones/{id}/aceptar`: Coordinación, sesión
  vigente, origen permitido y testigo CSRF. Devuelve 200 con número, equipo,
  año, correlativo, versión, estado, fecha del Acta y versión de la regla.
- `GET /api/expedientes/{id}`: Coordinación o profesional del equipo asignado,
  con auditoría confirmada antes de responder. También se admite
  `GET /api/coordinacion/expedientes/{id}`.

La regla vigente en la fecha del Acta se lee del catálogo. La carga provisional
D1 usa equipo-correlativo-año, dos cifras por componente y guiones. El límite
es 99 por equipo y año. No se impone en Go: se calcula a partir de la anchura
publicada. Municipio, centro y equipo se confirman por separado; la propuesta
geográfica no obliga a conservar la asignación sugerida.

Una transacción bloquea la clave por usuario, comprueba repetición y versión,
bloquea el expediente, incrementa el contador e inserta aceptación, menores,
episodio, hito, cambio de estado, historia estadística, auditoría y recibo.
Cualquier fallo revierte el contador. La misma clave y petición devuelven el
recibo durante 24 horas; cambiar contenido o destino con esa clave devuelve
422. Después de 24 horas, el estado y la versión siguen impidiendo aceptar
de nuevo un expediente. El rechazo queda auditado en otra transacción.

El estado queda `aceptada`, fase `recepcion`. Se conserva un hito de aceptación
con origen `documento_externo`, sin emisión enlazada. La generación del Acta
y oficios corresponde a F1.06. La pantalla de aceptación queda pendiente.

## Comprobación local

`python3 -B migraciones/expediente/ensayar_aceptacion.py` levanta PostgreSQL 18
y ClamAV desechables en una red interna. Usa contraseñas temporales, datos
inventados y almacenamiento en memoria. Aplica todas las migraciones sobre
versión anterior y base limpia, comprueba repetición y ejecuta pruebas de
aceptación concurrente, idempotencia, RLS, campos obligatorios, límite,
conflicto de versión, auditoría y formato del catálogo. Incluye la regresión
F1.01. Destruye los recursos propios al terminar.

Las migraciones nuevas son `catalogo/0004` y `expediente/0002`, reservadas en
los registros externos de ETF y VEC. No se cambia SQL ya aplicado. La revisión
independiente y los resultados se anotan en `EVIDENCIA_ACEPTACION.md`.
