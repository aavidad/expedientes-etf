// GAP1: familia editable y documentos de finalización, con datos inventados.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { chromium, baseLocal, restringir } from './comun.mjs';
const base = baseLocal(process.argv[2]);
// Renueva sólo la reserva de textos GAP1 del volumen de esta demo local.
for (const idioma of ['es', 'en']) {
  const esperado = JSON.parse(await readFile(new URL(`../../idiomas/${idioma}/textos/gap1.json`, import.meta.url)));
  const actual = await (await fetch(`${base}/api/idiomas/${idioma}/textos`)).json();
  if (Object.entries(esperado).some(([k, v]) => actual[k] !== v)) execFileSync('python3', ['-c', `import importlib.util,sys
from pathlib import Path
s=importlib.util.spec_from_file_location('demo','despliegue/demo/demo.py');d=importlib.util.module_from_spec(s);s.loader.exec_module(d)
d.compose('cp', str(Path('idiomas') / sys.argv[1] / 'textos/gap1.json'), 'app:/datos/idiomas/' + sys.argv[1] + '/textos/gap1.json', capture_output=True, text=True)`, idioma], { cwd: new URL('../../', import.meta.url), stdio: 'pipe' });
}
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const carpeta = new URL('../../docs/capturas/', import.meta.url);
await mkdir(carpeta, { recursive: true });
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];
async function entrar(cuenta) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'es-ES', timezoneId: 'Europe/Madrid', acceptDownloads: true });
  await restringir(context, base);
  const p = await context.newPage(); p.setDefaultTimeout(15000);
  p.on('pageerror', (e) => errores.push(e.message));
  await p.goto(base); await p.locator('#cuenta-ad').fill(cuenta); await p.locator('#contrasena-ad').fill(claves[cuenta]);
  await p.locator('button[type=submit]').click(); await p.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return p;
}
async function pedir(p, ruta, method = 'GET', datos, clave = randomUUID()) {
  return p.evaluate(async ({ ruta, method, datos, clave }) => {
    const s = await (await fetch('/api/sesion')).json();
    const r = await fetch(ruta, { method, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': s.csrf, 'Idempotency-Key': clave }, ...(datos ? { body: JSON.stringify(datos) } : {}) });
    return { estado: r.status, datos: await r.json() };
  }, { ruta, method, datos, clave });
}
async function guardar(p, nombre) {
  await p.getByRole('button', { name: nombre, exact: true }).click();
  await p.getByText('Cambios guardados con su fecha de vigencia.', { exact: true }).waitFor();
}
async function captura(p, nombre) {
  for (const [width, sufijo] of [[1440, 'escritorio'], [390, 'movil']]) {
    await p.setViewportSize({ width, height: 1000 });
    await p.evaluate(() => document.fonts.ready); await p.waitForTimeout(100);
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `desbordamiento ${width}`);
    await p.screenshot({ path: new URL(`equipo-gap1-${nombre}-${sufijo}.png`, carpeta).pathname, fullPage: true });
  }
  for (const width of [320, 360, 1024]) {
    await p.setViewportSize({ width, height: 1000 });
    assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `desbordamiento ${width}`);
  }
  await p.setViewportSize({ width: 1440, height: 1000 });
}

