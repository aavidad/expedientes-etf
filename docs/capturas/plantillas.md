# Comprobación de Plantillas Word (PL1)

Administración puede consultar las versiones por documento e idioma, revisar una subida Word y activarla con confirmación. La subida conserva la versión activa anterior. El historial muestra autor y fecha de las activaciones y permite volver a una versión anterior. Las versiones de distribución señalan los metadatos que no constan.

El recorrido usa Chrome del sistema, ClamAV y PostgreSQL locales, con Elena Salas Márquez para Administración e Inés Moreno Castillo para comprobar la denegación a Coordinación. Todos los datos son inventados. La plantilla válida es el Acta de Aceptación distribuida con la aplicación; la inválida cambia el marcador «numero» por «intruso» dentro del Word.

## Recorrido y capturas

Comandos ejecutados el 14/10/2026, con `TMPDIR`, caché de npm y caché de Go bajo `bin/temporales`:

```sh
ETF_DEMO_PUERTO=18231 make demo
node scripts/recorridos/plantillas.mjs http://127.0.0.1:18231
make demo-parar
make ci
```

Salida del recorrido:

```text
PL1-OK: Coordinación denegada; subida inválida con marcadores; subida válida sin activar; confirmación por teclado; autor y fecha; conflicto 409; restauración; 1440/1024/390/360; alto contraste e inglés; sin errores de navegador.
DEMO-PARADA
```

La subida inválida devolvió 422 con el marcador faltante y el sobrante, confirmó antivirus limpio y conservó la activa. La subida válida devolvió 201 y quedó pendiente de activar. Escape cerró la confirmación y devolvió el foco al botón; Enter volvió a abrirla. La confirmación activó la versión y registró autor y fecha. Una revisión antigua devolvió 409; la restauración conservó el historial. Coordinación recibió 403 en la API y en la pantalla.

Carga, vacío y error de lectura se comprobaron mediante respuestas simuladas del navegador; «Actualizar lista» recuperó después los datos reales. Las escrituras, la confirmación y el conflicto usaron el servidor real. También se comprobaron encabezado único, controles de al menos 44 px, ausencia de desplazamiento horizontal y de datos en almacenamiento del navegador.

- [Escritorio, 1440 px](plantillas-escritorio.png).
- [Móvil, 390 px](plantillas-movil.png).
- [Escritorio con alto contraste](plantillas-contraste-escritorio.png).
- [Móvil con alto contraste](plantillas-contraste-movil.png).

## Servidor, migración y revisión

`make ci` ejecutó tipos y pruebas del frontend, textos, temas, `go vet ./...`, `go test ./...`, compilación y los ensayos PostgreSQL 18 de todos los módulos presentes. La salida relevante fue:

```text
# pass 17
# fail 0
Textos: 2 idiomas, 477 claves usadas; sin literales visibles.
Temas: sin colores fijos en Go ni web/src.
--- PASS: TestGestionPlantillasPostgres (0.12s)
PL1-AUDITORIA-OK: fallo de auditoría revierte referencia e historial
CI mínima completada.
```

El ensayo de Documentos aplica `documento/0012` sobre la base anterior y sobre una base limpia, y comprueba la repetición del ejecutor. La prueba de gestión verifica subida sin activar, versión duplicada, revisión obsoleta, restauración y reactivación, autor y fecha, repetición sin cambiar revisión, rechazo por perfil, lectura del historial protegida por RLS y rechazo de un fichero alterado. El ensayo fuerza un fallo de auditoría y comprueba que no queda ni la nueva referencia activa ni su activación histórica.

Dos revisores independientes trabajaron en modo de sólo lectura: uno con `revisar-cambios-etf`, `revisar-sql-etf` y `security-audit` focal; otro con `programar-interfaz-etf`, `probar-recorridos-etf` y `humanizer`. Ambos dieron GO tras comprobar las correcciones. El revisor de seguridad no ejecutó pruebas por las restricciones de su entorno; los ensayos anteriores los ejecutó el autor. Semgrep se ejecutó localmente sobre los ficheros Go y TypeScript cambiados, con las tres reglas locales de `scripts/ci/temas/seguridad.yml`, métricas y comprobación de versión desactivadas: cero hallazgos. Gitleaks tampoco encontró secretos en la puerta completa.

La comprobación de teclado y capturas no incluye una sesión con NVDA. `gopls` no estaba disponible en la terminal; los tipos y el análisis Go se comprobaron con las herramientas indicadas.
