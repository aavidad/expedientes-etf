# No aceptación y lista de espera · F1.05b

La migración `0003_no_aceptacion_espera.sql` depende de `expediente/0002`.
Su número se reservó en el registro externo de migraciones antes de escribir SQL.
No cambia migraciones ya aplicadas ni permisos de otros módulos.

Coordinación dispone de estas operaciones con sesión vigente. Toda escritura
requiere CSRF, JSON, versión e `Idempotency-Key`:

- `POST /api/coordinacion/derivaciones/{id}/no-aceptar`: `version`, `fecha`
  (`AAAA-MM-DD`), `motivo` y `subprograma`. El motivo es obligatorio y admite
  hasta 20.000 bytes UTF-8. Anexo 3 usa `RIESGO_SOCIAL` si se omite el
  subprograma; no permite otro. Anexo 4 requiere confirmar `RIESGO_SOCIAL`
  o `REUNIFICACION` conforme al anexo. Puede partir de recepción o espera.
  Confirma un hito y el estado final `no_aceptada`, sin número ni equipo.
- `POST /api/coordinacion/derivaciones/{id}/lista-espera`: sólo `version`.
  Admite Anexo 3 en recepción. Guarda la fecha civil de Madrid y confirma
  el hito `lista_espera`, sin número, equipo ni episodio.
- `GET /api/coordinacion/lista-espera`: admite `limite` (1–200; defecto 50)
  y `desplazamiento` (desde 0). Orden estable por recepción e identificador.
  Devuelve identificador, estado, versión y fechas, sin nombres ni motivos.
  La lectura confirma su auditoría antes de responder.
- `POST /api/coordinacion/derivaciones/{id}/aceptar` admite también espera.
  Conserva el contrato F1.05a; desde espera exige que la fecha confirmada
  sea el día real en Madrid. El número usa la regla vigente y el contador
  del equipo para el año de esa fecha. Conserva el hito previo de espera.

La ficha existente recupera motivo, subprograma y fechas. Un caso rechazado
sale de la lista y no puede aceptarse ni volver a esperar. Una nueva derivación
requiere otro identificador. Estas operaciones no generan documentos: los
hitos y las emisiones pertenecen a acciones distintas.

Se conserva el mecanismo de autorización actual: actor desde sesión, perfil
`coordinacion` en aplicación y RLS en PostgreSQL. La tabla INT4 no está en
el `main` de partida. El servidor HTTP registra sólo las rutas exactas y
mantiene los límites de cuerpo y la validación estricta de campos.

La idempotencia comparte la tabla y el bloqueo de F1.05a: misma entrada y
clave devuelven el recibo durante 24 horas; distinto contenido o ruta bajo
la misma clave se rechazan. Estado, versión, hito, cambio de estado, recibo
y auditoría se confirman juntos. El fallo se audita después del rollback.
El motivo nunca se copia al recibo de idempotencia ni a la auditoría.

## Ensayo

`python3 -B migraciones/expediente/ensayar_aceptacion.py` usa PostgreSQL 18 y
ClamAV desechables con datos inventados. Aplica desde la versión anterior
(F1.05a), repite sin cambios y aplica también desde cero. Ejecuta todas las
pruebas de expediente, incluidas HTTP, transiciones, fecha Madrid con cambio
de año, concurrencia de ambas decisiones, auditoría y RLS sin equipo.
La red del ensayo es interna y los contenedores se eliminan al terminar.
