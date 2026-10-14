// Cartelas y hoja de fotogramas con el mismo tema capturado durante la grabación.
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { chrome, viewport, instalarTema, cartela } from './comun.mjs';
const [dir, modo = 'cartelas'] = process.argv.slice(2);
const guion = JSON.parse(readFileSync(resolve(dir, 'guion.json'), 'utf8'));
const tema = JSON.parse(readFileSync(resolve(dir, 'tema.json'), 'utf8'));
const browser = await chrome();
try {
  const contexto = await browser.newContext({ viewport, serviceWorkers: 'block' });
  await contexto.route('**/*', r => r.abort());
  const page = await contexto.newPage();
  await instalarTema(page, tema.tokens, tema.fuentes);
  if (modo === 'cartelas') {
    for (const [nombre, bloque] of [['intro', guion.intro], ['cierre', guion.cierre]]) {
      if (nombre === 'cierre') {
        const fondo = readFileSync(resolve(dir, 'ultima.png')).toString('base64');
        await page.evaluate(fondo => {
          const im = document.createElement('img'); im.src = `data:image/png;base64,${fondo}`;
          Object.assign(im.style, { position: 'fixed', inset: '0', width: '100%', height: '100%', filter: 'blur(7px)' });
          document.body.append(im);
        }, fondo);
      }
      await cartela(page, bloque);
      if (nombre === 'cierre') await page.evaluate(() => {
        const d = document.getElementById('recorrido-cartela'); d.style.background = 'transparent';
        for (const e of d.children) Object.assign(e.style, { background: 'var(--color-fondo)', padding: 'var(--espacio-tarjeta)', borderRadius: 'var(--radio-tarjeta)' });
      });
      await page.screenshot({ path: resolve(dir, `${nombre}.png`) });
    }
  } else {
    const filas = JSON.parse(readFileSync(resolve(dir, 'control.json'), 'utf8'));
    for (const fila of filas) fila.imagen = readFileSync(resolve(dir, fila.imagen)).toString('base64');
    await page.evaluate(({ filas, titulo }) => {
      const h = document.createElement('h1'); h.textContent = titulo; document.body.append(h);
      for (const fila of filas) {
        const d = document.createElement('section');
        Object.assign(d.style, { display: 'flex', alignItems: 'center', gap: 'var(--espacio-8)', padding: 'var(--espacio-tarjeta)', borderBottom: 'var(--borde-normal) solid var(--color-borde)' });
        const im = document.createElement('img'); im.src = `data:image/png;base64,${fila.imagen}`;
        im.style.width = '45%';
        const p = document.createElement('p'); p.textContent = `${fila.id} · ${fila.segundo.toFixed(2)} s\n${fila.texto}\n${fila.voz}`;
        Object.assign(p.style, { whiteSpace: 'pre-line', fontSize: 'var(--texto-titulo-seccion)', color: 'var(--color-texto)' });
        d.append(im, p); document.body.append(d);
      }
    }, { filas, titulo: guion.intro['Título'] });
    await page.screenshot({ path: resolve(dir, 'hoja-control.png'), fullPage: true });
  }
} finally { await browser.close(); }
