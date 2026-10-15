// Recorrido SEG1 contra la demostración local: sólo datos inventados.
import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';
const raiz = new URL('../../', import.meta.url).pathname;
const base = `http://127.0.0.1:${(await readFile(raiz + 'despliegue/demo/.local/puerto', 'utf8')).trim()}`;
const axe = await readFile(raiz + 'web/node_modules/axe-core/axe.min.js', 'utf8');
const claves = JSON.parse(await readFile(raiz + 'despliegue/demo/.local/usuarios.json'));
await mkdir(raiz + 'docs/capturas', { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
const errores = [];
const paginas = [];
async function entrar(cuenta) {
  const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 } });
  await contexto.route('**/*', r => new URL(r.request().url()).origin === base ? r.continue() : r.abort());
  const pagina = await contexto.newPage(); paginas.push(pagina);
  pagina.on('pageerror', e => errores.push(e.message));
  await pagina.goto(base + '/entrar');
  await pagina.locator('#cuenta-ad').fill(cuenta); await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
  await pagina.locator('button[type=submit]').click(); await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  return pagina;
}
async function sesion(pagina, method = 'GET') {
  return pagina.evaluate(async method => {
    const a = await (await fetch('/api/sesion')).json();
    const r = await fetch('/api/sesion', { method, headers: { 'X-CSRF-Token': a.csrf } });
    return { estado: r.status, caduca: r.headers.get('X-ETF-Sesion-Caduca'), maxima: r.headers.get('X-ETF-Sesion-Maxima') };
  }, method);
}
async function prepararTiempo(pagina, tipo) {
  const token = (await pagina.context().cookies()).find(c => c.name === '__Host-etf-sesion')?.value;
  assert(token);
  const huella = createHash('sha256').update(token).digest('hex');
  const campos = tipo === 'normal' ? "creada_en=now()-interval '1 hour',ultima_actividad_en=now(),caduca_en=now()+interval '7 hours'"
    : tipo === 'inactividad' ? "creada_en=now()-interval '1 hour',ultima_actividad_en=now()-interval '28 minutes 30 seconds',caduca_en=now()+interval '7 hours'"
    : tipo === 'caducada' ? "creada_en=now()-interval '1 hour',ultima_actividad_en=now()-interval '31 minutes',caduca_en=now()+interval '7 hours'"
    : "creada_en=now()-interval '7 hours 58 minutes 30 seconds',ultima_actividad_en=now(),caduca_en=now()+interval '90 seconds'";
  const sql = `UPDATE identidad.sesion SET ${campos} WHERE id_huella=decode('${huella}','hex');`;
  execFileSync('python3', ['-c', "import sys; sys.path.insert(0,'despliegue/demo'); import demo; demo.compose('exec','-T','postgres','sh','-c','PGPASSWORD=\"$POSTGRES_PASSWORD\" exec psql -U etf_instalador -d etf -v ON_ERROR_STOP=1',input=sys.stdin.read(),capture_output=True,text=True)"] , { cwd: raiz, input: sql, stdio: ['pipe', 'pipe', 'pipe'] });
}
async function captura(pagina, nombre, ancho) {
  await pagina.setViewportSize({ width: ancho, height: 1000 });
  await pagina.evaluate(() => document.fonts.ready);
  await pagina.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth);
  assert(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
  if (nombre === 'sesion') {
    await pagina.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await pagina.waitForFunction(() => { const r = document.querySelector('.seg1-aviso')?.getBoundingClientRect(); return r && r.top >= 0 && r.bottom <= innerHeight; });
  }
  await pagina.screenshot({ path: `${raiz}docs/capturas/seg1-${nombre}-${ancho}.png`, fullPage: nombre !== 'sesion' });
}
try {
  const consulta = await entrar('lucia.navarro');
  await consulta.goto(base + '/memoria');
  await consulta.locator('.memoria-apartado').first().waitFor();
  const cifras = await consulta.evaluate(async () => (await fetch('/api/consulta/memoria/2026')).json());
  assert.equal(cifras.umbral_privacidad, 3);
  let ocultas = 0;
  for (const apartado of cifras.apartados) {
    for (const n of Object.values(apartado.total)) assert(n === null || n >= 3);
    for (const celda of apartado.celdas) {
      assert(celda.numerador === null || celda.numerador >= 3);
      assert(celda.sin_dato === null || celda.sin_dato >= 3);
      if (celda.numerador === null) ocultas++;
    }
  }
  assert(ocultas > 0);
  await consulta.getByRole('note').filter({ hasText: 'proteger a las familias' }).waitFor();
  await captura(consulta, 'consulta', 1440); await captura(consulta, 'consulta', 390);
  const word = consulta.waitForResponse(r => new URL(r.url()).pathname.endsWith('/word'));
  const descarga = consulta.waitForEvent('download');
  await consulta.getByRole('button', { name: 'Descargar la Memoria en Word', exact: true }).click();
  const wordRespuesta = await word; assert.equal(wordRespuesta.status(), 200, wordRespuesta.status() === 200 ? undefined : await wordRespuesta.text());
  assert((await descarga).suggestedFilename().endsWith('.docx'));

  const coordinacion = await entrar('ines.moreno');
  const guardado = await coordinacion.evaluate(async () => (await fetch('/api/coordinacion/smi')).json());
  const primera = await sesion(coordinacion); const segunda = await sesion(coordinacion);
  assert.equal(primera.caduca, segunda.caduca, 'La lectura pasiva renueva actividad');
  await prepararTiempo(coordinacion, 'inactividad');
  const aviso = await sesion(coordinacion);
  await coordinacion.getByRole('heading', { name: 'Tu sesión va a terminar' }).waitFor();
  await coordinacion.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  await coordinacion.waitForFunction(() => { const r = document.querySelector('.seg1-aviso')?.getBoundingClientRect(); return r && r.top >= 0 && r.bottom <= innerHeight; });
  await captura(coordinacion, 'sesion', 1440); await captura(coordinacion, 'sesion', 390);
  await coordinacion.locator('.persona-conectada > summary').click();
  await coordinacion.getByRole('button', { name: 'Cerrar sesión', exact: true }).click({ trial: true });
  await coordinacion.locator('.persona-conectada > summary').press('Escape');
  await coordinacion.evaluate(axe);
  const accesibilidad = await coordinacion.evaluate(async () => (await window.axe.run(document.querySelector('.seg1-aviso'), { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21aa'] } })).violations);
  assert.deepEqual(accesibilidad, [], 'Aviso de sesión: incumplimientos automáticos de accesibilidad');
  await coordinacion.getByRole('button', { name: 'Continuar sesión' }).focus();
  const continuada = coordinacion.waitForResponse(r => r.url() === base + '/api/sesion' && r.request().method() === 'PUT');
  await coordinacion.keyboard.press('Enter'); assert.equal((await continuada).status(), 200);
  await coordinacion.getByRole('heading', { name: 'Tu sesión va a terminar' }).waitFor({ state: 'hidden' });
  assert(Date.parse((await sesion(coordinacion)).caduca) > Date.parse(aviso.caduca));
  await prepararTiempo(coordinacion, 'maxima');
  const maxima = await sesion(coordinacion); const renovada = await sesion(coordinacion, 'PUT');
  assert.equal(renovada.caduca, maxima.maxima, 'Continuar desplaza el límite absoluto');
  await coordinacion.getByRole('button', { name: 'Volver a entrar', exact: true }).waitFor();
  await captura(coordinacion, 'maxima', 390);
  // Un formulario pendiente no puede bloquear la caducidad con beforeunload.
  await prepararTiempo(coordinacion, 'normal'); await sesion(coordinacion);
  await coordinacion.goto(base + '/coordinacion/smi');
  await coordinacion.locator('.smi-tabla input').first().fill('1327.00');
  await coordinacion.locator('.persona-conectada > summary').click();
  await coordinacion.getByRole('button', { name: 'Cerrar sesión', exact: true }).click();
  await coordinacion.locator('#aviso-cambios').waitFor();
  let dialogos = 0; coordinacion.on('dialog', async d => { dialogos++; await d.dismiss(); });
  await prepararTiempo(coordinacion, 'inactividad'); await sesion(coordinacion);
  await prepararTiempo(coordinacion, 'caducada');
  // El reloj sólo acelera el temporizador de interfaz. La denegación la hace
  // el servidor contra una sesión realmente vencida en PostgreSQL.
  await coordinacion.clock.install(); await coordinacion.clock.fastForward(95000);
  await coordinacion.locator('#cuenta-ad').waitFor();
  assert.equal(new URL(coordinacion.url()).pathname, '/entrar');
  assert.equal(dialogos, 0, 'La caducidad espera una confirmación de navegación');
  assert.equal(await coordinacion.locator('#aviso-cambios').count(), 0, 'La entrada conserva una acción pendiente anterior');
  await coordinacion.getByRole('alert').filter({ hasText: 'guardado' }).waitFor();
  await captura(coordinacion, 'caducada', 390);
  await coordinacion.locator('#cuenta-ad').fill('ines.moreno'); await coordinacion.locator('#contrasena-ad').fill(claves['ines.moreno']);
  await coordinacion.locator('button[type=submit]').click(); await coordinacion.locator('#cuenta-ad').waitFor({ state: 'detached' });
  const recuperado = await coordinacion.evaluate(async () => (await fetch('/api/coordinacion/smi')).json());
  assert.deepEqual(recuperado, guardado, 'Los datos guardados cambian tras caducar');
  await consulta.setViewportSize({ width: 1024, height: 1000 });
  await consulta.locator('.persona-conectada > summary').click(); await consulta.locator('.preferencias > summary').click();
  await consulta.locator('#tema').selectOption('alto-contraste');
  await consulta.waitForFunction(() => document.querySelector('link[href="/api/temas/alto-contraste/tokens"]')?.sheet);
  await consulta.locator('#idioma').selectOption('en');
  await consulta.waitForFunction(() => document.documentElement.lang === 'en');
  await consulta.keyboard.press('Escape'); await consulta.keyboard.press('Escape');
  await consulta.getByRole('note').filter({ hasText: 'To protect families' }).waitFor();
  await consulta.setViewportSize({ width: 360, height: 1000 });
  await consulta.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth);
  assert.deepEqual(errores, []);
  console.log(`SEG1-CHROME-OK Consulta API/Word (${ocultas} celdas ocultas); sesión pasiva, aviso, teclado, continuación, límite absoluto y caducidad; capturas 1440/390`);
} finally { await navegador.close(); }
