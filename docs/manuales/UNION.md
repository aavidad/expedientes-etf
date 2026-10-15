# Comprobación de la unión de manuales

La rama `manuales-union` reúne, mediante tres fusiones consecutivas, las ramas revisadas `manual-sistemas`, `manual-programadores` y `manual-usuario-etf`. Las referencias comprobadas son `manual-sistemas` (4d8f159), `manual-programadores` (7e2634d), `manual-usuario-etf` (ad7a344). Conserva también el manual de Administración funcional recibido desde `main`.

El exportador [exportar.py](exportar.py) admite los cinco manuales. Sin selección sigue generando los tres manuales de usuario. Sistemas incluye sus cuatro anexos, tablas, órdenes por líneas, literales, negritas y enlaces relativos al árbol publicado. Programadores incluye diez capítulos, tablas con cabecera repetida, bloques de código, diagramas locales y capturas con pie. Profesionales y Coordinación incorporan la ayuda común y sus capturas de escritorio y móvil. Se conservan los tres capturadores originales.

Las fusiones conservan los enlaces del README y de la guía de documentación. La referencia a F5.02 sigue presente porque también figura en el `main` integrado y su informe existe.

## Regeneración y pruebas

Todas las operaciones se ejecutan en local, sin enviar documentos ni datos a servicios externos. Requieren LibreOffice, Pillow, Graphviz y las herramientas Poppler (`pdfinfo`, `pdftotext` y `pdfimages`). Los temporales se eliminan al terminar cada comprobación.

```sh
TMPDIR="$PWD/bin/temporales" python3 -B docs/manuales/exportar.py \
  --manual profesionales --manual coordinacion --manual administracion \
  --manual sistemas --manual programadores
python3 -B -m unittest discover -s docs/desarrollo -p 'test_*.py'
python3 -B -m unittest discover -s docs/manuales -p 'test_*.py'
TMPDIR="$PWD/bin/temporales" python3 -B docs/manuales/comprobar_union.py
python3 -B scripts/ci/indice/generar.py
make ci
```

Las tres pruebas heredadas de Programadores comprueban sus capítulos, figuras, tablas, código, autoría, enlaces y rechazo de diagramas no admitidos. Las tres pruebas de unión comprueban los anexos y estilos de Sistemas, su numeración explícita, la ayuda común de ambos perfiles, las capturas, los destinos internos y la ausencia de entradas duplicadas en el manifiesto del ODT. Administración conserva su contenido propio.

La comparación de PDF recupera los archivos publicados directamente de Git y los contrasta con los cinco regenerados. Exige igual número de páginas, igual secuencia y dimensiones de imágenes y sus máscaras de transparencia, igual número de enlaces internos y destinos de capítulos, y el mismo texto normalizado. Sólo elimina los pies de página y las diferencias de espacios. Comprueba además que todos los destinos internos apuntan a páginas existentes y que los enlaces locales del PDF publicado tienen archivo de destino.

| Manual | Rama de referencia | Páginas | Imágenes | Enlaces internos |
| --- | --- | --- | --- | --- |
| Sistemas | `manual-sistemas` | 22 | 3 | 21 |
| Programadores | `manual-programadores` | 27 | 7 | 10 |
| Profesionales | `manual-usuario-etf` | 27 | 24 | 10 |
| Coordinación | `manual-usuario-etf` | 21 | 17 | 5 |
| Administración funcional | `manual-usuario-etf` | 12 | 6 | 13 |

Los cinco PDF mantienen el texto y los capítulos de sus referencias. No hay diferencias de contenido debidas a la unión. Los bytes de los archivos cambian por los metadatos de la conversión.

Esta comprobación acredita la integración y exportación de los manuales revisados; no vuelve a ejecutar las demostraciones ni acredita decisiones de producto o condiciones de producción pendientes.

## Resultado de la validación

Las dos baterías terminaron con `Ran 3 tests` y `OK`: seis pruebas en total. La comparación de los cinco PDF terminó con cinco líneas `MANU-PDF-OK`. La comprobación de Markdown dio 186 documentos, 1129 enlaces locales y 0 errores. `make ci` completo terminó con:

```text
CI mínima completada.
```

Incluye `go vet ./...`, `go test ./...`, pruebas y compilación web, textos, temas, ensayos PostgreSQL y RLS, incorporación de recursos, entrada y permisos con Chrome, carga histórica y lectura de actas y anexos ODT y PDF firmados. El `main` local integrado es `5dd23bc`.

## Actualización posterior: MEMDES

MEMDES añade el desglose por expedientes a los manuales de profesionales y Coordinación. Tras integrar la unión se regeneraron los cinco PDF desde sus fuentes actuales y se comprobaron sus capítulos y enlaces locales. Profesionales tiene 28 páginas; Coordinación, 21; Administración, 13; Sistemas, 22; Programadores, 27. La comparación literal de `comprobar_union.py` con las ramas anteriores conserva su uso para la unión original; esas referencias aún describían el desglose como pendiente. La actualización se valida con las fuentes y los PDF actuales, según [la evidencia MEMDES](../evidencias/MEMDES.md).
