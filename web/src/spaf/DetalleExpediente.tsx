import { useEffect, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { Fallo, leer, type Fila, type Tabla, type Opciones } from './api';
import './spaf.css';

type Expediente = { id: string; numero: string | null; estado: string; aceptacion?: { fecha_acta: string } };
const estados = ['aceptada', 'en_intervencion', 'cerrada', 'propuesta_cierre', 'en_seguimiento'];

// Consulta acotada al expediente que compone una cifra; la API confirma su auditoría.
export function DetalleExpediente({ t, idioma }: PropsPantalla) {
  const id = location.pathname.split('/').at(-1) ?? '';
  const q = new URLSearchParams(location.search);
  const volver = `/equipo/tabla-spaf?${new URLSearchParams({ mes: q.get('mes') ?? '', equipo: q.get('equipo') ?? '' })}`;
  const [dato, ponerDato] = useState<Expediente>();
  const [movimiento, ponerMovimiento] = useState<Fila>();
  const [opciones, ponerOpciones] = useState<Opciones>();
  const [error, ponerError] = useState('');
  const [cargando, ponerCargando] = useState(true);
  const [revision, recargar] = useState(0);
  useEffect(() => { document.title = t('spaf.detalle_titulo'); }, [t]);
  useEffect(() => {
    const c = new AbortController(); ponerDato(undefined); ponerError(''); ponerCargando(true);
    const mes = q.get('mes') ?? '';
    const filtro = new URLSearchParams({ anio: mes.slice(0, 4), mes: String(Number(mes.slice(5))), etf: q.get('equipo') ?? '' });
    void Promise.all([leer<Expediente>(`/api/expedientes/${encodeURIComponent(id)}`, c.signal), leer<Tabla>(`/api/estadistica/spaf?${filtro}`, c.signal), leer<Opciones>('/api/estadistica/spaf/opciones', c.signal)]).then(([d, tabla, o]) => {
      if (c.signal.aborted) return;
      if (d.id !== id || typeof d.estado !== 'string' || (d.numero !== null && typeof d.numero !== 'string')) throw new Fallo('spaf.error');
      const fila = tabla.filas?.find(f => f.expediente_id === id);
      if (!fila) throw new Fallo('spaf.movimiento_no_disponible');
      ponerDato(d); ponerMovimiento(fila); ponerOpciones(o);
    }).catch((e: unknown) => { if (!c.signal.aborted) ponerError(e instanceof Fallo ? e.clave : 'spaf.error'); }).finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    return () => c.abort();
  }, [id, revision]);
  const fecha = dato?.aceptacion?.fecha_acta;
  const presentarFecha = (valor: string) => Number.isNaN(Date.parse(valor)) ? t('spaf.sin_dato') : new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: 'UTC' }).format(new Date(valor));
  const motivo = (codigo: string) => {
    const clave = codigo.startsWith('motivo_spaf.') ? codigo : `motivo_spaf.${codigo}`;
    return opciones?.textos[idioma]?.[clave] || opciones?.textos.es?.[clave] || t('spaf.sin_dato');
  };
  return <section className="spaf" aria-labelledby="spaf-detalle">
    <a className="spaf-volver" href={volver}>{t('spaf.volver_tabla')}</a>
    <h1 id="spaf-detalle">{t('spaf.detalle_titulo')}</h1>
    {cargando && <p role="status">{t('spaf.cargando_expediente')}</p>}
    {error && <div role="alert" className="spaf-aviso"><p>{t(error)}</p><button type="button" className="boton-secundario" onClick={() => recargar(n => n + 1)}>{t('spaf.reintentar')}</button></div>}
    {dato && <section className="spaf-tarjeta" aria-labelledby="spaf-datos-expediente"><h2 id="spaf-datos-expediente">{dato.numero || t('spaf.sin_numero')}</h2><dl>
      <dt>{t('spaf.estado_actual')}</dt><dd>{t(estados.includes(dato.estado) ? `spaf.estado_${dato.estado}` : 'spaf.estado_otro')}</dd>
      <dt>{t('spaf.acta_fecha')}</dt><dd>{fecha ? presentarFecha(fecha) : t('spaf.sin_dato')}</dd>
    </dl><h3>{t('spaf.movimientos_mes')}</h3>{movimiento?.alta && <p>{t('spaf.alta')}{t('spaf.separador')}{movimiento.aceptacion.Conocido ? presentarFecha(movimiento.aceptacion.Valor) : t('spaf.sin_dato')}</p>}{movimiento?.cierres?.map((c, i) => <p key={i}>{t('spaf.baja')}{t('spaf.separador')}{presentarFecha(c.fecha)}{t('spaf.separador')}{c.motivo.Conocido ? motivo(c.motivo.Valor) : t('spaf.sin_dato')}</p>)}</section>}
  </section>;
}
