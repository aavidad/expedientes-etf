// F2.07: ficha real, sesiones independientes, historia y comprobaciones al corte.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { randomUUID } from 'node:crypto';
import { chromium, baseLocal, restringir } from './comun.mjs';

const base = baseLocal(process.argv[2]);
assert(process.argv[3], 'Falta el directorio de evidencia');
const dir = resolve(process.argv[3]);
const raiz = resolve(new URL('../..', import.meta.url).pathname);
assert(dir !== raiz && !dir.startsWith(raiz + '/'), 'La evidencia debe quedar fuera del repositorio');
await mkdir(dir, { recursive: true });
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const textos = Object.assign({}, ...await Promise.all(['equipo', 'fi1', 'fi2', 'expediente'].map(async n =>
  JSON.parse(await readFile(new URL(`../../idiomas/es/textos/${n}.json`, import.meta.url))))));
const t = clave => { assert(textos[clave], clave); return textos[clave]; };
const navegadores = [];
const sesiones = [];
const pasos = [];
const fallos = [];
const errores = [];
const red = [];
const datos = {};
const tabs = ['familia', 'menores', 'intervencion', 'consentimiento', 'evaluacion', 'seguimiento', 'documentos', 'cierre', 'salidas', 'historial'];
const ruta = (id, sufijo = '') => `/api/expedientes/${id}${sufijo ? '/' + sufijo : ''}`;
function ok(paso) { pasos.push(paso); console.log(`F2.07-${paso}-OK`); }
function producto(condicion, codigo, observado) {
  if (!condicion) { fallos.push({ codigo, observado }); console.error(`F2.07-PRODUCTO: ${codigo}`); }
}
async function guardar(nombre, contenido) { await writeFile(resolve(dir, nombre), JSON.stringify(contenido, null, 2) + '\n'); }
function sinDatosPersonales(valor) {
  if (valor === null || typeof valor !== 'object') return;
  for (const [clave, dato] of Object.entries(valor)) {
    assert(!['nombre', 'apellidos', 'direccion', 'telefono', 'telefono_contacto', 'email', 'email_contacto', 'otros_datos'].includes(clave), clave);
    if (typeof dato === 'string') for (const nombre of ['Alicia', 'Lucía', 'Daniel', 'Torres Montes']) assert(!dato.includes(nombre), nombre);
    sinDatosPersonales(dato);
  }
}
async function entrar(cuenta, separado = false) {
  const navegador = separado || !navegadores.length
    ? await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true }) : navegadores[0];
  if (!navegadores.includes(navegador)) navegadores.push(navegador);
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
  await restringir(contexto, base);
  await contexto.tracing.start({ screenshots: true, snapshots: true });
  const p = await contexto.newPage();
  const indiceSesion = sesiones.length;
  p.setDefaultTimeout(15000);
  p.on('pageerror', e => errores.push(e.message));
  p.on('response', r => {
    if (r.url().includes('/api/')) red.push({ sesion: indiceSesion, metodo: r.request().method(), ruta: new URL(r.url()).pathname, estado: r.status() });
  });
  await p.goto(base);
  await p.locator('#cuenta-ad').fill(cuenta);
  await p.locator('#contrasena-ad').fill(claves[cuenta]);
  const respuesta = p.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/api/sesion'));
  await p.locator('button[type="submit"]').click();
  const r = await respuesta;
  assert.equal(r.status(), 200);
  const sesion = { p, contexto, csrf: (await r.json()).csrf };
  await p.locator('#cuenta-ad').waitFor({ state: 'detached' });
  sesiones.push(sesion);
  return sesion;
}
async function pedir(s, path, metodo = 'GET', body, anexo, clave = randomUUID()) {
  return s.p.evaluate(async ({ path, metodo, body, anexo, csrf, clave }) => {
    const headers = { 'X-CSRF-Token': csrf, 'Idempotency-Key': clave };
    if (anexo) {
      body = new FormData();
      body.append('anexo', new Blob([new Uint8Array(anexo)], { type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' }), 'anexo3.docx');
    } else if (body !== undefined) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(body); }
    const r = await fetch(path, { method: metodo, headers, body });
    const texto = await r.text();
    return { estado: r.status, datos: texto ? JSON.parse(texto) : null };
  }, { path, metodo, body, anexo, csrf: s.csrf, clave });
}
async function api(s, path, metodo, body, esperado = 200, anexo) {
  const r = await pedir(s, path, metodo, body, anexo);
  assert.equal(r.estado, esperado, `${path}: ${JSON.stringify(r.datos)}`);
  return r.datos;
}
async function abrir(s, id, tab) {
  await s.p.goto(`${base}/equipo/expediente?id=${id}&pestana=${tab}`);
  await s.p.locator(`#equipo-tab-${tab}[aria-selected="true"]`).waitFor();
  await s.p.locator(`#equipo-panel-${tab}`).waitFor();
}
async function guardarUI(s, id, esperado) {
  const r = s.p.waitForResponse(r => r.request().method() === 'PUT' && r.url().endsWith(ruta(id, 'menores')));
  await s.p.getByRole('button', { name: t('fi1.guardar'), exact: true }).click();
  const respuesta = await r;
  assert.equal(respuesta.status(), esperado, await respuesta.text());
  const datos = await respuesta.json();
  if (esperado === 200) {
    const peticion = respuesta.request();
    const repetida = await pedir(s, ruta(id, 'menores'), 'PUT', peticion.postDataJSON(), undefined, peticion.headers()['idempotency-key']);
    assert.equal(repetida.estado, 200);
    assert.deepEqual(repetida.datos, datos);
    assert.equal((await api(s, ruta(id))).version, datos.version);
  }
  return datos;
}
async function crear(s, equipo) {
  const opciones = await api(s, '/api/coordinacion/decision/opciones?idioma=es');
  const m = opciones.municipios.find(m => m.equipo_id === equipo && m.centro_ss_id);
  assert(m);
  const bytes = Array.from(await readFile(new URL('../../internal/adaptadores/anexos/docx/testdata/anexo3.docx', import.meta.url)));
  const lectura = await api(s, '/api/coordinacion/derivaciones', 'POST', undefined, 200, bytes);
  const d = await api(s, '/api/coordinacion/derivaciones/confirmar', 'POST', {
    lectura_id: lectura.lectura_anexo.id, tipo: 'anexo3',
    campos: lectura.derivacion_datos.contenido.campos.map(c => ({ ...c, dudoso: false, motivos: [] })),
    ubicacion: { municipio_id: m.id, centro_id: m.centro_ss_id, equipo_id: equipo },
  }, 201);
  await api(s, `/api/coordinacion/derivaciones/${d.id}/aceptar`, 'POST', {
    version: d.version, fecha_acta: '2026-07-01', municipio_id: m.id, centro_ss_id: m.centro_ss_id,
    equipo_id: equipo, subprograma: 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', profesional_referencia_id: null,
    menores: lectura.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: 'M' })),
    descripcion_situacion: 'La familia solicita apoyo para organizar los cuidados y la asistencia escolar.',
  });
  return d.id;
}

