# Accesibilidad: /coordinacion/derivaciones/{id}

Pasada local con Chrome del sistema y axe-core 4.10.3. Datos inventados de la demostración.

## Alcance comprobado

Perfiles: coordinacion. 8 comprobaciones; escritorio de 1440 px y móvil de 390 px, español e inglés, temas expedientes-etf y alto-contraste.

Cada combinación aparece debajo. Las pestañas y ayudas se auditan cuando están disponibles. Un resultado automático sin infracciones no acredita todos los criterios WCAG.

| Perfil | Idioma | Tema | Ancho | Estado | Infracciones | Reflujo de página |
| --- | --- | --- | --- | --- | --- | --- |
| coordinacion | es | expedientes-etf | 1440 | pantalla | 1 | Sin desbordamiento |
| coordinacion | es | expedientes-etf | 390 | pantalla | 0 | Sin desbordamiento |
| coordinacion | es | alto-contraste | 1440 | pantalla | 1 | Sin desbordamiento |
| coordinacion | es | alto-contraste | 390 | pantalla | 0 | Sin desbordamiento |
| coordinacion | en | expedientes-etf | 1440 | pantalla | 1 | Sin desbordamiento |
| coordinacion | en | expedientes-etf | 390 | pantalla | 0 | Sin desbordamiento |
| coordinacion | en | alto-contraste | 1440 | pantalla | 1 | Sin desbordamiento |
| coordinacion | en | alto-contraste | 390 | pantalla | 0 | Sin desbordamiento |

## Hallazgos automáticos

### landmark-unique

Criterio: Recomendación de estructura (axe best-practice). Gravedad axe: moderate.

Observado en 4 comprobaciones. Landmarks should have a unique role or role/label/title (i.e. accessible name) combination.

Para reproducir, entra con el perfil indicado, abre la ruta y selecciona la combinación de la tabla con infracciones. Ejecuta la auditoría con el comando del índice.

- Selector: `[".lateral-app"]`. Fix any of the following:
  The landmark must have a unique aria-label, aria-labelledby, or title to make landmarks distinguishable

Propuesta: revisar la semántica del componente en su módulo. No se modifican pantallas ajenas en este encargo.


## Revisión manual pendiente

axe dejó resultados incompletos para revisión humana: color-contrast (8).

Quedan fuera de esta evidencia el lector de pantalla NVDA, la inspección de los Word, los estados no abiertos y las combinaciones de datos que no contiene la demostración. Consulta el índice para la segunda revisión de usabilidad.

## Capturas

- [coordinacion, 1440 px](../capturas/accesibilidad-coordinacion-coordinacion-derivaciones-id-1440.png).
- [coordinacion, 390 px](../capturas/accesibilidad-coordinacion-coordinacion-derivaciones-id-390.png).
