// Utilidades de grabación. Playwright y Chrome del sistema; textos en el guion.
import { createRequire } from 'node:module';
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
const require = createRequire(new URL('../../web/package.json', import.meta.url));
export const { chromium } = require('playwright-core');
export const espera = ms => new Promise(r => setTimeout(r, ms));
export const viewport = { width: 1440, height: 900 };
export const chrome = () => chromium.launch({ executablePath: process.env.ETF_VIDEO_CHROME || '/usr/bin/google-chrome', headless: true });

export function baseLocal(valor) {
  const url = new URL(valor);
  if (!['http:', 'https:'].includes(url.protocol) || !['127.0.0.1', 'localhost', '[::1]'].includes(url.hostname)
      || url.username || url.password || url.search || url.hash || url.pathname !== '/') throw new Error('URL de demo local no válida');
  return url.origin;
}

// Impide que el navegador envíe datos a servidores ajenos al entorno local.
export async function restringir(contexto, base) {
  await contexto.route('**/*', route => {
    const u = new URL(route.request().url());
    return u.origin === base || ['data:', 'about:'].includes(u.protocol) ? route.continue() : route.abort();
  });
}

export async function temaActivo(page) {
  await page.waitForFunction(() => getComputedStyle(document.documentElement).getPropertyValue('--color-acento').trim());
  await page.evaluate(() => document.fonts.ready);
  return page.evaluate(() => {
    const estilo = getComputedStyle(document.documentElement);
    const tokens = {};
    // La hoja del tema activo contiene únicamente variables validadas por la app.
    for (const hoja of document.styleSheets) {
      try {
        for (const regla of hoja.cssRules) if (regla.style) {
          for (const nombre of regla.style) if (nombre.startsWith('--')) tokens[nombre] = estilo.getPropertyValue(nombre).trim();
        }
      } catch { /* Las hojas ajenas al origen no forman parte del tema. */ }
    }
    return tokens;
  });
}

export async function fuentesActivas(page) {
  return page.evaluate(async () => {
    let css = '';
    for (const hoja of document.styleSheets) for (const regla of hoja.cssRules) {
      if (regla.type !== CSSRule.FONT_FACE_RULE) continue;
      let texto = regla.cssText;
      for (const m of texto.matchAll(/url\(["']?([^"')]+)["']?\)/g)) {
        const url = new URL(m[1], hoja.href || location.href);
        if (url.origin !== location.origin) throw new Error('Fuente ajena a la demo');
        const respuesta = await fetch(url);
        if (!respuesta.ok) throw new Error('Fuente no disponible');
        const datos = new Uint8Array(await respuesta.arrayBuffer());
        let binario = ''; for (const n of datos) binario += String.fromCharCode(n);
        texto = texto.replace(m[0], `url(data:font/woff2;base64,${btoa(binario)})`);
      }
      css += texto;
    }
    return css;
  });
}

export async function instalarTema(page, tokens, fuentes) {
  await page.setContent('<!doctype html><html lang="es"><head><meta charset="utf-8"></head><body></body></html>');
  await page.evaluate(({ tokens, fuentes }) => {
    for (const [nombre, valor] of Object.entries(tokens)) {
      if (!/^--[a-z0-9-]+$/.test(nombre)) throw new Error('Token no válido');
      document.documentElement.style.setProperty(nombre, valor);
    }
    const s = document.createElement('style'); s.textContent = fuentes; document.head.append(s);
    Object.assign(document.body.style, { margin: '0', background: 'var(--color-fondo)', color: 'var(--color-texto)', fontFamily: 'var(--fuente-texto)' });
  }, { tokens, fuentes });
  await page.evaluate(() => document.fonts.ready);
}

