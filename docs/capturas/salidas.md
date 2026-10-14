# Salidas y cierre de Coordinación (CO2)

Datos inventados y servicios de la demo local. Chrome del sistema, mediante Playwright.

## Recorrido reproducible

Arranca con `ETF_DEMO_PUERTO=18242 make demo` y ejecuta `node scripts/recorridos/salidas.mjs http://127.0.0.1:18242`. Si cambias el puerto, ajusta ambos comandos. Las contraseñas se leen del fichero local generado por la demo; no aparecen en el guion. Para terminar, ejecuta `make demo-parar`. Antes de ejecutar `make ci` después de este recorrido, usa `make demo-limpiar`: borra los datos inventados de la demo de este worktree y permite que el ensayo de permisos arranque con su base limpia.

El guion crea derivaciones desde el Anexo 3 de prueba y las acepta en el ETF 2. Marta Cabrera propone el cierre por falta de consentimiento. Inés Moreno anota la respuesta fechada del SPM; Javier Medina descarga el Acta antes del cierre, confirma su fecha, vuelve a descargarla y reabre. Las descargas conservan estado y versión. Otro expediente prueba una excepción motivada, el cierre y la reapertura con el bloqueo del SPM todavía vigente.

También comprueba campos pendientes con resumen y enlaces al foco, ayuda con Escape y devolución del foco, conservación de borradores al guardar otra operación y reintento tras perder la respuesta de una escritura ya confirmada. Las dos peticiones del reintento conservan cuerpo y clave de idempotencia. Se comprueba el reflujo a 1440, 1024, 390 y 360 píxeles, alto contraste e inglés, sin errores de JavaScript.

## Capturas

- [Salidas y cierre en escritorio](salidas-escritorio.png).
- [Salidas y cierre en móvil](salidas-movil.png).
- [Cierres propuestos](salidas-cierres-escritorio.png).
- [Alto contraste en móvil](salidas-alto-contraste-movil.png).

## Revisión y límites

Segunda revisión independiente con `programar-interfaz-etf`, `revisar-cambios-etf`, `security-audit` focal y `humanizer`: GO de código y usabilidad tras corregir conservación de borradores, validación y ayuda. Semgrep local sin hallazgos; Gitleaks sin secretos. Los textos de personas se mantienen en castellano normal y se publican por idioma.

No hay SQL ni dependencias nuevas. Los permisos, la segunda persona, la auditoría y la persistencia corresponden a los servidores existentes. El recorrido de navegador comprueba los resultados HTTP y los estados recuperados; no inspecciona directamente la cadena de auditoría. No se ha probado NVDA.

Traslado no está en `main` al preparar CO2. La API de bandeja no admite el filtro de pendientes del SPM; esa vista queda fuera. El cierre necesita una propuesta pendiente del equipo. El motivo nuevo de Coordinación se aplica al confirmar el cierre; una descarga anterior utiliza el motivo ya guardado.
