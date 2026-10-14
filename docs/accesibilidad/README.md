# Revisión de accesibilidad F5.03

Esta revisión cubre las rutas declaradas por el registro INT5 del árbol auditado, la entrada y los estados de acceso. Los informes por pantalla están en la [matriz](matriz.md). Se ejecutó en la demostración local con datos inventados, Chrome del sistema y axe-core 4.10.3, fijado en `web/package.json` y con suma de integridad en `web/package-lock.json`.

Cada perfil autorizado se revisa en escritorio de 1440 px y móvil de 390 px, con los temas expedientes-etf y alto-contraste, en español e inglés. La ficha añade sus pestañas disponibles; Memoria añade sus cuatro bloques y la ayuda desplegada; las otras ayudas añaden sus diálogos. Consulta se revisa en Memoria, su pantalla de negocio registrada. La integración de main incorporó también Carga de 2026 y Alta histórica, revisadas con los perfiles de Coordinación y Profesional de ETF. La integración posterior de main añadió Memoria y la edición completa de Familia, que se audita sin modificar su código ni su composición.

La matriz final reúne 672 comprobaciones en una pasada completa de las 28 rutas y los estados de acceso, incluidas las pestañas de ficha, los cuatro bloques y Ayuda de Memoria. Los textos activos se contrastan con los catálogos de español e inglés del árbol auditado antes de abrir el navegador. No se detectaron infracciones serias ni críticas; las recomendaciones moderadas aparecen por pantalla.

## Cómo repetir la comprobación

La puerta automática queda incorporada a `make recorridos` mediante `scripts/recorridos/accesibilidad.py`. Puede ejecutarse por separado:

```bash
python3 -B scripts/recorridos/accesibilidad.py
```

El ejecutor arranca la demostración de este worktree, renueva sólo los ficheros de reserva local que difieren de los textos del árbol actual si el volumen procede de una demo anterior, reinicia sólo su aplicación para limpiar el contador de intentos de acceso, comprueba todos los perfiles y la detiene al terminar. Sale con error ante cualquier infracción axe seria o crítica; también falla si no puede completar una ruta o faltan datos necesarios. Antes de abrir el navegador verifica que los textos activos coinciden con los catálogos del árbol actual en ambos idiomas. No hay excepciones para silenciar infracciones. Si falla, conserva la evidencia en una carpeta temporal local; si pasa, elimina esa carpeta. Las otras regresiones de `make recorridos` conservan sus propios criterios de aceptación y pueden seguir detectando incidencias de producto independientes.

Para guardar capturas y regenerar informes sobre una demo ya arrancada:

```bash
make demo
ETF_AXE_CAPTURAS=1 node scripts/recorridos/accesibilidad.mjs http://127.0.0.1:18503 bin/temporales
python3 -B scripts/recorridos/informe_accesibilidad.py bin/temporales
make demo-parar
```

Ajusta el puerto al que haya elegido la demostración. No repitas autenticaciones rápidamente durante la depuración: se aplica el límite de intentos real. El navegador bloquea destinos ajenos al origen local. axe se carga desde la dependencia instalada, mediante el canal de evaluación de Playwright; la política de seguridad de la aplicación se conserva.

## Correcciones del armazón

El cambio de idioma desmontaba la pantalla y dejaba el foco en el cuerpo del documento. Ahora espera a que termine la carga del idioma y aparezca el nuevo control, y devuelve el foco a Persona; en la entrada, a Preferencias. Afecta al criterio 2.4.3, orden del foco. La revisión independiente comprobó el cambio en ambos sentidos entre español e inglés.

Preferencias interceptaba Escape aunque ya estuviera cerrado. Ahora sólo lo intercepta mientras está abierto. Un Escape cierra Preferencias y devuelve el foco a su título; el siguiente cierra Persona y devuelve el foco a Persona. Es una corrección de teclado y usabilidad relacionada con 2.1.1.

No se cambiaron colores ni textos visibles: se conservan los tokens y los catálogos de idioma existentes.

