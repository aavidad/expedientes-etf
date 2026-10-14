import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const url = process.argv[2];
assert.match(url, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const capturas = new URL('../../docs/capturas/', import.meta.url);
await mkdir(capturas, { recursive: true });
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
async function entrar(cuenta) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  const page = await context.newPage();
  await page.goto(url);
  await page.locator('#cuenta-ad').fill(cuenta);
  await page.locator('#contrasena-ad').fill(claves[cuenta]);
  await page.locator('button[type="submit"]').click();
  await page.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return { context, page };
}
async function pedir(page, ruta, metodo = 'GET', datos, clave = randomUUID()) {
  return page.evaluate(async ({ ruta, metodo, datos, clave }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const respuesta = await fetch(ruta, { method: metodo, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': clave }, ...(datos ? { body: JSON.stringify(datos) } : {}) });
    return { estado: respuesta.status, datos: await respuesta.json() };
  }, { ruta, metodo, datos, clave });
}
async function sinDesbordamiento(page, width) {
  await page.setViewportSize({ width, height: 1000 });
  await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 1, undefined, { timeout: 5000 });
  await page.evaluate(async () => { await document.fonts.ready; await new Promise((resolver) => requestAnimationFrame(() => requestAnimationFrame(resolver))); });
  await page.waitForTimeout(150);
}
try {
  const marta = await entrar('marta.cabrera');
  const p = marta.page;
  const lista = await pedir(p, '/api/equipo/expedientes');
  assert.equal(lista.estado, 200);
  const id = lista.datos.filas[0].id;
  const ruta = '/api/expedientes/' + id;
  const hoy = new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Madrid' }).format(new Date());
  const fichaURL = url + '/equipo/expediente?id=' + id;
  await p.goto(fichaURL);
  await p.getByRole('tab', { name: /^Menores/ }).click();
  await p.getByRole('button', { name: 'Añadir menor', exact: true }).waitFor();
  const originales = (await pedir(p, ruta + '/menores')).datos;
  assert(originales.menores.length > 0);
  const menor = originales.menores[0];
  const nombreID = `#fi1-nombre-${menor.id}`;
  const nacimientoID = `#fi1-fecha_nacimiento-${menor.id}`;
  await p.locator(nombreID).fill('Lucía');
  await p.locator(nacimientoID).fill('2016-05-11');
  await p.locator(`#fi1-propuesto_atencion_temprana-${menor.id}`).selectOption('false');
  await p.locator(`#fi1-centro_proteccion_texto-${menor.id}`).fill('Centro de protección Sierra Nevada');
  await p.getByRole('button', { name: 'Guardar cambios', exact: true }).click();
  await p.getByText('Cambios guardados con su fecha de vigencia.', { exact: true }).waitFor();
  const guardados = await pedir(p, ruta + '/menores');
  assert.equal(guardados.estado, 200); assert.equal(guardados.datos.menores[0].fecha_nacimiento, '2016-05-11');
  assert(guardados.datos.version > originales.version);
  const historia = await pedir(p, ruta + '/menores/historia?corte=' + hoy);
  assert.equal(historia.estado, 200); assert.equal(historia.datos.vigencia.desde ?? historia.datos.vigencia.vigente_desde, hoy);
  // La edición concurrente usa un guardado real con la versión vigente.
  const concurrente = guardados.datos;
  concurrente.menores[0].centro_educativo = 'Colegio Los Olivos';
  assert.equal((await pedir(p, ruta + '/menores', 'PUT', { version: concurrente.version, vigente_desde: hoy, menores: concurrente.menores })).estado, 200);
  await p.locator(nombreID).fill('Daniel');
  await p.getByRole('button', { name: 'Guardar cambios', exact: true }).click();
  await p.getByText('Otra persona ha cambiado el expediente.', { exact: true }).waitFor();
  assert.equal(await p.locator(nombreID).inputValue(), 'Daniel');
  await p.getByRole('button', { name: 'Ver cambios actuales', exact: true }).click();
  await p.getByText('Colegio Los Olivos', { exact: true }).waitFor();
  assert.equal(await p.locator(nombreID).inputValue(), 'Daniel');
  await p.getByRole('button', { name: 'Conservar mi formulario para guardar', exact: true }).click();
  await p.getByRole('button', { name: 'Guardar cambios', exact: true }).click();
  await p.getByText('Cambios guardados con su fecha de vigencia.', { exact: true }).waitFor();
  assert.equal((await pedir(p, ruta + '/menores')).datos.menores[0].nombre, 'Daniel');
  console.log('FI1-MENORES-OK: guardado real, vigencia, 409 y borrador conservado');
  await p.getByRole('tab', { name: 'Intervención', exact: true }).click();
  await p.locator('#fi1-fase').selectOption('INTERVENCION');
  await p.locator('#fi1-factor_menor').evaluate((e) => { e.closest('details').open = true; });
  await p.locator('#fi1-factor_menor').selectOption('conocido');
  await p.getByLabel('Absentismo escolar', { exact: true }).check();
  const opcion = p.getByLabel('Absentismo escolar', { exact: true }).locator('..').locator('..');
  await opcion.locator('summary').click();
  await opcion.locator('input[type="checkbox"]').nth(1).uncheck();
  await p.locator('#fi1-factor_familia').evaluate((e) => { e.closest('details').open = true; });
  await p.locator('#fi1-factor_familia').selectOption('conocido');
  await p.getByLabel('Desempleo', { exact: true }).check();
  await p.locator('#fi1-tipologia').evaluate((e) => { e.closest('details').open = true; });
  await p.locator('#fi1-tipologia').selectOption('conocido');
  await p.getByLabel('Negligencia', { exact: true }).check();
  await p.getByRole('button', { name: 'Guardar cambios', exact: true }).click();
  await p.getByText('Cambios guardados con su fecha de vigencia.', { exact: true }).waitFor();
  const intervencion = await pedir(p, ruta + '/intervencion');
  assert.equal(intervencion.estado, 200);
  assert(intervencion.datos.intervencion.listas.factor_menor[0].menores_excluidos.includes(menor.id));
  const corte = await pedir(p, ruta + '/intervencion/historia?corte=' + hoy);
  assert.equal(corte.estado, 200);
  assert(!corte.datos.menores.find((m) => m.menor_id === menor.id).listas.factor_menor.includes('ABSENTISMO'));
  console.log('FI1-INTERVENCION-OK: fase, listas y excepción por menor conservadas al corte');
  // Avisos: el enlace abre la pestaña y enfoca el campo de otro menor.
  const avisos = (await pedir(p, ruta + '/avisos')).datos.avisos;
  const aviso = avisos.find((a) => a.campo === 'fecha_nacimiento' && a.sujeto_id !== menor.id);
  if (aviso) {
    await p.locator('#fi1-ficha-avisos').click();
    await p.locator('#fi1-ficha-avisos').locator('..').getByRole('link', { name: /Fecha de nacimiento/ }).first().click();
    await p.waitForFunction((id) => document.activeElement?.id === id, 'fi1-fecha_nacimiento-' + aviso.sujeto_id);
    // Escribir después del aviso no vuelve a llevar el foco al destino anterior.
    await p.locator(nombreID).fill('Daniel');
    assert.equal(await p.locator(nombreID).evaluate((e) => e === document.activeElement), true);
  }
  await p.getByRole('tab', { name: /^Menores/ }).click();
  const guardia = p.getByRole('button', { name: 'Seguir editando', exact: true });
  if (await guardia.isVisible()) await p.getByRole('button', { name: /Descartar/ }).first().click();
  const cuenta = (await pedir(p, ruta + '/menores')).datos.menores.length;
  for (let i = cuenta; i < 15; i++) await p.getByRole('button', { name: 'Añadir menor', exact: true }).click();
  await p.getByText('Has registrado el máximo de 15 menores.', { exact: true }).waitFor();
  assert.equal(await p.getByRole('button', { name: 'Añadir menor', exact: true }).count(), 0);
  await p.getByRole('tab', { name: 'Intervención', exact: true }).click();
  await p.getByRole('button', { name: /Descartar/ }).first().click();
  console.log('FI1-GUARDIA-OK: límite de 15 y salida con cambios pendientes');
  // Capturas de datos inventados con Coordinación, cuya edición está denegada.
  const ines = await entrar('ines.moreno');
  const q = ines.page;
  await q.goto(fichaURL + '&pestana=menores');
  await q.getByText('Puedes consultar estos datos. La edición corresponde al profesional del ETF en un expediente abierto.', { exact: true }).waitFor();
  assert.equal(await q.getByRole('button', { name: 'Guardar cambios', exact: true }).count(), 0);
  assert.equal((await pedir(q, ruta + '/menores', 'PUT', { version: 1, vigente_desde: hoy, menores: [] })).estado, 403);
  for (const tab of ['menores', 'intervencion']) {
    await q.getByRole('tab', { name: tab === 'menores' ? /^Menores/ : 'Intervención', exact: tab !== 'menores' }).click();
    await q.getByRole('tabpanel').locator('fieldset').first().waitFor();
    await sinDesbordamiento(q, 1440);
    await q.screenshot({ path: new URL(`fi1-${tab}-escritorio.png`, capturas).pathname, fullPage: true });
    await sinDesbordamiento(q, 390);
    await q.goto(fichaURL + '&pestana=' + tab);
    await q.getByRole('tabpanel').locator('fieldset').first().waitFor();
    await sinDesbordamiento(q, 390);
    await q.screenshot({ path: new URL(`fi1-${tab}-movil.png`, capturas).pathname, fullPage: true });
    for (const width of [360, 320, 1024]) await sinDesbordamiento(q, width);
  }
  assert.doesNotMatch(await q.title(), /Romero|Molina|Daniel|Lucía/);
  await q.getByRole('tab', { name: 'Intervención', exact: true }).focus();
  await q.keyboard.press('ArrowLeft');
  await q.getByRole('tab', { name: /^Menores/ }).waitFor();
  assert.match(await q.evaluate(() => document.activeElement.id), /equipo-tab-menores/);
  await q.locator('.persona-conectada > summary').click();
  await q.locator('.preferencias > summary').click();
  await q.locator('#tema').selectOption('alto-contraste');
  await q.waitForFunction(() => document.querySelector('link[href="/api/temas/alto-contraste/tokens"]')?.sheet);
  await q.locator('#tema').press('Escape');
  await q.locator('.persona-conectada > summary').press('Escape');
  await sinDesbordamiento(q, 390);
  await q.screenshot({ path: new URL('fi1-menores-alto-contraste.png', capturas).pathname, fullPage: true });
  await q.locator('.persona-conectada > summary').click();
  await q.locator('.preferencias > summary').click();
  await q.locator('#idioma').selectOption('en');
  await q.waitForFunction(() => document.documentElement.lang === 'en');
  await q.getByRole('tab', { name: 'Intervention', exact: true }).click();
  await q.getByRole('tabpanel').getByText('Case phase', { exact: true }).waitFor();
  await sinDesbordamiento(q, 390);
  assert.doesNotMatch(await q.getByRole('tabpanel').innerText(), /INTERVENCION|ABSENTISMO|MALTRATO_FISICO|fi1\./);
  console.log('FI1-COORDINACION-OK: lectura, escritura 403, teclado, capturas 1440/390, reflujo 320/360/1024, alto contraste e inglés');
  // Las averías y el estado vacío se simulan sólo en GET; las escrituras anteriores son reales.
  await p.route('**/api/expedientes/*/menores', (r) => r.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: { clave: 'fi1.error' } }) }));
  await p.goto(url + '/equipo/menores?id=' + id);
  await p.getByRole('tabpanel').getByText('No hemos podido completar la operación. Conservamos lo que has escrito. Inténtalo de nuevo.', { exact: true }).waitFor();
  await p.unroute('**/api/expedientes/*/menores');
  await p.route('**/api/expedientes/*/menores', (r) => r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ...originales, menores: [] }) }));
  await p.goto(url + '/equipo/menores?id=' + id);
  await p.getByText('Todavía no hay menores registrados. Puedes añadir el primero.', { exact: true }).waitFor();
  await p.unroute('**/api/expedientes/*/menores');
  let liberar;
  const retenida = new Promise((resolver) => { liberar = resolver; });
  await p.route('**/api/expedientes/*/menores', async (r) => { await retenida; await r.continue(); });
  await p.goto(url + '/equipo/menores?id=' + id);
  await p.getByText('Estamos leyendo los datos de esta pestaña.', { exact: true }).waitFor();
  liberar();
  await p.getByRole('button', { name: 'Añadir menor', exact: true }).waitFor();
  await p.unroute('**/api/expedientes/*/menores');
  console.log('FI1-ESTADOS-OK: carga anunciada, error recuperable y vacío simulados en lecturas; sin escritura simulada');
} finally { await browser.close(); }
