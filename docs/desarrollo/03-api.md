# 3. API HTTP, sesiones y CSRF

El adaptador HTTP convierte una petición limitada y validada en una llamada al puerto de entrada. La identidad procede de la sesión del servidor. Los UUID y perfiles enviados por el navegador no sustituyen esa identidad.

## Registro de rutas

Cada capacidad declara método, patrón, handler y regla de permiso en su módulo HTTP. El registro también declara las rutas de pantalla. El arranque rechaza duplicados, métodos vacíos, permisos incompletos y escrituras sin la protección exigida. Una ruta desconocida queda denegada.

Consulta [internal/adaptadores/http](../../internal/adaptadores/http/) para localizar la declaración actual. El [esquema de API](../producto/ESQUEMA_BACKEND.md#7-api) explica los contratos, pero incluye rutas de diseño: para comprobar disponibilidad, contrasta el módulo con [la composición](../../cmd/etf/modulos.go).

| Capacidad | Ruta actual de referencia | Perfil |
| --- | --- | --- |
| Sesión | `GET`, `POST`, `PUT` y `DELETE /api/sesion` | Entrada y sesión propia |
| Plantillas | `GET /api/admin/plantillas` | Administración |
| Publicación de plantilla | `POST /api/admin/plantillas/versiones` | Administración, con CSRF |
| Activación | `POST /api/admin/plantillas/{id}/activar` | Administración, con CSRF y versión previa |
| Bandeja de equipo | `GET /api/equipo/expedientes` | Profesional de su equipo y Coordinación |
| SPAF | `GET /api/estadistica/spaf?anio=&mes=&etf=` | Profesional de su equipo y Coordinación |
| Word SPAF | `POST /api/equipos/{etf}/tabla-spaf/{anio}/{mes}.docx` | Misma autorización; crea emisión |
| Memoria de Consulta | `GET /api/consulta/memoria/{anio}?equipo=` | Consulta; contrato numérico propio |

No conviertas la generación de un Word en GET: crea una emisión y necesita protección de escritura. La confirmación de una firma o fecha real es otra acción; descargar un documento no confirma el hito.

## Entrada y sesión

El adaptador LDAPS comprueba la cuenta y vincula el identificador estable del directorio. La aplicación decide perfiles y equipo. Una cuenta válida sin perfil queda pendiente de alta. La concesión inicial de Administración se realiza una vez y se audita; retirar después ese perfil no permite recuperarlo por reiniciar el servicio.

[sesion.go](../../internal/adaptadores/http/sesion.go) define las cookies `__Host-etf-sesion` y `__Host-etf-csrf`. Son `Secure`, `HttpOnly`, con `Path=/` y `SameSite=Strict`. El token de sesión es opaco; el navegador no recibe una lista de permisos firmada que pueda usar como identidad independiente del servidor.

La respuesta de `GET /api/sesion` publica el estado, el testigo CSRF y las pantallas permitidas. SEG1 hace esta lectura pasiva: no renueva actividad. `PUT /api/sesion`, con CSRF y origen comprobados, continúa una sesión vigente sin desplazar el límite absoluto. Las respuestas autenticadas incluyen cabeceras con caducidad, máximo y hora del servidor para el aviso de dos minutos. [web/src/sesion.tsx](../../web/src/sesion.tsx) consulta esa respuesta y ajusta la navegación a `sin_sesion`, `pendiente_alta` o `activo`. No copies la contraseña ni el token de sesión a localStorage o sessionStorage.

## Protección CSRF y origen

La configuración usa el origen HTTPS público exacto y una clave CSRF de 32 bytes. El servidor contrasta `Origin`, exige una única cabecera `X-CSRF-Token` y comprueba el testigo mediante HMAC y comparación constante. Cuando existe `Sec-Fetch-Site`, debe indicar el mismo origen. Una petición de escritura sin sesión no obtiene permiso por llevar un token inventado.

El origen no se deriva de `Host` ni de `Forwarded` del cliente. Para la IP, sólo se admite `X-Forwarded-For` desde los proxies configurados; una cadena mal formada invalida la petición. Conserva esas comprobaciones al añadir un proxy o un endpoint.

La demo local adapta su origen exacto a ese contrato HTTPS mediante su proxy propio. Ese comportamiento se limita al entorno de demostración; no es una instrucción para desactivar cookies seguras en un despliegue.

## Petición, errores y recibos

Limita el cuerpo antes de decodificarlo. Valida formato, campos y tipo real de archivo. Pasa el contexto de cancelación al caso de uso. El handler no ejecuta SQL ni decide por sí solo si una persona puede modificar un expediente.

Los errores llevan claves y, cuando el contrato lo necesita, parámetros y campos. La interfaz los resuelve en `idiomas/<codigo>/textos/`. Reutiliza el formato del handler de la capacidad: existen respuestas con clave simple y otras con `error.clave`. [La API de Plantillas](../../web/src/plantillas/api.ts) admite ambas. No añadas frases castellanas a una respuesta Go.

| Estado HTTP | Significado en la aplicación |
| --- | --- |
| 400 | Cuerpo o parámetros mal formados |
| 401 | Falta una sesión válida |
| 403 | La operación no está autorizada |
| 404 | Recurso inexistente o fuera del ámbito, según el contrato |
| 409 | Conflicto de versión, estado o reproducción |
| 413 | Cuerpo o archivo por encima del límite |
| 415 | Formato no admitido |
| 422 | Incumplimiento de una regla o reutilización incompatible de idempotencia |
| 503 | Servicio necesario no disponible |

Las acciones devuelven un recibo con sus identificadores, versión y resultado. Ante una respuesta perdida, conserva la misma clave de idempotencia y el mismo cuerpo para recuperar la acción. Una petición modificada necesita una clave nueva. Ante 409, compara la versión actual y permite revisar el formulario; no repitas el guardado a ciegas.

Para una lectura protegida, reúne el resultado en memoria, registra el evento y confirma la transacción antes de escribir la respuesta. Si falla la auditoría, devuelve el error sin datos. La misma regla se aplica a descargas y exportaciones.

Fuentes: [sesión y autorización del TRD](../producto/TRD.md#4-identidad-y-sesión), [API](../producto/ESQUEMA_BACKEND.md#7-api) y [F0.11](../recorridos/F0.11.md).
