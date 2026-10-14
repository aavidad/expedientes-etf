# F1.11: publicación y activación de plantillas Word

Rama: `f1.11-plantillas`. Implementación: `8c41850`; endurecimiento de campos Word: `a33c0b7`. Integración de main:
`fec160e` (incluye SPAF). `4d01486` conserva INT5 y `a8faae3` integra los últimos ajustes de main. La primera integración sólo requirió regenerar `INDICE.md`. La integración posterior de `main` (`195f643`, con INT5) adapta composición y declaración de rutas al registro por módulos.
`c2501b8` sitúa en su línea la marca de SQL técnico que exige el comprobador de textos, sin cambiar el comportamiento.
La migración `documento/0004` se reservó antes de crearla en el registro de ETF,
fuera de Git.

## Capacidad entregada

Administración funcional consulta la referencia actual y publica un DOCX mediante
`/api/admin/plantillas`. El POST exige CSRF, versión previa y cuerpo Word de hasta
10 MiB. Se reutiliza el conector ClamAV; no se publica sin confirmación de limpieza.

La validación comprueba ZIP, raíces y relaciones OPC, tipos de las partes,
XML acotado y contrato de campos y bloques de la plantilla actual. Rechaza
macros, contenido activo (incluidos campos Word con instrucciones) y referencias externas. Una ruta por UUID y versión
se escribe exclusivamente y se sincroniza antes de activar. PostgreSQL bloquea
la referencia y confirma plantilla, archivo anterior, referencia nueva y auditoría
juntos. La limpieza de una confirmación incierta comparte el bloqueo de activación.
Los documentos generados siguen siendo transitorios.

## Pruebas ejecutadas

- Unitarias de aplicación: tamaño mayor de 10 MiB, EICAR con antivirus falso,
  antivirus ausente o caído, perfil incorrecto y dos activaciones simultáneas.
- Unitarias del validador: marcador desconocido o ausente, campos Word con instrucciones externas, marcador dividido entre
  fragmentos de texto, ZIP disfrazado, macro, XML con DTD, raíces OPC falsas,
  Override anidado, falta de tipo de relaciones, falta de officeDocument,
  destino inexistente, recurso externo y bomba de descompresión.
- HTTP con sesión: Administración, anónimo, Coordinación, Consulta y Equipo;
  CSRF, método incorrecto, cuerpo mayor del límite y exactamente en el límite,
  identidad suministrada por el cliente, parámetros duplicados y respuestas de
  conflicto, infección y antivirus caído.
- `go test -race ./internal/aplicacion/documento ./internal/adaptadores/docx/... ./internal/adaptadores/ficheros/documento ./internal/adaptadores/http`: todos los paquetes pasan.
- `python3 -B migraciones/documento/ensayar.py`: PostgreSQL 18.4 desechable, red
  externa deshabilitada, repositorio en sólo lectura y datos inventados. Aplica
  documento/0003, después /0004 y repite; también comprueba una base limpia.
  Publicación concurrente en dos conexiones, archivo, cadena de auditoría,
  rollback y permisos. El mismo test de publicación se repite en la misma base.

Salida del ensayo:

```text
PUBLICACION-OK PostgreSQL: una activación, archivo, auditoría, rollback y permisos
ENSAYO-OK base limpia, RLS, auditoría, concurrencia e integridad
```

## Revisión SQL y seguridad

Revisión independiente de sólo lectura con `revisar-cambios-etf`,
`revisar-sql-etf` y `security-audit` focal: GO estático para `8c41850`, repetido
para los enganches de `fec160e`. Se corrigieron la validación OPC incompleta y
la repetibilidad de la prueba PostgreSQL. La revisión focal final confirmó GO para `a33c0b7`, incluido el rechazo de `fldSimple`. La integración de enganches en `4d01486` obtuvo también GO independiente. Sin hallazgos bloqueantes pendientes
sobre permisos, CSRF, antivirus, confinamiento, concurrencia o atomicidad.
La revisión independiente no ejecutó código; los ensayos anteriores los ejecutó
el autor en el contenedor desechable.

