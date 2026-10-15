// Lector editorial: Chrome real, red limitada a los dos servidores locales.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createServer } from 'node:http';
import { readFile, mkdtemp, rm } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve, extname, join } from 'node:path';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const base = process.argv[2];
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
const raiz = resolve(import.meta.dirname, '../..');
const claves = JSON.parse(await readFile(join(raiz, 'despliegue/demo/.local/usuarios.json')));
const es = JSON.parse(await readFile(join(raiz, 'idiomas/es/textos/voz.json')));
const spaf = JSON.parse(await readFile(join(raiz, 'idiomas/es/textos/spaf.json')));
const temporal = await mkdtemp(join(process.env.TMPDIR || '/dev/shm/etf/tmp', 'voz-recorrido-'));
const audios = existsSync(join(raiz,'bin/voz/curso/manifiesto.json')) ? join(raiz,'bin/voz') : join(raiz,'bin/voz-simulada');
let curso = join(raiz, 'bin/curso/web');
if (!existsSync(join(curso,'index.html')) || !(await readFile(join(curso,'index.html'),'utf8')).includes('data-voz-pagina')) {
  execFileSync('python3', ['-B', 'scripts/voz/fixture_curso.py', '--salida', temporal, '--audios', audios], { cwd: raiz, stdio: 'pipe' });
  curso = join(temporal, 'web');
}
const tipos = { '.html':'text/html; charset=utf-8', '.js':'text/javascript; charset=utf-8', '.css':'text/css', '.mp3':'audio/mpeg', '.png':'image/png' };
const servidor = createServer(async (req,res) => {
  try {
    const ruta = resolve(curso, '.' + decodeURIComponent(new URL(req.url,'http://127.0.0.1').pathname));
    assert(ruta.startsWith(curso+'/'));
    const archivo = extname(ruta) ? ruta : join(ruta,'index.html');
    const contenido = await readFile(archivo);
    res.setHeader('Content-Type', tipos[extname(archivo)] || 'application/octet-stream');
    res.end(contenido);
  } catch { res.writeHead(404); res.end(); }
});
await new Promise(r => servidor.listen(0,'127.0.0.1',r));
const cursoBase = `http://127.0.0.1:${servidor.address().port}`;
const navegador = await chromium.launch({ executablePath:'/usr/bin/google-chrome', headless:true });
const axe = await readFile(join(raiz,'web/node_modules/axe-core/axe.min.js'),'utf8');
const capturas = process.env.ETF_VOZ_CAPTURAS === '1';
const fallos = [];
async function contexto() {
  const ctx = await navegador.newContext({ viewport:{width:1440,height:1000} });
  await ctx.addInitScript({ content: axe });
  await ctx.route('**/*', route => {
    const u=new URL(route.request().url());
    return [base,cursoBase].includes(u.origin) || ['data:','about:'].includes(u.protocol) ? route.continue() : route.abort();
  });
  return ctx;
}
async function entrar(cuenta) {
  const ctx=await contexto();const page=await ctx.newPage();
  page.on('pageerror', e => fallos.push(e.message));
  await page.goto(base);await page.locator('#cuenta-ad').fill(cuenta);await page.locator('#contrasena-ad').fill(claves[cuenta]);
  await page.locator('button[type="submit"]').click();await page.locator('#cuenta-ad').waitFor({state:'detached'});
  return page;
}
async function pedir(page) {
  return page.evaluate(async () => { const r = await fetch('/api/ayuda/manuales'); return { estado:r.status, datos:await r.json() }; });
}
async function foto(page,nombre,ancho) {
  await page.setViewportSize({width:ancho,height:1000});
  await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth+1);
  if(capturas) await page.screenshot({path:join(raiz,`docs/capturas/voz-${nombre}-${ancho}.png`)});
}
async function preferencias(page,tipo,valor) {
  await page.locator('.persona-conectada > summary').click();
  await page.locator('.preferencias > summary').click();
  await page.locator('#'+tipo).selectOption(valor);
  await page.waitForFunction(({tipo,valor}) => tipo==='idioma' ? document.documentElement.lang===valor : !!document.querySelector(`link[href='/api/temas/${valor}/tokens']`),{tipo,valor});
  await page.locator('.preferencias > summary').focus();await page.keyboard.press('Escape');
  await page.locator('.persona-conectada > summary').focus();await page.keyboard.press('Escape');
}
try {
  const page=await entrar('ines.moreno');
  await page.getByRole('link',{name:es['voz.titulo'],exact:true}).click();
  await page.getByRole('heading',{name:'Manual para Coordinación',exact:true}).waitFor();
  assert(await page.locator('.voz-control button').count()>0);
  assert(await page.locator('audio').evaluateAll(a=>a.every(x=>x.paused)));
  const primera=page.locator('#coordinacion-s1 > summary');
  await primera.focus();await page.keyboard.press('Enter');
  await page.locator('#coordinacion-s1 img').first().waitFor();
  await page.waitForFunction(()=>{const img=document.querySelector('#coordinacion-s1 img');return img?.complete && img.naturalWidth>0;});
  const boton=page.locator('#coordinacion-s1 .voz-control button');
  await boton.focus();await page.keyboard.press('Enter');
  await page.waitForFunction(()=>{const a=document.querySelector('#coordinacion-s1 audio');return a && !a.hidden && a.duration>0;});
  assert.equal((await pedir(page)).estado,200);
  for(const ancho of [1440,390,360,1024]) await foto(page,'ayuda',ancho);
  const accesibilidad=await page.evaluate(async()=> (await axe.run('.voz-ayuda')).violations.filter(v=>['serious','critical'].includes(v.impact)));
  assert.deepEqual(accesibilidad.map(v=>v.id),[]);
  await preferencias(page,'tema','alto-contraste');await foto(page,'contraste',390);
  await preferencias(page,'tema','expedientes-etf');
  await preferencias(page,'idioma','en');
  assert.equal(await page.locator('.voz-control').count(),0);
  assert.equal(await page.locator('.voz-manual').first().getAttribute('lang'),'es');
  await preferencias(page,'idioma','es');
  await page.goto(base+'/equipo/tabla-spaf');
  await page.getByRole('button',{name:spaf['spaf.abrir_ayuda'],exact:true}).click();
  const dialogo=page.locator('dialog[open]');
  await dialogo.getByRole('button',{name:es['voz.escuchar_aria'],exact:true}).waitFor();
  for(const ancho of [1440,390]) await foto(page,'contextual',ancho);
  await dialogo.getByRole('button',{name:es['voz.escuchar_aria'],exact:true}).click();
  await page.waitForFunction(()=>{const a=document.querySelector('dialog[open] audio');return a && a.duration>0;});
  await page.keyboard.press('Escape');assert.equal(await page.locator('.voz-contextual audio').count(),0);
  await page.goto(base+'/memoria');
  const ayudaMemoria=page.getByRole('button',{name:/Ayuda/}).first();
  await ayudaMemoria.click();await page.locator('.voz-contextual button').first().waitFor();
  await page.goto(base+'/ayuda');
  await page.route('**/voz/*/manifiesto.json',route=>route.fulfill({json:{}}));
  await page.reload();await page.getByRole('heading',{name:'Manual para Coordinación',exact:true}).waitFor();
  assert.equal(await page.locator('.voz-control button').count(),0);
  await page.unroute('**/voz/*/manifiesto.json');
  await page.route('**/api/ayuda/manuales',route=>route.fulfill({status:503,json:{error:{clave:'voz.manuales_no_disponibles'}}}));
  await page.reload();await page.getByRole('button',{name:es['voz.reintentar'],exact:true}).waitFor();
  await page.unroute('**/api/ayuda/manuales');await page.getByRole('button',{name:es['voz.reintentar'],exact:true}).click();
  await page.getByRole('heading',{name:'Manual para Coordinación',exact:true}).waitFor();
  for(const [cuenta,nombre] of [['marta.cabrera','profesionales'],['elena.salas','administracion'],['lucia.navarro',null]]) {
    const otra=await entrar(cuenta);await otra.goto(base+'/ayuda');
    const respuesta=await pedir(otra);const manuales=respuesta.datos;
    assert.equal(respuesta.estado,200);assert.deepEqual(manuales.map(m=>m.nombre),nombre?[nombre]:[]);
    if(!nombre) await otra.getByText(es['voz.vacio'],{exact:true}).waitFor();
    await otra.context().close();
  }
  const publico=await contexto();assert.equal((await publico.request.get(base+'/api/ayuda/manuales')).status(),401);await publico.close();
  const cursoCtx=await contexto();const cursoPage=await cursoCtx.newPage();
  await cursoPage.goto(cursoBase+'/index.html');
  await cursoPage.getByRole('button',{name:es['voz.escuchar_pagina'],exact:true}).waitFor();
  const enlaces=await cursoPage.locator('a[href]').evaluateAll(a=>a.map(x=>x.href));
  const conSecciones=enlaces.find(x=>x.includes('bienvenida.html'));
  if(conSecciones) await cursoPage.goto(conSecciones);
  else {const files=JSON.parse(await readFile(join(audios,'curso/manifiesto.json')));const id=Object.keys(files).find(x=>x.endsWith('-s2')).split('-s')[0];await cursoPage.goto(cursoBase+'/'+id+'.html');}
  assert(await cursoPage.locator('[data-voz-seccion]').count()>0);
  assert(await cursoPage.locator('audio').evaluateAll(a=>a.every(x=>x.paused)));
  for(const ancho of [1440,390]) await foto(cursoPage,'curso',ancho);
  await cursoPage.locator('[data-voz-pagina]').focus();await cursoPage.keyboard.press('Enter');
  await cursoPage.waitForFunction(()=>{const a=document.querySelector('.etf-voz-pagina audio');return a && a.duration>0;});
  const inicio=await cursoPage.locator('.etf-voz-pagina audio').getAttribute('src');
  await cursoPage.locator('.etf-voz-pagina audio').evaluate(a=>a.dispatchEvent(new Event('ended')));
  await cursoPage.waitForFunction(inicio=>document.querySelector('.etf-voz-pagina audio').getAttribute('src')!==inicio,inicio);
  await cursoPage.locator('[data-voz-seccion]').first().click();
  await cursoPage.waitForFunction(()=>document.querySelector('.etf-voz-pagina audio').paused);
  assert.deepEqual(fallos,[]);
  console.log('VOZ: curso, manuales por perfil, ayuda contextual, teclado, 1440/390/360/1024, inglés sin lector y audio ausente: OK.');
} finally {
  await navegador.close();await new Promise(r=>servidor.close(r));await rm(temporal,{recursive:true,force:true});
}
