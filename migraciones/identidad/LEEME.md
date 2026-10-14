# Identidad y sesión · F0.09b

`0001` depende de `plataforma/0001` y `auditoria/0001` y conserva la entrega F0.09b de usuarios y sesiones. `0002_perfiles.sql` añade perfiles con vigencia, asignaciones de equipo con FK al catálogo F0.12 y alta inicial única del administrador. Su manifiesto declara la dependencia de `catalogo/0002`. Véase [F0.10.md](F0.10.md).
El número se reservó en `ETF_RESERVAS_MIGRACIONES.md`, fuera de Git.

La aplicación conecta como `etf_app`. No recibe secretos del migrador.
Las funciones de identidad fijan su `search_path`, no se ejecutan por `PUBLIC`
y acceden a tablas con RLS forzada. La aplicación no puede leer la tabla de
sesiones directamente. Sólo se guarda SHA-256 del identificador de sesión.
Las contraseñas no llegan a PostgreSQL.

Después de validar AD, el alta o actualización del usuario y la creación de su
sesión se confirman en una transacción. Al consultar, el callback de `Modificar`
recibe el contexto de esa transacción para leer al usuario con RLS, sin abrir
otra conexión. Cerrar sesión elimina la fila; una actualización concurrente no
puede volver a insertarla. Esas funciones llaman al puerto SQL público
`auditoria.registrar` de F0.08b en la misma transacción; no leen ni escriben
sus tablas. El fallo de auditoría revierte el cambio y no permite entregar
una sesión o una lectura. Los fallos de aplicación se registran después del
rollback, en una transacción nueva sin identidad comprobada, con acción y
resultado; sin nombres, logins, IP ni contraseñas. Si falla ese apunte se
devuelve una clave de persistencia y se mantiene la denegación.

## Configuración del arranque

Además de las variables `ETF_BD_*` existentes, se requieren:

| Variable | Contenido |
| --- | --- |
| `ETF_HTTP_ORIGEN` | Origen HTTPS público exacto, sin ruta ni barra final |
| `ETF_AD_URL` | Dirección LDAPS del directorio |
| `ETF_AD_BASE_DN` | Base de búsqueda de personas |
| `ETF_AD_FORMATO_BIND` | Formato UPN, con una sustitución `{usuario}` |
| `ETF_AD_CA_RUTA` | Fichero PEM de la autoridad de la entidad, fuera del repositorio |
| `ETF_SESION_MAXIMA` | Duración absoluta elegida por el operador, en formato de Go, como `8h` |

No se fija una duración absoluta por defecto: sigue pendiente de decisión en
TRD §4. La inactividad máxima es de 30 minutos. Sin configuración válida se
rechazan nuevas entradas. La ruta `/api/sesion` devuelve claves de error, sin
diagnósticos del directorio ni de PostgreSQL.

La cookie `__Host-etf-sesion` es opaca, `HttpOnly`, `Secure`, `SameSite=Strict`,
con ruta `/` y sin dominio. GET devuelve estado y testigo CSRF; una sesión activa
incluye una proyección mínima de identidad. Una sesión pendiente no recibe
nombre ni cuenta. POST y DELETE exigen origen exacto y una cabecera
`X-CSRF-Token` ligada mediante HMAC a la cookie. La entrada inicial también
requiere ese testigo, obtenido mediante GET y una cookie anónima segura.
Antes de CSRF, JSON y casos de uso, el servidor admite como máximo 100 peticiones
por minuto y dirección de origen, con un mapa de 10 000 claves como máximo.
Una solicitud fuera de presupuesto recibe 429 y no inicia un acto de sesión
ni una escritura de auditoría; queda en el registro técnico HTTP. Esto evita
que una ráfaga de solicitudes inválidas haga crecer sin límite la cadena.
Las solicitudes admitidas rechazadas por CSRF o JSON sí se auditan mediante
el puerto de entrada, sin llamar a AD. Los límites de credenciales son
independientes: 5 por usuario y 30 por IP durante 15 minutos.
El testigo se recalcula en el servidor: no se guarda en la tabla de sesiones.
La clave HMAC se genera al arrancar; después de un reinicio se obtiene un nuevo
testigo con GET. No se usan cabeceras de origen de red aportadas por el cliente.

## Ensayo y recorrido

Con las imágenes locales `postgres:18`, `etf-f003-migrar` y
`etf-ldap-pruebas:f0.09a`, las dependencias npm instaladas y Chrome del sistema:

```sh
internal/adaptadores/postgres/testdata/ensayar-identidad.sh
ETF_NAVEGADOR_INTEGRACION=1 ETF_EVIDENCIA_RUTA=/tmp/etf-f009b-capturas \
  internal/adaptadores/postgres/testdata/ensayar-identidad.sh
go test -tags ldap_integracion -count=1 -v ./internal/adaptadores/ldap
```

