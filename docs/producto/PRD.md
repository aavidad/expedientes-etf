# PRD: requisitos de producto de Expedientes ETF
> Especificación de la edición pública. La entidad que la instale configura sus datos, identidad y plantillas.
> Las referencias D1–D22 se conservan en el registro de decisiones del PRD; las fuentes privadas no se distribuyen.

Este documento dice qué es la aplicación, para quién y qué tiene que hacer. No repite lo que está en los otros cinco documentos de producto:

- `TRD.md`: pila técnica, arranque, seguridad técnica, conectores y despliegue.
- `FLUJO_APP.md`: los pasos de cada pantalla y quién los da.
- `DISENO_UI_UX.md`: aspecto, temas, accesibilidad de las pantallas y textos.
- `ESQUEMA_BACKEND.md`: tablas, campos, permisos en base de datos y migraciones.
- `PLAN_IMPLEMENTACION.md`: entregas, orden y criterios de cada una.

Siglas que aparecen aquí:

| Sigla | Significado |
| --- | --- |
| ETF | Equipo de Tratamiento Familiar (trabajador/a social, educador/a social y psicólogo/a) |
| PTF | Programa de Tratamiento a Familias con Menores en Situación de Riesgo o Desprotección |
| SS.SS.CC. | Servicios Sociales Comunitarios (los centros de la entidad que derivan con el Anexo 3) |
| SPM | Servicio de Protección de Menores de la administración competente (deriva con el Anexo 4) |
| SPAF | Servicio de Prevención y Apoyo a la Familia de la administración competente, que recibe la Tabla Mensual |
| DSR | Declaración de la Situación de Riesgo |
| PIF | Plan de Intervención Familiar |
| SMI | Salario Mínimo Interprofesional |
| ZTS | Zona de Trabajo Social |
| AD | Active Directory, el directorio de usuarios de la entidad |
| LDAPS | protocolo cifrado con el que la aplicación pregunta a AD si la contraseña es buena |
| RLS | seguridad por filas de PostgreSQL: la base de datos sólo entrega las filas que el usuario puede ver |
| ENS | Esquema Nacional de Seguridad |
| RGPD / LOPDGDD | Reglamento General de Protección de Datos / Ley Orgánica de Protección de Datos y garantía de los derechos digitales |
| EIPD | Evaluación de Impacto en Protección de Datos |
| DPD | Delegado de Protección de Datos de la entidad |
| i18n | internacionalización: preparar la aplicación para varios idiomas sin tocar el código |
| MOAD | plataforma de administración electrónica que usa la entidad (registro y firma) |

## 0. Decisiones de base

Las 17 decisiones de base y la de temas están en la entrevista; aquí van las que la plantilla asigna al PRD. El resto se recoge en `TRD.md`, `DISENO_UI_UX.md`, `ESQUEMA_BACKEND.md` y `PLAN_IMPLEMENTACION.md`.

| Decisión | Elegido | Fecha y quién |
| --- | --- | --- |
| 8. Datos personales | Minimizar. En listados, nombres enmascarados (iniciales: «Familia O. S.») y número de expediente; el nombre completo sólo en la ficha, y abrir la ficha queda registrado. Datos de prueba inventados pero verosímiles, sin la palabra «ejemplo» | 13/10/2026 · responsable del producto |
| 9. Reglas que el negocio aún no ha decidido | Configurables en catálogo versionado con fecha de vigencia; la Memoria de un año usa la versión vigente al 31 de diciembre de ese año. Nunca constantes en el código | 13/10/2026 · responsable del producto |
| 10. Rapidez | Lecturas por debajo de 300 ms en el percentil 95 con volumen real; la Memoria por debajo de 3 s; sin consultas en bucle | 13/10/2026 · responsable del producto |
| 17. Idioma de documentos y comunicación | Castellano llano, sin siglas sin explicar | 13/10/2026 · responsable del producto |

## 1. Qué es y para qué sirve

Expedientes ETF es una aplicación web interna para cualquier diputación o administración que gestione el PTF. Lleva el expediente de cada familia desde que llega la derivación (Anexo 3 de los SS.SS.CC. o Anexo 4 del SPM) hasta el cierre y el seguimiento posterior. Genera en Word los documentos que hoy se rellenan a mano y calcula sin contar a mano la Tabla Mensual para el SPAF de cada equipo y la Memoria Técnica Anual que se envía a la administración competente, de un equipo o de todo el ámbito de la entidad.

Hoy el Programa trabaja con plantillas Word, PDF firmados y recuentos manuales. La Memoria es el motivo principal de la aplicación, y la Memoria sale de la ficha del expediente: si la ficha está completa, la Memoria se calcula sola.

Dimensiones: 8 equipos hoy, preparada para 12; 3 profesionales por equipo, preparada para 4 o 5 (hasta 60 usuarios de equipo); hasta 3 personas de coordinación; unos 300 expedientes al año (269 familias y 578 menores atendidos en 2025); hasta 15 menores por expediente; unas 180 familias en tratamiento, además de los expedientes cerrados y no aceptados durante 2026, que hay que cargar al arrancar en producción.

La aplicación no guarda anexos recibidos ni documentos generados: lee el anexo de derivación que se sube, saca los datos y lo descarta. Sí conserva las versiones de plantillas, idiomas y temas que administra la organización. Tampoco firma documentos (se firman fuera, como hoy), no envía correos, no registra salidas en otro sistema (de momento se anota el número de registro a mano) y no usa inteligencia artificial externa. No está publicada en Internet.

Identidad del proyecto: repositorio público `expedientes-etf`; dominio de ejemplo `etf.example.org`. Cada entidad configura su nombre, directorio corporativo y responsables.

## 2. Usuarios y perfiles

Los permisos se dan dentro de la aplicación. AD sólo comprueba usuario y contraseña. Quien entra con una cuenta válida de AD y no tiene perfil queda «pendiente de alta» y no ve nada hasta que Administración funcional le asigna perfil y, si es de un equipo, el equipo. La cuenta de AD se vincula mediante su identificador estable `objectGUID`, además del nombre de acceso. El primer administrador se fija en la configuración al instalar; la concesión se ejecuta una sola vez y queda registrada. Retirarle después el perfil no lo vuelve a conceder automáticamente. Una persona puede tener varios perfiles (por ejemplo, coordinación y administración funcional).

| Perfil | Quién es | Qué necesita hacer | Desde dónde entra |
| --- | --- | --- | --- |
| Coordinación | Coordinador/a provincial del Programa y, en el futuro, personal administrativo (hasta 3) | Registrar derivaciones subiendo el Anexo 3 o 4; aceptar, no aceptar o dejar en lista de espera; asignar equipo, centro, número y profesional de referencia; generar Acta de Aceptación, oficios, Acta de No Aceptación, Oficio de Lista de Espera y Acta de Cierre; cerrar expedientes propuestos por el equipo y reabrirlos; ver todos los equipos; mantener la tabla del SMI; sacar y cerrar la Memoria provincial | Navegador de escritorio en la red corporativa, la red provincial de ayuntamientos o por VPN |
| Profesional de ETF | Trabajador/a social, educador/a social o psicólogo/a de un equipo | Ver y editar sólo los expedientes de su equipo; completar la ficha; generar Anexos 1, 2, 5, 6, 11 y 12 y la Tabla SPAF; proponer el cierre; anotar comisiones técnicas e informes; sacar y cerrar la Memoria de su equipo | Igual que coordinación |
| Consulta | Pocas personas que necesitan estadísticas | Ver sólo cifras de la Memoria y totales. Nunca recibe nombres, identificadores de expedientes que forman una cifra ni textos libres | Igual |
| Administración funcional | 1 o 2 personas | Dar de alta usuarios y perfiles; mantener catálogos y listas, plazos y tramos, plantillas Word, firmantes y destinatarios, temas e idiomas; exportar auditoría para el DPD. No ve fichas de expedientes | Igual |

El personal de Informática mantiene el servidor y no tiene perfil en la aplicación.

Experiencia de los usuarios: básica, sin formación previa. La aplicación tiene que entenderse sin curso. El tono es cercano, de tú, y usa los términos del Programa (Acta, Anexo, subprograma) sin cambiarlos. Detalle en `DISENO_UI_UX.md`.

## 3. Alcance por módulo

F0.04 incorpora el esqueleto técnico y la lectura mínima de textos y temas. Las capacidades de negocio de los módulos siguen pendientes. La columna «Entrega» remite al orden de `PLAN_IMPLEMENTACION.md`.

