// Un único expediente nuevo atraviesa las escenas y conserva sus actos confirmados.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { sesionVideo } from './sesion-video.mjs';
import { pedir, descargar, pulsar, apuntar, elegir, escribir } from './datos-video.mjs';
import { guardar } from './comun.mjs';

export async function grabarCiclo(opciones) {
  const { guion, dir, referencias } = opciones;
  const c = guion.config;
  const puerto = readFileSync(new URL('../../despliegue/demo/.local/puerto', import.meta.url), 'utf8').trim();
  assert.equal(process.env.ETF_VIDEO_URL, `http://127.0.0.1:${puerto}`);
  assert.equal(process.env[c.coordinacion_env], c.coordinacion_demo);
  assert.equal(process.env[c.profesional_env], c.profesional_demo);
  assert.equal(process.env[c.segunda_env], c.segunda_demo);
  assert.equal(new Set([c.coordinacion_demo, c.profesional_demo, c.segunda_demo]).size, 3);
  // El arranque comprueba perfiles y consume el presupuesto HTTP de la demo.
  // La pausa precede a la grabación para conservar el ancla en los primeros segundos.
  await new Promise(resolve => setTimeout(resolve, 61000));
  const s = await sesionVideo(opciones, ['anexo', 'bandeja', 'decision', 'equipo', 'fi1', 'fi2', 'gap1', 'actividad', 'spaf', 'salidas', 'memoria']);
  const { page, t, boton, respuesta, leer } = s;
  const calendario = async fecha => { execFileSync('python3', ['-B', new URL('./reloj_video.py', import.meta.url).pathname, 'fijar', fecha], { cwd: new URL('../../', import.meta.url), stdio: 'pipe' }); await page.clock.setFixedTime(new Date(`${fecha}T10:00:00Z`)); };
  const estado = { modo: 'demo', recorrido: 'ciclo', escrituras: [], descargas: {} };
  let expediente, derivacion, equipo, menores, adulta, comision, informe;
  const tab = id => pulsar(page.locator(`#equipo-tab-${id}`));
  const salir = async () => { await pulsar(page.locator('details.persona-conectada > summary')); await pulsar(boton('identidad.salir')); await boton('identidad.entrar').waitFor(); };
  const entrar = async perfil => { await s.entrar(perfil); assert.equal((await leer('/api/sesion')).usuario.cuenta, c[`${perfil}_demo`]); };
  const escena = (id, a, b, d) => s.escena(id, [a, b, d]);
  const ruta = sufijo => `/api/expedientes/${expediente.id}${sufijo}`;
  const guardarCambio = async (url, metodo, accion) => {
    const registro = { ruta: url, metodo, confirmado: false }; estado.escrituras.push(registro);
    const pendiente = respuesta(url, metodo); await accion(); const r = await pendiente;
    assert([200, 201].includes(r.status()), `Escritura local ${url}: ${r.status()}`);
    const datos = r.headers()['content-type']?.startsWith('application/json') ? await r.json() : { id: r.headers()['x-emision-id'] }; registro.confirmado = true; registro.version = datos.version; registro.id = datos.id; return datos;
  };
  const abrirFicha = async () => { await page.goto(`${process.env.ETF_VIDEO_URL}/equipo/expediente?id=${expediente.id}`); await page.getByRole('tab').first().waitFor(); };
  const consulta = id => new URLSearchParams({ anio: c.anio, ...(id ? { etf: id } : {}) });
  const prepararMemorias = async () => {
    for (const id of [equipo, '']) {
      const anterior = await leer(`/api/estadistica/memoria/estado?${consulta(id)}`);
      if (anterior.estado === 'cerrada') await pedir(page, `/api/estadistica/memoria/reabrir?${consulta(id)}`, { version: anterior.version, motivo: c.motivo_preparacion });
    }
  };
  const abrirMemoria = async () => { await pulsar(page.locator('a[href="/memoria"]').first()); await page.locator('.memoria .pildora').waitFor(); if (await page.locator('#memoria-anio').inputValue() !== String(c.anio)) { await page.locator('#memoria-anio').fill(String(c.anio)); await pulsar(boton('memoria.consultar')); await page.locator('.memoria .pildora').waitFor(); } assert.equal(await page.locator('#memoria-anio').inputValue(), String(c.anio)); };
  const cerrarMemoria = async id => {
    const antes = await leer(`/api/estadistica/memoria/estado?${consulta(id)}`);
    const nuevo = await guardarCambio('/api/estadistica/memoria/cerrar', 'POST', () => pulsar(page.getByRole('dialog').getByRole('button', { name: t('memoria.confirmar_cierre'), exact: true })));
    assert.equal(nuevo.estado, 'cerrada'); assert.equal(nuevo.revision_actual, antes.revision_actual + 1);
    const foto = await leer(`/api/estadistica/memoria/instantanea?${consulta(id)}&revision=${nuevo.revision_actual}`);
    assert.equal(foto.informe.equipo_id, id); assert.equal(foto.informe.anio, c.anio);
    await page.getByRole('status').filter({ hasText: t('memoria.cierre_guardado') }).waitFor();
    return nuevo;
  };
  try {
    await calendario(c.fecha_aceptacion);
    await escena('bandeja', () => entrar('coordinacion'), () => apuntar(page.getByRole('heading', { name: t('bandeja.titulo'), exact: true })), () => apuntar(page.locator('a[href="/coordinacion"]').first()));
    await escena('subir', async () => { await pulsar(page.getByRole('link', { name: t('bandeja.subir'), exact: true })); await apuntar(page.locator('#anexo-fichero')); await page.locator('#anexo-fichero').setInputFiles(new URL(`../../${c.anexo}`, import.meta.url).pathname); }, async () => {
      const r = respuesta('/api/coordinacion/derivaciones'); await pulsar(boton('anexo.subir')); assert.equal((await r).status(), 200); assert.equal((await (await r).json()).lectura_anexo.antivirus, 'limpio');
      await page.getByRole('heading', { name: t('anexo.revisar'), exact: true }).waitFor();
    }, () => apuntar(page.getByText(t('anexo.antivirus_limpio'), { exact: true })));
    await escena('revisar', async () => { for (const id of ['municipio_id', 'centro_id', 'equipo_id']) assert(await page.locator(`#anexo-${id}`).inputValue()); await apuntar(page.locator('#anexo-equipo_id')); }, async () => {
      const bloque = page.locator('details.anexo-tarjeta').filter({ has: page.locator('summary', { hasText: t('anexo.apartado.familia') }) }).first(); await bloque.evaluate(n => { n.open = true; }); await apuntar(bloque);
    }, async () => { const bloque = page.locator('details.anexo-tarjeta').filter({ has: page.locator('summary', { hasText: t('anexo.apartado.menores') }) }).first(); await bloque.evaluate(n => { n.open = true; }); await apuntar(bloque); });
    await escena('crear', async () => { derivacion = await guardarCambio('/api/coordinacion/derivaciones/confirmar', 'POST', () => pulsar(boton('anexo.confirmar'))); assert.equal(derivacion.estado, 'derivacion_recibida'); assert(!derivacion.numero); }, async () => { await page.locator('#decision-estado').waitFor(); await apuntar(page.locator('#decision-estado')); }, () => apuntar(page.locator('#decision-subprograma')));
    await escena('aceptar', async () => { await elegir(page.locator('#decision-profesional_referencia_puesto'), 'TS'); await page.locator('#decision-fecha_acta').fill(c.fecha_aceptacion); }, async () => {
      for (const [i, sexo] of c.menores.map(m => m.sexo).entries()) await pulsar(page.locator(`input[name="sexo-${i + 1}"][value="${sexo}"]`));
    }, async () => { await pulsar(boton('decision.aceptar')); expediente = await guardarCambio(`/api/coordinacion/derivaciones/${derivacion.id}/aceptar`, 'POST', () => pulsar(page.getByRole('dialog').getByRole('button', { name: t('decision.confirmar.aceptar'), exact: true }))); assert.equal(expediente.estado, 'aceptada'); estado.expediente = expediente; equipo = expediente.equipo_id; await apuntar(page.locator('#decision-estado')); });
    await escena('acta', async () => { const d = page.waitForEvent('download'); await guardarCambio('/api/documentos/emisiones', 'POST', () => pulsar(page.getByRole('button', { name: `${t('decision.descargar')} ${t('decision.acta_aceptacion')}`, exact: true }))); const fichero = await d; assert.equal(await fichero.failure(), null); await fichero.saveAs(resolve(dir, 'acta-aceptacion.docx')); estado.descargas.acta_aceptacion = true; }, () => apuntar(page.getByRole('status').filter({ hasText: t('decision.descargado') })), () => apuntar(page.locator('#decision-estado')));
    await prepararMemorias(); await salir(); await calendario(c.fecha_datos);
    await escena('ficha', async () => { await entrar('profesional'); assert.equal((await leer('/api/sesion')).equipo_id, equipo);
      const fichaMenores = await leer(ruta('/menores')); assert.equal(fichaMenores.menores.length, c.menores.length);
      const datosMenores = fichaMenores.menores.map((m, i) => ({ ...m, ...c.menores[i], fecha_nacimiento: i ? c.menores[i].fecha_nacimiento : null, propuesto_atencion_temprana: false }));
      const preparada = await pedir(page, ruta('/menores'), { version: fichaMenores.version, vigente_desde: c.fecha_datos, menores: datosMenores }, 'PUT');
      estado.preparacion_menores = { version: preparada.version, nombres: datosMenores.map(m => m.nombre), expediente_id: expediente.id };
      assert.deepEqual((await leer(ruta('/menores'))).menores.map(m => m.nombre), c.menores.map(m => m.nombre));
      await pulsar(page.locator('a[href="/equipo"]').first()); await page.locator('#equipo-busqueda').fill(expediente.numero); await pulsar(boton('equipo.buscar_accion')); }, async () => { await pulsar(page.getByRole('link', { name: expediente.numero, exact: true })); await page.getByRole('tab').first().waitFor(); }, () => apuntar(page.locator('#fi1-ficha-avisos')));
    await escena('familia', async () => { await tab('familia'); await page.locator('#gap1-vigencia').fill(c.fecha_datos); await escribir(page.locator('#fi1-direccion'), c.direccion); await elegir(page.locator('#fi1-composicion_familiar'), 'EXTENSA'); }, async () => { await elegir(page.locator('#fi1-familia_numerosa'), 'true'); await elegir(page.locator('#fi1-vivienda'), 'ALQUILER'); }, async () => { await guardarCambio(ruta(''), 'PATCH', () => pulsar(boton('gap1.guardar_familia'))); assert.equal((await leer(ruta('/familia'))).familia.direccion, c.direccion); await apuntar(page.locator('#fi1-direccion')); });
    await escena('adultos', async () => {
      const familia = await leer(ruta('/familia')); assert([0, c.adultos.length].includes(familia.adultos.length));
      for (const [i, datos] of c.adultos.entries()) {
        const existente = familia.adultos[i];
        if (!existente) await pulsar(boton('gap1.anadir'));
        const sufijo = existente ? `-${existente.id}` : `-nuevo-${i}`;
        await page.locator(`#fi1-papel${sufijo}`).evaluate(n => { n.closest('details').open = true; });
        await escribir(page.locator(`#fi1-nombre${sufijo}`), datos.nombre);
        await escribir(page.locator(`#fi1-apellidos${sufijo}`), datos.apellidos);
        await elegir(page.locator(`#fi1-papel${sufijo}`), datos.papel);
        await elegir(page.locator(`#fi1-sexo${sufijo}`), datos.sexo);
        if (!existente?.fecha_nacimiento) await page.locator(`#fi1-edad_declarada${sufijo}`).fill(datos.edad);
      }
    }, async () => { await guardarCambio(ruta('/adultos'), 'PUT', () => pulsar(boton('gap1.guardar_adultos'))); adulta = (await leer(ruta('/familia'))).adultos[0]; assert.equal(adulta.papel, 'MADRE'); }, () => apuntar(page.locator(`#fi1-nombre-${adulta.id}`)));
    await escena('menores', async () => { await tab('menores'); await page.locator('#fi1-vigencia').fill(c.fecha_datos); menores = (await leer(ruta('/menores'))).menores; assert.equal(menores.length, c.menores.length); await apuntar(page.locator(`#fi1-fecha_nacimiento-${menores[0].id}`)); await page.locator(`#fi1-fecha_nacimiento-${menores[0].id}`).fill(c.menores[0].fecha_nacimiento); }, async () => { await guardarCambio(ruta('/menores'), 'PUT', () => pulsar(boton('fi1.guardar'))); assert.equal((await leer(ruta('/menores'))).menores[0].fecha_nacimiento, c.menores[0].fecha_nacimiento); }, () => apuntar(page.locator(`#fi1-fecha_nacimiento-${menores[0].id}`)));
    await escena('fase', async () => { await tab('intervencion'); await page.locator('#fi1-vigencia').fill(c.fecha_datos); await elegir(page.locator('#fi1-fase'), 'EVALUACION'); }, async () => { await guardarCambio(ruta('/intervencion'), 'PUT', () => pulsar(boton('fi1.guardar'))); assert.equal((await leer(ruta('/intervencion'))).intervencion.fase, 'EVALUACION'); }, () => apuntar(page.locator('#fi1-fase')));
    await escena('consentimiento', async () => { await tab('consentimiento'); estado.descargas.anexo1 = await descargar(s, ruta('/documentos/anexo1'), () => pulsar(boton('fi2.consentimiento.descargar')), resolve(dir, 'anexo1.docx')); }, async () => { assert(!(await leer(`/api/equipo/expedientes/${expediente.id}`)).hitos.some(h => h.codigo === 'consentimiento')); await apuntar(page.getByText(t('fi2.consentimiento.pendiente'), { exact: true }).first()); }, () => apuntar(page.locator('#fi2-fecha-consentimiento')));
    await calendario(c.fecha_consentimiento);
    await escena('firma', async () => { await apuntar(page.locator('#fi2-fecha-consentimiento')); await page.locator('#fi2-fecha-consentimiento').fill(c.fecha_consentimiento); }, async () => { await guardarCambio(ruta('/hitos'), 'POST', () => pulsar(page.locator('form').filter({ has: page.locator('#fi2-fecha-consentimiento') }).getByRole('button', { name: t('fi2.consentimiento.confirmar'), exact: true }))); estado.consentimiento = (await leer(`/api/equipo/expedientes/${expediente.id}`)).hitos.find(h => h.codigo === 'consentimiento'); assert(estado.consentimiento); }, () => apuntar(page.getByText(t('fi2.confirmado'), { exact: false }).first()));
    await calendario(c.fecha_evaluacion);
    await escena('evaluacion', async () => { await tab('evaluacion'); await elegir(page.locator('#fi2-adulto'), adulta.id); await elegir(page.locator('#fi2-instrumento'), 'SCORE15'); }, async () => { await page.locator('#fi2-fecha-aplicacion').fill(c.fecha_evaluacion); await guardarCambio(ruta('/evaluaciones'), 'POST', () => pulsar(boton('fi2.evaluacion.guardar'))); assert((await leer(ruta('/evaluaciones'))).instrumentos.some(i => i.codigo === 'SCORE15' && i.sujeto?.adulto_id === adulta.id)); }, () => apuntar(page.locator('#equipo-panel-evaluacion .equipo-sucesos li').last()));
    await calendario(c.fecha_comision);
    await escena('comision', async () => { await pulsar(page.locator('a[href="/equipo/comisiones"]').first()); await pulsar(boton('actividad.registrar_comisiones')); await page.locator('#actividad-fecha').fill(c.fecha_comision); }, () => escribir(page.locator('#actividad-observaciones'), c.comision), async () => { comision = await guardarCambio('/api/comisiones', 'POST', () => pulsar(boton('actividad.guardar'))); assert.equal(comision.equipo_id, equipo); assert.equal(comision.observaciones, c.comision); await page.getByRole('table').waitFor(); await apuntar(page.getByRole('table')); });
    await calendario(c.fecha_informe);
    await escena('informe', async () => { await pulsar(page.getByRole('radio', { name: t('actividad.informes'), exact: true })); await pulsar(boton('actividad.registrar_informes')); await page.locator('#actividad-fecha').fill(c.fecha_informe); }, () => elegir(page.locator('#actividad-organismo'), c.organismo), async () => { informe = await guardarCambio('/api/informes', 'POST', () => pulsar(boton('actividad.guardar'))); assert.equal(informe.equipo_id, equipo); await page.getByRole('table').waitFor(); await apuntar(page.getByRole('table')); });
    const mes = c.mes_previo; estado.mes = c.mes;
    await escena('spaf', async () => { await pulsar(page.locator('a[href="/equipo/tabla-spaf"]').first()); await page.locator('#spaf-mes').fill(mes); }, async () => { await page.locator('.spaf-indicadores').waitFor(); const tabla = await leer(`/api/estadistica/spaf?anio=${c.anio}&mes=${Number(mes.slice(5))}&etf=${equipo}`); assert(!tabla.filas.some(f => f.numero === expediente.numero)); estado.spaf_previo = tabla.resumen; await apuntar(page.locator('.spaf-indicadores')); }, () => apuntar(page.locator('.spaf-movimientos')));
    await escena('wordspaf', async () => { estado.descargas.spaf_previo = await descargar(s, `/api/equipos/${equipo}/tabla-spaf/${c.anio}/${Number(mes.slice(5))}.docx`, () => pulsar(page.getByRole('button', { name: new RegExp('^' + t('spaf.descargar_mes').split('{')[0]) })), resolve(dir, 'tabla-spaf-septiembre.docx')); }, () => apuntar(page.getByRole('status').filter({ hasText: t('spaf.descargado') }).or(page.getByRole('status').filter({ hasText: t('spaf.descargado_cambios') }))), () => apuntar(page.locator('.spaf-indicadores')));
    await calendario(c.fecha_cierre);
    await escena('finalizacion', async () => { await abrirFicha(); await tab('cierre'); await elegir(page.locator('#equipo-motivo'), 'OBJ_TOTAL'); }, async () => { for (const [campo, valor] of Object.entries(c.finalizacion)) await escribir(page.locator(`#equipo-cierre-${campo}`), valor); }, async () => { await guardarCambio(ruta('/finalizacion'), 'PATCH', () => pulsar(boton('equipo.cierre.guardar'))); await boton('equipo.cierre.proponer').waitFor(); await apuntar(boton('equipo.cierre.proponer')); });
    await escena('anexo', async () => { estado.descargas.anexo11 = await descargar(s, ruta('/documentos/anexo11'), () => pulsar(boton('gap1.anexo11')), resolve(dir, 'anexo11.docx')); }, async () => { assert.equal((await leer(ruta(''))).estado, 'en_intervencion'); await tab('documentos'); await apuntar(page.getByText(t('equipo.documento.finalizacion.anexo11'), { exact: true })); }, () => apuntar(page.locator('#equipo-tab-cierre')));
    await escena('propuesta', async () => { await tab('cierre'); estado.propuesta = await guardarCambio(ruta('/proponer-cierre'), 'POST', () => pulsar(boton('equipo.cierre.proponer'))); }, async () => { await boton('equipo.cierre.retirar').waitFor(); assert.equal((await leer(ruta(''))).estado, 'en_intervencion'); await apuntar(boton('equipo.cierre.retirar')); }, () => apuntar(page.locator('#equipo-panel-cierre')));
    await escena('revision', async () => { await salir(); await entrar('coordinacion'); await pulsar(page.locator('a[href="/coordinacion/cierres"]').first()); }, async () => { const enlace = page.getByRole('link', { name: expediente.numero, exact: true }); for (let paginas = 0; !await enlace.count(); paginas++) { assert(paginas < 30); assert(await boton('salidas.siguiente').isEnabled()); await pulsar(boton('salidas.siguiente')); await page.waitForTimeout(500); } await pulsar(enlace); await page.locator('#salidas-fecha_cierre').waitFor(); }, () => apuntar(page.getByRole('heading', { name: t('salidas.revisar_cierre'), exact: true })));
    await escena('cierre', async () => { await apuntar(page.locator('#salidas-fecha_cierre')); await page.locator('#salidas-fecha_cierre').fill(c.fecha_cierre); }, async () => { await pulsar(boton('salidas.accion.cerrar')); await apuntar(page.getByRole('dialog')); }, async () => { estado.cierre = await guardarCambio(ruta('/cerrar'), 'POST', () => pulsar(page.getByRole('dialog').getByRole('button', { name: t('salidas.accion.cerrar'), exact: true }))); assert.equal(estado.cierre.estado, 'cerrada'); await page.getByRole('heading', { name: t('salidas.cierre_confirmado'), exact: true }).waitFor(); await apuntar(page.getByRole('heading', { name: t('salidas.cierre_confirmado'), exact: true })); });
    await escena('actacierre', async () => { estado.descargas.acta_cierre = await descargar(s, ruta('/documentos/acta_cierre'), () => pulsar(boton('salidas.accion.acta')), resolve(dir, 'acta-cierre.docx')); }, () => apuntar(page.getByRole('status').filter({ hasText: t('salidas.descargada') })), async () => {
      await calendario(c.fecha_memoria);
      const tabla = await leer(`/api/estadistica/spaf?anio=${c.anio}&mes=${Number(c.mes.slice(5))}&etf=${equipo}`); estado.spaf_despues = tabla.filas.find(f => f.numero === expediente.numero); assert(estado.spaf_despues.alta && estado.spaf_despues.baja);
      await pulsar(page.locator('a[href="/equipo/tabla-spaf"]').first()); await elegir(page.locator('#spaf-equipo'), equipo); await page.locator('#spaf-mes').fill(c.mes); await page.locator('.spaf-movimientos a').filter({ hasText: expediente.numero }).first().waitFor();
      estado.descargas.spaf = await descargar(s, `/api/equipos/${equipo}/tabla-spaf/${c.anio}/${Number(c.mes.slice(5))}.docx`, () => pulsar(page.getByRole('button', { name: new RegExp('^' + t('spaf.descargar_mes').split('{')[0]) })), resolve(dir, 'tabla-spaf.docx'));
      await apuntar(page.locator('.spaf-movimientos'));
    });
    await salir();
    await escena('memoria', async () => { await entrar('profesional'); await abrirMemoria(); }, () => apuntar(page.locator('.memoria-filtros')), () => apuntar(page.locator('aside[aria-labelledby="memoria-avisos"]')));
    await escena('apartados', () => apuntar(page.locator('aside[aria-labelledby="memoria-avisos"]')), async () => { const avisos = page.locator('aside[aria-labelledby="memoria-avisos"] button'); assert(await avisos.count()); await pulsar(avisos.first()); const id = await page.evaluate(() => document.activeElement.id); assert(id.startsWith('memoria-apartado-')); await apuntar(page.locator(`[id="${id}"]`)); }, async () => { estado.memoria = await leer(`/api/estadistica/memoria?${consulta(equipo)}`); assert(estado.memoria.apartados.length > 0); await apuntar(page.locator('.memoria .pildora')); });
    await escena('narrativa', async () => { const actual = await leer(`/api/estadistica/memoria/narrativas?${consulta(equipo)}`); estado.narrativa = c.narrativa.replace('{version}', String(actual.version + 1)); await escribir(page.locator('#memoria-texto-d'), estado.narrativa); }, async () => { const nuevo = await guardarCambio('/api/estadistica/memoria/narrativas', 'PUT', () => pulsar(boton('memoria.guardar'))); assert.equal(nuevo.d, estado.narrativa); }, () => apuntar(page.getByRole('status').filter({ hasText: t('memoria.guardada') })));
    await escena('wordmemoria', async () => { estado.descargas.memoria = await descargar(s, '/api/estadistica/memoria/word', () => pulsar(boton('memoria.descargar')), resolve(dir, 'memoria-equipo.docx')); }, () => apuntar(page.getByRole('status').filter({ hasText: t('memoria.descargada') })), () => apuntar(page.locator('.memoria .pildora')));
    await escena('cerrarequipo', async () => { await pulsar(boton('memoria.cerrar')); await apuntar(page.getByRole('dialog')); }, async () => { estado.memoria_equipo = await cerrarMemoria(equipo); }, () => apuntar(page.locator('.memoria .pildora')));
    await escena('provincial', async () => { await salir(); await entrar('segunda'); await abrirMemoria(); }, async () => { assert.equal(await page.locator('#memoria-equipo').inputValue(), ''); await apuntar(page.locator('#memoria-equipo')); }, () => apuntar(page.locator('aside[aria-labelledby="memoria-avisos"]')));
    await escena('cerrarprovincia', async () => { await pulsar(boton('memoria.cerrar')); await apuntar(page.getByRole('dialog')); }, async () => { estado.memoria_provincial = await cerrarMemoria(''); }, () => apuntar(page.locator('.memoria .pildora')));
    estado.comision = comision; estado.informe = informe; estado.pasos = referencias.length;
    estado.recepcion = (await leer(ruta(''))).lectura_anexo.leida_en;
    assert.equal(estado.recepcion.slice(0, 10), c.fecha_aceptacion);
    assert.equal(estado.consentimiento.fecha.slice(0, 10), c.fecha_consentimiento);
    assert(c.fecha_datos < c.fecha_consentimiento && c.fecha_consentimiento < c.fecha_cierre);

    for (const archivo of ['acta-aceptacion', 'anexo1', 'anexo11', 'tabla-spaf-septiembre', 'tabla-spaf', 'acta-cierre', 'memoria-equipo']) assert.equal(readFileSync(resolve(dir, `${archivo}.docx`)).subarray(0, 2).toString(), 'PK');
    await s.terminar();
  } catch (error) {
    await page.screenshot({ path: resolve(dir, 'fallo.png'), fullPage: true, mask: [page.locator('input[type="password"]')] }).catch(() => {});
    throw error;
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`CICLO-VIDEO-OK: ${estado.pasos} pasos; un expediente; consentimiento, actividad, alta y baja; siete Word; Memorias cerradas`);
}
