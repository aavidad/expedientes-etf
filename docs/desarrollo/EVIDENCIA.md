# Evidencia del manual de programadores

Autor: Expedientes ETF.

La edición MANDEV añade nueve capítulos y el índice de lectura en esta carpeta. Las fuentes son el código de la rama, TRD y ESQUEMA_BACKEND. El manual distingue los contratos actuales de capacidades pendientes, incluida la copia pública sin generador en esta rama.

## Capturas de la aplicación

Las cuatro imágenes de `capturas/` se obtienen de una demostración propia iniciada con `make demo`, en el puerto 18214. Sólo utiliza cuentas y datos inventados. El capturador entra como Administración para Plantillas Word y como Consulta para Memoria. Abre Chrome del sistema con Playwright y comprueba sesión activa, respuestas API 200, ausencia de errores JavaScript y ancho sin desbordamiento.

```sh
ETF_DEMO_PUERTO=18214 make demo
node docs/desarrollo/capturar.mjs
```

Las capturas y sus textos alternativos están en [Interfaz React](04-interfaz.md) y [Memoria, SPAF y auditoría](07-calculos-auditoria.md). Las imágenes se revisan completas, a 1440 y 390 píxeles. Las cuentas, contraseñas y sesiones no forman parte de esta entrega.

Salida de la captura final:

```text
CAPTURA-OK: plantillas-1440.png; API 200; sin desbordamiento
CAPTURA-OK: plantillas-390.png; API 200; sin desbordamiento
CAPTURA-OK: memoria-1440.png; API 200; sin desbordamiento
CAPTURA-OK: memoria-390.png; API 200; sin desbordamiento
```

La primera comprobación de Memoria con Coordinación detectó desbordamiento a 390 píxeles. Las imágenes finales corresponden a Consulta, cuyo contrato numérico se comprueba en los dos tamaños. Esta entrega documental no corrige ni acredita la disposición móvil de Coordinación.

## Exportación y regresión

El exportador añade `--manual programadores`, capítulos, código con tipografía monoespaciada, tablas, diagramas locales y metadatos de autor. Sin selección conserva los tres manuales anteriores. La regresión compara todos los miembros internos de sus ODT antes y después del cambio; los PDF de esos manuales se convierten en una carpeta temporal, sin tocar sus archivos de publicación.

```sh
TMPDIR=/dev/shm/etf/tmp python3 -B docs/manuales/exportar.py --manual programadores
```

La revisión del PDF comprueba autor, capítulos, diagramas, capturas, bloques de código y tablas. La conversión usa LibreOffice, Pillow y Graphviz locales.

Resultados:

```text
REGRESION-OK: tres ODT anteriores idénticos en todos sus miembros
PDF-OK: profesionales.pdf
PDF-OK: coordinacion.pdf
PDF-OK: administracion.pdf
PDF-OK: programadores.pdf
Author:          Expedientes ETF
Tagged:          yes
Pages:           27
```

Las tres pruebas de `test_exportador.py` comprueban capítulos, enlaces internos, código, tablas, siete figuras y autor; también el rechazo de Mermaid no admitido y las bases de enlaces anteriores:

```sh
TMPDIR=/dev/shm/etf/tmp python3 -B -m unittest discover -s docs/desarrollo -p 'test_*.py'
```

```text
Ran 3 tests in 0.128s
OK
```

## Comprobaciones de entrega

La comprobación inicial de enlaces terminó con 179 documentos, 1024 enlaces locales y 0 errores. Después se integró el `main` local que incorporaba actas ODT y se actualizó el capítulo de lectores. README, guía y plan conservaron ambas aportaciones; el único conflicto de esa integración fue el índice de producto, resuelto con su generador. Una integración posterior incorporó MANADM y FX7: se conservaron el selector repetible y los ajustes de Administración en el exportador, ambas filas del plan y el índice regenerado. El manual recoge la incorporación de archivos ausentes sin limpiar la demo.

Las tres ejecuciones completas de `make ci` terminaron correctamente, incluida la última sobre la rama integrada con MANADM y FX7. Los recorridos generaron capturas fuera de MANDEV: se restauraron desde el índice, sin incluirlas en los commits de esta tarea.

```sh
make ci
```

Última línea de la ejecución final:

```text
CI mínima completada.
```

Incluye `go vet ./...`, `go test ./...`, pruebas y compilación web, textos, temas, ensayos PostgreSQL, incorporación de idiomas y temas FX7, F0.11 y carga con actas DOCX, ODT y PDF. La comprobación final de enlaces terminó con 180 documentos, 1050 enlaces locales y 0 errores.

No se modifica el servidor, el frontend de la aplicación, SQL ni los archivos de otros manuales; el cambio común es la ampliación de `docs/manuales/exportar.py` y los enlaces de las guías. Se actualizan PLAN_IMPLEMENTACION y su índice generado para registrar MANDEV. Los archivos de otras capacidades que entran con la integración proceden de `main`.

El manual señala dos límites del estado consultado: la copia pública carece de generador y la puerta local actual no ejecuta un Semgrep general ni reconstruye toda imagen de producción, aunque TRD enumera esas comprobaciones entre los controles de seguridad y despliegue. No se amplía la CI en esta tarea documental.
