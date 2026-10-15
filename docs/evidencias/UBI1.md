# UBI1 · Editor de equipos, municipios y centros

Rama: `editor-ubicaciones`. Código revisado: `3e2a0eb`, después de integrar `main` (`5dd23bc`).

Administración funcional dispone de `/admin/ubicaciones`. Puede dar de alta equipos y centros, indicar alta y baja, cambiar las dos asignaciones independientes de un municipio o entidad local autónoma, su ámbito y sus variantes. Los textos se publican en castellano e inglés mediante los paquetes de catálogo existentes: archivos inmutables por huella en los directorios de idioma, referencias activadas y auditoría en la misma transacción que los datos. No hay denominaciones ni variantes en la base de datos.

La migración `catalogo/0015_ubicaciones.sql` está reservada en `ETF_RESERVAS_MIGRACIONES.md`, obtenida con `siguiente.sh`. Conserva las identidades anteriores y añade instantáneas de vigencia y ámbito. Una publicación empieza después de la última; la fecha de alta de una identidad nueva no precede a su primera publicación. La baja es exclusiva: desde ese día la entrada deja de estar activa. La lectura de la base anterior a UBI1 conserva su contrato para los ensayos de actualización.

## Pruebas PostgreSQL

`python3 -B migraciones/catalogo/ensayar.py`, PostgreSQL 18 desechable, datos inventados:

```text
UBICACIONES-OK asignaciones independientes, ámbito, grafías publicadas, alta/baja e historia; permisos y auditoría fallida revierten datos y textos
ENSAYO-OK actualización desde catalogo/0002 + paquete/0001; UUID y versión semver conservados
ENSAYO-OK contenedor y volumen destruidos
```

El ensayo usa los mismos archivos de idioma que la prueba de edición: comprueba que un fallo de auditoría revierte tanto la versión de datos como las referencias activas. Las pruebas negativas rechazan la publicación por Coordinación, profesionales, Consulta, sistema y ausencia de perfil, también al llamar directamente a la función SQL. Comprueban que las asignaciones antiguas, el ámbito y la fecha de baja de una versión anterior se conservan y que un alta nueva no aparece en versiones antiguas.

## Recorrido Chrome

`make demo` con puerto `18095`; `node scripts/recorridos/ubicaciones.mjs http://127.0.0.1:18095`. Usuarios inventados: Elena Salas, Administración; Inés Moreno, Coordinación. Sus credenciales se leen del archivo privado de la demo y no se incorporan a esta evidencia.

El recorrido publica una asignación para Íllora con centro y equipo elegidos por separado, cambia el ámbito y añade variantes. Un DOCX inventado, basado en el Anexo 3 del corpus local, usa «Villa de Íllora»: el lector reconoce el municipio y su ámbito publicado. Antes de publicar se conservan los números y equipos de los expedientes visibles de la demo y se cierra la Memoria provincial de 2026; después se comparan esos datos y la instantánea cerrada, sin diferencias. Al terminar, el recorrido reabre sólo la Memoria que él mismo cerró, también si falla una comprobación, para conservar el estado previo de la demo.

También comprueba conflicto real con una segunda publicación, campos conservados, comparación de valores actuales, revisión expresa, CSRF, denegación a Coordinación, vacío, error de red traducido, recuperación y foco. Axe revisa la pantalla sin infracciones. Chrome se recorre a 1440, 1024, 390 y 360 píxeles, en castellano e inglés, con el tema ordinario y alto contraste. Las capturas están enlazadas en el apartado de capturas del README.

## Revisiones independientes

La revisión SQL y `security-audit` focal de fuente dio GO para `3e2a0eb`: permisos mínimos, RLS, funciones, inmutabilidad, consultas parametrizadas y publicación auditada. La segunda revisión de usabilidad dio GO tras comprobar Chrome, teclado, móvil, ambos idiomas y alto contraste. Se corrigieron los hallazgos iniciales: copia incorrecta de una baja nula, etiquetas estáticas, acceso a borradores, conflicto sin comparación, error de red vacío, foco y selector de equipo agotado.

Semgrep local, sin métricas ni servicios externos, con las reglas existentes de `scripts/ci/equipo/semgrep-local.yaml` y `despliegue/base/semgrep.yaml`:

```text
Ran 7 rules on 19 files: 0 findings.
```

La revisión automática y de teclado acredita este recorrido; no sustituye una evaluación completa con lector de pantalla.

## CI

`make ci` completo pasó sobre el código y la composición integrados con `main`:

```text
CI mínima completada.
```

Incluye `go vet ./...`, `go test ./...`, pruebas y compilación del frontend, comprobación de textos y temas, PostgreSQL limpio y actualización, y recorridos Chrome. La matriz INT5 pasó 1680 combinaciones; la prueba RLS pobló 54 tablas y comprobó 378 lecturas y 810 escrituras denegadas con siete actores.

```text
UBI1-NAVEGADOR-OK Chrome 1440/1024/390/360, es/en, alto contraste, teclado, vacío/error/conflicto, CSRF y Coordinación 403
UBI1-ANEXO-OK DOCX inventado con variante publicada reconoce Íllora y su ámbito
UBI1-REGRESION-OK expedientes numerados y Memoria cerrada conservados tras publicación; asignaciones y variantes verificadas
```

Se corrigieron dos fallos durante la validación: lectura del esquema anterior y limpieza de la Memoria creada por el recorrido. Una ejecución agotó la espera de Chrome en la prueba de PWA con dos idiomas sin conexión; su repetición aislada y la ejecución completa siguiente pasaron, sin modificar esa capacidad.

La demo se detuvo con `make demo-parar`; se retiraron los temporales de esta tarea. No se hizo push.
