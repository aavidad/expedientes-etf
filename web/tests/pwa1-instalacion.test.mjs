import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';

const fuente = readFileSync(new URL('../src/pwa1/instalacion.ts', import.meta.url), 'utf8');
const js = ts.transpileModule(fuente, { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } }).outputText;
const { actualizarInstalacion } = await import(`data:text/javascript;base64,${Buffer.from(js).toString('base64')}`);

test('cambiar idioma o tema actualiza el manifiesto sin registrar otro worker', async (t) => {
  const manifiesto = { href: '' };
  const registros = [];
  const mensajes = [];
  const oyentes = new Map();
  const anteriores = { document: globalThis.document, window: globalThis.window, navigator: globalThis.navigator };
  t.after(() => { for (const [nombre, valor] of Object.entries(anteriores)) Object.defineProperty(globalThis, nombre, { value: valor, configurable: true, writable: true }); });
  const serviceWorker = {
    controller: { postMessage: datos => mensajes.push(datos) },
    ready: Promise.resolve(),
    register: async (url, opciones) => { registros.push([url, opciones]); },
    addEventListener: (tipo, f) => oyentes.set(tipo, f),
    removeEventListener: tipo => oyentes.delete(tipo),
  };
  Object.defineProperty(globalThis, 'document', { value: { querySelector: () => manifiesto }, configurable: true, writable: true });
  Object.defineProperty(globalThis, 'window', { value: { isSecureContext: true }, configurable: true, writable: true });
  Object.defineProperty(globalThis, 'navigator', { value: { serviceWorker }, configurable: true, writable: true });

  const quitarEs = actualizarInstalacion('es', 'expedientes-etf');
  assert.equal(manifiesto.href, '/manifest.webmanifest?idioma=es&tema=expedientes-etf');
  quitarEs();
  const quitarEn = actualizarInstalacion('en', 'alto-contraste');
  assert.equal(manifiesto.href, '/manifest.webmanifest?idioma=en&tema=alto-contraste');
  await new Promise(r => setTimeout(r, 0));
  assert.deepEqual(registros.map(([url]) => url), ['/service-worker.js', '/service-worker.js']);
  assert.deepEqual(registros[0][1], { scope: '/', updateViaCache: 'none' });
  assert.deepEqual(mensajes.at(-1), { tipo: 'preferencias', idioma: 'en', tema: 'alto-contraste' });
  quitarEn();
  assert.equal(oyentes.size, 0);
});
