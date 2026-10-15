# AUD1 · Exportación de auditoría

Administración funcional dispone de `/admin/auditoria` y de `POST /api/admin/auditoria/exportaciones`, con sesión y CSRF. Los demás perfiles reciben 403. El actor y sus perfiles proceden de la sesión, y las funciones SQL vuelven a exigir Administración. No se concede lectura directa del registro ni acceso a fichas.

La descarga contiene el registro filtrado en CSV o JSON, una prueba contigua en `cadena.json`, un manifiesto con huellas SHA-256 y las instrucciones de comprobación. Las fechas son obligatorias, incluyen ambos días UTC y admiten hasta 31 días. Los filtros opcionales son ámbito del evento, persona usuaria y expediente por identificador; la API admite también acción exacta. La prueba conserva los eventos intermedios, incluso los que no coinciden con los filtros. No contiene nombres ni contenido de expedientes; los identificadores y sus vínculos siguen siendo datos personales.

El servidor lee páginas de hasta 256 eventos, verifica cada enlace y escribe temporales privados. Límites: 100.000 eventos de prueba, 64 MiB de registro más prueba sin comprimir y 64 MiB de ZIP, 30 segundos de preparación, 30 segundos de escritura y dos plazas. Compose monta tmpfs privado de 320 MiB y limita el contenedor a 512 MiB. Confirma el acto antes de entregar bytes; los fallos se registran después de revertir la lectura. Cerrar la descarga o fallar elimina los archivos.

## Comprobación local

```sh
go run ./cmd/verificar-exportacion /ruta/autorizada/auditoria.zip
go run ./cmd/verificar-exportacion /ruta/autorizada/auditoria.zip ID_ANTERIOR HUELLA_ANTERIOR ID_FINAL HUELLA_FINAL
```

El comprobador no extrae archivos ni abre conexiones. Comprueba la cadena, las huellas y la selección completa dentro del tramo. Código 0 indica comprobación correcta; código 1 indica fallo. Sin puntos externos sólo acredita coherencia interna. Contrastar ambos puntos custodiados por separado acredita el tramo frente a esas referencias, no el resto de la cadena ni las horas técnicas de registro.

## Evidencia ejecutada

- `go vet ./...` y `go test ./...`: correctos. Incluyen permisos, CSRF, fallo de auditoría sin entrega, cadena alterada, selección incompleta, fechas, cancelación, bytes y concurrencia. La interrupción de la escritura HTTP queda auditada; si falla también ese apunte, se registra únicamente una clave técnica sin datos.
- `npm test`: 36 pruebas correctas. `npm run build`: correcto.
- `./scripts/ci/textos/comprobar.sh`: dos idiomas, 995 claves usadas, sin literales visibles.
- `python3 -B migraciones/auditoria/ensayar.py`: `ENSAYO-OK: limpia, repetida y actualización desde plataforma/0001`. PostgreSQL 18 desechable; 260 eventos, páginas, Administración, Coordinación, Profesional, Consulta y contexto vacío; RLS forzada, lectura directa prohibida y contexto sin filtrarse. También exportador y verificador completos.
- `ETF_DEMO_PUERTO=18141 make demo` y `node scripts/recorridos/auditoria.mjs http://127.0.0.1:18141`: `AUD1-CHROME-OK`. Chrome del sistema con datos inventados: Inés Moreno sin acceso (403), Elena Salas con CSV y JSON, CLI y puntos externos correctos e incorrectos, registro vacío, rango y filtros rechazados, CSRF, ayuda y teclado, error simulado de descarga y recuperación, axe sin incidencias del módulo, inglés y alto contraste. Sin desbordamiento a 1440, 1024, 390, 360 y 320 px, después de esperar al reflujo de UI1.
- `EXPLAIN (ANALYZE, BUFFERS)` en la demo inventada: índice `evento_ocurrido` para fecha sin coincidencias (0,043 ms) e índice `evento_pkey` para página de 256 filas (0,142 ms). No acredita tiempos de producción.
- Semgrep local con reglas de `scripts/ci/equipo/semgrep-local.yaml` y `scripts/pwa1/semgrep.yml`, métricas y comprobación de versión desactivadas: cuatro reglas, sin hallazgos. No se descarga configuración ni se envía código.

[Captura de escritorio, 1440 px](../capturas/auditoria-escritorio.png) · [Captura móvil, 390 px](../capturas/auditoria-movil.png).

Revisión independiente de la candidata `bd4910c`: GO de SQL y seguridad focal. Revisión con `revisar-cambios-etf`, `revisar-sql-etf` y `security-audit` focal: corregida la auditoría de intentos fallidos y el temporal de Compose. Segunda revisión de usabilidad con `programar-interfaz-etf` y `probar-recorridos-etf`: GO tras corregir Escape, reintento y errores de ambas fechas. La cuenta junto a iniciales identifica al personal y evita ambigüedad; Administración ya puede consultar Usuarios.

Puerta completa `make ci`: ejecutada dos veces; la segunda sobre `bd4910c`, correcta. Última línea real:

```text
CI mínima completada.
```

Demo detenida con `make demo-parar`; temporales propios eliminados al cerrar la entrega. Las capturas que la CI vuelve a generar de otros módulos se restituyen, conservando sólo las dos de AUD1.

## Alcance y límites

Se modificó sólo la sección «Registro de auditoría» del manual de Administración, como pidió el encargo. La tabla de acceso del mismo manual y su PDF de `main` todavía describen la pantalla como pendiente; su actualización queda para quien mantiene el manual. Los seis documentos de producto y el índice reflejan el comportamiento entregado. NVDA no está disponible en este puesto Linux; se acreditan teclado y axe, no conformidad completa con lector de pantalla. La conservación y el alcance de los vínculos personales siguen sujetos al DPD en la EIPD.


## Ficheros y composición

Código propio en `internal/dominio/auditoria/`, `internal/aplicacion/auditoria/`, `internal/puertos/salida/auditoria/`, los adaptadores de auditoría HTTP/PostgreSQL/ficheros, `cmd/etf/auditoria.go`, `cmd/verificar-exportacion/`, `migraciones/auditoria/0004` y `web/src/auditoria/`. Textos es/en en los catálogos existentes de auditoría.

Enganches comunes: una entrada en cada registro `cmd/etf/modulos.go` y `web/src/pantallas.ts`; configuración y composición predeterminada del servidor; autorización de dependencias del CLI en la prueba de arquitectura, conservando la prohibición de red y PostgreSQL; expectativas del nuevo menú en pruebas de sesión y pantallas; temporal y memoria de Compose. El recorrido propio vive en `scripts/recorridos/auditoria.mjs`. Producto, README, índice y la sección del manual se actualizan junto al código.
