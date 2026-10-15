# RESP1 · respuestas del Programa del 14/10/2026

Estado: completado; CI completa verde, demo parada y revisiones independientes con GO. Reservas confirmadas en el registro de reservas de migraciones, fuera de Git: Catálogo 0014, Expediente 0027, Estadística 0007 y Documento 0016.

La fuente es el documento de respuestas del Programa del 14/10/2026, que no se distribuye, leído con Python `zipfile` y XML. Las respuestas se recogen en el registro de dudas del proyecto. RESP1 conserva D23 y D24 pendientes. D17 y D21 reciben los cambios de RESP3 y RESP2 al integrar main; RESP1 no modifica sus apartados de dudas.md.

## Comprobaciones

| Punto | Resultado y evidencia |
| --- | --- |
| D2a | El servidor ya rechaza espera para Anexo 4 y la pantalla oculta la acción. Comprobado con petición real y Chrome. |
| D2b | `TestDecisionEsperaAceptacionFechaRealYNumeracion` pasó en el ensayo PostgreSQL de Expediente. Conserva fecha civil del reloj y correlativo del equipo/año de aceptación. |
| D2c | Regla de dominio y proyección persistente con RLS, corte y conocimiento comprobados. La espera cuenta por equipo y provincia sin número ni aceptación. El rol técnico de Consulta recibe la proyección sin nombres. Los avisos incluyen entrar en espera y aceptar en un equipo distinto. |
| D8 | Datos de 2026 en `datos/catalogos/smi.json`; siembra de demostración por API sin sustituir cambios de Coordinación. Prueba con dos años: el mismo ingreso de 17.500 € cambia de tramo al cambiar el SMI del año de Memoria. `catalogo/0014` carga el importe inicial y respeta una edición posterior de Coordinación. |
| D10 | Chrome confirmó y aceptó una derivación de municipio fuera del ámbito con centro y equipo elegidos. Mensajes es/en explican que sigue el mismo trámite. |
| D18 | Diez versiones Word nuevas es/en. Prueba de marcadores y generación real; comparación independiente de XML: mismas partes, elementos y atributos. Referencias antiguas ausentes. `documento/0016` registra y activa las referencias iniciales intactas; preserva publicaciones y restauraciones administrativas. |
| D19 | ZTS retirada del contrato SPAF, pantalla, aviso y relleno Word. El lector obtiene el municipio del expediente y resuelve su código mediante el puerto de Catálogo en una consulta por lote. Regresión con Íllora, centro Pinos Puente-Moclín y ETF 6; pruebas Word es/en y capturas 1440/390. |
| D20 / D22 | Catálogo de tramos de la administración competente comprobado; pruebas existentes de meses completos y nota 43 verdes. |
| D1, D3, D4, D7, D9, D11, D12, D13, D14, D15 | Revisión independiente del código y pruebas existentes. Añadidas regresiones para abuelos, parejas y tutores y para la asignación independiente de Íllora. `go test ./...` verde; el ensayo de Expediente pasó las pruebas de decisión y traslado junto con las regresiones de espera y avisos. |

## Salidas reales

`go vet ./...`: salida vacía, código 0. `go test ./...`: código 0.

Pruebas focales:

```text
ok github.com/aavidad/expedientes-etf/internal/dominio/catalogo 0.006s
ok github.com/aavidad/expedientes-etf/internal/dominio/estadistica/pruebas 0.007s
ok github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros/estadistica 0.002s
ok github.com/aavidad/expedientes-etf/internal/adaptadores/docx 3.536s
ok github.com/aavidad/expedientes-etf/internal/aplicacion/documento 0.524s
```

`node --test web/tests/decision.test.mjs web/tests/spaf.test.mjs`: 8 pruebas, 8 aprobadas, 0 fallidas.

```text
RESP1-UI-OK: SMI 2026, SPM sin espera, espera en Memoria y aceptación real, municipio fuera de ámbito; SPAF y Word es/en sin ZTS ni anexos antiguos; 1440/390.
DEMO-PARADA
```

`gopls check` de los archivos Go de producción cambiados: salida vacía, código 0. Semgrep: `Ran 16 rules on 44 files: 0 findings.` Se ejecutó con reglas locales, métricas y comprobación de versiones desactivadas; sin envío de código. Revisión focal independiente de seguridad: sin defecto confirmado en lo revisado. Segunda revisión visual: GO para SPM y SPAF en escritorio y móvil. Las capturas están enlazadas en README.

## Ensayos SQL y revisión independiente

Las cuatro migraciones se ensayaron con PostgreSQL 18 en contenedores desechables, desde instalación limpia y versiones anteriores. No se usaron datos reales ni bases de producción. El ensayo de actualización incorpora una espera abierta y otra terminada antes de RESP1; reconstruye sus fechas y equipo, conserva un SMI editado y una versión de plantilla publicada por Administración. También comprueba el índice con EXPLAIN.

```text
RESP1-UPGRADE-OK: espera previa reconstruida, SMI editado y plantilla administrativa conservados
PLV-RESP1-UPGRADE-OK: nuevas referencias, restauración y versión administrativa conservadas
RESP1-ESPERA-OK: apartado 6, equipo/provincia, RLS, corte y conocimiento; sin aceptación ni número
RESP1-AVISOS-OK: cierre→espera→aviso y aceptación en otro equipo avisa origen, destino y provincia
RESP1-CONSULTA-OK: rol técnico, contexto y equipo; espera sin nombres ni contactos
RESP1-MUNICIPIO-OK: Íllora, centro Pinos Puente-Moclín y ETF 6; SPAF conserva sólo código municipal
```

La revisión SQL y de seguridad independiente dio GO. La segunda revisión tras integrar `819dac9` detectó la dependencia de Expediente 0027 respecto de 0026; se corrigió y obtuvo GO. La revisión final de `35eddb8` verificó también Documento 0013→0014→0015→0016, las referencias separadas de RESP1/RESP3 y el inventario de 72 Word. La matriz general de permisos cuenta sólo filas de espera con fecha de entrada para ETF, conforme a su RLS. Detectó y se corrigieron dos faltas de avisos: la entrada en espera y el equipo anterior cuando la aceptación cambia el equipo. Los triggers de la nueva proyección toman sólo tipo y equipo desde los datos confirmados; no releen el anexo bajo permisos de propietario. Las filas preparadas sin fecha de espera quedan ocultas a ETF y Consulta. Las políticas temporales del backfill se retiran en la misma transacción.

Los cortes históricos de los ensayos retiran las dependencias transitivas de las migraciones excluidas. Esto evita que el corte anterior a Expediente 27 conserve Estadística 7. La prueba de arranque confirmó después la carrera del servidor provisional y la espera TCP autenticada.

```text
ARRANQUE-OK espera TCP autenticada; versión anterior y actualización
```

## Validación final

`make ci` completa terminó con código 0 sobre `main` integrado hasta `bc68978`. Última línea real:

```text
CI mínima completada.
```

La demo se paró con `make demo-parar` (`DEMO-PARADA`). Las capturas ajenas a RESP1 que regenera CI se restauraron a la versión integrada. Las preguntas de uso de motivos de D5 y de numeración de actas de D18 siguen en dudas.md. D17 y D21 se incorporan desde otras ramas. D23/D24 siguen pendientes.

`main` integrado hasta `bc68978`, conservando ODT, carga desde actas, paquete Windows y escalas opcionales. La reserva Expediente 0024 se sustituyó por 0027; depende de 0026 para conservar el orden del migrador al reanudar. Documento 0016 depende de 0015 por el mismo motivo. Sin push.
