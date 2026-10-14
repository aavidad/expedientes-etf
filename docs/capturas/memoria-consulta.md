# Memoria: vista de Consulta (F4.07)

La vista de `/memoria` conserva los filtros y bloques de tablas de F4.02. Utiliza exclusivamente el contrato numérico de Consulta: año, equipo o provincia, fecha de las cifras y carácter provisional o definitivo. No carga narrativas, composición ni instantáneas completas. La descarga usa el Word numérico del mismo ámbito mostrado.

- [Escritorio](memoria-consulta-escritorio.png): provincia y cifras provisionales.
- [Móvil](memoria-consulta-movil.png): equipo, tablas con desplazamiento propio.
- [Alto contraste](memoria-consulta-contraste.png): mismo ámbito y contenido numérico.

Capturas con datos inventados de la demostración local y Chrome del sistema. El recorrido `web/tests/consulta-memoria.navegador.mjs` también comprueba teclado, anchuras de 1024, 390 y 320 píxeles, descarga en español e inglés, orden de apartados, fecha con año de cuatro cifras, errores de lectura y Word, cierre y reapertura reales y un perfil mixto Equipo+Consulta. La presentación definitiva se prueba con una Memoria cerrada mediante API; el recorrido la reabre al terminar. Los permisos modificados del perfil mixto se restituyen en un bloque de limpieza.

`make ci` y `make recorridos` ejecutan esta prueba después de F0.11. El recorrido de Memoria F4.02 se ejecutó por separado con su juego de nueve familias y 34 menores. El reinicio de la aplicación de demostración entre baterías evita que los límites de entrada acumulados alteren una prueba independiente; conserva su base de datos.
