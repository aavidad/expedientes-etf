import { useEffect, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { Escuchar, manifiesto, type Manifiesto } from './audio';
import './voz.css';

type Seccion = { sha: string; id: string; titulo: string; html: string };
type Manual = { idioma: string; nombre: string; titulo: string; secciones: Seccion[] };

export function Ayuda({ idioma, t }: PropsPantalla) {
  const [manuales, ponerManuales] = useState<Manual[]>([]);
  const [audios, ponerAudios] = useState<Manifiesto>({});
  const [cargando, cargar] = useState(true);
  const [error, fallar] = useState(false);
  const [intento, reintentar] = useState(0);
  useEffect(() => {
    const controlador = new AbortController();
    cargar(true); fallar(false); ponerAudios({});
    void fetch('/api/ayuda/manuales', { signal: controlador.signal }).then(async r => {
      if (!r.ok) throw new Error('voz.manuales_no_disponibles');
      const datos = await r.json() as Manual[];
      if (!controlador.signal.aborted) ponerManuales(datos);
    }).catch(() => { if (!controlador.signal.aborted) fallar(true); }).finally(() => { if (!controlador.signal.aborted) cargar(false); });
    if (idioma === 'es') void manifiesto('manuales', controlador.signal).then(datos => { if (!controlador.signal.aborted) ponerAudios(datos); });
    return () => controlador.abort();
  }, [idioma, intento]);
  const indice = t('voz.indice');
  const tituloSeccion = (s: Seccion) => s.id.endsWith('-s0') ? t('voz.presentacion') : s.titulo;
  return <div className="voz-ayuda">
    <h1>{t('voz.titulo')}</h1>
    {cargando ? <p role="status">{t('voz.cargando')}</p> : error ? <div role="alert"><p>{t('voz.error')}</p><button type="button" onClick={() => reintentar(n => n + 1)}>{t('voz.reintentar')}</button></div> : !manuales.length ? <p>{t('voz.vacio')}</p> : manuales.map(manual => <article lang={manual.idioma} className="voz-manual" id={`manual-${manual.nombre}`} key={manual.nombre} onClick={e => {
      const enlace = (e.target as HTMLElement).closest('a');
      if (!enlace?.hash) return;
      const destino = document.getElementById(decodeURIComponent(enlace.hash.slice(1)));
      const detalle = destino?.closest('details');
      if (detalle) { detalle.open = true; detalle.querySelector('summary')?.focus(); }
    }}>
      <h2>{manual.titulo}</h2>
      <details className="voz-seccion"><summary>{t('voz.indice')}</summary><nav aria-label={indice}><ol>{manual.secciones.map(seccion => <li key={seccion.id}><a href={`#${seccion.id}`} onClick={() => {
        const detalle = document.getElementById(seccion.id) as HTMLDetailsElement | null;
        if (detalle) { detalle.open = true; detalle.querySelector('summary')?.focus(); }
      }}>{tituloSeccion(seccion)}</a></li>)}</ol></nav></details>
      {manual.secciones.map((seccion, i) => <details className="voz-seccion" id={seccion.id} key={seccion.id} open={i === 0} onToggle={e => {
        if (!e.currentTarget.open) e.currentTarget.querySelector('audio')?.pause();
      }}>
        <summary><h3>{tituloSeccion(seccion)}</h3></summary>
        {idioma === 'es' && <Escuchar grupo="manuales" audio={audios[seccion.id]?.sha === seccion.sha ? audios[seccion.id] : undefined} t={t} titulo={tituloSeccion(seccion)} seccion />}
        <div className="voz-contenido" dangerouslySetInnerHTML={{ __html: seccion.html }} />
      </details>)}
    </article>)}
  </div>;
}
