#!/usr/bin/env python3
"""Añade marcadores a la plantilla de la Junta y conserva sus tablas y estilos."""
import copy
import json
import pathlib
import re
import zipfile
import xml.etree.ElementTree as ET

RAIZ = pathlib.Path(__file__).resolve().parents[4]
W = 'http://schemas.openxmlformats.org/wordprocessingml/2006/main'
NS = {'w': W}
ET.register_namespace('w', W)


def texto(n):
    return ''.join(e.text or '' for e in n.findall('.//w:t', NS)).strip()


def poner(n, marca):
    parrafos = n.findall('w:p', NS) if n.tag == '{'+W+'}tc' else [n]
    if not parrafos:
        parrafos = [ET.SubElement(n, '{'+W+'}p')]
    p = parrafos[0]
    textos = p.findall('.//w:t', NS)
    if not textos:
        textos = [ET.SubElement(ET.SubElement(p, '{'+W+'}r'), '{'+W+'}t')]
    textos[0].text = '{{'+marca+'}}'
    for t in textos[1:]:
        t.text = ''
    for extra in parrafos[1:]:
        n.remove(extra)


def parrafo(marca):
    p = ET.Element('{'+W+'}p')
    poner(p, marca)
    return p


def generar():
    fuente = Path(os.environ.get('ETF_PLANTILLAS_FUENTE', str(RAIZ / 'bin/plantillas-fuente'))) / 'Memoria Técnica a remitir a la Junta.docx'
    z = zipfile.ZipFile(fuente)
    doc = ET.fromstring(z.read('word/document.xml'))
    body = doc.find('w:body', NS)
    originales = []
    for p in doc.findall('.//w:p', NS):
        valor=texto(p)
        if valor and valor not in originales:
            originales.append(valor)
    claves = {v:f'texto_{i}' for i,v in enumerate(originales)}
    for p in doc.findall('.//w:p', NS):
        s = texto(p)
        if s:
            poner(p, claves[s])
    tablas = body.findall('w:tbl', NS)[1:]
    apartados = ['1','2','3','4','5','6','7','8','9','10','11.1','11.2','12.1','12.2','13.1','13.2','13.3','14','14.1','16','17','18','19','20','21','22','23','24','25','26','27','28']
    iniciales = [3,2,2,2,2,1,2,3,2,1,2,2,2,2,1,1,2,2,2,1,1,1,2,2,2,1,1,1,3,2,2,1]
    campos = []
    sexos = ['nino','nina']
    subprogramas = ['riesgo','reunificacion','']
    categorias = {
        '14':['SALUD_MENTAL','SALUD','ATENCION_TEMPRANA','ADICCIONES','EDUCACION','AYUDAS_FAMILIARES','VIOLENCIA_SEXUAL','SPM','ONG','MUJER','DISCAPACIDAD','DEPENDENCIA','FISCALIA','OTROS'],
        '16':['juzgados','fiscalia','spaf','spm','educacion','sanidad','servicios_sociales','otros',''],
        '17':['BIPARENTAL','MONOPARENTAL_PADRE','MONOPARENTAL_MADRE','CUSTODIA_COMPARTIDA','SIN_CUSTODIA_COMPARTIDA','RECONSTITUIDA','EXTENSA','OTRAS',''],
        '19':['edad_menos_20','edad_20_29','edad_30_39','edad_40_49','edad_50_mas',''],
        '20':['ANALFABETISMO','SIN_OBLIGATORIOS','OBLIGATORIOS','SECUNDARIOS','UNIVERSITARIOS',''],
        '21':['AUTONOMO','EMPLEO_FIJO','EMPLEO_TEMPORAL','ECONOMIA_SUMERGIDA','SIN_EMPLEO','PENSIONISTA','VARIAS','OTRAS',''],
        '22':['SIN_DIFICULTADES','DIFICULTADES_SIN_NECESIDADES_BASICAS','DIFICULTADES_NECESIDADES_BASICAS',''],
        '23':['MENOS_50','DE_50_A_100','DE_100_A_150','DE_150_A_200','MAS_200',''],
        '24':[f'FAMILIA_{i:02d}' for i in range(1,24)],
        '26':[f'MENOR_{i:02d}' for i in range(1,12)],
        '27':[f'DESPROTECCION_{i:02d}' for i in range(1,13)],
        '28':['PADRE_MADRE','PADRE','MADRE','PAREJA','TUTOR','ABUELO','TIO','HERMANO','PRIMO','FAMILIAR','ALLEGADO'],
    }
    for i,(tabla,a,inicial) in enumerate(zip(tablas,apartados,iniciales),1):
        filas = tabla.findall('w:tr', NS)
        for ri,fila in enumerate(filas):
            if ri < inicial:
                continue
            celdas = fila.findall('w:tc',NS)
            # La tabla 25 termina con filas de totales y cabeceras combinadas.
            for ci,celda in enumerate(celdas):
                if a == '25' and ri == 6 and ci >= 9:
                    continue
                if ci == 0 and a != '14.1':
                    continue
                marca=f'celda_{i}_{ri}_{ci}'
                filtros={}; unidad='familias'; porcentaje=False; total=False
                r=ri-inicial; c=ci-1
                if a in ['1','8']:
                    if subprogramas[min(r,2)]:filtros['subprograma']=subprogramas[r]
                    bloque=c//4;unidad=['familias','menores'][(c//2)%2];porcentaje=c%2==1
                    if bloque<2:filtros['origen' if a=='1' else 'motivo']=(['anterior','incorporado'] if a=='1' else ['objetivos_cumplidos','objetivos_parciales'])[bloque]
                elif a=='2':
                    if subprogramas[min(r,2)]:filtros['subprograma']=subprogramas[r]
                    if c//2<2:filtros['sexo']=sexos[c//2]
                    unidad='menores';porcentaje=c%2==1
                elif a=='3':
                    if subprogramas[min(r,2)]:filtros['subprograma']=subprogramas[r]
                    unidad=['familias','menores'][c//2];filtros['estado']='si';porcentaje=c%2==1
                elif a in ['4','5']:
                    if subprogramas[min(r,2)]:filtros['subprograma']=subprogramas[r]
                    if c<3:filtros['estado']=['SSSSCC','SPM','OTRO_ETF'][c]
                elif a=='6':
                    unidad=['familias','menores'][r]
                    if c<2:filtros['subprograma']=subprogramas[c]
                elif a=='7':
                    if subprogramas[min(r,2)]:filtros['subprograma']=subprogramas[r]
                    if c//2<2:filtros['origen']=['anterior','incorporado'][c//2]
                    porcentaje=c%2==1
                elif a=='9':
                    if subprogramas[min(r,2)]:filtros['subprograma']=subprogramas[r]
                    if c<8:filtros['motivo']=['rechazo','no_adherencia','traslado','mayoria_edad','propuesta_dsr','medida_protectora','ingreso_centro','otras_causas'][c]
                    porcentaje=c==9
                elif a=='10':
                    if subprogramas[min(r,2)]:filtros['subprograma']=subprogramas[r]
                    filtros['estado']='si';porcentaje=c%2==1
                elif a in ['11.1','11.2','12.1','13.3']:
                    unidad=['familias','menores'][r];porcentaje=c%2==1
                    filtros['subprograma']='reunificacion' if a=='13.3' else 'riesgo'
                    if a!='13.3':filtros['estado']='si'
                elif a=='12.2':
                    if c//2<3:filtros['estado']=['inicio_desamparo_o_medida','a_la_espera','no_necesaria'][c//2]
                    porcentaje=c%2==1
                elif a in ['13.1','13.2']:
                    filtros['subprograma']='riesgo' if a=='13.1' else 'reunificacion'
                    if r<4:filtros['tramo']=str(r)
                    porcentaje=c%2==1
                elif a=='14.1':
                    unidad='menores';c=ci;porcentaje=c%2==1;filtros['estado']='si'
                    if c//2<2:filtros['sexo']=sexos[c//2]
                elif a=='25':
                    unidad='menores';porcentaje=c%2==1
                    if ri<=5:filtros['tramo']=['0_5','6_11','12_17'][r]
                    if ri<=6 and c//4<2:filtros['subprograma']=subprogramas[c//4]
                    filtros['sexo']=sexos[(c//2)%2]
                else:
                    grupo=categorias.get(a,[]);valor=grupo[r] if r<len(grupo) else ''
                    dimension='tramo' if a in ['19','23'] else 'organismo' if a=='16' else 'estado'
                    if valor:filtros[dimension]=valor
                    porcentaje=c%2==1 and a!='16'
                    if a in ['14','26','27']:
                        unidad='menores'
                        if c//2<2:filtros['sexo']=sexos[c//2]
                    elif a in ['19','20','21']:
                        unidad='adultos';filtros['sexo']=sexos[c//2]
                    elif a=='16':unidad='informes';total=not valor
                    elif a=='18':filtros['estado']='si'
                poner(celda,marca)
                campos.append({'campo':marca,'apartado':a,'unidad':unidad,'filtros':filtros,'porcentaje':porcentaje,'total':total})
        clave=a.replace('.','_')
        pos=list(body).index(tabla)+1
        for marca in reversed([f'detalle_{clave}',f'aviso_{clave}']):
            body.insert(pos,parrafo(marca))
    # Datos generales y narrativas son negocio, separados de los textos fijos.
    insert = 8
    for marca in ['cabecera_a','datos_a','personal_a','observaciones_a']:
        body.insert(insert,parrafo(marca));insert+=1
    for p in body.findall('w:p',NS):
        if texto(p)=='{{texto_99}}':
            body.insert(list(body).index(p)+1,parrafo('cifra_15'))
    campos.append({'campo':'cifra_15','apartado':'15','unidad':'comisiones','filtros':{},'porcentaje':False,'total':True})
    pos=len(body)-1
    for letra in ['d','e','f','g']:
        body.insert(pos,parrafo('cabecera_'+letra));pos+=1
        body.insert(pos,parrafo('narrativa_'+letra));pos+=1
    payload=ET.tostring(doc,encoding='utf-8',xml_declaration=True)
    # Conservar los prefijos citados por mc:Ignorable aunque no tengan nodos.
    for prefijo,uri in re.findall(rb'xmlns:([\w]+)="([^"]+)"',z.read('word/document.xml')):
        if b'xmlns:'+prefijo+b'=' not in payload:
            payload=payload.replace(b'<w:document ',b'<w:document xmlns:'+prefijo+b'="'+uri+b'" ',1)
    for idioma in ['es','en']:
        carpeta=RAIZ/f'idiomas/{idioma}/plantillas/memoria'
        etiquetas=json.loads((carpeta/'textos-v1.json').read_text())
        extra=json.loads((carpeta/'extras-v1.json').read_text())
        etiquetas.update(extra)
        for a in apartados:
            k=a.replace('.','_')
            etiquetas['detalle_'+k]=extra['etiqueta_sin_dato']+' {{sin_dato_'+k+'}}; '+extra['etiqueta_sin_relacion']+' {{sin_relacion_'+k+'}}'
        # Los marcadores internos del detalle se expanden en el motor de Memoria.
        (carpeta/'v1.0.0.json').write_text(json.dumps({'campos':etiquetas,'celdas':campos},ensure_ascii=False,indent=2)+'\n')
        with zipfile.ZipFile(carpeta/'v1.0.0.docx','w') as salida:
            for entrada in z.infolist():
                salida.writestr(copy.copy(entrada),payload if entrada.filename=='word/document.xml' else z.read(entrada.filename))


if __name__ == '__main__':
    generar()
