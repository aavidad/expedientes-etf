import { useEffect, useRef, useState } from 'react';
import { useCambios } from '../cambios';
import type { PropsPantalla } from '../registro-pantallas';
import { Boton, Campo } from './componentes';
import './carga.css';

type Plazo = { fecha_cierre: string; version: number; dias_duplicados: number; abierta: boolean };
type Lote = { referencia: number; id: string; equipo_id: string; cargada_en: string; casos: number; anulado: boolean };

export function PlazoCarga({ t, idioma, csrf }: PropsPantalla) {
  const { ponerSucio } = useCambios();
  const [dias, ponerDias] = useState('');
  const [plazo, ponerPlazo] = useState<Plazo>();
  const [fecha, ponerFecha] = useState('');
  const [cargando, ponerCargando] = useState(true);
  const [ocupado, ponerOcupado] = useState(false);
  const [error, ponerError] = useState('');
  const [guardado, ponerGuardado] = useState(false);
  const inicializado = useRef(false);
  const [intento, ponerIntento] = useState(0);
  const peticion = useRef<AbortController | undefined>(undefined);
  const resumen = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const abortar = new AbortController(); ponerCargando(true); ponerError('');
    void fetch('/api/carga-2026/plazo', { signal: abortar.signal }).then(async r => {
      const d = await r.json(); if (!r.ok) throw new Error(d.error?.clave ?? d.error ?? 'carga.error'); return d as Plazo;
    }).then(p => { ponerPlazo(p); if (!inicializado.current) { ponerFecha(p.fecha_cierre); ponerDias(String(p.dias_duplicados)); inicializado.current = true; } }).catch(e => { if (!abortar.signal.aborted) ponerError(e instanceof Error ? e.message : 'carga.error'); }).finally(() => { if (!abortar.signal.aborted) ponerCargando(false); });
    return () => { abortar.abort(); peticion.current?.abort(); };
  }, [intento]);
  useEffect(() => { if (error) resumen.current?.focus(); }, [error]);
  async function guardar() {
    if (!fecha || !plazo) { ponerError('carga.control_invalido'); return; }
    if (!/^\d{1,3}$/.test(dias) || Number(dias) > 366) { ponerError('carga.dias_invalidos'); return; }
    const abortar = new AbortController(); peticion.current = abortar; ponerOcupado(true); ponerError(''); ponerGuardado(false);
    try {
      const r = await fetch('/api/carga-2026/plazo', { method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ fecha_cierre: fecha, version: plazo.version, dias_duplicados: Number(dias) }), signal: abortar.signal });
      const d = await r.json(); if (!r.ok) throw new Error(d.error?.clave ?? d.error ?? 'carga.error');
      ponerPlazo(d as Plazo); ponerGuardado(true); ponerSucio(false);
    } catch (e) { if (!abortar.signal.aborted) ponerError(e instanceof Error ? e.message : 'carga.error'); }
    finally { if (!abortar.signal.aborted) ponerOcupado(false); }
  }
  const textoError = t(error) === error ? t('carga.error') : t(error);
  const fechaTexto = plazo?.fecha_cierre ? new Intl.DateTimeFormat(idioma, { day:'2-digit',month:'2-digit',year:'numeric',timeZone: 'UTC' }).format(new Date(`${plazo.fecha_cierre}T12:00:00Z`)) : t('carga.sin_fecha_cierre');
  return <div className="carga"><header className="carga-cabecera"><h1>{t('carga.control_titulo')}</h1></header>
    {cargando && <p role="status">{t('carga.cargando')}</p>}
    {error && <div ref={resumen} tabIndex={-1} role="alert" className="carga-error"><p>{textoError}</p><Boton disabled={ocupado} accion={() => { ponerGuardado(false); ponerIntento(i => i + 1); }}>{t('carga.actualizar_plazo')}</Boton></div>}
    {plazo && !cargando && <section className="carga-tarjeta" aria-labelledby="carga-plazo-actual"><h2 id="carga-plazo-actual">{t('carga.plazo_actual')}</h2><p>{fechaTexto}</p><p>{t('carga.plazo_consecuencia')}</p>
      <Campo id="carga-fecha-cierre" etiqueta={t('carga.fecha_cierre')} tipo="date" valor={fecha} cambiar={v => { ponerFecha(v); ponerSucio(true); ponerGuardado(false); }} obligatorio disabled={ocupado} t={t} error={error === 'carga.control_invalido' ? textoError : undefined} />
      <Campo id="carga-dias-duplicados" etiqueta={t('carga.dias_duplicados')} tipo="number" valor={dias} cambiar={v => { ponerDias(v); ponerSucio(true); ponerGuardado(false); }} obligatorio disabled={ocupado} t={t} error={error === 'carga.dias_invalidos' ? textoError : undefined} />
      <p>{t('carga.dias_rango')}</p>
      <div className="carga-acciones"><Boton principal disabled={ocupado} accion={() => void guardar()}>{t('carga.guardar_plazo')}</Boton></div>
      {ocupado && <p role="status">{t('carga.guardando')}</p>}{guardado && <p role="status">{t('carga.plazo_guardado')}</p>}
    </section>}
  </div>;
}

