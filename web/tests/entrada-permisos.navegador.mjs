// Recorrido F0.11: Chrome real, registro INT5 y expectativas independientes.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import { chromium } from 'playwright-core';

const [base, registro, fixture, evidencia] = process.argv.slice(2);
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
const reglas = JSON.parse(await readFile(registro));
const f = JSON.parse(await readFile(fixture));
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const textos = Object.assign({}, ...await Promise.all(['usuarios', 'identidad', 'interfaz'].map(async nombre => {
  try { return JSON.parse(await readFile(new URL(`../../idiomas/es/textos/${nombre}.json`, import.meta.url))); }
  catch (e) { if (e.code === 'ENOENT') return {}; throw e; }
})));
const t = clave => { assert(textos[clave], clave); return textos[clave]; };

// Esta política expresa las fronteras del producto; nunca lee regla.Perfiles.
function perfilesEsperados(p, metodo) {
  if (['/ayuda', '/api/ayuda/manuales', '/manuales/recursos/{recurso}', '/voz/{grupo}/{recurso}'].includes(p)) return ['administracion', 'coordinacion', 'profesional_etf', 'consulta'];
  if (p === '/api/carga-2026/plazo') return ['GET','HEAD'].includes(metodo) ? ['administracion','coordinacion','profesional_etf'] : ['administracion'];
  if (p === '/api/carga-2026/lotes') return ['coordinacion','profesional_etf'];
  if (p === '/api/carga-2026/lotes/{lote}/anular') return ['coordinacion'];
  if (['/equipo/anexo6', '/api/anexo6/opciones'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (['/coordinacion/carga-2026', '/equipo/expedientes/alta-historica'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (p.startsWith('/admin/') || p.startsWith('/api/admin/')) return ['administracion'];
  if (['/api/conservacion/pendientes', '/api/conservacion/{id}/proponer', '/api/conservacion/{id}/anonimizar'].includes(p)) return ['administracion'];
  if (p === '/coordinacion' || p.startsWith('/coordinacion/') || p.startsWith('/api/coordinacion/')) return ['coordinacion'];
  if (p === '/api/documentos/emisiones') return ['coordinacion', 'profesional_etf'];
  if (p.startsWith('/api/documentos/')) return ['coordinacion'];
  if (p === '/api/expedientes/{id}') return ['GET', 'HEAD'].includes(metodo) ? ['coordinacion', 'profesional_etf'] : ['profesional_etf'];
  if (/\/documentos\/anexo[12]$/.test(p)) return ['coordinacion', 'profesional_etf'];
  if (p === '/api/expedientes/historicos' || p.startsWith('/api/carga-2026')) return ['coordinacion', 'profesional_etf'];
  if (p === '/api/expedientes/{id}/evaluaciones') return ['coordinacion', 'profesional_etf'];
  if (p === '/api/expedientes/{id}/hitos') return ['coordinacion', 'profesional_etf'];
  if (p === '/api/expedientes/{id}/salida-spm/forzado') return ['coordinacion'];
  if (['/api/expedientes/{id}/cerrar', '/api/expedientes/{id}/reabrir', '/api/expedientes/{id}/documentos/acta_cierre'].includes(p)) return ['coordinacion'];
  if (['/api/expedientes/{id}/trasladar', '/api/expedientes/{id}/seguimiento/reabrir'].includes(p)) return ['coordinacion'];
  if (['/api/seguimiento/opciones', '/api/avisos', '/api/expedientes/{id}/seguimiento', '/api/expedientes/{id}/seguimiento/terminar'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (['/api/expedientes/{id}/derivaciones-salida', '/api/expedientes/{id}/documentos/anexo6'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (p.startsWith('/api/expedientes/{id}/salida-spm')) return ['coordinacion', 'profesional_etf'];
  if (p === '/api/equipos/{etf}/tabla-spaf/{anio}/{mes}') return ['coordinacion', 'profesional_etf'];
  if (['/equipo/tabla-spaf', '/equipo/tabla-spaf/expedientes/{id}', '/api/estadistica/spaf/opciones'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (p === '/api/equipos/{equipo}/avisos/recuento') return ['coordinacion', 'profesional_etf'];
  if (p.startsWith('/api/expedientes/')) return ['GET', 'HEAD'].includes(metodo) ? ['coordinacion', 'profesional_etf'] : ['profesional_etf'];
  if (['/api/consulta/memoria/{anio}', '/api/consulta/memoria/{anio}/word'].includes(p)) return ['consulta'];
  if (['/memoria/desglose', '/api/memoria/{anio}/desglose/{apartado}', '/api/memoria/{anio}', '/api/memoria/{anio}/cifras/{apartado}'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (['/memoria', '/api/estadistica/memoria', '/api/estadistica/memoria/equipos'].includes(p)) return ['coordinacion', 'profesional_etf', 'consulta'];
  if (['/api/estadistica/memoria/reabrir', '/api/estadistica/memoria/discrepancias'].includes(p)) return ['coordinacion'];
  if (['/api/estadistica/memoria/estado', '/api/estadistica/memoria/cerrar', '/api/estadistica/memoria/instantanea'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (['/api/estadistica/spaf', '/api/estadistica/memoria/narrativas', '/api/estadistica/memoria/word'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (/^\/api\/(comisiones|informes)(\/|$)/.test(p)) return ['GET', 'HEAD'].includes(metodo) ? ['coordinacion', 'profesional_etf'] : ['profesional_etf'];
  if (p === '/equipo/comisiones' || p.startsWith('/api/actividad/')) return ['coordinacion', 'profesional_etf'];
  if (['/equipo', '/equipo/expediente', '/equipo/menores', '/equipo/intervencion', '/equipo/seguimiento', '/api/equipo/expedientes', '/api/equipo/expedientes/{id}', '/api/equipo/etiquetas/{idioma}'].includes(p)) return ['coordinacion', 'profesional_etf'];
  if (['/manifest.webmanifest', '/service-worker.js', '/api/sesion', '/favicon.ico', '/salud', '/api/idiomas', '/api/idiomas/{codigo}/textos', '/api/temas',
    '/api/temas/{nombre}/recursos/{recurso...}', '/api/temas/{nombre}/tokens', '/{$}', '/entrar', '/pendiente', '/assets/{recurso...}'].includes(p)) return [];
  assert.fail(`Ruta nueva sin expectativa independiente: ${p}`);
}
function camino(p) {
  let id = f.expediente;
  if (p.startsWith('/api/admin/usuarios/')) id = f.pendiente;
  if (p.startsWith('/api/documentos/')) id = f.emision;
  if (p.startsWith('/api/comisiones/')) id = f.comision;
  if (p.startsWith('/api/informes/')) id = f.informe;
  const valores = { grupo: 'ayuda', recurso: 'manifiesto.json', id, cambio: randomUUID(), lote: randomUUID(), etf: f.equipo, equipo: f.equipo, anio: '2026', apartado: '1', mes: '10.docx', ambito: 'asignacion_municipios', codigo: 'es', idioma: 'es', nombre: 'expedientes-etf', tipo: 'idioma', ticket: randomUUID(), 'recurso...': 'no-existe.css', '$': '' };
  let ruta = p.replace(/\{([^}]+)\}/g, (_, k) => { assert(k in valores, k); return valores[k]; });
  if (p.startsWith('/api/expedientes/') && p.endsWith('/historia')) ruta += '?corte=2026-10-14' + (p.includes('/familia/') ? '&anio=2026' : '');
  if (p.startsWith('/api/consulta/memoria/')) ruta += `?equipo=${f.equipo}`;
  if (p.startsWith('/api/memoria/')) ruta += `?etf=${f.equipo}`;
  if (p === '/api/memoria/{anio}/desglose/{apartado}') ruta += '&celda=-1&medida=familias';
  if (p === '/api/carga-2026/opciones') ruta += '?idioma=es';
  if (p === '/api/seguimiento/opciones') ruta += '?idioma=es';
  if (p === '/api/coordinacion/anexos/etiquetas') ruta += '?idioma=es';
  if (/\/catalogos\/.*\/(datos|vigente)$/.test(p)) ruta += '?fecha=2026-10-13';
  if (p === '/api/comisiones' || p === '/api/informes') ruta += `?anio=2026&etf=${f.equipo}`;
  if (p === '/api/estadistica/memoria') ruta += `?anio=2026&etf=${f.equipo}`;
  if (p === '/api/estadistica/spaf') ruta += `?anio=2026&mes=9&etf=${f.equipo}`;
  if (['narrativas', 'estado', 'cerrar', 'reabrir', 'instantanea', 'discrepancias'].some(accion => p === `/api/estadistica/memoria/${accion}`)) ruta += `?anio=2026&etf=${f.equipo}`;
  if (/^\/api\/(comisiones|informes)\/\{id\}$/.test(p)) ruta += '?version_esperada=1';
  return ruta;
}
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contextos = [];
const errores = [];
async function pagina(cuenta, width = 1024) {
  const contexto = await navegador.newContext({ viewport: { width, height: 900 } });
  contextos.push(contexto);
  await contexto.route('**/*', ruta => new URL(ruta.request().url()).origin === base ? ruta.continue() : ruta.abort());
  const page = await contexto.newPage(); page.setDefaultTimeout(15000);
  page.on('pageerror', e => errores.push(e.message));
  let peticionesLimitadas = false;
  page.on('response', respuesta => { if (respuesta.status() === 429) peticionesLimitadas = true; });
  await page.goto(base + '/entrar');
  try {
    await page.locator('#cuenta-ad').waitFor();
  } catch (error) {
    if (!peticionesLimitadas) throw error;
    console.log('Matriz INT5: espera antes de cargar el acceso por límite de peticiones.');
    await new Promise(resolve => setTimeout(resolve, 60000));
    await page.goto(base + '/entrar');
    await page.locator('#cuenta-ad').waitFor();
  }
  if (cuenta) {
    await page.locator('#cuenta-ad').fill(cuenta);
    await page.locator('#contrasena-ad').fill(claves[cuenta]);
    const respuesta = page.waitForResponse(r => r.url() === base + '/api/sesion' && r.request().method() === 'POST');
    await page.locator('#contrasena-ad').press('Tab'); await page.keyboard.press('Enter');
    assert.equal((await respuesta).status(), 200);
    await page.locator('#cuenta-ad').waitFor({ state: 'detached' });
  }
  if (!cuenta) await page.locator('#cuenta-ad').waitFor();
  return page;
}
async function pedirUnaVez(page, ruta, metodo = 'GET', datos, csrf = true) {
  // La interceptación de red de Playwright impide fetch de favicon en Chrome.
  if (ruta === '/favicon.ico') {
    const r = await page.request.fetch(base + ruta, { method: metodo, maxRedirects: 0 });
    return { estado: r.status(), contenido: await r.text() };
  }
  return page.evaluate(async ({ ruta, metodo, datos, csrf }) => {
    const sesion = window.__f011Sesion ??= await (await fetch('/api/sesion')).json();
    const headers = { 'X-Perfil': 'administracion', 'X-Equipo': 'ajeno' };
    if (csrf) headers['X-CSRF-Token'] = sesion.csrf;
    if (ruta.startsWith('/api/admin/paquetes/')) headers['X-ETF-Paquetes'] = '1';
    if (!['GET', 'HEAD', 'DELETE'].includes(metodo)) headers['Content-Type'] = 'application/json';
    headers['Idempotency-Key'] = crypto.randomUUID();
    const respuesta = await fetch(ruta, { method: metodo, headers, redirect: 'manual',
      body: ['GET', 'HEAD', 'DELETE'].includes(metodo) ? undefined : JSON.stringify(datos ?? {}) });
    const contenido = respuesta.headers.get('Content-Type')?.includes('wordprocessingml')
      ? `DOCX (${(await respuesta.arrayBuffer()).byteLength} bytes)` : await respuesta.text();
    let json; try { json = JSON.parse(contenido); } catch {}
    return { estado: respuesta.status, contenido, json };
  }, { ruta, metodo, datos, csrf });
}
async function guardar(page) {
  const respuesta = page.waitForResponse(r => r.request().method() === 'PUT' && r.url().endsWith('/perfiles'));
  await page.getByRole('button', { name: t('usuarios.guardar'), exact: true }).click();
  assert.equal((await respuesta).status(), 200);
  await page.getByRole('status').filter({ hasText: t('usuarios.guardado') }).waitFor();
}
let admin;
// La matriz respeta el límite real por IP. Un rechazo previo al manejador
// no ejecuta la operación; se repite una vez al terminar la ventana.
async function pedir(page, ruta, metodo = 'GET', datos, csrf = true) {
  let resultado = await pedirUnaVez(page, ruta, metodo, datos, csrf);
  if (resultado.estado === 429 && (metodo === 'HEAD' || resultado.json?.error?.clave === 'identidad.peticiones_limitadas')) {
    console.log('Matriz INT5: espera de la ventana de peticiones por IP.');
    await new Promise(resolve => setTimeout(resolve, 60000));
    resultado = await pedirUnaVez(page, ruta, metodo, datos, csrf);
  }
  return resultado;
}

try {
  await mkdir(evidencia, { recursive: true });
  admin = await pagina('elena.salas');
  const propio = await pagina('pablo.ruiz');
  const fichaPropia = await pedir(propio, `/api/expedientes/${f.expediente}`);
  assert.equal(fichaPropia.estado, 200, fichaPropia.contenido); assert.equal(fichaPropia.json.id, f.expediente);
  const organismos = (await pedir(propio, '/api/actividad/organismos')).json;
  const organismo = organismos.find(o => !o.admite_texto).codigo;
  for (const [ruta, campo, datos] of [['/api/comisiones', 'comision', { fecha: '2026-10-13', observaciones: 'Coordinación de recursos municipales.' }],
    ['/api/informes', 'informe', { fecha: '2026-10-13', organismo }]]) {
    const resultado = await pedir(propio, ruta, 'POST', datos);
    assert.equal(resultado.estado, 201, resultado.contenido); assert.equal(resultado.json.equipo_id, f.equipo); f[campo] = resultado.json.id;
  }
  // La misma sesión pendiente recupera los perfiles al recargar. Se restituye al terminar.
  for (const [width, tema] of [[1024, 'expedientes-etf'], [390, 'alto-contraste']]) {
    const usuario = await pagina('sergio.campos', width);
    await usuario.locator('.pendiente').waitFor();
    const acceso = (await pedir(usuario, '/api/sesion')).json;
    assert.deepEqual(Object.keys(acceso).sort(), ['csrf', 'estado']);
    assert.equal(acceso.estado, 'pendiente_alta');
    await admin.setViewportSize({ width, height: 900 });
    await admin.goto(base + '/admin/usuarios');
    await admin.locator('.tabla-usuarios tbody tr').filter({ hasText: 'sergio.campos' }).getByRole('button').click();
    for (const perfil of ['coordinacion', 'profesional_etf', 'consulta', 'administracion']) {
      await admin.getByRole('checkbox', { name: t(`usuarios.perfil_${perfil}`), exact: true }).check();
    }
    await admin.locator('#equipo-usuario').selectOption(f.equipo);
    await guardar(admin);
    await usuario.reload();
    const activa = (await pedir(usuario, '/api/sesion')).json;
    assert.deepEqual(activa.perfiles.sort(), ['administracion', 'consulta', 'coordinacion', 'profesional_etf']);
    assert.equal(activa.equipo_id, f.equipo);
    assert(activa.pantallas.some(p => p.ruta === '/admin/usuarios'));
    await usuario.goto(base + '/admin/usuarios');
    await usuario.locator(`#tema option[value="${tema}"]`).waitFor({ state: 'attached' });
    const selectorTema = usuario.locator('#tema');
    if (await selectorTema.count()) {
      // Abre los detalles que contienen preferencias antes de interactuar.
      if (!await usuario.locator('.persona-conectada').evaluate(e => e.open)) await usuario.locator('.persona-conectada > summary').click();
      if (!await usuario.locator('.preferencias').evaluate(e => e.open)) await usuario.locator('.preferencias > summary').click();
      await selectorTema.selectOption(tema);
      await usuario.locator(`link[href="/api/temas/${tema}/tokens"]`).waitFor({ state: 'attached' });
    } else assert.fail(`Tema ausente: ${tema}`);
    await usuario.screenshot({ path: `${evidencia}/alta-${width}.png`, fullPage: true });
    for (const perfil of ['coordinacion', 'profesional_etf', 'consulta', 'administracion']) await admin.getByRole('checkbox', { name: t(`usuarios.perfil_${perfil}`), exact: true }).uncheck();
    await guardar(admin);
    const retirada = await usuario.reload(); assert.equal(retirada.status(), 403);
    await usuario.goto(base); await usuario.locator('.pendiente').waitFor();
    assert.equal((await pedir(usuario, '/api/admin/usuarios')).estado, 403);
    await usuario.context().close();
  }
  const casos = [['anonimo', null], ['sin_perfil', 'sergio.campos'], ['profesional_etf', 'marta.cabrera'],
    ['coordinacion', 'ines.moreno'], ['consulta', 'lucia.navarro'], ['administracion', 'elena.salas']];
  const cobertura = [];
  for (const [perfil, cuenta] of casos) {
    const page = await pagina(cuenta);
    // DELETE de sesión se ejecuta al final para conservar el actor del resto de rutas.
    const pares = reglas.flatMap(regla => regla.Metodos.map(metodo => ({ regla, metodo })))
      .sort((a, b) => Number(a.regla.Patron === '/api/sesion' && a.metodo === 'DELETE') - Number(b.regla.Patron === '/api/sesion' && b.metodo === 'DELETE'));
    if (perfil === 'consulta') {
      for (const parametro of ['composicion=true', 'narrativas=true', 'fuentes=true', 'perfil=coordinacion', 'revision=1']) {
        const rechazo = await pedir(page, `/api/consulta/memoria/2026?equipo=${f.equipo}&${parametro}`);
        assert.equal(rechazo.estado, 400, parametro);
        assert.equal(rechazo.json.error.clave, 'estadistica.memoria_filtro_invalido');
      }
      const wordLibre = await pedir(page, `/api/consulta/memoria/2026/word?equipo=${f.equipo}`, 'POST', { idioma: 'es', narrativas: true });
      assert.equal(wordLibre.estado, 400);
    }
    for (const { regla, metodo } of pares) {
      const p = regla.Patron, ruta = camino(p), esperados = perfilesEsperados(p, metodo);
      const permitido = !esperados.length || esperados.includes(perfil);
      let datos = {};
      if (p === '/api/consulta/memoria/{anio}/word') datos = { idioma: 'es' };
      if (p.startsWith('/api/expedientes/') && !['GET', 'HEAD'].includes(metodo)) datos = p.includes('/documentos/') ? { idioma: 'es', version: 2 } : { version: 2 };
      if (p.startsWith('/api/carga-2026') && perfil === 'profesional_etf') datos = { equipo_id: f.equipo };
      if (p === '/api/estadistica/memoria/word' && perfil === 'profesional_etf') datos = { anio: 2026, equipo_id: f.equipo, idioma: 'es' };
      if (['/api/estadistica/memoria/cerrar', '/api/estadistica/memoria/reabrir'].includes(p)) datos = { version: -1 };
      if (p === '/api/estadistica/memoria/narrativas' && metodo === 'PUT') datos = { version: -1 };
      if (p.endsWith('/evaluaciones') && perfil === 'profesional_etf') datos = { clave_idempotencia: randomUUID(), instrumento: {}, version_esperada: 0 };
      if (p === '/api/documentos/emisiones' && perfil === 'profesional_etf') datos = { tipo: 'salida/spm/anexo5', expediente_id: f.expediente, idioma: 'es' };
      if (p.startsWith('/api/equipos/') && perfil === 'profesional_etf') datos = { idioma: 'es', versiones: { regla: '1', fichas: {} } };
      if (p === '/api/expedientes/{id}' && metodo === 'PATCH') datos = { version: 2, vigente_desde: '2026-10-14', familia: { direccion: 'Calle Castaños, 12' } };
      if (p.endsWith('/menores') && metodo === 'PUT') datos = { version: 2, vigente_desde: '2026-10-14', menores: [] };
      if (p.endsWith('/intervencion') && metodo === 'PUT') datos = { version: 2, vigente_desde: '2026-10-14', intervencion: { listas: { factor_menor: [] } } };
      if (p.endsWith('/adultos') && perfil === 'profesional_etf') datos = { version: 2, vigente_desde: '2026-10-14', adultos: [] };
      if (p.endsWith('/derivaciones-salida') && perfil === 'profesional_etf') datos = { version: 2, tipo: 'otro_servicio', servicio_destino: 'SALUD', motivo: 'La familia requiere atención del centro de salud.' };
      if (p.endsWith('/documentos/anexo6') && perfil === 'profesional_etf') datos = { idioma: 'es', salida_id: f.salida };
      if (/\/documentos\/anexo[12]$/.test(p)) datos = perfil === 'profesional_etf' ? { idioma: 'es' } : {};
      if (p === '/api/expedientes/{id}/hitos' && metodo === 'POST') datos = perfil === 'profesional_etf' ? { version: 2, tipo: 'consentimiento', fecha: '2026-10-13' } : {};
      if (/^\/api\/(comisiones|informes)\/\{id\}$/.test(p) && metodo === 'PUT') datos = { fecha: '2026-10-13', version_esperada: 1, ...(p.startsWith('/api/informes') ? { organismo } : {}) };
      if (p === '/api/sesion' && ['PUT', 'DELETE'].includes(metodo)) {
        const sinCSRF = await pedir(page, ruta, metodo, datos, false);
        assert.equal(sinCSRF.estado, 403); assert.equal(sinCSRF.json.error.clave, 'identidad.csrf_no_valido');
      }
      const etiqueta = `${perfil} ${metodo} ${p}`;
      const respuesta = await pedir(page, ruta, metodo, datos).catch(e => { throw new Error(`${etiqueta}: ${e.message}`); });
      if (permitido && !(perfil === 'profesional_etf' && p.includes('/{id}')) && ['GET', 'HEAD'].includes(metodo) && ['/api/expedientes/{id}', '/api/coordinacion/expedientes/{id}', '/api/coordinacion/derivaciones/{id}', '/api/admin/usuarios', '/api/actividad/organismos'].includes(p)) {
        assert.equal(respuesta.estado, 200, `${etiqueta}: ${respuesta.contenido}`);
        if (metodo === 'GET' && p.includes('/{id}')) assert.equal(respuesta.json.id, f.expediente);
      }
      if (p === '/api/sesion' && metodo === 'PUT') {
        assert.equal(respuesta.estado, cuenta ? 200 : 401, etiqueta);
        if (!cuenta) assert.equal(respuesta.json.error.clave, 'identidad.sesion_no_valida', etiqueta);
        else assert(['activo', 'pendiente_alta'].includes(respuesta.json.estado), etiqueta);
      } else if (p === '/api/sesion' && metodo === 'POST') {
        assert.equal(respuesta.estado, 401, etiqueta);
        assert.equal(respuesta.json.error.clave, 'identidad.credenciales_invalidas', etiqueta);
      } else if (!permitido) {
        assert.equal(respuesta.estado, cuenta ? 403 : 401, `${etiqueta}: ${respuesta.contenido}`);
        if (metodo !== 'HEAD') assert.equal(respuesta.json.error.clave, cuenta ? 'identidad.acceso_denegado' : 'identidad.sesion_no_valida', etiqueta);

      } else if (p === '/api/ayuda/manuales') {
        assert.equal(respuesta.estado, 200, etiqueta);
        if (metodo === 'GET') {
          const esperado = { administracion: ['administracion'], coordinacion: ['coordinacion'], profesional_etf: ['profesionales'], consulta: [] };
          assert.deepEqual(respuesta.json.map(manual => manual.nombre), esperado[perfil], etiqueta);
        }
      } else if (perfil === 'consulta' && p.startsWith('/api/consulta/memoria/')) {
        assert.equal(respuesta.estado, 200, `${etiqueta}: ${respuesta.contenido}`);
        if (p === '/api/consulta/memoria/{anio}' && metodo === 'GET') {
          assert.equal(respuesta.json.anio, 2026);
          assert.equal(respuesta.json.apartados.length, 33);
          assert.equal(typeof respuesta.json.definitivas, 'boolean');
          assert(respuesta.json.fecha_cifras);
          const cuerpo = JSON.stringify(respuesta.json);
          assert(!cuerpo.includes(f.expediente), etiqueta);
          for (const campo of ['Narrativas', 'Fuentes', 'Composicion', 'narrativas', 'fuentes', 'composicion']) assert(!(campo in respuesta.json), etiqueta);
        }
      } else if (perfil === 'profesional_etf' && p.startsWith('/api/memoria/')) {
        assert.equal(respuesta.estado, 403, etiqueta);
      } else if (perfil === 'coordinacion' && p.startsWith('/api/memoria/')) {
        assert.equal(respuesta.estado, 200, etiqueta);
      } else if (perfil === 'coordinacion' && p.endsWith('/documentos/acta_cierre')) {
        // El expediente de la matriz está aceptado; todavía no existe Acta de Cierre.
        assert.equal(respuesta.estado, 403, etiqueta);
        assert.equal(respuesta.json.error.clave, 'documento.acceso_denegado', etiqueta);
      } else if (['coordinacion', 'profesional_etf'].includes(perfil) && p === '/api/seguimiento/opciones') {
        assert.equal(respuesta.estado, 200, `${etiqueta}: ${respuesta.contenido}`);
      } else if (perfil === 'coordinacion' && p === '/api/estadistica/spaf') {
        assert.equal(respuesta.estado, 200, `${etiqueta}: ${respuesta.contenido}`);
      } else if (perfil === 'coordinacion' && p === '/api/estadistica/memoria/narrativas' && ['GET', 'HEAD'].includes(metodo)) {
        assert.equal(respuesta.estado, 200, `${etiqueta}: ${respuesta.contenido}`);
      } else if (perfil === 'coordinacion' && ['/api/expedientes/{id}/avisos', '/api/equipos/{equipo}/avisos/recuento'].includes(p)) {
        assert.equal(respuesta.estado, 200, `${etiqueta}: ${respuesta.contenido}`);
      } else if (['coordinacion', 'profesional_etf'].includes(perfil) && ['/equipo', '/equipo/expediente', '/equipo/menores', '/equipo/intervencion', '/equipo/seguimiento', '/api/equipo/expedientes', '/api/equipo/expedientes/{id}', '/api/equipo/etiquetas/{idioma}'].includes(p)) {
        // El expediente de la matriz pertenece al ETF 4; Marta pertenece al ETF 2.
        const ajeno = perfil === 'profesional_etf' && p === '/api/equipo/expedientes/{id}';
        assert.equal(respuesta.estado, ajeno ? 404 : 200, `${etiqueta}: ${respuesta.contenido}`);
        if (metodo === 'GET' && p === '/api/equipo/expedientes' && perfil === 'profesional_etf') {
          const sesion = (await pedir(page, '/api/sesion')).json;
          assert(respuesta.json.filas.length > 0, etiqueta);
          assert(respuesta.json.filas.every(fila => fila.equipo_id === sesion.equipo_id), etiqueta);
          assert(!respuesta.json.filas.some(fila => fila.id === f.expediente), etiqueta);
        }
      } else if (p === '/api/carga-2026/opciones' && ['GET', 'HEAD'].includes(metodo)) {
        assert.equal(respuesta.estado, 200, `${etiqueta}: ${respuesta.contenido}`);
        if (metodo === 'GET' && perfil === 'profesional_etf') {
          const sesion = (await pedir(page, '/api/sesion')).json;
          assert.equal(respuesta.json.equipos.length, 1, etiqueta);
          assert.equal(respuesta.json.equipos[0].id, sesion.equipo_id, etiqueta);
        }
      } else if (permitido && p === '/api/estadistica/memoria/equipos') {
        assert.equal(respuesta.estado, 200, etiqueta);
        if (metodo === 'GET' && perfil === 'profesional_etf') {
          const sesion = (await pedir(page, '/api/sesion')).json;
          assert(respuesta.json.every(equipo => equipo.id === sesion.equipo_id), etiqueta);
        }
      } else if (perfil === 'profesional_etf' && (p === '/api/estadistica/memoria' || p.startsWith('/api/estadistica/memoria/'))) {
        assert.equal(respuesta.estado, 403, `${etiqueta}: ${respuesta.contenido}`);
        if (metodo !== 'HEAD') assert.equal(respuesta.json.error.clave, 'estadistica.memoria_acceso_denegado', etiqueta);
      } else if (p === '/api/carga-2026/plazo' && ['GET','HEAD'].includes(metodo)) {
        assert.equal(respuesta.estado,200,etiqueta);
        if(metodo==='GET') assert.deepEqual(Object.keys(respuesta.json).sort(),['abierta','dias_duplicados','fecha_cierre','version']);
      } else if (p === '/api/carga-2026/lotes' && ['GET','HEAD'].includes(metodo)) {
        assert.equal(respuesta.estado,200,etiqueta);
        if(metodo==='GET') { assert(Array.isArray(respuesta.json)); if(perfil==='profesional_etf') assert(respuesta.json.every(lote=>lote.equipo_id!==f.equipo),etiqueta); }
      } else if (perfil === 'profesional_etf' && (p.startsWith('/api/expedientes/') && p !== '/api/expedientes/historicos' || p.startsWith('/api/equipos/') || p === '/api/documentos/emisiones' || p.startsWith('/api/carga-2026'))) {
        // Hoja y actas validan el cuerpo: sin equipo ni fichero no leen datos.
        const validos = p.startsWith('/api/carga-2026/hoja') || p.startsWith('/api/carga-2026/actas/leer/') ? [400, 403, 404] : [403, 404];
        assert(validos.includes(respuesta.estado), `${etiqueta}: ${respuesta.contenido}`);
      } else if (perfil === 'profesional_etf' && ['/api/comisiones', '/api/informes', '/api/estadistica/spaf'].includes(p) && ['GET', 'HEAD'].includes(metodo)) {
        assert.equal(respuesta.estado, 403, etiqueta); // El filtro pide el ETF 4; Marta pertenece al 2.
      } else if (perfil === 'profesional_etf' && /^\/api\/(comisiones|informes)\/\{id\}$/.test(p) && ['PUT', 'DELETE'].includes(metodo)) {
        assert.equal(respuesta.estado, 404, etiqueta);
      } else {
        assert([200, 204, 400, 404, 409, 415, 422].includes(respuesta.estado), `${etiqueta}: ${respuesta.contenido}`);
        assert.notEqual(respuesta.json?.error?.clave, 'identidad.acceso_denegado', etiqueta);
      }
      if (!['GET', 'HEAD'].includes(metodo) && permitido && !(p === '/api/sesion' && metodo === 'DELETE')) {
        const sinCSRF = await pedir(page, ruta, metodo, datos, false);
        assert.equal(sinCSRF.estado, 403, `CSRF ${etiqueta}`);
        assert.equal(sinCSRF.json.error.clave, 'identidad.csrf_no_valido', etiqueta);
      }
      cobertura.push({ perfil, metodo, patron: p, estado: respuesta.estado });
    }
    for (const p of new Set(reglas.map(r => r.Patron))) {
      const declarados = reglas.filter(r => r.Patron === p).flatMap(r => r.Metodos);
      for (const metodo of ['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'TRACE'].filter(m => !declarados.includes(m))) {
        // Chrome prohíbe TRACE en fetch; este cliente comparte las cookies del contexto.
        const respuesta = await page.request.fetch(base + camino(p), { method: metodo, maxRedirects: 0 });
        assert.equal(respuesta.status(), 405, `${perfil} ${metodo} ${p}`);
      }
    }

    for (const ruta of ['/api/capacidad-inexistente', '/admin/capacidad-inexistente', '/api/admin/capacidad-inexistente']) {
      const respuesta = await pedir(page, ruta);
      assert.equal(respuesta.estado, 403, `${perfil} desconocida ${ruta}`);
      assert.equal(respuesta.json.error.clave, 'identidad.acceso_denegado');
    }
    await page.context().close();
  }
  assert.deepEqual(errores, []);
  await writeFile(`${evidencia}/cobertura.json`, JSON.stringify(cobertura, null, 2));
  await writeFile(fixture, JSON.stringify(f));
  console.log(`F0.11-CHROME-OK: entrada por teclado; pendiente; alta de cuatro perfiles; recarga; retirada; 1024/390; dos temas; ${cobertura.length} combinaciones INT5; CSRF y métodos.`);
} finally {
  if (admin) {
    // Sólo restaura a la persona sintética de este recorrido; no reintenta escrituras fallidas.
    const lista = await pedir(admin, '/api/admin/usuarios');
    const u = lista.json?.find(u => u.cuenta === 'sergio.campos');
    if (u?.perfiles.some(p => p.hasta === null)) {
      const r = await pedir(admin, `/api/admin/usuarios/${u.id}/perfiles`, 'PUT', { version: u.version, estado: 'activo', perfiles: [], equipo: null });
      assert.equal(r.estado, 200, r.contenido);
    }
  }
  await navegador.close();
}
