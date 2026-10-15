## Anexo C. Resolución de problemas

Autor: Expedientes ETF. Revisión: 14 de octubre de 2026.

[Volver al manual](LEEME.md). Anota hora, versión, servicio y clave. Comprueba
`componer ps -a`, salud y registros acotados antes de reiniciar. Estos textos
proceden de los catálogos en castellano; las claves son las que devuelve o
registra el código. No adjuntes datos personales ni credenciales a soporte.

### C.1 Acceso y sesión

| Clave y mensaje real | Comprobación y resolución |
| --- | --- |
| `identidad.configuracion_invalida`: «El acceso aún no está disponible. Contacta con Informática.» | Revisar que el contenedor recibe origen, CA PEM, URL LDAPS, DN, formato de bind y duración explícita de sesión. Consultar el [anexo de configuración](CONFIGURACION.md); una variable sólo en el archivo externo no llega al contenedor sin `environment` |
| `identidad.directorio_no_disponible`: «El servicio de acceso de la entidad no responde. No puedes entrar ahora. Prueba más tarde.» | Comprobar DNS, salida privada del contenedor a 636, certificado, hora y disponibilidad del controlador. No cambiar a LDAP sin cifrar ni crear una entrada local alternativa |
| `identidad.credenciales_invalidas`: «No se ha podido entrar con ese usuario y contraseña. Compruébalos y vuelve a intentarlo.» | Confirmar login permitido y contraseña sin registrarla. Si una cuenta conocida falla, revisar con Informática formato de bind y lectura de `objectGUID`. No conceder perfiles para resolver credenciales |
| `identidad.intentos_limitados`: «Se han recibido demasiados intentos de acceso. Espera unos minutos y vuelve a intentarlo.» | No repetir intentos automatizados. El límite de fallos es cinco por usuario y treinta por IP en quince minutos; comprobar que proxy sustituye la cabecera y está declarado en `ETF_HTTP_PROXIES` |
| `identidad.sesion_no_valida`: «Tu sesión ha caducado. Vuelve a entrar.» | Entrar de nuevo. La sesión vence a los treinta minutos sin actividad o al máximo absoluto configurado. Si ocurre tras restaurar, la revocación es parte de la recuperación |
| `identidad.csrf_no_valido`: «No se ha podido confirmar la petición. Recarga la página y vuelve a intentarlo.» | Recargar y revisar origen y cabeceras del proxy. Un reinicio cambia la clave CSRF. Recuperar el resultado de operaciones idempotentes antes de repetir una creación |
| Pantalla de pendiente de alta | AD aceptó la cuenta, pero falta perfil. Administración funcional asigna perfil y equipo. Informática no corrige esto reiniciando ni editando permisos SQL |

Si una modificación da HTTP 409, otra operación cambió la versión. Actualiza y
revisa; no repitas con una versión antigua. Si da 403, comprueba perfil vigente
y equipo con Administración. El acceso de una persona de equipo a otro equipo
está denegado por diseño.

### C.2 Base, migraciones y preparación de roles

| Registro o clave real | Comprobación y resolución |
| --- | --- |
| `configuracion.reserva`, con nombre de variable | La configuración usa una reserva. Consultar la variable nombrada, sin imprimir su valor secreto |
| `postgres.configuracion_invalida` | En app revisar usuario `etf_app`, contraseña no vacía, host, puerto, máximo de conexiones y modo TLS. En preparación de roles puede indicar membresía inesperada; detener actualización y revisar con desarrollo, sin retirar membresías a ciegas |
| `postgres.no_disponible` o `/salud` con `base: no_disponible` | Revisar PostgreSQL, disco, red privada y autenticación. No borrar el volumen ni sembrar de nuevo. Si hay recuperación, seguir §5 del manual |
| `migracion.configuracion` | El migrador necesita host, puerto, base y su contraseña propia. No darle credenciales de app ni instalador |
| `migracion.postgres` | Revisar conexión autenticada y salida acotada del servicio. Un `pg_isready` correcto no demuestra que el migrador pueda entrar |
| `migracion.rol` | Se intentó ejecutar con otro usuario. Usar servicio `migrar` y `etf_migrador`; no ejecutar SQL manualmente |
| `migracion.huella` | Historial y artefacto no coinciden. Conservar copia y versión y avisar a desarrollo. No editar la huella ni el SQL ya aplicado |
| `migracion.retroceso` | Falta una migración de número inferior a otra ya aplicada en su módulo. Detener despliegue y solicitar una entrega con orden corregido. No borrar historial ni renumerar la base |
| `migracion.dependencia`, `migracion.historial` o `migracion.estado_previo` | Estado o dependencias incompatibles con la entrega. Conservar salida y artefacto, comprobar copia y consultar desarrollo. Cada paso confirma su transacción: un fallo no deshace los anteriores |
| `migracion.aplicada.<modulo>.<numero>` | Paso aplicado. Verificar salida final 0 y salud antes de abrir |
| `migracion.existente.<modulo>.<numero>` | Paso ya aplicado y compatible. No es un error ni exige repetir SQL |

`preparar-roles` se ejecuta antes de `migrar`, incluso con volumen antiguo. Si
su salida no es 0, no arranques migración ni app para saltar esa puerta. Verifica
socket temporal, UID/GID 70:70 y contraseñas del archivo privado. No uses
`componer run --no-deps migrar`: omite la preparación necesaria. La [evidencia
FX2](FX2.md) detalla instalación limpia y actualización de roles anteriores.

Si Docker rechaza publicar el puerto local porque ya está ocupado,
comprueba qué servicio lo usa. Elige un puerto libre y ajusta proxy y comprobaciones; para una
demo nueva puedes usar `ETF_DEMO_PUERTO`. Un solapamiento de subred Docker se
resuelve con rangos privados libres, sin parar servicios de otros proyectos.

