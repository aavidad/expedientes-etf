"""Convierte los Anexos 11 y 12 a plantillas DOCX reproducibles."""
from copy import deepcopy
import os
from pathlib import Path
import os
import re
import subprocess
import tempfile
from xml.etree import ElementTree as E
from zipfile import ZipFile

RAIZ = Path(__file__).resolve().parents[4]
W = '{http://schemas.openxmlformats.org/wordprocessingml/2006/main}'
X = '{http://www.w3.org/XML/1998/namespace}space'
ORIGENES = (
    ('Anexo 11 - (Finalización SubPrograma de Riesgo).odt', 'anexo11', 'Riesgo'),
    ('Anexo 12 - - (Finalización SubPrograma de Reintegración).odt', 'anexo12', 'Reintegración'),
)

TRADUCCIONES = {
    'INFORME DE FINALIZACIÓN DEL SUBPROGRAMA DE RIESGO': 'RISK SUBPROGRAMME COMPLETION REPORT',
    'INFORME DE FINALIZACIÓN DEL SUBPROGRAMA DE REINTEGRACIÓN': 'REUNIFICATION SUBPROGRAMME COMPLETION REPORT',
    'NÚMERO DE EXPEDIENTE': 'CASE NUMBER', 'Número expediente.de': 'Case number:',
    'Equipo PTF': 'ETF team', 'Fecha inicio': 'Start date', 'Fecha finalización': 'Completion date',
    'Equipo o recurso de procedencia de la familia:': 'Team or service that referred the family:',
    '1. DATOS DE IDENTIFICACIÓN FAMILIAR': '1. FAMILY IDENTIFICATION DETAILS',
    'Información de contacto de la unidad familiar': 'Family contact details',
    'Nombre': 'First name', 'Apellidos': 'Surname', 'Teléfono': 'Telephone',
    'Dirección': 'Address', 'Dirección completa.': 'Full address.',
    'Personas menores de edad por las que la familia ingresó en el programa': 'Children for whom the family entered the programme',
    'Nombre y apellidos': 'Full name', 'Fecha de nacimiento': 'Date of birth',
    'Nivel de escolarización': 'School year', 'Centro educativo': 'School',
    '2. MOTIVO DE LA FINALIZACIÓN DEL INTERVENCIÓN': '2. REASON FOR ENDING THE INTERVENTION',
    ' Finaliza la intervención a propuesta del/de la profesional con los objetivos satisfactoriamente cumplidos.':
        ' The professional proposes ending the intervention because the objectives have been fully achieved.',
    ' Finaliza la intervención a propuesta del/de la profesional con los objetivos parcialmente cumplidos.':
        ' The professional proposes ending the intervention because the objectives have been partly achieved.',
    ' Finaliza la intervención a propuesta del/de la profesional sin los objetivos logrados y sin existencia de mejora por falta de adherencia a la intervención.':
        ' The professional proposes ending the intervention because the objectives have not been achieved and there has been no improvement due to limited engagement.',
    ' La familia abandona la intervención por iniciativa propia.': ' The family chooses to stop the intervention.',
    ' Finaliza la intervención por traslado familiar.': ' The intervention ends because the family has moved.',
    ' Finaliza la intervención por mayoría de edad de los hijos/as.': ' The intervention ends because the children have reached adulthood.',
    ' Finaliza la intervención por ingreso en centro de la/s personas menor/es de edad.': ' The intervention ends because the child or children have entered residential care.',
    'Otros motivos de cierre: ': 'Other reason for ending the intervention: ',
    'La familia es derivada a otro recurso': 'The family is referred to another service',
    '   SÍ           NO': '   YES           NO',
    'Si la respuesta es SÍ, especificar el recurso/servicio, motivo de la derivación y orientaciones para la próxima intervención:.':
        'If yes, specify the service, reason for referral and guidance for the next intervention:.',
    '3. EVALUACIÓN DE LA INTERVENCIÓN ': '3. INTERVENTION ASSESSMENT ',
    'Datos de la intervención ': 'Intervention details ',
    'Tiempo total de intervención': 'Total intervention period', 'Número de sesiones totales': 'Total number of sessions',
    'Número de sesiones con figuras parentales': 'Sessions with parental figures',
    'Número de sesiones con niños y adolescentes': 'Sessions with children and adolescents',
    'Valoración de la situación familiar': 'Family situation assessment',
    'Riesgo de desprotección: VALÓRAME': 'Risk of harm: VALÓRAME',
    'EVALUACIÓN INICIAL': 'INITIAL ASSESSMENT', 'EVALUACIÓN INTERMEDIA': 'INTERIM ASSESSMENT',
    'EVALUACIÓN FINAL': 'FINAL ASSESSMENT', 'Valoración global de gravedad': 'Overall severity rating',
    'EVALUACIÓN': 'ASSESSMENT', 'INICIAL': 'INITIAL', 'INTERMEDIA': 'INTERIM', 'SÍ': 'YES',
    'Maltrato físico': 'Physical abuse', 'Maltrato psicológico/emocional': 'Psychological or emotional abuse',
    'Negligencia': 'Neglect', 'Abandono': 'Abandonment', 'Violencia sexual': 'Sexual abuse',
    'Incapacidad parental control': 'Parental inability to control', 'Maltrato prenatal': 'Prenatal abuse',
    'Corrupción': 'Corruption', 'Otras': 'Other',
    'Descripción de los resultados obtenidos tras la intervención: Descripción cualitativa':
        'Description of outcomes after the intervention: qualitative description',
    'Descripción de los resultados: Descripción cualitativa': 'Description of outcomes: qualitative description',
    'Puntuación.': 'Score.', 'Puntuación glocal': 'Overall score',
    'Actitudes y expectativas parentales inadecuadas AAPI-2.5': 'Inadequate parental attitudes and expectations AAPI-2.5',
    'Expectativas inadecuadas': 'Inadequate expectations',
    'Falta de empatía hacia las necesidades infantiles': 'Lack of empathy for children’s needs',
    'Creencias sobre el uso y el valor del castigo corporal': 'Beliefs about the use and value of corporal punishment',
    'Roles invertidos': 'Reversed roles', 'Opresión de la autonomía': 'Restriction of autonomy',
    'Competencias parentales ECP 12': 'Parental skills ECP 12',
    'Estimulación y estructuración': 'Stimulation and structure', 'Tiempo compartido en familia': 'Family time together',
    'Corresponsabilidad parental': 'Shared parenting responsibility',
    'Creencias/Expectativas desarrollo infantil': 'Beliefs and expectations about child development',
    'Afecto, comunicación y aceptación': 'Affection, communication and acceptance',
    'Implicación escolar': 'School involvement', 'Normas y supervisión': 'Rules and supervision',
    'Autorregulación emocional': 'Emotional self-regulation', 'Percepción adecuada del rol parental': 'Understanding of the parental role',
    'Afrontamiento de situaciones estresantes': 'Coping with stressful situations', 'Apoyo social': 'Social support',
    'Gestión doméstica': 'Household management',
    'Funcionamiento familiar SCORE 15': 'Family functioning SCORE 15',
    'Fortalezas y adaptación': 'Strengths and adaptation', 'Preocupación o angustia': 'Worry or distress',
    'Dificultad para comunicase': 'Communication difficulties',
    'Puntuación global funcionamiento familiar': 'Overall family functioning score',
    'Ajuste infantil y adolescente SDQ – Versión autoinformada': 'Child and adolescent adjustment SDQ – self-report version',
    'Ajuste infantil y adolescente SDQ – Versión informada por figuras parentales': 'Child and adolescent adjustment SDQ – parent-report version',
    'Síntomas emocionales': 'Emotional symptoms', 'Problemas de conducta': 'Conduct problems',
    'Hiperactividad/desatención': 'Hyperactivity/inattention', 'Problemas con los iguales': 'Peer problems',
    'Conducta prosocial': 'Prosocial behaviour', 'Puntuación total de dificultades': 'Total difficulties score',
    'Alianza profesional SOATIF – Versión para familias': 'Professional alliance SOATIF – family version',
    'Alianza profesional SOATIF – Versión para profesionales': 'Professional alliance SOATIF – professional version',
    'Enganche proceso': 'Engagement in the process', 'Conexión emocional': 'Emotional connection',
    'Seguridad': 'Safety', 'Compartir propósito': 'Shared purpose',
    'Puntuación total alianza profesional': 'Overall professional alliance score',
    'Puntuación total alianza terapéutica': 'Overall therapeutic alliance score',
    'Satisfacción con la intervención': 'Satisfaction with the intervention',
    'Figuras parentales': 'Parental figures', 'Niños, niñas y adolescentes': 'Children and adolescents',
    'Evaluación complementaria': 'Additional assessment',
    'Otros: indicadores socioeconómicos, psicosociales, de salud, educativos… (especificar)':
        'Other socioeconomic, psychosocial, health or education indicators (specify)',
    'Especificar o describir otros indicadores que se consideren relevantes .': 'Specify or describe any other relevant indicators.',
    'Coordinación con otros servicios': 'Coordination with other services',
    'Unidades de salud mental': 'Mental health units', 'Centros de salud/hospitales': 'Health centres/hospitals',
    'Atención Temprana': 'Early intervention', 'Centros/programas de atención a drogodependencia': 'Drug treatment centres/programmes',
    'Centros educativos': 'Schools', 'Asociaciones/ONG': 'Associations/NGOs',
    'Centros municipales de información a la mujer': 'Municipal women’s information centres',
    'Centros de valoración y orientación de personas con discapacidad': 'Disability assessment and guidance centres',
    'Valoración de la dependencia': 'Dependency assessment', 'Otro:': 'Other:',
    '4. OBJETIVOS ABORDADOS EN LA INTERVENCIÓN': '4. OBJECTIVES ADDRESSED DURING THE INTERVENTION',
    'Objetivos': 'Objectives', 'Consecución': 'Achievement', 'Trabajado con': 'Worked on with',
    'Actuaciones': 'Actions', 'Objetivos en proceso o no resueltas': 'Objectives in progress or unresolved',
    'RecogidoPIF': 'Included in the family intervention plan', 'Recogido': 'Included in',
    'Indicar resurso.': 'Specify service.', 'Ayuda requerida': 'Support needed',
    'Derivación': 'Referral', 'Elija un elemento.': 'Select an option.', 'Elija.': 'Select.',
    'Haga clic o pulse aquí para escribir texto.': 'Click or tap here to enter text.',
    'Nombre.': 'Name.', 'Apellidos.': 'Surname.', 'tlfn.': 'Telephone.', 'tfln.': 'Telephone.',
    'SÍ': 'YES', 'Haga clic o pulse aquí para escribir texto': 'Click or tap here to enter text',
    'dd/mm/aa.': 'dd/mm/yy.', 'Nombre centro educativo.': 'School name.',
    '(RI1) Potenciar los factores de protección y eliminar o reducir los factores de riesgo relacionados con la situación de riesgo o desprotección de las personas menores de edad.':
        ' (RI1) Strengthen protective factors and eliminate or reduce risk factors related to the child’s situation of risk or harm.',
    ' (RI2) Capacitar a las familias para asegurar un adecuado funcionamiento familiar que promueva un entorno seguro y protector y evite situaciones de violencia.':
        ' (RI2) Help families develop healthy family functioning that promotes a safe, protective environment and prevents violence.',
    ' (RI3) Adquirir y fortalecer las competencias necesarias para un ejercicio positivo y responsable de la parentalidad de las personas progenitoras o guardadoras que ejerzan el cuidado de la persona menor de edad.':
        ' (RI3) Develop the skills needed for positive, responsible parenting by the people caring for the child.',
    '(RI4) Promover, mejorar y recuperar el ajuste personal y desarrollo saludable de las personas menores de edad, favoreciendo su integración familiar, educativo y social.':
        ' (RI4) Promote and improve children’s wellbeing and healthy development, supporting their family, school and social integration.',
    '(RE1) Facilitar las condiciones necesarias para que se produzca la reunificación contando con la implicación de todos los agentes, entidades y servicios relacionados.':
        ' (RE1) Create the conditions for reunification with the involvement of all relevant people, organisations and services.',
    '(RE2) Reducir y/o eliminar las causas y problemáticas que motivaron la medida de protección para garantizar un entorno familiar seguro, protector y sin violencia.':
        ' (RE2) Reduce or remove the issues that led to the protective measure and ensure a safe, supportive family environment free from violence.',
    '(RE3) Adquirir y fortalecer las competencias necesarias para un ejercicio positivo y responsable de la parentalidad de las personas progenitoras o guardadoras que ejercerán el rol parental.':
        ' (RE3) Develop the skills needed for positive, responsible parenting by the people who will take on the parental role.',
    '(RE4) Promover, mejorar y recuperar el ajuste personal y desarrollo saludable de las personas menores de edad.':
        ' (RE4) Promote and improve children’s wellbeing and healthy development.',
    '(RE5) Facilitar el proceso de transición para garantizar la adaptación de la persona menor de edad en el medio familiar, educativo y social en el que se va a integrar.':
        ' (RE5) Support the child’s transition and adjustment to their new family, school and social environment.',
    'Otro: Haga clic o pulse aquí para escribir texto.': 'Other: Click or tap here to enter text.',
}


