# Recepción de derivaciones · F1.01 servidor

La migración `0001_derivacion_recibida.sql` crea el expediente sin número,
equipo ni municipio asignados, su lectura de anexo y los datos extraídos. La
propuesta municipal requiere confirmación humana. No hay columna para el
fichero, su nombre original ni texto completo del documento.

Los nombres municipales y las variantes de búsqueda se leen de
`idiomas/<código>/catalogos/ubicaciones.json` y `variantes_municipios.json`.

## Contrato HTTP

- `POST /api/coordinacion/derivaciones`: una parte multipart llamada `anexo`,
  con nombre terminado en `.docx` o `.pdf`. Máximo 10.000.000 bytes de fichero;
  el cuerpo permite además 64 KiB para cabeceras multipart. Devuelve `201`,
  la derivación y su dirección de recuperación en `Location`.
- `GET /api/coordinacion/derivaciones/{id}`: devuelve la derivación sólo tras
  confirmar la auditoría de su lectura.
- Ambas rutas usan la sesión vigente y exigen Coordinación. La subida exige
  además `Origin` y `X-CSRF-Token` del contrato de sesión existente. No se toman
  permisos de cabeceras, parámetros ni campos del formulario.

`Opciones.Derivaciones` recibe el puerto de entrada. `actorDerivaciones` es el
punto único de conexión con la identidad. La concesión persistente de perfiles
de F0.10 todavía no está en esta rama; hasta su integración se deniega el acceso
a sesiones sin ese perfil. No hay una excepción de desarrollo para subir datos.

## Validación y descarte

La validación precede al antivirus y la extracción. DOCX limita a 256 partes y
32 MiB expandidos, cotas más estrictas que la política general. Rechaza macros,
rutas inseguras, enlaces de fichero, archivos anidados y XML con declaraciones
de entidades. Sólo admite partes propias del documento y sus imágenes.

PDF usa el lector existente con su perfil de datos y motor `dslipak`. Se ejecuta
en un proceso efímero por tuberías, con cinco segundos de plazo, tres de CPU,
96 MiB de límite de datos y 2 GiB de espacio virtual; no produce volcados de
memoria. La ruta admite una subida en proceso cada vez. Un PDF sin texto se
guarda como derivación pendiente de teclear, con aviso. Ambos formatos rechazan
más de quince menores.

El fichero se lee mediante `MultipartReader`, se descarta y se limpia su buffer
al terminar. No se llama a `ParseMultipartForm`, `FormFile` ni a herramientas
que escriban temporales. La huella SHA-256 y el resultado del antivirus son
metadatos de la lectura. ClamAV infectado bloquea; su caída sólo en este caso de
uso permite guardar con aviso y un evento adicional de auditoría.

Se excluyen los números de identificación personal. VALÓRAME pertenece al
módulo de evaluación, según `ESQUEMA_BACKEND.md`; esta entrega mínima de
expediente no lo persiste en `derivacion_datos`. Su guardado por el futuro puerto
de evaluación queda pendiente. La pantalla de revisión llegará con el armazón.

Los anexos reconocidos, incluidos los PDF, conservan un aviso explícito de
VALÓRAME pendiente de registrar en evaluación.

## Ensayo local

Ejecutar `python3 -B migraciones/expediente/ensayar.py`. Usa las imágenes fijadas
de PostgreSQL 18 y ClamAV, una red interna propia y contraseñas temporales.
Aplica la versión anterior, la nueva, la repetición y una instalación limpia.
Comprueba RLS, guardado y recuperación, auditoría, reversión ante fallo de
auditoría, EICAR en DOCX, antivirus limpio y caída real del antivirus.

La base y los temporales del antivirus usan tmpfs; los contenedores del ensayo
tienen desactivado el intercambio a disco. Las firmas de ClamAV son sintéticas,
sin descarga externa. Se destruyen todos los recursos creados por el ensayo.
Las pruebas Go incluyen Inotify para detectar incluso temporales creados y
borrados durante éxito, error, cancelación, infección y caída del antivirus.

La ausencia de volcado de cuerpos en el proxy y de intercambio a disco en el
despliegue final requiere comprobación operativa. Este encargo de servidor no
cambia la configuración de producción ni acredita ese recorrido de despliegue.
