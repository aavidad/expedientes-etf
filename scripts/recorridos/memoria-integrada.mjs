// F4.09: recorrido de Memoria con datos inventados y resultados manuales.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { randomUUID } from 'node:crypto';
import { chromium, baseLocal, restringir } from './comun.mjs';

const [url, directorio] = process.argv.slice(2);
const base = baseLocal(url);
assert(directorio);
const evidencia = resolve(directorio);
await mkdir(evidencia, { recursive: true });
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const textos = JSON.parse(await readFile(new URL('../../idiomas/es/textos/memoria.json', import.meta.url)));
const t = clave => { assert(textos[clave], clave); return textos[clave]; };
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const sesiones = new Map();
const csrfPorPagina = new Map();
const errores = [];
const pasos = [];
const fallosProducto = [];
const resultados = [];
let pagina;
let opciones;
let anio;
let equipo;
let proximaPeticion = 0;
function ok(texto) { pasos.push(texto); console.log(texto); }
function cifra(observado, esperado, caso) {
  resultados.push({ caso, esperado, observado });
  try { assert.deepEqual(observado, esperado); }
  catch { fallosProducto.push({ caso, esperado, observado }); console.error(`F4.09-CIFRA: ${caso}: esperado=${JSON.stringify(esperado)}; observado=${JSON.stringify(observado)}`); }
}
async function entrar(cuenta) {
  if (sesiones.has(cuenta)) { pagina = sesiones.get(cuenta); return; }
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
  await restringir(contexto, base);
  // La demo permite 100 peticiones por IP y minuto, compartidas entre perfiles.
  // Regula antes de enviar; no reintenta escrituras tras una respuesta 429.
  await contexto.route(`${base}/api/**`, async route => {
    const inicio = Math.max(Date.now(), proximaPeticion); proximaPeticion = inicio + 700;
    await new Promise(resolve => setTimeout(resolve, Math.max(0, inicio - Date.now())));
    await route.fallback();
  });
  await contexto.tracing.start({ screenshots: true, snapshots: true });
  pagina = await contexto.newPage(); pagina.setDefaultTimeout(20000);
  pagina.on('pageerror', error => errores.push(error.message));
  pagina.on('response', r => { if (r.status() >= 400) console.log(`F4.09-HTTP: ${r.status()} ${new URL(r.url()).pathname}`); });
  await pagina.goto(base);
  await pagina.locator('#cuenta-ad').fill(cuenta);
  await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
  const [respuesta] = await Promise.all([
    pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/api/sesion')),
    pagina.locator('button[type="submit"]').click(),
  ]); assert.equal(respuesta.status(), 200);
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
  assert.equal(r.estado, esperado, `${metodo} ${ruta}: ${r.datos?.error?.clave ?? r.datos?.clave ?? Object.keys(r.datos ?? {}).join(',')}`);
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
const ruta = (id, accion = '') => `/api/expedientes/${id}${accion ? `/${accion}` : ''}`;
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
  if (fecha < '2026-01-01') {
    const usados = new Set();
    let offset = 0;
    let filas;
    do {
      filas = await api(`/api/equipo/expedientes?limite=100&desplazamiento=${offset}`);
      filas.filas.forEach(f => usados.add(f.numero)); offset += 100;
    } while (filas.hay_mas);
    const codigo = opciones.equipos.find(e => e.id === equipo).codigo ?? '02';
    const numero = Array.from({ length: 99 }, (_, i) => `${codigo}-${String(i + 1).padStart(2, '0')}-25`).find(n => !usados.has(n));
    assert(numero);
    const r = await api('/api/expedientes/historicos', 'POST', { equipo_id: equipo, casos: [{
      numero, fecha_derivacion: fecha, estado: 'aceptada', derivacion_datos: a.derivacion_datos,
      aceptacion: { version: 1, fecha_acta: fecha, municipio_id: m.id, centro_ss_id: m.centro_ss_id, equipo_id: equipo,
        subprograma: 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', profesional_referencia_id: null,
        menores: a.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: 'M' })),
        descripcion_situacion: 'La familia solicita apoyo para organizar los cuidados y la asistencia escolar.' },
      episodios: [{ apertura_fecha: fecha }], familia: [], menores: [],
    }] });
    return r.expedientes[0];
  }
  const d = await api('/api/coordinacion/derivaciones/confirmar', 'POST', {
    lectura_id: a.lectura_anexo.id, tipo: a.derivacion_datos.contenido.tipo,
    campos: a.derivacion_datos.contenido.campos.map(c => ({ ...c, dudoso: false, motivos: [] })),
    ubicacion: { municipio_id: m.id, centro_id: m.centro_ss_id, equipo_id: equipo },
  }, 201);
  await aceptar(d.id, equipo, fecha);
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
const consulta = id => new URLSearchParams({ anio, ...(id ? { etf: id } : {}) });
const memoria = id => api(`/api/estadistica/memoria?${consulta(id)}`);
const estadoMemoria = id => api(`/api/estadistica/memoria/estado?${consulta(id)}`);
const foto = (id, revision) => api(`/api/estadistica/memoria/instantanea?${consulta(id)}&revision=${revision}`);
const apartado = (informe, codigo) => { const a = informe.apartados.find(a => a.codigo === codigo); assert(a, codigo); return a; };
function cantidad(informe, codigo, unidad, grupo) {
  return apartado(informe, codigo).celdas.filter(c => c.unidad === unidad && Object.entries(grupo).every(([k, v]) => c.grupo[k] === v))
    .reduce((n, c) => n + c.numerador, 0);
}
function comprobarManual(antes, despues, ambito) {
  // Tres familias y seis menores. A viene de 2025; B y C entran en 2026.
  // A y C cierran con objetivos cumplidos; B permanece con consentimiento.
  const totales = {
    '1': { familias: 3, menores: 6 }, '2': { menores: 6 }, '3': { familias: 3 },
    '4': { familias: 3 }, '5': { familias: 0 }, '6': { familias: 0, menores: 0 },
    '7': { familias: 2, menores: 4 }, '8': { familias: 2, menores: 4 }, '9': { familias: 0, menores: 0 },
    '10': { familias: 2, menores: 0 }, '11.1': { familias: 2, menores: 4 }, '11.2': { familias: 2, menores: 4 },
    '12.1': { familias: 2, menores: 4 }, '12.2': { familias: 0 },
    '13.1': { familias: 1 }, '13.2': { familias: 1 }, '13.3': { familias: 0 },
    '14': { menores: 6 }, '14.1': { menores: 6 }, '15': { comisiones: 0 }, '16': { informes: 0 },
    '17': { familias: 3 }, '18': { familias: 3 }, '19': { adultos: 0 }, '20': { adultos: 0 }, '21': { adultos: 0 },
    '22': { familias: 3 }, '23': { familias: 3 }, '24': { familias: 3 }, '25': { menores: 6 },
    '26': { menores: 6 }, '27': { menores: 6 }, '28': { familias: 3 },
  };
  assert.deepEqual(despues.apartados.map(a => a.codigo).sort(), Object.keys(totales).sort());
  for (const [codigo, unidades] of Object.entries(totales)) for (const [unidad, n] of Object.entries(unidades)) {
    cifra(apartado(despues, codigo).total[unidad] - apartado(antes, codigo).total[unidad], n, `${ambito}/${codigo}/total/${unidad}`);
  }
  const celda = (codigo, unidad, grupo, n) => cifra(cantidad(despues, codigo, unidad, grupo) - cantidad(antes, codigo, unidad, grupo), n,
    `${ambito}/${codigo}/${unidad}/${JSON.stringify(grupo)}`);
  celda('1', 'familias', { origen: 'anterior', subprograma: 'riesgo' }, 1);
  celda('1', 'menores', { origen: 'anterior', subprograma: 'riesgo' }, 2);
  celda('1', 'familias', { origen: 'incorporado', subprograma: 'riesgo' }, 1);
  celda('1', 'familias', { origen: 'incorporado', subprograma: 'reunificacion' }, 1);
  for (const sexo of ['nino', 'nina']) {
    celda('2', 'menores', { sexo, subprograma: 'riesgo' }, 2);
    celda('2', 'menores', { sexo, subprograma: 'reunificacion' }, 1);
    celda('14', 'menores', { sexo, estado: 'EDUCACION' }, 2);
    celda('14', 'menores', { sexo, estado: 'SALUD' }, 1);
  }
  celda('7', 'familias', { origen: 'anterior', subprograma: 'riesgo' }, 1);
  celda('7', 'familias', { origen: 'incorporado', subprograma: 'reunificacion' }, 1);
  for (const codigo of ['14.1', '24', '26', '27']) {
    const unidad = codigo === '24' ? 'familias' : 'menores';
    celda(codigo, unidad, {}, 0);
    const sinDato = informe => apartado(informe, codigo).celdas.filter(c => c.unidad === unidad).reduce((n, c) => n + c.sin_dato, 0);
    cifra(sinDato(despues) - sinDato(antes), 0, `${ambito}/${codigo}/listas_valoradas_sin_dato`);
  }
  const desconocidos = informe => apartado(informe, '10').celdas.filter(c => c.unidad === 'familias').reduce((n, c) => n + c.sin_dato, 0);
  celda('10', 'familias', {}, 0);
  cifra(desconocidos(despues) - desconocidos(antes), 2, `${ambito}/10/destino_desconocido`);
  // 15/01/2025 a 15/02/2026: 13 meses. 01/01 a 31/05/2026: cuatro meses completos.
  celda('13.1', 'familias', { tramo: '2' }, 1);
  celda('13.2', 'familias', { tramo: '0' }, 1);
  for (const estado of ['SIN_DIFICULTADES', 'DIFICULTADES_SIN_NECESIDADES_BASICAS', 'DIFICULTADES_NECESIDADES_BASICAS']) celda('22', 'familias', { estado }, 1);
  celda('25', 'menores', { sexo: 'nina', tramo: 'edad_0_5' }, 1);
  celda('25', 'menores', { sexo: 'nino', tramo: 'edad_6_11' }, 2);
  celda('25', 'menores', { sexo: 'nina', tramo: 'edad_6_11' }, 1);
  celda('25', 'menores', { sexo: 'nino', tramo: 'edad_12_17' }, 1);
  celda('25', 'menores', { sexo: 'nina', tramo: 'edad_12_17' }, 1);
  // Contrasta el juego nuevo; la demo previa puede contener cierres ambiguos.
  for (const u of ['familias', 'menores']) {
    const incremento = codigo => apartado(despues, codigo).total[u] - apartado(antes, codigo).total[u];
    cifra(incremento('7'), incremento('8') + incremento('9'), `${ambito}/7=8+9/${u}`);
  }
}
async function preparar(tipo, fecha, nacimiento1, nacimiento2, economia, recurso, cierre) {
  const id = await crear(tipo, equipo, fecha);
  await entrar('marta.cabrera');
  fecha = fecha < '2026-01-01' ? '2026-01-15' : fecha;
  const familia = await api(ruta(id, 'familia'));
  await api(ruta(id), 'PATCH', { version: familia.version, vigente_desde: fecha, familia: { ...familia.familia, familia_numerosa: false, situacion_economica: economia } });
  await api(ruta(id, 'adultos'), 'PUT', { version: (await ficha(id)).version, vigente_desde: fecha, adultos: [] });
  const subprograma = tipo === 'anexo4' ? 'REUNIFICACION' : 'RIESGO_SOCIAL';
  await api(ruta(id, 'menores'), 'PUT', { version: (await ficha(id)).version, vigente_desde: fecha, menores: [
    { orden: 1, nombre: 'Daniel', apellidos: 'Torres Montes', sexo: 'H', fecha_nacimiento: nacimiento1, subprograma, propuesto_atencion_temprana: false },
    { orden: 2, nombre: 'Lucía', apellidos: 'Torres Montes', sexo: 'M', fecha_nacimiento: nacimiento2, subprograma, propuesto_atencion_temprana: false },
  ] });
  await api(ruta(id, 'intervencion'), 'PUT', { version: (await ficha(id)).version, vigente_desde: fecha, intervencion: { fase: 'INTERVENCION', listas: {
    coordinacion: [{ codigo: recurso, menores_excluidos: [] }], factor_familia: [], factor_menor: [], tipologia: [],
  } } });
  await api(ruta(id, 'hitos'), 'POST', { version: (await ficha(id)).version, tipo: 'consentimiento', fecha });
  if (cierre) {
    await api(ruta(id, 'finalizacion'), 'PATCH', { version: (await ficha(id)).version, motivo: 'OBJ_TOTAL', sin_consentimiento: false });
    await api(ruta(id, 'proponer-cierre'), 'POST', { version: (await ficha(id)).version });
    await entrar('javier.medina');
    await api(ruta(id, 'cerrar'), 'POST', { version: (await ficha(id)).version, fecha: cierre });
  }
  return id;
}
async function abrir(id, consultaSolo = false) {
  await pagina.goto(`${base}/memoria`);
  await pagina.locator('#memoria-anio').waitFor();
  await pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).waitFor({ state: 'visible' });
  // Espera la consulta inicial antes de cambiar los filtros.
  await pagina.waitForFunction(() => !document.querySelector('button[type="submit"]')?.disabled);
  await pagina.locator('#memoria-anio').fill(String(anio));
  if (await pagina.locator('#memoria-equipo').count()) await pagina.locator('#memoria-equipo').selectOption(id);
  const [respuesta] = await Promise.all([
    pagina.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === (consultaSolo ? `/api/consulta/memoria/${anio}` : '/api/estadistica/memoria')),
    pagina.getByRole('button', { name: t('memoria.consultar'), exact: true }).click(),
  ]);
  assert.equal(respuesta.status(), 200);
  await pagina.locator('#memoria-apartado-1').waitFor();
}
async function bloques(informe) {
  for (let i = 0; i < 4; i++) {
    const boton = pagina.getByRole('button', { name: t(`memoria.bloque.${i}`), exact: true });
    if (await boton.isEnabled()) await boton.click();
    const esperados = [['1', '2', '3', '4', '5', '6'], ['7', '8', '9', '10', '11.1', '11.2', '12.1', '12.2', '13.1', '13.2', '13.3'], ['14', '14.1', '15', '16', '17', '18', '19', '20', '21'], ['22', '23', '24', '25', '26', '27', '28']][i];
    for (const codigo of esperados) {
      const articulo = pagina.locator(`[id="memoria-apartado-${codigo}"]`); await articulo.waitFor();
      const observado = await articulo.locator('tbody tr').evaluateAll(filas => filas.map(fila => {
        const celdas = fila.querySelectorAll('td'); return [celdas[1].textContent.trim(), celdas[4].textContent.trim()];
      }));
      const formato = new Intl.NumberFormat('es');
      const esperado = apartado(informe, codigo).celdas.map(c => [formato.format(c.numerador), formato.format(c.sin_dato)]);
      assert.deepEqual(observado, esperado, `Pantalla, cantidades y sin dato del apartado ${codigo}`);
    }
  }
}
async function acto(accion, id) {
  const antes = await estadoMemoria(id);
  await pagina.getByRole('button', { name: t(`memoria.${accion}`), exact: true }).click();
  const dialogo = pagina.getByRole('dialog'); await dialogo.waitFor();
  if (accion === 'reabrir') await pagina.locator('#memoria-motivo').fill('Se ha corregido la situación económica tras revisar la ficha.');
  const [respuesta] = await Promise.all([
    pagina.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith(`/memoria/${accion}`)),
    dialogo.getByRole('button', { name: t(accion === 'cerrar' ? 'memoria.confirmar_cierre' : 'memoria.confirmar_reapertura'), exact: true }).click(),
  ]); assert.equal(respuesta.status(), 200);
  const despues = await respuesta.json();
  assert.equal(despues.estado, accion === 'cerrar' ? 'cerrada' : 'borrador');
  assert(despues.version > antes.version);
  assert.equal(despues.revision_actual, antes.revision_actual + (accion === 'cerrar' ? 1 : 0));
  await dialogo.waitFor({ state: 'hidden' });
  const peticion = respuesta.request();
  const repetida = await api(new URL(peticion.url()).pathname + new URL(peticion.url()).search, 'POST', peticion.postDataJSON(), 200, peticion.headers()['idempotency-key']);
  assert.equal(repetida.repetida, true);
  assert.equal(repetida.version, despues.version); assert.equal(repetida.revision_actual, despues.revision_actual);
  return despues;
}
async function descargar(nombre) {
  const [d, respuesta] = await Promise.all([
    pagina.waitForEvent('download'),
    pagina.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname.endsWith('/word')),
    pagina.getByRole('button', { name: t('memoria.descargar'), exact: true }).click(),
  ]);
  assert([200, 201].includes(respuesta.status())); assert.equal(await d.failure(), null);
  const path = resolve(evidencia, nombre); await d.saveAs(path);
  const bytes = await readFile(path); assert.equal(bytes.subarray(0, 2).toString(), 'PK'); assert(bytes.length > 1000);
}

