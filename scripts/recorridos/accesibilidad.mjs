// Auditoría local de las rutas INT5 con Chrome del sistema y axe fijado en npm.
import assert from 'node:assert/strict';
import { readFile, readdir, mkdir, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { chrome, baseLocal, restringir } from './comun.mjs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
const require = createRequire(new URL('../../web/package.json', import.meta.url));
const axe = require('axe-core');
const base = baseLocal(process.argv[2]);
const salida = process.argv[3] || join(tmpdir(), 'f503-axe');
await mkdir(salida, { recursive: true });
for (const idioma of ['es', 'en']) {
  const carpeta = new URL(`../../idiomas/${idioma}/textos/`, import.meta.url);
  const respuesta = await fetch(`${base}/api/idiomas/${idioma}/textos`, { signal: AbortSignal.timeout(10000), redirect: 'error' });
  assert.equal(respuesta.status, 200, `Idioma no disponible: ${idioma}`);
  const actuales = await respuesta.json();
  const ausentes = [];
  for (const fichero of await readdir(carpeta)) {
    if (!fichero.endsWith('.json')) continue;
    const esperados = JSON.parse(await readFile(new URL(fichero, carpeta)));
    for (const [clave, valor] of Object.entries(esperados)) {
      if (typeof valor === 'string' && valor.trim() && actuales[clave] !== valor) ausentes.push(clave);
    }
  }
  assert.equal(ausentes.length, 0, `Textos distintos al catálogo en ${idioma}: ${ausentes.join(', ')}`);
}
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const registro = await readFile(new URL('../../web/src/pantallas.ts', import.meta.url), 'utf8');
const rutas = [];
const modulos = new Map([...registro.matchAll(/import \{ pantallas as (\w+) \} from '\.\/([^']+)\/pantallas'/g)].map(m => [m[1], m[2]]));
const composicion = registro.match(/export const pantallas[^=]*= \[([^\]]*)\]/)?.[1];
assert(composicion, 'Composición INT5 no reconocida');
const aportaciones = composicion.split(',').map(v => v.trim()).filter(Boolean);
for (const aporte of aportaciones) {
  assert.match(aporte, /^\.\.\.\w+$/, 'Aportación INT5 no reconocida');
  const modulo = modulos.get(aporte.slice(3));
  assert(modulo, `Importación INT5 ausente: ${aporte}`);
  const contenido = await readFile(new URL(`../../web/src/${modulo}/pantallas.ts`, import.meta.url), 'utf8');
  const declaradas = [...contenido.matchAll(/ruta\s*:/g)].length;
  const antes = rutas.length;
  for (const r of contenido.matchAll(/ruta: '([^']+)', perfiles: \[([^\]]+)\]/g)) rutas.push({ ruta: r[1], perfiles: [...r[2].matchAll(/'([^']+)'/g)].map(p => p[1]) });
  assert.equal(rutas.length - antes, declaradas, `Registro no reconocido: ${modulo}`);
}
assert(rutas.length > 0, 'Registro vacío');
assert.equal(new Set(rutas.map(r => r.ruta)).size, rutas.length);
const cuentas = { coordinacion: 'ines.moreno', profesional_etf: 'marta.cabrera', administracion: 'elena.salas', consulta: 'lucia.navarro', pendiente: 'sergio.campos' };
const browser = await chrome();
const resultados = [];
const pendientes = new WeakMap();
async function estable(p) {
  await p.evaluate(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))));
  const limite = Date.now() + 20000;
  let quieto = Date.now();
  while (Date.now() - quieto < 300) {
    if (pendientes.get(p).size) quieto = Date.now();
    if (Date.now() > limite) throw new Error(`Peticiones sin terminar: ${[...pendientes.get(p)].map(r => new URL(r.url()).pathname).join(', ')}`);
    await p.waitForTimeout(50);
  }
}
let graves = 0;
let completada = false;
async function preferencia(p, id, valor, autenticado) {
  if (autenticado) await p.locator('.persona-conectada').waitFor();
  await estable(p);
  if (await p.locator(`#${id}`).inputValue() === valor) return;
  const persona = p.locator('.persona-conectada');
  if (await persona.count() && !await persona.evaluate(e => e.open)) await persona.locator('summary').first().click();
  const preferencias = p.locator('.preferencias');
  if (!await preferencias.evaluate(e => e.open)) await preferencias.locator('summary').click();
  await p.locator(`#${id}`).selectOption(valor);
  if (id === 'idioma') await p.waitForFunction(v => document.documentElement.lang === v, valor);
  else await p.waitForFunction(v => !!document.querySelector(`link[href="/api/temas/${v}/tokens"]`), valor);
  await estable(p);
  await p.evaluate(() => { document.querySelectorAll('details').forEach(e => { e.open = false; }); });
}
async function escanear(p, caso) {
  await estable(p);
  await p.evaluate(() => document.fonts.ready);
  await p.evaluate(axe.source);
  const a = await p.evaluate(async () => {
    const resultado = await axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'best-practice'] } });
    const reducir = v => ({ id: v.id, impact: v.impact, description: v.description, help: v.help, tags: v.tags, nodes: v.nodes.map(n => ({ target: n.target, failureSummary: n.failureSummary })) });
    return { violations: resultado.violations.map(reducir), incomplete: resultado.incomplete.map(reducir), passes: resultado.passes.length,
      lang: document.documentElement.lang, h1: document.querySelectorAll('main h1').length,
      desbordamiento: document.documentElement.scrollWidth > document.documentElement.clientWidth,
      encabezado: document.querySelector('main h1')?.textContent, rutaReal: location.pathname,
      alertas: [...document.querySelectorAll('main [role=alert]')].map(e => e.textContent.trim()) };
  });
  assert.equal(a.rutaReal, caso.destino.split('?')[0], `Ruta desviada: ${caso.ruta}`);
  assert.equal(a.lang, caso.idioma);
  graves += a.violations.filter(v => ['serious', 'critical'].includes(v.impact)).length;
  resultados.push({ estado: 'pantalla', ...caso, ...a });
  if (process.env.ETF_AXE_CAPTURAS === '1' && caso.idioma === 'es' && caso.tema === 'expedientes-etf' && !caso.estado) {
    const nombre = caso.ruta.replace(/[^a-z0-9]+/gi, '-').replace(/^-|-$/g, '') || 'entrada';
    await mkdir(new URL('../../docs/capturas/', import.meta.url), { recursive: true });
    await p.screenshot({ path: new URL(`../../docs/capturas/accesibilidad-${caso.perfil}-${nombre}-${caso.ancho}.png`, import.meta.url).pathname, fullPage: true });
  }
}
try {
  for (const [perfil, cuenta] of Object.entries({ sin_sesion: null, ...cuentas }).filter(([perfil]) => (!process.env.ETF_AXE_PERFIL || perfil === process.env.ETF_AXE_PERFIL) && (!process.env.ETF_AXE_RUTA || rutas.some(r => r.ruta === process.env.ETF_AXE_RUTA && r.perfiles.includes(perfil))))) {
    const contexto = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
    await restringir(contexto, base);
    const p = await contexto.newPage();
    const peticiones = new Set(); pendientes.set(p, peticiones);
    p.on('request', r => { if (['fetch', 'xhr'].includes(r.resourceType())) peticiones.add(r); });
    p.on('response', r => peticiones.delete(r.request()));
    p.on('requestfinished', r => peticiones.delete(r));
    p.on('requestfailed', r => peticiones.delete(r));
    if (process.env.ETF_AXE_DEBUG) p.on('response', r => console.log(r.status(), new URL(r.url()).pathname));
    p.setDefaultTimeout(20000);
    await p.goto(base + '/entrar');
    if (cuenta) {
      await p.locator('#cuenta-ad').fill(cuenta);
      await p.locator('#contrasena-ad').fill(claves[cuenta]);
      await p.locator('button[type="submit"]').click();
      await p.locator('#cuenta-ad').waitFor({ state: 'detached' });
    }
    let propias = rutas.filter(r => r.perfiles.includes(perfil) && (!process.env.ETF_AXE_RUTA || r.ruta === process.env.ETF_AXE_RUTA));
    if (!propias.length) propias = [{ ruta: await p.evaluate(() => location.pathname), perfiles: [perfil] }];
    const datos = cuenta && propias.some(r => r.ruta.includes('{id}') || r.ruta.startsWith('/equipo/')) ? await p.evaluate(async coordinacion => {
      const equipo = await fetch('/api/equipo/expedientes?desplazamiento=0').then(r => r.ok ? r.json() : null);
      const sesion = await fetch('/api/sesion').then(r => r.json());
      const opciones = await fetch('/api/estadistica/spaf/opciones').then(r => r.json());
      const etf = coordinacion ? opciones.equipos.find(e => e.clave_texto === 'equipo.02')?.id : sesion.equipo_id;
      const tabla = await fetch(`/api/estadistica/spaf?anio=2026&mes=9&etf=${etf}`).then(r => r.json());
      const bandeja = coordinacion ? await fetch('/api/coordinacion/bandeja/buscar', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf }, body: JSON.stringify({ estado: '', equipo_id: '', municipio_id: '', desde: '', hasta: '', busqueda: '', limite: 20, desplazamiento: 0 }) }).then(r => r.json()) : null;
      return { etf, spaf: tabla.filas?.[0]?.expediente_id, equipo: equipo?.filas?.[0]?.id, aceptada: bandeja?.filas?.find(f => f.numero)?.id, derivacion: bandeja?.filas?.find(f => f.estado === 'derivacion_recibida')?.id || bandeja?.filas?.[0]?.id };
    }, perfil === 'coordinacion') : {};
    for (const idioma of ['es', 'en']) for (const tema of ['expedientes-etf', 'alto-contraste']) {
      console.log(`AXE-COMBINACION: ${perfil}; ${idioma}; ${tema}`);
      await preferencia(p, 'idioma', idioma, !!cuenta);
      await preferencia(p, 'tema', tema, !!cuenta);
      for (const ancho of [1440, 390]) {
        await p.setViewportSize({ width: ancho, height: 1000 });
        for (const r of propias) {
          const id = r.ruta.includes('derivaciones') ? datos.derivacion : r.ruta.includes('tabla-spaf') ? datos.spaf : datos.aceptada;
          if (r.ruta.includes('{id}')) assert(id, `Falta expediente demo para ${r.ruta}`);
          let destino = r.ruta.replace('{id}', id || '');
          if (r.ruta.startsWith('/equipo/tabla-spaf')) {
            assert(datos.etf && datos.spaf, 'Faltan datos SPAF de septiembre de 2026');
            destino += `?equipo=${datos.etf}&mes=2026-09`;
          }
          if (['/equipo/expediente', '/equipo/menores', '/equipo/intervencion', '/equipo/seguimiento'].includes(r.ruta)) {
            assert(datos.equipo, 'Falta expediente del equipo'); destino += `?id=${datos.equipo}`;
          }
          await p.evaluate(destino => { history.pushState({}, '', destino); dispatchEvent(new PopStateEvent('popstate')); }, destino);
          const caso = { ruta: r.ruta, destino, perfil, idioma, tema, ancho };
          await escanear(p, caso);
          if (r.ruta === '/memoria') {
            const bloques = p.locator('nav.memoria-bloques button, .memoria-bloques[role="group"] > button');
            assert.equal(await bloques.count(), 4, 'Faltan bloques de Memoria');
            for (let i = 0; i < 4; i++) {
              if (!await bloques.nth(i).isDisabled()) await bloques.nth(i).click();
              await escanear(p, { ...caso, estado: `memoria-bloque-${i + 1}` });
            }
            const ayuda = p.locator('.memoria-cabecera button').first();
            await ayuda.click();
            await escanear(p, { ...caso, estado: 'memoria-ayuda' });
            await ayuda.click();
          }
          if (r.ruta === '/equipo/expediente') {
            const tabs = await p.getByRole('tab').evaluateAll(es => es.map(e => e.id));
            for (const tab of tabs) {
              await p.locator(`[id="${tab}"]`).click();
              await escanear(p, { ...caso, estado: tab });
            }
          }
          // Las ayudas son diálogos de sólo lectura; Escape comprueba también el retorno del foco.
          const ayudas = await p.locator('main button').evaluateAll(es => es.filter(e => e.textContent.trim() === '?' && !e.disabled).map(e => e.getAttribute('aria-label')).filter(Boolean));
          for (const ayuda of new Set(ayudas)) {
            const boton = p.getByRole('button', { name: ayuda, exact: true }).first();
            await boton.click();
            if (!await p.getByRole('dialog').count()) continue;
            await escanear(p, { ...caso, estado: `ayuda: ${ayuda}` });
            await p.keyboard.press('Escape');
            assert(await boton.evaluate(e => e === document.activeElement), `Ayuda no devuelve foco: ${r.ruta}`);
          }
        }
      }
    }
    await writeFile(`${salida}/resultados.json`, JSON.stringify({ axe: axe.version, chrome: browser.version(), registro: rutas, completada, filtrado: !!(process.env.ETF_AXE_PERFIL || process.env.ETF_AXE_RUTA), rutas: rutas.length, resultados }, null, 2));
    console.log(`AXE-PERFIL: ${perfil}; ${propias.length} pantallas × 8 combinaciones`);
    await contexto.close();
  }
  completada = true;
} finally {
  await browser.close();
  await writeFile(`${salida}/resultados.json`, JSON.stringify({ axe: axe.version, chrome: browser.version(), registro: rutas, completada, filtrado: !!(process.env.ETF_AXE_PERFIL || process.env.ETF_AXE_RUTA), rutas: rutas.length, resultados }, null, 2));
}
if (!process.env.ETF_AXE_PERFIL && !process.env.ETF_AXE_RUTA) {
  const previstas = (rutas.reduce((n, r) => n + Object.keys(cuentas).filter(p => r.perfiles.includes(p)).length, 0) + 1 + Object.keys(cuentas).filter(p => !rutas.some(r => r.perfiles.includes(p))).length) * 8;
  assert.equal(resultados.filter(r => r.estado === 'pantalla').length, previstas, 'Matriz incompleta');
}
console.log(`AXE: ${resultados.length} comprobaciones; ${graves} infracciones serias o críticas`);
if (graves) process.exitCode = 1;
