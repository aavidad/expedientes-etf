"""Calendario de formación en una compilación local con overlay; sesiones en tiempo real."""
import argparse
import datetime
import importlib.util
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

RAIZ = Path(__file__).resolve().parents[2]
FABRICAS = ('aceptaciones', 'derivaciones', 'documentos', 'bandeja', 'spaf',
            'memoria', 'consulta_memoria', 'evaluacion', 'actividad', 'decision')
RELOJ = '''
// Adaptador exclusivo de la compilación de formación; no modifica el reloj de sesión.
type relojRecorrido struct{}
func (relojRecorrido) Ahora() time.Time { return horaRecorrido() }
func horaRecorrido() time.Time {
    b, err := os.ReadFile("/tmp/reloj-f507")
    if err != nil { panic("recorrido.reloj_no_disponible") }
    fecha, err := time.Parse(time.RFC3339, string(b))
    if err != nil { panic("recorrido.reloj_invalido") }
    return fecha.UTC()
}
'''

FECHA_SQL = """
// Fecha civil del calendario inventado; sólo existe en el overlay de formación.
func fechaRecorridoSQL() string {
    b, err := os.ReadFile("/tmp/reloj-f507")
    if err != nil { panic("recorrido.reloj_no_disponible") }
    fecha, err := time.Parse(time.RFC3339, string(b))
    if err != nil { panic("recorrido.reloj_invalido") }
    zona, err := time.LoadLocation("Europe/Madrid")
    if err != nil { panic("recorrido.zona_no_disponible") }
    return "'" + fecha.In(zona).Format("2006-01-02") + "'::date"
}
"""


def demo():
    if os.environ.get('ETF_VIDEO_DEMO') != '1':
        raise ValueError('Exige ETF_VIDEO_DEMO=1 y sólo la demostración local')
    s = importlib.util.spec_from_file_location('demo_f507', RAIZ / 'despliegue/demo/demo.py')
    d = importlib.util.module_from_spec(s)
    # sembrar_equipo usa sys.modules; este recorrido prepara sólo usuarios.
    s.loader.exec_module(d)
    d.comprobar_docker_local()
    return d


def overlay(carpeta):
    reemplazos = {}
    for nombre in FABRICAS:
        origen = RAIZ / f'cmd/etf/{nombre}.go'
        texto = origen.read_text()
        texto = texto.replace('relojIdentidad{}', 'relojRecorrido{}').replace('time.Now', 'horaRecorrido')
        if nombre == 'aceptaciones':
            texto = texto.replace('import (', 'import (\n "os"', 1) + RELOJ
        elif not re.search(r'\btime\.', texto):
            texto = re.sub(r'\n\s*"time"', '', texto)
        destino = carpeta / f'{nombre}.go'
        destino.write_text(texto)
        subprocess.run(['gofmt', '-w', destino], check=True, capture_output=True)
        reemplazos[str(origen)] = str(destino)
    # La versión inicial ya existe desde su vigencia en el calendario inventado.
    # Esta proyección de lectura conserva las publicaciones posteriores y no escribe datos.
    origen = RAIZ / 'internal/adaptadores/postgres/catalogo/estadistica.go'
    texto = origen.read_text()
    if texto.count('v.publicada_en') != 2:
        raise ValueError('El catálogo SPAF ha cambiado; revisa el overlay')
    texto = texto.replace('v.publicada_en',
        '(CASE WHEN v.numero=1 THEN v.vigente_desde::timestamptz ELSE v.publicada_en END)')
    destino = carpeta / 'catalogo_estadistica.go'
    destino.write_text(texto)
    reemplazos[str(origen)] = str(destino)
    # Las consultas de vigencia usan la misma fecha que los puertos de negocio.
    # La fecha se analiza antes de producir el literal SQL de formato fijo.
    for nombre in ('aceptacion', 'familia', 'menores', 'intervencion'):
        origen = RAIZ / f'internal/adaptadores/postgres/expediente/{nombre}.go'
        texto = origen.read_text()
        expresion = "(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date"
        esperadas = {'aceptacion': 2, 'familia': 6, 'menores': 4, 'intervencion': 2}
        if texto.count(expresion) != esperadas[nombre]:
            raise ValueError('Las consultas de vigencia han cambiado; revisa el overlay')
        texto = texto.replace("(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date",
            '` + fechaRecorridoSQL() + `')
        if nombre == 'familia':
            texto = texto.replace('import (', 'import (\n "os"', 1) + FECHA_SQL
        destino = carpeta / f'repositorio_{nombre}.go'
        destino.write_text(texto)
        subprocess.run(['gofmt', '-w', destino], check=True, capture_output=True)
        reemplazos[str(origen)] = str(destino)
    fichero = carpeta / 'overlay.json'
    fichero.write_text(json.dumps({'Replace': reemplazos}))
    return fichero


