# Entrar y dar acceso a un profesional

## Introducción

Título: Entrar y dar acceso a un profesional
Cartela: Del primer acceso a la asignación de perfil y equipo. Datos inventados.
Locución: Este recorrido muestra cómo un profesional de los Equipos de Tratamiento Familiar obtiene acceso a su equipo. La persona que administra los permisos prepara su cuenta para que pueda empezar a trabajar.

## Preparación

Usa dos cuentas que ya existan en la demostración. La cuenta profesional debe estar sin perfil ni equipo. La herramienta rechaza cuentas de baja y cuentas con permisos previos. Las contraseñas se reciben por variables de entorno y no se guardan en este guion. No se crean usuarios ni expedientes.

Al terminar, la herramienta retira la concesión que acaba de grabar para poder repetir el recorrido. Sólo lo hace con una respuesta propia confirmada y si la versión y el perfil siguen siendo los que concedió: si alguien los ha cambiado, para y pide revisar la demostración. Quedan las versiones y los apuntes de auditoría de ambas operaciones. La cuenta queda activa sin perfiles ni equipo, por lo que al entrar vuelve a estar pendiente de alta.

```json
{
  "profesional_env": "ETF_VIDEO_PROFESIONAL",
  "clave_profesional_env": "ETF_VIDEO_CLAVE_PROFESIONAL",
  "administracion_env": "ETF_VIDEO_ADMINISTRACION",
  "clave_administracion_env": "ETF_VIDEO_CLAVE_ADMINISTRACION",
  "equipo_env": "ETF_VIDEO_EQUIPO_ID",
  "puesto": "ts",
  "profesional_demo": "sergio.campos",
  "administracion_demo": "elena.salas",
  "equipo_demo": "02",
  "sintetico": {
    "Título": "Prueba técnica de vídeo y voz",
    "Cartela": "Estas imágenes prueban el montaje. El recorrido en la aplicación queda pendiente."
  }
}
```

## Paso profesional

Rótulo: El profesional entra con su cuenta
Locución: El profesional escribe su cuenta y su contraseña. Pulsa Entrar para comprobar su acceso.

## Paso pendiente

Rótulo: Acceso pendiente de asignar permisos
Locución: La cuenta es válida, pero el acceso está pendiente de alta. Todavía necesita un perfil y un equipo para trabajar.

## Paso salir

Rótulo: El profesional cierra su sesión
Locución: El profesional abre su menú y pulsa Cerrar sesión.

## Paso administracion

Rótulo: Administración entra para preparar el acceso
Locución: La persona que administra los permisos escribe su cuenta y contraseña. Pulsa Entrar para preparar el acceso del profesional.

## Paso usuarios

Rótulo: Selecciona al profesional pendiente de alta
Locución: Abre Usuarios y permisos. Selecciona al profesional que está pendiente de alta.

## Paso perfil

Rótulo: Asigna perfil, equipo y puesto
Locución: Marca el perfil profesional de los Equipos de Tratamiento Familiar. Elige el equipo dos y el puesto de trabajo social.

## Paso guardado

Rótulo: Permisos guardados y acceso preparado
Locución: Pulsa Guardar permisos. La confirmación indica que el profesional ya tiene acceso a su equipo.

## Cierre

Título: El acceso ya está preparado
Cartela: Hemos visto la entrada, la espera del alta y la asignación de perfil y equipo.
Locución: El profesional ya tiene acceso a su equipo para empezar a trabajar.
