# Paquete de pruebas para Windows

`VERSION=1.0.1 make paquete-windows` construye el ZIP y su SHA-256 en
`bin/paquete-windows/`. La publicación es manual. La rama `publico` mantiene
la distribución genérica y nunca se integra en `main`.

La [guía para probar en casa](../windows/PROBAR-EN-CASA.md) explica los requisitos
y el recorrido. El [LEEME del paquete](../../despliegue/windows/LEEME.md) viaja
con el instalador. Sólo se usan datos inventados.

## Entorno Windows comprobado

La evidencia previa de WIN1 acredita Windows 10 22H2 de 64 bits, compilación
19045.6466, con 8 GiB instalados; WSL 3.0.3, núcleo 6.18.54.1;
Podman Desktop 1.29.3, Podman 6.1.4 y Compose 5.1.4.
Se usa el proveedor WSL y el modo rootful. No se necesita Docker Desktop.
Windows 11 y Rancher Desktop no tienen ensayo acreditado.

El instalador comprueba 8 GiB de RAM instalada y 15 GiB libres en la unidad
del perfil. Las imágenes viajan dentro del ZIP. Sólo el proxy publica un
puerto del PC: 127.0.0.1:18083. Los secretos se generan en cada instalación,
con permisos restringidos a la cuenta actual. Actualizar conserva datos;
reiniciar-datos exige BORRAR; desinstalar exige DESINSTALAR y conserva archivos
ajenos. La copia privada permite reabrir desde el escritorio después de parar.

## Paquete público

La distribución se preparó en `publico`, integrando `main` con el commit
`03453fd` y ajustando la guía en `a0c2a8d`. Esta rama nunca se integra en `main`.
No se ha hecho push ni se ha creado una Release.

### Archivo preparado para publicar

Orden ejecutada: `make paquete-windows VERSION=v1.0.0`, con el temporal del sistema
configurado para esta sesión. Resultado en `bin/paquete-windows/`:

| Archivo | Medida |
| --- | --- |
| `expedientes-etf-windows-v1.0.0.zip` | 340809868 bytes (325,02 MiB) |
| Contenido extraído, 23 archivos | 657371814 bytes (626,92 MiB) |
| `expedientes-etf-windows-v1.0.0.zip.sha256` | Suma en formato SHA-256 estándar |
| `NOTAS-RELEASE.md` | Texto para la publicación manual |

```text
PAQUETE-OK: expedientes-etf-windows-v1.0.0.zip; SHA-256 577d3221867f64a38724f3f1c40ca56a65be155b10d1682915ab1ecd7f1ff387
```

Windows calculó la misma suma con `Get-FileHash`. También se comprobaron las
sumas internas del paquete antes de cargar las imágenes y la identidad del tar
copiado a la instalación privada. El ZIP incluye `LEEME.md`, los lanzadores,
las imágenes, `Anexo-3-pruebas.docx`, `Anexo-3-pruebas.odt` y
`Acta-aceptacion-pruebas.docx`. No incluye contraseñas de ninguna instalación.

Los tamaños medidos explican la recomendación de la guía: 17 GiB libres antes
de descargar para mantener los 15 GiB que exige el instalador tras extraer.
La copia privada añade otros 627 MiB dentro de esa reserva, además del almacén
de imágenes y los volúmenes de pruebas.

### Limpieza y revisión local

Se conservaron borradas todas las rutas internas indicadas para esta edición,
incluidos los documentos nuevos del cliente. Los imports nuevos usan el módulo
`github.com/aavidad/expedientes-etf` y están formateados con gofmt.
Los municipios y equipos de Granada se conservan como catálogo de ejemplo.
La entidad sigue configurándose con `entidad.nombre` en los textos de idioma.

Las plantillas nuevas se regeneraron desde las versiones públicas sin membretes.
Se retiraron las imágenes institucionales de los ODT; se regeneraron sus PDF y
las actas desde las plantillas genéricas. Los metadatos de autor y modificación
son «Expedientes ETF»; los correos de las muestras ODT/PDF usan
`servicios.invalid`. Las firmas PDF autofirmadas de prueba se volvieron a generar
y su integridad criptográfica se comprobó localmente. No se distribuyen claves privadas.

Resultados del patrón de búsqueda solicitado:

