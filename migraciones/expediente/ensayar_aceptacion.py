#!/usr/bin/env python3
"""Ensaya F1.05a y F1.05b en PostgreSQL 18 y ClamAV desechables, sin salida a Internet.

Sólo se crean recursos propios. La base y los temporales de ClamAV viven en
tmpfs sin intercambio a disco. Las firmas de este ensayo son sintéticas.
"""
import argparse
import hashlib
import json
import os
import sys
from functools import partial
import pathlib
import secrets
import shutil
import subprocess
import tempfile

RAIZ = pathlib.Path(__file__).resolve().parents[2]
POSTGRES = 'postgres:18.4-alpine@sha256:9a8afca54e7861fd90fab5fdf4c42477a6b1cb7d293595148e674e0a3181de15'
CLAMAV = 'clamav/clamav:1.4.3@sha256:75fb5fd95fcbe1d7e6d240c369c1572b686ee2c95949d1042b5148de8eddebb4'


sys.path.insert(0, str(RAIZ / 'scripts' / 'ci'))
from ensayos import orden as ejecutar_orden, esperar_postgres, esperar_orden

orden = partial(ejecutar_orden, modulo='expediente')


esperar = partial(esperar_orden, modulo='expediente')


def retirar_migraciones(raiz, identificadores):
    from ensayos import recortar_dependientes
    for modulo, numero in identificadores:
        carpeta = raiz / modulo
        manifiesto = carpeta / 'manifiesto.json'
        entradas = json.loads(manifiesto.read_text())
        retirada = next(e for e in entradas if e['numero'] == numero)
        (carpeta / retirada['archivo']).unlink()
        manifiesto.write_text(json.dumps([e for e in entradas if e['numero'] != numero]))
    recortar_dependientes(raiz)


def retirar_desde(raiz, cortes):
    from ensayos import recortar_dependientes
    for modulo, numero in cortes:
        carpeta = raiz / modulo
        manifiesto = carpeta / 'manifiesto.json'
        entradas = json.loads(manifiesto.read_text())
        for retirada in entradas:
            if retirada['numero'] >= numero:
                (carpeta / retirada['archivo']).unlink()
        manifiesto.write_text(json.dumps([e for e in entradas if e['numero'] < numero]))
    recortar_dependientes(raiz)


def comprobar_guardia_seguimiento(bd, ejecutar, tmp):
    """Rechaza reglas que quedarían futuras y conserva la versión publicada."""
    orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador', '-d', 'etf',
           '-v', 'ON_ERROR_STOP=1', '-c',
           'CREATE DATABASE etf_guardia TEMPLATE etf_desde_traslado OWNER etf_propietario'])
    sesion = ejecutar + ['env', 'PGDATABASE=etf_guardia', 'PGUSER=etf_migrador', 'psql', '-v', 'ON_ERROR_STOP=1']
    preparar = tmp / 'guardia_preparar.sql'
    preparar.write_text("""BEGIN; SET ROLE etf_propietario;
DO $$ DECLARE anterior catalogo.version; nueva uuid; BEGIN
 SELECT * INTO STRICT anterior FROM catalogo.version WHERE ambito='plazos' AND estado='publicada' ORDER BY numero DESC LIMIT 1;
 INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
 VALUES('plazos',anterior.numero+1,(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date,
 'catalogo.avisos_iniciales','00000000-0000-4000-8000-000000000001') RETURNING id INTO nueva;
 INSERT INTO catalogo.parametro SELECT nueva,clave,valor_entero,unidad FROM catalogo.parametro WHERE version_id=anterior.id;
 INSERT INTO catalogo.fase_duracion SELECT nueva,subprograma,fase,dias_orientativos,dias_ampliable FROM catalogo.fase_duracion WHERE version_id=anterior.id;
 INSERT INTO catalogo.version_textos SELECT nueva,codigo,version_contenido FROM catalogo.version_textos WHERE version_id=anterior.id;
 PERFORM catalogo.publicar(nueva,'00000000-0000-4000-8000-000000000001');
END $$; COMMIT;
""")
    orden(sesion + ['-f', '/pruebas/guardia_preparar.sql'])
    aplicar = tmp / 'guardia_aplicar.sql'
    aplicar.write_text('BEGIN; SET ROLE etf_propietario;\n' +
                       (RAIZ / 'migraciones/catalogo/0013_seguimiento.sql').read_text() + '\nCOMMIT;\n')
    rechazo = subprocess.run(sesion + ['-f', '/pruebas/guardia_aplicar.sql'],
                             text=True, capture_output=True, timeout=120)
    if rechazo.returncode == 0 or 'migracion.estado_previo' not in rechazo.stdout + rechazo.stderr:
        raise RuntimeError('expediente.ensayo_semilla_futura')
    restantes = orden(sesion + ['-Atc', "SET ROLE etf_propietario; SELECT count(*) FROM catalogo.version WHERE motivo='catalogo.seguimiento_inicial'"])
    if restantes.splitlines()[-1] != '0':
        raise RuntimeError('expediente.ensayo_semilla_parcial')
    print('ENSAYO-OK catálogo: vigencia de hoy rechazada sin publicar reglas futuras')



