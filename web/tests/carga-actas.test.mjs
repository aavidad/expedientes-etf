import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import ts from 'typescript';
function cargar(nombre, dependencias = {}) {
 const codigo = ts.transpileModule(readFileSync(new URL(`../src/carga/${nombre}.ts`, import.meta.url), 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
 const exports = {}; new Function('exports', 'require', codigo)(exports, k => dependencias[k]); return exports;
}
const modelo = cargar('modelo');
const { desdeActas } = cargar('actas-modelo', { './modelo': modelo });
const opciones = { equipos: [{ id: 'equipo2', codigo: '02', clave_texto: 'equipo.2' }, { id: 'equipo4', codigo: '04', clave_texto: 'equipo.4' }], municipios: [{ id: 'santafe', equipo_id: 'equipo2', ambito_diputacion: true, clave_texto: 'municipio.santafe' }], centros: [{ id: 'centro', clave_texto: 'centro.santafe' }], subprogramas: [{ codigo: 'RIESGO_SOCIAL', clave_texto: 'subprograma.RIESGO_SOCIAL' }], textos: { 'equipo.2': 'ETF - 2 Vega Temple - Santa Fe', 'equipo.4': 'ETF - 4 Valle Lecrín', 'municipio.santafe': 'Santa Fe', 'centro.santafe': 'Santa Fe', 'subprograma.RIESGO_SOCIAL': 'Riesgo social', 'motivo_finalizacion.OBJ_TOTAL': 'Objetivos alcanzados' } };
const lectura = campos => ({ contenido: { tipo: 'aceptacion', plantilla_reconocida: true, campos: Object.entries(campos).map(([codigo, valor]) => ({ codigo, valor })) }, antivirus: 'limpio' });
test('lectura conserva año 2022 y deja desconocidos estado, derivación y sexo de adultos', () => {
 const f = desdeActas('equipo2', opciones, lectura({ numero: '02-71-22', fecha: '2022-05-17', familia: 'Familia Villena Robles', madre: 'Irene Robles Serrano', municipio: 'Santa Fe', centro: 'Santa Fe', subprograma: 'Riesgo social', 'menor.nombre': 'Clara Villena Robles', 'menor.sexo': '☐Varón ☒Femenino', 'menor.nacimiento': '2014-06-14' }));
 assert.equal(f.numero, '02-71-22'); assert.equal(f.aceptacion, '2022-05-17'); assert.equal(f.estado, ''); assert.equal(f.derivacion, ''); assert.equal(f.adultos[0].sexo, ''); assert.equal(f.adultos[0].papel, 'MADRE'); assert.equal(f.menores[0].sexo, 'M'); assert.equal(f.municipio, 'santafe');
 f.adultos[0].sexo = 'M'; f.estado = 'en_intervencion';
 const caso = modelo.preparar(f).casos[0]; assert.equal(caso.familia[0].adultos[0].nombre, 'Irene Robles Serrano'); assert.equal(caso.familia[0].vigente_desde, '2022-05-17'); assert.equal(caso.familia[0].adultos[0].fecha_nacimiento, null);
});
test('cierre de 2026 conserva motivo libre y exige elegir catálogo si no coincide', () => {
 const f = desdeActas('equipo2', opciones, lectura({ numero: '02-72-23', fecha: '2023-03-09' }), lectura({ numero: '02-72-23', fecha: '2026-06-20', motivo: 'La familia ha alcanzado los objetivos.' }));
 assert.equal(f.estado, 'cerrada'); assert.equal(f.episodios[0].cierre_fecha, '2026-06-20'); assert.equal(f.episodios[0].motivo_cierre, ''); assert.equal(f.episodios[0].motivo_acta_texto, 'La familia ha alcanzado los objetivos.');
});
test('actas de otro equipo o de distinto número no rellenan el formulario', () => {
 assert.throws(() => desdeActas('equipo2', opciones, lectura({ numero: '04-01-22' })), /equipo_distinto/);
 assert.throws(() => desdeActas('equipo2', opciones, lectura({ equipo: 'ETF - 4 Valle Lecrín' })), /equipo_distinto/);
 assert.throws(() => desdeActas('equipo2', opciones, lectura({ numero: '02-71-22' }), lectura({ numero: '02-72-22' })), /numero_distinto/);
});
test('valores ambiguos y casillas sin marcar quedan pendientes', () => {
 const a = lectura({ fecha: '2022-05-17', 'menor.nombre': 'Clara', 'menor.sexo': '☐Varón ☐Femenino' }); a.contenido.campos.push({ codigo: 'fecha', valor: '2022-05-18' });
 const f = desdeActas('equipo2', opciones, a); assert.equal(f.aceptacion, ''); assert.equal(f.menores[0].sexo, ''); assert.equal(f.centro, '');
});
test('separar nombres no infiere apellidos y excluye la propuesta del contrato', () => {
 const leido = { ...modelo.adulto(), nombre:'Irene Robles Serrano', papel:'MADRE', nombreCompleto:true };
 const separado = modelo.separarNombre(leido); assert.equal(separado.nombre,''); assert.equal(separado.apellidos,''); assert.equal(separado.nombreLeido,'Irene Robles Serrano'); assert.equal(separado.id,leido.id);
 const f=modelo.formulario(); assert.equal(modelo.tieneDatos(f),false); f.equipo='equipo2'; assert.equal(modelo.tieneDatos(f),false); f.adultos=[{...separado,nombre:'Irene',apellidos:'Robles Serrano',sexo:'M'}]; f.estado='aceptada'; assert.equal(modelo.tieneDatos(f),true);
 const payload=modelo.preparar(f).casos[0].familia[0].adultos[0]; assert.equal(payload.nombre,'Irene'); assert.equal(payload.apellidos,'Robles Serrano'); assert.equal(payload.nombreLeido,undefined); assert.equal(payload.nombreCompleto,undefined);
});