def texto(nodo):
    return ''.join(t.text or '' for t in nodo.iter(W + 't'))


def poner(nodo, valor):
    textos = list(nodo.iter(W + 't'))
    if textos:
        textos[0].text = valor
        textos[0].set(X, 'preserve')
        for t in textos[1:]:
            t.text = ''
    else:
        run = E.SubElement(nodo, W + 'r')
        t = E.SubElement(run, W + 't', {X: 'preserve'})
        t.text = valor


def sustituir(nodo, viejo, nuevo):
    actual = texto(nodo)
    if viejo not in actual:
        raise ValueError(f'No aparece el texto esperado: {viejo!r}')
    poner(nodo, actual.replace(viejo, nuevo, 1))


def deshacer_sdt_marcados(nodo):
    for padre in list(nodo.iter()):
        for hijo in list(padre):
            if hijo.tag != W + 'sdt' or '{{' not in texto(hijo):
                continue
            contenido = hijo.find(W + 'sdtContent')
            if contenido is None:
                continue
            indice = list(padre).index(hijo)
            padre.remove(hijo)
            for nieto in list(contenido):
                padre.insert(indice, nieto)
                indice += 1


def deshacer_todos_sdt(nodo):
    for padre in list(nodo.iter()):
        for hijo in list(padre):
            if hijo.tag != W + 'sdt':
                continue
            contenido = hijo.find(W + 'sdtContent')
            if contenido is None:
                continue
            indice = list(padre).index(hijo)
            padre.remove(hijo)
            for nieto in list(contenido):
                padre.insert(indice, nieto)
                indice += 1


