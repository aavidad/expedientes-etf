# RESP3 · Escalas opcionales

Decisión D17 de Alberto, 14/10/2026. Las escalas son opcionales en sus tres momentos. El Programa puede anotarlas directamente en el Word de los Anexos 11 y 12.

## Comportamiento

La extracción ya no añade el aviso de VALÓRAME pendiente y elimina ese aviso si procede de una lectura anterior. Evaluación muestra una línea de opcionalidad en castellano e inglés. No hay un aviso fijo de baremo pendiente ni se presentan los cuestionarios no publicados como datos que debas completar.

Los Anexos 11/12 es/en v1.0.2 conservan las tablas oficiales. Se elimina la protección de formulario en estas versiones nuevas. Sus casillas no anotadas quedan vacías y editables, sin «Sin dato», «Puntuación» ni opciones de muestra. Las versiones anteriores conservan sus bytes y referencias. Las pruebas comprueban que los Word nuevos no conservan la protección de formulario. `documento/0015` avanza las referencias del paquete inicial y respeta las publicaciones y activaciones de Administración.

La emisión lee las evaluaciones por el puerto del módulo dentro de su transacción. Copia los resultados y guarda las versiones de aplicaciones y definiciones. No suma respuestas ni aplica baremos. `campo_documento` en una definición inmutable asocia un resultado a una fila de la plantilla; el momento identifica su columna.

Varios sujetos en una casilla: iniciales y resultado de cada sujeto, separados por «; », sin nombres completos ni mezcla de puntuaciones, como decidió Alberto. También se conservan las iniciales de sujetos que ya no figuren en la composición vigente. Un cero anotado se mantiene como cero; una casilla sin anotación sigue vacía.

## Pruebas

`go test ./internal/aplicacion/evaluacion ./internal/aplicacion/documento ./internal/dominio/evaluacion ./internal/dominio/expediente ./internal/adaptadores/docx ./internal ./cmd/etf`: paquetes en verde. Ocho casos Word cubren es/en, Anexos 11/12, ninguna escala y resultados parciales; incluye varios sujetos, puntuación cero y versiones. El inventario de plantillas comprueba las 62 versiones y 124 generaciones. Las pruebas de lectura del puerto cubren perfiles indebidos, equipo ajeno y fallo sin entrega de datos.

`python3 -B migraciones/documento/ensayar.py`: ensayo desechable PostgreSQL 18 con datos inventados; base limpia, actualización desde la versión anterior, RLS y auditoría. Incluye una copia del esquema de main anterior a `documento/0015`, actualización, repetición y conservación de una activación administrativa. Otro ensayo parte del corte anterior a 0013, publica una v1.0.1 administrativa y comprueba que se conserva sin impedir la creación y activación de las cuatro v1.0.2 iniciales.

`node scripts/recorridos/resp3.mjs http://127.0.0.1:18133`: Chrome del sistema, Marta Cabrera y coordinación con Inés Moreno. Crea expedientes desde los Anexos 3 y 4 inventados. Comprueba Word vacío y parcial de ambos subprogramas, estados e hitos conservados, cierre sin escalas ni Anexo emitido, cierre con resultados parciales y consulta y cierre de Memoria sin exigir escalas. Comprueba teclado, inglés, alto contraste y anchos 1440, 390, 360 y 1024.

El catálogo de distribución sigue sin cuestionarios oficiales. Para comprobar el guardado y el volcado de resultados, el recorrido instala temporalmente una definición de prueba de un único resultado de SCORE-15 en su propia demo. No es un cuestionario ni un baremo oficial; no se incorpora a los catálogos distribuidos. La prueba escribe por la pantalla, lee por la API real, usa PostgreSQL y descarga el Word real. Restaura el fichero de reglas al terminar.

## Capturas

- Sin escalas: [escritorio](../capturas/resp3-sin-escalas-escritorio.png) y [móvil](../capturas/resp3-sin-escalas-movil.png).
- Resultados parciales: [escritorio](../capturas/resp3-parciales-escritorio.png) y [móvil](../capturas/resp3-parciales-movil.png).

## Revisión y cierre

Las revisiones independientes detectaron dos correcciones: alinear Diseño UI/UX con D17 y no depender de una v1.0.1 inicial al insertar las plantillas nuevas, pues Administración puede haber publicado esa misma versión. Ambas se han corregido; el caso SQL tiene ensayo propio. La candidata corregida `42cf3ad` recibió GO independiente de código/SQL con `revisar-cambios-etf`, `revisar-sql-etf` y `security-audit` focal, y GO de usabilidad/documentación con las skills de interfaz y los criterios de diseño ETF. Las revisiones fueron de sólo lectura. El posterior merge de main conservó el incremento y regeneró el índice de producto; añadió las comprobaciones de instalación Windows de main.

Resultados finales, realmente ejecutados:

```text
RESP3-SQL-OK: actualización desde main, repetición y activación administrativa conservada
RESP3-PUBLICACION-OK: colisión administrativa v1.0.1 conservada; cuatro referencias iniciales actualizadas
RESP3-NAVEGADOR-OK: Inés y Marta; Chrome del sistema; 1440/390/360/1024 sin desbordamiento; teclado, inglés y alto contraste.
Ran 11 rules on 20 files: 0 findings.
CI mínima completada.
```

`make ci` completo pasó de nuevo sobre la rama integrada. Incluye `go vet ./...`, `go test ./...`, frontend, textos, ensayos PostgreSQL y matriz de permisos. El recorrido RESP3 final se ejecutó después, sin solaparse con la matriz que cambia perfiles de la demo. Acepta el aviso nativo de salida sólo para descartar el estado de formulario de la preparación de prueba; cada escritura se confirma y se recupera por la API real.

Semgrep se ejecutó localmente con tres ficheros de reglas del repositorio, métricas y comprobación de versión desactivadas, sin token y con comparación contra main. No hubo hallazgos nuevos. La referencia a `net/url` que una regla general señala en el fichero histórico del motor es análisis de direcciones, sin llamadas a la red, y ya existía en main.

Los MCP de índice de código y PostgreSQL no estaban disponibles en esta sesión. Se usaron búsquedas concretas con `rg`, `gopls` en terminal y los ensayos locales de PostgreSQL 18 sobre esquemas clonados y bases limpias; no se accedió a cidonia. No se acredita una revisión completa con lector de pantalla ni conformidad WCAG de toda la aplicación.

No se añaden pantallas ni rutas: se reutiliza la pestaña Evaluación y el montaje de Documentos. Los cambios de composición se limitan a `cmd/etf/evaluacion.go` y `cmd/etf/documentos.go`. Los textos nuevos y revisados están en datos de idioma.
