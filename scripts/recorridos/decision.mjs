// Recorrido P3 con datos inventados, Chrome del sistema y servicios de la demo.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
const origen = process.argv[2] ?? 'http://127.0.0.1:18186';
assert.match(origen, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
const pagina = await contexto.newPage(); pagina.setDefaultTimeout(20000);
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
async function abrir(d) {
  console.log('PASO: Abrir detalle');
  await pagina.goto(`${origen}/coordinacion/derivaciones/${d.id}`);
  await pagina.getByRole('heading', { name: 'Municipio, centro y equipo', exact: true }).waitFor();
}
async function completar() {
  console.log('PASO: Completar aceptación');
  if (!(await pagina.locator('#decision-municipio_id').inputValue())) {
    const opciones = await api('/api/coordinacion/decision/opciones?idioma=es');
    const m = opciones.datos.municipios.find(m => m.equipo_id && m.centro_ss_id);
    await pagina.locator('#decision-municipio_id').selectOption(m.id);
  }
  if (!(await pagina.locator('#decision-descripcion_situacion').inputValue()).trim()) await pagina.locator('#decision-descripcion_situacion').fill('La familia solicita apoyo para organizar los cuidados y mejorar la asistencia escolar de los menores.');
  await pagina.locator('#decision-profesional_referencia_puesto').selectOption('TS');
  await pagina.locator('#decision-fecha_acta').fill('2026-10-13');
  if (await pagina.locator('#decision-subprograma').isEnabled()) await pagina.locator('#decision-subprograma').selectOption('RIESGO_SOCIAL');
  for (const campo of await pagina.locator('input[type="radio"][value="M"]').all()) await campo.check();
}
async function confirmar(nombre) {
  console.log('PASO: Confirmar decisión');
  const dialogo = pagina.getByRole('dialog'); await dialogo.waitFor();
  const r = pagina.waitForResponse(r => r.request().method() === 'POST' && /\/(aceptar|no-aceptar|lista-espera)$/.test(r.url()));
  await dialogo.getByRole('button', { name: nombre, exact: true }).click();
  const respuesta = await r; console.log('RESPUESTA: decisión', respuesta.status()); return { estado: respuesta.status(), datos: await respuesta.json() };
}
async function descargar(nombre) {
  console.log('PASO: Descargar documento');
  const descarga = pagina.waitForEvent('download').catch(() => null);
  const respuesta = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith('/api/documentos/emisiones'));
  await pagina.getByRole('button', { name: `Descargar ${nombre}`, exact: true }).click();
  const r = await respuesta; assert.equal(r.status(), 201, r.ok() ? '' : await r.text()); assert.match(r.headers()['x-emision-id'], /^[a-f0-9-]+$/);
  const archivo = await descarga; assert(archivo); assert.equal(await archivo.failure(), null);
  await pagina.getByText('Documento descargado.', { exact: false }).waitFor();
  return r.headers()['x-emision-id'];
}
async function registrar(id, numero) {
  console.log('PASO: Anotar registro');
  const n = pagina.locator(`#registro-numero-${id}`); await n.waitFor();
  await n.fill(numero); await pagina.locator(`#registro-fecha-${id}`).fill('2026-10-13');
  const formulario = n.locator('xpath=ancestor::form');
  const respuesta = pagina.waitForResponse(r => r.request().method() === 'POST' && r.url().endsWith(`/emisiones/${id}/registro-salida`));
  await formulario.getByRole('button', { name: 'Anotar registro de salida', exact: true }).click();
  assert.equal((await respuesta).status(), 200); await pagina.getByText(`Registro de salida anotado: ${numero}`, { exact: false }).waitFor();
}
async function ancho(width) {
  await pagina.setViewportSize({ width, height: 1000 });
  await pagina.waitForTimeout(100);
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `desbordamiento ${width}`);
}
try {
  async function entrar(cuenta) {
    await pagina.goto(origen);
    await pagina.locator('#cuenta-ad').fill(cuenta); await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
    await pagina.locator('button[type="submit"]').click(); await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  }
  await entrar('elena.salas');
  const firmantes = await api('/api/admin/firmantes'); assert.equal(firmantes.estado, 200);
  if (!firmantes.datos.firmantes.some(f => f.rol === 'jefe_spm')) {
    const alta = await api('/api/admin/firmantes', {
      version_esperada: 0, referencias_esperadas: firmantes.datos.referencias,
      vigente_desde: '2026-01-01T00:00:00Z',
      textos: { es: 'Jefatura del Servicio de Protección de Menores', en: 'Head of the Child Protection Service' },
      firmante: { tipo: 'destinatario', rol: 'jefe_spm', nombre: 'Beatriz Campos Romero', cargo_codigo: 'JEFATURA_SPM', organismo_codigo: 'JUNTA_ANDALUCIA' },
    }); assert.equal(alta.estado, 201, JSON.stringify(alta.datos));
  }
  await api('/api/sesion', undefined, undefined, 'DELETE');
  await entrar('ines.moreno');
  const recibida = await crear(); await abrir(recibida);
  await pagina.getByRole('button', { name: 'Aceptar derivación', exact: true }).click();
  await pagina.getByRole('heading', { name: 'Revisa estos campos antes de decidir' }).waitFor();
  await completar();
  await mkdir(new URL('../../docs/capturas/', import.meta.url), { recursive: true });
  await pagina.screenshot({ path: new URL('../../docs/capturas/decision-escritorio.png', import.meta.url).pathname, fullPage: true });
  for (const width of [1024, 390, 360]) await ancho(width);
  await ancho(390); await pagina.screenshot({ path: new URL('../../docs/capturas/decision-movil.png', import.meta.url).pathname, fullPage: true });
  await ancho(1440);
  await pagina.getByRole('button', { name: 'Aceptar derivación', exact: true }).click();
  await pagina.getByRole('dialog').waitFor(); await pagina.keyboard.press('Escape'); assert.equal(await pagina.getByRole('dialog').count(), 0);
  assert.equal((await api(`/api/coordinacion/derivaciones/${recibida.id}`)).datos.estado, 'derivacion_recibida');
  await pagina.getByRole('button', { name: 'Aceptar derivación', exact: true }).click();
  const aceptada = await confirmar('Aceptar y asignar número'); assert.equal(aceptada.estado, 200); assert.match(aceptada.datos.numero, /^\d{2}-\d{2}-26$/);
  await pagina.getByText(aceptada.datos.numero, { exact: true }).first().waitFor();
  await descargar('Acta de Aceptación'); const oficio = await descargar('Oficio a SS.SS.CC.'); await registrar(oficio, '2026/14873');
  await pagina.reload(); await pagina.getByText('Registro de salida anotado: 2026/14873', { exact: false }).waitFor();
  await pagina.locator(`#registro-numero-${oficio}`).fill('2026/14874');
  assert(await pagina.getByRole('button', { name: 'Descargar Oficio a SS.SS.CC.', exact: true }).isDisabled());
  await registrar(oficio, '2026/14874');
  const ficha = (await api(`/api/coordinacion/derivaciones/${recibida.id}`)).datos; assert.equal(ficha.aceptacion.fecha_acta, '2026-10-13'); assert.equal(ficha.numero, aceptada.datos.numero);
  const espera = await crear(); await abrir(espera);
  await pagina.getByRole('button', { name: 'Dejar en lista de espera', exact: true }).click(); assert.equal((await confirmar('Confirmar lista de espera')).estado, 200);
  const oficioEspera = await descargar('Oficio de Lista de Espera'); await registrar(oficioEspera, '2026/14875');
  await pagina.goto(`${origen}/coordinacion/lista-espera`); await pagina.getByRole('button', { name: 'Revisar derivación', exact: true }).first().waitFor();
  await abrir(espera); await completar(); await pagina.getByRole('button', { name: 'Aceptar derivación', exact: true }).click(); assert.equal((await confirmar('Aceptar y asignar número')).estado, 200);
  await registrar(oficioEspera, '2026/14875A');
  assert.equal(await pagina.getByRole('button', { name: 'Descargar Oficio de Lista de Espera', exact: true }).count(), 0);
  const no = await crear(); await abrir(no); await pagina.getByRole('button', { name: 'No aceptar derivación', exact: true }).click();
  await pagina.locator('#decision-motivo').fill('La intervención propuesta corresponde al servicio municipal de apoyo familiar.'); await pagina.locator('#decision-fecha_no').fill('2026-10-13');
  await pagina.getByRole('button', { name: 'Revisar no aceptación', exact: true }).click(); assert.equal((await confirmar('Confirmar no aceptación')).estado, 200);
  await descargar('Acta de No Aceptación'); assert.equal((await api(`/api/coordinacion/derivaciones/${no.id}`)).datos.numero, null);
  const spm = await crear('anexo4'); await abrir(spm); await completar(); await pagina.getByRole('button', { name: 'Aceptar derivación', exact: true }).click(); assert.equal((await confirmar('Aceptar y asignar número')).estado, 200);
  const oficioSPM = await descargar('Oficio al SPM'); await registrar(oficioSPM, '2026/14876');
  const concurrente = await crear(); await abrir(concurrente); await completar();
  assert.equal((await api(`/api/coordinacion/derivaciones/${concurrente.id}/lista-espera`, { version: concurrente.version })).estado, 200);
  await pagina.getByRole('button', { name: 'Aceptar derivación', exact: true }).click(); assert.equal((await confirmar('Aceptar y asignar número')).estado, 409);
  await pagina.getByRole('button', { name: 'Ver cambios', exact: true }).click(); await pagina.getByRole('button', { name: 'Descartar cambios', exact: true }).click();
  await pagina.getByText('Lista de espera', { exact: true }).last().waitFor();
  await ancho(1024);
  const orden = await pagina.locator('.decision-lateral').evaluate(e => getComputedStyle(e).order); assert.equal(orden, '-1');
  await ancho(1440);
  await pagina.locator('.persona-conectada > summary').click();
  await pagina.locator('.preferencias > summary').click();
  await pagina.locator('#tema').selectOption('alto-contraste');
  await pagina.waitForFunction(() => document.querySelector('link[href="/api/temas/alto-contraste/tokens"]'));
  await pagina.locator('#idioma').selectOption('en');
  await pagina.getByRole('heading', { name: 'Review referral', exact: true }).waitFor();
  await ancho(390); await ancho(360);
  assert.match(await pagina.locator('main').ariaSnapshot(), /Review referral/);
  assert.deepEqual(errores, []);
  console.log('RECORRIDO-P3-OK: Chrome; aceptación y número; espera→aceptación; no aceptación; cinco Word; tres registros persistentes; conflicto 409; Escape; 1440/1024/390/360 sin desbordamiento.');
} catch (error) { console.error(error); throw error; } finally { await navegador.close(); }
