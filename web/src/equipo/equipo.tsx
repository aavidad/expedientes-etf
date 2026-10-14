import { useEffect, useRef, useState, type KeyboardEvent } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useLectura, useEtiquetas, fecha, type Resumen, type Ficha, type Suceso } from './api';
import { useCambios } from '../cambios';
import { FamiliaEquipo } from './familia';
import { SalidasEquipo } from './salidas';
import { CierreEquipo } from './cierre';
import { pestanas as pestanasFI2, type PestanaFicha } from './pestanas';
import './equipo.css';
import { AvisosFichaEquipo } from './avisos-ficha';
import { pestanas as pestanasFI1 } from '../fi1/pestanas';
import { pestanas as pestanasTR1 } from '../traslado/pestanas';

type Textos = PropsPantalla['t'];

const pestañaAviso = (clave: string) => clave === 'equipo.aviso.spm_pendiente' ? 'salidas' : clave === 'equipo.aviso.cierre_pendiente' ? 'cierre' : undefined;

export function EstadoLectura({ cargando, error, recargar, t }: { cargando: boolean; error: string; recargar: () => void; t: Textos }) {
  return <>{cargando && <p role="status">{t('equipo.cargando')}</p>}{error && <div role="alert"><p>{t(error) || t('equipo.error')}</p><button type="button" onClick={recargar}>{t('comun.reintentar')}</button></div>}</>;
}

