# F0.17b · prueba del motor Word

13/10/2026 · rama `f0.17b-motor-word` · Go 1.25.5 · LibreOffice 26.2.6.3.

El relleno propio del XML funciona con los dos documentos del cliente. Recomiendo
reutilizar este motor en F1.06, después de fijar el contrato del puerto y validar
cada versión de plantilla. No hace falta LibreOffice en producción.

## Qué se ha comprobado

- `Generar([]byte, Datos)` recibe y devuelve bytes en memoria. El código de
  producción sólo usa la biblioteca estándar; no tiene acceso a disco, procesos,
  red, base de datos ni registros. Sólo la prueba local escribe DOCX y PDF en su
  directorio temporal, que Go elimina al acabar.
- La copia del Acta conserva sus textos fijos, estilos y cabeceras. Convierte las
  opciones en campos, sustituye el firmante por un marcador y convierte el bloque
  de menor en una fila con `{{#menores}}` / `{{/menores}}`. Se han rellenado 15 filas
  en orden y los campos vacíos quedan blancos. El script de `testdata/` reproduce
  la conversión; no modifica el original del cliente ni publica en `idiomas/`.
- `<`, `&`, comillas dobles y simples, tildes, ñ y emoji se recuperan sin cambios
  al leer el XML generado. Se conservan espacios y marcadores partidos entre
  varios runs. Los valores aportados nunca se interpretan como otra plantilla.
- Un marcador desconocido devuelve `Error{Clave: "docx.marcador_desconocido",
  Campo: "desconocido"}` sin resultado parcial. Los errores llevan claves e
  identificadores de campo, sin valores personales. También se rechazan bloques
  incompletos, caracteres XML prohibidos y controles inexistentes o vinculados.
- El Anexo 3 tiene 123 controles: se rellenan los 122 que contienen texto y se
  conserva el de imagen. Mantiene `sdt`, identificadores, tipos, fechas y opciones
  de las listas. Se retira el bloqueo y la marca de texto provisional de los
  controles rellenados; una segunda edición vuelve a cambiar sus valores.
- Las partes ZIP distintas de `word/document.xml` quedan idénticas al contenido
  de la plantilla. Dos generaciones con la misma plantilla y datos devuelven
  bytes idénticos, requisito previo para la huella de emisión de F1.06.
- Se prueban límites de entradas y descompresión, partes duplicadas, macros,
  estructura mínima, directivas XML y expansión del resultado. Las cuotas
  técnicas son 10 MiB de entrada/datos, 1.000 partes, 100 MiB descomprimidos y un
  presupuesto conservador de 100 MiB para el XML y sus copias; no son reglas
  de negocio nuevas.

## Evidencia ejecutada

`ETF_DOCX_PROBAR_LIBREOFFICE=1 go test -count=1 -v ./internal/adaptadores/docx`
ejecutó todas las pruebas, incluida la apertura local. La prueba llama a
`env -u LD_LIBRARY_PATH soffice --headless --convert-to pdf`, con perfil aislado y
directorio de salida temporal. Salida real de LibreOffice:

```text
convert /tmp/TestLibreOfficeLocal335726174/001/acta-15-menores.docx as a Writer document -> /tmp/TestLibreOfficeLocal335726174/001/acta-15-menores.pdf using filter : writer_pdf_Export
convert /tmp/TestLibreOfficeLocal335726174/001/anexo-3-editable.docx as a Writer document -> /tmp/TestLibreOfficeLocal335726174/001/anexo-3-editable.pdf using filter : writer_pdf_Export
--- PASS: TestLibreOfficeLocal (1.53s)
PASS
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/docx	1.804s
```

Los PDF tienen 108.656 y 208.720 bytes. `pdftotext -layout` comprobó el número en
ambos y la presencia del menor 15 en el Acta. No acredita revisión visual,
accesibilidad completa ni edición manual en Microsoft Word.

`go test ./...` terminó con código 0. Estas son las líneas de paquetes con
pruebas; los restantes indicaron `[no test files]`:

```text
ok  	github.com/aavidad/expedientes-etf/internal	0.004s
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/docx	0.257s
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros	(cached)
ok  	github.com/aavidad/expedientes-etf/internal/adaptadores/http	(cached)
ok  	github.com/aavidad/expedientes-etf/internal/plataforma	(cached)
```

`go vet ./...` y `gopls check` sobre los tres ficheros Go con lógica terminaron
con código 0 y sin salida. `go test -race ./internal/adaptadores/docx` pasó.
Semgrep local, sin métricas ni consulta de versiones, usó las tres reglas de
`testdata/semgrep-local.yaml` sobre los tres ficheros Go de producción:
`Ran 3 rules on 3 files: 0 findings.` Es una comprobación focal de fronteras,
no una auditoría completa ni una validación de publicación de ficheros.

## Lo que debe resolver F1.06

1. Definir `GeneradorDocumentos` y conectar POST, permisos, versiones inmutables,
   huellas e idempotencia. Confirmar emisión y auditoría antes de la descarga;
   guardar sólo metadatos y huellas. Generar nunca confirma un hito.
2. Publicar las plantillas revisadas en `idiomas/<código>/plantillas/`, con
   antivirus y validación del esquema esperado. Los originales llevan ActiveX:
   se retiraron de la copia del Acta; en el Anexo se conservaron para probar el
   relleno del original. Normalizar esos objetos antes de publicar plantillas.
3. Asociar los `w:id` a campos semánticos por versión; los originales no tienen
   `w:tag`. El llamador formatea los valores. Completar validación de selecciones,
   metadatos de fechas y textos multilínea para cada tipo de control.
4. Ampliar según las plantillas reales: esta prueba modifica sólo el XML principal,
   admite filas con entre 1 y 15 elementos y no admite controles dentro de filas
   repetidas ni enlaces `dataBinding`. Revisar cabeceras con campos, bloques
   anidados, listas y accesibilidad antes de añadir esos usos.

Las pruebas de este adaptador usan las plantillas públicas y datos inventados. El estado y los límites de la revisión se mantienen en el plan de implementación.
