// Registra actividad inventada y corrige únicamente el informe de esta ejecución.
import assert from 'node:assert/strict';
import { resolve } from 'node:path';
import { sesionVideo } from './sesion-video.mjs';
import { pulsar, apuntar, escribir, elegir, guardar } from './comun.mjs';

export async function grabarActividad(opciones) {
  const { guion, dir, referencias } = opciones;
  const c = guion.config;
  const s = await sesionVideo(opciones, ['actividad']);
  const { page, t, boton, respuesta, escena, leer } = s;
  const estado = { modo: 'demo', recorrido: 'actividad', pasos: 0, escrituras: [], consulta_provincial: false };
  let comision, informe, equipo;
  const abrir = async (provincial = false) => {
    await pulsar(page.locator('a[href="/equipo/comisiones"]').first());
    await page.getByText(t(provincial ? 'actividad.ambito_provincia' : 'actividad.ambito_equipo'), { exact: true }).waitFor();
    await page.getByRole('radio', { name: t('actividad.comisiones'), exact: true }).waitFor();
  };
  const fecha = async valor => { await apuntar(page.locator('#actividad-fecha')); await page.locator('#actividad-fecha').fill(valor); };
  const fila = async (tipo, registro) => {
    await page.getByRole('table').waitFor();
    const registros = await leer(`/api/${tipo}?anio=${registro.fecha.slice(0, 4)}`);
    const indice = registros.findIndex(r => r.id === registro.id && r.version === registro.version);
    assert(indice >= 0, 'El registro propio no aparece con su versión confirmada');
    const loc = page.locator('tbody tr').nth(indice);
    await loc.scrollIntoViewIfNeeded(); return loc;
  };
  const guardarRegistro = async (tipo, registro) => {
    const metodo = registro ? 'PUT' : 'POST';
    const ruta = `/api/${tipo}${registro ? `/${registro.id}` : ''}`;
    const recarga = respuesta(`/api/${tipo}`, 'GET');
    const cambio = respuesta(ruta, metodo); estado.escrituras.push({ ruta, metodo, confirmado: false });
    await pulsar(boton('actividad.guardar'));
    const r = await cambio; assert.equal(r.status(), registro ? 200 : 201);
    const nuevo = await r.json();
    estado.escrituras.at(-1).confirmado = true; estado.escrituras.at(-1).id = nuevo.id;
    assert.equal(nuevo.equipo_id, equipo);
    if (registro) { assert.equal(nuevo.id, registro.id); assert.equal(nuevo.version, registro.version + 1); }
    assert.equal((await recarga).status(), 200);
    await page.getByRole('heading', { name: t(`actividad.${tipo}`), exact: true }).waitFor();
    return nuevo;
  };
  try {
    await escena('abrir', [async () => {
      await s.entrar('profesional');
      const sesion = await leer('/api/sesion'); assert(sesion.perfiles.includes('profesional_etf')); equipo = sesion.equipo_id; assert(equipo);
    }, async () => {
      await abrir(); const equipos = await leer('/api/actividad/equipos'); const etiquetas = await leer('/api/actividad/etiquetas');
      const propio = equipos.find(e => e.id === equipo); assert(propio);
      assert.match(etiquetas.es[propio.clave_texto], new RegExp(`\\b0?${Number(c.equipo_codigo)}\\b`));
      await apuntar(page.getByText(t('actividad.ambito_equipo'), { exact: true }));
    }]);
    await escena('comision', [
      () => pulsar(boton('actividad.registrar_comisiones')),
      () => fecha(c.fecha_comision),
      async () => {
        const campo = page.locator('#actividad-observaciones');
        await campo.evaluate(n => n.scrollIntoView({ block: 'center', behavior: 'instant' }));
        await escribir(campo, c.observaciones);
      },
      async () => { comision = await guardarRegistro('comisiones'); assert.equal(comision.observaciones, c.observaciones); await apuntar(await fila('comisiones', comision)); },
    ]);
    await escena('informe', [async () => {
      await pulsar(page.getByRole('radio', { name: t('actividad.informes'), exact: true }));
      await pulsar(boton('actividad.registrar_informes'));
    }, () => fecha(c.fecha_informe), () => elegir(page.locator('#actividad-organismo'), c.organismo), async () => {
      informe = await guardarRegistro('informes'); assert.equal(informe.organismo, c.organismo); await apuntar(await fila('informes', informe));
    }]);
    await escena('corregir', [async () => {
      await pulsar((await fila('informes', informe)).getByRole('button', { name: t('actividad.corregir'), exact: true }));
    }, () => fecha(c.fecha_corregida), async () => {
      informe = await guardarRegistro('informes', informe); assert.equal(informe.fecha, c.fecha_corregida); await apuntar(await fila('informes', informe));
    }]);
    await escena('coordinacion', [async () => {
      await pulsar(page.locator('details.persona-conectada > summary')); await pulsar(boton('identidad.salir'));
      await boton('identidad.entrar').waitFor();
    }, async () => { await s.entrar('coordinacion'); await abrir(true); }, async () => {
      assert.equal(await page.locator('#actividad-equipo').inputValue(), '');
      await apuntar(page.locator('#actividad-equipo'));
      await apuntar(await fila('comisiones', comision));
      assert.equal(await boton('actividad.registrar_comisiones').count(), 0);
    }]);
    await escena('provincial', [async () => {
      const consulta = respuesta('/api/informes', 'GET');
      await pulsar(page.getByRole('radio', { name: t('actividad.informes'), exact: true }));
      assert.equal((await consulta).status(), 200);
    }, async () => {
      await apuntar(await fila('informes', informe));
      assert.equal(await boton('actividad.registrar_informes').count(), 0); estado.consulta_provincial = true;
    }]);
    estado.comision = comision; estado.informe = informe; estado.pasos = referencias.length;
    await s.terminar();
  } finally { guardar(resolve(dir, 'estado.json'), estado); await s.contexto.close(); }
  console.log(`ACTIVIDAD-OK: ${estado.pasos} pasos; comisión e informe nuevos; informe versión=${informe.version}; consulta provincial=${estado.consulta_provincial}`);
}
