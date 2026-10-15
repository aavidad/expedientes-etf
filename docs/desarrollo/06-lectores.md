# 6. Lectores de anexos y actas

Los lectores proponen datos para revisar. La subida de un anexo no crea un expediente: el caso de uso valida el archivo, pasa el antivirus, lee en memoria y devuelve campos, dudas y propuestas de catálogo. La persona confirma los datos en una acción posterior. El archivo recibido no se conserva.

## Entrada y revisión

El lector compuesto [internal/adaptadores/anexos/lectura](../../internal/adaptadores/anexos/lectura/) implementa el puerto de Expediente y limita la lectura a una operación simultánea. Selecciona DOCX, ODT o PDF por el formato validado. No confíes sólo en el nombre ni en el Content-Type proporcionado por el navegador.

El resultado contiene reconocimiento de plantilla, tipo, versión, avisos y campos. Cada campo lleva código, valor, página cuando se conoce, marca de duda y motivos. Esos motivos son claves que la interfaz traduce. Un campo ausente permanece pendiente de completar; no lo rellenes con una suposición.

Tras la revisión, el alta comprueba otra vez el ámbito, los códigos y las reglas. Descartar elimina la revisión temporal. La confirmación guarda datos estructurados, su procedencia y la auditoría; no guarda el anexo.

## DOCX y ODT

[El lector DOCX](../../internal/adaptadores/anexos/docx/lector.go) reconoce los Anexos 3 y 4 mediante perfiles de formato y extrae contenido del ZIP y XML limitado. Conserva las dudas de campos que no puede identificar con suficiente precisión. Los corpus de prueba están junto al adaptador.

[El lector ODT](../../internal/adaptadores/anexos/odt/lector.go) trabaja con los formatos de LibreOffice admitidos y sus perfiles. La migración `expediente/0026` incorpora el formato ODT al contrato persistente sin cambiar permisos. No conviertas el archivo con una aplicación de escritorio durante una petición del servidor.

Al ampliar un perfil, añade una muestra inventada y su resultado esperado: tipo reconocido, campos recuperados, dudas y vacíos. Incluye formato no reconocido, ZIP corrupto y límites de expansión. Un documento que se abre en Word puede seguir incumpliendo el contrato del lector.

## PDF

[El adaptador PDF](../../internal/adaptadores/anexos/pdf/adaptador.go) usa un trabajador efímero con entrada y salida por tuberías. Tiene concurrencia y recursos acotados; no escribe nombres ni contenido en archivos temporales. El lector limita la entrada a 10 MiB y el análisis a 20 páginas, además de objetos, lecturas y glifos.

El trabajador fija límites de memoria y CPU. Las pruebas Linux comprueban esos límites. Conserva el mecanismo al añadir extracción; pasar el PDF al proceso principal para acelerar un caso quitaría la protección frente a archivos costosos.

Se admiten PDF con texto y los formatos reconocidos de LibreOffice, incluidos los contrastados con firma digital local. La lectura de texto no verifica la validez jurídica de una firma. Un PDF escaneado devuelve el aviso para completar datos a mano. No hay reconocimiento óptico de caracteres en este recorrido.

La [evidencia RESP4](../recorridos/RESP4.md) muestra valores recuperados y campos que requieren revisión. `probar_subida.py` del adaptador permite ensayar HTTP con ClamAV de Compose y recursos desechables.

## Actas históricas

La lectura de actas es una capacidad separada para preparar carga histórica. [LeerActa](../../internal/adaptadores/anexos/lectura/lector.go) dirige DOCX y ODT a sus lectores correspondientes y PDF al trabajador limitado. [El extractor de actas](../../internal/adaptadores/anexos/actas/extraer.go) aplica su perfil y devuelve campos de aceptación o cierre. [El lector de actas ODT](../../internal/adaptadores/anexos/odt/acta.go) recupera líneas del XML de LibreOffice y utiliza ese mismo contrato; su confirmación se ensaya en el recorrido de carga.

Su resultado prepara un alta histórica; no ejecuta automáticamente aceptación, cierre o firma ni consume numeración nueva. Confirma números y fechas originales con los mismos servicios de carga. El [contrato RESP2](../producto/ESQUEMA_BACKEND.md#71b-lectura-de-actas-para-alta-histórica-resp2) explica las rutas y los límites. La web y el recorrido están en [carga](../../web/src/carga/) y [carga-actas.mjs](../../scripts/recorridos/carga-actas.mjs).

## Antivirus y errores

Un resultado positivo del antivirus bloquea el archivo. La indisponibilidad tiene un contrato distinto según la operación: la lectura de anexos contempla el aviso explícito permitido; la publicación de plantillas y paquetes y la importación de hojas requieren confirmación de limpieza. No extiendas la excepción de anexos a otra entrada.

Prueba un archivo limpio, EICAR, formato disfrazado, XML o ZIP excesivo, cancelación, lector ocupado y trabajador agotado. Comprueba además que un error no guarda el archivo ni crea un expediente parcial. Las evidencias y descargas de prueba deben quedarse fuera de Git salvo los corpus inventados y las capturas encargadas.

Fuentes: [entradas de archivos](../producto/ESQUEMA_BACKEND.md#19-entradas-de-ficheros-y-persistencia), [lectores](../../internal/adaptadores/anexos/) y [subida HTTP PDF](../../internal/adaptadores/anexos/pdf/probar_subida.py).