try {
  const a = await entrar('marta.cabrera');
  const b = await entrar('marta.cabrera', true);
  const otro = await entrar('pablo.ruiz');
  const coord = await entrar('ines.moreno');
  const equipo = (await api(a, '/api/sesion')).equipo_id;
  assert.notEqual((await api(otro, '/api/sesion')).equipo_id, equipo);
  const id = await crear(coord, equipo);
  datos.expediente = id;
  const familia = await api(a, ruta(id, 'familia'));
  await api(a, ruta(id), 'PATCH', { version: familia.version, vigente_desde: '2026-07-01', familia: { ...familia.familia, familia_numerosa: false } });
  await api(a, ruta(id, 'adultos'), 'PUT', {
    version: (await api(a, ruta(id))).version, vigente_desde: '2026-07-01',
    adultos: [{ orden: 1, nombre: 'Alicia', apellidos: 'Torres Montes', papel: 'MADRE', sexo: 'M', fecha_nacimiento: '1985-02-12' }],
  });
  await api(a, ruta(id, 'menores'), 'PUT', {
    version: (await api(a, ruta(id))).version, vigente_desde: '2026-07-01', menores: [
      { orden: 1, nombre: 'Lucía', apellidos: 'Torres Montes', sexo: 'M', fecha_nacimiento: '2016-05-11', subprograma: 'RIESGO_SOCIAL', propuesto_atencion_temprana: false },
      { orden: 2, nombre: 'Daniel', apellidos: 'Torres Montes', sexo: 'H', fecha_nacimiento: '2013-03-21', subprograma: 'RIESGO_SOCIAL', propuesto_atencion_temprana: false },
    ],
  });
  await api(a, ruta(id, 'intervencion'), 'PUT', { version: (await api(a, ruta(id))).version, vigente_desde: '2026-07-01', intervencion: { fase: 'RECEPCION', listas: { coordinacion: [], factor_familia: [], factor_menor: [], tipologia: [] } } });
  const tabla = mes => api(a, `/api/estadistica/spaf?anio=2026&mes=${mes}&etf=${equipo}`);
  datos.julioAntes = await tabla(7);
  const menores = await api(a, ruta(id, 'menores'));
  const menor = menores.menores[0];
  // Dos procesos de Chrome abren exactamente la misma versión antes de escribir.
  await abrir(a, id, 'menores');
  await abrir(b, id, 'menores');
  const nombre = `#fi1-nombre-${menor.id}`;
  await Promise.all([a.p.locator(nombre).waitFor(), b.p.locator(nombre).waitFor()]);
  await a.p.getByLabel(t('fi1.vigente_desde'), { exact: true }).fill('2026-09-01');
  await b.p.getByLabel(t('fi1.vigente_desde'), { exact: true }).fill('2026-09-01');
  await b.p.locator(nombre).fill('Elena');
  await a.p.locator(`#fi1-centro_educativo-${menor.id}`).fill('Colegio Los Olivos');
  datos.primerGuardado = await guardarUI(a, id, 200);
  await a.p.getByText(t('fi1.guardado'), { exact: true }).waitFor();
  const vigente = await api(a, ruta(id, 'menores'));
  datos.conflicto = await guardarUI(b, id, 409);
  await b.p.getByText(t('fi1.conflicto'), { exact: true }).waitFor();
  await b.p.getByText(t('fi1.conflicto_detalle'), { exact: true }).waitFor();
  assert.equal(await b.p.locator(nombre).inputValue(), 'Elena');
  assert.deepEqual(await api(a, ruta(id, 'menores')), vigente);
  await b.p.screenshot({ path: resolve(dir, 'conflicto.png'), fullPage: true });
  await b.p.getByRole('button', { name: t('fi1.ver_actuales'), exact: true }).click();
  await b.p.getByText('Colegio Los Olivos', { exact: true }).waitFor();
  assert.equal(await b.p.locator(nombre).inputValue(), 'Elena');
  await b.p.getByRole('button', { name: t('fi1.usar_actuales'), exact: true }).click();
  assert.equal(await b.p.locator(nombre).inputValue(), 'Lucía');
  ok('CONFLICTO');

  await api(a, ruta(id), 'PATCH', { version: (await api(a, ruta(id))).version, vigente_desde: '2026-09-01', familia: { ...familia.familia, familia_numerosa: true } });
  await api(a, ruta(id, 'menores'), 'PUT', { version: (await api(a, ruta(id))).version, vigente_desde: '2026-09-01', menores: [vigente.menores[0]] });
  await api(a, ruta(id, 'intervencion'), 'PUT', { version: (await api(a, ruta(id))).version, vigente_desde: '2026-09-01', intervencion: { fase: 'INTERVENCION', listas: { coordinacion: [], factor_familia: [], factor_menor: [], tipologia: [] } } });
  datos.historia = {};
  for (const corte of ['2026-07-31', '2026-08-31', '2026-09-01', '2026-09-30']) {
    const f = await api(a, ruta(id, `familia/historia?corte=${corte}&anio=2026`));
    const m = await api(a, ruta(id, `menores/historia?corte=${corte}`));
    const i = await api(a, ruta(id, `intervencion/historia?corte=${corte}`));
    const anterior = corte < '2026-09-01';
    assert.equal(f.familia.familia_numerosa, !anterior);
    assert.equal(m.menores.length, anterior ? 2 : 1);
    assert.equal(i.intervencion.fase, anterior ? 'RECEPCION' : 'INTERVENCION');
    sinDatosPersonales({ f, m, i });
    datos.historia[corte] = { familia: f, menores: m, intervencion: i };
  }
  ok('HISTORIA');
  datos.julioDespues = await tabla(7);
  datos.septiembre = await tabla(9);
  const fila = tabla => tabla.filas.find(f => f.expediente_id === id);
  assert(fila(datos.julioDespues)); assert(fila(datos.septiembre));
  assert.deepEqual(datos.julioDespues.resumen, datos.julioAntes.resumen);
  producto(fila(datos.julioDespues).numero_menores.Conocido && fila(datos.septiembre).numero_menores.Conocido
    && fila(datos.julioDespues).numero_menores.Valor === 2 && fila(datos.septiembre).numero_menores.Valor === 1,
    'estadistica_menores_al_corte', { julio: fila(datos.julioDespues), septiembre: fila(datos.septiembre), esperado: { julio: 2, septiembre: 1 } });
  producto(fila(datos.julioDespues).fase_intervencion.Conocido && fila(datos.septiembre).fase_intervencion.Conocido
    && fila(datos.julioDespues).fase_intervencion.Valor === 'RECEPCION' && fila(datos.septiembre).fase_intervencion.Valor === 'INTERVENCION',
    'estadistica_intervencion_al_corte', { julio: fila(datos.julioDespues).fase_intervencion, septiembre: fila(datos.septiembre).fase_intervencion });
  if (!fallos.length) ok('ESTADISTICA');

  datos.ficha = await api(a, `/api/equipo/expedientes/${id}`);
  datos.avisos = await api(a, ruta(id, 'avisos'));
  datos.pestanas = {};
  for (const tab of tabs) {
    await abrir(a, id, tab);
    assert.deepEqual(await a.p.getByRole('tab').evaluateAll(ts => ts.map(t => t.id.replace('equipo-tab-', ''))), tabs);
    await a.p.locator(`#equipo-panel-${tab}`).getByText(t('fi1.cargando'), { exact: true }).waitFor({ state: 'hidden' });
    const panel = a.p.locator(`#equipo-panel-${tab}`);
    if (tab === 'familia') await panel.locator('summary').filter({ hasText: 'Alicia Torres Montes' }).waitFor();
    if (tab === 'menores') assert.equal(await panel.locator(nombre).inputValue(), 'Lucía');
    if (tab === 'intervencion') assert.equal(await panel.locator('#fi1-fase').inputValue(), 'INTERVENCION');
    const estados = { consentimiento: 'fi2.consentimiento.separado', evaluacion: 'fi2.evaluacion.vacio',
      documentos: 'equipo.sin_emisiones', cierre: 'equipo.cierre.sin_propuesta', salidas: 'equipo.spm.preparar' };
    if (estados[tab]) await panel.getByText(t(estados[tab]), { exact: true }).waitFor();
    if (tab === 'historial') await panel.getByText(t('equipo.suceso.equipo.ficha_consultada'), { exact: true }).first().waitFor();
    datos.pestanas[tab] = await a.p.locator(`#equipo-panel-${tab}`).innerText();
    assert(datos.pestanas[tab].trim(), tab);
    await a.p.screenshot({ path: resolve(dir, `${tab}.png`), fullPage: true });
  }
  ok('NUEVE-PESTANAS');
  // Se reutiliza una página con datos propios: al denegar debe eliminar la ficha anterior.
  const propias = await api(otro, '/api/equipo/expedientes');
  assert(propias.filas.length);
  await abrir(otro, propias.filas[0].id, 'familia');
  datos.denegaciones = [];
  for (const path of [ruta(id), `/api/equipo/expedientes/${id}`, ...['familia', 'menores', 'intervencion', 'avisos', 'evaluaciones', 'salida-spm'].map(s => ruta(id, s)),
    ruta(id, 'familia/historia?corte=2026-07-31&anio=2026'), ruta(id, 'menores/historia?corte=2026-07-31'), ruta(id, 'intervencion/historia?corte=2026-07-31')]) {
    const r = await pedir(otro, path);
    assert([403, 404].includes(r.estado), `${path}: ${r.estado}`);
    assert.deepEqual(Object.keys(r.datos), ['error']);
    assert.deepEqual(Object.keys(r.datos.error), ['clave']);
    datos.denegaciones.push({ path, ...r });
  }
  for (const q of ['', `?q=${encodeURIComponent(datos.ficha.numero)}`, '?q=Torres']) {
    const lista = await api(otro, '/api/equipo/expedientes' + q);
    assert(!JSON.stringify(lista).includes(id));
    assert(!JSON.stringify(lista).includes('Torres Montes'));
    assert(lista.filas.every(f => f.id !== id));
  }
  await otro.p.goto(`${base}/equipo/expediente?id=${id}`);
  await otro.p.getByRole('alert').getByText(t(datos.denegaciones[0].datos.error.clave), { exact: true }).waitFor();
  assert.equal(await otro.p.getByRole('tab').count(), 0);
  assert(!await otro.p.locator('main').innerText().then(x => x.includes('Torres Montes')));
  await otro.p.screenshot({ path: resolve(dir, 'otro-equipo.png'), fullPage: true });
  ok('AISLAMIENTO');
  assert.deepEqual(errores, []);
} catch (error) {
  datos.error = error.stack;
  console.error(error.stack);
  process.exitCode = 1;
} finally {
  await guardar('resultado.json', { pasos, fallosProducto: fallos, errores, ...datos });
  await guardar('red.json', red);
  for (const [i, s] of sesiones.entries()) await s.contexto.tracing.stop({ path: resolve(dir, `traza-${i}.zip`) });
  for (const n of navegadores) await n.close();
  if (fallos.length) process.exitCode = 1;
}