```text
Repositorio: 0 coincidencias; grep termina en 1 (sin resultados).
ZIP extraído: 0 coincidencias.
Capas de imágenes extraídas: 0 coincidencias.
Texto y metadatos: 88 documentos del repositorio + 3 del ZIP + 127 de las capas; 0 coincidencias.
```

Se revisaron además los autores XML, los metadatos PNG y el contenido incrustado
de las plantillas. Las capturas nuevas usan autor «Expedientes ETF» y muestran
sólo ventanas o recortes pertinentes de Windows, sin datos de su cuenta local.

Gitleaks se ejecutó por separado sobre el ZIP extraído, el tar extraído y sus
47 capas descomprimidas. Las tres ejecuciones terminaron en 0:

```text
3:53PM INF scanned ~59075 bytes (59.08 KB) in 9.23ms
3:53PM INF no leaks found
3:53PM INF scanned ~99052 bytes (99.05 KB) in 17.2ms
3:53PM INF no leaks found
3:53PM INF scanned ~187200579 bytes (187.20 MB) in 21.4s
3:53PM INF no leaks found

```

Revisión focal de seguridad: fronteras HTTP de carga, autorización por perfil y
equipo, límites ZIP/XML, antivirus, lectura local sin persistencia y scripts de
instalación. No se identificaron hallazgos confirmados en este alcance. Semgrep
con reglas locales, sin métricas ni consulta de versiones:

```text
Ran 7 rules on 31 files: 0 findings.
```

### Integración continua real

`make ci` completo terminó en 0. Incluye formato, catálogo, enlaces, secretos,
frontend, textos, Go, ensayos PostgreSQL, RLS y recorridos con Chrome del sistema.
En intentos anteriores se detectaron una ejecución simultánea de npm, una
colisión aleatoria de subred con otro ensayo y un timeout de navegación PWA.
Se corrigieron también el generador de fecha del corpus público y sus correos
esperados. La última ejecución completa volvió a pasar, sin omitir puertas.

```text
F0.11-RLS-OK: 7 actores; 53 tablas pobladas; 371 comprobaciones de lectura; 795 escrituras denegadas.
F0.11-VISTAS-OK: 9 vistas; 63 lecturas con RLS.
CA1_OK: altas aceptada/intervención/cerrada/no aceptada, reapertura, CSV/XLSX, incidencias, recuperación idempotente, revisión recuperable, permisos, teclado, 1440/1024/390/360, temas e idiomas.
RESP2_CHROME_OK: aceptación firmada 2022, aceptación Word 2023 + cierre PDF 2026, datos confirmados y adultos en ficha; permisos propio equipo, sin almacenamiento, 1440/1024/390/360 y errores sin datos.
RESP4 anexo3.odt: HTTP 200/201; antivirus limpio; revisión con dos menores; derivación confirmada y recuperada; formato original conservado; capturas 1440/390.
RESP4 anexo4-firmado.pdf: HTTP 200/201; antivirus limpio; revisión con dos menores; derivación confirmada y recuperada; formato original conservado; capturas 1440/390.
CI mínima completada.
```

### Recorrido real de Windows

Se trabajó sólo en la copia de pruebas. Antes de instalar se retiró la instalación
anterior con su `desinstalar.cmd`, confirmación escrita y salida 0. Los requisitos
se verificaron de nuevo: virtualización habilitada, 8 GiB instalados, WSL2 y
Podman rootful. Tras un apagado anterior quedó un socket temporal de Podman;
se retiró con el motor parado y el arranque recuperó su conexión local para Compose.
No se cambió ninguna otra máquina.

Se transfirió el ZIP definitivo y se verificó su SHA-256 dentro de Windows.
Se comprobó la extracción nativa desde «Extraer todo». El instalador se lanzó
por el Explorador. Para reproducir una descarga se añadió la marca de Internet
al lanzador de la copia de ensayo; apareció «Advertencia de seguridad de Abrir
archivo / Editor desconocido». Se confirmó Ejecutar. No apareció la pantalla
azul de SmartScreen, por lo que no se presenta una captura de esa pantalla.

La instalación abrió Bloc de notas con las siete cuentas inventadas. Se
comprobaron los accesos de escritorio e Inicio y se abrió el del escritorio:
Microsoft Edge ejecutaba `--app=http://localhost:18083`. El recorrido se automatizó
con Playwright conectado temporalmente al Edge real de Windows 10, versión
154.0.4258.48. El paquete entregado no abre un puerto de depuración.

