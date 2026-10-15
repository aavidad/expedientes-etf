import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const url = process.argv[2];
assert.match(url, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const axe = await readFile(new URL('../../web/node_modules/axe-core/axe.min.js', import.meta.url), 'utf8');
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
async function entrar(cuenta) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, serviceWorkers: 'block' });
  const page = await context.newPage();
  await page.goto(url); await page.locator('#cuenta-ad').fill(cuenta); await page.locator('#contrasena-ad').fill(claves[cuenta]);
  await page.locator('button[type="submit"]').click(); await page.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return page;
}
async function pedir(page, ruta, metodo = 'GET', datos, csrf = true) {
  return page.evaluate(async ({ ruta, metodo, datos, csrf, clave }) => {
    const respuestaSesion = await fetch('/api/sesion');
    if (!respuestaSesion.ok) throw new Error('GET /api/sesion: HTTP ' + respuestaSesion.status);
    const sesion = await respuestaSesion.json();
    const r = await fetch(ruta, { method: metodo, headers: { 'Content-Type': 'application/json', ...(csrf ? { 'X-CSRF-Token': sesion.csrf } : {}), 'Idempotency-Key': clave }, ...(datos ? { body: JSON.stringify(datos) } : {}) });
    return { estado: r.status, datos: await r.json() };
  }, { ruta, metodo, datos, csrf, clave: randomUUID() });
}
async function preferencias(page) {
  if (!await page.locator('.persona-conectada').evaluate((e) => e.open)) await page.locator('.persona-conectada > summary').click();
  if (!await page.locator('.preferencias').evaluate((e) => e.open)) await page.locator('.preferencias > summary').click();
}
async function cerrarPreferencias(page) {
  await page.locator('.preferencias > summary').focus(); await page.keyboard.press('Escape');
  await page.locator('.persona-conectada > summary').focus(); await page.keyboard.press('Escape');
}
async function pantalla(page, ancho, nombre) {
  await page.setViewportSize({ width: ancho, height: 1000 });
  await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth + 1);
  await page.evaluate(async () => { await document.fonts.ready; });
  if (nombre) await page.screenshot({ path: new URL(`../../docs/capturas/ubicaciones-${nombre}.png`, import.meta.url).pathname, fullPage: true });
}
let coord;
let reabrirMemoriaDePrueba = false;
let falloRecorrido;
try {
  coord = await entrar('ines.moreno');
  assert.equal((await pedir(coord, '/api/admin/ubicaciones')).estado, 403);
  const expedientes = (await pedir(coord, '/api/equipo/expedientes')).datos;
  assert(expedientes.filas.some((e) => e.numero));
  const numerados = expedientes.filas.filter((e) => e.numero).map((e) => ({ id: e.id, numero: e.numero, equipo_id: e.equipo_id }));
  let memoria = await pedir(coord, '/api/estadistica/memoria/estado?anio=2026');
  assert.equal(memoria.estado, 200);
  if (memoria.datos.estado !== 'cerrada') {
    memoria = await pedir(coord, '/api/estadistica/memoria/cerrar?anio=2026', 'POST', { version: memoria.datos.version });
    assert.equal(memoria.estado, 200, JSON.stringify(memoria.datos));
    reabrirMemoriaDePrueba = true;
  }
  const foto = await pedir(coord, `/api/estadistica/memoria/instantanea?anio=2026&revision=${memoria.datos.revision_actual}`);
  assert.equal(foto.estado, 200);
  const admin = await entrar('elena.salas');
  await admin.goto(url + '/admin/ubicaciones');
  await admin.getByRole('heading', { name: 'Equipos, municipios y centros', exact: true }).waitFor();
  await admin.locator('#ubicaciones-buscar').fill('Íllora');
  await admin.getByRole('button', { name: 'Editar ubicación Íllora', exact: true }).click();
  await admin.locator('#ubicaciones-texto_es').waitFor();
  await admin.evaluate(axe);
  const accesibilidad = await admin.evaluate(async () => window.axe.run({ include: [['.ubicaciones']] }));
  assert.deepEqual(accesibilidad.violations.map((v) => ({ id: v.id, impacto: v.impact })), []);
  await pantalla(admin, 1440, '1440'); await pantalla(admin, 390, '390'); await pantalla(admin, 360);
  // Ayuda y foco por teclado.
  await admin.getByRole('button', { name: 'Ayuda sobre equipos, municipios y centros' }).focus().catch(async () => admin.locator('.ubicaciones-cabecera summary').focus());
  await admin.keyboard.press('Enter'); await admin.keyboard.press('Escape');
  assert.equal(await admin.locator('.ubicaciones-cabecera details').evaluate((e) => e.open), false);
  await admin.locator('#ubicaciones-variantes_es').fill('Villa de Íllora');
  await admin.locator('#ubicaciones-variantes_en').fill('Illora town');
  const original = (await pedir(admin, '/api/admin/ubicaciones')).datos;
  const centro = original.datos.centros.find((c) => c.codigo === 'ALHAMA') ?? original.datos.centros[0];
  const equipo = original.datos.equipos.find((e) => e.codigo === '06');
  await admin.locator('#ubicaciones-centro_ss_id').selectOption(centro.id);
  await admin.locator('#ubicaciones-equipo_id').selectOption(equipo.id);
  await admin.locator('#ubicaciones-ambito').selectOption('false');
  const desde = new Date(new Date(original.datos.version.vigente_desde).getTime() + 86400000).toISOString().slice(0, 10);
  await admin.locator('#ubicaciones-vigente_desde').fill(desde);
  const respuesta = admin.waitForResponse((r) => r.url().endsWith('/api/admin/ubicaciones') && r.request().method() === 'PUT');
  await admin.getByRole('button', { name: 'Publicar cambio', exact: true }).click();
  assert.equal((await respuesta).status(), 201);
  await admin.getByRole('status').filter({ hasText: 'Cambio publicado' }).waitFor();
  assert.equal(await admin.locator('#ubicaciones-titulo').evaluate((e) => e === document.activeElement), true);
  const actual = (await pedir(admin, '/api/admin/ubicaciones')).datos;
  const illora = actual.datos.municipios.find((m) => m.ine === '18102');
  assert.equal(illora.centro_ss_id, centro.id); assert.equal(illora.equipo_id, equipo.id); assert.equal(illora.ambito_diputacion, false);
  assert.equal(actual.textos.es['municipio.18102.variante.v1'], 'Villa de Íllora');
  const anexo = execFileSync('python3', ['-c', `import io,sys,zipfile
with zipfile.ZipFile('internal/adaptadores/anexos/docx/testdata/anexo3.docx') as origen:
 salida=io.BytesIO()
 with zipfile.ZipFile(salida,'w',zipfile.ZIP_DEFLATED) as destino:
  for entrada in origen.infolist():
   contenido=origen.read(entrada.filename)
   if entrada.filename=='word/document.xml': contenido=contenido.replace('18120 Alhama de Granada'.encode(),'Villa de Íllora'.encode())
   destino.writestr(entrada,contenido)
 sys.stdout.buffer.write(salida.getvalue())`]);
  await coord.goto(url + '/coordinacion/anexos/nuevo');
  await coord.locator('#anexo-fichero').setInputFiles({ name: 'anexo3-illora.docx', mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', buffer: anexo });
  const lectura = coord.waitForResponse((r) => r.url().endsWith('/api/coordinacion/derivaciones') && r.request().method() === 'POST');
  const textosAnexo = JSON.parse(await readFile(new URL('../../idiomas/es/textos/anexo.json', import.meta.url)));
  await coord.getByRole('button', { name: textosAnexo['anexo.subir'], exact: true }).click();
  const respuestaAnexo = await lectura; assert.equal(respuestaAnexo.status(), 200);
  const propuesta = (await respuestaAnexo.json()).derivacion_datos.propuesta_municipio;
  assert.equal(propuesta.Estado, 1); assert.equal(propuesta.Candidatos[0].Codigo, '18102');
  assert.equal(propuesta.Candidatos[0].FueraAmbitoDiputacion, true);

  const despues = (await pedir(coord, '/api/equipo/expedientes')).datos.filas.filter((e) => e.numero).map((e) => ({ id: e.id, numero: e.numero, equipo_id: e.equipo_id }));
  assert.deepEqual(despues, numerados);
  const fotoDespues = await pedir(coord, `/api/estadistica/memoria/instantanea?anio=2026&revision=${memoria.datos.revision_actual}`);
  assert.deepEqual(fotoDespues, foto);
  assert.equal((await pedir(admin, '/api/admin/ubicaciones', 'PUT', {}, false)).estado, 403);
  // Conflicto real: formulario conserva sus campos y exige comparar la versión actual.
  await admin.locator('#ubicaciones-buscar').fill('Íllora');
  await admin.getByRole('button', { name: 'Editar ubicación Íllora', exact: true }).click();
  await admin.locator('#ubicaciones-texto_es').fill('Íllora del catálogo');
  const cambio = { version_esperada: actual.datos.version.numero, referencias_esperadas: actual.referencias, vigente_desde: new Date(new Date(actual.datos.version.vigente_desde).getTime() + 86400000).toISOString(), textos: { es: 'Íllora', en: 'Íllora' }, ubicacion: { tipo: 'municipio', id: illora.id, codigo: illora.ine, centro_ss_id: centro.id, equipo_id: equipo.id, ambito_diputacion: false, variantes: { es: ['Villa de Íllora'], en: ['Illora town'] } } };
  assert.equal((await pedir(admin, '/api/admin/ubicaciones', 'PUT', cambio)).estado, 201);
  const conflicto = admin.waitForResponse((r) => r.url().endsWith('/api/admin/ubicaciones') && r.request().method() === 'PUT');
  await admin.getByRole('button', { name: 'Publicar cambio', exact: true }).click(); assert.equal((await conflicto).status(), 409);
  assert.equal(await admin.locator('#ubicaciones-texto_es').inputValue(), 'Íllora del catálogo');
  assert.equal(await admin.getByRole('button', { name: 'Publicar cambio', exact: true }).isDisabled(), true);
  await admin.getByRole('button', { name: 'Consultar versión actual', exact: true }).click();
  await admin.getByRole('heading', { name: 'Versión actual: compara antes de publicar' }).waitFor();
  await admin.waitForFunction(() => !document.querySelector('.formulario-admin input[type=checkbox]')?.disabled); assert.equal(await admin.getByRole('checkbox').isDisabled(), false);
  await admin.getByRole('button', { name: 'Volver sin guardar' }).click();
  await admin.locator('#ubicaciones-buscar').fill('No coincide ningún municipio'); await admin.getByRole('status').filter({ hasText: 'No hay ubicaciones' }).waitFor();
  await admin.locator('#ubicaciones-buscar').fill('Íllora');
  await preferencias(admin); await admin.locator('#tema').selectOption('alto-contraste'); await cerrarPreferencias(admin);
  await pantalla(admin, 1024); await pantalla(admin, 390, '390-alto-contraste');
  await preferencias(admin); await admin.locator('#idioma').selectOption('en'); await cerrarPreferencias(admin);
  await admin.getByRole('heading', { name: 'Teams, municipalities and centres', exact: true }).waitFor(); await pantalla(admin, 390, '390-en');
  // Error de red traducido, sin vaciar el formulario.
  await admin.route('**/api/admin/ubicaciones', (r) => r.abort());
  await admin.reload({ waitUntil: 'domcontentloaded' }); await admin.getByRole('alert').waitFor(); assert((await admin.getByRole('alert').innerText()).trim().length > 15);
  await admin.unroute('**/api/admin/ubicaciones');
  console.log('UBI1-NAVEGADOR-OK Chrome 1440/1024/390/360, es/en, alto contraste, teclado, vacío/error/conflicto, CSRF y Coordinación 403');
  console.log('UBI1-ANEXO-OK DOCX inventado con variante publicada reconoce Íllora y su ámbito');
  console.log('UBI1-REGRESION-OK expedientes numerados y Memoria cerrada conservados tras publicación; asignaciones y variantes verificadas');
} catch (error) {
  falloRecorrido = error;
  throw error;
} finally {
  try {
    if (reabrirMemoriaDePrueba && coord) {
      const estado = await pedir(coord, '/api/estadistica/memoria/estado?anio=2026');
      assert.equal(estado.estado, 200, JSON.stringify(estado.datos));
      const reapertura = await pedir(coord, '/api/estadistica/memoria/reabrir?anio=2026', 'POST', { version: estado.datos.version, motivo: 'Finaliza la comprobación del editor de ubicaciones.' });
      assert.equal(reapertura.estado, 200, JSON.stringify(reapertura.datos));
    }
  } catch (error) {
    if (falloRecorrido) throw new AggregateError([falloRecorrido, error], 'Fallaron el recorrido de ubicaciones y su limpieza.');
    throw error;
  } finally { await browser.close(); }
}
