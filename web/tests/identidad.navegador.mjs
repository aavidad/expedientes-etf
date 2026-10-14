// Ejecutado por TestNavegadorIdentidad con HTTP, PostgreSQL y LDAPS desechables.
import { chromium } from 'playwright-core';
import assert from 'node:assert/strict';
import { mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';

const base = process.env.ETF_PRUEBA_URL;
const evidencia = process.env.ETF_EVIDENCIA_RUTA;
assert.ok(base && new URL(base).hostname === '127.0.0.1');
mkdirSync(evidencia, { recursive: true });
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const crear = async (width = 1024) => browser.newContext({ ignoreHTTPSErrors: true, viewport: { width, height: 900 } });
async function persona(page) {
  const detalle = page.locator('.persona-conectada');
  if (await detalle.count() && !await detalle.evaluate((e) => e.open)) await detalle.locator(':scope > summary').click();
}
async function preferencias(page) {
  await persona(page);
  if (!await page.locator('.preferencias').evaluate((e) => e.open)) await page.locator('.preferencias > summary').click();
}
async function entrar(page, idioma = 'es') {
  const en = idioma === 'en';
  await page.getByLabel(en ? 'Diputación username' : 'Usuario de la Diputación').fill('ines.moreno');
  await page.getByLabel(en ? 'Password' : 'Contraseña', { exact: true }).fill(process.env.ETF_PRUEBA_CLAVE);
  // Teclado: tabular al botón y enviar el formulario con Enter.
  await page.getByLabel(en ? 'Password' : 'Contraseña', { exact: true }).press('Tab');
  assert.equal(await page.evaluate(() => document.activeElement?.tagName), 'BUTTON');
  await page.keyboard.press('Enter');
  await page.getByRole('heading', { name: en ? 'Access awaiting approval' : 'Acceso pendiente de alta' }).waitFor();
  assert.equal(new URL(page.url()).pathname, '/pendiente');
}
try {
  for (const [width, idioma, tema] of [[1024, 'es', 'expedientes-etf'], [390, 'en', 'alto-contraste'], [320, 'es', 'alto-contraste']]) {
    const contexto = await crear(width);
    await contexto.tracing.start({ screenshots: true, snapshots: true });
    const page = await contexto.newPage();
    const fallos = [];
    page.on('pageerror', (err) => fallos.push(err.message));
    await page.goto(`${base}/entrar`);
    await page.getByLabel('Usuario de la Diputación').waitFor();
    if (idioma === 'en') { await preferencias(page); await page.getByLabel('Idioma').selectOption('en'); }
    const boton = page.getByRole('button', { name: idioma === 'en' ? 'Sign in' : 'Entrar', exact: true });
    await boton.waitFor();
    if (tema !== 'expedientes-etf') {
      await preferencias(page);
      await page.getByLabel(idioma === 'en' ? 'Theme' : 'Tema').selectOption(tema);
      await page.keyboard.press('Escape');
      await page.waitForFunction(() => [...document.querySelectorAll('link')].some((l) => l.href.endsWith('/alto-contraste/tokens')) && ![...document.querySelectorAll('link')].some((l) => l.href.endsWith('/expedientes-etf/tokens')));
    }
    await page.screenshot({ path: join(evidencia, `entrada-${width}-${idioma}.png`) });
    await entrar(page, idioma);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length), 0);
    assert.equal((await page.locator('body').innerText()).includes('Inés Moreno'), false);
    const cookie = (await contexto.cookies()).find((c) => c.name === '__Host-etf-sesion');
    assert.ok(cookie?.httpOnly && cookie.secure && cookie.sameSite === 'Strict');
    assert.equal(await page.evaluate(() => document.cookie.includes('__Host-etf-sesion')), false);
    const consulta = await page.request.get(`${base}/api/sesion`);
    assert.equal(consulta.status(), 200);
    assert.equal((await consulta.json()).estado, 'pendiente_alta');
    const csrfRechazado = await page.request.delete(`${base}/api/sesion`, { headers: { Origin: base } });
    assert.equal(csrfRechazado.status(), 403);
    await page.screenshot({ path: join(evidencia, `pendiente-${width}-${idioma}.png`) });
    await page.reload();
    await page.getByRole('heading', { name: /^(Acceso pendiente de alta|Access awaiting approval)$/ }).waitFor();
    const idiomaRecarga = await page.locator('html').getAttribute('lang');
    if (idiomaRecarga !== idioma) { await preferencias(page); await page.getByLabel(idiomaRecarga === 'es' ? 'Idioma' : 'Language').selectOption(idioma); }
    await page.getByRole('heading', { name: idioma === 'en' ? 'Access awaiting approval' : 'Acceso pendiente de alta' }).waitFor();
    await persona(page);
    await page.getByRole('button', { name: idioma === 'en' ? 'Sign out' : 'Cerrar sesión' }).click();
    await page.getByRole('button', { name: idioma === 'en' ? 'Sign in' : 'Entrar', exact: true }).waitFor();
    assert.equal(new URL(page.url()).pathname, '/entrar');
    assert.equal((await contexto.cookies()).some((c) => c.name === '__Host-etf-sesion'), false);
    assert.deepEqual(fallos, []);
    await contexto.tracing.stop({ path: join(evidencia, `recorrido-${width}-${idioma}.zip`) });
    await contexto.close();
    console.log(`RECORRIDO-OK ${width}px ${idioma} ${tema}: entrar → pendiente → salir; teclado, cookies, CSRF, recarga y privacidad`);
  }
  const contexto = await crear(); const page = await contexto.newPage();
  await page.goto(`${base}/entrar`); await page.getByLabel('Usuario de la Diputación').waitFor();
  await page.getByLabel('Usuario de la Diputación').fill('ines.moreno');
  await page.getByLabel('Contraseña', { exact: true }).fill('clave-inventada-no-valida');
  await page.getByRole('button', { name: 'Entrar', exact: true }).click();
  await page.getByRole('alert').filter({ hasText: 'No se ha podido entrar con ese usuario y contraseña.' }).waitFor();
  assert.equal((await contexto.cookies()).some((c) => c.name === '__Host-etf-sesion'), false);
  await entrar(page);
  execFileSync('docker', ['stop', '--time', '1', process.env.ETF_PRUEBA_LDAP], { stdio: 'ignore' });
  const otro = await crear(); const nueva = await otro.newPage();
  // Detener el contenedor cambia las interfaces del host. Chrome puede cancelar
  // ese GET con ERR_NETWORK_CHANGED; se repite sólo la navegación, nunca el POST.
  try { await nueva.goto(`${base}/entrar`); } catch (error) {
    if (!String(error).includes('net::ERR_NETWORK_CHANGED')) throw error;
    await nueva.goto(`${base}/entrar`);
  }
  await nueva.getByLabel('Usuario de la Diputación').waitFor();
  await nueva.getByLabel('Usuario de la Diputación').fill('ines.moreno');
  await nueva.getByLabel('Contraseña', { exact: true }).fill(process.env.ETF_PRUEBA_CLAVE);
  await nueva.getByRole('button', { name: 'Entrar', exact: true }).click();
  await nueva.getByRole('alert').filter({ hasText: 'El servicio de acceso de la Diputación no responde.' }).waitFor();
  assert.equal((await otro.cookies()).some((c) => c.name === '__Host-etf-sesion'), false);
  assert.equal((await page.request.get(`${base}/api/sesion`)).status(), 200);
  await persona(page);
  await page.getByRole('button', { name: 'Cerrar sesión' }).click();
  await page.getByRole('button', { name: 'Entrar', exact: true }).waitFor();
  await otro.close(); await contexto.close();
  console.log('AD-CAIDO-OK: entrada bloqueada; sesión anterior consultable y cierre disponible');
  console.log(`EVIDENCIA ${evidencia}`);
} finally { await browser.close(); }
