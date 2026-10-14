import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const url = process.argv[2];
assert.match(url, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('.local/usuarios.json', import.meta.url)));
const capturas = new URL('../../docs/capturas/', import.meta.url);
await mkdir(capturas, { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
try {
  async function entrar(cuenta, ancho = 1440) {
    const contexto = await navegador.newContext({ viewport: { width: ancho, height: 1000 } });
    const pagina = await contexto.newPage();
    await pagina.goto(url);
    await pagina.locator('#cuenta-ad').fill(cuenta);
    await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
    await pagina.locator('button[type="submit"]').click();
    await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
    return { contexto, pagina };
  }
  async function pedir(pagina, ruta, metodo = 'GET', datos) {
    return pagina.evaluate(async ({ ruta, metodo, datos, clave }) => {
      const sesion = await (await fetch('/api/sesion')).json();
      const respuesta = await fetch(ruta, { method: metodo, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': clave }, ...(datos ? { body: JSON.stringify(datos) } : {}) });
      const word = respuesta.headers.get('content-type')?.includes('wordprocessingml');
      return { estado: respuesta.status, word, datos: word ? { bytes: (await respuesta.arrayBuffer()).byteLength } : await respuesta.json() };
    }, { ruta, metodo, datos, clave: randomUUID() });
  }
  const anonimo = await navegador.newContext();
  assert.equal((await anonimo.request.get(url + '/api/equipo/expedientes')).status(), 401);
  await anonimo.close();
  const coordinacion = await entrar('ines.moreno');
  await coordinacion.pagina.goto(url + '/equipo');
  await coordinacion.pagina.locator('caption').waitFor();
  const provincial = await pedir(coordinacion.pagina, '/api/equipo/expedientes');
  assert.equal(provincial.estado, 200);
  const propio = provincial.datos.filas.find((e) => e.numero.startsWith('02-'));
  const ajeno = provincial.datos.filas.find((e) => e.numero.startsWith('04-'));
  assert(propio && ajeno);
  assert.doesNotMatch(JSON.stringify(provincial.datos), /Romero|Molina|Serrano/);
  const emision = await pedir(coordinacion.pagina, '/api/documentos/emisiones', 'POST', { expediente_id: propio.id, tipo: 'entrada/aceptacion', idioma: 'es' });
  assert.equal(emision.estado, 201); assert(emision.word && emision.datos.bytes > 1000);
  console.log('COORDINACION-OK: bandeja provincial y Word emitido');
  const marta = await entrar('marta.cabrera');
  const pagina = marta.pagina;
  await pagina.goto(url + '/equipo');
  await pagina.locator('caption').waitFor();
  const lista = await pedir(pagina, '/api/equipo/expedientes');
  assert.equal(lista.estado, 200); assert.equal(lista.datos.filas.length, 2);
  assert(lista.datos.filas.every((e) => e.numero.startsWith('02-')));
  assert(lista.datos.filas.every((e) => e.avisos.includes('equipo.aviso.memoria')));
  await pagina.getByText('Quedan datos por completar para la Memoria.', { exact: true }).first().waitFor();
  assert.equal((await pedir(pagina, '/api/equipo/expedientes/' + ajeno.id)).estado, 404);
  assert.equal((await pedir(pagina, '/api/equipo/expedientes?estado=desconocido')).estado, 400);
  await pagina.screenshot({ path: new URL('equipo-bandeja-escritorio.png', capturas).pathname, fullPage: true });
  await pagina.locator('#equipo-busqueda').fill('familia-inexistente');
  await pagina.getByRole('button', { name: 'Buscar expedientes', exact: true }).click();
  await pagina.getByText('No hay expedientes con esta búsqueda y estos filtros.').waitFor();
  await pagina.getByRole('button', { name: 'Quitar filtros', exact: true }).first().click();
  await pagina.locator('caption').waitFor();
  await pagina.locator('#equipo-busqueda').fill('Romero');
  await pagina.getByRole('button', { name: 'Buscar expedientes', exact: true }).click();
  await pagina.waitForFunction(() => document.querySelectorAll('.equipo tbody tr').length === 1);
  assert.doesNotMatch(await pagina.locator('tbody').innerText(), /Romero/);
  await pagina.getByRole('button', { name: 'Quitar filtros', exact: true }).first().click();
  await pagina.waitForFunction(() => document.querySelectorAll('.equipo tbody tr').length === 2);
  await pagina.setViewportSize({ width: 390, height: 844 });
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
  await pagina.screenshot({ path: new URL('equipo-bandeja-movil.png', capturas).pathname, fullPage: true });
  for (const width of [360, 1024]) { await pagina.setViewportSize({ width, height: 900 }); assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)); }
  await pagina.setViewportSize({ width: 1440, height: 1000 });
  await pagina.goto(url + '/equipo/expediente?id=' + propio.id);
  await pagina.getByRole('tab', { name: 'Documentos', exact: true }).waitFor();
  await pagina.getByText('Acta de Aceptación', { exact: true }).first().waitFor();
  assert.match(await pagina.locator('h1').innerText(), /Romero/);
  assert.doesNotMatch(await pagina.title(), /Romero|Molina/);
  assert.equal(await pagina.getByRole('tab').count(), 9);
  await pagina.screenshot({ path: new URL('equipo-ficha-escritorio.png', capturas).pathname, fullPage: true });
  await pagina.getByRole('tab', { name: 'Documentos', exact: true }).focus();
  await pagina.keyboard.press('ArrowRight');
  await pagina.getByRole('button', { name: 'Retirar la propuesta de cierre', exact: true }).waitFor();
  await pagina.getByRole('button', { name: 'Retirar la propuesta de cierre', exact: true }).click();
  await pagina.getByRole('button', { name: 'Guardar datos de finalización', exact: true }).waitFor();
  await pagina.locator('#equipo-motivo').selectOption('SIN_CONSENTIMIENTO');
  await pagina.locator('input[type="checkbox"]').check();
  await pagina.locator('#equipo-cierre-coordinacion_recursos').fill('Reunión de seguimiento con el servicio derivante.');
  const finalizacionLeida = pagina.waitForResponse((r) => r.url().includes('/finalizacion') && r.request().method() === 'GET');
  await pagina.getByRole('button', { name: 'Guardar datos de finalización', exact: true }).click();
  await finalizacionLeida;
  const propuesta = pagina.waitForResponse((r) => r.url().endsWith('/proponer-cierre'));
  await pagina.getByRole('button', { name: 'Proponer el cierre a Coordinación', exact: true }).click();
  assert.equal((await propuesta).status(), 200);
  await pagina.getByRole('button', { name: 'Retirar la propuesta de cierre', exact: true }).waitFor();
  const ficha = await pedir(pagina, '/api/equipo/expedientes/' + propio.id);
  assert.equal(ficha.datos.estado, 'aceptada'); assert.equal(ficha.datos.hitos.length, 1); assert(ficha.datos.emisiones.length > 0);
  assert.equal(ficha.datos.hitos[0].tipo, 'hito'); assert.equal(ficha.datos.emisiones[0].tipo, 'emision');
  await pagina.setViewportSize({ width: 390, height: 844 });
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
  await pagina.screenshot({ path: new URL('equipo-ficha-movil.png', capturas).pathname, fullPage: true });
  await pagina.getByRole('tab', { name: 'Historial', exact: true }).click();
  await pagina.getByRole('heading', { name: 'Actividad reciente', exact: true }).waitFor();
  assert.doesNotMatch(await pagina.getByRole('tabpanel').innerText(), /equipo\.|expediente\.|[a-f0-9]{8}-[a-f0-9]{4}/);
  const segundo = lista.datos.filas.find((e) => e.id !== propio.id);
  await pagina.goto(url + '/equipo/expediente?id=' + segundo.id + '&pestana=salidas');
  const salidaAnterior = await pedir(pagina, '/api/expedientes/' + segundo.id + '/salida-spm');
  if (!salidaAnterior.datos) {
    await pagina.locator('#equipo-spm-motivo').waitFor();
    await pagina.locator('#equipo-spm-motivo').fill('Se solicita la valoración de las necesidades de protección de los menores.');
    await pagina.locator('#equipo-spm-medida').fill('Valoración coordinada con el servicio derivante.');
    const creada = pagina.waitForResponse((r) => r.url().endsWith('/salida-spm') && r.request().method() === 'POST');
    await pagina.getByRole('button', { name: 'Preparar la derivación al Servicio de Protección de Menores', exact: true }).click();
    assert.equal((await creada).status(), 200);
  }
  await pagina.getByRole('button', { name: 'Generar Anexo 5', exact: true }).waitFor();
  const descarga = pagina.waitForEvent('download');
  await pagina.getByRole('button', { name: 'Generar Anexo 5', exact: true }).click();
  assert.equal(await (await descarga).failure(), null);
  const preparada = await pedir(pagina, '/api/expedientes/' + segundo.id + '/salida-spm');
  assert.equal(preparada.datos.fecha_derivacion ?? '', salidaAnterior.datos?.fecha_derivacion ?? '');
  const hoy = new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Madrid' }).format(new Date());
  if (!preparada.datos.fecha_derivacion) {
  await pagina.locator('#equipo-spm-fecha').fill(hoy);
  const confirmada = pagina.waitForResponse((r) => r.url().endsWith('/salida-spm/fecha'));
  await pagina.getByRole('button', { name: 'Confirmar la fecha de envío', exact: true }).click();
  assert.equal((await confirmada).status(), 200);
  }
  await pagina.locator('#equipo-spm-respuesta').waitFor();
  const enEspera = pagina.waitForResponse((r) => r.url().endsWith('/salida-spm/respuesta'));
  await pagina.getByRole('button', { name: 'Guardar respuesta', exact: true }).click();
  assert.equal((await enEspera).status(), 200);
  const conAviso = await pedir(pagina, '/api/equipo/expedientes/' + segundo.id);
  assert(conAviso.datos.avisos.includes('equipo.aviso.spm_pendiente'));
  await pagina.getByRole('tab', { name: 'Familia', exact: true }).click();
  await pagina.getByRole('heading', { name: 'Datos de la familia', exact: true }).waitFor();
  let familia = await pedir(pagina, '/api/expedientes/' + segundo.id + '/familia');

  assert.equal(familia.estado, 200);
  if (familia.datos.adultos.length) await pagina.getByRole('table', { name: 'Figuras parentales', exact: true }).waitFor();
  else await pagina.getByText('Todavía no hay personas adultas registradas.', { exact: true }).waitFor();
  console.log('SALIDAS-FAMILIA-OK: Anexo 5 descargado sin confirmar envío, fecha separada, espera sin fecha y familia vigente');
  console.log('EQUIPO-OK: búsqueda enmascarada, vacío, teclado, cierre versionado, hitos separados, 360/390/1024/1440 px');
  for (const cuenta of ['elena.salas', 'lucia.navarro']) {
    const actor = await entrar(cuenta);
    assert.equal((await pedir(actor.pagina, '/api/equipo/expedientes')).estado, 403);
    assert.equal((await pedir(actor.pagina, '/api/equipo/expedientes/' + propio.id)).estado, 403);
    await actor.contexto.close();
  }
  console.log('PERMISOS-OK: anónimo, Administración, Consulta y equipo ajeno denegados');
} finally { await navegador.close(); }