Semgrep se ejecutó localmente con `migraciones/documento/semgrep-local.yaml`,
`--metrics off --disable-version-check --error --json`, sobre dominio, puertos, aplicación, enganches y adaptadores de
plantillas/documentos. No se envió código ni datos a servicios de análisis.

```text
SEMGREP-OK reglas=5 ficheros=28 hallazgos=0 errores=0
```

## Alcance y límites

Servidor únicamente. No se han creado pantallas. Con INT5, la composición vive
en `cmd/etf/documentos.go`; `dependenciasModulos` comparte el ClamAV del arranque.
Las rutas y sus permisos se declaran junto al handler de plantillas; la tabla
HTTP se deriva del registro de módulos. Se conservan los otros módulos de main.
Se actualizaron PRD, TRD, ESQUEMA, PLAN, INDICE y README en el commit de implementación.
No hay dependencias nuevas, push ni fusión hacia main.
La limpieza es inmediata y conservadora: si no puede comprobar la confirmación,
retiene el huérfano. Su limpieza periódica queda pendiente.

## Puerta completa

`npm_config_offline=true npm_config_install_strategy=linked make ci` terminó con código de salida 0 sobre la rama
integrada con código final `a8faae3`. Incluye instalación desde caché con el lock, tipos, pruebas y build web,
textos, temas, `go vet ./...`, `go test ./...`, compilación y ensayos PostgreSQL.
Gitleaks no detectó secretos. Última línea y pruebas del módulo:

```text
ok  github.com/aavidad/expedientes-etf/internal/aplicacion/documento  0.745s
Textos: 2 idiomas, 231 claves usadas; sin literales visibles.
CI mínima completada.
```

La CI sobre la integración final agotó el disco durante la instalación ordinaria.
Se detuvo esa ejecución y se eliminaron sólo `web/node_modules` y `bin` del propio
worktree, comprobando primero que eran directorios ignorados y sin ficheros
versionados. Se recuperaron 4,1 GB. La instalación `linked` de npm 11.12.1 reduce
las copias físicas de las mismas dependencias; respeta versiones y sumas del lock.
Con ella la puerta completa pasó sin omitir comprobaciones. No se cambiaron
`package.json`, `package-lock.json` ni la configuración de CI.

## Revalidación después del traslado a memoria

El worktree se trasladó a `/dev/shm/etf/wt-F1.11`; los temporales se sitúan en
`bin/temporales`. Se repiten las comprobaciones desde esa ubicación.
Gitleaks encontró una cabecera `Idempotency-Key` fija de una prueba HTTP en el
historial de otra rama. No es una credencial. `.gitleaksignore` exceptúa sólo su
huella exacta (commit, archivo, regla y línea); no excluye la prueba completa ni
desactiva el análisis de secretos. No se modifica la lógica de ese módulo.

La integración final en memoria es `cc99d6a`: conserva documento/0004 y /0008,
los conflictos HTTP de plantillas y finalización y todas las pruebas de ambos
recorridos. El snapshot previo del ensayo excluye 0003, 0004 y 0008; el migrador
aplica las tres con sus dependencias y comprueba también una base limpia.
La revisión independiente confirmó GO estático para esta integración.

`TMPDIR=bin/temporales npm_config_offline=true make ci` terminó con salida 0,
con instalación ordinaria de npm y todas las comprobaciones activas. Pasaron
`go vet ./...`, `go test ./...`, los ensayos PostgreSQL y los textos:

```text
Textos: 2 idiomas, 233 claves usadas; sin literales visibles.
PUBLICACION-OK PostgreSQL: una activación, archivo, auditoría, rollback y permisos
ENSAYO-OK base limpia, RLS, auditoría, concurrencia e integridad
CI mínima completada.
```

Las pruebas con `-race` del módulo y los adaptadores pasan también en memoria.
Semgrep local sobre 28 ficheros: cero hallazgos y cero errores. No se levantó
una demo; los contenedores de PostgreSQL fueron desechables.
