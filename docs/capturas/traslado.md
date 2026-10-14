# Traslado y seguimiento tras cese (TR1)

El módulo `web/src/traslado/` usa el armazón UI1, los textos publicados y los
controles de React Aria. Salidas y cierre de Coordinación enlaza con el traslado;
la ficha del equipo registra Seguimiento como pestaña independiente.

## Recorrido

Arranca la demo con `ETF_DEMO_PUERTO=18431 make demo` y ejecuta:

```sh
node scripts/recorridos/traslado.mjs http://127.0.0.1:18431
```

Si has cambiado los textos desde el primer arranque de la demo, ejecuta
`make demo-limpiar` antes de volver a iniciarla: sus paquetes de idioma se
conservan por versiones inmutables en volúmenes propios. Esta limpieza elimina
únicamente los datos inventados de esa demo.

El recorrido usa Playwright, `/usr/bin/google-chrome` y datos inventados leídos
del Anexo 3 y del Anexo 4 de las pruebas. Las cuentas y contraseñas se leen de la
instalación local; no se incorporan al guion ni a las capturas.

- Marta Cabrera propone el cierre por traslado. Javier Medina consulta la
  pantalla de Coordinación; Inés Moreno elige destino, motivo y fecha.
- El servidor aplica el traslado y el navegador pierde la respuesta. La acción
  «Recuperar el resultado del cambio» conserva el cuerpo y la clave enviados.
  Se comprueba el cierre, la única derivación enlazada y su estado sin número.
- Inés acepta en destino por la pantalla existente, comprueba su equipo y número
  nuevo y descarga el Acta de Aceptación. También descarga el Acta de Cierre del
  origen sin alterar su versión. Las descargas se confirman en el navegador.
- Marta Cabrera inicia el seguimiento de un expediente cerrado de Reunificación,
  consulta el plazo y lo termina antes con fecha y motivo de catálogo. Su intento
  de reapertura recibe 403; la acción tampoco aparece en su pantalla.
- Inés reabre con motivo. El plazo completo vuelve a contar desde hoy; se
  conservan la fecha de cese y las versiones anteriores.
- Se provoca un conflicto real: Javier termina mientras Inés conserva la versión
  anterior. El servidor devuelve 409 y la pantalla exige revisar los cambios,
  pasando por la guardia de cambios pendientes. Una caída simulada de la lectura
  de avisos tampoco bloquea el seguimiento.
- Se prueban 1440, 1024, 390 y 360 px, el resumen de campos pendientes, el foco de
  sus enlaces, la ayuda y Escape, el tema de alto contraste y el inglés.

## Capturas

- [Traslado en escritorio, 1440 px](traslado-escritorio.png).
- [Traslado en móvil, 390 px](traslado-movil.png).
- [Seguimiento en escritorio, 1440 px](traslado-seguimiento-escritorio.png).
- [Seguimiento en móvil, 390 px](traslado-seguimiento-movil.png).
- [Seguimiento con alto contraste en móvil](traslado-alto-contraste-movil.png).

## Alcance y límites

Los motivos seleccionables se leen de la versión vigente del catálogo. La ruta
`GET /api/seguimiento/opciones?idioma=…` autoriza a Coordinación y al equipo
asignado y confirma la auditoría antes de entregar los motivos y sus etiquetas.
Los motivos retirados mantienen su etiqueta para las lecturas históricas.
No se añaden migraciones ni permisos SQL.

La fecha de cierre se comprueba también en el servidor frente al inicio del
episodio vigente; un rechazo lleva al campo para corregirla. El seguimiento tiene
su propia versión. Una respuesta incierta mantiene la petición para recuperar
su resultado; un conflicto exige revisar los datos antes de volver a guardar.
La consulta de avisos falla por separado y no bloquea el seguimiento.

La cronología de intervención de la ficha es anterior a TR1. Se conserva en sus
otras pestañas; en Seguimiento se muestran sus propias fechas, para evitar
confundir la fase de intervención con el plazo del seguimiento. Esta entrega
no cambia el cálculo de fases ni los hitos de las otras pestañas.

El guion no acredita NVDA, una EIPD ni una puesta en producción. El vencimiento
automático y la persistencia histórica pertenecen al servidor de F3.02c; sus
ensayos existentes se ejecutan mediante la CI completa. Las Actas se firman y
registran fuera de la aplicación.

La revisión independiente aplica `programar-interfaz-etf`,
`revisar-cambios-etf` y `security-audit` en modo focal. Los ajustes revisados
incluyen nombres accesibles de ayuda, motivos históricos, errores por campo,
salidas de permiso, navegación con cambios pendientes y acceso móvil al panel.

## Resultado de la entrega

Recorrido final con Chrome: cinco comprobaciones TR1 terminan en `OK`, incluida
la aserción de ambos bordes de la pestaña activa antes de cada captura. Seguimiento
se captura desde el viewport real: la captura de página completa alteraba su
desplazamiento horizontal durante la imagen. El tema de alto contraste y el
idioma inglés pasan sin desbordamiento a 360 px.

`make ci` completa todos sus pasos y termina con `CI mínima completada.`;
incluye `go vet ./...`, `go test ./...`, las pruebas del frontend, ensayos SQL
existentes y la matriz independiente de permisos ampliada con las rutas nuevas.
Semgrep se ejecuta con reglas locales, sin métricas ni envío de código.

La segunda revisión independiente de fuente y capturas da GO de usabilidad e
integración, sin hallazgos pendientes. No ejecuta un segundo navegador: el
recorrido y la CI son evidencia del implementador. La demo se detiene al acabar.
