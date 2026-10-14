import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useButton } from 'react-aria';
import type { PropsPantalla } from '../registro-pantallas';
import './bandeja.css';

type Opcion = { id: string; etiqueta: string; codigo: string };
type Fila = { id: string; numero: string; iniciales: string; estado: string; fecha_recepcion: string; fecha_aceptacion: string; equipo_id: string; municipio_id: string; municipio_propuesto: string; subprograma: string };
type Resultado = { filas: Fila[]; total: number; resumen: Record<string, number>; equipos: Opcion[]; municipios: Opcion[] };
type Filtro = { estado: string; equipo_id: string; municipio_id: string; desde: string; hasta: string; busqueda: string; limite: number; desplazamiento: number };
const inicial: Filtro = { estado: 'derivacion_recibida', equipo_id: '', municipio_id: '', desde: '', hasta: '', busqueda: '', limite: 20, desplazamiento: 0 };
const vistas = ['derivacion_recibida', 'aceptada', 'no_aceptada', 'lista_espera'] as const;

function Boton({ children, pulsar, desactivado, seleccionado, clase }: { children: ReactNode; pulsar: () => void; desactivado?: boolean; seleccionado?: boolean; clase?: string }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: pulsar, isDisabled: desactivado, 'aria-pressed': seleccionado }, ref);
  return <button {...buttonProps} ref={ref} className={clase}>{children}</button>;
}