def fijar(d, fecha):
    dia = datetime.date.fromisoformat(fecha)
    instante = f'{dia.isoformat()}T10:00:00Z'
    # El usuario de app escribe sólo su tmpfs; el cambio de nombre es atómico.
    d.compose('exec', '-T', 'app', 'sh', '-c',
              'cat > /tmp/reloj-f507-nuevo && exec mv /tmp/reloj-f507-nuevo /tmp/reloj-f507',
              input=instante.encode(), capture_output=True)
    marcador = d.LOCAL / 'fecha-formacion-nueva'
    marcador.write_text(instante)
    marcador.replace(d.LOCAL / 'fecha-formacion')
    return instante


def iniciar(d):
    d.preparar()
    d.preparar_entrada()
    d.compilar()
    try:
        with tempfile.TemporaryDirectory(prefix='f507-overlay-') as temporal:
            entorno = dict(os.environ, CGO_ENABLED='0', GOOS='linux', GOTOOLCHAIN='auto')
            subprocess.run(['go', 'build', '-mod=readonly', '-trimpath', '-overlay', overlay(Path(temporal)),
                            '-o', d.LOCAL / 'bin/etf', './cmd/etf'], cwd=RAIZ, env=entorno, check=True, capture_output=True)
        d.compose('config', '--quiet', capture_output=True)
        with tempfile.TemporaryDirectory(prefix='f507-compose-') as temporal:
            aislamiento = Path(temporal) / 'reloj.yaml'
            aislamiento.write_text('services:\n  app:\n    tmpfs:\n      - /tmp:uid=10001,gid=10001,mode=0700,size=1m\n')
            d.compose('-f', aislamiento, 'up', '--detach', '--build', '--force-recreate', '--wait', '--wait-timeout', '180', 'app', capture_output=True)
        d.compose('restart', 'entrada-demo', capture_output=True)
        fijar(d, datetime.date.today().isoformat())
        d.sembrar()
        fijar(d, '2026-10-14')
        # No se crean otros expedientes: repetir conserva los casos y revisiones previos.
        print('RELOJ-DEMO-OK: calendario de negocio; usuarios inventados; sesiones en tiempo real')
        print(f'Demostración de formación disponible en {d.URL}.')
    except BaseException:
        restaurar(d)
        raise


def restaurar(d):
    d.compilar()
    d.compose('up', '--detach', '--build', '--wait', '--wait-timeout', '180', 'app', capture_output=True)
    d.compose('restart', 'entrada-demo', capture_output=True)
    (d.LOCAL / 'fecha-formacion').unlink(missing_ok=True)
    print('RELOJ-DEMO-RESTAURADO: compilación ordinaria')


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('accion', choices=('iniciar', 'fijar', 'restaurar'))
    ap.add_argument('fecha', nargs='?')
    args = ap.parse_args()
    d = demo()
    if args.accion == 'iniciar':
        iniciar(d)
    elif args.accion == 'restaurar':
        restaurar(d)
    else:
        if not args.fecha:
            ap.error('fijar exige fecha ISO')
        fijar(d, args.fecha)


if __name__ == '__main__':
    try:
        main()
    except (ValueError, OSError, subprocess.SubprocessError):
        raise SystemExit('RELOJ-DEMO-ERROR: revisa la demostración local y la fecha; no se repite ninguna escritura') from None
