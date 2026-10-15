# TRD: requisitos técnicos de Expedientes ETF
> Especificación de la edición pública. La entidad que la instale configura sus datos, identidad y plantillas.
> Las referencias D1–D22 se conservan en el registro de decisiones del PRD; las fuentes privadas no se distribuyen.

Expedientes ETF es la aplicación web interna de la entidad titular del servicio con la que la coordinación provincial y los Equipos de Tratamiento Familiar (ETF) llevan los expedientes de las familias desde la derivación hasta el cierre, generan los documentos Word y sacan la Tabla Mensual para el Servicio de Prevención y Apoyo a la Familia (SPAF) y la Memoria Técnica Anual. Este documento dice cómo se construye. Qué hace y para quién está en `PRD.md`; los recorridos, en `FLUJO_APP.md`; las pantallas, en `DISENO_UI_UX.md`; tablas, permisos de base de datos, migraciones y API, en `ESQUEMA_BACKEND.md`; el orden de trabajo, en `PLAN_IMPLEMENTACION.md`.

Todavía no hay código. Toda ruta del repositorio que aparece aquí está prevista; cuando exista, este documento se corrige para que coincida.

## 0. Decisiones de base

| Decisión | Elegido | Fecha y quién |
| --- | --- | --- |
| 1. Idiomas y textos | Internacionalización (i18n) pura: ningún texto visible ni idioma en el código. Cada idioma es un directorio de datos `idiomas/<código>/` fuera del código, leído en tiempo de ejecución. Añadir o cambiar un idioma es añadir o cambiar un directorio. Ver apartado 8. | 13/10/2026 · responsable del producto |
| 2. Arquitectura | Hexagonal estricta (dominio, aplicación, puertos, adaptadores); los módulos no tocan tablas ajenas; todo sistema externo entra por un conector (puerto y adaptador). | 13/10/2026 · responsable del producto |
| 3. Programación | Código limpio: lo más simple que funcione, sin capas de más ni código de demostración, errores que no se tragan, sin estado global mutable, ficheros cohesionados (aviso por tamaño, no tope). | 13/10/2026 · responsable del producto |
| 4. Lenguaje del dominio | Castellano para los nombres del dominio; términos técnicos universales en inglés (`handler`, `context`, `id`). | 13/10/2026 · responsable del producto |
| 5. Seguridad | Denegación por defecto, privilegio mínimo, secretos fuera del repositorio, identidad tomada sólo de una frontera confiable (la sesión del servidor), seguridad por filas en PostgreSQL. | 13/10/2026 · responsable del producto |
| 6. Identidad y sesión | Usuario y contraseña de Active Directory (AD) validados por LDAPS. Sesión guardada en el servidor; el navegador sólo lleva un identificador opaco en una cookie `HttpOnly`, `Secure`, `SameSite=Strict`. Sin doble factor. | 13/10/2026 · responsable del producto |
| 7. Auditoría | Se registra todo, también las lecturas; registro encadenado con huellas (hash) y exportable para el Delegado de Protección de Datos (DPD). | 13/10/2026 · responsable del producto |
| 10. Rapidez | Lecturas por debajo de 300 ms en el percentil 95 (p95) con volumen real; Memoria en menos de 3 s; sin consultas en bucle. | 13/10/2026 · responsable del producto |
| 13. Base de datos y migraciones | PostgreSQL 18; migraciones numeradas por módulo, sólo hacia delante, ensayadas en una copia antes de fusionar. | 13/10/2026 · responsable del producto |
| 14. Pruebas | Proporcionales al riesgo; en local sólo lo cambiado; la puerta completa en la integración continua (CI). | 13/10/2026 · responsable del producto |
| 15. Despliegue | Docker Compose; copia previa y vuelta atrás automática; una configuración nueva nunca impide arrancar. | 13/10/2026 · responsable del producto |
| 17. Idioma de documentos | Castellano llano, sin siglas sin explicar. | 13/10/2026 · responsable del producto |
| 18. Temas visuales | Temas instalables como en Joomla: paquetes `temas/<nombre>/` con tokens en datos, que Administración funcional sube y activa sin tocar código. El aspecto de la maqueta es el tema por defecto; habrá otro de alto contraste. Ver apartado 8b. | 13/10/2026 · responsable del producto |

## 1. Pila

| Pieza | Tecnología y versión | Dónde se fija (previsto) |
| --- | --- | --- |
| Lenguaje del servidor | Go, última versión estable publicada al crear el módulo. El módulo exige Go 1.26.9 (actualizado en F5.02). | `go.mod` (directivas `go` y `toolchain`) |
| Acceso a PostgreSQL | `pgx` v5 | `go.mod` |
| Acceso a AD | `go-ldap` v3, sólo LDAPS (puerto 636) | `go.mod` |
| Migraciones | Herramienta pendiente de elegir: `golang-migrate` o un ejecutor propio que respete las dependencias declaradas por migración. Decide el equipo técnico (asunto interno). | `go.mod`, `migraciones/` |
| Generación de Word | Relleno propio del XML de plantillas `.docx` (`word/document.xml`), sin LibreOffice en producción | `internal/adaptadores/docx/` |
| Lectura de Anexo 3 y 4 | `.docx` y `.odt`: recorrido de tablas y controles del XML. PDF: librería en Go puro, local, sin reconocimiento óptico de caracteres (OCR) | `internal/adaptadores/anexos/` |
| Interfaz | React + TypeScript (modo estricto) + Vite, componentes accesibles con React Aria. Se compila en la imagen y la sirve el ejecutable Go; no hay Node en producción. | `web/package.json`, `web/vite.config.ts` |
| Node (sólo para compilar la interfaz) | Versión de soporte largo (LTS) compatible con Vite, sólo en la etapa de compilación de la imagen | `despliegue/Dockerfile` |
| Base de datos | PostgreSQL 18 con seguridad por filas (RLS, *row level security*: la base de datos filtra qué filas ve cada usuario) | `despliegue/compose.yaml` (imagen fijada por versión y huella) |
| Antivirus | ClamAV en su propio contenedor, consultado por el protocolo de `clamd` | `despliegue/compose.yaml` |
| Ejecución | Docker Compose con tres servicios permanentes: `app`, `postgres` y `clamav`, y la orden o contenedor puntual `migrar` durante el despliegue. Nada instalado «a pelo» en la máquina. | `despliegue/compose.yaml` |
| Tipografías | Public Sans y Source Serif 4, servidas por la propia app dentro del tema (nunca desde Google Fonts ni otro servicio externo) | `temas/expedientes-etf/fuentes/` |

El repositorio público y la integración continua contienen sólo código y datos inventados. La entidad controla dónde se alojan los expedientes reales. Los análisis de seguridad se ejecutan localmente; las dependencias Go/npm se descargan con versiones y sumas fijadas.

## 2. Arquitectura

### 2.1 Estilo y reglas de dependencia

Monolito modular con arquitectura hexagonal estricta. Las reglas del Programa (estados del expediente, plazo de 30 días, numeración, cálculo de la Tabla SPAF y de la Memoria) viven en un núcleo que no sabe nada de PostgreSQL, de AD, de HTTP ni de Word.

| Capa | Qué contiene | Puede importar | No puede importar |
| --- | --- | --- | --- |
| `internal/dominio/<módulo>/` | Entidades, valores, reglas y errores | Biblioteca estándar sin acceso a sistemas externos; recibe hora y datos como valores | Aplicación, puertos, adaptadores, base de datos, HTTP, bibliotecas externas |
| `internal/aplicacion/<módulo>/puertos/entrada/` | Interfaces de los casos de uso que llama HTTP y sus datos de entrada/salida | Dominio | Implementaciones de aplicación, adaptadores |
| `internal/puertos/salida/<módulo>/` | Interfaces de repositorios, directorio, antivirus, Word, registro de salida y reloj | Dominio | Aplicación, adaptadores |
| `internal/aplicacion/<módulo>/` | Implementa los puertos de entrada; permisos, transacción, reglas y auditoría | Dominio, puertos de entrada y salida | Adaptadores |
| `internal/adaptadores/http/` | Adaptador de entrada; valida la petición y llama al caso de uso | Puertos de entrada de aplicación y sus tipos | Implementaciones de aplicación, adaptadores de salida, tablas |
| `internal/adaptadores/<tecnología>/` | Adaptadores de salida que implementan los puertos externos | Dominio, puertos de salida | Implementaciones de aplicación, otros adaptadores |
| `cmd/etf/` | Arranque y composición de la aplicación | Todas las capas para inyectar dependencias | Credenciales del migrador |
| `cmd/migrar/` | Ejecutor separado durante el despliegue | Infraestructura de migraciones | Ejecución desde la aplicación |

Los puertos de entrada pertenecen a aplicación: HTTP depende explícitamente de
ellos, y nunca de la implementación de un caso de uso. El reloj es un puerto de
salida usado por aplicación; el dominio recibe la fecha como un valor.

Ejemplo de la regla, cuando exista el código: `internal/dominio/expediente/` (prevista) calcula si un expediente está «pendiente de atender» con la fecha del Acta de Aceptación, la del consentimiento y el plazo vigente que recibe como dato [D3]; no importa `internal/adaptadores/postgres/`. Una prueba de arquitectura en la CI (`go list -deps` o reglas de `depguard`) falla si alguna capa rompe esta tabla.

### 2.2 Estructura de carpetas prevista del repositorio `expedientes-etf`

```
cmd/etf/                      arranque y composición
internal/dominio/<módulo>/    núcleo por módulo (ver 2.3)
internal/aplicacion/<módulo>/ casos de uso por módulo
internal/aplicacion/<módulo>/puertos/entrada/  interfaces de casos de uso
internal/puertos/salida/<módulo>/              interfaces hacia el exterior
internal/plataforma/          configuración y soporte técnico (PLAN §3)
cmd/migrar/                   ejecutor de migraciones de despliegue
internal/adaptadores/
  postgres/                   repositorios, RLS, auditoría
  ldap/                       validación de contraseña en AD
  clamav/                     antivirus
  docx/                       generación de Word desde plantillas
  anexos/                     lectura del Anexo 3 y del Anexo 4
  http/                       API JSON, sesión, CSRF, ficheros de la interfaz
  registro_manual/            registro de salida anotado a mano
  ficheros/                   lectura de idiomas y temas mediante puertos de salida
web/                          interfaz React + TypeScript + Vite
idiomas/<código>/             textos por idioma (datos, volumen en Docker)
temas/<nombre>/               paquetes de tema (datos, volumen en Docker)
idiomas/<código>/textos/      mensajes y textos de pantalla
idiomas/<código>/catalogos/   textos de opciones de catálogo
idiomas/<código>/plantillas/  plantillas Word inmutables por versión
migraciones/<módulo>/         SQL numerado por módulo
Dockerfile, compose.yaml     imágenes y arranque local (F0.03)
despliegue/                   configuración base, guiones de copia y subida
docs/producto/                los seis documentos de producto
```

F0.04 concreta el adaptador de ficheros ya previsto en §2.4 y la carpeta de
soporte de PLAN §3. Conserva los diez nombres del mapa en los esqueletos de
dominio, aplicación y puertos; los de `plataforma` son reservas sin lógica de
negocio. Su configuración técnica vive en `internal/plataforma/`, que sólo
importa biblioteca estándar y se conecta desde `cmd/etf/`.