try {
  anio = 2026;
  await entrar('marta.cabrera'); equipo = (await api('/api/sesion')).equipo_id;
  await entrar('ines.moreno'); opciones = await api('/api/coordinacion/decision/opciones?idioma=es');
  // Conserva las cifras iniciales: la demo puede contener otros recorridos del mismo año.
  for (const id of [equipo, '']) {
    const e = await estadoMemoria(id);
    if (e.estado === 'cerrada') await api(`/api/estadistica/memoria/reabrir?${consulta(id)}`, 'POST', { version: e.version, motivo: 'Preparación de una nueva comprobación de Memoria con datos inventados.' });
  }
  const baseEquipo = await memoria(equipo); const baseProvincia = await memoria('');
  const a = await preparar('anexo3', '2025-01-15', '2015-03-12', '2017-08-21', 'SIN_DIFICULTADES', 'EDUCACION', '2026-02-15');
  const b = await preparar('anexo3', '2026-07-01', '2008-12-20', '2021-02-11', 'DIFICULTADES_SIN_NECESIDADES_BASICAS', 'EDUCACION');
  const c = await preparar('anexo4', '2026-01-01', '2016-04-01', '2010-05-05', 'DIFICULTADES_NECESIDADES_BASICAS', 'SALUD', '2026-05-31');
  const numeros = [];
  for (const id of [a, b, c]) numeros.push((await ficha(id)).numero);
  assert(numeros.every(n => typeof n === 'string' && n.length > 0));
  await writeFile(resolve(evidencia, 'casos.json'), JSON.stringify({ anio, equipo, a, b, c, numeros }, null, 2));
  await entrar('marta.cabrera');
  const calculado = await memoria(equipo); comprobarManual(baseEquipo, calculado, 'equipo');
  await abrir(equipo); await bloques(calculado);
  const narrativaAntes = await api(`/api/estadistica/memoria/narrativas?${consulta(equipo)}`);
  const metodologia = `Se revisan las rutinas familiares y la asistencia escolar en reuniones semanales. Revisión de metodología número ${narrativaAntes.version + 1}.`;
  await pagina.locator('#memoria-texto-d').fill(metodologia);
  const [guardada] = await Promise.all([
    pagina.waitForResponse(r => r.request().method() === 'PUT' && new URL(r.url()).pathname.endsWith('/narrativas')),
    pagina.getByRole('button', { name: t('memoria.guardar'), exact: true }).click(),
  ]);
  assert.equal(guardada.status(), 200); assert.equal((await guardada.json()).version, narrativaAntes.version + 1);
  await pagina.getByRole('status').filter({ hasText: t('memoria.guardada') }).waitFor();
  await descargar('memoria-equipo.docx');
  ok('F4.09-CALCULO: 28 apartados; contrastes manuales en equipo; cuatro bloques; Word descargado.');
  const cierreEquipo = await acto('cerrar', equipo);
  const fotoEquipo = await foto(equipo, cierreEquipo.revision_actual);
  assert(await pagina.locator('#memoria-texto-d').isDisabled());
  assert.equal(cierreEquipo.equipo_id, equipo);
  assert.equal(await pagina.locator('#memoria-equipo').count(), 0);
  // Sin etf, el servidor infiere el equipo del profesional; no solicita la provincia.
  // El intento sobre el ámbito implícito ya cerrado no modifica la provincia.
  const provincialAntes = await (async () => { await entrar('ines.moreno'); return estadoMemoria(''); })();
  await entrar('marta.cabrera');
  const intento = await pedir(`/api/estadistica/memoria/cerrar?${consulta('')}`, 'POST', { version: cierreEquipo.version });
  assert([403, 409, 422].includes(intento.estado));
  await entrar('ines.moreno'); assert.deepEqual(await estadoMemoria(''), provincialAntes);
  await entrar('marta.cabrera');
  await api(`/api/estadistica/memoria/reabrir?${consulta(equipo)}`, 'POST', { version: cierreEquipo.version, motivo: 'Corrección de la ficha.' }, 403);
  await entrar('ines.moreno');
  const calculadoProvincia = await memoria(''); comprobarManual(baseProvincia, calculadoProvincia, 'provincia');
  await abrir(''); await bloques(calculadoProvincia); const cierreProvincia = await acto('cerrar', '');
  assert.equal(cierreProvincia.equipo_id, '');
  const fotoProvincia = await foto('', cierreProvincia.revision_actual);
  ok('F4.09-CIERRES: equipo cerrado por profesional; provincia por Coordinación; reapertura profesional denegada.');
  await entrar('marta.cabrera');
  const f = await api(ruta(b, 'familia'));
  await api(ruta(b), 'PATCH', { version: f.version, vigente_desde: '2026-07-01', familia: { ...f.familia, situacion_economica: 'SIN_DIFICULTADES' } });
  await entrar('ines.moreno');
  assert.deepEqual(await foto(equipo, cierreEquipo.revision_actual), fotoEquipo);
  assert.deepEqual(await foto('', cierreProvincia.revision_actual), fotoProvincia);
  const discrepancias = await api(`/api/estadistica/memoria/discrepancias?${consulta('')}`);
  await writeFile(resolve(evidencia, 'discrepancias.json'), JSON.stringify(discrepancias, null, 2));
  for (const [id, revision] of [[equipo, fotoEquipo], ['', fotoProvincia]]) {
    // El equipo del aviso identifica la fuente corregida, incluso en la provincial.
    assert(discrepancias.avisos.some(d => d.memoria_id === revision.memoria_id && d.equipo_id === equipo && d.expediente_id === b && d.apartados.includes('22')), `Aviso de corrección para ${id || 'provincia'}`);
    assert(discrepancias.diferencias.some(d => d.equipo_id === id && d.apartados.some(a => a.codigo === '22')));
  }
  ok('F4.09-DISCREPANCIA: economía corregida; aviso del apartado 22; fotos cerradas inmutables.');
  await entrar('lucia.navarro');
  const consultaEquipo = `/api/consulta/memoria/${anio}?equipo=${equipo}`;
  const soloCifras = await api(consultaEquipo);
  assert.equal(soloCifras.definitivas, true); assert(soloCifras.fecha_cifras);
  assert.deepEqual(Object.keys(soloCifras).sort(), ['anio', 'apartados', 'corte', 'definitivas', 'equipo_codigo', 'equipo_id', 'fecha_cifras'].sort());
  assert.deepEqual(soloCifras.apartados.map(a => [a.codigo, a.total]), calculado.apartados.map(a => [a.codigo, a.total]));
  const prohibidos = new Set(['narrativas', 'composicion', 'expedientes', 'fuentes', 'nombre', 'apellidos', 'texto_otros', 'expediente_id']);
  const sinDatos = v => { if (!v || typeof v !== 'object') return; for (const [k, d] of Object.entries(v)) { assert(!prohibidos.has(k.toLowerCase()), k); sinDatos(d); } };
  sinDatos(soloCifras);
  for (const valor of [a, b, c, 'Torres Montes', 'Daniel', 'Lucía', 'Se revisan las rutinas familiares']) assert(!JSON.stringify(soloCifras).includes(valor));
  for (const estado of ['SIN_DIFICULTADES', 'DIFICULTADES_SIN_NECESIDADES_BASICAS', 'DIFICULTADES_NECESIDADES_BASICAS'])
    cifra(cantidad(soloCifras, '22', 'familias', { estado }), cantidad(calculado, '22', 'familias', { estado }), `consulta/definitiva/22/${estado}`);
  await abrir(equipo, true); await bloques(soloCifras); await descargar('memoria-consulta.docx');
  assert.equal(await pagina.locator('#memoria-texto-d').count(), 0);
  for (const accion of ['cerrar', 'reabrir']) assert.equal(await pagina.getByRole('button', { name: t(`memoria.${accion}`), exact: true }).count(), 0);
  // La ruta anterior de Memoria también ofrece a Consulta su contrato numérico.
  assert.deepEqual(await api(`/api/estadistica/memoria?${consulta(equipo)}`), soloCifras);
  for (const rutaPrivada of [ruta(b), ruta(b, 'menores'), `/api/estadistica/memoria/discrepancias?${consulta('')}`, `/api/memoria/${anio}/cifras/22?etf=${equipo}`]) await api(rutaPrivada, 'GET', undefined, 403);
  await entrar('pablo.ruiz'); assert.notEqual((await api('/api/sesion')).equipo_id, equipo);
  for (const privada of [ruta(b), `/api/estadistica/memoria?${consulta(equipo)}`, `/api/estadistica/memoria/estado?${consulta(equipo)}`, `/api/estadistica/memoria/instantanea?${consulta(equipo)}&revision=${cierreEquipo.revision_actual}`, `/api/memoria/${anio}/cifras/22?etf=${equipo}`]) await denegarEquipo(privada);
  await denegarEquipo('/api/estadistica/memoria/word', 'POST', { anio, equipo_id: equipo, idioma: 'es' });
  await denegarEquipo(`/api/estadistica/memoria/cerrar?${consulta(equipo)}`, 'POST', { version: cierreEquipo.version });
  ok('F4.09-AISLAMIENTO: Consulta sólo cifras definitivas y Word; otro equipo sin acceso a cifras, ficha ni cierre.');
  await entrar('ines.moreno'); await abrir(''); await acto('reabrir', '');
  await abrir(equipo); await acto('reabrir', equipo);
  assert.deepEqual(await foto(equipo, cierreEquipo.revision_actual), fotoEquipo);
  assert.deepEqual(await foto('', cierreProvincia.revision_actual), fotoProvincia);
  const corregido = await memoria(equipo);
  cifra(cantidad(corregido, '22', 'familias', { estado: 'SIN_DIFICULTADES' }) - cantidad(calculado, '22', 'familias', { estado: 'SIN_DIFICULTADES' }), 1, 'reapertura/22/sin_dificultades');
  cifra(cantidad(corregido, '22', 'familias', { estado: 'DIFICULTADES_SIN_NECESIDADES_BASICAS' }) - cantidad(calculado, '22', 'familias', { estado: 'DIFICULTADES_SIN_NECESIDADES_BASICAS' }), -1, 'reapertura/22/dificultades');
  await entrar('lucia.navarro'); const provisional = await api(consultaEquipo); assert.equal(provisional.definitivas, false);
  for (const estado of ['SIN_DIFICULTADES', 'DIFICULTADES_SIN_NECESIDADES_BASICAS', 'DIFICULTADES_NECESIDADES_BASICAS'])
    cifra(cantidad(provisional, '22', 'familias', { estado }), cantidad(corregido, '22', 'familias', { estado }), `consulta/provisional/22/${estado}`);
  assert.deepEqual(provisional.apartados.map(a => [a.codigo, a.total]), corregido.apartados.map(a => [a.codigo, a.total]));
  ok('F4.09-REAPERTURA: Coordinación reabre ambas; cifras corregidas provisionales; revisiones originales conservadas.');
  assert.deepEqual(errores, []);
} catch (error) {
  fallosProducto.push({ caso: 'recorrido', error: error.stack });
  console.error(error);
  if (pagina) await pagina.screenshot({ path: resolve(evidencia, 'fallo.png'), fullPage: true }).catch(() => {});
} finally {
  await writeFile(resolve(evidencia, 'resultado.json'), JSON.stringify({ pasos, resultados, fallosProducto, errores }, null, 2));
  for (const [cuenta, p] of sesiones) {
    await p.context().tracing.stop({ path: resolve(evidencia, `${cuenta}.zip`) }); await p.context().close();
  }
  await navegador.close();
}
if (fallosProducto.length) { console.error(`F4.09-FALLO: ${fallosProducto.length} diferencias o pasos fallidos; ${resultados.length} contrastes ejecutados.`); process.exitCode = 1; }
else console.log(`F4.09-OK: ${resultados.length} contrastes manuales; cálculo, Word, cierres, corrección, discrepancia, aislamiento y reapertura.`);