### C.3 Antivirus, paquetes y documentos

Los anexos y las actas de carga histórica admiten DOCX, ODT y PDF, hasta 10 MB.
Se leen sin conservar el archivo y una persona confirma los datos. La aplicación
no valida la autenticidad de la firma de un PDF ni extrae texto de un escaneado.
Un documento sin texto legible requiere completar los datos a mano.

| Clave y mensaje real | Comprobación y resolución |
| --- | --- |
| `antivirus.configuracion_invalida`: «La revisión antivirus no está disponible. Contacta con Informática.» | Revisar dirección `clamav:3310` y configuración; no publicar 3310 al exterior |
| `antivirus.no_disponible`: «No se ha podido revisar el archivo con el antivirus. Inténtalo de nuevo.» | Comprobar daemon, memoria, `PONG` y firmas. El anexo admite continuar con aviso según la decisión vigente; paquetes, plantillas y hojas requieren revisión satisfactoria |
| `paquete.antivirus_no_disponible`: «No se ha podido completar la revisión antivirus. El paquete no se ha activado. Inténtalo de nuevo.» | Mantener activa la versión anterior; restablecer antivirus antes de subir de nuevo |
| `paquete.infectado`: «El antivirus ha detectado contenido peligroso. El paquete no se puede activar.» | Rechazar el archivo y seguir el procedimiento de incidente. No desactivar antivirus ni instalarlo a mano |
| `paquete.archivo_invalido`: «El paquete no cumple los requisitos de formato, contenido o tamaño. Revisa el archivo antes de volver a subirlo.» | Pedir a Administración un paquete válido del tipo admitido; no ampliar límites para aceptar formatos peligrosos |
| `idiomas.no_disponibles` o `tema.no_disponible` | Revisar rutas y permisos del UID 10001, referencia activa y archivos. Recuperar el paquete correspondiente; una reserva de imagen no recupera versiones administradas |
| `paquete.limpieza_pendiente` | La limpieza de archivos huérfanos no pudo completarse. Revisar almacenamiento y permisos; no borrar versiones referenciadas para silenciarlo |
| `documento.plantilla_no_disponible`: «No hay una plantilla válida para este documento en el idioma elegido.» | Administración revisa tipo, idioma, vigencia y versión activa. Informática comprueba archivos y referencias de la misma copia |
| `documento.plantilla_alterada`: «La plantilla ha cambiado sin publicar una versión nueva. Avise a Administración.» | Conservar evidencia y recuperar la versión íntegra o publicar otra desde la app. No actualizar la huella en la base |
| `documento.firmante_no_disponible`: «No hay un firmante vigente configurado para este documento.» | Administración revisa el firmante y su vigencia; no es un fallo de Docker |
| `documento.reproduccion_no_disponible`: «No se puede reproducir exactamente la descarga anterior. Se conserva el recibo de emisión. Solicite una emisión nueva.» | Comprobar historial y plantilla necesaria. Si ya no es reproducible, la persona solicita nueva emisión; no sustituir silenciosamente datos o versión |

Si tras una subida nada aparece en Administración, comprueba primero respuesta,
versión activa y permisos de publicación. El montaje base es de sólo lectura;
la publicación en producción necesita el almacenamiento del anexo A. No cambies
permisos del repositorio entero.

Si Docker marca ClamAV saludable pero la firma es antigua, repón el suministro
local aprobado. Cambiar imagen no reemplaza el volumen de firmas existente.
No elimines ese volumen en producción sin tener preparada su reposición.

### C.4 Auditoría, conservación y recuperación

| Clave o mensaje real | Actuación |
| --- | --- |
| `auditoria.cadena_coherente` | Comprobación interna correcta; falta el sello externo para acreditar el punto custodiado |
| `auditoria.sello_externo_coincide` | Cadena y sello independiente coinciden; conservar resultado y exportación |
| `auditoria.sello_externo_no_coincide` o `auditoria.ancla_externa_no_coincide` | No abrir tras restauración sin aclararlo. Verificar tramo, sello y copia correctos; preservar archivos y escalar, sin regenerar el sello desde el JSON |
| `conservacion.conflicto`: «El expediente o el plazo ha cambiado. Actualiza la lista y revisa de nuevo.» | Actualizar lista; la revisión anterior puede haber quedado invalidada |
| `conservacion.no_permitida`: «No se cumplen las condiciones para confirmar. Actualiza la lista y revisa el estado.» | Comprobar plazo cumplido, estado cerrado, seguimiento terminado y segunda persona distinta |
| `conservacion.sin_permiso`: «Tu sesión no permite esta acción. Vuelve a entrar con un perfil de Administración.» | Administración realiza la operación autorizada; Informática no fuerza SQL |

Una restauración que devuelve el acceso pero pierde plantillas está incompleta.
Mantén la entrada cerrada y recupera base y paquetes del mismo conjunto. Tras
una copia antigua, revoca sesiones y reconcilia permisos, bajas, traslados y
retiradas de datos posteriores. No borres la auditoría ni repitas la siembra.

Si el disco se llena o un contenedor reinicia por falta de memoria, para escrituras,
conserva registros y amplía capacidad o retira sólo copias y logs según el plazo
aprobado. No borres archivos del volumen de base. «Servicio no disponible» puede
ser consecuencia de una auditoría no confirmada, aunque PostgreSQL acepte conexiones;
revisa logs y permisos antes de reiniciar.

En Windows, sigue [la guía del paquete](../../despliegue/windows/LEEME.md): comprueba
motor WSL2 activo, suma de paquete y puerto guardado. No uses «Reiniciar» para
resolver una parada si quieres conservar datos. El paquete no convierte el PC
en un servidor de producción.
