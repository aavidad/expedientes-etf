# Del Anexo a la Memoria

## Introducción

Título: Del Anexo a la Memoria
Cartela: Datos inventados. Calendario de formación: octubre y noviembre de 2026.
Locución: Sigue un expediente desde la llegada del Anexo tres hasta la Memoria anual. Javier recibe la derivación, Marta completa la ficha y los dos revisan el cierre y las cifras que deja el trabajo del equipo.

## Preparación

Este recorrido usa una sola familia inventada: Ortega Salcedo, de Agrón, con seis menores. Parte de un Anexo 3 adaptado de la plantilla de pruebas con los datos inventados del aula. La ubicación propuesta corresponde al ETF 02. Javier Medina (`javier.medina`) recibe y acepta; Marta Cabrera (`marta.cabrera`) trabaja la ficha y propone; Javier confirma el cierre como persona distinta de quien propuso. Inés Moreno (`ines.moreno`), también de Coordinación, revisa y cierra la Memoria provincial. Las contraseñas se toman de la demo, nunca de este documento. Se reparten los actos de Coordinación entre Inés y Javier para no agotar las cinco entradas por cuenta en quince minutos tras la preparación del entorno.

Las fechas inventadas del trabajo pertenecen a octubre de 2026: aceptación el día 14, datos vigentes desde el 15, consentimiento el 16, instrumento el 17, comisión el 22, informe el 24 y cierre el 31. Se simula el paso del tiempo para enseñar el ciclo en una sesión. La tabla de septiembre se consulta el 24 de octubre y no incluye el caso recibido en octubre. Tras el cierre del 31 de octubre, el calendario avanza al 1 de noviembre: la tabla de octubre muestra alta y baja del mismo caso y permite descargar su Word. La Memoria se revisa y cierra el 1 de noviembre. El calendario sólo se simula en una compilación local para formación, mediante overlay de Go; los ficheros de producción y el reloj de sesión conservan su funcionamiento ordinario. Las consultas de vigencia de familia, menores e intervención usan también la fecha de formación en ese overlay, manteniendo los filtros de acceso y las reglas de guardado. Para poder repetir la práctica tras reinstalar la demo, la lectura del catálogo inicial toma como publicación su inicio de vigencia. Esa proyección sólo existe en el overlay; conserva las publicaciones posteriores y no escribe ni altera las versiones del catálogo en la base. Playwright fija también la fecha del navegador y mantiene los temporizadores en tiempo real.

Cada ejecución crea una derivación y conserva su número, datos, consentimiento, instrumento, finalización y emisiones. Antes de abrir la ficha, la preparación traslada a sus seis menores los nombres, apellidos, nacimientos y escolarización inventados del Anexo, con vigencia desde el 15 de octubre; sólo deja el nacimiento de Nerea pendiente para completarlo en la escena. No se solicitan propuestas de atención temprana en este caso inventado. Este traslado se comprueba por la API local y sólo afecta al expediente recién creado; la pantalla no copia por sí sola todos los datos del Anexo. Comisiones e informes son registros del equipo; no tienen vínculo individual con el expediente. La comisión explica el trabajo sobre el caso inventado sin copiar nombres a sus observaciones. La evaluación registra el instrumento sin cuestionario ni baremo oficial. Se conservan los avisos de otros expedientes en la Memoria y se señalan como pendientes; no se declara validado su cálculo por el Programa.

Antes de las escenas de ficha, Coordinación reabre con motivo las Memorias de equipo y provincial que hayan quedado cerradas en grabaciones anteriores. Mantiene sus instantáneas. El vídeo termina con ambas cerradas; el texto narrativo cambia el número de revisión para comprobar un nuevo guardado. No borra casos previos ni repite una escritura incierta. Los Word, MP4, escenas, estado y fotogramas se guardan fuera del repositorio.

Repite desde la raíz con `make video RECORRIDO=11-ciclo-completo`. Revisa la introducción en los segundos 1, 2, 3, 6, 10 y 12; la mitad de cada paso y cierre; también los resultados finales de aceptación, consentimiento, propuesta, cierre y ambas Memorias. Usa los tiempos de `plan.json`, no los estimados de esta guía. La duración final debe estar entre 480 y 720 segundos. Los capítulos y los tiempos comprobados se publican en [EVIDENCIA-11.md](../EVIDENCIA-11.md).