def celdas(fila):
    return fila.findall(W + 'tc')


def filas(tabla):
    return tabla.findall(W + 'tr')


def tabla_con(raiz, frase):
    for tabla in raiz.iter(W + 'tbl'):
        if frase in texto(tabla):
            return tabla
    raise ValueError(f'No se encuentra la tabla {frase!r}')


def parrafo_con(raiz, frase):
    for parrafo in raiz.iter(W + 'p'):
        if frase in texto(parrafo):
            return parrafo
    raise ValueError(f'No se encuentra el párrafo {frase!r}')


def marcador_objetivos(raiz, idioma):
    titulo = parrafo_con(raiz, '4. OBJETIVOS ABORDADOS EN LA INTERVENCIÓN')
    padre = next(p for p in raiz.iter() if titulo in list(p))
    pos = list(padre).index(titulo) + 1
    p = E.Element(W + 'p')
    r = E.SubElement(p, W + 'r')
    t = E.SubElement(r, W + 't', {X: 'preserve'})
    t.text = ('Objectives addressed: {{objetivos_abordados}}. Objectives pending: {{objetivos_pendientes}}.'
              if idioma == 'en' else
              'Objetivos abordados: {{objetivos_abordados}}. Objetivos pendientes: {{objetivos_pendientes}}.')
    padre.insert(pos, p)


