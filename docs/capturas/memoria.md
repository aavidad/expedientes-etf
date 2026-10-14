# Memoria Técnica Anual: recorrido de F4.02

La pantalla `/memoria` consulta los cálculos existentes de los 28 apartados, con
sus subapartados, y los agrupa en cuatro bloques. Cada tabla conserva población,
base del porcentaje, «Sin dato» y avisos. Los textos A, D, E, F y G y el Word
usan los servicios de F4.05. Estado, cierre confirmado, foto bloqueada y reapertura con motivo usan F4.06.
La composición por números de expediente sigue pendiente de F4.07.

## Comprobación con datos inventados

Demo local en el puerto 41889, Chrome del sistema y Playwright. El recorrido
existente del Anexo 3 registró la aceptación de una familia con seis menores.
La demo preparada, incluidos los nuevos juegos de equipo y SPAF de main,
devolvió nueve familias y 34 menores en la provincial; el ETF-04 tenía dos
familias y doce menores. Se contrastó con los registros de episodios y menores
por equipo (01: 1/1, 02: 5/15, 04: 2/12, 06: 1/6), sin usar el motor para
obtener los valores esperados.
Los nacimientos y la Atención Temprana que no publica la aceptación se leen
sólo de sus historias registradas; si faltan, quedan como «Sin dato».

Se comprobó:

- Coordinación elige equipo o provincia; la provincia se calcula desde los datos.
- Los cuatro bloques mantienen tablas con cabeceras accesibles y avisos.
- Tras integrar FX1, los 34 menores tienen cifras conocidas por sexo visibles en el apartado 2, contrastadas con los registros de la demo.
- Los textos guardados se recuperan desde el servidor.
- Una respuesta de cierre perdida se recupera con la misma clave y versión; no crea otra revisión. El cierre bloquea la edición y conserva la descarga Word. La reapertura exige un motivo y devuelve la edición a Coordinación.
- La descarga Word se bloquea mientras hay cambios sin guardar.
- Una respuesta Word perdida después de confirmar la emisión se recupera con
  la misma clave y el mismo identificador, también tras volver a consultar el
  mismo ámbito; no crea una segunda emisión. Cambiar de idioma genera el Word
  correspondiente con una intención nueva.
- El profesional sólo consulta su equipo; una petición a otro devuelve 403.
- Si falta una publicación, Consulta conserva el selector de equipos y el aviso.
- Consulta recibe cifras publicadas y su fecha, sin narrativas, composición,
  campos de edición ni descarga narrativa. Su petición a narrativas devuelve 403.
- Teclado y anchuras de 1440, 1024, 390, 360 y 320 px, sin desplazamiento
  horizontal de la página. Las tablas tienen una región propia de desplazamiento.
- Tema de alto contraste e inglés, también sin desbordamiento en móvil.

Órdenes del recorrido:

```sh
ETF_DEMO_PUERTO=41889 make demo
make demo-recorrido
ETF_MEMORIA_FAMILIAS=9 ETF_MEMORIA_MENORES=34 ETF_MEMORIA_EQUIPO=04 node web/tests/memoria.navegador.mjs
make demo-parar
```

## Capturas

- [Escritorio, 1440 px](memoria-escritorio.png).
- [Móvil, 390 px](memoria-movil.png).
- [Alto contraste](memoria-alto-contraste.png).
- [Memoria cerrada](memoria-cerrada.png).

## Fuente histórica y límites

El lector de FX1 usa vigencia y fecha de conocimiento para familia, adultos,
menores e intervención mediante las vistas históricas de `expediente/0022`.
Conserva sexo, nacimiento, subprograma, Atención Temprana, fase, factores,
tipologías y recursos con sus exclusiones por menor. Distingue cada lista sin
valorar de una lista vacía confirmada. Los campos sin fuente conservan «Sin dato»;
no se deducen de otros campos.

Consulta lee la última publicación de cifras. Coordinación y el equipo la
actualizan al consultar, en la misma transacción que confirma la auditoría.
La actualización nocturna sigue pendiente. Si no hay publicación, la pantalla
lo indica sin inventar ceros.

## Revisiones y situación de integración

Usabilidad revisada en una segunda pasada independiente. Las correcciones de
etiquetas por dimensión, avisos de porcentaje, recuperación Word, límites de
líneas y permisos combinados fueron comprobadas de nuevo. Revisión independiente
SQL y de seguridad favorable para `estadistica/0004`, ensayada en PostgreSQL 18 aislado:
Consulta no lee la tabla base ni escribe; sólo accede a la proyección de cifras.
La auditoría fallida revierte la publicación.

El ensayo incluyó la fuente real de los 33 apartados con subapartados y rechazó
narrativas, composición, nombres usados como códigos, identificadores de
expedientes y publicaciones fuera del ámbito. Semgrep local: tres reglas del proyecto, ocho ficheros, sin hallazgos.

La migración `estadistica/0004_cifras_memoria.sql` está registrada con dependencia
`0002`. La revisión final validó también los valores «sí/no» del apartado 10 y rechazó códigos ajenos a ese conjunto. Huella SHA-256 final: `0983e1ba0c78dc82b7049a5cfdbf4fae093064719bee759afa813410daac6d66`. El revisor compiló sus propios binarios y comprobó instalación limpia,
actualización desde main y repetición. Para la corrección final del apartado 10 reutilizó el binario de pruebas y añadió comprobaciones SQL independientes. El cierre mantuvo siete familias al
corregir la fuente a nueve; la reapertura permitió publicar las nueve. Consulta
conservó exclusivamente las cifras en ambos estados.

La pantalla obtiene cifras, textos y fecha de la misma revisión cerrada. El
cierre se confirma en un diálogo; la reapertura exige un motivo y sólo aparece
para Coordinación. Los campos narrativos quedan bloqueados mientras está
cerrada. Descargar Word conserva el estado.

No se ha validado con NVDA ni se han aprobado los resultados del Programa
pendientes de D22.
