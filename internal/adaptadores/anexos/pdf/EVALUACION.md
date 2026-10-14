# F0.17a: lectura local de PDF

Recomiendo probar **`github.com/dslipak/pdf v0.0.2` en F1.03**, con revisión humana de todos los campos. En este corpus recupera exactamente 91 de 92 valores de los apartados 1–4. La propuesta es provisional: faltan muestras inventadas del formato habitual del Programa [D6], documentos firmados y otras fuentes tipográficas.

## Comparación

Go 1.25.5, LibreOffice 26.2.6.3, Linux amd64. Los tres extractores son Go puro, locales y de licencia BSD de tres cláusulas; comparten ascendencia, por lo que esta comparación no representa tres motores independientes. Versiones y sumas están fijadas en `go.mod` y `go.sum` de este directorio.

| Biblioteca y versión | Anexo 3: valores exactos | Anexo 4: valores exactos | Resultado observado |
| --- | --- | --- | --- |
| [rsc.io/pdf v0.1.1](https://pkg.go.dev/rsc.io/pdf@v0.1.1) | 0/44 | 0/48 | La extracción posicional pierde caracteres de las fuentes incrustadas y no reconoce la plantilla. |
| [ledongthuc/pdf 6c8c28e0e8a0](https://github.com/ledongthuc/pdf/tree/6c8c28e0e8a0) (`v0.0.0-20260907135840-6c8c28e0e8a0`) | 1/44 | 0/48 | Reconoce los encabezados, pero `Content` añade letras al final de los bloques. En esta versión, el final de `TJ` pasa un salto de línea por la decodificación de la fuente. |
| [dslipak/pdf v0.0.2](https://github.com/dslipak/pdf/tree/v0.0.2) | 43/44 | 48/48 | Conserva nombres, acentos, fechas, contactos, tablas y continuaciones; altera una ligadura del Anexo 3. |

Licencias verificadas en los módulos descargados: [rsc](https://github.com/rsc/pdf/blob/v0.1.1/LICENSE), [ledongthuc](https://github.com/ledongthuc/pdf/blob/6c8c28e0e8a0/LICENSE), [dslipak](https://github.com/dslipak/pdf/blob/v0.0.2/LICENSE). No se ha ejecutado OCR ni enviado documentos, código o inventarios a servicios externos. Sólo se han consultado fuentes públicas y descargado dependencias fijadas.

La métrica compara cada valor con `testdata/esperado.json`, sin normalizar ni corregir nombres. Incluye el campo vacío de equipo de menores en el Anexo 3. Las pruebas exigen igualdad; sólo admiten la desviación concreta del Anexo 3 registrada en `desviaciones.json`: «Dificultades económicas» aparece como «Difci ultades económicas». No se corrige por sustitución ni se deduce el sexo de los menores.

## Corpus y extracción

`testdata/anexo3.pdf` y `anexo4.pdf` tienen cinco páginas cada uno. Sus DOCX proceden de las dos plantillas del cliente, con las tablas 1–4 cumplimentadas y datos de convivencia y personas significativas también rellenados. Hay dos contactos por familia, dos menores en el Anexo 3 y tres en el Anexo 4. Los nombres y contactos son inventados; los correos usan el dominio reservado `.invalid`. Los apartados 5–7 no forman parte de la extracción y conservan contenido de plantilla. El genograma es gráfico y requiere revisión manual.

`sin_texto.pdf` contiene únicamente una imagen de la primera página: se rasterizó con Poppler y se convirtió a PDF con LibreOffice. `corrupto.pdf` son los primeros 512 bytes del Anexo 3. Los casos grandes y de muchas páginas se construyen en memoria durante las pruebas; no se guardan ficheros de 10 MB.

`testdata/generar.py` rellena las tablas originales con `relleno.json` y, si recibe una imagen, crea el Word sin texto. Se ejecutan en una misma sesión aislada: generación de DOCX, conversión de ambos anexos, rasterización de la primera página con `pdftoppm`, creación del Word de imagen y su conversión. Cada conversión usa:

```sh
env -u LD_LIBRARY_PATH soffice -env:UserInstallation=file:///scratch/lo --headless --convert-to pdf --outdir /scratch /scratch/anexo3.docx /scratch/anexo4.docx
```

El entorno es efímero: al terminar se exportan sólo los cinco DOCX/PDF previstos mediante un archivo ZIP; el proceso padre comprueba nombres, tipos y tamaños antes de copiarlos a `testdata/`. No se exige igualdad binaria al regenerar: LibreOffice incorpora fechas e identificadores. El generador vacía las propiedades heredadas de autoría de las plantillas.

`Leer` recibe bytes, devuelve líneas y posiciones, y nunca abre ficheros. `Extraer` interpreta `testdata/perfil.json`: etiquetas y geometría son datos del ensayo. Lee servicio derivante, profesional, teléfono, correo y equipo de menores; contacto y dirección familiar; nombres, nacimientos, escolarización y centros de menores; estructura, vivienda, ingresos, situación laboral/económica, adultos y personas significativas. Cada propuesta lleva página de origen, `Dudoso=true` y clave de confirmación; los campos ausentes y ambiguos tienen claves adicionales. Un documento sin texto se distingue de una plantilla no reconocida. No se expone ninguna de estas claves a una interfaz todavía.

## Límites y seguridad

Se interpreta 10 MB como **10.000.000 bytes**. Las pruebas aceptan un PDF válido de ese tamaño y rechazan 10.000.001 bytes. Se acepta el límite exacto de cinco páginas del corpus y se rechaza con límite cuatro; además se rechaza un documento válido de 21 páginas con límite 20. El ensayo usa 20 páginas, 10.000 objetos, 100.000 lecturas y 200.000 glifos: son parámetros técnicos de evaluación, no decisiones nuevas de producto.

Se comprueban cabecera, final, estructura legible, declaraciones visibles de páginas/objetos, tamaño del tráiler, cuotas de lectura y salida. Los errores y las recuperaciones de pánico entregan claves sin contenido personal ni resultados parciales. Se evita llamar a `Content` sin stream: `dslipak` se quedaba leyendo una página vacía. El lector acotado interrumpe también lecturas agotadas o vacías porque esta biblioteca puede ignorar errores de entrada.

Estas cotas **no bastan para procesar PDF hostiles dentro del servidor**: no controlan asignaciones internas de xref, expansión de streams ni ciclos de referencias sin lectura adicional. La prevalidación conservadora de diccionarios visibles puede rechazar documentos válidos. F1.03 necesita validación estructural completa y un trabajador aislado con límites de CPU, memoria y tiempo, cancelación efectiva y datos sólo en memoria. Debe probar también cifrados, firmas, árboles de páginas incoherentes, tablas incompletas y hasta 15 menores. La confirmación humana sigue siendo obligatoria.

## Comprobaciones

`local.sh` ejecuta con Bubblewrap sin red ni credenciales del usuario, entorno explícito, fuente y dependencias de sólo lectura y escritura efímera en tmpfs con cuotas de 512 MiB para trabajo y 256 MiB para temporales (ampliados en F1.03 para compilar las dependencias HTTP). Impone 60 s de tiempo/CPU, 4 GiB de espacio de direcciones, 4.096 procesos/hilos por usuario y 512 MiB por fichero. Oculta los temporales Go al recorrido de arquitectura del repositorio. No reproduce proxy, antivirus ni política de intercambio a disco de producción.

```text
$ bash internal/adaptadores/anexos/pdf/local.sh go test ./... -count=1
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/anexos/pdf  0.529s
$ bash internal/adaptadores/anexos/pdf/local.sh go vet ./...
[sin salida; código 0]
$ bash internal/adaptadores/anexos/pdf/local.sh go -C /repo test ./...
ok  github.com/aavidad/expedientes-etf/internal  0.003s
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros  0.003s
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/http  0.004s
ok  github.com/aavidad/expedientes-etf/internal/plataforma  0.002s
[resto: no test files; código 0]
$ bash internal/adaptadores/anexos/pdf/local.sh go -C /repo vet ./...
[sin salida; código 0]
$ gopls check lector.go campos.go lector_test.go
[sin salida; código 0; ejecutado con sus rutas desde la raíz]
$ SEMGREP_SEND_METRICS=off semgrep --config internal/adaptadores/anexos/pdf/semgrep.yml --metrics=off --disable-version-check --no-git-ignore internal/adaptadores/anexos/pdf/lector.go internal/adaptadores/anexos/pdf/campos.go
Ran 2 rules on 2 files: 0 findings.
```

Semgrep usa dos reglas locales centradas en persistencia/red y registros de datos; no equivale a ejecutar todo `p/golang`. La revisión focal de `security-audit` abarca el flujo de bytes, errores, cuotas y ausencia de acceso externo. La revisión independiente inicial dio GO al ensayo; sus dos observaciones se corrigieron: excepción de prueba acotada y cuota total de disco.

En F0.17a el módulo aislado mantuvo todo el cambio en esta zona y quedó fuera de `go test ./...` de la raíz. La integración F1.03 descrita a continuación sustituye ese aislamiento de compilación. Las métricas anteriores son la evidencia histórica del ensayo comparativo.

## Integración F1.03

El adaptador implementa `LectorAnexo` y el selector de F1.01 lo elige para PDF. `dslipak/pdf v0.0.2` y su suma pasan al módulo principal; se retiran los motores descartados y el módulo anidado. Las pruebas PDF quedan incluidas en `go test ./...` y `make ci`.

Cada lectura se ejecuta con el propio binario en un proceso efímero, antes de cargar configuración. Entrada y salida van por tuberías, sin temporales, registros ni credenciales heredadas. Se permiten 10.000.000 bytes y 20 páginas; se conservan las cuotas de objetos, lecturas y glifos. El trabajador tiene límites de CPU de 3 segundos, tiempo de 5 segundos, datos de 256 MiB, espacio de direcciones de 4 GiB y tamaños de fichero y core de cero. La cancelación termina el proceso. La prohibición operativa de intercambio a disco sigue correspondiendo al despliegue.

Se valida el árbol de páginas, los recuentos y los tipos de streams antes de la extracción. Esta comprobación y la prevalidación no equivalen a un validador completo de todas las características PDF: los límites del proceso siguen siendo necesarios. No se comprueba la autenticidad de firmas digitales ni se ejecuta OCR.

Los valores recuperados de los apartados 1–4 conservan códigos, orden, página y motivos de confirmación de F0.17a. Los otros campos del lector Word se ofrecen vacíos y dudosos según el perfil de cada anexo; no se deducen valores de los apartados que el corpus no acredita. Se excluye VALÓRAME, que pertenece a evaluación. El escaneado se admite con una lista de campos vacía y el aviso `anexo.pdf_sin_texto`, publicado en castellano e inglés, para que la coordinación teclee los datos.

`adaptador_test.go` comprueba el trabajador y la compatibilidad de códigos con Word. `lector_test.go` conserva la comparación exacta del corpus y la desviación tipográfica conocida; añade árboles incoherentes y cíclicos. `subida_test.go` recorre POST y GET por un servidor HTTP real con validación, lector y caso de uso reales. Sesión, persistencia, catálogo y auditoría son dobles inventados; esta prueba no acredita PostgreSQL ni RLS. El recorrido con antivirus real es activable mediante `ETF_PRUEBA_CLAMAV_DIRECCION`; exige el resultado limpio, no acepta una caída como éxito y rechaza una firma EICAR dentro de un PDF.

Para reproducirlo sin usar los servicios en marcha:

```sh
python3 -B internal/adaptadores/anexos/pdf/probar_subida.py
```

El ejecutor hereda la imagen, configuración y aislamiento del servicio ClamAV de `compose.yaml`. Crea una red interna y contenedores desechables, usa firmas sintéticas en tmpfs y elimina sus recursos al terminar. No modifica la instalación existente ni descarga firmas o imágenes.

## Corpus de la edición pública

Los PDF de prueba se regeneran desde los Word inventados de `testdata/`, sin membretes. `testdata/exportar.py` usa LibreOffice, pdftoppm, Pillow, ExifTool y qpdf locales; acepta `--salida` y respeta `TMPDIR`. Produce Anexos 3/4 con texto y un escaneado rasterizado del Anexo 3, sin capa de texto. Identifica a Expedientes ETF como autor y creador y usa tablas PDF clásicas, compatibles con el lector. `corrupto.pdf` conserva el caso deliberadamente inválido y no tiene metadatos de autor.
