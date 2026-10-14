# Preparar la Tabla SPAF del mes

## Introducción

Título: Preparar la Tabla SPAF del mes
Cartela: Datos inventados de la demostración.
Locución: Coordinación prepara la tabla mensual del Servicio de Prevención y Apoyo a la Familia para el equipo dos. La vista previa permite comprobar los movimientos del mes y consultar sus expedientes antes de obtener el documento Word.

## Preparación

Inés Moreno entra con `ines.moreno`. Cada ejecución carga dos expedientes históricos nuevos del ETF 02 en el último mes terminado, por la API local y con números libres: ambos cuentan como alta y uno también como baja. Comprueba que el resumen aumenta en dos altas, una baja y dos casos y dos menores. Cada caso lleva un menor con datos históricos desde la apertura. Conserva los casos, sus episodios y la emisión Word; no altera los anteriores. Las contraseñas proceden del entorno local. Una escritura sin confirmación detiene el recorrido sin repetirla.

```json
{
  "coordinacion_env": "ETF_VIDEO_COORDINACION",
  "clave_coordinacion_env": "ETF_VIDEO_CLAVE_COORDINACION",
  "coordinacion_demo": "ines.moreno",
  "equipo_codigo": "02",
  "apellidos_menores": "Navarro Rueda",
  "descripcion_situacion": "La familia solicita apoyo para organizar los cuidados y mantener la asistencia escolar.",
  "automatizacion": "spaf",
  "perfiles": [
    "coordinacion"
  ],
  "campos": [
    {
      "codigo": "menores.persona.nombre",
      "valor": "Clara"
    },
    {
      "codigo": "familia.contacto.apellidos",
      "valor": "Navarro Rueda"
    }
  ],
  "sintetico": {
    "Título": "Prueba técnica de la Tabla SPAF",
    "Cartela": "Estas imágenes prueban el montaje."
  }
}
```

## Paso abrir

Rótulo: Abre la tabla mensual del equipo
Locución: Coordinación abre la Tabla Mensual para el Servicio de Prevención y Apoyo a la Familia.

## Paso periodo

Rótulo: Elige equipo y mes de consulta
Locución: Elige el equipo dos en el selector de equipo. Selecciona el último mes terminado y aparecen sus datos.

## Paso movimientos

Rótulo: Revisa el resumen y los movimientos
Locución: El resumen muestra las altas, las bajas, los casos y el total de menores del mes. Debajo aparecen los números de expediente que cuentan como altas y como bajas.

## Paso alta

Rótulo: Consulta un expediente contado como alta
Locución: Abre el número de uno de los expedientes que cuenta como alta. La consulta muestra sus datos y la fecha de aceptación que explica el movimiento.

## Paso volver

Rótulo: Conserva equipo y mes al volver
Locución: Vuelve a la Tabla Mensual y conserva el equipo y el mes elegidos.

## Paso word

Rótulo: Descarga el documento Word del mes
Locución: Descarga el Word y comprueba el aviso que aparece tras la descarga.

## Cierre

Título: Tabla mensual preparada
Cartela: Altas y bajas revisadas; Word descargado.
Locución: La tabla del mes queda revisada y el Word está descargado.
