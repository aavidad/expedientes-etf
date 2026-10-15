// MEMDES: cifras reales de la demo, privacidad y recorrido con Chrome del sistema.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';
import { chromium, baseLocal, restringir } from './comun.mjs';

const base = baseLocal(process.argv[2]);
// La reserva de textos de una demo persistente puede conservar una edición anterior.
// Sólo se renueva este módulo, en la demo local del mismo worktree.
for (const idioma of ['es', 'en']) {
  const esperados = JSON.parse(await readFile(new URL(`../../idiomas/${idioma}/textos/memoria-desglose.json`, import.meta.url)));
  const servidos = async () => { const r = await fetch(`${base}/api/idiomas/${idioma}/textos`); assert.equal(r.status, 200); return r.json(); };
  const actuales = await servidos();
  if (Object.entries(esperados).some(([k,v]) => actuales[k] !== v)) {
    execFileSync('python3', ['-B', '-c', `import importlib.util,sys
from pathlib import Path
raiz=Path(sys.argv[1])
spec=importlib.util.spec_from_file_location("demo_memdes",raiz/"despliegue/demo/demo.py")
demo=importlib.util.module_from_spec(spec);spec.loader.exec_module(demo)
if demo.URL != sys.argv[2] or sys.argv[3] not in ("es","en"):
 raise SystemExit("La actualización de textos requiere la demo local del worktree.")
idioma=sys.argv[3]
demo.compose("cp",str(raiz/"idiomas"/idioma/"textos/memoria-desglose.json"),"app:/datos/idiomas/"+idioma+"/textos/memoria-desglose.json",capture_output=True,text=True)
`, fileURLToPath(new URL('../../', import.meta.url)), base, idioma], { timeout:30000 });
    const renovados = await servidos();
    assert(Object.entries(esperados).every(([k,v]) => renovados[k] === v));
  }
}
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const textos = { ...JSON.parse(await readFile(new URL('../../idiomas/es/textos/memoria.json', import.meta.url))), ...JSON.parse(await readFile(new URL('../../idiomas/es/textos/memoria-desglose.json', import.meta.url))) };
const capturas = resolve('docs/capturas');
await mkdir(capturas, { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
let proxima = 0;
let cifras = 0;
const errores = [];
async function entrar(cuenta) {
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 } });
  await restringir(contexto, base);
  await contexto.route(`${base}/api/**`, async ruta => {
    const espera = Math.max(Date.now(), proxima); proxima = espera + 720;
    await new Promise(r => setTimeout(r, Math.max(0, espera - Date.now())));
    await ruta.fallback();
  });
  const pagina = await contexto.newPage();
  pagina.on('pageerror', e => errores.push(e.message));
  await pagina.goto(base);
  await pagina.locator('#cuenta-ad').fill(cuenta);
  await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
  await pagina.locator('button[type=submit]').click();
  await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return { pagina, contexto };
}
async function pedir(pagina, ruta) {
  return pagina.evaluate(async ruta => {
    const r = await fetch(ruta); return { estado: r.status, datos: await r.json() };
  }, ruta);
}
function rutaDetalle(anio, equipo, ap, celda, medida, c, revision) {
  const parametros = new URLSearchParams({ celda: String(celda), medida, ...(equipo ? { etf: equipo } : {}), ...(revision ? { revision: String(revision) } : {}), ...(c ? { clave: JSON.stringify({ Grupo: c.grupo, Unidad: c.unidad }) } : {}) });
  return `/api/memoria/${anio}/desglose/${ap}?${parametros}`;
}
async function suma(pagina, equipo, anio = 2026) {
  const filtro = new URLSearchParams({ anio: String(anio), ...(equipo ? { etf: equipo } : {}) });
  const memoria = await pedir(pagina, `/api/estadistica/memoria?${filtro}`); assert.equal(memoria.estado, 200);
  const estado = await pedir(pagina, `/api/estadistica/memoria/estado?${filtro}`); assert.equal(estado.estado, 200);
  const revision = estado.datos.estado === 'cerrada' ? estado.datos.revision_actual : 0;
  const apartados = new Set();
  for (const ap of memoria.datos.apartados) {
    const selecciones = Object.entries(ap.total).filter(([, valor]) => valor > 0).map(([medida, valor]) => ({ celda: -1, medida, valor }));
    for (const [i, c] of ap.celdas.entries()) {
      for (const medida of ['numerador', 'sin_dato']) if (c[medida] > 0) selecciones.push({ celda: i, medida, valor: c[medida], c });
      if (c.denominador.conocido && c.denominador.valor > 0) selecciones.push({ celda: i, medida: 'denominador', valor: c.denominador.valor, c });
    }
    if (!selecciones.length) selecciones.push({ celda: -1, medida: 'familias', valor: ap.total.familias });
    for (const s of selecciones) {
      const r = await pedir(pagina, rutaDetalle(anio, equipo, ap.codigo, s.celda, s.medida, s.c, revision));
      assert.equal(r.estado, 200, `${ap.codigo}: ${JSON.stringify(r.datos)}`);
      assert.equal(r.datos.cifra, s.valor, `${ap.codigo}/${s.medida}`);
      assert.equal(r.datos.definitivas, !!revision);
      if (['15', '16'].includes(ap.codigo)) { assert.equal(r.datos.aviso, 'memoria.desglose.actividad'); continue; }
      assert.equal(r.datos.aviso, '', `${ap.codigo}/${s.medida}: ${r.datos.aviso}`);
      assert.equal(r.datos.filas.reduce((n, f) => n + f.cantidad, 0), s.valor, `${ap.codigo}/${s.medida}`);
      assert.equal(new Set(r.datos.filas.map(f => f.expediente_id)).size, r.datos.filas.length);
      for (const f of r.datos.filas) {
        assert.deepEqual(Object.keys(f).sort(), ['cantidad', 'equipo_id', 'expediente_id', 'ficha', 'numero']);
        if (equipo) assert.equal(f.equipo_id, equipo);
      }
      cifras++;
    }
    apartados.add(ap.codigo);
    process.stdout.write(`MEMDES-SUMA: ${equipo ? 'equipo' : 'provincia'} apartado ${ap.codigo}; ${cifras} cifras verificadas.\n`);
  }
  assert.equal(apartados.size, 33);
}
async function reflujo(pagina, ancho) {
  await pagina.setViewportSize({ width: ancho, height: 1000 });
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `desbordamiento ${ancho}`);
}
try {
  const coord = await entrar('ines.moreno');
  const equipos = await pedir(coord.pagina, '/api/estadistica/memoria/equipos'); assert.equal(equipos.estado, 200);
  const propio = await entrar('marta.cabrera');
  const sesion = await pedir(propio.pagina, '/api/sesion'); assert.equal(sesion.estado, 200);
  const equipo = sesion.datos.equipo_id;
  await suma(coord.pagina, '');
  await suma(propio.pagina, equipo);
  const otro = equipos.datos.find(e => e.id !== equipo).id;
  assert.equal((await pedir(propio.pagina, rutaDetalle(2026, otro, '1', -1, 'familias'))).estado, 403);
  const consulta = await entrar('lucia.navarro');
  assert.equal((await pedir(consulta.pagina, rutaDetalle(2026, '', '1', -1, 'familias'))).estado, 403);
  await consulta.pagina.goto(`${base}/memoria`);
  await consulta.pagina.getByRole('heading', { name: textos['memoria.titulo'], exact: true }).waitFor();
  assert.equal(await consulta.pagina.locator('a[href^="/memoria/desglose"]').count(), 0);
  const pagina = coord.pagina;
  await pagina.goto(`${base}/memoria`);
  await pagina.locator('#memoria-apartado-1').waitFor();
  const enlace = pagina.locator('#memoria-apartado-1 a.memoria-cifra').filter({ hasText: /[1-9]/ }).first();
  const ruta = await enlace.getAttribute('href'); assert(ruta);
  assert(!(await enlace.getAttribute('aria-label')).includes('{grupo}'));
  const etiquetaSinDato = pagina.locator('#memoria-cifra-1--1-familias_sin_relacion_menores');
  assert((await etiquetaSinDato.getAttribute('aria-label')).includes(textos['memoria.desglose.total']));
  await enlace.focus(); await pagina.keyboard.press('Enter');
  await pagina.getByRole('heading', { name: textos['memoria.desglose.titulo'], exact: true }).waitFor();
  await pagina.locator('.memoria-desglose-fila').first().waitFor();
  for (const ancho of [1440, 1024, 390, 360, 320]) {
    await reflujo(pagina, ancho);
    const volver = pagina.getByRole('link', { name: textos['memoria.desglose.volver'], exact: true });
    assert((await volver.boundingBox()).height >= 44);
    if ([1440, 390].includes(ancho)) await pagina.screenshot({ path: resolve(capturas, `memoria-desglose-${ancho}.png`) });
  }
  const ficha = pagina.locator('.memoria-desglose-fila a').first(); assert.equal(await ficha.count(), 1);
  const rutaFicha = await ficha.getAttribute('href');
  await ficha.click(); await pagina.waitForURL(`**${rutaFicha}`);
  assert.equal((await pedir(pagina, `/api/expedientes/${new URL(rutaFicha, base).searchParams.get('id')}`)).estado, 200);
  await pagina.goto(base + ruta);
  await pagina.locator('.memoria-desglose-fila').first().waitFor();
  await pagina.getByRole('link', { name: textos['memoria.desglose.volver'], exact: true }).click();
  await pagina.locator('#memoria-apartado-1').waitFor();
  assert.equal(new URL(pagina.url()).searchParams.get('anio'), '2026');
  const seleccion = new URL(ruta, base).searchParams;
  const foco = `memoria-cifra-${seleccion.get('apartado')}-${seleccion.get('celda')}-${seleccion.get('medida')}`;
  await pagina.waitForFunction(id => document.activeElement?.id === id, foco);
  assert.deepEqual(errores, []);
  process.stdout.write(`MEMDES-DEMO-SUMA-OK: ${cifras} cifras, provincia y equipo; 33 apartados; Consulta/ajeno 403; Chrome 1440/1024/390/360/320; sin nombres y ficha autorizada.\n`);
} finally { await navegador.close(); }
