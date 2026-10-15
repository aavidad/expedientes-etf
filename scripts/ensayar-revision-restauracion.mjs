// Recorrido local de R-22. Sólo usa la demo inventada y Chrome del sistema.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../web/node_modules/playwright-core/index.mjs';
const url = process.argv[2];
assert.match(url, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../despliegue/demo/.local/usuarios.json', import.meta.url)));
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
const pagina = await contexto.newPage();
pagina.setDefaultTimeout(15000);
try {
 await pagina.goto(url);
 await pagina.locator('#cuenta-ad').fill('ines.moreno');
 await pagina.locator('#contrasena-ad').fill(claves['ines.moreno']);
 let respuesta = pagina.waitForResponse(r => r.url() === url + '/api/sesion' && r.request().method() === 'POST');
 await pagina.locator('button[type="submit"]').click();
 assert.equal((await respuesta).status(), 403);
 await pagina.getByRole('alert').waitFor();
 await pagina.locator('#cuenta-ad').fill('elena.salas');
 await pagina.locator('#contrasena-ad').fill(claves['elena.salas']);
 respuesta = pagina.waitForResponse(r => r.url() === url + '/api/sesion' && r.request().method() === 'POST');
 await pagina.locator('button[type="submit"]').click();
 assert.equal((await respuesta).status(), 200);
 await pagina.getByRole('heading', { level: 1, name: 'Revisión tras restaurar una copia' }).waitFor();
 if (process.argv[3] === 'vacia') {
  await pagina.getByText('No hay diferencias de permisos. Confirma la revisión para abrir la aplicación.', { exact: true }).waitFor();
  await pagina.getByRole('button', { name: 'Terminar revisión', exact: true }).click();
  respuesta = pagina.waitForResponse(r => r.url().endsWith('/confirmar') && r.request().method() === 'POST');
  await pagina.getByRole('button', { name: 'Confirmar revisión y abrir', exact: true }).click();
  assert.equal((await respuesta).status(), 200);
  await pagina.getByRole('heading', { level: 1, name: 'Usuarios y permisos' }).waitFor();
  console.log('SEG4-VACIO-OK: revisión sin diferencias exige confirmación visible antes de abrir.');
 } else {
 await pagina.getByRole('button', { name: /^Aplicar cambio/ }).first().waitFor();
 async function pedir(ruta, datos) {
  return pagina.evaluate(async ({ ruta, datos }) => {
   const sesion = await (await fetch('/api/sesion')).json();
   const r = await fetch(ruta, datos ? { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf }, body: JSON.stringify(datos) } : {});
   return { estado: r.status, datos: await r.json() };
  }, { ruta, datos });
 }
 assert.equal((await pedir('/api/admin/usuarios')).estado, 403);
 const revision = (await pedir('/api/admin/restauracion')).datos;
 assert.equal((await pedir(`/api/admin/restauracion/${revision.id}/confirmar`, { version: revision.version })).estado, 409);
 assert(revision.cambios.length >= 4);
 await pagina.getByRole('button', { name: 'Terminar revisión', exact: true }).click();
 await pagina.waitForFunction(() => document.activeElement?.id === 'error-restauracion');
 await pagina.getByRole('button', { name: 'Actualizar revisión', exact: true }).click();
 await pagina.locator('#error-restauracion').waitFor({ state: 'detached' });
 await pagina.getByRole('button', { name: /^Aplicar cambio/ }).first().waitFor();

 const csrf = await pagina.evaluate(async (ruta) => (await fetch(ruta, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ version: 1 }) })).status, `/api/admin/restauracion/${revision.id}/confirmar`);
 assert.equal(csrf, 403);

 // Teclado, temas e idioma largo sobre el armazón real.
 await pagina.keyboard.press('Tab');
 assert(await pagina.evaluate(() => document.activeElement !== document.body));
 await pagina.evaluate(await readFile(new URL('../web/node_modules/axe-core/axe.min.js', import.meta.url), 'utf8'));
 const accesibilidad = await pagina.evaluate(async () => (await axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a','wcag2aa','wcag21a','wcag21aa'] } })).violations.map(v => v.id));
 assert.deepEqual(accesibilidad, []);

 await mkdir(new URL('../docs/capturas/', import.meta.url), { recursive: true });
 await pagina.screenshot({ path: new URL('../docs/capturas/restauracion-escritorio.png', import.meta.url).pathname, fullPage: true });
 await pagina.setViewportSize({ width: 390, height: 844 });
 await pagina.locator('.cabecera-app[data-compacto="true"]').waitFor();
 assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
 await pagina.screenshot({ path: new URL('../docs/capturas/restauracion-movil.png', import.meta.url).pathname, fullPage: true });
 await pagina.setViewportSize({ width: 360, height: 800 });
 await pagina.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth);
 assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
 await pagina.setViewportSize({ width: 1024, height: 768 });

 await pagina.locator('.persona-conectada > summary').click();await pagina.locator('.preferencias > summary').click();
 await pagina.locator('#idioma').selectOption('en');
 await pagina.getByRole('heading', { level: 1, name: 'Review after restoring a backup' }).waitFor();
 await pagina.locator('.persona-conectada > summary').click();await pagina.locator('.preferencias > summary').click();
 await pagina.locator('#tema').selectOption('alto-contraste');
 assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
 await pagina.locator('#idioma').selectOption('es');
 await pagina.getByRole('heading', { level: 1, name: 'Revisión tras restaurar una copia' }).waitFor();
 await pagina.locator('.persona-conectada > summary').click();await pagina.keyboard.press('Escape');

 assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
 // Primera decisión desde UI y reintento rechazado con versión anterior.
 const cambio = revision.cambios[0];
 const r = pagina.waitForResponse(r => r.url().includes('/cambios/') && r.request().method() === 'POST');
 await pagina.getByRole('button', { name: /^Aplicar cambio/ }).first().click();
 assert.equal((await r).status(), 200);
 assert.equal((await pedir(`/api/admin/restauracion/${revision.id}/cambios/${cambio.id}`, { version: revision.version, decision: 'aplicar' })).estado, 409);
 // Resto de decisiones por el contrato HTTP real, incluida una descartada.
 for (;;) {
  const actual = (await pedir('/api/admin/restauracion')).datos;
  const pendiente = actual.cambios.find(c => !c.decision);
  if (!pendiente) break;
  const decision = pendiente.propuesto?.nombre === 'Lucía Navarro Benítez' ? 'descartar' : 'aplicar';
  assert.equal((await pedir(`/api/admin/restauracion/${actual.id}/cambios/${pendiente.id}`, { version: actual.version, decision })).estado, 200);
 }
 await pagina.reload();
 const terminar = pagina.getByRole('button', { name: 'Terminar revisión', exact: true });
 await terminar.focus();await pagina.keyboard.press('Enter');
 await pagina.getByRole('heading', { name: 'Abrir la aplicación al personal', exact: true }).waitFor();
 assert.equal(await pagina.evaluate(() => document.activeElement?.id), 'confirmar-restauracion');
 await pagina.getByRole('button', { name: 'Seguir revisando', exact: true }).focus();await pagina.keyboard.press('Enter');
 await pagina.waitForFunction(() => document.activeElement?.id === 'terminar-restauracion');
 assert.equal(await pagina.evaluate(() => document.activeElement?.id), 'terminar-restauracion');
 await terminar.focus();await pagina.keyboard.press('Enter');
 respuesta = pagina.waitForResponse(r => r.url().endsWith('/confirmar') && r.request().method() === 'POST');
 await pagina.getByRole('button', { name: 'Confirmar revisión y abrir', exact: true }).click();
 assert.equal((await respuesta).status(), 200);
 await pagina.getByRole('heading', { level: 1, name: 'Usuarios y permisos' }).waitFor();
 assert.equal((await pedir('/api/admin/usuarios')).estado, 200);
 // Coordinación vuelve a entrar sólo tras confirmación.
 const otro = await browser.newContext();const p = await otro.newPage();
 await p.goto(url);await p.locator('#cuenta-ad').fill('ines.moreno');await p.locator('#contrasena-ad').fill(claves['ines.moreno']);
 respuesta = p.waitForResponse(r => r.url() === url + '/api/sesion' && r.request().method() === 'POST');await p.locator('button[type="submit"]').click();assert.equal((await respuesta).status(), 200);
 await otro.close();
 console.log('SEG4-CHROME-OK: bloqueo, revisión, aplicar/descartar, conflicto, confirmación, CSRF, teclado, axe WCAG, es/en y capturas 1440/390; sin desbordamiento 360/1024 y alto contraste.');
}
} finally { await browser.close(); }
