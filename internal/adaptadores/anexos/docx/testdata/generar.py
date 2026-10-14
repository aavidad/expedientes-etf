"""Construye perfiles y copias inventadas de las dos plantillas del cliente."""
import argparse
import copy
import json
import os
import subprocess
from pathlib import Path
import zipfile
import xml.etree.ElementTree as E

W = 'http://schemas.openxmlformats.org/wordprocessingml/2006/main'
C = 'http://schemas.microsoft.com/office/word/2010/wordml'
A = 'http://schemas.microsoft.com/office/2006/activeX'
R = 'http://schemas.openxmlformats.org/officeDocument/2006/relationships'
N = {'w': W}
E.register_namespace('w', W)
E.register_namespace('w14', C)

def texto(n, estatico=False):
    def partes(x):
        if x.tag in [f'{{{W}}}sdtPr', f'{{{W}}}del'] or (estatico and x.tag == f'{{{W}}}sdt'):
            return ''
        return (x.text or '' if x.tag == f'{{{W}}}t' else '') + ''.join(partes(h) for h in x) + (' ' if x.tag == f'{{{W}}}p' else '')
    return ' '.join(partes(n).split())

def guardar(p, datos):
    with zipfile.ZipFile(p, 'w', zipfile.ZIP_DEFLATED) as z:
        for nombre, valor in datos.items():
            info = zipfile.ZipInfo(nombre, (2026, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(info, valor)

def poner(n, valor):
    for h in list(n):
        n.remove(h)
    p = E.SubElement(n, f'{{{W}}}r')
    E.SubElement(p, f'{{{W}}}t').text = valor

ESTUDIOS = {
    'Analfabetismo': 'Analfabetismo',
    'Sin estudios': 'Sabe leer y escribir, no finaliza estudios obligatorios',
    'Primarios': 'Estudios obligatorios (ESO, EGB)',
    'Secundarios': 'Estudios secundarios (Bachiller, FP, módulos profesionales)',
    'Superiores': 'Estudios universitarios',
}


def construir(numero, plantillas, salida):
    origen = plantillas / ('Anexo 3 (Derivación de SSCC a ETF).docx' if numero == 3 else 'Anexo 4 (Derivación de SPM a ETF).docx')
    with zipfile.ZipFile(origen) as z:
        datos = {n: z.read(n) for n in z.namelist()}
    raiz = E.fromstring(datos['word/document.xml'])
    tablas = raiz.findall('./w:body/w:tbl', N)
    perfil = dict(Tipo=f'anexo{numero}', Version='cliente-2026-v1',
                  Titulos=[texto(p) for p in raiz.findall('./w:body/w:p', N) if texto(p)],
                  Tablas=[[[[texto(p, True) for p in c.findall('w:p', N) if texto(p, True)] for c in r.findall('w:tc', N)] for r in t.findall('w:tr', N)] for t in tablas],
                  Reglas=[], Estudios=ESTUDIOS)
    def regla(codigo, ti, ri, ci, control=0, repetir=False, modo='', opciones=None):
        celda = tablas[ti].findall('w:tr', N)[ri].findall('w:tc', N)[ci]
        controles = celda.findall('.//w:sdt', N)
        relleno = texto(celda) if not controles and modo == '' else ''
        opts = []
        if modo not in ['papel', 'casillas', 'parrafo'] and control < len(controles):
            s = controles[control]
            relleno = texto(s.find('w:sdtContent', N))
            opts = [x.get(f'{{{W}}}displayText') for x in s.findall('w:sdtPr/w:dropDownList/w:listItem', N)][1:]
        if not controles and modo == '':
            perfil['Tablas'][ti][ri][ci] = []
        perfil['Reglas'].append(dict(Codigo=codigo, Tabla=ti, Fila=ri, Celda=ci, Control=control,
                                    Etiqueta=(texto(celda, True) if controles else '') if modo != 'parrafo' else 'Ingresos de la unidad familiar:',
                                    Relleno=relleno, Opciones=opciones if opciones is not None else opts,
                                    Repetir=repetir, Modo=modo))
    regla('servicio.nombre', 1, 0, 0)
    regla('servicio.profesional', 1, 1, 0)
    if numero == 4:
        regla('servicio.equipo_menores', 1, 2, 0)
        regla('servicio.etf', 0, 2, 1)
        regla('servicio.subprograma', 0, 3, 1)
    regla('servicio.telefono', 1, 2 if numero == 3 else 3, 0)
    regla('servicio.correo', 1, 2 if numero == 3 else 3, 1)
    regla('familia.direccion', 2, 3, 1)
    for ri in [1, 2]:
        for ci, c in enumerate(['nombre', 'apellidos', 'telefono', 'correo']):
            regla('familia.contacto.' + c, 2, ri, ci)
    for ci, c in enumerate(['nombre', 'nacimiento', 'escolarizacion', 'centro']):
        regla('menores.persona.' + c, 3, 1, ci, repetir=True)
    regla('sociodemografia.estructura', 4, 0, 1)
    regla('sociodemografia.otras_formas', 4, 0, 1, 1)
    regla('sociodemografia.ingresos', 4, 1, 1, 1, modo='parrafo')
    for i, c in enumerate(['vivienda', 'situacion_laboral', 'situacion_economica', 'tramo_ingresos']):
        regla('sociodemografia.' + c, 4, 1, 1, i)
    for ri in range(1, 6):
        for ci, c in enumerate(['papel', 'nombre', 'parentesco', 'edad', 'estudios', 'otros_datos']):
            regla('sociodemografia.adulto.' + c, 5, ri, ci, modo='papel' if ci == 0 else '')
    for ri in range(1, 5):
        for ci, c in enumerate(['nombre', 'apellidos', 'parentesco', 'telefono']):
            regla('sociodemografia.persona_significativa.' + c, 6, ri, ci)
    for ri, campos in [(0, ['tiempo_convivencia', 'historia']), (1, ['descripcion', 'duracion']), (2, ['integracion', 'dificultades', 'personas_dificultad', 'motivo_dificultades'])]:
        for i, c in enumerate(campos):
            regla('valoracion.' + c, 7, ri, 1, i)
    regla('valoracion.crisis', 7, 1, 1, modo='casillas', opciones=['SI', 'NO'])
    regla('valoracion.actitud', 7, 3, 1, modo='casillas', opciones=['Colaborativa', 'Demandante', 'Pasiva', 'Hostil', 'Evitativa', 'Sin información / pendiente de valorar'])
    for ri, c in enumerate(['adecuacion', 'conciencia', 'motivacion', 'expectativas', 'pronostico'], 4):
        regla('valoracion.' + c, 7, ri, 1)
    for ri, c in enumerate(['global', 'maltrato_fisico', 'maltrato_psicologico', 'negligencia', 'abandono', 'violencia_sexual', 'incapacidad_parental', 'maltrato_prenatal', 'corrupcion', 'otras']):
        regla('valorame.' + c, 8, ri, 2)
    regla('valorame.descripcion', 8, 10, 1)
    for ri, c in enumerate(['antecedentes', 'dificultades_parentales', 'imposibilidad_proteccion', 'migrantes_sin_referentes', 'migrantes_vinculo_no_acreditado'], 11):
        regla('desproteccion.' + c, 8, ri, 2)
    for ri, ci, c in [(0, 1, 'servicio'), (1, 1, 'profesional'), (2, 1, 'telefono'), (2, 3, 'correo'), (3, 1, 'descripcion'), (4, 1, 'resultados')]:
        regla('intervenciones.' + c, 9, ri, ci)
    if numero == 4:
        for ri, ci, c in [(0, 1, 'existencia'), (1, 1, 'tipo'), (2, 1, 'organismo'), (3, 1, 'medidas'), (3, 3, 'fecha'), (4, 1, 'resultados')]:
            regla('resoluciones.' + c, 10, ri, ci)
    for ri, c in [(0, 'texto'), (1, 'documentacion_obligatoria'), (2, 'otra_documentacion')]:
        regla('observaciones.' + c, 10 if numero == 3 else 11, ri, 1, modo='papel' if ri == 1 else '')
    # El encabezado y las etiquetas quedan intactos; sólo se rellenan controles.
    menores = tablas[3].findall('w:tr', N)
    for _ in range(2):
        tablas[3].append(copy.deepcopy(menores[1]))
    controles_motor = {}
    for i, control in enumerate(raiz.findall('.//w:sdt', N), 100000):
        pr = control.find('w:sdtPr', N)
        identificador = pr.find('w:id', N)
        if identificador is None:
            identificador = E.SubElement(pr, f'{{{W}}}id')
        identificador.set(f'{{{W}}}val', str(i))
    valores = {
        'servicio.nombre': 'Centro de Servicios Sociales de Alhama' if numero == 3 else 'Servicio de Protección de Menores de Granada',
        'servicio.profesional': 'Elena Márquez Roldán', 'servicio.telefono': '958 000 142',
        'servicio.correo': 'emarquez@servicios.invalid', 'servicio.etf': 'ETF Alhama',
        'servicio.equipo_menores': 'Equipo Sierra Sur', 'servicio.subprograma': 'Reintegracion',
        'familia.direccion': 'Calle del Olivo 18, 18120 Alhama de Granada',
        'familia.contacto.nombre': ['Lucía', 'Daniel'], 'familia.contacto.apellidos': ['Ortega Salcedo', 'Pérez Molina'],
        'familia.contacto.telefono': ['600 000 142', ''], 'familia.contacto.correo': ['lucia.ortega@correo.invalid', ''],
        'menores.persona.nombre': ['Nerea Pérez Ortega', 'Álvaro Pérez Ortega', 'Celia Pérez Ortega', 'Mateo Pérez Ortega', 'Irene Pérez Ortega', 'Hugo Pérez Ortega'],
        'menores.persona.nacimiento': ['14/02/2016', '23/11/2012', '08/03/2014', '19/06/2018', '12/04/2020', '03/09/2022'],
        'menores.persona.escolarizacion': ['ed primaria', 'ed secundaria', 'ed primaria', 'ed primaria', 'ed infantil', 'ed infantil'],
        'menores.persona.centro': ['CEIP Sierra de Alhama', 'IES Alhama', 'CEIP Sierra de Alhama', '', 'CEIP Sierra de Alhama', 'CEIP Sierra de Alhama'],
        'sociodemografia.ingresos': '1450 euros mensuales', 'sociodemografia.otras_formas': '',
        'sociodemografia.adulto.nombre': ['Lucía Ortega Salcedo', 'Daniel Pérez Molina', 'Rosa Salcedo Ávila', '', ''],
        'sociodemografia.adulto.edad': ['39', '41', '67', '', ''],
        'sociodemografia.adulto.parentesco': ['Madre', 'Padre', 'Abuela', '', ''],
        'sociodemografia.adulto.estudios': ['Primarios', 'Secundarios', 'Sin estudios', 'Superiores', 'Analfabetismo'],
        'sociodemografia.adulto.otros_datos': ['Trabajo temporal en comercio', 'Búsqueda de empleo', 'Apoyo en cuidados', '', ''],
        'sociodemografia.persona_significativa.nombre': ['Marta', '', '', ''],
        'sociodemografia.persona_significativa.apellidos': ['Ortega Salcedo', '', '', ''],
        'sociodemografia.persona_significativa.parentesco': ['Tía materna', '', '', ''],
        'sociodemografia.persona_significativa.telefono': ['600 000 144', '', '', ''],
        'valoracion.tiempo_convivencia': '16 años', 'valoracion.historia': 'La familia reside en Alhama desde 2010.',
        'valoracion.descripcion': 'La familia solicita apoyo para organizar los cuidados y mejorar la asistencia escolar.',
        'valoracion.duracion': '8', 'valoracion.integracion': 'Mantiene relación con la asociación vecinal.',
        'valoracion.dificultades': 'Conflictos puntuales de convivencia.', 'valoracion.personas_dificultad': 'Ambos progenitores.',
        'valoracion.motivo_dificultades': 'Dificultades para conciliar horarios.',
        'valorame.descripcion': 'Necesita apoyo educativo y seguimiento familiar.',
        'intervenciones.servicio': 'Servicios Sociales Comunitarios', 'intervenciones.profesional': 'Elena Márquez Roldán',
        'intervenciones.telefono': '958 000 142', 'intervenciones.correo': 'emarquez@servicios.invalid',
        'intervenciones.descripcion': 'Seguimiento de la asistencia escolar y orientación familiar.',
        'intervenciones.resultados': 'Mejora inicial de la coordinación con el centro educativo.',
        'resoluciones.tipo': 'Resolución administrativa', 'resoluciones.organismo': 'Delegación Territorial de Granada',
        'resoluciones.fecha': '15/05/2025', 'resoluciones.resultados': 'Seguimiento de las medidas acordadas.',
        'observaciones.texto': 'La familia prefiere las citas por la tarde.', 'observaciones.otra_documentacion': 'Informe educativo e informe social.',
    }
    esperado = {}
    for r in perfil['Reglas']:
        filas = tablas[r['Tabla']].findall('w:tr', N)
        for ri in range(r['Fila'], len(filas) if r['Repetir'] else r['Fila'] + 1):
            c = filas[ri].findall('w:tc', N)[r['Celda']]
            codigo = r['Codigo']
            contador = len(esperado.get(codigo, []))
            v = valores.get(codigo, r['Opciones'][0] if r['Opciones'] else 'Apoyo familiar')
            if isinstance(v, list):
                v = v[contador]
            if r['Modo'] == 'papel':
                v = texto(c)
            elif r['Modo'] == 'parrafo':
                poner(c.findall('w:p', N)[r['Control']], r['Etiqueta'] + ' ' + v)
            elif r['Modo'] == 'casillas':
                v = r['Opciones'][0]
                controles = c.findall(f'.//{{{C}}}checked')
                if controles:
                    for i, x in enumerate(controles):
                        key = next(k for k in x.attrib if k.endswith('}val'))
                        x.set(key, '1' if i == 0 else '0')
                else:
                    ids = [x.get(f'{{{R}}}id') for x in c.findall('.//w:control', N)]
                    rel = E.fromstring(datos['word/_rels/document.xml.rels'])
                    for i, rid in enumerate(ids):
                        target = next(x.get('Target') for x in rel if x.get('Id') == rid)
                        nombre = 'word/' + target
                        ax = E.fromstring(datos[nombre])
                        for x in ax.findall(f'{{{A}}}ocxPr'):
                            if x.get(f'{{{A}}}name') == 'Value':
                                x.set(f'{{{A}}}value', '1' if i == 0 else '0')
                        datos[nombre] = E.tostring(ax, encoding='utf-8', xml_declaration=True)
            else:
                controles = c.findall('.//w:sdt', N)
                if controles:
                    clave = controles[r['Control']].find('w:sdtPr/w:id', N).get(f'{{{W}}}val')
                    controles_motor[clave] = v
                else:
                    for h in list(c):
                        if h.tag != f'{{{W}}}tcPr':
                            c.remove(h)
                    poner(E.SubElement(c, f'{{{W}}}p'), v)
            esperado.setdefault(codigo, []).append(ESTUDIOS.get(v, v) if codigo.endswith('adulto.estudios') else v)
    datos['word/document.xml'] = E.tostring(raiz, encoding='utf-8', xml_declaration=True)
    # Elimina propiedades de autoría heredadas; los datos de prueba son inventados.
    for nombre in ['docProps/core.xml', 'docProps/app.xml']:
        if nombre in datos:
            x = E.fromstring(datos[nombre])
            for h in list(x):
                x.remove(h)
            datos[nombre] = E.tostring(x, encoding='utf-8', xml_declaration=True)
    guardar(salida / f'anexo{numero}.base.docx', datos)
    (salida / f'anexo{numero}.controles.json').write_text(json.dumps(controles_motor, ensure_ascii=False))
    if numero == 3:
        for p in raiz.findall('./w:body/w:p', N):
            if texto(p).startswith('Anexo 3'):
                poner(p, 'Anexo 3 versión 2099')
                break
        datos['word/document.xml'] = E.tostring(raiz, encoding='utf-8', xml_declaration=True)
        guardar(salida / 'version_desconocida.base.docx', datos)
        (salida / 'version_desconocida.controles.json').write_text(json.dumps(controles_motor, ensure_ascii=False))
    return perfil, esperado

if __name__ == '__main__':
    a = argparse.ArgumentParser()
    a.add_argument('--plantillas', type=Path, required=True)
    a.add_argument('--salida', type=Path, required=True)
    args = a.parse_args()
    args.salida.mkdir(parents=True, exist_ok=True)
    perfiles, esperados = [], {}
    for numero in [3, 4]:
        perfil, esperado = construir(numero, args.plantillas, args.salida)
        perfiles.append(perfil)
        esperados[f'anexo{numero}'] = esperado
    (args.salida / 'perfiles.json').write_text(json.dumps(perfiles, ensure_ascii=False, indent=2) + '\n')
    (args.salida / 'esperado.json').write_text(json.dumps(esperados, ensure_ascii=False, indent=2) + '\n')
    proceso = subprocess.run(['go', 'test', './internal/adaptadores/anexos/docx', '-tags', 'generar_corpus', '-run', '^TestGenerarCorpus$', '-count=1'], env={**os.environ, 'ETF_DOCX_CORPUS': str(args.salida)}, capture_output=True, text=True)
    if proceso.returncode:
        raise RuntimeError(proceso.stdout + proceso.stderr)
