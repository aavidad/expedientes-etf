# 9. Recetas de ampliación

Estas recetas siguen las fronteras de Expedientes ETF. Antes de aplicarlas, localiza una capacidad próxima y sus contratos. Si el cambio necesita una regla funcional aún sin decidir, resuelve esa regla con el responsable antes de guardar datos o publicar una opción.

## Añadir un campo a la ficha

1. Define quién lo lee y cambia, si admite desconocido, su validación y la fecha de vigencia. Si afecta a Memoria o SPAF, identifica apartado, corte y tratamiento de ausencia en el contrato estadístico.
2. Añade el valor y sus reglas en `internal/dominio/expediente/`. Conserva la distinción entre desconocido, cero y colección vacía. Evita importar HTTP o PostgreSQL.
3. Amplía el contrato del puerto de entrada y la lectura necesaria del puerto de salida. Define cómo participa el campo en la versión del expediente y en la historia; el servidor conserva la identidad de sesión.
4. Reserva una migración de Expediente. Añade columna o estructura histórica, restricciones y permisos, y actualiza la proyección publicada si el dato lo consume otro módulo. Declara sus dependencias en el manifiesto.
5. Implementa el repositorio y caso de uso con el mismo contexto transaccional. Comprueba ámbito, versión, vigencia, idempotencia cuando corresponda y evento de auditoría sin el valor personal.
6. Amplía el handler y los tipos del cliente de la ficha. Añade etiqueta, ayuda y errores a `idiomas/es/textos/` y `idiomas/en/textos/`; usa `t(clave)` y tokens del tema.
7. Si el campo interviene en documentos o cálculos, conecta productor, puerto, dato de dominio y salida. Prueba un corte anterior y otro posterior a su cambio de vigencia.
8. Ensaya guardado, recuperación, conflicto, equipo ajeno y lectura de Coordinación. Añade las pruebas de SQL y cálculo que correspondan, actualiza documentos y ejecuta `make ci`.

Puntos de partida: [familia y adultos](../../internal/aplicacion/expediente/), [adaptadores PostgreSQL](../../internal/adaptadores/postgres/expediente/), [ficha web](../../web/src/equipo/) y [lecturas históricas](../../internal/adaptadores/postgres/expediente/estadistica.go).

## Añadir una pantalla

1. Define ruta, perfiles, estado vacío y acción principal en el flujo de producto. Comprueba que exista el puerto de entrada para consultar o guardar los datos necesarios.
2. Crea la carpeta de capacidad en `web/src/`, su componente, tipos y cliente API. Recibe `PropsPantalla`; conserva borrador, versión y clave de recuperación de las acciones.
3. Añade `pantallas.ts` en esa carpeta con ruta, perfiles, clave de menú y componente. Si cuelga de otra entrada, utiliza `menuDe` según el contrato del registro.
4. Añade la lista a `web/src/pantallas.ts` en el orden acordado. Declara la SPA y sus endpoints en el módulo HTTP, con métodos y permisos completos, y compón la capacidad en `cmd/etf/modulos.go` si es nueva.
5. Publica todas las claves en los directorios de textos de los idiomas mantenidos. Usa variables de tema y controles accesibles. Comprueba que el menú requiera también la pantalla publicada por la sesión.
6. Prueba navegación directa, anónimo, perfil incorrecto, carga, vacío, error, conflicto y salida con cambios sin guardar. Recorre teclado, escritorio, 390 píxeles y alto contraste; contrasta el registro con F0.11.
7. Actualiza flujo, diseño y estado de la capacidad. Ejecuta tipos, pruebas web, textos, navegador y `make ci`.

Referencia pequeña: [pantalla de Plantillas](../../web/src/plantillas/pantallas.ts), [registro de pantallas](../../web/src/registro-pantallas.ts) y [composición HTTP](../../internal/adaptadores/http/modulos.go).

## Añadir una versión de plantilla

1. Elige tipo e idioma y consulta la activa y su revisión en `GET /api/admin/plantillas`. Conserva la versión previa de la referencia para detectar activaciones concurrentes.
2. Prepara un DOCX con versión nueva y el contrato de campos y bloques del tipo. Mantén los textos fijos acordados. Comprueba los marcadores incluso si Word los divide entre fragmentos.
3. Revisa el DOCX con el motor local y sus pruebas: XML, límites, contenido activo y recursos permitidos. Prueba valores con caracteres especiales y colecciones vacías.
4. Desde Administración, sube la versión en Plantillas Word. La API usa `POST /api/admin/plantillas/versiones`, cuerpo DOCX, parámetros de tipo, idioma, versión y versión previa, y cabecera CSRF. Revisa las listas de marcadores faltantes y sobrantes.
5. Confirma que la versión aparece en el historial y que la activa anterior sigue igual. Después activa la elegida con su versión previa actual. Ante 409, recarga y revisa qué activó otra persona.
6. Genera un Word real con datos inventados, comprueba emisión, huella, formato y recuperación idempotente. Comprueba también que una emisión histórica conserva la plantilla anterior.

No sobrescribas el DOCX activo desde el sistema de archivos. Si la versión forma parte de la distribución, registra la actualización mediante una migración reservada de Documento que conserve decisiones administrativas, como hacen las actualizaciones existentes.

## Añadir un tipo de documento

