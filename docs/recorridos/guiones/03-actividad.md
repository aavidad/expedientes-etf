# Registrar una comisión técnica y un informe

## Introducción

Título: Registrar una comisión técnica y un informe
Cartela: El equipo registra su actividad y Coordinación la consulta. Datos inventados.
Locución: Una profesional del Equipo de Tratamiento Familiar registra una comisión y un informe técnico ya emitido. Verás cómo guarda la actividad de su equipo, corrige una fecha y permite que Coordinación consulte los registros de todo el ámbito de la entidad.

## Preparación

Marta Cabrera entra con la cuenta inventada `marta.cabrera`, del ETF 02. Después entra Inés Moreno Castillo, con `ines.moreno`, para consultar todos los equipos. Las contraseñas proceden del entorno local y se muestran enmascaradas.

Cada ejecución registra una comisión y un informe nuevos. Corrige únicamente la fecha del informe creado durante esa grabación. Conserva ambos registros, sus versiones y la auditoría; no modifica registros anteriores. Comprueba las respuestas de escritura y los identificadores al consultar. Si una escritura no se confirma, se detiene sin repetirla.

```json
{
  "automatizacion": "actividad",
  "perfiles": ["profesional", "coordinacion"],
  "profesional_env": "ETF_VIDEO_ACTIVIDAD_PROFESIONAL",
  "clave_profesional_env": "ETF_VIDEO_CLAVE_ACTIVIDAD_PROFESIONAL",
  "profesional_demo": "marta.cabrera",
  "coordinacion_env": "ETF_VIDEO_COORDINACION",
  "clave_coordinacion_env": "ETF_VIDEO_CLAVE_COORDINACION",
  "coordinacion_demo": "ines.moreno",
  "equipo_codigo": "02",
  "fecha_comision": "2026-10-08",
  "observaciones": "Revisión de acuerdos y reparto de tareas para la siguiente reunión.",
  "fecha_informe": "2026-10-09",
  "fecha_corregida": "2026-10-10",
  "organismo": "JUZGADOS",
  "sintetico": {
    "Título": "Prueba técnica del registro de actividad",
    "Cartela": "Estas imágenes prueban el montaje. El recorrido en la aplicación queda pendiente."
  }
}
```

## Paso abrir

Rótulo: Consulta la actividad de tu equipo
Locución: La profesional entra con su cuenta del equipo dos. Abre Comisiones e informes y ve los registros de su equipo.

## Paso comision

Rótulo: Anota la fecha y los acuerdos
Locución: Pulsa Registrar comisión técnica para abrir el formulario. Anota el ocho de octubre como fecha de la reunión. Escribe las observaciones sobre los acuerdos y el reparto de tareas para preparar la siguiente reunión del equipo. Pulsa Guardar registro y la comisión aparece en el listado.

## Paso informe

Rótulo: Registra un informe enviado al juzgado
Locución: Cambia a Informes técnicos y pulsa Registrar informe técnico. Anota el nueve de octubre como fecha del informe emitido. Elige Juzgados como organismo destinatario. Guarda el registro y comprueba que aparece en el listado.

## Paso corregir

Rótulo: Corrige la fecha del informe registrado
Locución: Abre Corregir registro en el informe que acaba de guardar. Cambia la fecha al diez de octubre. Guarda la corrección y comprueba la nueva fecha en el listado.

## Paso coordinacion

Rótulo: Coordinación consulta todos los equipos
Locución: Cierra la sesión de la profesional desde el menú de persona conectada. Coordinación entra con su cuenta y abre Comisiones e informes. El filtro muestra Todos los equipos y la comisión del equipo dos aparece en la consulta.

## Paso provincial

Rótulo: Comprueba el informe en la consulta provincial
Locución: Cambia a Informes técnicos para consultar los de todos los equipos. El informe del equipo dos muestra Juzgados y la fecha corregida.

## Cierre

Título: Actividad registrada y consultada
Cartela: La comisión y el informe quedan guardados para el equipo y Coordinación.
Locución: La actividad del equipo queda guardada y Coordinación puede consultarla.
