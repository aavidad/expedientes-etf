import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
const base = process.argv[2] ?? 'http://127.0.0.1:18092';
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const corpus = new URL('../../internal/adaptadores/anexos/actas/testdata/', import.meta.url);
const capturas = new URL('../../docs/capturas/', import.meta.url); await mkdir(capturas, { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1100 } });
const p = await contexto.newPage(); p.setDefaultTimeout(20000); p.setDefaultNavigationTimeout(60000);
const errores = []; p.on('pageerror', e => errores.push(e.message));
async function entrar(cuenta) { await p.goto(base); await p.locator('#cuenta-ad').fill(cuenta); await p.locator('#contrasena-ad').fill(claves[cuenta]); await p.locator('button[type=submit]').click(); await p.locator('#cuenta-ad').waitFor({ state: 'detached' }); }
async function consultar(ruta) { return p.evaluate(async ruta => { const r = await fetch(ruta); return { estado: r.status, datos: await r.json() }; }, ruta); }
async function opciones() { const r = await consultar('/api/carga-2026/opciones?idioma=es'); assert.equal(r.estado, 200); return r.datos; }
async function alta() { await p.goto(base + '/equipo/expedientes/alta-historica'); await p.locator('#carga-equipo option').nth(1).waitFor({ state: 'attached' }); }
async function comprobar() {
 await p.evaluate(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))));
 assert.equal(await p.locator('h1').count(), 1);
 assert.ok(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth), JSON.stringify(await p.locator('body *').evaluateAll(es => es.filter(e => e.getBoundingClientRect().right > innerWidth + 1).slice(0,20).map(e => ({tag:e.tagName,id:e.id,clase:e.className,derecha:e.getBoundingClientRect().right})))));
 assert.equal(await p.evaluate(() => localStorage.length + sessionStorage.length), 0);
 for (const e of await p.locator('main input, main select, main textarea').all()) if (await e.isVisible()) { assert.ok(await e.evaluate(e => e.labels?.length)); assert.ok(await e.evaluate(e => e.getBoundingClientRect().height >= 44)); }
}
async function captura(nombre, ancho) { await p.setViewportSize({ width: ancho, height: 1100 }); await comprobar(); await p.screenshot({ path: new URL(nombre, capturas).pathname, fullPage: true }); }
async function subir(aceptacion, cierre) {
 await p.locator('#carga-acta-aceptacion').setInputFiles(new URL(aceptacion, corpus).pathname);
 if (cierre) await p.locator('#carga-acta-cierre').setInputFiles(new URL(cierre, corpus).pathname);
 const respuesta = p.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/api/carga-2026/actas/leer/' + (cierre ? 'cierre' : 'aceptacion')));
 await p.getByRole('button', { name: 'Leer y rellenar los datos', exact: true }).click();
 const r = await respuesta; assert.equal(r.status(), 200); const datos = await r.json(); assert.equal(datos.antivirus, 'limpio');
 await p.getByText(/Hemos rellenado lo que hemos podido leer/).waitFor();
}
async function separar(tipo, i, nombre, apellidos) {
 const campo = p.locator(`#carga-${tipo}-nombre-${i}`);
 await campo.locator('xpath=ancestor::fieldset').getByRole('button', { name: 'Separar nombre y apellidos', exact: true }).click();
 assert.equal(await campo.inputValue(), ''); await campo.fill(nombre); await p.locator(`#carga-${tipo}-apellidos-${i}`).fill(apellidos);
}
async function numeroLibre(anio, preferido) {
 const usados = new Set(); let desplazamiento = 0, mas;
 do { const r = await consultar(`/api/equipo/expedientes?limite=100&desplazamiento=${desplazamiento}`); assert.equal(r.estado, 200); const d = r.datos; for (const f of d.filas) usados.add(f.numero); mas = d.hay_mas; desplazamiento += 100; } while (mas);
 const pref = `02-${preferido}-${anio}`; if (!usados.has(pref)) return pref;
 const libre = Array.from({ length: 99 }, (_, i) => `02-${String(i + 1).padStart(2, '0')}-${anio}`).find(n => !usados.has(n)); assert.ok(libre); return libre;
}
try {
 await entrar('ines.moreno'); await alta(); const o = await opciones(); const equipo = o.equipos.find(e => e.codigo === '02'); assert.ok(equipo);
 await p.getByRole('button', { name: 'Leer y rellenar los datos', exact: true }).click(); await p.getByRole('alert').waitFor(); assert.equal(await p.evaluate(() => document.activeElement.getAttribute('role')), 'alert');
 await p.locator('#carga-equipo').selectOption(equipo.id);
 await captura('carga-actas-inicio-1440.png', 1440); await captura('carga-actas-inicio-390.png', 390); await p.setViewportSize({ width: 1440, height: 1100 });
 // Ambas actas de aceptación se leen antes de escoger el estado actual.
 const recibos = [];
 for (const [anio, archivo, cierre, numero, fecha] of [[2022, 'aceptacion-2022-firmada.pdf', undefined, '71', '2022-05-17'], [2023, 'aceptacion-2023.docx', 'cierre-2026.pdf', '72', '2023-03-09'], [2022, 'aceptacion-2022.odt', undefined, '71', '2022-05-17'], [2023, 'aceptacion-2023.odt', 'cierre-2026.odt', '72', '2023-03-09']]) {
  await alta(); await p.locator('#carga-equipo').selectOption(equipo.id); await subir(archivo, cierre);
  if (!cierre) { assert.equal(await p.locator('#carga-estado').inputValue(), ''); await p.locator('#carga-estado').selectOption('en_intervencion'); }
  assert.equal(await p.locator('#carga-numero').inputValue(), `02-${numero}-${String(anio).slice(2)}`);
  assert.equal(await p.locator('#carga-aceptacion').inputValue(), fecha);
  assert.equal(await p.locator('#carga-familiaNombre').inputValue(), 'Familia Villena Robles');
  assert.equal(await p.locator('#carga-menor-nombre-0').inputValue(), 'Clara Villena Robles');
  assert.equal(await p.locator('#carga-menor-fecha_nacimiento-0').inputValue(), '2014-06-14');
  assert.equal(await p.locator('#carga-adulto-sexo-0').inputValue(), '');
  await separar('adulto',0,'Irene','Robles Serrano'); await separar('adulto',1,'Daniel','Villena Márquez'); await separar('menor',0,'Clara','Villena Robles');
  if (cierre) { assert.equal(await p.locator('#carga-cierre-0').inputValue(), '2026-06-20'); await p.locator('#carga-motivo-0').selectOption('OBJ_TOTAL'); }
  await p.locator('#carga-tipo').selectOption('anexo3'); await p.locator('#carga-derivacion').fill(anio + '-02-01'); await p.locator('#carga-puesto').selectOption('TS');
  if (!await p.locator('#carga-municipio').inputValue()) { const m = o.municipios.find(m => m.equipo_id === equipo.id && m.ambito_diputacion); await p.locator('#carga-municipio').selectOption(m.id); }
  if (!await p.locator('#carga-centro').inputValue()) { const c = o.centros.find(c => o.textos[c.clave_texto] === 'Santa Fe'); await p.locator('#carga-centro').selectOption(c.id); }
  await p.locator('#carga-adulto-sexo-0').selectOption('M'); await p.locator('#carga-adulto-sexo-1').selectOption('H');
  const original = await numeroLibre(String(anio).slice(2), numero); await p.locator('#carga-numero').fill(original);
  if (!cierre) { await captura('carga-actas-datos-1440.png', 1440); await captura('carga-actas-datos-390.png', 390); await p.setViewportSize({ width: 1024, height: 1100 }); await comprobar(); await p.setViewportSize({ width: 360, height: 1100 }); await comprobar(); await p.setViewportSize({ width: 1440, height: 1100 }); }
  const previa = p.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/carga-2026');
  await p.getByRole('button', { name: 'Revisar el expediente', exact: true }).click(); const v = await previa; assert.equal(v.status(), 200, JSON.stringify(await v.json()));
  await p.locator('#carga-vista').waitFor(); assert.equal(await p.getByText('Clara Villena Robles', { exact: true }).count(), 0);
  const respuesta = p.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/confirmar'));
  if (await p.getByRole('checkbox', { name: 'He revisado las coincidencias y confirmo que son expedientes distintos.', exact: true }).count()) await p.getByRole('checkbox', { name: 'He revisado las coincidencias y confirmo que son expedientes distintos.', exact: true }).check(); await p.getByRole('button', { name: 'Confirmar el expediente', exact: true }).click(); const r = await respuesta; assert.equal(r.status(), 200, JSON.stringify(await r.json())); const d = await r.json(); recibos.push(d);
  assert.equal(d.marca, 'expediente.cargado_para_revision'); const id = d.expedientes[0];
  const ficha = await consultar('/api/equipo/expedientes/' + id); assert.equal(ficha.estado, 200); const datos = ficha.datos; assert.equal(datos.numero, original); assert.equal(datos.estado, cierre ? 'cerrada' : 'en_intervencion');
  const familia = await consultar('/api/expedientes/' + id + '/familia'); assert.equal(familia.estado, 200); const f = familia.datos; assert.equal(f.adultos.length, 2); assert.equal(f.adultos[0].nombre, 'Irene'); assert.equal(f.adultos[0].apellidos, 'Robles Serrano');
 }
 await captura('carga-actas-resultado-1440.png', 1440); await captura('carga-actas-resultado-390.png', 390);
 // Nueva sesión de profesional: opciones y lectura limitadas a su equipo.
 await contexto.clearCookies(); await entrar('marta.cabrera'); await alta(); const propias = await opciones(); assert.equal(propias.equipos.length, 1); assert.equal(propias.equipos[0].codigo, '02');
 const sesion = (await consultar('/api/sesion')).datos;
 const ajeno = o.equipos.find(e => e.codigo === '04'); const b = await readFile(new URL('aceptacion-2022.pdf', corpus));
 const denegada = await p.evaluate(async ({ csrf, equipo, contenido }) => { const f = new FormData(); f.append('anexo', new Blob([new Uint8Array(contenido)], { type: 'application/pdf' }), 'acta.pdf'); return (await fetch('/api/carga-2026/actas/leer/aceptacion', { method: 'POST', headers: { 'X-CSRF-Token': csrf, 'X-Equipo-Carga': equipo }, body: f })).status; }, { csrf: sesion.csrf, equipo: ajeno.id, contenido: [...b] });
 assert.equal(denegada, 403); await comprobar(); assert.deepEqual(errores, []);
 // Cancelar la sustitución conserva lo escrito y devuelve el foco.
 await p.locator('#carga-derivacion').fill('2022-04-01'); await p.locator('#carga-acta-aceptacion').setInputFiles(new URL('aceptacion-2022.pdf',corpus).pathname);
 await p.getByRole('button',{name:'Leer y rellenar los datos',exact:true}).click(); await p.getByRole('dialog').waitFor(); await p.keyboard.press('Escape'); assert.equal(await p.locator('#carga-derivacion').inputValue(),'2022-04-01'); assert.equal(await p.evaluate(() => document.activeElement.textContent),'Leer y rellenar los datos');
 // Alto contraste e inglés, sin modificar la propuesta ni guardar.
 await p.locator('#carga-derivacion').fill(''); await p.locator('.persona-conectada > summary').click(); await p.locator('.preferencias > summary').click(); await p.locator('#tema').selectOption('alto-contraste'); await p.locator('.preferencias > summary').focus(); await p.keyboard.press('Escape'); await p.locator('.persona-conectada > summary').focus(); await p.keyboard.press('Escape'); await captura('carga-actas-contraste-390.png',390);
 await p.locator('.persona-conectada > summary').click(); await p.locator('.preferencias > summary').click(); await p.locator('#idioma').selectOption('en');
 const descartar = p.getByRole('button', { name:'Descartar cambios',exact:true }); if (await descartar.isVisible()) await descartar.click();
 await p.getByRole('heading',{name:'Add a historical case file',exact:true}).waitFor(); await comprobar();
 assert.deepEqual(errores, []);
 console.log('RESP2_RESP5_CHROME_OK: actas ODT de aceptación y cierre confirmadas; aceptación firmada 2022, aceptación Word 2023 + cierre PDF 2026, datos confirmados y adultos en ficha; permisos propio equipo, sin almacenamiento, 1440/1024/390/360 y errores sin datos.');
} finally { await navegador.close(); }