def comprobar_configuracion_seguimiento(bd, ejecutar, tmp):
    """La instalación conserva reglas y opciones que ya publicó Administración."""
    orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador', '-d', 'etf',
           '-v', 'ON_ERROR_STOP=1', '-c',
           'CREATE DATABASE etf_configuracion TEMPLATE etf_desde_traslado OWNER etf_propietario'])
    sesion = ejecutar + ['env', 'PGDATABASE=etf_configuracion', 'PGUSER=etf_migrador', 'psql', '-v', 'ON_ERROR_STOP=1']
    preparar = tmp / 'configuracion_preparar.sql'
    preparar.write_text("""BEGIN; SET ROLE etf_propietario;
DO $$ DECLARE ambito_nuevo text; anterior catalogo.version; nueva uuid; BEGIN
 FOREACH ambito_nuevo IN ARRAY ARRAY['plazos','listas'] LOOP
  SELECT * INTO STRICT anterior FROM catalogo.version WHERE ambito=ambito_nuevo AND estado='publicada' ORDER BY numero DESC LIMIT 1;
  INSERT INTO catalogo.version(ambito,numero,vigente_desde,motivo,creada_por)
  VALUES(ambito_nuevo,anterior.numero+1,(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date-1,
   'catalogo.edicion','00000000-0000-4000-8000-000000000001') RETURNING id INTO nueva;
  IF ambito_nuevo='plazos' THEN
   INSERT INTO catalogo.parametro SELECT nueva,clave,valor_entero,unidad FROM catalogo.parametro WHERE version_id=anterior.id AND clave NOT IN ('seguimiento_tras_cese','aviso_fin_seguimiento');
   INSERT INTO catalogo.parametro VALUES(nueva,'seguimiento_tras_cese',18,'meses'),(nueva,'aviso_fin_seguimiento',60,'dias');
   INSERT INTO catalogo.fase_duracion SELECT nueva,subprograma,fase,dias_orientativos,dias_ampliable FROM catalogo.fase_duracion WHERE version_id=anterior.id;
  ELSE
   INSERT INTO catalogo.lista_opcion SELECT nueva,lista,codigo,orden,admite_texto,activa FROM catalogo.lista_opcion WHERE version_id=anterior.id AND lista<>'fin_seguimiento';
   INSERT INTO catalogo.lista_opcion VALUES(nueva,'fin_seguimiento','OBJETIVOS_CUMPLIDOS',1,false,true),(nueva,'fin_seguimiento','CAMBIO_SITUACION',4,true,false);
  END IF;
  INSERT INTO catalogo.version_textos SELECT nueva,codigo,version_contenido FROM catalogo.version_textos WHERE version_id=anterior.id;
  PERFORM catalogo.publicar(nueva,'00000000-0000-4000-8000-000000000001');
 END LOOP;
END $$; COMMIT;
""")
    orden(sesion + ['-f', '/pruebas/configuracion_preparar.sql'])
    contenido = """SET ROLE etf_propietario;
SELECT jsonb_build_object(
 'parametros',(SELECT jsonb_agg(to_jsonb(p)-'version_id' ORDER BY clave) FROM catalogo.parametro p JOIN catalogo.version v ON v.id=p.version_id WHERE v.ambito='plazos' AND v.estado='publicada' AND v.vigente_hasta IS NULL),
 'fases',(SELECT jsonb_agg(to_jsonb(p)-'version_id' ORDER BY subprograma,fase) FROM catalogo.fase_duracion p JOIN catalogo.version v ON v.id=p.version_id WHERE v.ambito='plazos' AND v.estado='publicada' AND v.vigente_hasta IS NULL),
 'listas',(SELECT jsonb_agg(to_jsonb(p)-'version_id' ORDER BY lista,orden) FROM catalogo.lista_opcion p JOIN catalogo.version v ON v.id=p.version_id WHERE v.ambito='listas' AND v.estado='publicada' AND v.vigente_hasta IS NULL));
"""
    antes = orden(sesion + ['-Atc', contenido]).splitlines()[-1]
    aplicar = tmp / 'configuracion_aplicar.sql'
    aplicar.write_text('BEGIN; SET ROLE etf_propietario;\n' +
                       (RAIZ / 'migraciones/catalogo/0013_seguimiento.sql').read_text() + '\nCOMMIT;\n')
    orden(sesion + ['-f', '/pruebas/configuracion_aplicar.sql'])
    despues = orden(sesion + ['-Atc', contenido]).splitlines()[-1]
    if json.loads(antes) != json.loads(despues):
        raise RuntimeError('expediente.ensayo_configuracion_sustituida')
    print('ENSAYO-OK catálogo: conserva 18 meses, aviso de 60 días y motivos publicados')


