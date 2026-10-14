// Traslado de Reunificación con aceptación en destino y seguimiento real en origen.
import assert from 'node:assert/strict';
import { resolve } from 'node:path';
import { sesionVideo } from './sesion-video.mjs';
import { crearExpediente, pedir, hoy, pulsar, apuntar, elegir, escribir } from './datos-video.mjs';
import { guardar } from './comun.mjs';

export async function grabarTraslado(opciones) {
  const { guion, dir, referencias } = opciones;
  const s = await sesionVideo(opciones, ['equipo', 'traslado', 'decision']);
  const { page, t, escena, boton, respuesta } = s;
  const estado = { modo: 'demo', recorrido: 'traslado' };
  const salir = async () => { await pulsar(page.locator('details.persona-conectada > summary')); await pulsar(boton('identidad.salir')); await boton('identidad.entrar').waitFor(); };
  const confirmar = async (clave, ruta) => {
    await pulsar(boton(clave)); const dialogo = page.getByRole('dialog'); await dialogo.waitFor();
    const r = respuesta(ruta); await pulsar(dialogo.getByRole('button', { name: t(clave), exact: true })); const recibo = await r;
    assert.equal(recibo.status(), 200); return recibo.json();
  };
  try {
    await s.entrar('coordinacion'); const expediente = await crearExpediente(page, guion.config); estado.origen = expediente;
    const ruta = `/api/expedientes/${expediente.id}`;
    const catalogo = await s.leer('/api/coordinacion/decision/opciones?idioma=es');
    const destino = catalogo.equipos.find(e => e.clave_texto === `equipo.${guion.config.destino_codigo}`); assert(destino);
    const municipio = catalogo.municipios.find(m => m.equipo_id === destino.id && m.centro_ss_id); assert(municipio);
    await salir(); await s.entrar('profesional');
    const hito = await pedir(page, `${ruta}/hitos`, { version: expediente.version, tipo: 'consentimiento', fecha: hoy() });
    const fin = await pedir(page, `${ruta}/finalizacion`, { version: hito.version, motivo: 'TRASLADO' }, 'PATCH');
    await pedir(page, `${ruta}/proponer-cierre`, { version: fin.version });
    await salir(); await s.entrar('coordinacion');
    await escena('abrir', [async () => { await page.goto(`${process.env.ETF_VIDEO_URL}/coordinacion/expedientes/${expediente.id}/salidas`); await pulsar(page.getByRole('link', { name: t('traslado.titulo'), exact: true })); await apuntar(page.getByRole('heading', { name: t('traslado.preparar'), exact: true })); }]);
    await escena('destino', [async () => { await elegir(page.locator('#traslado-destino'), destino.id); }, async () => { await escribir(page.locator('#traslado-motivo_traslado'), guion.config.motivo_traslado); await apuntar(page.locator('#traslado-fecha_traslado')); await page.locator('#traslado-fecha_traslado').fill(hoy()); }]);
    await escena('trasladar', [async () => { await pulsar(boton('traslado.trasladar')); await page.getByRole('dialog').waitFor(); }, async () => {
      const r = respuesta(`${ruta}/trasladar`); await pulsar(page.getByRole('dialog').getByRole('button', { name: t('traslado.trasladar'), exact: true })); assert.equal((await r).status(), 200); estado.traslado = await (await r).json();
      await apuntar(page.getByRole('heading', { name: t('traslado.hecho'), exact: true }));
      assert.equal((await s.leer(ruta)).estado, 'cerrada');
    }]);
    const aceptacionOrigen = (await s.leer(ruta)).aceptacion;
    await escena('aceptacion', [async () => {
      await pulsar(page.getByRole('link', { name: t('traslado.aceptar_destino'), exact: true })); await page.locator('#decision-fecha_acta').waitFor();
    }, async () => {
      await elegir(page.locator('#decision-municipio_id'), municipio.id); await elegir(page.locator('#decision-subprograma'), guion.config.subprograma);
    }, async () => {
      await apuntar(page.locator('#decision-fecha_acta')); await page.locator('#decision-fecha_acta').fill(hoy()); await elegir(page.locator('#decision-profesional_referencia_puesto'), 'TS');
    }, async () => {
      for (const menor of aceptacionOrigen.menores) await pulsar(page.locator(`input[name="sexo-${menor.orden}"][value="${menor.sexo}"]`));
    }, async () => {
      await escribir(page.locator('#decision-descripcion_situacion'), guion.config.descripcion_destino);
    }]);
    await escena('aceptar', [async () => { await pulsar(boton('decision.aceptar')); await page.getByRole('dialog').waitFor(); }, async () => {
      const r = respuesta(`/api/coordinacion/derivaciones/${estado.traslado.destino_id}/aceptar`); await pulsar(page.getByRole('dialog').getByRole('button', { name: t('decision.confirmar.aceptar'), exact: true })); assert.equal((await r).status(), 200);
      estado.destino = await (await r).json(); assert(estado.destino.numero && estado.destino.numero !== expediente.numero); assert.equal(estado.destino.equipo_id, destino.id);
      await apuntar(page.locator('#decision-estado').filter({ hasText: estado.destino.numero }));
    }]);
    await escena('equipo', [async () => { await salir(); await s.entrar('destino'); await pulsar(page.locator('a[href="/equipo"]').first()); await page.locator('#equipo-busqueda').fill(estado.destino.numero); await pulsar(boton('equipo.buscar_accion')); }, async () => {
      await pulsar(page.getByRole('link', { name: estado.destino.numero, exact: true })); await page.getByRole('tab').first().waitFor(); await apuntar(page.getByText(estado.destino.numero, { exact: true }).first()); estado.visto_destino = true;
    }]);
    await escena('origen', [async () => { await salir(); await s.entrar('profesional'); await page.goto(`${process.env.ETF_VIDEO_URL}/equipo/seguimiento?id=${expediente.id}`); await page.locator('#traslado-fecha_cese').waitFor(); }, async () => {
      await apuntar(page.locator('#traslado-fecha_cese')); await page.locator('#traslado-fecha_cese').fill(hoy()); estado.seguimiento = await confirmar('traslado.accion.iniciar', `${ruta}/seguimiento`); assert.equal(estado.seguimiento.estado, 'activo');
    }, async () => { await apuntar(page.getByText(t('traslado.seguimiento_activo'), { exact: true })); await page.getByText(t('traslado.fin_previsto'), { exact: true }).waitFor(); estado.visto_origen = true; }]);
    estado.pasos = referencias.length; await s.terminar();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`TRASLADO-VIDEO-OK: ${estado.pasos} pasos; número nuevo visto en destino; seguimiento activo en origen`);
}
