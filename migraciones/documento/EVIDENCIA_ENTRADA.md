# Evidencia de F1.07a y F1.08

Alcance: Acta de Aceptación 1.1.0, oficios de aceptación a SS.SS.CC. y al SPM
1.0.0, y anotación manual de registro de salida. No Aceptación y Lista de Espera
están convertidas, con copia idéntica en ambos idiomas, pero siguen inactivas.
No se han cambiado ficheros existentes del módulo expediente.

Las pruebas usan datos inventados. La matriz de emisiones cubre tres tipos,
1, 6 y 15 menores y los dos idiomas, con menores presentes exactamente una vez,
cinco adultos, firmante y destinatario, referencias antiguas, huellas y repetición
binaria. Las otras dos plantillas se rellenan con esos tamaños sin activar tipos.
No se deducen fechas de nacimiento a partir de edades.

La prueba activable de LibreOffice abrió los quince documentos en castellano
(cinco plantillas por tres tamaños) y produjo PDF válido en cada caso. El Acta
con quince menores tiene cuatro páginas A4 y PDF etiquetado; se revisaron
visualmente la primera página y el texto extraído. El fichero conservado es
`/tmp/etf-f107-evidencia/entrada_aceptacion15.pdf`, fuera del repositorio.
La salida completa está en `/tmp/etf-f107-evidencia/libreoffice.log`.

El ensayo desechable de PostgreSQL 18 aplica primero la versión anterior,
luego documento/0002, repite el migrador y aplica todo en base limpia. Sus pruebas
usan el puerto real de expediente en la transacción de emisión: doce emisiones
(Acta y oficio de cada origen para 1, 6 y 15 menores), reproducciones idénticas,
selección del destinatario SPM, control del origen y versiones. Registro de salida
por HTTP devuelve 200; misma clave alterada, 422; versión antigua, 409; sin CSRF,
403. Un fallo de auditoría revierte anotación y clave. RLS niega registros a
Administración, Consulta y Equipo. Se conserva el estado del expediente.

Tras publicar una segunda versión del firmante en el catálogo sintético,
la clave antigua se rechaza y una nueva emisión contiene el nombre nuevo
con aviso de regeneración. La versión anterior permanece en catálogo.
La salida completa está en `/tmp/etf-f107-evidencia/postgres.log`.

La lectura de presentación se prueba con un paquete de idioma activo distinto
de distribución y una referencia independiente de catálogo; se rechazan huellas
alteradas y rutas fuera de la raíz. La revisión de seguridad es local y focal;
no hay envío de código o datos a un servicio externo.

Las pantallas de descarga y anotación están pendientes del recorrido de
interfaz. El conector MOAD y las emisiones de No Aceptación y Lista de Espera
no se implementan en este alcance. La decisión vigente de sólo registrar oficios
corrige la mención anterior a registro de Actas en FLUJO_APP §3.17.

Revisión independiente focal: GO sobre la candidata `316d854`, con SQL,
plantillas y la integración de autorización de INT4 revisados. Semgrep local
(reglas de documento, métricas desactivadas): `Ran 5 rules on 24 files: 0 findings.`
Las pruebas con detector de carreras terminaron correctamente:

```text
ok  github.com/aavidad/expedientes-etf/internal/aplicacion/documento 2.933s
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/http 1.988s
ok  github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/documento 1.019s
```

Huella SHA-256 del PDF conservado:
`44baa730d8aa636ee3ecc3915aefb73656f0cba9e505840560ec67d9992bac68`.
La CI completa se conserva en `/tmp/etf-f107-evidencia/ci.log`.

La puerta completa terminó correctamente con las dependencias fijadas de la
caché local. Comando y última línea real:

```text
npm_config_offline=true make ci
CI mínima completada.
```

Incluye `go vet ./...`, `go test ./...`, frontend, textos, temas, secretos,
arquitectura y ambos ensayos PostgreSQL. Se corrigió una consulta SQL que
necesitaba la marca técnica del comprobador de textos. La instalación de npm
con red se detuvo al no progresar; la ejecución completa que pasó fue offline.