| Módulo | Qué cubre | Estado | Prioridad | Entrega |
| --- | --- | --- | --- | --- |
| `plataforma` | Soporte técnico, migraciones y contexto de ejecución; no es dominio de negocio | pendiente | alta | 0 |
| `identidad` | AD, sesión, usuarios, perfiles y pertenencia a un único equipo a la vez | pendiente | alta | 0 |
| `catalogo` | Listas, reglas con vigencia, equipos, municipios, centros, SMI, firmantes y destinatarios | pendiente | alta | 0 a 1 |
| `paquete` | Idiomas, temas y referencias activas a sus versiones inmutables | lectura e instalación de versiones disponibles en desarrollo; gestión restante pendiente | alta | 0 |
| `expediente` | Derivación, ficha, hitos confirmados, salidas, cierre, reapertura, seguimiento, avisos, conservación y carga de todos los expedientes de 2026 | pendiente | alta | 1 a 3; carga al acabar 2; conservación en 5 |
| `evaluacion` | Instrumentos y puntuaciones de escalas | pendiente | alta | 2 |
| `documento` | Plantillas, emisiones Word y registro de salida manual | servidor de emisiones, registro manual y publicación de plantillas disponible; pantallas y tipos restantes pendientes | alta | 1 a 4 |
| `actividad` | Comisiones e informes técnicos | pendiente | media | 2 |
| `estadistica` | Tabla SPAF, Memoria por equipo y provincial, totales y cierre de Memorias | Tabla SPAF con vista previa y descarga disponible; resto según PLAN_IMPLEMENTACION | la más alta | modelo y pruebas desde 0; integración en 3 y 4 |
| `auditoria` | Eventos encadenados, incluidas lecturas; exportación por Administración funcional | pendiente | alta | 0 |

Éste es el mapa único de módulos; las capacidades que se describen a continuación no crean módulos adicionales. Los conectores pertenecen al módulo que los necesita, con puertos y adaptadores definidos en `TRD.md` y `ESQUEMA_BACKEND.md`.

Fuera de alcance de la primera versión: firma electrónica, correo, guardar anexos recibidos o documentos generados, resumen automático con inteligencia artificial, doble factor de autenticación, registro automático en MOAD, generación de los Anexos 7 y 8 (sólo se anota su fecha).

## 4. Requisitos funcionales

### 4.1 Identidad, usuarios y perfiles

- **Qué debe permitir:** entrar con usuario y contraseña de AD; ver sólo lo que el perfil permite; que Administración funcional dé de alta, cambie o retire perfiles y equipos de cada persona y vea la lista de «pendientes de alta».
- **Reglas de negocio:**
  - Sin perfil no se ve ningún dato.
  - En Usuarios y permisos se identifica al personal por su nombre completo, según el encargo UI1. La cabecera de sesión activa muestra también su nombre; esta identificación no cambia el enmascarado de familias y menores.
  - Un profesional de ETF pertenece a un solo equipo en cada momento.
  - El aislamiento por equipo lo aplica la base de datos (RLS), no sólo las pantallas. Las relaciones impiden asociar menores, marcas o evaluaciones a otro expediente o equipo. Consulta accede a una vista de sólo cifras, sin composición por expediente ni textos libres.
  - Si AD no responde, nadie puede entrar; las sesiones ya abiertas siguen hasta caducar. La aplicación no guarda contraseñas.
- **Datos que se configuran y no se programan:** el nombre de usuario del primer administrador (configuración de instalación).
- **Fuera de alcance:** permisos por grupos de AD; doble factor (podrá añadirse como conector).

### 4.2 Catálogos, plazos y configuración

- **Qué debe permitir:** que Administración funcional mantenga sin programar:
  - equipos (01 a 12, con fecha de alta y baja), centros de SS.SS.CC., municipios y entidades locales autónomas con su centro, su equipo y sus variantes de escritura;
  - todas las listas de opciones y sus equivalencias (nivel educativo, composición, motivos de cierre);
  - plazos y tramos (30 días, duración orientativa de cada fase, días de espera de la respuesta del SPM, tramos de duración de la Memoria, tramos de ingresos);
  - plantillas Word (subir una versión nueva e inmutable);
  - firmantes y destinatarios de actas y oficios.

  Coordinación mantiene la tabla del SMI (un importe por año).
- **Reglas de negocio:**
  - Cada municipio tiene dos asignaciones independientes: centro de SS.SS.CC. (para actas, oficios y Tabla SPAF) y equipo (para asignar ETF y número). No se deduce una de la otra; Íllora es del centro Pinos Puente-Moclín y la atiende el ETF 6 [D9].
  - El ETF 6 se llama «ETF - 6 Huétor Tájar-Montefrío / Alhama / Íllora» (confirmado por el Programa).
  - Los municipios fuera del ámbito de la entidad (Granada capital y los de más de 20.000 habitantes) existen en el catálogo marcados como «fuera del ámbito»: la aplicación avisa y la coordinación decide [D10].
  - Toda regla aún no decidida va en catálogo versionado con vigencia. Los textos de las opciones viven en `idiomas/<código>/catalogos/`, nunca en la base de datos; Administración publica una versión nueva del directorio al cambiarlos. La Memoria de un año usa la versión vigente al 31 de diciembre de ese año.
  - Los nombres de los firmantes y destinatarios (coordinador del Programa, jefe de servicio del SPM y su dirección) nunca van escritos en la plantilla ni en el código.
  - Los listados del cliente traen erratas y minúsculas; se normalizan una vez al cargar el catálogo inicial (las plantillas de `idiomas/`, las plantillas de `idiomas/`).
- **Fuera de alcance:** que los equipos cambien catálogos.

Listas de opciones de partida (se cargan tal cual de las plantillas; su forma de guardarse está en `ESQUEMA_BACKEND.md`): nivel de escolarización del menor; estructura y composición familiar; vivienda; situación laboral; situación económica; tramos de ingresos sobre el SMI (<50 %, 50-100 %, 100-150 %, 150-200 %, >200 %); papel de cada adulto (madre, padre, pareja de la madre, pareja del padre, tutor/a legal o guardador/a, abuelo/a, otro familiar, otra persona); valoraciones del anexo y VALÓRAME; subprograma (Riesgo social, Reunificación; los anexos dicen «Reintegración» y es lo mismo); servicio derivante (SS.SS.CC., SPM, otro ETF); características de la familia (Riesgo social, Monoparental, Violencia intrafamiliar, Con menores bajo el sistema de protección); finalidad de la intervención (4 opciones); pronóstico (Favorable, Desfavorable); instrumentos de evaluación (18 opciones con fecha, de la Entrevista a «Otros»); fases de intervención; motivos de finalización; coordinación con recursos (Memoria 14); factores de riesgo de la familia (Memoria 24, 23 opciones) y de los menores (Memoria 26, 11 opciones); tipologías de desprotección (Memoria 27, 12 opciones); responsables de la desprotección (Memoria 28, 11 opciones); respuesta del SPM; organismos de informes técnicos (Memoria 16). Fuente: las plantillas de `idiomas/` y las plantillas de `idiomas/`.

### 4.3 Derivaciones y aceptación (coordinación)

F1.10 ofrece una bandeja provincial para Coordinación, con pendientes de revisar, aceptados recientes, no aceptados y lista de espera. Permite filtrar por estado, equipo, municipio y fechas de recepción, y buscar por número o familia con resultados enmascarados. Los enlaces llevan a las pantallas de subida y decisión de P2 y P3.

