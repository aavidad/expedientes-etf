## Anexo B. Copias, auditoría y conservación

Autor: Expedientes ETF. Revisión: 14 de octubre de 2026.

[Volver al manual](LEEME.md). Usa la función `componer` definida allí. La copia
incluye base y archivos de paquetes del mismo momento, con app parada.

### A. Cifrar y comprobar una copia

Informática decide el destino fuera del servidor y la herramienta corporativa.
Si utiliza GnuPG local, este procedimiento cifra un conjunto ya tomado según §5
del manual. Requiere `gpg`, `tar` y `sha256sum`. El directorio de trabajo debe
estar en almacenamiento cifrado, con permisos privados: hasta cifrarlo contiene
datos en claro. La clave de copia queda en `/etc/etf/copia.clave`, con permiso
0600, fuera de Git y fuera de la copia. Custodia una forma de recuperarla por
separado; perder esa clave impide restaurar.

1. Para app y cierra la entrada en el proxy. Toma `etf.dump` y comprueba su índice
   con las órdenes del manual. Copia los paquetes completos mientras sigue parada.
2. Para los montajes del anexo A, crea `paquetes.tar` y el inventario privado:

```sh
sudo tar -C /srv/etf/paquetes -cpf "$ETF_COPIA_DIR/paquetes.tar" idiomas temas
git rev-parse HEAD > "$ETF_COPIA_DIR/version.txt"
componer images --format json > "$ETF_COPIA_DIR/imagenes.json"
(
  cd "$ETF_COPIA_DIR"
  sha256sum etf.dump paquetes.tar version.txt imagenes.json > SHA256SUMS
)
```

3. Guarda también la configuración externa necesaria en su custodia cifrada
   autorizada. Los secretos y claves privadas tienen custodia separada. Conserva
   el sello de auditoría independiente y el inventario de bajas y equipos que
   necesitarás reconciliar tras restaurar.
4. Cifra el conjunto. `$ETF_DESTINO_COPIAS` debe ser el destino aprobado fuera
   del servidor, montado y accesible; usa una ruta nueva para cada copia. Estas
   variables pertenecen al procedimiento, no configuran app. Activa parada por
   error en la sesión de operación antes de ejecutar:

```sh
set -e
umask 077
tar -C "$ETF_COPIA_DIR" -cf "$ETF_COPIA_DIR/conjunto.tar" \
  etf.dump paquetes.tar version.txt imagenes.json SHA256SUMS
gpg --batch --pinentry-mode loopback \
  --passphrase-file /etc/etf/copia.clave --symmetric --cipher-algo AES256 \
  --output "$ETF_COPIA_DIR/conjunto.tar.gpg" "$ETF_COPIA_DIR/conjunto.tar"
(
  cd "$ETF_COPIA_DIR"
  sha256sum conjunto.tar.gpg > conjunto.tar.gpg.sha256
)
install -d -m 0700 "$ETF_DESTINO_COPIAS"
cp "$ETF_COPIA_DIR/conjunto.tar.gpg" "$ETF_DESTINO_COPIAS/conjunto.tar.gpg.part"
mv "$ETF_DESTINO_COPIAS/conjunto.tar.gpg.part" "$ETF_DESTINO_COPIAS/conjunto.tar.gpg"
cp "$ETF_COPIA_DIR/conjunto.tar.gpg.sha256" "$ETF_DESTINO_COPIAS/"
(
  cd "$ETF_DESTINO_COPIAS"
  sha256sum -c conjunto.tar.gpg.sha256
)
```

5. Descifra una copia recuperada del destino en una máquina aislada. El directorio
   de recuperación debe ser nuevo, privado y cifrado. No extraigas un archivo de
   procedencia desconocida. Verifica el cifrado y las sumas antes de restaurar:

```sh
install -d -m 0700 "$ETF_RECUPERACION"
gpg --batch --pinentry-mode loopback \
  --passphrase-file /etc/etf/copia.clave --decrypt \
  --output "$ETF_RECUPERACION/conjunto.tar" \
  "$ETF_DESTINO_COPIAS/conjunto.tar.gpg"
tar -C "$ETF_RECUPERACION" -xf "$ETF_RECUPERACION/conjunto.tar"
(
  cd "$ETF_RECUPERACION"
  sha256sum -c SHA256SUMS
)
```

