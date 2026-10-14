"""Prueba el límite de origen con Node, sin abrir navegador ni hacer peticiones."""
import subprocess
from herramientas import RAIZ


def validar_base(test):
    codigo = '''
import { baseLocal } from './scripts/recorridos/comun.mjs';
import assert from 'node:assert/strict';
for (const url of ['https://example.org', 'http://127.0.0.1.evil.test', 'http://user:pass@localhost', 'file:///tmp/demo', 'http://localhost/ruta', 'http://localhost/?clave=1']) assert.throws(() => baseLocal(url));
assert.equal(baseLocal('http://127.0.0.1:18083'), 'http://127.0.0.1:18083');
'''
    r = subprocess.run(['node', '--input-type=module', '-e', codigo], cwd=RAIZ, capture_output=True, text=True)
    test.assertEqual(r.returncode, 0, r.stderr)


def validar_restauracion(test):
    codigo = """
import { puedeRestaurar } from './scripts/recorridos/restaurar.mjs';
import assert from 'node:assert/strict';
const ajena = { id: 'persona', version: 2, perfiles: [{ perfil: 'profesional_etf' }], equipo: { equipo_id: 'equipo' } };
// Otra administración escribe N+1 y nuestro PUT recibe 409: nunca retirar su concesión.
assert.equal(puedeRestaurar(ajena, undefined, 'persona', 'equipo'), false);
// Nuestro PUT devuelve 2xx y confirma N+1: sí retirar nuestra concesión.
assert.equal(puedeRestaurar(ajena, { id: 'persona', version: 2 }, 'persona', 'equipo'), true);
// Cambio posterior, respuesta de otra persona o perfiles distintos: conservar.
assert.equal(puedeRestaurar({ ...ajena, version: 3 }, { id: 'persona', version: 2 }, 'persona', 'equipo'), false);
assert.equal(puedeRestaurar(ajena, { id: 'otra', version: 2 }, 'persona', 'equipo'), false);
assert.equal(puedeRestaurar({ ...ajena, perfiles: [{ perfil: 'consulta' }] }, { id: 'persona', version: 2 }, 'persona', 'equipo'), false);
"""
    r = subprocess.run(['node', '--input-type=module', '-e', codigo], cwd=RAIZ, capture_output=True, text=True)
    test.assertEqual(r.returncode, 0, r.stderr)
