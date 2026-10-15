// Recorrido AN6 con datos inventados, Chrome del sistema y servicios de la demo.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
const origen = process.argv[2] ?? 'http://127.0.0.1:18246';
assert.match(origen, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
let pagina = await contexto.newPage(); pagina.setDefaultTimeout(20000);
const errores = []; pagina.on('pageerror', e => errores.push(e.message));
async function api(ruta, datos, anexo, metodo) {
  console.log('PASO: Petición API');
  return pagina.evaluate(async ({ ruta, datos, anexo, metodo }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const headers = { 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': crypto.randomUUID() };
    let body;
    if (anexo) { body = new FormData(); body.append('anexo', new Blob([new Uint8Array(anexo.contenido)], { type: anexo.tipo }), anexo.nombre); }
    else if (datos) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(datos); }
    const r = await fetch(ruta, { method: metodo ?? (datos || anexo ? 'POST' : 'GET'), headers, body });
    return { estado: r.status, datos: r.status === 204 ? null : await r.json() };
  }, { ruta, datos, anexo, metodo });
}
async function crear(nombre = 'anexo3') {
  console.log('PASO: Crear derivación');
  const esPDF = nombre === 'anexo4';
  const extension = esPDF ? 'pdf' : 'docx';
  const carpeta = esPDF ? 'pdf' : 'docx';
  const contenido = Array.from(await readFile(new URL(`../../internal/adaptadores/anexos/${carpeta}/testdata/${nombre}.${extension}`, import.meta.url)));
  const r = await api('/api/coordinacion/derivaciones', undefined, { nombre: `${nombre}.${extension}`, tipo: esPDF ? 'application/pdf' : 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', contenido });
  assert.equal(r.estado, 200, JSON.stringify(r.datos));
  const a = r.datos;
  const opciones = a.derivacion_datos.opciones_ubicacion;
  const candidato = a.derivacion_datos.propuesta_municipio.Candidatos?.[0];
  const municipio = opciones.municipios.find(m => m.id === candidato?.MunicipioID && m.centro_id && m.equipo_id) ?? opciones.municipios.find(m => m.centro_id && m.equipo_id);
  assert(municipio);
  const confirmada = await api('/api/coordinacion/derivaciones/confirmar', {
    lectura_id: a.lectura_anexo.id, tipo: a.derivacion_datos.contenido.tipo,
    campos: a.derivacion_datos.contenido.campos.map(c => ({ ...c, dudoso: false, motivos: [] })),
    ubicacion: { municipio_id: municipio.id, centro_id: municipio.centro_id, equipo_id: municipio.equipo_id },
  }); assert.equal(confirmada.estado, 201, JSON.stringify(confirmada.datos));
  const guardada = await api(`/api/coordinacion/derivaciones/${confirmada.datos.id}`); assert.equal(guardada.estado, 200);
  console.log('ANEXO:', nombre, 'menores', guardada.datos.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').length); return guardada.datos;
}

const sesiones = new Map();
async function entrar(cuenta) {
  if (sesiones.has(cuenta)) { pagina = sesiones.get(cuenta); return; }
  const c = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
  pagina = await c.newPage(); pagina.setDefaultTimeout(20000); pagina.on('pageerror', e => errores.push(e.message));
  await pagina.goto(origen);
  await pagina.locator('#cuenta-ad').fill(cuenta);
  await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
  await pagina.locator('button[type="submit"]').click();
  await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  sesiones.set(cuenta, pagina);
}
async function abrirPreferencias() {
  if (await pagina.locator('.persona-conectada').getAttribute('open')===null) await pagina.locator('.persona-conectada > summary').click();
  if (await pagina.locator('.preferencias').getAttribute('open')===null) await pagina.locator('.preferencias > summary').click();
}
async function ponerDestino() { const campo=pagina.locator('#anexo6-servicio'); if (await campo.evaluate(e=>e.tagName==='SELECT')) await campo.selectOption({index:1}); else await campo.fill('Centro de atención sanitaria'); }
async function ficha(id) { const r = await api(`/api/expedientes/${id}`); assert.equal(r.estado, 200); return r.datos; }

try {
  await entrar('ines.moreno');
  const d=await crear();
  const opciones=(await api('/api/coordinacion/decision/opciones?idioma=es')).datos;
  const equipo=opciones.equipos.find(e=>e.codigo==='02');assert(equipo);
  const municipio=opciones.municipios.find(m=>m.equipo_id===equipo.id && m.centro_ss_id);assert(municipio);
  const aceptada=await api(`/api/coordinacion/derivaciones/${d.id}/aceptar`,{version:d.version,fecha_acta:'2026-09-14',municipio_id:municipio.id,centro_ss_id:municipio.centro_ss_id,equipo_id:equipo.id,subprograma:'RIESGO_SOCIAL',profesional_referencia_puesto:'TS',profesional_referencia_id:null,menores:d.derivacion_datos.contenido.campos.filter(c=>c.codigo==='menores.persona.nombre').map((_,i)=>({orden:i+1,sexo:'M'})),descripcion_situacion:'La familia solicita coordinación sanitaria y apoyo educativo.'});
  assert.equal(aceptada.estado,200,JSON.stringify(aceptada.datos));
  await pagina.goto(`${origen}/coordinacion/expedientes/${d.id}/salidas`);
  await pagina.getByRole('link',{name:'Derivar a otro servicio · Anexo 6',exact:true}).click();
  await pagina.getByRole('heading',{name:'Preparar una nueva salida',exact:true}).waitFor();
  await pagina.getByRole('button',{name:'Guardar propuesta de derivación',exact:true}).click();
  await pagina.getByRole('heading',{name:'Revisa los campos pendientes',exact:true}).waitFor();
  assert.equal(await pagina.locator('#anexo6-servicio').getAttribute('aria-invalid'),'true');
  await ponerDestino();
  await pagina.locator('#anexo6-motivo').fill('Coordinación de la atención sanitaria de la familia.');
  const creadaEspera=pagina.waitForResponse(r=>r.url().endsWith('/derivaciones-salida') && r.request().method()==='POST');
  await pagina.getByRole('button',{name:'Guardar propuesta de derivación',exact:true}).click();
  const creadaRespuesta=await creadaEspera;assert.equal(creadaRespuesta.status(),200);
  const salida=await creadaRespuesta.json();
  await pagina.getByRole('button',{name:'Descargar Anexo 6',exact:true}).waitFor();
  const antes=await ficha(d.id);
  const tablaAntes=await api(`/api/estadistica/spaf?anio=2026&mes=9&etf=${equipo.id}`);assert.equal(tablaAntes.estado,200);assert(tablaAntes.datos.resumen);
  const memoriaAntes=await api(`/api/estadistica/memoria?anio=2026&etf=${equipo.id}`);assert.equal(memoriaAntes.estado,200);assert(Array.isArray(memoriaAntes.datos.apartados));

  const descargaEspera=pagina.waitForEvent('download');
  const wordEspera=pagina.waitForResponse(r=>r.url().endsWith('/documentos/anexo6'));
  await pagina.getByRole('button',{name:'Descargar Anexo 6',exact:true}).click();
  const word=await wordEspera;assert.equal(word.status(),201);assert(word.headers()['x-plantilla-version']);
  const descarga=await descargaEspera;assert.equal(await descarga.failure(),null);
  const despues=await ficha(d.id);assert.equal(despues.version,antes.version);assert.equal(despues.estado,antes.estado);
  await pagina.getByText('Versión de plantilla:',{exact:false}).waitFor();
  await pagina.locator('#anexo6-fecha').fill('2026-09-30');
  await pagina.getByRole('button',{name:'Confirmar la fecha de envío',exact:true}).click();
  const dialogo=pagina.getByRole('dialog');await dialogo.waitFor();
  await pagina.keyboard.press('Escape');await dialogo.waitFor({state:'detached'});
  assert.equal(await pagina.evaluate(()=>document.activeElement?.textContent),'Confirmar la fecha de envío');
  await pagina.getByRole('button',{name:'Confirmar la fecha de envío',exact:true}).click();
  const hitoEspera=pagina.waitForResponse(r=>r.url().endsWith('/hitos') && r.request().method()==='POST');
  await pagina.getByRole('dialog').getByRole('button',{name:'Confirmar la fecha de envío',exact:true}).click();
  const hito=await hitoEspera;assert.equal(hito.status(),200);
  await pagina.getByText('Fecha de envío confirmada: 30/09/2026',{exact:true}).first().waitFor();
  const f=(await api(`/api/equipo/expedientes/${d.id}`)).datos;
  assert.equal(f.salidas_servicio.filter(s=>s.id===salida.id).length,1);
  assert.equal(f.salidas_servicio.find(s=>s.id===salida.id).fecha,'2026-09-30');
  assert.equal(f.estado,antes.estado);assert(f.emisiones.some(e=>e.codigo==='salida/anexo6' && e.version_plantilla));
  const tablaDespues=await api(`/api/estadistica/spaf?anio=2026&mes=9&etf=${equipo.id}`);assert.equal(tablaDespues.estado,200);assert.deepEqual(tablaDespues.datos.resumen,tablaAntes.datos.resumen);
  const memoriaDespues=await api(`/api/estadistica/memoria?anio=2026&etf=${equipo.id}`);assert.equal(memoriaDespues.estado,200);assert.deepEqual(memoriaDespues.datos.apartados,memoriaAntes.datos.apartados);
  assert(f.historial.some(e=>e.codigo==='expediente.confirmar_salida_servicio'));
  await pagina.evaluate(await readFile(new URL('../../web/node_modules/axe-core/axe.min.js',import.meta.url),'utf8'));
  const accesibilidad=await pagina.evaluate(()=>axe.run('.anexo6',{runOnly:{type:'tag',values:['wcag2a','wcag2aa','wcag21aa']}}));assert.deepEqual(accesibilidad.violations.map(v=>v.id),[]);console.log('AN6-AXE-OK: sin infracciones WCAG A/AA en la pantalla');
  await mkdir(new URL('../../docs/capturas/',import.meta.url),{recursive:true});
  for (const width of [1440,1024,390,360,320]) {
    await pagina.setViewportSize({width,height:1000});
    assert(await pagina.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`desbordamiento ${width}`);
    if ([1440,390].includes(width)) await pagina.screenshot({path:new URL(`../../docs/capturas/anexo6-${width}.png`,import.meta.url).pathname,fullPage:true});
  }
  await pagina.setViewportSize({width:1440,height:1000});
  if (await pagina.getByRole('button',{name:'Preparar otra salida',exact:true}).count()) await pagina.getByRole('button',{name:'Preparar otra salida',exact:true}).click();
  await ponerDestino();await pagina.locator('#anexo6-motivo').fill('Apoyo en el seguimiento sanitario.');
  const actual=await ficha(d.id);
  const destinos=(await api('/api/anexo6/opciones?idioma=es')).datos;
  const destinoSolicitud=destinos.servicios.length ? {servicio_destino:destinos.servicios[0].codigo} : {servicio_destino:'',servicio_texto:'Atención sanitaria'};
  assert.equal((await api(`/api/expedientes/${d.id}/derivaciones-salida`,{tipo:'otro_servicio',version:actual.version,...destinoSolicitud,motivo:'Atención familiar complementaria.'})).estado,200);
  const conflictoEspera=pagina.waitForResponse(r=>r.url().endsWith('/derivaciones-salida') && r.request().method()==='POST');
  await pagina.getByRole('button',{name:'Guardar propuesta de derivación',exact:true}).click();assert.equal((await conflictoEspera).status(),409);
  assert.equal(await pagina.locator('#anexo6-motivo').inputValue(),'Apoyo en el seguimiento sanitario.');
  await pagina.getByRole('button',{name:'Ver datos actuales',exact:true}).click();await pagina.getByRole('button',{name:'Guardar propuesta de derivación',exact:true}).waitFor({state:'visible'});
  await pagina.getByRole('button',{name:'Guardar propuesta de derivación',exact:true}).waitFor();
  await pagina.waitForFunction(()=>!document.querySelector('button[type=submit]')?.disabled);
  const clavesEnvio=[];
  let avisoEnviado, liberarRespuesta;
  const enviadoServidor=new Promise(resolve=>{avisoEnviado=resolve;});
  const respuestaRetenida=new Promise(resolve=>{liberarRespuesta=resolve;});
  await pagina.route('**/derivaciones-salida',async route=>{clavesEnvio.push(route.request().headers()['idempotency-key']);await route.fetch();avisoEnviado();await respuestaRetenida;await route.abort('failed').catch(()=>{});},{times:1});
  await pagina.getByRole('button',{name:'Guardar propuesta de derivación',exact:true}).click();
  await enviadoServidor;
  // Cambiar el idioma con resultado incierto conserva el recibo para recuperarlo.
  await abrirPreferencias();await pagina.locator('#idioma').selectOption('en');
  await pagina.getByRole('button',{name:'Recover the same submission',exact:true}).waitFor();
  await pagina.waitForFunction(()=>document.activeElement?.matches('.persona-conectada > summary'));await abrirPreferencias();await pagina.locator('#idioma').selectOption('es');await pagina.keyboard.press('Escape');await pagina.keyboard.press('Escape');
  await pagina.getByRole('button',{name:'Recuperar el mismo envío',exact:true}).waitFor();
  liberarRespuesta();
  const recuperadaEspera=pagina.waitForResponse(r=>r.url().endsWith('/derivaciones-salida') && r.request().method()==='POST');
  await pagina.getByRole('button',{name:'Recuperar el mismo envío',exact:true}).click();const recuperada=await recuperadaEspera;assert.equal(recuperada.status(),200);assert.equal(recuperada.request().headers()['idempotency-key'],clavesEnvio[0]);
  await pagina.getByRole('button',{name:'Preparar otra salida',exact:true}).waitFor();
  const final=(await api(`/api/equipo/expedientes/${d.id}`)).datos;
  assert.equal(final.salidas_servicio.length,3);
  // Fallo de carga real interceptado: debe ofrecer reintento sin mostrar diagnóstico interno.
  await pagina.route('**/api/anexo6/opciones?*',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:{clave:'catalogo.no_disponible'}})}));
  await pagina.reload();await pagina.getByRole('button',{name:'Volver a cargar',exact:true}).waitFor();
  await pagina.unroute('**/api/anexo6/opciones?*');await pagina.getByRole('button',{name:'Volver a cargar',exact:true}).click();await pagina.getByRole('heading',{name:'Salidas a otros servicios',exact:true}).waitFor();
  await abrirPreferencias();await pagina.locator('#tema').selectOption('alto-contraste');await pagina.keyboard.press('Escape');await pagina.keyboard.press('Escape');
  await pagina.setViewportSize({width:390,height:1000});assert(await pagina.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  await pagina.screenshot({path:new URL('../../docs/capturas/anexo6-contraste-390.png',import.meta.url).pathname,fullPage:true});
  await abrirPreferencias();await pagina.locator('#idioma').selectOption('en');
  await pagina.getByRole('heading',{name:'Referrals to other services',exact:true}).waitFor();await pagina.keyboard.press('Escape');await pagina.keyboard.press('Escape');
  assert(await pagina.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  await pagina.screenshot({path:new URL('../../docs/capturas/anexo6-en-390.png',import.meta.url).pathname,fullPage:true});
  await entrar('marta.cabrera');assert.equal((await api(`/api/equipo/expedientes/${d.id}`)).estado,200);assert.equal((await api('/api/anexo6/opciones?idioma=es')).estado,200);
  const actualProfesional=(await api(`/api/equipo/expedientes/${d.id}`)).datos;
  const opcionesProfesional=(await api('/api/anexo6/opciones?idioma=es')).datos;
  const destinoProfesional=opcionesProfesional.servicios.length ? {servicio_destino:opcionesProfesional.servicios[0].codigo} : {servicio_destino:'',servicio_texto:'Servicio de apoyo educativo'};
  const creadaProfesional=await api(`/api/expedientes/${d.id}/derivaciones-salida`,{tipo:'otro_servicio',version:actualProfesional.version,...destinoProfesional,motivo:'Coordinación del acompañamiento educativo.'});assert.equal(creadaProfesional.estado,200);
  const wordProfesional=await pagina.evaluate(async ({id,salida})=>{const s=await(await fetch('/api/sesion')).json();const r=await fetch(`/api/expedientes/${id}/documentos/anexo6`,{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':s.csrf,'Idempotency-Key':crypto.randomUUID()},body:JSON.stringify({idioma:'es',salida_id:salida})});return {estado:r.status,plantilla:r.headers.get('X-Plantilla-Version')};},{id:d.id,salida:creadaProfesional.datos.id});assert.equal(wordProfesional.estado,201);assert(wordProfesional.plantilla);
  const fechadaProfesional=await api(`/api/expedientes/${d.id}/hitos`,{tipo:'derivacion_otro_servicio',salida_id:creadaProfesional.datos.id,version:creadaProfesional.datos.version_expediente,version_salida:creadaProfesional.datos.version,fecha:'2026-10-14'});assert.equal(fechadaProfesional.estado,200);
  await entrar('pablo.ruiz');assert([403,404].includes((await api(`/api/equipo/expedientes/${d.id}`)).estado));
  await entrar('lucia.navarro');assert.equal((await api('/api/anexo6/opciones?idioma=es')).estado,403);
  await entrar('elena.salas');assert.equal((await api('/api/anexo6/opciones?idioma=es')).estado,403);
  assert.deepEqual(errores,[]);
  console.log('AN6-CHROME-OK: propuesta, Word versionado, hito separado, conflicto, recuperación sin duplicados, perfiles, Memoria/SPAF conservadas, 1440/1024/390/360/320, contraste e inglés');
} finally { await navegador.close(); }
