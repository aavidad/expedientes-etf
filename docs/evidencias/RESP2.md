# RESP2 · Carga desde actas

Rama `carga-desde-actas`. D21, respuesta del Programa del 14/10/2026:
no existe hoja; se cargan desde actas o a mano todos los expedientes no cerrados
de cualquier año anterior, los cerrados durante 2026 y las derivaciones no
aceptadas de 2026. 

El módulo CA1 existente aporta la confirmación histórica, idempotencia, historia,
marca de revisión y numeración. RESP2 añade lectura de aceptación y cierre opcional
en `/api/carga-2026/actas/leer/{tipo}` y la sección de actas del alta asistida.
La composición existente de `carga` en ambos registros sigue siendo la misma;
no se añade un módulo duplicado. No hay migración ni cambio SQL.

La lectura aplica el validador de anexos, ClamAV y trabajador PDF efímero acotado,
con modo de actas separado de derivaciones. Sólo conserva propuestas en memoria.
Confirma auditoría antes de entregar datos; infección bloquea y caída de antivirus
continúa con aviso visible y auditado, como anexos. No guarda fichero, nombre,
borrador ni huella del archivo. La confirmación guarda sólo datos corregidos.
PDF sin texto se completa a mano. La fecha declarada de firma se propone si falta
fecha explícita; no verifica firmas ni usa OCR. No deduce estado, sexo, fechas de
derivación ni el corte entre nombre y apellidos.

Si ya hay datos en el formulario, leer otras actas pide confirmar su sustitución;
cancelar conserva los datos y el foco. Los nombres completos leídos se separan
a mano antes de revisar; el original queda como referencia, fuera del contrato de
confirmación. La vista previa enmascara personas y separa adultos y menores.

## Corpus inventado

[Datos del corpus](../../internal/adaptadores/anexos/actas/testdata/datos.json):
Familia Villena Robles, Irene Robles Serrano, Daniel Villena Márquez y Clara Villena
Robles; ETF 02, Santa Fe. Aceptaciones de 2022 y 2023; cierre del segundo en 2026.
Se parte de `idiomas/es/plantillas/entrada/aceptacion/v1.1.1.docx` y
`idiomas/es/plantillas/cierre/acta_cierre/v1.0.1.docx`, con el motor Word existente.
El corpus añade fechas explícitas y subprograma conocidos, sin modificar plantillas.

- [Aceptación de 2022, Word](../../internal/adaptadores/anexos/actas/testdata/aceptacion-2022.docx), [PDF](../../internal/adaptadores/anexos/actas/testdata/aceptacion-2022.pdf) y [PDF firmado](../../internal/adaptadores/anexos/actas/testdata/aceptacion-2022-firmada.pdf).
- [Aceptación de 2023, Word](../../internal/adaptadores/anexos/actas/testdata/aceptacion-2023.docx) y [PDF](../../internal/adaptadores/anexos/actas/testdata/aceptacion-2023.pdf).
- [Cierre de 2026, Word](../../internal/adaptadores/anexos/actas/testdata/cierre-2026.docx) y [PDF](../../internal/adaptadores/anexos/actas/testdata/cierre-2026.pdf).

Conversión con LibreOffice 26.2.6.3 local; firma con pyHanko 0.25.3 y certificado
RSA autofirmado generado con OpenSSL. Validación criptográfica local, sin consulta
externa; la clave privada sólo vive en el temporal y se elimina. Se conserva
únicamente el [certificado público de prueba](../../internal/adaptadores/anexos/actas/testdata/certificado-prueba.pem).
Las dependencias del generador están [fijadas](../../scripts/actas/requirements.txt).

Regeneración con datos inventados, nunca actas reales:

```sh
python3 -m venv /dev/shm/etf/tmp/resp2-tools
/dev/shm/etf/tmp/resp2-tools/bin/pip install -r scripts/actas/requirements.txt
/dev/shm/etf/tmp/resp2-tools/bin/python scripts/actas/generar.py
```

Salida real del generador:

```text
RESP2_FIRMA_LOCAL: intacta y criptográficamente válida; certificado autofirmado de prueba.
```

## Pruebas

Go focal: lectores Word/PDF/firmado, PDF sin texto, separación de modos,
perfil y equipo, CSRF, multipart, límites, antivirus, auditoría y años anteriores.
Pruebas de interfaz: datos desconocidos, números incompatibles, motivo libre,
nombres sin inferencias, ausencia de marcas de lectura en el contrato.

