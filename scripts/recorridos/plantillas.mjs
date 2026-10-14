import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const base = process.argv[2] ?? 'http://127.0.0.1:18231';
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];
const capturas = new URL('../../docs/capturas/', import.meta.url);
await mkdir(capturas, { recursive: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 } });
const p = await contexto.newPage(); p.setDefaultTimeout(20000); p.on('pageerror', e => errores.push(e.message));
const plantilla = new URL('../../idiomas/es/plantillas/entrada/aceptacion/v1.1.0.docx', import.meta.url);
const buena = await readFile(plantilla);
const desconocida = execFileSync('python3', ['-c', `import sys,zipfile,io
z=zipfile.ZipFile(sys.argv[1]); b=io.BytesIO()
with zipfile.ZipFile(b,'w',zipfile.ZIP_DEFLATED) as out:
 for f in z.infolist():
  contenido=z.read(f.filename)
  if f.filename=='word/document.xml': contenido=contenido.replace(b'{{numero}}',b'{{intruso}}')
  out.writestr(f,contenido)
sys.stdout.buffer.write(b.getvalue())`, plantilla.pathname]);
const tipo = 'application/vnd.openxmlformats-officedocument.wordprocessingml.document';
async function entrar(cuenta) {
  await p.goto(base); await p.locator('#cuenta-ad').fill(cuenta); await p.locator('#contrasena-ad').fill(claves[cuenta]);
  await p.locator('button[type=submit]').click(); await p.locator('#cuenta-ad').waitFor({ state: 'detached' });
}
async function listar() { return p.evaluate(async () => (await fetch('/api/admin/plantillas')).json()); }
async function comprobar() {
  assert.equal(await p.locator('h1').count(), 1);
  assert.ok(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
  assert.equal(await p.evaluate(() => localStorage.length + sessionStorage.length), 0);
  for (const control of await p.locator('main button, main input, main select, main summary').all()) {
    if (await control.isVisible()) assert.ok(await control.evaluate(e => e.getBoundingClientRect().height >= 44));
  }
}
try {
  await entrar('ines.moreno');
  assert.equal(await p.getByRole('link', { name: 'Plantillas Word', exact: true }).count(), 0);
  for (const ruta of ['/api/admin/plantillas', '/admin/plantillas']) {
    assert.equal(await p.evaluate(async ruta => (await fetch(ruta)).status, ruta), 403);
  }
  await p.locator('.persona-conectada > summary').click();
  await p.getByRole('button', { name: 'Cerrar sesión', exact: true }).click();
  await entrar('elena.salas');
  await p.getByRole('link', { name: 'Plantillas Word', exact: true }).click();
  await p.locator('.plantillas-tarjeta').first().waitFor();
  await comprobar();
  const original = (await listar()).find(v => v.activa && v.plantilla.Tipo === 'entrada/aceptacion' && v.plantilla.Idioma === 'es'); assert(original);
  let minor = 0; const previas = await listar(); while (previas.some(v => v.plantilla.Version === `2.${minor}.0`)) minor++; const version = `2.${minor}.0`;
  await p.getByRole('button', { name: 'Subir una versión', exact: true }).click();
  await p.locator('#plantillas-tipo').selectOption('entrada/aceptacion'); await p.locator('#plantillas-idioma').selectOption('es');
  await p.locator('#plantillas-version').fill(version);
  await p.locator('#plantillas-archivo').setInputFiles({ name: 'acta-marcadores.docx', mimeType: tipo, buffer: desconocida });
  const rechazo = p.waitForResponse(r => r.url().includes('/api/admin/plantillas/versiones') && r.request().method() === 'POST');
  await p.getByRole('button', { name: 'Subir y revisar versión', exact: true }).click(); assert.equal((await rechazo).status(), 422);
  await p.getByText('Falta el marcador «numero». Añádelo al Word antes de volver a subirlo.', { exact: true }).waitFor();
  await p.getByText('Sobra el marcador «intruso». Retíralo o corrígelo antes de volver a subir el Word.', { exact: true }).waitFor();
  assert.equal((await listar()).find(v => v.activa && v.plantilla.Tipo === 'entrada/aceptacion' && v.plantilla.Idioma === 'es').plantilla.ID, original.plantilla.ID);
  assert.equal(await p.locator('#plantillas-version').inputValue(), version);
  assert.equal(await p.evaluate(() => document.activeElement.id), 'plantillas-error');
  await p.locator('#plantillas-archivo').setInputFiles({ name: 'acta-aceptacion.docx', mimeType: tipo, buffer: buena });
  const subida = p.waitForResponse(r => r.url().includes('/api/admin/plantillas/versiones') && r.request().method() === 'POST');
  await p.getByRole('button', { name: 'Subir y revisar versión', exact: true }).click(); assert.equal((await subida).status(), 201);
  await p.getByText('Versión guardada. Ha superado el antivirus y la revisión de marcadores. Está pendiente de activar.', { exact: true }).waitFor();
  const versiones = await listar(); const nueva = versiones.find(v => v.plantilla.Version === version); assert(nueva && !nueva.activa && nueva.antivirus === 'limpio');
  assert.equal(versiones.find(v => v.activa && v.plantilla.Tipo === 'entrada/aceptacion' && v.plantilla.Idioma === 'es').plantilla.ID, original.plantilla.ID);
  await p.getByRole('button', { name: 'Cerrar subida', exact: true }).click();
  const tarjeta = p.locator('.plantillas-tarjeta').filter({ has: p.getByRole('heading', { name: 'Acta de Aceptación', exact: true }) }).filter({ has: p.getByText('español', { exact: true }) });
  await tarjeta.locator('summary').click();
  await tarjeta.getByRole('button', { name: `Activar versión ${version}`, exact: true }).click();
  await p.getByRole('dialog').waitFor();
  await p.keyboard.press('Escape'); assert.equal(await p.getByRole('dialog').isVisible(), false);
  assert.equal(await p.evaluate(() => document.activeElement.textContent), `Activar versión ${version}`);
  await p.keyboard.press('Enter'); await p.getByRole('dialog').waitFor();
  await p.getByRole('button', { name: 'Confirmar activación', exact: true }).click();
  await p.getByText('Versión activada. Los próximos documentos usarán esta plantilla.', { exact: true }).waitFor();
  const activada = (await listar()).find(v => v.plantilla.ID === nueva.plantilla.ID); assert(activada.activa && activada.activaciones.length === 1);
  assert.ok(activada.activaciones[0].por && activada.activaciones[0].en);
  await tarjeta.getByText(/Elena Salas/, { exact: false }).first().waitFor();
  // Las capturas acreditan la pantalla activa y la autoría, sin cambiar datos reales.
  for (const ancho of [1440, 1024, 390, 360]) {
    await p.setViewportSize({ width: ancho, height: 1000 }); await p.evaluate(() => scrollTo(0, 0)); await comprobar();
    if (ancho === 1440 || ancho === 390) await p.screenshot({ path: new URL(ancho === 1440 ? 'plantillas-escritorio.png' : 'plantillas-movil.png', capturas).pathname });
  }
  // Conflicto real: la revisión anterior no activa otra versión.
  const conflicto = await p.evaluate(async ({ id, previa }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const r = await fetch(`/api/admin/plantillas/${id}/activar`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf }, body: JSON.stringify({ idioma: 'es', version_previa: previa }) });
    return { estado: r.status, cuerpo: await r.json() };
  }, { id: original.plantilla.ID, previa: original.version_actual }); assert.equal(conflicto.estado, 409);
  // Restaura la versión de distribución y conserva las dos activaciones.
  await p.setViewportSize({ width: 1440, height: 1000 });
  await tarjeta.getByRole('button', { name: `Activar versión ${original.plantilla.Version}`, exact: true }).click();
  await p.getByRole('button', { name: 'Confirmar activación', exact: true }).click();
  await p.getByText('Versión activada. Los próximos documentos usarán esta plantilla.', { exact: true }).waitFor();
  assert.equal((await listar()).find(v => v.activa && v.plantilla.Tipo === 'entrada/aceptacion' && v.plantilla.Idioma === 'es').plantilla.ID, original.plantilla.ID);
  // Las preferencias son recursos reales del servidor.
  await p.locator('.persona-conectada > summary').click();
  await p.locator('.preferencias > summary').click();
  await p.getByLabel('Tema', { exact: true }).selectOption('alto-contraste'); await p.keyboard.press('Escape');
  await comprobar(); await p.screenshot({ path: new URL('plantillas-contraste-escritorio.png', capturas).pathname });
  await p.setViewportSize({ width: 390, height: 1000 }); await comprobar(); await p.screenshot({ path: new URL('plantillas-contraste-movil.png', capturas).pathname });
  await p.locator('.persona-conectada > summary').click();
  if (!await p.locator('.preferencias').evaluate(e => e.open)) await p.locator('.preferencias > summary').click();
  await p.getByLabel('Idioma', { exact: true }).selectOption('en');
  await p.getByRole('heading', { name: 'Word templates', exact: true }).waitFor(); await comprobar();
  await p.keyboard.press('Escape');
  // Estados de lectura mediante respuestas simuladas; las escrituras anteriores fueron reales.
  let liberar; const espera = new Promise(resolve => { liberar = resolve; });
  await p.route('**/api/admin/plantillas', async ruta => { await espera; await ruta.fulfill({ status: 200, contentType: 'application/json', body: '[]' }); });
  const recarga = p.reload(); await p.getByText('Cargando plantillas e historial…', { exact: true }).waitFor(); liberar(); await recarga;
  await p.getByText('No hay plantillas disponibles. Pide a Informática que instale las plantillas iniciales.', { exact: true }).waitFor();
  await p.unroute('**/api/admin/plantillas');
  await p.route('**/api/admin/plantillas', ruta => ruta.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: 'documento.no_disponible' }) }));
  await p.reload(); await p.getByText('No hemos podido completar la operación. Actualiza la lista para comprobar su estado antes de volver a intentarlo.', { exact: true }).waitFor();
  await p.unroute('**/api/admin/plantillas'); await p.getByRole('button', { name: 'Actualizar lista', exact: true }).click(); await p.locator('.plantillas-tarjeta').first().waitFor();
  assert.deepEqual(errores, []);
  console.log('PL1-OK: Coordinación denegada; subida inválida con marcadores; subida válida sin activar; confirmación por teclado; autor y fecha; conflicto 409; restauración; 1440/1024/390/360; alto contraste e inglés; sin errores de navegador.');
} finally { await contexto.close(); await navegador.close(); }
