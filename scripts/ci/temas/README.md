# Comprobación de temas

`make temas` busca colores fijos en Go y `web/src/` y ejecuta las pruebas de
validación de paquetes. `make ci` también la ejecuta. Sólo se excluyen los
ficheros de prueba y las herramientas de CI.

`tema.json` mantiene los campos del predeterminado: identificador, nombre por
idioma, versión, autor, licencia, versión mínima de la app y tipo. Puede añadir
`fuentes`, un mapa opcional con `texto` y `titulo` apuntando a ficheros
`fuentes/<nombre>.woff2`. El servidor genera las reglas de fuente y las variables
CSS; nunca acepta CSS del paquete.

Los recursos admitidos son PNG y WOFF2, hasta 8 MiB por fichero y 20 MiB por
paquete, con un máximo de 1.000 entradas. PNG se decodifica con un máximo de
4.096 píxeles por lado. WOFF2 requiere cabecera y directorio de tablas coherentes, longitudes y
transformaciones válidas y un límite de expansión de 100 MiB. No se admiten enlaces simbólicos, SVG, código ni carpetas
ajenas a `fuentes/` e `iconos/`.

Los tokens ausentes de un tema se heredan del predeterminado antes de validar
el contraste. Se exige 4,5:1 para texto y 3:1 para controles y foco; alto
contraste exige 7:1 para texto y 4,5:1 para controles. El tema inválido no aparece
en la lista y su petición de estilos usa el predeterminado. Una copia de los
datos del predeterminado está incluida en el ejecutable para recuperar su estilo
si el volumen falla. Public Sans y Source Serif 4 se sirven en WOFF2 variable desde ambos paquetes,
con su licencia OFL en el manifiesto. `temas/fuentes.lock.json` fija versiones,
sumas del paquete npm y SHA-256 de los archivos; CI comprueba estas últimas.

La selección de esta pantalla no publica un tema para la organización ni guarda
preferencias. La instalación y activación administrativas pertenecen a F0.14c.
Las reglas de `seguridad.yml` son un análisis local acotado; no representan una
auditoría completa.
