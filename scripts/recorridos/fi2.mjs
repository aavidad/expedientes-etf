// FI2: recorrido local con datos inventados y Chrome del sistema.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
import { baseLocal, restringir } from './comun.mjs';
const base = baseLocal(process.argv[2] ?? 'http://127.0.0.1:18242');
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const carpeta = new URL('../../docs/capturas/', import.meta.url);
await mkdir(carpeta, { recursive: true });
const errores = [];
async function entrar(cuenta) {
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true, locale: 'es-ES', timezoneId: 'Europe/Madrid' });
  await restringir(contexto, base);
  const pagina = await contexto.newPage(); pagina.setDefaultTimeout(15000);
  pagina.on('pageerror', (e) => errores.push(e.message));
  await pagina.goto(base); await pagina.locator('#cuenta-ad').fill(cuenta); await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
  await pagina.locator('button[type=submit]').click(); await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return { pagina, contexto };
}
async function pedir(pagina, ruta, metodo = 'GET', datos, clave = crypto.randomUUID()) {
  return pagina.evaluate(async ({ ruta, metodo, datos, clave }) => {
    const s = await (await fetch('/api/sesion')).json();
    const r = await fetch(ruta, { method: metodo, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': s.csrf, 'Idempotency-Key': clave }, ...(datos ? { body: JSON.stringify(datos) } : {}) });
    return { estado: r.status, datos: await r.json() };
  }, { ruta, metodo, datos, clave });
}
async function ficha(pagina, id, pestana) {
  await pagina.goto(`${base}/equipo/expediente?id=${id}&pestana=${pestana}`);
  await pagina.locator('[role=tabpanel]:not([hidden]) h2').waitFor();
  assert.deepEqual(await pagina.getByRole('tab').evaluateAll((tabs) => tabs.map((tab) => tab.id.replace('equipo-tab-', ''))), ['familia', 'menores', 'intervencion', 'consentimiento', 'evaluacion', 'documentos', 'cierre', 'salidas', 'historial']);
}
async function captura(pagina, nombre) {
  for (const [ancho, sufijo] of [[1440, 'escritorio'], [390, 'movil']]) {
    await pagina.setViewportSize({ width: ancho, height: 1000 });
    await pagina.waitForTimeout(150);
    assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `desbordamiento ${ancho}`);
    await pagina.screenshot({ path: new URL(`fi2-${nombre}-${sufijo}.png`, carpeta).pathname, fullPage: true });
  }
  for (const width of [360, 1024]) { await pagina.setViewportSize({ width, height: 900 }); assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)); }
  await pagina.setViewportSize({ width: 1440, height: 1000 });
}
async function descargar(pagina, id, anexo) {
  const antes = (await pedir(pagina, `/api/equipo/expedientes/${id}`)).datos;
  const descarga = pagina.waitForEvent('download');
  const respuesta = pagina.waitForResponse((r) => r.url().endsWith(`/documentos/${anexo}`) && r.request().method() === 'POST');
  await pagina.getByRole('button', { name: 'Descargar consentimiento en Word', exact: true }).click();
  const r = await respuesta; assert.equal(r.status(), 201, r.ok() ? '' : await r.text());
  const archivo = await descarga; assert.equal(await archivo.failure(), null); assert.equal(archivo.suggestedFilename(), `${anexo}.docx`);
  const despues = (await pedir(pagina, `/api/equipo/expedientes/${id}`)).datos;
  assert.equal(despues.estado, antes.estado); assert.equal(despues.version, antes.version); assert.equal(despues.hitos.length, antes.hitos.length); assert(despues.emisiones.length > antes.emisiones.length);
}
try {
  const coordinacion = await entrar('ines.moreno');
  const p = coordinacion.pagina;
  const lista = (await pedir(p, '/api/equipo/expedientes')).datos.filas;
  const propio = lista.find((f) => f.numero === '02-02-26'); assert(propio);
  await ficha(p, propio.id, 'evaluacion');
  if (!(await pedir(p, `/api/expedientes/${propio.id}/familia`)).datos.adultos.length) await p.getByText('No hay adultos registrados.', { exact: false }).waitFor();
  const marta = await entrar('marta.cabrera');
  const f = (await pedir(marta.pagina, `/api/expedientes/${propio.id}/familia`)).datos;
  if (!f.adultos.length) {
    const alta = await pedir(marta.pagina, `/api/expedientes/${propio.id}/adultos`, 'PUT', { version: f.version, vigente_desde: f.fecha_aceptacion.slice(0, 10), adultos: [{ orden: 1, nombre: 'Alicia', apellidos: 'Molina Torres', papel: 'MADRE', sexo: 'M' }, { orden: 2, nombre: 'Rubén', apellidos: 'Vega Torres', papel: 'PADRE', sexo: 'H' }] });
    assert.equal(alta.estado, 200, JSON.stringify(alta.datos));
  }
  await ficha(p, propio.id, 'consentimiento');
  await descargar(p, propio.id, 'anexo1');
  await captura(p, 'consentimiento');
  const actual = (await pedir(p, `/api/equipo/expedientes/${propio.id}`)).datos;
  const hoy = new Date().toISOString().slice(0, 10);
  if (!actual.hitos.some((h) => h.codigo === 'consentimiento')) {
    await p.locator('#fi2-fecha-consentimiento').fill(hoy);
    await p.getByRole('tab', { name: 'Evaluación', exact: true }).click();
    await p.getByRole('button', { name: 'Seguir editando', exact: true }).click();
    assert.equal(await p.locator('#fi2-fecha-consentimiento').inputValue(), hoy);
    const r = p.waitForResponse((r) => r.url().endsWith('/hitos') && r.request().method() === 'POST');
    await p.locator('form').filter({ has: p.locator('#fi2-fecha-consentimiento') }).getByRole('button', { name: 'Confirmar fecha', exact: true }).click();
    assert.equal((await r).status(), 200);
    await p.getByText('Fecha confirmada:', { exact: false }).first().waitFor();
  }
  if (!actual.hitos.some((h) => h.codigo === 'anexo7')) {
  await p.locator('#fi2-fecha-anexo7').fill(hoy);
  const proyecto = p.waitForResponse((r) => r.url().endsWith('/hitos') && r.request().method() === 'POST');
  await p.locator('form').filter({ has: p.locator('#fi2-fecha-anexo7') }).getByRole('button', { name: 'Confirmar fecha', exact: true }).click(); assert.equal((await proyecto).status(), 200);
  }
  const despues = (await pedir(p, `/api/equipo/expedientes/${propio.id}`)).datos;
  assert.equal(despues.estado, 'en_intervencion'); assert(despues.hitos.some((h) => h.codigo === 'anexo7'));
  assert(!(await pedir(p, `/api/expedientes/${propio.id}/avisos`)).datos.avisos.some((a) => a.codigo === 'consentimiento'));
  console.log('CONSENTIMIENTO-OK: Anexo 1 real; descarga separada de firma; fecha Anexo 7; cambios sin guardar.');
  await ficha(p, propio.id, 'evaluacion'); await p.locator('#fi2-adulto option').nth(1).waitFor({ state: 'attached' });
  const anteriores = (await pedir(p, `/api/expedientes/${propio.id}/evaluaciones`)).datos;
  const personas = (await pedir(p, `/api/expedientes/${propio.id}/familia`)).datos.adultos;
  for (const [indice, momento] of [[1, 'inicial'], [2, 'intermedia'], [1, 'final']]) {
    if (anteriores.instrumentos.some((i) => i.codigo === 'SCORE15' && i.momento === momento && i.sujeto?.adulto_id === personas[indice - 1].id)) continue;
    await p.locator('#fi2-adulto').selectOption({ index: indice }); await p.locator('#fi2-momento').selectOption(momento); await p.locator('#fi2-instrumento').selectOption('SCORE15'); await p.locator('#fi2-fecha-aplicacion').fill(hoy);
    await p.getByText('Falta publicar el cuestionario oficial.', { exact: false }).waitFor();
    const r = p.waitForResponse((r) => r.url().endsWith('/evaluaciones') && r.request().method() === 'POST');
    await p.getByRole('button', { name: 'Guardar evaluación', exact: true }).click(); assert.equal((await r).status(), 200);
    await p.locator('#fi2-instrumento').waitFor();
  }
  const guardado = (await pedir(p, `/api/expedientes/${propio.id}/evaluaciones`)).datos;
  assert(guardado.instrumentos.some((i) => i.momento === 'inicial' && i.sujeto.adulto_id)); assert(guardado.instrumentos.some((i) => i.momento === 'intermedia')); assert(guardado.instrumentos.some((i) => i.momento === 'final'));
  await captura(p, 'evaluacion');
  async function preferencia(selector, valor) {
    if (!(await p.locator('.persona-conectada').evaluate((e) => e.open))) await p.locator('.persona-conectada > summary').click();
    if (!(await p.locator('details.preferencias').evaluate((e) => e.open))) await p.locator('details.preferencias > summary').click();
    await p.locator(selector).selectOption(valor);
  }
  await preferencia('#tema', 'alto-contraste');
  await preferencia('#idioma', 'en'); await p.getByRole('heading', { name: 'Evaluation by adult', exact: true }).waitFor();
  for (const width of [390, 1024]) { await p.setViewportSize({ width, height: 900 }); await p.waitForTimeout(150); assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)); }
  await p.setViewportSize({ width: 1440, height: 1000 });
  await preferencia('#idioma', 'es');
  await p.getByRole('heading', { name: 'Evaluación por adulto', exact: true }).waitFor();
  await preferencia('#tema', 'expedientes-etf');
  await ficha(marta.pagina, propio.id, 'evaluacion'); await marta.pagina.locator('#fi2-adulto').waitFor();
  const ajeno = lista.find((f) => f.numero.startsWith('04-')); assert.equal((await pedir(marta.pagina, `/api/expedientes/${ajeno.id}/evaluaciones`)).estado, 404);
  console.log('EVALUACION-OK: registros reales por adulto en tres momentos; persistencia; Marta y Coordinación; otro equipo denegado; 360/390/1024/1440; inglés y alto contraste.');
  // Preparación real de Reunificación mediante el Anexo 4 de pruebas.
  const anexo = Array.from(await readFile(new URL('../../internal/adaptadores/anexos/pdf/testdata/anexo4.pdf', import.meta.url)));
  const leido = await p.evaluate(async (bytes) => { const s = await (await fetch('/api/sesion')).json(); const body = new FormData(); body.append('anexo', new Blob([new Uint8Array(bytes)], { type: 'application/pdf' }), 'anexo4.pdf'); const r = await fetch('/api/coordinacion/derivaciones', { method: 'POST', headers: { 'X-CSRF-Token': s.csrf }, body }); return { estado: r.status, datos: await r.json() }; }, anexo);
  assert.equal(leido.estado, 200);
  const d = leido.datos; const opciones = (await pedir(p, '/api/coordinacion/decision/opciones?idioma=es')).datos;
  const municipio = opciones.municipios.find((m) => m.equipo_id === f.equipo_id && m.centro_ss_id); assert(municipio);
  const confirmada = await pedir(p, '/api/coordinacion/derivaciones/confirmar', 'POST', { lectura_id: d.lectura_anexo.id, tipo: 'anexo4', campos: d.derivacion_datos.contenido.campos, ubicacion: { municipio_id: municipio.id, centro_id: municipio.centro_ss_id, equipo_id: municipio.equipo_id } }); assert.equal(confirmada.estado, 201);
  const aceptada = await pedir(p, `/api/coordinacion/derivaciones/${confirmada.datos.id}/aceptar`, 'POST', { version: confirmada.datos.version, fecha_acta: hoy, municipio_id: municipio.id, centro_ss_id: municipio.centro_ss_id, equipo_id: municipio.equipo_id, subprograma: 'REUNIFICACION', profesional_referencia_puesto: 'TS', profesional_referencia_id: null, menores: d.derivacion_datos.contenido.campos.filter((c) => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: 'M' })), descripcion_situacion: 'La familia solicita apoyo para organizar los cuidados.' }); assert.equal(aceptada.estado, 200, JSON.stringify(aceptada.datos));
  await ficha(p, aceptada.datos.id, 'consentimiento'); await descargar(p, aceptada.datos.id, 'anexo2');
  await p.locator('#fi2-fecha-anexo8').fill(hoy); const a8 = p.waitForResponse((r) => r.url().endsWith('/hitos') && r.request().method() === 'POST');
  await p.locator('form').filter({ has: p.locator('#fi2-fecha-anexo8') }).getByRole('button', { name: 'Confirmar fecha', exact: true }).click(); assert.equal((await a8).status(), 200);
  console.log('REUNIFICACION-OK: Anexo 2 Word real; fecha Anexo 8.');
  // Fixture técnica de interfaz: no publica ni guarda un cuestionario oficial.
  await p.route('**/evaluaciones', async (ruta) => {
    if (ruta.request().method() === 'GET') {
      const respuesta = await ruta.fetch(); const datos = await respuesta.json();
      datos.definiciones.escalas = [{ id: '11111111-1111-4111-8111-111111111111', instrumento: 'SCORE15', version: 1, items: [{ codigo: 'item01', tipo: 'numerico', minimo: 0, maximo: 2, paso: 1 }, { codigo: 'item02', tipo: 'codigo', minimo: 0, maximo: 0, paso: 0, valores: ['SI', 'NO'] }] }];
      await ruta.fulfill({ response: respuesta, json: datos });
    } else {
      const datos = ruta.request().postDataJSON();
      assert.equal(datos.aplicacion.sujeto.tipo, 'figura_parental');
      assert.equal(datos.version_esperada, 0);
      assert.equal(datos.aplicacion.puntuaciones[0].valor_numerico, 0);
      assert.equal(datos.aplicacion.puntuaciones[1].valor_codigo, 'SI');
      assert(datos.clave_idempotencia);
      await ruta.fulfill({ status: 409, json: { clave: 'evaluacion.conflicto', version_actual: 1 } });
    }
  });
  await p.route('**/etiquetas/es', async (ruta) => {
    const respuesta = await ruta.fetch(); const datos = await respuesta.json();
    datos['evaluacion.item.SCORE15.item01'] = 'Respuesta numérica';
    datos['evaluacion.item.SCORE15.item02'] = 'Respuesta de opción';
    datos['evaluacion.respuesta.SCORE15.SI'] = 'Sí'; datos['evaluacion.respuesta.SCORE15.NO'] = 'No';
    await ruta.fulfill({ response: respuesta, json: datos });
  });
  await ficha(p, propio.id, 'evaluacion');
  await p.locator('#fi2-adulto').selectOption({ index: 1 }); await p.locator('#fi2-instrumento').selectOption('SCORE15'); await p.locator('#fi2-fecha-aplicacion').fill(hoy);
  await p.getByLabel('Respuesta numérica', { exact: true }).fill('0'); await p.locator('#fi2-item-1').selectOption('SI');
  await p.getByRole('button', { name: 'Guardar evaluación', exact: true }).click();
  await p.getByRole('button', { name: 'Ver datos actuales', exact: true }).waitFor();
  assert(await p.getByRole('button', { name: 'Guardar evaluación', exact: true }).isDisabled());
  await p.getByRole('button', { name: 'Ver datos actuales', exact: true }).click();
  const recuperada = p.waitForResponse((r) => r.url().endsWith('/evaluaciones') && r.request().method() === 'GET');
  await p.getByRole('button', { name: 'Descartar cambios', exact: true }).click();
  await recuperada; await p.locator('#fi2-adulto').waitFor();
  await p.unrouteAll({ behavior: 'wait' });
  await p.route('**/avisos', (ruta) => ruta.fulfill({ json: { avisos: [{ codigo: 'consentimiento', clave_texto: 'expediente.aviso_consentimiento', fecha_limite: '2026-09-01' }] } }));
  await ficha(p, aceptada.datos.id, 'consentimiento');
  await p.getByText('Pendiente de atender:', { exact: false }).waitFor();
  await p.getByRole('link', { name: 'Anotar la fecha de firma', exact: true }).click();
  await p.unrouteAll({ behavior: 'wait' });
  await p.route('**/evaluaciones', (ruta) => ruta.fulfill({ status: 403, json: { error: { clave: 'evaluacion.acceso_denegado' } } }));
  await ficha(p, propio.id, 'evaluacion');
  await p.getByRole('alert').waitFor(); assert.equal(await p.locator('#fi2-adulto').count(), 0);
  await p.unrouteAll({ behavior: 'wait' });
  console.log('FI2-FIXTURE-OK: ítems numéricos y de opción; valor cero; conflicto 409 y recuperación; aviso de plazo; sin permiso. No se publicaron cuestionarios ni baremos.');
  assert.deepEqual(errores, []);
  console.log('FI2-OK: recorrido real completado; sin errores JavaScript.');
} finally { await navegador.close(); }