El esqueleto usa `ETF_HTTP_DIRECCION` (`127.0.0.1:8080`), `ETF_IDIOMAS_RUTA`
(`idiomas`), `ETF_TEMAS_RUTA` (`temas`), `ETF_IDIOMA` (`es`) y `ETF_TEMA`
(`expedientes-etf`). Las rutas relativas parten del directorio de ejecución.
Las opciones de conectores se incorporan con sus entregas; F0.04 no abre
base de datos ni lee secretos. El puerto no disponible impide escuchar;
un paquete ausente sólo desactiva esa lectura y aparece como aviso en salud.
El idioma o tema elegido ausente vuelve al predeterminado del mismo directorio.
La copia de reserva de datos en imagen corresponde a F0.03.

`make build` instala las versiones del lock, compila Vite y después los dos
binarios en `bin/`. `web/embed.go` embebe sólo `web/dist/`; textos y temas
siguen en disco. El `.gitkeep` permite compilar Go desde un checkout sin web
construida, pero la página devuelve 503 hasta ejecutar `make build`.
`make test` y `make vet` ejecutan las comprobaciones locales. La prueba de
arquitectura examina también fuentes Go condicionadas por etiquetas de build.
El endpoint de tokens devuelve CSS generado a partir de valores JSON validados;
no acepta hojas de estilo. Instalación, activación, validación completa y fuentes
locales WOFF2 quedan para F0.07/F0.14c; de momento se usan las fuentes de reserva
de los tokens. `color.peligro-fondo` sigue pendiente de aprobación en DISENO §3.3
y no se usa ni se inventa un valor en este esqueleto.

### 2.3 Módulos

Mapa único, compartido con `ESQUEMA_BACKEND.md` §1 y
`PLAN_IMPLEMENTACION.md` §3. Cada módulo usa las carpetas de dominio, aplicación,
puertos y adaptadores del apartado 2.2; `plataforma` es soporte técnico, no un
módulo de negocio.

| Módulo | Responsabilidad |
| --- | --- |
| `plataforma` | Migraciones y contexto técnico de seguridad |
| `identidad` | Usuarios, perfiles, equipo y sesiones |
| `catalogo` | Equipos, centros, municipios, listas, reglas, SMI y firmantes |
| `paquete` | Idiomas, temas y referencias activas de sus versiones |
| `expediente` | Derivación, ficha, hitos, salidas, cierre, seguimiento, avisos, conservación y carga |
| `evaluacion` | Instrumentos y escalas |
| `documento` | Plantillas, emisiones y registro de salida |
| `actividad` | Comisiones e informes técnicos |
| `estadistica` | Tabla SPAF, Memoria y totales |
| `auditoria` | Eventos y cadena de auditoría |

Reglas entre módulos:

- Cada módulo es dueño de sus tablas y de su carpeta en `migraciones/<módulo>/`. Ningún módulo lee ni escribe tablas de otro.
- Si un módulo necesita datos de otro, llama a un caso de uso o a un puerto de lectura de ese módulo. Para los cálculos de la Tabla SPAF y la Memoria, que leen de casi todos, cada módulo publica vistas de sólo lectura que son parte de su contrato. Propuesta del equipo técnico; se fija en `ESQUEMA_BACKEND.md`.
- Comunicación por llamadas síncronas dentro del mismo proceso y la misma transacción. No hay colas ni bandeja de salida: la app no habla con sistemas externos de forma asíncrona. Si llega el conector con MOAD (registro electrónico de la entidad), se valorará una bandeja de salida.

INT5 compone cada capacidad HTTP con un tipo propio que implementa `Rutas() []Ruta`.
Cada entrada reúne el handler, patrón, métodos, permiso y pantalla SPA asociada
cuando la capacidad tiene interfaz. El arranque valida duplicados por patrón y
método, métodos vacíos, permisos ausentes o ambiguos, escrituras sin CSRF y
referencias a pantallas inexistentes. `NuevoValidado` devuelve el error antes de
abrir el servidor. Los módulos se componen de forma explícita y tipada en
`cmd/etf/modulos.go`; no se registran mediante `init()`.

Para añadir un módulo: crear `cmd/etf/<m>.go` con sus dependencias y su tipo HTTP,
`web/src/<m>/pantallas.ts` con la lista tipada de ruta, perfiles, clave de menú y
componente, e `idiomas/*/textos/<m>.json` con sus textos. Añadir una entrada de
composición en `cmd/etf/modulos.go` y en `web/src/pantallas.ts`, con el orden
acordado. El servidor, el selector de componentes y el menú leen esos registros;
los componentes y sus handlers permanecen en los ficheros del módulo. Las
pruebas comparan todas las pantallas web con las rutas SPA reales del servidor,
incluidos los perfiles y las claves, y rechazan rutas web duplicadas.

### 2.4 Conectores

Todo sistema externo es un puerto de salida en `internal/puertos/salida/<módulo>/` con al menos un adaptador. Añadir un conector nuevo es añadir un adaptador y una línea en el arranque; no toca dominio ni aplicación.

| Puerto (previsto) | Adaptador inicial | Para qué |
| --- | --- | --- |
| `Directorio` | `ldap` | Validar usuario y contraseña en AD |
| `Antivirus` | `clamav` | Revisar cada tipo de subida según el apartado 10 |
| `LectorAnexo` / `LectorActas` | `anexos` | Extraer Anexos 3/4 y actas de aceptación/cierre para revisión. Actas y derivaciones usan modos separados del trabajador PDF, con las mismas cotas |
| `GeneradorDocumentos` | `docx` | Rellenar las plantillas Word |
| `RegistroSalida` | `registro_manual` | Anotar número y fecha de registro de un oficio hecho fuera. Más adelante, adaptador MOAD |
| `Textos` | sistema de ficheros sobre `idiomas/` | Textos por idioma |
| `Temas` | sistema de ficheros sobre `temas/` | Paquetes de tema |
| `Plantillas` | sistema de ficheros sobre `idiomas/<código>/plantillas/` | Plantillas Word por idioma y versión |
| `Reloj` | reloj del sistema | Hora inyectable en pruebas |
| Repositorios por módulo | `postgres` | Persistencia |

Un segundo factor de autenticación o un resumen automático con un modelo instalado en un servidor de la entidad [D13] entrarían también como conectores nuevos.

## 3. Arranque y configuración

- La app se compone en `cmd/etf/` (previsto): lee la configuración, abre la base de datos con el rol de aplicación, comprueba la versión del esquema sin modificarla, crea cada adaptador, carga idiomas, tema activo y plantillas, y arranca el servidor HTTP y las tareas programadas internas (avisos de cada noche, marcado de expedientes a anonimizar).
- Las migraciones se ejecutan antes, con la orden o contenedor separado `migrar` y el rol `etf_migrador`. La app nunca migra ni recibe esas credenciales. Cada migración declara dependencias: `catalogo` precede a `identidad`; si hay un ciclo, se crean primero las tablas y después las relaciones cruzadas. El ejecutor ordena por esas dependencias y registra cada paso aplicado; no basta ordenar carpetas.
- La configuración entra por variables de entorno con prefijo `ETF_`, documentadas en `.env.plantilla` (F0.03, sin valores secretos). Los secretos (contraseña de la base de datos, cuenta de servicio de AD si hiciera falta) van en ficheros de secretos de Docker o en un fichero de entorno fuera del repositorio; `.env` ya está en `.gitignore`.
- Variables previstas: dirección de la base de datos; URL de AD y certificado de la autoridad de la entidad; formato del nombre de usuario para el *bind* (ver 4); dirección de ClamAV y su tiempo máximo; usuario AD del primer administrador; idioma por defecto (`es`); rutas de `idiomas/` y `temas/`; caducidad de la sesión por inactividad y máxima; límites de subida por tipo (apartado 10); adaptador de `RegistroSalida` (`manual` por defecto).

Regla: una configuración que falte o falle desactiva sólo su función, avisa en la página de salud y en el registro técnico, y nunca impide arrancar el resto.

| Si falla… | La app… |
| --- | --- |
| AD (configuración o conexión) | Arranca; no se puede iniciar sesión; las sesiones abiertas siguen hasta caducar |
| ClamAV | Sólo el anexo puede continuar con aviso visible y auditoría de la falta de revisión; las demás subidas esperan una revisión satisfactoria |
| Un directorio de idioma distinto de `es` | No ofrece ese idioma en el selector |
| `idiomas/es/` | Usa la copia de reserva de `es` que viaja como datos en la imagen (propuesta técnica; los textos siguen fuera del código) |
| Tema activo | Vuelve al tema por defecto |
| Una plantilla Word | Desactiva sólo la generación de ese documento, con mensaje claro |
| Adaptador MOAD (cuando exista) | Vuelve al registro manual |
| PostgreSQL | Es la única dependencia imprescindible: el proceso arranca, responde la página de salud con el error y reintenta; el resto de rutas devuelven «servicio no disponible» |

## 4. Identidad y sesión

SEG4 (R-22): la detección usa un paso obligatorio del procedimiento, con la aplicación parada. Sistemas custodia fuera de la copia la identidad vigente por `objectGUID`, restaura y activa `identidad.revision_restauracion` mediante `scripts/preparar-revision-restauracion.py` antes de arrancar. El paso compara ambas listas, borra sesiones y registra la revisión en una transacción. Cada consulta de sesión comprueba la marca persistente; sólo Administración vigente en ambas listas entra y sólo accede a la revisión. Cada decisión y la confirmación se guardan con auditoría y versión optimista. Una nueva restauración invalida revisiones anteriores. Sin referencia íntegra o sin administrador en ambas listas, se mantiene la aplicación parada; no se concede un administrador automático.