export async function cartela(page, bloque) {
  await page.evaluate(b => {
    document.getElementById('recorrido-cartela')?.remove();
    const d = document.createElement('section'); d.id = 'recorrido-cartela';
    Object.assign(d.style, { position: 'fixed', inset: '0', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
      boxSizing: 'border-box', padding: 'var(--espacio-pagina)', gap: 'var(--espacio-9)', background: 'var(--color-fondo)', color: 'var(--color-texto)', textAlign: 'center', zIndex: '2147483647' });
    const h = document.createElement('h1'); h.textContent = b['Título'];
    Object.assign(h.style, { fontFamily: 'var(--fuente-titulo)', fontSize: 'calc(var(--texto-titulo-pagina) * 1.5)', fontWeight: 'var(--peso-negrita)', margin: '0' });
    const p = document.createElement('p'); p.textContent = b.Cartela;
    Object.assign(p.style, { fontFamily: 'var(--fuente-texto)', fontSize: 'var(--texto-titulo-pagina)', margin: '0', maxWidth: '85%' });
    d.append(h, p); document.body.append(d);
  }, bloque);
}

export async function rotulo(page, paso, dir, referencias, ms = 5000) {
  await page.evaluate(texto => {
    document.getElementById('recorrido-rotulo')?.remove();
    const r = document.createElement('div'); r.id = 'recorrido-rotulo'; r.textContent = texto;
    Object.assign(r.style, { position: 'fixed', left: '5%', right: '5%', bottom: 'var(--espacio-8)', padding: 'var(--espacio-tarjeta)',
      borderRadius: 'var(--radio-tarjeta)', background: 'var(--color-acento)', color: 'var(--color-sobre-acento)',
      border: 'var(--borde-marcado) solid var(--color-borde)', fontFamily: 'var(--fuente-texto)', fontSize: 'var(--texto-titulo-pagina)',
      fontWeight: 'var(--peso-seminegrita)', boxSizing: 'border-box', zIndex: '2147483647', pointerEvents: 'none' });
    document.body.append(r);
  }, paso['Rótulo']);
  const caja = await page.locator('#recorrido-rotulo').boundingBox();
  if (!caja || caja.y < 720) throw new Error('Rótulo demasiado largo: debe caber en la franja inferior');
  // Captura completa para detectar la franja del rótulo en el WebM real.
  const fichero = resolve(dir, `${paso.id}.png`);
  await page.screenshot({ path: fichero });
  referencias.push({ id: paso.id, imagen: fichero, texto: paso['Rótulo'] });
  await espera(Math.max(5000, ms));
  await page.locator('#recorrido-rotulo').evaluate(n => n.remove());
}

export async function apuntar(loc) {
  await loc.scrollIntoViewIfNeeded();
  const caja = await loc.boundingBox();
  if (!caja) throw new Error('Control no visible');
  await loc.page().mouse.move(caja.x + caja.width / 2, caja.y + caja.height / 2, { steps: 16 });
  await loc.page().evaluate(() => {
    for (const anterior of document.querySelectorAll('[data-recorrido-foco]')) {
      anterior.style.outline = anterior.dataset.recorridoOutline;
      anterior.style.outlineOffset = anterior.dataset.recorridoOffset;
      delete anterior.dataset.recorridoFoco;
      delete anterior.dataset.recorridoOutline;
      delete anterior.dataset.recorridoOffset;
    }
  });
  await loc.evaluate(n => {
    n.dataset.recorridoFoco = '1'; n.dataset.recorridoOutline = n.style.outline; n.dataset.recorridoOffset = n.style.outlineOffset;
    n.style.outline = 'var(--borde-actual) solid var(--color-foco)'; n.style.outlineOffset = 'var(--espacio-2)';
  });
  await espera(350);
}

export async function pulsar(loc) {
  await apuntar(loc);
  await loc.click();
  await espera(250);
}

export async function escribir(loc, valor, secreto = false) {
  await apuntar(loc); await loc.click();
  if (secreto) await loc.fill(valor);
  else { await loc.fill(''); await loc.pressSequentially(valor, { delay: 65 }); }
}

