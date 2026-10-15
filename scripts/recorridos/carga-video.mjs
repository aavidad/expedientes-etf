// Carga sólo casos nuevos y comprueba cada recibo sin repetir escrituras inciertas.
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { sesionVideo } from './sesion-video.mjs';
import { pulsar, apuntar, escribir, elegir } from './datos-video.mjs';
import { guardar } from './comun.mjs';

export function renumerarHoja(csv, equipo, numeros) {
  assert.match(equipo, /^\d{2}$/);
  const sustituciones = new Map([[`${equipo}-91-26`, numeros[0]], [`${equipo}-92-26`, numeros[1]]]);
  return csv.replace(new RegExp(`${equipo}-(?:91|92)-26`, 'g'), numero => sustituciones.get(numero));
}

export async function grabarCarga(opciones) {
  const { guion, dir, referencias } = opciones;
  const c = guion.config;
  const s = await sesionVideo(opciones, ['carga']);
  const { page, t, boton, escena, leer } = s;
  const estado = { modo: 'demo', recorrido: 'carga', pasos: 0, escrituras: [] };
  // Chrome puede descartar del inspector el cuerpo de respuestas a una subida.
  // Conserva una copia real en memoria; la aplicación recibe la respuesta original.
  const instalarRecibos = () => {
    const original = window.fetch.bind(window);
    window.__vid6Carga = {};
    window.fetch = async (...args) => {
      const r = await original(...args);
      const u = new URL(r.url);
      if (u.origin === location.origin && u.pathname.startsWith('/api/carga-2026/hoja')) {
        let datos = null;
        try { datos = await r.clone().json(); } catch { /* El recorrido rechazará un recibo ilegible. */ }
        window.__vid6Carga[u.pathname] = { estado: r.status, datos };
      }
      return r;
    };
  };
  await page.addInitScript(instalarRecibos); await page.evaluate(instalarRecibos);
  const reciboHoja = async r => {
    const ruta = new URL(r.url()).pathname;
    await page.waitForFunction(ruta => !!window.__vid6Carga?.[ruta], ruta);
    const copia = await page.evaluate(ruta => window.__vid6Carga[ruta], ruta);
    assert.equal(copia.estado, r.status()); assert(copia.datos);
    return copia.datos;
  };
  let equipo, municipio, numeros, informe, recibo, csv;
  const fecha = async (id, valor) => { await apuntar(page.locator(id)); await page.locator(id).fill(valor); };
  const esperarOpciones = () => page.locator('#carga-equipo option').nth(1).waitFor({ state: 'attached' });
  const confirmar = async manual => {
    const ruta = `/api/carga-2026/${manual ? '' : 'hoja/'}${informe.lote_id}/confirmar`;
    const respuesta = s.respuesta(ruta); const registro = { ruta, confirmado: false }; estado.escrituras.push(registro);
    const casilla = page.locator('#carga-confirmar-duplicados');
    if (await casilla.count() && !await casilla.isChecked()) { await apuntar(casilla); await casilla.check(); }
    await pulsar(boton(manual ? 'carga.confirmar_expediente' : 'carga.confirmar_lote'));
    const r = await respuesta; assert.equal(r.status(), 200); registro.confirmado = true; registro.clave_idempotencia = r.request().headers()['idempotency-key']; assert(registro.clave_idempotencia);
    const datos = manual ? await r.json() : await reciboHoja(r); assert.equal(datos.expedientes.length, manual ? 1 : 3); assert.equal(datos.marca, 'expediente.cargado_para_revision');
    registro.expedientes = datos.expedientes; await page.locator('#carga-resultado').waitFor(); await apuntar(page.locator('#carga-resultado')); return datos;
  };
  const subir = async valido => {
    await page.evaluate(() => { delete window.__vid6Carga['/api/carga-2026/hoja']; });
    const respuesta = s.respuesta('/api/carga-2026/hoja'); await pulsar(boton('carga.revisar_hoja'));
    const r = await respuesta; assert.equal(r.status(), valido ? 200 : 422); informe = await reciboHoja(r);
    await page.locator('#carga-vista').waitFor();
    if (valido) { assert.equal(informe.casos, 3); assert.equal(informe.filas.length, 3); }
    else { assert(informe.incidencias.some(i => i.fila === 4)); assert.equal(await boton('carga.confirmar_lote').count(), 0); estado.incidencias = informe.incidencias; }
    await apuntar(page.locator('#carga-vista'));
  };
  const elegirHoja = async nombre => { await apuntar(page.locator('#carga-hoja')); await page.locator('#carga-hoja').setInputFiles(resolve(dir, nombre)); };
  try {
    await escena('abrir', [async () => {
      assert.equal(process.env[c.coordinacion_env], c.coordinacion_demo);
      await s.entrar('coordinacion'); assert.equal((await leer('/api/sesion')).usuario.cuenta, c.coordinacion_demo);
      await pulsar(page.locator('a[href="/coordinacion/carga-2026"]').first()); await esperarOpciones();
      const o = await leer('/api/carga-2026/opciones?idioma=es'); equipo = o.equipos.find(e => e.codigo === c.equipo_codigo); assert(equipo);
      municipio = o.municipios.find(m => m.equipo_id === equipo.id && m.centro_ss_id && m.ambito_diputacion); assert(municipio);
      const usados = new Set();
      for (let desplazamiento = 0; ; desplazamiento += 100) {
        const lista = await leer(`/api/equipo/expedientes?limite=100&desplazamiento=${desplazamiento}`);
        for (const f of lista.filas) usados.add(f.numero); if (!lista.hay_mas) break;
      }
      numeros = Array.from({ length: 99 }, (_, i) => `${equipo.codigo}-${String(i + 1).padStart(2, '0')}-26`).filter(n => !usados.has(n)).slice(0, 3);
      assert.equal(numeros.length, 3, 'La demo necesita tres números libres del equipo'); estado.numeros = numeros;
    }, async () => { await pulsar(boton('carga.ir_manual')); await esperarOpciones(); }]);
    await escena('origen', [async () => { await elegir(page.locator('#carga-equipo'), equipo.id); await elegir(page.locator('#carga-estado'), 'aceptada'); }, async () => {
      await elegir(page.locator('#carga-tipo'), 'anexo3'); await fecha('#carga-derivacion', c.fecha_derivacion);
    }, async () => { await escribir(page.locator('#carga-numero'), numeros[0]); await fecha('#carga-aceptacion', c.fecha_aceptacion); }]);
    await escena('familia', [async () => { await elegir(page.locator('#carga-municipio'), municipio.id); await elegir(page.locator('#carga-puesto'), 'TS'); }, async () => {
      const campo = page.locator('#carga-descripcion');
      await apuntar(campo); await campo.fill(''); await campo.pressSequentially(c.descripcion, { delay: 20 });
    }, async () => { await escribir(page.locator('#carga-menor-nombre-0'), c.nombre); await escribir(page.locator('#carga-menor-apellidos-0'), c.apellidos); await elegir(page.locator('#carga-menor-sexo-0'), 'M'); }, async () => {
      await fecha('#carga-menor-fecha_nacimiento-0', c.fecha_nacimiento); await escribir(page.locator('#carga-direccion'), c.direccion);
    }]);
    await escena('manual', [async () => {
      const respuesta = s.respuesta('/api/carga-2026'); await pulsar(boton('carga.revisar_manual')); const r = await respuesta; assert.equal(r.status(), 200); informe = await r.json();
      await page.locator('#carga-vista').waitFor(); await apuntar(page.locator('#carga-vista'));
    }, async () => {
      estado.manual = await confirmar(true);
      const ficha = await leer(`/api/expedientes/${estado.manual.expedientes[0]}`);
      assert.equal(ficha.numero, numeros[0]); assert.equal(ficha.fecha_derivacion, c.fecha_derivacion); assert.equal(ficha.aceptacion.fecha_acta, c.fecha_aceptacion); assert.equal(ficha.cargado_para_revision, true);
    }]);
    await escena('hoja', [async () => {
      await pulsar(page.locator('a[href="/coordinacion/carga-2026"]').first()); await esperarOpciones(); await elegir(page.locator('#carga-equipo'), equipo.id);
    }, async () => {
      const descarga = page.waitForEvent('download'); await pulsar(boton('carga.descargar')); const d = await descarga; assert.equal(await d.failure(), null);
      await d.saveAs(resolve(dir, 'hoja-original.csv'));
      csv = renumerarHoja(readFileSync(resolve(dir, 'hoja-original.csv'), 'utf8'), equipo.codigo, numeros.slice(1));
      assert(csv.includes(numeros[1]) && csv.includes(numeros[2]) && csv.includes(c.fecha_hoja));
      writeFileSync(resolve(dir, 'carga-2026.csv'), csv); writeFileSync(resolve(dir, 'carga-pendiente.csv'), csv.replace(c.fecha_hoja, c.fecha_incorrecta)); estado.hoja_descargada = true;
    }]);
    await escena('incidencia', [() => elegirHoja('carga-pendiente.csv'), () => subir(false), async () => {
      await apuntar(page.locator('.carga-fila').filter({ has: page.locator('.carga-aviso') }).first());
      assert.equal(await boton('carga.confirmar_lote').count(), 0);
    }]);
    await escena('corregida', [async () => { await pulsar(boton('carga.cambiar_hoja')); await elegirHoja('carga-2026.csv'); }, () => subir(true), async () => {
      assert.deepEqual(informe.filas.map(f => f.estado), ['en_intervencion', 'cerrada', 'no_aceptada']); assert.equal(informe.filas[2].numero, ''); estado.filas = informe.filas;
      await apuntar(page.locator('.carga-fila').last());
    }]);
    await escena('confirmar', [async () => { recibo = await confirmar(false); estado.lote = recibo; }, async () => {
      await apuntar(page.getByRole('status').filter({ hasText: t('carga.guardados').replace('{cantidad}', '3') }));
      for (const [i, id] of recibo.expedientes.entries()) {
        const ficha = await leer(`/api/expedientes/${id}`); assert.equal(ficha.cargado_para_revision, true); assert.equal(ficha.estado, estado.filas[i].estado);
        if (i < 2) assert.equal(ficha.numero, numeros[i + 1]);
      }
    }, async () => {
      const sinNumero = estado.filas.findIndex(f => !f.numero); assert(sinNumero >= 0);
      const respuesta = s.respuesta(`/api/expedientes/${recibo.expedientes[sinNumero]}`, 'GET');
      await pulsar(page.getByRole('button', { name: t('carga.revisar_guardado').replace('{cantidad}', String(sinNumero + 1)), exact: true })); assert.equal((await respuesta).status(), 200);
      await page.locator('#carga-consulta').waitFor(); await apuntar(page.locator('section[aria-labelledby="carga-consulta"]'));
    }]);
    estado.pasos = referencias.length; await s.terminar();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`CARGA-VIDEO-OK: ${estado.pasos} pasos; alta manual y lote de tres; incidencias por fila; cuatro casos nuevos`);
}