- Entrada con usuario y contraseña de la entidad. La app hace un *bind* (comprobación de credenciales) contra los controladores de dominio por LDAPS (puerto 636), validando el certificado con la autoridad de la entidad. No guarda contraseñas ni las escribe en registros.
- UI1 añade a la respuesta de sesión activa el nombre del personal y las pantallas autorizadas (`ruta`, `perfiles`, `clave`), derivadas de `tablaRutas()`. La sesión pendiente sigue sin identidad visible. La proyección de Usuarios añade `nombre` al dato que ya consulta; no cambia permisos, auditoría ni migraciones.
- AD valida las credenciales y devuelve `objectGUID`, su identificador estable. La identidad y los permisos se vinculan a ese identificador, además de conservar el login actualizado. Renombrar una cuenta no cambia sus permisos; reutilizar un login con otro `objectGUID` no hereda los anteriores. Perfiles y equipos se dan en la app (apartado 5), no salen de grupos de AD.
- Falta saber con Informática si tras el *bind* directo con `usuario@dominio` se puede leer su `objectGUID` o hace falta una cuenta de servicio de sólo lectura para buscarlo. La opción preferida es la primera, porque no exige guardar ningún secreto de AD.
- Se rechaza cualquier contraseña vacía antes de llamar a AD (un *bind* sin contraseña puede darse por bueno en AD sin comprobar nada). El nombre de usuario se escapa contra inyección LDAP. Intentos fallidos limitados por usuario y por dirección de origen; las entradas correctas no gastan cupo. Detrás del proxy HTTPS, la dirección de origen sale de `X-Forwarded-For` sólo si la conexión llega desde una red declarada en `ETF_HTTP_PROXIES` (F5.02); si no, se usa la del propio socket.
- Si AD no responde, nadie puede entrar; las sesiones abiertas siguen hasta caducar.
- Sesión en el servidor: al entrar se crea un identificador aleatorio de 256 bits; en la base de datos sólo se guarda su huella. El navegador recibe el identificador en una cookie opaca `HttpOnly`, `Secure`, `SameSite=Strict`, con ruta `/` y sin dominio. Se crea un identificador nuevo en cada inicio de sesión.
- Caducidad (el responsable del producto, 14/10/2026; SEG1): 30 minutos sin actividad y máximo absoluto de 8 horas. `ETF_SESION_INACTIVIDAD` y `ETF_SESION_MAXIMA` permiten configurarlos; la inactividad debe superar los dos minutos del aviso, no superar el máximo y éste tiene un tope técnico de 24 horas. El servidor valida ambos límites en cada petición. `GET /api/sesion` comprueba sin renovar; `PUT /api/sesion`, con CSRF y origen comprobados, renueva sólo una sesión vigente. La interfaz usa las cabeceras de caducidad y hora del servidor, avisa dos minutos antes y renueva por actividad humana o por «Continuar sesión». El límite absoluto exige volver a entrar. Al caducar vuelve a la entrada; los datos guardados permanecen. No guarda sesión ni formularios en almacenamiento del navegador.
- F0.09b implementa sesión PostgreSQL, HTTP y pantallas de entrada y pendiente; requiere `ETF_HTTP_ORIGEN`, configuración LDAPS y permite configurar los límites de sesión. Depende de `auditoria/0001` de F0.08b: lecturas y cambios confirman su apunte antes de responder; su fallo bloquea la entrega. Antes del caso de uso, el transporte limita peticiones por IP para proteger la cadena; las solicitudes no admitidas sólo dejan registro técnico HTTP. Protección contra peticiones falsificadas (CSRF): además de `SameSite=Strict`, cada sesión tiene un testigo que la interfaz envía en una cabecera en toda petición que cambie datos; el servidor comprueba también el origen.
- Cerrar sesión borra la sesión en el servidor. Retirar un perfil a alguien surte efecto en su siguiente petición, porque los permisos se leen del servidor en cada una.
- Primera entrada de una persona con cuenta de AD y sin perfil: queda «pendiente de alta» y sólo ve una pantalla que lo dice (`FLUJO_APP.md`). Administración funcional le asigna perfil y equipo.
- Primer administrador: el login configurado se resuelve a su `objectGUID` y la inicialización se ejecuta una sola vez. La concesión, la marca persistente de inicialización y la auditoría se confirman juntas. Quitar o desactivar después a todos los administradores nunca vuelve a conceder el perfil automáticamente; una recuperación exige una operación expresa del operador, registrada y auditada.
- Prohibido: tomar la identidad, el perfil o el equipo de lo que mande el navegador (parámetros, cabeceras propias, almacenamiento local). Sin doble factor por ahora (red interna); podría añadirse como conector.
- Acceso de red: sólo desde la red corporativa, la red provincial de ayuntamientos o la VPN. Nunca publicada en Internet. Dirección prevista dominio de ejemplo `etf.example.org`; configurar con Informática.

## 5. Autorización

- Perfiles: Coordinación, Profesional de ETF (sólo su equipo), Consulta (sólo totales, nunca nombres) y Administración funcional (usuarios y permisos, catálogos, plazos y tramos, plantillas Word, firmantes y destinatarios, temas e idiomas; no ve expedientes). Una persona puede tener varios perfiles a la vez, por ejemplo Coordinación y Administración funcional. Qué puede hacer cada uno, en `PRD.md` y `FLUJO_APP.md`.
- SEG2 (R-10, 14/10/2026): todo cambio de permisos cuyo destinatario sea el actor se deniega antes de escribir, también baja, vigencias y equipo. Otra persona de Administración debe hacerlo. `identidad/0003` repite la comprobación en la función de escritura; RLS y privilegios impiden DML directo. El rechazo HTTP devuelve 403 con una clave traducida y confirma su auditoría en una transacción independiente. La función SQL devuelve `false` tras registrar el rechazo, sin lanzar una excepción que borre ese apunte; `true` acredita el cambio. El alta inicial de instalación conserva su función separada y su marca de ejecución única.
- Denegación por defecto: un caso de uso sin permiso declarado no se ejecuta. El permiso se decide en la capa de aplicación con el perfil y el equipo leídos de la sesión, y otra vez en PostgreSQL.
- HTTP deriva su tabla de autorización de las rutas aportadas por los módulos: handler, patrón, métodos, perfiles, exigencia de ámbito y CSRF; cualquier API o pantalla sin regla queda denegada. La RLS verifica la pertenencia del expediente al equipo.
- Una prueba recorre todas las rutas registradas en el mux y exige su regla; otra genera desde la tabla la matriz de perfiles, rutas y métodos permitidos o denegados.
- RLS en PostgreSQL: en cada transacción la app fija con `SET LOCAL` el usuario, sus perfiles y su equipo; las políticas de cada tabla de expedientes filtran por equipo y perfil. Administración funcional y Consulta no tienen política que les deje leer fichas ni filas de expedientes. Administración sólo accede a los vínculos personales de auditoría mediante su permiso específico de exportación (§6); la conservación dispone de una proyección propia sin nombres ni ficha. El rol de conexión de la app no es propietario de las tablas ni tiene `BYPASSRLS`. Roles y políticas, en `ESQUEMA_BACKEND.md`.
- F4.07 usa `etf_calculo_memoria`, con credencial propia `ETF_CALCULO_MEMORIA_CONTRASENA`, sólo para calcular cifras abiertas de Consulta. No tiene membresías, propiedad, BYPASSRLS, escritura ni acceso a tablas personales. Lee proyecciones estadísticas explícitas bajo el usuario y los perfiles reales de sesión; su ámbito es el solicitado para cifras. El motor es compartido con Equipo/Coordinación y entrega un tipo numérico independiente. La conexión `etf_app` de Consulta permanece sin filas de expedientes ni acceso a las vistas técnicas. El ejecutor no necesita ni recibe credenciales de migración. Instalación existente: Compose prepara automáticamente el rol antes de `estadistica/0005` (FX2, §12).
- Consulta sólo recibe cifras mediante una proyección específica, sin `composicion` (identificadores de expedientes) ni textos libres. No recibe filas de expedientes ni acceso directo a las Memorias completas, aunque estén cerradas. La cerrada usa su revisión vigente; la abierta se calcula al momento. API y Word indican fecha y carácter definitivo o provisional; no hay fotos periódicas. La lectura de la foto cerrada usa una sola sentencia MVCC, sin bloqueo asesor (`estadistica/0006`, FX4). La transacción de `etf_app` termina antes del cálculo técnico; una segunda transacción confirma la auditoría antes de entregar cifras o Word. Un cierre concurrente no cambia el carácter provisional del cálculo iniciado con la Memoria abierta; la siguiente lectura obtiene la foto cerrada. SEG1 aplica la supresión de cifras pequeñas después de calcular o leer la foto cerrada y antes de auditar y entregar API o Word. Usa el umbral vigente hoy del catálogo de plazos, también para años anteriores, con valor inicial 3. La proyección entrega `null` para los recuentos ocultos; el Word y la pantalla muestran el texto del idioma. La supresión secundaria itera por filas y columnas hasta que no quede una sola oculta con otra cifra publicable. Cuando hay celdas ocultas se retiran los márgenes y porcentajes de la respuesta que podrían reconstruirlas entre apartados.
- Listados enmascarados: iniciales («Familia O. S.») y número. El nombre completo sólo en la ficha, y abrirla queda auditado.
- Dos personas: el equipo propone el cierre (Anexo 11 o 12) y Coordinación cierra (Acta de Cierre). F5.04 añade la retirada de datos personales al cumplir la conservación: dos personas distintas con perfil de Administración confirman, sin acceso a fichas. La lista sólo entrega número, fechas, plazo, versión y estado de revisión.
- El personal de Informática administra el servidor y no tiene perfil en la app.
- Un profesional pertenece a un solo equipo a la vez. Cada equipo puede cerrar su Memoria; sólo Coordinación cierra la provincial. Un expediente cerrado sólo admite editar el seguimiento; cualquier otra corrección exige reabrirlo.

## 6. Auditoría y trazabilidad

AUD1 prepara exportaciones por páginas y archivos privados, sin reunir el registro en memoria. Compose monta `/tmp` como tmpfs de 320 MiB para el usuario de la aplicación (10001), y asigna 512 MiB al contenedor: dos plazas, con hasta 64 MiB de registro y prueba más 64 MiB de ZIP por plaza. Se eliminan los temporales al terminar o fallar. Fuera de Compose, el servicio necesita un `TMPDIR` privado y escribible con capacidad equivalente. Los fallos de preparación se auditan después de revertir la lectura; si tampoco puede confirmarse el apunte, no se entrega archivo.

- Se registra todo acto de escritura y toda lectura (fichas, cifras, catálogos, documentos y exportaciones), además de inicios y fallos de sesión, cambios de perfiles, de catálogos, de plantillas, de idiomas y de tema, y subidas no revisadas por el antivirus. Cada apunte lleva quién, cuándo, qué acción, sobre qué identificador y con qué resultado.
- Las escrituras se confirman junto con su auditoría en la misma transacción. Las lecturas también confirman su apunte antes de entregar cualquier dato o comenzar una descarga: si no se puede auditar, no se entrega. Tras revertir una operación fallida, se registra el fallo en una transacción nueva; nunca se intenta conservarlo dentro de la transacción revertida.
- Tabla de sólo inserción: el rol de la app no tiene `UPDATE` ni `DELETE` sobre ella.
- Encadenado: cada apunte guarda la huella SHA-256 de su contenido y de la huella del anterior; las inserciones se serializan para que la cadena sea única. Una orden de comprobación recorre la cadena y señala el primer eslabón roto. La última huella de cada día viaja también en la copia diaria, para poder demostrar que nadie rehízo la cadena entera.
- La auditoría evita nombres y contenido de expedientes, pero los identificadores y los vínculos usuario-expediente siguen siendo datos personales. Su conservación y exportación se someten al DPD dentro de la EIPD, junto con la conservación del expediente [D14]. No se consideran anónimos por quitar los nombres.
- Administración funcional exporta la auditoría para el DPD en un formato abierto (CSV o JSON), con la cadena y la orden para comprobarla. Es un permiso específico, auditado, que no concede acceso a las fichas. El alcance de los vínculos exportados queda sujeto al DPD [D14].
- F5.04 conserva íntegra la cadena al retirar datos personales del expediente. La revisión y la confirmación se registran en la misma transacción; los recibos que reproducen datos del expediente se retiran. No se borra ni reescribe ningún evento. Las fechas y los vínculos conservados siguen sujetos al DPD [D14].
- Los registros técnicos (apartado 9) son otra cosa y no llevan datos personales.

## 7. Consistencia de datos

