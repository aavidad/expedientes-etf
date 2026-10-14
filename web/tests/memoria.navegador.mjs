// Recorrido real con datos inventados, Chrome del sistema y la demo local.
import { chromium } from '../node_modules/playwright-core/index.mjs';
import { readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const raiz = fileURLToPath(new URL('../../', import.meta.url));
const puerto = (await readFile(raiz + 'despliegue/demo/.local/puerto', 'utf8')).trim();
const base = `http://127.0.0.1:${puerto}`;
const claves = JSON.parse(await readFile(raiz + 'despliegue/demo/.local/usuarios.json', 'utf8'));
const textos = { ...JSON.parse(await readFile(raiz + 'idiomas/es/textos/memoria.json')), ...JSON.parse(await readFile(raiz + 'idiomas/es/textos/memoria_pantalla.json')) };
Object.assign(textos, JSON.parse(await readFile(raiz + 'idiomas/es/textos/consulta_memoria.json')));
const t = k => textos[k];
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const resultados = [];
async function entrar(cuenta, width = 1440) {
  const contexto = await navegador.newContext({ viewport: { width, height: 1000 } });
  const pagina = await contexto.newPage();
  await pagina.goto(base + '/entrar');
  await pagina.locator('#cuenta-ad').fill(cuenta); await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
  await pagina.locator('button[type=submit]').click(); await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  await pagina.waitForLoadState('networkidle');
  return { contexto, pagina };
}
async function pedir(pagina, ruta, opciones = {}) {
  return pagina.evaluate(async ({ ruta, opciones }) => {
    const r = await fetch(ruta, { credentials: 'same-origin', ...opciones });
    return { estado: r.status, datos: await r.json() };
  }, { ruta, opciones });
}
const respuesta = (pagina, ruta, metodo = 'GET') => pagina.waitForResponse(r => new URL(r.url()).pathname === ruta && r.request().method() === metodo);
async function abrir(pagina) {
  const pendiente = respuesta(pagina, '/api/estadistica/memoria');
  await pagina.goto(base + '/memoria'); const r = await pendiente;
  assert.equal(r.status(), 200); await pagina.locator('#memoria-texto-d').waitFor();
  return r.json();
}
async function abrirPreferencias(pagina) {
  const persona = pagina.locator('.persona-conectada');
  if (!await persona.evaluate(el => el.open)) await persona.locator(':scope > summary').click();
  const preferencias = pagina.locator('.panel-persona details');
  await preferencias.locator(':scope > summary').waitFor({ state: 'visible' });
  if (!await preferencias.evaluate(el => el.open)) await preferencias.locator(':scope > summary').click();
}

try {
  const sinPublicar = await entrar('lucia.navarro');
  const alMomento=respuesta(sinPublicar.pagina,'/api/consulta/memoria/2026');await sinPublicar.pagina.goto(base+'/memoria');
  assert.equal((await alMomento).status(),200);await sinPublicar.pagina.getByText(t('memoria.consulta_provisional'),{exact:true}).waitFor();
  assert(await sinPublicar.pagina.locator('#memoria-equipo option').count()>1);
  resultados.push('Consulta al momento sin publicación previa: PASS');await sinPublicar.contexto.close();
  const { contexto, pagina } = await entrar('ines.moreno');
  const provincial = await abrir(pagina);
  assert.equal(provincial.equipo_id, ''); assert.equal(provincial.apartados.length, 33);
  const atendidos = provincial.apartados.find(a => a.codigo === '1');
  console.log('Población de la demo en Memoria:', JSON.stringify(atendidos.total));
  assert.equal(atendidos.total.familias, Number(process.env.ETF_MEMORIA_FAMILIAS ?? '9')); assert.equal(atendidos.total.menores, Number(process.env.ETF_MEMORIA_MENORES ?? '34'));
  assert(!JSON.stringify(provincial).includes('narrativas'));
  const sexoMenores = provincial.apartados.find(a => a.codigo === '2').celdas.filter(c => c.unidad === 'menores');
  assert.equal(sexoMenores.reduce((n, c) => n + c.numerador, 0), Number(process.env.ETF_MEMORIA_MENORES ?? '34'));
  for (const celda of sexoMenores.filter(c => c.numerador > 0)) {
    assert(await pagina.locator('#memoria-apartado-2 td').filter({ hasText: new RegExp(`^${celda.numerador}$`) }).count() > 0);
  }
  resultados.push('Menores de la demo: cifras conocidas por sexo visibles en el apartado 2 y coincidentes con los registros: PASS');

  const equipos = await pedir(pagina, '/api/estadistica/memoria/equipos'); assert.equal(equipos.estado, 200);
  const equipo = equipos.datos.find(e => e.codigo === (process.env.ETF_MEMORIA_EQUIPO ?? '04')); assert(equipo);
  await pagina.locator('#memoria-equipo').selectOption(equipo.id);
  const pendiente = respuesta(pagina, '/api/estadistica/memoria'); await pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(); assert.equal((await pendiente).status(), 200);
  await pagina.locator('#memoria-texto-d').waitFor();
  const anterior = await pedir(pagina, `/api/estadistica/memoria/narrativas?anio=${provincial.anio}&etf=${equipo.id}`);
  const frase = `El equipo revisó los acuerdos de escolarización y acompañó a las familias en las reuniones de seguimiento. Revisión del documento: ${anterior.datos.version + 1}.`;
  await pagina.locator('#memoria-texto-d').fill(frase);
  assert(await pagina.getByRole('button', { name: t('memoria.descargar'), exact: true }).isDisabled());
  const guardado = respuesta(pagina, '/api/estadistica/memoria/narrativas', 'PUT'); await pagina.getByRole('button', { name: t('memoria.guardar'), exact: true }).click(); assert.equal((await guardado).status(), 200);
  await pagina.getByText(t('memoria.guardada'), { exact: true }).waitFor();
  const n = await pedir(pagina, `/api/estadistica/memoria/narrativas?anio=${provincial.anio}&etf=${equipo.id}`); assert.equal(n.datos.d, frase);
  for (const recuperar of ['reintentar', 'consultar']) {
    let perdida;
    await pagina.route('**/api/estadistica/memoria/word', async ruta => { const r = await ruta.fetch(); perdida = { estado: r.status(), emision: r.headers()['x-emision-id'], clave: ruta.request().headers()['idempotency-key'] }; await ruta.abort('failed'); });
    await pagina.getByRole('button', { name: t('memoria.descargar'), exact: true }).click(); await pagina.getByRole('alert').waitFor(); assert.equal(perdida.estado, 201);
    await pagina.unroute('**/api/estadistica/memoria/word');
    if (recuperar === 'consultar') {
      const datos = respuesta(pagina, '/api/estadistica/memoria');
      await pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(); assert.equal((await datos).status(), 200);
      await pagina.locator('#memoria-texto-d').waitFor();
    }
    const recuperacion = respuesta(pagina, '/api/estadistica/memoria/word', 'POST'); const descarga = pagina.waitForEvent('download');
    await pagina.getByRole('button', { name: t(recuperar === 'reintentar' ? 'memoria.reintentar' : 'memoria.descargar'), exact: true }).click(); const repetida = await recuperacion;
    assert.equal(repetida.status(), 200); assert.equal(repetida.headers()['x-idempotencia-repetida'], 'true'); assert.equal(repetida.headers()['x-emision-id'], perdida.emision); assert.equal(repetida.request().headers()['idempotency-key'], perdida.clave);
    await (await descarga).saveAs(join(tmpdir(), 'f402-memoria.docx'));
  }
  resultados.push('Coordinación: provincial/equipo, 33 tablas de cálculo, narrativas persistidas y Word recuperado con la misma emisión: PASS');
  let cierrePerdido;
  await pagina.route('**/api/estadistica/memoria/cerrar?*', async ruta => {
    const r = await ruta.fetch(); cierrePerdido = { datos: await r.json(), clave: ruta.request().headers()['idempotency-key'] }; assert.equal(r.status(), 200); await ruta.abort('failed');
  });
  await pagina.getByRole('button', { name: t('memoria.cerrar'), exact: true }).click();
  await pagina.getByRole('dialog').getByRole('button', { name: t('memoria.confirmar_cierre'), exact: true }).click();
  await pagina.getByRole('alert').waitFor(); await pagina.unroute('**/api/estadistica/memoria/cerrar?*');
  const repetirCierre = respuesta(pagina, '/api/estadistica/memoria/cerrar', 'POST');
  await pagina.getByRole('button', { name: t('memoria.reintentar'), exact: true }).click(); const cierre = await repetirCierre;
  assert.equal(cierre.status(), 200); assert.equal(cierre.request().headers()['idempotency-key'], cierrePerdido.clave); assert.equal((await cierre.json()).repetida, true);
  await pagina.getByText(t('memoria.estado_cerrada'), { exact: true }).waitFor();
  assert(await pagina.locator('#memoria-texto-d').isDisabled());
  assert(await pagina.getByRole('button', { name: t('memoria.guardar'), exact: true }).isDisabled());
  const wordCerrado = respuesta(pagina, '/api/estadistica/memoria/word', 'POST'); const descargaCerrada = pagina.waitForEvent('download');
  await pagina.getByRole('button', { name: t('memoria.descargar'), exact: true }).click(); assert.equal((await wordCerrado).status(), 201); await descargaCerrada;
  await pagina.getByText(t('memoria.descargada'), { exact: true }).waitFor();
  assert(await pagina.getByText(t('memoria.estado_cerrada'), { exact: true }).isVisible());
  assert.equal((await pedir(pagina, `/api/estadistica/memoria/estado?anio=${provincial.anio}&etf=${equipo.id}`)).datos.revision_actual, cierrePerdido.datos.revision_actual);
  await pagina.screenshot({ path: raiz + 'docs/capturas/memoria-cerrada.png', fullPage: true });
  await pagina.getByRole('button', { name: t('memoria.reabrir'), exact: true }).click();
  assert(await pagina.getByRole('dialog').getByRole('button', { name: t('memoria.confirmar_reapertura'), exact: true }).isDisabled());
  await pagina.locator('#memoria-motivo').fill('Se revisarán los datos de seguimiento del año antes de cerrar la versión definitiva.');
  const reapertura = respuesta(pagina, '/api/estadistica/memoria/reabrir', 'POST');
  await pagina.getByRole('dialog').getByRole('button', { name: t('memoria.confirmar_reapertura'), exact: true }).click(); assert.equal((await reapertura).status(), 200);
  await pagina.getByText(t('memoria.borrador'), { exact: true }).waitFor(); assert(await pagina.locator('#memoria-texto-d').isEnabled());
  resultados.push('Cierre confirmado, respuesta perdida recuperada sin nueva revisión, fecha y foto bloqueada; reapertura con motivo: PASS');
  for (let i = 0; i < 4; i++) {
    if (i) await pagina.getByRole('navigation', { name: t('memoria.bloques'), exact: true }).getByRole('button', { name: t(`memoria.bloque.${i}`), exact: true }).click();
    await pagina.getByRole('heading', { name: t(`memoria.bloque.${i}`), exact: true }).waitFor();
    assert(await pagina.locator('table caption').count() > 0);
  }
  await pagina.getByRole('navigation', { name: t('memoria.bloques'), exact: true }).getByRole('button', { name: t('memoria.bloque.0'), exact: true }).click();
  assert((await pagina.locator('#memoria-apartado-5').innerText()).includes(t('memoria.poblacion.no_aceptados')));
  await pagina.evaluate(() => window.scrollTo(0, 0)); await pagina.screenshot({ path: raiz + 'docs/capturas/memoria-escritorio.png', fullPage: true });
  for (const width of [1024, 390, 360, 320]) {
    await pagina.setViewportSize({ width, height: 1000 });
    await pagina.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth, null, { timeout: 3000 });
    assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `desbordamiento ${width}`);
    assert(await pagina.evaluate(() => [...document.querySelectorAll('thead th:nth-child(3), thead th:nth-child(5)')].every(th => { const rango = document.createRange(); rango.selectNodeContents(th); return rango.getClientRects().length === 1; })), `palabras partidas en cabeceras ${width}`);
    await pagina.keyboard.press('Tab'); assert(await pagina.evaluate(() => document.activeElement !== document.body));
    if (width === 390) await pagina.screenshot({ path: raiz + 'docs/capturas/memoria-movil.png', fullPage: true });
  }
  resultados.push('Cuatro bloques, tablas, teclado y anchuras 1440/1024/390/360/320 sin desbordamiento de página: PASS');
  await pagina.setViewportSize({ width: 1440, height: 1000 });
  await abrirPreferencias(pagina);
  await pagina.locator('#tema').selectOption('alto-contraste');
  await pagina.waitForFunction(() => !!document.querySelector('link[href="/api/temas/alto-contraste/tokens"]')?.sheet);
  await pagina.keyboard.press('Escape'); await pagina.keyboard.press('Escape');
  await pagina.screenshot({ path: raiz + 'docs/capturas/memoria-alto-contraste.png', fullPage: true });
  await abrirPreferencias(pagina);
  await pagina.locator('#idioma').selectOption('en');
  await pagina.getByRole('heading', { name: 'Annual Technical Report', exact: true }).waitFor(); await pagina.keyboard.press('Escape'); await pagina.keyboard.press('Escape'); await pagina.setViewportSize({ width: 390, height: 1000 });
  await pagina.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth);
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth)); const ingles = JSON.parse(await readFile(raiz + 'idiomas/en/textos/memoria.json'));
  const wordIngles = respuesta(pagina, '/api/estadistica/memoria/word', 'POST'); const archivoIngles = pagina.waitForEvent('download');
  await pagina.getByRole('button', { name: ingles['memoria.descargar'], exact: true }).click();
  const docIngles = await wordIngles; assert.equal(docIngles.status(), 201); assert.equal(docIngles.request().postDataJSON().idioma, 'en');
  await (await archivoIngles).saveAs(join(tmpdir(), 'f402-memoria-en.docx'));
  resultados.push('Alto contraste, inglés y Word del idioma elegido: PASS');
  await contexto.close();
  const profesional = await entrar('marta.cabrera');
  const propio = await abrir(profesional.pagina); assert.equal(propio.equipo_id, equipos.datos.find(e => e.codigo === '02').id); assert.equal(await profesional.pagina.locator('#memoria-equipo').count(), 0);
  const ajeno = equipo; const prohibido = await pedir(profesional.pagina, `/api/estadistica/memoria?anio=${propio.anio}&etf=${ajeno.id}`); assert.equal(prohibido.estado, 403);
  resultados.push('Profesional: sólo su equipo; equipo ajeno 403: PASS'); await profesional.contexto.close();
  const consulta = await entrar('lucia.navarro');
  const rConsulta = respuesta(consulta.pagina, '/api/consulta/memoria/2026'); await consulta.pagina.goto(base + '/memoria'); const cifras = await rConsulta; assert.equal(cifras.status(), 200);
  const json = await cifras.json(); assert(!JSON.stringify(json).includes('narrativas')); assert(!JSON.stringify(json).includes(frase));
  assert.equal(await consulta.pagina.locator('textarea').count(), 0); assert.equal(await consulta.pagina.getByRole('button', { name: t('memoria.descargar'), exact: true }).count(), 1);
  const textosDenegados = await pedir(consulta.pagina, `/api/estadistica/memoria/narrativas?anio=${propio.anio}`); assert.equal(textosDenegados.estado, 403);
  assert.equal((await pedir(consulta.pagina, `/api/estadistica/memoria/estado?anio=${propio.anio}`)).estado, 403);
  resultados.push('Consulta: sólo cifras al momento o foto cerrada; sin narrativas ni composición; descarga numérica y narrativas 403: PASS'); await consulta.contexto.close();
  console.log(resultados.join('\n'));
  await writeFile(join(tmpdir(), 'f402-recorrido.txt'), resultados.join('\n') + '\n');
} finally { await navegador.close(); }
