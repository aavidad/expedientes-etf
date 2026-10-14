// Recorrido UI3 con datos inventados, API real y Chrome del sistema.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const base = process.argv[2];
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const destino = new URL('../../docs/capturas/', import.meta.url);
await mkdir(destino, { recursive: true });
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];
let capturas = 0;
async function entrar(cuenta) {
  const contexto = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  const p = await contexto.newPage();
  p.on('pageerror', e => errores.push(e.message));
  await p.goto(base);
  await p.locator('#cuenta-ad').fill(cuenta);
  await p.locator('#contrasena-ad').fill(claves[cuenta]);
  await p.locator('button[type="submit"]').click();
  await p.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return p;
}
async function pedir(p, ruta, metodo = 'GET', datos) {
  return p.evaluate(async ({ ruta, metodo, datos, clave }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const r = await fetch(ruta, { method: metodo, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': clave }, ...(datos ? { body: JSON.stringify(datos) } : {}) });
    return { estado: r.status, datos: await r.json() };
  }, { ruta, metodo, datos, clave: randomUUID() });
}
async function geometria(p, ancho) {
  await p.setViewportSize({ width: ancho, height: 1000 });
  await p.waitForFunction(() => document.querySelector('.cabecera-app')?.getAttribute('data-compacto') === String(innerWidth < parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu')) * 4));
  await p.evaluate(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))));
  await p.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 1);
  await p.evaluate(() => document.fonts.ready);
  assert.equal(await p.locator('h1').count(), 1);
}
async function capturar(p, nombre) {
  for (const ancho of [1440, 390]) {
    await geometria(p, ancho);
    if (nombre === 'equipo-persona' && ancho === 390) await p.locator('.persona-conectada > summary').click();
    await p.screenshot({ path: new URL(`ui3-${nombre}-${ancho}.png`, destino).pathname, fullPage: true });
    if (nombre === 'equipo-persona' && ancho === 390) { await p.locator('.persona-conectada > summary').focus(); await p.keyboard.press('Escape'); }
    capturas++;
  }
}
async function preferencias(p, idioma, tema) {
  await p.locator('.persona-conectada > summary').click();
  await p.locator('.preferencias > summary').click();
  if (await p.locator('#idioma').inputValue() !== idioma) {
    await p.locator('#idioma').selectOption(idioma);
    await p.waitForFunction(l => document.documentElement.lang === l, idioma);
    await p.locator('.persona-conectada > summary').click();
    await p.locator('.preferencias > summary').click();
  }
  await p.locator('#tema').selectOption(tema);
  await p.waitForFunction(t => document.querySelector(`link[href="/api/temas/${t}/tokens"]`)?.sheet, tema);
  await p.locator('#tema').press('Escape');
  await p.locator('.persona-conectada > summary').focus();
  await p.keyboard.press('Escape');
}
try {
  const p = await entrar('ines.moreno');
  await p.goto(base + '/coordinacion');
  await p.locator('.bandeja-tabla tbody tr').first().waitFor();
  assert.equal(await p.locator('#menu-principal h2').allTextContents().then(t => t.join(' · ')), 'Derivaciones · Expedientes · Informes y Memoria · Configuración');
  const enlaces = p.locator('.bandeja-tabla th[scope="row"] a');
  assert(!(await enlaces.allTextContents()).includes('Revisar derivación'));
  assert.match(await enlaces.first().innerText(), /\d{2}\/\d{2}\/\d{4} · /);
  const revision = await enlaces.first().getAttribute('href');
  await capturar(p, 'bandeja');
  await geometria(p, 390);
  await p.getByRole('button', { name: 'Menú', exact: true }).click();
  await p.locator('#menu-principal a').first().focus();
  await p.keyboard.press('Escape');
  assert.equal(await p.getByRole('button', { name: 'Menú', exact: true }).evaluate(e => e === document.activeElement), true);
  await p.goto(base + revision);
  const plegado = p.locator('details').filter({ has: p.getByText(/^Datos que el anexo no trae \(\d+\)$/) });
  await plegado.waitFor();
  assert.equal(await plegado.getAttribute('open'), null);
  assert.equal(await p.getByText('Sin indicar en el anexo', { exact: true }).filter({ visible: true }).count(), 0);
  await capturar(p, 'decision');
  await plegado.locator('summary').focus();
  await p.keyboard.press('Enter');
  assert(await plegado.locator('dd').count() > 0);
  await p.keyboard.press('Enter');
  const m = await entrar('marta.cabrera');
  const lista = await pedir(m, '/api/equipo/expedientes');
  assert.equal(lista.estado, 200);
  const equipo = (await pedir(m, '/api/sesion')).datos.equipo_id;
  const historica = await pedir(p, `/api/estadistica/spaf?anio=2026&mes=9&etf=${equipo}`);
  assert.equal(historica.estado, 200);
  assert.equal(historica.datos.resumen.altas, 2);
  assert.equal(historica.datos.resumen.bajas, 2);
  assert.equal(historica.datos.resumen.casos, 3);
  for (const f of historica.datos.filas) {
    const ficha = await pedir(p, `/api/expedientes/${f.expediente_id}/menores`);
    assert.equal(ficha.estado, 200);
    assert.equal(ficha.datos.menores.length, 1);
    assert(ficha.datos.menores[0].fecha_nacimiento);
  }
  console.log(`UI3-SPAF: ${JSON.stringify(historica.datos.resumen)}; 3 menores históricos guardados en ficha.`);
  await p.goto(base + `/equipo/tabla-spaf?mes=2026-09&equipo=${equipo}`);
  await p.locator('.spaf-tabla-contenedor tbody tr').first().waitFor();
  await capturar(p, 'spaf');
  const vivos = lista.datos.filas.filter(f => f.estado !== 'cerrada');
  for (const f of vivos) {
    const ficha = (await pedir(m, `/api/expedientes/${f.id}/menores`)).datos;
    assert(ficha.menores.length > 0);
    assert.equal(new Set(ficha.menores.map(n => n.nombre)).size, ficha.menores.length);
    for (const n of ficha.menores) {
      assert(n.fecha_nacimiento && n.nivel_escolarizacion);
      if (n.nombre === 'Daniel') assert.equal(n.sexo, 'H');
    }
    const intervencion = (await pedir(m, `/api/expedientes/${f.id}/intervencion`)).datos.intervencion;
    assert(intervencion && Object.values(intervencion.listas).every(l => l !== null));
  }
  const id = vivos.find(f => f.menores > 1).id;
  const ruta = `/api/expedientes/${id}/menores`;
  const originales = (await pedir(m, ruta)).datos;
  const incompletos = structuredClone(originales.menores);
  incompletos[1].fecha_nacimiento = null;
  incompletos[1].propuesto_atencion_temprana = null;
  const vigente = originales.fecha_aceptacion.slice(0, 10);
  const guardado = await pedir(m, ruta, 'PUT', { version: originales.version, vigente_desde: vigente, menores: incompletos });
  assert.equal(guardado.estado, 200);
  try {
    await p.goto(base + `/equipo/expediente?id=${id}`);
    await p.getByRole('tab', { name: /^Menores/ }).click();
    const aviso = p.locator('.fi1-resumen a').filter({ hasText: `Fecha de nacimiento · Menor 2 (${incompletos[1].nombre})` });
    await aviso.waitFor();
    await aviso.click();
    const campo = p.locator(`#fi1-fecha_nacimiento-${incompletos[1].id}`);
    assert(await campo.locator('..').evaluate(e => e === document.activeElement));
    await p.evaluate(() => scrollTo(0, 0));
    await capturar(p, 'avisos-menores');
  } finally {
    const actual = (await pedir(m, ruta)).datos;
    assert.equal((await pedir(m, ruta, 'PUT', { version: actual.version, vigente_desde: vigente, menores: originales.menores })).estado, 200);
  }
  await m.goto(base + '/equipo');
  await m.locator('.equipo-persona').filter({ hasText: /ETF/ }).first().waitFor();
  assert(!(await m.locator('.equipo-persona').first().innerText()).includes('Equipo asignado'));
  await capturar(m, 'equipo-persona');
  for (const pagina of [p, m]) {
    for (const ancho of [1024, 360, 320]) await geometria(pagina, ancho);
    await preferencias(pagina, 'en', 'alto-contraste');
    for (const ancho of [1440, 390, 1024, 320]) await geometria(pagina, ancho);
    await preferencias(pagina, 'es', 'expedientes-etf');
  }
  assert.deepEqual(errores, []);
  console.log(`UI3-OK: identificadores, menú por secciones, plegado y teclado; menores únicos e intervención; SPAF 2 altas, 2 bajas, 3 casos; 3 menores históricos en ficha; avisos con nombre y foco; equipo; es/en, dos temas, 1440/1024/390/360/320; ${capturas} capturas; sin errores JS.`);
} finally { await browser.close(); }