export async function elegir(loc, valor) {
  await apuntar(loc); await loc.selectOption(valor); await espera(250);
}

export async function escenas(page, guion, dir, referencias, origen) {
  const tramos = [];
  let activo;
  const instalar = async () => {
    await page.evaluate(texto => {
      let cursor = document.getElementById('recorrido-raton');
      if (!cursor) {
        cursor = document.createElement('div'); cursor.id = 'recorrido-raton';
        Object.assign(cursor.style, { position: 'fixed', width: '26px', height: '34px', left: '50%', top: '50%',
          clipPath: 'polygon(0 0, 0 90%, 28% 68%, 48% 100%, 65% 91%, 44% 60%, 85% 60%)',
          background: 'var(--color-acento)', filter: 'drop-shadow(0 0 2px var(--color-fondo))',
          pointerEvents: 'none', zIndex: '2147483647' });
        document.body.append(cursor);
        document.addEventListener('mousemove', e => { cursor.style.left = `${e.clientX}px`; cursor.style.top = `${e.clientY}px`; });
      }
      document.getElementById('recorrido-rotulo')?.remove();
      if (!texto) return;
      const r = document.createElement('div'); r.id = 'recorrido-rotulo'; r.textContent = texto;
      Object.assign(r.style, { position: 'fixed', left: '5%', right: '5%', bottom: 'var(--espacio-8)', padding: 'var(--espacio-tarjeta)',
        borderRadius: 'var(--radio-tarjeta)', background: 'var(--color-acento)', color: 'var(--color-sobre-acento)',
        fontFamily: 'var(--fuente-texto)', fontSize: 'var(--texto-titulo-pagina)', fontWeight: 'var(--peso-seminegrita)',
        boxSizing: 'border-box', pointerEvents: 'none', zIndex: '2147483646' });
      document.body.append(r);
    }, activo?.['Rótulo'] ?? null);
  };
  page.on('domcontentloaded', () => { void instalar().catch(() => {}); });
  await instalar();
  const ahora = () => (performance.now() - origen) / 1000;
  return async (id, acciones) => {
    activo = guion.pasos.find(p => p.id === id);
    if (!activo?.frases?.length) throw new Error(`Falta la voz preparada de ${id}`);
    if (acciones.length !== activo.frases.length) throw new Error(`Acciones incompletas: ${id}`);
    await instalar();
    if (!tramos.length) {
      const reloj_s = ahora();
      await page.screenshot({ path: resolve(dir, 'ancla.png') });
      guardar(resolve(dir, 'ancla.json'), { reloj_s });
      await espera(500);
    }
    const inicio = ahora();
    const fases = [];
    for (let i = 0; i < activo.frases.length; i++) {
      const desde = ahora();
      await espera(150);
      await acciones[i]();
      const objetivo = activo.frases[i].duracion_s + 0.3;
      await espera(Math.max(0, (objetivo - (ahora() - desde)) * 1000));
      fases.push({ inicio_s: desde, fin_s: ahora(), voz_s: activo.frases[i].duracion_s });
    }
    const fin = ahora();
    const imagen = resolve(dir, `${id}.png`);
    await page.screenshot({ path: imagen });
    referencias.push({ id, imagen, texto: activo['Rótulo'] });
    tramos.push({ id, inicio_s: inicio, fin_s: fin, fases });
    await page.locator('#recorrido-rotulo').evaluate(n => n.remove());
    await page.screenshot({ path: resolve(dir, 'ultima.png') });
    activo = null;
    guardar(resolve(dir, 'referencias.json'), referencias);
    guardar(resolve(dir, 'escenas.json'), tramos);
  };
}

export function guardar(ruta, valor) {
  mkdirSync(resolve(ruta, '..'), { recursive: true });
  writeFileSync(ruta, JSON.stringify(valor, null, 2));
}
