# Comprobación de idiomas

`make textos` analiza Go y TypeScript, comprueba que las claves usadas existen
 en castellano y compara las claves de `textos/` y `catalogos/` entre idiomas.
Necesita las dependencias fijadas de `web/package-lock.json` instaladas.
`make ci` ejecuta esta comprobación después de `npm ci`.

Cada `idiomas/<código>/idioma.json` contiene `codigo`, `nombre` (autodenominación)
y `version` semántica. El directorio incluye `textos/`, `catalogos/` y
`plantillas/`. Los textos son mapas JSON de clave a cadena. Las plantillas sólo
tienen la estructura de directorios en F0.06.

## Cadenas técnicas en Go

El comentario exacto `//etf:tecnico` permite declarar una cadena técnica que
el análisis confunde con una respuesta visible. Debe usarse sólo para formatos
destinados a máquinas, como un guion SQL/psql o la estructura XML de Word.
No se excluyen paquetes por su nombre.

La marca al final de una línea alcanza los literales y escrituras de esa línea.
En cadenas multilínea puede colocarse en la línea de apertura o de cierre.
Si el literal se asigna a una variable local, la marca en su declaración también
se respeta cuando esa variable se escribe después.

Cuando toda una función genera contenido técnico, la marca puede figurar en
su comentario de documentación (sin línea vacía antes de la declaración) o en
la línea de declaración, antes de la primera instrucción. Su alcance termina
con esa función. Una marca en el
paquete, un comentario de bloque o el contenido de una cadena no concede
excepciones. La marca no desactiva la validación de errores ni la comprobación
de existencia de las claves utilizadas.

```go
salida.WriteString(` xml:space="preserve"`) //etf:tecnico

// Genera el guion que se envía por stdin a psql.
//etf:tecnico
func guion() string {
    return "SET ROLE etf_propietario;"
}
```

Los catálogos también son mapas planos de clave a cadena. Las ubicaciones usan
`equipo.<código>`, `centro.<CÓDIGO>` y `municipio.<INE>`, coincidentes con
`clave_texto` en `datos/catalogos/`. Los topónimos mantienen el mismo valor en
castellano e inglés.
Los JSON de ubicaciones usan cuatro espacios de sangrado para evitar que
Gitleaks confunda el topónimo «Capileira» y la siguiente clave con un secreto.

El servidor relee los paquetes en cada consulta. Una sustitución completa del
directorio se ve sin reiniciar. Si una traducción o clave falta, consulta el
castellano. No guarda preferencias: la selección dura mientras está abierta la
pantalla. La publicación administrativa de versiones, su referencia activa,
la auditoría y la persistencia de preferencias dependen de fases posteriores.
