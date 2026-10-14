# Evidencia F0.12 · 13 de octubre de 2026

Rama `f0.12-catalogo-bd`. Alcance y uso en [LEEME.md](LEEME.md).
La revisión independiente de SQL y seguridad focal dio GO tras corregir la
carga de listas y reservar la lectura de borradores a Administración.

`python3 -B migraciones/catalogo/ensayar.py` terminó con código 0 sobre la
imagen PostgreSQL 18 de Compose. Se ejecutó `cmd/migrar` del árbol. Salida
relevante:

```text
migracion.aplicada.plataforma.0001
migracion.aplicada.catalogo.0001
migracion.aplicada.catalogo.0002
migracion.existente.plataforma.0001
migracion.existente.catalogo.0001
migracion.existente.catalogo.0002
ENSAYO-OK migración limpia y repetida
LISTAS-OK 24 opciones del PRD; códigos sin etiquetas en BD
CARGA-OK 8 equipos, 14 centros, 179 municipios; Íllora y ELA; RLS por perfil
PERMISOS-OK app sólo SELECT, RLS forzada, sin CREATE ni publicación
VIGENCIAS-OK cierre controlado, historia, frontera 01/07, regla 31/12 e inmutabilidad
BORRADORES-OK sólo Administración ve versión y opciones sin publicar
HTTP-OK servidor real: 403 por defecto; desarrollo consulta PostgreSQL y regla 31/12
PASS
ENSAYO-OK actualización desde plataforma/0001
Index Scan using version_ambito_daterange_excl on version
ENSAYO-OK contenedor y volumen destruidos
```

También pasaron las pruebas de la base de transacciones: reutilización de
conexión sin contexto filtrado, confirmación, reversión, pánico, error SQL y
cancelación. Las pruebas de integración sí se ejecutaron en este ensayo;
sin su variable de activación se omiten en `go test ./...`.

`go vet ./...` terminó con código 0. `go test ./...` terminó con código 0,
incluidas las pruebas de arquitectura y las unitarias de catálogo, aplicación
y HTTP. No se añadieron dependencias Go ni npm.

Las comprobaciones de fuentes y documentación devolvieron:

```text
CATALOGOS_OK
CARGA-OK instantánea coincide con los cuatro JSON; etiquetas es/en completas
INDICE_OK
Markdown: 43 documentos, 61 enlaces locales, 7 enlaces externos sin consulta de red, 0 errores.
```

Semgrep se ejecutó localmente, con métricas y comprobación de versión
desactivadas, usando `despliegue/base/semgrep.yaml`: cuatro reglas, 23 ficheros,
ningún hallazgo. Su alcance son esas reglas y la revisión manual focal;
no constituye una auditoría completa. Gitleaks sobre el cambio preparado para
commit terminó con código 0 y `no leaks found`.

## Comprobación global de textos pendiente

`./scripts/ci/textos/comprobar.sh` falla en el caso «admite claves existentes y
cadenas técnicas». La comprobación directa de idiomas también falla:

```text
BASE/idiomas/es/catalogos/ubicaciones.json:1: Clave repetida o valor inválido: equipos
BASE/idiomas/es/catalogos/ubicaciones.json:1: Clave repetida o valor inválido: centros
BASE/idiomas/es/catalogos/ubicaciones.json:1: Clave repetida o valor inválido: municipios
en:1: Claves diferentes en catalogos
```

Se reprodujeron los mismos errores con los ficheros de `HEAD`, extraídos a un
directorio temporal. El comprobador espera valores planos, mientras que el
catálogo previo de ubicaciones usa objetos anidados. El análisis Go también
señala literales técnicos previos de DOCX y del ejecutor de migraciones.
No se modificaron esos módulos ni el comprobador en F0.12. Las nuevas listas
son planas, sus 24 claves coinciden en español e inglés y la comprobación de
carga verifica esa correspondencia.

Por este fallo previo no se declara la CI completa aprobada. F0.10 debe
integrar autorización antes de abrir la consulta; F0.14 debe registrar las
lecturas en la auditoría encadenada. La edición y las fuentes de las listas
restantes quedan fuera de esta entrega.
