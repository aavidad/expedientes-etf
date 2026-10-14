import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';
import { test } from 'node:test';
import ts from 'typescript';
const codigo = ts.transpileModule(readFileSync(new URL('../src/decision/api.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
const modulo = {};
new Function('exports', codigo)(modulo);
const { preparar, faltantes } = modulo;
const opciones = { municipios: [{ id: 'municipio', ine: '18003', equipo_id: 'equipo', centro_ss_id: 'centro' }] };
const derivacion = {
  derivacion_datos: { contenido: { tipo: 'anexo3', campos: [{ codigo: 'menores.persona.nombre', valor: 'Nerea Pérez Ortega' }, { codigo: 'valoracion.descripcion', valor: 'Apoyo familiar solicitado por el servicio municipal.' }] }, propuesta_municipio: { Candidatos: [{ Codigo: '18003' }] } },
};
test('municipio ambiguo no queda confirmado por la primera coincidencia', () => {
  const ambigua = structuredClone(derivacion); ambigua.derivacion_datos.propuesta_municipio.Candidatos.push({ Codigo: '18012' });
  const f = preparar(ambigua, opciones);
  assert.equal(f.municipio_id, ''); assert.equal(f.centro_ss_id, ''); assert.equal(f.equipo_id, '');
  assert(faltantes(f).includes('municipio_id'));
});
test('aceptación exige fecha real, puesto y sexo; descripción conserva lo recibido', () => {
  const f = preparar(derivacion, opciones);
  assert.equal(f.descripcion_situacion, 'Apoyo familiar solicitado por el servicio municipal.');
  assert.deepEqual(faltantes(f), ['fecha_acta', 'profesional_referencia_puesto', 'sexo-1']);
  const completa = { ...f, fecha_acta: '2026-10-13', profesional_referencia_puesto: 'TS', menores: [{ orden: 1, sexo: 'M' }] };
  assert.deepEqual(faltantes(completa), []);
  assert.strictEqual(preparar({ ...derivacion, aceptacion: completa }, opciones), completa);
});

test('la decisión conserva centro y equipo confirmados aunque difieran de la propuesta', () => {
  const confirmada = structuredClone(derivacion);
  confirmada.derivacion_datos.ubicacion_confirmada = { municipio_id: 'municipio', centro_id: 'centro-confirmado', equipo_id: 'equipo-confirmado' };
  const f = preparar(confirmada, opciones);
  assert.equal(f.municipio_id, 'municipio'); assert.equal(f.centro_ss_id, 'centro-confirmado'); assert.equal(f.equipo_id, 'equipo-confirmado');
});
