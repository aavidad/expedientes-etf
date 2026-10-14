// Lee, revisa y confirma las muestras sanitizadas en la demostración local.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const base = process.argv[2];
assert.match(base ?? '', /^http:\/\/127\.0\.0\.1:\d+$/);
const textos = Object.assign({}, ...await Promise.all(['identidad', 'anexo', 'decision'].map(async n =>
  JSON.parse(await readFile(new URL(`../../idiomas/es/textos/${n}.json`, import.meta.url))))));
const cuentas = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'es-ES', serviceWorkers: 'block' });
await context.route('**/*', route => {
  const url = new URL(route.request().url());
  return url.origin === base || ['data:', 'blob:'].includes(url.protocol) ? route.continue() : route.abort();
});
const page = await context.newPage(); page.setDefaultTimeout(30000);
const errores = []; page.on('pageerror', e => errores.push(e.message));
const salida = new URL('../../docs/capturas/', import.meta.url); await mkdir(salida, { recursive: true });
try {
  await page.goto(`${base}/entrar`);
  await page.getByLabel(textos['identidad.cuenta'], { exact: true }).fill('ines.moreno');
  await page.getByLabel(textos['identidad.contrasena'], { exact: true }).fill(cuentas['ines.moreno']);
  const entrada = page.waitForResponse(r => new URL(r.url()).pathname === '/api/sesion' && r.request().method() === 'POST');
  await page.getByRole('button', { name: textos['identidad.entrar'], exact: true }).click();
  assert.equal((await entrada).status(), 200);
  for (const [nombre,apellido] of [['anexo3.odt', 'Yosua Pérez González'], ['anexo4-firmado.pdf', 'Férdinant López Gutiérrez']]) {
    await page.setViewportSize({ width: 1440, height: 1000 });
    await page.goto(`${base}/coordinacion/anexos/nuevo`);
    await page.locator('#anexo-fichero').setInputFiles(new URL(`../../internal/adaptadores/anexos/odt/testdata/${nombre}`, import.meta.url).pathname);
    const respuesta = page.waitForResponse(r => new URL(r.url()).pathname === '/api/coordinacion/derivaciones' && r.request().method() === 'POST');
    await page.getByRole('button', { name: textos['anexo.subir'], exact: true }).click();
    const r = await respuesta; assert.equal(r.status(), 200, await r.text());
    const analisis = await r.json();
    assert.equal(analisis.lectura_anexo.antivirus, 'limpio');
    const campos = analisis.derivacion_datos.contenido.campos;
    assert.equal(campos.filter(c => c.codigo === 'menores.persona.nombre').length, 2);
    assert(campos.some(c => c.codigo === 'menores.persona.nombre' && c.valor === apellido));
    assert(campos.some(c => c.codigo === 'sociodemografia.tramo_ingresos' && c.valor.includes('SMI')));
    await page.getByRole('heading', { name: textos['anexo.revisar'], exact: true }).waitFor();
    const bloque = page.locator('details.anexo-tarjeta').filter({ has: page.locator('summary', { hasText: textos['anexo.apartado.menores'] }) });
    await bloque.evaluate(n => { n.open = true; n.scrollIntoView({ block: 'start' }); });
    assert(await bloque.locator('textarea').filter({ hasNotText: /^$/ }).count() > 0);
    assert(await page.locator('textarea').evaluateAll((ns, esperado) => ns.some(n => n.value === esperado), apellido));
    const id = nombre.endsWith('.odt') ? 'odt' : 'pdf-firmado';
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 });
      await page.evaluate(async () => { await document.fonts.ready; await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))); });
      await bloque.evaluate(n => window.scrollTo({ top: n.getBoundingClientRect().top + window.scrollY, behavior: 'instant' }));
      await page.waitForTimeout(150);
      assert(Math.abs((await bloque.boundingBox()).y) <= 1, 'El inicio del apartado debe verse en la captura');
      await page.screenshot({ path: new URL(`resp4-${id}-${width}.png`, salida).pathname });
      assert(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1));
    }
    for (const campo of ['municipio_id', 'centro_id', 'equipo_id']) {
      const selector = page.locator(`#anexo-${campo}`);
      if (!(await selector.inputValue())) {
        const valor = await selector.locator('option').evaluateAll(ns => ns.find(n => n.value)?.value);
        assert(valor); await selector.selectOption(valor);
      }
    }
    const lecturaPersistida = page.waitForResponse(r => /^\/api\/coordinacion\/derivaciones\/[a-f0-9-]{36}$/.test(new URL(r.url()).pathname) && r.request().method() === 'GET');
    const confirmacion = page.waitForResponse(r => new URL(r.url()).pathname === '/api/coordinacion/derivaciones/confirmar' && r.request().method() === 'POST');
    await page.getByRole('button', { name: textos['anexo.confirmar'], exact: true }).click();
    const creada = await confirmacion; assert.equal(creada.status(), 201, await creada.text());
    const recibo = await creada.json(); assert.equal(recibo.estado, 'derivacion_recibida');
    const ruta = `/api/coordinacion/derivaciones/${recibo.id}`;
    await page.waitForURL(`${base}/coordinacion/derivaciones/${recibo.id}`);
    const consulta = await lecturaPersistida; assert.equal(new URL(consulta.url()).pathname, ruta); assert.equal(consulta.status(), 200);
    const guardada = await consulta.json();
    assert.equal(guardada.lectura_anexo.formato, nombre.endsWith('.odt') ? 'odt' : 'pdf');
    assert.equal(guardada.lectura_anexo.id, analisis.lectura_anexo.id);
    assert(guardada.derivacion_datos.contenido.campos.some(c => c.codigo === 'menores.persona.nombre' && c.valor === apellido));
    const recuperada = page.waitForResponse(r => new URL(r.url()).pathname === ruta && r.request().method() === 'GET');
    await page.reload();
    assert.equal((await recuperada).status(), 200);
    await page.getByRole('heading', { name: textos['decision.titulo'], exact: true }).waitFor();
    console.log(`RESP4 ${nombre}: HTTP 200/201; antivirus limpio; revisión con dos menores; derivación confirmada y recuperada; formato original conservado; capturas 1440/390.`);
  }
  assert.deepEqual(errores, []);
} finally { await context.close(); await browser.close(); }
