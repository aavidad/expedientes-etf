# Manual de programadores de Expedientes ETF

Autor: Expedientes ETF.

Este manual explica cómo mantener y ampliar Expedientes ETF. Está dirigido a quien vaya a cambiar el servidor Go, la interfaz React, las migraciones o los paquetes de idiomas, temas y plantillas. Describe la aplicación y sus pruebas; las reglas funcionales se consultan en los documentos de producto.

## Capítulos

| Capítulo | Qué encontrarás |
| --- | --- |
| [1. Arquitectura y módulos](01-arquitectura.md) | Dependencias entre capas, composición, contratos y transacciones |
| [2. PostgreSQL y migraciones](02-postgresql.md) | Roles técnicos, RLS, historia, manifiestos y reserva de números |
| [3. API HTTP, sesiones y CSRF](03-api.md) | Registro de rutas, autorización, errores, versiones y recuperación |
| [4. Interfaz React](04-interfaz.md) | Estructura, navegación, formularios, traducción y temas |
| [5. Idiomas, catálogos, temas y Word](05-paquetes.md) | Directorios, publicación inmutable, marcadores y activación |
| [6. Lectores de anexos y actas](06-lectores.md) | DOCX, ODT, PDF, límites y revisión antes de guardar |
| [7. Memoria, SPAF y auditoría](07-calculos-auditoria.md) | Lectura histórica, cálculo, cierres, Consulta y cadena de eventos |
| [8. Pruebas y entrega](08-pruebas-entrega.md) | Go, PostgreSQL, navegador, F0.11, CI, demo, vídeos y Windows |
| [9. Recetas de ampliación](09-recetas.md) | Añadir un campo, pantalla, plantilla, idioma, catálogo o migración |

El [PDF del manual](programadores.pdf) contiene estos capítulos, sus diagramas y capturas. La [evidencia de edición](EVIDENCIA.md) recoge los comandos ejecutados y la procedencia de las capturas.

## Antes de cambiar una capacidad

Lee [PRD](../producto/PRD.md) y [PLAN_IMPLEMENTACION](../producto/PLAN_IMPLEMENTACION.md), además de los apartados pertinentes de [TRD](../producto/TRD.md) y [ESQUEMA_BACKEND](../producto/ESQUEMA_BACKEND.md). El [índice de producto](../producto/INDICE.md) permite encontrar sus rangos de líneas. Para una pantalla, consulta también [FLUJO_APP](../producto/FLUJO_APP.md) y [DISENO_UI_UX](../producto/DISENO_UI_UX.md).

Busca después la capacidad en el código. El código determina qué está implementado; una ruta descrita como prevista en producto puede no estar registrada aún. Los puntos de entrada son [cmd/etf/modulos.go](../../cmd/etf/modulos.go), [el registro HTTP](../../internal/adaptadores/http/) y [web/src/pantallas.ts](../../web/src/pantallas.ts). Sigue desde ellos el puerto de entrada, el caso de uso y sus puertos de salida.

Trabaja en la rama y carpeta asignadas. Conserva los cambios de otras tareas y limita la entrega a la capacidad encargada. Una decisión funcional pendiente se resuelve antes de convertirla en código o en una constante.

## Preparar el PDF

Desde la raíz, con LibreOffice, Pillow y Graphviz instalados en el equipo:

```sh
TMPDIR="$PWD/bin/temporales" python3 -B docs/manuales/exportar.py --manual programadores
```

El exportador usa los capítulos de esta carpeta y escribe `docs/desarrollo/programadores.pdf`. La conversión es local. Graphviz dibuja los diagramas de flujo Mermaid de estos capítulos a partir de sus nodos y flechas. No descarga recursos. Sin `--manual`, el exportador conserva la selección de los tres manuales de usuario existentes.

Para guardar una edición de revisión fuera del repositorio:

```sh
TMPDIR="$PWD/bin/temporales" python3 -B docs/manuales/exportar.py \
  --manual programadores --salida bin/temporales/MANDEV-pdf
```

Las capturas se renuevan con la demo de este worktree, Playwright y Chrome del sistema. [Pruebas y entrega](08-pruebas-entrega.md#capturas-y-pdf) explica el comando y las comprobaciones.
