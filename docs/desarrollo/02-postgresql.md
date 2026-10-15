# 2. PostgreSQL y migraciones

Las tablas viven en esquemas por módulo. La aplicación se conecta como `etf_app`; el despliegue prepara roles y aplica migraciones antes de arrancarla. PostgreSQL es la segunda barrera de autorización, además de los permisos que comprueban HTTP y los casos de uso.

## Roles técnicos y perfiles

| Rol PostgreSQL | Uso | Restricción que debes conservar |
| --- | --- | --- |
| `etf_propietario` | Propietario sin inicio de sesión | No se usa como cuenta de aplicación |
| `etf_migrador` | Ejecutor puntual de migraciones | Sus credenciales quedan fuera del contenedor de la app |
| `etf_app` | Peticiones ordinarias | Sin propiedad ni `BYPASSRLS`; permisos concedidos objeto a objeto |
| `etf_calculo_memoria` | Cálculo interno de cifras para Consulta | Sin tablas personales, escritura, membresías ni `BYPASSRLS` |
| `etf_copia` | Copia completa | Sólo lectura; su `BYPASSRLS` se limita al servicio de copias |

Los perfiles funcionales no son esos roles PostgreSQL. Coordinación, `profesional_etf`, Consulta y Administración viajan en el contexto de la transacción, derivados de la sesión. Administración mantiene usuarios y paquetes, pero no lee fichas. Consulta recibe una proyección de cifras sin nombres, composición ni narrativas. El profesional trabaja sobre su equipo; un perfil combinado conserva las restricciones de equipo aplicables a su sesión.

## Contexto RLS

Al iniciar cada transacción, [Base.Ejecutar](../../internal/adaptadores/postgres/base.go) fija:

```sql
SELECT set_config('app.usuario_id', $1, true),
       set_config('app.perfiles', $2, true),
       set_config('app.equipos', $3, true);
```

El tercer argumento `true` limita los valores a la transacción. La conexión vuelve al grupo sin identidad heredada de la petición anterior. El adaptador usa `DesdeContexto` para recuperar ese ejecutor; no debe consultar el grupo de conexiones por su cuenta dentro del caso de uso.

Las funciones de `plataforma` leen el contexto y las políticas deniegan cuando falta. Las tablas con datos personales o de equipos llevan `ENABLE ROW LEVEL SECURITY` y `FORCE ROW LEVEL SECURITY`. Prueba tanto lectura como escritura, con contexto vacío y con equipos distintos. Una lista vacía en la web no demuestra que una escritura ajena esté bloqueada.

Las funciones `SECURITY DEFINER` necesitan propietario sin login, `search_path` fijo, retirada de `EXECUTE` a `PUBLIC` y comprobación explícita del ámbito. Revisa el conjunto de políticas: varias políticas permisivas se combinan y pueden ampliar una lectura. Para Consulta, RLS filtra filas, pero no oculta columnas; su contrato requiere una proyección propia.

Conserva claves compuestas cuando un hijo pertenece a expediente, equipo y sujeto. Un UUID válido de otro expediente no autoriza a vincularlo como adulto o menor. Esas relaciones se prueban en SQL además de validarse en Go.

## Historia, versiones e idempotencia

El expediente es la unidad de versión optimista para su ficha. Los cambios comparan la versión enviada y suben la versión al guardar. Un conflicto devuelve 409 y conserva el formulario para comparar. Algunas capacidades, como seguimiento y referencias activas, tienen además su propia versión.

Los datos históricos conservan vigencia y momento de conocimiento. Para un cálculo pasado, el lector debe seleccionar la revisión válida al corte y conocida en el instante requerido. Una corrección posterior no borra la revisión utilizada para una Memoria cerrada.

Las acciones que consumen número, confirman hitos o crean emisiones usan `Idempotency-Key`. Se conserva la respuesta asociada a usuario, operación y huella de petición durante el plazo del contrato. La misma clave y petición recuperan el recibo; cambiar el cuerpo con esa clave se rechaza. No implementes reintentos automáticos de escrituras en el adaptador PostgreSQL.

## Plan de migraciones

Cada carpeta contiene SQL de cuatro cifras y un `manifiesto.json`. El [lector del plan](../../internal/plataforma/migracion/plan.go) exige declarar todos los SQL y sus dependencias. El número crece dentro de cada módulo; los huecos reservados son válidos.

Una entrada real del [manifiesto de Estadística](../../migraciones/estadistica/manifiesto.json) es:

```json
{
  "numero": "0007",
  "archivo": "0007_pendientes_lista_espera.sql",
  "depende_de": [
    { "modulo": "estadistica", "numero": "0006" },
    { "modulo": "expediente", "numero": "0027" }
  ]
}
```

El ejecutor ordena el grafo, rechaza ciclos y dependencias ausentes, toma un bloqueo asesor y ejecuta una transacción por migración. Registra módulo, número, nombre, huella y autor en `plataforma.migracion_aplicada`. La huella incluye los bytes SQL, un byte cero y el JSON canónico de dependencias ordenadas. Cambiar un SQL aplicado, su nombre o sus dependencias rompe la comprobación del historial.

`migracion.retroceso` significa que se intenta aplicar un número aún no registrado por debajo del último aplicado de su módulo. No se resuelve quitando registros del historial. Reserva otro número y conserva la secuencia de despliegue acordada.

## Reserva y ensayo

Antes de crear el SQL, reserva el número en el registro de reservas de migraciones, fuera de Git, fuera de Git, indicando módulo, número, rama, descripción, fecha y estado `Reservada`. Usa el mecanismo de reserva indicado en el encargo. Nunca tomes una reserva ajena ni deduzcas que un hueco está libre porque no aparece un SQL en tu rama.

Después de integrar el `main` actual, comprueba que no haya aparecido un número superior al tuyo en el mismo módulo. Si ocurre, solicita al director un número nuevo y actualiza nombre, manifiesto y ensayo antes de entregar.

Una migración nueva comprueba el estado previo y añade permisos, RLS e integridad en el mismo fichero. No concede acceso a una tabla a medio preparar. Ensáyala en PostgreSQL 18 local, con instalación limpia y actualización desde el estado anterior, y con datos inventados. Requiere revisión independiente de SQL y permisos. Las correcciones se hacen con una migración posterior; la recuperación de un despliegue restaura su copia previa.

Los datos inventados de demo se siembran por comandos separados, no como datos personales dentro de una migración. Los códigos iniciales y reglas del catálogo sí pueden tener migraciones de datos.

Fuentes: [roles y permisos](../producto/ESQUEMA_BACKEND.md#3-roles-y-permisos-de-la-base-de-datos), [historia e idempotencia](../producto/ESQUEMA_BACKEND.md#4-historia-versiones-e-idempotencia), [migraciones](../producto/ESQUEMA_BACKEND.md#6-migraciones) y [ejecutor](../../internal/plataforma/migracion/ejecutor.go).
