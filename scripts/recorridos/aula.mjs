// Chrome del aula: ventanas con sesiones separadas y calendario de formación.
import assert from 'node:assert/strict';
import { readFileSync, mkdirSync } from 'node:fs';
import { resolve, basename } from 'node:path';
import { chromium, restringir, baseLocal } from './comun.mjs';

assert.equal(process.env.ETF_VIDEO_DEMO, '1');
const local = new URL('../../despliegue/demo/.local/', import.meta.url);
const base = baseLocal(`http://127.0.0.1:${readFileSync(new URL('puerto', local), 'utf8').trim()}`);
const comprobar = process.argv.includes('--comprobar');
assert(process.argv.slice(2).every(v => v === '--comprobar'));
const fecha = () => {
  const valor = readFileSync(new URL('fecha-formacion', local), 'utf8');
  assert(/^\d{4}-\d{2}-\d{2}T10:00:00Z$/.test(valor));
  assert(Number.isFinite(Date.parse(valor)));
  return valor;
};
let actual = fecha();
const carpeta = new URL('descargas-formacion/', local).pathname;
mkdirSync(carpeta, { recursive: true, mode: 0o700 });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: comprobar && process.env.ETF_AULA_VISIBLE !== '1' });
const contextos = [];
const descargas = new Set();
let falloDescarga = false;
let cerrado = false;
navegador.on('disconnected', () => { cerrado = true; });
process.on('SIGINT', () => { cerrado = true; });
process.on('SIGTERM', () => { cerrado = true; });
try {
  for (let n = 0; n < 3; n++) {
    const contexto = await navegador.newContext({ timezoneId: 'Europe/Madrid' });
    contextos.push(contexto);
    await restringir(contexto, base);
    contexto.on('page', pagina => pagina.on('download', descarga => {
      const nombre = `${n + 1}-${Date.now()}-${basename(descarga.suggestedFilename())}`;
      const pendiente = descarga.saveAs(resolve(carpeta, nombre)).catch(() => { falloDescarga = true; cerrado = true; }).finally(() => descargas.delete(pendiente));
      descargas.add(pendiente);
    }));
    const pagina = await contexto.newPage();
    if (n < 2) await pagina.clock.setFixedTime(new Date(actual));
    await pagina.goto(`${base}/entrar`);
  }
  if (comprobar) {
    const pagina = contextos[0].pages()[0];
    assert.equal(await pagina.evaluate(() => new Date().toISOString()), actual.replace('Z', '.000Z'));
    const claves = JSON.parse(readFileSync(new URL('usuarios.json', local), 'utf8'));
    const estado = await pagina.evaluate(async clave => {
      const sesion = await (await fetch('/api/sesion')).json();
      const r = await fetch('/api/sesion', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf }, body: JSON.stringify({ cuenta: 'marta.cabrera', contrasena: clave }) });
      return { codigo: r.status, datos: await r.json() };
    }, claves['marta.cabrera']);
    assert.equal(estado.codigo, 200); assert.equal(estado.datos.estado, 'activo');
    await pagina.goto(`${base}/equipo/tabla-spaf`);
    const selector = pagina.locator('#spaf-mes'); await selector.waitFor();
    await pagina.evaluate(() => new Promise(r => setTimeout(r, 25)));
    assert(Math.abs(await contextos[2].pages()[0].evaluate(() => Date.now()) - Date.now()) < 5000);
    for (const contexto of contextos.slice(1)) {
      const sesion = await contexto.pages()[0].evaluate(async () => (await (await fetch('/api/sesion')).json()));
      assert(!sesion.usuario && !sesion.perfiles?.length);
    }
    const dia = new Date(actual); dia.setUTCMonth(dia.getUTCMonth() - 1);
    assert.equal(await selector.getAttribute('max'), dia.toISOString().slice(0, 7));
    const tabla = await pagina.evaluate(async ({anio, mes, equipo}) => (await fetch(`/api/estadistica/spaf?anio=${anio}&mes=${mes}&etf=${equipo}`)).status, {anio: dia.getUTCFullYear(), mes: dia.getUTCMonth() + 1, equipo: estado.datos.equipo_id});
    assert.equal(tabla, 200);
    console.log(`AULA-OK: Chrome del sistema; fecha ${actual.slice(0, 10)}; mes permitido ${await selector.getAttribute('max')}; tres sesiones separadas; Administración en tiempo real; temporizadores reales`);
  } else {
    console.log(`AULA: dos ventanas con calendario de formación y una para Administración en tiempo real; descargas en ${carpeta}`);
    while (!cerrado && contextos.some(c => c.pages().length)) {
      await new Promise(r => setTimeout(r, 500));
      if (cerrado) break;
      const nueva = fecha();
      if (nueva !== actual) {
        actual = nueva;
        for (const contexto of contextos.slice(0, 2)) for (const pagina of contexto.pages()) {
          await pagina.clock.setFixedTime(new Date(actual));
          await pagina.reload();
        }
        console.log(`AULA-FECHA: ${actual.slice(0, 10)}`);
      }
    }
  }
} finally { await Promise.allSettled([...descargas]); await navegador.close(); }
if (falloDescarga) throw new Error('AULA-DESCARGA-ERROR: comprueba la carpeta antes de cerrar las ventanas');
