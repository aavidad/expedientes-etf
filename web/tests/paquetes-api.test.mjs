import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';

const fuente = readFileSync(new URL('../src/paquetes-api.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(fuente, { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } }).outputText;
const api = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

test('traduce claves de servidor sin exponer diagnósticos desconocidos', () => {
  assert.equal(api.claveError({ error: { clave: 'paquete.conflicto' } }), 'paquete.conflicto');
  assert.equal(api.claveError({ error: 'SQL contraseña interna' }), 'paquete.no_disponible');
  assert.equal(api.claveError({ error: { clave: '<script>' } }), 'paquete.no_disponible');
});

test('no confirma una activación con respuesta incompleta o versión equivocada', async (t) => {
  const original = globalThis.fetch;
  t.after(() => { globalThis.fetch = original; });
  const p = { codigo: 'fr', tipo: 'idioma', version_contenido: '1.0.0', huella: 'a'.repeat(64), ticket: 'b'.repeat(64) };
  const c = new AbortController();
  for (const datos of [{}, { ...p, version: 9 }, { ...p, codigo: 'en', version: 1 }]) {
    globalThis.fetch = async () => ({ ok: true, json: async () => datos });
    await assert.rejects(api.activar(p, 0, c.signal, 'testigo-sesion'), { message: 'paquete.no_disponible' });
  }
  globalThis.fetch = async (_, opciones) => {
    assert.equal(opciones.headers['X-ETF-Paquetes'], '1');
    assert.equal(opciones.headers['X-CSRF-Token'], 'testigo-sesion');
    assert.deepEqual(JSON.parse(opciones.body), { ticket: p.ticket, version: 0 });
    return { ok: true, json: async () => ({ ...p, version: 1 }) };
  };
  assert.equal((await api.activar(p, 0, c.signal, 'testigo-sesion')).version, 1);
});

test('subida distingue transferencia de revisión antivirus y admite cancelación', async (t) => {
  const original = globalThis.XMLHttpRequest;
  t.after(() => { globalThis.XMLHttpRequest = original; });
  globalThis.XMLHttpRequest = class {
    upload = {};
    open() {}
    setRequestHeader(nombre, valor) { if (nombre === 'X-CSRF-Token') assert.equal(valor, 'testigo-sesion'); }
    abort() { this.onabort(); }
    send() {
      this.upload.onload();
      this.status = 201;
      this.response = { codigo: 'fr', tipo: 'idioma', version_contenido: '1.0.0', huella: 'a'.repeat(64), ticket: 'b'.repeat(64) };
      this.onload();
    }
  };
  const estados = [];
  const c = new AbortController();
  const p = await api.subir('idioma', 'fr', {}, c.signal, (estado) => estados.push(estado), 'testigo-sesion');
  assert.deepEqual(estados, ['subiendo', 'antivirus']);
  assert.equal(p.codigo, 'fr');
  c.abort();
  await assert.rejects(api.subir('idioma', 'fr', {}, c.signal, () => {}, 'testigo-sesion'), { name: 'AbortError' });
});