def marcador_motivo_especial(raiz, idioma):
    titulo = parrafo_con(raiz, '2. MOTIVO DE LA FINALIZACIÓN')
    padre = next(p for p in raiz.iter() if titulo in list(p))
    pos = list(padre).index(titulo) + 1
    p = E.Element(W + 'p')
    r = E.SubElement(p, W + 'r')
    t = E.SubElement(r, W + 't', {X: 'preserve'})
    t.text = ('Reason for ending the intervention: {{motivo}}' if idioma == 'en'
              else 'Motivo de finalización de la intervención: {{motivo}}')
    padre.insert(pos, p)


def parametrizar(raiz, numero_anexo, idioma):
    tablas = raiz.find(W + 'body')
    tabla_cabecera = tabla_con(raiz, 'NÚMERO DE EXPEDIENTE')
    for fila in filas(tabla_cabecera):
        cs = celdas(fila)
        if not cs:
            continue
        if 'Número expediente' in texto(fila):
            for celda in cs:
                if 'Elija un elemento.' in texto(celda):
                    sustituir(celda, 'Elija un elemento.', '{{numero}}')
                    break
        if 'Equipo PTF' in texto(fila):
            for celda in cs:
                if 'Haga clic o pulse aquí para escribir texto.' in texto(celda):
                    sustituir(celda, 'Haga clic o pulse aquí para escribir texto.', '{{equipo}}')
                    break
        if 'Fecha finalización' in texto(fila):
            if 'dd/mm/aa.' in texto(cs[-1]):
                sustituir(cs[-1], 'dd/mm/aa.', '{{fecha_documento}}')
    tabla_contacto = tabla_con(raiz, 'Dirección completa.')
    familia_puesta = False
    for fila in filas(tabla_contacto):
        cs = celdas(fila)
        if len(cs) >= 2 and 'Nombre.' in texto(fila) and not familia_puesta:
            sustituir(cs[0], 'Nombre.', '{{familia}}')
            familia_puesta = True
        if len(cs) >= 2 and 'Dirección completa.' in texto(fila):
            sustituir(cs[-1], 'Dirección completa.', '{{direccion}}')

    tabla_menores = tabla_con(raiz, 'Nombre y apellidos')
    filas_menores = filas(tabla_menores)
    fila_modelo = deepcopy(filas_menores[1])
    celdas_modelo = celdas(fila_modelo)
    for celda, viejo, nuevo in (
        (celdas_modelo[0], 'Nombre y apellidos.', '{{#menores}}{{nombre}}'),
        (celdas_modelo[1], 'dd/mm/aa.', '{{nacimiento}}'),
    ):
        sustituir(celda, viejo, nuevo)
    poner(celdas_modelo[-1], '{{/menores}}')
    deshacer_todos_sdt(fila_modelo)
    for celda in celdas(fila_modelo)[2:-1]:
        poner(celda, '')
    for fila in filas_menores[1:]:
        tabla_menores.remove(fila)
    tabla_menores.append(fila_modelo)

    opciones = {
        'objetivos satisfactoriamente cumplidos.': 'motivo_obj_total',
        'objetivos parcialmente cumplidos.': 'motivo_obj_parcial',
        'sin existencia de mejora por falta de adherencia a la intervención.': 'motivo_sin_adherencia',
        'La familia abandona la intervención': 'motivo_abandono',
        'por traslado familiar.': 'motivo_traslado',
        'por mayoría de edad de los hijos/as.': 'motivo_mayoria_edad',
        'por ingreso en centro de la/s personas menor/es de edad.': 'motivo_ingreso_centro',
    }
    for frase, campo in opciones.items():
        p = next((parrafo for parrafo in raiz.iter(W + 'p') if frase in texto(parrafo)), None)
        if p is None:
            continue
        poner(p, '{{' + campo + '}}' + texto(p))
    p_otro = parrafo_con(raiz, 'Otros motivos de cierre:')
    poner(p_otro, '{{motivo_otros}}' + texto(p_otro).replace('especificar motivos.', '{{motivo_texto}}.', 1))
    marcador_motivo_especial(raiz, idioma)

    tabla_derivacion = tabla_con(raiz, 'La familia es derivada a otro recurso')
    for fila in filas(tabla_derivacion):
        if 'Si la respuesta' in texto(fila):
            for celda in celdas(fila):
                if 'Si la respuesta' in texto(celda):
                    poner(celda, texto(celda) + ' {{derivacion_tras_cierre}}')
                    break
            break
    tabla_coordinacion = tabla_con(raiz, 'Coordinación con otros servicios')
    for parrafo in list(tabla_coordinacion.iter(W + 'p')):
        if 'Haga clic o pulse aquí para escribir texto.' in texto(parrafo):
            sustituir(parrafo, 'Haga clic o pulse aquí para escribir texto.', '{{coordinacion_recursos}}')
            break
    marcador_objetivos(raiz, idioma)
    deshacer_sdt_marcados(raiz)


