import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';

const leer = (ruta) => readFileSync(new URL(ruta, import.meta.url), 'utf8');
const exports = {};
const codigo = ts.transpileModule(leer('../src/equipo/api.ts'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
new Function('require', 'exports', codigo)(() => ({}), exports);

test('equipo presenta horas de Granada y conserva las fechas civiles', () => {
  assert.match(exports.fecha('2026-10-13T21:40:00Z', 'es', true), /13\/10\/2026.*23:40/);
  assert.match(exports.fecha('2026-01-14T23:30:00Z', 'es', true), /15\/01\/2026.*00:30/);
  assert.equal(exports.fecha('2026-10-13', 'es'), '13/10/2026');
  assert.equal(exports.fecha('2026-10-13T22:40:00Z', 'es'), '14/10/2026');
});

test('equipo usa tokens disponibles en ambos temas', () => {
  const css = leer('../src/equipo/equipo.css');
  for (const tema of ['expedientes-etf', 'alto-contraste']) {
    const tokens = new Set(Object.keys(JSON.parse(leer(`../../temas/${tema}/tokens.json`))).map((k) => `--${k.replaceAll('.', '-')}`));
    for (const [, token] of css.matchAll(/var\((--[\w-]+)\)/g)) assert(tokens.has(token), token);
  }
  assert.doesNotMatch(css.replace(/@media\s*\([^)]*\)/g, ''), /#[0-9a-f]{3,8}\b|\b(?:rgb|hsl)a?\(|\d(?:px|rem|em)\b/i);
});
