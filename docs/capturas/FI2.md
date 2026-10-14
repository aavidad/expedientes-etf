# FI2 · Consentimiento y Evaluación

Datos inventados de una demo local recién preparada, sin registros repetidos de los reintentos de desarrollo. Chrome del sistema, Playwright y sesiones de Inés Moreno (Coordinación) y Marta Cabrera (ETF 2).

## Qué se ha comprobado

La pestaña Consentimiento descarga el Anexo 1 en Word sin cambiar el estado, la versión ni los hitos del expediente. La firma se confirma aparte; pasa a intervención. También se ha descargado el Anexo 2 de un expediente de Reunificación y se ha registrado la fecha del Anexo 8. El Anexo 7 se prepara fuera de la aplicación y sólo se registra su fecha. La guardia de cambios sin guardar conserva la fecha al elegir seguir editando.

Evaluación registra instrumentos para dos adultos en sus momentos inicial, intermedio y final. Las lecturas posteriores recuperan persona y momento. Marta puede acceder a los expedientes del ETF 2; un expediente de otro equipo queda fuera de su ámbito. Coordinación puede abrir ambas pestañas.

Se ha comprobado el reflujo a 360, 390, 1024 y 1440 px, sin desplazamiento horizontal de la página. Las capturas se guardan a 1440 y 390 px. El recorrido también cambia el idioma a inglés y activa el tema de alto contraste.

## Capturas

- [Consentimiento en escritorio](fi2-consentimiento-escritorio.png).
- [Consentimiento en móvil](fi2-consentimiento-movil.png).
- [Evaluación en escritorio](fi2-evaluacion-escritorio.png).
- [Evaluación en móvil](fi2-evaluacion-movil.png).

## Revisión y límites

Una segunda persona ha revisado el código y las cuatro capturas con `programar-interfaz-etf`, `revisar-cambios-etf`, `security-audit` en modo focal y `humanizer`. Resultado: GO de código y visual, tras corregir el campo oculto de Otros, el refresco de las emisiones y la conservación del aviso de descarga. Semgrep se ejecuta localmente, sin enviar código.

Los cuestionarios oficiales y los baremos siguen pendientes de D17. No se han añadido ítems oficiales ni puntuaciones totales. Los formularios por ítem consumen definiciones publicadas; el recorrido distingue las pruebas de interfaz con una fixture técnica de las operaciones reales del servidor. PostgreSQL comprueba respuestas por ítem, vigencia del adulto, conflictos, idempotencia, permisos y auditoría con su fixture de integración.

No se ha ejecutado NVDA ni el comprobador de accesibilidad de Word. La revisión visual del tema de alto contraste no equivale a una medición completa de contraste WCAG.

## Resultados

- `node scripts/recorridos/fi2.mjs http://127.0.0.1:18244`: `FI2-OK: recorrido real completado; sin errores JavaScript.`
- Fixture de interfaz: `FI2-FIXTURE-OK`, con respuestas numéricas y de opción, valor cero, conflicto 409, recuperación, aviso de plazo y denegación.
- Semgrep local: `Ran 3 rules on 18 files: 0 findings.`
- Textos: `Textos: 2 idiomas, 738 claves usadas; sin literales visibles.`
- PostgreSQL: `TestEvaluacionPersistenciaHTTPRLSAuditoria` pasa con las comprobaciones añadidas de instrumentos por adulto, reintento sin duplicado y rechazo de adulto ajeno.

- `make ci`: `CI mínima completada.` Incluye `go vet ./...`, `go test ./...`, pruebas del frontend y ensayos de PostgreSQL.

## Repetir el recorrido

Tras `make demo`, ejecuta `node scripts/recorridos/fi2.mjs http://127.0.0.1:18244`, ajustando el puerto. El recorrido usa únicamente el origen local, lee las credenciales del fichero privado de la demo y crea datos inventados. Reutiliza los instrumentos de prueba ya registrados; crea un expediente de Reunificación desde el Anexo 4 de pruebas. Termina los servicios con `make demo-parar`.

El empaquetado de la demo incorpora el catálogo de tramos de Familia, con permiso de lectura para el usuario de la aplicación. Esto corrige el 503 previo al dar de alta adultos; no cambia el empaquetado de producción.

## Integración con FI1 y F3.02c

La ficha muestra, en este orden: Familia, Menores, Intervención, Consentimiento, Evaluación, Documentos, Cierre, Salidas e Historial. Un registro común relaciona cada código con su etiqueta y componente; FI1 conserva su registro y FI2 aporta el suyo. Los componentes tienen una identidad estable durante el refresco.

Se conserva la caché `ultimaFicha` de FI1, limitada al identificador del expediente, junto con la lectura que conserva datos al recargar y el parámetro `refrescando` de FI2. Un error retira los datos y limpia la caché. Las acciones de FI2 permanecen bloqueadas mientras llega la nueva versión.

La revisión independiente del registro y la caché dio GO. Semgrep local: `Ran 3 rules on 3 files: 0 findings.` El recorrido FI1 pasó sus cinco comprobaciones: Menores, Intervención, guardia, Coordinación y estados. La batería completa dio `CI mínima completada.` y `F0.11-OK: 71.4 s`; incluye las pruebas de Go, frontend y PostgreSQL.

El recorrido FI2 completó `FI2-OK: recorrido real completado; sin errores JavaScript.` Incluye una comprobación del orden exacto de las nueve pestañas, las operaciones reales de Consentimiento y Evaluación y la fixture de respuestas por ítem. El cambio de idioma espera a que aparezca el título traducido antes de cambiar el tema, para comprobar los controles cuando la nueva interfaz esté lista.
