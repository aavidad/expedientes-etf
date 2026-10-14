// Regenera la marca del armazón desde los tokens y la fuente de cada tema.
import { readFile, mkdir, readdir } from 'node:fs/promises';
import { chromium } from '../recorridos/comun.mjs';
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
try {
 const temas = await readdir(new URL('../../temas/', import.meta.url), { withFileTypes: true });
 for (const directorio of temas.filter(d => d.isDirectory())) {
  const tema = directorio.name;
  const raiz = new URL(`../../temas/${tema}/`, import.meta.url);
  const tokens = JSON.parse(await readFile(new URL('tokens.json', raiz)));
  const marca = JSON.parse(await readFile(new URL('../../idiomas/es/textos/comun.json', import.meta.url)))['comun.marca'];
  const metadatos = JSON.parse(await readFile(new URL('tema.json', raiz)));
  const fuente = await readFile(new URL(metadatos.fuentes.texto, raiz));
  await mkdir(new URL('iconos/', raiz), { recursive: true });
  for (const tamano of [192, 512]) for (const uso of ['any', 'maskable']) {
   const page = await browser.newPage({ viewport: { width: tamano, height: tamano }, deviceScaleFactor: 1 });
   await page.route('**/*', r => r.abort());
   await page.setContent('<html><head></head><body><span></span></body></html>');
   // Escala la marca; maskable mantiene todo el texto dentro del círculo seguro.
   await page.addStyleTag({ content: `@font-face{font-family:Marca;src:url(data:font/woff2;base64,${fuente.toString('base64')})}body{margin:0;width:${tamano}px;height:${tamano}px;display:flex;align-items:center;justify-content:center;background:${tokens['color.acento']};color:${tokens['color.sobre-acento']};font-family:Marca;font-size:${tamano * Number.parseFloat(tokens['texto.meta']) / Number.parseFloat(tokens['tamano.marca']) * (uso === 'maskable' ? 0.7 : 1)}px;font-weight:${tokens['peso.negrita']}}` });
   await page.locator('span').evaluate((n, texto) => { n.textContent = texto; }, marca);
   await page.evaluate(() => document.fonts.ready);
   await page.screenshot({ path: new URL(`iconos/etf-${tamano}-${uso}.png`, raiz).pathname });
   await page.close();
  }
 }
} finally { await browser.close(); }
console.log('PWA1: iconos regenerados desde los tokens.');
