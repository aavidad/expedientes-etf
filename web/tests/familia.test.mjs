import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';
const codigo = ts.transpileModule(readFileSync(new URL('../src/equipo/familia-modelo.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
const modulo = {};
new Function('exports', codigo)(modulo);
test('ingresos conserva céntimos exactos con coma o punto y distingue importe desconocido', () => {
  for (const [entrada, esperado] of [['18000,75', 1800075], ['0,29', 29], ['12.30', 1230], ['0', 0], ['', null]]) assert.equal(modulo.centimos(entrada), esperado);
});
test('ingresos rechaza miles ambiguos, decimales excesivos, negativos y cantidades inseguras', () => {
  for (const entrada of ['18.000,75', '1,234', '-10', '1e3', 'texto', '90071992547410']) assert(Number.isNaN(modulo.centimos(entrada)), entrada);
});
