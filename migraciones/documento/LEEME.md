# F1.06–F1.08 · documentos de entrada y registro de salida

`POST /api/documentos/emisiones` recibe `tipo`, `expediente_id` e `idioma`, con
sesión de Coordinación, origen autorizado, testigo CSRF e `Idempotency-Key`.
Los tipos activos son `entrada/aceptacion`, `entrada/aceptacion/oficio_sscc`,
`entrada/aceptacion/oficio_spm`, `entrada/no_aceptacion` y
`entrada/no_aceptacion/lista_espera`. Responde con DOCX y cabeceras `X-Emision-ID`,
`X-Documento-SHA256`, `X-Datos-SHA256` y `X-Plantilla-Version`. Una generación
cambiada incluye `X-Aviso-Clave: documento.regenerado`.

El puerto de consulta de expediente entrega la derivación, su recepción y su decisión
versionadas, con bloqueo de lectura dentro de la transacción de emisión.
Documento no consulta sus tablas ni acepta datos familiares por HTTP. Resuelve
ubicaciones por el puerto de catálogo y sus textos por la referencia publicada;
los textos de documento proceden del paquete de idioma fijado para la emisión.
La fecha del Acta procede de la aceptación; emitir no cambia fechas ni hitos.
Si el anexo sólo contiene la edad de un adulto, su fecha de nacimiento queda
vacía: no se calcula ni se inventa. Los parentescos identifican madre y padre;
los demás adultos ocupan hasta cinco filas. Se admiten de uno a quince menores.

El firmante se configura con el rol `aceptacion`; el destinatario del SPM,
con `jefe_spm` en el catálogo de firmantes/destinatarios. La vigencia se evalúa
el día de emisión. El oficio de SS.SS.CC. toma el servicio derivante del Anexo 3.
Los oficios sólo se emiten para el origen correspondiente. Los nombres del
firmante y del jefe del SPM nunca están escritos en las plantillas activas.

Las cinco plantillas del cliente conservan textos fijos, referencias antiguas,
estilos, encabezados y pies. Los controles ActiveX pasan a marcas de selección;
los bloques de personas pasan a filas repetibles. Se retira la nota editorial
del cliente que pide implementar los quince menores. El Acta activa es 1.1.0;
los otros documentos son 1.0.0. Las copias en `es` y `en` son idénticas.
No Aceptación y Lista de Espera están activas desde documento/0003 (F1.07b).
Requieren los estados `no_aceptada` y `lista_espera`, respectivamente; otro estado
devuelve 409 con claves disponibles en es/en. El motivo, subprograma y fecha
proceden de la decisión guardada por F1.05b; el oficio usa la fecha de recepción.
Sin aceptación no se inventan número, equipo, municipio confirmado ni sexo. El Acta 1.0.0 se conserva para emisiones históricas.

Cada plantilla conserva versión, ruta y huella inmutables. La referencia activa
selecciona la plantilla y el puerto de paquete fija el directorio del idioma.
Sin paquete instalado se usa distribución; con paquete instalado no se vuelve
silenciosamente a distribución. La lectura queda confinada al volumen y verifica
la huella. El motor sólo genera en memoria. Emisión y auditoría se confirman
antes de entregar bytes; no se conservan el fichero ni datos de relleno.

La clave de emisión dura 24 horas por usuario y operación. Repetir devuelve
el mismo recibo y los mismos bytes. Si cambian datos, catálogo, idioma, motor
o plantilla, reutilizar la clave devuelve 422; una reproducción no disponible
devuelve 409 con el recibo anterior. Una clave nueva permite una emisión nueva.

`POST /api/documentos/emisiones/{id}/registro-salida` recibe `version`,
`numero_registro` y `fecha_registro` (`AAAA-MM-DD`), también con sesión de
Coordinación, CSRF e `Idempotency-Key`. Cada oficio nace con registro pendiente,
versión 1. La anotación manual pasa a `anotado` e incrementa versión. Una versión
vieja devuelve 409; una clave repetida devuelve su respuesta original y una
petición distinta con la misma clave devuelve 422. Registro, respuesta segura
y auditoría comparten transacción. No se registra salida de las Actas.
El puerto `RegistroSalida` tiene adaptador manual; la tabla admite adaptador
MOAD y referencia externa para el futuro conector. No se implementa MOAD ahora.

`documento/0002` amplía los tipos, activa las versiones oficiales y añade
registro de salida e idempotencia con RLS forzada. Sólo añade tablas de documento;
los ficheros nuevos de expediente son un puerto de lectura y su implementación.
`documento/0003` activa los dos tipos y las cuatro referencias es/en, sin cambiar
las políticas existentes ni las plantillas. Las migraciones están reservadas fuera de Git. No hay migración de vuelta.

