# FI1: Menores e Intervención

La ficha incorpora las dos pestañas mediante el registro de FI1. Marta Cabrera,
del ETF 02, edita; Inés Moreno consulta. Los datos y las capturas son inventados.
La siembra completa los nombres y las opciones iniciales por las API del servidor.
El resumen de avisos se despliega arriba de las pestañas y nombra cada campo.
Cada menor y cada lista de Intervención se pueden desplegar por separado; los
enlaces a campos abren el bloque correspondiente.

## Recorrido reproducible

Arranca `make demo` y ejecuta:

```sh
node scripts/recorridos/fi1.mjs http://127.0.0.1:18341
```

Ajusta el puerto al de tu demostración. El recorrido conserva sus cambios en los
expedientes inventados y puede repetirse. Al terminar, ejecuta `make demo-parar`.

Con Chrome del sistema y Playwright se han comprobado:

- Guardado real de los datos del menor, incremento de versión y vigencia al corte.
- Conflicto real por un guardado concurrente: el formulario se conserva, los datos
  actuales se muestran aparte y se confirma qué formulario se volverá a guardar.
- Fase, factores y tipologías; una excepción por menor queda conservada y no se
  aplica a ese menor en la consulta histórica.
- Límite de 15 menores y guardia al salir con cambios pendientes.
- Aviso de nacimiento que abre Menores, despliega la persona y enfoca su campo;
  la edición posterior de otro campo conserva el foco.
- Coordinación sin botón de guardado y rechazo HTTP 403 de su escritura.
- Navegación de pestañas con flechas y ausencia de nombres en el título del navegador.
- Reflujo a 320, 360, 390, 1024 y 1440 px; tema de alto contraste e inglés sin claves
  técnicas visibles en las opciones.
- Carga anunciada al retener una lectura; error recuperable y estado vacío mediante respuestas simuladas exclusivamente
  en las lecturas. Las escrituras, el conflicto y las consultas al corte son reales.

## Capturas

| Pestaña | Escritorio, 1440 px | Móvil, 390 px |
| --- | --- | --- |
| Menores | [Ver captura](fi1-menores-escritorio.png) | [Ver captura](fi1-menores-movil.png) |
| Intervención | [Ver captura](fi1-intervencion-escritorio.png) | [Ver captura](fi1-intervencion-movil.png) |

[Menores con alto contraste](fi1-menores-alto-contraste.png).

## Alcance y límites

La edición usa los contratos existentes de F2.01b y F2.01c. No se añaden
migraciones ni se cambian permisos. Las nuevas rutas SPA ofrecen accesos directos
a estas pestañas; los datos siguen en las API de Expedientes. Los enganches comunes
son el registro de módulos, el de pantallas y la ficha de Equipo. Esta última
conserva la ficha durante su recarga para mantener la confirmación de guardado,
y la oculta si la nueva lectura falla.
El recorrido anterior de Equipo espera ahora siete pestañas. La matriz independiente
de permisos de F0.11 incluye expresamente los dos accesos directos de FI1.

Se ha revisado la usabilidad de forma independiente, con la skill de interfaz del
proyecto, y la seguridad de fuentes con `revisar-cambios-etf` y `security-audit`.
Semgrep local, con las reglas de Equipo, no encuentra incidencias. No se han
probado NVDA ni un lector de pantalla equivalente en este entorno; teclado,
etiquetas, regiones de estado y estructura de formularios sí están comprobados.
Los avisos sobre campos de otros módulos permanecen informativos mientras su
pantalla de edición no esté disponible.

## Comprobaciones finales

`make ci` completo termina con `CI mínima completada.` Incluye `go vet ./...`,
`go test ./...`, 19 pruebas web aprobadas, textos de ambos idiomas, ensayos de
PostgreSQL y el recorrido de permisos: 1344 combinaciones HTTP, 350 lecturas con
seguridad por filas y 750 escrituras denegadas. El recorrido propio de FI1 termina
con sus cinco bloques `FI1-…-OK`. Semgrep local aplica tres reglas a los 15 ficheros
seleccionados y devuelve cero hallazgos.

La segunda revisión independiente de las cinco capturas finales da GO visual.
La rama integra `main` con Salidas de Coordinación y el cierre de Memoria,
conservando todos los módulos y regenerando el índice de producto.