```json
{
  "automatizacion": "ciclo",
  "perfiles": [
    "coordinacion",
    "profesional",
    "segunda"
  ],
  "coordinacion_env": "ETF_VIDEO_COORDINACION",
  "clave_coordinacion_env": "ETF_VIDEO_CLAVE_COORDINACION",
  "coordinacion_demo": "javier.medina",
  "profesional_env": "ETF_VIDEO_ACTIVIDAD_PROFESIONAL",
  "clave_profesional_env": "ETF_VIDEO_CLAVE_ACTIVIDAD_PROFESIONAL",
  "profesional_demo": "marta.cabrera",
  "segunda_env": "ETF_VIDEO_SEGUNDA_COORDINACION",
  "clave_segunda_env": "ETF_VIDEO_CLAVE_SEGUNDA_COORDINACION",
  "segunda_demo": "ines.moreno",
  "anexo": "docs/formacion/anexo3-ortega-salcedo.docx",
  "capitulos": [
    {
      "titulo": "Derivación y aceptación",
      "paso": "bandeja"
    },
    {
      "titulo": "Ficha y consentimiento",
      "paso": "ficha"
    },
    {
      "titulo": "Comisiones e informes",
      "paso": "comision"
    },
    {
      "titulo": "Tabla SPAF del mes",
      "paso": "spaf"
    },
    {
      "titulo": "Propuesta y cierre",
      "paso": "finalizacion"
    },
    {
      "titulo": "Memoria anual",
      "paso": "memoria"
    }
  ],
  "anio": 2026,
  "mes": "2026-10",
  "mes_previo": "2026-09",
  "fecha_memoria": "2026-11-01",
  "fecha_aceptacion": "2026-10-14",
  "fecha_datos": "2026-10-15",
  "fecha_consentimiento": "2026-10-16",
  "fecha_evaluacion": "2026-10-17",
  "fecha_comision": "2026-10-22",
  "fecha_informe": "2026-10-24",
  "fecha_cierre": "2026-10-31",
  "direccion": "Calle del Olivo 18, Agrón",
  "adultos": [
    {
      "nombre": "Lucía",
      "apellidos": "Ortega Salcedo",
      "papel": "MADRE",
      "sexo": "M",
      "edad": "39"
    },
    {
      "nombre": "Daniel",
      "apellidos": "Pérez Molina",
      "papel": "PADRE",
      "sexo": "H",
      "edad": "41"
    },
    {
      "nombre": "Rosa",
      "apellidos": "Salcedo Ávila",
      "papel": "ABUELO",
      "sexo": "M",
      "edad": "67"
    }
  ],
  "menores": [
    {
      "nombre": "Nerea",
      "sexo": "M",
      "fecha_nacimiento": "2016-02-14",
      "apellidos": "Pérez Ortega",
      "nivel_escolarizacion": "PRIMARIA"
    },
    {
      "nombre": "Álvaro",
      "sexo": "H",
      "fecha_nacimiento": "2012-11-23",
      "apellidos": "Pérez Ortega",
      "nivel_escolarizacion": "ESO"
    },
    {
      "nombre": "Celia",
      "sexo": "M",
      "fecha_nacimiento": "2014-03-08",
      "apellidos": "Pérez Ortega",
      "nivel_escolarizacion": "PRIMARIA"
    },
    {
      "nombre": "Mateo",
      "sexo": "H",
      "fecha_nacimiento": "2018-06-19",
      "apellidos": "Pérez Ortega",
      "nivel_escolarizacion": "PRIMARIA"
    },
    {
      "nombre": "Irene",
      "sexo": "M",
      "fecha_nacimiento": "2020-04-12",
      "apellidos": "Pérez Ortega",
      "nivel_escolarizacion": "PRIMARIA"
    },
    {
      "nombre": "Hugo",
      "sexo": "H",
      "fecha_nacimiento": "2022-09-03",
      "apellidos": "Pérez Ortega",
      "nivel_escolarizacion": "INFANTIL"
    }
  ],
  "comision": "Reunión con el servicio derivante para revisar las rutinas de cuidado y los acuerdos de asistencia escolar.",
  "organismo": "SERVICIOS_SOCIALES",
  "finalizacion": {
    "derivacion_tras_cierre": "Seguimiento por Servicios Sociales Comunitarios.",
    "coordinacion_recursos": "Reunión con el servicio derivante y el centro educativo.",
    "objetivos_abordados": "Se han reforzado las rutinas de cuidado y la asistencia escolar.",
    "objetivos_pendientes": "Mantener los acuerdos de seguimiento con el servicio derivante."
  },
  "narrativa": "Durante el año se han coordinado entrevistas familiares y reuniones con los servicios derivantes. El equipo revisa los acuerdos de seguimiento y los datos pendientes antes del cierre anual. Revisión de trabajo {version}.",
  "motivo_preparacion": "Preparación del recorrido de formación con datos inventados; se conserva la revisión cerrada anterior.",
  "sintetico": {
    "Título": "Prueba técnica del ciclo",
    "Cartela": "Estas imágenes prueban el montaje."
  }
}
```

