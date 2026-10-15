#!/usr/bin/env python3
"""Comprueba el curso en Chrome del sistema, sin guardar credenciales ni sesiones."""
import argparse
from functools import partial
import html
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import re
import tempfile
import threading
import xml.etree.ElementTree as ET
import zipfile

from playwright.sync_api import sync_playwright

RAIZ = Path(__file__).resolve().parents[1]
SALIDA = RAIZ / 'bin/curso'


def responder(page):
    page.locator('.FormView_question').first.wait_for()
    datos = page.locator('.idevice_node.form').get_attribute('data-idevice-json-data')
    if not datos:
        datos = page.evaluate('JSON.stringify(Object.values($form.instances)[0])')
    preguntas = json.loads(datos)['questionsData']
    for i, pregunta in enumerate(preguntas):
        elemento = page.locator('.FormView_question').nth(i)
        tipo = pregunta['activityType']
        if tipo == 'selection':
            indice = next(n for n, respuesta in enumerate(pregunta['answers']) if respuesta[0 if isinstance(respuesta, list) else '0'])
            elemento.locator('input[type=radio]').nth(indice).check()
        elif tipo == 'true-false':
            elemento.locator('input[type=radio]').nth(0 if pregunta['answer'] == '1' else 1).check()
        elif tipo == 'fill':
            respuestas = re.findall(r'<u>(.*?)</u>', pregunta['baseText'], re.S)
            for n, respuesta in enumerate(respuestas):
                elemento.locator('input[type=text]').nth(n).fill(html.unescape(re.sub('<[^>]+>', '', respuesta)))
    page.get_by_role('button', name='Comprobar', exact=True).click()
    texto = page.locator('.form-results-container').inner_text()
    assert f'({len(preguntas)}/{len(preguntas)})' in texto, texto
    return len(preguntas)


def importar(page, credenciales, resumen):
    page.set_viewport_size({'width': 1440, 'height': 1000})
    valores = {}
    for linea in credenciales.read_text(encoding='utf-8').splitlines():
        m = re.match(r'^(ADMIN_EMAIL|ADMIN_PASSWORD)=(.*)$', linea)
        if m:
            valores[m[1]] = m[2].strip().strip('\"\'')
    try:
        page.goto('http://127.0.0.1:18090')
        page.get_by_role('textbox', name='Correo electrónico').fill(valores['ADMIN_EMAIL'])
        page.get_by_role('textbox', name='Contraseña').fill(valores['ADMIN_PASSWORD'])
        page.get_by_role('button', name='Iniciar sesión').click()
    except Exception:
        raise SystemExit('No se pudo iniciar sesión en eXeLearning; revisa el acceso local.') from None
    page.get_by_text('Archivo', exact=True).click()
    page.locator('#navbar-button-openuserodefiles').click()
    page.locator('#local-ode-modal-file-upload').set_input_files(str(SALIDA / 'expedientes-etf.elpx'))
    page.locator('.node-text-span').filter(has_text='Créditos y licencia').wait_for()
    page.wait_for_function('eXe.app.getProjectProperties().pp_author.value === "Alberto Avidad"')
    propiedades = page.evaluate('Object.fromEntries(["pp_title","pp_author","pp_license"].map(k=>[k,eXe.app.getProjectProperties()[k].value]))')
    assert propiedades['pp_title'] == 'Expedientes ETF: curso de la aplicación', propiedades
    assert propiedades['pp_license'] == 'creative commons: attribution - share alike 4.0', propiedades
    # La raíz del editor representa el título del proyecto, además de las páginas exportadas.
    assert page.locator('.node-text-span').count() == resumen['paginas'] + 1
    page.locator('.node-text-span').filter(has_text=re.compile('^Autoevaluación$')).first.click()
    n = responder(page)
    print(f'EXE_OK v4.0.5: {resumen["paginas"]} páginas; autor y licencia; autoevaluación {n}/{n}')


