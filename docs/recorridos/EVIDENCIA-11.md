# Revisión de «Del Anexo a la Memoria»

El vídeo del 14 de octubre de 2026 dura **11 minutos y 54,80 segundos**. Sigue el expediente inventado **02-06-26** de la familia Ortega Salcedo, de Agrón, desde la recepción del Anexo 3 hasta el cierre de las revisiones de Memoria del equipo y provincial.

## Grabación que puedes reproducir

MP4 y comprobaciones fuera de Git:

```text
$ETF_VIDEOS_SALIDA/11-ciclo-completo-20261014-090023-908885/video.mp4
$ETF_VIDEOS_SALIDA/11-ciclo-completo-20261014-090023-908885/hoja-control.png
$ETF_VIDEOS_SALIDA/11-ciclo-completo-20261014-090023-908885/revision-f507/
```

Repite desde la raíz del worktree con `make video RECORRIDO=11-ciclo-completo`. Usa el [guion 11](guiones/11-ciclo-completo.md) y las cuentas inventadas de esa demo. Cada ejecución conserva un expediente nuevo, sus documentos y las revisiones de Memoria; el correlativo cambia. La grabación se hizo después de integrar `main` en `bca07e8`. Después se integró también conservación (`7b05b44`) y se repitieron los 31 pasos y la comprobación del aula contra esa versión, con siete Word y ambas Memorias cerradas. Esa comprobación rápida está en `$ETF_VIDEOS_SALIDA/revision-rapida-11-/20261014-100424-805233/`; no sustituye al MP4 narrado indicado arriba.

El montaje mantiene las pantallas reales, el ratón y la escritura. Sólo retira pausas de imagen; cuando una frase necesita más tiempo conserva la última pantalla. El título ocupa 3,5 segundos, la introducción completa 12,56 y el cierre 5,28. El MP4 contiene vídeo H.264 de 1440 × 900 y audio AAC con voz castellana es-ES-AlvaroNeural.

## Capítulos del MP4

| Capítulo | Inicio | Final |
| --- | --- | --- |
| Derivación y aceptación | 00:00.00 | 02:14.44 |
| Ficha y consentimiento | 02:14.44 | 05:15.24 |
| Comisiones e informes | 05:15.24 | 06:03.52 |
| Tabla SPAF del mes | 06:03.52 | 06:51.08 |
| Propuesta y cierre | 06:51.08 | 09:12.84 |
| Memoria anual | 09:12.84 | 11:54.80 |

## Comprobaciones del caso

La recepción quedó registrada el 14 de octubre a las 10:00 UTC y el Acta de Aceptación tiene fecha del mismo día. El Anexo 1 se generó el 15 y la firma se confirmó el 16. La evaluación inicial corresponde al 17, la comisión al 22, el informe al 24 y el cierre al 31. Después, el calendario avanza al 1 de noviembre para consultar octubre terminado y revisar/cerrar la Memoria.

La tabla de septiembre no incluye este número, recibido en octubre. La de octubre contiene el mismo número como alta y baja. Los totales incluyen otros casos inventados conservados en la demo. Comisiones e informes pertenecen al equipo y no tienen vínculo individual con el expediente.

Se comprobaron 31 pasos, 17 escrituras confirmadas y siete Word íntegros: Acta de Aceptación, Anexo 1, tabla de septiembre, Anexo 11, Acta de Cierre, tabla de octubre y Memoria del equipo. Las dos Actas, el Anexo 11 y la tabla de octubre conservan el número del caso. El consentimiento firmado, el instrumento, la actividad y las dos instantáneas de Memoria se comprobaron mediante lecturas de la API local. Los Word y `estado.json` permanecen en la carpeta del vídeo.

El calendario de formación está explicado en el guion y en [preparar el aula](../formacion/preparar-aula.md). Sólo existe en una compilación local mediante overlay: relojes de negocio, fecha civil de consultas de vigencia y proyección de la publicación del catálogo inicial. Conserva las reglas, los filtros de acceso y las publicaciones posteriores. No modifica fuentes de producción ni versiones del catálogo en la base. La sesión mantiene el reloj real. Al terminar se verificó la compilación ordinaria y la retirada del tmpfs de formación.

## Revisión de fotogramas

Se extrajeron y revisaron **45 imágenes del MP4 terminado**: introducción en los segundos 1, 2, 3, 6, 10 y 12; mitad de los 31 pasos; resultados de aceptación, firma, propuesta, cierre, Acta/tabla, cierre de equipo y cierre provincial; mitad del cierre del vídeo. Las cuatro hojas `revision-f507/hoja-1.png` a `hoja-4.png` permiten revisarlas juntas; `tiempos.json` conserva los segundos exactos y `ffprobe.json`, los capítulos y formatos.