Comprobaciones locales:

```sh
python3 -B migraciones/documento/ensayar.py
ETF_DOCUMENTO_PROBAR_LIBREOFFICE=1 ETF_DOCUMENTO_EVIDENCIA=/tmp/etf-f107-evidencia go test -count=1 -v ./internal/aplicacion/documento
go test -race ./internal/aplicacion/documento ./internal/adaptadores/http ./internal/adaptadores/ficheros/documento
make ci
```

El ensayo usa PostgreSQL 18 desechable, sin red externa y con datos inventados;
comprueba actualización desde 0001, base limpia, puerto real de expediente,
concurrencia, RLS, auditoría, registro por HTTP y cambio versionado de firmante.
LibreOffice es activable y no participa en producción. El PDF de evidencia queda
fuera del repositorio. Evidencia detallada en [EVIDENCIA_ENTRADA.md](EVIDENCIA_ENTRADA.md).

Evidencia de activación y emisiones de No Aceptación y Lista de Espera: [F1.07b](EVIDENCIA_NO_ACEPTACION_ESPERA.md).

## F3.05 · Tabla mensual SPAF

`POST /api/equipos/{etf}/tabla-spaf/{anio}/{mes}.docx` requiere sesión de
Profesional del equipo o Coordinación, CSRF e `Idempotency-Key`. Cuerpo:
`{"idioma":"es","versiones":{"regla":"regla-1","fichas":{}}}`.
Las referencias proceden de `version_regla` y de cada `version_ficha` de la
vista previa F3.04; el mapa vacío corresponde a un mes sin casos.

`documento/0005` depende de `documento/0003` y `catalogo/0008`, sin requerir
migraciones de otras ramas. Añade alcance mensual sin expediente y la reserva
de idempotencia SPAF, sin guardar el Word ni datos familiares.
`catalogo/0008` corrige el perfil de lectura; conserva escrituras y SMI.
El ensayo prueba migración sobre versión anterior y base limpia, permisos,
Word es/en con 16 casos, repetición y concurrencia, versiones 409 y reversión
ante fallo de auditoría. La ficha histórica persistente espera F2.01;
se usa un productor sintético por su puerto con cálculo F3.04 real.

### Orden de despliegue de las reservas paralelas

El ensayo integrado cubre instalación limpia y una base anterior a las nuevas
emisiones. Si una base ya tiene `documento/0008` aplicada sin `0005`, el ejecutor
rechaza añadir `0005` con `migracion.retroceso`. El director debe comprobar el
historial de la base de destino antes del despliegue; no se altera ese control
ni se reutiliza otro número. Esta entrega conserva la reserva `documento/0005`.
F1.11 añade `documento/0004_publicar_plantillas.sql`. La API de Administración
consulta la referencia con `GET /api/admin/plantillas?tipo=…&idioma=…` y publica
con `POST` en esa ruta, usando `tipo`, `idioma`, `version` y `version_previa`.
El cuerpo es el DOCX completo (máximo 10 MiB), con
`Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document`,
sesión y CSRF. No se usa un formulario multipart ni se crea una pantalla.
ClamAV debe confirmar que el archivo está limpio. Los marcadores deben conservar
los campos y bloques de la versión actual; se rechazan macros, recursos externos,
contenido activo y límites de ZIP/XML excedidos.

La ruta exclusiva de publicación vive en el idioma de distribución y conserva
su acceso aunque se active otro paquete de idioma. `archivo_plantilla` conserva
las sustituciones como historia de sólo adición. La función de publicación
bloquea y comprueba la versión previa; registra nueva plantilla, archivo,
referencia activa y auditoría en una única transacción. Una carrera devuelve
409 con la versión actual. Una versión de contenido repetida devuelve 409.
La limpieza inmediata elimina sólo rutas exclusivas no registradas, después de
consultar bajo el mismo bloqueo; si la base no responde, conserva el huérfano.
No se incluye limpieza periódica de los huérfanos retenidos.

El ensayo integrado parte de documento/0002, aplica /0003, /0004, /0005, /0008 y /0009, junto con sus dependencias, y repite las migraciones,
ejecuta las pruebas reales y comprueba también una base limpia. La prueba de
publicación usa dos conexiones concurrentes, comprueba archivo y auditoría,
revierte una publicación ya ejecutada y verifica permisos SQL e inmutabilidad.