Chrome del sistema con Inés Moreno y Marta Cabrera, datos inventados y
confirmación real de abierto de 2022 y cerrado de 2023 durante 2026; números y
adultos comprobados por lecturas auditadas. Anchos 1440, 1024, 390 y 360,
sin almacenamiento local ni errores de JavaScript. [Recorrido ejecutable](../../scripts/recorridos/carga-actas.mjs).

[Inicio 1440](../capturas/carga-actas-inicio-1440.png) · [Inicio 390](../capturas/carga-actas-inicio-390.png) ·
[Datos 1440](../capturas/carga-actas-datos-1440.png) · [Datos 390](../capturas/carga-actas-datos-390.png) ·
[Resultado 1440](../capturas/carga-actas-resultado-1440.png) · [Resultado 390](../capturas/carga-actas-resultado-390.png).

La revisión independiente de seguridad detectó y corrigió aislamiento del lector,
contraste del equipo leído y persistencia de adultos. La segunda pasada de usabilidad
pidió proteger datos escritos, separar nombres sin inferencias y aclarar el resumen
y la entrada de carga por hoja. La revisión independiente de seguridad da GO por fuente. La revisión final independiente de usabilidad da GO tras Chrome: cancelar conserva datos y foco, los nombres sin separar bloquean la revisión, la vista previa distingue adultos y menores, no muestra nombres completos y no desborda a 390 px.
No se ha probado NVDA ni la lectura de todas las variantes de actas antiguas;
los datos que no se leen se completan a mano. No autoriza datos reales ni despliegue.

Salidas reales de las pruebas focales y del ensayo de PostgreSQL:

```text
ok  github.com/aavidad/etf-expedientes/internal/adaptadores/anexos/lectura
ok  github.com/aavidad/etf-expedientes/internal/aplicacion/expediente
ok  github.com/aavidad/etf-expedientes/internal/adaptadores/http
# tests 8
# pass 8
# fail 0
RESP2_POSTGRES_OK: abierto 2022 conserva 08-01-22; carga no consume contador; nueva aceptación 2026 salta correlativo histórico ocupado
RESP2_CHROME_OK: aceptación firmada 2022, aceptación Word 2023 + cierre PDF 2026, datos confirmados y adultos en ficha; permisos propio equipo, sin almacenamiento, 1440/1024/390/360 y errores sin datos.
```

El recorrido también verifica alto contraste, inglés y cancelación de la
sustitución con foco conservado. Los PDF de profesionales y Coordinación se
han regenerado con LibreOffice local; `pdftotext` confirma los pasos de actas y
la separación de nombre y apellidos.

## Cierre de la entrega

`make ci` completo terminó con salida 0. Incluye `go vet ./...`, `go test ./...`,
pruebas de interfaz, textos, arquitectura, ensayos PostgreSQL 18 y recorridos Chrome.
La matriz INT5 comprobó 1.644 combinaciones; RLS comprobó 53 tablas con 7 actores,
371 lecturas y 795 escrituras denegadas. CA1 y RESP2 terminaron verdes.

```text
CI mínima completada.
Ran 14 rules on 19 files: 0 findings.
```

Semgrep se ejecutó localmente, con métricas y comprobación de versiones desactivadas.
La regla que impide temporales señaló la escritura de Word del generador de fixtures;
la excepción queda sólo en esa línea, justificada por el corpus inventado solicitado.
Ninguna lectura de actas del servidor escribe ficheros. La revisión independiente
sensible da GO para `e1ad804`; la de usabilidad da GO con el recorrido final.

La primera CI detectó una expectativa de la matriz de permisos: el multipart de
actas devuelve 400 al recibir JSON sin equipo ni fichero. Se adaptó esa expectativa,
manteniendo las pruebas de equipo ajeno y CSRF. Otra ejecución encontró un timeout
de PWA; pasó al repetir sin cambios y la siguiente CI completa terminó verde.

Se integró el `main` actual. La última integración de documentación no cambió código, SQL ni configuración del
alcance comprobado. No se ha hecho push ni se ha fusionado la rama en main.

Los únicos enganches comunes modificados por RESP2 son `cmd/etf/carga.go`,
`scripts/ci/comprobar.sh` y la expectativa multipart de
`web/tests/entrada-permisos.navegador.mjs`. Los registros de módulos ya contienen
`carga`; se amplió esa entrada existente, sin duplicarla. Los demás cambios
pertenecen a lectores/puertos de actas, carga histórica, su pantalla, idiomas,
pruebas, corpus, capturas, manuales y documentos de producto.
