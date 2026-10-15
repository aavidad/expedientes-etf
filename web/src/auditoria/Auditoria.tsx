import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useButton } from 'react-aria';
import type { PropsPantalla } from '../registro-pantallas';
import './auditoria.css';

type Usuario = { id: string; cuenta: string; iniciales: string };
const tipos = ['identidad', 'catalogo', 'paquete', 'expediente', 'evaluacion', 'documento', 'actividad', 'estadistica', 'auditoria'] as const;
function Descargar({ ocupado, t }: { ocupado: boolean; t: PropsPantalla['t'] }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ type: 'submit', isDisabled: ocupado }, ref);
  return <button {...buttonProps} ref={ref}>{t(ocupado ? 'auditoria.preparando' : 'auditoria.descargar')}</button>;
}

export function Auditoria({ t, csrf, idioma }: PropsPantalla) {
  const [usuarios, ponerUsuarios] = useState<Usuario[]>([]);
  const [cargando, ponerCargando] = useState(true);
 const [errorCarga,ponerErrorCarga]=useState(false);
  const [error, ponerError] = useState('');
  const [intento, ponerIntento] = useState(0);
  const [ocupado, ponerOcupado] = useState(false);
  const [filas, ponerFilas] = useState<number | null>(null);
  const [desde, ponerDesde] = useState('');
  const [hasta, ponerHasta] = useState('');
  const [tipo, ponerTipo] = useState('');
  const [usuario, ponerUsuario] = useState('');
  const [expediente, ponerExpediente] = useState('');
  const [formato, ponerFormato] = useState('csv');
  const [ayuda, ponerAyuda] = useState(false);
  const peticion = useRef<AbortController | null>(null);
  const errorRef = useRef<HTMLParagraphElement>(null);
  const envio = useRef(false);
 const etiquetaAyuda=t('auditoria.ayuda_etiqueta');

  useEffect(() => {
    const c = new AbortController(); ponerCargando(true); ponerError('');ponerErrorCarga(false);
    void fetch('/api/admin/usuarios', { signal: c.signal }).then(async r => {
      if (!r.ok) throw new Error('auditoria.no_disponible');
      const lista = await r.json() as Usuario[];
      if (!Array.isArray(lista)) throw new Error('auditoria.no_disponible');
      if (!c.signal.aborted) ponerUsuarios(lista);
    }).catch(() => { if (!c.signal.aborted){ponerError('auditoria.personas_no_disponibles');ponerErrorCarga(true);} })
      .finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    return () => c.abort();
  }, [intento]);
  useEffect(() => () => peticion.current?.abort(), []);
  useEffect(() => { document.title = t('auditoria.titulo'); }, [t]);
  useEffect(() => { if (error) errorRef.current?.focus(); }, [error]);
  useEffect(() => { ponerFilas(null); }, [desde, hasta, tipo, usuario, expediente, formato]);

  async function descargar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); if (envio.current) return;
    ponerError(''); ponerFilas(null);
    if (!desde || !hasta || hasta < desde || (Date.parse(hasta) - Date.parse(desde)) / 86400000 >= 31) {
      ponerError('auditoria.fechas_invalidas'); document.getElementById('auditoria-desde')?.focus(); return;
    }
    envio.current = true; ponerOcupado(true);
    const c = new AbortController(); peticion.current = c;
    try {
      const r = await fetch('/api/admin/auditoria/exportaciones', { method: 'POST', signal: c.signal,
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
        body: JSON.stringify({ desde, hasta, tipo_evento: tipo, usuario_id: usuario, expediente_id: expediente.trim(), formato, idioma }) });
      if (!r.ok) { const b = await r.json() as { error?: { clave?: string } }; throw new Error(b.error?.clave ?? 'auditoria.no_disponible'); }
      if (r.headers.get('Content-Type') !== 'application/zip') throw new Error('auditoria.no_disponible');
      const blob = await r.blob(); if (c.signal.aborted) return;
      const cantidad = Number(r.headers.get('X-Auditoria-Filas'));
      const url = URL.createObjectURL(blob); const a = document.createElement('a');
      a.href = url; a.download = 'auditoria.zip'; document.body.append(a); a.click(); a.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
      ponerFilas(Number.isSafeInteger(cantidad) && cantidad >= 0 ? cantidad : 0);
    } catch (e) { if (!c.signal.aborted) ponerError(e instanceof Error ? e.message : 'auditoria.no_disponible'); }
    finally { envio.current = false; if (!c.signal.aborted) ponerOcupado(false); }
  }

  return <section className="auditoria" aria-labelledby="auditoria-titulo" onKeyDown={e=>{if(e.key==='Escape'&&ayuda){e.preventDefault();ponerAyuda(false);document.querySelector<HTMLButtonElement>('[aria-controls="auditoria-ayuda"]')?.focus();}}}>
    <div className="auditoria-cabecera"><h1 id="auditoria-titulo">{t('auditoria.titulo')}</h1>
      <button type="button" className="boton-secundario" aria-label={etiquetaAyuda} aria-expanded={ayuda} aria-controls="auditoria-ayuda" onClick={() => ponerAyuda(v => !v)}>{t('auditoria.ayuda_icono')}</button></div>
    <p>{t('auditoria.presentacion')}</p>
    {ayuda && <div id="auditoria-ayuda" className="auditoria-tarjeta" onKeyDown={e => { if (e.key === 'Escape') { ponerAyuda(false); document.querySelector<HTMLButtonElement>('[aria-controls="auditoria-ayuda"]')?.focus(); } }}><p>{t('auditoria.ayuda')}</p></div>}
    <p className="auditoria-privacidad" id="auditoria-privacidad">{t('auditoria.privacidad')}</p>
    {cargando && <p role="status">{t('auditoria.cargando')}</p>}
    {error && <p id="auditoria-error" ref={errorRef} tabIndex={-1} className="auditoria-error" role="alert">{t(error) || t('auditoria.no_disponible')}</p>}
    {!cargando && !errorCarga && usuarios.length === 0 && <p role="status">{t('auditoria.sin_usuarios')}</p>}
    {errorCarga && !ocupado && <button type="button" className="boton-secundario" onClick={() => ponerIntento(v => v + 1)}>{t('auditoria.recargar_personas')}</button>}
    <form className="auditoria-tarjeta" onSubmit={e => void descargar(e)} noValidate aria-busy={ocupado} aria-describedby="auditoria-privacidad">
      <h2>{t('auditoria.filtros')}</h2>
      <fieldset disabled={ocupado}><legend>{t('auditoria.periodo')}</legend>
        <p id="auditoria-periodo-ayuda">{t('auditoria.periodo_limite')}</p>
        <div className="auditoria-campos">
          <label htmlFor="auditoria-desde">{t('auditoria.desde')}<input id="auditoria-desde" type="date" required value={desde} onChange={e => ponerDesde(e.target.value)} aria-describedby={error === 'auditoria.fechas_invalidas' ? 'auditoria-error auditoria-periodo-ayuda' : 'auditoria-periodo-ayuda'} aria-invalid={error === 'auditoria.fechas_invalidas'} /></label>
          <label htmlFor="auditoria-hasta">{t('auditoria.hasta')}<input id="auditoria-hasta" type="date" required min={desde} value={hasta} onChange={e => ponerHasta(e.target.value)} aria-describedby={error === 'auditoria.fechas_invalidas' ? 'auditoria-error auditoria-periodo-ayuda' : 'auditoria-periodo-ayuda'} aria-invalid={error === 'auditoria.fechas_invalidas'} /></label>
        </div></fieldset>
      <fieldset disabled={ocupado}><legend>{t('auditoria.filtros_opcionales')}</legend><div className="auditoria-campos">
        <label htmlFor="auditoria-tipo">{t('auditoria.tipo')}<select id="auditoria-tipo" value={tipo} onChange={e => ponerTipo(e.target.value)}><option value="">{t('auditoria.todos_tipos')}</option>{tipos.map(v => <option value={v} key={v}>{t(`auditoria.tipo.${v}`)}</option>)}</select></label>
        <label htmlFor="auditoria-usuario">{t('auditoria.usuario')}<select id="auditoria-usuario" value={usuario} onChange={e => ponerUsuario(e.target.value)} disabled={cargando}><option value="">{t('auditoria.todas_personas')}</option><option value="sistema">{t('auditoria.sistema')}</option>{usuarios.map(u => <option value={u.id} key={u.id}>{t('auditoria.persona_etiqueta').replace('{iniciales}',u.iniciales).replace('{cuenta}',u.cuenta)}</option>)}</select></label>
        <label htmlFor="auditoria-expediente">{t('auditoria.expediente')}<input id="auditoria-expediente" value={expediente} maxLength={36} onChange={e => ponerExpediente(e.target.value)} aria-describedby="auditoria-expediente-ayuda" /><span id="auditoria-expediente-ayuda">{t('auditoria.expediente_ayuda')}</span></label>
        <label htmlFor="auditoria-formato">{t('auditoria.formato')}<select id="auditoria-formato" value={formato} onChange={e => ponerFormato(e.target.value)}><option value="csv">{t('auditoria.formato_csv')}</option><option value="json">{t('auditoria.formato_json')}</option></select></label>
      </div></fieldset>
      <p>{t('auditoria.consecuencia')}</p><Descargar ocupado={ocupado} t={t} />
    </form>
    <p role="status" aria-live="polite">{ocupado ? t('auditoria.preparando') : filas === 0 ? t('auditoria.vacio') : filas !== null ? t('auditoria.descargado').replace('{cantidad}', new Intl.NumberFormat(idioma).format(filas)) : ''}</p>
  </section>;
}
