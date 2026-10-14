// Comprueba la integración real sin guardar el anexo ni el Word descargado.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash, randomUUID } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { chromium } from '../../web/node_modules/playwright-core/index.mjs';

const url = process.argv[2];
assert.match(url, /^http:\/\/127\.0\.0\.1:\d+$/);
const claves = JSON.parse(await readFile(new URL('.local/usuarios.json', import.meta.url)));
const anexo = await readFile(new URL('../../internal/adaptadores/anexos/docx/testdata/anexo3.docx', import.meta.url));
const navegador = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
try {
  const contexto = await navegador.newContext();
  const pagina = await contexto.newPage();
  pagina.setDefaultTimeout(15000);
  async function entrar(cuenta) {
    await pagina.goto(url);
    await pagina.locator('#cuenta-ad').fill(cuenta);
    await pagina.locator('#contrasena-ad').fill(claves[cuenta]);
    const respuesta = pagina.waitForResponse(r => r.url() === url + '/api/sesion' && r.request().method() === 'POST');
    await pagina.locator('button[type="submit"]').click();
    assert.equal((await respuesta).status(), 200);
    await pagina.locator('#cuenta-ad').waitFor({ state: 'detached' });
  }
  async function pedir(ruta, { metodo = 'GET', datos, clave, fichero } = {}) {
    return pagina.evaluate(async ({ ruta, metodo, datos, clave, fichero }) => {
      const sesion = await (await fetch('/api/sesion')).json();
      const headers = { 'X-CSRF-Token': sesion.csrf };
      if (clave) headers['Idempotency-Key'] = clave;
      let body;
      if (fichero) {
        body = new FormData();
        body.append('anexo', new Blob([new Uint8Array(fichero)], {
          type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        }), 'anexo3.docx');
      } else if (datos) {
        headers['Content-Type'] = 'application/json';
        body = JSON.stringify(datos);
      }
      const respuesta = await fetch(ruta, { method: metodo, headers, body });
      const cabeceras = Object.fromEntries(respuesta.headers.entries());
      const word = cabeceras['content-type']?.includes('wordprocessingml');
      return {
        estado: respuesta.status, cabeceras,
        contenido: word ? Array.from(new Uint8Array(await respuesta.arrayBuffer()))
          : respuesta.status === 204 ? null : await respuesta.json(),
      };
    }, { ruta, metodo, datos, clave, fichero });
  }
  function json(resultado, esperado = 200) {
    assert.equal(resultado.estado, esperado, JSON.stringify(resultado.contenido));
    return resultado.contenido;
  }
  await entrar('elena.salas');
  const catalogo = json(await pedir('/api/admin/catalogos/asignacion_municipios/datos?fecha=2026-10-13'));
  const firmantes = json(await pedir('/api/admin/firmantes'));
  assert(firmantes.firmantes.some(f => f.rol === 'aceptacion'));
  json(await pedir('/api/sesion', { metodo: 'DELETE' }), 204);
  await entrar('ines.moreno');
  const analisis = json(await pedir('/api/coordinacion/derivaciones', {
    metodo: 'POST', fichero: Array.from(anexo),
  }));
  const opciones = analisis.derivacion_datos.opciones_ubicacion;
  const candidato = analisis.derivacion_datos.propuesta_municipio.Candidatos?.[0];
  const equipoPrueba = catalogo.equipos.find(e => e.codigo === '04');
  assert(equipoPrueba);
  const ubicacion = opciones.municipios.find(m => m.id === candidato?.MunicipioID && m.centro_id && m.equipo_id === equipoPrueba.id)
    ?? opciones.municipios.find(m => m.centro_id && m.equipo_id === equipoPrueba.id);
  assert(ubicacion);
  const confirmada = json(await pedir('/api/coordinacion/derivaciones/confirmar', {
    metodo: 'POST', clave: randomUUID(), datos: {
      lectura_id: analisis.lectura_anexo.id, tipo: analisis.derivacion_datos.contenido.tipo,
      campos: analisis.derivacion_datos.contenido.campos.map(c => ({ ...c, dudoso: false, motivos: [] })),
      ubicacion: { municipio_id: ubicacion.id, centro_id: ubicacion.centro_id, equipo_id: ubicacion.equipo_id },
    },
  }), 201);
  const derivacion = json(await pedir(`/api/coordinacion/derivaciones/${confirmada.id}`));
  assert.equal(derivacion.estado, 'derivacion_recibida');
  assert.equal(confirmada.version, 1);

  assert.equal(derivacion.numero, null);
  assert.equal(derivacion.lectura_anexo.antivirus, 'limpio');
  const campos = derivacion.derivacion_datos.contenido.campos;
  const municipio = catalogo.municipios.find(m => m.id === ubicacion.id);
  assert(municipio?.equipo_id && municipio?.centro_ss_id);
  // La coordinación confirma la asignación; la propuesta no la impone.
  const equipo = catalogo.equipos.find(e => e.id === municipio.equipo_id);
  assert(equipo);
  assert.equal(ubicacion.centro_id, municipio.centro_ss_id);
  assert.equal(ubicacion.equipo_id, equipo.id);

  const menores = campos.filter(c => c.codigo === 'menores.persona.nombre');
  const datos = {
    version: derivacion.version, fecha_acta: new Date().toLocaleDateString('sv-SE', { timeZone: 'Europe/Madrid' }), municipio_id: municipio.id,
    centro_ss_id: municipio.centro_ss_id, equipo_id: equipo.id,
    subprograma: 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS',
    profesional_referencia_id: null,
    menores: menores.map((_, i) => ({ orden: i + 1, sexo: i % 2 ? 'M' : 'H' })),
    descripcion_situacion: campos.find(c => c.codigo === 'valoracion.descripcion')?.valor,
  };
  const ruta = `/api/coordinacion/derivaciones/${derivacion.id}/aceptar`;
  const claveAceptacion = randomUUID();
  const aceptada = json(await pedir(ruta, { metodo: 'POST', datos, clave: claveAceptacion }));
  assert.equal(aceptada.estado, 'aceptada');
  assert.equal(aceptada.version, derivacion.version + 1);
  assert.equal(aceptada.numero, `${equipo.codigo}-${String(aceptada.correlativo).padStart(2, '0')}-${datos.fecha_acta.slice(2, 4)}`);
  assert.deepEqual(json(await pedir(ruta, { metodo: 'POST', datos, clave: claveAceptacion })), aceptada);
  const solicitud = { tipo: 'entrada/aceptacion', expediente_id: derivacion.id, idioma: 'es' };
  const claveEmision = randomUUID();
  const acta = await pedir('/api/documentos/emisiones', { metodo: 'POST', datos: solicitud, clave: claveEmision });
  assert.equal(acta.estado, 201, JSON.stringify(acta.contenido));
  assert.match(acta.cabeceras['content-disposition'], /attachment/);
  const contenido = Buffer.from(acta.contenido);
  assert.equal(createHash('sha256').update(contenido).digest('hex'), acta.cabeceras['x-documento-sha256']);
  const xml = spawnSync('python3', ['-c',
    'import io,sys,zipfile; z=zipfile.ZipFile(io.BytesIO(sys.stdin.buffer.read())); assert z.testzip() is None; sys.stdout.write(z.read("word/document.xml").decode())',
  ], { input: contenido, encoding: 'utf8', timeout: 15000, maxBuffer: 2 << 20 });
  assert.equal(xml.status, 0, xml.stderr);
  assert(xml.stdout.includes(aceptada.numero));
  assert(xml.stdout.includes('Inés Moreno Castillo'));
  for (const menor of menores) assert(xml.stdout.includes(menor.valor));
  const repetida = await pedir('/api/documentos/emisiones', { metodo: 'POST', datos: solicitud, clave: claveEmision });
  assert.equal(repetida.estado, 200);
  assert.equal(repetida.cabeceras['x-idempotencia-repetida'], 'true');
  assert.equal(repetida.cabeceras['x-emision-id'], acta.cabeceras['x-emision-id']);
  assert.deepEqual(Buffer.from(repetida.contenido), contenido);
  const ficha = json(await pedir(`/api/expedientes/${derivacion.id}`));
  assert.equal(ficha.numero, aceptada.numero);
  assert.equal(ficha.version, aceptada.version);
  assert.equal(ficha.aceptacion.fecha_acta, datos.fecha_acta);
  json(await pedir('/api/sesion', { metodo: 'DELETE' }), 204);
  console.log(`RECORRIDO-OK: Chrome del sistema; Anexo 3 limpio; ${menores.length} menores; aceptación ${aceptada.numero}; Acta DOCX íntegra (${contenido.length} bytes); idempotencia y fecha conservadas.`);
} finally {
  await navegador.close();
}