## Segunda revisión de usabilidad

Un revisor independiente recorrió Coordinación, Profesional de ETF y Administración con Chrome local, usando las skills programar-interfaz-etf, probar-recorridos-etf y revisar-cambios-etf, con las reglas de usabilidad y aspecto del proyecto. Comprobó el salto al contenido, el contorno de foco de 3 px, las diez pestañas de ficha con flechas y Home/End, el cierre y retorno de foco de las ayudas, los errores asociados de fecha de comisión y el resumen enfocado, el editor de permisos y las preferencias a 390 y 320 px. No guardó cambios de negocio. La revisión se amplió tras integrar main: las dos pantallas de carga pasaron en español e inglés, con alto contraste, a 390 y 320 px, incluidos resumen de errores, enlaces a campos y ayudas. Memoria se recorrió por Coordinación, Profesional de ETF y Consulta, en ambos idiomas y temas a 390 y 320 px, en sus cuatro bloques. Bandeja y Memoria también pasaron reflujo a 1024 y 360 px.

La candidata con el bundle `index-CZ78R0UQ.js` pasó la comprobación de foco tras cambiar el idioma. El doble Escape y el foco del editor de permisos también pasaron. La revisión manual acotada permite entregar estos cambios; conserva la incidencia de pestañas descrita debajo.

## Pendientes y propuestas

- En la ficha de expediente, Consentimiento y Salidas pueden quedar parcialmente fuera de la fila visible al cambiar con flechas en móvil. Gravedad moderada; criterios 2.4.7 y 1.4.10. Para reproducir: abre Familia a 390 px y pulsa flecha derecha tres veces. Propuesta: desplazar la pestaña seleccionada con `scrollIntoView({block: 'nearest', inline: 'nearest'})`. Corresponde a `web/src/equipo/equipo.tsx`, fuera del alcance de cambios en otros módulos.
- Las recomendaciones de regiones complementarias anidadas o duplicadas aparecen en los informes correspondientes con sus selectores. Son resultados moderados de las reglas de buenas prácticas de axe; requieren revisar la semántica en los módulos propietarios. No equivalen por sí solos a un incumplimiento WCAG.
- En Memoria, Ayuda muestra u oculta un párrafo sin anunciar su estado mediante `aria-expanded` ni relacionar el panel con `aria-controls`. Gravedad moderada, relacionada con 4.1.2 y 1.3.1. Para reproducir: abre Memoria con cualquiera de sus tres perfiles y pulsa Ayuda. Propuesta: publicar el estado y el identificador del panel. El módulo queda intacto.
- En la vista editable de Memoria, al activar otro bloque con Enter, el botón se deshabilita y el foco acaba en el cuerpo del documento. Mejora moderada relacionada con 2.4.3 y 2.4.7; requiere comprobación con lector de pantalla para concluir el incumplimiento. Propuesta: conservar el botón actual enfocable con `aria-current` o `aria-pressed`, o enfocar el nuevo encabezado. No se cambia `web/src/memoria/Memoria.tsx`.
- En Familia, el error de sexo de una persona adulta dice «Indica el sexo del menor». Gravedad menor de usabilidad, relacionada con 3.3.1. Para reproducir: entra como profesional, abre Familia, añade una persona adulta y revisa sin completar el sexo. Propuesta: una clave propia de GAP1 en español e inglés. Corresponde al módulo de equipo y queda pendiente por el límite de pantallas ajenas.
- No se ejecutó NVDA ni el comprobador de accesibilidad de Word. Quedan pendientes la lectura real con lector de pantalla, las plantillas y los estados que no se abrieron. Las reglas automáticas incompletas, en particular las de contraste que necesitan comprobación humana, aparecen por pantalla.
- Esta entrega no certifica conformidad completa con WCAG 2.1 AA. La fila F5.03 del plan mantiene pendientes los Word y las revisiones indicadas.

