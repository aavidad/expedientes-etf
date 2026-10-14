// Coordinación: escenas con acciones reales sincronizadas con cada frase.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { viewport, baseLocal, restringir, temaActivo, fuentesActivas, escenas, pulsar, escribir, elegir, apuntar, guardar } from './comun.mjs';

export async function grabarCoordinacion({ browser, guion, dir, referencias }) {
  if (process.env.ETF_VIDEO_DEMO !== '1') throw new Error('Sólo se graban datos inventados de la demo');
  const base = baseLocal(process.env.ETF_VIDEO_URL);
  const config = guion.config;
  const cuenta = process.env[config.coordinacion_env];
  const clave = process.env[config.clave_coordinacion_env];
  if (!cuenta || !clave) throw new Error('Falta la cuenta inventada de Coordinación');
  const anexo = 'internal/adaptadores/anexos/docx/testdata/anexo3.docx';
  assert.equal(config.anexo, anexo);
  const textos = Object.assign({}, ...['identidad', 'anexo', 'bandeja', 'decision'].map(n =>
    JSON.parse(readFileSync(new URL(`../../idiomas/es/textos/${n}.json`, import.meta.url), 'utf8'))));
  const t = k => { assert(textos[k], `Clave ausente: ${k}`); return textos[k]; };
  const contexto = await browser.newContext({ viewport, locale: 'es-ES', timezoneId: 'Europe/Madrid',
    acceptDownloads: true, recordVideo: { dir, size: viewport }, serviceWorkers: 'block' });
  await restringir(contexto, base);
  const origen = performance.now();
  const page = await contexto.newPage();
  page.setDefaultTimeout(30000);
  const errores = []; page.on('pageerror', e => errores.push(e.message));
  const estado = { modo: 'demo', recorrido: 'coordinacion', pasos: 0, escritura: false, derivacion_id: null, numero: null, descargado: false };
  const boton = k => page.getByRole('button', { name: t(k), exact: true });
  const respuesta = (ruta, metodo = 'POST') => page.waitForResponse(r => new URL(r.url()).pathname === ruta && r.request().method() === metodo);
  let derivacion, expediente;
  try {
    await page.goto(`${base}/entrar`);
    guardar(resolve(dir, 'tema.json'), { tokens: await temaActivo(page), fuentes: await fuentesActivas(page) });
    const escena = await escenas(page, guion, dir, referencias, origen);
    await escena('bandeja', [async () => {
      await escribir(page.getByLabel(t('identidad.cuenta'), { exact: true }), cuenta);
      await escribir(page.getByLabel(t('identidad.contrasena'), { exact: true }), clave, true);
      const entrada = respuesta('/api/sesion');
      await pulsar(boton('identidad.entrar')); assert.equal((await entrada).status(), 200);
      await page.getByRole('heading', { name: t('bandeja.titulo'), exact: true }).waitFor();
    }, async () => {
      await page.getByRole('link', { name: t('bandeja.revisar'), exact: true }).first().waitFor();
      await page.evaluate(() => window.scrollTo({ top: 120, behavior: 'smooth' }));
    }]);
    await escena('subir', [async () => {
      await pulsar(page.getByRole('link', { name: t('bandeja.subir'), exact: true }));
      const campo = page.locator('#anexo-fichero'); await apuntar(campo);
      await campo.setInputFiles(new URL(`../../${anexo}`, import.meta.url).pathname);
    }, async () => {
      const lectura = respuesta('/api/coordinacion/derivaciones');
      await pulsar(boton('anexo.subir'));
      const r = await lectura; assert.equal(r.status(), 200);
      assert.equal((await r.json()).lectura_anexo.antivirus, 'limpio');
      await page.getByRole('heading', { name: t('anexo.revisar'), exact: true }).waitFor();
    }]);
    await escena('revisar', [async () => {
      await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'smooth' }));
      await page.getByText(t('anexo.antivirus_limpio'), { exact: true }).waitFor();
    }, async () => {
      for (const id of ['municipio_id', 'centro_id', 'equipo_id']) assert(await page.locator(`#anexo-${id}`).inputValue());
      await page.locator('#anexo-municipio_id').evaluate(n => n.closest('fieldset').scrollIntoView({ block: 'start', behavior: 'smooth' }));
    }, async () => {
      for (const apartado of ['familia', 'menores']) {
        const bloque = page.locator('details.anexo-tarjeta').filter({ has: page.locator('summary', { hasText: t(`anexo.apartado.${apartado}`) }) }).first();
        await bloque.evaluate(n => { n.open = true; n.scrollIntoView({ block: 'start', behavior: 'smooth' }); });
        await page.waitForTimeout(1100);
      }
    }]);
    await escena('crear', [async () => {
      const creada = respuesta('/api/coordinacion/derivaciones/confirmar'); estado.escritura = true;
      await pulsar(boton('anexo.confirmar'));
      const r = await creada; assert.equal(r.status(), 201);
      derivacion = await r.json(); estado.derivacion_id = derivacion.id;
      assert.equal(derivacion.estado, 'derivacion_recibida'); assert(!derivacion.numero);
      await page.locator('#decision-estado').waitFor();
    }, () => page.evaluate(() => window.scrollTo({ top: 0, behavior: 'smooth' }))]);
    await escena('aceptar', [async () => {
      await elegir(page.locator('#decision-profesional_referencia_puesto'), config.puesto);
      const fecha = new Intl.DateTimeFormat('sv-SE', { timeZone: 'Europe/Madrid' }).format(new Date());
      const campo = page.locator('#decision-fecha_acta'); await apuntar(campo); await campo.fill(fecha);
    }, async () => {
      const radios = await page.locator('input[type="radio"][value="M"]').all(); assert(radios.length > 0);
      for (const radio of radios) await pulsar(radio);
    }, async () => {
      await pulsar(boton('decision.aceptar'));
      const dialogo = page.getByRole('dialog'); await dialogo.waitFor();
      const aceptada = respuesta(`/api/coordinacion/derivaciones/${derivacion.id}/aceptar`);
      await pulsar(dialogo.getByRole('button', { name: t('decision.confirmar.aceptar'), exact: true }));
      const r = await aceptada; assert.equal(r.status(), 200);
      expediente = await r.json(); estado.numero = expediente.numero;
      assert.match(expediente.numero, /^\d{2}-\d{2}-\d{2}$/); assert.equal(expediente.estado, 'aceptada');
    }, async () => {
      await page.locator('#decision-estado').filter({ hasText: expediente.numero }).waitFor();
      await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'smooth' }));
    }]);
    await escena('descargar', [async () => {
      const descarga = page.waitForEvent('download'); const emision = respuesta('/api/documentos/emisiones');
      await pulsar(page.getByRole('button', { name: `${t('decision.descargar')} ${t('decision.acta_aceptacion')}`, exact: true }));
      assert.equal((await emision).status(), 201);
      const fichero = await descarga; assert.equal(await fichero.failure(), null); assert.match(fichero.suggestedFilename(), /\.docx$/);
      await fichero.saveAs(resolve(dir, 'acta-aceptacion.docx'));
      const bytes = readFileSync(resolve(dir, 'acta-aceptacion.docx'));
      assert(bytes.length > 1000); assert.equal(bytes.subarray(0, 2).toString(), 'PK'); estado.descargado = true;
    }, async () => {
      const aviso = page.getByRole('status').filter({ hasText: t('decision.descargado') }); await aviso.waitFor();
      await aviso.evaluate(n => n.scrollIntoView({ block: 'center', behavior: 'smooth' }));
    }]);
    await escena('recientes', [async () => {
      await pulsar(page.locator('a[href="/coordinacion"]').first());
      await page.getByRole('heading', { name: t('bandeja.titulo'), exact: true }).waitFor();
      const lista = respuesta('/api/coordinacion/bandeja/buscar');
      await pulsar(page.getByRole('button', { name: new RegExp(t('bandeja.estado.aceptada')) })); assert.equal((await lista).status(), 200);
    }, async () => {
      await escribir(page.getByLabel(t('bandeja.buscar_etiqueta'), { exact: true }), expediente.numero);
      const busqueda = respuesta('/api/coordinacion/bandeja/buscar'); await pulsar(boton('bandeja.buscar'));
      const r = await busqueda; assert.equal(r.status(), 200);
      assert((await r.json()).filas.some(f => f.id === derivacion.id && f.numero === expediente.numero && f.estado === 'aceptada'));
      const fila = page.getByRole('row').filter({ hasText: expediente.numero }); await fila.waitFor();
      await fila.evaluate(n => n.scrollIntoView({ block: 'center', behavior: 'smooth' }));
    }]);
    assert.deepEqual(errores, []); estado.pasos = referencias.length;
    const video = page.video(); await contexto.close(); await video.saveAs(resolve(dir, 'original.webm')); await video.delete();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await contexto.close(); }
  console.log(`COORDINACION-OK: ${estado.pasos} pasos; número=${estado.numero}; Acta descargada=${estado.descargado}`);
}
