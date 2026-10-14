# UI3: retoques de la interfaz y de la demostración

Fecha: 14/10/2026. Rama: `ui3-retoques`. Datos inventados. Chrome del sistema y servidor real de la demostración, con Inés Moreno y Marta Cabrera.

## Cambios comprobados

- Los avisos de Menores dicen el campo y la persona: «Fecha de nacimiento · Menor 2 (Daniel)». El enlace abre su sección y lleva el foco al campo, también en consulta de Coordinación.
- La siembra usa nombres distintos dentro de cada familia, sexo coherente, fecha de nacimiento y escolarización. Los casos vivos tienen intervención con listas conocidas; el absentismo excluye a los menores sin escolarizar. Los casos históricos incluyen menores desde la aceptación, también cuando están cerrados.
- La decisión reúne los datos ausentes en un bloque plegado con su cantidad. Los datos con valor siguen visibles; no se ocultan los campos necesarios para decidir. Las fechas de nacimiento se muestran en formato del idioma.
- La bandeja enlaza desde el número o desde la fecha de recepción y municipio. Mantiene las familias enmascaradas.
- La cabecera muestra el nombre publicado del equipo mediante las API autorizadas de Equipo. No amplía permisos de Administración.
- El menú de Coordinación agrupa sus entradas autorizadas en Derivaciones, Expedientes, Informes y Memoria, y Configuración.

## Recorrido reproducible

```sh
ETF_DEMO_PUERTO=18343 make demo
node scripts/recorridos/ui3.mjs http://127.0.0.1:18343
make demo-parar
```

Si ya existe una demostración con la siembra anterior, recrearla con `make demo-limpiar` antes de arrancar. Elimina únicamente los datos locales inventados de esa demostración. No se reinician ni modifican otras instalaciones.

El recorrido guarda diez PNG a 1440 y 390 px, enlazados en el apartado de capturas del README. Comprueba un título principal, ausencia de desplazamiento horizontal de la página a 1440, 1024, 390, 360 y 320 px, teclado del menú y del bloque plegable, foco de los avisos, castellano e inglés y temas normal y de alto contraste. Los avisos se provocan retirando dos datos de un menor inventado y se restauran al terminar.

## Lista de comprobación de diseño

Se ha repasado DISENO_UI_UX §10. Los rangos originales del encargo se desplazaron al incorporar otras entregas; se usa el encabezado de la lista.

- Claves y paridad de idiomas comprobadas; no se añaden colores, medidas, fuentes ni CSS.
- Se conservan las acciones, los términos del Programa y la ayuda plegada. El nuevo plegado tiene nombre y cantidad; no añade una acción de negocio.
- Listados con iniciales, nombres completos sólo en ficha y revisión. Sin nombres en direcciones ni título de pestaña.
- Teclado del menú, del plegado y de los avisos comprobado; foco visible y etiquetas asociadas conservados. Contraste y tamaños se toman de los temas comunes, validados por la CI.
- Carga, error y reintento de equipo previstos; las demás pantallas conservan sus estados existentes. Los avisos indican lo que falta en texto, además del color.
- Inglés, alto contraste y reflujo comprobados en navegador. La tabla SPAF mantiene su desplazamiento dentro del contenedor.
- Emisiones, hitos, Word, perfil Consulta, auditoría, cargas y paquetes: sin cambios de comportamiento en esta entrega.
- No se ha ejecutado NVDA; no hay pantallas nuevas ni plantillas Word modificadas. La revisión con lector de pantalla queda pendiente.

## Pruebas y revisión

- `npm test`: 24 pruebas aprobadas, ninguna fallida.
- `make textos`: 8 pruebas aprobadas; claves usadas y paridad de los dos idiomas conformes.
- `node scripts/recorridos/ui3.mjs http://127.0.0.1:18343`: recorrido real aprobado, diez capturas y ningún error de JavaScript.
- Semgrep local: una regla de TypeScript sobre seis ficheros de pantalla; cero hallazgos. Se usan las reglas locales de Equipo y métricas desactivadas. No se envía código a servicios externos.
- Gitleaks: historial y ficheros de la candidata sin secretos detectados.
- `node web/tests/armazon.navegador.mjs`: «ARMAZON-CHROME-OK»; API simulada, 256–1440 px, es/en, dos temas, teclado, menús y guardia de cambios. Se actualiza la expectativa de equipo y la pantalla inicial actual de Coordinación.
- `make ci`: «CI mínima completada.» Incluye `go vet ./...`, `go test ./...`, compilación, ensayos PostgreSQL, auditoría y autorización con Chrome.
- `make demo-parar`: «DEMO-PARADA».
- `git merge main`: «Already up to date.»
- Segunda revisión independiente con `programar-interfaz-etf`, `revisar-cambios-etf`, `security-audit` focal y `humanizer`: conformes los puntos 1 a 5 y 7 tras revisar código, documentos y capturas. El punto 6 conserva la dependencia descrita abajo.

## Dependencia del punto 6

La siembra histórica guarda tres menores para los tres casos de ETF 02 en septiembre de 2026. Sus fichas confirman número, nombre y nacimiento. SPAF devuelve dos altas, dos bajas y tres casos, pero todavía cero menores conocidos y tres casos sin dato de menores.

La causa está en el lector estadístico: `internal/adaptadores/postgres/expediente/estadistica.go` no conecta las colecciones históricas de menores e intervención. PLAN_IMPLEMENTACION F4.05 ya declara esa conexión pendiente. No se cambian fórmulas ni se inventan cifras para ocultarlo. Por la misma dependencia, completar la siembra no basta para que la Memoria cuente esos menores. Se ha consultado al director si esa ampliación corresponde a UI3 o a la tarea de backend.

No se han modificado `web/src/salidas/` ni `web/src/carga/`. No se añaden módulos, rutas ni migraciones; no hacen falta entradas nuevas en los registros de composición.

Esta comprobación se realizó antes de F4.02. FX1 conecta la lectura histórica de menores e intervención usada por F4.02 para Memoria; el [recorrido de Memoria](memoria.md) acredita la lectura con los datos de la demo actual. Los campos sin fuente siguen como «Sin dato».
