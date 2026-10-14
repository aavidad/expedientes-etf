// Consulta se valida con sesión real; las respuestas de error se ensayan en el mismo navegador.
import { chromium } from '../node_modules/playwright-core/index.mjs';
import { readFile, mkdir } from 'node:fs/promises';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
const raiz = fileURLToPath(new URL('../../', import.meta.url));
const base = `http://127.0.0.1:${(await readFile(raiz + 'despliegue/demo/.local/puerto', 'utf8')).trim()}`;
const claves = JSON.parse(await readFile(raiz + 'despliegue/demo/.local/usuarios.json'));
const textos = Object.assign({}, ...await Promise.all(['memoria', 'memoria_pantalla', 'consulta_memoria'].map(async n => JSON.parse(await readFile(`${raiz}idiomas/es/textos/${n}.json`)))));
const t = k => textos[k];
await mkdir(raiz + 'docs/capturas', { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
async function entrar(cuenta) {
  const contexto = await navegador.newContext({ viewport: { width: 1024, height: 1000 } });
  const pagina = await contexto.newPage();
  await contexto.route('**/*', r => new URL(r.request().url()).origin === base ? r.continue() : r.abort());
  await pagina.goto(base + '/entrar'); await pagina.locator('#cuenta-ad').fill(cuenta); await pagina.locator('#contrasena-ad').fill(claves[cuenta]); await pagina.locator('button[type=submit]').click(); await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return { contexto, pagina };
}
async function pedir(pagina, ruta, method = 'GET', datos) {
  return pagina.evaluate(async ({ ruta, method, datos }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const r = await fetch(ruta, { method, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': crypto.randomUUID() }, body: datos === undefined ? undefined : JSON.stringify(datos) });
    return { estado: r.status, datos: await r.json() };
  }, { ruta, method, datos });
}
try {
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 } });
  const pagina = await contexto.newPage();
  const errores = [], llamadas = [];
  pagina.on('pageerror', e => errores.push(e.message));
  pagina.on('request', r => { if (new URL(r.url()).pathname.startsWith('/api/')) llamadas.push(new URL(r.url()).pathname); });
  await contexto.route('**/*', r => new URL(r.request().url()).origin === base ? r.continue() : r.abort());
  await pagina.goto(base + '/entrar'); await pagina.locator('#cuenta-ad').fill('lucia.navarro'); await pagina.locator('#contrasena-ad').fill(claves['lucia.navarro']); await pagina.locator('button[type=submit]').click(); await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  await pagina.goto(base + '/memoria'); await pagina.locator('table').first().waitFor();
  const datos = await pagina.evaluate(async () => { const r = await fetch('/api/consulta/memoria/2026'); if (!r.ok) throw new Error(String(r.status)); return r.json(); });
  assert.equal(datos.apartados.length, 33); assert(datos.fecha_cifras); assert.equal(typeof datos.definitivas, 'boolean');
  await pagina.getByText(t(datos.definitivas ? 'memoria.consulta_definitiva' : 'memoria.consulta_provisional'), { exact: true }).waitFor();
  assert((await pagina.getByText(t('memoria.consulta_fecha').split('{fecha}')[0], { exact: false }).innerText()).includes('2026'));
  await pagina.getByRole('button', { name: t('memoria.bloque.1'), exact: true }).click();
  assert.deepEqual(await pagina.locator('.memoria-apartado').evaluateAll(nodos => nodos.slice(0, 4).map(n => n.id)), ['memoria-apartado-7', 'memoria-apartado-8', 'memoria-apartado-9', 'memoria-apartado-10']);
  await pagina.getByRole('button', { name: t('memoria.bloque.0'), exact: true }).click();
  assert.equal(await pagina.locator('textarea').count(), 0);
  for (const clave of ['memoria.guardar', 'memoria.cerrar', 'memoria.reabrir']) assert.equal(await pagina.getByRole('button', { name: t(clave), exact: true }).count(), 0);
  assert(!llamadas.some(r => /narrativas|instantanea|discrepancias|composicion|\/cifras\//.test(r)));
  await pagina.screenshot({ path: raiz + 'docs/capturas/memoria-consulta-escritorio.png', fullPage: true });
  const equipo = await pagina.locator('#memoria-equipo option').nth(1).getAttribute('value');
  await pagina.locator('#memoria-equipo').selectOption(equipo);
  const equipoRespuesta = pagina.waitForResponse(r => new URL(r.url()).pathname === '/api/consulta/memoria/2026');
  await pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(); assert.equal((await equipoRespuesta).status(), 200);
  const descarga = pagina.waitForEvent('download'); await pagina.getByRole('button', { name: t('memoria.descargar'), exact: true }).click(); assert((await descarga).suggestedFilename().endsWith('.docx'));
  await pagina.getByRole('status').filter({ hasText: t('memoria.descargada') }).waitFor();
  for (const width of [1024, 390, 320]) {
    await pagina.setViewportSize({ width, height: 950 });
    await pagina.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 1);
    const medidas = await pagina.evaluate(() => ({ width: innerWidth, scroll: document.documentElement.scrollWidth, elementos: [...document.querySelectorAll('body *')].filter(e => e.getBoundingClientRect().right > innerWidth + 1).filter(e => !e.closest('.memoria-tabla')).slice(0, 20).map(e => [e.tagName, e.className, Math.round(e.getBoundingClientRect().right)]) }));
    assert(medidas.scroll <= width + 1, JSON.stringify(medidas));
    if (width === 390) await pagina.screenshot({ path: raiz + 'docs/capturas/memoria-consulta-movil.png', fullPage: true });
  }
  await pagina.setViewportSize({ width: 1024, height: 1000 });
  await pagina.locator('#memoria-anio').focus(); await pagina.keyboard.press('Tab'); assert.equal(await pagina.evaluate(() => document.activeElement.id), 'memoria-equipo');
  const persona = pagina.locator('.persona-conectada'); if (!await persona.evaluate(e => e.open)) await persona.locator(':scope > summary').click();
  const preferencias = pagina.locator('.panel-persona details'); await preferencias.locator(':scope > summary').click();
  await pagina.locator('#tema').selectOption('alto-contraste');
  await pagina.waitForFunction(() => !!document.querySelector('link[href="/api/temas/alto-contraste/tokens"]')?.sheet);
  if (await persona.evaluate(e => e.open)) await persona.locator(':scope > summary').click();
  await pagina.screenshot({ path: raiz + 'docs/capturas/memoria-consulta-contraste.png', fullPage: true });
  const coordinadora = await entrar('ines.moreno');
  const ambito = `/api/estadistica/memoria/estado?anio=2026&etf=${equipo}`;
  const actual = await pedir(coordinadora.pagina, ambito); assert.equal(actual.estado, 200);
  const cierre = await pedir(coordinadora.pagina, ambito.replace('/estado?', '/cerrar?'), 'POST', { version: actual.datos.version }); assert.equal(cierre.estado, 200);
  try {
    await pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(); await pagina.getByText(t('memoria.consulta_definitiva'), { exact: true }).waitFor();
    const foto = await pedir(coordinadora.pagina, `/api/estadistica/memoria/instantanea?anio=2026&etf=${equipo}&revision=${cierre.datos.revision_actual}`); assert.equal(foto.estado, 200);
    const cerradas = await pedir(pagina, `/api/consulta/memoria/2026?equipo=${equipo}`); assert.equal(cerradas.datos.definitivas, true); assert.equal(cerradas.datos.fecha_cifras, foto.datos.cerrada_en);
  } finally {
    const reapertura = await pedir(coordinadora.pagina, ambito.replace('/estado?', '/reabrir?'), 'POST', { version: cierre.datos.version, motivo: 'Se revisarán las cifras de seguimiento antes del cierre definitivo.' }); assert.equal(reapertura.estado, 200); await coordinadora.contexto.close();
  }
  await contexto.route('**/api/consulta/memoria/2026*', r => r.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: { clave: 'estadistica.memoria_no_disponible' } }) }));
  await pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(); await pagina.getByRole('alert').waitFor(); assert.equal(await pagina.locator('table').count(), 0);
  await contexto.unroute('**/api/consulta/memoria/2026*');
  await pagina.getByRole('button', { name: t('memoria.reintentar'), exact: true }).click(); await pagina.locator('table').first().waitFor();
  await contexto.route('**/api/consulta/memoria/2026/word*', r => r.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: { clave: 'estadistica.memoria_no_disponible' } }) }));
  await pagina.getByRole('button', { name: t('memoria.descargar'), exact: true }).click(); await pagina.getByRole('alert').waitFor();
  await contexto.unroute('**/api/consulta/memoria/2026/word*');
  const repetida = pagina.waitForEvent('download'); await pagina.getByRole('button', { name: t('memoria.reintentar'), exact: true }).click(); await repetida;
  await pagina.getByRole('status').filter({ hasText: t('memoria.descargada') }).waitFor();
  assert.deepEqual(errores, []);
  const admin = await entrar('elena.salas');
  const original = (await pedir(admin.pagina, '/api/admin/usuarios')).datos.find(u => u.cuenta === 'marta.cabrera');
  const rutaUsuario = `/api/admin/usuarios/${original.id}/perfiles`;
  const conceder = await pedir(admin.pagina, rutaUsuario, 'PUT', { version: original.version, estado: original.estado, equipo: original.equipo, perfiles: [...original.perfiles, { perfil: 'consulta', desde: new Date().toISOString(), hasta: null }] }); assert.equal(conceder.estado, 200);
  try {
    const mixta = await entrar('marta.cabrera'); const erroresMixta = []; mixta.pagina.on('pageerror', e => erroresMixta.push(e.message));
    try {
      await mixta.pagina.goto(base + '/memoria'); await mixta.pagina.locator('#memoria-texto-d').waitFor();
      assert.equal(await mixta.pagina.locator('#memoria-equipo').inputValue(), original.equipo.equipo_id);
      await mixta.pagina.locator('#memoria-equipo').selectOption(''); await mixta.pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(); await mixta.pagina.getByText(t('memoria.consulta_provisional'), { exact: true }).waitFor(); assert.equal(await mixta.pagina.locator('textarea').count(), 0);
      const otro = await mixta.pagina.locator('#memoria-equipo option').evaluateAll(opciones => opciones.map(o => o.value));
      await mixta.pagina.locator('#memoria-equipo').selectOption(otro.find(id => id && id !== original.equipo.equipo_id)); await mixta.pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(); await mixta.pagina.locator('table').first().waitFor(); assert.equal(await mixta.pagina.locator('textarea').count(), 0);
      await mixta.pagina.locator('#memoria-equipo').selectOption(original.equipo.equipo_id); await mixta.pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(); await mixta.pagina.locator('#memoria-texto-d').waitFor(); assert.deepEqual(erroresMixta, []);
    } finally { await mixta.contexto.close(); }
  } finally {
    const vigente = (await pedir(admin.pagina, '/api/admin/usuarios')).datos.find(u => u.id === original.id);
    const restaurar = await pedir(admin.pagina, rutaUsuario, 'PUT', { version: vigente.version, estado: original.estado, equipo: original.equipo, perfiles: original.perfiles }); assert.equal(restaurar.estado, 200); await admin.contexto.close();
  }
  if (!await persona.evaluate(e => e.open)) await persona.locator(':scope > summary').click();
  if (!await preferencias.evaluate(e => e.open)) await preferencias.locator(':scope > summary').click();
  await pagina.locator('#idioma').selectOption('en');
  const ingles = JSON.parse(await readFile(raiz + 'idiomas/en/textos/memoria.json'));
  await pagina.getByRole('heading', { name: ingles['memoria.titulo'], exact: true }).waitFor();
  if (await persona.evaluate(e => e.open)) await persona.locator(':scope > summary').click();
  const wordIngles = pagina.waitForResponse(r => new URL(r.url()).pathname === '/api/consulta/memoria/2026/word'); const archivoIngles = pagina.waitForEvent('download');
  await pagina.getByRole('button', { name: ingles['memoria.descargar'], exact: true }).click(); assert.equal((await wordIngles).request().postDataJSON().idioma, 'en'); await archivoIngles;
  await pagina.setViewportSize({ width: 320, height: 950 }); await pagina.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 1); assert.deepEqual(errores, []);
  console.log('F4.07-CONSULTA-VISTA-OK: API real; 33 códigos ordenados; fecha y estado; equipo/provincia; Word es/en; sin narrativas ni composición; teclado; 1024/390/320; contraste; cierre/reapertura reales; errores de lectura/Word y recuperación; Equipo+Consulta cambia de ámbito sin narrativas ajenas.');
  await contexto.close();
} finally { await navegador.close(); }
