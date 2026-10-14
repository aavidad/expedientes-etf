// Recorrido TR1 con datos inventados, Chrome del sistema y servicios de la demo.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
const origen = process.argv[2] ?? 'http://127.0.0.1:18431';
assert.match(origen, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
let pagina = await contexto.newPage(); pagina.setDefaultTimeout(20000);
const errores = []; pagina.on('pageerror', e => errores.push(e.message));
async function api(ruta, datos, anexo, metodo) {
  
  return pagina.evaluate(async ({ ruta, datos, anexo, metodo }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const headers = { 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': crypto.randomUUID() };
    let body;
    if (anexo) { body = new FormData(); body.append('anexo', new Blob([new Uint8Array(anexo.contenido)], { type: anexo.tipo }), anexo.nombre); }
    else if (datos) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(datos); }
    const r = await fetch(ruta, { method: metodo ?? (datos || anexo ? 'POST' : 'GET'), headers, body });
    return { estado: r.status, datos: r.status === 204 ? null : await r.json() };
  }, { ruta, datos, anexo, metodo });
}
async function crear(nombre = 'anexo3') {
  
  const esPDF = nombre === 'anexo4';
  const extension = esPDF ? 'pdf' : 'docx';
  const carpeta = esPDF ? 'pdf' : 'docx';
  const contenido = Array.from(await readFile(new URL(`../../internal/adaptadores/anexos/${carpeta}/testdata/${nombre}.${extension}`, import.meta.url)));
  const r = await api('/api/coordinacion/derivaciones', undefined, { nombre: `${nombre}.${extension}`, tipo: esPDF ? 'application/pdf' : 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', contenido });
  assert.equal(r.estado, 200, JSON.stringify(r.datos));
  const a = r.datos;
  const opciones = a.derivacion_datos.opciones_ubicacion;
  const candidato = a.derivacion_datos.propuesta_municipio.Candidatos?.[0];
  const municipio = opciones.municipios.find(m => m.id === candidato?.MunicipioID && m.centro_id && m.equipo_id) ?? opciones.municipios.find(m => m.centro_id && m.equipo_id);
  assert(municipio);
  const confirmada = await api('/api/coordinacion/derivaciones/confirmar', {
    lectura_id: a.lectura_anexo.id, tipo: a.derivacion_datos.contenido.tipo,
    campos: a.derivacion_datos.contenido.campos.map(c => ({ ...c, dudoso: false, motivos: [] })),
    ubicacion: { municipio_id: municipio.id, centro_id: municipio.centro_id, equipo_id: municipio.equipo_id },
  }); assert.equal(confirmada.estado, 201, JSON.stringify(confirmada.datos));
  const guardada = await api(`/api/coordinacion/derivaciones/${confirmada.datos.id}`); assert.equal(guardada.estado, 200);
  return guardada.datos;
}

const sesiones = new Map();
async function entrar(cuenta) {
  if (sesiones.has(cuenta)) { pagina = sesiones.get(cuenta); return; }
  const c = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
  pagina = await c.newPage(); pagina.setDefaultTimeout(20000); pagina.on('pageerror', e => errores.push(e.message));
  await pagina.goto(origen);
  await pagina.locator('#cuenta-ad').fill(cuenta);
  await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
  await pagina.locator('button[type="submit"]').click();
  await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  sesiones.set(cuenta, pagina);
}
async function ficha(id) { const r = await api(`/api/expedientes/${id}`); assert.equal(r.estado, 200); return r.datos; }
const salidas = JSON.parse(await readFile(new URL('../../idiomas/es/textos/salidas.json', import.meta.url)));
const decision = JSON.parse(await readFile(new URL('../../idiomas/es/textos/decision.json', import.meta.url)));
const es = JSON.parse(await readFile(new URL('../../idiomas/es/textos/traslado.json', import.meta.url)));
const t = k => es[`traslado.${k}`];
const hoy = new Intl.DateTimeFormat('sv', { timeZone: 'Europe/Madrid' }).format(new Date());
const ayer = new Date(Date.parse(hoy) - 86400000).toISOString().slice(0, 10);
let opciones, equipo, municipio, destino;
async function caso(reunificacion = false) {
  await entrar('ines.moreno'); const d = await crear(reunificacion ? 'anexo4' : 'anexo3');
  opciones ??= (await api('/api/coordinacion/decision/opciones?idioma=es')).datos;
  equipo ??= opciones.equipos.find(e => /0?2\b/.test(opciones.textos[e.clave_texto])); assert(equipo);
  municipio ??= opciones.municipios.find(m => m.equipo_id === equipo.id && m.centro_ss_id); assert(municipio);
  destino ??= opciones.equipos.find(e => e.id !== equipo.id && opciones.municipios.some(m => m.equipo_id === e.id && m.centro_ss_id)); assert(destino);
  const a = await api(`/api/coordinacion/derivaciones/${d.id}/aceptar`, {
    version: d.version, fecha_acta: ayer, municipio_id: municipio.id, centro_ss_id: municipio.centro_ss_id, equipo_id: equipo.id,
    subprograma: reunificacion ? 'REUNIFICACION' : 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', profesional_referencia_id: null,
    menores: d.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: i % 2 ? 'M' : 'H' })), descripcion_situacion: 'La familia necesita acompañamiento para organizar los cuidados de sus hijos.',
  }); assert.equal(a.estado, 200, JSON.stringify(a.datos));
  await entrar('marta.cabrera');
  const h = await api(`/api/expedientes/${d.id}/hitos`, { version: a.datos.version, tipo: 'consentimiento', fecha: ayer }); assert.equal(h.estado, 200, JSON.stringify(h.datos));
  return d.id;
}
async function proponer(id, motivo) {
  const f = await ficha(id); const r = await api(`/api/expedientes/${id}/finalizacion`, { version: f.version, motivo }, undefined, 'PATCH'); assert.equal(r.estado, 200, JSON.stringify(r.datos));
  const p = await api(`/api/expedientes/${id}/proponer-cierre`, { version: r.datos.version }); assert.equal(p.estado, 200, JSON.stringify(p.datos));
}
async function accion(clave, ruta) {
  await pagina.getByRole('button', { name: t(clave), exact: true }).click();
  const dialogo = pagina.getByRole('dialog'); await dialogo.waitFor();
  const espera = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith(ruta));
  await dialogo.getByRole('button', { name: t(clave), exact: true }).click(); const r = await espera;
  assert.equal(r.status(), 200, await r.text()); return r.json();
}
async function ancho(width) { await pagina.setViewportSize({ width, height: 1000 }); await pagina.waitForTimeout(150); assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `desbordamiento ${width}`); }
async function captura(nombre) {
  if (new URL(pagina.url()).pathname === '/equipo/seguimiento') {
    const limites = await pagina.locator('#equipo-tab-seguimiento').evaluate(el => { const a = el.getBoundingClientRect(); const b = el.parentElement.getBoundingClientRect(); return { izquierda: a.left >= b.left - 1, derecha: a.right <= b.right + 1 }; }); assert.deepEqual(limites, { izquierda: true, derecha: true });
  }
  await pagina.screenshot({ path: new URL(`../../docs/capturas/traslado-${nombre}.png`, import.meta.url).pathname, fullPage: new URL(pagina.url()).pathname !== '/equipo/seguimiento' }); }
