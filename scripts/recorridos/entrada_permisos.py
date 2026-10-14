#!/usr/bin/env python3
"""Ejecuta F0.11 sólo contra la demostración de este worktree."""
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time

RAIZ = Path(__file__).resolve().parents[2]
DEMO = RAIZ / 'despliegue/demo'


def ejecutar(args, **opciones):
    return subprocess.run(args, cwd=RAIZ, check=True, timeout=600, **opciones)


def main():
    inicio = time.monotonic()
    temporal = Path(os.environ.get('TMPDIR', str(RAIZ / 'bin/temporales')))
    temporal.mkdir(parents=True, exist_ok=True)
    evidencia = Path(tempfile.mkdtemp(prefix='f011-', dir=temporal))
    entorno = dict(os.environ)
    if not (DEMO / '.local/puerto').exists() and not entorno.get('ETF_DEMO_PUERTO'):
        with socket.socket() as s:
            s.bind(('127.0.0.1', 0))
            entorno['ETF_DEMO_PUERTO'] = str(s.getsockname()[1])
    os.environ.update({k: v for k, v in entorno.items() if k.startswith('ETF_DEMO_')})
    spec = importlib.util.spec_from_file_location('demo_f011', DEMO / 'demo.py')
    demo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(demo)
    print(f'F0.11-EVIDENCIA: {evidencia}', flush=True)
    # Una demo existente puede acumular intentos durante la depuración.
    if (demo.LOCAL / 'entorno.env').exists():
        activos = demo.compose('ps', '--status', 'running', '-q', 'app', capture_output=True, text=True).stdout.strip()
        if activos:
            demo.compose('restart', '--no-deps', 'app', stdout=subprocess.DEVNULL)
    # make demo imprime contraseñas sintéticas: su salida queda privada, fuera de Git.
    with (evidencia / 'demo.log').open('w') as salida:
        ejecutar(['make', 'demo'], env=entorno, stdout=salida, stderr=subprocess.STDOUT)
    # Reinicia sólo esta app de pruebas: los límites de entrada no dependen de ensayos anteriores.
    demo.compose('up', '--detach', '--force-recreate', '--no-deps', '--wait', 'app', stdout=subprocess.DEVNULL)
    # Las rutas públicas PWA1 ya deben tener textos antes de la matriz INT5.
    ejecutar(['python3', '-B', 'scripts/pwa1/preparar_demo.py', demo.URL])
    registro = evidencia / 'registro.json'
    ejecutar(['go', 'test', './cmd/etf', '-run', '^TestRegistroRecorridoEntrada$', '-count=1', '-v'],
             env=dict(os.environ, ETF_RECORRIDO_REGISTRO=str(registro)))
    # Reutiliza la aceptación y la emisión reales para poblar las fronteras de datos.
    ejecutar(['make', 'demo-recorrido'])
    def sql(texto):
        try:
            return demo.compose('exec', '-T', 'postgres', 'psql', '-X', '-U', 'etf_instalador', '-d', 'etf',
                            '-v', 'ON_ERROR_STOP=1', '-Atq', input=texto, capture_output=True, text=True).stdout.strip()
        except subprocess.CalledProcessError as error:
            raise RuntimeError(error.stderr.strip()) from None
    sql("""
INSERT INTO expediente.salida_servicio(id,expediente_id,equipo_id,servicio_destino,catalogo_id,motivo,creada_por,creada_en)
 SELECT gen_random_uuid(),e.id,e.equipo_id,'SALUD',(SELECT id FROM catalogo.version WHERE ambito='listas' LIMIT 1),
 'La familia requiere atención del centro de salud.',(SELECT id FROM identidad.usuario WHERE cuenta_ad='pablo.ruiz'),clock_timestamp()
 FROM expediente.expediente e WHERE e.estado='aceptada' ORDER BY e.fecha_recepcion DESC LIMIT 1;
""")
    datos = json.loads(sql("""
SELECT json_build_object(
 'expediente', e.id, 'equipo', e.equipo_id,
 'salida', (SELECT id FROM expediente.salida_servicio WHERE expediente_id=e.id LIMIT 1),
 'emision', (SELECT id FROM documento.emision WHERE expediente_id=e.id LIMIT 1),
 'pendiente', (SELECT id FROM identidad.usuario WHERE cuenta_ad='sergio.campos'))
FROM expediente.expediente e WHERE e.estado='aceptada' ORDER BY fecha_recepcion DESC LIMIT 1;
"""))
    fixture = evidencia / 'fixture.json'
    fixture.write_text(json.dumps(datos))
    ejecutar(['node', 'web/tests/entrada-permisos.navegador.mjs', demo.URL, str(registro), str(fixture), str(evidencia)])
    rls = sql((Path(__file__).with_name('entrada_permisos_rls.sql')).read_text())
    (evidencia / 'rls.log').write_text(rls + '\n')
    print('\n'.join(linea for linea in rls.splitlines() if linea.startswith('F0.11-')))
    demo.compose('up', '--detach', '--force-recreate', '--no-deps', '--wait', 'app', stdout=subprocess.DEVNULL)
    ejecutar(['node', 'web/tests/consulta-memoria.navegador.mjs'])
    ejecutar(['node', 'scripts/recorridos/pwa1.mjs', demo.URL])
    print(f'F0.11-OK: {time.monotonic() - inicio:.1f} s; evidencia en {evidencia}')


if __name__ == '__main__':
    main()
