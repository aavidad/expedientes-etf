# Plantilla de guion de recorrido

Copia esta plantilla en `guiones/<nn>-<nombre>.md`. La introducción explica de qué trata el recorrido, quién lo hace y para qué sirve, sin enumerar los pasos. Escribe entre 28 y 48 palabras: el montaje comprueba una duración total de 12–18 segundos. El título permanece 3,5 segundos; después se ven tres capturas reales del recorrido con movimiento suave y transiciones. Desarrolla las siglas en todas las locuciones.

Cada campo ocupa una línea. El rótulo resume el paso en 4–8 palabras, con una frase distinta de la voz. Cada oración de la locución corresponde a una acción o resultado visible: la automatización recibe una acción por oración y la ejecuta mientras esa frase suena. El ratón se ve y el control se resalta al pulsarlo. El cierre dura 4–6 segundos sobre la última pantalla real difuminada.

La preparación identifica las cuentas y los datos inventados y explica cómo repetir sin modificar datos anteriores. Un guion nuevo necesita sus acciones dentro de `scripts/recorridos/`; copiar el texto no implementa los pasos.

```markdown
# Título del recorrido

## Introducción

Título: Título de la cartela inicial.
Cartela: Resumen que se leerá en pantalla.
Locución: Describe quién realiza el recorrido, de qué trata y para qué sirve en su jornada. Usa palabras llanas y evita enumerar los pasos. Ajusta el texto a una duración total de doce a dieciocho segundos.

## Preparación

Explica qué datos de demostración se usan y cómo quedan al terminar.

BLOQUE_JSON_DE_CONFIGURACION

## Paso identificador

Rótulo: Resumen breve del paso visible.
Locución: Describe la acción que se realiza ahora. Después explica el resultado que aparece.

## Cierre

Título: Título breve de cierre.
Cartela: Lo que se ha visto.
Locución: Resumen hablado de lo que se ha visto.
```

Sustituye `BLOQUE_JSON_DE_CONFIGURACION` por un bloque cercado con lenguaje `json`. Consulta la configuración del [guion de entrada](guiones/01-entrada.md); sus nombres de variables pertenecen a ese recorrido.
