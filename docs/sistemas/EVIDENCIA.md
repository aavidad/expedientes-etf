# F5.05: ensayo local de copia y restauración

Fecha: 14 de octubre de 2026. Rama: `f5.05-manual-sistemas`.
Datos: sólo las personas inventadas y los registros de la demostración de este
worktree. No se ha accedido a producción ni a otros proyectos Docker.

## Procedimiento

```sh
ETF_DEMO_PUERTO=18155 make demo
python3 -B scripts/ensayar-restauracion.py
```

El script deriva el proyecto del worktree y comprueba que Docker es local.
Con app y entrada paradas toma un volcado PostgreSQL 18 en formato personalizado
y un tar de los volúmenes de idiomas y temas. Borra la base `etf`, comprueba que
no existe y borra el contenido de ambos directorios. Comprueba que están vacíos
antes de extraer la copia en el mismo contenedor: crear otro contenedor con el
volumen vacío haría que Docker lo repoblara desde la imagen.

Crea una base nueva con el propietario original, restaura con errores fatales,
repone los permisos de conexión y compara los datos y secuencias. La comparación
normaliza únicamente comentarios y testigos aleatorios de pg_dump 18 y ordena
las líneas del volcado de datos. También compara nombres, tipos, propietarios,
permisos y sumas de contenido de todos los miembros del tar. Antes de copiar comprueba el recorrido completo con Chrome. Reinicia app y entrada
tras restaurar; comprueba siete accesos, perfiles, equipos y rechazo CSRF con curl
y repite el recorrido de subida del Anexo 3, aceptación y Acta Word con Chrome.
La base copiada contiene expedientes y emisiones creados por esos recorridos.

## Límites y fallos detectados

- El primer intento con `etf_copia` falló antes de borrar: `permission denied for table smi`.
  Su atributo BYPASSRLS no sustituye los permisos de lectura. El ensayo y el manual
  usan el operador `etf_instalador` por socket local. Completar los permisos del rol
  de copia requiere otra tarea SQL con revisión y ensayo; no se han cambiado aquí.
- Una comprobación inmediata tras sembrar la demo encontró HTTP 429. El ensayo
  detiene app antes de copiar; su arranque posterior reinicia el limitador en memoria.
- El primer borrado de paquetes creó otro contenedor para comprobarlos y Docker
  repobló el volumen con archivos de la imagen. Esa recuperación intermedia repuso
  sólo la base y dejó fuera las versiones publicadas del catálogo. El recorrido y
  la primera CI fallaron en firmantes (`catalogo.no_disponible`, HTTP 503).
  Se recuperaron los paquetes originales de la copia conservada y el recorrido
  completo volvió a pasar. El script final exige el recorrido antes de copiar
  y después de restaurar, además de las huellas, para detectar referencias rotas.
- En un intento intermedio tar restauró contenidos sin conservar todos los permisos.
  Se corrigió con extracción `-p` y comprobación de los permisos. La copia conservada
  permitió recuperar la base tras ese fallo. El ensayo final exige ambas comparaciones.
- Ensayo lógico en el mismo servidor y roles ya inicializados; no prueba pérdida de
  máquina, recuperación de roles globales, cifrado ni traslado fuera del servidor.
  La restauración corporativa sigue siendo requisito previo a los datos reales.

## Resultado final

Salida real del ensayo final actualizado (código de salida 0):

```text
RECORRIDO-OK: Chrome del sistema; Anexo 3 limpio; 6 menores; aceptación 04-03-26; Acta DOCX íntegra (129325 bytes); idempotencia y fecha conservadas.
COPIA-OK: PostgreSQL y paquetes; datos SHA256 3b68398daa65afccc734372f2bc7de2b1a894230ef5d1e49bf253131b47a5ab9
BORRADO-OK: base inexistente; idiomas y temas vacíos antes de extraer la copia
RESTAURACION-OK: datos, secuencias y paquetes idénticos
RECORRIDO-OK: Chrome del sistema; Anexo 3 limpio; 6 menores; aceptación 04-04-26; Acta DOCX íntegra (129325 bytes); idempotencia y fecha conservadas.
DEMO-OK: 7 usuarios únicos, 8 equipos, perfiles, equipos, privacidad y rechazo CSRF comprobados con curl.
F5.05-OK: copia, borrado, restauración, siete accesos y recorrido Chrome
```

Semgrep local, sin métricas ni comprobación remota de versión:

```sh
SEMGREP_SEND_METRICS=off semgrep --metrics=off --disable-version-check \
  --config scripts/recorridos/seguridad.yml \
  scripts/ensayar-restauracion.py despliegue/demo/recorrido.mjs
```

Las reglas locales comprueban ejecución por shell en Python y ejecución de textos,
persistencia de sesiones y omisión de TLS en JavaScript. No hay dependencias nuevas. Salida real:

```text
Ran 3 rules on 2 files: 0 findings.
```

La revisión independiente estática corrigió la espera de salud en instalación.
Detección local de secretos por el revisor: `no leaks found`. La revisión independiente del
script final, manual y evidencia dio GO tras las correcciones. No autoriza producción ni sustituye el ensayo corporativo.

CI completa ejecutada de nuevo tras recuperar los paquetes originales:

```sh
make ci
```

Código de salida 0. Incluye `go vet ./...`, `go test ./...`, pruebas y construcción
del frontend, textos, temas, secretos, migraciones PostgreSQL y permisos con Chrome.
Últimas líneas reales:

```text
F0.11-RLS-OK: 7 actores; 52 tablas pobladas; 364 comprobaciones de lectura; 780 escrituras denegadas.
F0.11-VISTAS-OK: 7 vistas; 49 lecturas con RLS.
F0.11-OK: 56.6 s; evidencia en bin/temporales

CI mínima completada.
```

Comprobación final de enlaces: `Markdown: 97 documentos, 356 enlaces locales,
7 enlaces externos sin consulta de red, 0 errores.`
`main` integrado en la rama antes de entregar, sin conflictos.
