# GAP1: Familia y documentos de finalización

Datos inventados. Recorrido con `marta.cabrera` e `ines.moreno`, Playwright y Chrome del sistema contra la demo local del worktree.

```sh
ETF_DEMO_PUERTO=18513 make demo
node scripts/recorridos/gap1.mjs http://127.0.0.1:18513
```

El recorrido prepara expedientes de riesgo social y reunificación mediante los PDF de prueba de los Anexos 3 y 4 y las API de Coordinación. Comprueba:

- Familia: contacto, composición y texto de otras formas de convivencia, familia numerosa, vivienda y economía; ingresos con céntimos exactos y tramo directo excluyentes; guardado con vigencia y consulta de historia al corte.
- Personas adultas: alta completa, papel y sexo, nacimiento, educación, trabajo y descripción, tutoría y contacto. Recupera el identificador del servidor. Comprueba educación desconocida como `null` y retirada confirmada sin perder los datos familiares.
- Conflicto real: otro guardado incrementa la versión. La pantalla conserva el formulario, muestra los datos actuales y guarda el borrador elegido con la nueva versión.
- Avisos: desde Documentos, el aviso de educación abre Familia, despliega la persona y enfoca el campo.
- Cierre: los Word 11 y 12 se descargan en los respectivos subprogramas; tienen contenido DOCX y emisión registrada. La descarga conserva estado e hitos y no propone el cierre. Los cambios de finalización pendientes bloquean la descarga.
- Coordinación: consulta sin botones de edición; las escrituras de Familia y adultos y la emisión del Anexo 11 reciben 403.
- Teclado entre campos, castellano e inglés, tema por defecto y alto contraste; ausencia de errores JavaScript y de desplazamiento horizontal a 1440, 1024, 390, 360 y 320 píxeles.

Resultado del recorrido:

```text
GAP1-409-OK: borrador intacto, comparación y guardado tras conflicto real
GAP1-ANEXO12-OK: reunificación, Word real y cierre sin proponer
GAP1-RETIRADA-OK: confirmación, PUT y lectura sin persona retirada; teclado entre campos
GAP1-PERFILES-OK: Inés consulta; edición y anexos denegados en HTTP
GAP1-OK: Chrome del sistema; escritorio 1440, móvil 390, reflujo 320/360/1024; sin errores JS
```

Capturas: [Familia en escritorio](equipo-gap1-familia-escritorio.png), [Familia en móvil](equipo-gap1-familia-movil.png), [Cierre en escritorio](equipo-gap1-cierre-escritorio.png), [Cierre en móvil](equipo-gap1-cierre-movil.png) y [Familia en inglés con alto contraste](equipo-gap1-familia-contraste.png).

La revisión independiente de interfaz y seguridad dio GO de código tras corregir los valores desconocidos de educación, la navegación de avisos y la elección del anexo por subprograma. Semgrep local informó `0 findings` y Gitleaks `no leaks found`. Los textos y el manual se repasaron con humanizer.

Límites: no se ejecutó NVDA ni el comprobador de accesibilidad de un procesador de textos. No se cambiaron plantillas Word, rutas, permisos ni SQL. El registro de pantallas y módulos ya montaba Familia y Cierre; se conserva ese registro. `main` incorporó F5.06 durante la tarea. El conflicto del manual se resolvió conservando su contenido y sustituyendo los avisos de Familia y anexos por los recorridos ya disponibles.

## Validación de la entrega

Tras integrar el main que incorpora F5.06 y VID5, `make ci` completo terminó con:

```text
F0.11-CHROME-OK: entrada por teclado; pendiente; alta de cuatro perfiles; recarga; retirada; 1024/390; dos temas; 1536 combinaciones INT5; CSRF y métodos.
F0.11-RLS-OK: 7 actores; 53 tablas pobladas; 371 comprobaciones de lectura; 795 escrituras denegadas.
CI mínima completada.
```

También pasaron `go vet ./...`, `go test ./...`, la compilación TypeScript, las 31 pruebas del frontend y la comprobación de textos sin literales visibles. GAP1 se repitió sobre el binario final en el puerto 56101 elegido por CI, con el mismo resultado `GAP1-OK`. El PDF profesional se exportó de nuevo con LibreOffice local: se verificaron los textos de ambas tareas y las páginas renderizadas. Se retiraron sólo las capturas de CA1 regeneradas por CI para conservar las de main.
