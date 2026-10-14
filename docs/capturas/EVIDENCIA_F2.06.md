# Validación de la bandeja del equipo

Los recorridos usan Chrome del sistema y datos inventados de la demostración local.
Marta Cabrera consulta ETF2; Inés Moreno consulta la provincia. Se comprueban la
búsqueda enmascarada, los filtros, el estado vacío, las pestañas mediante teclado,
el cierre con versión y la separación entre documentos emitidos y fechas confirmadas.
También se comprueban las denegaciones para acceso anónimo, Administración,
Consulta y un expediente de otro equipo.

Salidas descarga el Anexo 5 sin confirmar el envío, confirma su fecha mediante
una operación separada y guarda «A la espera» sin arrastrar una fecha oculta.
Familia consulta los datos vigentes de F2.01a; las familias de este recorrido
no tienen personas adultas registradas y muestran ese estado vacío. La edición
de Familia no forma parte de esta pantalla de lectura.

Las capturas de [bandeja en escritorio](equipo-bandeja-escritorio.png),
[bandeja en móvil](equipo-bandeja-movil.png),
[ficha en escritorio](equipo-ficha-escritorio.png) y
[ficha en móvil](equipo-ficha-movil.png) corresponden a 1440 y 390 píxeles.
El recorrido también comprueba que no haya desbordamiento a 360 y 1024 píxeles.

Para repetir la comprobación:

```sh
make demo
node despliegue/demo/equipo-recorrido.mjs http://127.0.0.1:18083
python3 -B migraciones/expediente/ensayar_bandeja.py
make ci
make demo-parar
```

Si se elige otro puerto al arrancar, úsalo también en el recorrido.
El ensayo SQL comprueba instalación y actualización, iniciales, límites de
equipo, perfil doble, historial y ausencia de datos si falla la auditoría.
Las revisiones independientes de seguridad, SQL y usabilidad son locales.
No se ha ejecutado NVDA ni una auditoría completa de accesibilidad.