class Silencioso(SimpleHTTPRequestHandler):
    def log_message(self, *_args):
        pass


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--importar', action='store_true')
    parser.add_argument('--credenciales', type=Path, default=Path.home() / '.config/expedientes-etf/curso-exelearning.env')
    args = parser.parse_args()
    resumen = json.loads((SALIDA / 'resumen.json').read_text())
    with zipfile.ZipFile(SALIDA / 'scorm12.zip') as paquete:
        manifiesto = ET.fromstring(paquete.read('imsmanifest.xml'))
        metadata = manifiesto.find('{*}metadata')
        assert metadata.find('{*}schema').text == 'ADL SCORM'
        assert metadata.find('{*}schemaversion').text == '1.2'
        metatexto = ET.tostring(metadata, encoding='unicode')
        assert 'Alberto Avidad' in metatexto and 'CC BY-SA 4.0' in metatexto
    print('SCORM_METADATA_OK: autor Alberto Avidad; derechos CC BY-SA 4.0')
    errores = []
    with sync_playwright() as p:
        browser = p.chromium.launch(executable_path='/usr/bin/google-chrome', headless=True)
        page = browser.new_page(viewport={'width': 1440, 'height': 1000})
        page.on('pageerror', lambda e: errores.append(str(e)))
        page.on('console', lambda m: errores.append(m.text) if m.type == 'error' else None)
        paginas = sorted((SALIDA / 'web').rglob('*.html'))
        # Sólo documentos del curso: las plantillas de los iDevices también son HTML.
        paginas = [f for f in paginas if f.parent.name == 'html' or f.name == 'index.html']
        assert len(paginas) == resumen['paginas']
        evaluaciones = preguntas = 0
        for ruta in paginas:
            page.goto(ruta.as_uri())
            page.locator('#siteFooter').wait_for()
            assert page.locator('#siteFooter').inner_text().endswith('La aplicación Expedientes ETF es software libre (EUPL-1.2)')
            if page.locator('.idevice_node.form').count():
                preguntas += responder(page)
                evaluaciones += 1
            if page.locator('video').count():
                page.wait_for_function('Array.from(document.querySelectorAll("video")).every(v=>v.readyState>=1 && !v.error)')
            assert page.locator('img').evaluate_all('(els)=>els.every(e=>e.alt && e.complete && e.naturalWidth>0)'), ruta.name
        assert preguntas == resumen['preguntas']
        assert not errores, errores
        print(f'OFFLINE_OK file://: {len(paginas)} páginas; {evaluaciones} autoevaluaciones; {preguntas} respuestas correctas; 0 errores de consola')
        print(f'VIDEOS_OK: {len(resumen["videos"])} vídeos con metadatos cargados sin conexión')
        bienvenida = SALIDA / 'web/html/bienvenida.html'
        for ancho in (1440, 390):
            page.set_viewport_size({'width': ancho, 'height': 1000})
            page.goto(bienvenida.as_uri())
            page.locator('#siteFooter').wait_for()
            assert page.evaluate('document.documentElement.scrollWidth <= innerWidth'), f'Desbordamiento a {ancho}px'
            page.screenshot(path=str(RAIZ / f'docs/capturas/curso-{ancho}.png'), full_page=True)
        print('CAPTURAS_OK: curso-1440.png; curso-390.png; sin desbordamiento horizontal')
        # Un LMS simulado permite comprobar el envío de la puntuación SCORM real.
        with tempfile.TemporaryDirectory(prefix='curso-scorm-') as carpeta:
            with zipfile.ZipFile(SALIDA / 'scorm12.zip') as paquete:
                paquete.extractall(carpeta)
            servidor = ThreadingHTTPServer(('127.0.0.1', 0), partial(Silencioso, directory=carpeta))
            hilo = threading.Thread(target=servidor.serve_forever, daemon=True)
            hilo.start()
            contexto = browser.new_context()
            contexto.add_init_script('''window.etfNotas=[]; window.API={LMSInitialize:()=>"true",LMSFinish:()=>"true",LMSCommit:()=>"true",LMSGetValue:(k)=>k==="cmi.core.lesson_status"?"not attempted":"",LMSSetValue:(k,v)=>{window.etfNotas.push([k,v]);return "true";},LMSGetLastError:()=>"0",LMSGetErrorString:()=>"",LMSGetDiagnostic:()=>""};''')
            scorm = contexto.new_page()
            scorm.goto(f'http://127.0.0.1:{servidor.server_port}/html/autoevaluacion.html')
            responder(scorm)
            scorm.wait_for_function('window.etfNotas.some(([k,v])=>k==="cmi.core.score.raw" && Number(v)===100)')
            print('SCORM_NOTA_OK: cmi.core.score.raw=100 (LMS simulado)')
            contexto.close()
            servidor.shutdown()
            servidor.server_close()
        if args.importar:
            importar(page, args.credenciales, resumen)
        browser.close()


if __name__ == '__main__':
    main()
