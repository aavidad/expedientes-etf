// Usa cuentas presentes en la demo; cada recorrido define sus escrituras inventadas.
import { readFileSync, mkdirSync } from 'node:fs';
import { grabarCoordinacion } from './coordinacion.mjs';
import { grabarFicha } from './ficha-video.mjs';
import { grabarTraslado } from './traslado-video.mjs';
import { grabarTabla } from './tabla-video.mjs';
import { grabarCierre } from './cierre-video.mjs';
import { grabarActividad } from './actividad.mjs';
import { grabarPlantillas } from './publicar-plantillas.mjs';
import { grabarMemoria } from './memoria-video.mjs';
import { grabarCiclo } from './ciclo-video.mjs';
import { grabarCarga } from './carga-video.mjs';
import { puedeRestaurar } from './restaurar.mjs';
import { resolve } from 'node:path';
import { chrome, viewport, baseLocal, restringir, temaActivo, fuentesActivas, instalarTema, cartela, rotulo, pulsar, escribir, elegir, apuntar, escenas, guardar, espera } from './comun.mjs';

const [archivo, dir, modo] = process.argv.slice(2);
const guion = JSON.parse(readFileSync(archivo, 'utf8'));
mkdirSync(dir, { recursive: true });
const referencias = [];
const textos = Object.assign({}, ...['identidad', 'usuarios'].map(n => JSON.parse(readFileSync(new URL(`../../idiomas/es/textos/${n}.json`, import.meta.url), 'utf8'))));
const t = clave => { if (!textos[clave]) throw new Error(`Clave ausente: ${clave}`); return textos[clave]; };
const paso = id => { const p = guion.pasos.find(p => p.id === id); if (!p) throw new Error(`Paso ausente: ${id}`); return p; };
const mostrar = (page, id) => rotulo(page, paso(id), dir, referencias);
const browser = await chrome();
let contexto, previo, adminPage, original, asignado, escritura = false;
let restaurado = false;
const config = guion.config;
let base, profesional, admin, equipo;
const entorno = nombre => { const valor = process.env[nombre]; if (!valor) throw new Error(`Falta ${nombre}`); return valor; };
const boton = (page, clave) => page.getByRole('button', { name: t(clave), exact: true });

