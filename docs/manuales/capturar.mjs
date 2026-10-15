// Capturas de MANETF sobre la demostración local del propio worktree.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFile, mkdir } from 'node:fs/promises';
import { chromium, baseLocal, restringir } from '../../scripts/recorridos/comun.mjs';

const base = baseLocal(process.argv[2]);
// Comprueba el puerto de la demo de este worktree antes de introducir su cuenta.
const esperado = execFileSync('python3', ['-B', '-c',
  "import importlib.util;s=importlib.util.spec_from_file_location('demo','despliegue/demo/demo.py');d=importlib.util.module_from_spec(s);s.loader.exec_module(d);print(d.URL)"],
  { cwd: new URL('../../', import.meta.url), encoding: 'utf8' }).trim();
assert.equal(new URL(base).port, new URL(esperado).port, 'Puerto ajeno a la demo del worktree');
const claves = JSON.parse(await readFile(new URL('../../despliegue/demo/.local/usuarios.json', import.meta.url)));
const destino = new URL('capturas/', import.meta.url);
await mkdir(destino, { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
let capturas = 0;
try {
  for (const [cuenta, perfil, ruta] of [
    ['marta.cabrera', 'equipo', '/equipo'],
    ['ines.moreno', 'coordinacion', '/coordinacion'],
  ]) {
    const contexto = await navegador.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'es-ES' });
    await restringir(contexto, base);
    const pagina = await contexto.newPage();
    const errores = [];
    pagina.on('pageerror', e => errores.push(e.message));
    await pagina.goto(base);
    await pagina.locator('#cuenta-ad').fill(cuenta);
    await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
    await pagina.locator('button[type=submit]').click();
    await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
    await pagina.goto(base + ruta);
    await pagina.getByRole('heading', { level: 1 }).waitFor();
    await pagina.locator(perfil === 'equipo' ? 'main a[href*="id="]' : 'main a[href*="/derivaciones/"]').first().waitFor();
    await pagina.evaluate(() => document.fonts.ready);
    for (const ancho of [1440, 390]) {
      await pagina.setViewportSize({ width: ancho, height: ancho === 390 ? 844 : 1000 });
      await pagina.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
      await pagina.screenshot({ path: new URL(`manetf-${perfil}-${ancho}.png`, destino).pathname, fullPage: false });
      capturas++;
    }
    if (perfil === 'equipo') {
      await pagina.locator('main a[href*="id="]').first().click();
      await pagina.getByRole('tab', { name: 'Familia', exact: true }).click();
      await pagina.getByRole('button', { name: 'Guardar datos familiares', exact: true }).waitFor();
      console.log('PESTAÑAS: ' + (await pagina.getByRole('tab').allTextContents()).join(', '));
      for (const ancho of [1440, 390]) {
        await pagina.setViewportSize({ width: ancho, height: ancho === 390 ? 844 : 1000 });
        await pagina.evaluate(() => document.fonts.ready);
        if (ancho === 390) {
          await pagina.getByRole('tablist').evaluate(el => el.scrollIntoView({ block: 'start', behavior: 'instant' }));
        } else {
          await pagina.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
        }
        await pagina.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
        await pagina.screenshot({ path: new URL(`manetf-ficha-${ancho}.png`, destino).pathname, fullPage: false });
        capturas++;
      }
    }
    if (perfil === 'equipo') {
      await pagina.setViewportSize({ width: 1440, height: 1000 });
      await pagina.goto(base + '/equipo/tabla-spaf');
      await pagina.getByRole('heading', { level: 1 }).waitFor();
      await pagina.getByRole('columnheader', { name: 'Municipio', exact: true }).waitFor();
      await pagina.getByRole('columnheader', { name: 'Municipio', exact: true }).evaluate(el =>
        el.scrollIntoView({ block: 'center', inline: 'center', behavior: 'instant' }));
      await pagina.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
      await pagina.screenshot({ path: new URL('manetf-spaf-1440.png', destino).pathname });
      capturas++;
    }
    assert.deepEqual(errores, [], 'Errores de JavaScript');
    await contexto.close();
  }
  console.log(`MANETF-CAPTURAS-OK: ${capturas} capturas, dos perfiles, Chrome del sistema, 1440 y 390 px`);
} finally {
  await navegador.close();
}