export function Bandeja({ t, idioma, csrf }: PropsPantalla) {
  const [filtro, ponerFiltro] = useState<Filtro>(inicial);
  const [busqueda, ponerBusqueda] = useState('');
  const [resultado, ponerResultado] = useState<Resultado>();
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState('');
  const [intento, reintentar] = useState(0);
  const cambiar = (cambio: Partial<Filtro>) => ponerFiltro((previo) => ({ ...previo, ...cambio, desplazamiento: 0 }));
  const numero = (n: number) => new Intl.NumberFormat(idioma).format(n);
  const formato = (clave: string, valores: Record<string, string>) => Object.entries(valores).reduce((texto, [k, v]) => texto.replace(`{${k}}`, v), t(clave));
  const fecha = (valor: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: 'Europe/Madrid' }).format(new Date(valor.length === 10 ? `${valor}T12:00:00Z` : valor));

  useEffect(() => {
    const controlador = new AbortController();
    ponerCargando(true); ponerError('');
    void fetch('/api/coordinacion/bandeja/buscar', { method: 'POST', signal: controlador.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify(filtro) })
      .then(async (respuesta) => {
        if (!respuesta.ok) throw new Error(respuesta.status === 403 ? 'expediente.acceso_denegado' : respuesta.status === 400 ? 'expediente.peticion_invalida' : 'expediente.no_disponible');
        const datos = await respuesta.json() as Resultado;
        if (!controlador.signal.aborted) ponerResultado(datos);
      }).catch((fallo: unknown) => { if (!controlador.signal.aborted) ponerError(fallo instanceof Error && ['expediente.acceso_denegado', 'expediente.peticion_invalida', 'expediente.no_disponible'].includes(fallo.message) ? fallo.message : 'expediente.no_disponible'); })
      .finally(() => { if (!controlador.signal.aborted) ponerCargando(false); });
    return () => controlador.abort();
  }, [filtro, intento, csrf]);

  const opciones = (tipo: 'equipos' | 'municipios', id: string, propuesta = '') => resultado?.[tipo].find((o) => id ? o.id === id : o.codigo === propuesta)?.etiqueta ?? t('bandeja.por_confirmar');
  const etiquetaVistas = t('bandeja.vistas');
  const pagina = Math.floor(filtro.desplazamiento / filtro.limite) + 1;
  const paginas = Math.max(1, Math.ceil((resultado?.total ?? 0) / filtro.limite));

  return <section className="bandeja" aria-labelledby="bandeja-titulo">
    <header className="bandeja-cabecera"><div><h1 id="bandeja-titulo">{t('bandeja.titulo')}</h1><p>{t('bandeja.ambito')}</p></div><a className="bandeja-principal" href="/coordinacion/anexos/nuevo">{t('bandeja.subir')}</a></header>
    <div className="bandeja-vistas" role="group" aria-label={etiquetaVistas}>
      <a className="bandeja-indicador" href="/coordinacion/cierres">{t('salidas.cierres')}</a>
      {vistas.map((estado) => <Boton key={estado} pulsar={() => cambiar({ estado })} seleccionado={filtro.estado === estado} clase={`bandeja-indicador ${estado === 'derivacion_recibida' ? 'bandeja-aviso' : ''}`}><span>{t(`bandeja.estado.${estado}`)}</span><strong>{resultado ? numero(resultado.resumen[estado] ?? 0) : t('bandeja.sin_cifra')}</strong></Boton>)}
    </div>
    <section className="bandeja-listado" aria-labelledby="bandeja-lista">
      <h2 id="bandeja-lista">{t('bandeja.lista')}</h2>
      <form className="bandeja-buscar" onSubmit={(evento) => { evento.preventDefault(); cambiar({ busqueda }); }}>
        <label htmlFor="bandeja-busqueda">{t('bandeja.buscar_etiqueta')}</label><div><input id="bandeja-busqueda" type="search" maxLength={100} autoComplete="off" value={busqueda} onChange={(evento) => ponerBusqueda(evento.target.value)} /><button type="submit" className="boton-secundario">{t('bandeja.buscar')}</button></div>
      </form>
      <div className="bandeja-filtros">
        <label>{t('bandeja.estado')}<select value={filtro.estado} onChange={(e) => cambiar({ estado: e.target.value })}><option value="">{t('bandeja.todos_estados')}</option>{vistas.map((v) => <option key={v} value={v}>{t(`bandeja.estado.${v}`)}</option>)}</select></label>
        {(['equipos', 'municipios'] as const).map((tipo) => <label key={tipo}>{t(`bandeja.${tipo}`)}<select value={tipo === 'equipos' ? filtro.equipo_id : filtro.municipio_id} onChange={(e) => cambiar(tipo === 'equipos' ? { equipo_id: e.target.value } : { municipio_id: e.target.value })}><option value="">{t(`bandeja.todos_${tipo}`)}</option>{resultado?.[tipo].map((o) => <option key={o.id} value={o.id}>{o.etiqueta}</option>)}</select></label>)}
        <label>{t('bandeja.desde')}<input type="date" value={filtro.desde} onChange={(e) => cambiar({ desde: e.target.value })} /></label>
        <label>{t('bandeja.hasta')}<input type="date" value={filtro.hasta} onChange={(e) => cambiar({ hasta: e.target.value })} /></label>
      </div>
      <Boton clase="boton-secundario" pulsar={() => { ponerBusqueda(''); ponerFiltro(inicial); }}>{t('bandeja.limpiar')}</Boton>
      <p role="status" aria-live="polite">{cargando ? t('bandeja.cargando') : error ? '' : formato(resultado?.total === 1 ? 'bandeja.resultado' : 'bandeja.resultados', { total: numero(resultado?.total ?? 0) })}</p>
      {error && <div role="alert"><p>{t(error)}</p><Boton pulsar={() => reintentar((n) => n + 1)}>{t('bandeja.reintentar')}</Boton></div>}
      {!cargando && !error && resultado?.filas.length === 0 && <p>{t(filtro.busqueda || filtro.equipo_id || filtro.municipio_id || filtro.desde || filtro.hasta ? 'bandeja.vacio_filtros' : `bandeja.vacio.${filtro.estado || 'todos'}`)}</p>}
      {!cargando && !error && !!resultado?.filas.length && <table className="bandeja-tabla" aria-labelledby="bandeja-lista"><thead><tr>{['familia', 'estado', 'recepcion', 'municipios', 'equipos', 'subprograma'].map((col) => <th scope="col" key={col}>{t(`bandeja.${col}`)}</th>)}</tr></thead><tbody>{resultado.filas.map((fila) => <tr key={fila.id}>
        <th scope="row">{fila.numero && <a href={`/coordinacion/expedientes/${fila.id}/salidas`}>{t('salidas.titulo')}</a>}<a href={`/coordinacion/derivaciones/${fila.id}`}>{fila.numero || formato('bandeja.identificador_derivacion', { fecha: fecha(fila.fecha_recepcion), municipio: opciones('municipios', fila.municipio_id, fila.municipio_propuesto) })}</a><span>{fila.iniciales ? formato('bandeja.familia_iniciales', { iniciales: fila.iniciales }) : t('bandeja.familia_sin_nombre')}</span></th>
        <td><span className={`pildora ${fila.estado === 'derivacion_recibida' ? 'estado-pendiente_alta' : ''}`}>{t(`bandeja.estado.${fila.estado}`)}</span>{fila.fecha_aceptacion && <small>{t('bandeja.aceptacion')} {fecha(fila.fecha_aceptacion)}</small>}</td>
        <td><span className="bandeja-rotulo">{t('bandeja.recepcion')}</span>{fecha(fila.fecha_recepcion)}</td>
        <td><span className="bandeja-rotulo">{t('bandeja.municipios')}</span>{opciones('municipios', fila.municipio_id, fila.municipio_propuesto)}{!fila.municipio_id && fila.municipio_propuesto && <small>{t('bandeja.por_confirmar')}</small>}</td>
        <td><span className="bandeja-rotulo">{t('bandeja.equipos')}</span>{opciones('equipos', fila.equipo_id)}</td>
        <td><span className="bandeja-rotulo">{t('bandeja.subprograma')}</span>{fila.subprograma ? t(`bandeja.subprograma.${fila.subprograma.toLowerCase()}`) || t('bandeja.por_confirmar') : t('bandeja.por_confirmar')}</td>
      </tr>)}</tbody></table>}
      {!error && <div className="bandeja-paginacion"><label>{t('bandeja.tamano')}<select value={filtro.limite} onChange={(e) => cambiar({ limite: Number(e.target.value) })}>{[20, 50, 100].map((n) => <option key={n} value={n}>{numero(n)}</option>)}</select></label><span>{formato('bandeja.pagina', { actual: numero(pagina), paginas: numero(paginas) })}</span><Boton clase="boton-secundario" desactivado={cargando || pagina <= 1} pulsar={() => ponerFiltro((p) => ({ ...p, desplazamiento: Math.max(0, p.desplazamiento - p.limite) }))}>{t('bandeja.anterior')}</Boton><Boton clase="boton-secundario" desactivado={cargando || pagina >= paginas} pulsar={() => ponerFiltro((p) => ({ ...p, desplazamiento: p.desplazamiento + p.limite }))}>{t('bandeja.siguiente')}</Boton></div>}
    </section>
  </section>;
}