async function entrar(page, cuenta, clave) {
  if (!page.url().endsWith('/entrar')) await page.goto(`${base}/entrar`);
  await page.getByLabel(t('identidad.cuenta'), { exact: true }).fill(cuenta);
  await page.getByLabel(t('identidad.contrasena'), { exact: true }).fill(clave);
  const respuesta = page.waitForResponse(r => new URL(r.url()).pathname === '/api/sesion' && r.request().method() === 'POST');
  await pulsar(boton(page, 'identidad.entrar'));
  const r = await respuesta;
  if (!r.ok()) throw new Error(`La demo rechazó la entrada: HTTP ${r.status()}`);
}
async function pedir(ruta, datos) {
  return adminPage.evaluate(async ({ ruta, datos }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const r = await fetch(ruta, datos ? { method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': sesion.csrf }, body: JSON.stringify(datos) } : {});
    if (!r.ok) throw new Error(`Demo: HTTP ${r.status}`);
    return r.json();
  }, { ruta, datos });
}
async function usuarios() { return pedir('/api/admin/usuarios'); }

try {
  if (modo === 'sintetico') {
    const nombre = process.env.ETF_VIDEO_TEMA || 'expedientes-etf';
    if (!/^[a-z0-9-]+$/.test(nombre)) throw new Error('Tema no válido');
    const carpeta = new URL(`../../temas/${nombre}/`, import.meta.url);
    const datos = JSON.parse(readFileSync(new URL('tokens.json', carpeta), 'utf8'));
    const manifiesto = JSON.parse(readFileSync(new URL('tema.json', carpeta), 'utf8'));
    const tokens = Object.fromEntries(Object.entries(datos).map(([k, v]) => [`--${k.replaceAll('.', '-')}`, v]));
    const fuentes = Object.entries(manifiesto.fuentes).map(([tipo, ruta]) => {
      const familia = datos[`fuente.${tipo}`].split(',')[0];
      const bytes = readFileSync(new URL(ruta, carpeta));
      return `@font-face{font-family:${familia};src:url(data:font/woff2;base64,${bytes.toString('base64')}) format('woff2');font-weight:100 900}`;
    }).join('');
    guardar(resolve(dir, 'tema.json'), { tokens, fuentes });
    contexto = await browser.newContext({ viewport, recordVideo: { dir, size: viewport }, serviceWorkers: 'block' });
    await contexto.route('**/*', r => r.abort());
    const origen = performance.now();
    const page = await contexto.newPage();
    await instalarTema(page, tokens, fuentes);
    await cartela(page, config.sintetico);
    const escena = await escenas(page, guion, dir, referencias, origen);
    for (const p of guion.pasos) await escena(p.id, p.frases.map(() => async () => {}));
    guardar(resolve(dir, 'referencias.json'), referencias);
    const video = page.video();
    await contexto.close(); contexto = null;
    await video.saveAs(resolve(dir, 'original.webm'));
    await video.delete();
  } else if (config.automatizacion === 'coordinacion') {
    await grabarCoordinacion({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'ficha') {
    await grabarFicha({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'traslado') {
    await grabarTraslado({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'spaf') {
    await grabarTabla({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'cierre') {
    await grabarCierre({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'actividad') {
    await grabarActividad({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'plantillas') {
    await grabarPlantillas({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'memoria') {
    await grabarMemoria({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'ciclo') {
    await grabarCiclo({ browser, guion, dir, referencias });
  } else if (config.automatizacion === 'carga') {
    await grabarCarga({ browser, guion, dir, referencias });
  } else {
    if (process.env.ETF_VIDEO_DEMO !== '1') throw new Error('Arranque manual: exige ETF_VIDEO_DEMO=1 y sólo datos inventados');
    base = baseLocal(entorno('ETF_VIDEO_URL'));
    profesional = entorno(config.profesional_env); admin = entorno(config.administracion_env); equipo = process.env[config.equipo_env];
    if (profesional === admin) throw new Error('Las cuentas de administración y profesional deben ser distintas');
    previo = await browser.newContext({ viewport, serviceWorkers: 'block' });
    await restringir(previo, base); adminPage = await previo.newPage();
    await entrar(adminPage, admin, entorno(config.clave_administracion_env));
    if (!equipo) {
      const catalogo = await pedir(`/api/admin/catalogos/asignacion_municipios/datos?fecha=${new Date().toISOString().slice(0, 10)}`);
      equipo = catalogo.equipos.find(e => e.codigo === config.equipo_demo)?.id;
      if (!equipo) throw new Error('Falta el equipo previsto por el guion en la demo');
    }
    original = (await usuarios()).find(u => u.cuenta === profesional);
    if (!original || !['activo', 'pendiente_alta'].includes(original.estado) || original.perfiles?.length || original.equipo) {
      throw new Error('La cuenta profesional debe existir en la demo, sin perfiles ni equipo');
    }
    const tema = { tokens: await temaActivo(adminPage), fuentes: await fuentesActivas(adminPage) };
    guardar(resolve(dir, 'tema.json'), tema);
    contexto = await browser.newContext({ viewport, locale: 'es-ES', timezoneId: 'Europe/Madrid', recordVideo: { dir, size: viewport }, serviceWorkers: 'block' });
    await restringir(contexto, base);
    const origen = performance.now();
    const page = await contexto.newPage();
    await page.goto(`${base}/entrar`);
    const escena = await escenas(page, guion, dir, referencias, origen);
    const acceso = async (cuenta, clave) => {
      await escribir(page.getByLabel(t('identidad.cuenta'), { exact: true }), cuenta);
      await escribir(page.getByLabel(t('identidad.contrasena'), { exact: true }), clave, true);
    };
    const confirmarEntrada = async () => {
      const r = page.waitForResponse(r => new URL(r.url()).pathname === '/api/sesion' && r.request().method() === 'POST');
      await pulsar(boton(page, 'identidad.entrar'));
      if (!(await r).ok()) throw new Error('Entrada de demostración rechazada');
      await boton(page, 'identidad.entrar').waitFor({ state: 'detached' });
    };
    await escena('profesional', [
      () => acceso(profesional, entorno(config.clave_profesional_env)), confirmarEntrada,
    ]);
    await escena('pendiente', [async () => {
      await apuntar(page.getByRole('heading', { name: t('identidad.pendiente_titulo'), exact: true }));
    }, async () => {
      await apuntar(page.getByText(t('identidad.pendiente_descripcion'), { exact: true }));
    }]);
    await escena('salir', [async () => {
      await pulsar(page.locator('details.persona-conectada > summary'));
      await pulsar(boton(page, 'identidad.salir'));
      await boton(page, 'identidad.entrar').waitFor();
    }]);
    await escena('administracion', [
      () => acceso(admin, entorno(config.clave_administracion_env)), confirmarEntrada,
    ]);
    await escena('usuarios', [
      () => pulsar(page.locator('a[href="/admin/usuarios"]').first()),
      () => pulsar(page.getByRole('button', { name: t('usuarios.editar_persona').replace('{nombre}', original.nombre).replace('{cuenta}', profesional), exact: true })),
    ]);
    await escena('perfil', [
      () => pulsar(page.getByLabel(t('usuarios.perfil_profesional_etf'), { exact: true })),
      async () => {
        await elegir(page.getByLabel(t('usuarios.equipo'), { exact: true }), equipo);
        await elegir(page.getByLabel(t('usuarios.puesto'), { exact: true }), config.puesto);
      },
    ]);
    await escena('guardado', [async () => {
      const cambio = page.waitForResponse(r => new URL(r.url()).pathname === `/api/admin/usuarios/${original.id}/perfiles` && r.request().method() === 'PUT');
      escritura = true;
      await pulsar(boton(page, 'usuarios.guardar'));
      const respuesta = await cambio;
      if (!respuesta.ok()) throw new Error(`Guardar permisos: HTTP ${respuesta.status()}`);
      asignado = await respuesta.json();
      await page.getByRole('status').filter({ hasText: t('usuarios.guardado') }).waitFor();
    }, async () => {
      const aviso = page.getByRole('status').filter({ hasText: t('usuarios.guardado') });
      await apuntar(aviso);
    }]);
    guardar(resolve(dir, 'referencias.json'), referencias);
    const video = page.video();
    await contexto.close(); contexto = null;
    await video.saveAs(resolve(dir, 'original.webm')); await video.delete();
  }
} finally {
  try {
    if (escritura && !asignado) throw new Error('No hay confirmación propia de la asignación. Se conservan los permisos; revisar la demo');
    if (asignado) {
      const actual = (await usuarios()).find(u => u.id === original.id);
      if (!puedeRestaurar(actual, asignado, original.id, equipo)) {
        throw new Error('No se restaura: la cuenta cambió o no coincide con la asignación del vídeo. Revisar la demo');
      }
      await pedir(`/api/admin/usuarios/${original.id}/perfiles`, { version: actual.version, estado: 'activo', perfiles: [], equipo: null });
      const final = (await usuarios()).find(u => u.id === original.id);
      if (final.perfiles.length || final.equipo) throw new Error('La restauración no dejó la cuenta sin perfil');
      restaurado = true;
    }
  } finally {
    if (!["coordinacion", "actividad", "plantillas", "spaf", "cierre", "ficha", "traslado", "memoria", "carga", "ciclo"].includes(config.automatizacion) || modo === 'sintetico') {
      guardar(resolve(dir, 'estado.json'), { modo, pasos: referencias.length, restaurado, escritura });
    }
    await contexto?.close(); await previo?.close(); await browser.close();
  }
}
console.log(`GRABACION-OK ${modo}: ${referencias.length} rótulos; demo restaurada=${restaurado}; recorrido=${config.automatizacion ?? 'entrada'}`);
