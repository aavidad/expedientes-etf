# Diseño UI/UX de Expedientes ETF
> Especificación de la edición pública. La entidad que la instale configura sus datos, identidad y plantillas.
> Las referencias D1–D22 se conservan en el registro de decisiones del PRD; las fuentes privadas no se distribuyen.

Este documento dice cómo se ven y cómo se usan las pantallas de Expedientes ETF (ETF = Equipos de Tratamiento Familiar). Qué hace la app y para quién está en PRD.md; quién hace cada paso y en qué orden, en FLUJO_APP.md; cómo se construye (carga de temas e idiomas, seguridad, despliegue), en TRD.md; los datos y la API, en ESQUEMA_BACKEND.md; el orden de trabajo, en PLAN_IMPLEMENTACION.md.

Todavía no hay código. Las rutas del repositorio que aparecen aquí son previstas.

## 0. Decisiones de base

| Decisión | Elegido | Fecha y quién |
| --- | --- | --- |
| 1. Idiomas y textos | Internacionalización pura (i18n: preparar la app para varios idiomas). Ningún texto visible en el código, ni en Go ni en React: sólo claves como `expediente.aceptar`. Cada idioma vive en `idiomas/<código>/{textos,catalogos,plantillas}/`, fuera del código: textos de pantalla y mensajes en `textos/`, nombres de listas en `catalogos/` y Word en `plantillas/`. Los textos de catálogo nunca se guardan en la base de datos. Añadir un idioma es añadir una carpeta; la app lo detecta al arrancar o al recargar desde Administración. Si falta una clave, sale en castellano. | 13/10/2026 · responsable del producto |
| 8. Datos personales | Minimizar. En los listados, iniciales y número («Familia O. S.», 02-14-26). Nombre completo sólo en la ficha, y abrirla queda registrado. Datos de prueba inventados pero verosímiles, sin la palabra «ejemplo». | 13/10/2026 · responsable del producto |
| 11. Accesibilidad | WCAG 2.1 nivel AA (pautas de accesibilidad para contenido web) y documentos Word generados accesibles. | 13/10/2026 · responsable del producto |
| 12. Ayuda en pantalla | Sólo tras un botón «?». Nunca bloques fijos de ayuda en la pantalla. | 13/10/2026 · responsable del producto |
| 17. Idioma de documentos | Castellano llano, sin siglas sin explicar. | 13/10/2026 · responsable del producto |
| 18. Temas visuales | Temas instalables como en Joomla, en `temas/<nombre>/`, como paquetes de datos que Administración sube y activa en cualquier momento. Los componentes sólo usan tokens semánticos. El aspecto de la maqueta es el tema por defecto; habrá también uno de alto contraste. | 13/10/2026 · responsable del producto |
| Tono (entrevista 10.5) | Cercano y claro, de tú, con los términos del Programa sin cambiarlos (Acta, Anexo, subprograma…). | 13/10/2026 · responsable del producto |

## 1. Principios

1. **Se entiende sin curso.** Quien la usa tiene experiencia básica y no recibe formación previa (entrevista 3.4). Cada pantalla responde a tres preguntas a primera vista: dónde estoy, qué falta y qué puedo hacer ahora. Una acción principal por pantalla, en el botón verde.
2. **La ayuda se pide.** Las explicaciones de uso, definiciones y normativa van detrás de un botón «?» junto al título o al campo. En pantalla sólo queda texto que informa del caso: el estado, de dónde sale un dato («Encontrado en la dirección…», «Sale del municipio»), qué pasará al pulsar un botón y qué falta. Esta frontera es una interpretación de la decisión 12 a partir de la maqueta; responsable del producto la confirma (ver apartado 11).
3. **Los términos del Programa no se tocan.** Acta de Aceptación, Anexo 3, subprograma, consentimiento, pendiente de atender, Tabla Mensual para el SPAF (Servicio de Prevención y Apoyo a la Familia de la administración competente), Memoria Técnica Anual. Los equipos ya los usan; traducirlos a otro vocabulario confunde.
4. **Lo que falta se ve en naranja, con texto y con enlace.** El naranja nunca va solo: siempre acompaña una frase («Obligatorio para aceptar») y, en el aviso lateral, un enlace que lleva al campo. Nada se marca sólo por color.
5. **Los errores dicen qué pasó y qué hacer.** Sin códigos técnicos ni culpas. Por ejemplo: «El antivirus ha encontrado un problema en el fichero. No se ha guardado nada. Pide al servicio que lo envíe de nuevo.» Si hace falta soporte, se da un número de incidencia que no contiene datos personales.
6. **Los datos personales sólo se ven donde hacen falta.** Listados enmascarados, ficha completa con constancia, estadísticas sin nombres (apartado 9).
7. **El aspecto es un tema, no código.** Colores, fuentes, tamaños, radios y espacios salen de tokens semánticos que define el tema activo (apartado 3). Cambiar de tema no toca una línea de código.
8. **Los textos son datos.** Cada texto de interfaz sale del catálogo del idioma (apartado 8). Los datos libres del expediente y los textos narrativos de la Memoria son datos de negocio y conservan lo escrito por la persona. Los textos que aparecen en este documento son propuestas para el catálogo `idiomas/es/`, no cadenas para escribir en el código.

## 2. Público y contexto de uso

| Perfil | Quién | Pantallas que usa | Dispositivo y contexto |
| --- | --- | --- | --- |
| Coordinación provincial | Hasta 3 personas (el coordinador y, más adelante, personal administrativo) | Inicio de coordinación, derivaciones recibidas y su revisión, lista de espera, cierres propuestos, expedientes de la provincia, ficha, Memoria provincial, SMI (salario mínimo interprofesional), comisiones e informes | Ordenador de oficina con pantalla ancha, red corporativa o VPN (red privada virtual) |
| Profesional de ETF | 3 por equipo (trabajador/a social, educador/a social, psicólogo/a), preparado para 4-5; 8 equipos hoy, hasta 12 | Bandeja del equipo, ficha del expediente, documentos, Tabla SPAF, comisiones e informes, Memoria del equipo | Ordenador de oficina o portátil desde la red provincial de ayuntamientos o VPN; nunca desde Internet abierto |
| Consulta de estadísticas | Pocas personas | Sólo cifras de Memoria y totales, sin composición de las cifras ni textos libres | Ordenador de oficina |
| Administración funcional | 1-2 personas | Usuarios y perfiles, catálogos, plazos y tramos, plantillas Word, firmantes y destinatarios, temas, idiomas y exportación de auditoría. No ve expedientes. | Ordenador de oficina |

Un profesional pertenece a un solo equipo en cada momento. Una persona puede tener varios perfiles (por ejemplo, coordinación y administración); el menú reúne las entradas de todos sus perfiles. Quien entra con cuenta de AD (Active Directory, el directorio de usuarios de la entidad) y no tiene perfil ve una pantalla de «pendiente de alta». El detalle de perfiles está en PRD.md y el reparto de pasos en FLUJO_APP.md.

Uso real: jornada de 8 a 15 h, muchas pestañas abiertas a la vez, trabajo interrumpido por llamadas y visitas. Por eso los avisos de «falta» se recalculan al volver a la pantalla y las fichas avisan de cambios sin guardar.

## 3. Sistema visual

### 3.1 Temas: paquetes de datos que se instalan y activan

El aspecto de toda la app lo da un **tema**: un paquete independiente en `temas/<nombre>/`, como las plantillas de Joomla. Administración funcional lo sube, lo prueba en su propia sesión y lo activa cuando quiere; desde ese momento toda la app cambia de aspecto sin reiniciar ni tocar código. En Docker, `temas/` es un volumen montado (ver TRD.md).

Reglas que no admiten excepción:

- Ningún color, fuente, tamaño, radio, sombra ni espaciado escrito en el código de pantallas o componentes. Ninguna hoja de estilos de tema dentro del código. Los componentes sólo leen tokens semánticos (lista en 3.3).
- El paquete de tema es **sólo datos**: valores de tokens en JSON, fuentes WOFF2 e imágenes PNG para iconos y logotipo. No admite SVG ni recursos remotos. No lleva hojas de estilo ni JavaScript. La app valida los valores y genera las variables a partir de los tokens; no interpreta código del paquete. El motivo es de seguridad y de mantenimiento; los detalles de carga y validación están en TRD.md.
- Las fuentes van dentro del paquete (formato WOFF2). La maqueta las carga de Google Fonts; la app no, porque no puede llamar a servicios externos (entrevista 9.7).
- Un token que falte en un tema se toma del tema por defecto, igual que una clave de idioma que falte sale en castellano (propuesta; responsable del producto la confirma).

Contenido previsto de un paquete (la estructura exacta de ficheros la fija TRD.md):

```
temas/
  expedientes-etf/        tema por defecto; viene con la app y no se puede retirar
    tema.json             manifiesto: identificador, nombre por idioma, versión,
                          autor, licencia, versión mínima de la app, tipo
                          (claro / oscuro / alto contraste)
    tokens.json           valores de los tokens del apartado 3.3
    fuentes/              Public Sans y Source Serif 4 (licencia libre OFL)
    iconos/               opcional: iconos PNG que sustituyen a los comunes
    logotipo.png          opcional (hoy no hay logotipo de la entidad)
  alto-contraste/         segundo tema incluido
```

Temas incluidos: **Expedientes ETF** (el aspecto de la maqueta) y **Alto contraste** (por diseñar; ver 3.6). Un tema oscuro no está pedido; se podrá instalar como cualquier otro.

Quién elige: Administración funcional activa el tema de toda la organización. Si cada persona puede elegir para sí el tema de alto contraste (como preferencia de accesibilidad) no está decidido: **pendiente del responsable del producto**. Propuesta: sí, sólo entre los temas instalados de tipo alto contraste.

### 3.2 El tema por defecto, sacado de la maqueta

Diseño de referencia a 1440 px de ancho. Fondo crema, tarjetas blancas con borde fino, un verde petróleo como único color de acción, naranja para lo que falta y violeta suave como segundo color de categoría. Sin sombras: la profundidad la dan el borde y el contraste entre fondo crema y tarjeta blanca.

- **Tipografía.** Public Sans (400, 500, 600 y 700) para todo; Source Serif 4 (700) sólo para el título de página. Tamaño de texto normal 15 px; mínimo 13 px. Las cifras, fechas y números de expediente usan números tabulares para que se alineen en columna.
- **Color.** Ver la tabla de tokens (3.3). Todas las parejas de texto de la maqueta pasan el contraste AA salvo los casos anotados en 3.5.
- **Espaciado y retícula.** Cabecera de 64 px; menú lateral de 256 px; contenido con márgenes de 32-40 px; columna lateral de avisos y resumen de 360 px (380 en la maqueta de derivación; se unifica). Rejillas de formulario de 3 columnas iguales con 16 px de hueco; indicadores en 4 columnas.
- **Controles.** Todo lo que se pulsa mide 44 px de alto (campos, desplegables, botones, pestañas, entradas del menú). Los botones de decisión de la derivación, 48 px. La maqueta tiene algunos a 40 px; se suben a 44 (apartado 11).
- **Radios.** 8 px en campos de formulario, 10 px en botones, buscador y entradas del menú, 12 px en avisos informativos, 14 px en tarjetas, completamente redondeado en píldoras y fichas seleccionables.

