// Marta propone, Inés revisa y Javier confirma un expediente nuevo.
import assert from 'node:assert/strict';
import { resolve } from 'node:path';
import { writeFileSync } from 'node:fs';
import { sesionVideo } from './sesion-video.mjs';
import { crearExpediente, pedir, hoy, descargar, pulsar, apuntar, elegir, escribir } from './datos-video.mjs';
import { guardar } from './comun.mjs';

export async function grabarCierre(opciones) {
  const { guion, dir, referencias } = opciones;
  const s = await sesionVideo(opciones, ['equipo', 'salidas']);
  const { page, t, escena, boton, respuesta } = s;
  const estado = { modo: 'demo', recorrido: 'cierre', pasos: 0, cerrado: false, descargado: false };
  const salir = async () => { await pulsar(page.locator('details.persona-conectada > summary')); await pulsar(boton('identidad.salir')); await boton('identidad.entrar').waitFor(); };
  try {
    await s.entrar('segunda'); const expediente = await crearExpediente(page, guion.config); estado.expediente = expediente;
    await salir();
    await escena('ficha', [async () => { await s.entrar('profesional'); await page.goto(`${process.env.ETF_VIDEO_URL}/equipo/expediente?id=${expediente.id}`); await page.getByRole('tab').first().waitFor(); }, async () => { await pulsar(page.getByRole('tab', { name: t('equipo.pestana.cierre'), exact: true })); await page.locator('#equipo-motivo').waitFor(); }]);
    await escena('motivo', [async () => { await elegir(page.locator('#equipo-motivo'), 'SIN_CONSENTIMIENTO'); await pulsar(page.getByLabel(t('equipo.cierre.sin_consentimiento'), { exact: true })); }, ...Object.entries(guion.config.finalizacion).map(([campo, valor]) => async () => { await escribir(page.locator(`#equipo-cierre-${campo}`), valor); }), async () => {
      const r = respuesta(`/api/expedientes/${expediente.id}/finalizacion`, 'PATCH');
      const leida = respuesta(`/api/expedientes/${expediente.id}/finalizacion`, 'GET');
      const fichaLeida = respuesta(`/api/equipo/expedientes/${expediente.id}`, 'GET');
      await pulsar(boton('equipo.cierre.guardar')); assert.equal((await r).status(), 200); await leida; await fichaLeida;
      await boton('equipo.cierre.proponer').waitFor({ state: 'visible' }); await apuntar(boton('equipo.cierre.proponer'));
    }]);
    await escena('anexo', [async () => {
      // La ficha actual permite consultar emisiones, pero aún no tiene botón para generar Anexos 11/12.
      const ficha = await pedir(page, `/api/expedientes/${expediente.id}`);
      const emision = await page.evaluate(async ({ id, version }) => {
        const sesion = await (await fetch('/api/sesion')).json();
        const r = await fetch(`/api/expedientes/${id}/documentos/anexo11`, { method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': crypto.randomUUID() },
          body: JSON.stringify({ version, idioma: 'es' }) });
        return { estado: r.status, id: r.headers.get('x-emision-id'), bytes: Array.from(new Uint8Array(await r.arrayBuffer())) };
      }, { id: expediente.id, version: ficha.version });
      assert([200, 201].includes(emision.estado), `Anexo 11: ${emision.estado}`);
      estado.anexo_emision = emision.id; assert(estado.anexo_emision);
      const bytes = Buffer.from(emision.bytes); assert.equal(bytes.subarray(0, 2).toString(), 'PK'); writeFileSync(resolve(dir, 'anexo11.docx'), bytes);
      await page.reload(); await pulsar(page.getByRole('tab', { name: t('equipo.pestana.documentos'), exact: true }));
      await apuntar(page.getByText(t('equipo.documento.finalizacion.anexo11'), { exact: true }));
    }]);
    await escena('proponer', [async () => {
      await pulsar(page.getByRole('tab', { name: t('equipo.pestana.cierre'), exact: true }));
      const r = respuesta(`/api/expedientes/${expediente.id}/proponer-cierre`); await pulsar(boton('equipo.cierre.proponer')); assert.equal((await r).status(), 200);
      await boton('equipo.cierre.retirar').waitFor(); await apuntar(boton('equipo.cierre.retirar')); estado.propuesta = await (await r).json();
    }]);
    await escena('ines', [async () => { await salir(); await s.entrar('coordinacion'); await pulsar(page.locator('a[href="/coordinacion/cierres"]').first()); await page.getByRole('heading', { name: t('salidas.cierres'), exact: true }).waitFor(); }, async () => {
      const enlace = page.getByRole('link', { name: expediente.numero, exact: true });
      while (!await enlace.count()) { assert(await boton('salidas.siguiente').isEnabled()); await pulsar(boton('salidas.siguiente')); await page.waitForTimeout(500); }
      await apuntar(enlace);
      for (let offset = 0; ; offset += 20) {
        const propuestas = await s.leer(`/api/coordinacion/cierres?limite=20&desplazamiento=${offset}`);
        const propia = propuestas?.find(p => p.expediente_id === expediente.id);
        if (propia) { estado.propuesta_autor = propia.propuesta_por; break; }
        assert(propuestas?.length === 20, 'Falta la propuesta propia');
      }
      estado.revisada_por_ines = true;
    }]);
    await escena('javier', [async () => { await salir(); await s.entrar('segunda'); await pulsar(page.locator('a[href="/coordinacion/cierres"]').first()); }, async () => {
      const enlace = page.getByRole('link', { name: expediente.numero, exact: true }); await page.waitForTimeout(500);
      while (!await enlace.count()) { assert(await boton('salidas.siguiente').isEnabled()); await pulsar(boton('salidas.siguiente')); await page.waitForTimeout(500); }
      await pulsar(enlace); await page.locator('#salidas-fecha_cierre').waitFor();
      const sesion = await s.leer('/api/sesion'); estado.confirmada_por = sesion.usuario.id;
      assert(estado.propuesta_autor && estado.propuesta_autor !== estado.confirmada_por);
      await apuntar(page.getByRole('heading', { name: t('salidas.revisar_cierre'), exact: true }));
    }]);
    await escena('cerrar', [async () => { await apuntar(page.locator('#salidas-fecha_cierre')); await page.locator('#salidas-fecha_cierre').fill(hoy()); }, async () => { await pulsar(boton('salidas.accion.cerrar')); await page.getByRole('dialog').waitFor(); }, async () => {
      const r = respuesta(`/api/expedientes/${expediente.id}/cerrar`);
      await pulsar(page.getByRole('dialog').getByRole('button', { name: t('salidas.accion.cerrar'), exact: true }));
      assert.equal((await r).status(), 200); estado.cierre = await (await r).json(); assert.equal(estado.cierre.estado, 'cerrada'); estado.cerrado = true;
      await page.getByRole('heading', { name: t('salidas.cierre_confirmado'), exact: true }).waitFor();
      await apuntar(page.getByRole('heading', { name: t('salidas.cierre_confirmado'), exact: true }));
    }]);
    await escena('acta', [async () => {
      estado.acta_emision = await descargar(s, `/api/expedientes/${expediente.id}/documentos/acta_cierre`, () => pulsar(boton('salidas.accion.acta')), resolve(dir, 'acta-cierre.docx'));
      estado.descargado = true; await apuntar(page.getByRole('status').filter({ hasText: t('salidas.descargada') }));
    }]);
    estado.pasos = referencias.length; await s.terminar();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`CIERRE-VIDEO-OK: ${estado.pasos} pasos; expediente nuevo; cerrado=${estado.cerrado}; Acta=${estado.descargado}`);
}