El script crea un PostgreSQL 18 desechable en una red interna propia. Ensaya
migraciones desde la versión anterior, reaplicación y base vacía. Cuando la
rama de tarea aún no contiene F0.08b, toma sólo su migración desde `main` (o
`ETF_AUDITORIA_REF`) y registra el commit exacto. La monta desde una carpeta
temporal; no fusiona ni modifica el worktree. Ejecuta RLS,
rollback, cierre concurrente y consulta con un pool de una conexión. El recorrido
opcional compila la web, crea un LDAPS de prueba con certificado local y abre
Chrome mediante Playwright. Usa datos inventados. Comprueba entrada, pendiente,
salida, teclado, 1024/390/320 px, español/inglés, alto contraste y AD caído.
Las capturas y trazas quedan fuera del repositorio. El contenedor y su base se
destruyen al acabar. Sin activar la integración, las pruebas de PostgreSQL se
omiten; eso no acredita su resultado.

## Dependencias y límites de integración

- F0.08b está integrada en `main`; el manifiesto declara `auditoria/0001`.
  El director debe integrar esta rama sobre ese `main` antes del despliegue.
  Se han ensayado registro de accesos/fallos, cadena y bloqueo al fallar auditoría.
- La cuenta AD vigente es única y puede quedar vacía en la identidad anterior
  cuando AD reutiliza su login. `cuenta_ad_anterior` conserva el último login
  comprobado; el GUID, estado y permisos anteriores permanecen intactos. El
  nuevo GUID obtiene otra identidad pendiente. La liberación y el alta se
  serializan y se confirman junto con la sesión, sin transferencia de permisos.
- La migración 0001 no concede perfiles. F0.10 incorpora perfiles, equipos y alta inicial única del administrador en la migración 0002. Los permisos los asigna Administración funcional; la configuración sólo concede el primer administrador una vez.
- El límite por IP usa `RemoteAddr`. Si un proxy termina HTTPS, debe acordarse
  cómo obtener la dirección real desde un proxy de confianza, antes del despliegue.
- Cambiar idioma o tema funciona durante la página abierta. Recargar recupera
  las opciones configuradas en el servidor; aquí no se persisten preferencias.

## Resultado comprobado el 13/10/2026

- `go vet ./...` y `go test ./...`: salida 0. También pasan las pruebas con
  `-race` de aplicación de identidad, HTTP y PostgreSQL.
- `npm --prefix web test`: 4 pruebas aprobadas, ninguna fallida. La compilación
  TypeScript y Vite pasa.
- Script de PostgreSQL: `ENSAYO-OK`, tanto desde base vacía como desde la versión
  anterior. Las pruebas de GUID, reutilización de login, RLS, cierre concurrente
  y rollback de la liberación de cuenta se ejecutaron, sin omisiones. También
  pasa `TestIntegracionAuditoriaFallidaBloqueaDatosYCambios`: no entrega lecturas
  ni confirma entradas/cierres cuando falla la función de auditoría.
- LDAP: `TestIntegracionLDAPS` aprobado, incluidos certificado no confiable y
  directorio caído.
- Navegador: `TestNavegadorIdentidad` aprobado con PostgreSQL y LDAPS reales de
  prueba. Los tres recorridos de entrada, pendiente y salida pasan; también
  rechazo CSRF y caída de AD con una sesión anterior abierta.
- Semgrep local: `Ran 10 rules on 11 files: 0 findings.` Gitleaks: `no leaks found`.
  La revisión independiente comprueba SQL, atomicidad, identidad, CSRF y textos.
  La conexión con F0.08b pasa el ensayo de cadena y fallo de la función SQL.
- Temas, índice de producto y enlaces pasan. Comprobación de textos acotada a
  F0.09b: `Textos: 2 idiomas, 19 claves usadas; sin literales visibles.`
  El comprobador global falla en archivos previos de catálogos y del ejecutor
  de migraciones: trata el catálogo anidado como textos planos y marca SQL
  técnico como texto visible. Esos archivos no se cambiaron en F0.09b.

Los enganches comunes son `cmd/etf/main.go`,
`internal/adaptadores/http/servidor.go` y `web/src/main.tsx`. El ajuste del puerto
`Sesiones.Modificar` y de su caso de uso permite compartir el contexto RLS;
`Operaciones` añade la transacción del inicio de sesión. Se fijó
`playwright-core` 1.58.2 en el manifiesto y el lock de npm. No se hizo push,
fusión ni despliegue.
