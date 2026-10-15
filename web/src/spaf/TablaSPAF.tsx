import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useButton } from 'react-aria';
import type { Acceso } from '../armazon';
import type { PropsPantalla } from '../registro-pantallas';
import { emitir, Fallo, iniciales, leer, mesAnterior, mesValido, validarTabla, type Dato, type Fila, type Opciones, type Tabla } from './api';
import { olvidar, preparar, recuperar } from './recuperacion';
import './spaf.css';

function Boton({ children, accion, ocupado = false, principal = false }: { children: ReactNode; accion: () => void; ocupado?: boolean; principal?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion, isDisabled: ocupado }, ref);
  return <button {...buttonProps} ref={ref} className={principal ? undefined : 'boton-secundario'}>{children}</button>;
}
const columnas = ['numero', 'menores', 'numero_menores', 'guardadores', 'municipio_zts', 'caracteristicas', 'derivacion', 'servicio', 'fechas', 'fase', 'temporalizacion', 'finalidad_pronostico', 'instrumentos', 'cierres'] as const;
const avisosPermitidos = ['estadistica.spaf_ficha_sin_dato', 'estadistica.spaf_menores_sin_dato', 'estadistica.aceptacion_sin_dato', 'estadistica.spaf_motivo_sin_equivalencia', 'estadistica.spaf_derivacion_ambigua'];

