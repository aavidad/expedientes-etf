// Sesión local compartida por los recorridos de actividad y plantillas.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { viewport, baseLocal, restringir, temaActivo, fuentesActivas, escenas, pulsar, escribir, guardar } from './comun.mjs';

export async function sesionVideo({ browser, guion, dir, referencias }, catalogos) {
  assert.equal(process.env.ETF_VIDEO_DEMO, '1', 'Sólo se graban datos inventados de la demo');
  const base = baseLocal(process.env.ETF_VIDEO_URL);
  const textos = Object.assign({}, ...['identidad', ...catalogos].map(n =>
    JSON.parse(readFileSync(new URL(`../../idiomas/es/textos/${n}.json`, import.meta.url), 'utf8'))));
  const t = k => { assert(textos[k], `Clave ausente: ${k}`); return textos[k]; };
  const contexto = await browser.newContext({ viewport, locale: 'es-ES', timezoneId: 'Europe/Madrid',
    recordVideo: { dir, size: viewport }, serviceWorkers: 'block' });
  await restringir(contexto, base);
  const origen = performance.now();
  const page = await contexto.newPage();
  page.setDefaultTimeout(30000);
  const errores = []; page.on('pageerror', e => errores.push(e.message));
  await page.goto(`${base}/entrar`);
  guardar(resolve(dir, 'tema.json'), { tokens: await temaActivo(page), fuentes: await fuentesActivas(page) });
  const escena = await escenas(page, guion, dir, referencias, origen);
  const boton = k => page.getByRole('button', { name: t(k), exact: true });
  const respuesta = (ruta, metodo = 'POST') => page.waitForResponse(r => new URL(r.url()).pathname === ruta && r.request().method() === metodo);
  const entrar = async perfil => {
    const cuenta = process.env[guion.config[`${perfil}_env`]];
    const clave = process.env[guion.config[`clave_${perfil}_env`]];
    assert(cuenta && clave, `Falta la cuenta inventada de ${perfil}`);
    await escribir(page.getByLabel(t('identidad.cuenta'), { exact: true }), cuenta);
    await escribir(page.getByLabel(t('identidad.contrasena'), { exact: true }), clave, true);
    const entrada = respuesta('/api/sesion'); await pulsar(boton('identidad.entrar'));
    assert.equal((await entrada).status(), 200);
    await boton('identidad.entrar').waitFor({ state: 'detached' });
  };
  const leer = ruta => page.evaluate(async ruta => {
    const r = await fetch(ruta); if (!r.ok) throw new Error(`Consulta local: HTTP ${r.status}`);
    return r.json();
  }, ruta);
  const terminar = async () => {
    assert.deepEqual(errores, []);
    const video = page.video(); await contexto.close();
    await video.saveAs(resolve(dir, 'original.webm')); await video.delete();
  };
  return { page, contexto, t, boton, respuesta, entrar, leer, escena, terminar };
}
