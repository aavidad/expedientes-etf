import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';
const codigo = ts.transpileModule(readFileSync(new URL('../src/spaf/api.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
const api = {};
new Function('exports', codigo)(api);

test('SPAF: mes anterior incluye el cambio de año y privacidad de iniciales', () => {
  assert.equal(api.mesAnterior(new Date(2026, 0, 14)), '2025-12');
  assert.equal(api.mesAnterior(new Date(2026, 9, 14)), '2026-09');
  assert.equal(api.iniciales('  Lucía Serrano Ruiz '), 'L. S. R.');
  assert.equal(api.mesValido('2026-00'), false); assert.equal(api.mesValido('0000-01'), false); assert.equal(api.mesValido('2026-09'), true);
});

test('SPAF: recuperación en memoria conserva idioma/versiones y separa actores sin nombres', () => {
  const codigo = ts.transpileModule(readFileSync(new URL('../src/spaf/recuperacion.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
  const memoria = {}; new Function('exports', codigo)(memoria);
  const tabla = { equipo_id: 'equipo', anio: 2026, mes: 9, version_regla: 'regla-2', filas: [{ expediente_id: 'expediente', version_ficha: 'ficha-3', menores: { Valor: [{ nombre: 'Lucía' }] } }] };
  const primera = memoria.preparar('actor-1', tabla, 'es');
  assert.deepEqual(memoria.preparar('actor-1', { ...tabla, version_regla: 'nueva' }, 'en'), primera);
  assert.equal(memoria.recuperar('actor-2'), undefined);
  assert(!JSON.stringify(primera).includes('Lucía'));
  memoria.olvidar('actor-1'); assert.equal(memoria.recuperar('actor-1'), undefined);
});
test('SPAF: emisión conserva versiones de vista previa y distingue desconocido de cero', () => {
  const tabla = { equipo_id: 'equipo', anio: 2026, mes: 9, version_regla: 'regla-2', filas: [{ expediente_id: 'expediente', version_ficha: 'ficha-3' }] };
  assert.deepEqual(api.solicitudWord(tabla, 'en'), { idioma: 'en', versiones: { regla: 'regla-2', fichas: { expediente: 'ficha-3' } } });
  assert.deepEqual(api.solicitudWord({ ...tabla, filas: null }, 'es').versiones.fichas, {});
  assert.equal(api.validarTabla({ ...tabla, resumen: { casos: 0, altas: 0, bajas: 0, menores: 0, casos_sin_dato_menores: 0 }, filas: null }, 'equipo', '2026-09'), true);
  assert.equal(api.validarTabla({ ...tabla, resumen: { casos: 0 }, filas: null }, 'otro', '2026-09'), false);
});
test('SPAF: POST real usa CSRF y la misma clave de recuperación; 409 bloquea descarga', async () => {
  const anterior = globalThis.fetch; const peticiones = [];
  const tabla = { equipo_id: 'equipo', anio: 2026, mes: 9, version_regla: 'regla-2', filas: null };
  globalThis.fetch = async (url, p) => { peticiones.push({ url, ...p }); return new Response('{}', { status: 409 }); };
  try {
    for (let i = 0; i < 2; i++) await assert.rejects(api.emitir(tabla, 'es', 'csrf', 'clave-recuperacion', new AbortController().signal), e => e.estado === 409 && e.clave === 'spaf.conflicto');
    assert(peticiones.every(p => p.method === 'POST' && p.url.endsWith('/2026/9.docx') && p.headers['X-CSRF-Token'] === 'csrf' && p.headers['Idempotency-Key'] === 'clave-recuperacion'));
    assert.equal(peticiones[0].body, peticiones[1].body);
  } finally { globalThis.fetch = anterior; }
});
test('SPAF: un 5xx de descarga conserva resultado incierto para recuperar la misma petición', async () => {
  const anterior = globalThis.fetch;
  globalThis.fetch = async () => new Response('{}', { status: 502 });
  try {
    await assert.rejects(api.emitir({ equipo_id: 'equipo', anio: 2026, mes: 9, version_regla: 'regla', filas: null }, 'es', 'csrf', 'clave', new AbortController().signal), e => e.clave === 'spaf.descarga_desconocida' && e.estado === 0);
  } finally { globalThis.fetch = anterior; }
});