```text
WINDOWS-PUBLICO-ENTRADA-OK: Coordinación en Edge de Windows
WINDOWS-PUBLICO-ANEXO-OK: DOCX incluido; antivirus limpio; 6 menores; derivación confirmada
WINDOWS-PUBLICO-ACEPTACION-OK: Aceptada 06-02-26
WINDOWS-PUBLICO-DESCARGA-OK: Acta de Aceptación.docx
WINDOWS-PUBLICO-ACTA-INTEGRA: ZIP interno correcto; 6 menores; firmante; número 06-02-26; 31430 bytes; SHA-256 fb93fde69cb6f7fda2c59782ef215224e485709b167a4f93b0b5a9660adff1ea
WINDOWS-PUBLICO-ODT-OK: ejemplo del ZIP; antivirus limpio; 2 menores; confirmación HTTP 201
WINDOWS-PUBLICO-ACTA-HISTORICA-OK: Word del ZIP; lectura, revisión y confirmación HTTP 200; número 02-71-22; expediente persistido en intervención
FECHA-HISTORICA 2022-05-17

```

La verificación del Acta leyó el archivo realmente descargado al disco de Windows:
validó todas las entradas ZIP, los seis menores, el número, el firmante y los
metadatos públicos. La carga histórica se comprobó también mediante consulta
posterior del expediente guardado.

[Instalación real](../capturas/windows-publico-instalacion.png),
[aceptación](../capturas/windows-publico-aceptacion.png),
[descarga del Acta](../capturas/windows-publico-acta.png),
[carga histórica completada](../capturas/windows-publico-carga-historica.png) y
[acceso creado](../capturas/windows-publico-acceso-directo.png).
La [guía doméstica](../windows/PROBAR-EN-CASA.md) incluye capturas del
[SHA-256](../windows/capturas/comprobar-sha256.png),
[aviso real](../windows/capturas/aviso-seguridad.png),
[extracción](../windows/capturas/extraer-todo.png) y
[acceso del escritorio](../windows/capturas/acceso-directo.png).

Las instalaciones de WSL y Podman Desktop se acreditan con la evidencia previa
y las versiones actuales comprobadas; no se repitieron esas instalaciones ni
se desinstalaron herramientas compartidas. Windows 11, otras versiones de Podman
y la pantalla azul de SmartScreen conservan su límite de ensayo.

### Parada, reapertura y reinicio de datos

`parar.cmd` terminó en 0 y `/salud` dejó de responder. Se activó de nuevo el
acceso del escritorio con la aplicación parada: volvió a preparar el arranque,
respondió `/salud` con 200 y abrió Edge en modo aplicación. Las contraseñas
conservaron su huella; los accesos de escritorio e Inicio siguieron presentes.
La lectura del expediente aceptado volvió a devolver HTTP 200, su número y su
estado aceptado.

```text
Aplicación detenida. Tus datos de pruebas se conservan.
WINDOWS-PUBLICO-REABRIR-SALUD: 200
WINDOWS-PUBLICO-REABRIR-EDGE-APP: True
WINDOWS-REABRIR-CUENTAS-CONSERVADAS: True
ACCESO-INICIO: True
WINDOWS-EXTRAER-TODO-OK: True
ARCHIVOS-EXTRAIDOS: 23
```

El asistente nativo «Extraer todo» produjo una segunda copia con 23 archivos;
todas sus sumas internas coincidieron. El reinicio se ejecutó mediante
`reiniciar-datos.cmd`, confirmación BORRAR y salida 0. Generó siete cuentas
con contraseñas renovadas. No se publican valores ni huellas de credenciales.

```text
REINICIO-CLAVES-RENOVADAS: True
REINICIO-NUMERO-CUENTAS: 7
WINDOWS-PUBLICO-REINICIAR-OK: entrada con contraseña nueva HTTP 200; derivación anterior HTTP 404
```

La entrada posterior usó la contraseña nueva sin copiarla a registros ni a
capturas. Se comprobó que la derivación de la prueba anterior ya devolvía 404.
[Estado tras el reinicio](../capturas/windows-publico-datos-reiniciados.png).

Al terminar se ejecutó de nuevo `parar.cmd`, se detuvo la máquina de Podman y
se apagó Windows desde el propio sistema. Libvirt confirmó la copia apagada;
la máquina base permaneció apagada. Se retiraron las tareas temporales del
ensayo y se detuvo también la demo Linux usada por CI.

