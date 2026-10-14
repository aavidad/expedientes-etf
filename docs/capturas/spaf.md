# Tabla Mensual para el SPAF · SP1

La pantalla está en `/equipo/tabla-spaf`. Marta Cabrera consulta sólo el ETF 02;
Inés Moreno elige cualquier equipo. El mes anterior aparece seleccionado.
Los textos se cargan del idioma activo y el aspecto procede del tema.

La demo carga cuatro expedientes inventados por la API de alta histórica.
En septiembre de 2026, el ETF 02 tiene 2 altas, 2 bajas y 3 casos. El expediente
02-83-26 cuenta en alta y baja porque se aceptó y cerró en ese mes. El ETF 01
tiene 1 alta y 1 caso. Los recuentos de menores no están disponibles: el resumen
muestra «Sin dato», con la cantidad de expedientes afectados. No se sustituye
un dato desconocido por cero. Cuando hay un recuento parcial, se rotula
«Menores con dato». Los meses sin casos muestran cero y una salida clara.

Los enlaces del desglose abren una consulta auditada del expediente y muestran
las fechas y motivos de sus movimientos del mes. No presentan nombres ni
descripciones libres. Volver conserva equipo y mes. La vista previa contiene
las 14 columnas, con desplazamiento dentro de su tarjeta.

La descarga Word envía las versiones de la vista previa, CSRF y una clave de
idempotencia. Un resultado incierto conserva en memoria de la pestaña sólo
identificadores, versiones, idioma y clave, separados por usuario. Permite
recuperar la emisión original incluso tras cambiar de idioma. El aviso aclara
que sus datos pueden diferir de la vista actual. No se guarda información en
el almacenamiento del navegador; cerrar o recargar la pestaña pierde esa
recuperación local. Un conflicto pide revisar los cambios antes de emitir.

Capturas con datos inventados:

- [Escritorio, 1440 px](spaf-escritorio.png).
- [Móvil, 390 px](spaf-movil.png).
- [Alto contraste en móvil](spaf-alto-contraste-movil.png).

Comprobación reproducible, con la demo arrancada:

```sh
node scripts/recorridos/spaf.mjs http://127.0.0.1:18275
```

Ajusta el puerto al mostrado por `make demo`. El recorrido comprueba los dos
perfiles, los recuentos anteriores, las 14 columnas, el desglose y su retorno,
la descarga real por POST y su repetición idempotente. Comprueba ayuda por
teclado, Escape y devolución del foco, desplazamiento de la tabla, los anchos
1440, 1024, 390, 360 y 320, ambos temas e inglés. El resultado final es
`SPAF-RECORRIDO-OK`.

Los estados 503 y 409 se prueban con respuestas simuladas en navegador. La
pérdida de respuesta se simula después de una emisión real confirmada; el
reintento conserva cuerpo y clave y recibe la emisión repetida. Una fila
sintética comprueba iniciales y centro de protección. El aislamiento de Marta
se prueba con una petición real a otro equipo, que devuelve 403.

La segunda revisión independiente de código, textos y capturas corrigió el
centro de protección, la recuperación al cambiar idioma y la presentación de
los recuentos desconocidos. La revisión focal y Semgrep se ejecutan localmente,
sin enviar código ni datos a servicios externos. No hay migraciones nuevas.
La matriz independiente incluye las tres rutas SPAF nuevas. Su preparación RLS
respeta los expedientes cerrados. Tras integrar F2.06, compara las lecturas de
Marta y Pablo con sus equipos reales y conserva las pruebas de escritura ajena.
La pertenencia de Marta al ETF 02 también se comprueba por HTTP.

La integración con F2.06 conserva la bandeja y ficha del equipo junto a SPAF.
La demo prepara primero los expedientes de equipo y después los históricos
mensuales. Tras integrar también salidas de Coordinación y cierre de Memoria,
F0.11 verifica 1.356 combinaciones HTTP, 350 lecturas RLS y 750 escrituras
denegadas. El recorrido SPAF y la CI completa vuelven a pasar.

Límites: la proyección SPAF actual no recoge todos los datos de ficha; esas
celdas permanecen «Sin dato». La ZTS sigue pendiente de D19. No se ha ejecutado
NVDA ni el comprobador de accesibilidad de un procesador de textos. La prueba
de descarga verifica estructura DOCX y los números de los tres casos del ETF 02.
