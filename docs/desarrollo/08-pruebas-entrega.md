# 8. Pruebas y entrega

Las pruebas de una capacidad deben mostrar el comportamiento que se entrega. Un test Go con repositorio en memoria comprueba reglas y coordinación, pero no acredita RLS, persistencia ni una pantalla utilizable. Selecciona esas pruebas según el cambio y termina con la puerta completa `make ci`.

## Preparación local

Las versiones del servidor están fijadas en [go.mod](../../go.mod) y [go.sum](../../go.sum). La interfaz usa [package.json](../../web/package.json) y [package-lock.json](../../web/package-lock.json). Conserva las sumas y no instales un navegador alternativo para los recorridos.

La CI necesita Git, Bash, Python, Go, Node con npm, Docker, Gitleaks y Chrome del sistema. Las versiones y requisitos completos están en [scripts/ci/README.md](../../scripts/ci/README.md). Usa la caché Go predeterminada del equipo; no sitúes `GOMODCACHE` en memoria compartida. En este entorno los temporales se crean en `/dev/shm/etf/tmp`.

## Pruebas por cambio

| Cambio | Comprobaciones necesarias |
| --- | --- |
| Regla o cálculo Go | Pruebas del paquete, límites y resultados esperados independientes |
| Caso de uso | Permisos, transacción, reversión, auditoría, versión e idempotencia |
| PostgreSQL | Instalación y actualización, restricciones, RLS, privilegios y concurrencia reales |
| Pantalla | Tipos, pruebas de comportamiento, navegador, teclado y estados de carga, vacío, error y conflicto |
| Idioma o tema | Paridad y existencia de claves; tokens y contraste; lectura de la versión activada |
| Plantilla | Marcadores, seguridad DOCX, reproducción, activación y revisión del Word |
| Permiso o endpoint | Anónimo, perfil equivocado, equipo ajeno, método prohibido, CSRF y ruta desconocida |
| Manual o exportador | Enlaces, archivos, PDF legible y conservación de los manuales anteriores |

Desde la raíz:

```sh
go test ./internal/dominio/estadistica/...
go test ./internal/aplicacion/estadistica/...
go vet ./...
go test ./...
make textos
make temas
python3 -B scripts/ci/enlaces.py
```

Para la interfaz, usa los comandos de `web/package.json` y el filtro apropiado de sus pruebas:

```sh
cd web
npm ci --ignore-scripts --no-audit --no-fund
npm test
npm run build
```

Los ensayos PostgreSQL están junto a las migraciones. Consulta su alcance antes de ejecutar uno: pueden levantar contenedores desechables y necesitan Docker. Entre las puertas actuales están:

```sh
python3 -B migraciones/catalogo/ensayar.py
python3 -B migraciones/expediente/ensayar.py
python3 -B migraciones/documento/ensayar.py
python3 -B migraciones/estadistica/ensayar.py
```

Una migración sensible requiere además ensayo en el clon local con datos inventados y revisión independiente. Las revisiones de seguridad se ejecutan en local; no envíes código ni datos a servicios de análisis externos.

## Recorridos y permisos F0.11

```sh
python3 -B scripts/recorridos/entrada_permisos.py
make recorridos
```

F0.11 arranca la demo de este worktree, obtiene el registro HTTP real y recorre entrada y alta con Chrome del sistema. Contrasta todos los métodos declarados con cada perfil y comprueba RLS en PostgreSQL. La matriz incluye anónimo, perfil incorrecto, equipo ajeno, métodos no admitidos, CSRF y rutas desconocidas. Conserva evidencia fuera de Git. El registro de rutas no sustituye las pruebas negativas independientes.

`make recorridos` ejecuta entrada y permisos, ficha integrada, salidas y SPAF, Memoria integrada y accesibilidad. F0.11 también forma parte de `make ci`; los demás recorridos conservan sus propios criterios y pueden detectar fallos pendientes fuera de esa puerta. No presentes un recorrido fallido u omitido como aprobado. Consulta [F0.11](../recorridos/F0.11.md), [F2.07](../recorridos/F2.07.md), [F3.06/F3.07](../recorridos/F3.06-F3.07.md), [F4.09](../recorridos/F4.09.md) y [F5.03](../accesibilidad/README.md).

## CI local y GitHub Actions

```sh
make ci
```

[scripts/ci/comprobar.sh](../../scripts/ci/comprobar.sh) comprueba catálogos, índice de producto, enlaces, guiones, secretos, TypeScript, pruebas web, textos, temas, `go vet`, pruebas Go y compilación. Después ejecuta los ensayos PostgreSQL, la incorporación de idiomas y temas a la demo y los recorridos de entrada, permisos y carga. La comprobación de arranque exige TCP autenticado y evita aceptar el servidor provisional de inicialización.

[.github/workflows/ci.yml](../../.github/workflows/ci.yml) prepara Go y Node con las versiones fijadas y ejecuta el mismo `make ci`. Las acciones están fijadas por commit, el permiso general es lectura del repositorio y el trabajo dispone de 25 minutos. GitHub privado y su CI están autorizados; los datos usados son inventados.

La puerta actual no contiene un análisis Semgrep general ni reconstruye toda imagen de producción. Son comprobaciones adicionales exigibles según el cambio, aunque el TRD las enumere entre las puertas de seguridad y despliegue. Ejecuta Semgrep local sobre lo modificado con las reglas versionadas pertinentes y sin subida a su nube. Revisa dependencias y la imagen cuando las cambies.

