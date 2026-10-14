# Comisiones e informes técnicos: comprobación de A1

La pantalla de F2.04a y F2.04b está en `/equipo/comisiones`. Profesional de ETF
registra, corrige y retira registros de su equipo. Coordinación consulta todos
los equipos. Consulta no recibe registros ni textos libres; su acceso se limita
a cifras de Memoria, conforme al contrato del servidor y al diseño de privacidad.

## Recorrido en Chrome

Ejecutado con Playwright y `/usr/bin/google-chrome`, contra los servicios locales
de demostración y con las cuentas inventadas Marta Cabrera e Inés Moreno:

```sh
node web/tests/actividad.navegador.mjs
```

Resultado: `ACTIVIDAD-RECORRIDO-OK`.

Se comprobaron registro, corrección y retirada de ambos tipos; Otros con nombre;
conflicto real de versión y consulta fallida posterior sin perder el borrador;
filtros de fecha, equipo y organismo; lectura provincial y rechazo de escritura
de Coordinación, de otro equipo y de Consulta. Una respuesta de guardado inválida
no anuncia éxito ni permite repetir la escritura sin consultar.

La comprobación incluye teclado, retorno del foco al cerrar ayuda, estados vacío
y error con reintento, inglés, alto contraste y anchos de 1440, 1024, 390, 360 y
320 píxeles sin desplazamiento horizontal. Los errores de lectura y la respuesta
inválida se provocaron mediante respuestas controladas en el navegador; el
conflicto de versión y el resto del recorrido usan el servidor real.

Las capturas contienen datos inventados. El recorrido retira al terminar sólo
los registros que ha creado:

- [Comisiones en escritorio](actividad-comisiones-escritorio.png).
- [Informes en escritorio](actividad-informes-escritorio.png).
- [Informes en móvil](actividad-informes-movil.png).

## Revisión y comprobaciones

Segunda revisión independiente con `programar-interfaz-etf`, `revisar-cambios-etf`,
`security-audit` en modo focal y `humanizer`: GO sobre código y capturas de A1.
Los dos hallazgos sobre la conservación del borrador y el rótulo móvil están
corregidos. La revisión no cubre las capacidades ajenas incorporadas desde main.

La CI completa ejecuta `go vet ./...`, `go test ./...`, las 13 pruebas del frontend,
textos, temas y ensayos de PostgreSQL 18. Resultado: `CI mínima completada.`
Semgrep, con cuatro reglas locales centradas en este cambio, revisó diez ficheros:
`0 findings`. No se han añadido SQL ni migraciones.

## Límites del entorno

No se ha probado con NVDA; el entorno es Linux. Este módulo no sube anexos ni
genera documentos Word.

`make demo` completó el arranque antes de incorporar el último main. Después de
integrarlo, los servicios y las cuentas funcionan, pero la siembra de bandeja
espera 201 en `POST /api/coordinacion/derivaciones`, que ahora devuelve 200 para
la lectura previa a la confirmación. No se modificó ese módulo. El recorrido de
A1 pasó contra esos servicios. La subred alternativa se usó sólo durante las
pruebas y se restauró; `make demo-parar` dejó la demostración detenida.