- **Qué debe permitir:** subir el Anexo 3 o 4; revisar los datos leídos en sus siete apartados (servicio derivante, contacto de la familia, menores, datos sociodemográficos, valoración y diagnóstico, intervenciones previas, observaciones); confirmar municipio, centro y equipo; y elegir **Aceptar**, **Lista de espera** o **No aceptar**. Los pasos de pantalla están en `FLUJO_APP.md`.
- **Reglas de negocio:**
  - **Fichero subido.** Sólo DOCX o PDF, como máximo 10 MB. Se comprueba que es un Word o PDF de verdad, sin macros y con límite de descompresión. Pasa por el antivirus local; mientras sube y mientras se revisa, la pantalla lo dice («Revisando el fichero con el antivirus…»). Si el antivirus no responde, se acepta con un aviso y queda anotado que no se revisó. El fichero se procesa en memoria y se descarta: no se guarda nunca, tampoco en el proxy, el antivirus ni en temporales. Esta excepción por caída del antivirus sólo se aplica a anexos; no a plantillas, temas, idiomas ni hojas de carga.
  - **Lectura.** Del Word se leen los datos con fiabilidad; del PDF firmado se lee parte y la coordinación revisa más campos; un escaneo se teclea [D6]. Se leen todos los menores que traiga el anexo, hasta 15.
  - **Municipio.** Se busca en «Dirección» y, si no está, en «Valoración y diagnóstico de la familia». La búsqueda prefiere la coincidencia más larga, porque hay nombres contenidos en otros (Cúllar y Cúllar Vega, Huétor Santillán, Huétor Vega y Huétor Tájar, Alhama y Alhama de Granada…). La pantalla enseña de dónde ha salido. Siempre confirma una persona; con varios candidatos o ninguno, el campo queda vacío y es obligatorio.
  - **Sexo de los menores.** No viene en los anexos: lo teclea la coordinación al aceptar.
  - **Descripción de la situación familiar** para el Acta: se copia el texto «Descripción de la situación que motiva la derivación» y la coordinación lo recorta. Nada sale a servicios externos [D13].
  - **Subprograma.** Anexo 3: Riesgo. Anexo 4: el que indique el anexo [D12].
  - **Numeración.** El número se asigna al aceptar y nunca antes. Formato de tres pares de dígitos: equipo, correlativo del equipo en el año y dos últimas cifras del año del Acta de Aceptación. El correlativo empieza en 01 cada año para cada equipo. Decisión provisional: `08-01-26` para el primer expediente del ETF 8 en 2026, con guiones [D1]. El formato se define en un solo sitio para poder cambiarlo. Dos expedientes no pueden recibir el mismo número aunque se acepten a la vez. La no aceptación y la lista de espera no consumen número.
  - **Aceptar** requiere que una persona confirme la fecha del Acta de Aceptación y fija número, equipo, centro, profesional de referencia (TS, ES o psicólogo/a), sexo de cada menor y descripción. Genera el Acta de Aceptación y, según el origen, el Oficio a SS.SS.CC. (Anexo 3) o el Oficio al SPM (Anexo 4). El Acta se rellena con los datos del Anexo 3 o 4, que son los únicos que existen al aceptar [D15].
  - **No aceptar** fija fecha, causa (texto) y subprograma, y genera el Acta de No Aceptación. Cuenta en el apartado 5 de la Memoria.
  - **Lista de espera.** Genera el Oficio de Lista de Espera y deja el caso en la bandeja de coordinación, ordenado por fecha de recepción, para aceptarlo o rechazarlo después. Provisionalmente [D2]: sólo para casos de SS.SS.CC.; al aceptarlo, el número y la fecha son los del día de la aceptación real; mientras espera no cuenta en la Memoria.
- **Datos que se configuran y no se programan:** formato del número; firmantes y destinatarios; plantillas de actas y oficios.
- **Fuera de alcance:** resumen automático con inteligencia artificial (fase posterior, con modelo instalado en la entidad y su propia evaluación de impacto); guardar el anexo subido.

### 4.4 Expediente y ficha (equipos)

F2.06 incorpora en desarrollo la bandeja del equipo y la ficha de lectura. Los listados y las búsquedas devuelven iniciales; la ficha muestra los apellidos completos de la familia y audita su apertura. Familia (lectura de F2.01a), Menores e Intervención (FI1), Documentos, Cierre, Salidas e Historial conectan con las capacidades disponibles. Los avisos incluyen las propuestas de cierre pendientes, las derivaciones al Servicio de Protección de Menores enviadas sin respuesta y los avisos calculados de F2.02 sobre consentimiento, duración de fase y datos pendientes de la Memoria. Los avisos sin pantalla disponible para resolverlos se muestran como información.

- **Qué debe permitir:** ver la bandeja del equipo con avisos; abrir la ficha y completar familia, adultos, menores, intervención, coordinación y cierre; generar el consentimiento; anotar las fechas de los documentos hechos fuera (Anexos 7 y 8); anotar las escalas de evaluación. La ficha indica qué datos faltan para la Memoria y lleva a cada uno.
- **Reglas de negocio:**
  - **Estados del expediente:** derivación recibida; aceptada (fase de recepción); no aceptada (final); lista de espera; en intervención; derivada al SPM (esperando respuesta); cerrada; seguimiento tras cese (sólo Reunificación, 12 meses tras el cese de la medida, artículo 97.3 de la Ley 4/2021). Los cambios de estado los provocan acciones concretas. El detalle de transiciones está en `FLUJO_APP.md` y `ESQUEMA_BACKEND.md`.
  - **Fase de intervención** (la elige el equipo; la aplicación propone una por las fechas y avisa si se pasa de la duración orientativa):

    | Fase | Riesgo | Reunificación |
    | --- | --- | --- |
    | Recepción | hasta 30 días | hasta 30 días |
    | Evaluación | 3 meses | 1,5 meses |
    | Intervención | 15 meses, ampliable a 18 | 6 meses, ampliable a 12 |
    | Cierre y devolución | 1,5 meses | 1 mes |
    | Seguimiento | a los 4 meses | a los 4 meses del cierre y a los 12 meses del cese de la medida |

  - **Regla de los 30 días.** Si pasan más de 30 días naturales desde la fecha del Acta de Aceptación sin fecha de consentimiento (Anexo 1 o 2), el expediente está «pendiente de atender»: sale avisado en la bandeja y cuenta en el apartado 6 de la Memoria. El «contrato con la familia» de la ficha del cliente es el consentimiento informado [D3].
  - **Consentimiento.** Anexo 1 si el subprograma es Riesgo, Anexo 2 si es Reunificación. Generarlo sólo registra una emisión; una persona confirma después la fecha en que se firmó y ese hito cuenta para los plazos. El DNI del representante legal se deja en blanco en el Word y no se guarda: decisión firme del responsable del producto [D16].
  - **Subprograma por menor.** El expediente lleva el subprograma de la familia y cada menor el suyo (por defecto el mismo). Si hay mezcla, el equipo elige en qué subprograma cuenta la familia (respuesta 5 del Programa).
  - **Adultos.** Cada adulto lleva su papel y su sexo. Para la Memoria cuentan todos los que asumen funciones parentales; los hombres en «Padres» y las mujeres en «Madres» [D7].
  - **Coordinación con recursos, factores de riesgo de los menores y tipologías** se recogen por familia (respuesta 4 del Programa) y se aplican a todos sus menores, con la opción de desmarcar alguno [D4]. Atención Temprana se marca por menor.
  - **Ingresos.** Se puede escribir la cantidad anual neta (y la aplicación calcula el tramo con el SMI) o elegir el tramo directamente. Provisional: SMI anual = mensual × 14; se usa el SMI del año de la Memoria [D8].
  - **Escalas de evaluación** (VALÓRAME, ECP-12, SCORE-15, SDQ, SOATIF, AAPI-2.5 y satisfacción CSQ-8) se guardan con sus puntuaciones en los momentos inicial, intermedio y final, y se vuelcan en los Anexos 11 y 12: decisión del responsable del producto [D17]. El VALÓRAME de la derivación viene del anexo y se ve en sólo lectura.
  - **Datos para la Tabla SPAF.** Cada menor puede tener un centro de protección opcional, elegido de catálogo o escrito. El expediente conserva el nombre y contacto del coordinador del caso en el servicio derivante, distinto del profesional de referencia del ETF.
  - **Historia de los datos.** Se conserva desde cuándo y hasta cuándo valen composición, ingresos, menores, factores, subprogramas y demás datos estadísticos; corregir la ficha actual no sustituye la historia usada para otro año. Los cierres y reaperturas conservan sus episodios.
  - **Expediente cerrado.** Sólo se puede editar su seguimiento. Para cambiar cualquier otro dato, Coordinación debe reabrirlo con motivo auditado.
  - **Edición a la vez.** Si dos profesionales editan la misma ficha, quien guarda después recibe un aviso de que otra persona cambió los datos y puede verlos; nunca se pisa un cambio en silencio.
  - **Listados** con nombres enmascarados («Familia O. S.») y número; el nombre completo sólo en la ficha.
- **Datos que se configuran y no se programan:** duraciones de fase, días del plazo de consentimiento, listas de opciones.
- **Fuera de alcance:** generar los Anexos 7 y 8 (sólo su fecha); subir documentos al expediente.

F2.01c entrega la edición de intervención en el servidor: fase del expediente,
coordinación, factores y tipologías con exclusiones por menor. Conserva vigencias
y publica una consulta a fecha para la Memoria y los avisos. FI1 añade las pestañas Menores e Intervención a la ficha: el profesional del ETF
edita con fecha de vigencia y Coordinación consulta. El conflicto conserva el
formulario y permite comparar los datos actuales antes de volver a guardar. Los
avisos de F2.02 enlazan al campo concreto de estas pestañas.

### 4.5 Comisiones técnicas e informes técnicos