export function TablaSPAF({ t, idioma, csrf }: PropsPantalla) {
  const [mes, ponerMes] = useState(() => new URLSearchParams(location.search).get('mes') || mesAnterior());
  const [equipo, ponerEquipo] = useState(() => new URLSearchParams(location.search).get('equipo') || '');
  const [acceso, ponerAcceso] = useState<Acceso>();
  const [opciones, ponerOpciones] = useState<Opciones>();
  const [tabla, ponerTabla] = useState<Tabla>();
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState('');
  const [estado, ponerEstado] = useState('');
  const [revision, recargar] = useState(0);
  const [emitiendo, ponerEmitiendo] = useState(false);
  const [conflicto, ponerConflicto] = useState(false);
  const descarga = useRef<AbortController | null>(null);
  const [recuperacion, ponerRecuperacion] = useState(false);
  const ayuda = useRef<HTMLDialogElement>(null);
  const ayudaBoton = useRef<HTMLButtonElement>(null);
  const { buttonProps: propsAyuda } = useButton({ onPress: () => ayuda.current?.showModal() }, ayudaBoton);
  const provincial = acceso?.perfiles?.includes('coordinacion');
  const formato = (k: string, valores: Record<string, string | number>) => Object.entries(valores).reduce((s, [n, v]) => s.replaceAll(`{${n}}`, String(v)), t(k));
  const numero = (n: number) => new Intl.NumberFormat(idioma).format(n);
  const fecha = (s: string) => {
    const d = new Date(s); return Number.isFinite(d.getTime()) ? new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: 'UTC' }).format(d) : t('spaf.sin_dato');
  };
  const mesTexto = mesValido(mes) ? new Intl.DateTimeFormat(idioma, { month: 'long', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${mes}-01T12:00:00Z`)) : '';
  const etiqueta = (lista: string, codigo: string) => {
    const k = codigo.startsWith(`${lista}.`) ? codigo : `${lista}.${codigo}`;
    return opciones?.textos[idioma]?.[k] || opciones?.textos.es?.[k] || t('spaf.sin_dato');
  };
  const nombreEquipo = etiqueta('equipo', opciones?.equipos.find(e => e.id === equipo)?.clave_texto ?? '');
  const nombreAyuda = t('spaf.abrir_ayuda');
  const nombreRegion = t('spaf.tabla_region');
  const desconocido = <span className="spaf-sin-dato">{t('spaf.sin_dato')}</span>;
  const presentar = <T,>(d: Dato<T>, ver: (v: T) => ReactNode): ReactNode => d.Conocido ? ver(d.Valor) : desconocido;
  const lista = (d: Dato<string[]>, tipo: string) => presentar(d, valores => valores?.length ? valores.map(v => etiqueta(tipo, v)).join('; ') : t('spaf.ninguno'));

  useEffect(() => { document.title = t('spaf.titulo'); }, [t]);
  useEffect(() => {
    const q = new URLSearchParams({ mes, ...(equipo ? { equipo } : {}) });
    history.replaceState(history.state, '', `${location.pathname}?${q}`);
  }, [mes, equipo]);
  useEffect(() => {
    const c = new AbortController();
    ponerCargando(true); ponerError(''); ponerOpciones(undefined); ponerAcceso(undefined);
    void Promise.all([leer<Acceso>('/api/sesion', c.signal), leer<Opciones>('/api/estadistica/spaf/opciones', c.signal)]).then(([sesion, o]) => {
      if (c.signal.aborted) return;
      if (sesion.estado !== 'activo' || !Array.isArray(o.equipos) || !o.textos) throw new Fallo('spaf.error');
      ponerAcceso(sesion); ponerOpciones(o);
      const anterior = recuperar(sesion.usuario?.id ?? '');
      if (anterior) {
        ponerRecuperacion(true); ponerEquipo(anterior.tabla.equipo_id);
        ponerMes(`${anterior.tabla.anio}-${String(anterior.tabla.mes).padStart(2, '0')}`);
      } else if (!sesion.perfiles?.includes('coordinacion')) ponerEquipo(sesion.equipo_id ?? '');
    }).catch((e: unknown) => { if (!c.signal.aborted) ponerError(e instanceof Fallo ? e.clave : 'spaf.error'); }).finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    return () => c.abort();
  }, [revision]);

  useEffect(() => {
    const c = new AbortController();
    ponerTabla(undefined); ponerEstado(''); ponerConflicto(false); descarga.current?.abort(); ponerEmitiendo(false);
    if (!opciones) return () => c.abort();
    ponerError('');
    if (!equipo) { ponerCargando(false); return () => c.abort(); }
    if (!mesValido(mes) || mes > mesAnterior()) {
      ponerError('spaf.mes_invalido'); ponerCargando(false); return () => c.abort();
    }
    ponerCargando(true);
    const q = new URLSearchParams({ anio: mes.slice(0, 4), mes: String(Number(mes.slice(5))), etf: equipo });
    void leer<Tabla>(`/api/estadistica/spaf?${q}`, c.signal).then(v => {
      if (c.signal.aborted) return;
      if (!validarTabla(v, equipo, mes)) throw new Fallo('spaf.error');
      ponerTabla(v);
    }).catch((e: unknown) => { if (!c.signal.aborted) ponerError(e instanceof Fallo ? e.clave : 'spaf.error'); }).finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    return () => { c.abort(); descarga.current?.abort(); };
  }, [mes, equipo, opciones]);

  async function descargar() {
    if (!acceso?.usuario?.id || emitiendo || conflicto) return;
    const anterior = recuperar(acceso.usuario.id);
    if (!tabla && !anterior) return;
    const c = new AbortController(); descarga.current = c;
    const solicitud = anterior ?? preparar(acceso.usuario.id, tabla!, idioma);
    ponerRecuperacion(true);
    ponerEmitiendo(true); ponerError(''); ponerEstado('spaf.generando');
    try {
      const resultado = await emitir(solicitud.tabla, solicitud.idioma, csrf, solicitud.clave, c.signal);
      if (c.signal.aborted) return;
      const url = URL.createObjectURL(resultado.contenido);
      const enlace = document.createElement('a'); enlace.href = url; enlace.download = `spaf-${solicitud.tabla.anio}-${String(solicitud.tabla.mes).padStart(2, '0')}.docx`; enlace.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      olvidar(acceso.usuario.id); ponerRecuperacion(false); ponerEstado(resultado.repetida ? 'spaf.recuperado' : resultado.cambiado ? 'spaf.descargado_cambios' : 'spaf.descargado');
    } catch (e: unknown) {
      if (c.signal.aborted) return;
      const fallo = e instanceof Fallo ? e : new Fallo('spaf.descarga_desconocida');
      if (fallo.estado > 0) { olvidar(acceso.usuario.id); ponerRecuperacion(false); }
      ponerEstado(''); ponerError(fallo.clave); ponerConflicto(fallo.estado === 409);
    } finally { if (!c.signal.aborted) ponerEmitiendo(false); }
  }

  function guardadores(f: Fila): ReactNode {
    const menores = f.menores.Conocido ? f.menores.Valor ?? [] : [];
    const centros = menores.filter(m => m.centro_proteccion.Conocido && m.centro_proteccion.Valor.trim());
    const todosEnCentro = menores.length > 0 && centros.length === menores.length;
    return <>{!todosEnCentro && presentar(f.guardadores, v => v?.length ? v.map(g => `${iniciales(g.nombre)} · ${etiqueta('papel_parental', g.parentesco)}`).join('; ') : t('spaf.ninguno'))}{centros.map((m, i) => <div key={i}>{m.centro_proteccion.Valor}</div>)}</>;
  }
  function celdas(f: Fila): ReactNode[] {
    return [f.numero || t('spaf.sin_numero'),
      presentar(f.menores, v => v?.length ? v.map(m => `${iniciales(`${m.nombre} ${m.apellidos}`)} · ${m.nacimiento.Conocido ? fecha(m.nacimiento.Valor) : t('spaf.sin_dato')}`).join('; ') : t('spaf.ninguno')),
      presentar(f.numero_menores, numero),
      guardadores(f),
      presentar(f.municipio, v => etiqueta('municipio', v)),
      lista(f.caracteristicas_familia, 'caracteristica_familia'),
      presentar(f.derivacion, v => `${formato('spaf.anexo', { n: v.anexo })} · ${fecha(v.fecha)}`),
      <>{presentar(f.servicio_derivante, v => etiqueta('servicio_derivante', v))}<br />{presentar(f.coordinador_caso, iniciales)}</>,
      <>{t('spaf.acta')}{t('spaf.separador')}{presentar(f.aceptacion, fecha)}{(f.fechas_anexos ?? []).map((a, i) => <div key={i}>{formato('spaf.anexo', { n: a.anexo })}{t('spaf.separador')}{fecha(a.fecha)}</div>)}</>,
      presentar(f.fase_intervencion, v => etiqueta('fase_intervencion', v)), presentar(f.temporalizacion_meses, numero),
      <>{presentar(f.finalidad, v => etiqueta('finalidad', v))}<br />{presentar(f.pronostico, v => etiqueta('pronostico', v))}</>, lista(f.instrumentos, 'instrumento'),
      f.cierres?.length ? f.cierres.map((c, i) => <div key={i}>{fecha(c.fecha)}{t('spaf.separador')}{presentar(c.motivo, v => etiqueta('motivo_spaf', v))}</div>) : t('spaf.sin_cierre'),
    ];
  }
  const filas = tabla?.filas ?? [];
  const enlaceExpediente = (f: Fila) => `/equipo/tabla-spaf/expedientes/${encodeURIComponent(f.expediente_id)}?${new URLSearchParams({ mes, equipo })}`;
  return <section className={['contenido-admin', 'spaf'].join(' ')} aria-labelledby="spaf-titulo">
    <header className="spaf-cabecera"><div><h1 id="spaf-titulo">{t('spaf.titulo')}</h1>{equipo && <p>{nombreEquipo}{mesTexto && <span>{t('spaf.separador')}{mesTexto}</span>}</p>}</div><button {...propsAyuda} ref={ayudaBoton} className="boton-secundario" aria-label={nombreAyuda}>{t('spaf.simbolo_ayuda')}</button></header>
    <section className="spaf-tarjeta" aria-labelledby="spaf-periodo"><h2 id="spaf-periodo">{t('spaf.periodo')}</h2><div className="spaf-filtros">
      {provincial && <div className="spaf-campo"><label htmlFor="spaf-equipo">{t('spaf.equipo')}</label><select id="spaf-equipo" disabled={emitiendo || recuperacion} value={equipo} onChange={e => ponerEquipo(e.target.value)}><option value="">{t('spaf.elegir_equipo')}</option>{opciones?.equipos.map(e => <option key={e.id} value={e.id}>{etiqueta('equipo', e.clave_texto)}</option>)}</select></div>}
      <div className="spaf-campo"><label htmlFor="spaf-mes">{t('spaf.mes')}</label><input id="spaf-mes" type="month" min="0001-01" max={mesAnterior()} disabled={emitiendo || recuperacion} value={mes} aria-invalid={error === 'spaf.mes_invalido'} aria-describedby={error === 'spaf.mes_invalido' ? 'spaf-error' : undefined} onChange={e => ponerMes(e.target.value)} /></div>
    </div></section>
    {cargando && <p role="status">{t('spaf.cargando')}</p>}
    {error && <div id="spaf-error" role="alert" className="spaf-aviso"><p>{t(error)}</p><Boton accion={() => error === 'spaf.descarga_desconocida' ? void descargar() : recargar(n => n + 1)} ocupado={emitiendo}>{t(conflicto ? 'spaf.ver_cambios' : error === 'spaf.descarga_desconocida' ? 'spaf.reintentar_descarga' : 'spaf.reintentar')}</Boton>{error === 'spaf.sin_permiso' && <p>{t('spaf.volver_sesion')}</p>}</div>}
    {!cargando && opciones && !equipo && <p role="status">{t('spaf.elige_equipo')}</p>}
    {recuperacion && !emitiendo && error !== 'spaf.descarga_desconocida' && <div role="status" className="spaf-aviso"><p>{t('spaf.descarga_desconocida')}</p>{!tabla && <Boton accion={() => void descargar()}>{t('spaf.reintentar_descarga')}</Boton>}</div>}
    {tabla && !cargando && <>
      <section className="spaf-resumen" aria-labelledby="spaf-resumen"><h2 id="spaf-resumen">{t('spaf.resumen')}</h2><div className="spaf-indicadores">{(['altas', 'bajas', 'casos', 'menores'] as const).map(k => <div className="spaf-tarjeta" key={k}><span>{t(k === 'menores' && tabla.resumen.casos_sin_dato_menores > 0 ? 'spaf.menores_conocidos' : `spaf.${k}`)}</span><strong>{k === 'menores' && tabla.resumen.casos > 0 && tabla.resumen.casos_sin_dato_menores === tabla.resumen.casos ? desconocido : numero(tabla.resumen[k])}</strong>{k === 'menores' && tabla.resumen.casos_sin_dato_menores > 0 && <span className="spaf-sin-dato">{formato(`spaf.total_incompleto_${new Intl.PluralRules(idioma).select(tabla.resumen.casos_sin_dato_menores) === 'one' ? 'one' : 'other'}`, { n: numero(tabla.resumen.casos_sin_dato_menores) })}</span>}</div>)}</div></section>
      {Object.entries(tabla.avisos ?? {}).filter(([, n]) => n > 0).map(([k, n]) => <p className="spaf-aviso" key={k}>{t(avisosPermitidos.includes(k) ? k : 'spaf.datos_incompletos')}{t('spaf.separador')}{formato(`spaf.afecta_${new Intl.PluralRules(idioma).select(n) === 'one' ? 'one' : 'other'}`, { n: numero(n) })}</p>)}
      <section className="spaf-tarjeta" aria-labelledby="spaf-descarga"><h2 id="spaf-descarga">{t('spaf.documento')}</h2><p>{t('spaf.aviso_descarga')}</p><Boton principal accion={() => void descargar()} ocupado={emitiendo || conflicto}>{emitiendo ? t('spaf.generando') : recuperacion ? t('spaf.reintentar_descarga') : formato('spaf.descargar_mes', { mes: mesTexto })}</Boton><p role="status">{estado && t(estado)}</p></section>
      <section className="spaf-tarjeta" aria-labelledby="spaf-movimientos"><h2 id="spaf-movimientos">{t('spaf.movimientos')}</h2><div className="spaf-movimientos">{(['alta', 'baja'] as const).map(k => <div key={k}><h3>{t(`spaf.${k}s`)}</h3>{filas.some(f => f[k]) ? <ul>{filas.filter(f => f[k]).map(f => <li key={f.expediente_id}><a href={enlaceExpediente(f)}>{f.numero || t('spaf.sin_numero')}</a></li>)}</ul> : <p>{t(`spaf.sin_${k}s`)}</p>}</div>)}</div></section>
      <section className="spaf-tarjeta" aria-labelledby="spaf-vista"><h2 id="spaf-vista">{t('spaf.vista')}</h2>{!filas.length ? <p role="status">{t('spaf.vacio')}</p> : <><p>{t('spaf.desplazar')}</p><div className="spaf-tabla-contenedor" tabIndex={0} role="region" aria-label={nombreRegion}><table><caption>{nombreEquipo}{t('spaf.separador')}{mesTexto}</caption><thead><tr>{columnas.map(k => <th key={k} scope="col">{t(`spaf.columna_${k}`)}</th>)}</tr></thead><tbody>{filas.map(f => <tr key={f.expediente_id}>{celdas(f).map((v, i) => i === 0 ? <th scope="row" key={i}>{v}{f.alta && <span className="spaf-marca">{t('spaf.alta')}</span>}{f.baja && <span className="spaf-marca">{t('spaf.baja')}</span>}</th> : <td key={i}>{v}</td>)}</tr>)}</tbody></table></div></>}</section>
    </>}
    <dialog ref={ayuda} className="spaf-dialogo" aria-labelledby="spaf-ayuda" onClose={() => ayudaBoton.current?.focus()}><h2 id="spaf-ayuda">{t('spaf.ayuda_titulo')}</h2><p>{t('spaf.ayuda')}</p><Boton accion={() => ayuda.current?.close()}>{t('spaf.cerrar_ayuda')}</Boton></dialog>
  </section>;
}
