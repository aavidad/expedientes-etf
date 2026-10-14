#!/usr/bin/env python3
"""Genera informes por pantalla a partir de una pasada axe local terminada."""
import collections
import json
from pathlib import Path
import re
import sys

RAIZ = Path(__file__).resolve().parents[2]


def nombre(ruta):
    return re.sub(r'[^a-z0-9]+', '-', ruta.lower()).strip('-') or 'entrada'


def criterio(tags):
    criterios = []
    for tag in tags:
        m = re.fullmatch(r'wcag(\d)(\d)(\d+)', tag)
        if m:
            criterios.append('.'.join(m.groups()))
    return ', '.join(criterios) or 'Recomendación de estructura (axe best-practice)'


def main():
    datos = json.loads(Path(sys.argv[1]).read_text())
    if not datos.get('completada'):
        raise ValueError('La pasada no terminó; no se genera un informe completo')
    if datos.get('filtrado'):
        raise ValueError('Una pasada filtrada no sustituye la matriz completa')
    destino = RAIZ / 'docs/accesibilidad'
    destino.mkdir(parents=True, exist_ok=True)
    grupos = collections.defaultdict(list)
    for caso in datos['resultados']:
        grupos[caso['ruta']].append(caso)
    vigentes = {nombre(ruta) + '.md' for ruta in grupos} | {'README.md', 'matriz.md'}
    for anterior in destino.glob('*.md'):
        if anterior.name not in vigentes:
            anterior.unlink()
    indices = []
    for ruta, casos in sorted(grupos.items()):
        fichero = nombre(ruta) + '.md'
        perfiles = sorted({c['perfil'] for c in casos})
        fallos = {}
        revisables = collections.Counter(v['id'] for c in casos for v in c['incomplete'])
        for c in casos:
            for v in c['violations']:
                k = (v['id'], v['impact'])
                fallo = fallos.setdefault(k, {'v': v, 'casos': []})
                fallo['casos'].append(c)
        graves = sum(v['impact'] in ['serious', 'critical'] for c in casos for v in c['violations'])
        indices.append(f'| [{ruta}]({fichero}) | {", ".join(perfiles)} | {len(casos)} | {graves} |')
        lineas = [f'# Accesibilidad: {ruta}', '',
                  f'Pasada local con Chrome del sistema y axe-core {datos["axe"]}. Datos inventados de la demostración.', '',
                  '## Alcance comprobado', '',
                  f'Perfiles: {", ".join(perfiles)}. {len(casos)} comprobaciones; escritorio de 1440 px y móvil de 390 px, español e inglés, temas expedientes-etf y alto-contraste.', '',
                  'Cada combinación aparece debajo. Las pestañas y ayudas se auditan cuando están disponibles. Un resultado automático sin infracciones no acredita todos los criterios WCAG.', '',
                  '| Perfil | Idioma | Tema | Ancho | Estado | Infracciones | Reflujo de página |',
                  '| --- | --- | --- | --- | --- | --- | --- |']
        for c in casos:
            lineas.append(f'| {c["perfil"]} | {c["idioma"]} | {c["tema"]} | {c["ancho"]} | {c["estado"]} | {len(c["violations"])} | {"Desborda" if c["desbordamiento"] else "Sin desbordamiento"} |')
        lineas += ['', '## Hallazgos automáticos', '']
        if not fallos:
            lineas.append('axe no detectó infracciones en los estados examinados.')
        for (id, gravedad), fallo in sorted(fallos.items()):
            v = fallo['v']
            lineas += [f'### {id}', '', f'Criterio: {criterio(v["tags"])}. Gravedad axe: {gravedad}.', '',
                       f'Observado en {len(fallo["casos"])} comprobaciones. {v["help"]}.', '',
                       'Para reproducir, entra con el perfil indicado, abre la ruta y selecciona la combinación de la tabla con infracciones. Ejecuta la auditoría con el comando del índice.', '']
            for nodo in v['nodes']:
                lineas += [f'- Selector: `{json.dumps(nodo["target"], ensure_ascii=False)}`. {nodo.get("failureSummary", "") or ""}']
            lineas += ['', 'Propuesta: revisar la semántica del componente en su módulo. No se modifican pantallas ajenas en este encargo.', '']
        alertas = sorted({a for c in casos for a in c.get('alertas', []) if a})
        if alertas:
            lineas += ['', '## Estados de error observados', '',
                       'Estos mensajes limitan la revisión del contenido de negocio. Se auditó también el estado de error accesible que mostró la pantalla.', '']
            lineas += [f'- {a}' for a in alertas]
        if ruta in ['/equipo/expediente', '/equipo/menores', '/equipo/intervencion', '/equipo/seguimiento']:
            lineas += ['', '## Revisión manual de la ficha', '',
                       'Incidencia moderada, criterios 2.4.7 y 1.4.10: a 390 px, desde Familia, pulsa flecha derecha tres veces. Consentimiento conserva el foco pero queda parcialmente fuera de la fila visible. Salidas presenta el mismo comportamiento. Propuesta: desplazar la pestaña seleccionada con scrollIntoView al cambiar con flechas. El cambio corresponde a web/src/equipo/equipo.tsx y queda pendiente por el límite de otros módulos.', '']
        lineas += ['', '## Revisión manual pendiente', '',
                   f'axe dejó resultados incompletos para revisión humana: {", ".join(f"{k} ({v})" for k, v in sorted(revisables.items())) or "ninguno en esta pasada"}.', '',
                   'Quedan fuera de esta evidencia el lector de pantalla NVDA, la inspección de los Word, los estados no abiertos y las combinaciones de datos que no contiene la demostración. Consulta el índice para la segunda revisión de usabilidad.', '',
                   '## Capturas', '']
        for perfil in perfiles:
            for ancho in [1440, 390]:
                imagen = f'accesibilidad-{perfil}-{nombre(ruta)}-{ancho}.png'
                if (RAIZ / 'docs/capturas' / imagen).exists():
                    lineas.append(f'- [{perfil}, {ancho} px](../capturas/{imagen}).')
        (destino / fichero).write_text('\n'.join(lineas) + '\n')
    (destino / 'matriz.md').write_text('# Matriz automática de accesibilidad\n\n'
        f'axe-core {datos["axe"]}; {datos["rutas"]} rutas INT5; {len(datos["resultados"])} comprobaciones, incluidas entrada y estados de acceso.\n\n'
        '| Pantalla | Perfiles | Comprobaciones | Infracciones serias o críticas |\n| --- | --- | --- | --- |\n' + '\n'.join(indices) + '\n')
    print(f'F5.03-INFORMES: {len(grupos)} pantallas; {len(datos["resultados"])} comprobaciones.')


if __name__ == '__main__':
    main()