def traducir(raiz):
    for parrafo in raiz.iter(W + 'p'):
        actual = texto(parrafo)
        traducido = actual
        for es, en in sorted(TRADUCCIONES.items(), key=lambda par: len(par[0]), reverse=True):
            if es in traducido:
                traducido = traducido.replace(es, en)
        if traducido != actual:
            poner(parrafo, traducido)


def limpiar_active_x(raiz, xml):
    for p in raiz.iter(W + 'p'):
        for run in list(p):
            if run.tag == W + 'r' and run.find(W + 'object') is not None:
                p.remove(run)
    apertura = re.search(rb'<w:document\b[^>]*>', xml).group()
    salida = E.tostring(raiz, encoding='utf-8', xml_declaration=True)
    salida = re.sub(rb'<w:document\b[^>]*>', lambda _: apertura, salida, count=1)
    return salida


def convertir(origen, nombre, titulo, idioma):
    ruta_origen = Path(os.environ.get('ETF_PLANTILLAS_FUENTE', str(RAIZ / 'bin/plantillas-fuente'))) / origen
    entorno = os.environ.copy()
    entorno.pop('LD_LIBRARY_PATH', None)
    with tempfile.TemporaryDirectory(prefix='etf-finalizacion-') as tmp:
        perfil = (Path(tmp) / 'perfil').as_uri()
        subprocess.run(['soffice', f'-env:UserInstallation={perfil}', '--headless', '--convert-to', 'docx',
                        '--outdir', tmp, str(ruta_origen)],
                       check=True, env=entorno, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        fuente = Path(tmp) / (ruta_origen.stem + '.docx')
        with ZipFile(fuente) as z:
            xml = z.read('word/document.xml')
            apertura = re.search(rb'<w:document\b[^>]*>', xml).group()
            for prefijo, uri in re.findall(rb'xmlns:([^= ]+)="([^"]+)"', apertura):
                E.register_namespace(prefijo.decode(), uri.decode())
            raiz = E.fromstring(xml)
            parametrizar(raiz, nombre, idioma)
            if idioma == 'en':
                traducir(raiz)
            salida_xml = limpiar_active_x(raiz, xml)
            destino = RAIZ / 'idiomas' / idioma / 'plantillas' / 'finalizacion' / nombre / 'v1.0.0.docx'
            destino.parent.mkdir(parents=True, exist_ok=True)
            with ZipFile(destino, 'w') as out:
                for parte in z.infolist():
                    if parte.filename.startswith('word/activeX/'):
                        continue
                    datos = z.read(parte.filename)
                    if parte.filename == 'word/document.xml':
                        datos = salida_xml
                    elif parte.filename == 'word/_rels/document.xml.rels':
                        datos = re.sub(rb'<Relationship\b[^>]*Type="[^"]*/control"[^>]*/>', b'', datos)
                    elif parte.filename == '[Content_Types].xml':
                        datos = re.sub(rb'<Override\b[^>]*PartName="/word/activeX/[^>]*?/>', b'', datos)
                    estable = deepcopy(parte)
                    estable.date_time = (1980, 1, 1, 0, 0, 0)
                    estable.extra = b''
                    estable.comment = b''
                    out.writestr(estable, datos)


if __name__ == '__main__':
    for archivo, clave, titulo in ORIGENES:
        for idioma in ('es', 'en'):
            convertir(archivo, clave, titulo, idioma)
