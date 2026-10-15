# AN6 · Derivación a otro servicio

La ficha y Salidas/cierre de Coordinación permiten abrir el Anexo 6. Se guarda el destino de catálogo, o su nombre escrito cuando no hay opciones activas, y el motivo. La propuesta no confirma el envío: descargar el Word y confirmar la fecha real son acciones separadas. Se admiten varias salidas.

El Word usa Familia, adultos y menores vigentes; los datos borrados y las personas retiradas no se recuperan del anexo original. Los campos aún no recogidos conservan la lectura inicial. Los apartados 5 a 8 se completan fuera, según el contrato del Programa. Cada emisión conserva la versión de plantilla. Memoria y SPAF mantienen las reglas de población, propuestas protectoras y cierre; no se inventa un recuento propio de Anexos 6.

## Persistencia y permisos

`expediente/0029`, reservada en el registro externo de ETF, añade el destino escrito con una restricción exclusiva frente al destino de catálogo. Conserva la RLS, el equipo inmutable y los permisos de `0006`. El destino no admite edición al confirmar. La aplicación sólo permite texto cuando no hay opciones activas. Se conservan auditoría transaccional, versiones e idempotencia.

Ensayo real: `python3 -B migraciones/documento/ensayar.py`, PostgreSQL 18 desechable, base limpia y actualización desde versión anterior. Salida:

```text
AN6-FICHA-OK: dirección y colección de menores vigentes; emisión sin alterar el hito
AN6-VACIOS-OK: menores retirados no se recuperan del anexo original
--- PASS: TestAnexo6PostgresHTTPDescargaHitosPermisos
ENSAYO-OK base limpia, RLS, auditoría, concurrencia e integridad
```

Las pruebas cubren descargas es/en, 1/6/15 menores, fecha de Madrid al cambiar el día, rechazo de fecha futura, cambio de ficha, menores retirados, recibos repetidos, conflictos, Coordinación, equipo ajeno, Consulta y Administración, RLS directa y reversión ante fallo de auditoría.

Revisión SQL y `security-audit` focal independientes, sólo fuente: GO acotado, sin vulnerabilidad concreta. Se corrigieron los hallazgos de fecha UTC, contactos de adultos retirados y lista de menores vacía. Semgrep local usa `scripts/ci/equipo/semgrep-local.yaml` y `migraciones/documento/semgrep-local.yaml`; no se envía código a servicios externos.

## Pantalla y recorrido

`make demo`, puerto local 18246, cuentas inventadas de Coordinación y del equipo. `node scripts/recorridos/anexo6.mjs http://127.0.0.1:18246` automatiza Chrome del sistema. El mismo recorrido forma parte de `make ci`, incluida la recuperación de una petición al cambiar de idioma. Un retoque del enganche común `web/src/main.tsx` conserva la pantalla de Anexo 6 montada mientras se cargan los textos. En ese recorrido, el cambio de idioma mantiene formularios y recibos pendientes sin pedir que los descartes. Resultado real:

```text
AN6-CHROME-OK: propuesta, Word versionado, hito separado, conflicto, recuperación sin duplicados, perfiles, Memoria/SPAF conservadas, 1440/1024/390/360/320, contraste e inglés
```

La cuenta de Coordinación realiza el flujo por pantalla. La cuenta del equipo repite preparación, emisión y confirmación por las API reales; el equipo ajeno, Consulta y Administración reciben denegación. La comparación estadística usa el mes completo de septiembre de 2026 y la Memoria de 2026; confirmar una salida fechada en septiembre conserva sus cifras. Axe se ejecuta localmente sobre la pantalla; el diálogo se prueba con Escape y devolución del foco.

Capturas: [1440 px](../capturas/anexo6-1440.png), [390 px](../capturas/anexo6-390.png), [alto contraste](../capturas/anexo6-contraste-390.png), [inglés](../capturas/anexo6-en-390.png).

Revisión de usabilidad independiente por fuente: se corrigió la versión de plantilla atribuida a otra salida, la competencia entre acciones principales y los bloqueos cruzados entre formularios. La propuesta y la revisión se muestran por pasos. El título del navegador incorpora el número del expediente. La ayuda contempla tanto los destinos publicados como el destino escrito. Las cuatro capturas recibieron una segunda revisión independiente, sin cortes ni desbordes visibles.

No se ha probado NVDA ni el comprobador de accesibilidad de Word de un procesador de textos. Las capturas y las pruebas automáticas no acreditan por sí solas conformidad completa. D25 y D26 recogen las consultas pendientes al Programa sobre recuento propio y salida durante espera del SPM.

## Entorno de la puerta completa

Main se integra antes de la comprobación final. Se conserva el manual actualizado del Programa y se regenera sólo `profesionales.pdf` con LibreOffice local. El ensayo de incorporación de demo pasó con temporales propios (`TMPDIR=/dev/shm/etf/tmp/an6-ci`), después de que una primera ejecución fallase al reiniciar su copia temporal y no pudiera completar su limpieza. No se atribuye ese fallo al código de AN6 ni se presenta como una prueba pasada.

La puerta incluye las dos rutas nuevas en la expectativa independiente de permisos. AN6 se ejecuta al final, después de una ventana sin peticiones, para separar las ráfagas de las suites de navegador y conservar los límites de admisión por IP. El recorrido RESP4 pasó de nuevo aislado después de un timeout en la ejecución conjunta; no se modifica su comportamiento ni se rebajan los límites de la aplicación.

## Resultado final

`TMPDIR=/dev/shm/etf/tmp/an6-ci make ci` completó toda la puerta tras integrar main, incluidos `go vet ./...`, `go test ./...`, 36 pruebas del frontend, ensayos PostgreSQL y recorridos reales. Última línea:

```text
CI mínima completada.
```

Chrome ejecutó también el cambio de idioma durante un POST cuya escritura ya se había confirmado, con respuesta retenida: se recuperaron el cuerpo y la clave originales sin duplicar la salida. Axe no encontró infracciones A/AA en el contenido de la pantalla. Semgrep local examinó 21 fuentes y devolvió `0 findings`; las revisiones independientes de SQL, seguridad focal y usabilidad quedaron sin hallazgos abiertos del alcance.