- **Qué debe permitir:** que el equipo anote cada comisión técnica (fecha y observaciones) y cada informe técnico emitido para otro organismo (fecha y organismo: Juzgados, Fiscalía, SPAF, SPM, Centros educativos, Centros sanitarios, Servicios Sociales, Otros con texto). Coordinación los consulta de todos los equipos.
- **Reglas de negocio:** son registros del equipo, no de un expediente. Alimentan los apartados 15 y 16 de la Memoria. Sólo se cuentan informes técnicos emitidos.

### 4.6 Derivaciones de salida, cierre, reapertura y traslado

- **Qué debe permitir:** generar el Anexo 5 (al SPM) y anotar después la respuesta del SPM; generar el Anexo 6 (a otro servicio); generar el Anexo 11 (Riesgo) o 12 (Reunificación) con el motivo de finalización y proponer el cierre; que coordinación revise los cierres propuestos y genere el Acta de Cierre; reabrir; trasladar entre equipos; llevar el seguimiento tras cese.
- **Reglas de negocio:**
  - **Doble intervención.** El equipo propone el cierre (Anexo 11/12) y la coordinación cierra (Acta de Cierre). Es el único paso que exige dos personas.
  - **Anexos 5 y 6** salen con los apartados 1 a 4 rellenos y del 5 al 8 en blanco, con la fecha del día como propuesta para el documento. La fecha de la derivación se confirma por separado; regenerar no la cambia.
  - **El Anexo 5 bloquea el cierre ordinario** hasta registrar «se conoce inicio de procedimiento de desamparo o adopción de medida» o «el SPM no valora necesaria la medida». La respuesta «a la espera» mantiene el bloqueo. Coordinación puede forzar el cierre con un motivo escrito y auditado. El Anexo 6 no cambia el estado ni bloquea.
  - **Cerrar desde aceptada.** Si la familia no consiente, puede proponerse y confirmarse el cierre desde ACEPTADA sin pasar por intervención, con un motivo del catálogo.
  - **Cerrar exige** fecha de cierre confirmada por una persona (la del Acta de Cierre) y motivo de finalización. Si falta el Anexo 11 o 12, avisa pero deja cerrar.
  - **Motivo de cierre.** Se pide una sola vez, con la lista del Anexo 11/12, y se traduce a las otras dos listas (ver 4.9). En el Acta de Cierre la coordinación puede cambiar el motivo a mano [D5].
  - **Reabrir** sólo lo hace la coordinación, con motivo, y queda en la auditoría. Si la familia vuelve a ser derivada más adelante, se abre un expediente nuevo marcado con antecedente en el Programa y enlazado al anterior.
  - **Traslado entre equipos:** se cierra el expediente en el equipo de origen con motivo «traslado» y se abre uno nuevo con número nuevo en el de destino, enlazados; el servicio derivante del nuevo es «Otro ETF» [D11].
  - **Seguimiento tras cese** (Reunificación): se anota la fecha de cese de la medida; el plazo inicial es de 12 meses y se mantiene en el catálogo de reglas, editable por Administración. La aplicación avisa cuando se acerca el final, inicialmente 30 días antes. Decisión de Dirección para F3.02c [D23]: termina automáticamente al cumplir el plazo; el equipo puede terminarlo antes con motivo de catálogo auditado. Coordinación puede reabrirlo en cualquier momento, incluso tras el vencimiento; el plazo vigente vuelve a contar completo desde la fecha de reapertura. Se conserva la historia.
  - **Registro de salida.** Para cada oficio y acta se puede anotar a mano el número y la fecha de registro de salida. El registro automático en MOAD queda para el futuro sin cerrar esa puerta (detalle en `TRD.md`).
- **Fuera de alcance:** enviar los documentos; firmarlos.

### 4.7 Documentos Word

- **Qué debe permitir:** generar en Word, desde la plantilla oficial vigente y con los datos del expediente: Acta de Aceptación, Oficio de Acta de Aceptación a SS.SS.CC., Oficio de Acta de Aceptación al SPM, Acta de No Aceptación, Oficio de Lista de Espera, Acta de Cierre, Anexos 1, 2, 5, 6, 11 y 12, Tabla Mensual SPAF y Memoria. Las plantillas originales están en las plantillas de `idiomas/`.
- **Reglas de negocio:**
  - Los textos fijos de las plantillas no se cambian. Las referencias antiguas («Anexo 2L-A», «2J», «2P»; en la Tabla SPAF «Anexos 2L, 2M, 2O») se dejan como están mientras el Programa no diga otra cosa [D18].
  - **Emisión e hito son distintos.** Cada generación registra autor, fecha, versión y huella de la plantilla y los datos usados, sin conservar el fichero Word. El hito de negocio registra por separado la fecha que confirma una persona (por ejemplo, «firmado el…»). Generar o regenerar no cambia hitos, fechas de negocio ni estados.
  - **Repetir una descarga.** Si los datos han cambiado se crea una emisión nueva y se avisa. Una misma clave de repetición segura (idempotencia) nunca devuelve contenido distinto; si cambia la petición se rechaza su reutilización.
  - **Versiones de plantilla.** Administración escribe un archivo por versión y huella. Su referencia se registra y activa junto con la auditoría en una única transacción, con control de activaciones simultáneas. Si falla, el archivo huérfano no se sirve y se limpia.
  - Los documentos generados son accesibles y editables.
  - El Acta de Aceptación pone como fecha «a la fecha de la firma electrónica», como la plantilla.
- **Datos que se configuran y no se programan:** las plantillas (Administración sube una versión nueva), firmantes, destinatarios y las plantillas de cada idioma.
- **Fuera de alcance:** guardar el Word o el PDF firmado; firma electrónica.

### 4.8 Tabla Mensual para el SPAF

- **Qué debe permitir:** que el equipo elija un mes (a mes vencido), vea la tabla y la descargue en Word. Fuente: las plantillas de `idiomas/`.
- **Reglas de negocio:**
  - **Entran en la tabla del mes M** los expedientes del equipo con Acta de Aceptación hasta el último día de M y sin Acta de Cierre, o con Acta de Cierre entre el día 1 y el último día de M. Los cerrados antes del día 1 de M no salen.
  - **Columnas:** (1) número de expediente; (2) apellidos, nombre y fecha de nacimiento de cada menor; (3) número de menores; (4) guardadores y parentesco, o el centro de protección si están en uno; (5) municipio y ZTS [D19]; (6) características de la familia; (7) anexo de derivación y fecha (el último entre Anexos 3, 4, 5 y 6); (8) servicio derivante y coordinador/a del caso; (9) fechas de Acta de Aceptación, Anexo 1/2 y Anexo 7/8; (10) fase de intervención; (11) temporalización del proyecto en meses; (12) finalidad y pronóstico; (13) instrumentos de evaluación; (14) fecha y motivo de cierre, con el motivo traducido a la lista SPAF [D5].
  - **Resumen del mes:** altas (aceptados en M), bajas (cerrados en M), total de casos de la tabla y total de menores de la tabla.
- **Fuera de alcance:** enviarla al SPAF.

### 4.9 Equivalencias entre listas

Se configuran en catálogo, versionadas, y se aplican al importar o al calcular. La aplicación guarda el valor de la Memoria.

**Nivel educativo** (confirmada por el Programa; las plantillas de `idiomas/`):

| Anexos 3 y 4 | Memoria, apartado 20 |
| --- | --- |
| Analfabetismo | Analfabetismo |
| Sin estudios | Sabe leer y escribir, no finaliza estudios obligatorios |
| Primarios | Estudios obligatorios (ESO, EGB) |
| Secundarios | Estudios secundarios (Bachiller, FP, módulos profesionales) |
| Superiores | Estudios universitarios |

**Composición familiar** (confirmada por el Programa). La aplicación propone el valor a partir de los miembros de la unidad de convivencia del anexo y el equipo lo confirma o lo cambia. Las situaciones que el anexo no permite deducir se eligen a mano.

| Miembros del Anexo 3 o 4 | Memoria, apartado 17 |
| --- | --- |
| Madre y padre | Familias biparentales |
| Madre | Familias monoparentales (madre) |
| Padre | Familias monoparentales (padre) |
| Uno o dos progenitores con hijos/as que conviven con otros familiares | Familias extensas |
| (a mano) | Separación/divorcio con custodia compartida |
| (a mano) | Separación/divorcio sin custodia compartida |
| (a mano) | Familias reconstituidas (una o ambas figuras parentales tienen hijos/as de una relación anterior; prevalece sobre monoparental) |
| (a mano, con texto) | Otras formas de convivencia |