async function seguimiento(id) { await pagina.goto(`${origen}/equipo/seguimiento?id=${id}`); await pagina.getByRole('heading', { name: t('seguimiento'), exact: true }).waitFor(); await pagina.getByText(t('cargando'), { exact: true }).waitFor({ state: 'detached' }); }
try {
  const id = await caso(); await proponer(id, 'TRASLADO'); await entrar('javier.medina'); await pagina.goto(`${origen}/coordinacion/expedientes/${id}/traslado`); await pagina.getByRole('heading', { name: t('preparar'), exact: true }).waitFor(); assert(!await pagina.getByRole('button', { name: t('trasladar'), exact: true }).isDisabled()); await entrar('ines.moreno');
  await pagina.goto(`${origen}/coordinacion/expedientes/${id}/salidas`);
  await pagina.getByRole('link', { name: t('titulo'), exact: true }).click();
  await pagina.getByRole('heading', { name: t('preparar'), exact: true }).waitFor();
  await pagina.getByRole('button', { name: t('trasladar'), exact: true }).click();
  await pagina.getByRole('heading', { name: t('faltan_datos') }).waitFor();
  await pagina.getByRole('link', { name: t('destino'), exact: true }).click(); assert.equal(await pagina.evaluate(() => document.activeElement.id), 'traslado-destino');
  await pagina.locator('#traslado-destino').selectOption(destino.id);
  await pagina.locator('#traslado-motivo_traslado').fill('La familia se ha trasladado a otro municipio y necesita continuar la intervención.');
  await pagina.locator('#traslado-fecha_traslado').fill(hoy);
  await pagina.getByRole('button', { name: t('ayuda_titulo'), exact: true }).click(); await pagina.keyboard.press('Escape'); assert.equal(await pagina.evaluate(() => document.activeElement.textContent), t('ayuda'));
  await captura('escritorio'); for (const w of [1024, 390, 360]) await ancho(w); await ancho(390); await captura('movil'); await ancho(1440);
  // La primera petición se aplica; el navegador pierde su respuesta.
  let primera, reciboReal;
  const ruta = `/api/expedientes/${id}/trasladar`;
  await pagina.route(`**${ruta}`, async route => { primera = route.request(); const r = await route.fetch(); assert.equal(r.status(), 200, await r.text()); reciboReal = await r.json(); await route.abort('failed'); await pagina.unroute(`**${ruta}`); });
  await pagina.getByRole('button', { name: t('trasladar'), exact: true }).click(); await pagina.getByRole('dialog').getByRole('button', { name: t('trasladar'), exact: true }).click();
  await pagina.getByRole('button', { name: t('recuperar'), exact: true }).waitFor();
  const recuperada = pagina.waitForRequest(r => r.method() === 'POST' && r.url().endsWith(ruta));
  await pagina.getByRole('button', { name: t('recuperar'), exact: true }).click(); const repetida = await recuperada;
  assert.equal(repetida.postData(), primera.postData()); assert.equal(repetida.headers()['idempotency-key'], primera.headers()['idempotency-key']);
  await pagina.getByRole('heading', { name: t('hecho') }).waitFor(); const original = await ficha(id); assert.equal(original.estado, 'cerrada'); assert.equal(original.version, reciboReal.cierre.version); assert.equal(original.traslados.length, 1);
  const nuevo = await ficha(reciboReal.destino_id); assert.equal(nuevo.estado, 'derivacion_recibida'); assert.equal(nuevo.numero, null); assert.equal(nuevo.traslado.origen_id, id);
  console.log('TR1 traslado: cierre, alta enlazada y recuperación con misma petición/clave OK');
  await pagina.getByRole('link', { name: t('acta_origen'), exact: true }).click();
  const actaCierre = pagina.waitForEvent('download'); await pagina.getByRole('button', { name: salidas['salidas.accion.acta'], exact: true }).click(); assert.equal(await (await actaCierre).failure(), null); assert.equal((await ficha(id)).version, original.version);
  await pagina.goto(`${origen}/coordinacion/expedientes/${id}/traslado`); await pagina.getByRole('link', { name: t('ver_destino'), exact: true }).click();
  await pagina.locator('#decision-fecha_acta').fill(hoy); const md = opciones.municipios.find(m => m.equipo_id === destino.id && m.centro_ss_id);
  await pagina.locator('#decision-municipio_id').selectOption(md.id); await pagina.locator('#decision-profesional_referencia_puesto').selectOption('TS');
  await pagina.locator('input[type="radio"][value="H"]').evaluateAll(els => els.forEach(el => el.click()));
  await pagina.getByRole('button', { name: decision['decision.aceptar'], exact: true }).click();
  const respuesta = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/aceptar')); await pagina.getByRole('dialog').getByRole('button', { name: decision['decision.confirmar.aceptar'], exact: true }).click(); assert.equal((await respuesta).status(), 200);
  await pagina.getByRole('button', { name: `${decision['decision.descargar']} ${decision['decision.acta_aceptacion']}`, exact: true }).waitFor(); const descarga = pagina.waitForEvent('download'); await pagina.getByRole('button', { name: `${decision['decision.descargar']} ${decision['decision.acta_aceptacion']}`, exact: true }).click(); assert.equal(await (await descarga).failure(), null);
  const aceptado = await ficha(nuevo.id); assert.equal(aceptado.equipo_id, destino.id); assert.notEqual(aceptado.numero, original.numero); assert.equal(aceptado.estado, 'aceptada'); console.log('TR1 destino: número nuevo de destino y Acta separada OK');
  const reunificado = await caso(true); const fEquipo = (await api(`/api/equipo/expedientes/${reunificado}`)).datos;
  const motivoCierre = fEquipo.motivos_cierre.find(m => m.codigo !== 'TRASLADO' && !m.sin_consentimiento); assert(motivoCierre);
  await proponer(reunificado, motivoCierre.codigo); await entrar('ines.moreno'); const f = await ficha(reunificado); const cerrado = await api(`/api/expedientes/${reunificado}/cerrar`, { version: f.version, fecha: hoy }); assert.equal(cerrado.estado, 200, JSON.stringify(cerrado.datos));
  await entrar('marta.cabrera'); await seguimiento(reunificado); await pagina.locator('#traslado-fecha_cese').fill(ayer); const inicio = await accion('accion.iniciar', `/api/expedientes/${reunificado}/seguimiento`); assert.equal(inicio.estado, 'activo'); assert.equal(inicio.fecha_fin_prevista, `${Number(ayer.slice(0, 4)) + 1}${ayer.slice(4)}`);
  await pagina.getByText(t('seguimiento_activo'), { exact: true }).waitFor(); await pagina.getByText(t('fin_previsto'), { exact: true }).waitFor();
  await ancho(1440); await captura('seguimiento-escritorio'); await ancho(390); await captura('seguimiento-movil'); await ancho(360); await ancho(1024);
  await pagina.locator('#traslado-fecha_terminacion').fill(hoy); await pagina.locator('#traslado-motivo_fin').selectOption('CAMBIO_SITUACION'); const fin = await accion('accion.terminar', `/api/expedientes/${reunificado}/seguimiento/terminar`); assert.equal(fin.estado, 'terminado');
  await pagina.getByText(t('reabre_coordinacion'), { exact: true }).waitFor(); assert.equal(await pagina.getByRole('button', { name: t('accion.reabrir'), exact: true }).count(), 0);
  assert.equal((await api(`/api/expedientes/${reunificado}/seguimiento/reabrir`, { version: fin.version, motivo: 'La familia necesita retomar el acompañamiento.' })).estado, 403);
  await entrar('ines.moreno'); await seguimiento(reunificado); await pagina.locator('#traslado-motivo_reapertura').fill('La familia solicita retomar el acompañamiento.'); const abierto = await accion('accion.reabrir', `/api/expedientes/${reunificado}/seguimiento/reabrir`); assert.equal(abierto.tramo_desde, hoy); assert.equal(abierto.fecha_fin_prevista, `${Number(hoy.slice(0, 4)) + 1}${hoy.slice(4)}`); assert.equal(abierto.fecha_cese_medida, ayer); assert.equal(abierto.version, fin.version + 1);
  await pagina.getByText(t('seguimiento_activo'), { exact: true }).waitFor(); console.log('TR1 seguimiento: plazo, fin anticipado por equipo, denegación y reapertura por Coordinación OK');
  // Conflicto real: Javier termina mientras Inés conserva la versión anterior.
  await entrar('javier.medina'); const concurrente = await api(`/api/expedientes/${reunificado}/seguimiento/terminar`, { version: abierto.version, fecha: hoy, motivo: 'CAMBIO_SITUACION' }); assert.equal(concurrente.estado, 200);
  await entrar('ines.moreno'); await pagina.locator('#traslado-fecha_terminacion').fill(hoy); await pagina.locator('#traslado-motivo_fin').selectOption('CAMBIO_SITUACION');
  await pagina.getByRole('button', { name: t('accion.terminar'), exact: true }).click(); const conflicto = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/seguimiento/terminar')); await pagina.getByRole('dialog').getByRole('button', { name: t('accion.terminar'), exact: true }).click(); assert.equal((await conflicto).status(), 409);
  await pagina.getByRole('button', { name: t('ver_cambios'), exact: true }).click(); await pagina.getByRole('button', { name: 'Descartar cambios', exact: true }).click(); await pagina.getByText(t('seguimiento_terminado'), { exact: true }).waitFor();
  await pagina.locator('#traslado-motivo_reapertura').fill('Coordinación retoma el seguimiento después de revisar el cambio del equipo.'); await accion('accion.reabrir', `/api/expedientes/${reunificado}/seguimiento/reabrir`);
  // El aviso no es requisito para leer ni cambiar un seguimiento.
  await pagina.route('**/api/avisos', route => route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: { clave: 'expediente.no_disponible' } }) })); await seguimiento(reunificado); await pagina.getByText(t('avisos_no_disponibles'), { exact: false }).waitFor(); await pagina.getByRole('button', { name: t('accion.terminar'), exact: true }).waitFor(); await pagina.unroute('**/api/avisos');
  console.log('TR1 estados: conflicto real 409, revisión con guardia y aviso fallido sin bloqueo OK');
  async function preferencias() {
    if (!await pagina.locator('#tema').isVisible()) { const persona = pagina.locator('.persona-conectada > summary'); if ((await pagina.locator('.persona-conectada').getAttribute('open')) === null) await persona.click(); const preferencias = pagina.locator('.preferencias'); if ((await preferencias.getAttribute('open')) === null) await preferencias.locator('summary').click(); }
  }
  await preferencias(); await pagina.locator('#tema').selectOption('alto-contraste'); await pagina.keyboard.press('Escape'); await pagina.keyboard.press('Escape');
  await ancho(390); await captura('alto-contraste-movil'); await preferencias(); await pagina.locator('#idioma').selectOption('en'); await pagina.keyboard.press('Escape'); await pagina.keyboard.press('Escape'); await ancho(360); await ancho(1024);
  assert.deepEqual(errores, []); console.log('TR1 visual: escritorio, 1024/390/360, teclado, alto contraste e inglés OK');
} catch (e) { await pagina.screenshot({ path: `${process.env.TMPDIR ?? 'bin/temporales'}/tr1-fallo.png`, fullPage: true }); throw e; } finally { await navegador.close(); }
