# Arranque local de F0.03

Desde la raíz del worktree:

```sh
cp .env.plantilla .env
chmod 600 .env
# Rellenar las cuatro contraseñas con valores distintos.
docker compose config --quiet
docker compose up -d --build
docker compose ps -a
curl --fail http://127.0.0.1:18083/salud
```

`preparar-roles` y `migrar` son órdenes separadas. Compose espera que PostgreSQL
esté disponible, prepara los roles y exige la salida correcta del migrador antes
de arrancar `app`. Para repetir la migración, sin ejecutar el servidor:

```sh
docker compose run --rm migrar
python3 despliegue/base/comprobar-migraciones.py
go vet ./...
go test ./...
```

El ensayo Python crea una base vacía `etf_ensayo_f003`, comprueba permisos,
huellas, dependencias, concurrencia y reversión de una migración fallida, y la
elimina al terminar. Se niega a usar una base que ya exista. Sólo debe ejecutarse
en este Compose local con datos inventados.

La configuración del migrador es `ETF_BD_HOST`, `ETF_BD_PUERTO`,
`ETF_BD_NOMBRE`, `ETF_MIGRADOR_CONTRASENA` y, fuera de la imagen,
`ETF_MIGRACIONES_RUTA`. El usuario de conexión es siempre `etf_migrador`.
Se necesita `psql` de PostgreSQL 18 en el equipo cuando se ejecuta el binario
fuera del contenedor; la imagen de `migrar` ya lo incluye. Los errores y los
registros emiten claves técnicas; no imprimen SQL, valores de entorno ni el
diagnóstico bruto del servidor.

Las contraseñas de migrador, instalador y copias no entran en `app`. La
inicialización crea los roles y el servicio idempotente `preparar-roles` se
ejecuta después, en cada arranque con migración: crea los roles ausentes y
actualiza sus atributos y contraseñas desde el entorno privado, sin modificar
datos ni migraciones aplicadas. Esto permite
actualizar volúmenes anteriores a F4.07 sin habilitar roles a mano.
`ETF_CALCULO_MEMORIA_CONTRASENA` es propia del cálculo de Memoria; si falta o está
vacía, su rol existe sin contraseña y sin inicio de sesión. La demo genera esta
credencial también al reutilizar un entorno antiguo.

El superusuario `etf_instalador` permanece sin contraseña TCP. La preparación
usa exclusivamente un socket Unix con permiso 0700, compartido con PostgreSQL
en un volumen temporal con propietario 70:70. Sólo PostgreSQL y el servicio de
preparación lo montan; este último no tiene red. Una pertenencia inesperada de
los roles técnicos aborta la transacción y bloquea al migrador.

Para rotar contraseñas, parar los consumidores, actualizar el entorno privado y
ejecutar `docker compose run --rm preparar-roles`. Recrear después los
consumidores y comprobar conexiones. No usar `--no-deps` al actualizar: omitiría
la preparación obligatoria.

`postgres` y `clamav` sólo comparten la red interna. `app` tiene además una red
de entrada, necesaria para publicar el puerto con Docker 29.5.2. La publicación
es sólo en `127.0.0.1`, y esa red tiene desactivado el enmascaramiento de salida.
Las subredes se pueden ajustar con `ETF_RED_SUBRED` y `ETF_ENTRADA_SUBRED` si
coinciden con una red del equipo. El puerto se cambia con `ETF_HTTP_PUERTO`.
El HTTPS y el acceso corporativo corresponden a P.01, según TRD §12.

Los contenedores usan usuarios sin privilegios, raíz de sólo lectura,
capacidades vacías y `no-new-privileges`. PostgreSQL escribe en su volumen y en
memoria temporal; ClamAV sólo en memoria temporal. `idiomas/` y `temas/` se
montan en sólo lectura para esta entrega. La imagen conserva una copia de
reserva de sus datos: arranca también sin esos montajes. El mecanismo de
publicación y escritura de versiones administradas corresponde a F0.14c.

ClamAV arranca directamente con `clamd`. La imagen fijada contiene firmas y
Docker las copia en un volumen nuevo; no usa FreshClam ni accede al exterior.
Las firmas incluidas ya avisan de antigüedad en el ensayo del 13/10/2026. Este
arranque prueba el daemon y no acredita firmas actualizadas para producción.
El suministro y actualización periódica desde una fuente local siguen
pendientes en TRD §13; al actualizarlas debe reemplazarse el contenido del
volumen y reiniciarse `clamd`. No basta cambiar la imagen si el volumen ya existe.

La salud de app sigue siendo la del esqueleto F0.04. Su conexión real a la base
y la comprobación de compatibilidad del esquema aún no están implementadas en
`cmd/etf/`, fuera de la zona F0.03. Aquí se comprueba el arranque sin credenciales
de migrador, la separación de contenedores y la conectividad del migrador.

Para borrar únicamente este entorno local y sus datos inventados:

```sh
docker compose down --volumes
```