El título y los rótulos se leen completos. La introducción presenta capturas reales con movimiento. El ratón se ve en los controles y las frases corresponden a las acciones mostradas. Los resultados conservan el número y las fechas del caso; las dos Memorias muestran su ámbito y estado cerrado. El cierre del vídeo mantiene la última pantalla difuminada.

| Paso | Fotograma central |
| --- | --- |
| Coordinación recibe la derivación del caso (`bandeja`) | 00:22.22 |
| Sube el Anexo tres recibido (`subir`) | 00:41.22 |
| Revisa ubicación, familia y menores (`revisar`) | 01:01.00 |
| Crea la derivación tras revisar (`crear`) | 01:21.86 |
| Acepta y conserva el número (`aceptar`) | 01:42.32 |
| Descarga el Acta de Aceptación (`acta`) | 02:03.40 |
| El equipo abre su expediente (`ficha`) | 02:24.92 |
| Completa y guarda los datos familiares (`familia`) | 02:46.62 |
| Revisa las personas adultas del caso (`adultos`) | 03:13.50 |
| Comprueba el nacimiento de la menor (`menores`) | 03:39.60 |
| Registra la fase de intervención actual (`fase`) | 04:01.20 |
| Descarga el consentimiento sin confirmar firma (`consentimiento`) | 04:22.92 |
| Confirma la fecha del consentimiento firmado (`firma`) | 04:43.80 |
| Registra el instrumento aplicado a Lucía (`evaluacion`) | 05:04.68 |
| Anota la comisión técnica del equipo (`comision`) | 05:28.84 |
| Registra el informe y su destinatario (`informe`) | 05:52.98 |
| Consulta el mes anterior al caso (`spaf`) | 06:16.16 |
| Descarga la Tabla mensual en Word (`wordspaf`) | 06:39.94 |
| Prepara la finalización con sus objetivos (`finalizacion`) | 07:08.90 |
| Descarga el Anexo once de finalización (`anexo`) | 07:37.56 |
| Propón el cierre a Coordinación (`propuesta`) | 07:59.80 |
| Coordinación revisa la propuesta del equipo (`revision`) | 08:20.30 |
| Confirma la fecha de cierre acordada (`cierre`) | 08:40.06 |
| Descarga el Acta y conserva el cierre (`actacierre`) | 09:01.78 |
| El equipo consulta la Memoria anual (`memoria`) | 09:22.68 |
| Revisa los apartados y sus avisos (`apartados`) | 09:43.22 |
| Guarda la metodología del trabajo anual (`narrativa`) | 10:10.06 |
| Descarga la Memoria del equipo en Word (`wordmemoria`) | 10:36.92 |
| Conserva la revisión cerrada del equipo (`cerrarequipo`) | 10:58.36 |
| Coordinación consulta la Memoria provincial completa (`provincial`) | 11:19.00 |
| Cierra y conserva la revisión provincial (`cerrarprovincia`) | 11:39.22 |

## Resultado y límites

Salida real del recorrido y de sus comprobaciones:

```text
CICLO-VIDEO-OK: 31 pasos; un expediente; consentimiento, actividad, alta y baja; siete Word; Memorias cerradas
FOTOGRAMAS-OK: 45 imágenes; 714.8 s; seis capítulos; H.264/AAC; título 3.5 s; intro 12.56 s; cierre 5.28 s
RELOJ-ORDINARIO-OK: contenedor sin tmpfs de formación
```

La revisión independiente dio conformidad al montaje, los 45 fotogramas, la cronología y el aislamiento del calendario; comprobó también la restauración y no dejó hallazgos abiertos. Gitleaks local terminó con código 0. El lanzador del aula se ensayó con Chrome visible mediante Xvfb: fecha 01/11/2026, mes permitido octubre, consulta real, tres sesiones separadas, temporizadores reales y Administración con fecha ordinaria. Así los formularios admiten las fechas de práctica y los permisos se conceden en tiempo real. `make ci` completo pasó: `CI mínima completada.` Las 17 pruebas de la herramienta de vídeo terminaron con `OK`; `go test ./scripts/recorridos` comprueba que el Anexo del aula mantiene una plantilla reconocida, Agrón y los seis menores. Se conservan los rótulos de la plantilla original; sólo se adaptan los datos del caso. Semgrep local ejecutó tres reglas sobre ocho ficheros sin hallazgos.

La demostración registra un instrumento sin cuestionario ni baremos oficiales. Los avisos pendientes siguen visibles. Cerrar una revisión no acredita la validación del cálculo por el Programa. El material de [formación](../formacion/LEEME.md) está preparado; la sesión queda pendiente de impartir y recoger las dudas de quienes asistan.
