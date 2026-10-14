// Recorrido visual de UI1 con API simulada y datos inventados; no valida AD ni RLS.
import { contratoServidor } from './contrato-spa.mjs';
import { chromium } from 'playwright-core';
import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { readFileSync, readdirSync, mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { join, resolve } from 'node:path';

const raiz = fileURLToPath(new URL('../../', import.meta.url));
const leer = (ruta) => readFileSync(join(raiz, ruta));
const json = (ruta) => JSON.parse(leer(ruta));
const textos = (idioma) => Object.assign({}, ...readdirSync(join(raiz, `idiomas/${idioma}/textos`), { recursive: true })
  .filter((ruta) => ruta.endsWith('.json')).map((ruta) => json(`idiomas/${idioma}/textos/${ruta}`)));
const es = textos('es');
const en = textos('en');
const cuentas = [
  { nombre: 'Inés Moreno Castillo', id: 'persona-ines', cuenta: 'ines.moreno', iniciales: 'I. M. C.', estado: 'pendiente_alta', version: 1, perfiles: [], equipo: null },
  { nombre: 'Marta López Rivas', id: 'persona-marta', cuenta: 'marta.lopez', estado: 'activo', version: 1, perfiles: [{ perfil: 'coordinacion', desde: '2026-01-01T00:00:00Z', hasta: null }, { perfil: 'administracion', desde: '2026-01-01T00:00:00Z', hasta: null }], equipo: null },
  { nombre: 'Elena Guerrero Molina', id: 'persona-elena', cuenta: 'elena.guerrero', estado: 'activo', version: 1, perfiles: [{ perfil: 'profesional_etf', desde: '2026-01-01T00:00:00Z', hasta: null }], equipo: { equipo_id: 'equipo-vega', puesto: 'ts', desde: '2026-01-01T00:00:00Z', hasta: null } },
];
const pantallas = contratoServidor().filter((p) => p.Pantalla && p.ClaveTitulo)
  .map((p) => ({ ruta: p.Patron, clave: p.ClaveTitulo, perfiles: p.Perfiles }));
const administradora = { estado: 'activo', csrf: 'testigo-inventado', usuario: { id: 'persona-marta', cuenta: 'marta.lopez', nombre: 'Marta López Rivas' }, perfiles: ['coordinacion', 'administracion'], pantallas };
const catalogo = { datos: { version: { numero: 1, vigente_desde: '2026-01-01' }, opciones: [{ lista: 'subprograma', codigo: 'riesgo', orden: 1, activa: true, admite_texto: false }], plazos: [], tramos: [] }, firmantes: [{ id: 'firmante-luis', nombre: 'Luis García Salas', tipo: 'firmante', rol: 'coordinador', cargo_codigo: 'coordinador', organismo_codigo: 'diputacion', direccion: '', vigente_desde: '2026-01-01', vigente_hasta: null, version: 1, cargo_textos: { es: 'Coordinador provincial', en: 'Provincial coordinator' } }], referencias: { es: 1, en: 1 }, textos: { es: { 'subprograma.riesgo': 'Riesgo social' }, en: { 'subprograma.riesgo': 'Social risk' } } };
const temas = ['expedientes-etf', 'alto-contraste'].map((nombre) => json(`temas/${nombre}/tema.json`));
const html = leer('web/dist/index.html').toString().replaceAll('{{.Idioma}}', 'es').replaceAll('{{.Tema}}', 'expedientes-etf')
  .replaceAll('{{.Titulo}}', es['comun.titulo']).replaceAll('{{.ErrorCarga}}', es['comun.error_carga']).replaceAll('{{.Reintentar}}', es['comun.reintentar']);

const servidor = createServer((peticion, respuesta) => {
  const ruta = new URL(peticion.url, 'http://127.0.0.1').pathname;
  if (ruta.startsWith('/assets/') && !ruta.includes('..')) {
    respuesta.setHeader('Content-Type', ruta.endsWith('.js') ? 'text/javascript' : 'text/css');
    respuesta.end(leer(`web/dist${ruta}`));
  } else if (ruta.endsWith('.woff2')) {
    const partes = ruta.split('/');
    if (!temas.some((tema) => tema.identificador === partes[3]) || !['public-sans.woff2', 'source-serif-4.woff2'].includes(partes.at(-1))) {
      respuesta.writeHead(404).end(); return;
    }
    respuesta.setHeader('Content-Type', 'font/woff2');
    respuesta.end(leer(`temas/${partes[3]}/fuentes/${partes.at(-1)}`));
  } else { respuesta.setHeader('Content-Type', 'text/html'); respuesta.end(html); }
});
await new Promise((res) => servidor.listen(0, '127.0.0.1', res));
const base = `http://127.0.0.1:${servidor.address().port}`;
const evidencia = process.env.ETF_EVIDENCIA_RUTA && resolve(process.env.ETF_EVIDENCIA_RUTA);
if (evidencia) mkdirSync(evidencia, { recursive: true });
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
let capturas = 0;

async function crear(width, acceso = { estado: 'sin_sesion', csrf: 'testigo-inventado' }) {
  const contexto = await navegador.newContext({ viewport: { width, height: 900 } });
  const pagina = await contexto.newPage();
  const errores = [];
  const peticiones = [];
  pagina.on('pageerror', (error) => errores.push(error.message));
  await pagina.route(`${base}/api/**`, async (route) => {
    const solicitud = route.request();
    const ruta = new URL(solicitud.url()).pathname;
    peticiones.push(ruta);
    const enviar = (body, status = 200) => route.fulfill({ status, json: body });
    if (ruta === '/api/sesion') {
      if (solicitud.method() === 'POST') acceso = administradora;
      if (solicitud.method() === 'DELETE') acceso = { estado: 'sin_sesion', csrf: 'testigo-inventado' };
      return enviar(acceso);
    }
    if (ruta === '/api/idiomas') return enviar(['es', 'en'].map((i) => json(`idiomas/${i}/idioma.json`)));
    if (ruta === '/api/temas') return enviar(temas);
    if (ruta === '/api/idiomas/es/textos') return enviar(es);
    if (ruta === '/api/idiomas/en/textos') return enviar(en);
    const tema = temas.find((t) => ruta === `/api/temas/${t.identificador}/tokens`);
    if (tema) {
      const tokens = json(`temas/${tema.identificador}/tokens.json`);
      const css = ':root{' + Object.entries(tokens).map(([k, v]) => `--${k.replaceAll('.', '-')}:${v};`).join('') + '}'
        + ['texto', 'titulo'].map((tipo) => `@font-face{font-family:${tokens[`fuente.${tipo}`].split(',')[0]};src:url('/api/temas/${tema.identificador}/recursos/${tema.fuentes[tipo]}');font-weight:400 700;}`).join('');
      return route.fulfill({ contentType: 'text/css', body: css });
    }
    if (ruta.endsWith('.woff2')) return route.continue();
    if (ruta === '/api/equipo/expedientes') return enviar({ equipo_clave: 'equipo.02', filas: [], hay_mas: false });
    if (ruta.startsWith('/api/equipo/etiquetas/')) return enviar(json('idiomas/es/catalogos/ubicaciones.json'));
    if (ruta === '/api/admin/usuarios') return enviar(cuentas);
    if (ruta === '/api/admin/catalogos/asignacion_municipios/datos') return enviar({ equipos: [{ id: 'equipo-vega', codigo: '02' }] });
    if (ruta === '/api/coordinacion/smi') return enviar([{ anio: 2025, importe_mensual_centimos: 118400, importe_anual_centimos: 1657600, version: 1 }, { anio: 2026, importe_mensual_centimos: null, importe_anual_centimos: null, version: 0 }]);
    if (['/api/admin/catalogos/listas', '/api/admin/firmantes', '/api/admin/reglas'].includes(ruta)) return enviar(catalogo);
    return enviar({ error: { clave: 'identidad.acceso_denegado' } }, 403);
  });
  return { contexto, pagina, errores, peticiones };
}

async function abrirPersona(pagina, idioma = 'es') {
  const detalle = pagina.locator('.persona-conectada');
  if (await detalle.count() && !await detalle.evaluate((e) => e.open)) await detalle.locator(':scope > summary').click();
}
async function preferencias(pagina, idioma = 'es') {
  await abrirPersona(pagina, idioma);
  const resumen = pagina.locator('.preferencias summary');
  assert.equal(await resumen.innerText(), (idioma === 'es' ? es : en)['comun.preferencias']);
  if (!await pagina.locator('.preferencias').evaluate((e) => e.open)) await resumen.click();
}
async function seleccionarTema(pagina, tema) {
  await preferencias(pagina);
  await pagina.getByLabel(es['comun.tema'], { exact: true }).selectOption(tema);
  await pagina.waitForFunction((seleccionado) => getComputedStyle(document.body).backgroundColor === seleccionado,
    tema === 'alto-contraste' ? 'rgb(255, 255, 255)' : 'rgb(244, 242, 236)');
  await pagina.keyboard.press('Escape');
  assert.equal(await pagina.locator('.preferencias > summary').evaluate((e) => document.activeElement === e), true);
  if (await pagina.locator('.persona-conectada').count()) {
    await pagina.locator('.persona-conectada > summary').focus(); await pagina.keyboard.press('Escape');
  }
}
async function sinDesborde(pagina) {
  await pagina.waitForFunction(() => document.querySelector('.cabecera-app').dataset.compacto === String(document.documentElement.clientWidth < parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu')) * 4));
  assert.equal(await pagina.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, JSON.stringify(await pagina.evaluate(() => ({ ancho: innerWidth, elementos: [...document.querySelectorAll('body *')].filter((e) => e.getBoundingClientRect().right > innerWidth).map((e) => e.tagName + '.' + e.className).slice(0, 8) }))));
  assert.equal(await pagina.locator('main').count(), 1);
  assert.equal(await pagina.locator('h1').count(), 1);
  assert.equal(await pagina.evaluate(() => localStorage.length + sessionStorage.length), 0);
}

try {
  const escenas = [
    ['entrada', '/entrar', 'identidad.entrar_titulo'], ['pendiente', '/entrar', 'identidad.pendiente_titulo'],
    ['smi', '/coordinacion/smi', 'smi.titulo'], ['usuarios', '/admin/usuarios', 'usuarios.titulo'], ['catalogos', '/admin/catalogos', 'catalogo.pantalla_listas'],
    ['reglas', '/admin/reglas', 'catalogo.pantalla_plazos'], ['firmantes', '/admin/firmantes', 'catalogo.pantalla_firmantes'],
    ['paquetes', '/admin/paquetes', 'paquetes.titulo'], ['idiomas', '/admin/idiomas', 'paquetes.titulo'], ['temas', '/admin/temas', 'paquetes.titulo'],
  ];
  for (const width of [1280, 390]) for (const tema of ['expedientes-etf', 'alto-contraste']) {
    for (const [pantalla, ruta, clave] of escenas) {
      const acceso = pantalla === 'entrada' ? { estado: 'sin_sesion', csrf: 'testigo-inventado' }
        : pantalla === 'pendiente' ? { estado: 'pendiente_alta', csrf: 'testigo-inventado' } : administradora;
      const { contexto, pagina, errores } = await crear(width, acceso);
      await pagina.goto(base + ruta);
      await pagina.getByRole('heading', { level: 1, name: es[clave] }).waitFor();
      if (pantalla === 'usuarios') await pagina.getByRole('button', { name: 'Editar permisos de Inés Moreno Castillo (ines.moreno)' }).waitFor();
      if (['catalogos', 'reglas', 'firmantes'].includes(pantalla)) await pagina.waitForFunction(() => !document.querySelector('.contenido-admin [role=alert]') && !document.querySelector('.contenido-admin [role=status]').textContent);
      await seleccionarTema(pagina, tema);
      await pagina.evaluate(() => document.fonts.ready);
      await sinDesborde(pagina);
      assert.equal(await pagina.locator('.marca').innerText(), es['comun.marca']);
      assert.equal(await pagina.locator('.marca-app > span:last-child').innerText(), es['comun.titulo']);
      assert.equal(await pagina.evaluate(() => getComputedStyle(document.querySelector('.saltar-contenido')).clipPath), 'inset(100%)');
      if (width < 1024) {
        assert.ok(await pagina.locator('.cabecera-app').evaluate((e) => e.getBoundingClientRect().height <= parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--alto-cabecera')) + parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--espacio-4')) * 2));
      }
      if (acceso.estado === 'activo') {
        assert.equal(await pagina.getByRole('navigation', { name: es['usuarios.menu'] }).isVisible(), width >= 1024);
        if (width < 1024) await pagina.getByRole('button', { name: es['comun.menu'], exact: true }).click();
        assert.equal(await pagina.locator('#menu-principal a').count(), pantallas.length - 2);
        assert.equal(await pagina.getByRole('link', { name: es['paquetes.titulo'], exact: true }).count(), 1);
        assert.equal(await pagina.getByRole('link', { name: es['paquetes.idiomas'], exact: true }).count(), 0);
        assert.equal(await pagina.getByRole('link', { name: es['paquetes.temas'], exact: true }).count(), 0);
        assert.equal(await pagina.locator('#menu-principal a[aria-current=page]').getAttribute('href'), ['/admin/idiomas', '/admin/temas'].includes(ruta) ? '/admin/paquetes' : ruta);
        if (width < 1024) {
          await pagina.locator('#menu-principal a').first().focus(); await pagina.keyboard.press('Escape');
          assert.equal(await pagina.getByRole('button', { name: es['comun.menu'], exact: true }).evaluate((e) => document.activeElement === e), true);
        }
        await abrirPersona(pagina);
        assert.equal(await pagina.locator('.panel-persona > strong').innerText(), 'Marta López Rivas');
        assert.equal(await pagina.getByRole('button', { name: es['identidad.salir'] }).isVisible(), true);
        await pagina.locator('.persona-conectada > summary').focus(); await pagina.keyboard.press('Escape');
      }
      if (pantalla === 'usuarios') {
        assert.equal(await pagina.getByRole('table', { name: es['usuarios.titulo'] }).count(), 1);
        assert.equal(await pagina.getByRole('columnheader').count(), 5);
        assert.equal(await pagina.getByRole('rowheader', { name: 'Inés Moreno Castillo ines.moreno', exact: true }).count(), 1);
        assert.equal(await pagina.getByText('I. M. C.', { exact: true }).count(), 0);
      }
      if (pantalla === 'catalogos') {
        assert.equal(await pagina.locator('.fila-admin strong').first().innerText(), 'Riesgo social');
        assert.equal(await pagina.locator('.fila-admin').first().innerText().then((s) => /subprograma riesgo/.test(s)), false);
        assert.equal(await pagina.locator('.contenido-admin button:not(.boton-secundario)').count(), 1);
      }
      if (pantalla === 'firmantes') assert.equal(await pagina.locator('.fila-admin strong').first().innerText(), 'Luis García Salas');
      if (pantalla === 'smi' && width < 1024) {
        assert.ok(await pagina.locator('.smi-tabla input').first().evaluate((e) => e.clientWidth > 100));
        assert.ok(await pagina.locator('.smi-tabla button').first().evaluate((e) => e.clientWidth > 100));
        assert.equal(await pagina.getByRole('table', { name: es['smi.tabla'] }).count(), 1);
      }
      await pagina.locator('main').click({ position: { x: 0, y: 0 } });
      if (evidencia) { await pagina.screenshot({ path: join(evidencia, `ui1-${pantalla}-${width}-${tema}.png`), fullPage: true }); capturas++; }
      if (pantalla === 'entrada') {
        await pagina.goto(`${base}/entrar`); await pagina.getByLabel(es['identidad.cuenta']).waitFor();
        await pagina.keyboard.press('Tab');
        const salto = pagina.getByRole('link', { name: es['comun.saltar_contenido'] });
        assert.equal(await salto.evaluate((e) => e === document.activeElement), true);
        assert.equal(await salto.evaluate((e) => getComputedStyle(e).clipPath), 'none');
        await pagina.keyboard.press('Enter');
        assert.equal(await pagina.locator('main').evaluate((e) => e === document.activeElement), true);
      }
      if (pantalla === 'pendiente') {
        assert.equal(await pagina.locator('nav a').count(), 0);
        await abrirPersona(pagina);
        await pagina.getByRole('button', { name: es['identidad.salir'] }).click();
        await pagina.getByRole('button', { name: es['identidad.entrar'], exact: true }).waitFor();
        assert.equal(new URL(pagina.url()).pathname, '/entrar');
      }
      await pagina.setViewportSize({ width: 320, height: 900 }); await sinDesborde(pagina);
      assert.equal(await pagina.locator('button, input:not([type=checkbox]), select, summary').evaluateAll((controles) => controles.filter((e) => e.getClientRects().length).every((e) => e.getBoundingClientRect().height >= 44)), true);
      assert.deepEqual(errores, []); await contexto.close();
    }
  }
  // Etiquetas con reserva y ausentes; vacío, fallo y denegación con API simulada.
  {
    const { contexto, pagina } = await crear(390, administradora);
    let estado = 'etiquetas';
    const contenido = { ...catalogo, datos: { ...catalogo.datos, opciones: [
      ...catalogo.datos.opciones,
      { lista: 'subprograma', codigo: 'reunificacion', orden: 2, activa: true, admite_texto: false },
      { lista: 'subprograma', codigo: 'pendiente_publicacion', orden: 3, activa: false, admite_texto: false },
    ] }, textos: { ...catalogo.textos, es: { ...catalogo.textos.es, 'subprograma.reunificacion': 'Reunificación' } } };
    await pagina.route(base + '/api/admin/catalogos/listas', (route) => {
      if (estado === 'error') return route.fulfill({ status: 503, json: { error: 'catalogo.no_disponible' } });
      if (estado === 'denegado') return route.fulfill({ status: 403, json: { error: 'identidad.acceso_denegado' } });
      return route.fulfill({ json: estado === 'vacio' ? { ...catalogo, datos: { ...catalogo.datos, opciones: [] } } : contenido });
    });
    await pagina.goto(base + '/admin/catalogos');
    await pagina.locator('.fila-admin').nth(2).waitFor();
    assert.deepEqual(await pagina.locator('.fila-admin strong').allTextContents(), ['Riesgo social', 'Reunificación', es['catalogo.etiqueta_pendiente']]);
    await abrirPersona(pagina); await pagina.locator('.preferencias > summary').click();
    await pagina.getByLabel(es['comun.idioma'], { exact: true }).selectOption('en');
    await pagina.getByRole('heading', { level: 1, name: en['catalogo.pantalla_listas'] }).waitFor();
    assert.deepEqual(await pagina.locator('.fila-admin strong').allTextContents(), ['Social risk', 'Reunificación', en['catalogo.etiqueta_pendiente']]);
    assert.doesNotMatch(await pagina.locator('main').innerText(), /pendiente_publicacion|subprograma\./);
    estado = 'error'; await pagina.reload();
    await pagina.getByRole('alert').getByText(es['catalogo.no_disponible']).waitFor();
    estado = 'vacio'; await pagina.getByRole('button', { name: es['comun.reintentar'], exact: true }).click();
    await pagina.getByText(es['catalogo.vacio'], { exact: true }).waitFor();
    estado = 'denegado'; await pagina.reload();
    await pagina.getByRole('alert').getByText(es['identidad.acceso_denegado']).waitFor();
    await contexto.close();
  }
  // Perfiles sin pantallas disponibles: sin enlaces inventados ni consultas de Administración.
  for (const perfil of ['profesional_etf', 'consulta']) {
    const { contexto, pagina, peticiones } = await crear(320, { estado: 'activo', csrf: 'testigo-inventado',
      usuario: { id: 'persona-ines', cuenta: 'ines.moreno', nombre: 'Inés Moreno Castillo' }, perfiles: [perfil], pantallas: [], ...(perfil === 'profesional_etf' ? { equipo_id: '1cc62546-2f3c-452b-a874-445f738f22db' } : {}) });
    await pagina.goto(base); await pagina.getByRole('heading', { name: es['identidad.acceso_titulo'] }).waitFor(); await sinDesborde(pagina);
    assert.equal(await pagina.locator('nav a').count(), 0); assert.ok(!peticiones.some((ruta) => ruta.startsWith('/api/admin/')));
    await abrirPersona(pagina); if (perfil === 'profesional_etf') await pagina.locator('.panel-persona .equipo-persona').getByText(json('idiomas/es/catalogos/ubicaciones.json')['equipo.02'], { exact: true }).waitFor();
    await contexto.close();
  }
  // Entrar → Usuarios → Catálogos → Paquetes; historial, recarga y guardia de cambios.
  const { contexto, pagina } = await crear(1280);
  await pagina.goto(`${base}/entrar`); await pagina.getByLabel(es['identidad.cuenta']).fill('marta.lopez');
  await pagina.getByLabel(es['identidad.contrasena']).fill('clave-inventada'); await pagina.getByRole('button', { name: es['identidad.entrar'], exact: true }).click();
  await pagina.getByRole('heading', { name: es['bandeja.titulo'] }).waitFor();
  await pagina.getByRole('link', { name: es['usuarios.titulo'], exact: true }).click();
  await pagina.getByRole('heading', { name: es['usuarios.titulo'] }).waitFor();
  await pagina.getByRole('button', { name: 'Editar permisos de Inés Moreno Castillo (ines.moreno)' }).click();
  await pagina.getByLabel(es['usuarios.perfil_consulta'], { exact: true }).check();
  await preferencias(pagina);
  await pagina.getByRole('button', { name: es['identidad.salir'], exact: true }).click();
  await pagina.getByRole('button', { name: es['usuarios.seguir_editando'] }).click();
  assert.equal(new URL(pagina.url()).pathname, '/admin/usuarios');
  assert.equal(await pagina.getByLabel(es['usuarios.perfil_consulta'], { exact: true }).isChecked(), true);
  await pagina.getByRole('link', { name: es['catalogo.menu_listas'], exact: true }).click();
  await pagina.getByRole('button', { name: es['usuarios.seguir_editando'] }).click();
  assert.equal(new URL(pagina.url()).pathname, '/admin/usuarios'); assert.equal(await pagina.getByLabel(es['usuarios.perfil_consulta'], { exact: true }).isChecked(), true);
  await pagina.getByRole('link', { name: es['catalogo.menu_listas'], exact: true }).click();
  await pagina.getByRole('button', { name: es['usuarios.descartar'] }).click();
  await pagina.getByRole('heading', { name: es['catalogo.pantalla_listas'] }).waitFor();
  await pagina.getByRole('button', { name: es['catalogo.editar_entrada'].replace('{nombre}', 'Riesgo social'), exact: true }).first().click();
  const borrador = pagina.locator('input[id$=texto_es]');
  await borrador.fill('Riesgo social revisado');
  await borrador.press('Tab');
  await pagina.goBack(); await pagina.getByRole('button', { name: es['usuarios.seguir_editando'] }).click();
  assert.equal(new URL(pagina.url()).pathname, '/admin/catalogos'); assert.equal(await borrador.inputValue(), 'Riesgo social revisado');
  await pagina.getByRole('link', { name: es['paquetes.titulo'], exact: true }).click();
  await pagina.getByRole('button', { name: es['usuarios.descartar'] }).click();
  await pagina.getByRole('heading', { name: es['paquetes.titulo'] }).waitFor();
  await pagina.goBack(); await pagina.getByRole('heading', { name: es['catalogo.pantalla_listas'] }).waitFor();
  await pagina.goForward(); await pagina.getByRole('heading', { name: es['paquetes.titulo'] }).waitFor();
  await pagina.reload(); await pagina.getByRole('heading', { name: es['paquetes.titulo'] }).waitFor();
  await pagina.getByRole('link', { name: es['usuarios.titulo'], exact: true }).click();
  await preferencias(pagina); await pagina.getByLabel(es['comun.idioma']).selectOption('en');
  await pagina.getByRole('heading', { name: en['usuarios.titulo'] }).waitFor(); assert.equal(await pagina.locator('html').getAttribute('lang'), 'en');
  await pagina.getByRole('button', { name: 'Edit permissions for Inés Moreno Castillo (ines.moreno)' }).click();
  await pagina.getByLabel(en['usuarios.perfil_profesional_etf'], { exact: true }).check();
  for (const width of [1440, 1024, 768, 390, 320, 256]) { await pagina.setViewportSize({ width, height: 900 }); await sinDesborde(pagina); }
  await contexto.close();
  console.log(`ARMAZON-CHROME-OK: entrar → Usuarios → Catálogos → Paquetes; todas las pantallas de main, 256–1440 px, es/en, dos temas, teclado, menús y guardia; ${capturas} capturas`);
} finally { await navegador.close(); await new Promise((res) => servidor.close(res)); }
