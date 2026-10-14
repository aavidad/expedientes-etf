import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';
const leer = ruta => readFileSync(new URL(ruta, import.meta.url), 'utf8');
const exports = {};
const codigo = ts.transpileModule(leer('../src/traslado/api.ts'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
new Function('require', 'exports', codigo)(() => ({}), exports);
test('traslado rechaza fechas inexistentes y conserva los límites civiles', () => {
  assert.equal(exports.fechaValida('2026-02-30', '', '2026-10-14'), false);
  assert.equal(exports.fechaValida('2026-02-29', '', '2026-10-14'), false);
  assert.equal(exports.fechaValida('2024-02-29', '', '2026-10-14'), true);
  assert.equal(exports.fechaValida('2026-10-12', '2026-10-13', '2026-10-14'), false);
  assert.equal(exports.fechaValida('2026-10-13', '2026-10-13', '2026-10-14'), true);
  assert.equal(exports.fechaValida('2026-10-15', '', '2026-10-14'), false);
  assert.equal(exports.fecha('2026-10-25', 'es'), '25/10/2026');
});
test('traslado usa tokens publicados por ambos temas', () => {
  const css = leer('../src/traslado/traslado.css');
  for (const tema of ['expedientes-etf', 'alto-contraste']) {
    const tokens = new Set(Object.keys(JSON.parse(leer(`../../temas/${tema}/tokens.json`))).map(k => `--${k.replaceAll('.', '-')}`));
    for (const [, token] of css.matchAll(/var\((--[\w-]+)\)/g)) assert(tokens.has(token), token);
  }
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b|\b(?:rgb|hsl)a?\(|\d(?:px|rem|em)\b/i);
});