1. Define datos, marcadores, bloques, perfiles, idioma y si necesita registro de salida. Separa emisión de confirmación de firma o fecha.
2. Añade el contrato de dominio y puertos de Documento y el lector autorizado de los datos del módulo propietario. Implementa generación con versiones fijadas y escape XML.
3. Reserva la migración para metadatos, tipo y referencias iniciales de plantillas. Incluye RLS y permisos de emisiones y registros si el tipo los necesita.
4. Prepara las plantillas de los idiomas mantenidos y sus huellas. Añade la ruta POST con CSRF, idempotencia y recibo; conecta después el botón traducido de la pantalla.
5. Prueba permiso por tipo, recuperación, reproducción imposible, error de auditoría, concurrencia y ausencia de archivos generados persistentes. Revisa el Word y actualiza los contratos antes de ejecutar la CI.

Fuentes de trabajo: [puertos de Documento](../../internal/puertos/salida/documento/), [motor](../../internal/adaptadores/docx/) y [migraciones](../../migraciones/documento/).

## Añadir un idioma

1. Crea `idiomas/<codigo>/idioma.json` con código válido, nombre y versión. Añade `textos/`, `catalogos/` y `plantillas/` con la estructura de un idioma mantenido.
2. Traduce todas las claves de pantalla y error conservando parámetros. Comprueba la paridad con `make textos`. No añadas condicionales de idioma al componente para suplir traducciones ausentes.
3. Traduce etiquetas de catálogo conservando los códigos y contratos de versión. Prepara las plantillas del idioma que vaya a ofrecerse para cada tipo disponible; un idioma de pantalla no garantiza un Word traducido.
4. Valida el paquete y publícalo mediante la capacidad de idiomas, con antivirus y versión inmutable. Usa la recarga y activación que expone la administración de paquetes. Confirma que el selector devuelve la referencia activada.
5. Recorre entrada, formularios, fechas, números, errores, documentos y aviso sin conexión. Comprueba `lang`, localización, desbordamientos y respaldo al idioma predeterminado.
6. Conserva las versiones anteriores, actualiza la documentación de idiomas admitidos y ejecuta textos, temas, recorridos afectados y CI.

Revisa [el adaptador de paquetes](../../internal/adaptadores/ficheros/paquetes.go), [publicación](../../internal/aplicacion/paquete/publicacion.go) y [la administración web](../../web/src/paquete/). No registres traducciones en PostgreSQL.

## Añadir un catálogo o una opción

1. Comprueba si la opción cabe en un ámbito existente. Fija código estable, vigencia, perfiles de edición y equivalencias que consuman evaluación o estadística.
2. Para una opción de un catálogo existente, utiliza su borrador y publicación administrativa. Para un ámbito o contrato nuevo, amplía dominio, puertos y validaciones; reserva migración sólo cuando cambie estructura o datos de distribución.
3. Publica las etiquetas por idioma mediante los contratos de paquetes. Conserva versiones históricas y verifica que una etiqueta no cambie el significado del código.
4. Conecta el selector a la lectura autorizada del catálogo. El navegador envía el código; el caso de uso valida que sea admisible en la fecha de la operación.
5. Prueba borrador, publicación, vigencia anterior y posterior, desconocido y concurrencia. Si hay equivalencia estadística, comprueba un cálculo pasado y una revisión cerrada.
6. Ejecuta comprobaciones de catálogos, textos, ensayo PostgreSQL cuando corresponda y `make ci`. Actualiza el esquema de datos y la receta funcional de administración afectada.

Puntos de partida: [módulo Catálogo](../../internal/aplicacion/catalogo/), [textos de catálogo](../../idiomas/es/catalogos/) y [comprobador](../../scripts/ci/catalogos/comprobar.py).

## Añadir una migración

1. Consulta la reserva externa del proyecto y el mecanismo del encargo. Reserva número antes de crear el fichero; registra módulo, número, rama, descripción, fecha y `Reservada`.
2. Crea `migraciones/<modulo>/NNNN_descripcion.sql`. Comprueba estado previo, integridad y compatibilidad; añade permisos, RLS y `search_path` de funciones cuando corresponda. No incluyas contraseñas.
3. Añade su entrada al manifiesto con `numero`, `archivo` y `depende_de`, aunque la lista sea vacía cuando el contrato lo permita. Declara las dependencias reales por número y evita ciclos.
4. Ejecuta las pruebas del plan y el ensayo del módulo sobre PostgreSQL 18 limpio y actualizado. Comprueba también rechazo de historial alterado, transacciones, aislamiento y datos previos.
5. Obtén revisión independiente de SQL y permisos y ensayo en el clon local con datos inventados. Registra el resultado, no sólo el comando previsto.
6. Integra `main` antes de entregar. Si apareció una migración superior en el mismo módulo, pide número nuevo al director y renombra SQL, manifiesto y pruebas. El migrador rechazará el retroceso.
7. Actualiza el esquema y los documentos de despliegue afectados, regenera el índice y ejecuta `make ci`. Una migración aplicada se corrige con otra posterior; nunca se edita para hacer coincidir una huella.

```sh
go test ./internal/plataforma/migracion
python3 -B scripts/ci/indice/generar.py
python3 -B scripts/ci/enlaces.py
make ci
```
