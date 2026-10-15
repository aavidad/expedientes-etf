import { useEffect, useRef, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { FalloMemoria, pedir, type Equipo } from '../memoria/api';
import { etiquetaCifra } from '../memoria/presentacion';
import '../memoria/memoria.css';

type Fila = { expediente_id: string; numero: string; equipo_id: string; cantidad: number; ficha: boolean };
type Detalle = { anio: number; equipo_id: string; apartado: string; grupo: Record<string, string>; unidad: string; medida: string; cifra: number; poblacion_base: string; filas: Fila[]; aviso: string; definitivas: boolean; fecha: string };

export function DesgloseMemoria({ t, idioma, navegarRuta }: PropsPantalla) {
  const parametros = new URLSearchParams(location.search);
  const anio = parametros.get('anio') ?? '';
  const equipo = parametros.get('etf') ?? '';
  const apartado = parametros.get('apartado') ?? '';
  const [detalle, ponerDetalle] = useState<Detalle>();
  const [equipos, ponerEquipos] = useState<Equipo[]>([]);
  const [error, ponerError] = useState('');
  const [intento, ponerIntento] = useState(0);
  const [ayuda, ponerAyuda] = useState(false);
  const titulo = useRef<HTMLHeadingElement>(null);
  const retorno = `/memoria?${new URLSearchParams({ anio, ...(equipo ? { etf: equipo } : {}), apartado, celda: parametros.get('celda') ?? '-1', medida: parametros.get('medida') ?? '' })}`;
  const n = (valor: number) => new Intl.NumberFormat(idioma).format(valor);
  useEffect(() => { document.title = t('memoria.desglose.titulo'); }, [t]);
  useEffect(() => {
    const c = new AbortController();
    ponerDetalle(undefined); ponerError('');
    const consulta = new URLSearchParams({ celda: parametros.get('celda') ?? '', medida: parametros.get('medida') ?? '', ...(equipo ? { etf: equipo } : {}), ...(parametros.has('clave') ? { clave: parametros.get('clave')! } : {}), ...(parametros.has('revision') ? { revision: parametros.get('revision')! } : {}) });
    void Promise.all([
      pedir<Detalle>(`/api/memoria/${encodeURIComponent(anio)}/desglose/${encodeURIComponent(apartado)}?${consulta}`, { signal: c.signal }),
      pedir<Equipo[]>('/api/estadistica/memoria/equipos', { signal: c.signal }),
    ]).then(([datos, lista]) => {
      if (c.signal.aborted) return;
      if (parametros.has('esperado') && datos.cifra !== Number(parametros.get('esperado'))) { ponerError('memoria.desglose.cifra_cambiada'); return; }
      ponerDetalle(datos); ponerEquipos(lista); requestAnimationFrame(() => titulo.current?.focus());
    }).catch((e: unknown) => { if (!c.signal.aborted) ponerError(e instanceof FalloMemoria ? e.clave : 'estadistica.memoria_no_disponible'); });
    titulo.current?.focus();
    return () => c.abort();
  }, [anio, equipo, apartado, intento, location.search]);
  const enlazar = (ruta: string, evento: React.MouseEvent<HTMLAnchorElement>) => {
    if (evento.button === 0 && !evento.ctrlKey && !evento.metaKey && !evento.shiftKey && !evento.altKey) { evento.preventDefault(); navegarRuta(ruta); }
  };
  const nombreEquipo = (id: string) => {
    const codigo = equipos.find((e) => e.id === id)?.codigo;
    return codigo ? t('memoria.equipo_opcion').replace('{codigo}', codigo) : t('memoria.desglose.sin_equipo');
  };
  const grupo = detalle ? Object.entries(detalle.grupo).filter(([dimension, v]) => !!v && !(detalle.medida === 'sin_dato' && dimension.toLowerCase() === 'estado' && v === 'no')).map(([dimension, valor]) => etiquetaCifra(t, detalle.apartado, dimension.toLowerCase(), valor)).join(t('memoria.separador')) : '';
  return <section className="memoria" aria-labelledby="memoria-desglose-titulo">
    <a className="memoria-retorno" href={retorno} onClick={(e) => enlazar(retorno, e)}>{t('memoria.desglose.volver')}</a>
    <div className="memoria-cabecera"><h1 id="memoria-desglose-titulo" ref={titulo} tabIndex={-1}>{t('memoria.desglose.titulo')}</h1><button className="boton-secundario" aria-expanded={ayuda} onClick={() => ponerAyuda(!ayuda)}>{t('memoria.ayuda')}</button></div>
    {ayuda && <p className="memoria-tarjeta">{t('memoria.desglose.ayuda')}</p>}
    <p>{t(`memoria.apartado.${apartado}`)}</p>
    {!detalle && !error && <p role="status">{t('memoria.desglose.cargando')}</p>}
    {error && <div role="alert" className="memoria-aviso"><p>{t(error) || t('estadistica.memoria_no_disponible')}</p><button className="boton-secundario" onClick={() => ponerIntento(intento + 1)}>{t('memoria.reintentar')}</button></div>}
    {detalle && <>
      <p>{t('memoria.ambito').replace('{anio}', new Intl.NumberFormat(idioma, { useGrouping: false }).format(detalle.anio)).replace('{equipo}', detalle.equipo_id ? nombreEquipo(detalle.equipo_id) : t('memoria.provincial'))}</p>
      <p className="pildora">{t(detalle.definitivas ? 'memoria.desglose.definitiva' : 'memoria.desglose.provisional')}</p>
      <p>{t('memoria.calculada_en').replace('{fecha}', new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(detalle.fecha)))}</p>
      <section className="memoria-tarjeta" aria-labelledby="memoria-desglose-cifra">
        <h2 id="memoria-desglose-cifra">{t('memoria.desglose.cifra').replace('{cantidad}', n(detalle.cifra))}</h2>
        <p>{[detalle.medida === 'denominador' ? t(`memoria.denominador.${detalle.poblacion_base?.toLowerCase()}`) || t('memoria.denominador') : grupo || t('memoria.desglose.total'), t(`memoria.unidad.${detalle.unidad}`) || t('memoria.relacion_desconocida'), t(`memoria.desglose.medida.${detalle.medida}`)].filter(Boolean).join(t('memoria.separador'))}</p>
        {detalle.aviso ? <p role="status" className="memoria-aviso">{t(detalle.aviso)}</p> : <>
          <p role="status">{t('memoria.desglose.suma').replace('{cantidad}', n(detalle.filas.reduce((suma, f) => suma + f.cantidad, 0)))}</p>
          {detalle.filas.length === 0 ? <p>{t('memoria.desglose.vacio')}</p> : <ul className="memoria-desglose-lista">
            {detalle.filas.map((f) => {
              const ruta = `/equipo/expediente?id=${encodeURIComponent(f.expediente_id)}`;
              const etiquetaFicha = t('memoria.desglose.abrir').replace('{numero}', f.numero || t('memoria.desglose.sin_numero'));
              return <li key={f.expediente_id} className="memoria-desglose-fila"><div>{f.ficha ? <a href={ruta} onClick={(e) => enlazar(ruta, e)} aria-label={etiquetaFicha}>{f.numero || t('memoria.desglose.sin_numero')}</a> : <strong>{f.numero || t('memoria.desglose.sin_numero')}</strong>}<p>{nombreEquipo(f.equipo_id)}</p></div><p>{t('memoria.desglose.aporte').replace('{cantidad}', n(f.cantidad)).replace('{unidad}', t(`memoria.unidad.${detalle.unidad}`) || t('memoria.unidad.familias'))}</p></li>;
            })}
          </ul>}
        </>}
      </section>
    </>}
  </section>;
}
