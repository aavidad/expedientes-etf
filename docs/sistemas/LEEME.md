# Manual de sistemas de Expedientes ETF

Autor: Expedientes ETF. Revisión: 14 de octubre de 2026.

[PDF para imprimir](sistemas.pdf). Los anexos de configuración, operación y
resolución de problemas forman parte del PDF.

- [Requisitos e instalación](#1-requisitos-y-responsabilidades).
- [Arquitectura y contenedores](#11-arquitectura-y-contenedores).
- [Configuración de producción](CONFIGURACION.md).
- [Arranque y actualización](#4-arranque-parada-y-actualización).
- [Copias y restauración](#5-copias-y-restauración).
- [Cifrado, auditoría, supervisión y conservación](OPERACION.md).
- [Mensajes reales y resolución de problemas](DIAGNOSTICO.md).
- [Riesgos pendientes de Informática](#12-riesgos-abiertos-que-dependen-de-informática).
- [Pruebas de esta revisión](MANSIS.md).

Este manual está dirigido a Informática de la entidad titular del servicio. Los comandos
se ejecutan desde la raíz del repositorio, salvo que se indique otra cosa.
La [evidencia de F5.05](EVIDENCIA.md) recoge el ensayo real de restauración.
La [evidencia de FX2](FX2.md) recoge la actualización de roles en un volumen
anterior, la instalación limpia y las comprobaciones de credenciales y permisos.
Las reglas de despliegue y conservación están en [TRD §§10–14](../producto/TRD.md#10-seguridad).

Para pruebas en un PC Windows 10/11, utiliza [la guía del paquete](../../despliegue/windows/LEEME.md).
`make paquete-windows` prepara imágenes, ZIP versionado y suma SHA-256. El motor
principal es Podman Desktop con WSL2; Rancher Desktop con Moby es la alternativa.
Crea accesos de escritorio e Inicio con una copia privada del paquete; parar
conserva los datos y permite reabrir, desinstalar retira sus accesos y volúmenes.
Sólo admite datos inventados. La operación y el alcance comprobado se recogen en
[WIN1](WIN1.md); este paquete no sustituye el despliegue de producción.

La [demo local](../../despliegue/demo/LEEME.md) incorpora las nuevas plantillas,
idiomas y temas al ejecutar `make demo`, sin borrar sus volúmenes. Comparte con
Windows el incorporador de FX8: sustituye la base de la distribución y conserva
lo publicado y lo cambiado a mano (§4). `make ci` lo comprueba en una demo
desechable con textos y versión de idioma antiguos (FX7 y FX8).

## 1. Requisitos y responsabilidades

La aplicación es interna. Sólo deben acceder la red corporativa, la red provincial
de ayuntamientos y la VPN. No se publica en Internet. El responsable del producto autoriza las versiones
y la entrada de datos reales; Informática mantiene máquinas, red, certificados,
firmas antivirus, copias y supervisión. Informática no necesita perfil funcional.

| Elemento | Requisito |
| --- | --- |
| Máquina | Linux con Docker Engine y Compose 2.24.4 o posterior; comprobar `docker version` y `docker compose version` |
| Preproducción | Recursos según la carga prevista; sólo datos inventados |
| Producción | Dimensionar CPU, memoria y disco cifrado según el volumen; vigilar crecimiento de base y copias |
| Base de datos | PostgreSQL 18; Compose fija 18.4 y la huella de imagen |
| Antivirus | ClamAV local; Compose fija 1.4.3 y su huella; firmas actualizadas mediante suministro local aprobado |
| Directorio | Active Directory mediante LDAPS, TCP 636, certificado válido y autoridad de confianza de la entidad |
| Web | Nombre interno y certificado HTTPS; `etf.example.org` es un dominio de ejemplo; configurar el propio |
| Construcción | Git y acceso autorizado al repositorio público; imágenes fijadas en Dockerfile y Compose; npm usa package-lock |
| Ensayo local | Go 1.26.9, Node/npm, Python 3, curl, OpenSSL y Chrome del sistema; Docker local |

La red interna Docker no publica PostgreSQL (5432) ni ClamAV (3310).
La app publica por defecto `127.0.0.1:18083`. El proxy HTTPS debe alcanzar ese
puerto sin abrirlo a Internet. Si el proxy está en otra máquina, Informática debe
preparar una conexión privada y una configuración adicional revisada.
La red base tampoco proporciona salida a AD: hay que habilitar sólo el destino
LDAPS autorizado. Verificar DNS, rutas, cortafuegos, hora y cadena de certificados.

Compose base permite comprobar el arranque local, pero no instala el proxy HTTPS
ni configura el AD corporativo. La demo añade un directorio inventado y una entrada
HTTP local; **no se usa en producción**. Quién termina HTTPS, acceso a AD, política
de firmas, destino de copias y supervisión siguen pendientes de Informática.

La construcción opcional de audios se describe en [Grabaciones del curso y de la ayuda](VOZ.md). Ejecuta `make voz` antes de preparar la imagen o el paquete Windows; la instalación reproduce los ficheros ya construidos.

## 2. Instalación desde el repositorio

1. Obtener la versión autorizada del repositorio público en una carpeta de servicio.
   Registrar `git rev-parse HEAD`; no desplegar una rama con cambios sin confirmar.
2. Crear fuera del repositorio `/etc/etf/entorno.env`, propiedad de la cuenta de
   servicio y permiso 0600. Tomar los nombres de [.env.plantilla](../../.env.plantilla)
   y generar contraseñas distintas para instalación, migración, aplicación,
   copias y cálculo de Memoria. No pegar valores en incidencias o Git.
3. Preparar un archivo adicional `/etc/etf/compose.interno.yaml` con los ajustes
   aprobados de AD, origen HTTPS, certificados y almacenamiento de paquetes.
4. Definir en la sesión de operación esta función; usar siempre los mismos archivos
   y nombre de proyecto. La ruta del repositorio debe ser absoluta.

```sh
ETF_REPOSITORIO="$PWD"
componer() {
  docker compose --project-directory "$ETF_REPOSITORIO" \
    --env-file /etc/etf/entorno.env -p etf \
    -f "$ETF_REPOSITORIO/compose.yaml" \
    -f /etc/etf/compose.interno.yaml "$@"
}
componer config --quiet
componer build app migrar
componer up -d --wait --wait-timeout 300 postgres clamav
componer run --rm migrar
componer up -d --no-deps --wait app
componer ps -a
curl --fail http://127.0.0.1:18083/salud
```

No ejecutar `compose config` sin `--quiet`: puede mostrar secretos. Revisar los
registros de preparación y migración y exigir salida 0. La app recibe credenciales
de `etf_app` y, para las cifras abiertas de Consulta, la credencial de lectura
técnica `etf_calculo_memoria`; nunca las de `etf_migrador`, `etf_copia` o instalación.
El binario comprueba la conexión a la base, pero no modifica el esquema.
La comprobación del historial y sus huellas corresponde al migrador.

Compose ejecuta `preparar-roles` antes de `migrar`, tanto con un volumen nuevo
como con uno existente. La preparación crea los roles que falten y sincroniza
sus atributos y contraseñas con el entorno privado en una transacción. No toca
expedientes, auditoría ni historial de migraciones. Usa el instalador por socket
local, sin red ni contraseña TCP. El socket es temporal, con permiso 0700,
compartido exclusivamente por PostgreSQL y el servicio de preparación.
Si encuentra una pertenencia inesperada de un rol técnico, falla y no arranca
el migrador. No ejecutar la actualización con `--no-deps`.

La primera cuenta de Administración funcional no se crea desde Usuarios y permisos:

1. Informática configura `ETF_PRIMER_ADMINISTRADOR` con el login autorizado en el entorno privado y lo entrega a `services.app.environment`.
2. Arranca Compose con sus dependencias. `preparar-roles` prepara los roles técnicos de PostgreSQL; después `migrar` aplica el esquema. Preparar roles no concede perfiles funcionales.
3. La persona autorizada entra con su cuenta de AD. El servidor resuelve su `objectGUID` y ejecuta el alta inicial una sola vez. Perfil de Administración, marca de inicialización y auditoría se confirman juntos.
4. Esa persona concede Administración a una segunda persona autorizada. Desde entonces, cada una pide a la otra los cambios de sus propios perfiles, vigencias, equipo o baja.

Retirar posteriormente todos los administradores no repite el alta inicial. Cambiar la variable o volver a entrar tampoco la repite. Una recuperación requiere una operación expresa y auditada de Informática; no se borra la marca para provocar otra concesión (SEG2, R-10, 14/10/2026).

El archivo adicional debe pasar las variables a `services.app.environment`:
ponerlas sólo en el fichero de entorno no las entrega automáticamente al contenedor.
Como mínimo, añadir origen HTTPS, las cuatro opciones AD, los límites de sesión si se cambian los valores iniciales y primer
administrador de la tabla siguiente. Montar la CA en sólo lectura en la ruta
indicada. No copiar certificados privados al repositorio.

Los montajes base de `idiomas/` y `temas/` son de sólo lectura. Para publicar paquetes
mediante Administración, el despliegue aprobado debe sustituirlos por almacenamiento
persistente escribible por UID/GID 10001, inicializado con el contenido de la versión.
Respaldar esos directorios, incluidas todas las versiones de plantillas. La demo usa
volúmenes propios y comprueba esta variante. No cambiar permisos de toda la carpeta
del repositorio para resolver un fallo de escritura.

Comprobar después por HTTPS: entrada AD, primer administrador, asignación de perfiles,
subida con antivirus y descarga Word. El primer administrador se concede una sola
vez: cambiar la variable o reiniciar no recupera un perfil retirado.

La [configuración de producción](CONFIGURACION.md) incluye los montajes de host,
la comprobación LDAPS, el contrato del proxy y la puesta en servicio. El paquete
Windows tiene [sus propios pasos](../../despliegue/windows/LEEME.md): comprobar la
suma del ZIP, instalar el motor y WSL2, instalar, parar y actualizar con sus
lanzadores. «Reiniciar» borra la demo tras confirmación; «Parar» conserva los datos.
Informática no debe cargar expedientes reales en ese paquete.

## 3. Variables de configuración

| Variables | Uso y valor inicial |
| --- | --- |
| `ETF_INSTALADOR_CONTRASENA` | Inicialización del volumen PostgreSQL; su contraseña se retira al crear los roles |
| `ETF_APP_CONTRASENA`, `ETF_MIGRADOR_CONTRASENA`, `ETF_COPIA_CONTRASENA` | Contraseñas distintas; inicialización y preparación las aplican a los roles y cada consumidor recibe sólo la suya |
| `ETF_CALCULO_MEMORIA_CONTRASENA` | Contraseña propia de lectura técnica para las cifras abiertas de Consulta, sin membresías, propiedad ni BYPASSRLS. PostgreSQL, preparación y app reciben la misma variable. Si falta o está vacía, el rol queda sin contraseña y sin inicio de sesión; no queda una credencial anterior activa |
| `ETF_HTTP_PUERTO` | Publicación local, 18083 por defecto |
| `ETF_RED_SUBRED`, `ETF_ENTRADA_SUBRED` | Redes Docker; elegir rangos privados sin colisión |
| `ETF_HTTP_DIRECCION` | Escucha de la app; Compose usa `0.0.0.0:8080` |
| `ETF_HTTP_ORIGEN` | Origen HTTPS exacto, sin ruta; debe coincidir con el acceso del navegador |
| `ETF_HTTP_PROXIES` | Redes CIDR, separadas por comas, de los proxies HTTPS que reenvían a la app (como `10.250.4.17/32`). Sólo de ellas se acepta `X-Forwarded-For`, y el proxy debe sustituir esa cabecera por la IP del cliente, no añadirla. Si se deja vacía con un proxy delante, el límite de intentos cuenta a todo el personal como un único origen |
| `ETF_BD_HOST`, `ETF_BD_PUERTO`, `ETF_BD_NOMBRE`, `ETF_BD_USUARIO`, `ETF_BD_CONTRASENA` | App: postgres, 5432, etf, etf_app; contraseña tomada de `ETF_APP_CONTRASENA` |
| `ETF_BD_MAX_CONEXIONES` | Máximo de conexiones de la app: 4 por defecto, entre 1 y 64; la conexión de cálculo tiene su propio pool |
| `ETF_BD_SSLMODE`, `ETF_BD_SSLROOTCERT` | `disable` en la red privada de Compose; sólo se admite también `verify-full`, con CA y PostgreSQL preparado para TLS. El migrador de Compose no configura TLS |
| `ETF_MIGRACIONES_RUTA` | Ruta de manifiestos del migrador; la imagen incluye `/opt/etf/migraciones` |
| `ETF_AD_URL` | URL `ldaps://` del controlador autorizado |
| `ETF_AD_BASE_DN` | Base de búsqueda del directorio; confirmar con Informática |
| `ETF_AD_FORMATO_BIND` | Formato con marcador `{usuario}`; confirmar que permite leer `objectGUID` |
| `ETF_AD_CA_RUTA` | Ruta dentro del contenedor al certificado público de la autoridad |
| `ETF_SESION_MAXIMA` | Duración absoluta en formato Go; por defecto `8h`, con tope técnico `24h` y nunca menor que la inactividad. Decisión del responsable del producto del 14/10/2026. No se prolonga al continuar la sesión |
| `ETF_SESION_INACTIVIDAD` | Por defecto `30m`; debe superar `2m` para permitir el aviso. La interfaz avisa dos minutos antes y permite continuar; al caducar vuelve a la entrada conservando los datos guardados. Configurar ambas variables en `services.app.environment` si se cambian |
| `ETF_PRIMER_ADMINISTRADOR` | Login de la persona autorizada para el alta inicial |
| `ETF_CLAMAV_DIRECCION` | `clamav:3310` en Compose |
| `ETF_IDIOMAS_RUTA`, `ETF_TEMAS_RUTA` | `/datos/idiomas`, `/datos/temas` en Compose |
| `ETF_IDIOMA`, `ETF_TEMA` | `es`, `expedientes-etf` |

Los tiempos de AD (5 segundos), ClamAV (45 segundos en el arranque) y parte de los
límites de subida están fijados actualmente en los adaptadores. TRD prevé ampliar
la configuración; no hay que inventar variables para cambiarlos.
No hay variables SMTP ni servicio de correo: la aplicación no envía avisos por correo.
Los avisos se consultan dentro de ella. No configurar una cuenta de correo.

`ETF_MODO=desarrollo` pertenece a la demostración, no a producción.

## 4. Arranque, parada y actualización

Arrancar: `componer up -d --no-deps --wait app`, con PostgreSQL, ClamAV y esquema
previamente comprobados. Parar escrituras: `componer stop app`. Parar el conjunto:
`componer down`; conserva los volúmenes. **No usar `down --volumes` en producción.**

Para actualizar, reservar una ventana y avisar a los equipos. Registrar versión
anterior, configuración y huellas de imágenes. Parar app y tomar copia de base y
paquetes como en §5. Guardar la imagen anterior antes de reconstruir la etiqueta.
Obtener la nueva versión autorizada, construir app y migrador y ejecutar
`componer run --rm migrar`, conservando sus dependencias. La preparación de roles
se ejecuta antes de migrar también al actualizar un volumen anterior a F4.07:
no hay que ejecutar `habilitar_calculo_memoria.sh` a mano. Añadir la contraseña
propia de cálculo al entorno privado si se va a habilitar esa conexión.
No editar SQL ya aplicado ni ejecutar manualmente sus archivos: el
migrador comprueba huellas, dependencias y orden por módulo; rechaza retrocesos.

Si migrar termina correctamente, arrancar app y comprobar salud, acceso, ficha y
Word. Si falla, mantener cerrada la entrada, conservar registros y comprobar qué
migraciones quedaron aplicadas. No asumir que fallar un paso revierte todo el lote.
Para volver atrás, usar la imagen anterior; si no entiende el esquema nuevo,
restaurar la copia previa y los paquetes de esa misma versión antes de abrir.
Se perderían escrituras posteriores a la copia: comunicarlo y reconciliarlas.
Idiomas y temas (FX8): la actualización sustituye la base de la distribución y
conserva lo publicado por Administración (todo lo que cuelga de `.versiones`) y
los ficheros cambiados a mano, que se avisan. Sólo se retira un fichero que la
entrega nueva ya no trae si lo instaló la entrega anterior, según su manifiesto,
y no se ha tocado; las copias manuales se conservan. Si después los textos no cargan,
por JSON dañado o claves repetidas, la aplicación no arranca y el diagnóstico
indica fichero y clave. Reinstalar la versión anterior del paquete no devuelve
la base a su estado previo, porque su incorporador no sustituye ficheros
existentes. Para volver atrás, restaurar la copia previa de los volúmenes de
idiomas y temas (§5) junto con la base de datos, o reinstalar con el
actualizador nuevo.

TRD exige automatizar copia, salud y vuelta atrás; Compose base no aporta todavía
esa automatización completa. Este manual describe la operación supervisada.

## 5. Copias y restauración

Hacer copia diaria cifrada fuera del servidor y otra antes de cada actualización.
TRD tolera como máximo un día de pérdida y una parada de unas horas. Informática
fija destino, retención de copias, cifrado, custodios y alarmas. La conservación de
expedientes durante 25 años no implica guardar todas las copias durante 25 años.
Una copia no está validada hasta comprobar su restauración.

El conjunto recuperable incluye: base PostgreSQL, todas las versiones de idiomas,
catálogos de archivos, plantillas y temas, versión de imágenes/código, configuración
externa y certificados necesarios. Los secretos se custodian por separado.
Las firmas antivirus se pueden reponer desde el suministro local aprobado.

Con app parada y directorio de copia en almacenamiento cifrado, permisos 0700:

```sh
umask 077
ETF_COPIA_DIR=/ruta/cifrada/etf/copia-previa
mkdir -p "$ETF_COPIA_DIR"
componer exec -T postgres pg_dump -U etf_instalador -d etf -Fc > "$ETF_COPIA_DIR/etf.dump"
componer exec -T postgres pg_restore --list < "$ETF_COPIA_DIR/etf.dump" > /dev/null
```

Comprobar ambos códigos de salida. El rol `etf_copia` tiene `BYPASSRLS`, pero el
ensayo detectó que carece de lectura en algunas tablas (primero `catalogo.smi`).
No dar por operativa una copia con ese rol hasta completar y probar sus permisos
en una tarea SQL independiente. Estos comandos usan al operador local
`etf_instalador`, que incluye todos los equipos. No usar `etf_app`, que podría
omitir datos o fallar por la seguridad por filas. Aquí se usa el socket local del contenedor, accesible sólo
al operador de Docker; no se expone al exterior ni exige imprimir contraseñas.
El [anexo de operación](OPERACION.md#a-cifrar-y-comprobar-una-copia) concreta el cifrado y la verificación del conjunto.
El formato personalizado contiene objetos, propietarios, permisos, datos y secuencias.
No contiene roles globales ni los permisos de la propia base al no usar `--create`.

Copiar los directorios persistentes de idiomas y temas con app parada, conservando
permisos y todas las versiones. En montajes de host: `tar -cf` desde el directorio
que contiene ambos. En volúmenes: montar los mismos volúmenes en un contenedor
puntual sin red y usar tar. El script de ensayo usa los volúmenes de la demo.
Registrar sumas SHA-256 y tamaño; trasladar al destino cifrado fuera del servidor y
comprobar la recepción. Un archivo `.dump` sin cifrar no cumple la política de copias.

Antes de restaurar, parar `app` y la entrada. No abrirlas hasta completar el paso de revisión. Con la base vigente todavía disponible, exportar la lista completa de identidad a un fichero custodiado por Informática, fuera del volumen y de la copia que se va a restaurar:

```sh
umask 077
componer exec -T postgres psql -XAt -v ON_ERROR_STOP=1 -U etf_instalador -d etf \
  -c 'SELECT identidad.exportar_identidad();' > identidad-vigente.json
```

Conservar el fichero sin permisos para otras personas y verificar que contiene la lista completa. Tiene nombres y permisos del personal; no enviarlo por correo ni añadirlo al repositorio. Si la base vigente se ha perdido, recuperar una exportación reciente custodiada fuera del servidor y contrastarla con el Programa. Si no hay una referencia fiable, mantener la aplicación parada. La base restaurada debe incluir la migración `identidad/0004` y la imagen de la aplicación debe incorporar SEG4. Para una copia anterior, aplicar primero las migraciones con el ejecutor separado y usar una imagen compatible que incluya la revisión, sin abrir la aplicación.

Restaurar primero en una máquina aislada, con la misma versión PostgreSQL 18 y
roles inicializados desde Compose; no importar un volcado desconocido. Parar app y
cerrar acceso del proxy. Conservar copia del estado dañado para investigar. Los
siguientes comandos **borran la base `etf` del proyecto elegido**; revisar destino,
copia y autorización de la intervención antes de ejecutarlos:

```sh
componer exec -T postgres dropdb -U etf_instalador etf
componer exec -T postgres createdb -U etf_instalador -O etf_propietario etf
componer exec -T postgres pg_restore -U etf_instalador -d etf \
  --exit-on-error < "$ETF_COPIA_DIR/etf.dump"
componer exec -T postgres psql -X -v ON_ERROR_STOP=1 -U etf_instalador -d postgres \
  -c 'REVOKE ALL ON DATABASE etf FROM PUBLIC;
      GRANT CONNECT ON DATABASE etf
      TO etf_app, etf_migrador, etf_copia, etf_calculo_memoria;'
```

Restaurar idiomas y temas de esa copia, con sus propietarios y permisos. Verificar migraciones, cantidades por equipo, secuencias, cadena de auditoría y referencias de paquetes. Tomar la fecha de la copia del registro de la operación de copia, con su zona horaria; no usar la fecha del fichero restaurado. Antes de arrancar, activar la revisión obligatoria:

```sh
set -o pipefail
python3 -B scripts/preparar-revision-restauracion.py \
  --referencia identidad-vigente.json --fecha-copia '2026-10-14T08:00:00+02:00' | \
  componer exec -T postgres psql -X -v ON_ERROR_STOP=1 -U etf_instalador -d etf
```

Comprobar que ambos comandos terminan correctamente. Si falla cualquiera, no arrancar. El paso crea una marca persistente, compara altas, bajas, perfiles y equipos por el identificador estable de AD, borra todas las sesiones restauradas y registra la operación. La aplicación no puede ejecutar este paso. Una repetición crea otra revisión, anula la anterior y vuelve a cerrar las sesiones. Se exige al menos una persona administradora activa en ambas listas; si falta, Informática y el responsable del Programa deben recuperar una identidad administrativa válida antes de repetir, dejando constancia de la operación. No se concede un perfil por cambiar la configuración de arranque.

Arrancar la imagen correspondiente y la entrada, sin sembrar de nuevo. Sólo Administración funcional puede entrar, con AD y permisos vigentes en ambas listas. La única pantalla disponible es «Revisión tras restaurar una copia». Muestra la fecha de la copia y los permisos de la copia frente a los anteriores a la restauración. Administración aplica o descarta cada cambio; descartar conserva los permisos de la copia, incluso si se habían retirado después. Si un equipo de destino falta en la copia, Informática debe recuperarlo antes de aplicar, o Administración descarta el cambio conscientemente. Las altas pendientes siguen pendientes. Otra persona de Administración debe aplicar los cambios de tus propios perfiles o equipo; no puedes aplicarlos a tu cuenta.

Hasta decidir todos los cambios y confirmar la revisión, el resto del personal no puede entrar y las demás funciones permanecen cerradas. Cada decisión y la confirmación quedan auditadas. Tras confirmar, comprobar acceso de cada perfil, aislamiento entre equipos, ficha y generación Word. Comparar también la cadena de auditoría con el último sello guardado fuera del servidor mediante el [comprobador local](OPERACION.md#b-registros-y-cadena-de-auditoría). Registrar hora, copia y pérdida de datos; retirar la exportación temporal según la política de custodia.

Ensayo reproducible local, sólo con datos inventados del worktree:

```sh
make demo
python3 -B scripts/ensayar-restauracion.py
```

El script no admite nombres de proyectos externos. Serializa operaciones con la
demo y copia mediante el operador local. Recupera paquetes con el preparador
puntual `incorporar`, sin red, para conservar permisos y UID/GID originales sin
dar privilegios a app. Borra realmente la base y el contenido de idiomas
y temas, restaura y compara todos los datos y secuencias y el contenido y permisos de los paquetes.
Antes de copiar y después de restaurar ejecuta el recorrido de Anexo 3 a Acta
con Chrome del sistema; también comprueba los siete accesos de la demo.
Así verifica que las referencias de la base resuelven los archivos recuperados.
Las copias quedan en `TMPDIR`, con permiso privado; se borran sólo tras éxito.
Si falla después de anunciar la ruta de copia, conservarla y recuperar desde ella
antes de limpiar la demo. Un fallo del recorrido previo no llega a crear la copia.
Es un ensayo lógico local; no acredita cifrado, traslado ni recuperación de un disco
perdido. Repetirlo en preproducción con el sistema corporativo antes de datos reales.

## 6. Salud, registros y firmas antivirus

Usar `componer ps -a`, `componer logs --tail 100 app migrar` y
`curl --fail http://127.0.0.1:18083/salud`. Docker comprueba esa misma ruta.
Ahora `/salud` informa del estado básico de la base y avisos de configuración;
HTTP 200 no demuestra acceso AD ni firmas recientes. TRD prevé una salud detallada
y rutas separadas de vivo/listo; no están disponibles en esta versión.
Añadir comprobaciones operativas de LDAPS, antigüedad de firmas, disco, copia diaria,
restauración periódica y caducidad de certificados. La política y el receptor de
alertas se concretan con Informática.

Los registros Docker usan controlador local, máximo 10 MB por archivo y tres
archivos por contenedor. Revisar códigos técnicos sin adjuntar cuerpos HTTP,
contraseñas, nombres, contactos ni textos de fichas. La auditoría funcional queda
encadenada en PostgreSQL; no se sustituye por logs Docker. Restringir acceso y
exportar sólo por el procedimiento autorizado.

ClamAV no ejecuta FreshClam ni descarga firmas. El volumen se rellena al crearlo;
cambiar la imagen no actualiza un volumen existente. Informática debe reponer
firmas desde su fuente local y reiniciar ClamAV, verificar `PONG`, fecha de firmas y
un ensayo antivirus aprobado. Las firmas de la imagen local pueden estar antiguas.
Si ClamAV no responde, el anexo permite continuar con aviso y auditoría; plantillas,
paquetes y hojas de carga necesitan revisión satisfactoria. Supervisar ese estado.

## 7. Rotación de claves y certificados

En ventana de mantenimiento, parar app y los procesos de migración/copia.
Actualizar las contraseñas en el gestor o fichero externo privado y ejecutar
`componer run --rm preparar-roles`. El guion aplica las contraseñas a los roles
existentes; repetirlo es seguro. Recrear los servicios consumidores y verificar
conexiones y una copia nueva. Una contraseña de cálculo vacía deshabilita ese
inicio de sesión y elimina su contraseña anterior. No poner `ALTER ROLE ... PASSWORD`
con una contraseña literal en el historial. Mantener `etf_instalador` sin contraseña
TCP. El acceso al grupo Docker equivale a administración del servidor: restringirlo.

Renovar el certificado HTTPS en el proxy decidido por Informática, comprobar nombre,
cadena y fecha y recargarlo. Para cambiar la CA de AD, montar la cadena nueva en sólo
lectura, recrear app y probar el acceso; nunca desactivar la validación TLS.
La aplicación genera identificadores aleatorios de sesión y la clave CSRF al
arrancar: no existe una variable de clave de firma que el operador deba rotar.
Tras una sospecha de robo de sesiones, revocarlas según el procedimiento de identidad;
reiniciar por sí solo no garantiza borrarlas de PostgreSQL.
Rotar claves de cifrado de copias con la herramienta corporativa conservando la
capacidad de descifrar las copias retenidas y ensayando la recuperación.

## 8. Datos guardados y ubicación

| Ubicación | Contenido |
| --- | --- |
| Volumen `postgres_datos`, montado en `/var/lib/postgresql` | Familias, adultos, menores, contactos, intervención, evaluaciones, episodios, comisiones, informes, emisiones, metadatos y referencias de versiones, usuarios/perfiles/equipos, huellas de sesiones, migraciones y auditoría |
| `/datos/idiomas` | Textos, catálogos y versiones inmutables de plantillas Word y paquetes de idioma |
| `/datos/temas` | Manifiestos, tokens, fuentes y recursos de las versiones de temas |
| Volumen `clamav_firmas` | Firmas antivirus, no documentos de expedientes |
| Docker, controlador local | Registros técnicos limitados; no deben contener datos personales |
| Configuración fuera de Git | Contraseñas, parámetros del entorno y material de certificados |

Los anexos recibidos, las actas de carga, las hojas y los documentos generados se procesan en
memoria y se descartan; no hay archivo documental de expedientes. Las plantillas
administradas sí se conservan. No se guardan contraseñas de AD. Los documentos
que una persona descarga quedan bajo la custodia de su equipo, fuera de la app.
La conservación prevista es de 25 años desde el cierre, con retirada de datos personales revisada
por dos personas distintas de Administración funcional. El procedimiento técnico
y sus límites están en [el anexo de operación](OPERACION.md#d-conservación-y-retirada-de-datos-personales). La copia debe respetar ese procedimiento y los plazos
aprobados; una restauración no debe reintroducir datos ya eliminados sin revisión.

## 9. Incidentes

Registrar hora, síntomas y versión sin copiar datos personales. Avisar a Informática
y responsable del producto; involucrar al DPD y al responsable de seguridad si hay exposición o pérdida
de datos. Aplicar el procedimiento corporativo y sus plazos, sin fijarlos aquí.

Ante acceso indebido, cerrar la entrada web, conservar base y logs con acceso
restringido, revocar cuentas/sesiones afectadas y rotar secretos comprometidos.
No limpiar volúmenes ni ejecutar otra siembra. Ante pérdida de base, detener app,
identificar la última copia verificada y seguir §5. Ante fallo AD, comprobar red,
hora y certificados; no sustituirlo por acceso sin contraseña. Ante ClamAV caído,
restablecer daemon y firmas y revisar los anexos admitidos con aviso. Ante disco
lleno, ampliar almacenamiento o retirar sólo registros/copias conforme a retención;
no borrar archivos PostgreSQL manualmente.

Documentar alcance, recuperación y comprobaciones. Abrir de nuevo con autorización
de los responsables. Revisar causa y repetir la prueba que falló.

## 10. Lista de comprobación para producción

- [ ] Máquina, disco cifrado, hora, DNS y acceso administrativo restringido.
- [ ] Versión autorizada, imágenes fijadas y sin cambios locales pendientes.
- [ ] HTTPS y certificados válidos; acceso sólo corporativo, provincial y VPN.
- [ ] LDAPS autorizado y `objectGUID` comprobado; límites de sesión comprobados (30 minutos sin actividad y 8 horas absolutas por defecto).
- [ ] Secretos externos con permiso 0600; separación de roles y app sin migrador.
- [ ] Migraciones aplicadas por su ejecutor; salud y pruebas funcionales aprobadas.
- [ ] Almacenamiento persistente de paquetes, versiones anteriores y permisos probados.
- [ ] ClamAV con firmas recientes y suministro local supervisado.
- [ ] Copia diaria cifrada fuera del servidor, alarmas y restauración corporativa ensayada.
- [ ] Vuelta atrás preparada y probada con imágenes, base y paquetes de la misma versión.
- [ ] EIPD aprobada por el DPD y revisión completa de seguridad sin fallos graves.
- [ ] Perfiles y aislamiento de equipos verificados; auditoría y conservación revisadas.
- [ ] Supervisión, responsables y procedimiento de incidentes acordados.
- [ ] el responsable del producto acepta el despliegue antes de introducir expedientes reales de 2026.


## 11. Arquitectura y contenedores

El navegador carga la interfaz React y llama a la API Go del mismo origen.
Go sirve la interfaz y conecta con PostgreSQL, Active Directory y ClamAV.
Es un monolito modular: las reglas de negocio están separadas de HTTP, SQL,
lectura de anexos y generación Word mediante puertos y adaptadores. No hay que
instalar un servidor independiente por módulo ni un servidor Node en producción.

```text
Chrome / Edge en red autorizada
              |
          HTTPS 443
              |
      Proxy de Informática
              |
      127.0.0.1:18083
              |
          app Go :8080
          |      |      |
   PostgreSQL  ClamAV  AD por LDAPS
       :5432    :3310      :636

Instalación y actualización:
PostgreSQL -> preparar-roles -> migrar -> app
```

| Servicio de Compose | Función y permanencia | Datos y acceso |
| --- | --- | --- |
| `postgres` | Base PostgreSQL 18.4; permanece encendida | UID/GID 70:70; volumen `postgres_datos`; sin puerto publicado |
| `preparar-roles` | Orden puntual antes de migrar y al rotar contraseñas | UID/GID 70:70; sólo socket privado `postgres_socket`; sin red |
| `migrar` | Orden puntual con `etf_migrador`; aplica dependencias y huellas | Imagen con SQL y manifiestos; no comparte credenciales con app |
| `app` | API, interfaz, identidad, cifras y Word | UID/GID 10001:10001; paquetes persistentes; único puerto local publicado |
| `clamav` | Antivirus local para subidas | UID 100, GID 101; volumen de firmas de sólo lectura; sin puerto publicado |

Las raíces son de sólo lectura, sin capacidades Linux y con
`no-new-privileges`. Compose limita procesos, CPU y memoria: PostgreSQL 512 MiB,
app 256 MiB, migrador 128 MiB, preparación 64 MiB y ClamAV 2 GiB. Estos límites no
sustituyen la prueba de carga; vigilar reinicios y falta de memoria antes de
cambiarlos. `GOMEMLIMIT=192MiB` fija el objetivo de memoria del proceso Go.

`interna` no tiene salida al exterior. `entrada` permite publicar app en el
host, pero desactiva el enmascaramiento de salida. El acceso a AD requiere la
conexión privada que prepare Informática. El puerto de PostgreSQL nunca se
abre para que funcione el navegador. Las redes, el proxy y sus reglas aprobadas
se conservan con la configuración externa.

El rol `etf_propietario` posee el esquema y no inicia sesión. `etf_migrador`
puede asumirlo durante la migración; `etf_app` queda sujeto a permisos y seguridad
por filas. `etf_calculo_memoria` sólo calcula las cifras autorizadas para Consulta;
no tiene `BYPASSRLS`. `etf_copia` tiene ese atributo técnico, con los permisos
incompletos indicados en §5. `etf_instalador` queda para operación por socket
local y sin contraseña TCP. Restringir Docker: quien accede a él puede leer la base.

No hay caché de expedientes en la aplicación instalada en escritorio. Su aviso
sin conexión no permite trabajar fuera de línea. No se deben esperar correos,
firma electrónica ni registro automático externo en esta versión.

## 12. Riesgos abiertos que dependen de Informática

El registro de riesgos del informe de seguridad de cada instalación
recoge los riesgos y responsables. Esta revisión del manual no los da por cerrados.

| Riesgo | Actuación sobre esta aplicación | Evidencia necesaria |
| --- | --- | --- |
| R-01, HTTPS e IP del cliente | Decidir proxy, certificado, HSTS, conservación de `Origin`, sustitución de `X-Forwarded-For` y límites de rutas públicas | Acceso desde cada red autorizada, rechazo desde otra y dos clientes con límites independientes |
| R-02, copias | Programar copia diaria cifrada fuera del servidor, custodiar claves y ensayar restauración en preproducción | Archivo recuperado del destino real, descifrado y recorrido de acceso, fichas y Word |
| R-04, coste de rutas públicas | Limitar conexiones y peticiones por cliente en el proxy, coordinado con las mejoras de desarrollo | Prueba de carga sin impedir el uso normal |
| R-05, sello de auditoría | Coordinar el sellado diario y custodiar ID y huella en un registro independiente de sólo adición | Comprobador contra sello externo; tarea programada y alarma si no hay sello |
| R-06, AD | Confirmar principal de bind, búsqueda y `objectGUID` sobre la misma cuenta | Prueba con cuenta autorizada, renombrada y login reutilizado; coordinar corrección con desarrollo |
| R-07, antivirus | Suministrar y supervisar firmas locales; acordar tratamiento de anexos admitidos sin revisión | Firmas recientes, `PONG` y ensayo aprobado de rechazo de fichero infectado |
| R-12, imágenes | Coordinar actualización de versiones y huellas con desarrollo | Análisis local de imágenes y preproducción; conservar artefacto anterior |
| R-14, red y rol de copia | Revisar red de ClamAV, reglas `pg_hba` del rol de copia y registro seguro de fallos de conexión | Permisos y conexiones ensayados sin publicar 5432 ni registrar datos personales |
| R-22, recuperación de permisos | Invalidar sesiones y reconciliar bajas, perfiles y equipos posteriores a la copia | Prueba de que una persona trasladada no recupera el acceso a su equipo anterior |

Antes de datos reales siguen pendientes el visto bueno del responsable del producto, el criterio
del DPD sobre conservación y vínculos de auditoría, la prueba en preproducción y
la caducidad máxima de sesión. Informática aporta operación y evidencias; no
resuelve esas decisiones modificando SQL o concediéndose un perfil funcional.

## 13. Preparar el PDF

Desde la raíz del repositorio, con Python 3, Pillow y LibreOffice local:

```sh
TMPDIR="$PWD/bin/temporales" python3 -B docs/manuales/exportar.py \
  --manual sistemas --salida docs/sistemas
```

El PDF incluye este manual y los anexos de configuración, operación, diagnóstico
y evidencia MANSIS. Las evidencias históricas F5.05, FX2 y WIN1 siguen enlazadas
por separado. Sin `--manual`, el exportador conserva los tres manuales de usuario
existentes. Para compartir el PDF, acompáñalo del árbol de documentación si se
necesitan los enlaces a otros archivos; el PDF no los incorpora como adjuntos.
