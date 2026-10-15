// AUD1: descarga real y permisos, exclusivamente contra una demo local.
import assert from 'node:assert/strict';
import { readFile, mkdir, mkdtemp, rm } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { chromium, baseLocal, restringir } from './comun.mjs';
const base = baseLocal(process.argv[2]);
const raiz = new URL('../../', import.meta.url);
const claves = JSON.parse(await readFile(new URL('despliegue/demo/.local/usuarios.json', raiz)));
const temporal = await mkdtemp('/dev/shm/etf/tmp/aud1-recorrido-');
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];
const hoy = new Date().toISOString().slice(0, 10);
async function entrar(cuenta) {
  const c = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'es-ES', acceptDownloads: true });
  await restringir(c, base); const p = await c.newPage();
  p.on('pageerror', e => errores.push(e.message));
  await p.goto(base); await p.locator('#cuenta-ad').fill(cuenta); await p.locator('#contrasena-ad').fill(claves[cuenta]);
  const respuesta=p.waitForResponse(r=>r.url()===base+'/api/sesion'&&r.request().method()==='POST');await p.locator('button[type=submit]').click();assert.equal((await respuesta).status(),200); await p.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return p;
}
async function pedir(p, datos, csrf = true) {
  return p.evaluate(async ({ datos, csrf }) => {
    const s = await (await fetch('/api/sesion')).json();
    const r = await fetch('/api/admin/auditoria/exportaciones', { method: 'POST', headers: { 'Content-Type': 'application/json', ...(csrf ? { 'X-CSRF-Token': s.csrf } : {}) }, body: JSON.stringify(datos) });
    return { sesion:s.estado, perfiles:s.perfiles, estado: r.status, tipo: r.headers.get('Content-Type') };
  }, { datos, csrf });
}
try {
  const coordinacion = await entrar('ines.moreno');
  const negativa=await pedir(coordinacion, { desde: hoy, hasta: hoy, formato: 'csv', idioma: 'es' });assert.equal(negativa.estado,403);
  assert.equal(await coordinacion.evaluate(async ()=>(await fetch('/admin/auditoria')).status),403);
  assert.equal(await coordinacion.getByRole('link', { name: 'Auditoría', exact: true }).count(), 0);
  await coordinacion.context().close();
  const p = await entrar('elena.salas');
  await p.getByRole('link', { name: 'Auditoría', exact: true }).click();
  await p.locator('#auditoria-titulo').waitFor(); await p.locator('#auditoria-usuario:not([disabled])').waitFor();
  await p.locator('#auditoria-desde').fill(hoy); await p.locator('#auditoria-hasta').fill(hoy);
  const datos = { desde: hoy, hasta: hoy, formato: 'json', idioma: 'es' };
  assert.equal((await pedir(p, datos, false)).estado, 403);
  assert.equal((await pedir(p, { ...datos, desde: '2026-01-01', hasta: '2026-02-10' })).estado, 400);
  assert.equal((await pedir(p, { ...datos, usuario_id: 'ines.moreno' })).estado, 400);
  assert.equal((await pedir(p, { ...datos, actor: { perfiles: ['administracion'] } })).estado, 400);
  execFileSync('go', ['build', '-o', temporal + '/verificar-exportacion', './cmd/verificar-exportacion'], { cwd: raiz });
  for (const formato of ['csv', 'json']) {
    await p.locator('#auditoria-formato').selectOption(formato);
    await p.locator('#auditoria-tipo').selectOption('identidad');
    const descargaPromesa = p.waitForEvent('download'); await p.getByRole('button', { name: 'Descargar registro y comprobación', exact: true }).click();
    const descarga = await descargaPromesa.catch(async e=>{console.log('AVISO',await p.locator('#auditoria-error').textContent().catch(()=>''));throw e;}); const archivo = temporal + '/' + formato + '.zip'; await descarga.saveAs(archivo);
    const comprobacion = JSON.parse(execFileSync(temporal + '/verificar-exportacion', [archivo], { encoding: 'utf8' }));
    assert.equal(comprobacion.clave, 'auditoria.exportacion_verificada'); assert(comprobacion.filas > 0);
    const t = comprobacion.tramo;
    assert.equal(JSON.parse(execFileSync(temporal + '/verificar-exportacion', [archivo, String(t.anterior.id), t.anterior.huella, String(t.final.id), t.final.huella], { encoding: 'utf8' })).clave, 'auditoria.exportacion_verificada');
    assert.throws(() => execFileSync(temporal + '/verificar-exportacion', [archivo, String(t.anterior.id), t.anterior.huella, String(t.final.id + 1), t.final.huella], { stdio: 'pipe' }));
  }
  await p.route('**/api/admin/auditoria/exportaciones',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:{clave:'auditoria.no_disponible'}})}),{times:1});
  await p.getByRole('button',{name:'Descargar registro y comprobación',exact:true}).click();await p.locator('#auditoria-error').waitFor();
  assert.equal(await p.locator('#auditoria-desde').inputValue(),hoy);assert.equal(await p.locator('#auditoria-formato').inputValue(),'json');
  const recuperadaPromesa=p.waitForEvent('download');await p.getByRole('button',{name:'Descargar registro y comprobación',exact:true}).click();await recuperadaPromesa;
  await mkdir(new URL('docs/capturas/', raiz), { recursive: true });
  await p.screenshot({ path: new URL('docs/capturas/auditoria-escritorio.png', raiz).pathname, fullPage: true });
  for (const width of [1024, 390, 360, 320]) {
    await p.setViewportSize({ width, height: 1000 });
    if(width<768)await p.locator('.cuerpo-app[data-compacto="true"]').waitFor();
    await p.evaluate(()=>new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r))));

    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'desbordamiento ' + width);
    if (width === 390) await p.screenshot({ path: new URL('docs/capturas/auditoria-movil.png', raiz).pathname, fullPage: true });
  }
  await p.setViewportSize({ width: 1440, height: 1000 });
  await p.locator('#auditoria-desde').fill('1995-01-01'); await p.locator('#auditoria-hasta').fill('1995-01-01');
  const vacioPromesa = p.waitForEvent('download'); await p.getByRole('button', { name: 'Descargar registro y comprobación', exact: true }).click();
  const vacio = await vacioPromesa; await vacio.saveAs(temporal + '/vacio.zip');
  assert.equal(JSON.parse(execFileSync(temporal + '/verificar-exportacion', [temporal + '/vacio.zip'], { encoding: 'utf8' })).filas, 0);
  await p.getByText('No hay eventos que cumplan tus filtros.', { exact: false }).waitFor();
  await p.getByRole('button', { name: 'Ayuda sobre la exportación de auditoría' }).focus(); await p.keyboard.press('Enter');
  assert.equal(await p.locator('#auditoria-ayuda').count(), 1);
  await p.keyboard.press('Escape');
  assert.equal(await p.locator('#auditoria-ayuda').count(), 0);
  assert(await p.getByRole('button', { name: 'Ayuda sobre la exportación de auditoría' }).evaluate(e => e === document.activeElement));
  await p.evaluate(await readFile(new URL('web/node_modules/axe-core/axe.min.js', raiz),'utf8'));
  const a = await p.evaluate(async () => (await window.axe.run('.auditoria')).violations.map(v => ({ id: v.id, impacto: v.impact })));
  assert.deepEqual(a, []);
  await p.locator('.persona-conectada > summary').click(); await p.locator('.preferencias > summary').click();
  await p.locator('#tema').selectOption('alto-contraste');
  await p.waitForFunction(()=>document.querySelector('link[href*="alto-contraste"]'));await p.evaluate(()=>new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r))));
  await p.locator('#idioma').selectOption('en'); await p.getByRole('heading',{name:'Audit log',exact:true}).waitFor();
  await p.keyboard.press('Escape'); await p.keyboard.press('Escape');
  for(const width of [1440,390,360,320]){await p.setViewportSize({width,height:1000});if(width<768)await p.locator('.cuerpo-app[data-compacto="true"]').waitFor();await p.evaluate(()=>new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r))));assert(await p.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));}
  const alto=await p.evaluate(async()=>(await window.axe.run('.auditoria')).violations.map(v=>v.id));assert.deepEqual(alto,[]);
  assert.deepEqual(errores, []);
  console.log('AUD1-CHROME-OK: coordinación 403; Administración CSV/JSON, cadena y sellos; vacío; límites; CSRF; teclado; axe; alto contraste/en; 1440/1024/390/360/320 sin desbordamiento');
} finally { await browser.close(); await rm(temporal, { recursive: true, force: true }); }