- Versión optimista: cada expediente (y cada agregado editable) tiene un número de versión. Toda modificación envía la versión leída; si otra persona guardó antes, el servidor responde 409 con la versión actual y la pantalla ofrece ver los cambios (`DISENO_UI_UX.md`).
- Idempotencia: los actos que crean algo irrepetible (aceptar y dar número, cerrar, confirmar un hito o solicitar una emisión) aceptan una clave de idempotencia; la misma clave y la misma entrada devuelven el mismo resultado sin duplicar. Si cambian los datos o la plantilla, la generación usa una nueva clave y crea otra emisión con aviso; reutilizar una clave con contenido distinto se rechaza. Sin el fichero guardado, sólo se repite una descarga si se puede reconstruir desde las mismas versiones de datos y plantilla, nunca con los datos actuales silenciosamente.
- Numeración `equipo-correlativo-año` [D1]: el correlativo se asigna dentro de la transacción de aceptación, con bloqueo del contador de ese equipo y año. Nunca se reutiliza un número ordinario. SEG3 permite liberar una reserva histórica al anular su lote intacto, sin reducir el contador ordinario.
- Un caso de uso, una transacción, con su apunte de auditoría dentro. Excepción de lectura FX4: Consulta termina primero la lectura MVCC de la foto, calcula fuera de la transacción de `etf_app` y confirma su apunte en otra transacción antes de responder. No realiza escrituras de negocio entre ambas.
- Historia de sólo adición: los datos que alimentan la Memoria (composición, ingresos, adultos, menores, factores, subprograma y demás) conservan sus versiones y su intervalo de vigencia. Cada cierre y reapertura queda en un episodio; reabrir nunca borra el cierre anterior. La Memoria usa los datos vigentes del año evaluado, también antes de haber sido cerrada.
- Reglas no decididas y parámetros (plazo de 30 días, duraciones de fase, tramos de duración [D20], tramos y años del SMI [D8], equivalencias de motivos de cierre [D5]) en catálogo versionado con fecha de vigencia. La Memoria de un año usa la versión vigente el 31/12 del año evaluado. Nunca constantes en el código.
- El Anexo 3 o 4 y los documentos generados no se guardan. Cada generación es una emisión: autor, fecha, versión inmutable de plantilla y huella de los datos y versiones usados. El hito es distinto: una persona confirma la fecha que cuenta («firmado el…»). Generar o regenerar nunca cambia por sí solo fechas ni estados. Las rutas que generan documentos son POST. Las plantillas y paquetes de configuración sí se conservan.
- La Memoria provincial se calcula directamente de los datos, con las mismas reglas que la de equipo; no suma Memorias cerradas. Se avisa si la parte provincial difiere de una Memoria de equipo cerrada. Por apartado se definen población y denominador, los desconocidos se cuentan aparte como «sin dato» y un divisor cero muestra 0,00 con aviso. Quien cumple 18 años durante el año cuenta en 12–17 (nota 43).

F3.02c separa el seguimiento del episodio cerrado de intervención. Tiene su
propia versión optimista e historia. Su vencimiento es un acto técnico sin ruta
HTTP: una función acotada del módulo termina sólo los plazos ya cumplidos y el
caso de uso confirma los eventos de auditoría en la misma transacción. La
comprobación se ejecuta al arrancar y cada minuto; si falla, conserva los datos
anteriores y vuelve a intentarlo. La duración se lee del catálogo versionado al
registrar o reabrir el seguimiento; cada reapertura inicia el plazo completo desde
su fecha y conserva la regla anterior en la historia. Los avisos usan la antelación
vigente al consultarlos. El catálogo inicial fija 12 meses y un aviso de 30 días.

SEG3 aplica el plazo de carga en aplicación y PostgreSQL, también mediante un control diferido antes de confirmar. La fecha se interpreta en Europe/Madrid y se usa el reloj real tras las esperas. La configuración administrativa conserva versiones y fechas en una historia propia y registra cada cambio en auditoría. Carga y numeración ordinaria comparten bloqueo por equipo y año; la anulación se serializa por lote. El marcador de actividad posterior es monotónico y no se puede modificar desde la aplicación. Una función acotada de anulación conserva el lote y los identificadores técnicos retirados, sin conceder borrado general al rol de aplicación; los vínculos de otros módulos impiden retirar un expediente con documentos o evaluaciones.

## 8. Textos e idiomas

- Ningún texto visible ni idioma está escrito en el código, ni en Go ni en React: el código sólo usa claves (`expediente.aceptar`). Tampoco se compilan textos dentro del ejecutable ni del JavaScript.
- Cada idioma vive en `idiomas/<código>/` (`es`, `en`…), con `textos/` para pantalla, mensajes, errores y avisos; `catalogos/` para nombres de opciones; y `plantillas/` para Word. El formato concreto (JSON con marcadores y plurales) lo fija el equipo técnico.
- El servidor devuelve errores como clave y parámetros; la interfaz los traduce. Cuando el servidor necesita texto (por ejemplo al rellenar un Word), lo pide al puerto `Textos`.
- Añadir o cambiar un idioma es añadir o cambiar un directorio, sin compilar. La app lee los directorios al arrancar y cuando Administración funcional pulsa «recargar idiomas»; la lista del selector sale de las versiones válidas cuya referencia esté activada.
- Idioma por defecto: castellano. Si falta una clave en un idioma, se muestra la de castellano.
- Fechas, números y plurales se formatean según la localización (en el navegador con `Intl`; en el servidor con `golang.org/x/text`).
- En Docker, `idiomas/` es un volumen montado.
- Comprobaciones en la CI: (1) una prueba busca texto escrito a mano en el código Go y TypeScript (literales en JSX, `aria-label`, `title`, `placeholder`, mensajes de error devueltos) y falla si lo encuentra; (2) otra compara las claves entre idiomas; (3) otra comprueba que toda clave usada en el código existe en `es`.
- Administración funcional crea los textos de cada opción en los directorios de idioma; la base de datos guarda códigos, versiones y referencias, nunca traducciones ni textos de catálogo. Los textos libres del expediente y las narraciones de la Memoria son datos de negocio y siguen en la base de datos con sus permisos.
- Publicación común para idiomas, temas y plantillas: validar y escribir una versión nueva inmutable, con nombre por versión o huella; registrar sus metadatos; activar su referencia y confirmar la auditoría en la misma transacción. Sólo las referencias activadas se sirven. Si falla el registro, el fichero huérfano no se usa y se limpia; si falla la activación, sigue la versión anterior. Las activaciones concurrentes comprueban la versión previa para no perder cambios. Nunca se sobrescribe un fichero activo. Las copias conservan las versiones aún referenciadas.
- Actualizar la distribución (FX8) no es publicar. La base de cada idioma y tema, es decir, todo lo que no cuelga de una carpeta `.versiones`, pertenece a la entrega instalada y se sustituye fichero a fichero de forma atómica. Lo que cuelga de `.versiones` a cualquier profundidad (versiones de idioma y tema, catálogos y plantillas publicadas) sólo se añade si falta. Un fichero de la base se sustituye sólo si su huella, en esa misma ruta, es de una distribución: la del manifiesto `.distribucion.json` de la entrega anterior o la del inventario de todas las versiones históricas de `idiomas/` y `temas/`, generado con Git al construir e incluidas las fusiones. Sólo se retira un fichero que la entrega nueva ya no trae si figura en el manifiesto anterior con la misma ruta y la misma huella; el inventario nunca autoriza a retirar. Lo cambiado a mano se conserva y se avisa una vez. Windows y la demo paran la aplicación antes de preparar. Tras incorporar, se cargan los textos de cada idioma; si hay JSON dañado o claves repetidas, la aplicación no arranca.
- Resolución de textos: versión activa, después la distribución del mismo idioma y después la del castellano. Sólo se completan claves ausentes y nunca se sustituye una existente; un JSON dañado o con claves repetidas sigue siendo un error. La siguen la API de textos, los errores, los Word y el Word de Consulta. Una plantilla de distribución que no esté en la versión de idioma activa se busca en la base del mismo idioma, siempre con la huella registrada en la base de datos; la presentación JSON de la Memoria, que no tiene huella registrada, se busca igual y se valida con su decodificación estricta. Los catálogos no tienen reserva.

## 8b. Temas