```text
Machine "podman-machine-default" stopped successfully
WINDOWS-MOTOR-DETENIDO
Estado final de la copia de pruebas: shut off
DEMO-PARADA
```

Comprobaciones finales: nueve pruebas del paquete Windows, `OK`; enlaces
Markdown sin errores; análisis local de secretos del historial y los archivos,
`no leaks found`. La publicación queda a cargo de la dirección del proyecto.

## Paquete público 1.0.1

La rama `publico` integra `main` en el commit `47ec548` y repite la limpieza
pública sobre lo nuevo:

- Las plantillas Word nuevas (Acta de aceptación 1.1.2, No aceptación,
  Lista de espera, Acta de cierre y Tabla SPAF 1.0.2) se generan desde las
  versiones neutras anteriores; sus huellas se actualizan en
  `migraciones/documento/0016_respuestas_programa.sql`.
- Las actas ODT de prueba se generan de nuevo desde los DOCX inventados, sin
  logotipos ni metadatos.
- Los textos de ubicaciones usan `{entidad.nombre}`. Las capturas de
  ubicaciones y de sesión caducada se repiten con la demostración pública, y la
  CI vuelve a generar el resto con los mismos textos.
- Los manuales en PDF de profesionales, Coordinación, Administración, sistemas
  y programadores se exportan de nuevo con autor «Expedientes ETF».

La primera ejecución de `make ci` se detuvo porque quedaba el estado local de
una demostración anterior; la segunda, por un fallo intermitente del recorrido
del desglose de la Memoria, que después pasó tres veces seguidas sobre la misma
demostración. La tercera, desde una demostración limpia, terminó bien:

```text
CI mínima completada.
```

Orden: `VERSION=1.0.1 make paquete-windows`, con el temporal del sistema.

| Archivo | Medida |
| --- | --- |
| `expedientes-etf-windows-1.0.1.zip` | 341577925 bytes (325,75 MiB) |
| Contenido extraído, 23 archivos | 659239532 bytes (628,70 MiB) |
| `expedientes-etf-windows-1.0.1.zip.sha256` | Suma en formato SHA-256 estándar |

```text
PAQUETE-OK: expedientes-etf-windows-1.0.1.zip; SHA-256 9da39dc80f92562f8d0cb2155ed32bf5407c9e63c7e085b1ed4725dee25ff7d5
```

Comprobaciones del paquete: `sha256sum -c` correcto; análisis local de
secretos sobre el contenido extraído, el tar de imágenes y sus 48 capas
extraídas, `no leaks found`; la búsqueda de nombres reales, dominios privados,
rutas personales y la entidad original no encuentra nada en los archivos, las
capas ni los textos de idioma incluidos en las imágenes, ni en el texto y los
metadatos de los DOCX, ODT, PDF y PNG del repositorio. Pruebas del paquete
Windows: 20, `OK`.

La versión 1.0.1 no se ha probado todavía en un Windows real. No se ha hecho
push ni se ha creado una Release.

## Paquete público 1.0.2

La rama `publico` integra `main` en el commit `e12ce85`, con el curso, el lector
de voz, la pantalla Ayuda, el arranque con Podman y la espera al antivirus. La
limpieza pública se repite sobre lo nuevo:

- Las rutas personales del curso pasan a la carpeta del usuario
  (`~/Videos/Expedientes-ETF` y `~/.config/expedientes-etf/`). El curso
  mantiene su autoría y la licencia CC BY-SA 4.0.
- Los manuales web de la pantalla Ayuda se generan de nuevo desde los
  manuales públicos, con autor «Expedientes ETF» y «cuenta de la entidad».
- La captura de entrada del curso y una captura de Windows con una ruta de
  usuario se retiran; las de Ayuda y voz se repiten con la demostración pública.
- Los audios se generan con `make voz` (edge-tts 7.3.1) desde los textos de la
  rama pública: 135 secciones del curso, 48 grabaciones de ayuda y 60 de
  manuales. Se eliminan de `bin/voz` los MP3 que no cita ningún manifiesto.

`make ci` falló primero por un enlace a la captura retirada y después por un
choque de subred de Docker con otra prueba; la tercera ejecución terminó bien:

```text
CI mínima completada.
```

Orden: `VERSION=1.0.2 make paquete-windows`.

| Archivo | Medida |
| --- | --- |
| `expedientes-etf-windows-1.0.2.zip` | 446957440 bytes (426,25 MiB) |
| Contenido extraído, 23 archivos | 777320496 bytes (741,31 MiB) |
| `expedientes-etf-windows-1.0.2.zip.sha256` | Suma en formato SHA-256 estándar |