Las sumas detectan cambios de contenido; la verificación del contenedor cifrado
comprueba su integridad. Restaurar sigue las órdenes de §5 del manual: preparar
roles, recuperar base y paquetes, invalidar sesiones, reconciliar permisos y
retiradas de datos, comprobar auditoría y realizar el recorrido antes de abrir.
No ejecutes migraciones nuevas sobre la copia para probar una versión anterior.
Registra fecha, versión, tamaño, destino, suma, resultado y tiempo de recuperación;
no escribas la clave en la evidencia. Retira las copias de trabajo en claro según
el procedimiento de almacenamiento cifrado de Informática.

El ensayo [F5.05](EVIDENCIA.md) y la [repetición MANSIS](MANSIS.md) comprueban la
recuperación lógica. El script no cifra ni traslada fuera del servidor. Informática
debe ensayar ese circuito completo, incluida recuperación de claves, antes de
producción y repetirlo tras cambios que afecten a base o paquetes. Configura una
alarma si falla la copia diaria o si la última válida supera un día.

### B. Registros y cadena de auditoría

Los registros técnicos salen en JSON por stdout/stderr. Docker rota tres archivos
de 10 MB por servicio. Puedes consultar sin volcar secretos:

```sh
componer logs --since 30m --tail 100 app migrar preparar-roles
componer ps -a
```

El controlador `local` no es un archivo de texto para abrir directamente en
`/var/lib/docker`. Usa la interfaz de Docker. Limita también el acceso y la
retención del proxy y del colector corporativo. Para una incidencia conserva
hora, versión, servicio, clave de error y estado HTTP; no incluyas nombres,
contenidos de expediente, cookies, contraseñas ni cuerpos de peticiones.

La auditoría funcional está en PostgreSQL. Las lecturas y escrituras registran
actor, acción, identificador y resultado, con huellas encadenadas. Si no se puede
confirmar la auditoría, no se entregan los datos. Informática no debe corregir
una cadena editando o borrando filas ni restaurar sólo la tabla de eventos.

Administración funcional exporta JSON desde Auditoría con su permiso específico.
El [comprobador](../../cmd/comprobar-auditoria/main.go) lee ese archivo local, no
conecta con la base ni lo modifica. Desde la raíz, con Go disponible:

```sh
go build -o "$ETF_HERRAMIENTAS/comprobar-auditoria" ./cmd/comprobar-auditoria
"$ETF_HERRAMIENTAS/comprobar-auditoria" "$ETF_AUDITORIA_JSON"
"$ETF_HERRAMIENTAS/comprobar-auditoria" "$ETF_AUDITORIA_JSON" \
  "$ETF_SELLO_ID" "$ETF_SELLO_HUELLA"
```

Crea antes `$ETF_HERRAMIENTAS` en una ruta privada. Las variables anteriores son
rutas y sello del procedimiento. El primer modo comprueba coherencia interna;
no demuestra que nadie rehízo toda la cadena. El segundo compara también ID y
huella final con un sello custodiado fuera de esa exportación. Para un tramo:

```sh
"$ETF_HERRAMIENTAS/comprobar-auditoria" "$ETF_AUDITORIA_JSON" \
  "$ETF_ID_ANTERIOR" "$ETF_HUELLA_ANTERIOR" "$ETF_SELLO_ID" "$ETF_SELLO_HUELLA"
```

Salida 0: `auditoria.cadena_coherente` o `auditoria.sello_externo_coincide`.
Salida 1: verificación fallida; salida 2: uso, punto o archivo inválido.
Conserva la exportación y el resultado con acceso restringido; los vínculos
usuario-expediente siguen siendo datos personales. El sellado diario externo
no está automatizado por Compose: sigue pendiente R-05. No extraigas la huella
final del mismo JSON para presentarla como sello independiente.

### C. Monitorización y salud

`GET /salud` es público en la red que deja pasar el proxy. Con base disponible
responde HTTP 200 y `{"estado":"ok","base":"ok"}`; puede añadir `avisos`.
Con base no disponible responde HTTP 503 y
`{"estado":"degradado","base":"no_disponible"}`. La comprobación de base tiene
un límite de dos segundos. Docker usa esta ruta para la salud de app; no hay
rutas `/vivo` o `/listo` separadas en esta versión.

```sh
curl --fail --max-time 5 http://127.0.0.1:18083/salud
componer ps -a
componer stats --no-stream
```

