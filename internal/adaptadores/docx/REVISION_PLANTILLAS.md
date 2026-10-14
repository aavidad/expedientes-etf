# Revisión de las plantillas públicas

Las 58 versiones de `idiomas/*/plantillas/` se distribuyen sin membretes ni contactos de una instalación concreta. Sus metadatos de autor identifican a Expedientes ETF. Las huellas de las cargas iniciales se corresponden con estos bytes; una instalación nueva usa esta colección desde el principio.

`TestRevisionTodasLasPlantillas` rellena todas las versiones con una y dos filas por bloque. Comprueba los marcadores de cuerpos, cabeceras y pies, la equivalencia entre versiones y que no queden marcadores pendientes. Los datos de prueba son inventados.

Las comparaciones visuales con documentos privados se han retirado de la edición pública. Para revisar una plantilla nueva, genera documentos de prueba con `ETF_DOCX_EVIDENCIA_DIR`, conviértelos con LibreOffice local y revisa las páginas y la accesibilidad antes de activarla. Los conversores de desarrollo admiten una carpeta de fuentes mediante `ETF_PLANTILLAS_FUENTE`; por defecto buscan en `bin/plantillas-fuente`.

Las plantillas de entrada, Anexo 5 y SPAF de `en` conservan texto castellano y declaran su idioma real. Cada administración debe revisar las traducciones, formularios y referencias normativas aplicables. Las plantillas siguen rechazando macros, objetos ejecutables, instrucciones de campo ajenas a la paginación y enlaces inseguros.
