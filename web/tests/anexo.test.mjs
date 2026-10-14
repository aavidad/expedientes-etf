import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';
const salida = {};
const codigo = ts.transpileModule(readFileSync(new URL('../src/anexo/modelo.ts', import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
new Function('exports', codigo)(salida);
const analisis = (candidatos, campos = []) => ({ derivacion_datos: { propuesta_municipio: { Candidatos: candidatos }, contenido: { campos }, opciones_ubicacion: { centros: [{ id: 'c' }], equipos: [{ id: 'e' }] } } });
test('municipio ambiguo permanece vacío, con centro y equipo independientes', () => {
  assert.deepEqual(salida.propuestaInicial(analisis([{ MunicipioID: 'a' }, { MunicipioID: 'b' }])), { municipio_id: '', centro_id: '', equipo_id: '' });
  assert.deepEqual(salida.propuestaInicial(analisis([{ MunicipioID: 'a', CentroID: 'c', EquipoID: 'e' }])), { municipio_id: 'a', centro_id: 'c', equipo_id: 'e' });
});
test('PDF sin texto abre campos manuales sin inventar valores', () => {
  const campos = salida.camposIniciales(analisis([]));
  assert.ok(campos.some((c) => c.codigo === 'menores.persona.nombre'));
  assert.ok(campos.every((c) => c.valor === '' && c.dudoso));
});

test('asignación caducada queda vacía para elegir una vigente', () => {
 assert.deepEqual(salida.propuestaInicial(analisis([{ MunicipioID: 'a', CentroID: 'caducado', EquipoID: 'e' }])), { municipio_id: 'a', centro_id: '', equipo_id: 'e' });
});
