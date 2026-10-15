# SEG1: privacidad de Consulta y duración de sesión

Decisión del responsable del producto del 14/10/2026 sobre R-09 y R-13. Rama `seg-consulta-sesion`; implementación en `80cb7dcd` y `d160f24e`; candidata integrada de código `254948d7`, sobre main `013995f2`.

## Comportamiento

Consulta recibe las cifras inferiores al umbral como `null` en la API y «Oculto» en pantalla y Word. El umbral inicial y mínimo es 3 casos; Administración puede aumentarlo en Plazos y tramos. Se aplica la regla vigente al consultar, también a fotos cerradas. La supresión secundaria recorre filas y columnas y oculta la siguiente menor cuando sólo hay una oculta; se repite hasta estabilizarse. Se retiran márgenes y porcentajes que permitirían reconstruir las cifras entre apartados. Los cálculos originales y las fotos cerradas se conservan. Equipos y Coordinación mantienen sus cifras completas.

`catalogo/0016`, reservada en el registro externo y dependiente de `catalogo/0015` tras integrar Ubicaciones, publica otra versión sin modificar las anteriores y conserva parámetros, duraciones y referencias de idioma. Si la última versión empieza hoy o después, publica para el día siguiente a esa versión. Hasta su vigencia se aplica 3. La nueva política de Consulta sólo permite leer ese parámetro publicado, sin abrir los demás ni conceder escrituras.

La sesión termina por inactividad a los 30 minutos y por duración absoluta a las 8 horas. Ambos valores son configurables; el máximo técnico es 24 horas y la inactividad debe superar los dos minutos de aviso. La lectura de sesión no renueva actividad. La continuación exige CSRF y no desplaza el límite absoluto. El aviso permanece visible al desplazarte. Al caducar se vuelve a la entrada; lo guardado permanece.

## Evidencia ejecutada

- Pruebas Go: matriz 2×2 calculada aparte, selección de la siguiente menor, ceros, umbral configurable, JSON sin cifras ocultas, fuente intacta y conservación de cifras seguras. Word protegido es/en, incluidas casillas sin selección; sesión pasiva, continuación y límites exactos de inactividad y duración absoluta. Pruebas focales verdes.
- `python3 -B migraciones/catalogo/ensayar.py`: instalación limpia, repetición, actualización y RLS. Salida: `PRIVACIDAD-RLS-OK Consulta usa umbral 4; sin contexto no ve reglas; otros parámetros cerrados`. Contenedor y volumen destruidos.
- `python3 -B migraciones/estadistica/ensayar.py`: fotos cerradas y cálculo abierto, Word y aislamiento. Salida: `CONSULTA-CONCURRENCIA-OK 6 lectores, cierre antes de liberar cálculo, 33 apartados coherentes y foto sin bloqueo asesor`.
- `python3 -B migraciones/expediente/ensayar.py`: `ENSAYO-OK base limpia, RLS, auditoría, EICAR y ClamAV caído`. El reloj de la prueba de reapertura de seguimiento se ajusta a la fecha efectiva de la regla que publica; evita asumir que toda publicación entra en vigor hoy.
- `node scripts/recorridos/seg1.mjs`, después de `make demo`, puerto 18191: API y Word reales con Consulta; sesión de Coordinación con `ines.moreno`, lectura pasiva sin renovación, aviso y continuación con Enter, máximo absoluto y vuelta a la entrada tras caducar. Las fechas de una sesión inventada se adelantan en el PostgreSQL de esta demo para reproducir los límites; la denegación la decide el servidor. La caducidad se prueba también con SMI modificado y el aviso de cambios sin guardar abierto: no aparece confirmación nativa de navegación ni queda una acción pendiente sobre la entrada. Las cifras guardadas de SMI coinciden tras volver a entrar. axe no encuentra incumplimientos WCAG A/AA en el aviso; el menú de persona sigue siendo accesible cuando aparece. Inglés, alto contraste y 360 px sin desbordamiento. Las capturas de 1440/390 están enlazadas en README.
- Semgrep local, sin métricas ni servicios externos: 17 reglas sobre los 29 ficheros Go/TypeScript cambiados, cero hallazgos. Sin dependencias nuevas.
- Revisión independiente de fuente, SQL/RLS, seguridad focal y usabilidad con las skills ETF: GO para la candidata integrada `254948d7`, incluida la corrección que limpia cambios y acciones pendientes al caducar. La integración conserva las implementaciones de sesión y supresión revisadas y añade las entregas aprobadas de auditoría, Ubicaciones y SEG2. La dependencia 0016→0015 garantiza el orden de instalación; la demo inventada se recrea para no intentar aplicar 0015 sobre una base donde ya se había ensayado 0016. No ejecutó pruebas el revisor; los ensayos y recorridos anteriores los ejecutó quien implementó.