## Paso bandeja

Rótulo: Coordinación recibe la derivación del caso
Locución: Javier entra con su cuenta de Coordinación y abre la bandeja donde recibirá la derivación y podrá revisar, aceptar o dejar casos en espera. Seguiremos a la familia Ortega Salcedo, aunque haya otras familias inventadas de grabaciones anteriores en esta demostración. El menú te permite volver a la bandeja.

## Paso subir

Rótulo: Sube el Anexo tres recibido
Locución: Javier abre Subir Anexo y selecciona el documento inventado con la información de la familia y sus seis menores. Pulsa Subir y leer anexo para revisar el fichero y extraer los datos que comprobará antes de aceptar. El antivirus termina sin amenazas y la pantalla permite continuar con la revisión del contenido recibido.

## Paso revisar

Rótulo: Revisa ubicación, familia y menores
Locución: Javier comprueba el municipio, el centro y el equipo propuestos a partir de la lectura, que corresponden a Agrón. Despliega Familia para contrastar los datos con el Anexo, sin dar por correcto un campo sólo porque se haya reconocido. Después abre Menores y revisa nombres y fechas, conservando las dudas que necesiten aclaración.

## Paso crear

Rótulo: Crea la derivación tras revisar
Locución: Javier confirma los datos y crea la derivación que utilizará durante el resto del recorrido, con su estado y su versión. La pantalla muestra la derivación recibida, todavía sin el número que se asigna al aceptar; esa decisión sigue pendiente. El subprograma de la entrada por Anexo tres queda fijado según la regla del Programa.

## Paso aceptar

Rótulo: Acepta y conserva el número
Locución: Javier selecciona el puesto de trabajo social como referencia y anota el catorce de octubre como fecha inventada del Acta. Marca el sexo de los seis menores por separado y comprueba que no queda ninguno sin indicar. Acepta y confirma en el diálogo; el número asignado identificará este mismo caso en la ficha y el cierre.

## Paso acta

Rótulo: Descarga el Acta de Aceptación
Locución: Javier descarga el Acta de Aceptación en Word con los datos del expediente aceptado, sin volver a crear la derivación. El aviso confirma la descarga para continuar el trámite de firma fuera de la aplicación y guardar el documento donde indique el Programa. El expediente mantiene su número y su estado; descargar añade una emisión sin confirmar una firma.

## Paso ficha

Rótulo: El equipo abre su expediente
Locución: Marta entra con su cuenta profesional y busca el número en la bandeja de su equipo, donde la familia aparece con sus iniciales. Abre el resultado y encuentra la ficha del mismo caso, con las pestañas de datos, intervención y documentos. Los avisos ayudan a localizar información pendiente antes de preparar las cifras o la finalización.

## Paso familia

Rótulo: Completa y guarda los datos familiares
Locución: En Familia, Marta confirma la dirección y selecciona familia extensa porque conviven los progenitores, la abuela y los seis menores inventados. Anota familia numerosa y vivienda en alquiler; los datos desconocidos quedan pendientes de comprobar y no se sustituyen por un cero. Guarda con vigencia desde el quince de octubre, por separado de las personas adultas.

## Paso adultos

Rótulo: Revisa las personas adultas del caso
Locución: Marta registra las tres personas adultas del Anexo con sus nombres y apellidos: Lucía como madre, Daniel como padre y Rosa como abuela, indicando el sexo y la edad conocida de cada persona para conservar quién forma parte de la familia y a quién corresponden las evaluaciones registradas después. Guarda las personas adultas por separado para conservar las referencias que podrá seleccionar al registrar una evaluación aplicada a una de ellas. La ficha muestra la madre; guardar no significa que estén completos todos los datos pendientes.

## Paso menores

Rótulo: Comprueba el nacimiento de la menor
Locución: Marta abre Menores y confirma el catorce de febrero de dos mil dieciséis como fecha de nacimiento de Nerea en el caso inventado. Guarda el apartado con su vigencia, manteniendo a los seis menores del Anexo dentro del mismo expediente. El formulario conserva la fecha y permite seguir revisando escolarización y características que influirán en las cifras.

## Paso fase

Rótulo: Registra la fase de intervención actual
Locución: En Intervención, Marta selecciona evaluación como fase actual del trabajo con la familia; la fase pertenece al expediente y se aplica a todos sus menores. Guarda desde el quince de octubre para conservar la vigencia del cambio. La pantalla mantiene evaluación; factores y tipologías requieren su propia revisión, con las excepciones por menor que correspondan.

