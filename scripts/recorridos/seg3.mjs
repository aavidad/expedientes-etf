import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { readFile } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
const base = process.argv[2];
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
// Actualiza únicamente los textos de carga de la demo inventada de este worktree.
execFileSync('python3',['-B','-c',`import importlib.util,sys
from pathlib import Path
raiz=Path(sys.argv[1])
spec=importlib.util.spec_from_file_location("demo_seg3",raiz/"despliegue/demo/demo.py")
demo=importlib.util.module_from_spec(spec);spec.loader.exec_module(demo)
if demo.URL!=sys.argv[2]: raise SystemExit("demo.ambito_incorrecto")
for idioma in ("es","en"):
 demo.compose("cp",str(raiz/"idiomas"/idioma/"textos/carga.json"),"app:/datos/idiomas/"+idioma+"/textos/carga.json",capture_output=True,text=True)
`,fileURLToPath(new URL('../../',import.meta.url)),base],{timeout:30000});
const cuentas = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const t = JSON.parse(await readFile(new URL('../../idiomas/es/textos/carga.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];
async function entrar(cuenta) {
 const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1100 } });
 const p = await contexto.newPage(); p.setDefaultTimeout(20000); p.on('pageerror', e => errores.push(e.message));
 await p.goto(base); await p.locator('#cuenta-ad').fill(cuenta); await p.locator('#contrasena-ad').fill(cuentas[cuenta]); await p.locator('button[type=submit]').click(); await p.locator('#cuenta-ad').waitFor({ state: 'detached' }); return p;
}
async function api(p, ruta, metodo = 'GET', cuerpo, clave) {
 return p.evaluate(async ({ ruta, metodo, cuerpo, clave }) => {
  const headers = { 'Content-Type': 'application/json' };
  if (metodo !== 'GET') { const s = await (await fetch('/api/sesion')).json(); headers['X-CSRF-Token'] = s.csrf; }
  if (clave) headers['Idempotency-Key'] = clave;
  const r = await fetch(ruta, { method: metodo, headers, body: cuerpo ? JSON.stringify(cuerpo) : undefined }); return { estado: r.status, datos: await r.json() };
 }, { ruta, metodo, cuerpo, clave });
}
async function comprobar(p, ancho) {
 await p.setViewportSize({ width: ancho, height: 1100 });
 await p.evaluate(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))));
 assert.ok(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `desbordamiento ${ancho}`);
 for (const campo of await p.locator('main input, main select').all()) if (await campo.isVisible()) { assert.ok(await campo.evaluate(e => e.labels?.length)); assert.ok(await campo.evaluate(e => e.getBoundingClientRect().height >= 44)); }
 assert.equal(await p.evaluate(() => localStorage.length + sessionStorage.length), 0);
}
async function captura(p, nombre) {
 for (const ancho of [1440, 390]) { await comprobar(p, ancho); await p.screenshot({ path: new URL(`../../docs/capturas/carga-seg3-${nombre}-${ancho}.png`, import.meta.url).pathname, fullPage: true }); }
 await comprobar(p, 360); await comprobar(p, 1024); await p.setViewportSize({ width: 1440, height: 1100 });
}
let admin;
try {
 admin = await entrar('elena.salas');
 await admin.goto(base + '/admin/carga-historica'); await admin.locator('#carga-fecha-cierre').waitFor();
 await admin.locator('#carga-dias-duplicados').fill('367'); await admin.getByRole('button', { name: t['carga.guardar_plazo'], exact: true }).click(); await admin.getByText(t['carga.dias_invalidos'], { exact: true }).first().waitFor();
 assert.ok(await admin.locator('#carga-dias-duplicados').getAttribute('aria-describedby'));
 await admin.locator('#carga-dias-duplicados').fill('30'); await admin.locator('#carga-fecha-cierre').fill('2026-01-01');
 const guardar = admin.waitForResponse(r => new URL(r.url()).pathname === '/api/carga-2026/plazo' && r.request().method() === 'PUT');
 await admin.getByRole('button', { name: t['carga.guardar_plazo'], exact: true }).click(); assert.equal((await guardar).status(), 200);
 const coord = await entrar('ines.moreno');
 await coord.goto(base + '/equipo/expedientes/alta-historica'); await coord.getByText(/Carga histórica cerrada/).waitFor();
 await captura(coord, 'cerrada');
 const opciones = (await api(coord, '/api/carga-2026/opciones?idioma=es')).datos;
 const equipo = opciones.equipos.find(e => e.codigo === '02'); assert.ok(equipo);
 const municipio = opciones.municipios.find(m => m.equipo_id === equipo.id); assert.ok(municipio);
 const caso = { numero: '', fecha_derivacion: '2026-01-15', estado: 'en_intervencion', derivacion_datos: { esquema_version: '1', contenido: { tipo: 'anexo3', campos: [{ codigo: 'familia.nombre', valor: 'Familia Cárdenas Vera' }, { codigo: 'menores.persona.nombre', valor: 'Mateo Cárdenas Vera' }] } }, aceptacion: { version: 1, equipo_id: equipo.id, fecha_acta: '2026-02-03', municipio_id: municipio.id, centro_ss_id: municipio.centro_ss_id, subprograma: 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', descripcion_situacion: 'La familia necesita organizar los cuidados y mantener la asistencia escolar.', menores: [{ orden: 1, sexo: 'H' }] }, episodios: [{ apertura_fecha: '2026-02-03' }], familia: [], menores: [] };
 const carga = { equipo_id: equipo.id, casos: [caso], confirmar_duplicados: true };
 assert.equal((await api(coord, '/api/carga-2026', 'POST', carga)).datos.error.clave, 'carga.plazo_cerrado');
 let plazo = (await api(admin, '/api/carga-2026/plazo')).datos;
 let r = await api(admin, '/api/carga-2026/plazo', 'PUT', { fecha_cierre: '2026-12-31', version: plazo.version, dias_duplicados: 30 }); assert.equal(r.estado, 200);
 // Conserva la propuesta cuando otra sesión ha avanzado la versión.
 await admin.locator('#carga-fecha-cierre').fill('2026-12-20');
 await admin.getByRole('button', { name: t['carga.guardar_plazo'], exact: true }).click(); await admin.getByRole('button', { name: t['carga.actualizar_plazo'], exact: true }).waitFor();
 await admin.getByRole('button', { name: t['carga.actualizar_plazo'], exact: true }).click(); await admin.locator('#carga-fecha-cierre').waitFor(); assert.equal(await admin.locator('#carga-fecha-cierre').inputValue(), '2026-12-20');
 await admin.locator('#carga-fecha-cierre').fill('2026-12-31'); await admin.getByRole('button', { name: t['carga.guardar_plazo'], exact: true }).click(); await admin.getByText(t['carga.plazo_guardado'], { exact: true }).waitFor();
 await captura(admin, 'plazo');
 r = await api(coord, '/api/carga-2026', 'POST', carga); assert.equal(r.estado, 200, JSON.stringify(r.datos));
 let recibo = await api(coord, `/api/carga-2026/${r.datos.lote_id}/confirmar`, 'POST', carga, `seg3-${r.datos.lote_id}`); assert.equal(recibo.estado, 200, JSON.stringify(recibo.datos));
 r = await api(coord, '/api/carga-2026', 'POST', carga); assert.equal(r.estado, 200); assert.ok(r.datos.avisos.length);
 assert.equal((await api(coord, `/api/carga-2026/${r.datos.lote_id}/confirmar`, 'POST', { ...carga, confirmar_duplicados:false }, `seg3-${r.datos.lote_id}`)).datos.error.clave, 'carga.duplicados_sin_confirmar');
 recibo = await api(coord, `/api/carga-2026/${r.datos.lote_id}/confirmar`, 'POST', { ...carga, confirmar_duplicados: true }, `seg3-${r.datos.lote_id}`); assert.equal(recibo.estado, 200);
 await coord.goto(base + '/equipo/expedientes/alta-historica'); await coord.locator('#carga-equipo option').nth(1).waitFor({state:'attached'});
 await coord.locator('#carga-equipo').selectOption(equipo.id); await coord.locator('#carga-estado').selectOption('aceptada'); await coord.locator('#carga-tipo').selectOption('anexo3');
 await coord.locator('#carga-derivacion').fill('2026-01-15'); await coord.locator('#carga-aceptacion').fill('2026-02-03'); await coord.locator('#carga-municipio').selectOption(municipio.id);
 await coord.locator('#carga-puesto').selectOption('TS'); await coord.locator('#carga-descripcion').fill('La familia necesita apoyo para mantener la asistencia escolar.');
 await coord.locator('#carga-menor-nombre-0').fill('Mateo'); await coord.locator('#carga-menor-apellidos-0').fill('Cárdenas Vera'); await coord.locator('#carga-menor-sexo-0').selectOption('H');
 await coord.getByRole('button',{name:t['carga.revisar_manual'],exact:true}).click(); await coord.locator('#carga-vista').waitFor();
 await coord.getByRole('checkbox',{name:t['carga.confirmar_duplicados'],exact:true}).waitFor();
 await coord.getByRole('button',{name:t['carga.confirmar_expediente'],exact:true}).click(); await coord.getByRole('link',{name:t['carga.revisar_coincidencias'],exact:true}).click();
 assert.equal(await coord.evaluate(()=>document.activeElement?.id),'carga-confirmar-duplicados'); await captura(coord,'duplicados');
 await coord.locator('#carga-confirmar-duplicados').focus(); await coord.keyboard.press('Space'); assert.ok(await coord.locator('#carga-confirmar-duplicados').isChecked());
 const confirmada=coord.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/confirmar'));
 await coord.getByRole('button',{name:t['carga.confirmar_expediente'],exact:true}).click(); const respuesta=await confirmada;assert.equal(respuesta.status(),200);recibo={estado:200,datos:await respuesta.json()};await coord.locator('#carga-resultado').waitFor();
 await coord.goto(base + '/coordinacion/carga-lotes'); await coord.getByRole('button', { name: t['carga.anular_lote'], exact: true }).first().waitFor();
 await captura(coord, 'lotes');
 assert.ok(recibo.datos.referencia > 0); const tarjeta=coord.locator('.carga-lista>li').filter({has:coord.getByRole('heading',{name:t['carga.lote_referencia'].replace('{numero}',new Intl.NumberFormat('es').format(recibo.datos.referencia)),exact:true})}); await tarjeta.getByRole('button', { name: t['carga.anular_lote'], exact: true }).click(); await coord.locator('dialog').waitFor();
 await coord.locator('dialog').getByRole('button', { name: t['carga.anular_lote'], exact: true }).click(); await coord.getByText(t['carga.marcar_confirmacion'], { exact: true }).waitFor();
 await coord.getByRole('checkbox', { name: t['carga.confirmar_anulacion'], exact: true }).check(); await captura(coord, 'anular');
 const ruta = base + `/api/carga-2026/lotes/${recibo.datos.lote_id}/anular`;
 await coord.route(ruta, async route => { const respuesta = await route.fetch(); assert.equal(respuesta.status(), 200); await route.abort(); }, { times: 1 });
 await coord.locator('dialog').getByRole('button', { name: t['carga.anular_lote'], exact: true }).click(); await coord.getByText(t['carga.anulacion_incierta'], { exact: true }).waitFor();
 for (const boton of await coord.getByRole('button', { name: t['carga.anular_lote'], exact: true }).all()) assert.ok(await boton.isDisabled());
 await coord.getByRole('button', { name: t['carga.actualizar_lotes'], exact: true }).click(); await coord.getByText(t['carga.estado_anulado'], { exact: true }).first().waitFor();
 assert.equal((await api(coord, `/api/expedientes/${recibo.datos.expedientes[0]}`)).estado, 404);
 await coord.bringToFront(); await coord.getByRole('button',{name:t['carga.actualizar_lotes'],exact:true}).focus();
 const refrescoTeclado=coord.waitForResponse(r=>r.request().method()==='GET'&&new URL(r.url()).pathname==='/api/carga-2026/lotes');
 await coord.keyboard.press('Enter');assert.equal((await refrescoTeclado).status(),200);await coord.waitForFunction(()=>document.activeElement?.tagName==='H1');
 // Preferencias reales: alto contraste e idioma inglés.
 await coord.locator('.persona-conectada > summary').click(); await coord.locator('.preferencias > summary').click(); await coord.locator('#tema').selectOption('alto-contraste'); await coord.locator('#idioma').selectOption('en');
 await coord.locator('.preferencias > summary').focus(); await coord.keyboard.press('Escape'); await coord.locator('.persona-conectada > summary').focus(); await coord.keyboard.press('Escape');
 await captura(coord, 'contraste-en');
 // Paginación de interfaz con metadatos inventados; escrituras anteriores reales.
 const actuales=(await api(coord,'/api/carga-2026/lotes')).datos;const lotes=Array.from({length:101},(_,i)=>({...actuales[0],id:`00000000-0000-4000-8000-${String(i).padStart(12,'0')}`,referencia:i+1,anulado:true}));
 let fallarPagina=true;
 await coord.route('**/api/carga-2026/lotes?desplazamiento=*',async route=>{
  const offset=Number(new URL(route.request().url()).searchParams.get('desplazamiento'));
  if(offset===100&&fallarPagina){fallarPagina=false;await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:{clave:'carga.error'}})});return;}
  await route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(lotes.slice(offset,offset+100))});
 });
 await coord.goto(base+'/coordinacion/carga-lotes');await coord.waitForFunction(()=>document.querySelectorAll('.carga-lista>li').length===100);
 const en=JSON.parse(await readFile(new URL('../../idiomas/en/textos/carga.json',import.meta.url)));
 await coord.getByRole('button',{name:new RegExp([t['carga.mas_lotes'],en['carga.mas_lotes']].join('|'))}).click();await coord.locator('[role=alert]').waitFor();
 await coord.getByRole('button',{name:new RegExp([t['carga.mas_lotes'],en['carga.mas_lotes']].join('|'))}).click();await coord.waitForFunction(()=>document.querySelectorAll('.carga-lista>li').length===101);
 await coord.locator('.persona-conectada > summary').click();await coord.locator('.preferencias > summary').click();await coord.locator('#idioma').selectOption(await coord.locator('#idioma').inputValue()==='es'?'en':'es');
 await coord.waitForFunction(()=>document.querySelectorAll('.carga-lista>li').length===100);
 assert.equal(errores.length, 0, errores.join('\n'));
 console.log('SEG3_CHROME_OK: plazo, conflicto sin perder propuesta, duplicados, anulación con respuesta perdida, teclado, 1440/1024/390/360, contraste, inglés y paginación con fallo recuperado y cambio de idioma.');
} finally {
 if (admin) { const p = (await api(admin, '/api/carga-2026/plazo')).datos; await api(admin, '/api/carga-2026/plazo', 'PUT', { fecha_cierre: '2026-12-31', version: p.version, dias_duplicados: 30 }); }
 await navegador.close();
}
