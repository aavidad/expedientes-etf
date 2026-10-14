# Catálogo en PostgreSQL · F0.12

`cmd/migrar` aplica las migraciones de `catalogo/manifiesto.json` en orden
causal. Los números están reservados en el registro de ETF. La primera
migración crea las tablas, las restricciones y los permisos; la segunda carga
una instantánea de los JSON de `datos/catalogos/`. El ejecutor registra su
huella y omite las migraciones ya aplicadas. Repetir la orden no duplica datos.

Se eligió una migración de datos para mantener la carga y su registro en la
misma transacción. No hay otro proceso con credenciales de escritura y la
aplicación nunca carga ni migra catálogos.

La carga contiene ocho equipos, catorce centros, 174 municipios y cinco ELA.
Guarda centro y equipo por separado; Íllora pertenece al centro
Pinos Puente-Moclín y al ETF 6. Las ELA conservan su municipio matriz y los
municipios fuera del ámbito mantienen su marca. `zts` sigue sin valor.

`listas.json` incorpora 24 opciones expresamente enumeradas en PRD §4.2:
subprograma, servicio derivante, características familiares, pronóstico,
papel parental y tramos de ingresos sobre el SMI. La BD sólo recibe los
códigos y sus propiedades. Las etiquetas están en
`idiomas/{es,en}/catalogos/listas.json`. Las demás listas, equivalencias y
reglas necesitan sus fuentes confirmadas y nuevas migraciones; no se han
inventado sus opciones ni las decisiones pendientes del Programa.

Las versiones iniciales de `listas` y `asignacion_municipios` rigen desde el
1 de enero de 2026. Los intervalos son `[desde, hasta)`: la fecha del cambio
pertenece a la nueva versión. La Memoria elige la versión que rige el
31 de diciembre del año solicitado. El índice GiST de exclusión impide
solapamientos en un mismo ámbito.

La operación `catalogo.publicar` sólo puede ejecutarla el propietario durante
el despliegue. Bloquea el ámbito, cierra la vigencia anterior y publica el
borrador en la misma transacción. `catalogo.publicacion` conserva el límite
anterior y el autor. Los disparadores impiden cambiar o borrar contenido
publicado, trasladarlo a un borrador y truncar las tablas. Esta historia de
publicación no sustituye la auditoría encadenada prevista en F0.14. Los UUID
de autor aún no tienen relación con identidad; esa relación deberá añadirse
cuando exista su tabla, mediante una migración con dependencia explícita.

Todas las tablas tienen RLS forzada. `etf_app` sólo tiene `SELECT`; sin
identidad válida en el contexto de la transacción no ve filas. Sólo
Administración puede leer borradores; los otros perfiles leen versiones
publicadas. La app no puede crear objetos, escribir ni publicar. F0.12 no
implementa la edición administrativa ni las otras tablas de reglas previstas
en ESQUEMA §1.2. `catalogo/0005` crea SMI por año entre 2026 y 2040,
inicialmente sin importes. El importe anual se calcula como el mensual por 14
pagas. El historial es inmutable; la edición exige Coordinación y registra
cambio y auditoría en una transacción con versión optimista.

## Consultas HTTP

El arranque registra estas rutas:

- `GET /api/admin/catalogos/{ambito}/versiones`
- `GET /api/admin/catalogos/{ambito}/vigente?fecha=2026-07-01`
- `GET /api/admin/catalogos/{ambito}/vigente?anio=2026` (usa el 31/12)
- `GET /api/admin/catalogos/{ambito}/datos?fecha=2026-07-01`

`datos` devuelve códigos, referencias y claves de texto de ubicaciones o
listas; nunca etiquetas. Las respuestas de error sólo llevan claves de idioma.
Las consultas usan una transacción y el contexto RLS de la identidad fijada
por el servidor. No se entrega el resultado si falla la confirmación.

Todavía no hay autorización de Administración: corresponde a F0.10.
Un middleware devuelve 403 por defecto antes de llamar al caso de uso.
Sólo el valor explícito `ETF_MODO=desarrollo`, leído al arrancar, abre estas
rutas con una identidad sintética fijada en el servidor. Los parámetros y las
cabeceras no pueden activar ese modo ni aportar identidad. No se debe activar
este modo en una instalación con datos reales. La autorización y el apunte
encadenado de cada lectura deben integrarse antes de abrir estas rutas en
producción.

La tabla SMI usa `GET /api/coordinacion/smi` para consultar los años y
`GET /api/coordinacion/smi?anio=2028` para consultar un año. `PUT` actualiza
el importe mensual e incluye `version_esperada`; un cambio concurrente devuelve
409 con la versión actual. Estas rutas usan una sesión activa, CSRF en PUT y
autorización de Coordinación en la aplicación y PostgreSQL.

## Comprobaciones

```sh
python3 -B migraciones/catalogo/generar_carga.py --comprobar
python3 -B migraciones/catalogo/ensayar.py
go vet ./...
go test ./...
```

El generador comprueba que la instantánea SQL coincide con los cuatro JSON.
Una migración ya aplicada no se regenera: una modificación exige otra
migración reservada. La comprobación queda incorporada en la CI.

El ensayo usa la imagen PostgreSQL 18 fijada en Compose, secretos temporales,
un contenedor y un volumen nuevos. Ejecuta el `cmd/migrar` del árbol sobre una
base limpia y repite la orden; después actualiza otra base que sólo tiene
`plataforma/0001`. Ejecuta las pruebas de catálogo y de transacciones con RLS,
y comprueba el plan de consulta. Al terminar destruye los recursos creados.
Las pruebas de integración se omiten sin `ETF_POSTGRES_INTEGRACION=1`; una
vez activadas, una base no disponible hace fallar la prueba.

F2.02 añade `0012_avisos.sql`: duraciones de fase en el ámbito `plazos` y
consentimiento a los 30 días naturales. La edición administrativa conserva
los parámetros y las fases de cada versión anterior. Las reglas publicadas
siguen siendo inmutables y la lectura usa RLS.

La actualización publica las nuevas reglas con vigencia desde el día de la
migración. Si ya hay una versión de plazos publicada con fecha de inicio de
hoy o futura, la migración rechaza el estado previo antes de crear la tabla:
no modifica reglas publicadas ni instala avisos sin reglas vigentes. Una
versión que empieza hoy permite repetir la actualización al día siguiente;
una versión futura requiere que el director revise la fecha de instalación.