Ajusta el puerto si lo cambiaste. Comprueba también la URL HTTPS desde una red
autorizada: el loopback no prueba proxy, DNS o certificado. PostgreSQL usa
`pg_isready` en su healthcheck, que no demuestra autenticación ni permisos; la
prueba de app y el migrador completan esa comprobación. ClamAV comprueba `PONG`,
que no acredita la actualidad de sus firmas.

| Control para Informática | Qué debe alertar y cómo comprobar después |
| --- | --- |
| HTTP interno y HTTPS | 503, errores de acceso o tiempos crecientes; comprobar base, proxy y DNS |
| CPU, memoria y reinicios | Reinicios o límites alcanzados; revisar `stats`, estado y logs sin ampliar a ciegas |
| Disco de base, paquetes y copias | Espacio insuficiente o crecimiento anormal; nunca borrar archivos PostgreSQL a mano |
| AD | Fallo de conexión o certificado; probar LDAPS y entrada autorizada, sin almacenar contraseña |
| Certificados HTTPS y CA de AD | Caducidad próxima o cadena inválida; renovar y probar antes de vencer |
| Antivirus | Falta de respuesta o firmas antiguas; reponer suministro local y ensayar rechazo |
| Copias | Tarea fallida, recepción incompleta o última copia válida de más de un día |
| Auditoría | Falta de sello externo o fallo del comprobador; preservar evidencia y escalar |
| Paquetes | Errores de lectura o publicación; comprobar referencias, permisos y archivos de la misma versión |

Informática fija frecuencia, umbrales, destinatarios y guardias para estos controles.
Compose no instala un colector, panel de métricas, alarmas, programador de copias
ni supervisión de la caducidad TLS. La salud tampoco informa de última copia,
fecha de firmas o acceso real AD, aunque el TRD los prevé.

### D. Conservación y retirada de datos personales

La función está implementada; no es una tarea de borrar filas directamente.
El plazo inicial es 25 años desde el cierre y se configura desde Plazos y tramos
con clave `conservacion` y unidad `anos`. El código admite de 1 a 200 años; cambiar
el plazo requiere decisión autorizada, nueva versión y revisión de propuestas.
La marca se recalcula al cerrar o consultar conservación. No hay que instalar
un cron que invoque SQL de borrado ni adelantar el reloj de producción.

1. Confirmar con el responsable del producto el criterio aprobado para la aplicación y el tratamiento
   de copias, archivos descargados y vínculos de auditoría. Su validación sigue
   pendiente; este procedimiento no la sustituye.
2. Administración funcional entra en Conservación. La lista muestra número,
   cierre, fin del plazo y estado; no da acceso a nombres ni fichas.
3. Una persona revisa número, plazo y alcance y confirma la primera revisión.
   Otra persona de Administración distinta debe confirmar la anonimización.
   Informática prepara el servicio y el respaldo; no se concede un perfil para
   ejecutar esa decisión por su cuenta.
4. El servidor exige expediente cerrado, plazo cumplido, revisión vigente y
   ausencia de seguimiento activo. Si cambian expediente o catálogo, actualizar
   la lista y revisar otra vez. No repetir una petición antigua para saltar el
   conflicto ni llamar directamente a funciones SQL.
5. La confirmación retira nombres, contacto y textos libres, también de las
   versiones históricas, y los recibos que reproduzcan esos datos. Incluye la
   retirada de datos identificativos en Evaluación y Documentos dentro de la
   misma transacción. Se conservan códigos, cifras, fechas necesarias para
   estadísticas y vínculos de auditoría; el expediente queda protegido frente
   a nuevas ediciones. La cadena de auditoría permanece íntegra.
6. Registrar el resultado y comprobar «Datos personales retirados. Se conservan
   las cifras y los vínculos de auditoría.». Una retirada confirmada no tiene
   botón de deshacer. No restaurar una copia antigua para revertirla.

Aunque la pantalla diga «Anonimizado», las fechas y vínculos conservados requieren
el criterio del DPD; no se deben tratar como datos totalmente anónimos. Las
narraciones ya cerradas de Memoria y archivos descargados tienen su propio
alcance de revisión: no prometas que esta acción elimina cualquier texto de toda
la aplicación o de los equipos.

Tras restaurar una copia anterior, mantén la entrada cerrada y reconcilia también
las retiradas realizadas desde su fecha. El estado recuperado puede contener de
nuevo datos retirados. Conserva un registro restringido de esas operaciones para
que las personas autorizadas puedan repetirlas antes de abrir. La retención de
copias y su retirada segura se acuerdan con Informática y el responsable del producto; 25 años de
expediente no significa 25 años para cada copia.