- Paquete de tema: `temas/<nombre>/` con un manifiesto (`tema.json`: nombre, versión, autor, versión mínima de la app), un fichero de tokens semánticos (fondo, superficie, texto, acento, aviso, error, tipografía, espaciado, radios, alturas de control) y, si hace falta, fuentes WOFF2 y recursos PNG. No se admite SVG ni otros formatos de recursos.
- El tema son datos. El servidor genera la hoja de estilos con variables CSS a partir de tokens validados; un tema no puede traer CSS libre ni JavaScript. Los componentes de `web/` sólo usan esas variables, nunca colores ni medidas fijas.
- Validación al instalar: formato del manifiesto, tipos y rangos de cada token, contraste mínimo de WCAG 2.1 AA entre los pares texto y fondo, lista cerrada de ficheros permitidos y tamaño máximo. El paquete se sube como ZIP con los mismos controles que cualquier ZIP (apartado 10).
- Administración funcional instala, activa y retira temas desde la app; cada acción queda auditada. El tema activo se guarda en la configuración de la base de datos y se aplica a toda la app sin recompilar ni reiniciar.
- `temas/expedientes-etf/` reproduce la maqueta (Public Sans y Source Serif 4, fondo #f4f2ec, verde petróleo #0f5e63, bordes #dcd8cc, radios de 10 a 14 px, botones de 44 px). Habrá además `temas/alto-contraste/`. Una copia del predeterminado viaja en la imagen: si el tema activo falla, se vuelve a él y la app nunca queda sin estilo.
- En Docker, `temas/` es un volumen montado. Los tokens y su uso en pantalla, en `DISENO_UI_UX.md`.

### Aplicación de escritorio (PWA1)

Chrome y Edge pueden instalar la aplicación y abrirla en una ventana propia, sin pestañas. Go sirve `/manifest.webmanifest`: el nombre y la descripción salen de `idiomas/<código>/textos/pwa1.json`; los colores, de `color.acento` y `color.fondo` del tema seleccionado. El manifiesto usa `display: standalone`, inicio y ámbito `/`, e iconos PNG de 192 y 512 píxeles, normales y maskable. `node scripts/pwa1/iconos.mjs` regenera la marca ETF desde los tokens y la fuente de cada tema y la guarda en `temas/<nombre>/iconos/`. La referencia al manifiesto cambia al elegir idioma o tema. Un tema anterior a PWA1 que no incluya iconos utiliza la marca del tema de reserva; para publicar sus propios iconos se ejecuta el mismo script antes de instalar su nueva versión.

El service worker sólo se registra en contexto seguro (HTTPS o localhost). Atiende únicamente navegaciones GET del mismo origen, excluye `/api` y `/api/` y siempre consulta la red. No usa CacheStorage, IndexedDB ni almacenamiento de sesión. Si falla la conexión, devuelve un aviso fijo traducido, con «Volver a intentarlo», sin datos personales ni expedientes disponibles sin conexión. Los avisos públicos se incorporan al código del worker desde los catálogos de idiomas y los tokens de temas. Cada ventana comunica sólo sus preferencias de idioma y tema, conservadas en memoria por identificador de cliente; no se guarda ninguna respuesta de la aplicación. Las respuestas de manifiesto y worker usan `Cache-Control: no-store`. La CSP añade sólo `manifest-src 'self'` y `worker-src 'self'`; el aviso permite únicamente su estilo y el script de preferencias mediante su huella SHA-256. Este script sólo devuelve idioma y tema al worker para conservarlos también cuando se recarga el aviso tras una suspensión. El worker se registra siempre en `/service-worker.js`, sin preferencias en la dirección, y Go lo sirve idéntico byte a byte mientras no cambien catálogos ni temas: cambiar idioma o tema, abrir otra ventana o una comprobación de actualización del navegador no instalan otro worker. Así se evita el relevo del worker activo, que con la máquina cargada Chrome puede retrasar unos 30 segundos y que, con DevTools enganchado como en Playwright, deja sin respuesta las navegaciones de otras ventanas.

La prueba `node scripts/recorridos/pwa1.mjs <URL-local>` usa Chrome del sistema con un perfil temporal normal. Comprueba el manifiesto, la señal `beforeinstallprompt`, los errores de instalabilidad de Chrome, los iconos, el cambio de tema, el aviso sin conexión en castellano e inglés y CacheStorage vacío tras consultar la API con una sesión de Coordinación. Se ejecuta también dentro de F0.11 y de `make ci`. Antes del recorrido, `scripts/pwa1/preparar_demo.py` comprueba que la URL corresponde a la demo de este worktree y utiliza la preparación común de FX7. Tanto esta preparación como `make demo` paran la aplicación e instalan la distribución en los volúmenes con el incorporador común del paquete Windows (FX8, §8): sustituyen la base, conservan lo publicado y lo cambiado a mano y validan los textos antes de arrancar. F0.11 la ejecuta antes de la matriz de rutas públicas; el recorrido PWA1 también la ejecuta cuando se lanza por separado. Esta preparación no borra expedientes ni modifica versiones publicadas de idiomas o temas.

Go devuelve 503 con `paquete.no_disponible` si faltan los textos obligatorios del manifiesto o del aviso sin conexión, en lugar de emitir contenido vacío. En una instalación real, los textos nuevos se publican mediante el procedimiento de actualización de idiomas; el preparador anterior sólo se usa en la demostración local.

## 8c. Plantillas Word

- Las plantillas oficiales de las plantillas de `idiomas/` (Actas, Oficios y Anexos 1, 2, 5, 6, 11 y 12, Tabla SPAF y Memoria) se convierten una vez a `.docx` con marcadores; las que llegaron en `.odt` se pasan con LibreOffice en el puesto de desarrollo y se revisan a mano. Los textos fijos de la plantilla no se tocan.
- Viven en `idiomas/<código>/plantillas/`, dentro del volumen de idiomas. Administración funcional sube versiones nuevas desde la app; cada versión inmutable queda registrada (huella, fecha de vigencia, quién la subió) y se valida antes de activarla (que todos los marcadores esperados existen), mediante la publicación del apartado 8.
- PL1 amplía el contrato de F1.11: `GET /api/admin/plantillas?tipo=…&idioma=…` devuelve la referencia y su versión; `POST` en la misma ruta recibe el DOCX como cuerpo, con `Content-Type` de Word y parámetros `tipo`, `idioma`, `version` y `version_previa`; desde PL1 guarda sin activar. Sólo Administración funcional; la escritura exige CSRF. PL1 añade la pantalla `/admin/plantillas`: `GET` sin parámetros devuelve todas las versiones por tipo e idioma, incluida la activa y su historial. `POST /api/admin/plantillas/versiones` usa el mismo cuerpo y parámetros para subir sin activar; confirma antivirus limpio y marcadores válidos. Ante diferencias devuelve listas de marcadores faltantes y sobrantes. `POST /api/admin/plantillas/{id}/activar` recibe `idioma` y `version_previa`, comprueba huella y contrato y activa tras la confirmación de la persona. El historial incluye autor y fecha; las versiones iniciales sin esos datos se identifican como tales.
- El límite es de 10 MiB (10 × 1024 × 1024 bytes). ClamAV debe confirmar limpieza; si falta o no responde, se rechaza. Se comprueba el formato real de Word sin macros, recursos externos ni contenido activo, con límites de ZIP y XML. Los campos y bloques deben conservar el contrato de marcadores de la plantilla actual, también cuando Word divide un marcador entre fragmentos de texto.
- Cada publicación se escribe exclusivamente en `idiomas/<código>/plantillas/<tipo>/.versiones/<UUID>/v<versión>.docx`, sin sobrescribir. La función de activación bloquea la referencia, comprueba su versión, registra la nueva plantilla y la activación y audita en una transacción. `documento/0012` separa guardar y activar y conserva cada activación, también las restauraciones; el archivo previo de `documento/0004` sigue legible. Un conflicto devuelve 409 con la versión actual. Los ficheros históricos se conservan; una confirmación incierta se consulta bajo el mismo bloqueo antes de limpiar el huérfano. Las plantillas publicadas se leen desde esa ruta aunque cambie el paquete activo del idioma.
- El paquete inicial incorpora las 28 versiones corregidas de F3.03 (`1.0.1` y `1.1.1`). `documento/0013` registra sus huellas y activa las 26 referencias vigentes en instalaciones limpias; el Acta de Aceptación usa `1.1.1`. En una actualización conserva las versiones, publicaciones y restauraciones decididas por Administración. Los originales siguen disponibles para emisiones históricas.
- RESP1 (`documento/0016`) registra diez versiones nuevas es/en: `1.1.2` para el Acta de Aceptación y `1.0.2` para Acta de Cierre, Acta de No Aceptación, Oficio de Lista de Espera y Tabla SPAF. Sustituye sólo las referencias de distribución intactas. Conserva bytes, huellas y referencias anteriores, así como las decisiones administrativas; se comprueba también una colisión de versión publicada por Administración. La Tabla SPAF muestra sólo el municipio, según D19 del 14/10/2026.
- Los valores se escapan como XML al rellenar, para que un nombre con `<` o `&` no rompa el documento. En los anexos con controles de contenido se rellena el control para que el Word siga siendo editable.
- Los Word generados tienen que ser accesibles (`DISENO_UI_UX.md`).

## 9. Rendimiento y observabilidad

- Objetivos: lecturas por debajo de 300 ms en el p95 con volumen real; Memoria provincial en menos de 3 s; ninguna consulta dentro de un bucle.
- Volumen de referencia: 12 equipos, hasta 60 profesionales, unos 300 expedientes al año, hasta 15 menores por expediente, 25 años de conservación.
- Cómo se mide: una prueba de carga en la CI o antes de cada entrega sobre una base sintética con ese volumen; `EXPLAIN` de las consultas de bandeja, ficha, Tabla SPAF y Memoria en una copia local. Las consultas obtienen conjuntos históricos mediante las vistas de contrato; las reglas de cálculo viven en el dominio de `estadistica`. Se pueden agregar datos en SQL cuando se compruebe su equivalencia con las pruebas del dominio, sin trasladar allí la única definición de una regla.
- Registros técnicos en JSON con `log/slog`: identificador de petición, ruta, código de respuesta, duración e identificadores internos. Nunca nombres, contactos, textos de la ficha ni contraseñas.
- Página de salud para Informática: estado de PostgreSQL, AD, ClamAV, idiomas cargados, tema activo, plantillas y versión de la app, sin datos personales. Una ruta de «vivo» y otra de «listo» para Docker.
- Métricas para un sistema de monitorización: sin decidir; depende de lo que use Informática.

## 10. Seguridad

Marco: Reglamento General de Protección de Datos (RGPD), Ley Orgánica de Protección de Datos (LOPDGDD), Esquema Nacional de Seguridad (ENS) en nivel ALTO de confidencialidad, accesibilidad del Real Decreto 1112/2018. Hay datos de menores, violencia de género, salud mental y adicciones (categorías especiales).

- Secretos fuera del repositorio. Análisis de secretos (`gitleaks` en local y en la CI).
- Cifrado: HTTPS siempre; LDAPS; disco y copias cifrados. La conexión de la app a PostgreSQL va por una red interna de Docker sin salida al exterior.
- Cabeceras: política de seguridad de contenido (CSP) estricta sólo con el propio origen, sin estilos ni scripts en línea; `Strict-Transport-Security`; `X-Content-Type-Options: nosniff`; `frame-ancestors 'none'`; `Referrer-Policy: no-referrer`; `Cache-Control: no-store` en la API.
- Límites: tamaño de cuerpo por ruta, tiempos máximos de lectura y escritura del servidor HTTP, tiempo máximo de cada llamada a AD y a ClamAV, límite de intentos de inicio de sesión.
- Política por tipo de subida. Límites técnicos iniciales, configurables por tipo; siempre se comprueba el contenido real, no sólo la extensión:

  | Tipo | Límite de entrada | Formatos | Antivirus local | Persistencia |
  | --- | --- | --- | --- | --- |
  | Anexo 3 o 4 | 10 MB | DOCX, ODT o PDF | Rechazar infección; si no responde, aviso y auditoría según decisión vigente | Nunca; sólo datos confirmados |
  | Plantilla Word | 10 MB | DOCX sin macros | Debe superar la revisión antes de publicar | Versión inmutable en `idiomas/<código>/plantillas/` |
  | Paquete de tema | 20 MB | ZIP: manifiesto y tokens JSON, PNG y WOFF2; sin SVG | Debe superar la revisión antes de publicar | Versión inmutable en `temas/<nombre>/` |
  | Paquete de idioma | 20 MB | ZIP: JSON y plantillas DOCX validadas | Debe superar la revisión antes de publicar | Versión inmutable en `idiomas/<código>/` |
  | Hoja de carga | 10 MB | XLSX sin macros o CSV | Debe superar la revisión antes de importar | Fichero descartado; sólo datos validados, en producción si son reales |

  Los archivos comprimidos admiten como máximo 100 MB expandidos y 1.000 entradas. Se rechazan rutas absolutas, saltos de directorio, enlaces, archivos anidados no permitidos y contenidos ajenos a la lista de cada tipo. Estos límites no sustituyen el límite de entrada.

- Seguridad de publicación de plantillas DOCX: la candidata completa supera los límites de ZIP y XML y el ensayo del motor. La predecesora se lee sólo para comparar el contrato de marcadores de `word/document.xml`, con límites propios; sus defectos históricos no autorizan contenidos peligrosos en la candidata. Los campos complejos (`w:fldChar`/`w:instrText`) sólo admiten las instrucciones completas `PAGE`, `NUMPAGES` y `SECTIONPAGES`, sin opciones, anidamientos ni instrucciones huérfanas. Se concatenan las instrucciones divididas entre runs antes de validarlas. Se rechazan todos los demás campos, incluidos DDE, INCLUDETEXT, HYPERLINK y MACROBUTTON, y los campos simples. Las relaciones externas sólo admiten el tipo hipervínculo y destinos `https://` válidos, sin credenciales; se rechazan `file:`, `smb:`, `http:` y rutas UNC, así como cualquier otro tipo de relación externa. Se rechazan macros (`vbaProject`), OLE, ActiveX, `w:object`, `w:control` y `altChunk`. Estos permisos sólo conservan la paginación y los enlaces web oficiales; el servidor no los ejecuta ni descarga su contenido.

- Recorrido del Anexo 3 o 4:
  - Word (`.docx`), LibreOffice (`.odt`) o PDF, como mucho 10 MB, cortado en la frontera HTTP.
  - Se lee en memoria con un lector de partes en flujo. No se usa el análisis de formularios de Go que vuelca a ficheros temporales por encima de cierto tamaño: el fichero nunca toca el disco. Se verifica en todo el recorrido: proxy sin volcado de cuerpos, antivirus en flujo sin temporales persistentes, aplicación y lector sin temporales, registros ni volcados que contengan el anexo. La prueba abarca también errores, cancelaciones y caída del antivirus. Si una herramienta necesita espacio temporal, sólo memoria sin intercambio a disco; no basta borrar después un fichero persistido.
  - Se comprueba el tipo real por su contenido y no por la extensión. Word: ZIP válido con su estructura, sin macros (`vbaProject.bin`, `.docm` rechazado), límite de número de entradas y de tamaño descomprimido contra bombas ZIP. PDF: cabecera válida y límites de páginas y objetos. El lector se ejecuta en un proceso efímero con CPU de 3 s, tiempo de 5 s, datos de 256 MiB y espacio virtual de 4 GiB; conserva límites heredados más estrictos. Go usa un objetivo blando de memoria de 64 MiB. El diagnóstico interno del hijo conserva sólo líneas técnicas permitidas de un stderr acotado a 4 KiB; HTTP y auditoría reciben únicamente la clave de error.
  - RESP4 admite LibreOffice (`.odt`, `application/vnd.oasis.opendocument.text`) con la misma política de antivirus y descarte que Word. El ZIP se limita a 256 entradas y 32 MiB expandidos; las partes XML, a 8 MiB, 500.000 tokens y profundidad 64, con comprobación previa de atributos. Rechaza rutas inseguras, duplicados, enlaces, macros, objetos incrustados, cifrado, DTD y repeticiones de tablas. Los controles se interpretan como datos; no se ejecutan ni se siguen enlaces. `expediente/0026` amplía el CHECK de `lectura_anexo.formato` a `docx/odt/pdf`; conserva permisos y RLS y permite confirmar ODT con su procedencia original. Los prefijos de conversión y los marcadores se limpian; las filas vacías no generan personas. Las fechas de dos cifras se leen con el criterio de Go: 00–68 pertenecen a 2000–2068 y 69–99 a 1969–1999; se muestran completas para revisión.
  - Las muestras inventadas recibidas del Programa se ensayan también como PDF exportado por LibreOffice y firmado con certificado autofirmado local. La firma no cambia la lectura del texto; la aplicación no valida su autenticidad ni la confianza del certificado. En esos PDF quedan para revisión manual las casillas de crisis, actitud y documentación, y la situación económica con glifos desordenados. Los PDF escaneados siguen completándose a mano. Fixtures y valores esperados en `internal/adaptadores/anexos/odt/testdata/`; [evidencia RESP4](../recorridos/RESP4.md).
  - Pasa por ClamAV. Mientras sube y mientras el antivirus revisa, la pantalla lo dice («Revisando el fichero con el antivirus…»). Con virus, se rechaza y se audita. Si ClamAV no responde en su tiempo máximo, se acepta con aviso y queda anotado en la derivación y en la auditoría que no se revisó.
  - Los valores extraídos se tratan como texto no confiable y los confirma siempre una persona (`FLUJO_APP.md`).
- XML seguro: el lector de Go no resuelve entidades externas; además se rechaza cualquier documento con declaración de tipo (DTD).
- Hoja de 2026 (F2.08b): lector en memoria, 100 casos por equipo, XLSX con una sola hoja visible, sin fórmulas ni recursos externos; máximo de 64 entradas ZIP y 40 MB descomprimidos. CSV UTF-8 con coma o punto y coma. Antivirus obligatorio también al confirmar; formato y API en ESQUEMA_BACKEND §7.2.
- Paquetes de tema: mismos controles de ZIP, lista cerrada de ficheros y sin código ejecutable (apartado 8b).
- La aplicación no integra correo, firma, almacenamiento externo ni inteligencia artificial. Rige la autorización expresa de Repositorio público y CI sin datos reales y GitHub Actions del apartado 1; nunca se suben datos personales ni se envía código a análisis de seguridad en la nube.
- Análisis estático en local: Semgrep con reglas locales (`p/golang`, `p/javascript`, `p/typescript`), con las métricas desactivadas y sin subir nada a su nube; `go vet` y `staticcheck`. Revisión de dependencias local con bases de avisos descargadas o un espejo interno; no se envían inventarios del proyecto a servicios de análisis en la nube. Las descargas autorizadas de dependencias mantienen sus versiones y sumas fijadas.
- Antes de cada PR que toque autenticación, permisos, SQL, ficheros subidos o fronteras HTTP: revisión de seguridad centrada en lo cambiado con la skill `security-audit`, en local. La auditoría completa, antes de producción.
- Revisión completa F5.02 (14/10/2026): informe para el Comité de Seguridad (informe específico de cada instalación) con hallazgos, registro de riesgos y puertas 0-5. La prueba de intrusión repetible contra la demo está en `scripts/seguridad/intrusion_demo.py`.
- Límites añadidos en F5.02 contra la amplificación de memoria: los cuerpos JSON con listas (carga histórica, familia, menores, intervención y confirmación de derivación) y el JSON de las celdas de la hoja se recorren antes de decodificarlos y se rechazan si una lista pasa de 500 elementos o el total de 20.000; ninguna celda de hoja pasa de 64 KiB y se procesan como mucho dos hojas a la vez; cada parte XML de un anexo Word se rechaza con etiquetas de más de 64 KiB o 128 atributos, o más de 300.000 atributos; en las plantillas, toda parte declarada como XML debe llamarse `.xml` o `.rels`. El proceso de la app fija `GOMEMLIMIT=192MiB` por debajo del límite de 256 MiB del contenedor.
- Los rechazos anónimos en rutas protegidas gastan el cupo de peticiones del origen antes de escribir en la auditoría; fuera de cupo se responde 429 sin apunte.
- Contenedores: usuario sin privilegios, sistema de ficheros de sólo lectura salvo los volúmenes, `no-new-privileges`, sin capacidades extra, imágenes fijadas por versión y huella. Sólo `app` expone puerto; `postgres` y `clamav` quedan en una red interna.
- Conservación: 25 años desde el cierre, configurables. F5.04 marca al cerrar o consultar la lista; dos personas distintas de Administración revisan y confirman la retirada de datos personales actuales e históricos, sin seguimiento activo. Conserva cifras y cadena de auditoría; las fechas y vínculos restantes siguen sujetos al DPD [D14]. No requiere un cron de borrado. Procedimiento técnico y copias en [el manual de sistemas](../sistemas/OPERACION.md#d-conservación-y-retirada-de-datos-personales).


RESP2 amplía la política transitoria de anexos a actas de carga histórica: PDF, DOCX u ODT de hasta 10 MB, sólo memoria y tuberías, infección bloqueada y caída de ClamAV con aviso visible y auditado. Sólo se guardan datos confirmados por la persona; no se verifica firma digital ni se usa OCR externo.

## 11. Pruebas e integración continua

F0.11 espera cuando el acceso devuelve HTTP 429 y recarga únicamente su GET; conserva el límite de peticiones y no repite el POST de inicio de sesión. Main corrige la prueba de Traslado desde espera para que tome la fecha del reloj inyectado, evitando el fallo de una fecha fija tras medianoche.

La CI completa incluye el ensayo destructivo de restauración de la demo local (SEG4), con revocación de sesiones, revisión de identidad, PostgreSQL/RLS, lectura auditada en Go y recorrido Chrome. Sólo afecta al proyecto Docker derivado del worktree y a datos inventados.

- Repositorio privado en GitHub (`etf-expedientes`). En la CI sólo hay datos inventados y verosímiles, sin la palabra «ejemplo».
- FX5: los ensayos de migraciones comparten `scripts/ci/ensayos.py`. Esperan una consulta TCP autenticada con `etf_migrador` antes de migrar, porque el socket de PostgreSQL responde durante la inicialización provisional. Un fallo muestra stdout y stderr acotados, con credenciales y detalles SQL ocultos. `make ci` reproduce también esa carrera en un contenedor desechable; [evidencia FX5](../evidencias/FX5.md).
- FX6 incorpora a `make ci` los ensayos completos de Catálogo y del Expediente inicial. Comprueban las opciones y versiones actuales, la actualización desde cortes históricos con sus dependencias, los permisos y el antivirus; no se modifican catálogos ni migraciones. Ambos duran menos de un minuto en la medición local; [evidencia FX6](../evidencias/FX6.md).
- El trabajo de GitHub Actions dispone de 25 minutos para la compilación en frío y los recorridos completos. Se conserva cada límite interno de las pruebas; el presupuesto anterior de 15 minutos canceló la matriz HTTP después de superar las migraciones, según la evidencia FX5.
- En local, lo cambiado: pruebas del paquete tocado, comprobaciones de textos si se tocan idiomas o pantallas, ensayo de la migración si hay SQL.
- Puertas progresivas en la CI, obligatorias desde que exista la pieza correspondiente (detalle en `PLAN_IMPLEMENTACION.md` §4.3). La documentación empieza con enlaces y coherencia; al crear el esqueleto se añaden compilación y tipos; con cada capacidad se activan sus comprobaciones. La fase 0 no exige recorridos de fases aún sin construir. Al terminar la aplicación, la puerta completa incluye:
  - `gofmt`, `go vet`, `staticcheck` y `go test -race` de todo;
  - prueba de arquitectura (reglas de dependencia del apartado 2.1);
  - migraciones desde cero sobre PostgreSQL 18 en contenedor y pruebas de RLS (cada perfil sólo ve lo suyo; Consulta sólo recibe cifras; Administración no lee fichas y sólo exporta vínculos mediante el permiso de auditoría);
  - prueba de la cadena de auditoría;
  - TypeScript sin errores, `eslint` y pruebas de componentes;
  - pruebas de textos (literales en el código, claves iguales entre idiomas, claves usadas existentes) y validación de los temas incluidos;
  - pruebas de lectura del Anexo 3 y 4 y de generación de cada Word con datos inventados;
  - recorrido de punta a punta con Playwright y Chrome: de la derivación a la Memoria;
  - F0.11, integrado también en `make ci`, comprueba entrada, alta y permisos con Chrome del sistema contra `make demo`; obtiene todos los métodos del registro INT5 y ensaya RLS con filas pobladas y todos los perfiles. `make recorridos` ejecuta además F3.06/F3.07: salidas, cierre, traslado, seguimiento y seis tablas SPAF con cifras manuales; esta regresión queda fuera de CI mientras expone fallos de producto. F2.07 añade las nueve pestañas de la ficha, conflicto sin perder el formulario, aislamiento HTTP/RLS y lectura histórica; exige menores y fase al corte en SPAF y termina con error mientras falte esa conexión. También permanece fuera de CI. F4.09 incorpora a `make recorridos` la Memoria integrada: 28 apartados, cifras manuales por equipo y provincia, Word, cierres, corrección, aviso de discrepancia por API, aislamiento y reapertura; alcance en [F4.09](../recorridos/F4.09.md). Alcance y límites en [F2.07](../recorridos/F2.07.md), [F0.11](../recorridos/F0.11.md) y [F3.06/F3.07](../recorridos/F3.06-F3.07.md);
  - F5.03 añade a `make recorridos` una auditoría axe-core local, fijada por versión y suma npm, de todas las rutas web INT5 por perfil, idioma, tema y tamaño. Rechaza infracciones serias o críticas; no sustituye la revisión manual ni los Word. Informes y límites en [F5.03](../accesibilidad/README.md);
  - CA1, integrado en `make ci`: `scripts/recorridos/carga.mjs` comprueba alta histórica, reapertura, CSV/XLSX, incidencias, recuperación idempotente y consulta recuperable de no aceptados con Chrome del sistema; antes de abrir Chrome verifica los textos es/en de CA1 y renueva únicamente la reserva de ese módulo en la demo local del worktree si conserva textos anteriores;
  - Semgrep local, `gitleaks` y revisión de dependencias;
  - compilación de la imagen de Docker.
- Revisión independiente obligatoria en los cambios de SQL, permisos y seguridad. El equipo de mantenimiento revisa e integra.
- Prueba de aceptación de la Memoria: datos inventados y resultados esperados por apartado validados por el Programa. No se reciben datos reales en desarrollo ni preproducción. D22 queda resuelta el 14/10/2026: el Programa indica que las cifras citadas no corresponden a su Memoria y confirma la nota 43. Las pruebas usan resultados calculados aparte con datos inventados. El modelo estadístico y las pruebas de cálculo, lector PDF y motor Word empiezan en fase 0.

F5.07 prepara un calendario de formación con `scripts/recorridos/reloj_video.py`: una compilación local mediante overlay de Go sustituye los relojes de negocio y lee una fecha en el tmpfs de la demo. Conserva el reloj real de identidad, los puertos y las reglas de negocio; no modifica los ficheros Go de producción. Las consultas de vigencia de familia, menores e intervención usan también la fecha de formación en ese overlay, manteniendo los filtros de acceso y las reglas de guardado. Para poder repetir la práctica tras reinstalar la demo, la lectura del catálogo inicial toma como publicación su inicio de vigencia. Esa proyección sólo existe en el overlay; conserva las publicaciones posteriores y no escribe ni altera las versiones del catálogo en la base. Playwright fija también la fecha del navegador sin detener sus temporizadores. `scripts/recorridos/aula.mjs` abre tres ventanas de Chrome del sistema con sesiones separadas y sincroniza ese calendario en las de equipos y Coordinación para practicar los formularios; Administración conserva el reloj real para gestionar permisos. Guarda los Word en la carpeta privada de la demo. El script de restauración recompila la aplicación ordinaria. El calendario permite grabar recepción, documentos e hitos en orden y consultar después el mes terminado; sólo se usa con datos inventados.

## 12. Despliegue

- Paquete de pruebas Windows (WIN1): `make paquete-windows` genera en `bin/paquete-windows/` un ZIP versionado y, con NSIS 3.10 (`despliegue/windows/instalador.nsi`), el asistente `Expedientes-ETF-<versión>-instalador.exe` de la misma carpeta, cada uno con su SHA-256; `make instalador-windows` rehace sólo el `.exe` desde un ZIP ya construido y comprobado. El `.exe` se instala por usuario sin administrador (`RequestExecutionLevel user`) directamente en la copia estable `%LOCALAPPDATA%\Expedientes-ETF\pruebas\paquete`, para no duplicar las imágenes; crea la marca de la carpeta privada si falta y se detiene si esa carpeta tiene archivos ajenos; ejecuta `actualizar.cmd` si existe `entorno.env` y si no `instalar.cmd`, en una consola visible de 64 bits; registra «Expedientes ETF (pruebas)» en `HKCU\…\Uninstall\Expedientes-ETF`; su desinstalador ejecuta lo mismo que `desinstalar.cmd` (en una orden en memoria, porque el script borra el propio `.cmd`), sólo continúa si se ha desinstalado y borra únicamente los archivos que copió. Textos del asistente en `idiomas/{es,en}/textos/windows.json` (`windows.instalador_*`), convertidos a `LangString` al construir. Sin firma de código; firma opcional con `osslsigncode` mediante `ETF_FIRMA_CERTIFICADO` y `ETF_FIRMA_CLAVE` fuera del repositorio. Los accesos usan un lanzador .exe (`Expedientes ETF.exe`, `despliegue/windows/lanzador.nsi`) que ejecuta `abrir.ps1` con la PowerShell de `System32`, para no activar las reglas de los antivirus que bloquean scripts lanzados desde accesos directos; al actualizar, los accesos anteriores que apuntaban a PowerShell se reconocen y se cambian al lanzador. Los accesos los gestiona sólo código NSIS (`despliegue/windows/accesos.nsh`): el lanzador con `/accesos crear` o `/accesos quitar` (código de salida 0, 1 o 2 si el acceso es de otra instalación), el `.exe`, que comprueba los accesos antes de copiar y los migra al actualizar, y su desinstalador, que los borra al final, después de PowerShell. Ningún script `.ps1` ni `.cmd` abre, crea ni borra un acceso directo, porque algunos antivirus de comportamiento (Fibratus, EDR corporativos) matan la secuencia «intérprete que abre un `.lnk`, crea un ejecutable o script en AppData y lanza otro intérprete»; los scripts sólo ejecutan el lanzador y esperan su código. Incluye imágenes Linux, LDAP de pruebas, PostgreSQL, ClamAV y la siembra de `despliegue/demo`; no necesita herramientas de desarrollo en Windows. Instalación con WSL2 y Podman gratuito en modo rootful; `instalar.cmd` instala lo que falte (WREQ): WSL2 con `wsl --install --no-distribution` tras pedir permiso con UAC, y si Windows pide reiniciar deja una entrada RunOnce de la cuenta que relanza la instalación al volver a entrar; Podman 6.1.3 con su MSI oficial por usuario (`MSIINSTALLPERUSER=1`, sin permiso de administrador); la máquina WSL con `podman machine init --rootful --cpus 2 --memory 4096` (Podman descarga su imagen del sistema la primera vez) y Compose 5.5.1, copiado a la carpeta privada y usado mediante `PODMAN_COMPOSE_PROVIDER` sólo si Podman no encuentra otro proveedor. Los dos instaladores viajan en el ZIP con versión y SHA-256 fijados en `paquete.py`, se descargan al construir y se comprueban de nuevo en Windows antes de ejecutarlos. Con Rancher no se instala nada; alternativa Rancher Desktop con Moby. Lanzadores `.cmd` y PowerShell 5.1/7 para instalar, parar, actualizar, reiniciar y desinstalar. Crea accesos de escritorio e Inicio con icono generado desde los tokens; abren Edge con `--app=http://localhost:<puerto>` y Chrome como alternativa. Los accesos usan una copia privada verificada del paquete, conservan el puerto y pueden arrancar la demostración parada ejecutando las funciones del instalador en su propia consola; desinstalar elimina sólo sus accesos, archivos registrados y volúmenes y conserva archivos añadidos por la persona. Sólo publica en `127.0.0.1`; por defecto usa el puerto 18083. Contraseñas aleatorias y cuentas privadas en `%LOCALAPPDATA%\Expedientes-ETF\pruebas`, fuera del paquete, con permisos de la cuenta de Windows. Actualizar conserva volúmenes y credenciales, sustituye la base de idiomas, plantillas y temas con la regla FX8 (§8), conserva lo publicado y lo cambiado a mano, valida los textos y ejecuta FX2 antes de migrar. Con Podman, si la máquina está encendida pero falta la tubería de Compose, el instalador reinicia la máquina una vez y, si no basta, lanza el `win-sshproxy.exe` de Podman con la conexión local por defecto; anota su identificador y hora de inicio en `proxy.json` de la carpeta privada para cerrarlo después sin confundirlo con otro proceso. Si la máquina se está encendiendo (Podman Desktop la enciende al iniciar sesión), se espera hasta 3 minutos a que termine en lugar de pararla o arrancarla encima; un `machine start` que falla porque otro programa la ha empezado a encender también espera. Antes de reiniciar la máquina por falta de tubería se esperan 15 segundos. El acceso registra sus pasos y errores en `registro-abrir.log` de la carpeta privada, con las contraseñas de `entorno.env` y `usuarios.json` sustituidas, y muestra los fallos en un aviso por encima de las demás ventanas; si el aviso no puede mostrarse, la consola espera una tecla. Parar y desinstalar apagan la máquina de Podman con límite de 2 minutos y cierran el proxy propio si la máquina no tiene otros contenedores en marcha en ninguna de sus dos conexiones (con y sin permisos de administrador); si ninguna responde, no la apagan y avisan. No se instala ninguna tarea al cerrar sesión o apagar: sin permisos de administrador no hay un mecanismo fiable, y la guía pide parar la aplicación antes de apagar Windows. Con la máquina ya apagada, parar sólo cierra la ventana y el proxy propio; actualizar, reiniciar y desinstalar la encienden si hace falta. Ninguna espera es indefinida: el permiso de administrador para WSL espera como máximo 30 minutos y el aviso de error del acceso se cierra solo a los 10 minutos. Lo que instala `instalar.cmd` (WSL, Podman con el código de producto de su MSI y la máquina creada) queda en `instalado.json` de la carpeta privada. Desinstalar sólo ofrece quitar lo anotado, tras escribir QUITAR y sin contenedores ajenos en la máquina: borra la máquina con `podman machine rm --force` y después Podman con `msiexec /x <código>`; nunca quita WSL ni lo que ya existía, y retira la entrada RunOnce, los accesos, la copia de Compose y los archivos propios de la carpeta privada. No se cambia ninguna otra configuración global de Windows. Instalar, abrir y actualizar esperan hasta cinco minutos a que clamd analice limpio un texto de prueba antes de sembrar y abrir. Reinstalar el ZIP anterior no revierte la base, porque su incorporador omite los ficheros existentes; reiniciar exige confirmación escrita y borra sólo la demostración. Guía en [despliegue/windows](../../despliegue/windows/LEEME.md) y evidencia, con los límites del ensayo, en [WIN1](../sistemas/WIN1.md). Datos reales sólo en producción.

- Entornos: desarrollo (Docker Compose en el puesto, datos inventados), preproducción (máquina virtual de 2 procesadores virtuales y 4 GB, datos inventados) y producción (4 procesadores virtuales, 8 GB, unos 60 GB de disco; al no guardar ficheros sobran los 100 GB del primer informe). El responsable del producto sube a producción después de probar en preproducción.
- Demostración local: `make demo` y `make demo-parar`, con [instrucciones propias](../../despliegue/demo/LEEME.md), directorio LDAPS de pruebas y siete usuarios inventados. El archivo adicional de Compose activa desarrollo sólo en esta demostración; el Compose base conserva su comportamiento. Los catálogos y la plantilla de Acta se cargan al migrar; los perfiles y el firmante se asignan después mediante la API auditada. Antes de arrancar o actualizar, la preparación para la aplicación e instala en los volúmenes propios la distribución con la regla FX8 (§8), sin exigir `make demo-limpiar`. Su ensayo automatizado en `make ci` parte de una demo existente con textos anteriores a SEG1 y una versión de idioma activa antigua: comprueba el 503 previo, el Word de Consulta con 200 y los textos correctos, la emisión Word con una plantilla nueva y las huellas intactas de lo publicado. El puerto local se puede elegir con `ETF_DEMO_PUERTO` y queda guardado. `make demo-comprobar` verifica accesos; `make demo-recorrido` comprueba entrada en Chrome del sistema, subida del Anexo 3, aceptación y descarga del Acta por API, con idempotencia. Las pantallas de coordinación siguen pendientes. Las familias sintéticas no se cargan mientras no exista su cargador. Secretos y certificado se generan en local y Git los ignora; parada elimina contenedores y redes y conserva los datos, `make demo-limpiar` borra sólo la demostración.
- Compose (F0.03): `compose.yaml` y `Dockerfile` en la raíz; `app`, PostgreSQL 18.4 (volumen de datos propio) y `clamav` con `clamd`. Servicio puntual `migrar` separado de `app`, con sus propias credenciales; Compose exige su salida correcta antes de arrancar la app. Imágenes externas fijadas por versión y huella. Instrucciones y evidencia en `despliegue/base/`. Volúmenes montados: `idiomas/` y `temas/`, que incluyen plantillas y paquetes publicados. Los anexos, las hojas de carga y los documentos generados no tienen almacenamiento persistente.
- Arranque local F0.03: sólo app publica `127.0.0.1:18083` (configurable). PostgreSQL y ClamAV quedan en la red interna; app añade una red de entrada sin enmascaramiento de salida para que Docker 29.5.2 publique el puerto. Todos usan usuario sin privilegios, raíz de sólo lectura, capacidades vacías y `no-new-privileges`; datos PostgreSQL persistentes y temporales en memoria. Idiomas y temas se montan en sólo lectura, con reserva incluida en imagen; su escritura administrada se incorporará con F0.14c. Las subredes se configuran en `.env.plantilla`.
- FX2 añade la preparación idempotente `despliegue/base/preparar_roles.sh` después de la inicialización y antes de las migraciones. Compose ejecuta el servicio puntual `preparar-roles` antes de `migrar`, también con volúmenes anteriores a F4.07. En una transacción con bloqueo crea los roles ausentes y sincroniza atributos y contraseñas desde el entorno privado; no modifica datos ni el historial de migraciones. `etf_calculo_memoria` existe siempre; `ETF_CALCULO_MEMORIA_CONTRASENA` habilita su conexión y, si falta o está vacía, queda sin inicio de sesión y sin contraseña. Una pertenencia técnica inesperada bloquea la preparación. El servicio no tiene red y usa sólo el socket Unix temporal de PostgreSQL, con permiso 0700 y propietario 70:70. Sólo esos dos servicios montan el socket. El instalador conserva su contraseña TCP retirada; app no recibe credenciales de instalación, migración ni copias. Cambiar el entorno privado y repetir la preparación permite rotar credenciales antes de recrear los consumidores.
- ClamAV usa las firmas incluidas en la imagen, copiadas al volumen nuevo, sin conexiones externas ni FreshClam. El ensayo avisa de firmas antiguas: el suministro local y la actualización periódica siguen pendientes (§13), antes de producción.
- Subir una versión (guion previsto en `despliegue/`), fuera del horario de 8 a 15 h:
  1. copia previa: volcado cifrado de la base de datos y copia de `idiomas/` y `temas/`;
  2. descarga de la imagen nueva, etiquetada con su versión;
  3. ejecutar `migrar` con `etf_migrador`, conservando su dependencia de `preparar-roles`, por dependencias declaradas, y sólo si ambos terminan correctamente arrancar la app sin credenciales de migración;
  4. comprobación: la página de salud tiene que decir «listo» en un tiempo dado y una prueba de humo tiene que pasar;
  5. si falla, vuelta atrás automática: se arranca la imagen anterior y, si la migración cambió algo que la versión anterior no entiende, se restaura la copia previa. Para que casi nunca haga falta restaurar, las migraciones se escriben compatibles con la versión anterior (primero se añade, en otra entrega se quita).
- Una variable de configuración nueva que falte nunca impide arrancar (apartado 3).
- Copias: diarias, cifradas y fuera del servidor; pérdida máxima tolerada, un día. Destino por fijar con Informática. Hay que ensayar una restauración antes de producción y después de vez en cuando.
- Parada tolerable de unas horas; uso en horario laboral.
- Hueco: el primer informe proponía un nginx delante con el certificado de la entidad. Con Docker Compose falta decidir quién termina el HTTPS: un proxy dentro del propio Compose, la propia app o un proxy de la entidad. Deciden responsable del producto e Informática.

## 13. Integraciones externas

**Excepción autorizada: voz con edge-tts.** El responsable del producto autoriza el 14/10/2026 edge-tts 7.2.8 (`~/.local/bin/edge-tts`) y `es-ES-AlvaroNeural` para curso, ayuda y manuales. `make voz` envía a Microsoft sólo texto editorial limpio, sin datos de expedientes, de PostgreSQL ni entradas de las personas. Genera MP3 mono, 24 kHz y 48 kbps en `bin/voz/`, fuera de Git, con caché por SHA-256 de voz y texto y manifiestos por sección. La aplicación y el curso no llaman a Microsoft en ejecución. La CI usa un sintetizador simulado local y nunca ejecuta edge-tts. Docker y el paquete Windows copian las grabaciones existentes; su ausencia oculta el lector y conserva la lectura. El lector sólo aparece en castellano. [Operación y estado de la generación real](../sistemas/VOZ.md).

Los manuales de Profesionales, Coordinación y Administración, con la ayuda común, se convierten al construir a HTML estático saneado con HTML crudo desactivado. Los datos en `docs/manuales/web/` se entregan junto a sus fuentes para la construcción Docker sin Python. El módulo VOZ sirve únicamente recursos editoriales de una lista cerrada, con sesión, bajo `/manuales/recursos/` y `/voz/`; CSP permite `media-src 'self'`. La pantalla `/ayuda` muestra los manuales de los perfiles de la sesión; Consulta conserva un estado vacío. Sistemas y programación no se publican en la app. La web y el paquete SCORM incluyen reproducción local de página y sección cuando hay grabaciones vigentes.

Las grabaciones de manual se filtran por los perfiles de la sesión, tanto en el manifiesto como al pedir el MP3. Las imágenes y PDF sólo se sirven cuando los referencia uno de sus manuales. Los enlaces a otro manual de personas usuarias conservan su referencia mediante el PDF editorial; esa referencia concede su lectura desde el manual que lo enlaza. Consulta no recibe recursos de manuales. Este contenido contiene datos inventados y no amplía el acceso a expedientes.


| Sistema | Para qué | Estado | Qué pasa si no responde |
| --- | --- | --- | --- |
| Active Directory (LDAPS 636) | Validar usuario y contraseña | Pendiente: acceso desde la máquina, certificado de la autoridad, formato del *bind* o cuenta de servicio (Informática) | No se puede entrar; las sesiones abiertas siguen hasta caducar |
| ClamAV (contenedor local) | Revisar el Anexo 3 o 4 subido | Previsto. Falta cómo se actualizan las firmas de virus: necesitan bajar de Internet o de un espejo interno (Informática) | Se acepta con aviso y queda anotado que no se revisó |
| Registro de salida | Anotar número y fecha de registro de los oficios | Adaptador manual en la primera versión | No aplica: lo escribe una persona |
| MOAD (registro electrónico de la entidad) | Registrar oficios desde la app | Futuro; el puerto `RegistroSalida` queda preparado | Volver al registro manual |
| Sistema de copias de la entidad | Copia diaria cifrada | Pendiente con Informática | Aviso en la página de salud; la app sigue |
| Firma electrónica, correo, IA | Fuera de la app por decisión | No se integran | No aplica |

## 14. Lo que falta para producción

- Servidor de producción, copias y evaluación de impacto en protección de datos (EIPD) se preparan desde ahora en paralelo. responsable del producto la tramita con el DPD, incluidos los vínculos usuario-expediente de la auditoría [D14]. Su aprobación, la revisión de seguridad del alcance desplegado y una restauración probada preceden a los datos reales al acabar la fase 2. Manuales, vídeo y formación quedan al final.
- Informática: entornos de preproducción y producción, permiso para Docker en ellas, nombre `etf.example.org`, certificado, salida a LDAPS, formato del *bind*, acceso desde ayuntamientos y VPN, actualización de firmas de ClamAV, destino de las copias, quién termina el HTTPS y supervisión.
- Dudas del Programa que cambian datos o lectura: [D1] numeración, [D6] formato del Anexo 3 y 4, [D14] validación de conservación y anonimización por el DPD, [D20] tramos, [D21] formato de la hoja para cargar todos los expedientes de 2026: vivos, cerrados y no aceptados.
- Un Anexo 3 y un Anexo 4 rellenos con datos inventados, en el formato en que llegan, para probar el lector.
- Caducidad máxima resuelta por el responsable del producto el 14/10/2026: 8 horas por defecto, configurable (§4). Exportación por Administración funcional, un solo equipo activo, textos en directorios y plantillas dentro de idiomas quedan resueltos por el consenso del 13/10/2026.
- Revisión de seguridad completa sin fallos graves, comprobación de WCAG 2.1 AA, prueba de carga con volumen real, ensayo de restauración de copia y recorrido completo en preproducción firmado por el responsable del producto.
- Manual de sistemas para Informática en [docs/sistemas/LEEME.md](../sistemas/LEEME.md), con ensayo local de copia, borrado y restauración de PostgreSQL y paquetes. MANSIS amplía el manual con anexos de configuración, operación y diagnóstico, capturas y [PDF](../sistemas/sistemas.pdf). La [evidencia MANSIS](../sistemas/MANSIS.md) repite la restauración lógica con conservación de permisos y comprueba cifrado local; junto con [F5.05](../sistemas/EVIDENCIA.md), no sustituye la restauración cifrada fuera del servidor que debe verificar Informática antes de producción.
- Actualizar Go en el puesto de desarrollo a la última estable.

## 15. Contradicciones encontradas

Resueltas por las decisiones del responsable del producto y el consenso del 13/10/2026:

1. Mapas de módulos distintos y adaptadores sin acceso a aplicación: mapa único en §2.3 y puertos de entrada de aplicación que importa HTTP; puertos de salida para sistemas externos.
2. La app migraba con credenciales de propietario: ejecutor separado `migrar`, rol `etf_migrador`, dependencias declaradas, `catalogo` antes de `identidad` y ciclos separados en tablas y relaciones.
3. Plantillas fuera de idiomas y traducciones en base de datos: todo texto de interfaz, catálogo y plantilla vive en `idiomas/<código>/{textos,catalogos,plantillas}/`. Publicación inmutable y referencia activa confirmada con auditoría (§8).
4. Perfiles por grupos o login de AD e inicialización repetida: permisos en la app vinculados a `objectGUID`, primer administrador una sola vez y un solo equipo activo por profesional.
5. Generación confundida con fecha válida: emisiones separadas de hitos confirmados por una persona, rutas POST y nueva emisión avisada si cambian datos. Reaperturas conservan episodios e historia estadística.
6. Auditoría posterior a la lectura o supuestamente anónima: se confirma antes de entregar datos; fallos tras revertir. Administración funcional exporta y el DPD decide la conservación y el alcance de los vínculos [D14].
7. Sólo se documentaba la subida del anexo y se admitía SVG: política por tipo en §10, sin SVG, y prueba de ausencia de persistencia del anexo en proxy, antivirus y temporales.
8. «Nada sale fuera» impedía GitHub y dependencias: autorización expresa de Repositorio público y CI sin datos reales, GitHub Actions y descargas de registros públicos con versiones y sumas. Datos personales dentro de la entidad y análisis de seguridad local.
9. Datos reales en preproducción y producción tardía: sólo datos inventados en desarrollo y preproducción; carga de todo 2026 en producción al acabar fase 2, con servidor, copias y EIPD preparados en paralelo. Resultados por apartado validados por el Programa [D22].
10. Memoria provincial sumada y reglas sin fecha precisa: cálculo directo, reglas vigentes al 31/12 y aviso de diferencias; equipo cierra la suya y sólo Coordinación la provincial. Consulta recibe sólo cifras.
11. Anteriores fuentes indicaban instalación directa, 5 MB sólo Word, escalas en blanco, cookies prohibidas o firma en la app: siguen resueltas por Docker Compose, anexo Word/PDF de 10 MB, escalas guardadas [D17], cookie opaca autorizada y firma fuera. El DNI del representante no se guarda [D16].

Pendientes concretos: quién termina HTTPS y cómo llegan las firmas de ClamAV
(Informática); conservación y exportación de vínculos
(DPD, EIPD). No reabren las decisiones anteriores ni bloquean el esqueleto.

## 16. Cómo mantener este documento

Se actualiza en el mismo cambio cuando varían pila, arranque, configuración, identidad, permisos, auditoría, idiomas, temas, plantillas, seguridad, despliegue o una integración. Cuando exista código, cada ruta marcada como prevista se corrige o se quita, y el ejemplo de la regla de dependencia se cambia por uno real. Las decisiones nuevas del responsable del producto entran en la tabla del apartado 0 con fecha. Las dudas del Programa se citan por su número [Dn] y su texto completo sigue sólo en el registro de decisiones del PRD. No se pegan aquí secretos, direcciones privadas ni datos personales.

## Identidad configurable de la entidad

El catálogo `entidad.nombre` reside en `idiomas/<idioma>/textos/comun.json`. Su valor incluye el artículo y se escribe como se lee dentro de una frase («Usa tu usuario y contraseña de la entidad»). Los textos pueden referenciarlo con `{entidad.nombre}`; la interfaz realiza una sustitución literal y React conserva el escape de los valores. No se evalúa HTML ni se añade un nombre institucional al código. La configuración inicial se aplica antes de instalar; las posteriores se publican como versiones del paquete. Los membretes, firmantes y formularios se adaptan mediante plantillas y catálogos.
