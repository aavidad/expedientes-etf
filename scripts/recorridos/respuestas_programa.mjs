// Comprueba las respuestas del Programa con datos inventados y Chrome local.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
const base = process.argv[2];
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const browser = await chromium.launch({ executablePath:'/usr/bin/google-chrome',headless:true });
try {
 const page = await browser.newPage({viewport:{width:1440,height:1000}});
 await page.goto(base);
 await page.locator('#cuenta-ad').fill('ines.moreno');
 await page.locator('#contrasena-ad').fill(claves['ines.moreno']);
 const acceso=page.waitForResponse(r=>r.url()===base+'/api/sesion'&&r.request().method()==='POST');
 await page.locator('button[type="submit"]').click();
 assert.equal((await acceso).status(),200);
 await page.locator('#cuenta-ad').waitFor({state:'detached'});
 async function pedir(ruta,metodo='GET',datos,archivo) {
  return page.evaluate(async ({ruta,metodo,datos,archivo})=>{
   const sesion=await (await fetch('/api/sesion')).json();
   const headers={'X-CSRF-Token':sesion.csrf,'Idempotency-Key':crypto.randomUUID()};
   let body;
   if(archivo){body=new FormData();body.append('anexo',new Blob([new Uint8Array(archivo)]),archivo[0]===80&&archivo[1]===75?'derivacion.docx':'derivacion.pdf');}
   else if(datos){headers['Content-Type']='application/json';body=JSON.stringify(datos);}
   const r=await fetch(ruta,{method:metodo,headers,body});
   return {estado:r.status,datos:await r.json()};
  },{ruta,metodo,datos,archivo});
 }
 await page.goto(base+'/coordinacion/smi');
 await page.locator('#smi-2026').waitFor({timeout:10000}).catch(async e=>{ console.log('SMI diagnóstico:',page.url(),(await page.locator('body').innerText()).slice(0,900));throw e; });
 assert.equal(await page.locator('#smi-2026').inputValue(),'1221.00');
 assert.match(await page.locator('tr').filter({has:page.locator('#smi-2026')}).innerText(),/17.094/);
 const a4=await readFile(new URL('../../internal/adaptadores/anexos/pdf/testdata/anexo4.pdf',import.meta.url));
 const analisis=await pedir('/api/coordinacion/derivaciones','POST',undefined,Array.from(a4));
 assert.equal(analisis.estado,200);
 const a=analisis.datos;
 assert.equal(a.derivacion_datos.contenido.tipo,'anexo4');
 const u=a.derivacion_datos.opciones_ubicacion.municipios.find(m=>m.centro_id&&m.equipo_id);
 const creada=await pedir('/api/coordinacion/derivaciones/confirmar','POST',{
  lectura_id:a.lectura_anexo.id,tipo:'anexo4',
  campos:a.derivacion_datos.contenido.campos.map(c=>({...c,dudoso:false,motivos:[]})),
  ubicacion:{municipio_id:u.id,centro_id:u.centro_id,equipo_id:u.equipo_id},
 });
 assert.equal(creada.estado,201);
 await page.goto(base+'/coordinacion/derivaciones/'+creada.datos.id);
 await page.getByRole('button',{name:'Aceptar derivación',exact:true}).waitFor();
 assert.equal(await page.getByRole('button',{name:'Dejar en lista de espera',exact:true}).count(),0);
 const espera=await pedir(`/api/coordinacion/derivaciones/${creada.datos.id}/lista-espera`,'POST',{version:creada.datos.version});
 assert(espera.estado>=400&&espera.estado<500);
 assert.equal(espera.datos.error.clave,'expediente.espera_solo_comunitarios');
 await mkdir(new URL('../../docs/capturas/',import.meta.url),{recursive:true});
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:1000});
  await page.screenshot({path:new URL(`../../docs/capturas/resp1-spm-${width}.png`,import.meta.url).pathname,fullPage:true});
 }

 const a3=await readFile(new URL('../../internal/adaptadores/anexos/docx/testdata/anexo3.docx',import.meta.url));
 const lectura3=await pedir('/api/coordinacion/derivaciones','POST',undefined,Array.from(a3));
 assert.equal(lectura3.estado,200);
 const l3=lectura3.datos;
 const externo=l3.derivacion_datos.opciones_ubicacion.municipios.find(m=>m.fuera_ambito);
 assert(externo);
 const asignacion=l3.derivacion_datos.opciones_ubicacion.municipios.find(m=>m.centro_id&&m.equipo_id);
 const fuera=await pedir('/api/coordinacion/derivaciones/confirmar','POST',{
  lectura_id:l3.lectura_anexo.id,tipo:'anexo3',
  campos:l3.derivacion_datos.contenido.campos.map(c=>({...c,dudoso:false,motivos:[]})),
  ubicacion:{municipio_id:externo.id,centro_id:asignacion.centro_id,equipo_id:asignacion.equipo_id},
 });
 assert.equal(fuera.estado,201);
 const campos3=l3.derivacion_datos.contenido.campos;
 const aceptada=await pedir(`/api/coordinacion/derivaciones/${fuera.datos.id}/aceptar`,'POST',{
  version:fuera.datos.version,fecha_acta:new Date().toLocaleDateString('sv-SE',{timeZone:'Europe/Madrid'}),
  municipio_id:externo.id,centro_ss_id:asignacion.centro_id,equipo_id:asignacion.equipo_id,
  subprograma:'RIESGO_SOCIAL',profesional_referencia_puesto:'TS',profesional_referencia_id:null,
  menores:campos3.filter(c=>c.codigo==='menores.persona.nombre').map((_,i)=>({orden:i+1,sexo:i%2?'M':'H'})),
  descripcion_situacion:campos3.find(c=>c.codigo==='valoracion.descripcion')?.valor,
 });
 assert.equal(aceptada.estado,200,JSON.stringify(aceptada.datos));
 assert.equal(aceptada.datos.estado,'aceptada');

 const lecturaEspera=await pedir('/api/coordinacion/derivaciones','POST',undefined,Array.from(a3));
 assert.equal(lecturaEspera.estado,200);
 const le=lecturaEspera.datos;
 const esperaCreada=await pedir('/api/coordinacion/derivaciones/confirmar','POST',{
  lectura_id:le.lectura_anexo.id,tipo:'anexo3',
  campos:le.derivacion_datos.contenido.campos.map(c=>({...c,dudoso:false,motivos:[]})),
  ubicacion:{municipio_id:asignacion.id,centro_id:asignacion.centro_id,equipo_id:asignacion.equipo_id},
 });
 assert.equal(esperaCreada.estado,201);
 async function pendientes(equipo){
  const r=await pedir('/api/estadistica/memoria?anio=2026'+(equipo?'&etf='+equipo:''));
  assert.equal(r.estado,200,JSON.stringify(r.datos));
  return r.datos.apartados.find(a=>a.codigo==='6').total.familias;
 }
 const antesEquipo=await pendientes(asignacion.equipo_id),antesProvincia=await pendientes('');
 const enEspera=await pedir(`/api/coordinacion/derivaciones/${esperaCreada.datos.id}/lista-espera`,'POST',{version:esperaCreada.datos.version});
 assert.equal(enEspera.estado,200);
 assert.equal(await pendientes(asignacion.equipo_id),antesEquipo+1);
 assert.equal(await pendientes(''),antesProvincia+1);
 const esperando=await pedir('/api/coordinacion/derivaciones/'+esperaCreada.datos.id);
 assert.equal(esperando.datos.numero,null);
 const aceptarEspera=await pedir(`/api/coordinacion/derivaciones/${esperaCreada.datos.id}/aceptar`,'POST',{
  version:enEspera.datos.version,fecha_acta:new Date().toLocaleDateString('sv-SE',{timeZone:'Europe/Madrid'}),
  municipio_id:asignacion.id,centro_ss_id:asignacion.centro_id,equipo_id:asignacion.equipo_id,
  subprograma:'RIESGO_SOCIAL',profesional_referencia_puesto:'TS',profesional_referencia_id:null,
  menores:le.derivacion_datos.contenido.campos.filter(c=>c.codigo==='menores.persona.nombre').map((_,i)=>({orden:i+1,sexo:i%2?'M':'H'})),
  descripcion_situacion:le.derivacion_datos.contenido.campos.find(c=>c.codigo==='valoracion.descripcion')?.valor,
 });
 assert.equal(aceptarEspera.estado,200,JSON.stringify(aceptarEspera.datos));
 assert(aceptarEspera.datos.numero);
 assert.equal(aceptarEspera.datos.correlativo,aceptada.datos.correlativo+1);
 assert.equal(aceptarEspera.datos.fecha_acta,new Date().toLocaleDateString('sv-SE',{timeZone:'Europe/Madrid'}));
 assert.equal(await pendientes(asignacion.equipo_id),antesEquipo);
 assert.equal(await pendientes(''),antesProvincia);
 await page.goto(base+'/equipo/tabla-spaf');
 await page.locator('#spaf-equipo').waitFor();
 const opciones=await pedir('/api/estadistica/spaf/opciones');
 assert.equal(opciones.estado,200);
 const equipo=opciones.datos.equipos.find(e=>e.clave_texto==='equipo.02');
 await page.locator('#spaf-equipo').selectOption(equipo.id);
 await page.locator('#spaf-mes').fill('2026-09');
 await page.getByRole('columnheader',{name:'Municipio',exact:true}).waitFor();
 assert(!/ZTS/.test(await page.locator('main').innerText()));
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:1000});
  await page.screenshot({path:new URL(`../../docs/capturas/resp1-spaf-${width}.png`,import.meta.url).pathname,fullPage:true});
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 }

 const tabla=await pedir(`/api/estadistica/spaf?anio=2026&mes=9&etf=${equipo.id}`);
 assert.equal(tabla.estado,200);
 assert((tabla.datos.filas??[]).length>0);
 assert((tabla.datos.filas??[]).every(f=>f.municipio.Conocido && /^18[0-9]{3}$/.test(f.municipio.Valor)));
 const versiones={regla:tabla.datos.version_regla,fichas:Object.fromEntries((tabla.datos.filas??[]).map(f=>[f.expediente_id,f.version_ficha]))};
 for(const idioma of ['es','en']){
  const word=await page.evaluate(async({equipo,idioma,versiones})=>{
   const sesion=await(await fetch('/api/sesion')).json();
   const r=await fetch(`/api/equipos/${equipo}/tabla-spaf/2026/9.docx`,{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':sesion.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({idioma,versiones})});
   return {estado:r.status,version:r.headers.get('X-Plantilla-Version'),bytes:Array.from(new Uint8Array(await r.arrayBuffer()))};
  },{equipo:equipo.id,idioma,versiones});
  assert([200,201].includes(word.estado));assert.equal(word.version,'1.0.2');
  const comprobacion=JSON.parse(execFileSync('python3',['-c',`import sys,io,zipfile,xml.etree.ElementTree as E,json,re
with zipfile.ZipFile(io.BytesIO(sys.stdin.buffer.read())) as z:r=E.fromstring(z.read('word/document.xml'))
ns='{http://schemas.openxmlformats.org/wordprocessingml/2006/main}'
t=chr(10).join(''.join(n.text or '' for n in p.iter(ns+'t')) for p in r.iter(ns+'p'))
print(json.dumps({'zts':'ZTS' in t,'antiguos':bool(re.search(r'2[LJPMO]',t)),'marcadores':'{{' in t,'municipio':'Municipio' in t or 'Municipality' in t}))`],{input:Buffer.from(word.bytes),encoding:'utf8'}));
  assert.deepEqual(comprobacion,{zts:false,antiguos:false,marcadores:false,municipio:true});
 }
 console.log('RESP1-UI-OK: SMI 2026, SPM sin espera, espera en Memoria y aceptación real, municipio fuera de ámbito; SPAF y Word es/en sin ZTS ni anexos antiguos; 1440/390.');
} finally {await browser.close();}