async function crearCaso(q, tipo, equipo, hoy) {
  const bytes = Array.from(await readFile(new URL(`../../internal/adaptadores/anexos/pdf/testdata/${tipo}.pdf`, import.meta.url)));
  const leido = await q.evaluate(async ({ bytes, tipo }) => { const s = await (await fetch('/api/sesion')).json(); const body = new FormData(); body.append('anexo', new Blob([new Uint8Array(bytes)], { type: 'application/pdf' }), tipo + '.pdf'); const r = await fetch('/api/coordinacion/derivaciones', { method: 'POST', headers: { 'X-CSRF-Token': s.csrf }, body }); return { estado: r.status, datos: await r.json() }; }, { bytes, tipo });
  assert.equal(leido.estado, 200);
  const d = leido.datos; const opciones = (await pedir(q, '/api/coordinacion/decision/opciones?idioma=es')).datos;
  const municipio = opciones.municipios.find((m) => m.equipo_id === equipo && m.centro_ss_id); assert(municipio);
  const confirmada = await pedir(q, '/api/coordinacion/derivaciones/confirmar', 'POST', { lectura_id: d.lectura_anexo.id, tipo, campos: d.derivacion_datos.contenido.campos, ubicacion: { municipio_id: municipio.id, centro_id: municipio.centro_ss_id, equipo_id: equipo } }); assert.equal(confirmada.estado, 201);
  const aceptada = await pedir(q, `/api/coordinacion/derivaciones/${confirmada.datos.id}/aceptar`, 'POST', { version: confirmada.datos.version, fecha_acta: hoy, municipio_id: municipio.id, centro_ss_id: municipio.centro_ss_id, equipo_id: equipo, subprograma: tipo === 'anexo4' ? 'REUNIFICACION' : 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', profesional_referencia_id: null, menores: d.derivacion_datos.contenido.campos.filter((c) => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: 'M' })), descripcion_situacion: 'La familia solicita apoyo para organizar los cuidados y la asistencia escolar.' }); assert.equal(aceptada.estado, 200, JSON.stringify(aceptada.datos));
  return aceptada.datos.id;
}

try {
  const p = await entrar('marta.cabrera');
  const lista = await pedir(p, '/api/equipo/expedientes'); assert.equal(lista.estado, 200);
  const hoy = new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Madrid' }).format(new Date());
  const semilla = (await pedir(p, '/api/expedientes/' + lista.datos.filas[0].id + '/familia')).datos;
  const q = await entrar('ines.moreno');
  const id = await crearCaso(q, 'anexo3', semilla.equipo_id, hoy);
  const reunificacion = await crearCaso(q, 'anexo4', semilla.equipo_id, hoy);
  const caso = { id, subprograma: 'RIESGO_SOCIAL' };
  const ruta = '/api/expedientes/' + caso.id;
  const fichaURL = base + '/equipo/expediente?id=' + caso.id;
  await p.goto(fichaURL + '&pestana=familia');
  await p.getByRole('button', { name: 'Guardar datos familiares', exact: true }).waitFor().catch(async e => { console.log((await p.locator('main').innerText()).slice(-2000)); console.log(await pedir(p, ruta + '/familia')); throw e; });
  await p.locator('#gap1-vigencia').fill(hoy);
  await p.locator('#fi1-direccion').fill('Calle Arrayanes, 18');
  assert(await p.getByRole('button', { name: 'Guardar personas adultas', exact: true }).isDisabled());
  await p.locator('#fi1-telefono_contacto').fill('958234671'); await p.locator('#fi1-email_contacto').fill('familia.arrayanes@correo.invalid');
  await p.locator('#fi1-composicion_familiar').selectOption('OTRAS');
  await p.locator('#fi1-composicion_otras_texto').fill('Convive con una tía y mantiene contacto con ambos progenitores.');
  await p.locator('#fi1-familia_numerosa').selectOption('true'); await p.locator('#fi1-vivienda').selectOption('ALQUILER');
  await p.locator('#fi1-situacion_economica').selectOption('DIFICULTADES_NECESIDADES_BASICAS');
  await p.locator('#fi1-ingresos_anuales_centimos').fill('18000,75');
  await guardar(p, 'Guardar datos familiares');
  let actual = (await pedir(p, ruta + '/familia')).datos;
  assert.equal(actual.familia.ingresos_anuales_centimos, 1800075); assert.equal(actual.familia.tramo_smi, null);
  assert.equal(actual.familia.direccion, 'Calle Arrayanes, 18'); assert.equal(actual.familia.familia_numerosa, true);
  const historia = await pedir(p, ruta + '/familia/historia?corte=' + hoy + '&anio=' + hoy.slice(0, 4));
  assert.equal(historia.estado, 200); assert.equal(historia.datos.familia.ingresos_anuales_centimos, 1800075);
  console.log('GAP1-FAMILIA-OK: PATCH, vigencia y céntimos exactos recuperados; historia al corte');
  // Tramo directo e importe se excluyen también en la petición real.
  const opcionTramo = await p.locator('#fi1-tramo_smi option').nth(1).getAttribute('value'); assert(opcionTramo);
  await p.locator('#fi1-tramo_smi').selectOption(opcionTramo);
  assert.equal(await p.locator('#fi1-ingresos_anuales_centimos').inputValue(), '');
  await guardar(p, 'Guardar datos familiares');
  actual = (await pedir(p, ruta + '/familia')).datos;
  assert.equal(actual.familia.ingresos_anuales_centimos, null); assert.equal(actual.familia.tramo_smi, opcionTramo);
  // Alta completa de persona adulta; el identificador se recupera antes de editar otra vez.
  await p.getByRole('button', { name: 'Añadir persona adulta', exact: true }).click();
  const indice = actual.adultos.length; const sufijo = '-nuevo-' + indice;
  await p.locator('#fi1-nombre' + sufijo).fill('Elena'); await p.locator('#fi1-apellidos' + sufijo).fill('Serrano Molina');
  await p.locator('#fi1-papel' + sufijo).selectOption({ label: 'Madre' }); await p.locator('#fi1-sexo' + sufijo).selectOption('M');
  await p.locator('#fi1-fecha_nacimiento' + sufijo).fill('1987-03-14'); await p.locator('#fi1-es_tutor' + sufijo).selectOption('true');
  await p.locator('#fi1-nivel_educativo' + sufijo).selectOption('OBLIGATORIOS'); await p.locator('#fi1-situacion_laboral' + sufijo).selectOption('OTRAS');
  await p.locator('#fi1-situacion_laboral_texto' + sufijo).fill('Trabajo por temporadas en un vivero.');
  await p.locator('#fi1-telefono' + sufijo).fill('658231746'); await p.locator('#fi1-email' + sufijo).fill('elena.serrano@correo.invalid');
  await p.locator('#fi1-otros_datos' + sufijo).fill('Prefiere recibir llamadas por la tarde.');
  // La siembra puede conservar adultos sin papel/sexo: completar antes del PUT total.
  for (const adulto of actual.adultos) {
    await p.locator('#fi1-papel-' + adulto.id).evaluate((e) => { e.closest('details').open = true; });
    if (!adulto.papel) await p.locator('#fi1-papel-' + adulto.id).selectOption({ label: 'Madre' });
    if (!adulto.sexo) await p.locator('#fi1-sexo-' + adulto.id).selectOption('M');
  }
  await guardar(p, 'Guardar personas adultas');
  actual = (await pedir(p, ruta + '/familia')).datos;
  const adulta = actual.adultos.find((a) => a.nombre === 'Elena'); assert(adulta?.id); assert.equal(adulta.situacion_laboral_texto, 'Trabajo por temporadas en un vivero.');
  assert.equal(actual.familia.tramo_smi, opcionTramo);
  console.log('GAP1-ADULTOS-OK: PUT, alta completa, identidad estable y familia conservada');
  // F2.02: el enlace mueve el foco y abre el desplegable de la persona.
  const aviso = (await pedir(p, ruta + '/avisos')).datos.avisos.find((a) => a.sujeto_id && actual.adultos.some((s) => s.id === a.sujeto_id));
  if (aviso) {
    await p.locator('#gap1-avisos').locator('..').locator(`a[href="#fi1-${aviso.campo}-${aviso.sujeto_id}"]`).click();
    assert.equal(await p.evaluate(() => document.activeElement.id), `fi1-${aviso.campo}-${aviso.sujeto_id}`);
  }
  // Validación y guardia de salida.
  await p.locator('#fi1-ingresos_anuales_centimos').fill('18.000,50');
  await p.getByRole('button', { name: 'Guardar datos familiares', exact: true }).click();
  await p.getByText('Introduce los ingresos en euros, sin separador de miles y con un máximo de dos decimales.', { exact: true }).first().waitFor();
  await p.getByRole('tab', { name: 'Cierre', exact: true }).click();
  await p.getByRole('button', { name: 'Seguir editando', exact: true }).click(); assert.equal(await p.locator('#fi1-ingresos_anuales_centimos').inputValue(), '18.000,50');
  // Conflicto real: otro guardado incrementa versión; comparar no destruye borrador.
  await p.locator('#fi1-ingresos_anuales_centimos').fill('21000,40'); await p.locator('#fi1-direccion').fill('Calle Laurel, 7');
  const paralelo = await pedir(p, ruta, 'PATCH', { version: actual.version, vigente_desde: hoy, familia: { ...actual.familia, direccion: 'Calle Olmos, 9' } }); assert.equal(paralelo.estado, 200);
  await p.getByRole('button', { name: 'Guardar datos familiares', exact: true }).click();
  await p.getByRole('button', { name: 'Ver cambios actuales', exact: true }).click();
  await p.getByText('Calle Olmos, 9', { exact: true }).waitFor();
  assert.equal(await p.locator('#fi1-direccion').inputValue(), 'Calle Laurel, 7');
  await p.getByRole('button', { name: 'Conservar mi formulario para guardar', exact: true }).click(); await guardar(p, 'Guardar datos familiares');
  actual = (await pedir(p, ruta + '/familia')).datos; assert.equal(actual.familia.direccion, 'Calle Laurel, 7');
  assert.equal(actual.familia.ingresos_anuales_centimos, 2100040);
  console.log('GAP1-409-OK: borrador intacto, comparación y guardado tras conflicto real');
  // Aviso global de un adulto: navegación desde Documentos y foco dentro del desplegable.
  await p.locator('#fi1-nivel_educativo-' + adulta.id).evaluate(e => { e.closest('details').open = true; });
  await p.locator('#fi1-nivel_educativo-' + adulta.id).selectOption(''); await guardar(p, 'Guardar personas adultas');
  assert.equal((await pedir(p, ruta + '/familia')).datos.adultos.find(a => a.id === adulta.id).nivel_educativo, null);
  await p.getByRole('tab', { name: 'Documentos', exact: true }).click();
  await p.locator('#fi1-ficha-avisos').click();
  await p.locator('#fi1-ficha-avisos').locator('..').locator('a[href="#fi1-nivel_educativo-' + adulta.id + '"]').click();
  await p.waitForFunction(id => document.activeElement?.id === id, 'fi1-nivel_educativo-' + adulta.id);
  assert.equal(await p.getByRole('tab', { name: 'Familia', exact: true }).getAttribute('aria-selected'), 'true');
  await p.locator('#fi1-nivel_educativo-' + adulta.id).selectOption('OBLIGATORIOS'); await guardar(p, 'Guardar personas adultas');
  console.log('GAP1-AVISOS-OK: educación desconocida guardada como null; aviso global abre Familia y enfoca adulto');
  await captura(p, 'familia');
  await p.getByRole('tab', { name: 'Cierre', exact: true }).click();
  await p.getByRole('button', { name: 'Descargar Anexo 11', exact: true }).waitFor();
  const motivo = p.locator('#equipo-motivo');
  if (await motivo.isVisible()) {
    await motivo.selectOption(await motivo.locator('option').nth(1).getAttribute('value'));
    const sinConsentimiento = p.locator('.equipo-casilla input[type=checkbox]');
    if (await sinConsentimiento.isVisible()) await sinConsentimiento.check();
    await p.locator('#equipo-cierre-objetivos_abordados').fill('Se han reforzado las rutinas de cuidado y la asistencia escolar.');
    assert(await p.getByRole('button', { name: 'Descargar Anexo 11', exact: true }).isDisabled());
    await p.getByRole('button', { name: 'Guardar datos de finalización', exact: true }).click();
    await p.getByText('Se han guardado los cambios.', { exact: true }).waitFor();
  }
  const anterior = (await pedir(p, '/api/equipo/expedientes/' + caso.id)).datos;
  for (const anexo of [caso.subprograma === 'REUNIFICACION' ? 12 : 11]) {
    const descarga = p.waitForEvent('download');
    await p.getByRole('button', { name: 'Descargar Anexo ' + anexo, exact: true }).click();
    const fichero = await descarga; assert.equal(fichero.suggestedFilename(), 'anexo' + anexo + '.docx');
    const contenido = await readFile(await fichero.path()); assert.equal(contenido.subarray(0, 2).toString(), 'PK'); assert(contenido.length > 1000);
    await p.getByText('Se ha descargado el Word. Contiene datos personales: guárdalo sólo donde te indique el Programa.', { exact: true }).waitFor();
  }
  const posterior = (await pedir(p, '/api/equipo/expedientes/' + caso.id)).datos;
  assert.deepEqual(posterior.hitos, anterior.hitos); assert.equal(posterior.estado, anterior.estado);
  assert(!posterior.avisos.includes('equipo.aviso.cierre_pendiente'));
  assert(posterior.emisiones.some((e) => e.codigo === 'finalizacion/' + (caso.subprograma === 'REUNIFICACION' ? 'anexo12' : 'anexo11')));
  console.log('GAP1-ANEXO-OK: Word del subprograma real, emisión auditada; estado e hitos conservados');
  await captura(p, 'cierre');
  // Reunificación: su acción independiente descarga Anexo 12 y omite el incompatible.
  await p.goto(base + '/equipo/expediente?id=' + reunificacion + '&pestana=cierre');
  await p.getByRole('button', { name: 'Descargar Anexo 12', exact: true }).waitFor();
  assert.equal(await p.getByRole('button', { name: 'Descargar Anexo 11', exact: true }).count(), 0);
  const motivo12 = p.locator('#equipo-motivo'); await motivo12.selectOption(await motivo12.locator('option').nth(1).getAttribute('value'));
  if (await p.locator('.equipo-casilla input').isVisible()) await p.locator('.equipo-casilla input').check();
  await p.getByRole('button', { name: 'Guardar datos de finalización', exact: true }).click(); await p.getByText('Se han guardado los cambios.', { exact: true }).waitFor();
  const download12 = p.waitForEvent('download'); await p.getByRole('button', { name: 'Descargar Anexo 12', exact: true }).click();
  const doc12 = await download12; assert.equal(doc12.suggestedFilename(), 'anexo12.docx'); assert.equal((await readFile(await doc12.path())).subarray(0, 2).toString(), 'PK');
  const despues12 = (await pedir(p, '/api/equipo/expedientes/' + reunificacion)).datos;
  assert.equal(despues12.estado, 'aceptada'); assert(!despues12.avisos.includes('equipo.aviso.cierre_pendiente'));
  assert(despues12.emisiones.some((e) => e.codigo === 'finalizacion/anexo12'));
  console.log('GAP1-ANEXO12-OK: reunificación, Word real y cierre sin proponer');
  // Teclado y retirada con confirmación conservan el resto del expediente.
  await p.goto(fichaURL + '&pestana=familia'); await p.getByRole('button', { name: 'Guardar personas adultas', exact: true }).waitFor();
  await p.locator('#fi1-nombre-' + adulta.id).evaluate(e => { e.closest('details').open = true; });
  await p.locator('#fi1-nombre-' + adulta.id).focus(); await p.keyboard.press('Tab');
  assert.equal(await p.evaluate(() => document.activeElement.id), 'fi1-apellidos-' + adulta.id);
  await p.getByRole('button', { name: 'Retirar persona adulta', exact: true }).last().click();
  await p.getByRole('button', { name: 'Confirmar retirada del formulario', exact: true }).click(); await guardar(p, 'Guardar personas adultas');
  assert(!(await pedir(p, ruta + '/familia')).datos.adultos.some(a => a.id === adulta.id));
  console.log('GAP1-RETIRADA-OK: confirmación, PUT y lectura sin persona retirada; teclado entre campos');
  async function preferencia(selector, valor) {
    if (!await p.locator('.persona-conectada').evaluate(e => e.open)) await p.locator('.persona-conectada > summary').click();
    if (!await p.locator('details.preferencias').evaluate(e => e.open)) await p.locator('details.preferencias > summary').click();
    await p.locator(selector).selectOption(valor);
  }
  await preferencia('#tema', 'alto-contraste'); await preferencia('#idioma', 'en');
  await p.getByRole('button', { name: 'Save family details', exact: true }).waitFor();
  for (const width of [390, 1024]) { await p.setViewportSize({ width, height: 1000 }); assert(await p.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)); }
  await p.screenshot({ path: new URL('equipo-gap1-familia-contraste.png', carpeta).pathname, fullPage: true });
  await preferencia('#idioma', 'es'); await preferencia('#tema', 'expedientes-etf');
  console.log('GAP1-TEMA-OK: inglés y alto contraste con reflujo 390/1024');
  // Coordinación consulta; servidor y pantalla deniegan la edición y la emisión.
  await q.goto(fichaURL + '&pestana=familia');
  await q.getByText('Puedes consultar estos datos. La edición corresponde al profesional del ETF en un expediente abierto.', { exact: true }).waitFor();
  assert.equal(await q.getByRole('button', { name: 'Guardar datos familiares', exact: true }).count(), 0);
  assert.equal((await pedir(q, ruta, 'PATCH', { version: actual.version, vigente_desde: hoy, familia: actual.familia })).estado, 403);
  assert.equal((await pedir(q, ruta + '/adultos', 'PUT', { version: actual.version, vigente_desde: hoy, adultos: actual.adultos })).estado, 403);
  await q.getByRole('tab', { name: 'Cierre', exact: true }).click();
  assert.equal(await q.getByRole('button', { name: 'Descargar Anexo 11', exact: true }).count(), 0);
  assert.equal((await pedir(q, ruta + '/documentos/anexo11', 'POST', { version: actual.version, idioma: 'es' })).estado, 403);
  console.log('GAP1-PERFILES-OK: Inés consulta; edición y anexos denegados en HTTP');
  assert.deepEqual(errores, []); console.log('GAP1-OK: Chrome del sistema; escritorio 1440, móvil 390, reflujo 320/360/1024; sin errores JS');
} finally { await browser.close(); }
