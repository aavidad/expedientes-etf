# Retoques de presentación UI2

Catálogos muestra las etiquetas publicadas del idioma activo, con reserva en
castellano. Los códigos quedan dentro de «Identificadores de la entrada» al
editar. La ayuda se abre con «?». Firmantes muestra los nombres del personal.
El menú reúne idiomas y temas en una sola entrada y mantiene las direcciones
anteriores. SMI presenta los importes como tarjetas en la disposición compacta,
con una etiqueta visible en cada campo. Los metadatos separan función, versión
y fecha de vigencia.

El recorrido de UI2 usa la demostración local desechable, Playwright y el Chrome
del sistema. Entra con Inés Moreno Castillo para SMI y con Elena Salas Márquez
para Administración; todas las cuentas y datos son inventados. Las contraseñas
se leen del fichero local generado por la demo y no se guardan en las capturas.

```sh
ETF_DEMO_PUERTO=18089 make demo
ETF_UI2_URL=http://127.0.0.1:18089 node web/tests/ui2.demo.mjs
node web/tests/armazon.navegador.mjs
make ci
make demo-parar
```

El script genera capturas `ui2-*.png` a 1440 y 390 px de Usuarios, Catálogos,
Plazos, Tramos, Firmantes y SMI, con los dos temas incluidos. Comprueba etiquetas,
acceso único de paquetes, teclado, controles de 44 px, inglés y ausencia de
desbordes a 1440, 1024, 390, 360 y 320 px. Las capturas muestran la zona visible;
no pretenden recoger todas las filas del catálogo.

El recorrido real terminó con `UI2-DEMO-OK`: seis pantallas y 24 capturas, en
los anchos, idiomas y temas indicados. SMI se probó también con un importe
inventado de 1.325,00 € al mes; guardar confirmó la versión siguiente y
18.550,00 € anuales. El ensayo independiente de sesión terminó con `DEMO-OK`:
siete usuarios, ocho equipos, permisos, privacidad y rechazo de CSRF.

El recorrido simulado terminó con `ARMAZON-CHROME-OK`. Además de los menús y
el teclado, comprueba etiquetas sin traducción, reserva en castellano, vacío,
error y denegación. Las diez pruebas de frontend y la comprobación de textos
pasaron. La revisión independiente pidió una etiqueta visible por tarjeta del
SMI; las capturas finales recogen esa corrección. La segunda revisión terminó
con GO de fuentes y capturas.

La demo se ejecutó desde `/dev/shm/etf/wt-UI2`, con temporales en
`bin/temporales`, puerto 18089 y subredes locales `10.250.89.0/28` y
`10.250.89.16/28` para convivir con otra demo. El ajuste de subred fue temporal;
se restaura la configuración compartida tras parar los servicios.

El `make ci` completo terminó con `CI mínima completada.` Incluyó las diez
pruebas de frontend, TypeScript, textos, temas, `go vet ./...`, `go test ./...`,
compilación y ensayos PostgreSQL de las capacidades integradas en main.
Semgrep local sobre el armazón y Catálogos terminó con cero hallazgos; no se
enviaron fuentes a servicios de análisis. La demo terminó con `DEMO-PARADA`.

La revisión con NVDA requiere el entorno del director. En el Chrome del ensayo,
el campo editable nativo del SMI muestra `1325.00`, mientras el importe anual
usó el formato castellano `18.550,00 €`. El formato del campo sigue pendiente
en el módulo SMI; no se cambia su lógica de edición en UI2. UI2 no modifica reglas de
servidor, permisos, migraciones, emisiones Word ni lectura de anexos.
