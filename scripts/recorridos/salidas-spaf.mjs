// F3.06/F3.07: datos inventados por API, Chrome del sistema y cifras manuales.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { randomUUID } from 'node:crypto';
import { chromium, baseLocal, restringir } from './comun.mjs';

const [url, directorio] = process.argv.slice(2);
const base = baseLocal(url);
assert(directorio, 'Se necesita un directorio de evidencia fuera del repositorio');
const evidencia = resolve(directorio);
await mkdir(evidencia, { recursive: true });
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const textos = Object.assign({}, ...await Promise.all(['equipo', 'salidas', 'identidad'].map(async nombre =>
  JSON.parse(await readFile(new URL(`../../idiomas/es/textos/${nombre}.json`, import.meta.url))))));
const t = clave => { assert(textos[clave], clave); return textos[clave]; };
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const sesiones = new Map();
const csrfPorPagina = new Map();
const errores = [];
const pasos = [];
const fallosProducto = [];
let pagina;
let usuarioRestaurar;
const hoy = new Intl.DateTimeFormat('fr-CA', { timeZone: 'Europe/Madrid', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());

function ok(texto) { pasos.push(texto); console.log(texto); }
function producto(condicion, descripcion) {
  if (!condicion) { fallosProducto.push(descripcion); console.error(`FALLO-PRODUCTO: ${descripcion}`); }
}
async function entrar(cuenta) {
  if (sesiones.has(cuenta)) { pagina = sesiones.get(cuenta); return; }
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
  await restringir(contexto, base);
  await contexto.tracing.start({ screenshots: true, snapshots: true });
  pagina = await contexto.newPage(); pagina.setDefaultTimeout(20000);
  pagina.on('pageerror', error => errores.push(error.message));
  await pagina.goto(base);
  await pagina.locator('#cuenta-ad').fill(cuenta);
  await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
  const sesion = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/api/sesion'));
  await pagina.locator('button[type="submit"]').click();
  const respuesta = await sesion; assert.equal(respuesta.status(), 200);
  csrfPorPagina.set(pagina, (await respuesta.json()).csrf);
  await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  sesiones.set(cuenta, pagina);
}
async function pedir(ruta, metodo = 'GET', datos, clave = randomUUID(), anexo) {
  return pagina.evaluate(async ({ ruta, metodo, datos, clave, anexo, csrf }) => {
    const headers = { 'X-CSRF-Token': csrf, 'Idempotency-Key': clave };
    let body;
    if (anexo) {
      body = new FormData();
      body.append('anexo', new Blob([new Uint8Array(anexo.bytes)], { type: anexo.tipo }), anexo.nombre);
    } else if (datos !== undefined) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(datos); }
    const r = await fetch(ruta, { method: metodo, headers, body });
    if (r.headers.get('content-type')?.includes('wordprocessingml')) {
      const bytes = new Uint8Array(await r.arrayBuffer());
      const huella = Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256', bytes)), b => b.toString(16).padStart(2, '0')).join('');
      return { estado: r.status, emision: r.headers.get('x-emision-id'), huella, firma: Array.from(bytes.slice(0, 2)), longitud: bytes.length };
    }
    const texto = await r.text();
    return { estado: r.status, datos: texto ? JSON.parse(texto) : null };
  }, { ruta, metodo, datos, clave, anexo, csrf: csrfPorPagina.get(pagina) });
}
async function api(ruta, metodo = 'GET', datos, esperado = 200, clave, anexo) {
  const r = await pedir(ruta, metodo, datos, clave, anexo);
  assert.equal(r.estado, esperado, `${metodo} ${ruta}: ${JSON.stringify(r.datos)}`);
  return r.datos ?? r;
}
async function denegarEquipo(path, metodo = 'GET', datos) {
  const r = await pedir(path, metodo, datos);
  // RLS puede ocultar una fila ajena con 404; ambos estados impiden el acceso.
  assert([403, 404].includes(r.estado), `${metodo} ${path}: HTTP ${r.estado}`);
  assert.deepEqual(Object.keys(r.datos), ['error']);
  assert.equal(typeof r.datos.error.clave, 'string');
}
const ficha = id => api(`/api/expedientes/${id}`);
const ruta = (id, accion) => `/api/expedientes/${id}/${accion}`;
async function estado(id, esperado) { assert.equal((await ficha(id)).estado, esperado); }
async function crear(anexo, equipo, fecha) {
  await entrar('ines.moreno');
  const pdf = anexo === 'anexo4';
  const nombre = `${anexo}.${pdf ? 'pdf' : 'docx'}`;
  const bytes = Array.from(await readFile(new URL(`../../internal/adaptadores/anexos/${pdf ? 'pdf' : 'docx'}/testdata/${nombre}`, import.meta.url)));
  const a = await api('/api/coordinacion/derivaciones', 'POST', undefined, 200, undefined,
    { nombre, tipo: pdf ? 'application/pdf' : 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', bytes });
  const m = opciones.municipios.find(m => m.equipo_id === equipo && m.centro_ss_id);
  assert(m);
  const d = await api('/api/coordinacion/derivaciones/confirmar', 'POST', {
    lectura_id: a.lectura_anexo.id, tipo: a.derivacion_datos.contenido.tipo,
    campos: a.derivacion_datos.contenido.campos.map(c => ({ ...c, dudoso: false, motivos: [] })),
    ubicacion: { municipio_id: m.id, centro_id: m.centro_ss_id, equipo_id: equipo },
  }, 201);
  await aceptar(d.id, equipo, fecha);
  assert.equal(a.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').length,
    pdf ? 3 : 6, 'La prueba manual exige seis menores en Anexo 3 y tres en Anexo 4');
  return d.id;
}
async function aceptar(id, equipo, fecha) {
  const d = await api(`/api/coordinacion/derivaciones/${id}`);
  const m = opciones.municipios.find(m => m.equipo_id === equipo && m.centro_ss_id);
  const subprograma = d.derivacion_datos.contenido.tipo === 'anexo4' ? 'REUNIFICACION' : 'RIESGO_SOCIAL';
  await api(`/api/coordinacion/derivaciones/${id}/aceptar`, 'POST', {
    version: d.version, fecha_acta: fecha, municipio_id: m.id, centro_ss_id: m.centro_ss_id, equipo_id: equipo,
    subprograma, profesional_referencia_puesto: 'TS', profesional_referencia_id: null,
    menores: d.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: 'M' })),
    descripcion_situacion: 'La familia solicita apoyo para organizar los cuidados y mejorar la asistencia escolar.',
  });
}
async function consentir(id, fecha, cuenta = 'marta.cabrera') {
  await entrar(cuenta);
  await api(ruta(id, 'hitos'), 'POST', { version: (await ficha(id)).version, tipo: 'consentimiento', fecha });
  await estado(id, 'en_intervencion');
}
async function proponer(id, motivo = 'OBJ_TOTAL', cuenta = 'marta.cabrera', sinConsentimiento = false) {
  await entrar(cuenta);
  const f = await ficha(id);
  await api(ruta(id, 'finalizacion'), 'PATCH', { version: f.version, motivo, sin_consentimiento: sinConsentimiento });
  await api(ruta(id, 'proponer-cierre'), 'POST', { version: (await ficha(id)).version });
}
async function cerrar(id, fecha) {
  await entrar('javier.medina');
  return api(ruta(id, 'cerrar'), 'POST', { version: (await ficha(id)).version, fecha });
}
async function emitir(id, tipo, version) {
  const antes = await ficha(id);
  const clave = randomUUID();
  const p = { tipo, expediente_id: id, idioma: 'es', version };
  const primera = await api('/api/documentos/emisiones', 'POST', p, 201, clave);
  const repetida = await api('/api/documentos/emisiones', 'POST', p, 200, clave);
  assert.deepEqual(repetida, { ...primera, estado: 200 });
  assert.match(primera.emision, /^[a-f0-9-]{36}$/);
  assert.deepEqual(primera.firma, [80, 75]); assert(primera.longitud > 1000);
  const despues = await ficha(id);
  for (const campo of ['version', 'estado', 'hitos', 'historial']) assert.deepEqual(despues[campo], antes[campo], `${tipo}: cambia ${campo}`);
}
async function clicAPI(selector, final, metodo = 'POST') {
  const respuesta = pagina.waitForResponse(r => r.request().method() === metodo && r.url().endsWith(final));
  await selector.click();
  const r = await respuesta; assert.equal(r.status(), 200, await r.text());
  return r.json();
}
async function confirmarUI(nombre, final) {
  await pagina.getByRole('button', { name: nombre, exact: true }).click();
  return clicAPI(pagina.getByRole('dialog').getByRole('button', { name: nombre, exact: true }), final,
    final.endsWith('/respuesta') ? 'PATCH' : 'POST');
}
async function abrirEquipo(id, pestaña) {
  await pagina.goto(`${base}/equipo/expediente?id=${id}&pestana=${pestaña}`);
  await pagina.locator(`#equipo-tab-${pestaña}[aria-selected="true"]`).waitFor();
}
async function abrirCierre(id) {
  await pagina.goto(`${base}/coordinacion/expedientes/${id}/salidas`);
  await pagina.locator('#salidas-fecha_respuesta, #salidas-fecha_cierre, #salidas-fecha_reapertura').first().waitFor();
}
async function tabla(equipo, mes) { return api(`/api/estadistica/spaf?anio=2026&mes=${mes}&etf=${equipo}`); }
let opciones;
try {
  await entrar('ines.moreno');
  opciones = await api('/api/coordinacion/decision/opciones?idioma=es');
  const spaf = await api('/api/estadistica/spaf/opciones');
  const equipo02 = spaf.equipos.find(e => e.codigo === '02' || spaf.textos?.[e.clave_texto]?.includes('02'))?.id;
  // La pertenencia sale de la sesión real, sin inferir etiquetas traducidas.
  await entrar('marta.cabrera');
  const propio = await api('/api/estadistica/spaf/opciones');
  const origen = propio.equipos[0].id;
  assert.equal(propio.equipos.length, 1);
  if (equipo02) assert.equal(origen, equipo02);
  await entrar('pablo.ruiz');
  const destino = (await api('/api/estadistica/spaf/opciones')).equipos[0].id;
  assert.notEqual(origen, destino);
  await entrar('ines.moreno');
  const iniciales = {};
  for (const e of [origen, destino]) for (const mes of [7, 8, 9]) iniciales[`${e}/${mes}`] = (await tabla(e, mes)).resumen;

  const a = await crear('anexo3', origen, '2026-07-31');
  await consentir(a, '2026-07-31');
  await abrirEquipo(a, 'salidas');
  await pagina.locator('#equipo-spm-motivo').fill('Se necesita valorar la protección de los menores.');
  await pagina.locator('#equipo-spm-medida').fill('Valoración de una medida de protección.');
  await clicAPI(pagina.getByRole('button', { name: t('equipo.spm.preparar'), exact: true }), '/salida-spm');
  const salidaInicial = await api(ruta(a, 'salida-spm'));
  await estado(a, 'en_intervencion');
  const antesAnexo = await ficha(a);
  const download = pagina.waitForEvent('download');
  const word = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/api/documentos/emisiones'));
  await pagina.getByRole('button', { name: t('equipo.spm.generar'), exact: true }).click();
  assert.equal((await word).status(), 201); assert.equal(await (await download).failure(), null);
  assert.equal((await ficha(a)).version, antesAnexo.version);
  assert.equal((await api(ruta(a, 'salida-spm'))).fecha_derivacion, undefined);
  await pagina.locator('#equipo-spm-fecha').fill('2026-07-31');
  await clicAPI(pagina.locator('#equipo-panel-salidas button[type="submit"]'), '/salida-spm/fecha');
  producto((await ficha(a)).estado === 'derivada_spm', 'Confirmar la fecha de salida SPM no cambia el expediente a derivada_spm.');

  // La espera conserva el bloqueo; la respuesta habilitante lo elimina.
  await entrar('ines.moreno');
  await abrirCierre(a);
  const bloqueo = await api(ruta(a, 'salida-spm'));
  assert.equal(bloqueo.respuesta, 'a_la_espera');
  assert.equal(await pagina.getByRole('button', { name: t('salidas.accion.cerrar'), exact: true }).count(), 0);
  await pagina.locator('#salidas-respuesta').selectOption('medida_no_necesaria');
  await pagina.locator('#salidas-fecha_respuesta').fill('2026-08-01');
  await confirmarUI(t('salidas.accion.respuesta'), '/salida-spm/respuesta');
  await estado(a, 'en_intervencion');
  await entrar('marta.cabrera');
  await abrirEquipo(a, 'cierre');
  await pagina.locator('#equipo-motivo').selectOption('OBJ_TOTAL');
  await clicAPI(pagina.getByRole('button', { name: t('equipo.cierre.guardar'), exact: true }), '/finalizacion', 'PATCH');
  await emitir(a, 'finalizacion/anexo11', (await ficha(a)).version);
  await clicAPI(pagina.getByRole('button', { name: t('equipo.cierre.proponer'), exact: true }), '/proponer-cierre');
  await api(ruta(a, 'cerrar'), 'POST', { version: (await ficha(a)).version, fecha: '2026-08-01' }, 403);
  await entrar('javier.medina');
  const propuestas = await api('/api/coordinacion/cierres');
  assert(JSON.stringify(propuestas).includes(a));
  await abrirCierre(a);
  await pagina.locator('#salidas-fecha_cierre').fill('2026-08-01');
  const cierre1 = await confirmarUI(t('salidas.accion.cerrar'), '/cerrar');
  await estado(a, 'cerrada');
  await entrar('marta.cabrera');
  await api(ruta(a, 'finalizacion'), 'PATCH', { version: (await ficha(a)).version, motivo: 'OBJ_TOTAL' }, 409);
  await entrar('javier.medina');
  const acta = pagina.waitForEvent('download');
  const respuestaActa = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/documentos/acta_cierre'));
  const antesActa = await ficha(a);
  await pagina.getByRole('button', { name: t('salidas.accion.acta'), exact: true }).click();
  assert.equal((await respuestaActa).status(), 201); assert.equal(await (await acta).failure(), null);
  assert.equal((await ficha(a)).version, antesActa.version);
  await pagina.locator('#salidas-fecha_reapertura').fill('2026-08-15');
  await pagina.locator('#salidas-motivo_reapertura').fill('La familia solicita retomar el apoyo para los cuidados.');
  const reabierta = await confirmarUI(t('salidas.accion.reabrir'), '/reabrir');
  assert.notEqual(reabierta.episodio_id, cierre1.episodio_id);
  await estado(a, 'en_intervencion');
  await pagina.screenshot({ path: resolve(evidencia, 'reapertura.png'), fullPage: true });
  ok('F3.06-CIRCUITO-OK: Anexo 5, fecha separada, bloqueo, respuesta, Anexo 11, propuesta de Marta, cierre de Javier, Acta y reapertura.');

  await proponer(a, 'TRASLADO');
  await entrar('javier.medina');
  const peticionTraslado = { version: (await ficha(a)).version, fecha: '2026-08-31', equipo_destino_id: destino };
  const claveTraslado = randomUUID();
  const traslado = await api(ruta(a, 'trasladar'), 'POST', peticionTraslado, 200, claveTraslado);
  assert.deepEqual(await api(ruta(a, 'trasladar'), 'POST', peticionTraslado, 200, claveTraslado), traslado);
  assert.equal(traslado.cierre.episodio_id, reabierta.episodio_id);
  const nuevo = traslado.destino_id;
  await estado(a, 'cerrada');
  await estado(nuevo, 'derivacion_recibida');
  const derivada = await api(`/api/coordinacion/derivaciones/${nuevo}`);
  assert.equal(derivada.numero, null);
  assert.equal(derivada.servicio_derivante, 'OTRO_ETF');
  await aceptar(nuevo, destino, '2026-08-31');
  assert.notEqual((await ficha(nuevo)).numero, (await ficha(a)).numero);
  assert.equal((await api(ruta(nuevo, 'hitos'))).some(h => h.tipo === 'consentimiento'), false);
  await consentir(nuevo, '2026-08-31', 'pablo.ruiz');
  await proponer(nuevo, 'OBJ_TOTAL', 'pablo.ruiz');
  await cerrar(nuevo, '2026-09-01');
  ok('F3.06-TRASLADO-OK: origen cerrado, destino enlazado con número nuevo, consentimiento independiente y repetición sin duplicado.');

  const b = await crear('anexo4', origen, '2026-07-31');
  await consentir(b, '2026-07-31');
  await entrar('marta.cabrera');
  await api(ruta(b, 'finalizacion'), 'PATCH', { version: (await ficha(b)).version, motivo: 'OBJ_TOTAL', sin_consentimiento: false });
  await emitir(b, 'finalizacion/anexo12', (await ficha(b)).version);
  await api(ruta(b, 'proponer-cierre'), 'POST', { version: (await ficha(b)).version });
  await cerrar(b, '2026-07-31');
  await entrar('marta.cabrera');
  const pSeguimiento = { version: (await ficha(b)).version, fecha: hoy };
  const claveSeguimiento = randomUUID();
  const seguimiento = await api(ruta(b, 'seguimiento'), 'POST', pSeguimiento, 200, claveSeguimiento);
  assert.deepEqual(await api(ruta(b, 'seguimiento'), 'POST', pSeguimiento, 200, claveSeguimiento), seguimiento);
  assert.equal(seguimiento.fecha_cese_medida, hoy); assert.equal(seguimiento.version, 1);
  assert.equal(seguimiento.fecha_fin_prevista, `${Number(hoy.slice(0, 4)) + 1}${hoy.slice(4)}`);
  await estado(b, 'seguimiento_tras_cese');
  const final = await api(ruta(b, 'seguimiento/terminar'), 'POST', { version: seguimiento.version, fecha: hoy, motivo: 'CAMBIO_SITUACION' });
  await estado(b, 'cerrada');
  await api(ruta(b, 'seguimiento/reabrir'), 'POST', { version: final.version, fecha: hoy, motivo: 'La familia necesita retomar el seguimiento de la reunificación.' }, 403);
  await entrar('javier.medina');
  const retomado = await api(ruta(b, 'seguimiento/reabrir'), 'POST', { version: final.version, motivo: 'La familia necesita retomar el seguimiento de la reunificación.' });
  assert.equal(retomado.version, 3); assert.equal(retomado.fecha_cese_medida, hoy);
  assert.equal(retomado.fecha_fin_prevista, seguimiento.fecha_fin_prevista);
  await estado(b, 'seguimiento_tras_cese');
  await entrar('marta.cabrera');
  await api(ruta(b, 'finalizacion'), 'PATCH', { version: (await ficha(b)).version, motivo: 'OBJ_TOTAL' }, 409);
  await entrar('javier.medina');
  ok('F3.06-SEGUIMIENTO-OK: Anexo 12, cese, plazo manual de doce meses, terminación y reapertura sólo por Coordinación.');

  const c = await crear('anexo3', origen, '2026-08-31');
  await proponer(c, 'SIN_CONSENTIMIENTO', 'marta.cabrera', true);
  const sinAnexo = await cerrar(c, '2026-09-01');
  assert(sinAnexo.aviso);
  ok('F3.06-ACEPTADA-OK: cierre sin consentimiento ni Anexo 11 avisa y permite, sin inventar intervención.');

  // Cálculo manual por expediente, independiente del algoritmo de SPAF:
  // A: seis menores; alta 31/07, bajas 01/08 y 31/08; reapertura no es aceptación nueva.
  // B: tres menores; alta y baja 31/07; no aparece en agosto ni septiembre por seguimiento.
  // C: seis menores; alta 31/08, baja 01/09. D: seis menores, alta 31/08, baja 01/09.
  // Dos cierres de A siguen visibles, pero cuentan una sola baja y un solo caso.
  const esperadas = [
    [origen, 7, [[a, true, false, []], [b, true, true, ['2026-07-31']]], { altas: 2, bajas: 1, casos: 2, menores: 9 }],
    [origen, 8, [[a, false, true, ['2026-08-01', '2026-08-31']], [c, true, false, []]], { altas: 1, bajas: 1, casos: 2, menores: 12 }],
    [origen, 9, [[c, false, true, ['2026-09-01']]], { altas: 0, bajas: 1, casos: 1, menores: 6 }],
    [destino, 7, [], { altas: 0, bajas: 0, casos: 0, menores: 0 }],
    [destino, 8, [[nuevo, true, false, []]], { altas: 1, bajas: 0, casos: 1, menores: 6 }],
    [destino, 9, [[nuevo, false, true, ['2026-09-01']]], { altas: 0, bajas: 1, casos: 1, menores: 6 }],
  ];
  const ids = [a, b, c, nuevo];
  const evidenciaSPAF = [];
  for (const [e, mes, filas, suma] of esperadas) {
    const tablaMes = await tabla(e, mes);
    const propias = tablaMes.filas.filter(f => ids.includes(f.expediente_id));
    assert.deepEqual(propias.map(f => f.expediente_id).sort(), filas.map(f => f[0]).sort());
    for (const [id, alta, baja, cierres] of filas) {
      const f = propias.find(f => f.expediente_id === id);
      assert.equal(f.alta, alta); assert.equal(f.baja, baja);
      assert.deepEqual(f.cierres.map(c => c.fecha.slice(0, 10)).sort(), cierres);
      if (id === a && mes === 8) assert.deepEqual(new Set(f.cierres.map(c => c.episodio_id)), new Set([cierre1.episodio_id, reabierta.episodio_id]));
    }
    for (const campo of ['altas', 'bajas', 'casos']) assert.equal(tablaMes.resumen[campo], iniciales[`${e}/${mes}`][campo] + suma[campo], `${e}/${mes} ${campo}`);
    producto(tablaMes.resumen.menores === iniciales[`${e}/${mes}`].menores + suma.menores,
      `SPAF ${mes}/2026 equipo ${e}: incremento de menores esperado ${suma.menores}, observado ${tablaMes.resumen.menores - iniciales[`${e}/${mes}`].menores}; casos sin dato ${tablaMes.resumen.casos_sin_dato_menores}.`);
    evidenciaSPAF.push({ equipo: e, mes, base: iniciales[`${e}/${mes}`], incremento_manual: suma, resumen: tablaMes.resumen, filas: propias });
  }
  await writeFile(resolve(evidencia, 'spaf.json'), JSON.stringify(evidenciaSPAF, null, 2));
  ok('F3.07-MOVIMIENTOS-OK: seis tablas; alta último día, baja primero, cierre último día anterior, dos episodios sin pérdida y traslado en ambos equipos.');

  // Un perfil combinado permite probar la regla de dos personas en el dominio.
  const e = await crear('anexo3', origen, hoy);
  await entrar('elena.salas');
  usuarioRestaurar = (await api('/api/admin/usuarios')).find(u => u.cuenta === 'sergio.campos');
  assert(usuarioRestaurar);
  const desde = new Date(Date.now() - 60000).toISOString();
  await api(`/api/admin/usuarios/${usuarioRestaurar.id}/perfiles`, 'PUT', {
    version: usuarioRestaurar.version, estado: 'activo',
    perfiles: ['coordinacion', 'profesional_etf'].map(perfil => ({ perfil, desde, hasta: null })),
    equipo: { equipo_id: origen, puesto: 'ts', desde, hasta: null },
  });
  await proponer(e, 'SIN_CONSENTIMIENTO', 'sergio.campos', true);
  const propuestaVersion = (await ficha(e)).version;
  const mismaPersona = await api(ruta(e, 'cerrar'), 'POST', { version: propuestaVersion, fecha: hoy }, 422);
  assert.equal(mismaPersona.error.clave, 'expediente.cierre_segunda_persona');
  await estado(e, 'aceptada');
  const spmPendiente = await api(ruta(e, 'salida-spm'), 'POST', {
    version: propuestaVersion, motivo: 'Es necesario valorar una medida protectora.', propuesta_medida: 'Valoración por el Servicio de Protección de Menores.',
  });
  await api(ruta(e, 'salida-spm/fecha'), 'POST', { version: spmPendiente.version, fecha: hoy });
  await entrar('javier.medina');
  const bloqueada = await api(ruta(e, 'cerrar'), 'POST', { version: (await ficha(e)).version, fecha: hoy }, 422);
  assert.equal(bloqueada.error.clave, 'expediente.cierre_bloqueado_anexo5');
  await abrirCierre(e);
  assert(await pagina.getByRole('button', { name: t('salidas.accion.cerrar'), exact: true }).isDisabled());
  const pendiente = await api(ruta(e, 'salida-spm'));
  await api(ruta(e, 'salida-spm/forzado'), 'POST', { version: pendiente.version, motivo: '' }, 422);
  const motivoForzado = 'Coordinación acuerda finalizar el episodio mientras llega la respuesta del SPM.';
  const forzado = await api(ruta(e, 'salida-spm/forzado'), 'POST', { version: pendiente.version, motivo: motivoForzado });
  assert(forzado.forzado_id); assert.equal(forzado.motivo_forzado, motivoForzado);
  await api(ruta(e, 'cerrar'), 'POST', { version: (await ficha(e)).version, fecha: hoy, forzado_id: forzado.forzado_id });
  await estado(e, 'cerrada');
  await api(ruta(e, 'reabrir'), 'POST', { version: (await ficha(e)).version, fecha: hoy, motivo: 'Se retoma el apoyo mientras llega la respuesta del SPM.' });
  await estado(e, 'derivada_spm');
  await writeFile(resolve(evidencia, 'persistencia-fixture.json'), JSON.stringify({ origen: a, destino: nuevo, seguimiento: b, forzado: e }));
  ok('F3.06-BARRERAS-OK: misma persona rechazada con 422, cierre ordinario bloqueado con 422, excepción motivada y reapertura con bloqueo conservado.');

  await entrar('marta.cabrera');
  await pagina.goto(`${base}/equipo/tabla-spaf`);
  await pagina.locator('#spaf-mes').fill('2026-08');
  await pagina.getByRole('link', { name: (await ficha(a)).numero, exact: true }).first().waitFor();
  await pagina.screenshot({ path: resolve(evidencia, 'spaf-agosto.png'), fullPage: true });
  await api(`/api/estadistica/spaf?anio=2026&mes=9&etf=${destino}`, 'GET', undefined, 403);
  await denegarEquipo(`/api/expedientes/${nuevo}`);
  await entrar('pablo.ruiz');
  const ajenas = [
    ['GET', `/api/expedientes/${a}`], ['GET', ruta(a, 'salida-spm')], ['GET', ruta(a, 'finalizacion')],
    ['GET', ruta(b, 'seguimiento')], ['PATCH', ruta(a, 'finalizacion')], ['POST', ruta(a, 'proponer-cierre')],
    ['POST', ruta(b, 'seguimiento/terminar')],
  ];
  for (const [metodo, path] of ajenas) await denegarEquipo(path, metodo, metodo === 'GET' ? undefined : { version: 1 });
  await entrar('lucia.navarro');
  for (const [metodo, path] of [...ajenas, ['GET', `/api/estadistica/spaf?anio=2026&mes=9&etf=${origen}`],
    ['POST', ruta(a, 'cerrar')], ['POST', ruta(a, 'reabrir')], ['POST', ruta(a, 'trasladar')],
    ['POST', ruta(b, 'seguimiento')], ['POST', ruta(b, 'seguimiento/reabrir')],
    ['POST', ruta(a, 'documentos/acta_cierre')]]) {
    await api(path, metodo, metodo === 'GET' ? undefined : { version: 1 }, 403);
  }
  const cifras = await pedir('/api/consulta/memoria/2026');
  producto(cifras.estado === 200, `Consulta no obtiene cifras en GET /api/consulta/memoria/2026: HTTP ${cifras.estado}.`);
  if (cifras.estado === 200) {
    const serializado = JSON.stringify(cifras.datos);
    for (const id of ids) assert(!serializado.includes(id));
    assert(!/"(?:expediente_id|nombre|apellidos|descripcion_situacion|motivo_texto)"/.test(serializado));
    ok('F3.06-CONSULTA-CIFRAS-OK: respuesta positiva sin identificadores ni textos personales.');
  }
  ok('F3.06-PERMISOS-OK: equipo ajeno sin ficha ni acciones; origen sin destino; Consulta no accede a fichas, SPAF ni acciones.');
  assert.deepEqual(errores, []);
  assert(salidaInicial.id);
  ok('F3.06/F3.07-NAVEGADOR-OK: Chrome del sistema, servicios reales, sin errores JavaScript.');
  if (fallosProducto.length) {
    console.error(`F3.06/F3.07-FALLOS-PRODUCTO: ${fallosProducto.length} comprobaciones; consulta resultado.json.`);
    process.exitCode = 1;
  }
} catch (error) {
  if (pagina) await pagina.screenshot({ path: resolve(evidencia, 'fallo.png'), fullPage: true }).catch(() => {});
  throw error;
} finally {
  try {
    if (usuarioRestaurar) {
      await entrar('elena.salas');
      const actual = (await api('/api/admin/usuarios')).find(u => u.id === usuarioRestaurar.id);
      await api(`/api/admin/usuarios/${actual.id}/perfiles`, 'PUT', {
        version: actual.version, estado: usuarioRestaurar.estado === 'pendiente_alta' ? 'activo' : usuarioRestaurar.estado,
        perfiles: usuarioRestaurar.perfiles, equipo: usuarioRestaurar.equipo,
      });
      const restaurado = (await api('/api/admin/usuarios')).find(u => u.id === usuarioRestaurar.id);
      for (const campo of ['estado', 'perfiles', 'equipo']) assert.deepEqual(restaurado[campo], usuarioRestaurar[campo], `Restaurar Sergio: ${campo}`);
    }
  } finally {
    try {
      for (const [cuenta, p] of sesiones) await p.context().tracing.stop({ path: resolve(evidencia, `${cuenta}.zip`) });
      await writeFile(resolve(evidencia, 'resultado.json'), JSON.stringify({ pasos, errores, fallosProducto }, null, 2));
    } finally { await navegador.close(); }
  }
}