## Demo y vídeos

```sh
make demo
make demo-comprobar
make demo-recorrido
make demo-parar
```

La [demo](../../despliegue/demo/LEEME.md) tiene sus propios contenedores, redes, volúmenes y usuarios inventados. Si el puerto está ocupado al crearla, elige uno libre con `ETF_DEMO_PUERTO=18214 make demo`. El puerto se conserva. La carpeta privada `despliegue/demo/.local/` guarda las cuentas y la configuración; no la añadas a Git.

Parar conserva datos y versiones. `make demo-limpiar` los borra: úsalo sólo cuando quieras reiniciar esa demostración. Al arrancar o actualizar, la preparación para la aplicación, sustituye la base de idiomas, temas y plantillas de la distribución, conserva lo publicado en `.versiones` y lo cambiado a mano y valida los textos (FX8); no exige limpiar los datos. [Su ensayo](../../despliegue/demo/ensayar_incorporacion.py) forma parte de la CI: parte de una demo con textos anteriores a SEG1, sin manifiesto y con una versión de idioma activa antigua, y comprueba el Word de Consulta, la emisión Word con una plantilla nueva y las huellas de lo publicado. Los preparadores de recorridos renuevan únicamente sus recursos de distribución cuando el contrato lo permite.

Para vídeos, consulta [la guía y los guiones](../recorridos/LEEME.md):

```sh
make video RECORRIDO=11-ciclo-completo
make videos RECORRIDOS="09-memoria 10-carga-2026"
```

Las grabaciones conservan un caso nuevo por ejecución y sus emisiones. Los MP4 se guardan fuera de Git, en la carpeta de vídeos indicada por la herramienta. Revisa duración, fotogramas y capítulos. El reloj de formación se usa sólo mediante el overlay local del recorrido y con datos inventados; no cambia el reloj de identidad ni el código de producción.

La locución de textos redactados por el proyecto tiene autorización específica para el servicio de voz documentado. No se envían datos de expedientes ni contenido introducido por personas usuarias. Los audios se construyen antes y se sirven como archivos estáticos; la app no llama al servicio de voz durante el uso.

## Capturas y PDF

Con la demo arrancada:

```sh
node docs/desarrollo/capturar.mjs
TMPDIR="$PWD/bin/temporales" python3 -B docs/manuales/exportar.py --manual programadores
python3 -B scripts/ci/enlaces.py
```

El capturador exige que la dirección sea la demo local de este worktree. Lee sus cuentas inventadas, abre `/usr/bin/google-chrome` con Playwright y guarda Plantillas Word y Memoria a 1440 y 390 píxeles en `docs/desarrollo/capturas/`. Comprueba sesión, respuestas API, errores JavaScript y ausencia de desbordamiento horizontal. No publica contraseñas ni conserva sesiones de navegador.

Revisa cada imagen y su texto alternativo. Abre el PDF, comprueba diagramas, código, paginación y legibilidad. El exportador conserva por defecto la selección de los tres manuales de usuario; el modo `--manual programadores` sólo escribe este manual. [EVIDENCIA.md](EVIDENCIA.md) recoge la validación de esta edición.

## Windows y copia pública

```sh
make paquete-windows
```

El [paquete Windows](../../despliegue/windows/LEEME.md) incluye imágenes, demo y lanzadores para Windows 10/11 con WSL2 y Podman; instalar.cmd instala WSL2, Podman, su máquina y Compose si faltan. Genera ZIP versionado y SHA-256 en `bin/paquete-windows/`. Crea accesos que abren Edge o Chrome en ventana propia. Actualizar conserva datos y credenciales y ejecuta la preparación de roles antes de migrar; reiniciar exige confirmación y borra sólo la demo. El estado y límites del ensayo están en [WIN1](../sistemas/WIN1.md). Este paquete es de pruebas, con datos inventados.

La rama consultada no contiene un generador ni una guía de copia pública. No hay un comando de publicación que pueda documentarse como disponible. El ZIP de pruebas Windows tampoco equivale a una autorización para publicar el repositorio. Si se encarga una copia pública, habrá que fijar su alcance y revisar archivos, licencias, historial, capturas y ausencia de secretos o datos personales antes de prepararla. Este manual no publica contenido ni habilita acceso externo a la aplicación.

## Convenciones y entrega de cambios

Los nombres de dominio van en castellano. Mantén paquetes y responsabilidades pequeñas; usa puertos para todo sistema externo. Formatea Go con `gofmt`, utiliza `gopls` para definiciones y referencias y conserva los tipos TypeScript del contrato. Las claves de error identifican la causa sin redactar el mensaje dentro del código. Los comentarios explican decisiones o límites que no se deducen de la implementación.

Actualiza en el mismo cambio los documentos de producto y las guías que describan el comportamiento modificado. Regenera `docs/producto/INDICE.md` con su script después de editar esos documentos. Los títulos de commit van en castellano y describen la capacidad entregada, por ejemplo `Añade consulta histórica de menores al cálculo mensual`. Incluye las atribuciones exigidas por el encargo.

Antes de entregar, integra el `main` local actual en la rama asignada, resuelve conflictos conservando ambas capacidades y repite las comprobaciones afectadas. Termina con `make ci` completo. Deja commits limpios, sin hacer push ni integrar tu rama en `main`. La entrega indica rama, commits, archivos, pruebas reales, documentos actualizados y límites pendientes.