Los anexos dicen «reconstruidas» y no traen «extensas»: la aplicación usa la lista de la Memoria.

**Motivos de cierre.** El de la Memoria está confirmado por el Programa; las columnas del Acta de Cierre y de la Tabla SPAF son la propuesta de [D5], pendiente de confirmar. En pantalla, «Otros motivos de cierre» se desdobla en tres opciones para que la Memoria salga sola.

| Anexo 11/12 | Memoria | Acta de Cierre [D5] | Tabla SPAF [D5] |
| --- | --- | --- | --- |
| Objetivos satisfactoriamente cumplidos | 8 · Objetivos alcanzados | Consecución de objetivos | Consecución de objetivos |
| Objetivos parcialmente cumplidos | 8 · Objetivos no alcanzados | Consecución parcial de objetivos | Consecución de objetivos |
| Sin objetivos ni mejora por falta de adherencia | 9 · No adherencia (falta de colaboración) | Otros | No colaboración de la familia |
| La familia abandona por iniciativa propia | 9 · Rechazo (firman renuncia expresa) | No aceptación de la intervención por los progenitores | No colaboración de la familia |
| Traslado familiar | 9 · Traslado localidad | Otros | Traslado de domicilio |
| Mayoría de edad de los hijos/as | 9 · Mayoría edad | Otros | Mayoría de edad |
| Ingreso en centro de los menores (sólo Anexo 11) | 9 · Ingreso centro reforma | Otros | Adopción de medida protectora |
| Otros: Declaración de Riesgo | 9 · Propuesta DSR | Otros | No consecución de objetivos |
| Otros: Medida protectora o desamparo | 9 · Propuesta medida protectora | Otros | Adopción de medida protectora |
| Otros: cualquier otro (con texto) | 9 · Otras causas | Otros | No consecución de objetivos |

Falta saber cuándo se usan «Ha transcurrido el período estimado» e «Irregularidades en la derivación» del Acta de Cierre (las plantillas de `idiomas/`) [D5].

### 4.10 Memoria Técnica Anual

- **Qué debe permitir:** elegir año y equipo (uno o todos); ver las cifras de cada apartado; escribir los apartados de texto; descargar la Memoria en Word con la plantilla de la administración competente; cerrarla. Al pulsar una cifra, coordinación y equipo ven la lista de números de expediente que la forman, para comprobarla. Consulta sólo recibe cifras, sin listas de identificadores ni textos libres, también en las descargas. Las pantallas de Memoria nunca muestran nombres.
- **Reglas generales:**
  - Datos del año natural evaluado (A), con los valores cuya vigencia corresponda al año y a la fecha de corte de cada apartado; no se toma la ficha actual para todos los años. Las reglas son las vigentes al 31/12/A. Edades a 31/12/A, con la excepción de la nota 43: quien cumple 18 durante A cuenta en 12-17.
  - Cada expediente es una familia; los menores se cuentan uno a uno, cada uno en su subprograma.
  - Cada apartado define población incluida, denominador y tratamiento de desconocidos. Los «sin dato» se muestran por separado y visibles; no se asignan a una categoría conocida. Porcentajes con dos decimales y coma decimal; un divisor cero muestra 0,00 y un aviso.
  - Cada equipo hace la suya y Coordinación hace la provincial, calculada siempre directamente desde los datos con las mismas reglas; nunca suma Memorias de equipo. Si el resultado difiere de una Memoria de equipo cerrada, se muestra el aviso y la diferencia.
  - **Consulta (F4.07).** Puede elegir año, equipo o provincia. Si la Memoria está cerrada recibe sus cifras definitivas; si está abierta se calculan al momento. La vista `/memoria`, la API y el Word indican la fecha y el carácter provisional o definitivo de las cifras. La vista conserva los filtros y bloques de F4.02, sin narrativas, desglose ni acciones de cierre.
  - **Cerrar** guarda una foto de las cifras: una Memoria cerrada no cambia aunque se corrija después un expediente. Reabrirla es una acción de coordinación que queda en la auditoría. El equipo cierra la de su equipo; sólo coordinación cierra la provincial.
  - Cada apartado se calcula en el servidor con pruebas y resultados esperados por apartado validados por el Programa. La Memoria oficial de 2025 sirve como referencia documental, no obliga a reproducir repartos incoherentes: su apartado 2 da 294 niños y 284 niñas, y el 25 da 290 y 288; el Programa debe indicar cuál es correcto y por qué [D22]. Hay comprobaciones automáticas de coherencia (por ejemplo, total del 7 = total del 8 + total del 9).
- **Contrato de cálculo:** población, denominador, desconocidos y fecha de corte de cada apartado se detallan en `ESQUEMA_BACKEND.md` §1.10; los resultados esperados los valida el Programa.
- **Definición de cada apartado** (fuentes: las plantillas de `idiomas/`, notas de las plantillas de `idiomas/` y respuestas del Programa):

| Apdo. | Qué cuenta | Regla |
| --- | --- | --- |
| A | Datos generales | Texto y cifras escritos a mano: entidad local, nº de equipos del convenio, denominación, profesionales por sexo (sin coordinación ni personal pagado sólo por la entidad local), observaciones sobre bajas y sustituciones |
| 1 | Casos atendidos | Familias y menores con Acta de Aceptación hasta el 31/12/A y sin cierre o con cierre desde el 01/01/A, más las familias de Reunificación en seguimiento tras cese durante A. «De años anteriores»: aceptados antes del 01/01/A. «Incorporados»: aceptados en A. Por subprograma |
| 2 | Menores atendidos por sexo | Menores del apartado 1, niños y niñas, por subprograma |
| 3 | Casos con antecedentes | Atendidos en A que ya estuvieron en el Programa (cerrados en años anteriores) |
| 4 | Según servicio derivante | Familias atendidas por subprograma y servicio (SS.SS.CC., SPM, otro ETF) |
| 5 | Derivados y no aceptados | Fecha de no aceptación en A, por servicio derivante y subprograma, con la lista de causas |
| 6 | Pendientes de atender | Aceptados con más de 30 días sin consentimiento a 31/12/A, o a la fecha de cálculo si el año está en curso. Familias y menores por subprograma [D3] |
| 7 | Cerrados | Fecha de Acta de Cierre en A. «De años anteriores»: aceptados antes de A (la definición de la plantilla era una errata, confirmado por el Programa). «Incorporados»: aceptados en A |
| 8 | Cerrados con tratamiento concluido | Motivo «objetivos satisfactoriamente cumplidos» (alcanzados) o «parcialmente cumplidos» (no alcanzados). Familias y menores |
| 9 | Cerrados sin concluir el tratamiento | Resto de motivos, repartidos en rechazo, no adherencia, traslado, mayoría de edad, propuesta DSR, propuesta de medida protectora, ingreso en centro y otras causas (con texto) |
| 10 | Derivados a SS.SS.CC. tras el cierre | Cerrados en A con derivación tras el cierre a SS.SS.CC. |
| 11.1 | Propuestas de DSR | Atendidos con «Propuesta de Declaración de Riesgo» marcada; la familia cuenta 1 y suma todos sus menores. Porcentaje sobre los atendidos de Riesgo del apartado 1 |
| 11.2 | PIF derivado de DSR | Familias con Plan de Intervención Familiar derivado de una DSR; mismo porcentaje |
| 12.1 | Propuestas de medida protectora | Atendidos con propuesta de medida protectora (Anexo 5); porcentaje sobre los atendidos de Riesgo del apartado 1 |
| 12.2 | Respuesta del SPM | Reparto de las del 12.1 entre «se conoce inicio de procedimiento de desamparo o adopción de medida», «a la espera» y «el SPM no valora necesaria la medida»; porcentaje sobre el 12.1 |
| 13.1 / 13.2 | Duración del tratamiento | Sólo los del apartado 8. De la fecha del Acta de Aceptación a la del Acta de Cierre (respuesta del Programa), en meses completos, por tramos. Tramos provisionales de la plantilla: menos de 6, 6-12, 13-18, más de 18, configurables [D20] |
| 13.3 | Seguimiento tras cese | Familias de Reunificación en seguimiento de 12 meses tras el cese de la medida durante A; porcentaje sobre Reunificación del apartado 1 |
| 14 | Coordinación con recursos | Menores por sexo de familias con coordinación con cada recurso (14 recursos y «otros» con texto). Dato de familia aplicado a sus menores [D4] |
| 14.1 | Atención Temprana | Menores por sexo propuestos para Atención Temprana (dato por menor) |
| 15 | Comisiones técnicas | Comisiones anotadas por el equipo con fecha en A |
| 16 | Informes técnicos | Informes emitidos con fecha en A, por organismo |
| 17 | Composición familiar | Familias atendidas por composición (ver 4.9) |
| 18 | Familias numerosas | Familias atendidas con condición de familia numerosa según la Ley 40/2003 |
| 19 | Edad parental | Adultos con papel parental, en «Padres» y «Madres» por sexo, en tramos de menos de 20, 20-29, 30-39, 40-49 y 50 o más [D7] |
| 20 | Nivel educativo parental | Ídem, con la lista de la Memoria (ver 4.9) |
| 21 | Situación laboral parental | Ídem, con «otras» con texto |
| 22 | Situación económica | Familias atendidas por las tres situaciones |
| 23 | Ingresos de la unidad de convivencia | Familias por tramo sobre el SMI anual (suma de ingresos netos anuales de la unidad) [D8] |
| 24 | Factores de riesgo de la familia | Familias con cada uno de los 23 factores (varios por familia) |
| 25 | Menores por sexo y edad | Niños y niñas en 0-5, 6-11 y 12-17 años, por subprograma; quien cumple 18 durante el año cuenta en 12-17 (nota 43) |
| 26 | Factores de riesgo de los menores | Menores por sexo con cada uno de los 11 factores. Dato de familia aplicado a sus menores [D4] |
| 27 | Tipologías de desprotección | Menores por sexo con cada una de las 12 tipologías. Ídem [D4] |
| 28 | Responsables de la desprotección | Familias por persona responsable (11 opciones) |
| D | Metodología de intervención | Texto, máximo 25 líneas |
| E | Dificultades | Texto, máximo 25 líneas |
| F | Propuestas de mejora | Texto, máximo 25 líneas |
| G | Valoración global | Texto, máximo 30 líneas. Está en la Memoria oficial de 2025 y no en la plantilla de la administración competente; se incluye a falta de que responsable del producto lo confirme |

