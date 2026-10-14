# Evaluaciones: registro y respuestas por ítem

F2.05 incorpora `GET` y `POST /api/expedientes/{id}/evaluaciones` en INT5, sin pantallas. Profesional de ETF accede a su equipo; Coordinación accede a todos. Consulta y Administración no acceden. Un expediente cerrado no admite cambios.

El POST recibe `clave_idempotencia`, `version_esperada` y exactamente uno de `instrumento` o `aplicacion`. El cliente proporciona un UUID nuevo al crear. Los instrumentos aplicados conservan código, fecha y texto cuando el código es OTROS. Las aplicaciones conservan momento, sujeto, fecha, versión de escala, descripción y cada respuesta en `evaluacion.respuesta`, con una fila por ítem. La actualización sustituye las respuestas de esa aplicación dentro de la misma transacción. La versión pertenece a la aplicación: 0 crea; una versión desactualizada devuelve 409 y `version_actual`. La identidad nunca procede del cuerpo.

La clave de idempotencia pertenece al usuario y expediente. Una repetición idéntica devuelve el resultado original; reutilizarla con otra entrada devuelve 409. Escritura, recibo y auditoría se confirman juntos; las lecturas se entregan después de confirmar su auditoría. El bloqueo del expediente serializa las ediciones con el cierre. Los vínculos expediente/equipo y adulto/expediente/equipo son claves compuestas. La aplicación comprueba la vigencia del adulto mediante el puerto de F2.01a.

Los anexos 7, 8, 11 y 12 identifican VALÓRAME, ECP-12, SCORE-15, SDQ, SOATIF, AAPI-2.5 y satisfacción. La Tabla Mensual para el SPAF aporta los demás instrumentos del catálogo. No contienen las preguntas de los cuestionarios ni las respuestas admitidas por ítem. `idiomas/<código>/catalogos/evaluacion.json` contiene sus nombres; `evaluacion-v1.reglas` contiene códigos y las definiciones de escala, inicialmente vacías. La extensión `.reglas` distingue los datos de validación de los catálogos de traducciones. Las reglas de cada versión se publicarán como un fichero nuevo inmutable; las versiones utilizadas deben conservarse. El servidor rechaza versiones desconocidas o incompletas. Las pruebas usan una escala técnica inventada, que no se distribuye en el catálogo.

**Pendiente del Programa:** cuestionarios, respuestas admitidas y puntuación según baremo. El servidor no suma respuestas, transforma puntuaciones ni atribuye baremos. Cada definición debe identificar su versión y cada ítem antes de habilitar la escala. D17 en el registro de decisiones del PRD recoge la petición.

El tipo de sujeto menor y el puerto `ConsultaMenores` están preparados, pero su uso devuelve indisponibilidad mientras F2.01b no publique el contrato. La migración rechaza sujetos menores hasta añadir, en otra migración reservada, el vínculo compuesto con el contrato persistente de F2.01b. No basta con inyectar el puerto. Los sujetos profesionales se rechazan hasta disponer de un puerto que confirme la adscripción al equipo en la fecha aplicada. El momento `derivacion` es de sólo lectura y el POST rechaza su edición.

## Ensayo local

```sh
TMPDIR=bin/temporales python3 migraciones/evaluacion/ensayar.py
```

El ensayo usa PostgreSQL 18 desechable sin red, aplica la versión anterior, la amplía y repite; después prueba instalación limpia. Comprueba HTTP y PostgreSQL con datos inventados: respuestas por ítem, adulto vigente, conflicto, repetición, aislamiento, CSRF y reversión por fallo de auditoría. No usa ninguna base real.
