#!/usr/bin/env python3
"""Ensaya el lector en CI sin eXeLearning, Microsoft ni otra red externa."""
import argparse
import importlib.util
from pathlib import Path
import zipfile
from curso import integrar

RAIZ = Path(__file__).resolve().parents[2]


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--salida',type=Path,required=True)
    parser.add_argument('--audios',type=Path,required=True)
    args=parser.parse_args()
    spec=importlib.util.spec_from_file_location('curso_fuentes',RAIZ/'curso/construir.py')
    modulo=importlib.util.module_from_spec(spec);spec.loader.exec_module(modulo)
    constructor=modulo.Constructor(modulo.cargar_yaml(RAIZ/'curso/estructura.yaml'),RAIZ/'bin')
    constructor.leer()
    web=args.salida/'web';web.mkdir(parents=True)
    paginas={}
    for i,pagina in enumerate(constructor.paginas):
        nombre='index.html' if i==0 else pagina['id']+'.html'
        contenido=pagina.get('html','')
        paginas[nombre]=f'<!doctype html><html lang="es"><head><meta charset="utf-8"><title>{pagina["titulo"]}</title></head><body><main><h1>{pagina["titulo"]}</h1><div id="page-content-{pagina["id"]}">{contenido}</div></main></body></html>'.encode()
        (web/nombre).write_bytes(paginas[nombre])
    with zipfile.ZipFile(args.salida/'scorm12.zip','w') as paquete:
        for nombre,contenido in paginas.items(): paquete.writestr(nombre,contenido)
        paquete.writestr('imsmanifest.xml','<manifest><resources><resource></resource></resources></manifest>')
    integrar(constructor,args.salida,args.audios)
    print('Curso de prueba local preparado sin red.')


if __name__=='__main__':main()
