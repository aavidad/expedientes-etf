# Carga de expedientes de 2026 · CA1

Las pantallas de carga están disponibles en `/coordinacion/carga-2026` y
`/equipo/expedientes/alta-historica`. Coordinación elige equipo; el profesional
sólo carga el suyo. El alta conserva número original, fechas y episodios, permite indicar nombres
conocidos de menores también en no aceptados,
permite añadir menores y datos básicos de contacto, y revisa antes de guardar.
La carga desde CSV o XLSX muestra filas enmascaradas e incidencias por fila.
Confirmar guarda el lote completo y deja la marca «Cargado para revisión».

La pantalla permite descargar un CSV con las 18 columnas de ESQUEMA_BACKEND
§7.2. Usa municipio y centro del equipo elegido y datos inventados de un vivo
con reapertura, un cerrado y un no aceptado. Si el equipo carece de municipio
asignado, incluye sólo el no aceptado. Los números originales inventados de
los aceptados son los correlativos 91 y 92 del equipo elegido; si ya están
cargados, el servidor rechaza el lote y hay que corregirlos en la hoja. No
se conserva el fichero ni el borrador. El navegador los mantiene en memoria
hasta terminar o salir de la pantalla.

## Repetir el recorrido

```sh
ETF_DEMO_PUERTO=51481 make demo
node scripts/recorridos/carga.mjs http://127.0.0.1:51481
make demo-parar
```

Elige otro puerto si está ocupado. El recorrido usa las cuentas inventadas
Inés Moreno y Marta Cabrera, y Elena Salas para comprobar la denegación a
Administración. Lee las contraseñas del fichero local de la demo y abre
`/usr/bin/google-chrome` mediante Playwright. No escribe credenciales en los
artefactos. Conserva los expedientes de cada ejecución en la demo.

## Comprobaciones

El recorrido comprueba alta aceptada, en intervención con reapertura, cerrada
y no aceptada; errores enlazados al campo y foco de ayuda; revisión sin nombres
completos; fechas históricas; estado y marca recuperados del servidor; descarga
CSV, hoja inválida sin botón de confirmar, CSV y XLSX válidos; y resultado.
Simula una pérdida de conexión después de guardar la hoja y recupera el
mismo recibo con la misma clave de idempotencia. Comprueba además el equipo de
Marta, una petición a otro equipo rechazada y la denegación a Administración.

Revisa etiquetas de controles, un solo título principal, identificadores
únicos, controles de al menos 44 px, ausencia de persistencia del navegador y
reflujo a 1440, 1024, 390 y 360 px. Comprueba los temas predeterminado y alto
contraste, castellano e inglés. Las capturas enlazadas en README contienen
sólo datos inventados.

La revisión independiente de fuente aplicó las skills del proyecto de
interfaz, recorridos y cambios, `security-audit` en modo focal y `humanizer`.
Se corrigieron la etiqueta de identidad del profesional y el resumen de
fechas y episodios antes de confirmar. El recorrido detectó que la imagen de demo no incluía los parámetros locales
de familia que ya lleva la imagen de producción; se añadió su copia al
preparar y construir la demo. ESQUEMA §7.2 se corrigió para usar
`vigente_desde`, conforme al contrato real de familia y menores.

No se añadieron migraciones ni
permisos de escritura. La consulta de opciones conserva la auditoría y el
permiso exclusivo de Coordinación de la consulta de decisiones.

## Límites

Las pruebas locales no acreditan las condiciones operativas de producción,
la EIPD ni la carga de datos reales. `make ci` incorpora este recorrido tras F0.11. No se ejecutó NVDA en Linux ni una
revisión completa con lector de pantalla. El resultado permite consultar los datos guardados mediante la lectura auditada
`GET /api/expedientes/{id}`, también para no aceptados y números desconocidos.
La marca de revisión se recupera del servidor. El enlace
`/equipo/expedientes/alta-historica?expediente=<id>` permite volver a la
consulta y recargarla; exige la sesión y el equipo autorizado. El recorrido
lo comprueba con el no aceptado de Marta. Las pantallas de otros módulos
quedan fuera de este cambio.

