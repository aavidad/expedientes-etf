// HTTP, LDAPS y PostgreSQL desechables; Chrome instalado en el sistema.
import { chromium } from 'playwright-core';
import assert from 'node:assert/strict';
import { mkdirSync } from 'node:fs';
import { join } from 'node:path';
const base = process.env.ETF_PRUEBA_URL;
assert.equal(new URL(base).hostname, '127.0.0.1');
const evidencia = process.env.ETF_EVIDENCIA_RUTA;
mkdirSync(evidencia, { recursive: true });
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1280, height: 900 } });
const contextoUsuario = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1024, height: 900 } });
async function entrar(page, cuenta) {
  await page.goto(`${base}/entrar`); await page.getByLabel('Usuario de la Diputación').fill(cuenta);
  await page.getByLabel('Contraseña', { exact: true }).fill(process.env.ETF_PRUEBA_CLAVE);
  await page.getByLabel('Contraseña', { exact: true }).press('Tab'); await page.keyboard.press('Enter');
}
async function persona(page) {
  if (!await page.locator('.persona-conectada').evaluate((e) => e.open)) await page.locator('.persona-conectada > summary').click();
}
async function preferencias(page) {
  await persona(page);
  if (!await page.locator('.preferencias').evaluate((e) => e.open)) await page.locator('.preferencias > summary').click();
}
async function guardar(page, idioma = 'es') {
  const respuesta = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().endsWith('/perfiles'));
  await page.getByRole('button', { name: idioma === 'es' ? 'Guardar permisos' : 'Save permissions', exact: true }).click();
  assert.equal((await respuesta).status(), 200);
  await page.getByRole('status').filter({ hasText: idioma === 'es' ? 'Permisos guardados.' : 'Permissions saved.' }).waitFor();
}
async function navegar(page, nombre) {
  if (await page.locator('.cabecera-app').getAttribute('data-compacto') === 'true' && !await page.locator('.lateral-app').isVisible()) await page.getByRole('button', { name: /^(Menú|Menu)$/ }).click();
  await page.getByRole('link', { name: nombre, exact: true }).click();
}
try {
  await contexto.tracing.start({ screenshots: true, snapshots: true });
  const admin = await contexto.newPage(); const usuario = await contextoUsuario.newPage();
  const errores = [];
  admin.on('pageerror', (e) => errores.push(e.message)); usuario.on('pageerror', (e) => errores.push(e.message));
  await entrar(usuario, 'ines.moreno'); await usuario.getByRole('heading', { name: 'Acceso pendiente de alta' }).waitFor();
  assert.equal((await usuario.request.get(`${base}/api/admin/usuarios`)).status(), 403);
  await entrar(admin, 'marta.lopez'); await admin.getByRole('heading', { name: 'Salario mínimo interprofesional' }).waitFor();
  await navegar(admin, 'Usuarios y permisos'); await admin.getByRole('heading', { name: 'Usuarios y permisos' }).waitFor();
  assert.equal(await admin.getByRole('table', { name: 'Usuarios y permisos' }).count(), 1);
  await admin.getByRole('rowheader', { name: 'Inés Moreno Castillo ines.moreno', exact: true }).waitFor();
  assert.equal(await admin.locator('.resumen-persona > strong').innerText(), 'Marta López Rivas');
  await admin.getByRole('button', { name: 'Editar permisos de Inés Moreno Castillo (ines.moreno)' }).click();
  await admin.getByLabel('Profesional de ETF', { exact: true }).check(); await admin.getByLabel('Consulta', { exact: true }).check();
  const opciones = await admin.locator('#equipo-usuario option').evaluateAll((els) => els.map((e) => e.value).filter(Boolean));
  assert.ok(opciones.length >= 2); await admin.getByLabel('Equipo', { exact: true }).selectOption(opciones[0]); await guardar(admin);
  await usuario.reload(); await usuario.getByRole('heading', { name: 'Acceso confirmado' }).waitFor();
  await persona(usuario); await usuario.locator('.perfiles-persona').getByText('Profesional de ETF', { exact: true }).waitFor();
  await usuario.locator('.perfiles-persona').getByText('Consulta', { exact: true }).waitFor();
  assert.equal(await usuario.getByRole('navigation', { name: 'Principal' }).count(), 0);
  const acceso = await (await usuario.request.get(`${base}/api/sesion`)).json();
  assert.deepEqual(acceso.perfiles.sort(), ['consulta', 'profesional_etf']); assert.equal(acceso.equipo_id, opciones[0]);
  assert.equal(acceso.usuario.nombre, 'Inés Moreno Castillo'); assert.deepEqual(acceso.pantallas, []);
  const accesoAdmin = await (await admin.request.get(`${base}/api/sesion`)).json();
  const listaAdmin = await (await admin.request.get(`${base}/api/admin/usuarios`)).json();
  const administradora = listaAdmin.find((u) => u.cuenta === 'marta.lopez');
  const cabeceras = { Origin: base, 'X-CSRF-Token': accesoAdmin.csrf };
  const permisoTemporal = await admin.request.put(`${base}/api/admin/usuarios/${administradora.id}/perfiles`, {
    headers: cabeceras, data: { version: administradora.version, estado: 'activo', perfiles: administradora.perfiles.filter((p) => p.perfil === 'administracion'), equipo: null },
  });
  assert.equal(permisoTemporal.status(), 200);
  assert.equal((await admin.request.get(`${base}/api/coordinacion/derivaciones/00000000-0000-4000-8000-000000000011`)).status(), 403);
  const restauracion = await admin.request.put(`${base}/api/admin/usuarios/${administradora.id}/perfiles`, {
    headers: cabeceras, data: { version: (await permisoTemporal.json()).version, estado: 'activo', perfiles: administradora.perfiles, equipo: null },
  }); assert.equal(restauracion.status(), 200);
  for (const ruta of ['/api/admin/usuarios', '/api/admin/catalogos/listas', '/api/admin/firmantes', '/api/admin/paquetes/idioma/es', '/api/coordinacion/derivaciones/00000000-0000-4000-8000-000000000011']) assert.equal((await usuario.request.get(base + ruta)).status(), 403);
  for (const ruta of ['/api/expedientes', '/api/memoria', '/api/spaf', '/api/comisiones']) assert.equal((await admin.request.get(base + ruta)).status(), 403);
  const denegada = await usuario.goto(`${base}/admin/catalogos`); assert.equal(denegada.status(), 403);
  await usuario.goto(base); await usuario.getByRole('heading', { name: 'Acceso confirmado' }).waitFor();

  // Recorrido con respuestas reales y una entrada anterior útil para probar Atrás.
  const catalogo = admin.waitForResponse((r) => new URL(r.url()).pathname === '/api/admin/catalogos/listas');
  await navegar(admin, 'Catálogos'); await admin.getByRole('heading', { name: 'Catálogos' }).waitFor(); assert.equal((await catalogo).status(), 200);
  await navegar(admin, 'Administración de idiomas y temas'); await admin.getByRole('heading', { name: 'Administración de idiomas y temas' }).waitFor();
  await navegar(admin, 'Usuarios y permisos'); await admin.getByRole('button', { name: 'Editar permisos de Inés Moreno Castillo (ines.moreno)' }).click();
  await admin.getByLabel('Consulta', { exact: true }).uncheck();
  await preferencias(admin); await admin.getByLabel('Idioma').selectOption('en');
  await admin.getByRole('button', { name: 'Seguir editando' }).click(); assert.equal(await admin.getByLabel('Idioma').inputValue(), 'es');
  assert.equal(await admin.getByLabel('Consulta', { exact: true }).isChecked(), false);
  await persona(admin); await admin.getByRole('button', { name: 'Cerrar sesión' }).click();
  await admin.getByRole('button', { name: 'Seguir editando' }).click(); assert.equal((await admin.request.get(`${base}/api/sesion`)).status(), 200);
  await navegar(admin, 'Catálogos'); await admin.getByRole('button', { name: 'Seguir editando' }).click(); assert.equal(new URL(admin.url()).pathname, '/admin/usuarios');
  await admin.goBack(); await admin.getByRole('button', { name: 'Seguir editando' }).click(); assert.equal(new URL(admin.url()).pathname, '/admin/usuarios');
  await admin.getByLabel('Consulta', { exact: true }).check(); await guardar(admin);
  await admin.goBack(); await admin.getByRole('heading', { name: 'Administración de idiomas y temas' }).waitFor();
  await admin.goForward(); await admin.getByRole('heading', { name: 'Usuarios y permisos' }).waitFor();
  await admin.getByRole('button', { name: 'Editar permisos de Inés Moreno Castillo (ines.moreno)' }).click();
  await admin.getByLabel('Equipo', { exact: true }).selectOption(opciones[1]); await guardar(admin);
  assert.equal((await (await usuario.request.get(`${base}/api/sesion`)).json()).equipo_id, opciones[1]);
  await preferencias(admin); await admin.getByLabel('Idioma').selectOption('en'); await admin.getByRole('heading', { name: 'Users and permissions' }).waitFor();
  await preferencias(admin); await admin.getByLabel('Theme').selectOption('alto-contraste'); await admin.keyboard.press('Escape');
  await admin.locator('.persona-conectada > summary').focus(); await admin.keyboard.press('Escape');
  await admin.waitForFunction(() => [...document.querySelectorAll('link')].some((e) => e.href.endsWith('/alto-contraste/tokens')));
  for (const width of [390, 256]) {
    await admin.setViewportSize({ width, height: 900 }); await admin.waitForFunction(() => document.querySelector('.cabecera-app').dataset.compacto === 'true');
    assert.equal(await admin.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    assert.equal(await admin.locator('.lateral-app').isVisible(), false);
    await admin.screenshot({ path: join(evidencia, `usuarios-${width}-en-contraste.png`), fullPage: true });
  }
  assert.equal(await admin.evaluate(() => localStorage.length + sessionStorage.length), 0);
  await admin.getByRole('button', { name: 'Edit permissions for Inés Moreno Castillo (ines.moreno)' }).click();
  await admin.getByLabel('Deactivate access').check(); await guardar(admin, 'en');
  assert.equal((await usuario.request.get(`${base}/api/sesion`)).status(), 403); assert.equal((await usuario.request.get(`${base}/api/admin/usuarios`)).status(), 401);
  await usuario.reload(); await usuario.getByRole('button', { name: 'Entrar', exact: true }).waitFor();
  await admin.setViewportSize({ width: 1280, height: 900 });
  await admin.getByRole('button', { name: 'Edit permissions for Marta López Rivas (marta.lopez)' }).click();
  await admin.getByRole('checkbox', { name: 'Functional administration', exact: true }).uncheck();
  const retirada = admin.waitForResponse((r) => r.request().method() === 'PUT' && r.url().endsWith('/perfiles'));
  await admin.getByRole('button', { name: 'Save permissions' }).click(); assert.equal((await retirada).status(), 200);
  await admin.getByRole('heading', { name: 'National minimum wage' }).waitFor();
  assert.equal(await admin.getByRole('link', { name: 'Users and permissions' }).count(), 0);
  assert.equal(await admin.getByRole('table', { name: 'Users and permissions' }).count(), 0);
  assert.equal((await admin.request.get(`${base}/api/admin/usuarios`)).status(), 403);
  assert.deepEqual(errores, []);
  const contextoEntrada = await browser.newContext({ ignoreHTTPSErrors: true }); const directa = await contextoEntrada.newPage();
  assert.equal((await directa.goto(`${base}/admin/catalogos`)).status(), 401);
  await entrar(directa, 'marta.lopez'); await directa.getByRole('heading', { name: 'Salario mínimo interprofesional' }).waitFor();
  assert.equal(await directa.locator('h1').count(), 1); assert.equal(await directa.getByRole('link', { name: 'Catálogos' }).count(), 0);
  await contextoEntrada.close(); await contexto.tracing.stop({ path: join(evidencia, 'perfiles-recorrido.zip') });
  console.log('PERFILES-CHROME-OK: entrar → Usuarios → Catálogos → Paquetes; AD/PostgreSQL reales, nombres del personal, perfiles/equipos, traslado, baja, denegaciones, guardia, historial y móvil cerrado');
  console.log(`EVIDENCIA ${evidencia}`);
} finally { await browser.close(); }