### 4.11 Avisos

- **Qué debe permitir:** ver avisos en la bandeja y en la ficha. Sólo en la aplicación, sin correo.
- **Avisos:** más de 30 días desde la aceptación sin consentimiento; fase por encima de su duración orientativa; Anexo 5 sin respuesta del SPM pasados N días (N configurable); motivo de finalización anotado sin Acta de Cierre (en coordinación, como «cierres propuestos»); fin del seguimiento tras cese próximo; datos que faltan para la Memoria (en enero, para los expedientes del año anterior).

### 4.12 Carga de todos los expedientes de 2026

- **Qué debe permitir:** cargar todos los expedientes necesarios para la Memoria de 2026: vivos, cerrados durante 2026 y no aceptados durante 2026. Desde una hoja de cálculo si existe o mediante tecleo de los equipos [D21]. La carga se hace sólo en producción, con la aplicación desplegada, al acabar la fase 2.
- **Reglas de negocio:** los expedientes cargados conservan sus números y fechas reales y su historia estadística; no consumen correlativos nuevos. Los no aceptados conservan su fecha y causa, sin recibir número de aceptación. Se identifican como cargados para su revisión. La hoja se valida con informe de errores y una importación completa o ninguna; no se conserva el fichero. Si no hay hoja, los equipos teclean los mismos datos con las mismas validaciones.
- **Condiciones:** EIPD aprobada, seguridad revisada y restauración de copias comprobada antes de introducir datos reales. Desarrollo y preproducción sólo usan datos inventados. No se esperan datos reales de 2025 para construir ni aceptar el cálculo.
- **Servidor de hoja F2.08b:** entrada CSV/XLSX sobre el alta histórica de F2.08a, con antivirus obligatorio, informe por fila sin valores personales y confirmación atómica e idempotente. Formato provisional D21 en ESQUEMA_BACKEND §7.2: hasta 100 casos por equipo y hoja, sin conservar fichero ni vista previa. CA1 añade alta asistida y carga por hoja con revisión previa, incidencias por fila y resultado. La pantalla permite descargar una hoja CSV con las 18 columnas del formato provisional y datos inventados para el equipo elegido.
- **Fuera de alcance:** carga histórica ajena a los expedientes necesarios para 2026, salvo nueva decisión del responsable del producto.

### 4.13 Qué se guarda y qué no

Regla del responsable del producto: lo que digan los documentos del cliente; si no dicen nada, se guarda todo; si dicen que nada, nada. Las decisiones expresas de minimización, como no guardar el DNI del representante [D16], prevalecen.

| Se guarda | No se guarda |
| --- | --- |
| Todos los datos del expediente, de la familia, de los adultos y de los menores | El Anexo 3 o 4 subido (se lee en memoria y se descarta) |
| Las puntuaciones de las escalas de evaluación en sus tres momentos [D17] | Los Word generados |
| Emisiones con autor, fecha, tipo, versión y huella de plantilla y datos usados; hitos y fechas confirmadas por separado (también Anexos 7 y 8) | Los PDF firmados (respuesta del Programa: «Sólo datos») |
| Número y fecha de registro de salida anotados a mano | El DNI del representante legal en el consentimiento: decisión firme [D16] |
| La foto de cifras de cada Memoria cerrada | Contraseñas |
| La auditoría de todo, también lecturas | Datos personales en los registros técnicos |

**Conservación:** 25 años desde el cierre, configurables mediante el plazo de catálogo `conservacion` (unidad `anos`). Administración funcional dispone de una lista sin nombres ni acceso a fichas. Dos personas distintas de Administración deben revisar y confirmar la retirada irreversible; un seguimiento activo la bloquea. Después la aplicación marca el expediente y lo anonimiza: borra nombres, apellidos y datos de contacto y deja los datos estadísticos. Está pendiente de validar con el DPD [D14], incluida la conservación y exportación de vínculos entre usuario y expediente en auditoría; se resolverá en la EIPD y el registro de decisiones del PRD.

### 4.14 Auditoría

- **Qué debe permitir:** que quede registrado quién hizo qué, cuándo y sobre qué expediente, también al abrir una ficha; que la coordinación vea el historial de un expediente; que Administración funcional exporte el registro para el DPD.
- **Reglas de negocio:** el registro es encadenado (cada entrada depende de la anterior, así que una manipulación se nota) y nadie puede borrarlo desde la aplicación. La auditoría debe quedar confirmada antes de entregar cualquier dato, incluidas las lecturas; si falla, no se entregan datos. Los fallos se registran después de revertir la operación fallida. Los plazos y vínculos personales de las exportaciones se validan con el DPD en la EIPD.

### 4.15 Idiomas y temas

- **Idiomas:** castellano principal, preparada para más. Ningún texto visible está en el código: los textos de pantalla, los mensajes del servidor, los nombres de listas y catálogos, los avisos y las plantillas Word de cada idioma viven en `idiomas/<código>/{textos,catalogos,plantillas}/`, nunca como textos de catálogo en la base de datos. Los textos libres de expedientes y los apartados narrativos de la Memoria son datos de negocio, no traducciones. Añadir un idioma es publicar su directorio por versión y activar su referencia; la aplicación lo detecta al arrancar o al recargar desde administración y sólo ofrece en el selector versiones activadas. Si falta un texto en un idioma, se muestra en castellano.
- **Temas:** instalables, como en Joomla, en `temas/<nombre>/`. Administración los sube y activa sin programar. Idiomas y temas se publican por versiones inmutables; la referencia activa y la auditoría se confirman juntas y se impiden activaciones simultáneas que pierdan cambios. El tema por defecto es el aspecto de la maqueta y habrá otro de alto contraste. Detalle en `DISENO_UI_UX.md` y `TRD.md`.

## 5. Requisitos no funcionales de producto