## Resultado del recorrido

```text
CA1_OK: altas aceptada/intervención/cerrada/no aceptada, reapertura, CSV/XLSX, incidencias, recuperación idempotente, revisión recuperable, permisos, teclado, 1440/1024/390/360, temas e idiomas.
```

La segunda revisión visual comprobó etiquetas y contraste de botones, reflujo
a 390 px, alto contraste y consulta del no aceptado por el equipo. Los
ajustes de singular y nombres conocidos se comprobaron en el recorrido
posterior.

La matriz independiente F0.11 incluye ambas rutas de CA1 para Coordinación y
Profesional de ETF. El resumen de iniciales se construye en tiempo lineal
respecto al nombre recibido.

El ensayo RLS F0.11 cuenta también los históricos cuyo ámbito está en
`equipo_carga_id`, incluidos no aceptados sin número, para el expediente y sus
datos de derivación. Los recuentos esperados se obtienen como propietario antes
de aplicar cada perfil; las políticas de producción no se modifican.

En CI, tras la matriz F0.11 se reinicia sólo la app de la demo y se espera su
salud: las pruebas anteriores alcanzan el límite de admisión de sesión en
memoria. Los límites de producción no se alteran. El recorrido consulta todas
las páginas de expedientes y usa cinco números originales libres de dos
cifras, para repetirlo sin borrar los registros de ejecuciones anteriores.

La puerta completa `make ci` terminó con `CI mínima completada.`: incluye
`go vet ./...`, `go test ./...`, pruebas y compilación web, textos, temas,
ensayos PostgreSQL, F0.11 y CA1. Semgrep local terminó con
`Ran 5 rules on 8 files: 0 findings.`

Tras integrar UI3 y los vídeos de main, se repitió la puerta completa y CA1.
Las capturas finales usan los textos actuales de los idiomas de main. La
segunda revisión visual confirmó el menú integrado y la revisión del no
aceptado a 390 px. La demo quedó parada con `DEMO-PARADA`.

## Regresión al integrar FX1, UI3 y VID4

El timeout al buscar «Revisar el expediente» se reprodujo quitando
`carga.revisar_manual` de la reserva de textos de la demo. La ruta y los
campos de equipo seguían disponibles, pero el botón tenía una etiqueta
vacía. El volumen de idiomas conserva ficheros de ejecuciones anteriores;
recompilar la imagen no sustituye esos ficheros.

El recorrido compara los textos de CA1 que sirve la demo con
`idiomas/{es,en}/textos/carga.json` antes de abrir Chrome. Si son anteriores,
renueva sólo esos dos ficheros de la reserva de la demo del worktree. Comprueba
que la URL coincide con la demo local antes de copiar y vuelve a verificar
las respuestas de los dos idiomas. No modifica versiones publicadas,
catálogos, expedientes ni permisos. Si un idioma activo sigue sin servir las
claves actuales, termina con una explicación antes de buscar botones.

La reproducción falló con `locator.click: Timeout 20000ms exceeded` en la
línea 48 del recorrido anterior. Con la corrección, el mismo volumen produjo:

```text
CA1_TEXTOS_DEMO_ACTUALIZADOS: es.
CA1_OK: altas aceptada/intervención/cerrada/no aceptada, reapertura, CSV/XLSX, incidencias, recuperación idempotente, revisión recuperable, permisos, teclado, 1440/1024/390/360, temas e idiomas.
```

Tras incorporar FX1, TR1, los recorridos de ficha y el manual de sistemas de
main, `make ci` completo volvió a terminar con `CI mínima completada.` y el
recorrido con `CA1_OK`. La revisión independiente del guard y la copia de
textos dio GO. Las consultas de textos tienen un límite de diez segundos.
Semgrep local del recorrido: `Ran 3 rules on 1 file: 0 findings.`
