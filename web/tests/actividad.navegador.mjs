// Recorrido real con datos inventados; requiere make demo en este worktree.
import { chromium } from 'playwright-core';
import { readFileSync, mkdirSync } from 'node:fs';
import assert from 'node:assert/strict';
import { join, resolve } from 'node:path';
import { tmpdir } from 'node:os';
const raiz = resolve(import.meta.dirname, '../..');
const usuarios = readFileSync(resolve(raiz, 'despliegue/demo/USUARIOS.local.md'), 'utf8');
const puerto = readFileSync(resolve(raiz, 'despliegue/demo/.local/puerto'), 'utf8').trim();
const base = `http://127.0.0.1:${puerto}`;
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contextos = [];
const creados = [];
const capturas = resolve(raiz, 'docs/capturas'); mkdirSync(capturas, { recursive: true });
async function entrar(cuenta) {
  const contrasena = usuarios.match(new RegExp('`' + cuenta.replaceAll('.', '\\.') + '` \\| `([^`]+)`'))?.[1];
  assert.ok(contrasena, 'Credencial local disponible');
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 } }); contextos.push(contexto);
  const pagina = await contexto.newPage();
  pagina.on('pageerror', (e) => { throw e; });
  await pagina.goto(base);
  await pagina.getByLabel('Usuario de la Diputación', { exact: true }).fill(cuenta);
  await pagina.getByLabel('Contraseña', { exact: true }).fill(contrasena);
  await pagina.getByRole('button', { name: 'Entrar', exact: true }).click();
  try { await pagina.waitForURL((u) => u.pathname !== '/entrar', { timeout: 10000 }); } catch (e) { console.log({ cuenta, avisos: await pagina.locator('[role=alert]').allTextContents() }); throw e; }
  return pagina;
}
async function api(pagina, ruta, metodo = 'GET', datos) {
  return pagina.evaluate(async ({ ruta, metodo, datos }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const r = await fetch(ruta, { method: metodo, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf }, ...(datos ? { body: JSON.stringify(datos) } : {}) });
    return { estado: r.status, cuerpo: await r.json() };
  }, { ruta, metodo, datos });
}
async function lista(pagina, tipo) {
  await pagina.getByRole('radio', { name: tipo === 'comisiones' ? 'Comisiones técnicas' : 'Informes técnicos', exact: true }).check();
  await pagina.getByLabel('Año', { exact: true }).fill('2026');
  await pagina.getByRole('heading', { name: tipo === 'comisiones' ? 'Comisiones técnicas' : 'Informes técnicos', exact: true }).waitFor();
  await pagina.waitForFunction(() => !document.querySelector('.actividad')?.getAttribute('aria-busy')?.includes('true') && !document.querySelector('.actividad [role="status"]')?.textContent?.includes('Consultando'));
}
async function guardar(pagina, tipo, dato, fecha = '2026-10-10') {
  await lista(pagina, tipo);
  await pagina.getByRole('button', { name: tipo === 'comisiones' ? 'Registrar comisión técnica' : 'Registrar informe técnico', exact: true }).click();
  await pagina.getByLabel('Fecha', { exact: true }).fill(fecha);
  if (tipo === 'comisiones') await pagina.getByLabel('Observaciones (opcional)', { exact: true }).fill(dato);
  else { await pagina.getByLabel('Organismo destinatario', { exact: true }).selectOption('OTROS'); await pagina.getByLabel('Nombre del organismo', { exact: true }).fill(dato); }
  await pagina.getByRole('button', { name: 'Guardar registro', exact: true }).click();
  await pagina.getByText('El registro se ha guardado.', { exact: true }).waitFor();
  const r = await api(pagina, `/api/${tipo}?anio=2026`); assert.equal(r.estado, 200);
  const f = r.cuerpo.find((f) => (f.observaciones || f.organismo_otros) === dato); assert.ok(f); creados.push({ tipo, id: f.id }); return f;
}
async function retirar(pagina, tipo, dato) {
  await lista(pagina, tipo);
  const fila = pagina.getByRole('row').filter({ hasText: dato });
  await fila.getByRole('button', { name: 'Retirar registro' }).click();
  await pagina.getByRole('dialog').getByRole('button', { name: 'Cancelar', exact: true }).click();
  await fila.getByRole('button', { name: 'Retirar registro' }).click();
  await pagina.getByRole('dialog').getByRole('button', { name: 'Confirmar retirada', exact: true }).click();
  await pagina.getByText('El registro se ha retirado.', { exact: true }).waitFor();
  assert.equal((await api(pagina, `/api/${tipo}?anio=2026`)).cuerpo.some((f) => (f.observaciones || f.organismo_otros) === dato), false);
}
function sinDesborde(pagina) { return pagina.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth); }
let marta;
try {
  marta = await entrar('marta.cabrera');
  await marta.getByRole('link', { name: 'Comisiones e informes', exact: true }).click();
  await marta.getByRole('button', { name: 'Registrar comisión técnica', exact: true }).click();
  await marta.getByRole('button', { name: 'Guardar registro', exact: true }).click();
  await marta.getByRole('link', { name: 'Fecha', exact: true }).click();
  assert.equal(await marta.getByLabel('Fecha', { exact: true }).evaluate((e) => e === document.activeElement), true);
  await marta.getByRole('button', { name: 'Cancelar', exact: true }).click();
  await marta.getByRole('button', { name: 'Registrar comisión técnica', exact: true }).click();
  await marta.getByLabel('Fecha', { exact: true }).fill('2026-10-09'); await marta.getByLabel('Observaciones (opcional)').fill('Seguimiento de acuerdos comunitarios');
  await marta.route('**/api/comisiones', (ruta) => ruta.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ id: 'registro-invalido', version: 0 }) }));
  await marta.getByRole('button', { name: 'Guardar registro', exact: true }).click();
  await marta.getByText('No hemos recibido la confirmación. Consulta los registros actuales antes de volver a guardar o retirar.', { exact: true }).waitFor();
  assert.equal(await marta.getByRole('button', { name: 'Guardar registro', exact: true }).isDisabled(), true); assert.equal(await marta.getByLabel('Observaciones (opcional)').inputValue(), 'Seguimiento de acuerdos comunitarios');
  await marta.unroute('**/api/comisiones'); await marta.getByRole('button', { name: 'Ver registros actuales', exact: true }).click();
  const textosUsuario = JSON.parse(readFileSync(resolve(raiz, 'idiomas/es/textos/usuarios.json'), 'utf8'));
  await marta.getByRole('button', { name: textosUsuario['usuarios.descartar'], exact: true }).click();
  const temporal = await guardar(marta, 'comisiones', `Revisión del seguimiento comunitario`);
  let fila = marta.getByRole('row').filter({ hasText: temporal.observaciones });
  await fila.getByRole('button', { name: 'Corregir registro' }).click();
  const borrador = `Acuerdos de seguimiento trimestral`;
  await marta.getByLabel('Observaciones (opcional)').fill(borrador);
  const concurrente = await api(marta, `/api/comisiones/${temporal.id}`, 'PUT', { fecha: temporal.fecha, observaciones: `Revisión compartida`, version_esperada: temporal.version }); assert.equal(concurrente.estado, 200);
  const respuestaConflicto = marta.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes('/api/comisiones/'));
  await marta.getByRole('button', { name: 'Guardar registro', exact: true }).click(); assert.equal((await respuestaConflicto).status(), 409);
  await marta.route('**/api/comisiones?anio=2026', (ruta) => ruta.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: 'actividad.no_disponible' }) }));
  await marta.getByRole('button', { name: 'Ver registros actuales', exact: true }).click();
  await marta.getByText('No se puede completar la operación. Inténtalo de nuevo.', { exact: true }).waitFor();
  assert.equal(await marta.getByLabel('Observaciones (opcional)').inputValue(), borrador);
  await marta.unroute('**/api/comisiones?anio=2026');
  await marta.getByRole('button', { name: 'Ver registros actuales', exact: true }).click();
  await marta.getByRole('heading', { name: 'Datos guardados ahora', exact: true }).waitFor();
  assert.equal(await marta.getByLabel('Observaciones (opcional)').inputValue(), borrador);
  await marta.getByRole('button', { name: 'Conservar lo escrito y corregir esta versión', exact: true }).click();
  await marta.getByRole('button', { name: 'Guardar registro', exact: true }).click();
  await marta.getByText('El registro se ha guardado.', { exact: true }).waitFor();
  const corregida = (await api(marta, '/api/comisiones?anio=2026')).cuerpo.find((f) => f.id === temporal.id); assert.equal(corregida.version, 3); assert.equal(corregida.observaciones, borrador);
  await retirar(marta, 'comisiones', borrador);
  const comision = await guardar(marta, 'comisiones', `Coordinación de recursos comunitarios de la comarca`, '2026-10-08');
  const informe = await guardar(marta, 'informes', `Unidad de apoyo familiar del Valle`, '2026-10-09');
  fila = marta.getByRole('row').filter({ hasText: informe.organismo_otros }); await fila.getByRole('button', { name: 'Corregir registro' }).click();
  await marta.getByLabel('Fecha', { exact: true }).fill('2026-10-11'); await marta.getByRole('button', { name: 'Guardar registro', exact: true }).click(); await marta.getByText('El registro se ha guardado.', { exact: true }).waitFor();
  assert.equal((await api(marta, '/api/informes?anio=2026')).cuerpo.find((f) => f.id === informe.id).version, 2);
  const informeRetirado = await guardar(marta, 'informes', `Centro de mediación comunitaria`); await retirar(marta, 'informes', informeRetirado.organismo_otros);
  const equiposMarta = await api(marta, '/api/actividad/equipos'); assert.equal(equiposMarta.cuerpo.length, 1); assert.equal(equiposMarta.cuerpo[0].id, comision.equipo_id);
  const ines = await entrar('ines.moreno'); await ines.getByRole('link', { name: 'Comisiones e informes', exact: true }).click(); await lista(ines, 'comisiones');
  assert.equal(await ines.getByRole('button', { name: 'Registrar comisión técnica', exact: true }).count(), 0);
  const equiposInes = await api(ines, '/api/actividad/equipos'); assert.equal(equiposInes.cuerpo.length, 8);
  await ines.getByLabel('Equipo', { exact: true }).selectOption(comision.equipo_id); await ines.getByRole('row').filter({ hasText: comision.observaciones }).waitFor();
  assert.equal((await api(marta, `/api/comisiones?anio=2026&etf=${equiposInes.cuerpo.find((e) => e.id !== comision.equipo_id).id}`)).estado, 403);
  assert.equal((await api(ines, '/api/informes', 'POST', { fecha: '2026-10-12', organismo: 'SPAF' })).estado, 403);
  await ines.screenshot({ path: resolve(capturas, 'actividad-comisiones-escritorio.png'), fullPage: true });
  await lista(ines, 'informes'); await ines.getByRole('row').filter({ hasText: informe.organismo_otros }).waitFor();
  await ines.getByLabel('Organismo destinatario', { exact: true }).selectOption('OTROS');
  await ines.screenshot({ path: resolve(capturas, 'actividad-informes-escritorio.png'), fullPage: true });
  await ines.getByLabel('Desde el día', { exact: true }).fill('2026-10-12'); await ines.getByText('No hay informes técnicos con estos filtros.', { exact: true }).waitFor(); await ines.getByLabel('Desde el día', { exact: true }).fill('');
  for (const ancho of [1024, 390, 360, 320]) { await ines.setViewportSize({ width: ancho, height: 1000 }); await ines.waitForTimeout(100); const sinDesbordar = await sinDesborde(ines); if (!sinDesbordar) { console.log(await ines.evaluate(() => [...document.querySelectorAll('body *')].map((e) => ({ tag: e.tagName, clase: e.className, right: e.getBoundingClientRect().right, width: e.getBoundingClientRect().width })).filter((e) => e.right > document.documentElement.clientWidth))); await ines.screenshot({ path: join(tmpdir(), 'a1-desborde.png'), fullPage: true }); } assert.equal(sinDesbordar, true, `Sin desborde a ${ancho}`); }
  await ines.setViewportSize({ width: 390, height: 1000 }); await ines.screenshot({ path: resolve(capturas, 'actividad-informes-movil.png'), fullPage: true });
  await ines.getByRole('button', { name: 'Ayuda sobre comisiones e informes' }).focus(); await ines.keyboard.press('Enter'); await ines.getByRole('dialog').waitFor(); await ines.keyboard.press('Escape'); assert.equal(await ines.getByRole('button', { name: 'Ayuda sobre comisiones e informes' }).evaluate((e) => e === document.activeElement), true);

  await ines.route('**/api/informes?anio=2025*', (ruta) => ruta.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: 'actividad.no_disponible' }) }));
  await ines.getByLabel('Año', { exact: true }).fill('2025'); await ines.getByText('No se puede completar la operación. Inténtalo de nuevo.', { exact: true }).waitFor();
  await ines.unroute('**/api/informes?anio=2025*'); await ines.getByRole('button', { name: 'Volver a consultar', exact: true }).click(); await ines.getByText('No hay informes técnicos con estos filtros.', { exact: true }).waitFor();
  await ines.getByLabel('Año', { exact: true }).fill('2026'); await ines.getByRole('row').filter({ hasText: informe.organismo_otros }).waitFor();
  await ines.locator('.persona-conectada > summary').click(); await ines.locator('.preferencias > summary').click();
  await ines.getByLabel('Tema', { exact: true }).selectOption('alto-contraste');
  const contraste = JSON.parse(readFileSync(resolve(raiz, 'temas/alto-contraste/tokens.json'), 'utf8'));
  await ines.waitForFunction((fondo) => getComputedStyle(document.documentElement).getPropertyValue('--color-fondo').trim() === fondo, contraste['color.fondo']);
  await ines.getByLabel('Idioma', { exact: true }).selectOption('en'); await ines.getByRole('heading', { name: 'Technical committees and reports', exact: true }).waitFor(); assert.equal(await sinDesborde(ines), true);
  await ines.keyboard.press('Escape'); await ines.keyboard.press('Escape');
  await ines.getByRole('button', { name: 'Help with technical committees and reports', exact: true }).focus(); await ines.keyboard.press('Enter'); await ines.getByRole('dialog').waitFor(); await ines.keyboard.press('Escape');
  const lucia = await entrar('lucia.navarro'); assert.equal((await api(lucia, '/api/informes?anio=2026')).estado, 403); assert.equal(await lucia.getByRole('link', { name: 'Comisiones e informes', exact: true }).count(), 0);
  console.log('ACTIVIDAD-RECORRIDO-OK: registro, corrección, 409 y reintento sin perder borrador, retirada, filtros, ámbitos, Consulta, respuesta inválida, teclado, error/reintento, alto contraste/inglés y 1440/1024/390/360/320; tres capturas PNG.');
} finally {
  // Se retiran sólo los registros creados por este recorrido.
  if (marta) for (const { tipo, id } of creados) { try { const r = await api(marta, `/api/${tipo}?anio=2026`); const f = r.cuerpo.find((f) => f.id === id); if (f) await api(marta, `/api/${tipo}/${id}?version_esperada=${f.version}`, 'DELETE'); } catch {} }
  for (const c of contextos) await c.close(); await navegador.close();
}
