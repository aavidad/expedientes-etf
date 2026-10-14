# Comisiones e informes técnicos · F2.04

Son registros del equipo, sin vínculo a un expediente. Cada informe registrado
representa un informe ya emitido. El profesional registra, corrige y retira los
de su equipo; Coordinación consulta todos. Consulta y Administración no acceden
a estos registros. No se incluyen pantallas en esta entrega.

| Operación | API | Datos |
| --- | --- | --- |
| Registrar comisión | `POST /api/comisiones` | `fecha`, `observaciones` opcional; `version_esperada` ausente o 0 |
| Registrar informe emitido | `POST /api/informes` | `fecha`, `organismo`; `organismo_otros` obligatorio para `OTROS` |
| Consultar | `GET /api/comisiones?anio=2026` y `/api/informes?anio=2026` | `etf` opcional, UUID de equipo; profesional sólo su equipo |
| Corregir | `PUT /api/comisiones/{id}` y `/api/informes/{id}` | Datos completos y `version_esperada` positiva |
| Retirar | `DELETE /api/comisiones/{id}?version_esperada=1` y equivalente para informes | Retirada lógica; devuelve registro y versión nueva |
| Catálogo inicial | `GET /api/actividad/organismos` | Ocho códigos, claves de texto, indicador de texto y versión 1 |

Todas las rutas exigen sesión y perfil autorizado. Las escrituras exigen CSRF.
El equipo de una escritura se toma de la sesión, nunca del cuerpo. Fecha en
formato `AAAA-MM-DD`; observaciones hasta 4.000 caracteres y organismo libre
hasta 500. Un conflicto devuelve 409 con `version_actual`. Un registro ajeno
o retirado devuelve 404 al intentar cambiarlo. `OTROS` sin texto devuelve 422.

El catálogo inicial de organismos pertenece a `actividad`, con códigos y claves
en la base y etiquetas en `idiomas/{es,en}/catalogos/actividad.json`. Su versión
1 es inmutable. No modifica el catálogo de otro módulo.

El puerto de entrada `ConsultaMemoria.Recontar(context.Context, Actor, Filtro)`
ofrece número de comisiones, número de informes emitidos y desglose por código
de organismo. Usa una sola sentencia SQL para leer ambas poblaciones sobre la
misma instantánea. Excluye retirados y otras anualidades; Coordinación puede
consultar todos los equipos directamente. No contiene observaciones, textos de
«Otros» ni identificadores de registros. No cambia revisiones de Memoria cerradas.

La migración `actividad/0001` depende de `catalogo/0002` y `auditoria/0001`.
Incluye RLS forzada, permisos mínimos, equipo inmutable, incremento de versión
y protección frente a borrado físico y recuperación de retirados. La aplicación
confirma datos y auditoría en una transacción; no entrega una lectura si falla
su apunte. Los errores se auditan tras revertir la operación.

Para ensayar con PostgreSQL 18 desechable y datos inventados:

```sh
python3 -B migraciones/actividad/ensayar.py
```

El ensayo aplica la versión anterior y la ampliación, repite el ejecutor, prueba
dos equipos, recuentos, límites de año, «Otros», conflictos y reversión por fallo
de auditoría. Después instala desde cero. Destruye su contenedor al terminar.
La misma prueba forma parte de `make ci`.
