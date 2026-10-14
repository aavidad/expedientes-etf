# FX2: actualización automática de roles técnicos

Ensayo local del 14 de octubre de 2026, en `fx2-roles-actualizacion` y el proyecto
Docker exclusivo de `/dev/shm/etf/wt-FX2`. Sólo datos y contraseñas inventados.
No se ha accedido a instalaciones compartidas ni a producción.

## Volumen anterior y actualización

El worktree empezó en `7828247`, antes de F4.07. Se ejecutó
`ETF_DEMO_PUERTO=18272 make demo` con el código anterior y se comprobó que no
existía `etf_calculo_memoria`. Se conservaron el volumen, sus expedientes y el
entorno privado. Tras añadir FX2, `make demo` creó automáticamente la credencial
nueva y el rol, sin cambiar los recuentos de expedientes ni de migraciones.
Después se integró `main 949705a`, ya con F4.07, y se repitió `make demo` sobre
el mismo volumen: se aplicó `estadistica/0005` y arrancó la aplicación.
No se ejecutó el guion manual de habilitación.

Salida de las comprobaciones reales:

```text
ANTERIOR: etf_calculo_memoria inexistente=t
ACTUALIZACION-OK: volumen anterior conservado; roles preparados; demo saludable; expedientes e historial sin cambios
F4.07-OK: estadistica/0005
```

## Instalación limpia y repetición

Se borró únicamente esta demostración mediante `make demo-limpiar` y se ejecutó
de nuevo `ETF_DEMO_PUERTO=18272 make demo`. La inicialización, preparación,
migración y aplicación terminaron correctamente.

Un ensayo temporal concedió una pertenencia inesperada al rol de cálculo y
repitió la orden normal de migración. Compose volvió a ejecutar la preparación
y bloqueó al migrador. Tras retirar esa pertenencia de prueba, la misma orden
terminó sin volver a aplicar la migración:

```text
ORDEN-OK: repetir migrar vuelve a preparar roles; preparación inválida bloquea ejecución
LIMPIA-OK: instalación nueva; estadistica/0005 aplicada; repetición sin duplicados
```

## Credenciales, datos y seguridad

Los ensayos PostgreSQL usaron el entorno privado de la demo, sin secretos en
argumentos. Compararon un volcado de datos y secuencias normalizado antes y
después de preparar repetidamente, rotar y deshabilitar la credencial, y provocar
una denegación. Las credenciales originales se repusieron al terminar.

```text
REPETICION-OK: cinco roles; credenciales válidas; todos los datos y secuencias idénticos
ROTACION-OK: credencial nueva aceptada; anterior rechazada
OPCIONAL-OK: contraseña vacía deshabilita LOGIN y elimina credencial anterior
DENEGACION-OK: pertenencia inesperada aborta sin rotar credenciales ni modificar datos
SEGURIDAD-OK: instalador sin TCP; app sin privilegios extra; socket 0700; preparación sin red; logs sin secretos
DATOS-OK: SHA256 0d88a8e0398a0712a8877f719492e333194305d0305fa43d75c46b52df4cb180
70:70 700 /var/run/postgresql/.s.PGSQL.5432
```

`python3 -B -m unittest discover -s despliegue/demo -p 'test_*.py'` ejecutó tres
pruebas, resultado `OK`: ampliación del entorno anterior, repetición y credencial
vacía, y fallo al publicar sin pérdida de las claves anteriores. La CI incorpora
estas pruebas. `sh -n` de los guiones y `git diff --check` terminaron con salida 0.
Semgrep local, con las reglas de `despliegue/base/semgrep.yaml` y
`scripts/recorridos/seguridad.yml`, sin métricas ni consulta remota de versión:

```text
Ran 3 rules on 5 files: 0 findings.
```

Revisión independiente de SQL, permisos e integración, sólo lectura: GO sobre
`main 949705a` y los cambios FX2. No acredita un despliegue de producción.

## CI completa

`make ci` terminó con salida 0, incluidos `go vet ./...`, `go test ./...`,
frontend, textos, secretos, ensayos PostgreSQL y recorridos de entrada, permisos
y carga con Chrome del sistema. Última línea real:

```text
CI mínima completada.
```