export function BandejaEquipo({ t, idioma, navegarRuta }: PropsPantalla) {
  const catalogo = useEtiquetas(idioma);
  const etiqueta = (clave: string) => catalogo.datos?.[clave] || t('equipo.sin_dato');
  const [busqueda, ponerBusqueda] = useState('');
  const [q, ponerQ] = useState('');
  const [estado, ponerEstado] = useState('');
  const [subprograma, ponerSubprograma] = useState('');
  const [fase, ponerFase] = useState('');
  const [desplazamiento, ponerDesplazamiento] = useState(0);
  const parametros = new URLSearchParams({ q, estado, subprograma, fase, desplazamiento: String(desplazamiento) });
  const lectura = useLectura<{ filas: Resumen[]; hay_mas: boolean; equipo_clave: string }>(`/api/equipo/expedientes?${parametros}`);
  useEffect(() => { document.title = t('equipo.titulo'); }, [t]);
  const limpiar = () => { ponerBusqueda(''); ponerQ(''); ponerEstado(''); ponerSubprograma(''); ponerFase(''); ponerDesplazamiento(0); };
  const etiquetaPaginas = t('equipo.paginas');
  const abrir = (id: string) => navegarRuta(`/equipo/expediente?id=${encodeURIComponent(id)}`);
  return <section className="equipo" aria-labelledby="equipo-titulo">
    <h1 id="equipo-titulo">{t('equipo.titulo')}</h1>{lectura.datos && <p>{lectura.datos.equipo_clave ? etiqueta(lectura.datos.equipo_clave) : t('equipo.ambito_provincia')}</p>}
    <form className={['equipo-filtros', 'equipo-tarjeta'].join(' ')} onSubmit={(e) => { e.preventDefault(); ponerQ(busqueda.trim()); ponerDesplazamiento(0); }}>
      <label className="equipo-buscador" htmlFor="equipo-busqueda">{t('equipo.buscar')}<input id="equipo-busqueda" type="search" maxLength={100} value={busqueda} onChange={(e) => ponerBusqueda(e.target.value)} /></label>
      <button type="submit" className="principal">{t('equipo.buscar_accion')}</button>
      <label htmlFor="equipo-estado">{t('equipo.estado')}<select id="equipo-estado" value={estado} onChange={(e) => { ponerEstado(e.target.value); ponerDesplazamiento(0); }}>
        <option value="">{t('equipo.todos')}</option>{['aceptada', 'en_intervencion', 'cerrada'].map((v) => <option key={v} value={v}>{t(`equipo.estado.${v}`)}</option>)}
      </select></label>
      <label htmlFor="equipo-subprograma">{t('equipo.subprograma')}<select id="equipo-subprograma" value={subprograma} onChange={(e) => { ponerSubprograma(e.target.value); ponerDesplazamiento(0); }}>
        <option value="">{t('equipo.todos')}</option>{['RIESGO_SOCIAL', 'REUNIFICACION'].map((v) => <option key={v} value={v}>{etiqueta(`subprograma.${v}`)}</option>)}
      </select></label>
      <label htmlFor="equipo-fase">{t('equipo.fase')}<select id="equipo-fase" value={fase} onChange={(e) => { ponerFase(e.target.value); ponerDesplazamiento(0); }}>
        <option value="">{t('equipo.todos')}</option>{['recepcion', 'evaluacion', 'intervencion', 'cierre_devolucion', 'seguimiento'].map((v) => <option key={v} value={v}>{t(`equipo.fase.${v}`)}</option>)}
      </select></label>
      <button type="button" onClick={limpiar}>{t('equipo.limpiar')}</button>
    </form>
    <EstadoLectura {...lectura} t={t} /><EstadoLectura {...catalogo} t={t} />
    {lectura.datos && <>
      <p role="status">{t('equipo.resultados')} {new Intl.NumberFormat(idioma).format(lectura.datos.filas.length)}</p>
      {lectura.datos.filas.length === 0 ? <div className="equipo-tarjeta"><p>{q || estado || subprograma || fase ? t('equipo.sin_resultados') : t('equipo.vacio')}</p>{(q || estado || subprograma || fase) && <button type="button" onClick={limpiar}>{t('equipo.limpiar')}</button>}</div>
        : <div className={['equipo-tarjeta', 'equipo-listado'].join(' ')}><table>
          <caption>{t('equipo.tabla')}</caption>
          <thead><tr>{['numero', 'familia', 'subprograma', 'menores', 'estado', 'fechas', 'avisos'].map((c) => <th scope="col" key={c}>{t(`equipo.${c}`)}</th>)}</tr></thead>
          <tbody>{lectura.datos.filas.map((r) => <tr key={r.id}>
            <th scope="row"><a href={`/equipo/expediente?id=${encodeURIComponent(r.id)}`} onClick={(e) => { if (!e.ctrlKey && !e.metaKey && !e.shiftKey && e.button === 0) { e.preventDefault(); abrir(r.id); } }}>{r.numero}</a></th>
            <td data-etiqueta={t('equipo.familia')}>{t('equipo.familia')} {r.familia || t('equipo.sin_dato')}</td>
            <td data-etiqueta={t('equipo.subprograma')}><span className="equipo-pildora">{etiqueta(`subprograma.${r.subprograma}`)}</span></td>
            <td data-etiqueta={t('equipo.menores')}>{new Intl.NumberFormat(idioma).format(r.menores)}</td>
            <td data-etiqueta={t('equipo.estado')}><span>{t(`equipo.estado.${r.estado}`) || t('equipo.sin_dato')}</span><small>{t(`equipo.fase.${r.fase}`) || t('equipo.sin_dato')}</small></td>
            <td data-etiqueta={t('equipo.fechas')}><span>{t('equipo.aceptacion')} {fecha(r.fecha_aceptacion, idioma)}</span><small>{t('equipo.recepcion')} {fecha(r.fecha_recepcion, idioma)}</small></td>
            <td data-etiqueta={t('equipo.avisos')}>{r.avisos.length ? r.avisos.map((a) => pestañaAviso(a) ? <a className="equipo-aviso" key={a} href={`/equipo/expediente?id=${r.id}&pestana=${pestañaAviso(a)!}`} onClick={(e) => { e.preventDefault(); navegarRuta(`/equipo/expediente?id=${r.id}&pestana=${pestañaAviso(a)!}`); }}>{t(a)}</a> : <p className="equipo-aviso" key={a}>{t(a)}</p>) : t('equipo.sin_avisos')}</td>
          </tr>)}</tbody>
        </table></div>}
      <nav className="equipo-paginacion" aria-label={etiquetaPaginas}>
        <button type="button" disabled={desplazamiento === 0} onClick={() => ponerDesplazamiento((d) => Math.max(0, d - 50))}>{t('equipo.anterior')}</button>
        <span>{t('equipo.pagina')} {new Intl.NumberFormat(idioma).format(desplazamiento / 50 + 1)}</span>
        <button type="button" disabled={!lectura.datos.hay_mas} onClick={() => ponerDesplazamiento((d) => d + 50)}>{t('equipo.siguiente')}</button>
      </nav>
    </>}
  </section>;
}

