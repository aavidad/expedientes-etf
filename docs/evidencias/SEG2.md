# SEG2 · Cambios propios de perfiles y equipos

Fecha: 14/10/2026. Rama: `seg-autoconcesion`. Decisión del responsable del producto sobre R-10.

Administración funcional puede consultar sus permisos. Otra persona de Administración debe concederle o retirarle perfiles, cambiar sus vigencias o equipo y dar de baja su acceso. El servidor rechaza cualquier cambio propio antes de escribir y confirma el intento en la auditoría con resultado `denegado`. Si falla la auditoría, devuelve un error de persistencia.

La migración `identidad/0003_impedir_cambios_propios.sql` repite la comprobación bajo el bloqueo de cambios de permisos. `identidad.cambiar_perfiles` devuelve `false` y registra el rechazo, sin alterar estado, versión, perfiles ni equipos. Una llamada SQL directa debe confirmar la transacción para conservar el apunte; lanzar una excepción lo revertiría. Devuelve `true` después de un cambio autorizado y auditado. Las tablas conservan RLS forzada y `etf_app` carece de escritura directa.

El número 0003 se obtuvo con `siguiente.sh identidad` y quedó reservado en el registro de reservas de migraciones, fuera de Git. No se modifica una migración anterior.

La primera cuenta mantiene el procedimiento de sistemas: configuración privada de `ETF_PRIMER_ADMINISTRADOR`, preparación de roles técnicos y migraciones, y primera entrada validada por AD. La concesión, su marca única y la auditoría se confirman juntas. Retirar el perfil no provoca su reconcesión. Procedimiento en el [manual de sistemas, instalación](../sistemas/LEEME.md#2-instalación-desde-el-repositorio); uso ordinario en el [manual de Administración](../manuales/administracion.md#cambiar-perfiles-o-vigencias).

## Pruebas ejecutadas

- `go vet ./...`: terminó con código 0, sin salida.
- `go test ./...`: terminó con código 0; incluye dominio, aplicación, HTTP y arquitectura. La integración PostgreSQL se ejecutó por separado con el entorno activado.
- `go test -tags navegador_integracion -run '^$' ./cmd/etf`: compiló la preparación del recorrido anterior, adaptada para que una segunda persona conceda la combinación de perfiles. Este comando no ejecuta ese recorrido.
- `npm test`, desde `web`: 36 pruebas, 36 aprobadas, 0 fallidas y 0 omitidas.
- `npm run build`, desde `web`: compilación TypeScript y Vite completadas.
- `./scripts/ci/textos/comprobar.sh`: catálogos es/en y ausencia de literales visibles comprobados.
- `python3 -B migraciones/identidad/ensayar.py`: PostgreSQL 18 desechable, actualización desde identidad/0002, repetición sin reaplicar e instalación desde cero. Ejecuta la matriz de permisos tanto sobre la base actualizada como sobre la limpia; rechaza cualquier prueba omitida. Comprueba cinco cambios propios en servidor y SQL, historia y versión intactas, auditoría del rechazo, fallo de auditoría, ausencia de DML directo, retirada por otra persona, concurrencia y alta inicial sin reconcesión.

Salida real del ensayo:

```text
SEG2-RLS-OK: cinco autocambios denegados en servidor y SQL, auditoría persistida, historia intacta y DML directo prohibido
ENSAYO-OK SEG2: actualización, repetición, instalación limpia, autocambios, RLS, auditoría y alta inicial
```

## Recorrido de navegador

Se ejecutó `ETF_DEMO_PUERTO=18142 make demo` y `node scripts/recorridos/seg2.mjs http://127.0.0.1:18142`, con Playwright y `/usr/bin/google-chrome`, usando sólo personas y datos inventados de la demo de este worktree.

El recorrido comprueba que Inés Moreno, con Coordinación, no accede a Usuarios. Una persona de Administración concede temporalmente Administración a una segunda cuenta. Esta segunda persona asigna un equipo a la primera. Cinco intentos propios por API devuelven 403: añadir Coordinación, retirar Profesional de ETF y su equipo, cambiar de equipo, dar de baja el acceso y modificar vigencias. Después, la segunda persona retira el equipo y la primera retira el permiso temporal de la segunda. Los cambios de prueba y las preferencias de idioma y tema se restauran.

La consulta propia se abre con teclado y muestra «Mis permisos», el aviso y todos los controles desactivados. Se prueban es/en, el tema por defecto y alto contraste; anchos 1440, 1024, 390 y 360 sin desbordes. La reserva de una demo existente conserva sus textos anteriores: el guion verifica las claves servidas, renueva sólo `usuarios.json` es/en de esta demo si hace falta y exige el título antes de capturar.

Salida real:

```text
SEG2-CHROME-OK: coordinación denegada; cinco autocambios API 403; consulta propia por teclado; segunda persona concede y retira; es/en y alto contraste; 1440/1024/390/360 sin desbordes; capturas 1440/390
```

Capturas: [1440 px](../capturas/usuarios-seg2-1440.png) y [390 px](../capturas/usuarios-seg2-390.png). Enlazadas también desde el README.

## Revisiones independientes y seguridad local

- Revisión independiente de SQL, permisos, identidad y fronteras HTTP con `revisar-cambios-etf`, `revisar-sql-etf` y `security-audit` focal: GO estático, sin defectos bloqueantes. La persona revisora leyó el código y las pruebas; no ejecutó los ensayos.
- Segunda pasada independiente de usabilidad con las skills de interfaz y pruebas: GO sobre código y capturas finales. La primera pasada detectó el título vacío por un catálogo anterior de la demo; quedó corregida la preparación del recorrido. No se ensayó un lector de pantalla en esta revisión.
- Textos de pantalla y documentación repasados con `humanizer`, en castellano claro y con traducción inglesa para la interfaz.
- Semgrep local, con métricas y comprobación de versión desactivadas: reglas de `internal/adaptadores/http/testdata/seguridad-identidad.yml` y `migraciones/actividad/semgrep-local.yaml`, sobre los once ficheros Go/TypeScript/JavaScript cambiados de implementación y pruebas. No se usaron servicios de análisis externos.

```text
Ran 9 rules on 11 files: 0 findings.
```

## Documentación y montaje

Actualizados R-10 en el informe, PRD, TRD, FLUJO_APP, ESQUEMA_BACKEND, PLAN_IMPLEMENTACION e INDICE; sólo los apartados afectados de los manuales de Administración y sistemas. Se reutiliza el módulo de Usuarios ya registrado en servidor y web. La pantalla pasa a `web/src/usuarios/pantalla.tsx`, con una reexportación compatible en su ubicación anterior. Los dos cambios de enganche en `scripts/ci/comprobar.sh` incorporan el ensayo SEG2 y el recorrido Chrome a la puerta completa.

## Puerta completa y cierre

`git merge main` terminó con `Already up to date.` antes y después de la puerta completa. No hubo una migración posterior de identidad que obligara a renumerar.

`make ci` completo terminó con código 0. Incluyó `go vet ./...`, `go test ./...`, pruebas de arquitectura, ensayos PostgreSQL, matriz HTTP/RLS INT5, recorrido SEG2 y recorridos de carga y actas. Su última línea fue:

```text
CI mínima completada.
```

`make demo-parar` terminó con código 0. No se hizo push ni se integró la rama en main.

Ficheros de implementación y pruebas:

- Identidad: `internal/dominio/identidad/perfiles.go`, `internal/aplicacion/identidad/perfiles.go` y `perfiles_test.go`, `internal/adaptadores/http/usuarios.go`, `internal/adaptadores/postgres/perfiles.go` y `perfiles_integracion_test.go`.
- Base de datos: `migraciones/identidad/0003_impedir_cambios_propios.sql`, `manifiesto.json` y `ensayar.py`.
- Interfaz y textos: `web/src/usuarios/pantalla.tsx`, reexportación `web/src/usuarios.tsx` e `idiomas/{es,en}/textos/usuarios.json`.
- Regresión y CI: `cmd/etf/perfiles_navegador_test.go`, `web/tests/perfiles.navegador.mjs`, `web/tests/recursos.test.mjs`, `scripts/recorridos/seg2.mjs` y `scripts/ci/comprobar.sh`.
- Documentos: los seis documentos de producto citados arriba, README, la fila R-10 del informe de seguridad, los apartados afectados de ambos manuales, esta evidencia y las dos capturas.
