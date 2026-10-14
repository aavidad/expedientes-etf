// Prepara expedientes nuevos por las API locales; nunca reintenta escrituras inciertas.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { apuntar as apuntarBase, pulsar as pulsarBase, elegir as elegirBase, escribir as escribirBase } from './comun.mjs';

async function centrar(loc) {
  await loc.evaluate(n => { if (n.getBoundingClientRect().height < 400) n.scrollIntoView({ block: 'center', behavior: 'instant' }); });
}
export async function apuntar(loc) { await centrar(loc); await apuntarBase(loc); }
export async function pulsar(loc) { await centrar(loc); await pulsarBase(loc); }
export async function elegir(loc, valor) { await centrar(loc); await elegirBase(loc, valor); }
export async function escribir(loc, valor) { await centrar(loc); await escribirBase(loc, valor); }

export const hoy = () => new Intl.DateTimeFormat('sv-SE', { timeZone: 'Europe/Madrid' }).format(new Date());
export async function pedir(page, ruta, datos, metodo = datos ? 'POST' : 'GET', archivo) {
  const resultado = await page.evaluate(async ({ ruta, datos, metodo, archivo }) => {
    const sesion = await (await fetch('/api/sesion')).json();
    const headers = { 'X-CSRF-Token': sesion.csrf, 'Idempotency-Key': crypto.randomUUID() };
    let body;
    if (archivo) { body = new FormData(); body.append('anexo', new Blob([new Uint8Array(archivo.contenido ?? archivo)]), archivo.nombre ?? 'anexo3.docx'); }
    else if (datos) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(datos); }
    const r = await fetch(ruta, { method: metodo, headers, body });
    return { estado: r.status, datos: r.status === 204 ? null : await r.json() };
  }, { ruta, datos, metodo, archivo });
  assert([200, 201, 204].includes(resultado.estado), `API local ${ruta}: ${resultado.estado}, ${JSON.stringify(resultado.datos)}`);
  return resultado.datos;
}
export async function crearExpediente(page, config) {
  const anexo = config.anexo === 'anexo4' ? 'anexo4' : 'anexo3';
  const formato = anexo === 'anexo4' ? 'pdf' : 'docx';
  const contenido = Array.from(readFileSync(new URL(`../../internal/adaptadores/anexos/${formato}/testdata/${anexo}.${formato}`, import.meta.url)));
  const lectura = await pedir(page, '/api/coordinacion/derivaciones', undefined, 'POST', { contenido, nombre: `${anexo}.${formato}` });
  assert.equal(lectura.lectura_anexo.antivirus, 'limpio');
  const opciones = await pedir(page, '/api/coordinacion/decision/opciones?idioma=es');
  const equipo = opciones.equipos.find(e => e.clave_texto === `equipo.${config.equipo_codigo}`);
  assert(equipo);
  const municipio = opciones.municipios.find(m => m.equipo_id === equipo.id && m.centro_ss_id); assert(municipio);
  const campos = lectura.derivacion_datos.contenido.campos.map(c => ({ ...c, dudoso: false, motivos: [] }));
  const derivacion = await pedir(page, '/api/coordinacion/derivaciones/confirmar', {
    lectura_id: lectura.lectura_anexo.id, tipo: anexo, campos,
    ubicacion: { municipio_id: municipio.id, centro_id: municipio.centro_ss_id, equipo_id: equipo.id },
  });
  const expediente = await pedir(page, `/api/coordinacion/derivaciones/${derivacion.id}/aceptar`, {
    version: derivacion.version, fecha_acta: hoy(), municipio_id: municipio.id, centro_ss_id: municipio.centro_ss_id,
    equipo_id: equipo.id, subprograma: config.subprograma ?? 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', profesional_referencia_id: null,
    menores: campos.filter(c => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: config.sexos_menores?.[i] ?? 'M' })),
    descripcion_situacion: config.descripcion_situacion,
  });
  assert.equal(expediente.estado, 'aceptada');
  return { ...expediente, equipo_id: equipo.id };
}
export async function descargar(s, ruta, boton, destino) {
  const respuesta = s.respuesta(ruta); const descarga = s.page.waitForEvent('download'); descarga.catch(() => {});
  await boton(); const r = await respuesta; if (![200, 201].includes(r.status())) throw new Error(`Descarga local ${ruta}: ${r.status()}`); assert(r.headers()['x-emision-id']);
  const d = await descarga; assert.equal(await d.failure(), null); await d.saveAs(destino);
  assert.equal(readFileSync(destino).subarray(0, 2).toString(), 'PK');
  return r.headers()['x-emision-id'];
}