function Sucesos({ sucesos, t, idioma, clase }: { sucesos: Suceso[]; t: Textos; idioma: string; clase?: string }) {
  return <ol className={`equipo-sucesos ${clase ?? ''}`}>{sucesos.map((e) => <li key={`${e.tipo}-${e.id}`}>
    <span className="equipo-pildora">{t(`equipo.tipo.${e.tipo}`)}</span>
    <strong>{t(`equipo.suceso.${e.codigo.startsWith('equipo.bandeja_consultada.filas_') ? 'equipo.bandeja_consultada' : e.codigo}`) || t(`equipo.documento.${e.codigo.replaceAll('/', '.')}`) || t('equipo.actividad_registrada')}</strong>
    {e.codigo.startsWith('equipo.bandeja_consultada.filas_') && <span>{new Intl.NumberFormat(idioma).format(Number(e.codigo.split('_').at(-1)))}</span>}<span>{fecha(e.fecha, idioma, e.tipo !== 'hito')}</span>
    <small>{t('equipo.autor')} {e.autor ? t('equipo.personal_programa') : t('equipo.sistema')}</small>
    {e.version_plantilla && <small>{t('equipo.version_plantilla')} {e.version_plantilla}</small>}
    {e.resultado && <small>{t(`equipo.resultado.${e.resultado}`)}</small>}
  </li>)}</ol>;
}

const registrosFicha: readonly PestanaFicha[] = [
  { codigo: 'familia', clave: 'equipo.pestana.familia', componente: (p) => <FamiliaEquipo {...p} /> },
  ...pestanasFI1.map((p) => ({ ...p, clave: `fi1.${p.codigo}` })),
  ...pestanasFI2,
  ...pestanasTR1.map((p) => ({ ...p, clave: 'traslado.seguimiento' })),
  { codigo: 'documentos', clave: 'equipo.pestana.documentos', componente: ({ ficha, t, idioma }) => <><h2>{t('equipo.emisiones')}</h2>{ficha.emisiones.length ? <Sucesos sucesos={ficha.emisiones} t={t} idioma={idioma} /> : <p>{t('equipo.sin_emisiones')}</p>}</> },
  { codigo: 'cierre', clave: 'equipo.pestana.cierre', componente: CierreEquipo },
  { codigo: 'salidas', clave: 'equipo.pestana.salidas', componente: SalidasEquipo },
  { codigo: 'historial', clave: 'equipo.pestana.historial', componente: ({ ficha, t, idioma }) => <><h2>{t('equipo.historial')}</h2>{ficha.historial.length ? <Sucesos sucesos={ficha.historial} t={t} idioma={idioma} /> : <p>{t('equipo.sin_historial')}</p>}</> },
];

