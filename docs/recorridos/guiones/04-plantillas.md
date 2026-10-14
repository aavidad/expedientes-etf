# Publicar una plantilla Word nueva

## Introducción

Título: Publicar una plantilla Word nueva
Cartela: Administración revisa y activa una versión del Acta de Aceptación. Datos inventados.
Locución: Administración mantiene las plantillas Word que usa el Programa. Verás cómo la aplicación identifica un marcador que sobra antes de publicar una versión y cómo deja constancia de quién activa la plantilla y en qué momento.

## Preparación

Elena Salas entra con la cuenta inventada `elena.salas`. La contraseña se toma del entorno de demostración y aparece enmascarada. Usa únicamente el Acta inventada versionada en `idiomas/es/plantillas/entrada/aceptacion/v1.1.0.docx`, nunca un documento del cliente.

La herramienta prepara dos Word en la carpeta de salida: uno con el marcador añadido `intruso` y otro con los marcadores correctos. El primero conserva todos los marcadores válidos para que el aviso señale sólo el sobrante. Cada ejecución elige una versión `3.n.0` libre, la sube y la activa con confirmación. Conserva las versiones anteriores y la auditoría; no restaura la referencia activa. Una escritura sin confirmación detiene el recorrido, sin repetirla. Si alguien cambia la versión activa durante la grabación, se detiene por conflicto.

```json
{
  "automatizacion": "plantillas",
  "perfiles": ["administracion"],
  "administracion_env": "ETF_VIDEO_ADMINISTRACION",
  "clave_administracion_env": "ETF_VIDEO_CLAVE_ADMINISTRACION",
  "administracion_demo": "elena.salas",
  "tipo": "entrada/aceptacion",
  "idioma": "es",
  "plantilla": "idiomas/es/plantillas/entrada/aceptacion/v1.1.0.docx",
  "marcador": "intruso",
  "archivo_invalido": "acta-marcador-desconocido.docx",
  "archivo_valido": "acta-aceptacion.docx",
  "sintetico": {
    "Título": "Prueba técnica de publicación de plantillas",
    "Cartela": "Estas imágenes prueban el montaje. El recorrido en la aplicación queda pendiente."
  }
}
```

## Paso abrir

Rótulo: Administración abre las plantillas Word
Locución: Administración entra con su cuenta. Abre Plantillas Word para consultar los documentos disponibles.

## Paso historial

Rótulo: Revisa la versión activa y su historial
Locución: El Acta de Aceptación muestra la versión activa en castellano. Abre su historial para ver las versiones conservadas.

## Paso invalida

Rótulo: Revisa un Word con marcador desconocido
Locución: Pulsa Subir una versión para abrir el formulario. Elige el Acta de Aceptación en castellano y escribe un número de versión nuevo. Selecciona el Word que contiene un marcador desconocido. Pulsa Subir y revisar versión para comprobar el archivo.

## Paso aviso

Rótulo: El aviso identifica el marcador sobrante
Locución: El aviso indica que sobra el marcador intruso y pide retirarlo o corregirlo en el Word. La versión activa sigue siendo la anterior y la subida conserva el número escrito.

## Paso buena

Rótulo: Sube el Word con marcadores correctos
Locución: Selecciona el Word corregido para sustituir el archivo rechazado. Pulsa Subir y revisar versión otra vez. El aviso confirma que ha superado el antivirus y la revisión de marcadores, pero aún está pendiente de activar.

## Paso activar

Rótulo: Confirma qué versión usarán los documentos
Locución: Cierra la subida y busca la nueva versión en el historial. Pulsa Activar versión para revisar la confirmación. Confirma la activación para que los próximos documentos usen esta plantilla.

## Paso autoria

Rótulo: Comprueba quién la activó y cuándo
Locución: La tarjeta muestra la nueva versión activa del Acta. El historial conserva el nombre de quien la activó y la fecha de la activación.

## Cierre

Título: Nueva plantilla activa
Cartela: La versión revisada queda activa, con su autoría y su historial.
Locución: La plantilla revisada está activa y su historial conserva quién la activó y cuándo.
