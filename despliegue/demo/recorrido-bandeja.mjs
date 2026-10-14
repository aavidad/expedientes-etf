// Recorrido sobre la demostración local y sus datos inventados.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const url = process.argv[2];
assert.match(url, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 } });
const pagina = await contexto.newPage();
pagina.setDefaultTimeout(20000);
const errores = [];
pagina.on('pageerror', (error) => errores.push(error.message));
const rutasBusqueda = [];
pagina.on('request', (r) => { if (r.url().includes('/bandeja')) rutasBusqueda.push(r.url()); });
async function esperarLista(accion) {
  const espera = pagina.waitForResponse((r) => r.url() === url + '/api/coordinacion/bandeja/buscar' && r.request().method() === 'POST');
  await accion();
  const r = await espera;
  assert.equal(r.status(), 200);
  const datos = await r.json();
  await pagina.getByRole('status').filter({ hasText: /\d+ derivaci[oó]n|\d+ referral/ }).waitFor();
  return datos;
}
async function sinDesbordamiento(ancho) {
  await pagina.setViewportSize({ width: ancho, height: 1000 });
  await pagina.waitForTimeout(250);
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), `desbordamiento a ${ancho}`);
}
try {
  await pagina.goto(url);
  await pagina.locator('#cuenta-ad').fill('ines.moreno');
  await pagina.locator('#contrasena-ad').fill(claves['ines.moreno']);
  const datos = await esperarLista(() => pagina.locator('button[type="submit"]').click());
  await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  assert(datos.total > 20);
  assert.equal(datos.filas.length, 20);
  assert(datos.filas.every((f) => f.estado === 'derivacion_recibida'));
  assert(!JSON.stringify(datos).includes('Serrano'));
  assert.equal(await pagina.getByRole('link', { name: 'Subir Anexo' }).getAttribute('href'), '/coordinacion/anexos/nuevo');
  assert.match(await pagina.getByRole('link', { name: 'Revisar derivación' }).first().getAttribute('href'), /^\/coordinacion\/derivaciones\/[a-f0-9-]+$/);
  await mkdir(new URL('../../docs/capturas/', import.meta.url), { recursive: true });
  await pagina.screenshot({ path: new URL('../../docs/capturas/bandeja-escritorio.png', import.meta.url).pathname, fullPage: true });
  const segunda = await esperarLista(async () => { await pagina.getByRole('button', { name: 'Página siguiente', exact: true }).focus(); await pagina.keyboard.press('Enter'); });
  assert.equal(segunda.filas.length, datos.total - 20);
  await esperarLista(() => pagina.getByRole('button', { name: 'Página anterior', exact: true }).click());
  for (const [etiqueta, estado] of [['Aceptados recientes', 'aceptada'], ['No aceptados', 'no_aceptada'], ['Lista de espera', 'lista_espera']]) {
    const r = await esperarLista(() => pagina.getByRole('button', { name: new RegExp(etiqueta) }).click());
    assert(r.total > 0);
    assert(r.filas.every((f) => f.estado === estado));
    assert(r.filas.every((f) => f.subprograma === 'RIESGO_SOCIAL'));
  }
  await esperarLista(() => pagina.getByRole('button', { name: /Pendientes de revisar/ }).click());
  await pagina.getByLabel('Buscar por número de expediente o familia').fill('Serrano');
  const buscados = await esperarLista(() => pagina.getByRole('button', { name: 'Buscar', exact: true }).click());
  assert(buscados.total > 0);
  assert(buscados.filas.every((f) => f.iniciales === 'S. F.'));
  assert(!JSON.stringify(buscados).includes('Serrano'));
  assert(rutasBusqueda.every((ruta) => !ruta.includes('Serrano') && !ruta.includes('busqueda=')));
  await pagina.getByLabel('Buscar por número de expediente o familia').fill('Familia inexistente');
  await esperarLista(() => pagina.getByRole('button', { name: 'Buscar', exact: true }).click());
  await pagina.getByText('No hay derivaciones con estos filtros. Prueba a quitar alguno.').waitFor();
  await esperarLista(() => pagina.getByRole('button', { name: 'Quitar filtros', exact: true }).click());
  await sinDesbordamiento(1024);
  await sinDesbordamiento(390);
  await pagina.screenshot({ path: new URL('../../docs/capturas/bandeja-movil.png', import.meta.url).pathname, fullPage: true });
  await sinDesbordamiento(360);
  await sinDesbordamiento(320);
  await pagina.keyboard.press('Tab');
  assert(await pagina.evaluate(() => document.activeElement?.tagName !== 'BODY'));
  await pagina.setViewportSize({ width: 1440, height: 1000 });
  await pagina.locator('.persona-conectada > summary').click();
  await pagina.getByText('Preferencias', { exact: true }).click();
  await pagina.getByLabel('Tema', { exact: true }).selectOption('alto-contraste');
  await pagina.waitForFunction(() => document.querySelector('link[href*="alto-contraste"]')?.sheet);
  await pagina.keyboard.press('Escape');
  await sinDesbordamiento(390);
  await pagina.setViewportSize({ width: 1440, height: 1000 });
  await pagina.locator('.persona-conectada > summary').click();
  await pagina.getByText('Preferencias', { exact: true }).click();
  await pagina.getByLabel('Idioma', { exact: true }).selectOption('en');
  await pagina.getByRole('heading', { name: 'Coordination inbox', exact: true }).waitFor();
  await sinDesbordamiento(390);
  await pagina.route('**/api/coordinacion/bandeja/buscar', (ruta) => ruta.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: { clave: 'expediente.no_disponible' } }) }));
  await pagina.getByRole('button', { name: /Recently accepted/ }).click();
  await pagina.getByRole('alert').waitFor();
  await pagina.unroute('**/api/coordinacion/bandeja/buscar');
  await esperarLista(() => pagina.getByRole('button', { name: 'Try again', exact: true }).click());
  assert.deepEqual(errores, []);
  console.log('BANDEJA-RECORRIDO-OK: Chrome del sistema; cuatro estados; paginación; búsqueda enmascarada; vacío, error y reintento; teclado; 1440/1024/390/360/320; alto contraste e inglés.');
} finally {
  await contexto.close();
  await navegador.close();
}
