import { useEffect, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { claveFallo, leer, type Opciones, type Propuesta } from './api';
import { Boton } from './componentes';
import './salidas.css';
export function CierresPropuestos({ t, idioma, navegarRuta }: PropsPantalla) {
  const [filas, ponerFilas] = useState<Propuesta[]>([]);
  const [opciones, ponerOpciones] = useState<Opciones>();
  const [pagina, ponerPagina] = useState(0);
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState('');
  const [intento, ponerIntento] = useState(0);
  useEffect(() => {
    const c = new AbortController(); ponerCargando(true); ponerError('');
    void Promise.all([leer<Propuesta[] | null>(`/api/coordinacion/cierres?limite=20&desplazamiento=${pagina * 20}`, c.signal), leer<Opciones>(`/api/coordinacion/decision/opciones?idioma=${encodeURIComponent(idioma)}`, c.signal)])
      .then(([f, o]) => { if (!c.signal.aborted) { ponerFilas(f ?? []); ponerOpciones(o); } })
      .catch((e: unknown) => { if (!c.signal.aborted) ponerError(claveFallo(e)); })
      .finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    return () => c.abort();
  }, [pagina, intento, idioma]);
  const fecha = (s: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(s));
  return <section className="salidas"><nav><a href="/coordinacion">{t('salidas.bandeja')}</a></nav><header><h1>{t('salidas.cierres')}</h1><p>{t('salidas.lista_consecuencia')}</p></header>
    <p role="status">{cargando ? t('salidas.cargando') : ''}</p>
    {error && <div role="alert"><p>{t(error)}</p><Boton accion={() => ponerIntento(n => n + 1)}>{t('salidas.reintentar')}</Boton></div>}
    {!cargando && !error && <section className="salidas-tarjeta"><h2>{t('salidas.pendientes_revision')}</h2>{!filas.length && <p>{t('salidas.vacio')}</p>}
      <ul className="salidas-lista">{filas.map(f => <li key={f.expediente_id}><h3><a href={`/coordinacion/expedientes/${f.expediente_id}/salidas`} onClick={e => { e.preventDefault(); navegarRuta(`/coordinacion/expedientes/${f.expediente_id}/salidas`); }}>{f.numero || t('salidas.sin_numero')}</a></h3><p>{opciones?.textos[`motivo_finalizacion.${f.motivo}`] || t('salidas.etiqueta_pendiente')}</p><p>{t('salidas.propuesto_el')} {fecha(f.propuesta_en)}</p></li>)}</ul>
      <div className="salidas-acciones"><Boton disabled={pagina === 0} accion={() => ponerPagina(n => n - 1)}>{t('salidas.anterior')}</Boton><Boton disabled={filas.length < 20} accion={() => ponerPagina(n => n + 1)}>{t('salidas.siguiente')}</Boton></div>
    </section>}
  </section>;
}
