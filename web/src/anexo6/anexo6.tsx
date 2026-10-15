import { useEffect, useRef, useState, type ComponentType } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useCambios } from '../cambios';
import { Boton, Confirmar } from '../salidas/componentes';
import { comprobar, leer, Fallo } from '../salidas/api';
import './anexo6.css';
import { AyudaAnexo6 } from './ayuda';

type Salida = { id: string; servicio_destino: string; servicio_texto?: string; motivo: string; estado: string; fecha?: string; version: number };
type Ficha = { numero: string; estado: string; version: number; salidas_servicio: Salida[]; emisiones: { codigo: string; version_plantilla?: string }[] };
type Destinos = { servicios: { codigo: string; clave_texto: string }[]; textos: Record<string, string> };
type Operacion = { accion: 'guardar' | 'confirmar' | 'descargar'; ruta: string; cuerpo: string; clave: string };
const expedienteActual = () => new URLSearchParams(location.search).get('id') ?? location.pathname.split('/')[3] ?? '';

// El enganche común añade la acción sin cambiar pantallas de otros módulos.
export function AccesoAnexo6({ componente: Original, ...props }: PropsPantalla & { componente: ComponentType<PropsPantalla> }) {
  const etiquetaRuta = props.t('anexo6.navegacion');
  const ruta = `/equipo/anexo6?id=${encodeURIComponent(expedienteActual())}`;
  return <><nav className="anexo6-enlace" aria-label={etiquetaRuta}><a href={ruta} onClick={e => { e.preventDefault(); props.navegarRuta(ruta); }}>{props.t('anexo6.abrir')}</a></nav><Original {...props} /></>;
}