export function LotesCarga({ t, idioma, csrf }: PropsPantalla) {
  const [lotes, ponerLotes] = useState<Lote[]>();
  const [equipos, ponerEquipos] = useState<Record<string, string>>({});
  const ultimoIdioma = useRef(idioma);
  const [desplazamiento, ponerDesplazamiento] = useState(0);
  const [hayMas, ponerHayMas] = useState(false);
  const [leyendo, ponerLeyendo] = useState(false);
  const [intento, ponerIntento] = useState(0);
  const [error, ponerError] = useState('');
  const [elegido, ponerElegido] = useState<Lote>();
  const [ocupado, ponerOcupado] = useState(false);
  const [anulado, ponerAnulado] = useState(false);
  const [incierto, ponerIncierto] = useState(false);
  const titulo = useRef<HTMLHeadingElement>(null);
  const estado = useRef<HTMLParagraphElement>(null);
  const refrescoSolicitado = useRef(false);
  const peticion = useRef<AbortController | undefined>(undefined);
  const resumen = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (ultimoIdioma.current !== idioma) { ultimoIdioma.current = idioma; ponerLotes(undefined); ponerDesplazamiento(0); if (desplazamiento) return; }
    const abortar = new AbortController(); if (!desplazamiento) ponerLotes(undefined); ponerLeyendo(true); ponerError('');
    void Promise.all([fetch(`/api/carga-2026/lotes?desplazamiento=${desplazamiento}`, { signal: abortar.signal }), fetch(`/api/carga-2026/opciones?idioma=${encodeURIComponent(idioma)}`, { signal: abortar.signal })]).then(async ([r, o]) => {
      const d = await r.json(); if (!r.ok || !o.ok) throw new Error(d.error?.clave ?? d.error ?? 'carga.error');
      const opciones = await o.json() as { equipos: { id: string; clave_texto: string }[]; textos: Record<string, string> };
      ponerEquipos(Object.fromEntries(opciones.equipos.map(e => [e.id, opciones.textos[e.clave_texto] || t('carga.etiqueta_pendiente')]))); if (!Array.isArray(d)) throw new Error('carga.error'); ponerLotes(v => desplazamiento ? [...(v ?? []), ...d as Lote[]] : d as Lote[]); ponerHayMas(d.length === 100); ponerIncierto(false); if (refrescoSolicitado.current) { titulo.current?.focus(); refrescoSolicitado.current = false; }
    }).catch(e => { if (!abortar.signal.aborted) ponerError(e instanceof Error ? e.message : 'carga.error'); }).finally(() => { if (!abortar.signal.aborted) ponerLeyendo(false); });
    return () => { abortar.abort(); peticion.current?.abort(); };
  }, [intento, idioma, desplazamiento]);
  useEffect(() => { if (error && !elegido) resumen.current?.focus(); else if (anulado && !elegido) estado.current?.focus(); }, [error, elegido, anulado]);
  async function anular(lote: Lote) {
    const abortar = new AbortController(); peticion.current = abortar; ponerOcupado(true); ponerError(''); ponerAnulado(false);
    let rechazoConfirmado = false;
    try {
      const r = await fetch(`/api/carga-2026/lotes/${encodeURIComponent(lote.id)}/anular`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ confirmar: true }), signal: abortar.signal });
      const d = await r.json(); if (!r.ok) { rechazoConfirmado = typeof d.error === 'string' || typeof d.error?.clave === 'string'; throw new Error(d.error?.clave ?? d.error ?? 'carga.error'); }
      if (d.anulado !== true) throw new Error('carga.error');
      ponerLotes(v => v?.map(x => x.id === lote.id ? { ...x, anulado: true } : x)); ponerAnulado(true);
    } catch (e) { if (!abortar.signal.aborted) { ponerIncierto(!rechazoConfirmado); ponerError(rechazoConfirmado && e instanceof Error ? e.message : 'carga.anulacion_incierta'); } }
    finally { if (!abortar.signal.aborted) { ponerOcupado(false); ponerElegido(undefined); } }
  }
  const fecha = (v: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(v));
  return <div className="carga"><header className="carga-cabecera"><h1 ref={titulo} tabIndex={-1}>{t('carga.lotes_titulo')}</h1><Boton disabled={ocupado} accion={() => { refrescoSolicitado.current = true; ponerDesplazamiento(0); ponerIntento(i => i + 1); }}>{t('carga.actualizar_lotes')}</Boton></header>
    {error && <div ref={resumen} tabIndex={-1} role="alert" className="carga-error"><p>{t(error) === error ? t('carga.error') : t(error)}</p></div>}
    {!lotes && !error && <p role="status">{t('carga.cargando')}</p>}{lotes?.length === 0 && <p>{t('carga.lotes_vacios')}</p>}{anulado && <p ref={estado} tabIndex={-1} role="status">{t('carga.lote_anulado')}</p>}
    <ol className="carga-lista">{lotes?.map(lote => <li key={lote.id} className="carga-fila"><h2>{t('carga.lote_referencia').replace('{numero}',new Intl.NumberFormat(idioma).format(lote.referencia))}</h2><p>{equipos[lote.equipo_id] || t('carga.etiqueta_pendiente')}</p><dl><div><dt>{t('carga.fecha_carga')}</dt><dd>{fecha(lote.cargada_en)}</dd></div><div><dt>{t('carga.casos_lote')}</dt><dd>{new Intl.NumberFormat(idioma).format(lote.casos)}</dd></div><div><dt>{t('carga.campo.estado')}</dt><dd>{t(lote.anulado ? 'carga.estado_anulado' : 'carga.estado_cargado')}</dd></div></dl>{!lote.anulado && <Boton disabled={ocupado || incierto} accion={() => ponerElegido(lote)}>{t('carga.anular_lote')}</Boton>}</li>)}</ol>
    {hayMas && <Boton disabled={ocupado || leyendo || incierto} accion={() => { ponerDesplazamiento(lotes?.length ?? 0); ponerIntento(i => i + 1); }}>{t('carga.mas_lotes')}</Boton>}
    {leyendo && !!lotes && <p role="status">{t('carga.cargando')}</p>}
    {elegido && <ConfirmarAnulacion t={t} detalle={t('carga.lote_referencia').replace('{numero}',new Intl.NumberFormat(idioma).format(elegido.referencia)) + t('carga.separador') + t(new Intl.PluralRules(idioma).select(elegido.casos) === 'one' ? 'carga.lote_detalle_uno' : 'carga.lote_detalle').replace('{equipo}',equipos[elegido.equipo_id] || t('carga.etiqueta_pendiente')).replace('{fecha}',fecha(elegido.cargada_en)).replace('{cantidad}',new Intl.NumberFormat(idioma).format(elegido.casos))} ocupado={ocupado} cancelar={() => ponerElegido(undefined)} aceptar={() => void anular(elegido)} />}
  </div>;
}
function ConfirmarAnulacion({ t, detalle, ocupado, cancelar, aceptar }: { t: (k: string) => string; detalle: string; ocupado: boolean; cancelar: () => void; aceptar: () => void }) {
  const dialogo = useRef<HTMLDialogElement>(null); const foco = useRef<HTMLButtonElement>(null); const [confirmado, ponerConfirmado] = useState(false); const [error, ponerError] = useState(false);
  useEffect(() => { const antes = document.activeElement as HTMLElement | null; dialogo.current?.showModal(); foco.current?.focus(); return () => { dialogo.current?.close(); if (antes?.isConnected) antes.focus(); }; }, []);
  return <dialog ref={dialogo} className="carga" aria-labelledby="carga-anular-titulo" onCancel={e => { e.preventDefault(); if (!ocupado) cancelar(); }}><h2 id="carga-anular-titulo">{t('carga.anular_lote')}</h2><p>{detalle}</p><p>{t('carga.anular_consecuencia')}</p><label className="carga-casilla"><input type="checkbox" checked={confirmado} disabled={ocupado} aria-describedby={error ? 'carga-anular-error' : undefined} onChange={e => { ponerConfirmado(e.target.checked); ponerError(false); }} />{t('carga.confirmar_anulacion')}</label>{error && <p id="carga-anular-error" role="alert" className="carga-aviso">{t('carga.marcar_confirmacion')}</p>}{ocupado && <p role="status">{t('carga.guardando')}</p>}<div className="carga-acciones"><button ref={foco} type="button" disabled={ocupado} onClick={cancelar}>{t('carga.conservar_lote')}</button><Boton principal disabled={ocupado} accion={() => { if (!confirmado) ponerError(true); else aceptar(); }}>{t('carga.anular_lote')}</Boton></div></dialog>;
}

export function EstadoPlazoCarga({ t, idioma }: Pick<PropsPantalla, 't' | 'idioma'>) {
  const [plazo, ponerPlazo] = useState<Plazo>(); const [error, ponerError] = useState(false);
  useEffect(() => { const abortar = new AbortController(); void fetch('/api/carga-2026/plazo', { signal: abortar.signal }).then(async r => { if (!r.ok) throw new Error(); return r.json() as Promise<Plazo>; }).then(ponerPlazo).catch(() => { if (!abortar.signal.aborted) ponerError(true); }); return () => abortar.abort(); }, []);
  const fecha = plazo?.fecha_cierre ? new Intl.DateTimeFormat(idioma, { day:'2-digit',month:'2-digit',year:'numeric',timeZone: 'UTC' }).format(new Date(`${plazo.fecha_cierre}T12:00:00Z`)) : t('carga.sin_fecha_cierre');
  return <p className="carga-info" role="status">{error ? t('carga.error_plazo') : plazo ? t(plazo.abierta ? 'carga.cierre_carga' : 'carga.carga_cerrada_estado').replace('{fecha}', fecha) : t('carga.cargando')}</p>;
}
