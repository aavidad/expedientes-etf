import { useEffect, useRef, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useCambios } from '../cambios';
import { Boton, Campo, Ayuda } from './componentes';
import { formulario, episodio, menor, adulto, preparar, iniciales, tieneDatos, separarNombre, hojaInventada, columnas, estados, type Formulario, type Opciones, type Informe, type Recibo, type Resumen } from './modelo';
import './carga.css';
import { EstadoPlazoCarga } from './control';
import { Actas } from './actas';

type Consulta = { numero: string | null; estado: string; cargado_para_revision: boolean; fecha_derivacion: string; aceptacion?: { fecha_acta: string }; no_aceptacion?: { fecha: string; motivo: string }; derivacion_datos: { contenido: { campos: { codigo: string; valor: string }[] } } };
type Envio = { ruta: string; cuerpo: BodyInit; tipo: string; equipo?: string; huella?: string; clave: string };
function Carga({ t, idioma, csrf, navegarRuta, manual }: PropsPantalla & { manual: boolean }) {
  const idConsulta = new URLSearchParams(location.search).get('expediente') ?? '';
  const { ponerSucio } = useCambios();
  const [f, ponerF] = useState<Formulario>(formulario);
  const [opciones, ponerOpciones] = useState<Opciones>();
  const [intento, ponerIntento] = useState(0);
  const [cargando, ponerCargando] = useState(true);
  const [ayuda, ponerAyuda] = useState(false);
  const [fichero, ponerFichero] = useState<File>();
  const [duplicadosConfirmados, ponerDuplicadosConfirmados] = useState(false);
  const [informe, ponerInforme] = useState<Informe>();
  const [consulta, ponerConsulta] = useState<Consulta>();
  const [leyendo, ponerLeyendo] = useState(false);
  const [recibo, ponerRecibo] = useState<Recibo>();
  const [error, ponerError] = useState('');
  const [errores, ponerErrores] = useState<Record<string, string>>({});
  const [ocupado, ponerOcupado] = useState(false);
  const [leyendoActas, ponerLeyendoActas] = useState(false);
  const [incierto, ponerIncierto] = useState(false);
  const peticion = useRef<AbortController | null>(null);
  const envio = useRef<Envio | null>(null);
  const borrador = useRef<ReturnType<typeof preparar> | null>(null);
  const resumenError = useRef<HTMLDivElement>(null);
  const focoPaso = useRef<HTMLHeadingElement>(null);
  const bloqueado = ocupado || incierto || leyendoActas;
  const texto = (clave: string) => { const valor = t(clave); return valor === clave ? t('carga.error') : valor; };
  useEffect(() => {
    const abortar = new AbortController(); ponerCargando(true); ponerOpciones(undefined);
    void fetch(`/api/carga-2026/opciones?idioma=${encodeURIComponent(idioma)}`, { signal: abortar.signal })
      .then(async r => { if (!r.ok) throw new Error(r.status === 403 ? 'carga.sin_permiso' : 'carga.error_opciones'); return r.json() as Promise<Opciones>; })
      .then(o => { ponerOpciones(o); if (o.equipos.length === 1) ponerF(v => ({ ...v, equipo: o.equipos[0].id })); })
      .catch((e: unknown) => { if (!abortar.signal.aborted) ponerError(e instanceof Error ? e.message : 'carga.error_opciones'); })
      .finally(() => { if (!abortar.signal.aborted) ponerCargando(false); });
    return () => abortar.abort();
  }, [idioma, intento]);
  useEffect(() => { if (idConsulta) void leerResultado(idConsulta); return () => peticion.current?.abort(); }, [idConsulta]);
  useEffect(() => () => { peticion.current?.abort(); envio.current = null; borrador.current = null; }, []);
  useEffect(() => { if (error || Object.keys(errores).length) resumenError.current?.focus(); }, [error, errores]);
  useEffect(() => { if (informe || recibo) focoPaso.current?.focus(); }, [informe, recibo]);
  function editar(actualizar: (v: Formulario) => Formulario) { ponerF(actualizar); ponerSucio(true); ponerInforme(undefined); ponerRecibo(undefined); ponerError(''); ponerErrores({}); borrador.current = null; envio.current = null; }
  function valor(clave: keyof Formulario, v: string) {
    editar(actual => {
      const nuevo = { ...actual, [clave]: v };
      if (clave === 'equipo') { nuevo.municipio = ''; nuevo.centro = ''; }
      if (clave === 'municipio') { const m = opciones?.municipios.find(o => o.id === v); nuevo.centro = m?.centro_ss_id ?? ''; }
      if (clave === 'tipo' && v === 'anexo3') nuevo.subprograma = 'RIESGO_SOCIAL';
      return nuevo;
    });
  }
  const nombre = (clave: string) => opciones?.textos[clave] || t('carga.etiqueta_pendiente');
  const lista = (opcionesLista: { id: string; clave_texto: string }[]) => opcionesLista.map(o => ({ valor: o.id, etiqueta: nombre(o.clave_texto) }));
  function campo(clave: keyof Formulario, obligatorio = true, tipo = 'text', opcionesCampo?: { valor: string; etiqueta: string }[], area = false) {
    return <Campo key={clave} id={`carga-${clave}`} etiqueta={t(`carga.campo.${clave.toLowerCase()}`)} valor={String(f[clave])} cambiar={v => valor(clave, v)} obligatorio={obligatorio} tipo={tipo} opciones={opcionesCampo} area={area} error={errores[`carga-${clave}`]} disabled={bloqueado} t={t} />;
  }
  async function solicitar(ruta: string, cuerpo: BodyInit, tipo: string, confirmar = false, equipo?: string, huella?: string) {
    if (ocupado) return;
    const abortar = new AbortController(); peticion.current = abortar; ponerOcupado(true); ponerError('');
    if (confirmar) {
      envio.current ??= { ruta, cuerpo, tipo, equipo, huella, clave: crypto.randomUUID() };
      ({ ruta, cuerpo, tipo, equipo, huella } = envio.current);
    }
    try {
      const headers: Record<string, string> = { 'Content-Type': tipo, 'X-CSRF-Token': csrf, 'Accept-Language': idioma };
      if (equipo) headers['X-Equipo-Carga'] = equipo;
      if (confirmar && duplicadosConfirmados) headers['X-Confirmar-Duplicados'] = 'true';
      if (huella) headers['X-Huella-Hoja'] = huella;
      if (confirmar && envio.current) headers['Idempotency-Key'] = envio.current.clave;
      const r = await fetch(ruta, { method: 'POST', headers, body: cuerpo, signal: abortar.signal });
      const datos = await r.json();
      if (confirmar && r.ok) {
        if (!Array.isArray(datos.expedientes) || !datos.expedientes.length || typeof datos.lote_id !== 'string') throw new Error('carga.error');
        ponerRecibo(datos as Recibo); ponerIncierto(false); ponerSucio(false); ponerFichero(undefined); borrador.current = null; envio.current = null;
      } else if (datos.incidencias && Array.isArray(datos.incidencias)) {
        ponerDuplicadosConfirmados(false); ponerInforme(datos as Informe); ponerIncierto(false); envio.current = null;
      } else if (!r.ok) {
        const clave = typeof datos.error === 'string' ? datos.error : datos.error?.clave;
        ponerError(clave || 'carga.error');
        // Una respuesta de rechazo permite corregir; un fallo de red exige recuperar el mismo envío.
        ponerIncierto(false); envio.current = null;
        if (confirmar) ponerInforme(undefined);
      } else if (!confirmar && datos.lote_id) {
        ponerDuplicadosConfirmados(false); ponerInforme({ ...datos, valida: true, incidencias: [] } as Informe);
      } else throw new Error('carga.error');
    } catch {
      ponerError(confirmar ? 'carga.resultado_incierto' : 'carga.error');
      if (confirmar) ponerIncierto(true);
    } finally { if (peticion.current === abortar) { ponerOcupado(false); peticion.current = null; } }
  }
  function validarManual() {
    const problemas: Record<string, string> = {};
    const falta = (id: string, v: string) => { if (!v.trim()) problemas[id] = t('carga.obligatorio'); };
    for (const clave of ['equipo', 'estado', 'tipo', 'derivacion', 'subprograma'] as const) falta(`carga-${clave}`, f[clave]);
    if (f.estado === 'no_aceptada') { falta('carga-noAceptacion', f.noAceptacion); falta('carga-causa', f.causa); }
    else if (f.estado) {
      for (const clave of ['aceptacion', 'municipio', 'centro', 'puesto', 'descripcion'] as const) falta(`carga-${clave}`, f[clave]);
      if (f.numero && !/^\d{2}-\d{2}-\d{2}$/.test(f.numero)) problemas['carga-numero'] = t('carga.numero_formato');
      f.menores.forEach((m, i) => { if (m.nombreCompleto) problemas[`carga-menor-nombre-${i}`] = t('carga.separar_para_guardar'); falta(`carga-menor-nombre-${i}`, m.nombre); falta(`carga-menor-sexo-${i}`, m.sexo); });
      f.adultos.forEach((a,i) => { if (a.nombreCompleto) problemas[`carga-adulto-nombre-${i}`] = t('carga.separar_para_guardar'); for (const clave of ['nombre','papel','sexo'] as const) falta(`carga-adulto-${clave}-${i}`,a[clave]); });
      f.episodios.forEach((e, i) => {
        if (i) { falta(`carga-apertura-${i}`, e.apertura_fecha); falta(`carga-reapertura-${i}`, e.motivo_reapertura); }
        if (i < f.episodios.length - 1 || f.estado === 'cerrada') falta(`carga-cierre-${i}`, e.cierre_fecha);
        if (e.cierre_fecha) falta(`carga-motivo-${i}`, e.motivo_cierre);
      });
    }
    ponerErrores(problemas);
    if (Object.keys(problemas).length) return;
    const p = preparar(f); borrador.current = p;
    void solicitar('/api/carga-2026', JSON.stringify(p), 'application/json');
  }
  function subirHoja() {
    if (!f.equipo || !fichero) { ponerErrores({ ...(!f.equipo ? { 'carga-equipo': t('carga.obligatorio') } : {}), ...(!fichero ? { 'carga-hoja': t('carga.elegir_fichero') } : {}) }); return; }
    if (!/\.(csv|xlsx)$/i.test(fichero.name) || fichero.size > 10000000 || !fichero.size) { ponerError('carga.fichero_invalido'); return; }
    ponerErrores({}); ponerInforme(undefined);
    void solicitar('/api/carga-2026/hoja', fichero, tipoHoja(fichero), false, f.equipo);
  }
  function tipoHoja(file: File) { return /\.xlsx$/i.test(file.name) ? 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' : 'text/csv'; }
  function confirmar() {
    if (informe?.avisos?.length && !duplicadosConfirmados) { ponerError('carga.duplicados_sin_confirmar'); return; }
    if (incierto && envio.current) { const e = envio.current; void solicitar(e.ruta, e.cuerpo, e.tipo, true, e.equipo, e.huella); return; }
    if (!informe?.valida || !informe.lote_id) return;
    if (manual && borrador.current) void solicitar(`/api/carga-2026/${encodeURIComponent(informe.lote_id)}/confirmar`, JSON.stringify({ ...borrador.current, confirmar_duplicados: duplicadosConfirmados }), 'application/json', true);
    else if (fichero) void solicitar(`/api/carga-2026/hoja/${encodeURIComponent(informe.lote_id)}/confirmar`, fichero, tipoHoja(fichero), true, f.equipo, informe.huella);
  }
  function reiniciar() { ponerConsulta(undefined); ponerInforme(undefined); ponerRecibo(undefined); ponerFichero(undefined); ponerF({ ...formulario(), equipo: f.equipo }); ponerError(''); ponerErrores({}); ponerSucio(false); }
  async function leerResultado(id: string) {
    const abortar = new AbortController(); peticion.current?.abort(); peticion.current = abortar;
    ponerLeyendo(true); ponerOcupado(true); ponerConsulta(undefined); ponerError('');
    try {
      const r = await fetch(`/api/expedientes/${encodeURIComponent(id)}`, { signal: abortar.signal });
      const datos = await r.json();
      if (!r.ok) { ponerError(r.status === 403 ? 'carga.sin_permiso' : 'carga.error_lectura'); return; }
      if (!abortar.signal.aborted) ponerConsulta(datos as Consulta);
    } catch { if (!abortar.signal.aborted) ponerError('carga.error_lectura'); }
    finally { if (peticion.current === abortar) { ponerLeyendo(false); ponerOcupado(false); peticion.current = null; } }
  }
  function descargar() {
    if (!f.equipo) { ponerErrores({ 'carga-equipo': t('carga.obligatorio') }); return; }
    const url = URL.createObjectURL(new Blob([hojaInventada(t, opciones, f.equipo)], { type: 'text/csv;charset=utf-8' }));
    const a = document.createElement('a'); a.href = url; a.download = 'carga-2026.csv'; a.click(); URL.revokeObjectURL(url);
  }
  const filas: Resumen[] = manual ? [{ fila: 1, numero: f.estado === 'no_aceptada' ? '' : f.numero, estado: f.estado, iniciales: iniciales([f.menores[0]?.nombre, f.menores[0]?.apellidos].filter(Boolean).join(' ')) }] : informe?.filas ?? [];
  const inc = informe?.incidencias ?? [];
  function fecha(dia: string) { return dia ? new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: 'UTC' }).format(new Date(dia + 'T12:00:00Z')) : t('carga.sin_datos'); }
  function irCampo(id: string) { document.getElementById(id)?.focus(); }
  const singular = (cantidad: number) => new Intl.PluralRules(idioma).select(cantidad) === 'one';
  function datosGuardados() { return consulta && <section className="carga-tarjeta" aria-labelledby="carga-consulta"><h3 id="carga-consulta">{t('carga.datos_guardados')}</h3>{consulta.cargado_para_revision && <p className="carga-marca" role="status">{t('carga.marca')}</p>}<dl><dt>{t('carga.campo.numero')}</dt><dd>{consulta.numero || t('carga.sin_numero')}</dd><dt>{t('carga.campo.estado')}</dt><dd>{t(estados.includes(consulta.estado) ? `carga.estado.${consulta.estado}` : 'carga.estado_pendiente')}</dd><dt>{t('carga.campo.derivacion')}</dt><dd>{fecha(consulta.fecha_derivacion)}</dd>{consulta.no_aceptacion && <><dt>{t('carga.campo.noaceptacion')}</dt><dd>{fecha(consulta.no_aceptacion.fecha)}</dd><dt>{t('carga.campo.causa')}</dt><dd>{consulta.no_aceptacion.motivo}</dd></>}{consulta.aceptacion && <><dt>{t('carga.campo.aceptacion')}</dt><dd>{fecha(consulta.aceptacion.fecha_acta)}</dd></>}{consulta.derivacion_datos.contenido.campos.filter(c => c.codigo === 'menores.persona.nombre').map((c, i) => <div key={i}><dt>{t('carga.menor_n').replace('{cantidad}', new Intl.NumberFormat(idioma).format(i + 1))}</dt><dd>{c.valor}</dd></div>)}</dl></section>; }
  if (idConsulta) return <div className="carga"><h1>{t('carga.revision_guardada')}</h1>{leyendo && <p role="status">{t('carga.leyendo')}</p>}{error && <div className="carga-aviso" role="alert"><p>{texto(error)}</p>{error === 'carga.duplicados_sin_confirmar' && <a href="#carga-confirmar-duplicados" onClick={e => { e.preventDefault(); irCampo('carga-confirmar-duplicados'); }}>{t('carga.revisar_coincidencias')}</a>}<Boton disabled={ocupado} accion={() => void leerResultado(idConsulta)}>{t('carga.reintentar_lectura')}</Boton></div>}{datosGuardados()}<Boton accion={() => navegarRuta('/equipo/expedientes/alta-historica')}>{t('carga.volver_alta')}</Boton></div>;
  return <div className="carga">
    <div className="carga-encabezado"><h1>{t(manual ? 'carga.manual_titulo' : 'carga.titulo')}</h1><Boton accion={() => ponerAyuda(true)}>{t('carga.ayuda_boton')}</Boton></div>
    <p>{t('carga.contexto')}</p><EstadoPlazoCarga t={t} idioma={idioma} />
    {(error || Object.keys(errores).length > 0) && <div id="carga-error-resumen" className="carga-aviso" tabIndex={-1} ref={resumenError} role="alert"><h2>{t('carga.revisar_errores')}</h2>{error && <><p>{texto(error)}</p>{error === 'carga.duplicados_sin_confirmar' && <a href="#carga-confirmar-duplicados" onClick={e => { e.preventDefault(); irCampo('carga-confirmar-duplicados'); }}>{t('carga.revisar_coincidencias')}</a>}</>}{Object.keys(errores).length > 0 && <ul>{Object.entries(errores).map(([id, mensaje]) => <li key={id}><a href={`#${id}`} onClick={e => { e.preventDefault(); irCampo(id); }}>{(document.getElementById(id) as HTMLInputElement | null)?.labels?.[0]?.textContent ?? t('carga.campo_pendiente')}{t('carga.separador')}{mensaje}</a></li>)}</ul>}{error === 'carga.error_opciones' && <Boton accion={() => { ponerError(''); ponerIntento(v => v + 1); }}>{t('carga.reintentar')}</Boton>}{incierto && <Boton principal disabled={ocupado} accion={confirmar}>{t('carga.recuperar')}</Boton>}</div>}
    {cargando && <p role="status">{t('carga.cargando')}</p>}
    {!cargando && opciones && !opciones.equipos.length && <p className="carga-info" role="status">{t('carga.sin_equipo')}</p>}
    {ocupado && <div className="carga-info" role="status" aria-live="polite">{t(leyendo ? 'carga.leyendo' : envio.current ? 'carga.guardando' : manual ? 'carga.validando' : 'carga.antivirus')}</div>}
    {recibo ? <section className="carga-tarjeta" aria-labelledby="carga-resultado"><h2 id="carga-resultado" tabIndex={-1} ref={focoPaso}>{t('carga.resultado')}</h2>{recibo.referencia && <p>{t('carga.lote_referencia').replace('{numero}',new Intl.NumberFormat(idioma).format(recibo.referencia))}</p>}<p role="status">{t(singular(recibo.expedientes.length) ? 'carga.guardado' : 'carga.guardados').replace('{cantidad}', new Intl.NumberFormat(idioma).format(recibo.expedientes.length))}</p><p className="carga-marca">{t('carga.marca')}</p><ul>{recibo.expedientes.map((id, i) => <li key={id}>{!filas[i]?.numero && <a href={`/equipo/expedientes/alta-historica?expediente=${encodeURIComponent(id)}`} onClick={e => { if (e.button === 0 && !e.ctrlKey && !e.metaKey && !e.shiftKey) { e.preventDefault(); navegarRuta(`/equipo/expedientes/alta-historica?expediente=${encodeURIComponent(id)}`); } }}>{t('carga.enlace_revision').replace('{cantidad}', new Intl.NumberFormat(idioma).format(i + 1))}</a>}{!filas[i]?.numero && <Boton disabled={ocupado} accion={() => void leerResultado(id)}>{t('carga.revisar_guardado').replace('{cantidad}', new Intl.NumberFormat(idioma).format(i + 1))}</Boton>}{filas[i]?.numero ? <a href={`/equipo/expediente?id=${encodeURIComponent(id)}`} onClick={e => { e.preventDefault(); navegarRuta(`/equipo/expediente?id=${encodeURIComponent(id)}`); }}>{t('carga.abrir').replace('{cantidad}', new Intl.NumberFormat(idioma).format(i + 1))}</a> : <p>{t(estados.includes(filas[i]?.estado) ? `carga.estado.${filas[i].estado}` : 'carga.estado_pendiente')}{t('carga.separador')}{t('carga.sin_numero')}</p>}</li>)}</ul>{datosGuardados()}<Boton disabled={ocupado} accion={reiniciar}>{t('carga.nueva_carga')}</Boton></section> : informe ? <section className="carga-tarjeta" aria-labelledby="carga-vista"><h2 id="carga-vista" tabIndex={-1} ref={focoPaso}>{t('carga.vista')}</h2><p role="status">{t(informe.valida ? (singular(informe.casos) ? 'carga.lote_valido_uno' : 'carga.lote_valido') : (singular(informe.casos) ? 'carga.lote_invalido_uno' : 'carga.lote_invalido')).replace('{cantidad}', new Intl.NumberFormat(idioma).format(informe.casos))}</p>
      {manual && f.estado !== 'no_aceptada' && <section aria-labelledby="carga-resumen-ubicacion"><h3 id="carga-resumen-ubicacion">{t('carga.resumen_datos')}</h3><dl><dt>{t('carga.campo.equipo')}</dt><dd>{nombre(opciones?.equipos.find(o => o.id === f.equipo)?.clave_texto ?? '')}</dd><dt>{t('carga.campo.municipio')}</dt><dd>{nombre(opciones?.municipios.find(o => o.id === f.municipio)?.clave_texto ?? '')}</dd><dt>{t('carga.campo.centro')}</dt><dd>{nombre(opciones?.centros.find(o => o.id === f.centro)?.clave_texto ?? '')}</dd><dt>{t('carga.campo.subprograma')}</dt><dd>{nombre(opciones?.subprogramas.find(o => o.codigo === f.subprograma)?.clave_texto ?? '')}</dd><dt>{t('carga.campo.familianombre')}</dt><dd>{iniciales(f.familiaNombre) || t('carga.sin_datos')}</dd></dl><h4>{t('carga.adultos_resumen')}</h4>{!f.adultos.length && <p>{t('carga.sin_datos')}</p>}<ul>{f.adultos.map(persona => <li key={persona.id}>{iniciales([persona.nombre,persona.apellidos].join(' '))}{t('carga.separador')}{nombre(`papel_parental.${persona.papel}`)}{t('carga.separador')}{persona.sexo ? t(`carga.adulto.sexo.${persona.sexo.toLowerCase()}`) : t('carga.sin_datos')}{t('carga.separador')}{fecha(persona.fecha_nacimiento)}</li>)}</ul><h4>{t('carga.menores_resumen')}</h4><ul>{f.menores.map(persona => <li key={persona.id}>{iniciales([persona.nombre,persona.apellidos].join(' '))}{t('carga.separador')}{persona.sexo ? t(`carga.sexo.${persona.sexo.toLowerCase()}`) : t('carga.sin_datos')}{t('carga.separador')}{fecha(persona.fecha_nacimiento)}</li>)}</ul><p>{t('carga.nombres_revisados')}</p></section>}
      {manual && <section aria-labelledby="carga-fechas"><h3 id="carga-fechas">{t('carga.datos_historicos')}</h3><dl><dt>{t('carga.campo.derivacion')}</dt><dd>{fecha(f.derivacion)}</dd><dt>{t(f.estado === 'no_aceptada' ? 'carga.campo.noaceptacion' : 'carga.campo.aceptacion')}</dt><dd>{fecha(f.estado === 'no_aceptada' ? f.noAceptacion : f.aceptacion)}</dd></dl>{f.estado !== 'no_aceptada' && f.episodios.map((e, i) => <dl key={i}><dt>{t('carga.episodio').replace('{cantidad}', new Intl.NumberFormat(idioma).format(i + 1))}</dt><dd>{t('carga.episodio.apertura_fecha')}{t('carga.separador')}{fecha(i ? e.apertura_fecha : f.aceptacion)}</dd>{e.cierre_fecha && <><dd>{t('carga.episodio.cierre_fecha')}{t('carga.separador')}{fecha(e.cierre_fecha)}</dd><dd>{nombre(`motivo_finalizacion.${e.motivo_cierre}`)}</dd></>}</dl>)}</section>}
      {inc.filter(x => !x.fila).map((x, i) => <p className="carga-aviso" key={i}>{texto(x.clave)}</p>)}
      <ol className="carga-lista">{filas.map(fila => <li className="carga-fila" key={fila.fila}><dl><div><dt>{t('carga.fila')}</dt><dd>{new Intl.NumberFormat(idioma).format(fila.fila)}</dd></div><div><dt>{t('carga.campo.numero')}</dt><dd>{fila.numero || t('carga.sin_numero')}</dd></div><div><dt>{t('carga.campo.estado')}</dt><dd>{t(estados.includes(fila.estado) ? `carga.estado.${fila.estado}` : 'carga.estado_pendiente')}</dd></div><div><dt>{t('carga.menor')}</dt><dd>{fila.iniciales || t('carga.sin_datos')}</dd></div></dl>{inc.filter(x => x.fila === fila.fila).map((x, i) => <p className="carga-aviso" key={i}>{t(columnas.includes(x.columna) ? `carga.columna.${x.columna}` : 'carga.fila_completa')}{t('carga.separador')}{texto(x.clave)}</p>)}</li>)}</ol>
      {inc.filter(x => x.fila && !filas.some(fila => fila.fila === x.fila)).map((x, i) => <p className="carga-aviso" key={i}>{t('carga.fila_numero').replace('{cantidad}', new Intl.NumberFormat(idioma).format(x.fila))}{t('carga.separador')}{texto(x.clave)}</p>)}
      {informe.avisos?.map((a, i) => <p className="carga-aviso" key={i}>{t('carga.fila_numero').replace('{cantidad}',new Intl.NumberFormat(idioma).format(a.fila))}{t('carga.separador')}{texto(a.clave)}</p>)}
      {!!informe.avisos?.length && <label className="carga-casilla"><input id="carga-confirmar-duplicados" aria-describedby={error === 'carga.duplicados_sin_confirmar' ? 'carga-error-resumen' : undefined} type="checkbox" checked={duplicadosConfirmados} onChange={e => { ponerDuplicadosConfirmados(e.target.checked); ponerError(''); }} disabled={bloqueado} />{t('carga.confirmar_duplicados')}</label>}
      {informe.valida && <p>{t('carga.consecuencia')}</p>}<div className="carga-acciones"><Boton disabled={bloqueado} accion={() => { ponerInforme(undefined); ponerError(''); }}>{t(manual ? 'carga.corregir' : 'carga.cambiar_hoja')}</Boton>{informe.valida && !incierto && <Boton principal disabled={ocupado} accion={confirmar}>{t(manual ? 'carga.confirmar_expediente' : 'carga.confirmar_lote')}</Boton>}</div>
    </section> : opciones && !!opciones.equipos.length && <>
      {manual && <><section className="carga-tarjeta" aria-labelledby="carga-equipo-titulo"><h2 id="carga-equipo-titulo">{t('carga.actas.equipo_titulo')}</h2>{campo('equipo', true, 'text', lista(opciones.equipos))}</section><Actas hayDatos={tieneDatos(f)} equipo={f.equipo} opciones={opciones} csrf={csrf} t={t} disabled={bloqueado} ocupado={ponerLeyendoActas} aplicar={nuevo => { editar(() => nuevo); document.getElementById('carga-estado')?.focus(); }} /></>}
      <section className="carga-tarjeta" aria-labelledby="carga-datos"><h2 id="carga-datos">{t(manual ? 'carga.datos' : 'carga.subida')}</h2><div className="carga-rejilla">{!manual && campo('equipo', true, 'text', lista(opciones.equipos))}
        {manual && <>{campo('estado', true, 'text', estados.map(v => ({ valor: v, etiqueta: t(`carga.estado.${v}`) })))}{campo('tipo', true, 'text', ['anexo3', 'anexo4'].map(v => ({ valor: v, etiqueta: t(`carga.${v}`) })))}{campo('derivacion', true, 'date')}{campo('subprograma', true, 'text', opciones.subprogramas.filter(o => f.tipo !== 'anexo3' || o.codigo === 'RIESGO_SOCIAL').map(o => ({ valor: o.codigo, etiqueta: nombre(o.clave_texto) })))}</>}
      </div>
      {!manual && <><label htmlFor="carga-hoja">{t('carga.elegir_fichero')}</label><input id="carga-hoja" type="file" accept=".csv,.xlsx" disabled={bloqueado} aria-describedby={errores['carga-hoja'] ? 'carga-hoja-error' : 'carga-hoja-limite'} onChange={e => { const file = e.target.files?.[0]; ponerFichero(file); ponerSucio(!!file); ponerInforme(undefined); ponerError(''); ponerErrores({}); }} />{fichero && <p className="carga-nombre-hoja">{fichero.name}</p>}<p id="carga-hoja-limite">{t('carga.limite')}</p>{errores['carga-hoja'] && <p id="carga-hoja-error">{errores['carga-hoja']}</p>}<div className="carga-acciones"><Boton accion={descargar}>{t('carga.descargar')}</Boton><Boton accion={() => navegarRuta('/equipo/expedientes/alta-historica')}>{t('carga.ir_manual')}</Boton></div></>}
      </section>
      {manual && f.estado === 'no_aceptada' && <section className="carga-tarjeta" aria-labelledby="carga-decision"><h2 id="carga-decision">{t('carga.no_aceptacion')}</h2><div className="carga-rejilla">{campo('noAceptacion', true, 'date')}{campo('causa', true, 'text', undefined, true)}</div></section>}
      {manual && f.estado === 'no_aceptada' && <section className="carga-tarjeta" aria-labelledby="carga-nombres"><h2 id="carga-nombres">{t('carga.menores_derivacion')}</h2>{f.menores.map((m, i) => <fieldset key={m.id}><legend>{t('carga.menor_n').replace('{cantidad}', new Intl.NumberFormat(idioma).format(i + 1))}</legend>{m.nombreLeido && <p>{t('carga.nombre_leido')}{t('carga.separador')}{m.nombreLeido}</p>}<div className="carga-rejilla">{(['nombre', 'apellidos'] as const).map(clave => <Campo key={clave} id={`carga-menor-${clave}-${i}`} etiqueta={t(`carga.menor.${clave}`)} valor={m[clave]} cambiar={v => editar(a => ({ ...a, menores: a.menores.map((x, j) => i === j ? { ...x, [clave]: v } : x) }))} disabled={bloqueado} t={t} />)}</div>{m.nombreCompleto && <Boton disabled={bloqueado} accion={() => editar(a => ({ ...a, menores: a.menores.map((x,j) => i===j ? separarNombre(x) : x) }))}>{t('carga.separar_nombre')}</Boton>}{f.menores.length > 1 && <Boton disabled={bloqueado} accion={() => editar(a => ({ ...a, menores: a.menores.filter((_, j) => i !== j) }))}>{t('carga.quitar_menor')}</Boton>}</fieldset>)}<Boton disabled={bloqueado || f.menores.length >= 15} accion={() => editar(a => ({ ...a, menores: [...a.menores, menor()] }))}>{t('carga.anadir_menor')}</Boton></section>}
      {manual && f.estado && f.estado !== 'no_aceptada' && <>
        <section className="carga-tarjeta" aria-labelledby="carga-aceptacion-titulo"><h2 id="carga-aceptacion-titulo">{t('carga.aceptacion')}</h2><div className="carga-rejilla">{campo('numero', false)}{campo('aceptacion', true, 'date')}{campo('municipio', true, 'text', lista(opciones.municipios.filter(o => o.ambito_diputacion && o.equipo_id === f.equipo)))}{campo('centro', true, 'text', lista(opciones.centros))}{campo('puesto', true, 'text', ['TS', 'ES', 'PS'].map(v => ({ valor: v, etiqueta: t(`carga.puesto.${v.toLowerCase()}`) })))}</div>{campo('descripcion', true, 'text', undefined, true)}<p>{t('carga.numero_desconocido')}</p></section>
        <section className="carga-tarjeta" aria-labelledby="carga-familia"><h2 id="carga-familia">{t('carga.familia')}</h2><div className="carga-rejilla">{campo('familiaNombre', false)}{campo('direccion', false)}{campo('telefono', false, 'tel')}{campo('email', false, 'email')}</div>
          {f.adultos.map((a,i) => <fieldset key={a.id}><legend>{t('carga.adulto_n').replace('{cantidad}',new Intl.NumberFormat(idioma).format(i+1))}</legend>{a.nombreLeido && <p>{t('carga.nombre_leido')}{t('carga.separador')}{a.nombreLeido}</p>}<div className="carga-rejilla">{(['nombre','apellidos','papel','sexo','fecha_nacimiento'] as const).filter(clave => clave !== 'apellidos' || !a.nombreCompleto).map(clave => <Campo key={clave} id={`carga-adulto-${clave}-${i}`} etiqueta={t(clave === 'nombre' && a.nombreCompleto ? 'carga.nombre_completo' : `carga.adulto.${clave}`)} valor={a[clave]} obligatorio={['nombre','papel','sexo'].includes(clave)} tipo={clave === 'fecha_nacimiento' ? 'date' : 'text'} opciones={clave === 'papel' ? Object.keys(opciones.textos).filter(k => k.startsWith('papel_parental.')).map(k => ({valor:k.split('.')[1],etiqueta:nombre(k)})) : clave === 'sexo' ? ['H','M'].map(v => ({valor:v,etiqueta:t(`carga.adulto.sexo.${v.toLowerCase()}`)})) : undefined} cambiar={v => editar(f => ({...f,adultos:f.adultos.map((x,j) => i===j ? {...x,[clave]:v} : x)}))} disabled={bloqueado} error={errores[`carga-adulto-${clave}-${i}`]} max={clave === 'nombre' || clave === 'apellidos' ? 254 : 4000} t={t} />)}</div>{a.nombreCompleto && <Boton disabled={bloqueado} accion={() => editar(f => ({...f,adultos:f.adultos.map((x,j) => i===j ? separarNombre(x) : x)}))}>{t('carga.separar_nombre')}</Boton>}<Boton disabled={bloqueado} accion={() => editar(f => ({...f,adultos:f.adultos.filter((_,j) => i!==j)}))}>{t('carga.quitar_adulto')}</Boton></fieldset>)}
          <Boton disabled={bloqueado || f.adultos.length >= 100} accion={() => editar(f => ({...f,adultos:[...f.adultos,adulto()]}))}>{t('carga.anadir_adulto')}</Boton>
          {f.menores.map((m, i) => <fieldset key={m.id}><legend>{t('carga.menor_n').replace('{cantidad}', new Intl.NumberFormat(idioma).format(i + 1))}</legend>{m.nombreLeido && <p>{t('carga.nombre_leido')}{t('carga.separador')}{m.nombreLeido}</p>}<div className="carga-rejilla">{(['nombre', 'apellidos', 'sexo', 'fecha_nacimiento'] as const).filter(clave => clave !== 'apellidos' || !m.nombreCompleto).map(clave => <Campo key={clave} id={`carga-menor-${clave === 'nombre' || clave === 'sexo' ? clave : clave}-${i}`} etiqueta={t(clave === 'nombre' && m.nombreCompleto ? 'carga.nombre_completo' : `carga.menor.${clave}`)} valor={m[clave]} obligatorio={clave === 'nombre' || clave === 'sexo'} tipo={clave === 'fecha_nacimiento' ? 'date' : 'text'} opciones={clave === 'sexo' ? ['H', 'M'].map(v => ({ valor: v, etiqueta: t(`carga.sexo.${v.toLowerCase()}`) })) : undefined} cambiar={v => editar(a => ({ ...a, menores: a.menores.map((x, j) => i === j ? { ...x, [clave]: v } : x) }))} disabled={bloqueado} error={errores[`carga-menor-${clave}-${i}`]} t={t} />)}</div>{m.nombreCompleto && <Boton disabled={bloqueado} accion={() => editar(a => ({ ...a, menores: a.menores.map((x,j) => i===j ? separarNombre(x) : x) }))}>{t('carga.separar_nombre')}</Boton>}{f.menores.length > 1 && <Boton disabled={bloqueado} accion={() => editar(a => ({ ...a, menores: a.menores.filter((_, j) => i !== j) }))}>{t('carga.quitar_menor')}</Boton>}</fieldset>)}
          <Boton disabled={bloqueado || f.menores.length >= 15} accion={() => editar(a => ({ ...a, menores: [...a.menores, menor()] }))}>{t('carga.anadir_menor')}</Boton>
        </section>
        <section className="carga-tarjeta" aria-labelledby="carga-historia"><h2 id="carga-historia">{t('carga.historia')}</h2>{f.episodios.map((e, i) => <fieldset key={i}><legend>{t('carga.episodio').replace('{cantidad}', new Intl.NumberFormat(idioma).format(i + 1))}</legend><div className="carga-rejilla">{(['apertura_fecha', 'cierre_fecha', 'motivo_cierre', 'motivo_acta_texto', 'motivo_reapertura'] as const).filter(c => i || c !== 'motivo_reapertura').map(clave => {
          const id = `carga-${clave === 'apertura_fecha' ? 'apertura' : clave === 'cierre_fecha' ? 'cierre' : clave === 'motivo_cierre' ? 'motivo' : clave === 'motivo_reapertura' ? 'reapertura' : 'acta'}-${i}`;
          return <Campo key={clave} id={id} etiqueta={t(`carga.episodio.${clave}`)} valor={i === 0 && clave === 'apertura_fecha' ? f.aceptacion : e[clave]} tipo={clave.endsWith('_fecha') ? 'date' : 'text'} obligatorio={clave === 'apertura_fecha' || clave === 'motivo_reapertura' || clave === 'cierre_fecha' && (f.estado === 'cerrada' || i < f.episodios.length - 1) || clave === 'motivo_cierre' && !!e.cierre_fecha} opciones={clave === 'motivo_cierre' ? Object.keys(opciones.textos).filter(k => k.startsWith('motivo_finalizacion.')).map(k => ({ valor: k.split('.')[1], etiqueta: nombre(k) })) : undefined} disabled={bloqueado || i === 0 && clave === 'apertura_fecha'} error={errores[id]} cambiar={v => editar(a => ({ ...a, episodios: a.episodios.map((x, j) => i === j ? { ...x, [clave]: v } : x) }))} t={t} />;
        })}</div>{i > 0 && <Boton disabled={bloqueado} accion={() => editar(a => ({ ...a, episodios: a.episodios.filter((_, j) => i !== j) }))}>{t('carga.quitar_episodio')}</Boton>}</fieldset>)}<Boton disabled={bloqueado || f.episodios.length >= 100} accion={() => editar(a => ({ ...a, episodios: [...a.episodios, episodio()] }))}>{t('carga.anadir_episodio')}</Boton></section>
      </>}
      <p>{t('carga.sin_guardar')}</p><div className="carga-acciones"><Boton principal disabled={bloqueado} accion={manual ? validarManual : subirHoja}>{t(manual ? 'carga.revisar_manual' : 'carga.revisar_hoja')}</Boton></div>
    </>}
    {ayuda && <Ayuda t={t} cerrar={() => ponerAyuda(false)} />}
  </div>;
}
export function AltaHistorica(props: PropsPantalla) { return <Carga {...props} manual />; }
export function CargaHoja(props: PropsPantalla) { return <Carga {...props} manual={false} />; }
