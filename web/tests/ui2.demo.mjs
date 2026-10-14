// Recorrido de presentación de UI2 contra la demostración local desechable.
import assert from 'node:assert/strict';
import { readFile, readdir, mkdir } from 'node:fs/promises';
import { chromium } from 'playwright-core';

const base = process.env.ETF_UI2_URL || 'http://127.0.0.1:18089';
assert.match(base, /^http:\/\/127\.0\.0\.1:\d+$/);
const raiz = new URL('../../', import.meta.url);
const claves = JSON.parse(await readFile(new URL('despliegue/demo/.local/usuarios.json', raiz)));
const textos = {};
for (const idioma of ['es', 'en']) {
  textos[idioma] = {};
  for (const fichero of await readdir(new URL(`idiomas/${idioma}/textos/`, raiz))) {
    if (fichero.endsWith('.json')) Object.assign(textos[idioma], JSON.parse(await readFile(new URL(`idiomas/${idioma}/textos/${fichero}`, raiz))));
  }
}
const evidencia = new URL('docs/capturas/', raiz);
await mkdir(evidencia, { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
let capturas = 0;
const errores = [];
async function preferencias(pagina, idioma, tema) {
  await pagina.locator('.persona-conectada > summary').click();
  await pagina.locator('.preferencias > summary').click();
  if (await pagina.locator('#idioma').inputValue() !== idioma) {
    await pagina.locator('#idioma').selectOption(idioma);
    await pagina.waitForFunction((l) => document.documentElement.lang === l, idioma);
    await pagina.locator('.persona-conectada > summary').click();
    await pagina.locator('.preferencias > summary').click();
  }
  await pagina.locator('#tema').selectOption(tema);
  await pagina.waitForFunction((tema) => document.querySelector(`link[href="/api/temas/${tema}/tokens"]`)?.sheet, tema);
  await pagina.locator('#tema').press('Escape');
  await pagina.locator('.persona-conectada > summary').focus();
  await pagina.keyboard.press('Escape');
  await pagina.evaluate(() => document.fonts.ready);
  const publicados = await pagina.evaluate(async (codigo) => (await fetch(`/api/idiomas/${codigo}/textos`)).json(), idioma);
  for (const clave of ['catalogo.version_vigente', 'catalogo.funcion_cargo']) assert.equal(publicados[clave], textos[idioma][clave], clave);
}
async function geometria(pagina) {
  await pagina.waitForFunction(() => document.querySelector('.cabecera-app')?.dataset.compacto === String(document.documentElement.clientWidth < parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu')) * 4));
  const medida = await pagina.evaluate(() => ({ ruta: location.pathname, ancho: innerWidth, desbordes: [...document.querySelectorAll('main *')].filter((e) => e.getBoundingClientRect().right > innerWidth).map((e) => ({ etiqueta: e.tagName, clase: e.className, ancho: e.getBoundingClientRect().width, borde: e.getBoundingClientRect().right })).slice(0, 8) }));
  assert.ok(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth), JSON.stringify(medida));
  assert.equal(await pagina.locator('h1').count(), 1);
  assert.equal(await pagina.locator('main').count(), 1);
  assert.equal(await pagina.evaluate(() => localStorage.length + sessionStorage.length), 0);
  for (const control of await pagina.locator('main button, main input, main select, main summary').all()) {
    if (await control.isVisible()) assert.ok(await control.evaluate((e) => e.getBoundingClientRect().height >= 44));
  }
}
try {
  for (const cuenta of ['elena.salas', 'ines.moreno']) {
    const contexto = await navegador.newContext({ viewport: { width: 1440, height: 900 } });
    const pagina = await contexto.newPage();
    pagina.on('pageerror', (e) => errores.push(e.message));
    await pagina.goto(base);
    await pagina.locator('#cuenta-ad').fill(cuenta);
    await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
    const sesion = pagina.waitForResponse((r) => r.url() === base + '/api/sesion' && r.request().method() === 'POST');
    await pagina.locator('button[type=submit]').click();
    assert.equal((await sesion).status(), 200);
    await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
    if (cuenta === 'ines.moreno') {
      await pagina.goto(base + '/coordinacion/smi');
      await pagina.locator('.smi-tabla input').first().waitFor();
      const anteriores = await pagina.evaluate(async () => (await fetch('/api/coordinacion/smi')).json());
      const anio = anteriores[0].anio;
      if (anteriores[0].importe_mensual_centimos !== 132500) {
        const importe = pagina.locator(`#smi-${anio}`);
        await importe.fill('1325.00');
        const publicada = pagina.waitForResponse((r) => r.url() === base + '/api/coordinacion/smi' && r.request().method() === 'PUT');
        await pagina.locator('tr').filter({ has: importe }).getByRole('button').click();
        const respuesta = await publicada;
        assert.equal(respuesta.status(), 200);
        const fila = await respuesta.json();
        assert.equal(fila.version, anteriores[0].version + 1);
        assert.equal(fila.importe_mensual_centimos, 132500);
        assert.equal(fila.importe_anual_centimos, 1855000);
        await pagina.locator('tr').filter({ has: importe }).getByRole('button').waitFor();
        await pagina.waitForFunction((anio) => document.querySelector(`#smi-${anio}`).closest('tr').querySelector('button').disabled, anio);
      }
    }
    const pantallas = cuenta === 'ines.moreno' ? [['smi', '/coordinacion/smi']]
      : [['usuarios', '/admin/usuarios'], ['catalogos', '/admin/catalogos'], ['reglas', '/admin/reglas'], ['tramos', '/admin/reglas?tipo=tramos'], ['firmantes', '/admin/firmantes']];
    for (const [nombre, ruta] of pantallas) {
      for (const tema of ['expedientes-etf', 'alto-contraste']) for (const width of [1440, 390]) {
        await pagina.setViewportSize({ width, height: 900 });
        await pagina.goto(base + ruta);
        await pagina.locator('h1').waitFor();
        await pagina.waitForFunction(() => [...document.querySelectorAll('main [role=status]')].every((e) => !e.textContent));
        await preferencias(pagina, 'es', tema);
        await geometria(pagina);
        assert.equal(await pagina.locator('main [role=alert]').count(), 0);
        if (nombre === 'catalogos') {
          const datos = await pagina.evaluate(async () => { const respuesta = await fetch('/api/admin/catalogos/listas'); if (!respuesta.ok) throw new Error(`UI2: catálogo HTTP ${respuesta.status}`); return respuesta.json(); });
          const esperadas = datos.datos.opciones.map((o) => datos.textos.es?.[`${o.lista}.${o.codigo}`] ?? textos.es['catalogo.etiqueta_pendiente']);
          assert.deepEqual(await pagina.locator('.fila-admin strong').allTextContents(), esperadas);
          assert.equal(await pagina.locator('main button:not(.boton-secundario)').count(), 1);
        }
        if (nombre === 'smi' && width === 390) {
          for (const control of await pagina.locator('.smi-tabla input, .smi-tabla button').all()) assert.ok(await control.evaluate((e) => e.clientWidth > 100));
        }
        await pagina.screenshot({ path: new URL(`ui2-${nombre}-${width}-${tema}.png`, evidencia).pathname });
        capturas++;
      }
      await preferencias(pagina, 'en', 'expedientes-etf');
      await geometria(pagina);
      if (nombre === 'catalogos') {
        const datos = await pagina.evaluate(async () => { const respuesta = await fetch('/api/admin/catalogos/listas'); if (!respuesta.ok) throw new Error(`UI2: catálogo HTTP ${respuesta.status}`); return respuesta.json(); });
        assert.deepEqual(await pagina.locator('.fila-admin strong').allTextContents(), datos.datos.opciones.map((o) => datos.textos.en?.[`${o.lista}.${o.codigo}`] ?? datos.textos.es?.[`${o.lista}.${o.codigo}`] ?? textos.en['catalogo.etiqueta_pendiente']));
        await pagina.locator('.fila-admin button').first().click();
        assert.equal(await pagina.locator('.identificadores-catalogo').getAttribute('open'), null);
        await pagina.locator('input[id$=texto_es]').focus();
        await pagina.keyboard.press('Tab');
        assert.equal(await pagina.locator('input[id$=texto_en]').evaluate((e) => document.activeElement === e), true);
      }
      for (const width of [1024, 360, 320]) {
        await pagina.setViewportSize({ width, height: 900 });
        await geometria(pagina);
      }
    }
    if (cuenta === 'elena.salas') {
      await pagina.setViewportSize({ width: 1440, height: 900 });
      for (const ruta of ['/admin/paquetes', '/admin/idiomas', '/admin/temas']) {
        await pagina.goto(base + ruta); await pagina.locator('.paquetes').waitFor();
        assert.equal(await pagina.locator('#menu-principal a[href="/admin/paquetes"]').count(), 1);
        assert.equal(await pagina.locator('#menu-principal a[href="/admin/idiomas"], #menu-principal a[href="/admin/temas"]').count(), 0);
        assert.equal(await pagina.locator('#menu-principal a[aria-current=page]').getAttribute('href'), '/admin/paquetes');
      }
    }
    await contexto.close();
  }
  assert.deepEqual(errores, []);
  console.log(`UI2-DEMO-OK: 6 pantallas; ${capturas} capturas; 1440/1024/390/360/320 px; es/en; dos temas; etiquetas, menú, teclado y reflujo; cuentas de Administración y Coordinación reales de la demo.`);
} finally { await navegador.close(); }
