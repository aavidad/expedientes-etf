# Demostración local de Expedientes ETF

Este entorno sirve para probar la aplicación en este equipo. Todas las personas
son inventadas. No lo uses con expedientes reales ni para instalar producción.

Necesitas Docker en marcha con Compose 2.24.4 o posterior, Go 1.25, Node con npm,
Python 3, curl, OpenSSL y Chrome del sistema (`/usr/bin/google-chrome`). El puerto
18083 debe estar libre. Si está ocupado, elige otro al arrancar por primera vez:
`ETF_DEMO_PUERTO=18084 make demo`. Esta demostración no detiene otras aplicaciones.
El puerto elegido se conserva para los siguientes comandos. Cada worktree usa dos subredes privadas propias para poder arrancar demostraciones en paralelo.

Cada worktree recibe también una subred privada propia. Si coincide con una
red ya ocupada, elige un prefijo libre con `ETF_DEMO_RED=10.253.11 make demo`.
Se conserva en la configuración local. Cambiar puerto o red después de preparar
la demostración requiere `make demo-limpiar`, que borra sus datos inventados.

## Arrancar y entrar

Desde la raíz de este repositorio, ejecuta:

```sh
make demo
```

Espera a que aparezca «Demostración disponible». Abre
<http://127.0.0.1:18083> en Chrome de este equipo. Usa exactamente esta dirección;
si elegiste otro puerto, usa la dirección que muestra la consola. El acceso está
preparado para esa dirección. El primer arranque compila la aplicación y
puede tardar unos minutos. Necesita las dependencias e imágenes fijadas por el
repositorio; su descarga inicial requiere conexión.

La consola muestra los usuarios y sus contraseñas. También quedan en
`despliegue/demo/USUARIOS.local.md`, que Git ignora. Hay siete personas:

| Usuario | Acceso inicial |
| --- | --- |
| `elena.salas` | Administración funcional; primera administradora |
| `ines.moreno` | Coordinación |
| `javier.medina` | Coordinación |
| `marta.cabrera` | Profesional del ETF 2 |
| `pablo.ruiz` | Profesional del ETF 4 |
| `lucia.navarro` | Consulta |
| `sergio.campos` | Pendiente de alta; sin perfil |

Las contraseñas se generan al preparar el primer arranque. Se conservan junto
con los datos para que puedas continuar después de parar. Repetir `make demo`
no crea usuarios duplicados ni vuelve a conceder permisos retirados.

## Qué está disponible

Puedes entrar con cada perfil, ver su menú y gestionar usuarios, catálogos,
firmantes y paquetes desde Administración. La API permite subir un Anexo 3,
aceptar la derivación y descargar el Acta de Aceptación. Las pantallas de
coordinación permiten revisar y resolver las derivaciones. La bandeja del equipo
y la ficha permiten consultar los expedientes, preparar salidas y proponer su
cierre. El recorrido automático comprueba estas operaciones desde Chrome.

El arranque prepara la base de datos y carga los catálogos existentes: ocho
equipos, centros, municipios y las listas ya confirmadas. Después entra con
la primera administradora y da de alta los otros perfiles mediante la misma
aplicación; el profesional recibe su equipo y estos cambios quedan registrados.
Sergio conserva el estado «pendiente de alta». La primera administradora se
concede una sola vez mediante `ETF_PRIMER_ADMINISTRADOR`; si retiras su perfil,
reiniciar no lo recupera.

La siembra también registra a Inés Moreno Castillo como firmante de aceptación,
con vigencia desde el 1 de enero de 2026, mediante la API de Administración. El
cargo se publica en los dos idiomas. La plantilla de Acta se carga al migrar y
está vigente desde el 13 de octubre de 2026. Las modificaciones de idiomas y
temas quedan en volúmenes propios de la demostración.

Las 64 familias inventadas de [datos/prueba](../../datos/prueba/LEEME.md) **no se
cargan**: esta rama todavía no tiene una orden que las incorpore a la base de
datos.

## Parar, comprobar o empezar de nuevo

`make recorridos` ejecuta las [pruebas de entrada, alta y permisos
F0.11](../../docs/recorridos/F0.11.md) contra esta demostración, con puerto libre
cuando aún no hay uno guardado y evidencia fuera de Git.

Para parar la demostración y conservar lo que hayas cambiado:

```sh
make demo-parar
```

Para volver a entrar, ejecuta otra vez `make demo`. Para comprobar los siete
accesos iniciales con la aplicación arrancada:

```sh
make demo-comprobar
```

Esta comprobación exige los perfiles iniciales de la tabla. Si los cambias para
probar la pantalla de Administración, puede fallar; la parada conserva esos
cambios. También comprueba equipos, usuarios únicos y rechazo de peticiones
procedentes de otra dirección.

Para comprobar la subida, la aceptación y la descarga del Acta:

```sh
make demo-recorrido
```

Este recorrido entra con Elena y con Inés en Chrome del sistema. Sube
`internal/adaptadores/anexos/docx/testdata/anexo3.docx`, comprueba el antivirus,
confirma los datos revisados, el municipio y su centro y equipo del catálogo,
selecciona el puesto de trabajo social y el
sexo de cada menor, y
acepta con fecha del 13 de octubre de 2026. Descarga el Acta en memoria, comprueba
su contenido y su huella, y repite aceptación y emisión con las mismas claves
para comprobar que no se duplican. Cada ejecución crea una nueva derivación de
prueba; necesita los perfiles iniciales. El Word y el anexo no se guardan.

