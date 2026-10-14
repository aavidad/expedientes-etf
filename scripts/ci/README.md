# Puerta mínima de integración

Desde la raíz del repositorio:

```sh
make ci
```

Requiere Bash, Make, Git, tar, Python 3.9 o posterior, la versión Go de `go.mod`,
Node 20.19.2 con npm, Docker con Compose, Chrome del sistema y Gitleaks 8.30.1 en `PATH`. GitHub Actions prepara las
mismas versiones de Go, Node y Gitleaks. El CLI de Gitleaks para el runner
Linux x64 se descarga de su release con SHA-256 fijado en
`instalar-gitleaks.sh`; `make ci` usa la herramienta ya instalada y no la descarga.
Las dependencias de la web se instalan
con `npm ci` desde el archivo de bloqueo, sin auditoría remota ni scripts de
instalación. Las dependencias Go quedan fijadas en `go.mod` y `go.sum`.

La puerta falla ante formato Go incorrecto, enlaces locales rotos, secretos,
errores de tipos, pruebas fallidas o fallos de compilación. Comprueba primero
la web para que los binarios Go incorporen su compilación actualizada.
`go test ./...` incluye la prueba de arquitectura.

El comprobador Markdown revisa los archivos versionados y los nuevos que Git
no ignora, incluidas las skills. Comprueba destinos relativos o desde la raíz,
anclas Markdown, imágenes y referencias. Ignora bloques de código y código
en línea. Los enlaces externos se cuentan y no se consultan: la comprobación
no hace peticiones de red. No valida rutas citadas como código ni prosa.

Gitleaks analiza todo el historial disponible y una copia de los ficheros
locales que Git no ignora, incluidos cambios sin commit. La copia temporal
se crea en `bin/`, ya ignorado, y se elimina al terminar. Los valores detectados
se ocultan en la salida. No se suben informes ni código a servicios de análisis.

`.gitleaksignore` recoge únicamente huellas de falsos positivos comprobados.
Cada entrada identifica un commit, fichero, regla y línea; el resto del
historial y de los ficheros locales se sigue comprobando.

El workflow se activa en push, PR y ejecución manual. El trabajo tiene un
presupuesto de 25 minutos para compilación en frío y recorridos completos,
conservando los límites internos de las pruebas. Las acciones están
fijadas por SHA de commit y el token sólo tiene `contents: read`; checkout no
conserva credenciales. La acción de Gitleaks se ejecuta en push y ejecución
manual. En PR, `make ci` hace el mismo análisis completo con el CLI, evitando
el permiso adicional de lectura de PR que requiere la acción. En todos los
eventos se analiza el historial y los ficheros locales. Sólo el paso de la
acción recibe el token. Se desactivan comentarios, resúmenes
y artefactos según las opciones de la
[acción oficial de Gitleaks](https://github.com/gitleaks/gitleaks-action/tree/e0c47f4f8be36e29cdc102c57e68cb5cbf0e8d1e).
Esta acción no exige licencia para repositorios de una cuenta personal; si se
traslada a una organización, su licencia debe resolverse antes de usarla allí.

La puerta cubre también migraciones, RLS, sesión, idiomas, temas y los recorridos
ya incorporados en `comprobar.sh`, según TRD §11 y PLAN §4.3.

`make ci` ejecuta también `migraciones/catalogo/ensayar.py` y
`migraciones/expediente/ensayar.py`. FX6 actualiza sus expectativas y cortes
históricos sin modificar los catálogos ni las migraciones. Ambos duran menos
de un minuto en la medición local; evidencia en [FX6](../../docs/evidencias/FX6.md).

Los ensayos de `migraciones/*/ensayar*.py` usan `ensayos.py` para las órdenes.
Al fallar, muestran código de salida (o timeout), stdout y stderr, limitados
a 4.096 caracteres por canal, conservando principio y final. Ocultan las
credenciales del entorno, asignaciones y ficheros `--env-file`, además de los
apartados DETAIL, CONTEXT y STATEMENT de PostgreSQL. Nunca imprimen argumentos
ni entrada SQL. Esta utilidad sólo se usa con contenedores desechables y datos
inventados; no es un registrador para producción.

Antes de migrar se espera una consulta por TCP con la contraseña de
`etf_migrador`, hasta 60 segundos. El servidor provisional de la imagen oficial
acepta consultas por socket durante initdb, con los roles ya creados, y después
se detiene. Ese criterio provocaba el fallo intermitente de FX5. La espera
sondea una lectura; una migración fallida termina el ensayo.

`ensayar-arranque-postgres.py`, ejecutado tras compilar `bin/migrar`, detiene
initdb después de crear los roles para reproducir la carrera. Comprueba el
fallo prematuro, que la nueva espera sigue bloqueada y que la instalación
anterior y su actualización pasan tras liberar el arranque. Evidencia y
repeticiones en [FX5](../../docs/evidencias/FX5.md).
