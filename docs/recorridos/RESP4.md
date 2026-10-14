# RESP4: Anexos 3 y 4 en ODT y PDF firmado

La respuesta D6 indica que lo habitual es recibir un PDF firmado. Las muestras inventadas remitidas por el Programa son ODT de LibreOffice convertidos desde Word. La distribución pública incluye únicamente las muestras sanitizadas de `internal/adaptadores/anexos/odt/testdata/`.

Las copias sustituyen autor y autor inicial por «Expedientes ETF» y los dominios de correo por `servicios.invalid`. Los PDF eliminan el bloque XMP y usan el mismo autor. La clave privada se crea en memoria durante la generación y se descarta; sólo se conservan los PDF con su certificado público incorporado. Es una firma autofirmada de prueba, sin sello de tiempo ni servicios externos.

## Resultado de lectura

ODT propone 86 campos del Anexo 3 y 95 del Anexo 4: dos menores, dos adultos y una persona significativa en cada uno. Limpia prefijos sueltos, fechas de dos cifras y marcadores; «ed infantil» se propone como «Educación infantil». Conserva el tramo SMI y ETF/Subprograma del Anexo 4. Los valores completos están en `esperado.json` y se contrastan con ambas muestras en Go.

Los PDF firmados y sin firmar producen los mismos campos, valores, páginas y motivos. Requieren confirmación humana. `revision_pdf.json` enumera únicamente las diferencias respecto al ODT; todas son valores vacíos:

| Campo | Qué debe revisar Coordinación |
| --- | --- |
| `valoracion.crisis` en el Anexo 3 | La conversión imprime «Casilla» y pierde el estado marcado. En el Anexo 4 las dos casillas originales ya estaban vacías. |
| `valoracion.actitud` | La marca del control no se conserva como texto legible. |
| `observaciones.documentacion_obligatoria` | Revisar en el original qué documentos se han marcado. |
| `sociodemografia.situacion_economica` | La exportación y el lector alteran el orden de algunos glifos y solapan texto. El tramo de ingresos se recupera por separado. |

No se deduce una selección a partir de una casilla sin marcar ni se conserva un marcador como dato. La aplicación lee el contenido; no valida la firma ni la confianza del certificado. Un PDF escaneado sigue abriendo la revisión para teclear, sin OCR.

## Regenerar las muestras

LibreOffice necesita sus bibliotecas locales en este puesto. Python y las dependencias de generación viven en un entorno temporal, con todas las versiones fijadas en `requirements.txt`:

```sh
python3 -m venv "$TMPDIR/resp4-fixtures"
"$TMPDIR/resp4-fixtures/bin/pip" install -r internal/adaptadores/anexos/odt/testdata/requirements.txt
"$TMPDIR/resp4-fixtures/bin/python" internal/adaptadores/anexos/odt/testdata/generar.py
```

La generación usa LibreOffice headless, pypdf 5.1.0 y pyHanko 0.25.3. La firma se verifica localmente sin consultas de red, confiando sólo en el certificado de prueba recién creado. Las fechas y el certificado cambian al regenerar; los valores de lectura no deben cambiar.

## Comprobaciones

```sh
go test ./internal/adaptadores/anexos/... -count=1
node scripts/recorridos/resp4.mjs http://127.0.0.1:18314
```

El recorrido usa la demostración de este worktree y Chrome del sistema. Inés Moreno sube el ODT del Anexo 3 y el PDF firmado del Anexo 4; ambas peticiones responden 200, con antivirus limpio, y abren la revisión con dos menores y tramo SMI. Después elige municipio, centro y equipo, confirma cada derivación con respuesta 201 y recupera el caso con GET y recarga. Comprueba que mantiene el formato original y los menores leídos. Cada ejecución crea dos casos nuevos, sin aceptar ni asignar número. Comprueba los valores de los campos, errores de navegador y ausencia de desbordamiento horizontal a 1440 y 390 px.

- [ODT a 1440 px](../capturas/resp4-odt-1440.png) y [a 390 px](../capturas/resp4-odt-390.png).
- [PDF firmado a 1440 px](../capturas/resp4-pdf-firmado-1440.png) y [a 390 px](../capturas/resp4-pdf-firmado-390.png).

La revisión focal cubre subida, ZIP, XML y el enganche al trabajador PDF. Se ejecuta Semgrep con reglas locales y métricas desactivadas; no se envía código ni datos a servicios de análisis. La comprobación de firmas de prueba no acredita certificados reales ni sustituye la revisión del original por el Programa.

La validación completa del 14/10/2026 pasó `go vet ./...`, `go test ./...`, las comprobaciones de frontend, idiomas, enlaces, migraciones y permisos, y los recorridos de Chrome. La última línea de `make ci` fue `CI mínima completada.` La captura espera a las fuentes y fija el desplazamiento antes de guardar la imagen; comprueba que el inicio del apartado está visible.

## Confirmación y actualización SQL

La migración reservada `expediente/0026` amplía el CHECK de `lectura_anexo.formato` a `docx`, `odt` y `pdf`. Comprueba la restricción original validada y los roles de migración; el cambio ocurre en la transacción del migrador, con tiempos máximos de bloqueo y ejecución. No modifica la migración aplicada, RLS ni permisos. El formato se toma del análisis del servidor.

`migraciones/expediente/ensayar.py` instala desde cero y actualiza desde el esquema anterior a `0026` con una derivación DOCX confirmada. Aplica la ampliación y repite el migrador sin duplicar la versión. La prueba Go contra PostgreSQL confirma ODT y PDF firmado, comprueba procedencia, una lectura y un evento, repite la confirmación con el mismo recibo y exige que un formato ajeno falle por el CHECK. El recorrido de Chrome confirma ambos formatos y recupera los casos desde la pantalla de decisión.