La revisión independiente se amplió a GAP1 después de integrar main: Familia y Cierre, Coordinación y Profesional de ETF, español e inglés real, alto contraste y 390/320 px. Pasaron etiquetas, ayudas, nombres, reflujo y asociación y foco de errores locales. Memoria inglesa se volvió a recorrer en sus cuatro bloques por los tres perfiles a 390/320 px y con ambos temas. Las reservas antiguas que devolvían castellano al pedir inglés se renovaron; la comprobación previa valida ahora los valores, además de las claves.

Tras integrar F4.07, la vista separada de Consulta pasó 24 combinaciones manuales: provincia y ETF 01, español e inglés, ambos temas, y 390/320/1024 px. Sus botones anuncian selección con `aria-pressed` y mantienen el foco. Tablas, regiones y reflujo correctos; no ofrece narrativas ni enlaces de composición. La cuenta profesional disponible tiene sólo ese perfil, por lo que no se probó la variante que exige además Consulta.

La demo reutilizada rechazó el arranque con `migracion.estado_previo` al incorporar las migraciones de Consulta. Se recreó únicamente la demo sintética de este worktree y se regeneraron sus datos. Esta auditoría no modifica migraciones ni acredita la actualización de una base real.

## Validación final del árbol integrado

El 14 de octubre de 2026 se integró `main` (`af3c814`), incluida F5.02, en la candidata `3cce145`. Se conservaron las pruebas de Memoria de F4.09 y la auditoría de accesibilidad en `make recorridos`, y se regeneró el índice de producto. La pasada final usó Chrome 155.0.8059.39 del sistema y axe-core 4.10.3 sobre las 28 rutas INT5, más Entrada y Pendiente de alta. Los informes por pantalla y las capturas enlazadas corresponden a esta pasada completa, sin filtros:

```text
AXE: 672 comprobaciones; 0 infracciones serias o críticas
F5.03-INFORMES: 30 pantallas; 672 comprobaciones.
```

Todas las comprobaciones tuvieron un único h1 y reflujo sin desbordamiento de página.

La segunda revisión independiente dio GO para los cambios F5.03 de `776fde0`. Comprobó con teclado el cambio de español a inglés en Entrada y Coordinación, el retorno del foco a Preferencias o Persona, el doble Escape y el salto al contenido. Con alto contraste, Entrada, Coordinación, Comisiones y Memoria no desbordaron la página a 1024, 390 y 320 px. En Comisiones, con el perfil profesional, un envío inválido sin guardar enfocó el resumen de errores; la fecha quedó marcada con `aria-invalid` y enlazada mediante `aria-describedby`, y el enlace del resumen llevó el foco al campo. No guardó datos de negocio. Confirmó también el pendiente de Ayuda de Memoria descrito arriba.

El revisor amplió el GO a `3cce145` después de integrar F5.02. Verificó que los componentes y scripts F5.03 conservaban su código y repitió el cambio de idioma, Escape, salto al contenido y reflujo con alto contraste a 1024, 390 y 320 px en la nueva demo. Confirmó las dependencias instaladas `axe-core@4.10.3` y `vite@7.3.7`. No encontró incidencias nuevas de usabilidad en ese alcance.

La revisión local de los scripts no encontró problemas de seguridad en el alcance revisado. Semgrep ejecutó la regla TypeScript instalada sobre `web/src/main.tsx` y `web/src/armazon.tsx`: `Ran 1 rule on 2 files: 0 findings.` Los scripts se revisaron por lectura de código. Se comprobó la versión exacta de axe-core y su integridad SHA-512 en el archivo de dependencias.

`make ci` se ejecutó completo antes y después de integrar F5.02; ambas ejecuciones terminaron con código 0. La última línea fue `CI mínima completada.` Incluyó las pruebas del frontend, textos y temas, `go vet ./...`, `go test ./...`, los ensayos PostgreSQL y RLS, y los recorridos de entrada/permisos y carga histórica. El recorrido de permisos respetó las ventanas reales de peticiones por origen añadidas en F5.02.