```text
PAQUETE-OK: expedientes-etf-windows-1.0.2.zip; SHA-256 f4b29737477ba8b7b322ef3afc9f702bf38313e84852616b5e69c1756da5d641
```

Comprobaciones del paquete: `sha256sum -c` correcto; análisis local de
secretos sobre el contenido extraído, el tar de imágenes y sus 49 capas
extraídas, `no leaks found`; la búsqueda de nombres reales, dominios privados,
rutas personales y la entidad original no encuentra nada en los archivos, las
capas, los textos de idioma ni los 243 audios incluidos, ni en el texto y los
metadatos de los DOCX, ODT y PDF del paquete y del repositorio.

La versión 1.0.2 no se ha probado todavía en un Windows real. No se ha hecho
push ni se ha creado una Release.

## Requisitos instalados por instalar.cmd (15/10/2026)

Con el motor Podman, `instalar.cmd` instala lo que falte antes de comprobar el
paquete. Cada paso se puede repetir y no rehace lo que ya está:

- WSL2: si `wsl --version` o `wsl --status` fallan, ejecuta
  `%SystemRoot%\System32\wsl.exe --install --no-distribution` con permiso UAC.
  Si la persona lo rechaza, se detiene con un mensaje claro. Si Windows tiene
  que reiniciarse, guarda el arranque en `wsl-pendiente.txt` (carpeta privada) y
  una entrada `ExpedientesETF` en `HKCU\...\RunOnce` que relanza `instalar.ps1`
  con la misma carpeta y puerto. Si tras reiniciar WSL2 sigue sin funcionar,
  pide activar la virtualización en vez de pedir otro reinicio.
- Podman 6.1.3: MSI oficial incluido, comprobado con `SUMAS.json` y ejecutado
  con `msiexec /package … /quiet /norestart MSIINSTALLPERUSER=1
  MACHINE_PROVIDER=wsl`, sin permiso de administrador. Se respeta un Podman ya
  instalado por usuario, por equipo o con el instalador antiguo de Red Hat.
- Máquina: si no hay ninguna WSL, `podman machine init --rootful --cpus 2
  --memory 4096` con `CONTAINERS_MACHINE_PROVIDER=wsl` y `podman machine start`.
  Podman descarga la imagen de su máquina de `quay.io/podman/machine-os` la
  primera vez; el disco de WSL crece según el uso y lo limita la comprobación
  de 15 GB libres.