Los PDF de Administración, Sistemas y Programadores se regeneraron con `python3 -B docs/manuales/exportar.py --manual administracion --manual sistemas` y `python3 -B docs/manuales/exportar.py --manual programadores`; Administración se regeneró después de integrar las otras entregas. Salida `PDF-OK` para los tres; el PDF de Administración conserva también los apartados de las entregas integradas. Se comprobó con `pdftotext` la presencia del umbral y de los límites de sesión.

La espera de navegación de `scripts/recorridos/carga-actas.mjs` pasa de 20 a 60 segundos tras un timeout de `goto` con carga concurrente de la máquina. Las esperas de los controles conservan 20 segundos; no se omite ninguna comprobación del recorrido.

La prueba PWA de main agotó una navegación de la segunda pestaña después de simular desconexión; pasó al ejecutarla aislada. El recorrido vuelve a establecer el contexto en línea y activa esa pestaña antes de navegar. Con ese estado explícito pasó el bloque completo de entrada en 266 segundos, pero otra ejecución completa volvió a agotar la espera de DOMContentLoaded. La navegación de esa pestaña espera ahora el compromiso de la respuesta y después exige las condiciones reales ya comprobadas por el recorrido: documento en castellano y worker controlador. El recorrido focal pasó con ese ajuste. Conserva las comprobaciones de dos pestañas, es/en, desconexión y CacheStorage vacío; la causa del timeout de ese evento no está confirmada.

El recorrido de Ubicaciones devolvió un 403 al publicar y al reabrir la Memoria durante la limpieza de la CI completa. Dos repeticiones aisladas pasaron, incluida la que parte de borrador y prueba cierre y reapertura. Al conservar ambos errores y contrastar el registro HTTP se confirmó la causa: el GET de sesión devolvía 429 porque los recorridos anteriores habían consumido la ventana compartida de 100 peticiones por IP y minuto; el helper usaba después un CSRF vacío. La CI reinicia sólo su app de demo antes del bloque independiente de Ubicaciones, como ya hacía antes del bloque de carga. No borra datos ni modifica la política productiva. El helper comprueba ahora el estado del GET de sesión y el recorrido conserva ambos fallos si también falla su limpieza. No se relaja ninguna comprobación.

## Cierre de validación del 14/10/2026

`ETF_DEMO_PUERTO=18191 make ci` terminó con código 0 sobre `63ab3d67`, con main `013995f2` integrado. Incluye `go vet ./...`, `go test ./...`, frontend, textos, temas, secretos, instalación limpia, actualización y repetición de migraciones, PostgreSQL/RLS y todos los recorridos de Chrome. La demo se recreó antes de esta ejecución: la limpieza fallida anterior había dejado la Memoria global cerrada, mientras el recorrido de Consulta parte de borrador. No se cambió su aserción de cifras provisionales.

Salidas de la ejecución completa:

```text
PRIVACIDAD-RLS-OK Consulta usa umbral 4; sin contexto no ve reglas; otros parámetros cerrados
CONSULTA-CONCURRENCIA-OK 6 lectores, cierre antes de liberar cálculo, 33 apartados coherentes y foto sin bloqueo asesor
F0.11-RLS-OK: 7 actores; 54 tablas pobladas; 378 comprobaciones de lectura; 810 escrituras denegadas.
F0.11-VISTAS-OK: 9 vistas; 63 lecturas con RLS.
F0.11-OK: 292.8 s
CI mínima completada.
```

El recorrido final propio también terminó con código 0 sobre la demo recién creada:

```text
SEG1-CHROME-OK Consulta API/Word (154 celdas ocultas); sesión pasiva, aviso, teclado, continuación, límite absoluto y caducidad; capturas 1440/390
```

La revisión independiente dio GO a los ajustes de espera de PWA y al aislamiento de la ventana por IP de Ubicaciones. El código funcional no cambió después de su revisión. Se actualizaron las capturas propias y se restauraron las capturas de otros módulos generadas por CI. Se ejecutó `make demo-parar`; los temporales propios quedaron eliminados. Sin push ni integración de esta rama en main.

## Límites

La supresión secundaria es la sencilla encargada. No demuestra anonimización frente a todas las comparaciones posibles entre años o publicaciones; ese criterio sigue sujeto al DPD. AD se vuelve a comprobar al entrar, no durante una sesión vigente. La comprobación automática de accesibilidad y el recorrido de teclado no sustituyen una validación completa con lector de pantalla.