### 3.3 Lista de tokens semánticos

Los componentes usan estos nombres y nada más. Un tema da un valor a cada uno. La app los expone como variables (por ejemplo `--color-fondo`). Añadir un token es un cambio de este documento.

**Color**

| Token | Para qué se usa | Tema por defecto |
| --- | --- | --- |
| `color.fondo` | Fondo de la página | #f4f2ec |
| `color.superficie` | Tarjetas, cabecera, menú lateral, tablas, campos | #ffffff |
| `color.superficie-suave` | Cabecera de tabla, buscador | #faf9f5 |
| `color.superficie-neutra` | Píldoras de dato, número que se asignará, contadores neutros | #eef0f2 |
| `color.borde` | Borde de tarjetas, cabecera y menú (decorativo) | #dcd8cc |
| `color.borde-suave` | Separadores dentro de una tarjeta, subtarjetas | #e6e2d8 |
| `color.borde-fila` | Separador entre filas de tabla | #ece8de |
| `color.borde-control` | Borde de campos, desplegables, botones secundarios, fichas sin marcar, paso pendiente | maqueta #c9c4b6; **propuesta #8a8577** (ver 3.5) |
| `color.texto` | Texto principal | #1d2327 |
| `color.texto-secundario` | Etiquetas, metadatos, subtítulos, ruta | #4a5055 |
| `color.texto-apagado` | «Sin avisos», paso futuro. Sólo sobre `color.superficie` | #6b7075 |
| `color.icono` | Iconos neutros (lupa del buscador) | #5b6166 |
| `color.acento` | Botón principal, enlaces, pestaña activa, paso hecho, borde de ficha marcada | #0f5e63 |
| `color.acento-fuerte` | Enlace al pasar el ratón, texto sobre `color.acento-suave` | #0a4346 |
| `color.acento-suave` | Entrada de menú actual, ficha marcada, píldora del subprograma de Riesgo, avatar | #e3efed (la maqueta usa #d9e8e6 en el avatar; se unifica) |
| `color.sobre-acento` | Texto e iconos sobre `color.acento` | #ffffff |
| `color.calculado-fondo` | Campo calculado o deducido, de sólo lectura; fondo del aviso informativo | #f1f6f5 |
| `color.calculado-texto` | Texto del campo calculado | #0a4346 |
| `color.info-borde` | Borde del aviso informativo | #b9d3cf |
| `color.aviso-fondo` | Fondo de tarjeta o campo con algo pendiente | #fff8f0 |
| `color.aviso-borde` | Borde de la tarjeta de aviso | #e8b886 |
| `color.aviso-borde-campo` | Borde de 2 px del campo pendiente | #c97a2b |
| `color.aviso-texto` | Título y enlaces del aviso, «Obligatorio para aceptar», «Sin indicar» | #8a3b00 |
| `color.aviso-texto-cuerpo` | Texto de listas y notas dentro del aviso | #4a2a0c (la maqueta usa también #6b3a10; se unifica) |
| `color.aviso-pildora` | Fondo de la píldora y del contador de aviso | #fbe6d0 |
| `color.destacado-fondo` | Segunda categoría: píldora de Reunificación, «Atención Temprana propuesta», avatar de coordinación | #e7e4f4 |
| `color.destacado-texto` | Texto sobre `color.destacado-fondo` | #3b2f7a |
| `color.neutro-texto` | Texto de píldoras neutras y pestañas no activas | #3a4046 |
| `color.peligro` | Texto de acciones negativas o destructivas («No aceptar»), mensajes de error | #9a2a1f |
| `color.peligro-fondo` | Fondo del resumen de errores | **falta en la maqueta**; lo propone quien diseñe el primer error y lo aprueba responsable del producto |
| `color.foco` | Contorno de foco del teclado | **falta en la maqueta**; propuesta: el valor de `color.acento` |

**Tipografía**

| Token | Uso | Tema por defecto |
| --- | --- | --- |
| `fuente.texto` | Todo el texto | Public Sans, con reserva system-ui, sans-serif |
| `fuente.titulo` | Título de página (h1) | Source Serif 4, con reserva Georgia, serif |
| `texto.titulo-pagina` | h1 | 32 px · 700 · interletra -0,01 em |
| `texto.cifra` | Cifra grande de un indicador | 34 px · 700 |
| `texto.titulo-seccion` | h2 de tarjeta en la columna principal | 18 px · 700 |
| `texto.marca` | Nombre de la app en la cabecera | 17 px · 700 · interletra -0,01 em |
| `texto.titulo-lateral` | h2 de tarjeta en la columna lateral | 16 px · 700 |
| `texto.normal` | Cuerpo, tablas, campos | 15 px · 400 · interlineado 1,5 |
| `texto.etiqueta` | Etiqueta de campo (600), filtros, ruta, listas de avisos | 14 px |
| `texto.meta` | Metadatos, cabecera de tabla (600), píldoras (600) | 13 px · interlineado 1,45 |
| `peso.normal`, `peso.medio`, `peso.seminegrita`, `peso.negrita` | Pesos | 400, 500, 600, 700 |

Ningún texto por debajo de 13 px (la maqueta tiene píldoras de 12,5 px; se corrigen).

**Espacio, tamaño, radio y borde**

| Token | Uso | Tema por defecto |
| --- | --- | --- |
| `espacio.1` … `espacio.10` | Escala de huecos | 4, 6, 8, 10, 12, 16, 20, 24, 32, 40 px |
| `espacio.etiqueta` | Entre etiqueta y campo | 6 px |
| `espacio.tarjeta` | Relleno de tarjeta (vertical · horizontal) | 20 · 24 px (columna lateral 18 · 20) |
| `espacio.entre-tarjetas` | Hueco entre tarjetas y bloques | 20 px (24 entre columnas) |
| `espacio.pagina` | Margen del contenido | 32 · 40 px |
| `alto.control` | Campos, botones, pestañas, entradas de menú, filtros, fichas | 44 px |
| `alto.boton-decision` | Botones de decisión (Aceptar, Lista de espera, No aceptar) | 48 px |
| `alto.cabecera` | Cabecera de la app | 64 px |
| `ancho.menu` | Menú lateral | 256 px |
| `ancho.lateral` | Columna lateral de avisos y resumen | 360 px |
| `ancho.buscador` | Máximo del buscador | 520 px |
| `tamano.avatar` · `tamano.paso` · `tamano.marca` · `tamano.icono` · `tamano.casilla` | Círculo de iniciales, círculo de la línea de tiempo, logotipo, iconos, casillas y botones de opción | 40 · 28 · 32 · 18-20 · 18 px |
| `radio.control` | Campos, desplegables, botones pequeños | 8 px |
| `radio.boton` | Botones, buscador, entradas de menú, subtarjetas | 10 px |
| `radio.aviso` | Aviso informativo | 12 px |
| `radio.tarjeta` | Tarjetas e indicadores | 14 px |
| `radio.pildora` | Píldoras, fichas seleccionables, contadores | 999 px |
| `borde.normal` · `borde.marcado` · `borde.actual` | Bordes de 1 px; campo pendiente y ficha marcada; paso actual y pestaña activa | 1 · 2 · 3 px |
| `sombra.tarjeta` | Sombra de tarjeta | ninguna |
| `movimiento.duracion` | Transiciones | 0 si la persona pide reducir movimiento; el tema por defecto no anima nada |

### 3.4 Iconos y marca

La maqueta usa pocos iconos: la marca (un cuadrado verde con tres líneas), la lupa del buscador, el círculo de información y la marca de hecho («✓»). Todos son decorativos (`aria-hidden`) y van acompañados de texto. Los iconos y la marca viven en el paquete del tema como PNG; sus variantes deben cumplir el contraste del tema. No se admiten SVG en los paquetes. Sin logotipo de la entidad por ahora (entrevista 1.4).

### 3.5 Contraste del tema por defecto

Medido con la fórmula de WCAG sobre los valores de la maqueta:

- Texto principal y secundario: entre 7,3:1 y 15,9:1. Correcto.
- Botón principal (blanco sobre #0f5e63): 7,5:1. Píldoras de categoría y aviso: entre 6,4:1 y 9,4:1. «No aceptar» (#9a2a1f): 7,7:1. Correcto.
- `color.texto-apagado` (#6b7075): 5,0:1 sobre blanco, pero 4,47:1 sobre el fondo crema. Sólo se usa sobre superficie blanca, o se oscurece.
- `color.borde-control` (#c9c4b6): 1,74:1 sobre blanco. **No cumple** el criterio 1.4.11 (los límites de un campo deben contrastar 3:1). Propuesta: #8a8577 (3,7:1). Cambia el aspecto, así que lo aprueba responsable del producto.
- `color.aviso-borde-campo` (#c97a2b): 3,3:1 sobre blanco y 3,2:1 sobre el fondo naranja. Cumple, justo.
- `color.borde` (#dcd8cc) no cumple 3:1, pero sólo dibuja contornos decorativos de tarjeta. Nunca debe ser el único límite de algo que se pulsa.

Al instalar un tema, la app comprueba estas mismas parejas y no deja activarlo si alguna baja de AA (propuesta para TRD.md).

### 3.6 Tema de alto contraste

Pedido en la entrevista (10.2) y sin diseñar. Requisitos mínimos: texto a 7:1, límites de controles a 4,5:1, foco de 3 px, sin depender del fondo crema para separar tarjetas, mismos tamaños y espacios que el tema por defecto. Lo diseña el agente de interfaz y lo aprueba responsable del producto antes de la fase 0 (ver PLAN_IMPLEMENTACION.md).

### Aplicación instalada en el escritorio (PWA1)

Edge y Chrome pueden abrir Expedientes ETF en una ventana propia. Conserva las mismas pantallas, permisos y preferencias que en el navegador. La marca ETF de la cabecera sirve también como icono de la aplicación, con los colores del tema seleccionado. No se añade una pantalla ni un botón de instalación dentro de la aplicación; se utiliza la opción del navegador, descrita en el manual de usuario.

Sin conexión, una navegación muestra «Sin conexión con el servidor», una indicación para comprobar la conexión a la red de la Diputación y «Volver a intentarlo». El aviso usa el idioma y los tokens del tema seleccionados, admite teclado y ancho móvil, y no muestra nombres ni datos de expedientes. La aplicación instalada necesita conexión con el servidor para consultar o guardar datos.

## 4. Componentes

Biblioteca de componentes accesibles sobre React Aria (ver TRD.md). Ruta prevista: `web/src/componentes/` (la fija TRD.md). Cada componente toma colores y medidas sólo de tokens y textos sólo de claves de idioma.

UI1 reúne la cabecera, las preferencias y el menú en `web/src/armazon.tsx`, compartido por entrada, pendiente, Usuarios, Catálogos, Plazos y tramos, Firmantes y Paquetes (incluidas sus entradas de idiomas y temas), además del SMI de Coordinación. La marca es un cuadrado con «ETF», sin apariencia de control. Las rutas disponibles y sus perfiles se reciben en la sesión desde el registro HTTP por módulos, del que se deriva `tablaRutas()` para las pruebas de autorización. INT5 reúne los componentes y las entradas del menú en `web/src/<m>/pantallas.ts`, con orden explícito en `web/src/pantallas.ts`; el menú muestra las entradas cuyo perfil y ruta autoriza también la sesión. Los módulos sin pantalla no aparecen. El enlace «Saltar al contenido» aparece al recibir foco. Entrada lleva una tarjeta centrada y un solo título, «Entrar».

UI2 reúne el acceso de idiomas y temas en una sola entrada, «Administración de idiomas y temas»; las direcciones anteriores siguen disponibles y señalan la misma entrada activa. Catálogos presenta la etiqueta publicada del idioma activo, con reserva en castellano y un aviso legible cuando falta; nunca sustituye la etiqueta por la clave. Los identificadores para Administración quedan en un desplegable de la edición. La ayuda aparece junto al título y se cierra con Escape. Plazos y tramos muestran fechas numéricas y acciones secundarias. Firmantes identifica al personal por su nombre completo. SMI presenta filas como tarjetas en la disposición compacta, con la etiqueta visible de cada importe mensual y las cabeceras accesibles de la tabla.

Comisiones e informes usa UI1 en `/equipo/comisiones`. Ofrece filtros de año,
fechas, equipo y organismo, y presenta la tabla como tarjetas en ancho reducido.
El formulario comprueba los datos y conserva los cambios ante un conflicto.
Retirar exige confirmar su consecuencia; la ayuda se abre aparte. Profesional
de ETF escribe en su equipo y Coordinación consulta el ámbito provincial.

TR1 añade Traslado desde Salidas y cierre de Coordinación y Seguimiento como pestaña independiente de la ficha. El traslado identifica los bloqueos antes del formulario y enlaza con la aceptación del destino. La fecha de cierre se valida de nuevo en el servidor frente a la apertura o reapertura vigente; un rechazo dirige al campo. Seguimiento muestra las fechas del plazo, el estado y el motivo, con confirmación para terminar o reabrir. En esta pestaña se omite la cronología de intervención y su fase: el seguimiento conserva su propio plazo. Su pestaña activa se mantiene visible en móvil y el acceso directo lleva al panel. Los motivos activos se eligen por su etiqueta; los retirados conservan su etiqueta en las lecturas históricas.

La cabecera muestra el nombre completo del personal y debajo sus perfiles y equipo. El nombre se recibe sólo en la sesión activa; pendiente de alta sigue sin datos de la persona. La sesión aporta el identificador del equipo; UI3 resuelve su nombre publicado mediante las lecturas autorizadas de Equipo y lo muestra bajo los perfiles. Mientras carga anuncia la espera; si falla, permite reintentar desde el menú de persona. No se amplían permisos de catálogo ni se consultan API de Administración desde otros perfiles. Usuarios es una tabla accesible con nombre completo del personal, estado en píldora, perfiles, equipo y acción «Editar». Esta identificación del personal es una excepción expresa del encargo UI1; las familias y los menores siguen enmascarados en los listados.

UI3 organiza el menú de Coordinación en Derivaciones, Expedientes, Informes y Memoria, y Configuración. Sólo aparecen secciones con entradas autorizadas. La bandeja identifica cada derivación por su número o, si todavía no lo tiene, por fecha de recepción y municipio. La revisión deja visibles los datos del anexo con valor y reúne los ausentes en «Datos que el anexo no trae (N)», plegado al abrir. Los avisos de Menores identifican campo, orden y nombre, y llevan el foco al campo correspondiente.

Por debajo de cuatro veces el token `ancho.menu` (1024 px en los temas incluidos), la cabecera ocupa una fila: marca, «Menú» cuando hay pantallas y «Persona». El menú lateral empieza recogido y vuelve a recogerse al navegar. El menú de persona contiene nombre, perfiles, equipo, preferencias y «Cerrar sesión». Escape cierra el panel correspondiente y devuelve el foco a su control. Las filas de Usuarios se presentan como tarjetas con rótulos, conservando la tabla y sus cabeceras accesibles. La guardia de cambios sin guardar se conserva al navegar, cambiar de idioma, salir y usar Atrás o Adelante.

| Componente | Cuándo se usa | Dónde está | Reglas |
| --- | --- | --- | --- |
| Cabecera de la app | Siempre | `web/src/componentes/cabecera/` (prevista) | 64 px. Marca y nombre a la izquierda; buscador (máx. 520 px; el texto de muestra cambia por perfil: «Buscar por nº de expediente o familia» / «Buscar en todos los equipos»); selector de idioma con etiqueta visible «Idioma»; persona con iniciales, nombre, puesto y equipo. Los resultados del buscador salen enmascarados. |
| Menú lateral | Siempre | `…/menu-lateral/` (prevista) | 256 px, `nav` con nombre accesible «Principal». Entrada actual con `aria-current="page"`, fondo `color.acento-suave` y seminegrita. Contadores a la derecha: naranja si exige acción (derivaciones recibidas), neutro si sólo informa (lista de espera, cierres propuestos). Al pie, el ámbito: «Equipo · ETF-2…» o «Ámbito · Los 8 equipos de la provincia». Las entradas dependen del perfil (FLUJO_APP.md). |
| Ruta (migas) | Pantallas de segundo nivel | `…/ruta/` (prevista) | `nav` «Ruta», 14 px, separador «›» oculto al lector de pantalla. |
| Encabezado de página | Toda pantalla | `…/encabezado-pagina/` (prevista) | h1 con `fuente.titulo` 32 px; debajo, una línea de contexto (equipo, anexo y fecha). Acciones a la derecha, la principal la última. Botón «?» junto al h1 cuando la pantalla tiene ayuda. |
| Botón | Acciones | `…/boton/` (prevista) | 44 px, radio 10, seminegrita. Variantes: principal (fondo `color.acento`, una por pantalla), secundario (superficie y `color.borde-control`), peligro (secundario con texto `color.peligro`). Etiqueta con verbo y objeto («Generar documento», «Derivar al SPM»). No se desactiva para explicar que falta algo: se pulsa y muestra qué falta. |
| Botón de decisión | Decisiones con consecuencias (aceptar, lista de espera, no aceptar, cerrar) | `…/decision/` (prevista) | 48 px, texto alineado a la izquierda, con la consecuencia debajo en 13 px («Asigna el número y descarga el Acta de Aceptación y el oficio al Centro de Servicios Sociales»). La consecuencia no es ayuda: se queda fija. |
| Botón «?» | Ayuda de una pantalla o de un campo | `…/ayuda/` (prevista) | 44 × 44 px, nombre accesible «Ayuda sobre <tema>». Abre un panel o diálogo con el texto del catálogo `ayuda.*`; se cierra con Escape y devuelve el foco. **No aparece en la maqueta** (apartado 11). |
| Tarjeta o sección | Agrupar campos o datos | `…/tarjeta/` (prevista) | `section` con h2 enlazado por `aria-labelledby`. Superficie, borde 1 px, radio 14, relleno 20 · 24. Acción secundaria de la tarjeta a la derecha del h2 («Añadir persona adulta»). |
| Indicador | Cifras de la bandeja e inicio | `…/indicador/` (prevista) | Etiqueta 14 px, cifra 34 px, detalle 13 px. Variante aviso (fondo, borde y textos naranjas) cuando la cifra exige actuar («Sin consentimiento tras 30 días»). Si la cifra es mayor que cero, enlaza a la lista filtrada. |
| Tabla de datos | Listados y tablas de la ficha | `…/tabla/` (prevista) | `table` real con `th scope="col"`, cabecera 13 px seminegrita sobre `color.superficie-suave`, filas de 15 px separadas por `color.borde-fila`. El número de expediente es el enlace de la fila. Cifras con números tabulares. Título asociado (`caption` o `aria-labelledby`). |
| Filtro | Encima de una tabla | `…/filtro/` (prevista) | Desplegable con etiqueta visible a su izquierda; 44 px (la maqueta usa 40). Cambiar un filtro no recarga la página entera y anuncia «N expedientes» en una región de estado. |
| Píldora | Categoría o estado de un dato | `…/pildora/` (prevista) | Radio completo, 13 px seminegrita. Tipos: categoría A (subprograma de Riesgo: acento suave), categoría B (Reunificación, Atención Temprana: destacado), neutra (número, servicio derivante, referente), aviso (naranja). El texto dice el estado; el color sólo lo refuerza. |
| Ficha seleccionable | Elegir varios de una lista corta (responsables, tipologías) | `…/ficha-seleccionable/` (prevista) | Botón con `aria-pressed`. Marcada: borde 2 px `color.acento`, fondo acento suave y «✓» delante. Sin marcar: borde `color.borde-control`. Si la lista es larga, se ven las frecuentes y un «Ver las 12». |
| Campo | Texto, número, fecha, desplegable, opción única, casilla, área de texto | `…/campo/` (prevista) | Etiqueta visible encima (14 px seminegrita), separación 6 px, control de 44 px y radio 8. Las opciones únicas y grupos de casillas van en `fieldset` con `legend`. Estados: normal, foco, **pendiente** (borde 2 px `color.aviso-borde-campo`, fondo `color.aviso-fondo` y frase naranja debajo), error (frase en `color.peligro` enlazada con `aria-describedby`), sólo lectura. |
| Campo calculado | Dato que la app deduce y no se escribe (tramo para la Memoria, centro y equipo según municipio, subprograma del Anexo 3) | `…/campo-calculado/` (prevista) | Mismo alto que un campo, fondo `color.calculado-fondo`, texto seminegrita. Debajo, de dónde sale («Sale del municipio», «calculado»). No parece editable. |
| Origen del dato | Datos leídos del anexo | dentro de Campo (prevista) | Línea de 13 px bajo el campo con la frase del anexo entre comillas latinas: «Encontrado en la dirección: «C/ Real 12, Chauchina»». Si se ha traducido a la lista del Programa: «En el anexo: «Primarios»». |
| Aviso informativo | Explicar el estado de la pantalla tras una acción | `…/aviso/` (prevista) | `role="status"`, fondo calculado, borde `color.info-borde`, radio 12, icono de información. Una o dos frases. |
| Aviso «Falta…» | Lo que impide avanzar o falta para la Memoria | `…/aviso-falta/` (prevista) | Tarjeta naranja en la columna lateral, h2 con la cuenta («Faltan 2 datos para la Memoria», «Falta para poder aceptar»), lista de enlaces que llevan al campo y le dan el foco. Lo que se puede dejar para luego, en una nota aparte. |
| Resumen de errores | Al pulsar una acción con datos que faltan o son incorrectos | `…/resumen-errores/` (prevista) | Sale arriba del formulario, recibe el foco, enumera cada problema con enlace a su campo. |
| Línea de tiempo | Fases del expediente en la ficha | `…/linea-tiempo/` (prevista) | Lista ordenada con nombre «Fases del expediente». Paso hecho: círculo de 28 px relleno de acento con «✓», fecha efectiva confirmada y documento debajo; una generación de Word no completa el paso. Paso actual: anillo de 3 px de acento, `aria-current="step"`, nombre en negrita `color.acento-fuerte`, progreso («Mes 8 de 15»). Paso pendiente: anillo de 2 px `color.borde-control`, texto apagado. El estado se dice también en texto oculto («hecho», «pendiente»), porque el «✓» está oculto al lector. |
| Pestañas | Secciones de la ficha | `…/pestanas/` (prevista) | `tablist` con nombre, 44 px. Activa: negrita, línea inferior de 3 px de acento. Cuenta entre paréntesis cuando ayuda («Menores (3)»). Flechas para moverse; Tab entra en el panel. Cambiar de pestaña con cambios sin guardar pregunta antes. |
| Subtarjeta de persona | Menores en la columna lateral | `…/subtarjeta/` (prevista) | Borde suave, radio 10. Nombre, sexo y edad, píldoras de subprograma y escolarización. |
| Lista de fechas | «Próximas fechas» | `…/lista-fechas/` (prevista) | Fecha en columna fija de 88 px con números tabulares y el hito al lado. |
| Actividad reciente | Últimos cambios del expediente | `…/actividad/` (prevista) | Quién (seminegrita), qué, y cuándo en 13 px («Ayer, 12:40»). Sale del registro de auditoría; nunca muestra valores de datos personales. |
| Desplegable de detalle | Datos secundarios largos | `…/detalle/` (prevista) | `details`/`summary` nativo; el resumen dice qué contiene. |
| Subida de fichero | Anexo 3 o 4, plantilla, tema, idioma u hoja de carga | `…/subida/` (prevista) | Estados en 5.5 y política por tipo en 5.10. |
| Diálogo de confirmación | Acciones que no se deshacen (aceptar, no aceptar, cerrar, reabrir, cerrar la Memoria, activar un tema) | `…/dialogo/` (prevista) | Título con la acción, qué va a pasar y qué no se puede deshacer, botón con el mismo verbo («Aceptar y asignar el 02-32-26»), «Volver» como segunda opción. Foco atrapado, Escape cancela. |
| Estado vacío | Lista sin filas | `…/vacio/` (prevista) | Una frase que diga por qué y, si procede, la acción («No hay derivaciones por revisar»). Sin ilustraciones. |
| Cargando | Esperas de más de medio segundo | `…/cargando/` (prevista) | Texto en región de estado («Cargando expedientes…»); nunca una rueda sin texto. |
| Paginación | Listados de provincia | `…/paginacion/` (prevista) | Página anterior y siguiente, «Página 2 de 7», número total. El tamaño de página se configura (FLUJO_APP.md). |
| Selector de idioma | Cabecera | `…/selector-idioma/` (prevista) | Lista los idiomas encontrados en `idiomas/`, cada uno con su nombre en su propio idioma («Español», «English»). Cambia `lang` de la página. |

## 5. Patrones por tipo de pantalla

Estructura común a todas: cabecera arriba, menú lateral a la izquierda, contenido con ruta, encabezado de página y bloques en tarjetas. Las pantallas de trabajo con un caso (ficha, revisión de derivación) añaden una columna lateral derecha con lo que falta, el resumen y la decisión.

### 5.1 Bandeja del equipo (maqueta «Bandeja del equipo»)

F2.06 ofrece búsqueda por número, apellidos o iniciales y filtros de estado, subprograma y fase, con páginas de hasta 50 expedientes. Muestra número, familia enmascarada, subprograma, número de menores, estado, fase, fechas y avisos de cierre pendiente, derivación al Servicio de Protección de Menores sin respuesta y avisos calculados de F2.02. Los campos pendientes de la Memoria se agrupan en un aviso por expediente. Coordinación puede consultar la provincia; el profesional sólo su equipo, también cuando tiene ambos perfiles. Los indicadores de capacidades aún pendientes se incorporarán con sus fuentes de datos.

- Encabezado: «Bandeja del equipo» y debajo el nombre del equipo. Acción secundaria «Registrar comisión o informe» y principal «Tabla SPAF de <mes anterior>».
- Cuatro indicadores: expedientes activos (con reparto por subprograma), sin consentimiento tras 30 días (en naranja, «Cuentan como pendientes de atender»), derivados al SPM (Servicio de Protección de Menores de la administración competente) sin respuesta, cerrados en el año. Cada cifra mayor que cero filtra la tabla.
- Tabla «Expedientes del equipo» con filtros de subprograma y fase. Columnas: número (enlace a la ficha), familia **enmascarada** («Familia O. S.»), subprograma (píldora), menores, fase, referente y aviso (píldora naranja si exige acción, neutra si informa, texto apagado «Sin avisos»).
- Los avisos son los de FLUJO_APP.md (30 días sin consentimiento, fase vencida, Anexo 5 sin respuesta, falta Anexo 11/12, fin de seguimiento, datos para la Memoria). Sólo dentro de la app; no hay correo.
- Orden por defecto: primero los que tienen aviso naranja, luego por número (propuesta).

F1.10 dispone de bandeja en `/coordinacion`, con cuatro indicadores que filtran: pendientes de revisar, aceptados recientes, no aceptados y lista de espera. Incluye filtros, búsqueda enmascarada, paginación y tarjetas en móvil. Los indicadores cuentan los resultados de los filtros de equipo, municipio, fechas y búsqueda. Los indicadores adicionales del **inicio de coordinación** siguen el mismo patrón con otros indicadores (derivaciones por resolver, lista de espera, cierres propuestos, pendientes de atender de todo el ámbito de la entidad, carga por equipo). No está en la maqueta.

### 5.2 Ficha del expediente (maqueta «Ficha del expediente»)

F2.06 muestra los hitos confirmados y las emisiones en bloques separados. Las pestañas disponibles son Familia (lectura de F2.01a), Menores e Intervención (FI1), Consentimiento, Evaluación, Documentos, Cierre, Salidas e Historial; las capacidades pendientes se ocultan. Cierre permite guardar la finalización y proponer o retirar el cierre mediante F3.02a, con versión, protección de cambios sin guardar y errores traducidos. Salidas conecta la preparación, descarga del Anexo 5, fecha de envío y respuesta al Servicio de Protección de Menores de F3.01a. Historial presenta las últimas 100 acciones, sin valores personales ni códigos técnicos. Los autores se conservan como identificadores en los metadatos; la pantalla los muestra como personal del Programa mientras no exista el puerto de nombres autorizado.

La ficha monta sus nueve pestañas desde un registro común de código, etiqueta y componente. Conserva la última ficha del mismo expediente durante el refresco y la elimina ante un error; Consentimiento y Evaluación bloquean sus acciones mientras llega la nueva versión.

Consentimiento descarga el Anexo 1 o 2 según el subprograma y confirma por separado la fecha de firma; también registra la fecha del Anexo 7 u 8, preparado fuera de la aplicación. El aviso de plazo procede de F2.02. Evaluación registra instrumentos por adulto y momento; las respuestas por ítem usan definiciones publicadas. Cuando un cuestionario todavía no esté publicado, permite registrar qué instrumento has aplicado. Las escalas son opcionales: una línea explica que puedes anotarlas aquí o rellenarlas después en el Word de los Anexos 11 y 12. No calcula puntuaciones totales ni muestra avisos de escalas pendientes. Su ausencia no bloquea documentos, cierre o Memoria. Ambas pestañas protegen cambios sin guardar y ofrecen recuperar los datos actuales ante un conflicto.

- Ruta «Bandeja del equipo › Expediente 02-14-26». h1 con el nombre completo de la familia («Familia Ortega Salas»). Debajo, píldoras: subprograma, número, servicio derivante y referente.
- Acciones: «Generar documento» y «Derivar al SPM» (secundarias) y «Guardar cambios» (principal). «Generar documento» abre la misma lista que la pestaña Documentos.
- Línea de tiempo de fases (componente en 4). Pasos según FLUJO_APP.md: derivación, aceptación, consentimiento, proyecto de intervención, intervención, cierre y, sólo en Reunificación, seguimiento tras cese. La maqueta junta proyecto e intervención y no tiene seguimiento (apartado 11).
- Pestañas: Familia, Menores (n), Intervención, Coordinación y derivaciones, Cierre, Documentos, Historial. Qué campos van en cada una lo dice FLUJO_APP.md; los datos, ESQUEMA_BACKEND.md.
- Pestaña Familia (la única dibujada): tarjetas «Composición y situación económica» (rejilla de 3 columnas con el tramo del SMI como campo calculado), «Figuras parentales» (tabla con papel, nacimiento y edad, nivel educativo y situación laboral; un dato que falta sale como «Sin indicar» en naranja), «Factores de riesgo de la familia» (casillas en 3 columnas, cuenta «5 marcados de 23» y «ver todos»), «Responsables de la situación de desprotección» y «Tipologías de desprotección» (fichas seleccionables; las tipologías indican «Se aplican a los 3 menores · desmarcar alguno», según [D4]).
- Columna lateral: aviso «Faltan N datos para la Memoria» con enlaces a cada campo; «Menores del expediente» con subtarjetas y «Añadir»; «Próximas fechas»; «Actividad reciente».
- Escalas de evaluación (VALÓRAME, ECP-12, SCORE-15, SDQ, SOATIF, AAPI-2.5 y satisfacción) en sus momentos inicial, intermedio y final: **son opcionales**, decisión de responsable del producto del 14/10/2026 [D17]. Se anotan en Evaluación cuando se quieran registrar. Los resultados guardados rellenan sus casillas de los Anexos 11 y 12; las demás quedan vacías y editables. Varios sujetos por casilla se identifican con iniciales y sus resultados se separan por «; », sin nombres completos ni combinación de puntuaciones. No están dibujadas en la maqueta.
- FI1 permite editar hasta 15 menores, su subprograma y el centro de protección opcional. Intervención recoge la fase y las listas familiares, distingue datos desconocidos de ninguna opción y permite excluir menores por opción. Ambas pestañas llevan fecha de vigencia, avisan de cambios sin guardar y conservan el formulario ante un conflicto; los datos actuales se consultan aparte antes de elegir cuál guardar. Coordinación consulta sin controles de guardado. Los avisos de F2.02 abren la pestaña y llevan el foco al campo concreto; los campos sin pantalla siguen como información.
- Guardado: la maqueta tiene un único «Guardar cambios». Regla: si hay cambios sin guardar, se marca la pestaña y se pregunta antes de salir. Guardar solo cada campo al salir de él no está decidido: **pendiente del responsable del producto**.
- En Menores, «Centro de protección» es opcional por menor, con texto o selección de catálogo. En Coordinación y derivaciones se recogen nombre y contacto del coordinador del caso del servicio derivante, separados del profesional de referencia del ETF.
- Los cambios de composición, ingresos, menores, factores y subprograma llevan fecha de vigencia; el Historial permite distinguirlos y conserva los episodios de cierre y reapertura. La Memoria usa los valores vigentes del año consultado.
- En un expediente cerrado sólo se editan datos de seguimiento. La coordinación ve «Reabrir» con motivo para permitir otros cambios; el cierre previo sigue en el Historial.
- El equipo puede proponer el cierre desde ACEPTADA si la familia no consiente, con el motivo de catálogo correspondiente, sin inventar consentimiento ni intervención. Coordinación confirma fecha y motivo del cierre.
- Con Anexo 5, el cierre ordinario sólo se permite tras registrar «se conoce inicio de procedimiento de desamparo o adopción de medida» o «el SPM no valora necesaria la medida». «A la espera» mantiene el bloqueo. La coordinación dispone de «Forzar cierre», con motivo obligatorio y auditado. La pantalla explica el bloqueo al pulsar «Cerrar».

### 5.3 Revisión de una derivación (maqueta «Coordinación · revisar derivación»)

- Ruta «Derivaciones recibidas › Revisar derivación». Contexto: anexo, servicio y fecha de derivación.
- Aviso informativo arriba: «Hemos leído los datos del anexo. El fichero no se guarda en la aplicación. Revisa los campos marcados en naranja antes de decidir.»
- P3 conserva municipio, centro y equipo confirmados en la revisión del anexo; la aceptación no sustituye esa asignación por la propuesta automática.
- Tarjeta «Municipio, centro y equipo»: municipio en desplegable con el origen del dato debajo; centro, equipo y subprograma como campos calculados con su origen; profesional de referencia como campo **pendiente** («Obligatorio para aceptar»); número que se asignará en campo neutro con «Equipo, correlativo y año. Se fija al aceptar.» (formato pendiente de [D1]). Si el municipio no está en el catálogo o hay varios candidatos, el campo queda vacío y pendiente ([D10]).
- Tarjeta «Menores por los que se deriva (n)»: tabla con nombre completo, nacimiento y edad, escolarización y sexo. El sexo no viene en el anexo: grupo de opciones Niña/Niño marcado como pendiente en cada fila. «Añadir menor».
- Tarjeta «Familia»: personas con papel, edad, nivel educativo ya traducido a la lista de la Memoria y el texto original debajo; lo que no viene, «Sin indicar en el anexo» en naranja. Estructura, situación económica e ingresos como pares etiqueta-valor.
- Tarjeta «Descripción de la situación familiar (para el Acta)»: área de texto con lo copiado del anexo para que la coordinación lo recorte ([D13]).
- Desplegable «Resto del anexo» con los apartados que no se revisan aquí.
- Columna lateral: aviso «Falta para poder aceptar» con enlaces; tarjeta «Decisión» con los tres botones de decisión y su consecuencia; tarjeta «Servicio derivante», con nombre y contacto de su coordinador del caso, diferenciados del profesional ETF.
- Al pulsar «Aceptar» con algo pendiente, sale el resumen de errores. P3 muestra un diálogo con la consecuencia y asigna el número sólo tras la respuesta confirmada del servidor; la API no ofrece una reserva o vista previa del número. Después se descargan el Acta y el oficio con acciones separadas. «No aceptar» pide motivo, fecha y subprograma antes de confirmar. El profesional de referencia se elige por puesto, como permite el contrato actual; la selección de una persona concreta queda pendiente del puerto de consulta de profesionales.

### 5.4 Generación de documentos y confirmación de hitos

CO2 reúne la respuesta del SPM y el cierre en el detalle de Coordinación. La propuesta muestra los motivos publicados en catálogo; confirmar fecha y cierre exige una propuesta pendiente y una persona distinta de quien la presentó. Descargar el Acta de Cierre tiene su propia acción, antes o después del cierre. La excepción del SPM pide motivo auditado y la reapertura pide fecha y motivo. La bandeja enlaza a los cierres propuestos; el listado muestra sólo números, motivos y fechas, sin nombres.

- La pestaña Documentos lista lo que corresponde al estado del expediente, con «Generar», última emisión, autor y versión de plantilla. P3 ofrece una sección «Actas y oficios» en la revisión: muestra la última emisión por tipo en los idiomas disponibles y su versión de plantilla, junto con el registro de cada oficio. Un oficio emitido en lista de espera conserva su registro al aceptar la derivación; se muestra como documento anterior y no se vuelve a emitir desde ese estado. Los campos de registro se bloquean durante la confirmación; si hay cambios sin guardar, se guarda el registro antes de regenerar ese oficio. Se distinguen «Generado el…» y el hito efectivo «Firmado el…» o la fecha confirmada que corresponda al trámite.
- Generar descarga un Word (.docx) y registra una **emisión**: tipo, autor, fecha, versión inmutable de plantilla y huella de los datos utilizados. El fichero no se guarda. La emisión queda registrada y auditada antes de entregar el Word.
- El **hito** se confirma por separado: una persona anota la fecha efectiva. Descargar o regenerar no fija el consentimiento, no elimina su aviso de 30 días, no cierra el expediente ni modifica estados o fechas de negocio. La línea de tiempo usa los hitos confirmados.
- Si se vuelve a descargar con datos distintos, se avisa de que se creará una nueva emisión. Un reintento de la misma operación nunca devuelve otro contenido bajo la misma clave de petición; el contrato se detalla en TRD.md. Las emisiones anteriores conservan su referencia de plantilla y huella.
- Si falta un dato exclusivo del documento, el diálogo lo solicita. El DNI del representante legal **no se pide ni se guarda**: su hueco en el consentimiento Word queda siempre en blanco, decisión firme [D16].
- Para actas y oficios, un campo posterior «Registro de salida: número y fecha» permite anotar el registro hecho fuera; no equivale a generar ni a confirmar la firma. El conector manual y la futura conexión con MOAD se describen en TRD.md.
- Al descargar, aviso breve en región de estado: «Se ha descargado el Acta de Aceptación. Contiene datos personales: guárdala sólo donde te indique el Programa.» (texto propuesto).

### 5.5 Subida del Anexo 3 o 4

El Anexo 3 o 4 admite Word (.docx), LibreOffice (.odt) o PDF, máximo 10 MB. Se procesa en memoria y se descarta: ni el servidor intermediario, ni el antivirus ni los temporales lo escriben en disco. Las otras subidas se describen en 5.10. Estados, todos anunciados en una región de estado para lectores de pantalla:

| Estado | Qué ve la persona (texto propuesto para `idiomas/es/`) |
| --- | --- |
| Elegir | Zona con «Elegir fichero» (botón, no sólo arrastrar), formatos admitidos y tamaño máximo |
| Comprobación previa en el navegador | Si no es Word, LibreOffice o PDF o pasa de 10 MB: error al momento, sin subir nada |
| Subiendo | «Subiendo el fichero… 45 %» con barra de progreso que lleva su valor accesible |
| Revisando | «Revisando el fichero con el antivirus…» con indicador sin porcentaje. Es la espera que la entrevista pide explicar |
| Leyendo | «Leyendo los datos del anexo…» |
| Listo | Pasa a la revisión (5.3) |
| Antivirus encuentra algo | Error: el fichero no se ha usado ni guardado; pide al servicio que lo envíe de nuevo |
| Antivirus no responde | Se sigue con aviso naranja en la revisión: «No se ha podido revisar el fichero con el antivirus. Queda anotado.» |
| Fichero con macros, dañado o demasiado grande al descomprimir | Error que dice cuál de las tres cosas y qué hacer |
| Escaneado o ilegible | «No hemos podido leer los datos de este fichero. Tendrás que escribirlos.» y se abre la revisión en blanco |

P2 monta subida y revisión en `/coordinacion/anexos/nuevo`. La API de análisis no crea derivación. Mientras responde, la pantalla anuncia conjuntamente antivirus y lectura, sin inventar porcentajes ni fases del servidor. «Confirmar y crear derivación» guarda los campos revisados y abre la decisión de P3. Un dato dudoso correcto dispone de «Confirmar dato».

Botón «Cancelar» disponible mientras sube. Los otros tipos de subida comparten los estados de progreso, pero aplican su propia política (5.10); la excepción de antivirus caído se aplica al anexo y a las actas de carga histórica.

### 5.6 Memoria Técnica Anual

F4.02 monta `/memoria` en el armazón UI1. Los cuatro bloques conservan tablas con cabeceras y región de desplazamiento propia, también en móvil. Los avisos aparecen antes de los bloques y permiten ir al primer apartado afectado. Se muestran fecha de corte y fecha de cálculo. La edición de A, D, E, F y G y la descarga Word usan F4.05; la guardia conserva los cambios al navegar y la recuperación Word repite la misma emisión. Estado, fecha de cierre, cierre confirmado y reapertura con motivo usan F4.06; la composición por números sigue en F4.07.

- Filtros: año y equipo (uno o todos; el equipo sólo ve el suyo). Estado visible: abierta o cerrada, con fecha de cierre cuando exista. Se muestran el año de los datos y las reglas vigentes a 31 de diciembre de ese año.
- Los 28 apartados siguen la plantilla de la administración competente (las plantillas de `idiomas/`), cada uno con su título y tabla, cabeceras reales y porcentajes con dos decimales y coma. Cada apartado identifica la población que cuenta y el denominador de sus porcentajes; la explicación detallada va tras «?».
- Los valores desconocidos aparecen aparte como «Sin dato», con su cantidad visible; nunca se convierten en cero ni en una categoría conocida. Si el divisor es cero, el porcentaje muestra «0,00 %» y un aviso de que no hay población para calcularlo.
- La edad usa el criterio anual de la Memoria: quien cumple 18 durante el año se mantiene en 12–17 conforme a la nota 43. Las características se obtienen de su historia con vigencia, no de la ficha actual sin fecha.
- Para coordinación y equipo, cada cifra enlaza a los **números** de expediente que la forman, dentro de sus permisos. Consulta recibe una vista independiente sólo de cifras: sin enlaces, identificadores de composición, textos libres ni descarga de la Memoria con narrativas.
- La provincial se calcula directamente de los datos con las mismas reglas, sin sumar las Memorias de equipo. Si difiere de una Memoria de equipo cerrada, se muestra el apartado, el valor cerrado, el actual y su diferencia; ninguna foto cerrada se cambia automáticamente.
- Comprobaciones automáticas a la vista: si el apartado 7 no suma lo mismo que 8 y 9, aviso naranja. Si faltan datos, aviso con cantidad y enlace para coordinación y equipo; Consulta sólo ve las cantidades.
- Coordinación y equipo editan sus apartados de texto A, D, E y F, con los límites de la plantilla; G conserva su límite de 30 líneas mientras se resuelve su inclusión según PRD.md. Estos textos son datos de negocio, no traducciones, y no están disponibles para Consulta.
- Acciones de coordinación y equipo: «Descargar la Memoria» (Word) y «Cerrar la Memoria», con confirmación. El equipo cierra la suya; sólo Coordinación cierra la provincial. El cierre guarda cifras y textos; reabrir con motivo es una acción de Coordinación y queda auditada. Descargar no cierra.
- La validación usa resultados esperados por apartado, aprobados por el Programa. No se reproducen repartos incoherentes de 2025: el apartado 2 da 294 niños y 284 niñas, y el 25 da 290 y 288 [D22]. El Programa debe decidir cuál vale y explicar la diferencia.
- Las tablas y las listas de composición nunca muestran nombres. La redacción de los apartados narrativos debe evitar datos identificativos.

### 5.7 Tabla Mensual para el SPAF

- Elegir mes (por defecto el anterior) y, para coordinación, equipo.
- Vista previa en pantalla con el resumen (altas, bajas, total de casos, total de menores) y la tabla. Las 14 columnas no caben a lo ancho: la tabla se desplaza dentro de su tarjeta, nunca la página.
- En la vista previa los nombres de menores van enmascarados, porque es un listado; el Word descargado lleva los nombres completos que pide el SPAF y su descarga queda registrada. Esta separación es una interpretación de la decisión 8; responsable del producto la confirma.
- Acción principal: «Descargar la Tabla SPAF de septiembre».
- SP1 usa el armazón UI1 y muestra el equipo por su etiqueta de catálogo. Coordinación elige un equipo; el profesional consulta el suyo. Las cifras proceden de la API, con aviso de total de menores incompleto cuando faltan datos. Si todos los recuentos faltan, la cifra principal dice «Sin dato»; si el recuento es parcial, se rotula «Menores con dato».
- El desglose de altas y bajas enlaza números de expediente, sin nombres. La consulta acotada muestra estado actual, aceptación y movimientos históricos del mes, con fechas y motivos; volver conserva equipo y mes. La vista previa presenta las 14 columnas; las ausencias dicen «Sin dato» y el centro de protección ocupa la columna de guardadores cuando corresponde. La tabla tiene una región desplazable accesible por teclado.
- Antes de descargar, avisa de que el Word puede contener nombres completos y de que la emisión no confirma firma ni envío. Si los datos cambian, ofrece «Ver cambios». Una descarga sin respuesta confirmada ofrece recuperar la misma emisión conservando su petición y clave.

### 5.8 Listas de trabajo de coordinación

Derivaciones recibidas, lista de espera (por fecha de recepción, con subprograma y municipio), cierres propuestos y expedientes de la provincia siguen el patrón de la tabla de la bandeja: filtros arriba, familia enmascarada, número como enlace, paginación en la de provincia. Ninguna está dibujada.

### 5.9 Administración de temas e idiomas

**Temas** (pantalla prevista, sin maqueta):

- Lista de temas instalados en tarjetas: nombre, versión, autor, tipo, muestras de color de los tokens principales y una píldora «Activo» en el que está en uso.
- «Instalar tema»: subida del paquete con el patrón de 5.5 (incluido el antivirus). La app valida el paquete (manifiesto, que estén todos los tokens o se completen con los del tema por defecto, fuentes WOFF2 e imágenes PNG, sin SVG ni recursos remotos) y comprueba los contrastes de 3.5. Muestra el resultado de cada comprobación antes de dejar activarlo.
- «Probar en mi sesión»: aplica el tema sólo a quien lo prueba, con una barra fija «Estás probando el tema <nombre> · Volver al tema activo».
- «Activar»: diálogo de confirmación con la versión; el aviso de éxito sólo aparece cuando se confirma la activación con su auditoría. Si otra persona activó una versión mientras tanto, se avisa del conflicto y se pide revisar la versión actual.
- «Retirar»: no se puede retirar el tema activo ni el tema por defecto.

**Idiomas** (pantalla prevista, sin maqueta):

- Lista de idiomas encontrados en `idiomas/`, con código, nombre, si es el idioma por defecto y cuántas claves le faltan frente a castellano (esas salen en castellano).
- «Recargar idiomas» vuelve a leer las carpetas sin reiniciar.
- Administración puede instalar un paquete de idioma y editar sus textos de interfaz y de catálogo. Cada guardado crea una versión inmutable en `idiomas/<código>/{textos,catalogos,plantillas}/`; la base de datos guarda claves, reglas y referencias, nunca los textos de catálogo. Recargar sólo ofrece versiones validadas y activadas.

Los temas, idiomas y plantillas se guardan en archivos de versión inmutable, identificados por versión o huella. Registrar y activar su referencia junto con la auditoría es una sola transacción: si falla, se conserva la versión activa anterior; un archivo huérfano no se ofrece y se limpia. Las activaciones simultáneas detectan el conflicto. La interfaz muestra versión activa, resultado de validación y estado de activación por separado.

PL1 ofrece Plantillas Word en `/admin/plantillas`, sólo para Administración. Las tarjetas agrupan documento e idioma y muestran versión activa, autor y fecha de activación; el historial conserva subidas y activaciones. «Subir una versión» abre el formulario con formato y límite visibles. La espera anuncia antivirus y revisión de marcadores. Un rechazo indica qué marcador falta o sobra y conserva el archivo seleccionado para corregirlo. La subida válida queda pendiente de activar. «Activar versión» abre una confirmación con documento, idioma, versión y consecuencias; sólo la respuesta confirmada anuncia éxito. Un conflicto permite actualizar la lista antes de confirmar de nuevo. La autoría se resuelve desde la consulta autorizada de Usuarios; no se muestran identificadores internos. Las versiones iniciales sin constancia de antivirus o activación dicen que esos datos no están registrados.

Las otras pantallas de Administración (usuarios y perfiles, catálogos, plazos y tramos versionados con vigencia, plantillas Word, firmantes, destinatarios y auditoría) siguen el patrón de tabla y ficha; su contenido lo fija FLUJO_APP.md. La exportación de auditoría para el DPD (Delegado de Protección de Datos) corresponde a Administración funcional, desde «Auditoría», con filtros y constancia de la exportación. La conservación y el alcance de los vínculos usuario–expediente exportados se validan con el DPD en la EIPD (Evaluación de Impacto en Protección de Datos).

### 5.10 Política visible por tipo de fichero

El selector informa del formato y límite antes de subir. Todas las subidas pasan por antivirus local y validación del tipo real; un cambio de extensión no basta. El servidor aplica los límites aunque el navegador no los compruebe. Límites y controles completos en TRD.md:

| Uso | Formatos y límite | Persistencia y caída del antivirus |
| --- | --- | --- |
| Anexo 3 o 4 | DOCX o PDF, 10 MB | Sólo memoria; se descarta. Si el antivirus cae, se acepta con aviso visible y auditado, según la excepción acordada |
| Actas históricas | DOCX o PDF, 10 MB por acta | Sólo memoria; se descartan. Si el antivirus cae, aviso visible y auditado, como anexos |
| Plantilla | DOCX, 10 MB | Versión inmutable tras validar; sin antivirus no se publica |
| Tema | ZIP con JSON, PNG y WOFF2, 20 MB | Versión inmutable; sin SVG; sin antivirus no se publica |
| Idioma | ZIP con JSON y DOCX, 20 MB | Versión inmutable; sin antivirus no se publica |
| Hoja de carga | XLSX o CSV, 10 MB | El fichero no persiste; sin antivirus no se importa |

Los paquetes tienen además un límite de 100 MB descomprimidos y 1.000 entradas. Ante antivirus no disponible en plantilla, tema, idioma u hoja, se explica que hay que reintentar cuando pueda revisarse; no se ofrece «publicar igualmente» ni «importar igualmente».

### 5.11 Carga de los expedientes de 2026

La pantalla cubre todos los expedientes necesarios para la Memoria de 2026: vivos, cerrados durante 2026 y no aceptados durante 2026. Incluye todos los abiertos de cualquier año anterior. No existe hoja [D21, 14/10/2026]: los equipos suben actas o introducen los datos a mano. Conserva número y fechas históricas de los aceptados, sin consumir nuevos correlativos, y el estado real de cada caso; los no aceptados no reciben número de aceptación.

El servidor de hoja de F2.08b devuelve incidencias por fila y columna con mensajes en castellano o inglés, sin valores personales, y conserva el número original de fila. El formato provisional D21 está en ESQUEMA_BACKEND §7.2. CA1 añade `/coordinacion/carga-2026` para hoja y `/equipo/expedientes/alta-historica` para tecleo, con ambos perfiles autorizados por el servidor y el profesional limitado a su equipo. El alta agrupa derivación, aceptación, familia y menores, y episodios; su revisión muestra iniciales y fechas históricas. La carga por hoja anuncia el antivirus, conserva las filas originales enmascaradas y traduce las columnas de las incidencias a etiquetas. La confirmación muestra el resultado y la marca «Cargado para revisión». Cada resultado permite consultar los datos guardados por la lectura auditada existente y ofrece un enlace recuperable, también para no aceptados y números desconocidos. El alta manual admite nombres conocidos de menores en no aceptados, sin exigir sexo ni datos de aceptación. Si se pierde la conexión, permite recuperar el mismo envío sin preparar otro. La ayuda se abre aparte; desde la pantalla se descarga una hoja CSV inventada con el formato provisional.

La vista previa del lote muestra filas enmascaradas, estado, incidencias y campos pendientes, sin guardar parcialmente. Al confirmar se aplica todo el lote o ninguno. RESP2 añade al alta una sección inicial de equipo y lectura opcional de Acta de Aceptación y Acta de Cierre, PDF/DOCX hasta 10 MB. La lectura avisa que sustituye el formulario, anuncia antivirus y deja una propuesta editable. Los campos que no se leen quedan pendientes; el estado actual y el sexo desconocido de adultos se confirman a mano. El PDF sin texto permite seguir a mano. La fecha leída de firma se identifica como propuesta sin validar la firma. Los ficheros se descartan y sólo se guardan datos confirmados. La carga por hoja queda como compatibilidad. El alta manual ofrece los mismos estados y controles. Las fechas efectivas históricas se confirman sin generar documentos para simular hitos.

La carga con datos reales sólo se hace en producción, con la aplicación desplegada al acabar la fase 2 y tras aprobar EIPD, seguridad, copias y restauración. Desarrollo y preproducción usan exclusivamente datos inventados. No se espera a recibir datos reales de 2025. Servidor, copias y EIPD avanzan desde ahora; manuales, vídeo y formación quedan para el final (PLAN_IMPLEMENTACION.md).

GAP1 completa la pestaña Familia con formularios de datos familiares y personas adultas, guardados por separado y una fecha de vigencia. Conserva el formulario ante un conflicto y muestra los datos actuales para comparar. Los avisos de la ficha abren Familia y llevan el foco al campo, también dentro de un desplegable. Cierre ofrece el anexo de finalización del subprograma, con aviso antes de descargar y bloqueo mientras haya datos sin guardar. Coordinación conserva la consulta.


En la revisión de actas, los nombres completos leídos deben separarse a mano en nombre y apellidos antes de validar. La referencia leída permanece visible y no se envía al guardar. Si ya hay datos, leer otras actas exige confirmar la sustitución; cancelar conserva el formulario y devuelve el foco. La vista previa agrupa adultos y menores por iniciales y muestra ubicación, subprograma y motivos de cierre. La entrada de compatibilidad se llama «Cargar desde una hoja».

## 6. Responsive

Se trabaja sobre todo en ordenador de oficina, pero la página debe funcionar al 400 % de zoom (equivale a 320 px de ancho, criterio 1.4.10) sin desplazamiento horizontal de la página.

| Ancho | Qué cambia |
| --- | --- |
| 1280 px o más | Diseño de la maqueta: menú de 256 px, contenido y columna lateral de 360 px |
| 1024 a 1279 px | La columna lateral pasa debajo del contenido, con los avisos «Falta…» y la decisión **arriba** del contenido para que no se pierdan. Indicadores en 2 columnas |
| 768 a 1023 px | El menú lateral se pliega tras un botón «Menú». Rejillas de formulario en 2 columnas. Pestañas con desplazamiento propio |
| Menos de 768 px | Una columna. Las tablas de listado se convierten en lista de tarjetas (número, familia enmascarada, aviso). El buscador se abre desde un botón en la cabecera |

Lo que no debe pasar nunca: desplazamiento horizontal de la página (sólo se admite dentro de una tabla ancha como la del SPAF), texto cortado con puntos suspensivos sin forma de leerlo entero, botones por debajo de 44 px, y avisos de «Falta…» escondidos al final de la página en pantallas estrechas.

## 7. Accesibilidad

Norma: WCAG 2.1 nivel AA, exigida por el RD 1112/2018 de accesibilidad de sitios web del sector público y la norma UNE-EN 301 549. Comprobarla es condición para dar la app por terminada (entrevista 13.3).

- **Teclado.** Todo se hace con teclado, en orden lógico. Enlace «Saltar al contenido» al principio. Pestañas con flechas; diálogos que atrapan el foco y lo devuelven. Los enlaces de los avisos «Falta…» mueven el foco al campo.
- **Foco visible** en todo elemento, con `color.foco` y al menos 2 px de contorno separado del borde. La maqueta no lo define (apartado 11).
- **Contraste** según 3.5: texto 4,5:1, texto grande y límites de controles 3:1.
- **Estructura.** Regiones (`header`, `nav` «Principal» y «Ruta», `main`, `aside`), un h1 por pantalla y h2 por tarjeta, `lang` de la página según el idioma elegido, título de la pestaña del navegador con la pantalla y el número de expediente (nunca el nombre de la familia).
- **Formularios.** Etiqueta visible y asociada en cada campo (la maqueta lo hace, incluso con etiquetas ocultas visualmente como «Sexo de Lucía»); `fieldset` y `legend` en grupos; errores en texto, enlazados con `aria-describedby` e identificados por algo más que el color.
- **Estado y cambios.** Avisos informativos con `role="status"`; subida y generación anunciadas en una región viva; `aria-current` en menú y línea de tiempo; `aria-pressed` en fichas seleccionables.
- **Zoom y espaciado.** Usable al 200 % sin perder contenido y al 400 % con reflujo (apartado 6); soporta que la persona aumente interlineado y espaciado (criterio 1.4.12).
- **Movimiento.** Sin animaciones en el tema por defecto; se respeta la preferencia de reducir movimiento.
- **Tiempo.** Si la sesión va a caducar, se avisa con tiempo para prolongarla sin perder lo escrito (detalles de sesión en TRD.md).
- **Comprobación.** Revisión automática en cada cambio de pantalla y revisión manual con teclado y lector de pantalla (NVDA) antes de cada entrega; ver PLAN_IMPLEMENTACION.md.

F5.03 incorpora la matriz automática y los informes por pantalla en [Accesibilidad](../accesibilidad/README.md). El armazón devuelve el foco tras cambiar el idioma y permite cerrar Preferencias y Persona con dos pulsaciones de Escape. La revisión de pestañas de ficha, lector de pantalla y Word conserva los pendientes detallados en el informe; la matriz axe no acredita conformidad completa.

**Documentos Word accesibles.** Las plantillas que llegaron del Programa se convierten una vez a .docx con marcadores. En esa conversión, sin cambiar sus textos, se les da:

- estilos de título reales (Título 1, Título 2) en lugar de texto en negrita;
- tablas con la fila de encabezado marcada y repetida en cada página, y sin celdas vacías usadas para maquetar donde se pueda evitar;
- texto alternativo en logotipos e imágenes;
- idioma del documento según la carpeta de `idiomas/` de la que sale la plantilla;
- propiedad de título del documento rellenada (por ejemplo «Acta de Aceptación 02-32-26»);
- orden de lectura correcto y sin texto en cuadros flotantes.

Se comprueba con el comprobador de accesibilidad de un procesador de textos local, nunca en un servicio externo. Las plantillas de la administración competente (Memoria, Tabla SPAF) pueden traer problemas de origen que no se pueden tocar sin cambiar el modelo oficial: se anotan y el responsable del producto decide si se trasladan al Programa.

## 8. Textos e idiomas en pantalla

### 8.1 Dónde viven los textos

Ningún texto visible está en el código: ni en Go, ni en React, ni compilado en el JavaScript. El código sólo usa claves (`expediente.aceptar`, `derivacion.falta.titulo`). Los textos están en `idiomas/<código>/`, fuera del código, junto con los nombres de las listas y catálogos, los mensajes y errores del servidor, los textos de avisos, los textos de ayuda del botón «?» y las plantillas Word de ese idioma. Cambiar o añadir un idioma es cambiar o añadir una carpeta. Si falta una clave en un idioma, sale en castellano. La CI (integración continua: las pruebas automáticas de cada cambio) busca textos escritos a mano en el código y falla, y compara las claves entre idiomas.

La estructura es `idiomas/<código>/textos/` para interfaz, mensajes y ayudas, `catalogos/` para nombres y opciones, y `plantillas/` para Word. Administración escribe versiones inmutables en estas carpetas; TRD.md fija su validación y activación. La base de datos no almacena textos traducibles de catálogo. Los nombres y textos libres del expediente y las narrativas de Memoria son datos de negocio: no se trasladan a los paquetes de idioma.

Los nombres de las listas del Programa (subprogramas, motivos de cierre, niveles educativos, tipologías…) salen del catálogo de cada idioma, no de este documento ni del código. Por eso la maqueta dice «Riesgo social» y «Reunificación» y los anexos dicen «Subprograma de Riesgo» y «Subprograma de Reintegración»: qué nombre corto se muestra lo fija el catálogo (apartado 11).

### 8.2 Tono

- De tú, cercano y directo: «Revisa los campos marcados en naranja antes de decidir», «Elige una persona del ETF-2».
- Frases cortas, en positivo, con el verbo delante. Botones con verbo y objeto.
- Primera persona del plural para lo que hace la app: «Hemos leído los datos del anexo».
- Sin culpas en los errores, sin exclamaciones, sin jerga informática («subir», «descargar», «guardar»; no «procesar», «sincronizar» ni «error 500»).
- Siglas del Programa tal como las usa el Programa: ETF, SPM, SPAF, SS.SS.CC. (Servicios Sociales Comunitarios), DSR (Declaración de la Situación de Riesgo), PIF (Plan de Intervención Familiar). Su forma larga va en la ayuda del «?» la primera vez que aparece cada una en una pantalla.

### 8.3 Términos

| Usar | No usar | Por qué |
| --- | --- | --- |
| Consentimiento (Anexo 1 o 2) | Contrato | Corrección de la maqueta del 13/10; el Programa lo confirma en [D3] |
| Pendiente de atender | Atrasado, retrasado | Término de la Memoria (apartado 6) |
| Acta de Aceptación, Acta de No Aceptación, Acta de Cierre | Resolución, informe de aceptación | Nombres del Programa |
| Anexo 3, Anexo 4, Anexo 5… | Formulario, documento de derivación | Nombres del Programa; los nombres antiguos (2L-A, 2J…) se quedan sólo dentro de las plantillas mientras no se resuelva [D18] |
| Subprograma | Programa, tipo | Término del Programa |
| Profesional de referencia | Responsable, gestor | Término del Programa |
| Lista de espera | Cola, pendientes | Término del Programa |
| Fase (recepción, evaluación, intervención, cierre y devolución, seguimiento) | Etapa, paso | Términos del Programa |
| Expediente | Caso, ficha | «Caso» sólo en textos del Programa que ya lo usan |

### 8.4 Formatos

- Fechas: dd/mm/aaaa (14/11/2026). Horas: 24 h (12:40). Relativas sólo para la actividad reciente («Ayer, 12:40»).
- Números: punto de miles y coma decimal (9.600 €, 37,50 %). Espacio antes de «%» y de «€».
- Intervalos con raya corta: «50–100 % del SMI».
- Número de expediente: equipo, correlativo y año con guiones (02-14-26) mientras no se resuelva [D1].
- Plurales por la regla de cada idioma, con mensajes que llevan la cantidad («1 menor», «3 menores»; «Falta 1 dato», «Faltan 2 datos»). Nunca «menor(es)».
- Todos los formatos salen de la configuración regional del idioma, no del código.

## 9. Privacidad visible

La app trata datos de menores y categorías especiales (violencia de género, salud mental, adicciones), con nivel alto de confidencialidad del ENS (Esquema Nacional de Seguridad). En pantalla se aplica así:

| Dónde | Qué se ve | Qué queda registrado |
| --- | --- | --- |
| Listados (bandeja, provincia, derivaciones, lista de espera, cierres, resultados del buscador, vista previa SPAF) | Número de expediente y familia enmascarada con iniciales («Familia O. S.»); menores por iniciales | La consulta del listado, en la auditoría |
| Ficha del expediente y revisión de una derivación | Nombres completos, contacto y todos los datos | Cada apertura, con persona, fecha y hora (auditoría encadenada, también de lecturas; ver ESQUEMA_BACKEND.md). La pestaña Historial lo muestra |
| Documentos Word | Nombres completos | Cada generación, con persona, fecha y hora |
| Memoria y estadísticas | Cifras y, para equipo y coordinación, narrativas y lista de **números** de una cifra dentro de su ámbito | La consulta |
| Perfil Consulta | Sólo cifras; sin nombres, composición, identificadores de expedientes ni textos libres, tampoco en la respuesta del servidor | La consulta |
| Administración funcional | Ninguna ficha de expediente; exportación controlada de auditoría para el DPD | La consulta y exportación de auditoría |
| Título de la pestaña del navegador, direcciones de la página, mensajes de error, registros técnicos | Nunca nombres; como mucho el número de expediente | — |

F4.07 entrega a Consulta un contrato numérico propio y un Word de cifras. Ambos indican fecha y si las cifras son definitivas (Memoria cerrada) o provisionales (cálculo al momento). La vista de Consulta en `/memoria` muestra esos datos con los filtros y bloques de F4.02. No ofrece composición, narrativas ni acciones de cierre. Los perfiles mixtos Equipo+Consulta editan sólo su equipo y usan la vista numérica para otros equipos o la provincia. Las tablas se desplazan dentro de su región en móvil; los errores conservan la operación para reintentar la lectura o descarga. La navegación y el desglose por números del servidor sólo admiten al profesional de su equipo o a Coordinación.

Antes de mostrar datos o entregar una descarga, su auditoría debe estar confirmada. Si falla, no se entregan los datos: se muestra un error con referencia de incidencia y opción de reintentar. Los fallos se registran después de revertir la operación; nunca se anuncia como completada una acción rechazada.

La maqueta enseña a los menores con iniciales también dentro de la ficha («Daniel O. S.»). La decisión 8 dice nombre completo en la ficha; mientras responsable del producto no diga otra cosa, la ficha muestra el nombre completo (apartado 11).

Los datos que se ven en desarrollo, preproducción, maquetas y capturas son inventados pero verosímiles, y nunca llevan la palabra «ejemplo» ni marcas parecidas («[Nombre del coordinador]», «[Texto de ejemplo…]»).

Tras 25 años desde el cierre, el expediente se anonimiza ([D14]): en pantalla sólo quedan número y datos estadísticos, sin nombres ni contacto, con el estado «Anonimizado». En `/admin/conservacion`, Administración sólo ve número, cierre, fin del plazo y estado. La primera revisión y la confirmación de otra persona requieren revisar la consecuencia y marcar una casilla en un diálogo. Si falla la respuesta, la pantalla pide actualizar antes de volver a actuar.

## 10. Lista de comprobación antes de dar por buena una pantalla

- [ ] Ningún texto visible en el código: todo sale de claves de `idiomas/<código>/` y la prueba de la CI pasa.
- [ ] Ningún color, fuente, medida, radio ni sombra fijos: sólo tokens del apartado 3.3.
- [ ] Se ve bien con el tema por defecto y con el de alto contraste.
- [ ] Se ve bien en castellano y con un idioma de textos más largos (por ejemplo inglés), sin cortes.
- [ ] Una sola acción principal; los botones dicen verbo y objeto.
- [ ] Lo que falta está en naranja con texto y con enlace al campo; nada se indica sólo por color.
- [ ] No hay texto de ayuda fijo; la ayuda está tras «?». Lo que queda fijo es estado, origen del dato o consecuencia.
- [ ] Términos del Programa sin cambiar (tabla 8.3) y tono de tú.
- [ ] Listados enmascarados; nombres completos sólo en ficha, revisión y documentos; nada personal en el título de la pestaña ni en la dirección.
- [ ] Datos de prueba verosímiles y sin «ejemplo».
- [ ] Todo se hace con teclado; foco visible; orden lógico; diálogos que devuelven el foco.
- [ ] Contraste AA comprobado (texto 4,5:1; controles y texto grande 3:1).
- [ ] Controles de 44 px de alto; texto mínimo de 13 px.
- [ ] Etiquetas asociadas, grupos con `fieldset` y `legend`, errores enlazados.
- [ ] Estados de carga, vacío, error y sin permiso diseñados y con texto.
- [ ] Esperas largas (subida, antivirus, Memoria) explicadas en una región de estado.
- [ ] Funciona a 1024 px y al 400 % de zoom sin desplazamiento horizontal de la página.
- [ ] Revisión con lector de pantalla hecha en las pantallas nuevas.
- [ ] Revisión de usabilidad independiente antes de fusionar (ver PLAN_IMPLEMENTACION.md).
- [ ] Los Word que genera la pantalla pasan el comprobador de accesibilidad.
- [ ] Emisión y fecha efectiva se distinguen; regenerar no altera hitos ni estados.
- [ ] Consulta recibe sólo cifras, sin composición ni textos libres.
- [ ] «Sin dato», población, denominador y aviso de divisor cero son visibles en cada apartado aplicable.
- [ ] No se entregan datos cuando falla la auditoría.
- [ ] Cada subida aplica la política de su tipo; los temas rechazan SVG.
- [ ] Los cambios de paquetes crean versiones inmutables y no anuncian activación antes de confirmarla.

## 11. Contradicciones encontradas

### Resueltas por la entrevista del 13/10/2026

| Contradicción | Cómo queda |
| --- | --- |
| El análisis decía que perfil y equipo salen de grupos de AD | Los permisos se dan en la app; AD sólo valida la contraseña. Hace falta la pantalla «pendiente de alta» y la gestión de usuarios en Administración |
| El análisis limitaba el fichero subido a 5 MB | 10 MB, Word o PDF, con antivirus local y estados «subiendo» y «revisando con el antivirus» (5.5) |
| El análisis dejaba en blanco las escalas de evaluación de los Anexos 11 y 12 ([D17]) | Se guardan en tres momentos. Falta su pantalla |
| El análisis no preveía temas ni idiomas como paquetes | Temas en `temas/<nombre>/` y idiomas en `idiomas/<código>/`, fuera del código, gestionados desde Administración |
| Administración funcional sólo mantenía catálogos | Además usuarios, plantillas, firmantes, temas e idiomas, y no ve expedientes |
| El análisis contemplaba avisos sin decir el canal | Avisos sólo dentro de la app |

### Resueltas en el consenso del 13/10/2026

| Contradicción | Cómo queda |
| --- | --- |
| Generar Word fijaba fechas y estados | Emisión e hito separados; fecha efectiva confirmada por una persona; regenerar no altera el expediente (5.4) |
| DNI del representante pendiente [D16] | Decisión firme: no se pide ni se guarda; hueco del Word en blanco |
| Memoria de equipo cerrada sólo por Coordinación | El equipo cierra la suya; sólo Coordinación cierra la provincial (5.6) |
| Provincial como suma de Memorias cerradas | Cálculo directo de datos; diferencias visibles frente a fotos de equipo (5.6) |
| Consulta podía recibir narrativas o composición | Vista exclusivamente de cifras, también en los datos recibidos del servidor (5.6 y 9) |
| Carga sólo de vivos y al final del proyecto | Vivos, cerrados y no aceptados de 2026, sólo en producción al acabar la fase 2 (5.11) |
| Temas con SVG y archivos sobrescritos | JSON, PNG y WOFF2; versiones inmutables y activación auditada (3.1, 5.9 y 5.10) |
| Subida de idioma y exportador de auditoría pendientes | Administración funcional gestiona los idiomas y exporta auditoría (5.9) |
| Anexo era el único fichero subido; caída del antivirus permitía todos | Política por tipo; excepción de antivirus para anexos y actas históricas (5.10) |
| Cerrar desde ACEPTADA y editar cerrados no estaban fijados | Cierre por falta de consentimiento permitido; tras cerrar sólo seguimiento editable, resto exige reapertura (5.2) |
| Cualquier respuesta al Anexo 5 desbloqueaba | Sólo las dos respuestas definitivas acordadas; «a la espera» bloquea; Coordinación puede forzar con motivo (5.2) |
| Textos de catálogo en base de datos | Todos en `idiomas/<código>/catalogos/`; los textos libres de negocio sí se conservan en sus datos (8.1) |
| Estadística sin historia y sin distinguir desconocidos | Historia con vigencia, reglas a 31/12, nota 43 y «Sin dato» separado (5.2 y 5.6) |
| Memoria 2025 como resultado íntegro de referencia | Validación por apartado del Programa; discrepancia de sexos 2 frente a 25 abierta [D22] (5.6) |

### Cambios pendientes en la maqueta

Choques con decisiones ya tomadas (se corrigen antes de programar la pantalla):

1. Quitar el subtítulo «datos de ejemplo, inventados» de la bandeja y de la revisión de derivación, y el texto «[Texto de ejemplo copiado del anexo]»: chocan con la decisión 8. Sustituir por datos inventados verosímiles.
2. Sustituir los marcadores «[Nombre del coordinador]», «[Profesional de referencia del centro] · [teléfono] · [correo]» por datos inventados verosímiles.
3. Enmascarar la columna Familia de la bandeja («Ortega Salas» → «Familia O. S.»).
4. Decidir los nombres de menores en la ficha: la maqueta usa iniciales y la decisión 8 pide completo. Provisional: completo. **Decidel responsable del producto.**
5. El desplegable de profesional de referencia ofrece puestos («Trabajador/a social») en vez de personas del equipo: debe listar personas con su puesto.
6. Unificar en 44 px los controles de 40 px (filtros de la bandeja, «Añadir persona adulta», «Añadir menor», fichas seleccionables).
7. Subir a 13 px las píldoras de 12,5 px de las subtarjetas de menores.
8. Oscurecer `color.borde-control` para que cumpla 3:1 (propuesta #8a8577). Cambia el aspecto: **decidel responsable del producto**.
9. Añadir el estilo de foco visible y el estado de error con `color.peligro-fondo`.
10. Añadir el botón «?» donde haya ayuda y confirmar la frontera entre ayuda y texto informativo fijo (principio 2). **Decidel responsable del producto.**
11. Línea de tiempo: separar «Proyecto» de «Intervención», añadir «Seguimiento tras cese» en Reunificación y decir el estado de cada paso en texto oculto.
12. Cargar las fuentes desde el paquete del tema, no desde Google Fonts.
13. Unificar el nombre del equipo («ETF-2 Vega Temple / Santa Fe» frente a «ETF-2 Vega Temple - Santa Fe»): sale del catálogo.
14. Unificar los nombres cortos de subprograma («Riesgo social», «Reunificación») con los del catálogo; los anexos usan «Riesgo» y «Reintegración» y la Memoria «Riesgo Social» y «Reunificación Familiar». **Decidel responsable del producto** con el Programa.
15. Unificar el ancho de la columna lateral (360 y 380 px) y los fondos casi iguales (#d9e8e6 / #e3efed; #4a2a0c / #6b3a10).
16. Añadir una entrada para empezar una derivación nueva en el menú de coordinación (la maqueta sólo tiene «Derivaciones recibidas»).

Pantallas que faltan (ver FLUJO_APP.md para su contenido y PLAN_IMPLEMENTACION.md para el orden):

- Entrada con usuario y contraseña, «pendiente de alta», sin permiso, sesión caducada y errores generales.
- Inicio de coordinación; nueva derivación con la subida (5.5); lista de espera; cierres propuestos con el Acta de Cierre; expedientes de la provincia; diálogos de no aceptar, cerrar, reabrir y traslado entre equipos.
- Ficha: pestañas Menores, Intervención (con escalas de evaluación), Coordinación y derivaciones (Anexo 5 y respuesta del SPM, Anexo 6), Cierre, Documentos e Historial.
- Tabla SPAF; comisiones e informes; Memoria (equipo y provincial, cerrar y reabrir); SMI; vista del perfil Consulta.
- Administración: usuarios y perfiles, catálogos, plazos y tramos versionados, plantillas Word, firmantes y destinatarios, temas, idiomas y exportación de auditoría.
- Carga de todos los expedientes de 2026: vivos, cerrados y no aceptados (5.11), al acabar la fase 2 en producción.
- Versiones de las tres pantallas existentes a 1024 px y en móvil, y con el tema de alto contraste.

### Huecos sin resolver

- Si cada persona puede elegir el tema de alto contraste para sí misma: decidel responsable del producto.
- Guardado explícito o automático en la ficha: decidel responsable del producto.
- Vista previa de la Tabla SPAF enmascarada y Word completo (5.7): confirma responsable del producto.
- Conservación y alcance de la exportación de los vínculos usuario–expediente en auditoría: validar con el DPD en la EIPD; Administración funcional es quien exporta.
- Formato del número de expediente: [D1], lo decide el Programa a través del responsable del producto.

## 12. Cómo mantener este documento

- Se actualiza en el mismo cambio en que varía un patrón visual, un componente, un token o una regla de pantalla.
- Un token nuevo se añade aquí antes que en el código y se da valor en el tema por defecto y en el de alto contraste en el mismo cambio.
- Cambiar los valores del tema por defecto es cambiar el paquete `temas/expedientes-etf/`, no el código; si cambia el aspecto, lo aprueba responsable del producto (entrevista 12.5) y se anota aquí en 3.2 y 3.3.
- Los textos de pantalla no se copian aquí como cadenas definitivas: los ejemplos de este documento son propuestas; la versión vigente es la del catálogo `idiomas/es/`.
- Cuando se corrija un punto de «Cambios pendientes en la maqueta» o se dibuje una pantalla que falta, se tacha de la lista.
- Cuando el Programa responda una duda [Dn] que afecte a pantallas, se actualiza el apartado que la cita.
- Cambio pequeño basta: no se reescribe el documento por una línea.