export function FichaEquipo(props: PropsPantalla) {
  const { t, idioma, navegarRuta } = props;
  const { solicitarSalida } = useCambios();
  const catalogo = useEtiquetas(idioma);
  const etiqueta = (clave: string) => catalogo.datos?.[clave] || t('equipo.sin_dato');
  const id = new URLSearchParams(location.search).get('id') ?? '';
  const lectura = useLectura<Ficha>(`/api/equipo/expedientes/${encodeURIComponent(id)}`, true);
  const [activa, ponerActiva] = useState(new URLSearchParams(location.search).get('pestana') ?? (location.pathname === '/equipo/seguimiento' ? 'seguimiento' : location.pathname === '/equipo/menores' ? 'menores' : location.pathname === '/equipo/intervencion' ? 'intervencion' : 'documentos'));
  const ultimaFicha = useRef<Ficha | undefined>(undefined);
  if (lectura.error) ultimaFicha.current = undefined;
  else if (lectura.datos?.id === id) ultimaFicha.current = lectura.datos;
  const f = lectura.error ? undefined : lectura.datos?.id === id ? lectura.datos : ultimaFicha.current?.id === id ? ultimaFicha.current : undefined;
  useEffect(() => { document.title = f ? `${t('equipo.expediente')} ${f.numero}` : t('equipo.expediente'); }, [f, t]);
  const disponibles = registrosFicha.filter((p) => f?.capacidades.includes(p.codigo));
  const pestañas = disponibles.map((p) => p.codigo);
  const seleccionada = pestañas.includes(activa) ? activa : pestañas[0];
  useEffect(() => {
    if (seleccionada !== 'seguimiento') return;
    const tab = document.getElementById('equipo-tab-seguimiento'); const fila = tab?.parentElement;
    if (!tab || !fila) return;
    let cuadro = 0;
    const mostrar = () => { cancelAnimationFrame(cuadro); cuadro = requestAnimationFrame(() => tab.scrollIntoView({ block: 'nearest', inline: 'end' })); };
    mostrar(); const observar = new ResizeObserver(mostrar); observar.observe(fila); return () => { cancelAnimationFrame(cuadro); observar.disconnect(); };
  }, [seleccionada, !!f]);
  const etiquetaRuta = t('equipo.ruta');
  const etiquetaAvisos = t('equipo.avisos');
  const etiquetaSecciones = t('equipo.secciones');
  const mover = (e: KeyboardEvent<HTMLButtonElement>, indice: number) => {
    let destino: number | undefined;
    if (e.key === 'ArrowRight') destino = (indice + 1) % pestañas.length;
    if (e.key === 'ArrowLeft') destino = (indice + pestañas.length - 1) % pestañas.length;
    if (e.key === 'Home') destino = 0;
    if (e.key === 'End') destino = pestañas.length - 1;
    if (destino !== undefined) { e.preventDefault(); const siguiente = pestañas[destino]; solicitarSalida(() => { ponerActiva(siguiente); document.getElementById(`equipo-tab-${siguiente}`)?.focus(); }); }
  };
  return <section className="equipo" aria-labelledby="equipo-ficha-titulo">
    <nav aria-label={etiquetaRuta}><a href="/equipo" onClick={(e) => { e.preventDefault(); navegarRuta('/equipo'); }}>{t('equipo.titulo')}</a>{f && <span> {t('equipo.expediente')} {f.numero}</span>}</nav>
    <EstadoLectura {...lectura} t={t} /><EstadoLectura {...catalogo} t={t} />
    <h1 id="equipo-ficha-titulo">{f ? `${t('equipo.familia')} ${f.familia || t('equipo.sin_dato')}` : t('equipo.expediente')}</h1>
    {f && <>
      <div className="equipo-resumen"><span className="equipo-pildora">{f.numero}</span><span className="equipo-pildora">{etiqueta(`subprograma.${f.subprograma}`)}</span><span>{t(`equipo.estado.${f.estado}`)}</span>{seleccionada !== 'seguimiento' && <span>{t(`equipo.fase.${f.fase}`)}</span>}</div>
      {f.avisos.length > 0 && <aside className={['equipo-tarjeta', 'equipo-aviso'].join(' ')} aria-label={etiquetaAvisos}>{f.avisos.map((a) => pestañaAviso(a) ? <button key={a} type="button" onClick={() => { solicitarSalida(() => { const pestana = pestañaAviso(a)!; ponerActiva(pestana); document.getElementById(`equipo-tab-${pestana}`)?.focus(); }); }}>{t(a)}</button> : <p key={a}>{t(a)}</p>)}</aside>}
      <AvisosFichaEquipo key={`${id}-${f.version}`} id={id} t={t} abrir={(pestana, foco) => solicitarSalida(() => { ponerActiva(pestana); foco(); })} />
      {seleccionada !== 'seguimiento' && <><LineaTiempo ficha={f} t={t} idioma={idioma} /><details className="equipo-tarjeta"><summary>{t('equipo.ver_hitos')}</summary><Sucesos sucesos={f.hitos} t={t} idioma={idioma} /></details></>}
      <div className="equipo-pestanas" role="tablist" aria-label={etiquetaSecciones}>{pestañas.map((c, indice) => <button type="button" role="tab" id={`equipo-tab-${c}`} key={c} aria-selected={seleccionada === c} aria-controls={`equipo-panel-${c}`} tabIndex={seleccionada === c ? 0 : -1} onClick={() => solicitarSalida(() => ponerActiva(c))} onKeyDown={(e) => mover(e, indice)}>{t(disponibles.find((p) => p.codigo === c)!.clave)}{c === 'menores' && <span> {new Intl.NumberFormat(idioma).format(f.menores)}</span>}</button>)}</div>
      {disponibles.map(({ codigo, componente: Componente }) => <section key={codigo} className="equipo-tarjeta" role="tabpanel" id={`equipo-panel-${codigo}`} aria-labelledby={`equipo-tab-${codigo}`} tabIndex={0} hidden={seleccionada !== codigo}>
        {seleccionada === codigo && <Componente {...props} id={id} ficha={f} alCambiar={lectura.recargar} refrescando={lectura.cargando} />}
      </section>)}
    </>}
  </section>;
}

