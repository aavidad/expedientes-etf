import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import type { PropsPantalla } from '../registro-pantallas';
import { FalloMemoria, pedir, type Equipo } from './api';
import { descargarCifras, leerCifras, type ApartadoConsulta, type CifrasConsulta } from './consulta-api';
import { etiquetaCifra } from './presentacion';
import './memoria.css';

const bloques = [[1, 6], [7, 13], [14, 21], [22, 28]] as const;
function BotonConsulta({ children, onPress, disabled = false, secundario = false }: { children: React.ReactNode; onPress: () => void; disabled?: boolean; secundario?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress, isDisabled: disabled }, ref);
  return <button {...buttonProps} ref={ref} className={secundario ? 'boton-secundario' : undefined}>{children}</button>;
}

export function ConsultaMemoria({ t, idioma, csrf, anioInicial, equipoInicial, equipoPropio, alConsultarEquipoPropio }: Pick<PropsPantalla, 't' | 'idioma' | 'csrf'> & { anioInicial?: number; equipoInicial?: string; equipoPropio?: string; alConsultarEquipoPropio?: (anio: number, equipo: string) => void }) {
  const [anio, ponerAnio] = useState(anioInicial ?? new Date().getFullYear());
  const [equipo, ponerEquipo] = useState(equipoInicial ?? '');
  const [equipos, ponerEquipos] = useState<Equipo[]>([]);
  const [cifras, ponerCifras] = useState<CifrasConsulta>();
  const [bloque, ponerBloque] = useState(0);
  const [estado, ponerEstado] = useState('cargando');
  const [error, ponerError] = useState<FalloMemoria>();
  const [ayuda, ponerAyuda] = useState(false);
  const [aviso, ponerAviso] = useState('');
  const carga = useRef<AbortController | undefined>(undefined);
  const descarga = useRef<AbortController | undefined>(undefined);
  const incidencia = useRef<HTMLDivElement>(null);
  const accionError = useRef('cargar');
  const nombreBloques = t('memoria.bloques');
  const claseError = ['memoria-aviso', 'memoria-tarjeta'].join(' ');
  const ocupado = estado === 'cargando' || estado === 'descargando';
  const numero = (n: number | null) => n === null ? t('memoria.cifra_oculta') : new Intl.NumberFormat(idioma).format(n);
  const fecha = (valor: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(valor));
  const fallo = (e: unknown, signal: AbortSignal) => {
    if (signal.aborted) return;
    ponerError(e instanceof FalloMemoria ? e : new FalloMemoria('estadistica.memoria_no_disponible'));
    ponerEstado('error');
  };
  const cargar = async (a = anio, e = equipo) => {
    carga.current?.abort(); const c = new AbortController(); carga.current = c;
    accionError.current = 'cargar';
    ponerEstado('cargando'); ponerCifras(undefined); ponerError(undefined); ponerAviso('');
    try {
      const [datos, lista] = await Promise.all([leerCifras(a, e, c.signal), pedir<Equipo[]>('/api/estadistica/memoria/equipos', { signal: c.signal })]);
      if (c.signal.aborted) return;
      ponerCifras(datos); ponerEquipos(lista); ponerEstado('lista');
    } catch (e) { fallo(e, c.signal); }
  };
  useEffect(() => { void cargar(); return () => { carga.current?.abort(); descarga.current?.abort(); }; }, []);
  useEffect(() => { document.title = t('memoria.titulo'); }, [t]);
  useEffect(() => { if (error) incidencia.current?.focus(); }, [error]);
  const word = async () => {
    if (!cifras || ocupado) return;
    const c = new AbortController(); descarga.current = c;
    accionError.current = 'descargar';
    ponerEstado('descargando'); ponerError(undefined); ponerAviso('');
    try {
      const archivo = await descargarCifras(cifras.anio, cifras.equipo_id, idioma, csrf, c.signal);
      if (c.signal.aborted) return;
      const url = URL.createObjectURL(archivo); const enlace = document.createElement('a'); enlace.href = url; enlace.download = `memoria-cifras-${cifras.anio}.docx`; enlace.click(); URL.revokeObjectURL(url);
      ponerEstado('lista'); ponerAviso('memoria.descargada');
    } catch (e) { fallo(e, c.signal); }
  };
  const mostrar = (a: ApartadoConsulta) => <article className="memoria-apartado" key={a.codigo} id={`memoria-apartado-${a.codigo}`}>
    <h3>{t(`memoria.apartado.${a.codigo}`)}</h3>
    {Object.entries(a.avisos ?? {}).filter(([, n]) => n > 0).map(([clave, n]) => <p className="memoria-aviso" key={clave}>{t(clave) || t('memoria.datos_faltantes')} {numero(n)}</p>)}
    <div className="memoria-tabla" role="region" tabIndex={0} aria-label={t(`memoria.apartado.${a.codigo}`)}>
      <table><caption>{t(`memoria.apartado.${a.codigo}`)}</caption><thead><tr>{['grupo', 'unidad', 'cantidad', 'base', 'porcentaje', 'sin_dato'].map((c) => <th scope="col" key={c}>{t(`memoria.${c}`)}</th>)}</tr></thead>
        <tbody>{a.celdas.map((c, i) => <tr key={i}>
          <th scope="row">{[...Object.entries(c.grupo).filter(([dimension, valor]) => !!valor && !((c.sin_dato ?? 0) > 0 && c.numerador === 0 && dimension === 'estado' && valor === 'no')).map(([dimension, valor]) => etiquetaCifra(t, a.codigo, dimension, valor)), ...((c.sin_dato ?? 0) > 0 && c.numerador === 0 ? [t('memoria.sin_dato')] : [])].join(t('memoria.separador')) || t('memoria.sin_dato')}</th>
          <td>{t(`memoria.unidad.${c.unidad}`)}</td><td>{numero(c.numerador)}</td><td>{c.denominador === null ? t(cifras?.margenes_ocultos ? 'memoria.cifra_oculta' : 'memoria.sin_dato') : numero(c.denominador)}</td>
          <td>{c.porcentaje_centesimas === null ? t(cifras?.margenes_ocultos ? 'memoria.cifra_oculta' : 'memoria.sin_dato') : new Intl.NumberFormat(idioma, { style: 'percent', minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(c.porcentaje_centesimas / 10000)}{c.aviso_porcentaje && <span className="memoria-aviso">{t(c.aviso_porcentaje)}</span>}</td><td className={c.sin_dato ? 'memoria-aviso' : undefined}>{numero(c.sin_dato)}</td>
        </tr>)}</tbody>
      </table>
    </div>
    {!a.celdas.length && <p>{t('memoria.sin_poblacion')}</p>}
    <p>{t('memoria.relacion_desconocida')} {numero(a.total.familias_sin_relacion_menores)}</p>
  </article>;
  return <section className="memoria" aria-labelledby="memoria-titulo">
    <div className="memoria-cabecera"><h1 id="memoria-titulo">{t('memoria.titulo')}</h1><BotonConsulta secundario onPress={() => ponerAyuda(!ayuda)}>{t('memoria.ayuda')}</BotonConsulta><BotonConsulta disabled={ocupado || !cifras} onPress={() => void word()}>{t('memoria.descargar')}</BotonConsulta></div>
    {ayuda && <p className="memoria-tarjeta">{t('memoria.consulta_ayuda')}</p>}
    <form className="memoria-filtros" onSubmit={(e) => { e.preventDefault(); if (equipoPropio && equipo === equipoPropio && alConsultarEquipoPropio) alConsultarEquipoPropio(anio, equipo); else void cargar(); }}>
      <div><label htmlFor="memoria-anio">{t('memoria.anio')}</label><input id="memoria-anio" type="number" required min={1} max={new Date().getFullYear()} disabled={ocupado} value={anio} onChange={(e) => ponerAnio(Number(e.target.value))} /></div>
      <div><label htmlFor="memoria-equipo">{t('memoria.equipo')}</label><select id="memoria-equipo" disabled={ocupado} value={equipo} onChange={(e) => ponerEquipo(e.target.value)}><option value="">{t('memoria.provincial')}</option>{equipos.map((e) => <option key={e.id} value={e.id}>{t('memoria.equipo_opcion').replace('{codigo}', e.codigo)}</option>)}</select></div>
      <button type="submit" className="boton-secundario" disabled={ocupado}>{t('memoria.consultar')}</button>
    </form>
    <p role="status">{ocupado ? t(`memoria.${estado}`) : aviso ? t(aviso) : ''}</p>
    {error && <div className={claseError} role="alert" tabIndex={-1} ref={incidencia}><p>{t(error.clave) || t('estadistica.memoria_no_disponible')}</p><BotonConsulta secundario onPress={() => void (accionError.current === 'descargar' ? word() : cargar())}>{t('memoria.reintentar')}</BotonConsulta></div>}
    {cifras && <>
      <p>{t('memoria.ambito').replace('{anio}', new Intl.NumberFormat(idioma, { useGrouping: false }).format(cifras.anio)).replace('{equipo}', cifras.equipo_codigo ? t('memoria.equipo_opcion').replace('{codigo}', cifras.equipo_codigo) : t('memoria.provincial'))}</p>
      <p>{t('memoria.consulta_fecha').replace('{fecha}', fecha(cifras.fecha_cifras))}</p><p className="pildora">{t(cifras.definitivas ? 'memoria.consulta_definitiva' : 'memoria.consulta_provisional')}</p>
      {cifras.umbral_privacidad && <p role="note" className="memoria-aviso">{t('memoria.privacidad_consulta').replace('{umbral}', numero(cifras.umbral_privacidad))}</p>}
      <div className="memoria-bloques" role="group" aria-label={nombreBloques}>{bloques.map((_, i) => <button className="boton-secundario" key={i} type="button" aria-pressed={bloque === i} onClick={() => ponerBloque(i)}>{t(`memoria.bloque.${i}`)}</button>)}</div>
      <h2>{t(`memoria.bloque.${bloque}`)}</h2>{cifras.apartados.filter((a) => Number(a.codigo.split('.')[0]) >= bloques[bloque][0] && Number(a.codigo.split('.')[0]) <= bloques[bloque][1]).sort((a, b) => a.codigo.localeCompare(b.codigo, undefined, { numeric: true })).map(mostrar)}
      {!cifras.apartados.length && <p>{t('memoria.consulta_vacia')}</p>}
    </>}
  </section>;
}
