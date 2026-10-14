# Recibir una derivación y aceptarla

## Introducción

Título: Recibir una derivación y aceptarla
Cartela: Coordinación recibe el Anexo, revisa los datos y acepta la derivación. Datos inventados.
Locución: La persona de coordinación del Programa recibe una derivación y la acepta. Verás cómo la aplicación ayuda a revisar la información y deja preparado el expediente y su Acta para continuar el trabajo diario.

## Preparación

Entra Inés Moreno Castillo, persona de coordinación del Programa, con la cuenta inventada `ines.moreno` de la demostración local. La contraseña se toma del entorno, sin incorporarla al guion ni a la grabación. La bandeja contiene derivaciones pendientes sembradas por la demo.

Sube `internal/adaptadores/anexos/docx/testdata/anexo3.docx`: contiene datos inventados de la familia Ortega Salcedo, en Alhama de Granada. Se usa el documento de pruebas, nunca un original de las plantillas de `idiomas/`. La aplicación propone municipio, centro y equipo a partir de la lectura y los catálogos vigentes; el vídeo muestra esos valores, sin imponer los de otro guion.

Cada ejecución crea una derivación nueva mediante la pantalla y la acepta una sola vez. Se conservan el expediente, su número y la emisión del Acta, con su auditoría. Al repetir avanza el correlativo; no se borran números ni se cambian expedientes previos. El recorrido busca el número que acaba de recibir en «Aceptados recientes», aunque haya varias familias con las mismas iniciales. Si una escritura falla o no se confirma, se detiene sin repetirla.

```json
{
  "automatizacion": "coordinacion",
  "coordinacion_env": "ETF_VIDEO_COORDINACION",
  "clave_coordinacion_env": "ETF_VIDEO_CLAVE_COORDINACION",
  "coordinacion_demo": "ines.moreno",
  "anexo": "internal/adaptadores/anexos/docx/testdata/anexo3.docx",
  "puesto": "TS",
  "sintetico": {
    "Título": "Prueba técnica del recorrido de Coordinación",
    "Cartela": "Estas imágenes prueban el montaje. El recorrido en la aplicación queda pendiente."
  }
}
```

## Paso bandeja

Rótulo: Coordinación revisa las derivaciones pendientes
Locución: Coordinación escribe su cuenta y entra en la aplicación. La bandeja muestra las derivaciones pendientes de revisar.

## Paso subir

Rótulo: Sube y lee el Anexo tres
Locución: Pulsa Subir Anexo y elige el Anexo tres de la familia. Pulsa Subir y leer anexo para revisar el fichero.

## Paso revisar

Rótulo: Comprueba los datos y la ubicación
Locución: El antivirus ha terminado sin amenazas. Comprueba el municipio, el centro y el equipo propuestos. Revisa los datos de la familia y de sus menores.

## Paso crear

Rótulo: Confirma y crea la derivación
Locución: Pulsa Confirmar y crear derivación. La derivación queda recibida, todavía sin número de expediente.

## Paso aceptar

Rótulo: Acepta y obtiene el número asignado
Locución: Elige el puesto de referencia y confirma la fecha del Acta. Indica si cada menor de la derivación es niña o niño y comprueba que no queda ninguno sin marcar. Pulsa Aceptar derivación y confirma la decisión. La aplicación asigna el número de expediente.

## Paso descargar

Rótulo: Descarga el Acta de Aceptación
Locución: Pulsa Descargar Acta de Aceptación. El aviso confirma que has descargado el Word.

## Paso recientes

Rótulo: Encuentra el expediente en Aceptados recientes
Locución: Vuelve a la bandeja y abre Aceptados recientes. Busca el número del expediente aceptado.

## Cierre

Título: Derivación aceptada
Cartela: Has revisado el Anexo, obtenido el número de expediente y descargado el Acta.
Locución: La derivación está aceptada y el Acta descargada para continuar el trabajo.