## Paso consentimiento

Rótulo: Descarga el consentimiento sin confirmar firma
Locución: Marta descarga el Anexo uno en Word para explicar el trabajo a la familia y recoger su consentimiento fuera de la aplicación. La pantalla conserva la firma pendiente, porque generar el documento prepara su contenido y registra una emisión. El campo de fecha permite anotar el día conocido de firma, sin confundir la descarga con ese acto.

## Paso firma

Rótulo: Confirma la fecha del consentimiento firmado
Locución: Marta anota el dieciséis de octubre como fecha inventada de firma; durante la formación simulamos el tiempo transcurrido desde la aceptación. Confirma la fecha y la aplicación guarda el hito del consentimiento, separado de la emisión del Word. El aviso de confirmación permite comprobar que se ha conservado antes de seguir trabajando con la familia.

## Paso evaluacion

Rótulo: Registra el instrumento aplicado a Lucía
Locución: En Evaluación, Marta selecciona a la madre y el instrumento aplicado para identificar a qué persona corresponde este registro inicial. Anota el diecisiete de octubre y guarda sin introducir un cuestionario oficial ni calcular puntuaciones o baremos. El instrumento aparece en el listado, sin que registrarlo equivalga a un resultado clínico o una valoración terminada.

## Paso comision

Rótulo: Anota la comisión técnica del equipo
Locución: Marta abre Comisiones e informes y registra la comisión técnica del veintidós de octubre para revisar el trabajo con el servicio derivante. Resume los acuerdos de cuidados y asistencia escolar sin copiar nombres de familiares a las observaciones, indicando para qué se reunieron y qué aspectos necesitan coordinación con el servicio derivante. Al guardar, la comisión aparece como actividad del equipo; este registro no está vinculado individualmente al expediente.

## Paso informe

Rótulo: Registra el informe y su destinatario
Locución: Marta cambia a Informes y anota el veinticuatro de octubre como fecha del informe elaborado durante el trabajo sobre el caso inventado. Selecciona Servicios Sociales como organismo destinatario para agrupar la actividad en las cifras anuales del equipo. Guarda y comprueba el listado; registrar estos datos no envía el informe ni conserva su contenido como fichero.

## Paso spaf

Rótulo: Consulta el mes anterior al caso
Locución: Marta abre la Tabla mensual para el servicio de prevención y apoyo a las familias y consulta septiembre, el último mes terminado en este punto del calendario simulado. La familia llegó en octubre y todavía no aparece en septiembre; cada tabla cuenta los movimientos que corresponden a su propio período. Revisa el resumen y los movimientos, sin atribuir a esta familia cifras de otro mes o de otros casos que hayas conservado.

## Paso wordspaf

Rótulo: Descarga la Tabla mensual en Word
Locución: Marta descarga la tabla de septiembre en Word con las cifras que corresponden a ese período terminado y a su equipo. El aviso confirma la descarga y registra la emisión; el caso recibido en octubre queda fuera de esta tabla de septiembre. El selector conserva el mes consultado; después del cierre avanzaremos a noviembre para consultar octubre y encontrar el alta y la baja de la familia.

## Paso finalizacion

Rótulo: Prepara la finalización con sus objetivos
Locución: Marta vuelve a la ficha del mismo número y selecciona en Cierre el motivo de objetivos satisfactoriamente cumplidos para esta simulación. Describe el seguimiento por Servicios Sociales Comunitarios tras el cierre, la reunión con el servicio derivante y el centro educativo, las rutinas de cuidado y asistencia escolar reforzadas, y los acuerdos que mantendrá la familia para conservar los cambios logrados, revisando cada campo antes de pasar al siguiente para que el documento de finalización recoja los datos que se han trabajado. Guarda la finalización antes de descargar el documento o proponer el cierre, para utilizar los datos revisados.

## Paso anexo

Rótulo: Descarga el Anexo once de finalización
Locución: Marta descarga el Anexo once con los datos guardados, que resumen la finalización y corresponden al subprograma de este expediente. En Documentos encuentra la emisión, mientras el expediente sigue en intervención porque descargar no propone el cierre ni confirma una firma. El equipo debe enviar todavía su propuesta a Coordinación y esperar la confirmación.

## Paso propuesta

Rótulo: Propón el cierre a Coordinación
Locución: Marta vuelve a Cierre y envía la propuesta para que Coordinación revise la finalización y confirme la fecha que corresponde al expediente. Mientras sigue pendiente, puede retirar la propuesta; el estado permanece en intervención porque proponer no equivale a cerrar. Los objetivos son inventados para la formación; en tu trabajo debes contrastarlos con la intervención realizada.

