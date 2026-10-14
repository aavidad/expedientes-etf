# Capturas y recorrido del armazón UI1

Las 40 capturas `ui1-*.png` muestran entrada, pendiente, Usuarios, Catálogos,
Plazos y tramos, Firmantes, Paquetes (con sus entradas de idiomas y temas) y SMI.
Cada pantalla se captura a 1280 y 390 píxeles con `expedientes-etf` y
`alto-contraste`, mediante Playwright y `/usr/bin/google-chrome`.

La marca muestra «ETF». Usuarios presenta nombres completos del personal,
estados en píldora, perfiles, equipo y acción «Editar». En móvil la cabecera
ocupa una fila y el menú lateral empieza recogido. «Persona» abre identificación,
preferencias y cierre de sesión. Las familias y menores conservan el enmascarado.
El nombre de sesión procede del servidor. La sesión aporta el identificador de
equipo y la cabecera indica «Equipo asignado». Resolver su nombre requiere un
puerto de catálogos; esta entrega no amplía permisos ni cruza tablas de módulos.

El recorrido visual sirve la aplicación compilada desde un servidor local e
intercepta la API con datos inventados. Las pantallas del contrato simulado se
extraen de `tablaRutas()`. Prueba entrar → Usuarios → Catálogos → Paquetes,
preferencias, permisos del menú, salida, Atrás y Adelante, recarga, inglés,
Escape y devolución del foco, un único `main` y `h1`, y ausencia de desbordes
entre 256 y 1440 píxeles. Comprueba también que la sesión no se almacena en el
navegador y que los perfiles sin pantallas no consultan API de Administración.
Este recorrido visual no demuestra AD, PostgreSQL ni RLS.

```sh
npm --prefix web run build
ETF_EVIDENCIA_RUTA="$PWD/docs/capturas" node web/tests/armazon.navegador.mjs
```

El segundo recorrido usa PostgreSQL 18 y LDAPS desechables
con Chrome del sistema. Comprueba nombre del personal, alta y retirada de
perfiles, adscripción, traslado de equipo, denegaciones, cambios sin guardar,
historial y entrada → Usuarios → Catálogos → Paquetes. Las trazas y capturas de
ese ensayo se guardan fuera del repositorio.

```sh
ETF_NAVEGADOR_INTEGRACION=1 ETF_EVIDENCIA_RUTA=/tmp/etf-ui1-integracion \
  ./internal/adaptadores/postgres/testdata/ensayar-identidad.sh
make ci
```

SMI se incorpora desde la pantalla de Coordinación declarada en main. Una API
sin pantalla no crea una entrada de menú. La revisión con NVDA queda para el director.
