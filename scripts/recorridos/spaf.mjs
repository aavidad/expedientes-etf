// Recorrido SPAF con datos inventados, servicios de demo y Chrome del sistema.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const origen = process.argv[2] ?? 'http://127.0.0.1:18273';
assert.match(origen, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
const pagina = await contexto.newPage(); pagina.setDefaultTimeout(20000);
const errores = []; pagina.on('pageerror', e => errores.push(e.message));
const capturas = new URL('../../docs/capturas/', import.meta.url);
await mkdir(capturas, { recursive: true });

async function entrar(cuenta) {
  await pagina.goto(`${origen}/entrar`);
  await pagina.getByLabel('Usuario de la Diputación').fill(cuenta);
  await pagina.getByLabel('Contraseña', { exact: true }).fill(claves[cuenta]);
  await pagina.getByRole('button', { name: 'Entrar', exact: true }).click();
  await pagina.locator('.persona-conectada').waitFor();
  await pagina.goto(`${origen}/equipo/tabla-spaf`);
  await pagina.locator('#spaf-mes').waitFor();
}
async function resumen(esperado) {
  const valores = esperado.map(String);
  // La siembra SPAF no dispone de recuentos de menores conocidos.
  if (esperado[2] > 0 && esperado[3] === 0) valores[3] = await pagina.evaluate(() => document.documentElement.lang === 'en' ? 'No data' : 'Sin dato');
  await pagina.waitForFunction(valores => JSON.stringify(Array.from(document.querySelectorAll('.spaf-indicadores strong'), e => e.textContent)) === JSON.stringify(valores), valores);
  assert.deepEqual(await pagina.locator('.spaf-indicadores strong').allTextContents(), valores);
}
async function ancho(w) {
  await pagina.setViewportSize({ width: w, height: 1000 });
  await pagina.waitForTimeout(150);
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), `desbordamiento a ${w}`);
}
async function preferencia(tipo, valor) {
  const persona = pagina.locator('.persona-conectada');
  if (!await persona.evaluate(e => e.open)) await persona.locator('summary').first().click();
  const preferencias = pagina.locator('.preferencias');
  if (!await preferencias.evaluate(e => e.open)) await preferencias.locator('summary').first().click();
  await pagina.getByLabel(tipo, { exact: true }).selectOption(valor);
}
try {
  await entrar('ines.moreno');
  const opciones = await pagina.evaluate(async () => (await fetch('/api/estadistica/spaf/opciones')).json());
  const equipo02 = opciones.equipos.find(e => e.clave_texto === 'equipo.02');
  const equipo01 = opciones.equipos.find(e => e.clave_texto === 'equipo.01');
  assert(equipo02 && equipo01);
  await pagina.getByLabel('Equipo', { exact: true }).selectOption(equipo02.id);
  await pagina.getByLabel('Mes', { exact: true }).fill('2026-09');
  await resumen([2, 2, 3, 0]);
  assert.equal(await pagina.locator('.spaf table thead th').count(), 14);
  assert.equal(await pagina.locator('.spaf table tbody tr').count(), 3);
  assert.equal(await pagina.locator('.spaf-movimientos li').count(), 4);
  assert.match(await pagina.locator('.spaf').innerText(), /Total incompleto: 3 casos sin dato/);
  assert(await pagina.getByText('Sin dato', { exact: true }).count() > 0);
  assert(!/equipo\.0|motivo_spaf\.|OBJ_TOTAL/.test(await pagina.locator('.spaf').innerText()));
  console.log('SPAF-RESUMEN-OK: ETF 02, septiembre 2026: 2 altas, 2 bajas, 3 casos; menores sin dato separados.');
  await pagina.locator('.spaf-movimientos a').filter({ hasText: '02-82-26' }).click();
  await pagina.getByRole('heading', { name: '02-82-26', exact: true }).waitFor();
  assert.match(await pagina.locator('.spaf').innerText(), /18\/09\/2026/);
  await pagina.getByRole('link', { name: 'Volver a la Tabla SPAF' }).click(); await resumen([2, 2, 3, 0]);
  console.log('SPAF-ENLACES-OK: expediente de baja consultado por API auditada; fecha histórica y retorno al mismo mes/equipo.');
  const titulo = await pagina.title(); assert(!titulo.includes('Romero'));
  const ayuda = pagina.getByRole('button', { name: 'Ayuda sobre la Tabla SPAF' });
  await ayuda.focus(); await pagina.keyboard.press('Enter');
  await pagina.getByRole('dialog').waitFor(); await pagina.keyboard.press('Escape');
  assert(await ayuda.evaluate(e => e === document.activeElement));
  const region = pagina.getByRole('region', { name: 'Tabla SPAF con desplazamiento horizontal' });
  await region.focus(); await pagina.keyboard.press('ArrowRight');
  await pagina.waitForTimeout(150); assert(await region.evaluate(e => e.scrollLeft > 0));
  await region.evaluate(e => { e.scrollLeft = 0; });
  await pagina.screenshot({ path: new URL('spaf-escritorio.png', capturas).pathname, fullPage: true });
  await ancho(1024); await ancho(390);
  await pagina.screenshot({ path: new URL('spaf-movil.png', capturas).pathname, fullPage: true });
  await ancho(360); await ancho(320);
  await ancho(1440);

  const respuestaWord = pagina.waitForResponse(r => r.request().method() === 'POST' && /tabla-spaf/.test(r.url()));
  const descargado = pagina.waitForEvent('download');
  await pagina.getByRole('button', { name: /^Descargar la Tabla SPAF/ }).click();
  const r = await respuestaWord;
  assert([200, 201].includes(r.status()));
  assert(r.headers()['x-emision-id']);
  const peticion = r.request();
  const fichero = await descargado;
  const bytes = await readFile(await fichero.path());
  assert.equal(bytes.subarray(0, 2).toString(), 'PK');
  const word = spawnSync('python3', ['-c', 'import sys,zipfile,xml.etree.ElementTree as E; z=zipfile.ZipFile(sys.argv[1]); assert z.testzip() is None; raiz=E.fromstring(z.read("word/document.xml")); texto="".join(raiz.itertext()); assert all(n in texto for n in ["02-81-26","02-82-26","02-83-26"])', await fichero.path()], { encoding: 'utf8' });
  assert.equal(word.status, 0, word.stderr);
  await pagina.getByRole('status').filter({ hasText: 'La emisión ha quedado registrada.' }).waitFor();
  const repetida = await pagina.evaluate(async ({ ruta, cuerpo, clave }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const respuesta = await fetch(ruta, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': clave }, body: cuerpo });
    return { estado: respuesta.status, repetida: respuesta.headers.get('X-Idempotencia-Repetida'), huella: respuesta.headers.get('X-Documento-SHA256') };
  }, { ruta: new URL(peticion.url()).pathname, cuerpo: peticion.postData(), clave: peticion.headers()['idempotency-key'] });
  assert.equal(repetida.estado, 200); assert.equal(repetida.repetida, 'true'); assert.equal(repetida.huella, r.headers()['x-documento-sha256']);
  console.log('SPAF-WORD-OK: POST, DOCX válido, aviso de descarga, versiones e idempotencia real.');

  const recuperadas = []; let perderRespuesta = true;
  await pagina.route('**/api/equipos/*/tabla-spaf/*/*.docx', async route => {
    const solicitud = route.request(); recuperadas.push({ clave: solicitud.headers()['idempotency-key'], cuerpo: solicitud.postData() });
    const respuesta = await route.fetch();
    if (perderRespuesta) { perderRespuesta = false; await route.abort('failed'); } else { assert.equal(respuesta.status(), 200); await route.fulfill({ response: respuesta }); }
  });
  await pagina.getByRole('button', { name: /^Descargar la Tabla SPAF/ }).click();
  await pagina.getByRole('alert').filter({ hasText: 'No se ha podido confirmar la descarga' }).waitFor();
  await preferencia('Idioma', 'en');
  await pagina.getByRole('heading', { name: 'Monthly SPAF Table', exact: true }).waitFor();
  await pagina.locator('.persona-conectada > summary').click();
  await resumen([2, 2, 3, 0]);
  const recuperacionWord = pagina.waitForEvent('download');
  await pagina.getByRole('button', { name: 'Retry download', exact: true }).click();
  await recuperacionWord; await pagina.getByRole('status').filter({ hasText: 'recovered the original' }).waitFor();
  assert.equal(recuperadas.length, 2); assert.deepEqual(recuperadas[0], recuperadas[1]);
  await pagina.unroute('**/api/equipos/*/tabla-spaf/*/*.docx');
  await preferencia('Language', 'es');
  await pagina.getByRole('heading', { name: 'Tabla Mensual para el SPAF', exact: true }).waitFor();
  await pagina.locator('.persona-conectada > summary').click();
  await resumen([2, 2, 3, 0]);
  console.log('SPAF-RECUPERACION-OK: emisión real confirmada, respuesta perdida simulada y cambio de idioma; misma clave y cuerpo, respuesta 200 repetida.');

  await preferencia('Tema', 'alto-contraste');
  await pagina.locator('.persona-conectada > summary').click();
  await resumen([2, 2, 3, 0]); await ancho(390);
  await pagina.screenshot({ path: new URL('spaf-alto-contraste-movil.png', capturas).pathname, fullPage: true });
  await ancho(1440);
  await preferencia('Idioma', 'en');
  await pagina.locator('.persona-conectada > summary').click();
  await pagina.getByRole('heading', { name: 'Monthly SPAF Table', exact: true }).waitFor();
  await resumen([2, 2, 3, 0]); await ancho(390); await ancho(1024);
  await preferencia('Language', 'es');
  await pagina.locator('.persona-conectada > summary').click();
  await preferencia('Tema', 'expedientes-etf');
  await pagina.locator('.persona-conectada > summary').click();
  console.log('SPAF-ACCESIBILIDAD-OK: ayuda por teclado, Escape/foco, tabla desplazable, 1440/1024/390/360/320, dos temas e inglés.');

  await pagina.getByLabel('Equipo', { exact: true }).selectOption(equipo01.id);
  await resumen([1, 0, 1, 0]);
  await pagina.getByLabel('Mes', { exact: true }).fill('2026-07');
  await resumen([0, 0, 0, 0]);
  await pagina.getByRole('status').filter({ hasText: 'No hay expedientes que cuenten' }).waitFor();
  console.log('SPAF-VACIO-OK: cambio de equipo y mes sin casos.');

  await pagina.route('**/api/estadistica/spaf?*', route => route.fulfill({ status: 503, contentType: 'application/json', body: '{"error":"estadistica.spaf_no_disponible"}' }));
  await pagina.getByLabel('Mes', { exact: true }).fill('2026-08');
  await pagina.getByRole('alert').filter({ hasText: 'No se puede consultar' }).waitFor();
  assert.equal(await pagina.locator('.spaf-indicadores').count(), 0);
  await pagina.unroute('**/api/estadistica/spaf?*');
  await pagina.getByRole('button', { name: 'Volver a consultar' }).click(); await resumen([0, 0, 0, 0]);
  await pagina.getByLabel('Mes', { exact: true }).fill('2026-09'); await resumen([1, 0, 1, 0]);
  await pagina.route('**/api/equipos/*/tabla-spaf/*/*.docx', route => route.fulfill({ status: 409, contentType: 'application/json', body: '{"error":"error.conflicto_version"}' }));
  await pagina.getByRole('button', { name: /^Descargar la Tabla SPAF/ }).click();
  await pagina.getByRole('alert').filter({ hasText: 'Los datos han cambiado' }).waitFor();
  assert(await pagina.getByRole('button', { name: /^Descargar la Tabla SPAF/ }).isDisabled());
  await pagina.unroute('**/api/equipos/*/tabla-spaf/*/*.docx');
  await pagina.getByRole('button', { name: 'Ver cambios' }).click(); await resumen([1, 0, 1, 0]);
  console.log('SPAF-ESTADOS-OK: 503 y 409 simulados en navegador; error sin cifras antiguas, recuperación y revisión de cambios.');

  await pagina.route('**/api/estadistica/spaf?*', async route => {
    const real = await route.fetch(); const tabla = await real.json();
    if (tabla.filas?.length) {
      tabla.filas[0].menores = { Conocido: true, Valor: [{ nombre: 'Nerea', apellidos: 'Romero Cano', nacimiento: { Conocido: true, Valor: '2014-06-10T00:00:00Z' }, centro_proteccion: { Conocido: true, Valor: 'Centro Sierra Nevada' } }] };
      tabla.filas[0].guardadores = { Conocido: true, Valor: [] };
    }
    await route.fulfill({ response: real, json: tabla });
  });
  await pagina.getByRole('button', { name: 'Ayuda sobre la Tabla SPAF' }).focus();
  await pagina.getByLabel('Mes', { exact: true }).fill('2026-08'); await resumen([0, 0, 0, 0]);
  await pagina.getByLabel('Mes', { exact: true }).fill('2026-09'); await resumen([1, 0, 1, 0]);
  await pagina.getByText('Centro Sierra Nevada', { exact: true }).waitFor();
  assert.match(await pagina.locator('.spaf table').innerText(), /N\. R\. C\./);
  assert(!/Nerea|Romero Cano/.test(await pagina.locator('.spaf table').innerText()));
  await pagina.unroute('**/api/estadistica/spaf?*');
  console.log('SPAF-PRIVACIDAD-OK: fila sintética en navegador; iniciales y centro de protección visible, sin nombre completo.');

  await pagina.locator('.persona-conectada > summary').click();
  await pagina.getByRole('button', { name: 'Cerrar sesión', exact: true }).click();
  await entrar('marta.cabrera');
  await pagina.getByLabel('Mes', { exact: true }).fill('2026-09');
  await resumen([2, 2, 3, 0]);
  assert.equal(await pagina.locator('#spaf-equipo').count(), 0);
  const aislado = await pagina.evaluate(async id => ({ opciones: await (await fetch('/api/estadistica/spaf/opciones')).json(), estado: (await fetch(`/api/estadistica/spaf?anio=2026&mes=9&etf=${id}`)).status }), equipo01.id);
  assert.equal(aislado.opciones.equipos.length, 1); assert.equal(aislado.estado, 403);
  const descargaEquipo = pagina.waitForEvent('download');
  const wordEquipo = pagina.waitForResponse(r => r.request().method() === 'POST' && /tabla-spaf/.test(r.url()));
  await pagina.getByRole('button', { name: /^Descargar la Tabla SPAF/ }).click();
  assert([200, 201].includes((await wordEquipo).status())); await descargaEquipo;
  console.log('SPAF-PERMISOS-OK: Marta ve sólo ETF 02; petición real a ETF 01 devuelve 403.');
  assert.deepEqual(errores, []);
  console.log('SPAF-RECORRIDO-OK');
} catch (e) {
  console.error('SPAF-DIAGNOSTICO', await pagina.locator('h1, h2, label, [role="alert"]').allTextContents(), errores);
  throw e;
} finally { await navegador.close(); }