export function Anexo6({ t, idioma, csrf, navegarRuta }: PropsPantalla) {
  const id = expedienteActual();
  const [datos, ponerDatos] = useState<{ ficha: Ficha; destinos: Destinos }>();
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState('');
  const [aviso, ponerAviso] = useState('');
  const [versionPlantilla, ponerVersionPlantilla] = useState<Record<string, string>>({});
  const [servicio, ponerServicio] = useState('');
  const [motivo, ponerMotivo] = useState('');
  const [seleccion, ponerSeleccion] = useState('');
  const [fecha, ponerFecha] = useState('');
  const [confirmacion, ponerConfirmacion] = useState(false);
  const [ayuda, ponerAyuda] = useState(false);
  const [errores, ponerErrores] = useState<string[]>([]);
  const [ocupado, ponerOcupado] = useState(false);
  const [pendiente, ponerPendiente] = useState<Operacion>();
  const [intento, ponerIntento] = useState(0);
  const controlador = useRef<AbortController | null>(null);
  const enviando = useRef(false);
  const resumen = useRef<HTMLElement>(null);
  const { ponerSucio, solicitarSalida } = useCambios();
  useEffect(() => {
    const c = new AbortController(); controlador.current = c; ponerDatos(undefined); ponerCargando(true); ponerOcupado(false); enviando.current = false;
    void Promise.all([leer<Ficha>(`/api/equipo/expedientes/${encodeURIComponent(id)}`, c.signal), leer<Destinos>(`/api/anexo6/opciones?idioma=${encodeURIComponent(idioma)}`, c.signal)])
      .then(([ficha, destinos]) => { if (!c.signal.aborted) ponerDatos({ ficha, destinos }); })
      .catch((e: unknown) => { if (!c.signal.aborted) ponerError(claveError(e)); })
      .finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    return () => c.abort();
  }, [id, idioma, intento]);
  useEffect(() => { ponerSucio(!!(servicio || motivo || fecha || pendiente)); }, [servicio, motivo, fecha, pendiente, ponerSucio]);
  useEffect(() => () => ponerSucio(false), [ponerSucio]);
  useEffect(() => { document.title = `${t('anexo6.titulo')} ${datos?.ficha.numero ?? ''}`.trim(); }, [t, datos?.ficha.numero]);
  useEffect(() => { if (errores.length) resumen.current?.focus(); }, [errores]);
  function claveError(e: unknown) {
    if (!(e instanceof Fallo)) return 'anexo6.error';
    if (e.estado === 409) return 'anexo6.conflicto';
    if (e.estado === 401 || e.estado === 403) return 'anexo6.sin_permiso';
    if (e.estado === 404) return 'anexo6.no_encontrado';
    if (e.estado === 400 || e.estado === 422) return 'anexo6.revisar';
    return 'anexo6.error';
  }
  const salida = datos?.ficha.salidas_servicio.find(s => s.id === seleccion);
  const activa = !!datos && ['aceptada', 'intervencion', 'en_intervencion', 'propuesta_cierre'].includes(datos.ficha.estado);
  const bloqueado = cargando || ocupado || !!pendiente || ['anexo6.conflicto', 'anexo6.sin_permiso'].includes(error);
  const partesHoy = new Intl.DateTimeFormat(idioma, { timeZone: 'Europe/Madrid', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date());
  const hoy = ['year', 'month', 'day'].map(k => partesHoy.find(p => p.type === k)?.value).join('-');
  const fechaVisible = (dia: string) => new Intl.DateTimeFormat(idioma, { timeZone: 'UTC', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(`${dia}T12:00:00Z`));
  const etiqueta = (s: Salida) => s.servicio_texto || datos?.destinos.textos[`servicio_destino.${s.servicio_destino}`] || t('anexo6.destino_sin_etiqueta');
  async function enviar(op: Operacion) {
    if (enviando.current) return;
    enviando.current = true; ponerPendiente(op); ponerOcupado(true); ponerError(''); ponerAviso('');
    const signal = controlador.current?.signal;
    try {
      const r = await fetch(op.ruta, { method: 'POST', signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': op.clave }, body: op.cuerpo });
      await comprobar(r);
      if (op.accion === 'descargar') {
        const version = r.headers.get('X-Plantilla-Version');
        if (!r.headers.get('Content-Type')?.includes('wordprocessingml') || !r.headers.get('X-Emision-ID') || !version) throw new Error('anexo6.error');
        const b = await r.blob(); if (!b.size) throw new Error('anexo6.error');
        const url = URL.createObjectURL(b); const a = document.createElement('a');
        a.href = url; a.download = `${datos?.ficha.numero}-${t('anexo6.archivo')}.docx`; a.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
        ponerVersionPlantilla(actual => ({ ...actual, [seleccion]: version }));
      } else {
        const recibo = await r.json() as Salida & { version_expediente: number };
        if (!recibo.id || !Number.isSafeInteger(recibo.version) || !Number.isSafeInteger(recibo.version_expediente)) throw new Error('anexo6.error');
        ponerSeleccion(recibo.id);
        if (op.accion === 'guardar') { ponerServicio(''); ponerMotivo(''); }
        ponerFecha(''); ponerErrores([]);
      }
      ponerPendiente(undefined); ponerAviso(`anexo6.hecho.${op.accion}`); ponerIntento(n => n + 1);
    } catch (e: unknown) {
      if (signal?.aborted) return;
      ponerError(claveError(e));
      if (!(e instanceof Fallo) || e.estado >= 500) ponerPendiente(op); else ponerPendiente(undefined);
    } finally { enviando.current = false; if (!signal?.aborted) ponerOcupado(false); }
  }
  function operar(accion: Operacion['accion']) {
    if (!datos || bloqueado || !activa) return;
    const base = `/api/expedientes/${encodeURIComponent(id)}`;
    const cuerpo = accion === 'guardar' ? { tipo: 'otro_servicio', version: datos.ficha.version, servicio_destino: datos.destinos.servicios.length ? servicio : '', ...(datos.destinos.servicios.length ? {} : { servicio_texto: servicio.trim() }), motivo: motivo.trim() }
      : accion === 'confirmar' ? { tipo: 'derivacion_otro_servicio', salida_id: salida?.id, version: datos.ficha.version, version_salida: salida?.version, fecha }
      : { salida_id: salida?.id, idioma };
    ponerConfirmacion(false);
    void enviar({ accion, ruta: accion === 'guardar' ? `${base}/derivaciones-salida` : accion === 'confirmar' ? `${base}/hitos` : `${base}/documentos/anexo6`, cuerpo: JSON.stringify(cuerpo), clave: crypto.randomUUID() });
  }
  function validar(accion: 'guardar' | 'confirmar') {
    const campos = accion === 'guardar' ? [...(!servicio.trim() ? ['servicio'] : []), ...(!motivo.trim() ? ['motivo'] : [])]
      : (!/^\d{4}-\d{2}-\d{2}$/.test(fecha) || !Number.isFinite(Date.parse(fecha)) || fecha > hoy ? ['fecha'] : []);
    ponerErrores(campos); if (!campos.length) { if (accion === 'confirmar') ponerConfirmacion(true); else operar('guardar'); }
  }
  const volver = `/equipo/expediente?id=${encodeURIComponent(id)}&pestana=salidas`;
  const descripcionError = (campo: string) => errores.includes(campo) ? `anexo6-error-${campo}` : undefined;
  const recargar = () => { ponerError(''); ponerIntento(n => n + 1); };
  const etiquetaRuta = t('anexo6.navegacion');
  return <section className="anexo6" aria-busy={cargando || ocupado}>
    <nav aria-label={etiquetaRuta}><a href={volver} onClick={e => { e.preventDefault(); navegarRuta(volver); }}>{t('anexo6.volver')}</a></nav>
    <header className="anexo6-acciones"><h1>{t('anexo6.titulo')} {datos?.ficha.numero}</h1><Boton accion={() => ponerAyuda(true)}>{t('anexo6.ayuda')}</Boton></header>
    <p role="status">{cargando ? t('anexo6.cargando') : ocupado ? t('anexo6.guardando') : aviso ? t(aviso) : ''}</p>
    {(error || pendiente && !ocupado) && <section role="alert" className="pendiente"><p>{t(error || 'anexo6.error')}</p>{pendiente ? <Boton disabled={ocupado} accion={() => void enviar(pendiente)}>{t('anexo6.recuperar')}</Boton> : <Boton disabled={ocupado} accion={recargar}>{t(error === 'anexo6.conflicto' ? 'anexo6.ver_cambios' : 'anexo6.reintentar')}</Boton>}</section>}
    {!!errores.length && <section ref={resumen} tabIndex={-1} role="alert" className="pendiente"><h2>{t('anexo6.faltan')}</h2><ul>{errores.map(c => <li key={c}><a href={`#anexo6-${c}`} onClick={e => { e.preventDefault(); document.getElementById(`anexo6-${c}`)?.focus(); }}>{t(`anexo6.${c}`)}</a></li>)}</ul></section>}
    {datos && <>
      <p>{t('anexo6.consecuencia')}</p>{activa && salida && <Boton disabled={bloqueado} accion={() => solicitarSalida(() => { ponerSeleccion(''); ponerFecha(''); ponerErrores([]); ponerAviso(''); })}>{t('anexo6.preparar_otra')}</Boton>}
      {!activa && <p className="pendiente">{t('anexo6.inactiva')}</p>}
      {activa && !salida && <section className="anexo6-tarjeta"><h2>{t('anexo6.preparar')}</h2>
        {!datos.destinos.servicios.length && <p>{t('anexo6.sin_destinos')}</p>}<form noValidate onSubmit={e => { e.preventDefault(); validar('guardar'); }}>
          <div className="anexo6-campo"><label htmlFor="anexo6-servicio">{t('anexo6.servicio')}</label>{datos.destinos.servicios.length ? <select id="anexo6-servicio" value={servicio} disabled={bloqueado} aria-invalid={errores.includes('servicio')} aria-describedby={descripcionError('servicio')} onChange={e => ponerServicio(e.target.value)}><option value="">{t('anexo6.elegir')}</option>{datos.destinos.servicios.map(s => <option key={s.codigo} value={s.codigo}>{datos.destinos.textos[s.clave_texto] || t('anexo6.destino_sin_etiqueta')}</option>)}</select> : <input id="anexo6-servicio" type="text" maxLength={254} value={servicio} disabled={bloqueado} aria-invalid={errores.includes('servicio')} aria-describedby={descripcionError('servicio')} onChange={e => ponerServicio(e.target.value)} />}{errores.includes('servicio') && <p id="anexo6-error-servicio" className="pendiente">{t('anexo6.obligatorio')}</p>}</div>
          <div className="anexo6-campo"><label htmlFor="anexo6-motivo">{t('anexo6.motivo')}</label><textarea id="anexo6-motivo" value={motivo} maxLength={4000} disabled={bloqueado} aria-invalid={errores.includes('motivo')} aria-describedby={descripcionError('motivo')} onChange={e => ponerMotivo(e.target.value)} />{errores.includes('motivo') && <p id="anexo6-error-motivo" className="pendiente">{t('anexo6.obligatorio')}</p>}</div>
          <button className="boton-primario" disabled={bloqueado} type="submit">{t('anexo6.guardar')}</button>
        </form>
      </section>}
      <section className="anexo6-tarjeta"><h2>{t('anexo6.registradas')}</h2>{!datos.ficha.salidas_servicio.length ? <p>{t('anexo6.vacio')}</p> : <ul className="anexo6-lista">{datos.ficha.salidas_servicio.map(s => <li key={s.id}><h3>{etiqueta(s)}</h3><p>{s.motivo}</p><p>{s.fecha ? `${t('anexo6.fecha_registrada')} ${fechaVisible(s.fecha)}` : t('anexo6.fecha_pendiente')}</p><Boton disabled={bloqueado} accion={() => solicitarSalida(() => { ponerSeleccion(s.id); ponerFecha(''); ponerServicio(''); ponerMotivo(''); ponerErrores([]); ponerAviso(''); })}>{t('anexo6.revisar_salida')}</Boton></li>)}</ul>}</section>
      {salida && <section className="anexo6-tarjeta" aria-labelledby="anexo6-revision"><h2 id="anexo6-revision">{t('anexo6.revision')} {etiqueta(salida)}</h2>
        <p>{t('anexo6.word_consecuencia')}</p>
        {activa && <Boton disabled={bloqueado} accion={() => operar('descargar')}>{t('anexo6.descargar')}</Boton>}
        {versionPlantilla[seleccion] && <p>{t('anexo6.plantilla')} {versionPlantilla[seleccion]}</p>}
        {salida.fecha ? <p>{t('anexo6.fecha_registrada')} {fechaVisible(salida.fecha)}</p> : activa && <form noValidate onSubmit={e => { e.preventDefault(); validar('confirmar'); }}>
          <p>{t('anexo6.fecha_consecuencia')}</p><div className="anexo6-campo"><label htmlFor="anexo6-fecha">{t('anexo6.fecha')}</label><input id="anexo6-fecha" type="date" value={fecha} max={hoy} disabled={bloqueado} aria-invalid={errores.includes('fecha')} aria-describedby={descripcionError('fecha')} onChange={e => ponerFecha(e.target.value)} />{errores.includes('fecha') && <p id="anexo6-error-fecha" className="pendiente">{t('anexo6.fecha_invalida')}</p>}</div>
          <button className="boton-primario" type="submit" disabled={bloqueado}>{t('anexo6.confirmar')}</button>
        </form>}
      </section>}
    </>}
    {confirmacion && <Confirmar t={t} titulo={t('anexo6.confirmar')} texto={t('anexo6.confirmacion')} cancelar={() => ponerConfirmacion(false)} aceptar={() => operar('confirmar')} />}
    {ayuda && <AyudaAnexo6 t={t} cerrar={() => ponerAyuda(false)} />}
  </section>;
}