## Paso revision

Rótulo: Coordinación revisa la propuesta del equipo
Locución: Marta sale de su cuenta y Javier entra con Coordinación para abrir la bandeja de cierres propuestos por los equipos. Localiza el número asignado al aceptar y abre la propuesta con el motivo y la finalización preparada por Marta. Revisa una persona distinta de quien propuso, conservando quién intervino en cada paso del cierre.

## Paso cierre

Rótulo: Confirma la fecha de cierre acordada
Locución: Javier anota el treinta y uno de octubre como fecha del cierre inventado, después de revisar la propuesta y sus datos de finalización. Pulsa Confirmar cierre y lee el diálogo, comprobando el mismo número que seguimos desde la derivación. Confirma y la pantalla muestra el expediente cerrado; su fecha podrá entrar en las cifras mensuales y anuales.

## Paso actacierre

Rótulo: Descarga el Acta y conserva el cierre
Locución: Javier descarga el Acta de Cierre en Word del expediente ya cerrado, separando la confirmación del cierre de la preparación del documento. El aviso permite continuar la firma fuera de la aplicación, sin modificar la fecha guardada. En noviembre, consulta octubre: el mismo número figura como alta y baja; descarga su tabla en Word y conserva el cierre.

## Paso memoria

Rótulo: El equipo consulta la Memoria anual
Locución: Marta entra con su cuenta profesional y abre la Memoria anual del equipo, que reúne las cifras del año a partir de los registros. Los filtros muestran dos mil veintiséis y el equipo del expediente que acabamos de cerrar. El resumen incluye otros casos y sus datos pendientes, que habrá que revisar antes de terminar la Memoria.

## Paso apartados

Rótulo: Revisa los apartados y sus avisos
Locución: El resumen señala los apartados pendientes para que Marta localice los datos que debe comprobar en fichas o registros de actividad. Abre uno y consulta las cifras y la información sin dato, que no debe confundirse con una ausencia confirmada. La Memoria conserva el ámbito de equipo; la grabación no acredita la validación del cálculo por el Programa.

## Paso narrativa

Rótulo: Guarda la metodología del trabajo anual
Locución: En Metodología, Marta resume el trabajo anual y la coordinación con los servicios derivantes, sin identificar familias ni copiar detalles de sus fichas, y explica que el equipo revisa los acuerdos de seguimiento y los datos pendientes antes de preparar el cierre anual, incluyendo las entrevistas y reuniones de coordinación realizadas. Guarda el texto; su número de revisión inventado permite comprobar un guardado nuevo al repetir la grabación. La pantalla confirma los cambios para continuar la revisión del documento, con cifras y apartados narrativos.

## Paso wordmemoria

Rótulo: Descarga la Memoria del equipo en Word
Locución: Marta descarga la Memoria del equipo en Word con las cifras y los textos guardados para revisar el documento que se ha preparado. El aviso confirma una emisión, sin cerrar la Memoria ni resolver los avisos pendientes. La revisión sigue abierta; comprueba el contenido y los datos de origen antes de conservar una revisión cerrada para la entrega.

## Paso cerrarequipo

Rótulo: Conserva la revisión cerrada del equipo
Locución: Marta pulsa Cerrar la Memoria y lee el diálogo que explica la conservación de una foto de cifras y textos del equipo. Confirma y la aplicación guarda una instantánea nueva, manteniendo las revisiones cerradas anteriores para consultarlas. La pantalla muestra el cierre, aunque los avisos vistos siguen pendientes y la grabación no sustituye la validación del Programa.

## Paso provincial

Rótulo: Coordinación consulta la Memoria provincial completa
Locución: Marta sale y Inés entra con Coordinación para consultar la Memoria anual de toda la provincia, según los permisos de su perfil. El selector muestra Todos los equipos; las cifras reúnen actividad y expedientes provinciales, además de este caso inventado. Los avisos permiten revisar los datos pendientes.

## Paso cerrarprovincia

Rótulo: Cierra y conserva la revisión provincial
Locución: Inés pulsa Cerrar la Memoria provincial y revisa en el diálogo el ámbito de todos los equipos y el año que quiere conservar. Confirma la simulación y se guarda una instantánea provincial distinta de la revisión que cerró Marta. La pantalla muestra la Memoria cerrada; si debes corregirla, Coordinación puede reabrir con motivo y mantener su historia.

## Cierre

Título: Ciclo registrado
Cartela: Del Anexo a las revisiones de Memoria.
Locución: El expediente conserva su historia y ambas Memorias su revisión cerrada.
