// Recorrido CO2 con datos inventados, Chrome del sistema y servicios de la demo.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
const origen = process.argv[2] ?? 'http://127.0.0.1:18242';
assert.match(origen, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
let pagina = await contexto.newPage(); pagina.setDefaultTimeout(20000);
const errores = []; pagina.on('pageerror', e => errores.push(e.message));
async function api(ruta, datos, anexo, metodo) {
  console.log('PASO: Petición API');
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
  console.log('PASO: Crear derivación');
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
  console.log('ANEXO:', nombre, 'menores', guardada.datos.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').length); return guardada.datos;
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
async function preparar() {
  await entrar('ines.moreno');
  const d = await crear();
  const o = (await api('/api/coordinacion/decision/opciones?idioma=es')).datos;
  const equipo = o.equipos.find(e => o.textos[e.clave_texto]?.includes('2') || o.textos[e.clave_texto]?.includes('02'));
  assert(equipo, JSON.stringify(o.equipos));
  const m = o.municipios.find(m => m.equipo_id === equipo.id && m.centro_ss_id);
  assert(m);
  const aceptada = await api(`/api/coordinacion/derivaciones/${d.id}/aceptar`, {
    version: d.version, fecha_acta: '2026-10-13', municipio_id: m.id, centro_ss_id: m.centro_ss_id, equipo_id: equipo.id,
    subprograma: 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', profesional_referencia_id: null,
    menores: d.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: 'M' })),
    descripcion_situacion: 'La familia solicita apoyo para organizar los cuidados y mejorar la asistencia escolar.',
  });
  assert.equal(aceptada.estado, 200, JSON.stringify(aceptada.datos));
  await entrar('marta.cabrera');
  const f = await ficha(d.id);
  const final = await api(`/api/expedientes/${d.id}/finalizacion`, { version: f.version, motivo: 'SIN_CONSENTIMIENTO', sin_consentimiento: true }, undefined, 'PATCH');
  assert.equal(final.estado, 200, JSON.stringify(final.datos));
  const p = await api(`/api/expedientes/${d.id}/proponer-cierre`, { version: final.datos.version });
  assert.equal(p.estado, 200, JSON.stringify(p.datos));
  await entrar('ines.moreno');
  const actual = await ficha(d.id);
  const salida = await api(`/api/expedientes/${d.id}/salida-spm`, { version: actual.version, motivo: 'Se necesita valorar la protección de los menores.', propuesta_medida: 'Valoración de una medida de protección.' });
  assert.equal(salida.estado, 200, JSON.stringify(salida.datos));
  const fecha = await api(`/api/expedientes/${d.id}/salida-spm/fecha`, { version: salida.datos.version, fecha: '2026-10-13' });
  assert.equal(fecha.estado, 200, JSON.stringify(fecha.datos));
  return d.id;
}
async function abrir(id) {
  await pagina.goto(`${origen}/coordinacion/expedientes/${id}/salidas`);
  await pagina.getByRole('heading', { name: 'Respuesta del SPM', exact: true }).waitFor();
}
async function accion(nombre, final) {
  await pagina.getByRole('button', { name: nombre, exact: true }).click();
  const dialogo = pagina.getByRole('dialog'); await dialogo.waitFor();
  const espera = pagina.waitForResponse(r => r.request().method() !== 'GET' && r.url().endsWith(final));
  await dialogo.getByRole('button', { name: nombre, exact: true }).click();
  const r = await espera; assert.equal(r.status(), 200, await r.text());
  await pagina.getByRole('dialog').waitFor({ state: 'detached' });
  return r;
}
async function ancho(width) {
  await pagina.setViewportSize({ width, height: 1000 }); await pagina.waitForTimeout(150);
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `desbordamiento ${width}`);
}
async function descargar(id) {
  const antes = await ficha(id);
  const descarga = pagina.waitForEvent('download');
  const espera = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/documentos/acta_cierre'));
  await pagina.getByRole('button', { name: 'Descargar Acta de Cierre', exact: true }).click();
  const r = await espera; assert.equal(r.status(), 201, await r.text());
  assert.equal(await (await descarga).failure(), null);
  const despues = await ficha(id); assert.equal(despues.version, antes.version); assert.equal(despues.estado, antes.estado);
}
try {
  await pagina.goto(origen);
  const id = await preparar(); await abrir(id);
  assert(await pagina.getByRole('button', { name: 'Confirmar fecha y cerrar expediente', exact: true }).isDisabled());
  await mkdir(new URL('../../docs/capturas/', import.meta.url), { recursive: true });
  await pagina.screenshot({ path: new URL('../../docs/capturas/salidas-escritorio.png', import.meta.url).pathname, fullPage: true });
  for (const width of [1024, 390, 360]) await ancho(width);
  await ancho(390); await pagina.screenshot({ path: new URL('../../docs/capturas/salidas-movil.png', import.meta.url).pathname, fullPage: true });
  await ancho(1440);
  await pagina.getByRole('button', { name: 'Ayuda', exact: true }).click();
  await pagina.getByRole('dialog').waitFor(); await pagina.keyboard.press('Escape');
  assert.equal(await pagina.getByRole('dialog').count(), 0);
  assert.equal(await pagina.evaluate(() => document.activeElement?.textContent), 'Ayuda');
  await pagina.getByRole('button', { name: 'Guardar respuesta del SPM', exact: true }).click();
  await pagina.getByRole('heading', { name: 'Revisa estos campos antes de continuar', exact: true }).waitFor();
  await pagina.getByRole('link', { name: 'Fecha de la respuesta', exact: true }).click();
  assert.equal(await pagina.evaluate(() => document.activeElement?.id), 'salidas-fecha_respuesta');
  await pagina.locator('#salidas-fecha_cierre').fill('2026-10-13');
  await pagina.locator('#salidas-motivo_acta_nuevo').fill('La familia no consiente la intervención.');
  await pagina.locator('#salidas-respuesta').selectOption('medida_no_necesaria');
  await pagina.locator('#salidas-fecha_respuesta').fill('2026-10-13');
  const peticionesRespuesta = [];
  let interrumpir = true;
  await pagina.route('**/salida-spm/respuesta', async ruta => {
    peticionesRespuesta.push({ clave: ruta.request().headers()['idempotency-key'], cuerpo: ruta.request().postData() });
    if (interrumpir) { interrumpir = false; const recibo = await ruta.fetch(); assert.equal(recibo.status(), 200); await ruta.abort(); }
    else await ruta.continue();
  });
  await pagina.getByRole('button', { name: 'Guardar respuesta del SPM', exact: true }).click();
  await pagina.getByRole('dialog').getByRole('button', { name: 'Guardar respuesta del SPM', exact: true }).click();
  await pagina.getByRole('button', { name: 'Reintentar la misma operación', exact: true }).waitFor();
  const repetida = pagina.waitForResponse(r => r.request().method() === 'PATCH' && r.url().endsWith('/salida-spm/respuesta'));
  await pagina.getByRole('button', { name: 'Reintentar la misma operación', exact: true }).click();
  assert.equal((await repetida).status(), 200);
  assert.equal(peticionesRespuesta.length, 2); assert.deepEqual(peticionesRespuesta[0], peticionesRespuesta[1]);
  await pagina.unroute('**/salida-spm/respuesta');
  await pagina.getByText('Respuesta del SPM guardada.', { exact: true }).waitFor();
  assert.equal(await pagina.locator('#salidas-fecha_cierre').inputValue(), '2026-10-13');
  assert.equal(await pagina.locator('#salidas-motivo_acta_nuevo').inputValue(), 'La familia no consiente la intervención.');
  assert.equal((await api(`/api/expedientes/${id}/salida-spm`)).datos.respuesta, 'medida_no_necesaria');
  await entrar('javier.medina');
  await pagina.goto(`${origen}/coordinacion/cierres`);
  await pagina.getByRole('link', { name: (await ficha(id)).numero, exact: true }).waitFor();
  await pagina.screenshot({ path: new URL('../../docs/capturas/salidas-cierres-escritorio.png', import.meta.url).pathname, fullPage: true });
  await abrir(id); await descargar(id);
  await pagina.locator('#salidas-fecha_cierre').fill('2026-10-13');
  await pagina.locator('#salidas-motivo_acta_nuevo').fill('La familia no consiente la intervención.');
  await accion('Confirmar fecha y cerrar expediente', '/cerrar');
  await pagina.getByRole('heading', { name: 'Expediente cerrado', exact: true }).waitFor();
  assert.equal((await ficha(id)).estado, 'cerrada'); await descargar(id);
  await pagina.locator('#salidas-fecha_reapertura').fill('2026-10-13');
  await pagina.locator('#salidas-motivo_reapertura').fill('La familia solicita una nueva valoración de su situación.');
  await accion('Reabrir expediente', '/reabrir');
  await pagina.getByText('Expediente reabierto. Se conserva el cierre anterior.', { exact: true }).waitFor();
  assert.equal((await ficha(id)).estado, 'aceptada');
  console.log('CO2: respuesta fechada, cierre por Javier, Acta separada antes/después y reapertura sin consentimiento: OK');
  const forzado = await preparar(); await abrir(forzado);
  await pagina.locator('#salidas-motivo_forzado').fill('La familia se traslada y Coordinación acuerda finalizar el episodio mientras llega la respuesta.');
  await accion('Autorizar cierre sin respuesta', '/salida-spm/forzado');
  await pagina.getByText('Excepción guardada y auditada. Ya puedes confirmar el cierre.', { exact: true }).waitFor();
  assert((await api(`/api/expedientes/${forzado}/salida-spm`)).datos.forzado_id);
  await pagina.locator('#salidas-fecha_cierre').fill('2026-10-13');
  await accion('Confirmar fecha y cerrar expediente', '/cerrar');
  await pagina.getByRole('heading', { name: 'Expediente cerrado', exact: true }).waitFor();
  await pagina.locator('#salidas-fecha_reapertura').fill('2026-10-13');
  await pagina.locator('#salidas-motivo_reapertura').fill('Se retoma el seguimiento con la respuesta del SPM aún pendiente.');
  await accion('Reabrir expediente', '/reabrir');
  await pagina.getByText('Expediente reabierto. Se conserva el cierre anterior.', { exact: true }).waitFor();
  assert.equal((await ficha(forzado)).estado, 'derivada_spm');
  console.log('CO2: forzado motivado, cierre y reapertura con bloqueo SPM conservado: OK');
  await pagina.locator('.persona-conectada > summary').click();
  await pagina.locator('.preferencias > summary').click();
  await pagina.locator('#tema').selectOption('alto-contraste');
  await pagina.getByRole('heading', { name: 'Respuesta del SPM', exact: true }).waitFor();
  await ancho(390); await pagina.screenshot({ path: new URL('../../docs/capturas/salidas-alto-contraste-movil.png', import.meta.url).pathname, fullPage: true });
  await pagina.locator('.persona-conectada > summary').click();
  await pagina.locator('.preferencias > summary').click();
  await pagina.locator('#idioma').selectOption('en');
  await pagina.getByRole('heading', { name: 'SPM response', exact: true }).waitFor();
  for (const width of [1024, 390, 360]) await ancho(width);
  assert.deepEqual(errores, []);
  console.log('CO2: escritorio 1440, 1024, móvil 390/360, alto contraste y sin errores de navegador: OK');
} finally { await navegador.close(); }
