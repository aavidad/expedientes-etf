# Preparar el aula

Esta lista prepara una demo local con datos inventados. Para un entorno de formación compartido, acuerda con Informática el acceso y las cuentas; la demo local escucha sólo en este equipo y no sustituye esa instalación.

## Antes de la sesión

- [ ] Reserva 150 minutos y comprueba los perfiles de quienes van a asistir. Imprime o comparte ejercicios, sin las soluciones al principio.
- [ ] Comprueba proyector, tamaño de letra, sonido y reproducción del MP4 «Del Anexo a la Memoria». Los vídeos están en la carpeta de vídeos configurada (`ETF_VIDEOS_SALIDA`, por defecto `~/Videos/Expedientes-ETF/`); no necesitan red al reproducirse.
- [ ] Prepara Chrome del sistema. Evita que dos personas guarden el mismo apartado a la vez.
- [ ] En el equipo que alojará la demo, comprueba Docker, Go, Node, npm y Python según [la guía de demostración](../../despliegue/demo/LEEME.md). Prepara las dependencias fijadas antes de la sesión.
- [ ] Desde la raíz del worktree destinado al aula, ejecuta `ETF_VIDEO_DEMO=1 ETF_DEMO_PUERTO=18507 python3 -B scripts/recorridos/reloj_video.py iniciar`. Prepara usuarios y un calendario de negocio simulado; las sesiones mantienen el tiempo real. Si ya existe la demo, conserva su puerto. Cada ejecución conserva casos anteriores.
- [ ] Abre el aula con `ETF_VIDEO_DEMO=1 node scripts/recorridos/aula.mjs` en otra terminal. El lanzador abre tres ventanas con sesiones separadas: una para Javier e Inés, otra para Marta y otra para Elena. Las dos primeras reciben el calendario de formación; la de Elena conserva el tiempo real para activar permisos de uso inmediato. Usa esas ventanas para practicar. Los Word se guardan en `despliegue/demo/.local/descargas-formacion/`; copia los que quieras conservar antes de limpiar la demo.
- [ ] Anota la dirección que muestra «Demostración de formación disponible» y comprueba `make demo-comprobar`. Abre esa dirección en el mismo equipo; no cambies a una dirección de red para otros puestos.
- [ ] Consulta `despliegue/demo/USUARIOS.local.md` en local para repartir contraseñas. No las copies a documentos compartidos, capturas, vídeo ni Git. No proyectes la consola que las muestra.
- [ ] Comprueba Inés Moreno (`ines.moreno`), Coordinación; Marta Cabrera (`marta.cabrera`), ETF 02; Elena Salas (`elena.salas`), Administración; y Sergio Campos (`sergio.campos`), pendiente de alta. Javier Medina (`javier.medina`) recibe y cierra el caso del vídeo; Pablo Ruiz (`pablo.ruiz`) permite comprobar ETF 04; Lucía Navarro, consulta de cifras.
- [ ] Comprueba que Sergio no conserve el permiso de otra práctica. Administración retira su perfil y equipo si procede; verifica el acceso con una sesión nueva.
- [ ] Copia a la carpeta de prácticas el [Anexo 3 inventado](anexo3-ortega-salcedo.docx), sin sustituirlo por documentos recibidos de familias reales. Comprueba el antivirus antes de la clase.
- [ ] Prepara una hoja para anotar el número asignado a cada pareja. Las iniciales se repiten si varias parejas suben el mismo Anexo.
- [ ] Consulta las Memorias de equipo 02 y provincial de 2026. Si están cerradas, Coordinación las reabre con el motivo «Preparación de una nueva sesión de formación con datos inventados»; conserva sus revisiones anteriores. Si necesitas consultar el equipo 02 como Coordinación, selecciónalo en el filtro antes de reabrir.
- [ ] Reserva una carpeta local para los Word descargados. Decide quién conservará los ejercicios y durante cuánto tiempo.

## Datos y resultados de la práctica

Usa octubre de 2026 y las fechas de [los ejercicios](ejercicios.md), con el equipo 02 y los seis menores del Anexo. Antes de empezar, anota los totales de septiembre; al avanzar a noviembre, anota también los de octubre para distinguir el incremento de las nuevas familias del total acumulado. Cada pareja crea un expediente nuevo; repetir `make demo` conserva los anteriores y no devuelve los datos al inicio.

El vídeo 11 también conserva sus casos y cierra las Memorias. Si lo grabas antes de impartir, reabre las revisiones con motivo antes de los ejercicios. La pareja responsable de la Memoria cierra una sola revisión para el grupo; evita cierres simultáneos desde varias cuentas.

## Avanzar el calendario durante la práctica

