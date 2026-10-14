// Conserva cada revisión cerrada y registra el motivo al preparar una repetición.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { sesionVideo } from './sesion-video.mjs';
import { pedir, pulsar, apuntar } from './datos-video.mjs';
import { guardar } from './comun.mjs';

export async function grabarMemoria(opciones) {
  const { guion, dir, referencias } = opciones;
  const c = guion.config;
  const s = await sesionVideo(opciones, ['memoria']);
  const { page, t, boton, escena, leer } = s;
  const estado = { modo: 'demo', recorrido: 'memoria', anio: c.anio, preparacion: [], escrituras: [], pasos: 0 };
  page.on('response', r => {
    if (r.status() === 429) {
      estado.limitacion = { ruta: new URL(r.url()).pathname, espera_s: r.headers()['retry-after'] };
      void r.json().then(d => { estado.limitacion.clave = d.error?.clave ?? d.clave; }).catch(() => {});
    }
  });
  let equipo;
  const entrar = async perfil => {
    assert.equal(process.env[c[`${perfil}_env`]], c[`${perfil}_demo`]);
    await s.entrar(perfil);
    assert.equal((await leer('/api/sesion')).usuario.cuenta, c[`${perfil}_demo`]);
  };
  const consulta = id => new URLSearchParams({ anio: c.anio, ...(id ? { etf: id } : {}) }).toString();
  const situacion = id => leer(`/api/estadistica/memoria/estado?${consulta(id)}`);
  const salir = async () => {
    await pulsar(page.locator('details.persona-conectada > summary'));
    await pulsar(boton('identidad.salir')); await boton('identidad.entrar').waitFor();
  };
  const abrir = async () => {
    await pulsar(page.locator('a[href="/memoria"]').first());
    await boton('memoria.consultar').waitFor({ state: 'visible' });
    await page.locator('.memoria .pildora').waitFor();
    if (await page.locator('#memoria-anio').inputValue() !== String(c.anio)) {
      await page.locator('#memoria-anio').fill(String(c.anio));
      await pulsar(boton('memoria.consultar')); await page.locator('.memoria .pildora').waitFor();
    }
    await boton('memoria.cerrar').waitFor();
  };
  const acto = async (accion, id) => {
    const antes = await situacion(id);
    const ruta = `/api/estadistica/memoria/${accion}`;
    const respuesta = s.respuesta(ruta);
    const registro = { ruta, equipo_id: id, version: antes.version, confirmado: false };
    estado.escrituras.push(registro);
    await pulsar(page.getByRole('dialog').getByRole('button', { name: t(accion === 'cerrar' ? 'memoria.confirmar_cierre' : 'memoria.confirmar_reapertura'), exact: true }));
    const r = await respuesta; assert.equal(r.status(), 200);
    const despues = await r.json(); registro.confirmado = true; registro.resultado = despues;
    assert.equal(despues.estado, accion === 'cerrar' ? 'cerrada' : 'borrador');
    assert(despues.version > antes.version);
    if (accion === 'cerrar') {
      assert.equal(despues.revision_actual, antes.revision_actual + 1);
      const foto = await leer(`/api/estadistica/memoria/instantanea?${consulta(id)}&revision=${despues.revision_actual}`);
      assert.equal(foto.informe.anio, c.anio); assert.equal(foto.informe.equipo_id, id);
      registro.foto = foto;
    } else registro.motivo = c.motivo_reapertura;
    await page.getByRole('status').filter({ hasText: t(accion === 'cerrar' ? 'memoria.cierre_guardado' : 'memoria.reapertura_guardada') }).waitFor();
    await apuntar(page.locator('.memoria .pildora'));
    return despues;
  };
  try {
    // Las dos Memorias pueden seguir cerradas tras una grabación interrumpida.
    await entrar('coordinacion');
    equipo = (await leer('/api/estadistica/memoria/equipos')).find(e => e.codigo === c.equipo_codigo); assert(equipo);
    for (const id of [equipo.id, '']) {
      const antes = await situacion(id);
      if (antes.estado === 'cerrada') {
        const registro = { equipo_id: id, revision_anterior: antes.revision_actual, confirmado: false };
        estado.preparacion.push(registro);
        const nuevo = await pedir(page, `/api/estadistica/memoria/reabrir?${consulta(id)}`, { version: antes.version, motivo: c.motivo_preparacion });
        assert.equal(nuevo.estado, 'borrador'); assert.equal(nuevo.revision_actual, antes.revision_actual); registro.confirmado = true;
      }
    }
    await salir();
    await escena('abrir', [async () => { await entrar('profesional'); assert.equal((await leer('/api/sesion')).equipo_id, equipo.id); await abrir(); }, async () => {
      assert.equal(await page.locator('#memoria-anio').inputValue(), String(c.anio));
      await apuntar(page.locator('.memoria-filtros'));
    }]);
    await escena('avisos', [async () => {
      await apuntar(page.locator('aside[aria-labelledby="memoria-avisos"]'));
      const informe = await leer(`/api/estadistica/memoria?${consulta(equipo.id)}`);
      estado.avisos = informe.apartados.filter(a => Object.values(a.avisos ?? {}).some(n => n > 0) || a.celdas.some(c => c.sin_dato > 0 || c.porcentaje?.aviso)).map(a => a.codigo);
      assert(estado.avisos.length > 0);
    }, async () => {
      await pulsar(page.locator('aside[aria-labelledby="memoria-avisos"] button').first());
      const id = await page.evaluate(() => document.activeElement.id); assert(id.startsWith('memoria-apartado-'));
      estado.apartado_revisado = id; await apuntar(page.locator(`[id="${id}"]`));
    }]);
    await escena('narrativa', [async () => {
      const antes = await leer(`/api/estadistica/memoria/narrativas?${consulta(equipo.id)}`);
      estado.narrativa = c.narrativa.replace('{version}', String(antes.version + 1)); estado.version_narrativa_antes = antes.version;
      const campo = page.locator('#memoria-texto-d');
      await apuntar(campo); await campo.fill(''); await campo.pressSequentially(estado.narrativa, { delay: 20 });
    }, async () => {
      const respuesta = s.respuesta('/api/estadistica/memoria/narrativas', 'PUT');
      const registro = { ruta: '/api/estadistica/memoria/narrativas', confirmado: false }; estado.escrituras.push(registro);
      await pulsar(boton('memoria.guardar')); const r = await respuesta; assert.equal(r.status(), 200);
      const n = await r.json(); assert.equal(n.d, estado.narrativa); assert(n.version > estado.version_narrativa_antes); registro.confirmado = true; registro.version = n.version;
      await apuntar(page.getByRole('status').filter({ hasText: t('memoria.guardada') }));
    }]);
    await escena('word', [async () => {
      const respuesta = s.respuesta('/api/estadistica/memoria/word'); const descarga = page.waitForEvent('download');
      await pulsar(boton('memoria.descargar')); const r = await respuesta; assert([200, 201].includes(r.status()));
      estado.emision_word = r.headers()['x-emision-id']; assert(estado.emision_word);
      const d = await descarga; assert.equal(await d.failure(), null); await d.saveAs(resolve(dir, 'memoria-equipo.docx'));
      assert.equal(readFileSync(resolve(dir, 'memoria-equipo.docx')).subarray(0, 2).toString(), 'PK'); estado.word = true;
      await apuntar(page.getByRole('status').filter({ hasText: t('memoria.descargada') }));
    }]);
    await escena('equipo', [async () => { await pulsar(boton('memoria.cerrar')); await apuntar(page.getByRole('dialog')); }, async () => {
      estado.cierre_equipo = await acto('cerrar', equipo.id); assert(await page.locator('#memoria-texto-d').isDisabled());
    }]);
    // Las consultas de los dos perfiles pueden agotar el presupuesto HTTP por IP.
    // Esta pausa queda entre escenas y no forma parte del montaje.
    await new Promise(resolve => setTimeout(resolve, 61000));
    await escena('provincial', [async () => { await salir(); await entrar('coordinacion'); await abrir(); }, async () => {
      assert.equal(await page.locator('#memoria-equipo').inputValue(), '');
      assert.equal((await leer(`/api/estadistica/memoria?${consulta('')}`)).equipo_id, ''); await apuntar(page.locator('#memoria-equipo'));
    }]);
    await escena('cerrar', [async () => { await pulsar(boton('memoria.cerrar')); await apuntar(page.getByRole('dialog')); }, async () => { estado.cierre_provincial = await acto('cerrar', ''); }]);
    await escena('reabrir', [async () => { await pulsar(boton('memoria.reabrir')); await apuntar(page.getByRole('dialog')); }, async () => {
      const campo = page.locator('#memoria-motivo');
      await apuntar(campo); await campo.fill(''); await campo.pressSequentially(c.motivo_reapertura, { delay: 20 });
    }, async () => {
      estado.reapertura = await acto('reabrir', ''); assert.equal(estado.reapertura.revision_actual, estado.cierre_provincial.revision_actual);
      const foto = await leer(`/api/estadistica/memoria/instantanea?${consulta('')}&revision=${estado.cierre_provincial.revision_actual}`); assert(foto.cerrada_en);
      assert.equal((await situacion(equipo.id)).estado, 'cerrada');
    }]);
    estado.pasos = referencias.length; await s.terminar();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`MEMORIA-VIDEO-OK: ${estado.pasos} pasos; Word=${estado.word}; equipo cerrado; provincial cerrada y reabierta con motivo`);
}
