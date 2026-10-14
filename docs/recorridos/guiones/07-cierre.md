# Cerrar un expediente

## Introducción

Título: Cerrar un expediente
Cartela: Datos inventados de la demostración.
Locución: El equipo propone el cierre de un expediente y Coordinación revisa la propuesta. Verás cómo otra persona confirma la fecha y deja constancia del cierre, con el Acta preparada para continuar la tramitación fuera de la aplicación.

## Preparación

Marta Cabrera (`marta.cabrera`) propone el cierre. Inés Moreno (`ines.moreno`) lo ve en Cierres propuestos. Javier Medina (`javier.medina`), también de Coordinación, confirma la fecha y el cierre y descarga el Acta. Cada ejecución acepta un expediente nuevo del ETF 02 con el Anexo 3 inventado del proyecto; conserva su número, propuesta, cierre y emisiones. La falta de consentimiento permite proponer desde ACEPTADA sin inventar una intervención. El Anexo 11 se genera por la API local y se muestra su emisión en Documentos: la ficha actual todavía no ofrece un botón para generarlo. El vídeo acredita la consulta de esa emisión, no una generación mediante un botón. No se reintentan escrituras sin confirmación. Las contraseñas se leen del entorno local.

```json
{
  "coordinacion_env": "ETF_VIDEO_COORDINACION",
  "clave_coordinacion_env": "ETF_VIDEO_CLAVE_COORDINACION",
  "coordinacion_demo": "ines.moreno",
  "equipo_codigo": "02",
  "descripcion_situacion": "La familia solicita apoyo para organizar los cuidados y mantener la asistencia escolar.",
  "automatizacion": "cierre",
  "perfiles": [
    "coordinacion",
    "profesional",
    "segunda"
  ],
  "profesional_env": "ETF_VIDEO_ACTIVIDAD_PROFESIONAL",
  "clave_profesional_env": "ETF_VIDEO_CLAVE_ACTIVIDAD_PROFESIONAL",
  "profesional_demo": "marta.cabrera",
  "segunda_env": "ETF_VIDEO_SEGUNDA_COORDINACION",
  "clave_segunda_env": "ETF_VIDEO_CLAVE_SEGUNDA_COORDINACION",
  "segunda_demo": "javier.medina",
  "finalizacion": {
    "derivacion_tras_cierre": "Seguimiento por Servicios Sociales Comunitarios.",
    "coordinacion_recursos": "Reunión de seguimiento con el servicio derivante.",
    "objetivos_abordados": "Contacto inicial con la familia.",
    "objetivos_pendientes": "La familia no ha consentido la intervención."
  },
  "sintetico": {
    "Título": "Prueba técnica del cierre",
    "Cartela": "Estas imágenes prueban el montaje."
  }
}
```

## Paso ficha

Rótulo: Abre la ficha y revisa Cierre
Locución: La profesional entra con su cuenta y abre la ficha del expediente nuevo. Selecciona la pestaña Cierre para preparar la finalización.

## Paso motivo

Rótulo: Completa el motivo de finalización
Locución: Selecciona el motivo de falta de consentimiento y confirma que la familia no consiente la intervención. Indica que Servicios Sociales Comunitarios continuará el seguimiento después del cierre del expediente. Anota la reunión de seguimiento con el servicio derivante para coordinar los recursos disponibles. Resume los objetivos abordados durante el contacto inicial que se mantuvo con la familia. Deja constancia de que la intervención queda pendiente porque la familia no ha dado su consentimiento. Guarda los datos de finalización antes de proponer el cierre.

## Paso anexo

Rótulo: Comprueba el Anexo de finalización emitido
Locución: La pestaña Documentos muestra la emisión del Anexo once preparado con los datos guardados.

## Paso proponer

Rótulo: Envía la propuesta a Coordinación
Locución: Vuelve a Cierre y propone la finalización a Coordinación; el expediente mantiene su estado hasta que se confirme el cierre.

## Paso ines

Rótulo: Coordinación revisa los cierres propuestos
Locución: Marta cierra su sesión e Inés entra como Coordinación con su propia cuenta para abrir la bandeja de Cierres propuestos. Localiza el número del expediente y comprueba el motivo de la propuesta del equipo.

## Paso javier

Rótulo: Otra persona revisa la propuesta
Locución: Inés cierra su sesión y Javier entra con su propia cuenta de Coordinación para abrir la bandeja de Cierres propuestos. Abre el expediente para revisar el motivo y confirmar el cierre como persona distinta de quien lo propuso.

## Paso cerrar

Rótulo: Confirma la fecha y el cierre
Locución: Anota la fecha real de cierre en el formulario. Pulsa Confirmar cierre y revisa el diálogo de confirmación. Confirma la operación y la pantalla muestra el cierre confirmado.

## Paso acta

Rótulo: Descarga el Acta del expediente cerrado
Locución: Descarga el Acta de Cierre en Word y comprueba el aviso que confirma la descarga.

## Cierre

Título: Expediente cerrado y Acta descargada
Cartela: Propuesta del equipo revisada; fecha y cierre confirmados.
Locución: El cierre queda confirmado y el Acta está descargada para su tramitación.
