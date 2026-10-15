// Renueva las capturas del manual con las cuentas inventadas de su demo local.
import assert from 'node:assert/strict';
import { mkdir, readFile } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const local = new URL('../../despliegue/demo/.local/', import.meta.url);
const puerto = (await readFile(new URL('puerto', local), 'utf8')).trim();
assert.match(puerto, /^\d+$/);
const base = `http://127.0.0.1:${puerto}`;
const claves = JSON.parse(await readFile(new URL('usuarios.json', local), 'utf8'));
const capturas = new URL('./capturas/', import.meta.url);
await mkdir(capturas, { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];

try {
  for (const [nombre, cuenta, ruta, api, espera] of [
    ['plantillas', 'elena.salas', '/admin/plantillas', '/api/admin/plantillas', '.plantillas-tarjeta'],
    ['memoria', 'lucia.navarro', '/memoria', '/api/consulta/memoria/', 'main h1'],
  ]) {
    const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'es-ES' });
    const pagina = await contexto.newPage();
    pagina.setDefaultTimeout(30000);
    pagina.on('pageerror', error => errores.push(error.message));
    const respuestas = [];
    pagina.on('response', respuesta => {
      if (new URL(respuesta.url()).pathname.startsWith(api)) respuestas.push(respuesta);
    });
    await pagina.goto(base);
    await pagina.locator('#cuenta-ad').fill(cuenta);
    await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
    await pagina.locator('button[type=submit]').click();
    await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
    const acceso = await pagina.evaluate(async () => (await fetch('/api/sesion')).json());
    assert.equal(acceso.estado, 'activo');
    await pagina.goto(`${base}${ruta}`);
    await pagina.locator(espera).first().waitFor();
    await pagina.waitForFunction(() => !document.querySelector('[aria-busy="true"]'));
    // Espera la respuesta de la capacidad antes de capturar su estado confirmado.
    await pagina.waitForFunction(() => document.readyState === 'complete');
    if (nombre === 'memoria') {
      await pagina.getByRole('heading', { name: 'Casos atendidos', exact: false }).first().waitFor();
    }
    assert.ok(respuestas.length > 0, `Falta respuesta API: ${nombre}`);
    for (const respuesta of respuestas) assert.equal(respuesta.status(), 200);
    for (const ancho of [1440, 390]) {
      await pagina.setViewportSize({ width: ancho, height: ancho === 390 ? 844 : 1000 });
      await pagina.evaluate(async () => { await document.fonts.ready; });
      await pagina.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth);
      await pagina.screenshot({ path: new URL(`${nombre}-${ancho}.png`, capturas).pathname });
      console.log(`CAPTURA-OK: ${nombre}-${ancho}.png; API 200; sin desbordamiento`);
    }
    await contexto.close();
  }
  assert.deepEqual(errores, []);
} finally {
  await navegador.close();
}
