#!/usr/bin/env python3
"""Prueba la subida PDF por HTTP con ClamAV de compose en recursos desechables.

No usa la instalación en marcha. Firmas sintéticas y ejecutable viven en tmpfs;
los PDF se leen del corpus y recorren multipart, antivirus y lector en memoria.
"""
import hashlib
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import time

RAIZ = Path(__file__).resolve().parents[4]


def main():
    with tempfile.TemporaryDirectory(prefix='etf-f103-', dir='/dev/shm') as carpeta:
        tmp = Path(carpeta)
        tmp.chmod(0o755)
        firmas = tmp / 'firmas'
        firmas.mkdir(mode=0o755)
        eicar = b'X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*'
        (firmas / 'local.hdb').write_text(hashlib.md5(eicar).hexdigest()+':68:ETF.Eicar.Test\n')
        (firmas / 'local.hdb').chmod(0o644)
        (firmas / 'local.ndb').write_text('ETF.Eicar.Test:0:*:'+eicar.hex()+'\n')
        (firmas / 'local.ndb').chmod(0o644)
        overlay = tmp / 'compose.yaml'
        # Se heredan imagen, configuración sin registros y aislamiento de compose.
        overlay.write_text('services:\n  clamav:\n    restart: "no"\n'
            '    mem_limit: 512m\n    memswap_limit: 512m\n'
            '    volumes: !override\n'
            f'      - {firmas}:/var/lib/clamav:ro\n'
            f'      - {RAIZ}/despliegue/base/clamd.conf:/etc/clamav/clamd.conf:ro\n')
        env = dict(os.environ)
        for clave in ('ETF_INSTALADOR_CONTRASENA', 'ETF_MIGRADOR_CONTRASENA',
                      'ETF_APP_CONTRASENA', 'ETF_COPIA_CONTRASENA'):
            env[clave] = secrets.token_hex(24)
        env['ETF_RED_SUBRED'] = f'10.253.{secrets.randbelow(240)}.0/28'
        proyecto = 'etf-f103-' + secrets.token_hex(4)
        compose = ['docker', 'compose', '-p', proyecto, '-f', str(RAIZ/'compose.yaml'),
                   '-f', str(overlay)]

        def orden(args):
            r = subprocess.run(args, env=env, cwd=RAIZ, capture_output=True,
                               text=True, timeout=60)
            if r.returncode:
                raise RuntimeError('anexo.ensayo_fallido: ' + r.stdout[-4000:] + r.stderr[-2000:])
            return r.stdout.strip()

        # Compilación sin red. El binario se copia al tmpfs ejecutable de clamd.
        # /dev/shm del anfitrión puede estar montado con noexec.
        env['GOPROXY'] = 'off'
        env['CGO_ENABLED'] = '0'
        orden(['go', 'test', '-c', '-o', str(tmp/'pdf.test'),
               './internal/adaptadores/anexos/pdf'])
        try:
            orden(compose + ['up', '-d', '--no-deps', '--pull', 'never', 'clamav'])
            contenedor = orden(compose + ['ps', '-q', 'clamav'])
            for _ in range(100):
                estado = json.loads(orden(['docker', 'inspect', contenedor]))[0]
                if estado['State'].get('Health', {}).get('Status') == 'healthy':
                    break
                time.sleep(0.3)
            else:
                raise RuntimeError('anexo.ensayo_antivirus_no_disponible')
            # Ruta del corpus de sólo lectura mediante un contenedor separado;
            # comparte únicamente la red aislada de esta prueba.
            imagen = estado['Config']['Image']
            red = next(iter(estado['NetworkSettings']['Networks']))
            salida = orden(['docker', 'run', '--rm', '--network', red,
                '--read-only', '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges:true',
                '--memory', '256m', '--memory-swap', '256m', '--cpus', '1', '--pids-limit', '64',
                '--mount', f'type=bind,source={RAIZ},target=/repo,readonly',
                '--mount', f'type=bind,source={tmp},target=/pruebas,readonly',
                '--tmpfs', '/tmp:rw,exec,size=64m',
                '--workdir', '/repo/internal/adaptadores/anexos/pdf',
                '--env', 'ETF_PRUEBA_CLAMAV_DIRECCION=clamav:3310',
                '--entrypoint', '/bin/sh', imagen, '-c',
                'cp /pruebas/pdf.test /tmp/pdf.test && exec /tmp/pdf.test -test.v -test.run TestSubidaHTTPPDFClamAV'])
            print(salida)
            print('ENSAYO-OK F1.03: HTTP PDF, ClamAV de compose, escaneado y rechazos')
        finally:
            orden(compose + ['down', '--volumes', '--remove-orphans'])


if __name__ == '__main__':
    main()
