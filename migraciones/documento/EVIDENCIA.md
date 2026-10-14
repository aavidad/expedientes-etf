# F1.06 · evidencia de entrega

13/10/2026 · rama `f1.06-emision-word` · candidata de código `bbbd872`.

La emisión rellena el Acta de Aceptación en memoria, registra huellas y
referencias de versiones, confirma auditoría y entrega el DOCX. Conserva
metadatos; los datos de relleno y el Word no se guardan. El endpoint utiliza
sesión y CSRF. La composición queda preparada para el lector `FuenteDatos`
del módulo expediente: mientras falta ese proveedor, responde 503. Las pruebas
completas usan datos inventados suministrados por ese contrato.

## Pruebas ejecutadas

`python3 -B migraciones/documento/ensayar.py`, PostgreSQL 18.4 en contenedor
temporal sin red, con volumen de datos en memoria y repositorio de sólo lectura:

```text
ENSAYO-OK versión anterior, migración nueva y repetición
POST 201 y repetición 200: DOCX idéntico; 6 peticiones concurrentes; RLS; auditoría atómica; expediente intacto; regeneración con aviso
--- PASS: TestIntegracionEmisionConcurrenteRLSAuditoriaYRegeneracion (0.13s)
PASS
ENSAYO-OK base limpia, RLS, auditoría, concurrencia e integridad
```

La misma clave con contenido cambiado devuelve 422. Una clave nueva crea otra
emisión y su aviso. El fallo de auditoría revierte emisión y clave; la lectura
por perfiles sin permiso devuelve cero filas. La prueba compara estado,
versión y fecha de recepción del expediente antes y después. Se rechazan una
pertenencia falsa y el intento de modificar una emisión. El ensayo elimina
su contenedor y sus temporales.

`ETF_DOCUMENTO_PROBAR_LIBREOFFICE=1 go test -count=1 -v ./internal/aplicacion/documento`:

```text
--- PASS: TestEmisionNoEscribeFicheros (0.00s)
--- PASS: TestEmisionLibreOfficeLocal (0.61s)
PASS
ok  github.com/aavidad/expedientes-etf/internal/aplicacion/documento 0.673s
```

LibreOffice abrió el DOCX emitido y lo convirtió a PDF en un temporal de la
prueba. Se comprobaron la ausencia de marcadores pendientes y el firmante del
catálogo. No acredita una revisión visual o de accesibilidad del documento.
La prueba de disco compara los archivos antes y después de emitir, con el
directorio de plantilla sin permiso de escritura.

`go test -race ./internal/aplicacion/documento ./internal/adaptadores/http ./internal/adaptadores/ficheros/documento`:

```text
ok  github.com/aavidad/expedientes-etf/internal/aplicacion/documento 1.461s
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/http 1.578s
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento 1.016s
```

`make ci` completo, incluidos textos es/en, temas, secretos, frontend,
arquitectura, `go vet ./...`, `go test ./...` y compilación:

```text
CI mínima completada.
```

`gopls check` sobre dominio, aplicación, repositorio y HTTP: código 0, sin
diagnósticos. No se añadieron dependencias.

## Revisión local e independiente

Semgrep, con `migraciones/documento/semgrep-local.yaml`, métricas y consulta
de versiones desactivadas, sobre los módulos nuevos y HTTP de documentos:

```text
Ran 5 rules on 13 files: 0 findings.
```

Revisión independiente de `bbbd872` con `revisar-cambios-etf`,
`revisar-sql-etf` y `security-audit` en modo focal: **GO**, sin bloqueantes
detectados. El revisor examinó código, SQL, permisos, sesión, CSRF,
idempotencia, lector y pruebas; comprobó la evidencia del ensayo sin repetirlo.
Gitleaks sobre la candidata: `no leaks found`. No se ejecutó una auditoría
completa ni se enviaron código o datos a servicios de análisis externos.

## Integración

Enganches comunes: `cmd/etf/main.go` y `internal/adaptadores/http/servidor.go`.
Las pruebas del motor y del instalador de paquetes ahora leen la plantilla
publicada en `idiomas/`. La conversión de preparación conserva una salida
separada para revisar futuras versiones sin sobrescribir la publicada.

Documentos de producto actualizados: nota de F1.06 en `ESQUEMA_BACKEND.md`
y rangos regenerados de `INDICE.md`. La migración `documento/0001` quedó
reservada en el registro externo de ETF. La pertenencia se valida mediante
disparador mientras expediente no publique la clave compuesta definitiva;
no se cambiaron sus tablas. Plantillas es/en idénticas, como pidió el encargo.
No se publicaron versiones administrativas nuevas ni se implementó el registro
de salida de oficios. No se hizo push ni fusión.
