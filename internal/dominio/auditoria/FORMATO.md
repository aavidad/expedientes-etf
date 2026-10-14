# Auditoría F0.08a

El evento conserva fecha, usuario interno, códigos de perfil, acción y entidad,
identificadores de entidad y expediente, resultado, ID y huellas. No admite
nombres, direcciones, contenido de fichas ni un detalle de texto libre. Los
identificadores y sus vínculos siguen protegidos; el DPD decide su conservación
y qué se puede exportar. Los códigos llegan de los módulos autorizados, nunca
de texto escrito por una persona. El usuario y los perfiles proceden del
contexto de confianza del servidor.

`Registro.Confirmar` asigna el ID y encadena las inserciones de una en una. Sólo
puede devolver éxito tras confirmar el evento. El caso de uso recibe los datos
ya reunidos en memoria y ejecuta la entrega después de esa confirmación. Si
falla el registro, no llama a la entrega. Si falla la entrega, conserva el evento
confirmado y devuelve la causa bajo una clave de error. El llamante registra ese
fallo aparte, después de revertir lo que proceda. Este caso de uso no autoriza
lecturas ni exportaciones y no sustituye la transacción conjunta de escritura.

## Forma canónica binaria v1

La huella es SHA-256 de los siguientes bytes, en este orden. Los enteros usan
8 bytes sin signo en orden big endian. Cada cadena es su longitud en bytes,
codificada como ese entero, seguida de los bytes UTF-8 de la cadena:

1. Huella anterior: 32 bytes binarios, decodificados del hexadecimal.
2. Cadena `etf.auditoria.v1`.
3. ID del evento, entero positivo hasta el máximo de un bigint con signo.
4. Fecha UTC como `AAAA-MM-DDTHH:MM:SS.ffffffZ`, con seis decimales. La entrada
   admite precisión hasta microsegundos; una precisión mayor se rechaza.
5. Usuario interno: UUID en hexadecimal minúsculo, o código `sistema`.
6. Número de perfiles y cada perfil como cadena, en el orden recibido.
7. Acción, entidad, identificador de entidad, identificador de expediente y
   resultado, cada uno como cadena. Un identificador opcional ausente es `""`.

La huella propia queda fuera del contenido firmado y se representa con 64
caracteres hexadecimales minúsculos. El origen es ID cero y 32 bytes cero.
Los ID crecen y pueden tener saltos. El orden se decide por ID, nunca por la
fecha del evento. Primero se validan los puntos del tramo. La verificación no
reordena: comprueba ID, enlace, datos,
huella propia y, al terminar, el sello del último evento. El error señala la
primera posición rota, empezando en cero. Una posición igual al número de
eventos indica un sello final que no coincide.

El vector de prueba independiente para el evento de `cadena_test.go` es:
`8fe3ef13875a0b5fbe10c10987cbde71df7cf1e261d79912dc4ba53bab99a2c6`.

## Exportación JSON y comprobador

`NuevaExportacion` recibe un tramo contiguo, el punto inmediatamente anterior
y el punto final esperado. El punto anterior no está en el tramo; el final sí.
Un tramo vacío conserva dos puntos iguales. No se exporta una selección de
eventos filtrada por expediente o usuario como si fuera una cadena completa.

El objeto JSON lleva `version: "etf.auditoria.v1"`, `algoritmo: "sha256"`,
`canonizacion: "etf.auditoria.binaria.v1"`, `orden: "id_ascendente"`,
`anterior`, `final` y `eventos`. Cada punto lleva `id` y `huella`. Los eventos
llevan los campos definidos en `evento.go`. Se rechazan documentos mayores de
64 MiB, claves desconocidas o duplicadas, claves en mayúsculas, valores null
y documentos concatenados. El espacio en blanco del JSON no cambia las huellas.
Si el tramo supera el límite, se divide en tramos contiguos con sus puntos.

Desde la raíz del repositorio:

```sh
go build -o /ruta/autorizada/comprobar-auditoria ./cmd/comprobar-auditoria
/ruta/autorizada/comprobar-auditoria /ruta/autorizada/auditoria.json
/ruta/autorizada/comprobar-auditoria /ruta/autorizada/auditoria.json ID_SELLO HUELLA_SELLO
/ruta/autorizada/comprobar-auditoria /ruta/autorizada/auditoria.json ID_ANTERIOR HUELLA_ANTERIOR ID_SELLO HUELLA_SELLO
```

El comprobador sólo lee un fichero regular y devuelve JSON con claves y, cuando
hay un evento roto, su posición e ID. Códigos de salida: 0 para cadena coherente
o sello externo coincidente, 1 para contenido o sello que no pasa la comprobación
y 2 para uso, fichero, sello de entrada o salida del comando inválidos. No
imprime rutas ni contenido del fichero ni abre conexiones.

Los puntos incluidos en el fichero permiten detectar alteración, borrado
(también en los extremos), inserción e inversión. Quien pueda recalcular los
eventos y cambiar ambos puntos puede fabricar otra cadena coherente. Para
contrastar el origen se pasa el sello final custodiado en una copia independiente;
no se toma ese sello del mismo fichero. Con dos argumentos después del fichero,
el comprobador exige que la cadena parta del origen (ID cero y huella inicial).
Para un tramo parcial se pasan los dos puntos esperados, anterior y final,
custodiados por separado. Esto detecta también que se haya eliminado un prefijo
y cambiado el punto anterior dejando intacto el sello final. La coincidencia
acredita el tramo contra esos puntos, con la confianza que tenga su custodia.

F0.08b implementa persistencia en `migraciones/auditoria/0001`, con bloqueo de
transacción antes de reservar el ID, sellos diarios y exportación para el perfil
`administracion`. `RegistroAuditoria.Confirmar` confirma su propia transacción;
`RegistrarEnTransaccion` participa en la transacción de escritura del llamante.
Una exportación registra su acto y se confirma antes de devolver el documento.
Los tramos SQL usan límites de ID inclusivos; admiten saltos por rollback. La
verificación SQL devuelve NULL si no hay fallo, o el primer ID roto.

El actor `sistema` se persiste con `actor_sistema = true` y UUID nulo; una persona
lleva UUID y el indicador falso. Una restricción impide las otras combinaciones.
La representación canónica y exportada sigue siendo la cadena `sistema` o el
UUID: las huellas v1 no cambian. Actor y perfiles se leen del contexto de sesión,
no de parámetros SQL. Un contexto vacío representa tareas automáticas y accesos
sin identidad; puede verificar y sellar, pero no exportar.

`registrado_en` lo fija la base bajo el bloqueo y delimita el día UTC del sello,
independientemente de la fecha `ocurrido_en` del acto. Sólo se sellan días ya
terminados. Se conserva el último ID confirmado hasta ese día, o el punto cero
si no había eventos; repetir devuelve el mismo sello. El adaptador devuelve el
punto confirmado para su custodia fuera del servidor. La programación nocturna,
registro técnico, copia independiente, pantalla y HTTP quedan pendientes de
integración. El ensayo local es `python3 -B migraciones/auditoria/ensayar.py`.
El verificador comprueba también que cada sello fija el último ID anterior al
cierre UTC. `registrado_en` es metadato técnico fuera de la forma canónica v1:
esa comprobación protege la frontera sellada, no acredita su hora exacta.

Tampoco se incluye aún `detalle` (campos cambiados, claves de error y número de
filas). Ampliarlo exige otra versión canónica, sin reescribir eventos.