Para borrar **sólo los datos inventados de esta demostración**, sus contraseñas
y su certificado, incluidos los volúmenes de idiomas y temas, y preparar
después una demostración nueva:

```sh
make demo-limpiar
make demo
```

Si actualizas la rama con nuevas plantillas o textos de distribución, usa también
`make demo-limpiar` antes de arrancar. Los volúmenes conservados mantienen sus
propias versiones y no se sustituyen al reconstruir la imagen.

Si el arranque falla, puedes volver a ejecutar `make demo`. Si la preparación de
usuarios quedó incompleta y cambiaste sus permisos, revisa los usuarios o empieza de nuevo con
`make demo-limpiar`.

## Separación del entorno habitual

La demostración combina [compose.yaml](../../compose.yaml) con
[compose.demo.yaml](compose.demo.yaml). `make demo` utiliza un nombre propio de
instalación, redes y almacenamiento separados, y no lee ni cambia `.env`.
Cada arranque conserva las credenciales existentes y añade la de cálculo de
Memoria si procede de una versión antigua. Compose prepara automáticamente los
roles antes de migrar, sin borrar el volumen ni ejecutar habilitaciones manuales.
La aplicación publica únicamente el puerto local elegido. El directorio de
pruebas y la base de datos no publican puertos ni tienen salida a Internet.
Las redes reservadas son `10.250.54.0/28` y `10.250.54.16/28`; si coinciden con
otra red del equipo, cambia únicamente este archivo de demostración.

El directorio de pruebas usa una conexión cifrada y un certificado creado en
este equipo, válido durante un año. Los ficheros locales están excluidos de Git
y su carpeta está protegida para el usuario del equipo. Sólo se monta en la
aplicación el certificado público; la clave privada queda en el directorio de
pruebas. Para renovar todo, usa `make demo-limpiar` y `make demo`.

La entrada local adapta únicamente la dirección exacta de esta demostración
a la dirección segura que exige la aplicación. Conserva la protección de las
cookies y rechaza las peticiones de otras direcciones. Este acceso por HTTP sólo
sirve desde este equipo. `ETF_MODO=desarrollo` aparece únicamente en el archivo
de demostración; [el arranque habitual](../base/LEEME.md) conserva su configuración.
La aplicación recibe sólo la contraseña de su propia cuenta de base de datos.
`make demo-parar` elimina todos los contenedores y las redes de la demostración
y libera su puerto; conserva sus volúmenes y los secretos locales para continuar.
`make demo-limpiar` elimina también esos datos y secretos.

## Bandeja del equipo

La siembra añade dos expedientes aceptados del ETF2 y uno del ETF4 cuando Marta Cabrera no tiene expedientes aceptados. Una propuesta de cierre permite comprobar el aviso pendiente. Los anexos son inventados y se eliminan de la carpeta temporal tras su lectura. La asignación deja sin identificar a la persona de referencia, un dato opcional; conserva el puesto de Trabajo Social.

El recorrido `node despliegue/demo/equipo-recorrido.mjs http://127.0.0.1:18083` usa Chrome del sistema, entra con Coordinación y con Marta, comprueba permisos, búsqueda, ficha y cierre, y guarda las capturas solicitadas en `docs/capturas/`. El ensayo SQL se ejecuta con `python3 -B migraciones/expediente/ensayar_bandeja.py` y destruye su contenedor al terminar.

Cada instalación nueva calcula sus subredes locales a partir de la ruta del worktree. Si coinciden con otra red de Docker, puedes fijar `ETF_DEMO_SUBRED_INTERNA` y `ETF_DEMO_SUBRED_ENTRADA` en `despliegue/demo/.local/entorno.env`; las redes deben ser distintas y estar libres. Esto permite mantener varias demostraciones sin ocupar las redes de otra carpeta.

Las redes de una demo se pueden separar de las de otra instalación: añade `ETF_DEMO_RED_SUBRED` y `ETF_DEMO_ENTRADA_SUBRED` a `.local/entorno.env` con dos subredes libres. Los valores por defecto se mantienen si no las defines.

La bandeja de Coordinación se carga con 24 derivaciones inventadas si no hay ninguna: pendientes de revisar, una aceptada, una no aceptada y una en espera. La siembra usa la API con el perfil de Coordinación y permite recorrer dos páginas de pendientes. No añade derivaciones si ya hay datos.

Al recompilar, `make demo` reinicia sólo el proxy de esta demostración para que vuelva a resolver la dirección del contenedor de la aplicación.

## Tabla SPAF

La siembra histórica añade cuatro expedientes inventados. En septiembre de 2026, el ETF 02 tiene 2 altas, 2 bajas y 3 casos; el ETF 01 tiene 1 alta. Uno de los casos del ETF 02 se acepta y cierra durante ese mes. Entra con Inés Moreno para elegir equipo, o con Marta Cabrera para consultar sólo el ETF 02. La carga conserva números y fechas por la API de alta histórica, sin generar hitos ficticios. Los datos de ficha que el lector todavía no recoge se muestran como «Sin dato».

La demo prepara primero los expedientes de la bandeja de equipo y después los históricos de SPAF. Así conserva la propuesta de cierre del recorrido de equipo y los movimientos de septiembre de 2026. Las dos siembras usan sus propias marcas para evitar repetir altas.
