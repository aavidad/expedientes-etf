// RESP3: casos reales sin escalas y con resultados parciales; sólo demo local.
import assert from 'node:assert/strict';
import { readFile, writeFile, mkdir, mkdtemp, rm } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { chromium, baseLocal, restringir } from './comun.mjs';

const base = baseLocal(process.argv[2]);
const raiz = new URL('../../', import.meta.url);
const claves = JSON.parse(await readFile(new URL('despliegue/demo/.local/usuarios.json', raiz)));
const temporal = await mkdtemp('/dev/shm/etf/tmp/resp3-recorrido-');
const original = await readFile(new URL('idiomas/es/catalogos/evaluacion-v1.reglas', raiz));
// Definición exclusiva del ensayo. Copia un resultado anotado, sin sumar ítems
// ni introducir un cuestionario o baremo en los catálogos de distribución.
const versionEscala = '12345678-1234-4234-8234-123456789abc';
const fixture = JSON.parse(original);
fixture.escalas.push({ id: versionEscala, instrumento: 'SCORE15', version: 1,
  items: [{ codigo: 'fortalezas', tipo: 'numerico', minimo: 0, maximo: 100, paso: 0.5, campo_documento: 'escala_score15_1' }] });
const archivo = temporal + '/evaluacion.reglas';
await writeFile(archivo, JSON.stringify(fixture));
function copiar(origen) {
  execFileSync('python3', ['-c', `import importlib.util,sys
s=importlib.util.spec_from_file_location('demo','despliegue/demo/demo.py');d=importlib.util.module_from_spec(s);s.loader.exec_module(d)
d.compose('cp',sys.argv[1],'app:/datos/idiomas/es/catalogos/evaluacion-v1.reglas',capture_output=True,text=True)`, origen], { cwd: raiz, stdio: 'pipe' });
}
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];
async function entrar(cuenta) {
  const contexto = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'es-ES', timezoneId: 'Europe/Madrid', acceptDownloads: true });
  await restringir(contexto, base);
  const p = await contexto.newPage(); p.setDefaultTimeout(15000);
  p.on('pageerror', e => errores.push(e.message));
  // La preparación usa escrituras API ya confirmadas; al cambiar de ficha se
  // descarta sólo el estado de formulario de esa demo, no datos pendientes.
  p.on('dialog', async dialogo => {
    if (dialogo.type() === 'beforeunload') await dialogo.accept();
    else { errores.push('dialogo_inesperado'); await dialogo.dismiss(); }
  });
  await p.goto(base, { waitUntil: 'domcontentloaded', timeout: 30000 }); await p.locator('#cuenta-ad').fill(cuenta); await p.locator('#contrasena-ad').fill(claves[cuenta]);
  await p.locator('button[type=submit]').click(); await p.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return p;
}
async function pedir(p, ruta, method = 'GET', datos, clave = randomUUID()) {
  return p.evaluate(async ({ ruta, method, datos, clave }) => {
    const s = await (await fetch('/api/sesion')).json();
    const r = await fetch(ruta, { method, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': s.csrf, 'Idempotency-Key': clave }, ...(datos ? { body: JSON.stringify(datos) } : {}) });
    return { estado: r.status, datos: await r.json() };
  }, { ruta, method, datos, clave });
}
const hoy = new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Madrid' }).format(new Date());
async function crearCaso(q, tipo, equipo) {
  const bytes = Array.from(await readFile(new URL(`internal/adaptadores/anexos/pdf/testdata/${tipo}.pdf`, raiz)));
  const leido = await q.evaluate(async ({ bytes, tipo }) => {
    const s = await (await fetch('/api/sesion')).json(); const body = new FormData();
    body.append('anexo', new Blob([new Uint8Array(bytes)], { type: 'application/pdf' }), tipo + '.pdf');
    const r = await fetch('/api/coordinacion/derivaciones', { method: 'POST', headers: { 'X-CSRF-Token': s.csrf }, body });
    return { estado: r.status, datos: await r.json() };
  }, { bytes, tipo });
  assert.equal(leido.estado, 200);
  assert(!leido.datos.derivacion_datos.contenido.avisos?.includes('expediente.valorame_pendiente'));
  const d = leido.datos;
  const respuestaOpciones = await pedir(q, '/api/coordinacion/decision/opciones?idioma=es'); assert.equal(respuestaOpciones.estado, 200, JSON.stringify(respuestaOpciones.datos));
  const opciones = respuestaOpciones.datos;
  const municipio = opciones.municipios.find(m => m.equipo_id === equipo && m.centro_ss_id); assert(municipio);
  const confirmada = await pedir(q, '/api/coordinacion/derivaciones/confirmar', 'POST', { lectura_id: d.lectura_anexo.id, tipo,
    campos: d.derivacion_datos.contenido.campos, ubicacion: { municipio_id: municipio.id, centro_id: municipio.centro_ss_id, equipo_id: equipo } });
  assert.equal(confirmada.estado, 201);
  const aceptada = await pedir(q, `/api/coordinacion/derivaciones/${confirmada.datos.id}/aceptar`, 'POST', {
    version: confirmada.datos.version, fecha_acta: hoy, municipio_id: municipio.id, centro_ss_id: municipio.centro_ss_id, equipo_id: equipo,
    subprograma: tipo === 'anexo4' ? 'REUNIFICACION' : 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', profesional_referencia_id: null,
    menores: d.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: 'M' })),
    descripcion_situacion: 'La familia solicita apoyo para organizar los cuidados y la asistencia escolar.' });
  assert.equal(aceptada.estado, 200, JSON.stringify(aceptada.datos));
  return aceptada.datos.id;
}
async function ficha(p, id, pestana = 'evaluacion', recargar = false) {
  if (!recargar && new URL(p.url()).searchParams.get('id') === id) {
    await p.getByRole('tab', { name: pestana === 'cierre' ? 'Cierre' : 'Evaluación', exact: true }).click();
  } else await p.goto(`${base}/equipo/expediente?id=${id}&pestana=${pestana}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await p.getByRole('tab', { name: 'Evaluación', exact: true }).waitFor();
}
async function capturas(q, nombre) {
  await mkdir(new URL('docs/capturas/', raiz), { recursive: true });
  for (const width of [1440, 390, 360, 1024]) {
    await q.setViewportSize({ width, height: 1000 }); await q.evaluate(() => document.fonts.ready); await q.waitForTimeout(150);
    assert(await q.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `desbordamiento ${width}`);
    if (width === 1440 || width === 390) await q.screenshot({ path: new URL(`docs/capturas/resp3-${nombre}-${width === 1440 ? 'escritorio' : 'movil'}.png`, raiz).pathname, fullPage: true });
  }
  await q.setViewportSize({ width: 1440, height: 1000 });
}
function evaluarWord(bytes, esperado = '') {
  const texto = execFileSync('python3', ['-c', `import sys,io,json
from zipfile import ZipFile
from xml.etree import ElementTree as E
w='{http://schemas.openxmlformats.org/wordprocessingml/2006/main}'
with ZipFile(io.BytesIO(sys.stdin.buffer.read())) as z:
 assert b'documentProtection' not in z.read('word/settings.xml')
 r=E.fromstring(z.read('word/document.xml'))
tablas=[]
for t in r.iter(w+'tbl'):
 rows=[[''.join(x.text or '' for x in c.iter(w+'t')) for c in f.findall(w+'tc')] for f in t.findall(w+'tr')]
 if any(any(m in c for m in ['VALÓRAME','ECP 12','AAPI-2.5','SCORE 15','SDQ','SOATIF','Satisfacción con la intervención','Satisfaction with the intervention']) for row in rows for c in row):tablas.append(rows)
print(json.dumps(tablas,ensure_ascii=False))`], { input: bytes, encoding: 'utf8' });
  const tablas = JSON.parse(texto); assert.equal(tablas.length, 8);
  const celdas = tablas.flat().filter(r => r.length === 4).slice();
  const resultados = celdas.flatMap(r => r.slice(1)).filter(c => !/EVALUACIÓN|ASSESSMENT/.test(c));
  assert(!resultados.some(c => /Sin dato|No data|Puntuación\.|Score\.|Elija|Select an option|\{\{/.test(c)));
  if (esperado) { const inicial = resultados.find(c => c.split('; ').length === 2); assert(inicial); assert.deepEqual(inicial.split('; ').sort(), esperado.split('; ').sort()); assert(resultados.some(c => c === 'L. O. S.: 0')); }
  else assert(resultados.every(c => c === ''));
  return resultados;
}
async function descargar(p, id, anexo, esperado = '') {
  await ficha(p, id, 'cierre');
  const antes = (await pedir(p, `/api/equipo/expedientes/${id}`)).datos;
  const evento = p.waitForEvent('download');
  await p.getByRole('button', { name: `Descargar Anexo ${anexo}`, exact: true }).click();
  const descarga = await evento; const bytes = await readFile(await descarga.path()); evaluarWord(bytes, esperado);
  const despues = (await pedir(p, `/api/equipo/expedientes/${id}`)).datos;
  assert.equal(despues.estado, antes.estado); assert.deepEqual(despues.hitos, antes.hitos);
  assert(despues.emisiones.some(e => e.codigo === `finalizacion/anexo${anexo}`));
}

try {
  const p = await entrar('marta.cabrera'); const q = await entrar('ines.moreno');
  const equipo = (await pedir(p, '/api/sesion')).datos.equipo_id; assert(equipo);
  const casos = [];
  for (const tipo of ['anexo3', 'anexo4']) {
    const id = await crearCaso(q, tipo, equipo); const anexo = tipo === 'anexo3' ? 11 : 12;
    assert.deepEqual((await pedir(q, `/api/expedientes/${id}/evaluaciones`)).datos.escalas, []);
    await ficha(q, id); await q.getByText('Las escalas son opcionales.', { exact: false }).waitFor();
    await q.getByText('Todavía no hay instrumentos ni escalas registrados.', { exact: true }).waitFor();
    await capturas(q, 'sin-escalas');
    const version = (await pedir(p, `/api/expedientes/${id}/familia`)).datos.version;
    const final = await pedir(p, `/api/expedientes/${id}/finalizacion`, 'PATCH', { version, motivo: 'SIN_CONSENTIMIENTO', sin_consentimiento: true });
    assert.equal(final.estado, 200, JSON.stringify(final.datos));
    await descargar(p, id, anexo);
    assert(!(await pedir(q, `/api/expedientes/${id}/avisos`)).datos.avisos.some(a => /escala|valorame|evaluacion/i.test(JSON.stringify(a))));
    casos.push({ id, anexo });
  }
  console.log('RESP3-VACIO-OK: Anexos 11 y 12 reales, casillas vacías; sin avisos de escalas; emisión conserva estado e hitos.');
  const sinEscalas = await crearCaso(q, 'anexo3', equipo);
  const sinRuta = `/api/expedientes/${sinEscalas}`;
  const sinVersion = (await pedir(p, sinRuta + '/familia')).datos.version;
  const sinFinal = await pedir(p, sinRuta + '/finalizacion', 'PATCH', { version: sinVersion, motivo: 'SIN_CONSENTIMIENTO', sin_consentimiento: true });
  assert.equal(sinFinal.estado, 200);
  assert.equal((await pedir(p, sinRuta + '/proponer-cierre', 'POST', { version: sinFinal.datos.version })).estado, 200);
  const sinActual = (await pedir(q, sinRuta + '/finalizacion')).datos;
  const sinCierre = await pedir(q, sinRuta + '/cerrar', 'POST', { version: sinActual.version, fecha: hoy });
  assert.equal(sinCierre.estado, 200, JSON.stringify(sinCierre.datos)); assert.equal(sinCierre.datos.estado, 'cerrada');
  assert.equal(sinCierre.datos.aviso, 'expediente.cierre_sin_anexo');
  console.log('RESP3-CIERRE-VACIO-OK: cierre real sin escalas ni Anexo emitido; sólo aviso permitido del Anexo.');
  const memoriaAntes = (await pedir(q, `/api/estadistica/memoria?anio=2026&etf=${equipo}`)).datos;
  copiar(archivo);
  for (const caso of casos) {
    const ruta = `/api/expedientes/${caso.id}`;
    const familia = (await pedir(p, ruta + '/familia')).datos;
    let adultos = [
      { id: '', orden: 1, nombre: 'Lucía', apellidos: 'Ortega Salas', papel: 'MADRE', sexo: 'M' },
      { id: '', orden: 2, nombre: 'Daniel', apellidos: 'Pérez Molina', papel: 'PADRE', sexo: 'H' }];
    const guardada = await pedir(p, ruta + '/adultos', 'PUT', { version: familia.version, vigente_desde: hoy, adultos });
    assert.equal(guardada.estado, 200, JSON.stringify(guardada.datos));
    adultos = (await pedir(p, ruta + '/familia')).datos.adultos; assert.equal(adultos.length, 2);
    await ficha(p, caso.id, 'evaluacion', true);
    for (const [sujeto, momento, valor] of [[adultos[0], 'inicial', 7.5], [adultos[1], 'inicial', 4], [adultos[0], 'final', 0]]) {
      await p.locator('#fi2-adulto').selectOption(sujeto.id);
      await p.locator('#fi2-momento').selectOption(momento); await p.locator('#fi2-instrumento').selectOption('SCORE15');
      await p.locator('#fi2-fecha-aplicacion').fill(hoy); await p.locator('#fi2-item-0').fill(String(valor));
      const respuesta = p.waitForResponse(r => r.url().endsWith('/evaluaciones') && r.request().method() === 'POST');
      await p.getByRole('button', { name: 'Guardar evaluación', exact: true }).click();
      assert.equal((await respuesta).status(), 200);
      await p.getByText('Se han guardado los datos.', { exact: true }).waitFor();
      await p.waitForFunction(() => document.getElementById('fi2-instrumento')?.value === '');
    }
    const registro = (await pedir(q, ruta + '/evaluaciones')).datos;
    assert.equal(registro.escalas.length, 3); assert.equal(registro.instrumentos.length, 0);
    await ficha(q, caso.id); await q.getByText('Evaluaciones registradas', { exact: true }).waitFor();
    await capturas(q, 'parciales');
    await descargar(p, caso.id, caso.anexo, 'L. O. S.: 7,5; D. P. M.: 4');
    const final = (await pedir(p, ruta + '/finalizacion')).datos;
    const propuesta = await pedir(p, ruta + '/proponer-cierre', 'POST', { version: final.version });
    assert.equal(propuesta.estado, 200, JSON.stringify(propuesta.datos));
    const actual = (await pedir(q, ruta + '/finalizacion')).datos;
    const cierre = await pedir(q, ruta + '/cerrar', 'POST', { version: actual.version, fecha: hoy });
    assert.equal(cierre.estado, 200, JSON.stringify(cierre.datos)); assert.equal(cierre.datos.estado, 'cerrada');
    assert(!/escala|valorame|evaluacion/i.test(cierre.datos.aviso ?? ''));
  }
  console.log('RESP3-PARCIAL-OK: guardado por pantalla y lectura real; inicial/final y cero; varios adultos con iniciales; Anexos 11/12; cierre confirmado.');
  const memoria = await pedir(q, `/api/estadistica/memoria?anio=2026&etf=${equipo}`); assert.equal(memoria.estado, 200);
  assert(!/escala|valorame_pendiente/i.test(JSON.stringify(memoria.datos.avisos ?? {})));
  assert(memoriaAntes);
  const estado = (await pedir(q, `/api/estadistica/memoria/estado?anio=2026&etf=${equipo}`)).datos;
  if (estado.estado === 'cerrada') {
    assert.equal((await pedir(q, `/api/estadistica/memoria/reabrir?anio=2026&etf=${equipo}`, 'POST', { version: estado.version, motivo: 'Revisión del cierre anual.' })).estado, 200);
  }
  const actual = (await pedir(q, `/api/estadistica/memoria/estado?anio=2026&etf=${equipo}`)).datos;
  const cerrada = await pedir(q, `/api/estadistica/memoria/cerrar?anio=2026&etf=${equipo}`, 'POST', { version: actual.version });
  assert.equal(cerrada.estado, 200, JSON.stringify(cerrada.datos));
  console.log('RESP3-MEMORIA-OK: consulta y cierre anual reales con escalas incompletas; sin avisos de escalas.');
  await q.getByRole('tab', { name: 'Evaluación', exact: true }).focus();
  const siguiente = await q.getByRole('tab', { name: 'Evaluación', exact: true }).evaluate(e => e.nextElementSibling.id);
  await q.keyboard.press('ArrowRight');
  await q.locator('#' + siguiente + '[aria-selected=true]').waitFor();
  await q.keyboard.press('ArrowLeft');
  await q.locator('#equipo-tab-evaluacion[aria-selected=true]').waitFor();
  async function preferencia(selector, valor) {
    if (!(await q.locator('.persona-conectada').evaluate(e => e.open))) await q.locator('.persona-conectada > summary').click();
    if (!(await q.locator('details.preferencias').evaluate(e => e.open))) await q.locator('details.preferencias > summary').click();
    await q.locator(selector).selectOption(valor);
  }
  await preferencia('#tema', 'alto-contraste'); await preferencia('#idioma', 'en');
  await q.getByText('Assessment scales are optional.', { exact: false }).waitFor();
  for (const width of [390, 1024]) {
    await q.setViewportSize({ width, height: 1000 }); await q.waitForTimeout(150);
    assert(await q.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
  }
  await q.setViewportSize({ width: 1440, height: 1000 });
  await preferencia('#idioma', 'es'); await q.getByRole('heading', { name: 'Evaluación por adulto', exact: true }).waitFor();
  await preferencia('#tema', 'expedientes-etf');
  assert.deepEqual(errores, []);
  console.log('RESP3-NAVEGADOR-OK: Inés y Marta; Chrome del sistema; 1440/390/360/1024 sin desbordamiento; teclado, inglés y alto contraste.');
} finally {
  copiar(new URL('idiomas/es/catalogos/evaluacion-v1.reglas', raiz).pathname);
  await browser.close(); await rm(temporal, { recursive: true, force: true });
}
