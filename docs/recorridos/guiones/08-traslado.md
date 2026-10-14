# Trasladar un expediente a otro equipo

## Introducción

Título: Trasladar un expediente a otro equipo
Cartela: Datos inventados de la demostración.
Locución: Coordinación organiza el traslado de una familia cuando cambia de municipio. La aplicación conserva el expediente de origen y enlaza el nuevo del equipo de destino, para que cada equipo consulte su trabajo y continúe el acompañamiento que le corresponde.

## Preparación

Javier prepara por API local un expediente nuevo de Reunificación en el ETF 02 con el Anexo 4 inventado. Marta confirma el consentimiento y propone el cierre por traslado. Javier traslada al ETF 04 y acepta la derivación enlazada. Pablo consulta el número nuevo; Marta inicia el seguimiento tras cese del origen. Conserva ambos expedientes y el seguimiento. Sólo Reunificación admite seguimiento tras cese.

```json
{
  "profesional_env": "ETF_VIDEO_ACTIVIDAD_PROFESIONAL",
  "clave_profesional_env": "ETF_VIDEO_CLAVE_ACTIVIDAD_PROFESIONAL",
  "profesional_demo": "marta.cabrera",
  "coordinacion_env": "ETF_VIDEO_TRASLADO_COORDINACION",
  "clave_coordinacion_env": "ETF_VIDEO_CLAVE_TRASLADO_COORDINACION",
  "coordinacion_demo": "javier.medina",
  "equipo_codigo": "02",
  "descripcion_situacion": "La familia solicita apoyo para organizar los cuidados y mantener la asistencia escolar.",
  "perfiles": [
    "profesional",
    "coordinacion",
    "destino"
  ],
  "automatizacion": "traslado",
  "anexo": "anexo4",
  "sexos_menores": ["M", "H", "M"],
  "subprograma": "REUNIFICACION",
  "destino_codigo": "04",
  "destino_env": "ETF_VIDEO_DESTINO",
  "clave_destino_env": "ETF_VIDEO_CLAVE_DESTINO",
  "destino_demo": "pablo.ruiz",
  "descripcion_destino": "La familia necesita continuar el apoyo tras cambiar de municipio.",
  "motivo_traslado": "La familia cambia de municipio y necesita continuar el apoyo.",
  "sintetico": {
    "Título": "Prueba técnica del traslado",
    "Cartela": "Estas imágenes prueban el montaje."
  }
}
```

## Paso abrir

Rótulo: Revisa el traslado propuesto por origen
Locución: Coordinación abre el traslado del expediente que ha propuesto el equipo de origen.

## Paso destino

Rótulo: Indica equipo destino y fecha de cierre
Locución: Elige el equipo cuatro como destino. Escribe el motivo que explica por qué la familia cambia de equipo y anota la fecha de cierre del expediente de origen.

## Paso trasladar

Rótulo: Confirma el traslado entre ambos equipos
Locución: Revisa la confirmación antes de trasladar el expediente. Confirma y aparece la derivación de destino pendiente de aceptación.

## Paso aceptacion

Rótulo: Completa la aceptación del nuevo expediente
Locución: Abre la derivación enlazada del equipo de destino. Selecciona el municipio y el subprograma de reunificación. Anota la fecha del acta y el puesto de referencia. Confirma el sexo de cada menor antes de aceptar la derivación. Describe la necesidad de continuar el acompañamiento de la familia en el nuevo municipio, con datos que se incluirán en el acta.

## Paso aceptar

Rótulo: Asigna el número nuevo en destino
Locución: Revisa la aceptación y confirma los datos. La aplicación asigna un número nuevo al expediente del equipo cuatro.

## Paso equipo

Rótulo: Destino consulta su expediente y número nuevo
Locución: Pablo, profesional del equipo cuatro, entra con su cuenta y busca en la bandeja el expediente de la familia que se acaba de trasladar. Abre el expediente trasladado y comprueba su número nuevo.

## Paso origen

Rótulo: Origen mantiene el seguimiento tras cese
Locución: Marta vuelve a entrar con su cuenta y consulta el seguimiento tras cese del expediente cerrado en el equipo de origen. Anota la fecha de cese y confirma el inicio del seguimiento. Aparecen el seguimiento activo y su fecha prevista de fin.

## Cierre

Título: Traslado y seguimiento registrados
Cartela: Destino y origen conservan sus expedientes enlazados.
Locución: Destino tiene su número nuevo y origen conserva el seguimiento.