- **Ficheros administrados:** cada tipo tiene su propia política de tamaño, tipo real, antivirus local y persistencia: anexo DOCX/PDF de 10 MB sin persistencia; plantilla DOCX de 10 MB; tema ZIP de 20 MB con JSON, PNG y WOFF2, sin SVG; idioma ZIP de 20 MB con JSON y DOCX; hoja XLSX/CSV de 10 MB sin persistencia. Los paquetes admiten como máximo 100 MB descomprimidos y 1.000 entradas. Plantillas, temas e idiomas sí se conservan por versión. Si el antivirus no está disponible, no se publican ni importan; la excepción con aviso auditado se limita al anexo de derivación. Detalle en `TRD.md`.
- **Rapidez:** lecturas por debajo de 300 ms en el percentil 95 con el volumen real; la Memoria por debajo de 3 s.
- **Seguridad y privacidad:** datos de menores, violencia de género, salud mental y adicciones (categorías especiales del RGPD). Se nota en la aplicación así: nombres enmascarados en listados; nombre completo sólo en la ficha, y abrirla queda registrado; cada equipo ve sólo lo suyo; Consulta sólo recibe cifras, sin identificadores ni textos libres; entrada sólo desde la red corporativa, la red provincial o VPN; ningún dato personal sale de la entidad. Repositorio público y CI sin datos reales y GitHub Actions están autorizados para código y pruebas con datos inventados; se permite descargar dependencias Go/npm con versiones y sumas fijadas. Ni datos ni código se envían a inteligencia artificial, reconocimiento óptico de caracteres (OCR) ni análisis de seguridad en la nube; el análisis se hace en local. Medidas técnicas en `TRD.md`.
- **Trazabilidad y auditoría:** ver 4.14. Para coordinación (historial del expediente) y para el DPD (exportación).
- **Accesibilidad:** WCAG 2.1 AA (RD 1112/2018) en pantallas y en los Word generados; todo usable con teclado. Ayuda sólo tras un botón «?».
- **Idiomas:** ver 4.15.
- **Disponibilidad:** horario laboral de 8 a 15 h; se tolera una parada de horas; las actualizaciones se hacen fuera de horario. Copia diaria cifrada fuera del servidor; se acepta perder como mucho un día. Si AD cae, no se entra; si el antivirus cae, sólo el anexo de derivación puede aceptarse con aviso auditado; los demás ficheros esperan a la revisión local.
- **Entornos:** desarrollo y preproducción sólo con datos inventados; datos reales exclusivamente en producción. responsable del producto despliega tras probar en preproducción. Servidor de producción, copias y EIPD se preparan en paralelo desde ahora; se aprueban y verifican antes de la carga real al acabar la fase 2.
- **Soporte:** el equipo que designe la entidad o su herramienta de incidencias.

## 6. Normativa y obligaciones

| Norma | En qué condiciona el producto |
| --- | --- |
| RGPD y LOPDGDD | Minimización, enmascarado, conservación de 25 años y anonimización, auditoría, EIPD antes de producción con el DPD (la abre responsable del producto ya) |
| ENS, nivel ALTO en confidencialidad | Acceso sólo desde red interna, cifrado, copias, auditoría inalterable (detalle en `TRD.md`) |
| RD 1112/2018 de accesibilidad | WCAG 2.1 AA en pantallas y documentos |
| Ley 4/2021 de Infancia y Adolescencia de Andalucía, art. 97.3 | Seguimiento de 12 meses tras el cese de la medida (Reunificación) y apartado 13.3 de la Memoria |
| Ley 40/2003 de Protección a las Familias Numerosas | Definición de familia numerosa en el apartado 18 |
| Manual de referencia de los ETF de la administración competente | Fases y duraciones, subprogramas, anexos y escalas |
| Ley 39/2015 (documentos firmados con copia auténtica) | Los documentos se firman fuera, en MOAD; la aplicación no firma |

## 7. Métricas de éxito y criterio de cierre

**Éxito del producto:** la Memoria de 2026 (que se entrega en enero-febrero de 2027) sale de la aplicación sin contar a mano y la coordinación la da por buena.

**Criterio de terminado** (lo firma responsable del producto):

1. Recorrido completo en preproducción, de la derivación a la Memoria, con datos inventados y probado en navegador (ver apartado 8).
2. Revisión de seguridad sin fallos graves.
3. WCAG 2.1 AA comprobado.
4. Todos los expedientes de 2026 (vivos, cerrados y no aceptados) cargados en producción al acabar la fase 2.
5. Cada apartado de Memoria supera las pruebas con datos inventados y resultados esperados validados por el Programa; las discrepancias de la referencia 2025 se documentan, incluida [D22], sin reproducir repartos incoherentes ni esperar datos reales de 2025.
6. Rapidez dentro de los objetivos del apartado 5 con volumen real.
7. Manual de usuario (equipos y coordinación), manual de sistemas, vídeo de demostración y sesión de formación.

**Condición para introducir datos reales al acabar la fase 2:** aplicación desplegada en producción, EIPD aprobada con el DPD, revisión de seguridad superada y restauración de copias comprobada. Servidor, copias y EIPD avanzan en paralelo desde el inicio. Manuales, vídeo y formación se terminan al final del proyecto.

## 8. Recorrido de demostración

Datos inventados. Los pasos de cada pantalla están en `FLUJO_APP.md`.

1. **Coordinación** (Inés Moreno Castillo) sube un Anexo 3 en Word del Centro Valle Lecrín-Ogíjares sobre la familia Ortega Salas, de Dúrcal, con dos menores. Ve el aviso del antivirus mientras se revisa.
2. **Coordinación** revisa los datos leídos. La aplicación propone Dúrcal, el centro Valle Lecrín-Ogíjares y el ETF 4, y enseña de dónde lo ha sacado. Confirma.
3. **Coordinación** acepta: elige profesional de referencia, marca el sexo de cada menor, recorta la descripción. La aplicación asigna el número `04-01-26` (provisional [D1]) y genera el Acta de Aceptación y el Oficio a SS.SS.CC. Anota a mano el número de registro de salida del oficio.
4. **Profesional del ETF 4** (Pablo Ruiz Vílchez, trabajador social) ve el expediente en su bandeja como «Familia O. S.», genera el Anexo 1; tras la firma, confirma la fecha del consentimiento y entonces desaparece el aviso de 30 días. Regenerarlo sólo añade una emisión.
5. **Profesional** completa la ficha: composición, adultos, ingresos, factores, tipologías, instrumentos, escalas iniciales, fase. El aviso «Faltan N datos para la Memoria» baja a cero.
6. **Profesional** genera el Anexo 11 con motivo «objetivos satisfactoriamente cumplidos» y propone el cierre.
7. **Coordinación** ve el cierre propuesto, revisa, genera el Acta de Cierre y confirma la fecha del cierre; la mera generación no cierra el expediente.
8. **Profesional** saca la Tabla SPAF del mes: el expediente aparece como alta y como baja.
9. **Coordinación** saca la Memoria provincial del año: la familia cuenta en los apartados 1, 2, 7, 8, 13.1 y en los de características. Pulsa una cifra y ve el número del expediente. Descarga el Word.

## 9. Decisiones pendientes

Las decide el Programa a través del responsable del producto. Se siguen en el registro de decisiones del PRD. Mientras no haya respuesta, se aplica la decisión provisional, que va en catálogo configurable siempre que se pueda.

| Duda | Resumen y decisión provisional | Quién decide | Dónde se sigue |
| --- | --- | --- | --- |
| D1 | Orden del número (texto «equipo-correlativo-año» frente a ejemplo `01-08-26`). Provisional: `08-01-26` con guiones | Programa vía responsable del producto | el registro de decisiones del PRD |
| D2 | Lista de espera: si vale para Anexo 4, fecha y número al aceptar, si cuenta en la Memoria. Provisional: sólo SS.SS.CC.; fecha y número del día de la aceptación; no cuenta | Programa vía responsable del producto | el registro de decisiones del PRD |
| D3 | 30 días: el contrato es el consentimiento y son días naturales. Provisional: sí | Programa vía responsable del producto | el registro de decisiones del PRD |
| D4 | Datos por familia contados por menor en Memoria 14, 26 y 27. Provisional: se aplican a todos los menores con opción de desmarcar | Programa vía responsable del producto | el registro de decisiones del PRD |
| D5 | Tres listas de motivos de cierre. Provisional: un solo motivo traducido; el Acta de Cierre se puede cambiar a mano | Programa vía responsable del producto | el registro de decisiones del PRD |
| D6 | Formato del Anexo 3/4 que llega. Provisional: Word y PDF; con PDF se revisa más a mano | Programa vía responsable del producto | el registro de decisiones del PRD |
| D7 | Figuras parentales en la Memoria. Provisional: todos los adultos con funciones parentales, por su sexo | Programa vía responsable del producto | el registro de decisiones del PRD |
| D8 | SMI: 14 pagas, año que se usa, cantidad o tramo. Provisional: 14 pagas, SMI del año de la Memoria, ambas formas | Programa vía responsable del producto | el registro de decisiones del PRD |
| D9 | Íllora: centro Pinos Puente-Moclín y ETF 6. Provisional: así | Programa vía responsable del producto | el registro de decisiones del PRD |
| D10 | Municipios fuera de la entidad. Provisional: avisa y decide coordinación | Programa vía responsable del producto | el registro de decisiones del PRD |
| D11 | Traslado entre equipos. Provisional: cierre y expediente nuevo con número nuevo, enlazados | Programa vía responsable del producto | el registro de decisiones del PRD |
| D12 | Subprograma de los no aceptados. Provisional: Anexo 3 = Riesgo; Anexo 4 = el del anexo | Programa vía responsable del producto | el registro de decisiones del PRD |
| D13 | Resumen automático. Provisional: se copia el texto del anexo; sin IA externa | Programa vía responsable del producto | el registro de decisiones del PRD |
| D14 | Qué se hace a los 25 años. Provisional (entrevista): anonimizar; validar con el DPD | responsable del producto con el DPD | el registro de decisiones del PRD |
| D15 | Datos del Acta de Aceptación: Anexos 3 y 4, no 1 y 2. Provisional: Anexo 3 o 4 | Programa vía responsable del producto | el registro de decisiones del PRD |
| D18 | Nombres antiguos de los anexos en las plantillas. Provisional: se dejan | Programa vía responsable del producto | el registro de decisiones del PRD |
| D19 | Qué es la ZTS de la Tabla SPAF (¿el centro de SS.SS.CC. u otra división?). el registro de decisiones del PRD no da provisional: lo fija responsable del producto antes de la entrega 3 | Programa vía responsable del producto | el registro de decisiones del PRD |
| D20 | Tramos de duración 13.1 y 13.2. Provisional: los de la plantilla de la administración competente, configurables | Programa vía responsable del producto | el registro de decisiones del PRD |
| D21 | Carga decidida: todos los expedientes de 2026 en producción al acabar la fase 2. Sólo falta saber si hay hoja disponible; si no, teclean los equipos | Programa vía responsable del producto | el registro de decisiones del PRD |
| D22 | La Memoria 2025 da 294 niños/284 niñas en el apartado 2 y 290/288 en el 25. Indicar qué reparto es correcto y por qué | Programa vía responsable del producto | el registro de decisiones del PRD |
| Auditoría y DPD | Conservación y exportación de vínculos usuario-expediente; resolver en la EIPD | responsable del producto con el DPD | el registro de decisiones del PRD |
| Apartado G | Si la Memoria lleva el apartado G (valoración global), que está en la oficial de 2025 y no en la plantilla de la administración competente | responsable del producto | este documento |
| Dirección interna | dominio de ejemplo `etf.example.org`; configurar con Informática | responsable del producto con Informática | `TRD.md` |

