// Renderiza la marca existente con sus textos, fuente y tokens del tema.
import { readFile, writeFile } from 'node:fs/promises';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const tokens = JSON.parse(await readFile(new URL('../../temas/expedientes-etf/tokens.json', import.meta.url)));
const textos = JSON.parse(await readFile(new URL('../../idiomas/es/textos/comun.json', import.meta.url)));
const fuente = await readFile(new URL('../../temas/expedientes-etf/fuentes/public-sans.woff2', import.meta.url));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
try {
  const imagenes = [];
  for (const lado of [16, 32, 48, 256]) {
    const pagina = await navegador.newPage({ viewport: { width: lado, height: lado }, deviceScaleFactor: 1 });
    await pagina.setContent('<style>body{margin:0}</style><div></div>');
    await pagina.evaluate(async ({ tokens, marca, fuente, lado }) => {
      const familia = tokens['fuente.texto'].split(',')[0].trim().replace(/^['"]|['"]$/g, '');
      const tipografia = new FontFace(familia, `url(data:font/woff2;base64,${fuente})`, { weight: `${tokens['peso.normal']} ${tokens['peso.negrita']}` });
      await tipografia.load(); document.fonts.add(tipografia);
      const escala = lado / parseFloat(tokens['tamano.marca']);
      const elemento = document.querySelector('div');
      Object.assign(elemento.style, {
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        width: `${lado}px`, height: `${lado}px`,
        borderRadius: `${parseFloat(tokens['radio.control']) * escala}px`,
        background: tokens['color.acento'], color: tokens['color.sobre-acento'],
        fontFamily: tokens['fuente.texto'], fontSize: `${parseFloat(tokens['texto.meta']) * escala}px`,
        fontWeight: tokens['peso.negrita'],
      });
      elemento.textContent = marca;
      await document.fonts.ready;
    }, { tokens, marca: textos['comun.marca'], fuente: fuente.toString('base64'), lado });
    imagenes.push({ lado, png: await pagina.screenshot({ omitBackground: true }) });
    await pagina.close();
  }
  const cabecera = Buffer.alloc(6 + 16 * imagenes.length);
  cabecera.writeUInt16LE(1, 2); cabecera.writeUInt16LE(imagenes.length, 4);
  let posicion = cabecera.length;
  imagenes.forEach(({ lado, png }, indice) => {
    const base = 6 + 16 * indice;
    cabecera[base] = lado === 256 ? 0 : lado; cabecera[base + 1] = cabecera[base];
    cabecera.writeUInt16LE(1, base + 4); cabecera.writeUInt16LE(32, base + 6);
    cabecera.writeUInt32LE(png.length, base + 8); cabecera.writeUInt32LE(posicion, base + 12);
    posicion += png.length;
  });
  await writeFile(process.argv[2], Buffer.concat([cabecera, ...imagenes.map(i => i.png)]));
} finally { await navegador.close(); }