function LineaTiempo({ ficha, t, idioma }: { ficha: Ficha; t: Textos; idioma: string }) {
  const confirmada = (codigo: string) => ficha.hitos.find((h) => h.codigo === codigo)?.fecha;
  const pasos = [
    { codigo: 'derivacion', fecha: ficha.fecha_recepcion },
    { codigo: 'aceptacion', fecha: confirmada('aceptacion') },
    { codigo: 'consentimiento', fecha: confirmada('consentimiento') },
    { codigo: 'proyecto', fecha: confirmada('anexo7') ?? confirmada('anexo8') },
    { codigo: 'intervencion', fecha: undefined },
    { codigo: 'cierre', fecha: confirmada('cierre') },
    ...(ficha.subprograma === 'REUNIFICACION' ? [{ codigo: 'seguimiento', fecha: undefined }] : []),
  ];
  const pendiente = pasos.findIndex((p) => !p.fecha);
  const actual = ficha.fase === 'seguimiento' ? 6 : ficha.fase === 'cierre_devolucion' ? 5 : ficha.estado === 'en_intervencion' ? 4 : pendiente;
  const etiqueta = t('equipo.fases');
  return <section className="equipo-tarjeta" aria-labelledby="equipo-hitos"><h2 id="equipo-hitos">{etiqueta}</h2><p>{t('equipo.fase_actual')} {t(`equipo.fase.${ficha.fase}`)}</p>
    <ol className="equipo-fases" aria-label={etiqueta}>{pasos.map((p, i) => <li key={p.codigo} aria-current={i === actual ? 'step' : undefined}>
      <strong>{t(`equipo.paso.${p.codigo}`)}</strong><span>{t(p.fecha ? 'equipo.paso.confirmado' : i === actual ? 'equipo.paso.actual' : 'equipo.paso.pendiente')}</span>
      {p.fecha && <time dateTime={p.fecha}>{fecha(p.fecha, idioma)}</time>}
    </li>)}</ol>
  </section>;
}
