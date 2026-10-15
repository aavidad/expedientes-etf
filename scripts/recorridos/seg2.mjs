// SEG2: permisos propios en consulta, API real y segunda persona en Chrome.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const base = process.argv[2];
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
const raiz = new URL('../../', import.meta.url);
const claves = JSON.parse(await readFile(new URL('despliegue/demo/.local/usuarios.json', raiz)));
const es = JSON.parse(await readFile(new URL('idiomas/es/textos/usuarios.json', raiz)));
const en = JSON.parse(await readFile(new URL('idiomas/en/textos/usuarios.json', raiz)));
// La reserva de una demo ya creada conserva sus textos anteriores. Sólo se
// sincroniza este módulo en el contenedor local de este mismo worktree.
const anteriores = [];
for (const [idioma, textos] of [['es', es], ['en', en]]) {
  const respuesta = await fetch(`${base}/api/idiomas/${idioma}/textos`);
  assert.equal(respuesta.status, 200);
  const actuales = await respuesta.json();
  if (Object.entries(textos).some(([clave, valor]) => actuales[clave] !== valor)) anteriores.push(idioma);
}
if (anteriores.length) execFileSync('python3', ['-B', '-c', `import importlib.util, sys
from pathlib import Path
raiz = Path(sys.argv[1])
spec = importlib.util.spec_from_file_location("demo_recorrido_seg2", raiz / "despliegue/demo/demo.py")
demo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(demo)
if demo.URL != sys.argv[2]:
 raise SystemExit("seg2.demo_distinta")
for idioma in sys.argv[3:]:
 if idioma not in ("es", "en"):
  raise SystemExit("seg2.idioma_no_valido")
 demo.compose("cp", str(raiz / "idiomas" / idioma / "textos/usuarios.json"), "app:/datos/idiomas/" + idioma + "/textos/usuarios.json", capture_output=True, text=True)
`, fileURLToPath(raiz), base, ...anteriores], { timeout: 30000 });
for (const [idioma, textos] of [['es', es], ['en', en]]) {
  const actuales = await (await fetch(`${base}/api/idiomas/${idioma}/textos`)).json();
  assert.ok(Object.entries(textos).every(([clave, valor]) => actuales[clave] === valor), `seg2.textos_desactualizados.${idioma}`);
}
await mkdir(new URL('docs/capturas/', raiz), { recursive: true });
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];
const contextos = [];
async function entrar(cuenta) {
  const c = await browser.newContext({ viewport: { width: 1440, height: 900 } }); contextos.push(c);
  await c.route('**/*', (ruta) => new URL(ruta.request().url()).origin === base ? ruta.continue() : ruta.abort());
  const page = await c.newPage(); page.setDefaultTimeout(15000);
  page.on('pageerror', e => errores.push(e.message));
  await page.goto(base + '/entrar');
  await page.locator('#cuenta-ad').fill(cuenta); await page.locator('#contrasena-ad').fill(claves[cuenta]);
  const respuesta = page.waitForResponse(r => r.url() === base + '/api/sesion' && r.request().method() === 'POST');
  await page.locator('#contrasena-ad').press('Tab'); await page.keyboard.press('Enter');
  assert.equal((await respuesta).status(), 200);
  await page.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return page;
}
// fetch en Chrome conserva la cookie Secure de la demo en loopback.
async function peticion(page, url, opciones = {}) {
  const r = await page.evaluate(async ({ url, opciones }) => {
    const respuesta = await fetch(url, { method: opciones.method ?? 'GET', credentials: 'same-origin', headers: { 'Content-Type': 'application/json', ...opciones.headers }, body: opciones.data === undefined ? undefined : JSON.stringify(opciones.data) });
    return { status: respuesta.status, datos: await respuesta.json() };
  }, { url, opciones });
  return { status: () => r.status, ok: () => r.status >= 200 && r.status < 300, json: async () => r.datos };
}
async function sesion(p) { const r = await peticion(p, base + '/api/sesion'); assert.equal(r.status(), 200); return r.json(); }
async function lista(p) { const r = await peticion(p, base + '/api/admin/usuarios'); assert.equal(r.status(), 200); return r.json(); }
async function cambiar(p, usuario, perfiles, equipo = null, estado = 'activo', esperado = 200) {
  const r = await peticion(p, `${base}/api/admin/usuarios/${usuario.id}/perfiles`, { method: 'PUT',
    headers: { Origin: base, 'X-CSRF-Token': (await sesion(p)).csrf },
    data: { version: usuario.version, estado, perfiles, equipo },
  });
  assert.equal(r.status(), esperado);
  const datos = await r.json();
  if (esperado === 403) assert.equal(datos.error.clave, 'identidad.cambio_propio_denegado');
  return datos;
}
async function preferencias(page, idioma, tema) {
  for (const selector of ['.persona-conectada', '.preferencias']) {
    if (!await page.locator(selector).evaluate(e => e.open)) await page.locator(selector + ' > summary').click();
  }
  await page.locator('#tema').selectOption(tema);
  await page.waitForFunction(tema => document.querySelector(`link[href="/api/temas/${tema}/tokens"]`)?.sheet, tema);
  await page.locator('#idioma').selectOption(idioma);
  await page.waitForFunction(idioma => document.documentElement.lang === idioma, idioma);
  if (await page.locator('.persona-conectada').evaluate(e => e.open)) await page.locator('.persona-conectada > summary').click();
}
function concesion(perfil) { return { perfil, desde: new Date().toISOString(), hasta: null }; }
let admin, segunda, original, respaldo;
try {
  const coord = await entrar('ines.moreno');
  assert.equal((await peticion(coord, base + '/api/admin/usuarios')).status(), 403);
  await coord.goto(base + '/coordinacion/smi'); await coord.getByRole('heading', { level: 1 }).waitFor();
  admin = await entrar('elena.salas');
  await preferencias(admin, 'es', 'expedientes-etf');
  let usuarios = await lista(admin);
  original = usuarios.find(u => u.cuenta === 'elena.salas'); respaldo = usuarios.find(u => u.cuenta === 'sergio.campos');
  assert.equal(original.propio, true); assert.equal(respaldo.propio, false);
  await cambiar(admin, respaldo, [concesion('administracion')]);
  segunda = await entrar('sergio.campos');
  const datosCatalogo = await (await peticion(admin, base + '/api/admin/catalogos/asignacion_municipios/datos?fecha=2026-10-14')).json();
  const profesional = concesion('profesional_etf');
  const equipo = { equipo_id: datosCatalogo.equipos[0].id, puesto: 'ts', desde: profesional.desde, hasta: null };
  // Otra persona concede el equipo; el destinatario nunca puede modificarlo.
  await cambiar(segunda, original, [...original.perfiles, profesional], equipo);
  const propia = (await lista(admin)).find(u => u.id === original.id);
  const intentos = [
    { perfiles: [...propia.perfiles, concesion('coordinacion')], equipo: propia.equipo },
    { perfiles: propia.perfiles.filter(p => p.perfil !== 'profesional_etf'), equipo: null },
    { perfiles: propia.perfiles, equipo: { ...propia.equipo, equipo_id: datosCatalogo.equipos[1].id } },
    { perfiles: [], equipo: null, estado: 'baja' },
    { perfiles: propia.perfiles.map(p => ({ ...p, desde: new Date(Date.now() + 3600000).toISOString() })), equipo: propia.equipo },
  ];
  for (const intento of intentos) await cambiar(admin, propia, intento.perfiles, intento.equipo, intento.estado ?? 'activo', 403);
  const intacta = (await lista(admin)).find(u => u.id === original.id);
  assert.deepEqual(intacta, propia);
  await admin.goto(base + '/admin/usuarios');
  await admin.getByRole('button', { name: es['usuarios.ver_propios'], exact: true }).focus();
  await admin.keyboard.press('Enter');
  await admin.getByRole('heading', { name: es['usuarios.permisos_propios'], exact: true }).waitFor();
  await admin.getByText(es['identidad.cambio_propio_denegado'], { exact: true }).waitFor();
  for (const control of await admin.locator('form input, form select, form button').all()) assert.equal(await control.isDisabled(), true);
  assert.equal(await admin.getByRole('button', { name: es['usuarios.guardar'], exact: true }).isDisabled(), true);
  for (const width of [1440, 1024, 390, 360]) {
    await admin.setViewportSize({ width, height: 900 });
    await admin.waitForFunction(() => document.querySelector('.cabecera-app')?.dataset.compacto === String(innerWidth < parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu')) * 4));
    assert.ok(await admin.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `desborde ${width}`);
    if ([1440, 390].includes(width)) await admin.screenshot({ path: new URL(`docs/capturas/usuarios-seg2-${width}.png`, raiz).pathname, fullPage: true });
  }
  await admin.setViewportSize({ width: 1440, height: 900 });
  await preferencias(admin, 'en', 'alto-contraste');
  await admin.getByRole('button', { name: en['usuarios.ver_propios'], exact: true }).click();
  await admin.getByRole('heading', { name: en['usuarios.permisos_propios'], exact: true }).waitFor();
  await admin.getByText(en['identidad.cambio_propio_denegado'], { exact: true }).waitFor();
  for (const width of [1440, 390]) {
    await admin.setViewportSize({ width, height: 900 });
    await admin.waitForFunction(() => document.querySelector('.cabecera-app')?.dataset.compacto === String(innerWidth < parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu')) * 4));
    assert.ok(await admin.evaluate(() => document.documentElement.scrollWidth <= innerWidth), JSON.stringify(await admin.evaluate(() => ({ ancho: innerWidth, elementos: [...document.querySelectorAll('main *')].filter(e => e.getBoundingClientRect().right > innerWidth).map(e => ({ tipo: e.tagName, clase: e.className, ancho: e.getBoundingClientRect().width })).slice(0,8) }))));
  }
  assert.equal(await admin.evaluate(() => localStorage.length + sessionStorage.length), 0);
  await preferencias(admin, 'es', 'expedientes-etf');
  // Revocar los permisos del segundo administrador también exige otra persona.
  await cambiar(segunda, propia, original.perfiles, original.equipo);
  usuarios = await lista(admin);
  await cambiar(admin, usuarios.find(u => u.id === respaldo.id), respaldo.perfiles, respaldo.equipo, respaldo.estado === 'baja' ? 'baja' : 'activo');
  assert.deepEqual(errores, []);
  console.log('SEG2-CHROME-OK: coordinación denegada; cinco autocambios API 403; consulta propia por teclado; segunda persona concede y retira; es/en y alto contraste; 1440/1024/390/360 sin desbordes; capturas 1440/390');
} finally {
  // Restaura exclusivamente las concesiones inventadas usadas por el recorrido.
  if (admin) await preferencias(admin, 'es', 'expedientes-etf');
  if (segunda && original && admin) {
    const r = await peticion(segunda, base + '/api/admin/usuarios');
    if (r.ok()) {
      const u = (await r.json()).find(u => u.id === original.id);
      if (u && JSON.stringify(u.perfiles) !== JSON.stringify(original.perfiles)) await cambiar(segunda, u, original.perfiles, original.equipo);
    }
    const r2 = await peticion(admin, base + '/api/admin/usuarios');
    if (r2.ok() && respaldo) {
      const u = (await r2.json()).find(u => u.id === respaldo.id);
      if (u && u.perfiles.some(p => p.perfil === 'administracion')) await cambiar(admin, u, respaldo.perfiles, respaldo.equipo, respaldo.estado === 'baja' ? 'baja' : 'activo');
    }
  }
  await Promise.all(contextos.map(c => c.close())); await browser.close();
}