- Compose 5.5.1: sólo si `podman compose version` no encuentra proveedor; el
  binario incluido se comprueba, se copia a `herramientas\` de la carpeta
  privada y se usa con `PODMAN_COMPOSE_PROVIDER` en cada lanzador.

Pruebas: `despliegue/windows/test_requisitos.py` simula cada ausencia en
PowerShell 7 sobre Linux (WSL ausente, permiso rechazado, reinicio pendiente y
repetido, Podman ausente, MSI alterado o fallido, máquina ausente, Compose
ausente o alterado) y `test_paquete.py` comprueba las versiones y sumas fijadas
y la descarga verificada. Falta el ensayo en un Windows limpio.

## Paquete público 1.0.3

La rama `publico` integra `main` en el commit `06bacda`, con el instalador
que pone lo que falta (WSL2, Podman 6.1.3, su máquina y Compose 5.5.1), el
arranque que espera a Podman tras reiniciar, la parada que apaga su máquina, la
desinstalación de lo instalado y las plantillas Word inglesas. La limpieza
pública se repite sobre lo nuevo:

- Las doce plantillas inglesas nuevas se generan desde las versiones neutras
  anteriores con los mapas `idiomas/en/plantillas/traduccion-*.json`, sin
  logotipos ni membretes y con autor «Expedientes ETF». Sus huellas se
  actualizan en `migraciones/documento/0017_plantillas_en.sql`. El detector de
  texto castellano admite el nombre del producto, que figura en el pie neutro.
- El LEEME del paquete y la guía para probar en casa describen la instalación
  automática; la preparación manual queda como plan B en el LEEME.
- La entrada dice «Usa tu usuario y contraseña de la entidad»: `entidad.nombre`
  vale «la entidad» y «your organisation», con los textos ingleses ajustados.
- Los audios se regeneran con `make voz` (edge-tts 7.3.1) desde los textos
  públicos; se eliminan de `bin/voz` los 13 MP3 que no cita ningún manifiesto.
  Quedan 243 grabaciones.

La primera ejecución de `make ci` falló porque el detector de castellano de las
plantillas inglesas encontraba «expediente» en el pie «Expedientes ETF»; tras
admitir el nombre del producto, terminó bien:

```text
CI mínima completada.
```

Orden: `VERSION=1.0.3 make paquete-windows`.

| Archivo | Medida |
| --- | --- |
| `expedientes-etf-windows-1.0.3.zip` | 488129947 bytes (465,52 MiB) |
| Contenido extraído, 25 archivos | 839864892 bytes (800,96 MiB) |
| `expedientes-etf-windows-1.0.3.zip.sha256` | Suma en formato SHA-256 estándar |

```text
PAQUETE-OK: expedientes-etf-windows-1.0.3.zip; SHA-256 7404c5d4c5d06b890556a8d1f44f721adb2e8683b15736a49748325b463d43c5
```

Comprobaciones del paquete: `sha256sum -c` correcto; análisis local de
secretos sobre el contenido extraído, el tar de imágenes y sus 49 capas
extraídas, `no leaks found`; la búsqueda de nombres reales, dominios privados,
rutas personales y la entidad original no encuentra nada en los archivos, las
capas (también en binario), los textos de idioma ni los audios incluidos, ni en
el texto y los metadatos de los DOCX, ODT y PDF del paquete y del repositorio.

La versión 1.0.3 pública no se ha probado todavía en un Windows real. No se ha
hecho push ni se ha creado una Release.

## Paquete público 1.0.4

La rama `publico` integra `main` en el commit `92a2f436`, con el asistente de
instalación `.exe` hecho con NSIS y el lanzador `Expedientes ETF.exe` al que
apuntan los accesos del escritorio y de Inicio. La limpieza pública se repite
sobre lo nuevo:

- La estimación de necesidades que trae `main` en `docs/envios` no entra:
  esa carpeta sigue borrada en la copia pública.
- El LEEME del paquete habla del certificado de firma «de la entidad» y de
  comprobar la procedencia en la página de descargas.
- El asistente muestra la licencia EUPL del archivo `LICENSE`, se registra
  como «Expedientes ETF (pruebas)» en la lista de aplicaciones y tiene como
  editor «Alberto Avidad · Expedientes ETF».
- El LEEME del paquete, la guía para probar en casa, el README y las notas de
  la Release presentan el `.exe` como descarga principal y el ZIP como
  alternativa, con el aviso de SmartScreen.

`make ci` terminó bien:

```text
CI mínima completada.
```

Orden: `VERSION=1.0.4 make paquete-windows`, que reutiliza las grabaciones de
`bin/voz` de la 1.0.3.

| Archivo | Medida |
| --- | --- |
| `Expedientes-ETF-1.0.4-instalador.exe` | 387483777 bytes (369,53 MiB) |
| `expedientes-etf-windows-1.0.4.zip` | 489504105 bytes (466,83 MiB) |
| Contenido extraído del ZIP, 26 archivos | 841474261 bytes (802,49 MiB) |

```text
PAQUETE-OK: expedientes-etf-windows-1.0.4.zip; SHA-256 8a2ce1bf90d454b4cb33e8919b49ce16d9dd8b3cbc433f402ef274f4adba5ff8
INSTALADOR-OK: Expedientes-ETF-1.0.4-instalador.exe; 387483777 bytes; SHA-256 e4e307248b9a581827ac3a55af262276c4b49238c807ff7c415fb6952d06b04b
```

Comprobaciones: `sha256sum -c` correcto para los dos archivos. El `.exe` se
extrae con `7z x` y contiene los mismos archivos que el ZIP. El análisis local
de secretos no encuentra nada en el contenido del ZIP, en el del `.exe`, en el
tar de imágenes ni en sus 49 capas extraídas. La búsqueda de nombres reales,
dominios privados, rutas personales y la entidad original no encuentra nada en
el repositorio, en esos contenidos (también en binario) ni en el texto y los
metadatos de los 382 DOCX, ODT y PDF del repositorio y del paquete. Los
ejemplos de Anexo 3 sólo usan correos de dominios `.invalid`.

La versión 1.0.4 pública no se ha probado todavía en un Windows real. No se ha
hecho push ni se ha creado una Release.
