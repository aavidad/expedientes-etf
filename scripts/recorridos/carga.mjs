import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
import { tmpdir } from 'node:os';
const base = process.argv[2] ?? 'http://127.0.0.1:51481';
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
// Las imágenes nuevas no sustituyen los ficheros del volumen de idiomas ya creado.
// Renueva sólo la reserva de textos de CA1 en la demo de este worktree.
async function prepararTextosDemo() {
 const esperados = new Map(); const anteriores = [];
 async function servidos(idioma) {
  const r = await fetch(`${base}/api/idiomas/${idioma}/textos`, { signal: AbortSignal.timeout(10000) });
  assert.equal(r.status, 200, `No se pueden consultar los textos de la demo (${idioma}).`);
  return r.json();
 }
 for (const idioma of ['es', 'en']) {
  const textos = JSON.parse(await readFile(new URL(`../../idiomas/${idioma}/textos/carga.json`, import.meta.url)));
  esperados.set(idioma, textos); const actuales = await servidos(idioma);
  if (Object.entries(textos).some(([clave, valor]) => actuales[clave] !== valor)) anteriores.push(idioma);
 }
 if (anteriores.length) {
  execFileSync('python3', ['-B', '-c', `import importlib.util, sys
from pathlib import Path
raiz = Path(sys.argv[1])
spec = importlib.util.spec_from_file_location("demo_recorrido_carga", raiz / "despliegue/demo/demo.py")
demo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(demo)
if demo.URL != sys.argv[2]:
 raise SystemExit("Los textos sólo se pueden renovar en la demo de este worktree.")
for idioma in sys.argv[3:]:
 if idioma not in ("es", "en"):
  raise SystemExit("Idioma no previsto para este recorrido.")
 demo.compose("cp", str(raiz / "idiomas" / idioma / "textos/carga.json"), "app:/datos/idiomas/" + idioma + "/textos/carga.json", capture_output=True, text=True)
`, fileURLToPath(new URL('../../', import.meta.url)), base, ...anteriores], { timeout: 30000 });
  console.log(`CA1_TEXTOS_DEMO_ACTUALIZADOS: ${anteriores.join(', ')}.`);
 }
 for (const [idioma, textos] of esperados) {
  const actuales = await servidos(idioma);
  assert.ok(Object.entries(textos).every(([clave, valor]) => actuales[clave] === valor), `La demo no sirve los textos actuales de CA1 (${idioma}). Revisa el idioma activo antes de probar las pantallas.`);
 }
}
await prepararTextosDemo();
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1100 }, acceptDownloads: true });
const p = await contexto.newPage(); p.setDefaultTimeout(20000);
const errores = []; p.on('pageerror', e => errores.push(e.message));
const capturas = new URL('../../docs/capturas/', import.meta.url); await mkdir(capturas, { recursive: true });
let recibos = []; let previsualizaciones = [];
p.on('response', async r => { if (r.request().method() === 'POST' && r.url().includes('/api/carga-2026')) { const d = await r.json().catch(() => null); if (d?.expedientes) recibos.push(d); if (d?.valida !== undefined) previsualizaciones.push(d); } });
async function entrar(cuenta) {
 await p.goto(base); await p.locator('#cuenta-ad').fill(cuenta); await p.locator('#contrasena-ad').fill(claves[cuenta]); await p.locator('button[type=submit]').click(); await p.locator('#cuenta-ad').waitFor({ state: 'detached' });
}
async function preferencias() { if (!await p.locator('.persona-conectada').evaluate(e => e.open)) await p.locator('.persona-conectada > summary').click(); if (!await p.locator('.preferencias').evaluate(e => e.open)) await p.locator('.preferencias > summary').click(); }
async function cerrarPreferencias() { await p.locator('.preferencias > summary').focus(); await p.keyboard.press('Escape'); await p.locator('.persona-conectada > summary').focus(); await p.keyboard.press('Escape'); }
async function salir() { await p.locator('.persona-conectada').evaluate(e => e.open = true); await p.locator('.persona-conectada > summary').focus(); await p.getByRole('button', { name: 'Cerrar sesión', exact: true }).click(); }
async function leer(ruta) { return p.evaluate(async ruta => { const r = await fetch(ruta); return { estado: r.status, datos: await r.json() }; }, ruta); }
async function comprobar() {
 await p.evaluate(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))));
 assert.equal(await p.locator('h1').count(), 1); const ids = await p.locator('main [id]').evaluateAll(es => es.map(e => e.id)); assert.equal(new Set(ids).size, ids.length); assert.ok(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth), JSON.stringify(await p.locator('body *').evaluateAll(es => es.filter(e => e.getBoundingClientRect().right > innerWidth + 1).map(e => ({tag:e.tagName, id:e.id, clase:e.className, ancho:e.getBoundingClientRect().width, derecha:e.getBoundingClientRect().right}))))); assert.equal(await p.evaluate(() => localStorage.length + sessionStorage.length), 0);
 for (const e of await p.locator('main button, main input, main select, main summary').all()) if (await e.isVisible()) assert.ok(await e.evaluate(e => e.getBoundingClientRect().height >= 44));
 for (const e of await p.locator('main input, main select, main textarea').all()) assert.ok(await e.evaluate(e => !!e.labels?.length));
}
async function captura(nombre, ancho) { await p.setViewportSize({ width: ancho, height: 1100 }); await p.evaluate(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))); await comprobar(); await p.screenshot({ path: new URL(nombre, capturas).pathname, fullPage: true }); }
async function manual() { await p.goto(base + '/equipo/expedientes/alta-historica'); await p.locator('#carga-equipo option').nth(1).waitFor({ state: 'attached' }); }
async function completar(estado, numero) {
 await manual(); const o = (await leer('/api/carga-2026/opciones?idioma=es')).datos;
 const e = o.equipos.find(e => e.codigo === '02') ?? o.equipos[0]; const m = o.municipios.find(m => m.equipo_id === e.id && m.centro_ss_id && m.ambito_diputacion); assert(m);
 await p.locator('#carga-equipo').selectOption(e.id); await p.locator('#carga-estado').selectOption(estado); await p.locator('#carga-tipo').selectOption('anexo3'); await p.locator('#carga-derivacion').fill('2026-01-08');
 if (estado === 'no_aceptada') { await p.locator('#carga-noAceptacion').fill('2026-02-12'); await p.locator('#carga-causa').fill('El servicio derivante retiró la solicitud.'); await p.locator('#carga-menor-nombre-0').fill('Sofía'); await p.locator('#carga-menor-apellidos-0').fill('Molina Daza'); }
 else {
  await p.locator('#carga-numero').fill(`${e.codigo}-${numero}-26`); await p.locator('#carga-aceptacion').fill('2026-01-15'); await p.locator('#carga-municipio').selectOption(m.id); await p.locator('#carga-puesto').selectOption('TS'); await p.locator('#carga-descripcion').fill('La familia solicita apoyo para las rutinas y la asistencia escolar.');
  await p.locator('#carga-menor-nombre-0').fill('Clara'); await p.locator('#carga-menor-apellidos-0').fill('Villena Robles'); await p.locator('#carga-menor-sexo-0').selectOption('M'); await p.locator('#carga-menor-fecha_nacimiento-0').fill('2018-02-19');
  await p.locator('#carga-direccion').fill('Calle Laurel 18');
  if (estado === 'cerrada' || estado === 'en_intervencion') { await p.locator('#carga-cierre-0').fill('2026-03-10'); await p.locator('#carga-motivo-0').selectOption(estado === 'cerrada' ? 'OBJ_TOTAL' : 'TRASLADO'); }
  if (estado === 'en_intervencion') { await p.getByRole('button', { name: 'Añadir una reapertura', exact: true }).click(); await p.locator('#carga-apertura-1').fill('2026-05-04'); await p.locator('#carga-reapertura-1').fill('La familia vuelve al municipio.'); }
 }
 return e;
}
async function revisar() { const r = p.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/carga-2026'); await p.getByRole('button', { name: 'Revisar el expediente', exact: true }).click(); const response = await r; assert.equal(response.status(), 200, JSON.stringify(await response.json())); await p.locator('#carga-vista').waitFor(); }
async function confirmarManual() { const r = p.waitForResponse(r => r.url().endsWith('/confirmar') && r.request().method() === 'POST'); await p.getByRole('button', { name: 'Confirmar el expediente', exact: true }).click(); const response = await r; assert.equal(response.status(), 200, JSON.stringify(await response.json())); await p.locator('#carga-resultado').waitFor(); return response.json(); }
async function subir(nombre, buffer) { await p.locator('#carga-hoja').setInputFiles({ name: nombre, mimeType: nombre.endsWith('xlsx') ? 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' : 'text/csv', buffer }); const respuesta = p.waitForResponse(r => r.url().endsWith('/api/carga-2026/hoja') && r.request().method() === 'POST'); await p.getByRole('button', { name: 'Subir y revisar la hoja', exact: true }).click(); return respuesta; }
try {
 await entrar('ines.moreno'); await manual(); await p.getByRole('button', { name: 'Revisar el expediente', exact: true }).click(); await p.getByRole('alert').waitFor(); assert.equal(await p.evaluate(() => document.activeElement.getAttribute('role')), 'alert');
 await p.getByRole('alert').getByRole('link').first().click(); assert.equal(await p.evaluate(() => document.activeElement.id), 'carga-equipo');
 await p.getByRole('button', { name: 'Ayuda sobre la carga de 2026' }).click(); await p.keyboard.press('Escape'); assert.equal(await p.evaluate(() => document.activeElement.textContent), 'Ayuda sobre la carga de 2026');
 const usados = new Set(); let desplazamiento = 0; let hayMas;
 do { const lista = await leer(`/api/equipo/expedientes?limite=100&desplazamiento=${desplazamiento}`); assert.equal(lista.estado, 200); for (const fila of lista.datos.filas) usados.add(fila.numero); hayMas = lista.datos.hay_mas; desplazamiento += 100; } while (hayMas);
 const numeros = Array.from({ length: 99 }, (_, i) => String(i + 1).padStart(2, '0')).filter(n => !usados.has(`02-${n}-26`)).slice(0, 5); assert.equal(numeros.length, 5, 'La demo necesita cinco números libres del ETF 02 para repetir este recorrido.');
 for (const [i, estado] of ['aceptada', 'en_intervencion', 'cerrada', 'no_aceptada'].entries()) {
  await completar(estado, numeros[i]); if (i === 1) { await captura('carga-alta-escritorio.png', 1440); await captura('carga-alta-movil.png', 390); await p.setViewportSize({ width: 1440, height: 1100 }); }
  await revisar(); assert.equal(await p.getByText('Clara Villena Robles', { exact: true }).count(), 0);
  if (i === 1) { await p.getByText('04/05/2026', { exact: false }).waitFor(); await captura('carga-revision-escritorio.png', 1440); await captura('carga-revision-movil.png', 390); }
  const d = await confirmarManual(); assert.equal(d.expedientes.length, 1); assert.equal(d.marca, 'expediente.cargado_para_revision');
  if (estado !== 'no_aceptada') { const ficha = await leer('/api/equipo/expedientes/' + d.expedientes[0]); assert.equal(ficha.estado, 200); assert.equal(ficha.datos.estado, estado); const derivacion = await leer('/api/coordinacion/derivaciones/' + d.expedientes[0]); assert.equal(derivacion.estado, 200); assert.equal(derivacion.datos.cargado_para_revision, true); }
 }
 await p.goto(base + '/coordinacion/carga-2026'); await p.locator('#carga-equipo option').nth(1).waitFor({ state: 'attached' }); const opciones = (await leer('/api/carga-2026/opciones?idioma=es')).datos; const equipo = opciones.equipos.find(e => e.codigo === '02'); await p.locator('#carga-equipo').selectOption(equipo.id);
 await captura('carga-hoja-escritorio.png', 1440); await captura('carga-hoja-movil.png', 390);
 const descarga = p.waitForEvent('download'); await p.getByRole('button', { name: 'Descargar una hoja con datos inventados' }).click(); const archivo = await descarga; const originalCsv = await readFile(await archivo.path()); const csv = Buffer.from(originalCsv.toString().replaceAll('-91-26', `-${numeros[3]}-26`).replaceAll('-92-26', `-${numeros[4]}-26`)); assert.ok(csv.toString().includes('en_intervencion')); assert.ok(csv.toString().includes('cerrada')); assert.ok(csv.toString().includes('no_aceptada'));
 const invalida = Buffer.from(csv.toString().replace('2026-02-04', 'fecha-pendiente')); const r = await subir('carga-pendiente.csv', invalida); assert.equal(r.status(), 422); await p.locator('#carga-vista').waitFor(); assert.equal(await p.getByRole('button', { name: 'Confirmar el lote completo' }).count(), 0); assert.equal(await p.getByText('Clara Villena Robles', { exact: true }).count(), 0); await captura('carga-incidencias-escritorio.png', 1440); await captura('carga-incidencias-movil.png', 390);
 await p.getByRole('button', { name: 'Elegir otra hoja', exact: true }).click(); const rr = await subir('carga-2026.csv', csv); assert.equal(rr.status(), 200, JSON.stringify(await rr.json())); await p.locator('#carga-vista').waitFor(); const informe = await rr.json(); assert.equal(informe.casos, 3); assert.equal(informe.filas.length, 3); assert.ok(informe.filas[0].iniciales.includes('C.')); assert.equal(informe.filas[2].numero, '');
 let primero; let clave; await p.route('**/api/carga-2026/hoja/*/confirmar', async ruta => { clave = ruta.request().headers()['idempotency-key']; const respuesta = await ruta.fetch(); primero = await respuesta.json(); await ruta.abort('failed'); }, { times: 1 });
 await p.getByRole('button', { name: 'Confirmar el lote completo', exact: true }).click(); await p.getByRole('button', { name: 'Recuperar el resultado', exact: true }).waitFor();
 const recuperacion = p.waitForResponse(r => r.url().includes('/hoja/') && r.url().endsWith('/confirmar')); await p.getByRole('button', { name: 'Recuperar el resultado', exact: true }).click(); const recuperado = await recuperacion; assert.equal(recuperado.request().headers()['idempotency-key'], clave); assert.deepEqual(await recuperado.json(), primero); await p.locator('#carga-resultado').waitFor(); assert.equal(primero.expedientes.length, 3); await captura('carga-resultado-escritorio.png', 1440); await captura('carga-resultado-movil.png', 390);
 await p.getByRole('button', { name: 'Preparar otra carga', exact: true }).click();
 // XLSX con un caso no aceptado, a partir de la misma cabecera descargada.
 const temp = tmpdir(); const csvRuta = `${temp}/ca1-hoja.csv`; await writeFile(csvRuta, csv);
 const xlsx = execFileSync('python3', ['-c', `import sys,csv,io,zipfile,html
r=list(csv.reader(open(sys.argv[1],encoding='utf-8-sig'),delimiter=';'));r=[r[0],r[-1]]
f=['<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>']
for i,row in enumerate(r,1):
 f.append('<row r="%d">'%i)
 for j,v in enumerate(row):f.append('<c r="%s%d" t="inlineStr"><is><t>%s</t></is></c>'%(chr(65+j),i,html.escape(v)))
 f.append('</row>')
f.append('</sheetData></worksheet>')
p={'[Content_Types].xml':'<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/></Types>','_rels/.rels':'<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>','xl/workbook.xml':'<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Casos" sheetId="1" r:id="rId1"/></sheets></workbook>','xl/_rels/workbook.xml.rels':'<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>','xl/worksheets/sheet1.xml':''.join(f)}
b=io.BytesIO()
with zipfile.ZipFile(b,'w',zipfile.ZIP_DEFLATED) as z:
 for k,v in p.items():z.writestr(k,v)
sys.stdout.buffer.write(b.getvalue())`, csvRuta]);
 const xr = await subir('carga-2026.xlsx', xlsx); assert.equal(xr.status(), 200); await p.locator('#carga-vista').waitFor(); const xc = p.waitForResponse(r => r.url().endsWith('/confirmar') && r.request().method() === 'POST'); await p.getByRole('button', { name: 'Confirmar el lote completo', exact: true }).click(); assert.equal((await xc).status(), 200); await p.locator('#carga-resultado').waitFor();
 await p.getByRole('button', { name: 'Preparar otra carga', exact: true }).click();
 for (const ancho of [1024, 360, 390]) { await p.setViewportSize({ width: ancho, height: 1100 }); await comprobar(); }
 await preferencias(); await p.locator('#tema').selectOption('alto-contraste'); await cerrarPreferencias(); await captura('carga-alto-contraste-movil.png', 390);
 await preferencias(); await p.locator('#idioma').selectOption('en'); await p.getByRole('heading', { name: 'Load from a spreadsheet', exact: true }).waitFor(); await cerrarPreferencias(); await comprobar();
 await preferencias(); await p.locator('#idioma').selectOption('es'); await p.getByRole('heading', { name: 'Cargar desde una hoja', exact: true }).waitFor(); await preferencias(); await p.locator('#tema').selectOption('expedientes-etf'); await cerrarPreferencias();
 await salir(); await entrar('marta.cabrera'); await manual(); const own = (await leer('/api/carga-2026/opciones?idioma=es')).datos; assert.equal(own.equipos.length, 1); assert.equal(own.equipos[0].codigo, '02');
 const otro = opciones.equipos.find(e => e.id !== own.equipos[0].id); assert(otro);
 const denegado = await p.evaluate(async ({ equipo, csv }) => { const sesion = await (await fetch('/api/sesion')).json(); return (await fetch('/api/carga-2026/hoja', { method: 'POST', headers: { 'Content-Type': 'text/csv', 'X-CSRF-Token': sesion.csrf, 'X-Equipo-Carga': equipo }, body: csv })).status; }, { equipo: otro.id, csv: csv.toString() }); assert.equal(denegado, 403);
 await completar('no_aceptada', 0); await revisar(); const guardadoMarta = await confirmarManual(); const lecturaMarta = p.waitForResponse(r => r.url().includes('/api/expedientes/' + guardadoMarta.expedientes[0])); await p.getByRole('button', { name: 'Revisar los datos guardados del expediente 1', exact: true }).click(); assert.equal((await lecturaMarta).status(), 200); await p.getByRole('heading', { name: 'Datos guardados del expediente', exact: true }).waitFor(); assert.equal(await p.getByText('Cargado para revisión', { exact: true }).count(), 2); await captura('carga-no-aceptado-profesional-escritorio.png', 1440); await captura('carga-no-aceptado-profesional-movil.png', 390); await comprobar(); await p.getByRole('link', { name: 'Abrir la revisión del expediente 1', exact: true }).click(); await p.getByRole('heading', { name: 'Revisar un expediente cargado', exact: true }).waitFor(); await p.getByText('Sofía Molina Daza', { exact: true }).waitFor(); await p.reload(); await p.getByText('Sofía Molina Daza', { exact: true }).waitFor(); assert.equal(await p.getByText('Cargado para revisión', { exact: true }).count(), 1); await comprobar();
 await salir(); await entrar('elena.salas'); assert.equal((await leer('/api/carga-2026/opciones?idioma=es')).estado, 403); assert.equal(await p.evaluate(async () => (await fetch('/coordinacion/carga-2026')).status), 403);
 assert.deepEqual(errores, []); console.log('CA1_OK: altas aceptada/intervención/cerrada/no aceptada, reapertura, CSV/XLSX, incidencias, recuperación idempotente, revisión recuperable, permisos, teclado, 1440/1024/390/360, temas e idiomas.');
} finally { await navegador.close(); }