## 10. Contradicciones encontradas

| Contradicción | Cómo se resuelve |
| --- | --- |
| El documento del cliente y el análisis daban perfiles y equipo por grupos de AD (`ETF-Coordinacion`, `ETF-01`…) | Resuelta en la entrevista: los permisos se dan en la aplicación; AD sólo valida la contraseña; «pendiente de alta» para quien entra sin perfil |
| Administración funcional sólo mantenía catálogos | Resuelta: también usuarios, plazos, plantillas, firmantes, temas e idiomas y exportación de auditoría; no ve fichas de expedientes |
| Escalas de los Anexos 11 y 12: el análisis y el registro de decisiones del PRD decían «en blanco» | Resuelta por el responsable del producto: se guardan [D17] |
| Fichero subido: el análisis decía sólo Word y 5 MB | Resuelta: anexo DOCX o PDF, 10 MB, antivirus local y aviso de «revisando»; sólo el anexo puede aceptarse con aviso auditado si el antivirus cae |
| El análisis preveía textos en `i18n/es/` dentro del frontend | Resuelta: textos en `idiomas/<código>/` fuera del código, también mensajes del servidor y plantillas |
| El análisis no preveía temas | Resuelta: temas instalables en `temas/<nombre>/` |
| Número de expediente: el texto dice equipo-correlativo-año y el ejemplo `01-08-26` dice lo contrario | Abierta [D1]; provisional `08-01-26`, formato configurable |
| Apartado 7 de la plantilla: «de años anteriores» definido como aceptados en el año evaluado | Resuelta por el Programa: errata; son los aceptados antes |
| Duración del tratamiento: la nota 22 de la Memoria dice del primer contacto a la despedida tras el seguimiento | Resuelta por el Programa: de la fecha del Acta de Aceptación a la del Acta de Cierre |
| Tramos de duración: plantilla de la administración competente frente a Memoria 2025 | Abierta [D20]; provisional la plantilla, configurable |
| Datos por familia (Programa) frente a recuentos por menor y sexo (Memoria 14, 26, 27) | Abierta [D4]; provisional aplicar a todos los menores con opción de desmarcar |
| Íllora: centro Pinos Puente-Moclín en un listado, ETF 6 en el otro | Dos asignaciones independientes por municipio [D9] |
| El Acta de Aceptación «con datos de los Anexos 1, 2, 3 y 4» cuando al aceptar sólo existe el 3 o el 4 | Abierta [D15]; provisional Anexo 3 o 4 |
| Anexos con «reconstruidas» y sin «extensas»; anexos con «Reintegración» frente a «Reunificación» | Se usa la lista de la Memoria; Reintegración = Reunificación |
| La regla «si no dicen nada, se guarda todo» frente a no guardar el DNI [D16] | Resuelta por el responsable del producto: no se guarda el DNI y el hueco del Word queda en blanco [D16] |
| El análisis no incluía el apartado G de la Memoria, que está en la oficial de 2025 y no en la plantilla de la administración competente | Hueco: se incluye como texto a falta de confirmación del responsable del producto |
| Quién cierra la Memoria de un equipo: el cliente dice que coordinación «cierra la Memoria anual»; el análisis deja que el equipo cierre la suya | Resuelta por el responsable del producto: el equipo cierra la suya y sólo Coordinación la provincial |
| Carga de sólo vivos y prevista en fase 4 | Resuelta: todos los expedientes de 2026, incluidos cerrados y no aceptados, en producción al acabar la fase 2; hoja si existe o tecleo [D21] |
| Datos reales de 2025 en preproducción como condición para validar Memoria | Resuelta: desarrollo y preproducción sólo usan datos inventados; resultados esperados por apartado validados por el Programa, sin esperar datos reales de 2025 |
| Producción y EIPD al final impedían cargar en fase 2 | Resuelta: servidor, copias y EIPD en paralelo desde el inicio; aprobación, seguridad y restauración verificadas antes de la carga real |
| Generar Word fijaba fechas de consentimiento o cierre | Resuelta: emisión separada del hito confirmado por una persona; regenerar no cambia fechas ni estados |
| Provincial descrita como suma de Memorias de equipo | Resuelta: cálculo directo desde datos con las mismas reglas; discrepancias con equipos cerrados visibles |
| Estadísticas con ficha actual y edades sin excepción | Resuelta: historia con vigencia, episodios de cierre/reapertura y nota 43 para quienes cumplen 18; reglas vigentes al 31/12 del año |
| Totales por sexo incompatibles en apartados 2 y 25 de Memoria 2025 | Abierta [D22]: el Programa debe fijar qué reparto es correcto y por qué |
| Auditoría posterior a una lectura o sin responsable de exportación | Resuelta: confirmar auditoría antes de entregar datos; exporta Administración funcional; conservación de vínculos al DPD |
| Textos de catálogo en base de datos y paquetes editables en el sitio | Resuelta: textos en directorios de idioma y versiones inmutables; activar referencia y auditoría juntas, sin carreras |
| Cierre de Anexo 5 ante cualquier respuesta | Resuelta: «a la espera» bloquea; sólo las dos respuestas resolutivas permiten cierre ordinario; Coordinación puede forzar con motivo auditado |
| Prohibición general de servicios externos frente a repositorio y pruebas | Resuelta: Repositorio público y CI sin datos reales y GitHub Actions autorizados sin datos personales; dependencias fijadas; sin IA, OCR ni análisis de seguridad en la nube |
| Despliegue con ejecutable y systemd en el análisis | Resuelta: Docker Compose (detalle en `TRD.md`) |

## 11. Cómo mantener este documento

Se actualiza en el mismo cambio cuando varían el alcance, los perfiles, la prioridad o una regla de negocio, y cuando el Programa contesta una duda [Dn]: se cambia la regla afectada, se quita la marca de provisional y se actualiza la fila del apartado 9. Lo que cambie qué hace la aplicación o cómo se ve lo decidel responsable del producto. Responsable de revisarlo: Responsable del producto.

## Configuración de la edición pública

Cada entidad fija su nombre visible con `entidad.nombre` en los catálogos de textos por idioma, su directorio corporativo, sus firmantes y su dominio. Las plantillas iniciales no incorporan logotipos ni contactos institucionales. Administración puede publicar sus propias versiones. Los municipios, equipos y asignaciones incluidos se conservan como datos de partida; su generalización queda pendiente. Las referencias normativas y los plazos requieren validación de la entidad y de su DPD antes de usar datos reales.