def main():
    argumentos = argparse.ArgumentParser(description=__doc__)
    argumentos.add_argument('--pruebas', default='.', help='Expresión de go test para el recorrido sobre la base migrada.')
    seleccion = argumentos.parse_args().pruebas
    with tempfile.TemporaryDirectory(prefix='etf-f105a-') as carpeta:
        tmp = pathlib.Path(carpeta)
        tmp.chmod(0o755)
        prefijo = 'etf-f105a-' + secrets.token_hex(4)
        red, bd, antivirus = prefijo + '-red', prefijo + '-bd', prefijo + '-av'
        claves = {k: secrets.token_hex(24) for k in ('ETF_MIGRADOR_CONTRASENA',
                  'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA')}
        env = tmp / 'base.env'
        env.write_text('POSTGRES_USER=etf_instalador\nPOSTGRES_DB=etf\nPOSTGRES_PASSWORD=' +
                       secrets.token_hex(24) + '\n' + ''.join(k+'='+v+'\n' for k,v in claves.items()))
        env.chmod(0o600)
        runner = tmp / 'runner.env'
        runner.write_text('ETF_BD_HOST=127.0.0.1\nETF_BD_PUERTO=5432\nETF_BD_NOMBRE=etf\n'
                          'ETF_BD_USUARIO=etf_app\nETF_POSTGRES_INTEGRACION=1\n'
                          'ETF_PRUEBA_CLAMAV_DIRECCION=clamav:3310\n'
                          'ETF_BD_CONTRASENA='+claves['ETF_APP_CONTRASENA']+'\n'
                          'ETF_MIGRADOR_CONTRASENA='+claves['ETF_MIGRADOR_CONTRASENA']+'\n')
        runner.chmod(0o600)
        firmas = tmp / 'firmas'
        firmas.mkdir()
        eicar = b'X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*'
        (firmas / 'local.hdb').write_text(hashlib.md5(eicar).hexdigest()+':68:ETF.Eicar.Test\n')
        (tmp / 'clamd.conf').write_text('Foreground yes\nTCPSocket 3310\nTCPAddr 0.0.0.0\n'
            'DatabaseDirectory /pruebas/firmas\nTemporaryDirectory /tmp\n'
            'StreamMaxLength 10M\nMaxScanSize 100M\nMaxFileSize 10M\nMaxRecursion 8\n'
            'MaxFiles 1000\nLogTime no\nLogClean no\nLogFile /dev/null\n')
        compilacion = dict(os.environ, CGO_ENABLED='0')
        orden(['go', 'build', '-o', str(tmp / 'migrar'), './cmd/migrar'], entorno=compilacion)
        orden(['go', 'test', '-c', '-o', str(tmp / 'expediente.test'),
               './internal/adaptadores/postgres/expediente'], entorno=compilacion)
        previo = tmp / 'previo'
        previo.mkdir()
        for modulo in ('plataforma', 'catalogo', 'auditoria', 'identidad', 'paquete', 'expediente'):
            shutil.copytree(RAIZ / 'migraciones' / modulo, previo / modulo)
        # Conservar los cortes de familia, SPM y salida a otro servicio;
        # cada corte elimina también las migraciones posteriores del módulo.
        retirar_desde(previo, [('expediente','0004'), ('expediente','0005'),
                              ('expediente','0006'), ('catalogo','0006'),
                              ('catalogo','0007'), ('expediente','0007'),
                              ('expediente','0008'), ('paquete','0003'), ('auditoria','0002')])
        previo_cierre = tmp / 'previo_cierre'
        for modulo in ('plataforma', 'catalogo', 'auditoria', 'identidad', 'paquete', 'expediente'):
            shutil.copytree(RAIZ / 'migraciones' / modulo, previo_cierre / modulo)
        retirar_desde(previo_cierre, [('expediente','0008'),('catalogo','0008'),('paquete','0003'),('auditoria','0002')])
        retirar_migraciones(previo_cierre, [('expediente','0004'),('catalogo','0006')])
        previo_traslado = tmp / 'previo_traslado'
        for modulo in ('plataforma', 'catalogo', 'auditoria', 'identidad', 'paquete', 'expediente'):
            shutil.copytree(RAIZ / 'migraciones' / modulo, previo_traslado / modulo)
        retirar_desde(previo_traslado, [('expediente','0021')])
        retirar_migraciones(previo_traslado, [('catalogo','0013')])
        previo_historica = tmp / 'previo_historica'
        for modulo in ('plataforma', 'catalogo', 'auditoria', 'identidad', 'paquete', 'expediente'):
            shutil.copytree(RAIZ / 'migraciones' / modulo, previo_historica / modulo)
        retirar_desde(previo_historica, [('expediente', '0018'), ('auditoria', '0002')])
        previo_hoja = tmp / 'previo_hoja'
        for modulo in RAIZ.joinpath('migraciones').iterdir():
            if modulo.is_dir() and (modulo / 'manifiesto.json').exists() and modulo.name != 'carga':
                shutil.copytree(modulo, previo_hoja / modulo.name)
        antes_decisiones = tmp / 'antes_decisiones'
        shutil.copytree(previo, antes_decisiones)
        retirar_desde(antes_decisiones, [('catalogo','0005'),('expediente','0003')])
        montar = ['--mount', f'type=bind,source={tmp},target=/pruebas,readonly',
                  '--mount', f'type=bind,source={RAIZ},target=/repo,readonly']
        try:
            orden(['docker', 'network', 'create', '--internal', '--subnet',
                   f'10.252.{secrets.randbelow(240)}.0/28', red])
            orden(['docker', 'run', '-d', '--name', bd, '--network', red,
                   '--read-only', '--memory', '512m', '--memory-swap', '512m', '--pids-limit', '128',
                   '--env-file', str(env), '--tmpfs', '/var/lib/postgresql:rw,size=512m',
                   '--tmpfs', '/var/run/postgresql:rw,size=16m',
                   '--tmpfs', '/tmp:rw,size=16m', *montar,
                   '--mount', f'type=bind,source={RAIZ}/despliegue/base/01_roles.sh,target=/docker-entrypoint-initdb.d/01_roles.sh,readonly',
                   POSTGRES])
            esperar_postgres(bd, runner, modulo='expediente')
            ejecutar = ['docker', 'exec', '--env-file', str(runner), bd]
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/antes_decisiones', '/pruebas/migrar']))
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/previo', '/pruebas/migrar']))
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir', '/repo/internal/adaptadores/postgres/expediente', bd, '/pruebas/expediente.test', '-test.v', '-test.run', '^TestAceptacionHTTPConPostgresYRecuperacion$']))
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/pruebas/previo_historica', '/pruebas/migrar']))
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            print(orden(ejecutar + ['env', 'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            print('ENSAYO-OK versión anterior, migración nueva y repetición')
            orden(['docker', 'run', '-d', '--name', antivirus, '--network', red,
                   '--network-alias', 'clamav', '--read-only', '--memory', '512m',
                   '--memory-swap', '512m', '--pids-limit', '64', '--cap-drop', 'ALL',
                   '--security-opt', 'no-new-privileges:true', '--tmpfs', '/tmp:rw,size=64m',
                   *montar, '--entrypoint', '/usr/sbin/clamd', CLAMAV,
                   '--config-file=/pruebas/clamd.conf'])
            esperar(['docker', 'exec', antivirus, 'sh', '-c',
                     "printf 'nPING\\n' | nc -w 1 127.0.0.1 3310 | grep -qx PONG"])
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir',
                         '/repo/internal/adaptadores/postgres/expediente', bd,
                         '/pruebas/expediente.test', '-test.v', '-test.run', seleccion]))
            # Otra base, desde cero, acredita también el camino de instalación.
            orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador', '-d', 'etf',
                   '-v', 'ON_ERROR_STOP=1', '-c', 'CREATE DATABASE etf_limpia OWNER etf_propietario'])
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_limpia',
                        'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            print(orden(['docker','exec','--env-file',str(runner),'--workdir','/repo/internal/adaptadores/postgres/expediente',bd] + ['env', 'ETF_BD_NOMBRE=etf_limpia', '/pruebas/expediente.test',
                         '-test.v', '-test.run', seleccion if seleccion != '.' else 'TestAceptacion|TestNumeracion|TestDecision|TestFamilia|TestMenores|TestIntervencion|TestAvisos|TestCierre|TestPropuesta|TestHitos|TestConsentimiento|TestSPM|TestTraslado|TestSeguimiento|TestHoja'], entorno=None))
            # La versión inmediatamente anterior contiene datos antes de aplicar
            # únicamente las dos migraciones de traslado y seguimiento.
            orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador', '-d', 'etf',
                   '-v', 'ON_ERROR_STOP=1', '-c', 'CREATE DATABASE etf_desde_traslado OWNER etf_propietario'])
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_desde_traslado',
                        'ETF_MIGRACIONES_RUTA=/pruebas/previo_traslado', '/pruebas/migrar']))
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir',
                         '/repo/internal/adaptadores/postgres/expediente', bd,
                         'env', 'ETF_BD_NOMBRE=etf_desde_traslado', '/pruebas/expediente.test',
                         '-test.v', '-test.run', '^TestAceptacionHTTPConPostgresYRecuperacion$']))
            comprobar_guardia_seguimiento(bd, ejecutar, tmp)
            comprobar_configuracion_seguimiento(bd, ejecutar, tmp)
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_desde_traslado',
                        'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            vigencia = orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador',
                             '-d', 'etf_desde_traslado', '-Atc',
                             "SELECT count(*) FROM catalogo.version WHERE motivo='catalogo.seguimiento_inicial' AND vigente_desde=(CURRENT_TIMESTAMP AT TIME ZONE 'Europe/Madrid')::date"])
            if vigencia != '2':
                raise RuntimeError('expediente.ensayo_semilla_retroactiva')
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir',
                         '/repo/internal/adaptadores/postgres/expediente', bd,
                         'env', 'ETF_BD_NOMBRE=etf_desde_traslado', '/pruebas/expediente.test',
                         '-test.v', '-test.run', 'TestTraslado|TestSeguimiento|TestHoja']))
            print('ENSAYO-OK traslado y seguimiento sobre versión inmediata anterior con datos')

            # La reserva carga/0001 se ensaya también sobre F2.08a ya aplicada.
            orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador', '-d', 'etf',
                   '-v', 'ON_ERROR_STOP=1', '-c', 'CREATE DATABASE etf_desde_hoja OWNER etf_propietario'])
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_desde_hoja',
                         'ETF_MIGRACIONES_RUTA=/pruebas/previo_hoja', '/pruebas/migrar']))
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_desde_hoja',
                         'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar']))
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir',
                         '/repo/internal/adaptadores/postgres/expediente', bd,
                         'env', 'ETF_BD_NOMBRE=etf_desde_hoja', '/pruebas/expediente.test',
                         '-test.v', '-test.run', '^TestHoja']))
            # El control de retrocesos sigue rechazando números anteriores
            # si una base ya registró la propuesta sin las migraciones de familia.
            orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador', '-d', 'etf',
                   '-v', 'ON_ERROR_STOP=1', '-c', 'CREATE DATABASE etf_desde_cierre OWNER etf_propietario'])
            print(orden(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_desde_cierre',
                        'ETF_MIGRACIONES_RUTA=/pruebas/previo_cierre', '/pruebas/migrar']))
            print(orden(['docker', 'exec', '--env-file', str(runner), '--workdir',
                         '/repo/internal/adaptadores/postgres/expediente', bd,
                         'env', 'ETF_BD_NOMBRE=etf_desde_cierre', '/pruebas/expediente.test',
                         '-test.v', '-test.run', '^TestAceptacionHTTPConPostgresYRecuperacion$']))
            rechazo = subprocess.run(ejecutar + ['env', 'ETF_BD_NOMBRE=etf_desde_cierre',
                        'ETF_MIGRACIONES_RUTA=/repo/migraciones', '/pruebas/migrar'],
                        text=True, capture_output=True, timeout=120)
            if rechazo.returncode == 0 or 'migracion.retroceso' not in rechazo.stdout + rechazo.stderr:
                raise RuntimeError('expediente.ensayo_retroceso_no_rechazado')
            pendientes = orden(['docker', 'exec', bd, 'psql', '-U', 'etf_instalador',
                        '-d', 'etf_desde_cierre', '-Atc',
                        "SELECT count(*) FROM plataforma.migracion_aplicada WHERE (modulo,numero) IN (('catalogo','0006'),('expediente','0004'),('expediente','0011'))"])
            if pendientes != '0':
                raise RuntimeError('expediente.ensayo_retroceso_parcial')
            print('ENSAYO-OK retroceso rechazado sin aplicar familia sobre F3.02a ya registrada')
            print('ENSAYO-OK expediente: base limpia y anteriores con datos; selección de pruebas ' + seleccion)
        finally:
            for nombre in (antivirus, bd):
                subprocess.run(['docker', 'rm', '-fv', nombre], stdout=subprocess.DEVNULL,
                               stderr=subprocess.DEVNULL)
            subprocess.run(['docker', 'network', 'rm', red], stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL)


if __name__ == '__main__':
    main()
