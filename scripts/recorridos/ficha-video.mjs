// Recorrido de ficha sobre un expediente nuevo; comprueba cada escritura real.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { sesionVideo } from './sesion-video.mjs';
import { crearExpediente, pedir, hoy, descargar, pulsar, apuntar, elegir } from './datos-video.mjs';
import { guardar } from './comun.mjs';

export async function grabarFicha(opciones) {
  const { guion, dir, referencias } = opciones;
  const s = await sesionVideo(opciones, ['equipo', 'fi1', 'fi2']);
  const { page, t, escena, boton, respuesta } = s;
  const estado = { modo: 'demo', recorrido: 'ficha' };
  const salir = async () => { await pulsar(page.locator('details.persona-conectada > summary')); await pulsar(boton('identidad.salir')); await boton('identidad.entrar').waitFor(); };
  const tab = id => pulsar(page.locator(`#equipo-tab-${id}`));
  try {
    await s.entrar('coordinacion');
    const expediente = await crearExpediente(page, guion.config); estado.expediente = expediente;
    await salir(); await s.entrar('profesional');
    const ruta = `/api/expedientes/${expediente.id}`;
    const ficha = await s.leer(`${ruta}/menores`);
    const personas = JSON.parse(readFileSync(new URL('../../despliegue/demo/menores.json', import.meta.url)));
    const menores = ficha.menores.map((m, i) => ({ ...m, ...personas[i % personas.length], fecha_nacimiento: i ? personas[i % personas.length].fecha_nacimiento : null, propuesto_atencion_temprana: false }));
    await pedir(page, `${ruta}/menores`, { version: ficha.version, vigente_desde: hoy(), menores }, 'PUT');
    const familia = await s.leer(`${ruta}/familia`);
    if (!familia.adultos.length) await pedir(page, `${ruta}/adultos`, { version: familia.version, vigente_desde: hoy(), adultos: [guion.config.adulto] }, 'PUT');
    const menor = menores[0];
    const aviso = a => a.campo === 'fecha_nacimiento' && a.sujeto_id === menor.id;
    assert((await s.leer(`${ruta}/avisos`)).avisos.some(aviso));
    await escena('bandeja', [async () => { await pulsar(page.locator('a[href="/equipo"]').first()); await page.locator('#equipo-busqueda').fill(expediente.numero); await pulsar(boton('equipo.buscar_accion')); }]);
    await escena('ficha', [async () => { await pulsar(page.getByRole('link', { name: expediente.numero, exact: true })); await page.getByRole('tab').first().waitFor(); }, async () => { await pulsar(page.locator('#fi1-ficha-avisos')); await apuntar(page.locator(`a[href="#fi1-fecha_nacimiento-${menor.id}"]`)); }]);
    await escena('menores', [async () => { await tab('menores'); await apuntar(page.locator(`#fi1-fecha_nacimiento-${menor.id}`)); await page.locator(`#fi1-fecha_nacimiento-${menor.id}`).fill(personas[0].fecha_nacimiento); }, async () => {
      const r = respuesta(`${ruta}/menores`, 'PUT'); await pulsar(boton('fi1.guardar')); assert.equal((await r).status(), 200);
      await page.getByText(t('fi1.guardado'), { exact: true }).waitFor();
      assert(!(await s.leer(`${ruta}/avisos`)).avisos.some(aviso));
      assert.equal(await page.locator(`#fi1-fecha_nacimiento-${menor.id}-aviso`).count(), 0);
      estado.aviso_desaparecido = true; await apuntar(page.locator(`#fi1-fecha_nacimiento-${menor.id}`));
    }]);
    await escena('fase', [async () => { await tab('intervencion'); await elegir(page.locator('#fi1-fase'), 'EVALUACION'); }, async () => {
      const r = respuesta(`${ruta}/intervencion`, 'PUT'); await pulsar(boton('fi1.guardar')); assert.equal((await r).status(), 200);
      assert.equal((await s.leer(`${ruta}/intervencion`)).intervencion.fase, 'EVALUACION'); await apuntar(page.locator('#fi1-fase'));
    }]);
    await escena('anexo', [async () => { await tab('consentimiento'); estado.anexo_emision = await descargar(s, `${ruta}/documentos/anexo1`, () => pulsar(boton('fi2.consentimiento.descargar')), resolve(dir, 'anexo1.docx')); }, async () => { await apuntar(page.getByText(t('fi2.consentimiento.pendiente'), { exact: true }).first()); assert(!(await s.leer(`/api/equipo/expedientes/${expediente.id}`)).hitos.some(h => h.codigo === 'consentimiento')); }]);
    await escena('firma', [async () => { await apuntar(page.locator('#fi2-fecha-consentimiento')); await page.locator('#fi2-fecha-consentimiento').fill(hoy()); }, async () => {
      const r = respuesta(`${ruta}/hitos`); await pulsar(page.locator('form').filter({ has: page.locator('#fi2-fecha-consentimiento') }).getByRole('button', { name: t('fi2.consentimiento.confirmar'), exact: true })); assert.equal((await r).status(), 200);
      await apuntar(page.getByText(t('fi2.confirmado'), { exact: false }).first()); estado.firma = (await s.leer(`/api/equipo/expedientes/${expediente.id}`)).hitos.find(h => h.codigo === 'consentimiento'); assert(estado.firma);
    }]);
    await escena('evaluacion', [async () => { await tab('evaluacion'); await elegir(page.locator('#fi2-adulto'), (await s.leer(`${ruta}/familia`)).adultos[0].id); await elegir(page.locator('#fi2-instrumento'), 'SCORE15'); }, async () => {
      await apuntar(page.locator('#fi2-fecha-aplicacion')); await page.locator('#fi2-fecha-aplicacion').fill(hoy()); const r = respuesta(`${ruta}/evaluaciones`); await pulsar(boton('fi2.evaluacion.guardar')); assert.equal((await r).status(), 200);
    }, async () => { estado.evaluacion = (await s.leer(`${ruta}/evaluaciones`)).instrumentos.find(i => i.codigo === 'SCORE15' && i.momento === 'inicial'); assert(estado.evaluacion); await apuntar(page.locator('#equipo-panel-evaluacion .equipo-sucesos li').last()); }]);
    estado.pasos = referencias.length; await s.terminar();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`FICHA-VIDEO-OK: ${estado.pasos} pasos; aviso desaparecido; Anexo 1, firma e instrumento confirmados`);
}
