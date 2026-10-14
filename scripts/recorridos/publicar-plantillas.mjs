// Publica una versión nueva; conserva el historial y la activación confirmada.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { sesionVideo } from './sesion-video.mjs';
import { pulsar, apuntar, escribir, elegir, guardar } from './comun.mjs';

export async function grabarPlantillas(opciones) {
  const { guion, dir, referencias } = opciones;
  const c = guion.config;
  const origen = 'idiomas/es/plantillas/entrada/aceptacion/v1.1.0.docx';
  assert.equal(c.plantilla, origen); assert.equal(c.tipo, 'entrada/aceptacion'); assert.equal(c.idioma, 'es');
  assert.match(c.marcador, /^[a-z_]{1,32}$/);
  for (const nombre of [c.archivo_valido, c.archivo_invalido]) assert.match(nombre, /^[a-z-]+\.docx$/);
  const buena = readFileSync(new URL(`../../${origen}`, import.meta.url));
  const desconocida = execFileSync('python3', ['-B', '-c', `import sys,zipfile,io
z=zipfile.ZipFile(sys.argv[1]); b=io.BytesIO()
with zipfile.ZipFile(b,'w',zipfile.ZIP_DEFLATED) as out:
 for f in z.infolist():
  contenido=z.read(f.filename)
  if f.filename=='word/document.xml':
   assert b'</w:body>' in contenido
   contenido=contenido.replace(b'</w:body>', ('<w:p><w:r><w:t>{{'+sys.argv[2]+'}}</w:t></w:r></w:p></w:body>').encode())
  out.writestr(f,contenido)
sys.stdout.buffer.write(b.getvalue())`, new URL(`../../${origen}`, import.meta.url).pathname, c.marcador]);
  writeFileSync(resolve(dir, c.archivo_valido), buena); writeFileSync(resolve(dir, c.archivo_invalido), desconocida);
  const s = await sesionVideo(opciones, ['plantillas']);
  const { page, t, boton, respuesta, escena, leer } = s;
  const estado = { modo: 'demo', recorrido: 'plantillas', pasos: 0, rechazo: false, subida: false, activada: false };
  const listar = () => leer('/api/admin/plantillas');
  const delDocumento = v => v.plantilla.Tipo === c.tipo && v.plantilla.Idioma === c.idioma;
  const tarjeta = () => page.locator('.plantillas-tarjeta').filter({ has: page.getByRole('heading', { name: t('plantillas.tipo.entrada.aceptacion'), exact: true }) })
    .filter({ has: page.getByText(new Intl.DisplayNames(['es'], { type: 'language' }).of(c.idioma), { exact: true }) });
  let original, nueva, version, actor;
  try {
    await escena('abrir', [async () => {
      await s.entrar('administracion'); actor = (await leer('/api/sesion')).usuario?.id; assert(actor);
    }, async () => {
      await pulsar(page.locator('a[href="/admin/plantillas"]').first()); await tarjeta().waitFor();
      const versiones = await listar(); original = versiones.find(v => delDocumento(v) && v.activa); assert(original);
      let menor = 0; while (versiones.some(v => delDocumento(v) && v.plantilla.Version === `3.${menor}.0`)) menor++;
      version = `3.${menor}.0`; estado.version = version; estado.original = original.plantilla.ID;
    }]);
    await escena('historial', [() => apuntar(tarjeta().locator('.plantillas-activa')), () => pulsar(tarjeta().locator('summary'))]);
    await escena('invalida', [() => pulsar(boton('plantillas.subir')), async () => {
      await elegir(page.locator('#plantillas-tipo'), c.tipo); await elegir(page.locator('#plantillas-idioma'), c.idioma);
      await escribir(page.locator('#plantillas-version'), version);
    }, async () => {
      await apuntar(page.locator('#plantillas-archivo')); await page.locator('#plantillas-archivo').setInputFiles(resolve(dir, c.archivo_invalido));
    }, async () => {
      const rechazo = respuesta('/api/admin/plantillas/versiones'); await pulsar(boton('plantillas.subir_revisar'));
      const r = await rechazo; assert.equal(r.status(), 422);
      await page.getByText(t('plantillas.marcador_sobra').replace('{valor}', c.marcador), { exact: true }).waitFor();
      assert(!(await page.locator('#plantillas-error').innerText()).includes(t('plantillas.marcador_falta').split('{valor}')[0]));
      estado.rechazo = true;
    }]);
    await escena('aviso', [async () => {
      await apuntar(page.getByText(t('plantillas.marcador_sobra').replace('{valor}', c.marcador), { exact: true }));
    }, async () => {
      assert.equal((await listar()).find(v => delDocumento(v) && v.activa).plantilla.ID, original.plantilla.ID);
      assert(!(await listar()).some(v => delDocumento(v) && v.plantilla.Version === version));
      assert.equal(await page.locator('#plantillas-version').inputValue(), version); await apuntar(page.locator('#plantillas-version'));
    }]);
    await escena('buena', [async () => {
      await apuntar(page.locator('#plantillas-archivo')); await page.locator('#plantillas-archivo').setInputFiles(resolve(dir, c.archivo_valido));
    }, async () => {
      const subida = respuesta('/api/admin/plantillas/versiones'); await pulsar(boton('plantillas.subir_revisar'));
      assert.equal((await subida).status(), 201); estado.subida = true;
      await page.getByRole('status').filter({ hasText: t('plantillas.subida') }).waitFor();
      const versiones = await listar(); nueva = versiones.find(v => delDocumento(v) && v.plantilla.Version === version);
      assert(nueva && !nueva.activa); assert.equal(nueva.antivirus, 'limpio'); assert.equal(nueva.subida_por, actor);
      assert.equal(versiones.find(v => delDocumento(v) && v.activa).plantilla.ID, original.plantilla.ID);
      estado.nueva = nueva.plantilla.ID;
    }, () => apuntar(page.getByRole('status').filter({ hasText: t('plantillas.subida') }))]);
    await escena('activar', [async () => {
      await pulsar(boton('plantillas.cerrar'));
      const historia = tarjeta().locator('details');
      if (!await historia.evaluate(n => n.open)) await pulsar(historia.locator('summary'));
      await apuntar(tarjeta().getByRole('button', { name: t('plantillas.activar_version').replace('{valor}', version), exact: true }));
    }, async () => {
      await pulsar(tarjeta().getByRole('button', { name: t('plantillas.activar_version').replace('{valor}', version), exact: true }));
      await page.getByRole('dialog').waitFor();
    }, async () => {
      const activacion = respuesta(`/api/admin/plantillas/${nueva.plantilla.ID}/activar`);
      await pulsar(page.getByRole('dialog').getByRole('button', { name: t('plantillas.confirmar'), exact: true }));
      assert.equal((await activacion).status(), 200); estado.activada = true;
      await page.getByRole('status').filter({ hasText: t('plantillas.activada') }).waitFor();
    }]);
    await escena('autoria', [async () => {
      const versiones = await listar(); const actual = versiones.find(v => v.plantilla.ID === nueva.plantilla.ID);
      assert(actual.activa && actual.activaciones.length === 1);
      assert.equal(actual.activaciones[0].por, actor); assert(Number.isFinite(Date.parse(actual.activaciones[0].en)));
      assert(versiones.some(v => v.plantilla.ID === original.plantilla.ID && !v.activa)); estado.activacion = actual.activaciones[0];
      await apuntar(tarjeta().locator('.plantillas-activa')); assert.equal(await tarjeta().locator('.plantillas-activa strong').innerText(), version);
    }, async () => {
      const bloque = tarjeta().locator('li').filter({ has: page.getByRole('heading', { name: `${t('plantillas.version')} ${version} ${t('plantillas.activa')}`, exact: true }) });
      await apuntar(bloque.locator('p').filter({ hasText: t('plantillas.activada_por') }));
      const usuarios = await leer('/api/admin/usuarios'); const nombre = usuarios.find(u => u.id === actor)?.nombre; assert(nombre);
      assert((await bloque.innerText()).includes(nombre));
    }]);
    estado.pasos = referencias.length; await s.terminar();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`PLANTILLAS-OK: ${estado.pasos} pasos; marcador rechazado=${estado.rechazo}; versión=${version}; activada=${estado.activada}; autor y fecha comprobados`);
}
