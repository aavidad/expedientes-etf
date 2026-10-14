"""Órdenes de ensayos locales: diagnósticos acotados y sin credenciales.

Sólo se usa con bases desechables y datos inventados. No publica argumentos,
entrada SQL ni el entorno del operador.
"""
import os
from pathlib import Path
import re
import subprocess
import time

LIMITE_SALIDA = 4096
_SECRETO = re.compile(r"PASSWORD|CONTRASENA|SECRET|TOKEN", re.IGNORECASE)


def _secretos(args, entorno):
    valores = set()
    variables = dict(os.environ if entorno is None else entorno)
    for indice, argumento in enumerate(args):
        if argumento == '--env-file' and indice + 1 < len(args):
            try:
                lineas = Path(args[indice + 1]).read_text().splitlines()
            except OSError:
                continue
            for linea in lineas:
                clave, separador, valor = linea.partition('=')
                if separador and _SECRETO.search(clave) and valor:
                    valores.add(valor)
        clave, separador, valor = argumento.partition('=')
        if separador and _SECRETO.search(clave) and valor:
            valores.add(valor)
    valores.update(str(v) for k, v in variables.items() if _SECRETO.search(k) and v)
    return valores


def _salida(texto, secretos):
    if isinstance(texto, bytes):
        texto = texto.decode('utf-8', errors='replace')
    texto = texto or ''
    for valor in sorted(secretos, key=len, reverse=True):
        texto = texto.replace(valor, '[secreto]')
    # PostgreSQL puede incluir valores de filas y SQL en estos apartados.
    texto = re.sub(r'(?m)^(?:DETAIL|CONTEXT|STATEMENT):.*(?:\n[ \t].*)*',
                   '[detalle SQL omitido]', texto)
    if len(texto) > LIMITE_SALIDA:
        mitad = (LIMITE_SALIDA - 32) // 2
        texto = texto[:mitad] + '\n[salida truncada]\n' + texto[-mitad:]
    return texto.strip() or '[vacío]'


def diagnostico(modulo, estado, stdout, stderr, secretos=()):
    return (f'{modulo}.ensayo_fallido: {estado}\n'
            f'stdout:\n{_salida(stdout, secretos)}\n'
            f'stderr:\n{_salida(stderr, secretos)}')


def ejecutar(args, entrada=None, entorno=None, timeout=120, *, modulo='ci', cwd=None):
    secretos = _secretos(args, entorno)
    try:
        resultado = subprocess.run(args, input=entrada, text=True, capture_output=True,
                                   env=entorno, timeout=timeout, cwd=cwd)
    except subprocess.TimeoutExpired as fallo:
        raise RuntimeError(diagnostico(modulo, 'timeout', fallo.stdout, fallo.stderr,
                                      secretos)) from None
    except OSError as fallo:
        raise RuntimeError(diagnostico(modulo, f'error_sistema={fallo.errno}', '',
                                      fallo.strerror, secretos)) from None
    if resultado.returncode:
        raise RuntimeError(diagnostico(modulo, f'codigo={resultado.returncode}',
                                      resultado.stdout, resultado.stderr, secretos))
    return resultado


def orden(args, entrada=None, entorno=None, timeout=120, *, modulo='ci', cwd=None):
    return ejecutar(args, entrada, entorno, timeout, modulo=modulo, cwd=cwd).stdout.strip()


def esperar_postgres(contenedor, archivo_entorno, *, modulo='ci', timeout=60):
    """Espera al servidor definitivo mediante TCP y el rol del migrador.

    El servidor provisional de la imagen oficial acepta consultas por socket
    durante initdb, incluso con los roles creados, pero no escucha por TCP.
    Sólo se sondea una lectura: nunca se reintenta una migración.
    """
    args = ['docker', 'exec', '--env-file', str(archivo_entorno), contenedor,
            'sh', '-c', 'PGPASSWORD="$ETF_MIGRADOR_CONTRASENA" '
            'PGCONNECT_TIMEOUT=2 PGSSLMODE=disable exec psql -X --no-password '
            '-h 127.0.0.1 -p 5432 -U etf_migrador -d etf '
            '-v ON_ERROR_STOP=1 -Atc "SELECT 1"']
    esperar_orden(args, modulo=modulo, timeout=timeout, salida_esperada='1')


def esperar_orden(args, *, modulo='ci', timeout=20, salida_esperada=None):
    """Sondea disponibilidad mediante una orden de lectura, con plazo total."""
    limite = time.monotonic() + timeout
    secretos = _secretos(args, None)
    stdout, stderr = '', ''
    while time.monotonic() < limite:
        try:
            resultado = subprocess.run(args, text=True, capture_output=True,
                                       timeout=min(5, max(0.01, limite - time.monotonic())))
            stdout, stderr = resultado.stdout, resultado.stderr
            if resultado.returncode == 0 and (salida_esperada is None or stdout.strip() == salida_esperada):
                return
        except subprocess.TimeoutExpired as fallo:
            stdout, stderr = fallo.stdout, fallo.stderr
        except OSError as fallo:
            raise RuntimeError(diagnostico(modulo, f'error_sistema={fallo.errno}', '',
                                          fallo.strerror, secretos)) from None
        restante = limite - time.monotonic()
        if restante > 0:
            time.sleep(min(0.2, restante))
    raise RuntimeError(diagnostico(modulo, 'servicio_no_disponible', stdout, stderr, secretos))