El reloj de formación pertenece sólo a la compilación local preparada por el script. No modifica la hora del equipo ni los ficheros Go de la aplicación: compila un overlay temporal que sustituye los puertos de reloj de negocio. No se usa con datos reales. Quien imparte avanza la fecha cuando todas las parejas han terminado el paso anterior; los participantes usan las fechas de los ejercicios.

Desde la raíz del worktree, ejecuta `ETF_VIDEO_DEMO=1 python3 -B scripts/recorridos/reloj_video.py fijar FECHA`, sustituyendo `FECHA` por la fila que corresponda:

| Antes de | Fecha |
| --- | --- |
| Recibir, aceptar y descargar el Acta | 2026-10-14 |
| Familia, Menores, Intervención y descarga del Anexo 1 | 2026-10-15 |
| Confirmar el consentimiento firmado | 2026-10-16 |
| Registrar la evaluación | 2026-10-17 |
| Registrar la comisión | 2026-10-22 |
| Registrar el informe y consultar septiembre en SPAF | 2026-10-24 |
| Finalización, propuesta, cierre y Acta | 2026-10-31 |
| Consultar octubre en SPAF y revisar/cerrar Memoria | 2026-11-01 |

Si alguien sigue recibiendo la derivación, espera antes de avanzar desde el día 14: la fecha de recepción debe preceder a la aceptación. Genera el Anexo 1 el día 15 y confirma la firma el día 16. Así los documentos y los hitos aparecen en el orden que se está explicando.

Las ventanas de equipos y Coordinación simulan la misma fecha que la demo sin cambiar la hora del equipo. La de Administración conserva la fecha ordinaria, como el servicio de identidad. Al avanzar con `fijar`, las ventanas se recargan: espera a que todas las parejas hayan guardado y vuelve a abrir el número anotado si hace falta. Un Chrome abierto fuera del lanzador conserva su fecha ordinaria y puede impedir las fechas de la práctica; usa las ventanas del aula. Las consultas de vigencia de familia, menores e intervención usan también la fecha de formación en ese overlay, manteniendo los filtros de acceso y las reglas de guardado. Para poder repetir la práctica tras reinstalar la demo, la lectura del catálogo inicial toma como publicación su inicio de vigencia. Esa proyección sólo existe en el overlay; conserva las publicaciones posteriores y no escribe ni altera las versiones del catálogo en la base. El aula y el vídeo fijan la fecha del navegador mediante Playwright, manteniendo los temporizadores en tiempo real. Para comprobarlo antes de la clase, usa `ETF_VIDEO_DEMO=1 node scripts/recorridos/aula.mjs --comprobar`: verifica fecha, mes permitido, consulta real, separación de sesiones y reloj ordinario de Administración. Al terminar, `ETF_VIDEO_DEMO=1 python3 -B scripts/recorridos/reloj_video.py restaurar` recompila la versión ordinaria y conserva los datos de práctica. Si una grabación se interrumpe, usa también ese comando para volver al reloj ordinario.

## Reiniciar sólo la demo del aula

`make demo-limpiar` borra los datos inventados y las credenciales de la demostración de este worktree. Comprueba la carpeta con `pwd` y guarda primero las prácticas que quieras conservar. No ejecutes ese comando en un entorno con datos reales ni desde una carpeta de otra instalación.

Desde la raíz del worktree del aula:

```sh
make demo-parar
make demo-limpiar
ETF_VIDEO_DEMO=1 ETF_DEMO_PUERTO=18507 python3 -B scripts/recorridos/reloj_video.py iniciar
make demo-comprobar
```

El reinicio genera contraseñas nuevas: reparte las nuevas y comprueba otra vez los perfiles. Recupera el Anexo de prácticas y vuelve a anotar los totales iniciales. Los vídeos fuera del repositorio permanecen; no necesitas volver a grabarlos para impartir la sesión.

Si quieres continuar otro día con los mismos datos, para la demo sin limpiarla. Vuelve a iniciar el calendario con `reloj_video.py iniciar` y avanza con `fijar` a la fecha del paso que retomas. Los números, cambios y contraseñas se conservan.

## Al terminar

- [ ] Comprueba que los Word aparecen en la carpeta de descargas antes de cerrar las ventanas. Cierra las sesiones y las ventanas que muestren credenciales.
- [ ] Retira el permiso concedido a Sergio si vas a conservar la demo para otra clase.
- [ ] Recoge los números de práctica y las dudas, sin contraseñas ni datos reales.
- [ ] Comprueba qué Memorias quedaron cerradas y anota si deben reabrirse para la siguiente sesión.
- [ ] Restaura la compilación ordinaria con `ETF_VIDEO_DEMO=1 python3 -B scripts/recorridos/reloj_video.py restaurar`. Conserva o elimina los Word inventados según lo acordado y para la demo cuando no se use.
