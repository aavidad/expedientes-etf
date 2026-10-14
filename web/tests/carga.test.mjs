import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import ts from 'typescript';
const codigo = ts.transpileModule(readFileSync(new URL('../src/carga/modelo.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
const exports = {}; new Function('exports', codigo)(exports);
const { formulario, preparar, hojaInventada, columnas } = exports;
test('no aceptación elimina número y datos exclusivos de aceptación', () => {
 const f = { ...formulario(), equipo: 'equipo', estado: 'no_aceptada', numero: '02-12-26', causa: 'Retirada', noAceptacion: '2026-02-02' };
 const p = preparar(f).casos[0]; assert.equal(p.numero, ''); assert.equal(p.aceptacion, undefined); assert.equal(p.no_aceptacion.motivo, 'Retirada');
 for (const campo of ['episodios', 'familia', 'menores']) assert.deepEqual(p[campo], []);
});
test('aceptación conserva número, vigencia, identidad de menores y reapertura', () => {
 const f = { ...formulario(), equipo: 'equipo', numero: '02-12-25', estado: 'en_intervencion', aceptacion: '2025-12-01', subprograma: 'RIESGO_SOCIAL' };
 f.menores[0] = { ...f.menores[0], nombre: 'Clara', apellidos: 'Villena Robles', sexo: 'M' };
 f.episodios[0].cierre_fecha = '2026-01-14'; f.episodios[0].motivo_cierre = 'TRASLADO';
 f.episodios.push({ apertura_fecha: '2026-03-03', cierre_fecha: '', motivo_cierre: '', motivo_acta_texto: '', motivo_reapertura: 'Regreso al municipio' });
 const p = preparar(f).casos[0]; assert.equal(p.numero, '02-12-25'); assert.equal(p.episodios[0].apertura_fecha, '2025-12-01'); assert.equal(p.episodios[1].motivo_reapertura, 'Regreso al municipio');
 assert.equal(p.menores[0].menores[0].id, f.menores[0].id); assert.equal(p.menores[0].menores[0].fecha_nacimiento, null); assert.equal(p.menores[0].vigente_desde, f.aceptacion);
});
test('hoja descargable incluye las dieciocho columnas y datos inventados de idioma', () => {
 const textos = JSON.parse(readFileSync(new URL('../../idiomas/es/textos/carga.json', import.meta.url)));
 const csv = hojaInventada(k => textos[k]); assert.equal(columnas.length, 18); assert.ok(csv.startsWith('\uFEFF' + columnas.join(';'))); assert.ok(csv.includes('Sofía Molina Daza')); assert.ok(csv.includes('2026-02-12'));
});
