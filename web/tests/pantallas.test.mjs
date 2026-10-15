import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { contratoServidor } from './contrato-spa.mjs';
import { createRequire } from 'node:module';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';

const raiz = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const require = createRequire(import.meta.url);
const cache = new Map();
// Ejecuta las listas y validadores TypeScript reales. Los componentes se
// sustituyen aquí: su montaje se comprueba en los recorridos de navegador.
function cargar(fichero) {
  const ruta = resolve(raiz, fichero);
  if (cache.has(ruta)) return cache.get(ruta);
  const salida = {};
  cache.set(ruta, salida);
  const codigo = ts.transpileModule(readFileSync(ruta, 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
  const importar = (nombre) => {
    if (nombre === 'react') return require(nombre);
    const destino = resolve(dirname(ruta), nombre) + '.ts';
    if (destino.endsWith('/pantallas.ts') || destino.endsWith('/registro-pantallas.ts')) return cargar(destino);
    return new Proxy({}, { get: () => () => null });
  };
  new Function('require', 'exports', codigo)(importar, salida);
  return salida;
}
const { pantallas } = cargar('web/src/pantallas.ts');
const { validarPantallas, menuPermitido, pantallaPermitida } = cargar('web/src/registro-pantallas.ts');

test('registro de pantallas rechaza rutas duplicadas y vacío', () => {
  assert.doesNotThrow(() => validarPantallas(pantallas));
  assert.throws(() => validarPantallas([...pantallas, pantallas[0]]), /http.registro_invalido/);
  assert.throws(() => validarPantallas([]), /http.registro_invalido/);
  assert.throws(() => validarPantallas([{ ...pantallas[0], menuDe: '/ausente' }]), /http.registro_invalido/);
});

test('cada pantalla y entrada del menú tiene SPA, perfiles y clave en el registro del servidor', () => {
  const servidor = contratoServidor();
  for (const pantalla of pantallas) {
    const ruta = servidor.find((r) => r.Patron === pantalla.ruta);
    assert.ok(ruta, pantalla.ruta);
    assert.ok(ruta.Pantalla);
    assert.ok(ruta.Metodos.includes('GET'));
    assert.deepEqual(ruta.Perfiles, pantalla.perfiles);
    assert.equal(ruta.ClaveTitulo, pantalla.clave);
    assert.equal(typeof pantalla.componente, 'function');
  }
  assert.deepEqual(servidor.filter((p) => p.ClaveTitulo).map((p) => p.Patron), pantallas.filter((p) => p.clave).map((p) => p.ruta));
});

test('menú exige sesión, perfil y pantalla publicada; conserva orden y alias', () => {
  const admin = { estado: 'activo', csrf: '', perfiles: ['administracion'], pantallas: pantallas.filter((p) => p.clave && p.perfiles.includes('administracion') && p.ruta !== '/admin/restauracion').map(({ ruta, clave, perfiles }) => ({ ruta, clave, perfiles: perfiles.filter((perfil) => perfil === 'administracion') })) };
  assert.deepEqual(menuPermitido(pantallas, admin).map((p) => p.ruta), ['/admin/usuarios', '/admin/catalogos', '/admin/reglas', '/admin/firmantes', '/admin/ubicaciones', '/admin/paquetes', '/admin/idiomas', '/admin/temas', '/admin/plantillas', '/admin/carga-historica', '/admin/conservacion', '/admin/auditoria', '/ayuda']);

  assert.equal(pantallaPermitida(pantallas, admin, '/admin/catalogos/listas')?.menuDe, '/admin/catalogos');
  assert.deepEqual(menuPermitido(pantallas, { ...admin, perfiles: ['consulta'] }), []);
  assert.deepEqual(menuPermitido(pantallas, { ...admin, estado: 'pendiente_alta' }), []);
  assert.deepEqual(menuPermitido(pantallas, { ...admin, pantallas: [] }), []);
  assert.deepEqual(menuPermitido(pantallas), []);
  assert.equal(pantallaPermitida(pantallas, admin, '/desconocida'), undefined);
});

test('detalle de decisión exige coordinación y un solo segmento de derivación', () => {
  const acceso = { estado: 'activo', perfiles: ['coordinacion'], pantallas: [{ ruta: '/coordinacion/lista-espera', perfiles: ['coordinacion'] }] };
  assert.equal(pantallaPermitida(pantallas, acceso, '/coordinacion/derivaciones/22222222-2222-4222-8222-222222222222')?.menuDe, '/coordinacion/lista-espera');
  for (const ruta of ['/coordinacion/derivaciones/', '/coordinacion/derivaciones/id/otra', '/coordinacion/derivaciones/..', '/coordinacion/derivaciones/%2f']) assert.equal(pantallaPermitida(pantallas, acceso, ruta), undefined);
  assert.equal(pantallaPermitida(pantallas, { ...acceso, perfiles: ['consulta'] }, '/coordinacion/derivaciones/id'), undefined);
});


test('revisión tras restauración sólo ofrece la pantalla publicada por la sesión', () => {
  const acceso = { estado: 'activo', perfiles: ['administracion', 'coordinacion'], pantallas: [{ ruta: '/admin/restauracion', perfiles: ['administracion'], clave: 'restauracion.titulo' }] };
  assert.deepEqual(menuPermitido(pantallas, acceso).map((p) => p.ruta), ['/admin/restauracion']);
  assert.equal(pantallaPermitida(pantallas, acceso, '/admin/usuarios'), undefined);
  assert.equal(pantallaPermitida(pantallas, acceso, '/coordinacion'), undefined);
});
