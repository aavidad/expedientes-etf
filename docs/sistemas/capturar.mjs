// Capturas del manual de sistemas, limitadas a la demo local del worktree.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const demo = new URL('../../despliegue/demo/.local/', import.meta.url);
const puerto = (await readFile(new URL('puerto', demo), 'utf8')).trim();
assert.match(puerto, /^\d+$/);
const base = `http://127.0.0.1:${puerto}`;
const claves = JSON.parse(await readFile(new URL('usuarios.json', demo), 'utf8'));
const destino = new URL('./capturas/', import.meta.url);
await mkdir(destino, { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
try {
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 } });
  const pagina = await contexto.newPage();
  pagina.setDefaultTimeout(20000);
  const errores = [];
  pagina.on('pageerror', error => errores.push(error.message));
  await pagina.goto(base);
  await pagina.locator('#cuenta-ad').fill('elena.salas');
  await pagina.locator('#contrasena-ad').fill(claves['elena.salas']);
  await pagina.locator('button[type=submit]').click();
  await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  await pagina.goto(`${base}/admin/plantillas`);
  await pagina.getByRole('heading', { name: 'Plantillas Word', exact: true }).waitFor();
  await pagina.getByRole('heading', { name: 'Acta de Aceptación', exact: true }).first().waitFor();
  for (const ancho of [1440, 390]) {
    await pagina.setViewportSize({ width: ancho, height: 1000 });
    await pagina.evaluate(() => document.fonts.ready);
    assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await pagina.screenshot({ path: new URL(`plantillas-${ancho}.png`, destino).pathname });
  }
  await pagina.setViewportSize({ width: 1440, height: 1000 });
  await pagina.goto(`${base}/admin/conservacion`);
  await pagina.getByRole('heading', { name: 'Conservación de expedientes', exact: true }).waitFor();
  await pagina.getByText('No hay expedientes pendientes de revisión ni anonimizados con el plazo cumplido.', { exact: true }).waitFor();
  await pagina.screenshot({ path: new URL('conservacion-1440.png', destino).pathname });
  const respuesta = await contexto.request.get(`${base}/salud`);
  assert.equal(respuesta.status(), 200);
  assert.equal((await respuesta.json()).base, 'ok');
  assert.deepEqual(errores, []);
  console.log('MANSIS-CAPTURAS-OK: Plantillas 1440/390; Conservación 1440; salud 200; sin errores JavaScript.');
} finally {
  await navegador.close();
}
