// Tabla SPAF con un alta y una baja nuevas de esta ejecución.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { sesionVideo } from './sesion-video.mjs';
import { pedir, hoy, descargar, pulsar, apuntar, elegir } from './datos-video.mjs';
import { guardar } from './comun.mjs';

export async function grabarTabla(opciones) {
  const { guion, dir, referencias } = opciones;
  const s = await sesionVideo(opciones, ['spaf', 'equipo']);
  const { page, t, escena } = s;
  const estado = { modo: 'demo', recorrido: 'spaf', pasos: 0, descargado: false };
  try {
    await s.entrar('coordinacion');
    const fecha = new Date(`${hoy()}T12:00:00Z`); fecha.setUTCDate(1); fecha.setUTCMonth(fecha.getUTCMonth() - 1);
    const mes = fecha.toISOString().slice(0, 7); estado.mes = mes;
    const opciones = await s.leer('/api/coordinacion/decision/opciones?idioma=es');
    const equipo = opciones.equipos.find(e => e.clave_texto === `equipo.${guion.config.equipo_codigo}`); assert(equipo);
    const municipio = opciones.municipios.find(m => m.equipo_id === equipo.id && m.centro_ss_id); assert(municipio);
    const usados = new Set();
    for (let desplazamiento = 0; ; desplazamiento += 50) {
      const lista = await s.leer(`/api/equipo/expedientes?desplazamiento=${desplazamiento}`);
      for (const e of lista.filas) usados.add(e.numero);
      if (!lista.hay_mas) break;
    }
    const libres = Array.from({ length: 99 }, (_, i) => `${guion.config.equipo_codigo}-${String(i + 1).padStart(2, '0')}-${mes.slice(2, 4)}`).filter(n => !usados.has(n));
    assert(libres.length >= 2, 'La demo necesita dos números libres para nuevos casos históricos');
    const [anio, m] = mes.split('-').map(Number);
    const ruta = `/api/estadistica/spaf?anio=${anio}&mes=${m}&etf=${equipo.id}`;
    const antes = await s.leer(ruta);
    const personas = JSON.parse(readFileSync(new URL('../../despliegue/demo/menores.json', import.meta.url)));
    for (let i = 0; i < 2; i++) {
      const episodio = { apertura_fecha: `${mes}-02`, ...(i ? { cierre_fecha: `${mes}-20`, motivo_cierre: 'OBJ_TOTAL' } : {}) };
      await pedir(page, '/api/expedientes/historicos', { equipo_id: equipo.id, casos: [{
        numero: libres[i], fecha_derivacion: `${mes}-01`, estado: i ? 'cerrada' : 'aceptada',
        derivacion_datos: { esquema_version: '1', contenido: { tipo: 'anexo3', campos: guion.config.campos } },
        aceptacion: { version: 1, fecha_acta: episodio.apertura_fecha, municipio_id: municipio.id, centro_ss_id: municipio.centro_ss_id,
          equipo_id: equipo.id, subprograma: 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', profesional_referencia_id: null,
          menores: [{ orden: 1, sexo: personas[i].sexo }], descripcion_situacion: guion.config.descripcion_situacion }, episodios: [episodio],
        menores: [{ version: 1, vigente_desde: episodio.apertura_fecha, menores: [{ ...personas[i], orden: 1, apellidos: guion.config.apellidos_menores, subprograma: 'RIESGO_SOCIAL', propuesto_atencion_temprana: false }] }],
      }] });
    }
    const despues = await s.leer(ruta);
    assert.equal(despues.resumen.altas, antes.resumen.altas + 2);
    assert.equal(despues.resumen.bajas, antes.resumen.bajas + 1);
    assert.equal(despues.resumen.casos, antes.resumen.casos + 2);
    assert.equal(despues.resumen.menores, antes.resumen.menores + 2);
    assert.equal(despues.resumen.casos_sin_dato_menores, antes.resumen.casos_sin_dato_menores);
    assert(despues.resumen.menores > 0);
    estado.menores_incremento = 2;
    const alta = despues.filas.find(f => f.numero === libres[0]); const baja = despues.filas.find(f => f.numero === libres[1]);
    assert(alta.alta && baja.baja); alta.equipo_id = equipo.id;
    estado.alta = alta; estado.baja = baja; estado.resumen = despues.resumen;
    const esperar = async () => {
      await page.locator('.spaf-movimientos a').filter({ hasText: alta.numero }).first().waitFor();
      await page.locator('.spaf-movimientos a').filter({ hasText: baja.numero }).first().waitFor();
    };
    await escena('abrir', [async () => { await pulsar(page.locator('a[href="/equipo/tabla-spaf"]').first()); await page.locator('#spaf-mes').waitFor(); }]);
    await escena('periodo', [async () => { await elegir(page.locator('#spaf-equipo'), alta.equipo_id); }, async () => { await apuntar(page.locator('#spaf-mes')); await page.locator('#spaf-mes').fill(mes); await esperar(); }]);
    await escena('movimientos', [async () => { await apuntar(page.locator('.spaf-indicadores')); const tarjeta = page.locator('.spaf-indicadores > div').filter({ has: page.getByText(t(despues.resumen.casos_sin_dato_menores ? 'spaf.menores_conocidos' : 'spaf.menores'), { exact: true }) }); assert.equal(await tarjeta.locator('strong').innerText(), new Intl.NumberFormat('es').format(despues.resumen.menores)); }, async () => { await apuntar(page.locator('.spaf-movimientos').first()); }]);
    await escena('alta', [async () => {
      await pulsar(page.locator('.spaf-movimientos a').filter({ hasText: alta.numero }).first());
      await page.getByRole('heading', { name: alta.numero, exact: true }).waitFor();
    }, async () => { await apuntar(page.locator('.spaf')); assert((await page.locator('.spaf').innerText()).includes(alta.numero)); }]);
    await escena('volver', [async () => { const consulta = page.waitForResponse(r => new URL(r.url()).pathname === '/api/estadistica/spaf');
      await pulsar(page.getByRole('link', { name: t('spaf.volver_tabla'), exact: true })); assert.equal((await consulta).status(), 200); await esperar(); await apuntar(page.locator('.spaf-indicadores')); }]);
    await escena('word', [async () => {
      const [anio, m] = mes.split('-').map(Number);
      estado.emision = await descargar(s, `/api/equipos/${alta.equipo_id}/tabla-spaf/${anio}/${m}.docx`, () => pulsar(page.getByRole('button', { name: new RegExp('^' + t('spaf.descargar_mes').split('{')[0]) })), resolve(dir, 'tabla-spaf.docx'));
      estado.descargado = true;
      const aviso = page.getByRole('status').filter({ hasText: t('spaf.descargado') }).or(page.getByRole('status').filter({ hasText: t('spaf.descargado_cambios') }));
      await apuntar(aviso); estado.aviso = await aviso.innerText();
    }]);
    estado.pasos = referencias.length; await s.terminar();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`SPAF-VIDEO-OK: ${estado.pasos} pasos; alta y baja nuevas; Word=${estado.descargado}`);
}
