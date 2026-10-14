# Accesibilidad: /equipo/intervencion

Pasada local con Chrome del sistema y axe-core 4.10.3. Datos inventados de la demostración.

## Alcance comprobado

Perfiles: coordinacion, profesional_etf. 16 comprobaciones; escritorio de 1440 px y móvil de 390 px, español e inglés, temas expedientes-etf y alto-contraste.

Cada combinación aparece debajo. Las pestañas y ayudas se auditan cuando están disponibles. Un resultado automático sin infracciones no acredita todos los criterios WCAG.

| Perfil | Idioma | Tema | Ancho | Estado | Infracciones | Reflujo de página |
| --- | --- | --- | --- | --- | --- | --- |
| coordinacion | es | expedientes-etf | 1440 | pantalla | 1 | Sin desbordamiento |
| coordinacion | es | expedientes-etf | 390 | pantalla | 1 | Sin desbordamiento |
| coordinacion | es | alto-contraste | 1440 | pantalla | 1 | Sin desbordamiento |
| coordinacion | es | alto-contraste | 390 | pantalla | 1 | Sin desbordamiento |
| coordinacion | en | expedientes-etf | 1440 | pantalla | 1 | Sin desbordamiento |
| coordinacion | en | expedientes-etf | 390 | pantalla | 1 | Sin desbordamiento |
| coordinacion | en | alto-contraste | 1440 | pantalla | 1 | Sin desbordamiento |
| coordinacion | en | alto-contraste | 390 | pantalla | 1 | Sin desbordamiento |
| profesional_etf | es | expedientes-etf | 1440 | pantalla | 1 | Sin desbordamiento |
| profesional_etf | es | expedientes-etf | 390 | pantalla | 1 | Sin desbordamiento |
| profesional_etf | es | alto-contraste | 1440 | pantalla | 1 | Sin desbordamiento |
| profesional_etf | es | alto-contraste | 390 | pantalla | 1 | Sin desbordamiento |
| profesional_etf | en | expedientes-etf | 1440 | pantalla | 1 | Sin desbordamiento |
| profesional_etf | en | expedientes-etf | 390 | pantalla | 1 | Sin desbordamiento |
| profesional_etf | en | alto-contraste | 1440 | pantalla | 1 | Sin desbordamiento |
| profesional_etf | en | alto-contraste | 390 | pantalla | 1 | Sin desbordamiento |

## Hallazgos automáticos

### landmark-complementary-is-top-level

Criterio: Recomendación de estructura (axe best-practice). Gravedad axe: moderate.

Observado en 16 comprobaciones. Aside should not be contained in another landmark.

Para reproducir, entra con el perfil indicado, abre la ruta y selecciona la combinación de la tabla con infracciones. Ejecuta la auditoría con el comando del índice.

- Selector: `[".equipo-aviso"]`. Fix any of the following:
  The complementary landmark is contained in another landmark.
- Selector: `[".fi1-resumen"]`. Fix any of the following:
  The complementary landmark is contained in another landmark.

Propuesta: revisar la semántica del componente en su módulo. No se modifican pantallas ajenas en este encargo.


## Revisión manual de la ficha

Incidencia moderada, criterios 2.4.7 y 1.4.10: a 390 px, desde Familia, pulsa flecha derecha tres veces. Consentimiento conserva el foco pero queda parcialmente fuera de la fila visible. Salidas presenta el mismo comportamiento. Propuesta: desplazar la pestaña seleccionada con scrollIntoView al cambiar con flechas. El cambio corresponde a web/src/equipo/equipo.tsx y queda pendiente por el límite de otros módulos.


## Revisión manual pendiente

axe dejó resultados incompletos para revisión humana: ninguno en esta pasada.

Quedan fuera de esta evidencia el lector de pantalla NVDA, la inspección de los Word, los estados no abiertos y las combinaciones de datos que no contiene la demostración. Consulta el índice para la segunda revisión de usabilidad.

## Capturas

- [coordinacion, 1440 px](../capturas/accesibilidad-coordinacion-equipo-intervencion-1440.png).
- [coordinacion, 390 px](../capturas/accesibilidad-coordinacion-equipo-intervencion-390.png).
- [profesional_etf, 1440 px](../capturas/accesibilidad-profesional_etf-equipo-intervencion-1440.png).
- [profesional_etf, 390 px](../capturas/accesibilidad-profesional_etf-equipo-intervencion-390.png).
