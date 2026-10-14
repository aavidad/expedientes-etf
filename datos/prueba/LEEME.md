# Datos de prueba

`expedientes.json` contiene 64 familias completamente sintéticas, generadas con semilla fija. Hay ocho equipos (ocho familias por equipo), derivaciones de 2025 y 2026, adultos y menores, los dos subprogramas, estados de recepción e intervención, ingresos, factores, tipologías y fechas. Las denominaciones de centros educativos son ficticias; los municipios son localidades reales de Granada. No incluye DNI ni información de personas reales.

Para regenerar el fichero, ejecuta `python3 datos/prueba/generar/generar.py`. Para comprobar que el fichero coincide con el generador y pasa los controles de privacidad, ejecuta:

```sh
python3 -B datos/prueba/generar/generar.py --check
python3 -B datos/prueba/generar/comprobar.py
```

## Casos frontera

| Familia | Caso |
| --- | --- |
| familia-01 | Consentimiento justo el día 30 desde la aceptación. |
| familia-02 | Aceptada, sin consentimiento al día 31. |
| familia-03 | Cambio de composición durante el año. |
| familia-04 | Cierre y reapertura en episodios distintos. |
| familia-05 | Traslado del equipo 5 al equipo 6. |
| familia-06 | Menores de Riesgo y Reunificación en una misma familia. |
| familia-07 | No aceptación sin número asignado. |
| familia-08 | Lista de espera sin número asignado. |
| familia-09 | Quince menores, máximo de composición previsto. |
| familia-10 | Un menor cumple 18 años durante el año evaluado. |
| familia-11 | Cierre por renuncia sin consentimiento. |
| familia-12 | Cambio de composición con alta y baja de menores. |
| familia-13 | Cierre y reapertura en episodios distintos. |
| familia-14 | Traslado del equipo 6 al equipo 7. |
| familia-15 | Segunda familia con menores de ambos subprogramas. |
| familia-16 | Derivación fechada el 31 de diciembre de 2025. |

Los hitos guardan la fecha de cada transición y el equipo de origen/destino de los traslados. La numeración de aceptados sigue provisionalmente `equipo-correlativo-año`, con guiones; los casos no aceptados y en espera no consumen número.
