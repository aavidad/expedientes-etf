# F1.02a: lector de Anexos 3 y 4 en DOCX

`Leer([]byte) (Lectura, error)` interpreta las dos plantillas del cliente sin abrir
ficheros, crear temporales ni llamar a servicios. No se conecta todavía a HTTP ni
a aplicación. No añade dependencias.

## Contrato y versiones

`Campo` conserva los campos del ensayo PDF: `Codigo`, `Valor`, `Pagina`, `Dudoso`
y `Motivos`. `Lectura` conserva `PlantillaReconocida`, `Campos` y `Avisos`, y añade
`Tipo` (`anexo3` o `anexo4`) y `Version` (`cliente-2026-v1`). No importa el ensayo
PDF, que es un módulo independiente. `Pagina` vale cero porque Word no proporciona
una paginación estable en `document.xml`.

El reconocimiento compara títulos, 11 o 12 tablas, filas, celdas y etiquetas
estáticas con `perfiles.json`. Este fichero contiene reglas de interpretación de
las plantillas, sus textos de relleno y sus listas originales; no es un catálogo
publicable. Una plantilla desconocida devuelve `anexo.plantilla_desconocida`, sin
campos parciales. Los controles de contenido se leen por posición dentro de la
celda y se conserva el orden de las filas. La primera fila de personas
significativas contiene tres celdas sin controles: también se leen.

Se mantienen los códigos PDF de los apartados 1–4 y se añaden los códigos
`valoracion.*`, `valorame.*`, `desproteccion.*`, `intervenciones.*`,
`resoluciones.*` y `observaciones.*`. El Anexo 4 añade `servicio.etf`,
`servicio.subprograma` y `servicio.equipo_menores`. No se inventa sexo ni municipio.
Se extraen todas las filas de menores, incluidas las vacías, hasta 15; con 16 se
devuelve `anexo.docx_demasiados_menores`. Los códigos repetidos identifican
colecciones por su orden, igual que en el ensayo PDF.

Los vacíos tienen motivo `anexo.campo_ausente`; los textos de relleno,
`anexo.texto_relleno`; las opciones no previstas, `anexo.valor_fuera_lista`.
Se conserva el valor de origen para revisión. Las opciones insertadas en el
documento recibido no amplían las listas conocidas. Las cinco equivalencias
educativas confirmadas en PRD §4.9 se aplican a `sociodemografia.adulto.estudios`.
La lectura incluye siempre `anexo.confirmar_docx`: incluso los campos no dudosos
necesitan confirmación humana en la integración futura.

Las casillas de crisis y actitud se leen del estado Word 2010 o, en el Anexo 3,
de la propiedad XML `Value` de ActiveX. No se ejecuta ningún control ni se lee su
almacenamiento binario. Una selección doble se marca `anexo.valor_ambiguo`;
un estado no disponible o un formato binario sin estado XML se marca
`anexo.casilla_no_legible`. No se deduce una selección de la etiqueta SI/NO ni de
la imagen. El genograma gráfico requiere revisión manual.

## Límites y revisión local de seguridad

- Entrada: 10 MB (10 000 000 bytes); hasta 256 partes ZIP.
- Descompresión total: 32 MiB; XML retenido: hasta 8 MiB por parte.
- Razón máxima de descompresión: 200 para partes mayores de 1 MiB.
- XML: profundidad 64, 150 000 elementos y 500 000 tokens.
- Se comprueban tamaños reales y CRC de todas las partes, también de las que se
  descartan. Se rechazan rutas no canónicas, duplicadas y partes no regulares.
- Se comprueban el tipo principal Word y la ausencia de proyectos VBA o tipos
  principales con macros. Esta validación no sustituye al antivirus de F1.01.
- El decodificador XML no resuelve entidades externas; se rechazan todas las
  directivas y las entidades desconocidas. Las relaciones ActiveX sólo se
  resuelven contra partes locales del ZIP. No se siguen relaciones externas.
- Los errores son claves sin nombres, contactos, texto del documento ni datos
  parciales. No hay registros de contenido ni acceso a disco o red.

`local.sh` ejecuta pruebas y herramientas en Bubblewrap: repositorio y
dependencias de sólo lectura, sin red, entorno vacío con variables permitidas,
temporales en memoria con cuota y límites de CPU, memoria, procesos y tiempo.
No descarga herramientas ni dependencias.

## Corpus y comprobaciones

`testdata/anexo3.docx` y `anexo4.docx` son copias cumplimentadas de las plantillas
con seis menores, contactos inventados y correos `.invalid`. Incluyen campos
vacíos y los cinco niveles educativos. `version_desconocida.docx` conserva las
tablas pero cambia el título. Se eliminan las propiedades heredadas de autoría.
`testdata/generar.py` prepara perfiles, filas y casillas con la biblioteca estándar
de Python y llama a la prueba auxiliar `generar_corpus_test.go`, activada sólo con la etiqueta
`generar_corpus`, que rellena los controles mediante
`internal/adaptadores/docx.Generar`. No necesita LibreOffice, python-docx ni acceso
a la red. Recibe `--plantillas` y `--salida`; al regenerar, la salida debe ir a
`/scratch` mediante `local.sh`.
La regeneración aislada con el motor Word produjo cinco artefactos idénticos
byte a byte. `esperado.json` permite comparar cada valor extraído; las pruebas añaden casos
de 15 y 16 menores, rellenos, listas manipuladas, selecciones dobles, XML hostil,
ZIP truncado, duplicados, rutas, macros y cuotas. Un contraste escrito aparte
del generador verifica campos concretos y las cinco equivalencias educativas.

Comandos ejecutados desde la raíz, mediante `bash internal/adaptadores/anexos/docx/local.sh`:

```text
go vet ./...
go test ./...
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/docx  0.326s
gopls check internal/adaptadores/anexos/docx/{lector,campos,xml,lector_test}.go
CGO_ENABLED=1 go test -race ./internal/adaptadores/anexos/docx -count=1
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/docx  6.736s
go test ./internal/adaptadores/anexos/docx -cover -count=1
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/docx  0.339s  coverage: 88.8% of statements
go test ./internal/adaptadores/anexos/docx -run '^$' -fuzz FuzzLeer -fuzztime=3s
fuzz: elapsed: 3s, execs: 20181 (6682/sec), new interesting: 1 (total: 5)
PASS
gitleaks dir internal/adaptadores/anexos/docx --no-banner --redact
no leaks found
semgrep --config internal/adaptadores/anexos/docx/semgrep.yml --metrics=off --disable-version-check --no-git-ignore -j 1 --max-memory 512 internal/adaptadores/anexos/docx/{lector,campos,xml}.go
Ran 2 rules on 3 files: 0 findings.
```

`go vet`, `gopls` y el resto de paquetes terminaron con código cero. Semgrep
comprueba con dos reglas locales la ausencia de acceso externo y registros de
contenido; no equivale a una auditoría completa. No se modifican documentos de
producto: se entrega sólo el lector de F1.02a; pantalla, HTTP, puerto de aplicación
y antivirus pertenecen a las integraciones posteriores.
