import { useEffect, useRef, useState, type ReactNode, type FormEvent } from 'react';
import { useButton } from 'react-aria';
import { useCambios } from '../cambios';
import type { Acceso } from '../armazon';
import type { PropsPantalla } from '../registro-pantallas';
import './actividad.css';

type Tipo = 'comisiones' | 'informes';
type Registro = { id: string; equipo_id: string; fecha: string; version: number; observaciones?: string; organismo?: string; organismo_otros?: string };
type Opcion = { codigo: string; clave_texto: string; admite_texto: boolean };
type Equipo = { id: string; clave_texto: string };
type Campos = { fecha: string; observaciones: string; organismo: string; organismo_otros: string };
const vacios = (): Campos => ({ fecha: '', observaciones: '', organismo: '', organismo_otros: '' });
const uuidValido = (v: unknown): v is string => typeof v === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(v);
function fechaValida(v: unknown): v is string {
 if (typeof v !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(v) || +v.slice(0, 4) < 1 || +v.slice(0, 4) > 9998) return false;
 const d = new Date(`${v}T12:00:00Z`); return Number.isFinite(d.getTime()) && d.toISOString().slice(0, 10) === v;
}
function registroValido(v: unknown): v is Registro {
 if (!v || typeof v !== 'object') return false;
 const r = v as Registro;
 return uuidValido(r.id) && uuidValido(r.equipo_id) && fechaValida(r.fecha) && Number.isSafeInteger(r.version) && r.version > 0 &&
  [r.observaciones, r.organismo, r.organismo_otros].every((valor) => valor === undefined || typeof valor === 'string');
}

const errores = ['actividad.datos_invalidos', 'actividad.otros_obligatorio', 'actividad.acceso_denegado', 'actividad.no_encontrado', 'actividad.no_disponible', 'actividad.conflicto'];
class Fallo extends Error { constructor(public clave: string, public estado = 0) { super(clave); } }
async function pedir<T>(ruta: string, opciones: RequestInit): Promise<T> {
  let respuesta: Response;
  try { respuesta = await fetch(ruta, { credentials: 'same-origin', ...opciones }); }
  catch (e) { if (opciones.signal?.aborted) throw e; throw new Fallo('actividad.respuesta_desconocida'); }
  const cuerpo = await respuesta.json().catch(() => { throw new Fallo('actividad.respuesta_desconocida'); });
  if (!respuesta.ok) {
    const clave = typeof cuerpo.error === 'string' ? cuerpo.error : cuerpo.error?.clave;
    throw new Fallo(respuesta.status === 403 || respuesta.status === 401 ? 'actividad.acceso_denegado' : errores.includes(clave) ? clave : 'actividad.no_disponible', respuesta.status);
  }
  return cuerpo as T;
}
function Boton({ children, accion, principal = false, disabled = false, type = 'button', etiqueta }: { children: ReactNode; accion?: () => void; principal?: boolean; disabled?: boolean; type?: 'button' | 'submit'; etiqueta?: string }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion, isDisabled: disabled, type, 'aria-label': etiqueta }, ref);
  return <button {...buttonProps} ref={ref} className={principal ? undefined : 'boton-secundario'}>{children}</button>;
}

export function Actividad({ t, idioma, csrf }: PropsPantalla) {
  const { ponerSucio, solicitarSalida } = useCambios();
  const raiz = useRef<HTMLElement>(null);
  const [compacto, ponerCompacto] = useState(true);
  const [tipo, ponerTipo] = useState<Tipo>('comisiones');
  const [anio, ponerAnio] = useState(String(new Date().getFullYear()));
  const [equipo, ponerEquipo] = useState('');
  const [organismo, ponerOrganismo] = useState('');
  const [desde, ponerDesde] = useState('');
  const [hasta, ponerHasta] = useState('');
  const [filas, ponerFilas] = useState<Registro[]>([]);
  const [etiquetas, ponerEtiquetas] = useState<Record<string, Record<string, string>>>({});
  const [equipos, ponerEquipos] = useState<Equipo[]>([]);
  const [organismos, ponerOrganismos] = useState<Opcion[]>([]);
  const [acceso, ponerAcceso] = useState<Acceso>();
  const [revision, recargar] = useState(0);
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState('');
  const [estado, ponerEstado] = useState('');
  const [editando, ponerEditando] = useState<Registro | 'nuevo' | null>(null);
  const [campos, ponerCampos] = useState<Campos>(vacios);
  const [original, ponerOriginal] = useState<Campos>(vacios);
  const [faltas, ponerFaltas] = useState<string[]>([]);
  const [guardando, ponerGuardando] = useState(false);
  const [actual, ponerActual] = useState<Registro | null>();
  const [bloqueado, ponerBloqueado] = useState(false);
  const [retirada, ponerRetirada] = useState<Registro | null>(null);
  const tituloForm = useRef<HTMLHeadingElement>(null);
  const resumen = useRef<HTMLDivElement>(null);
  const dialogo = useRef<HTMLDialogElement>(null);
  const ayuda = useRef<HTMLDialogElement>(null);
  const mutacion = useRef<AbortController | null>(null);
  const profesional = acceso?.perfiles?.includes('profesional_etf') && !!acceso.equipo_id;
  const provincial = acceso?.perfiles?.includes('coordinacion');
  const traducir = (clave: string) => etiquetas[idioma]?.[clave] || etiquetas.es?.[clave] || t('actividad.etiqueta_pendiente');
  const fecha = (valor: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(`${valor}T12:00:00`));
  const nombreEquipo = (id: string) => traducir(equipos.find((e) => e.id === id)?.clave_texto ?? 'actividad.etiqueta_pendiente');
  const nombreOrganismo = (fila: Registro) => fila.organismo_otros || traducir(organismos.find((o) => o.codigo === fila.organismo)?.clave_texto ?? 'actividad.etiqueta_pendiente');
  const muestra = filas.filter((f) => (!desde || f.fecha >= desde) && (!hasta || f.fecha <= hasta) && (!organismo || f.organismo === organismo));
  const sucio = editando !== null && JSON.stringify(campos) !== JSON.stringify(original);
  useEffect(() => { ponerSucio(sucio || guardando); }, [sucio, guardando, ponerSucio]);
  useEffect(() => () => { mutacion.current?.abort(); ponerSucio(false); }, [ponerSucio]);
  useEffect(() => { if (editando) tituloForm.current?.focus(); }, [editando]);
  useEffect(() => { if (faltas.length) resumen.current?.focus(); }, [faltas]);
  useEffect(() => {
    const controlador = new AbortController();
    ponerCargando(true); ponerError(''); ponerFilas([]);
    if (!/^\d{1,4}$/.test(anio) || +anio < 1 || +anio > 9998) { ponerError('actividad.anio_invalido'); ponerCargando(false); return () => controlador.abort(); }
    const q = new URLSearchParams({ anio }); if (equipo) q.set('etf', equipo);
    void Promise.all([
      pedir<Registro[]>(`/api/${tipo}?${q}`, { signal: controlador.signal }),
      pedir<Equipo[]>('/api/actividad/equipos', { signal: controlador.signal }),
      pedir<Opcion[]>('/api/actividad/organismos', { signal: controlador.signal }),
      pedir<Record<string, Record<string, string>>>('/api/actividad/etiquetas', { signal: controlador.signal }),
      pedir<Acceso>('/api/sesion', { signal: controlador.signal }),
    ]).then(([registros, listaEquipos, listaOrganismos, textos, sesion]) => {
      if (controlador.signal.aborted) return;
      if (!Array.isArray(registros) || !registros.every(registroValido) || !Array.isArray(listaEquipos) || !listaEquipos.every((e) => uuidValido(e.id) && typeof e.clave_texto === 'string') || !Array.isArray(listaOrganismos) || !listaOrganismos.every((o) => typeof o.codigo === 'string' && typeof o.clave_texto === 'string' && typeof o.admite_texto === 'boolean')) throw new Fallo('actividad.no_disponible');
      if (!textos || typeof textos !== 'object' || !textos.es) throw new Fallo('actividad.no_disponible');
      ponerEtiquetas(textos); ponerFilas(registros); ponerEquipos(listaEquipos); ponerOrganismos(listaOrganismos); ponerAcceso(sesion);
    }).catch((e: unknown) => { if (!controlador.signal.aborted) ponerError(e instanceof Fallo ? e.clave === 'actividad.respuesta_desconocida' ? 'actividad.no_disponible' : e.clave : 'actividad.no_disponible'); })
      .finally(() => { if (!controlador.signal.aborted) ponerCargando(false); });
    return () => controlador.abort();
  }, [tipo, anio, equipo, revision]);

  useEffect(() => {
    const observar = new ResizeObserver(() => { const ancho = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu')); ponerCompacto((raiz.current?.clientWidth ?? 0) < ancho * 3); });
    if (raiz.current) observar.observe(raiz.current); return () => observar.disconnect();
  }, []);

  function cerrarFormulario() { ponerEditando(null); ponerFaltas([]); ponerError(''); ponerBloqueado(false); ponerActual(undefined); ponerSucio(false); requestAnimationFrame(() => document.getElementById('actividad-listado')?.focus()); }
  function seleccionar(fila?: Registro) {
    const c = fila ? { fecha: fila.fecha, observaciones: fila.observaciones ?? '', organismo: fila.organismo ?? '', organismo_otros: fila.organismo_otros ?? '' } : vacios();
    ponerCampos(c); ponerOriginal(c); ponerEditando(fila ?? 'nuevo'); ponerFaltas([]); ponerError(''); ponerEstado(''); ponerBloqueado(false); ponerActual(undefined);
  }
  function editar(clave: keyof Campos, valor: string) { ponerCampos((c) => ({ ...c, [clave]: valor, ...(clave === 'organismo' ? { organismo_otros: '' } : {}) })); ponerFaltas((f) => f.filter((v) => v !== clave)); }
  async function guardar(evento: FormEvent) {
    evento.preventDefault();
    if (guardando || bloqueado || !editando) return;
    const faltan = [!fechaValida(campos.fecha) ? 'fecha' : '', tipo === 'informes' && !campos.organismo ? 'organismo' : '', tipo === 'informes' && organismos.find((o) => o.codigo === campos.organismo)?.admite_texto && !campos.organismo_otros.trim() ? 'organismo_otros' : ''].filter(Boolean);
    ponerFaltas(faltan); if (faltan.length) return;
    const fila = editando === 'nuevo' ? null : editando;
    const cuerpo = { fecha: campos.fecha, version_esperada: fila?.version ?? 0, ...(tipo === 'comisiones' ? { observaciones: campos.observaciones } : { organismo: campos.organismo, organismo_otros: campos.organismo_otros }) };
    await escribir(fila ? 'PUT' : 'POST', fila ? `/${fila.id}` : '', cuerpo);
  }
  async function escribir(metodo: string, sufijo: string, cuerpo?: object) {
    if (guardando) return;
    const controlador = new AbortController(); mutacion.current = controlador;
    ponerGuardando(true); ponerError(''); ponerEstado('');
    try {
      const fila = await pedir<Registro>(`/api/${tipo}${sufijo}`, { method: metodo, signal: controlador.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}) });
      if (!registroValido(fila) || fila.equipo_id !== acceso?.equipo_id || (sufijo && fila.id !== sufijo.split('?')[0].slice(1))) throw new Fallo('actividad.respuesta_desconocida');
      cerrarFormulario(); ponerRetirada(null); dialogo.current?.close();
      ponerEstado(metodo === 'DELETE' ? 'actividad.retirado' : 'actividad.guardado');
      ponerAnio(fila.fecha.slice(0, 4)); ponerDesde(''); ponerHasta(''); ponerOrganismo(''); recargar((n) => n + 1);
    } catch (e: unknown) {
      if (controlador.signal.aborted) return;
      const fallo = e instanceof Fallo ? e : new Fallo('actividad.respuesta_desconocida');
      ponerError(fallo.clave); ponerBloqueado(fallo.estado === 409 || fallo.clave === 'actividad.respuesta_desconocida' || fallo.clave === 'actividad.acceso_denegado' || fallo.clave === 'actividad.no_encontrado');
      ponerRetirada(null); dialogo.current?.close();
    } finally { if (!controlador.signal.aborted) ponerGuardando(false); }
  }
  async function verActual() {
    if (!editando || editando === 'nuevo' || guardando) return;
    const controlador = new AbortController(); mutacion.current = controlador; ponerGuardando(true);
    try {
      const registros = await pedir<Registro[]>(`/api/${tipo}?anio=${editando.fecha.slice(0, 4)}`, { signal: controlador.signal });
      if (!Array.isArray(registros) || !registros.every(registroValido)) throw new Fallo('actividad.no_disponible');
      if (!controlador.signal.aborted) ponerActual(registros.find((f) => f.id === editando.id) ?? null);
    } catch (e) { if (!controlador.signal.aborted) ponerError(e instanceof Fallo ? e.clave : 'actividad.no_disponible'); }
    finally { if (!controlador.signal.aborted) ponerGuardando(false); }
  }
  function usarActual() {
    if (!actual) return;
    ponerEditando(actual); ponerOriginal({ fecha: actual.fecha, observaciones: actual.observaciones ?? '', organismo: actual.organismo ?? '', organismo_otros: actual.organismo_otros ?? '' });
    ponerActual(undefined); ponerBloqueado(false); ponerError('');
  }
  function actualizarLista() { solicitarSalida(() => { cerrarFormulario(); recargar((n) => n + 1); }); }
  const campo = (clave: keyof Campos, control: ReactNode) => <div className="actividad-campo"><label htmlFor={`actividad-${clave}`}>{t(`actividad.${clave}`)}</label>{control}{faltas.includes(clave) && <span id={`falta-${clave}`}>{t('actividad.obligatorio')}</span>}</div>;
  const atributos = (clave: keyof Campos) => ({ id: `actividad-${clave}`, 'aria-invalid': faltas.includes(clave), 'aria-describedby': faltas.includes(clave) ? `falta-${clave}` : undefined });

  return <section ref={raiz} data-compacto={compacto} className="actividad" aria-labelledby="actividad-titulo" aria-busy={guardando}>
    <header className="actividad-cabecera"><div><h1 id="actividad-titulo">{t('actividad.titulo')}</h1><p>{t(provincial ? 'actividad.ambito_provincia' : 'actividad.ambito_equipo')}</p></div><Boton accion={() => ayuda.current?.showModal()} etiqueta={t('actividad.abrir_ayuda')}>{t('actividad.simbolo_ayuda')}</Boton></header>
    <fieldset className="actividad-tipos" disabled={guardando}><legend>{t('actividad.tipo')}</legend>{(['comisiones', 'informes'] as Tipo[]).map((valor) => <label key={valor}><input type="radio" name="actividad-tipo" checked={tipo === valor} onChange={() => solicitarSalida(() => { cerrarFormulario(); ponerTipo(valor); ponerOrganismo(''); ponerEstado(''); })} />{t(`actividad.${valor}`)}</label>)}</fieldset>
    <p role="status">{guardando ? t('actividad.guardando') : cargando ? t('actividad.cargando') : estado ? t(estado) : ''}</p>
    {error && <div role="alert"><p>{t(error)}</p><Boton disabled={guardando} accion={bloqueado && editando && editando !== 'nuevo' ? () => void verActual() : actualizarLista}>{t(bloqueado ? 'actividad.ver_cambios' : 'actividad.reintentar')}</Boton></div>}
    {editando ? <form onSubmit={(e) => void guardar(e)} noValidate className="actividad-tarjeta">
      <h2 ref={tituloForm} tabIndex={-1}>{t(editando === 'nuevo' ? `actividad.registrar_${tipo}` : `actividad.corregir_${tipo}`)}</h2>
      {actual !== undefined && <section className="actividad-tarjeta" aria-labelledby="actividad-actual-titulo"><h3 id="actividad-actual-titulo">{t('actividad.datos_actuales')}</h3>{actual ? <><dl><dt>{t('actividad.fecha')}</dt><dd>{fecha(actual.fecha)}</dd><dt>{t(tipo === 'comisiones' ? 'actividad.observaciones_tabla' : 'actividad.organismo')}</dt><dd>{tipo === 'comisiones' ? actual.observaciones || t('actividad.sin_observaciones') : nombreOrganismo(actual)}</dd></dl><Boton accion={usarActual}>{t('actividad.usar_version_actual')}</Boton></> : <p>{t('actividad.fuera_del_anio')}</p>}</section>}
      {faltas.length > 0 && <div ref={resumen} tabIndex={-1} role="alert" className="actividad-faltas"><p>{t('actividad.faltan_datos')}</p><ul>{faltas.map((clave) => <li key={clave}><a href={`#actividad-${clave}`} onClick={(e) => { e.preventDefault(); document.getElementById(`actividad-${clave}`)?.focus(); }}>{t(`actividad.${clave}`)}</a></li>)}</ul></div>}
      <fieldset disabled={guardando}><legend>{t('actividad.datos_registro')}</legend>
        {campo('fecha', <input {...atributos('fecha')} type="date" required max="9998-12-31" value={campos.fecha} onChange={(e) => editar('fecha', e.target.value)} />)}
        {tipo === 'comisiones' ? campo('observaciones', <textarea {...atributos('observaciones')} maxLength={4000} value={campos.observaciones} onChange={(e) => editar('observaciones', e.target.value)} />) : <>
          {campo('organismo', <select {...atributos('organismo')} required value={campos.organismo} onChange={(e) => editar('organismo', e.target.value)}><option value="">{t('actividad.elige_organismo')}</option>{organismos.map((o) => <option key={o.codigo} value={o.codigo}>{traducir(o.clave_texto)}</option>)}</select>)}
          {organismos.find((o) => o.codigo === campos.organismo)?.admite_texto && campo('organismo_otros', <input {...atributos('organismo_otros')} required maxLength={500} value={campos.organismo_otros} onChange={(e) => editar('organismo_otros', e.target.value)} />)}
          <p>{t('actividad.informe_emitido')}</p>
        </>}
      </fieldset>
      <div className="actividad-acciones"><Boton type="submit" principal disabled={guardando || bloqueado}>{t('actividad.guardar')}</Boton><Boton disabled={guardando} accion={() => solicitarSalida(cerrarFormulario)}>{t('actividad.cancelar')}</Boton></div>
    </form> : <>
      <div className="actividad-filtros">
        <div className="actividad-campo"><label htmlFor="actividad-anio">{t('actividad.anio')}</label><input id="actividad-anio" type="number" min="1" max="9998" value={anio} onChange={(e) => { ponerAnio(e.target.value); ponerDesde(''); ponerHasta(''); }} /></div>
        {provincial && <div className="actividad-campo"><label htmlFor="actividad-equipo">{t('actividad.equipo')}</label><select id="actividad-equipo" value={equipo} onChange={(e) => ponerEquipo(e.target.value)}><option value="">{t('actividad.todos_equipos')}</option>{equipos.map((e) => <option key={e.id} value={e.id}>{traducir(e.clave_texto)}</option>)}</select></div>}
        <div className="actividad-campo"><label htmlFor="actividad-desde">{t('actividad.desde')}</label><input id="actividad-desde" type="date" value={desde} onChange={(e) => ponerDesde(e.target.value)} /></div>
        <div className="actividad-campo"><label htmlFor="actividad-hasta">{t('actividad.hasta')}</label><input id="actividad-hasta" type="date" value={hasta} onChange={(e) => ponerHasta(e.target.value)} /></div>
        {tipo === 'informes' && <div className="actividad-campo"><label htmlFor="actividad-filtro-organismo">{t('actividad.organismo')}</label><select id="actividad-filtro-organismo" value={organismo} onChange={(e) => ponerOrganismo(e.target.value)}><option value="">{t('actividad.todos_organismos')}</option>{organismos.map((o) => <option key={o.codigo} value={o.codigo}>{traducir(o.clave_texto)}</option>)}</select></div>}
      </div>
      <div className="actividad-cabecera"><h2 id="actividad-listado" tabIndex={-1}>{t(`actividad.${tipo}`)}</h2>{profesional && !error && !cargando && <Boton principal accion={() => seleccionar()}>{t(`actividad.registrar_${tipo}`)}</Boton>}</div>
      {!cargando && !error && <><p role="status">{new Intl.NumberFormat(idioma).format(muestra.length)} {t(`actividad.cantidad_${tipo}_${new Intl.PluralRules(idioma).select(muestra.length)}`)}</p>
        {!muestra.length ? <p className="actividad-tarjeta">{t(`actividad.vacio_${tipo}`)}</p> : <table className="actividad-tabla"><caption>{t(`actividad.lista_${tipo}`)}</caption><thead><tr><th scope="col">{t('actividad.fecha')}</th><th scope="col">{t('actividad.equipo')}</th><th scope="col">{t(tipo === 'comisiones' ? 'actividad.observaciones_tabla' : 'actividad.organismo')}</th>{profesional && <th scope="col">{t('actividad.acciones')}</th>}</tr></thead><tbody>{muestra.map((f) => <tr key={f.id}><th scope="row"><span className="actividad-rotulo">{t('actividad.fecha')}</span>{fecha(f.fecha)}</th><td><span className="actividad-rotulo">{t('actividad.equipo')}</span>{nombreEquipo(f.equipo_id)}</td><td><span className="actividad-rotulo">{t(tipo === 'comisiones' ? 'actividad.observaciones_tabla' : 'actividad.organismo')}</span>{tipo === 'comisiones' ? f.observaciones || t('actividad.sin_observaciones') : nombreOrganismo(f)}</td>{profesional && <td>{f.equipo_id === acceso?.equipo_id && <div className="actividad-acciones"><Boton accion={() => seleccionar(f)}>{t('actividad.corregir')}</Boton><Boton disabled={bloqueado || guardando} accion={() => { ponerRetirada(f); dialogo.current?.showModal(); }}>{t('actividad.retirar')}</Boton></div>}</td>}</tr>)}</tbody></table>}
      </>}
    </>}
    <dialog ref={dialogo} className="actividad-dialogo" aria-labelledby="actividad-retirar-titulo" onCancel={(e) => { if (guardando) e.preventDefault(); else ponerRetirada(null); }}><h2 id="actividad-retirar-titulo">{t(`actividad.retirar_${tipo}`)}</h2>{retirada && <p>{fecha(retirada.fecha)}{' '}{nombreEquipo(retirada.equipo_id)}</p>}<p>{t('actividad.consecuencia_retirar')}</p><div className="actividad-acciones"><Boton disabled={guardando} accion={() => { dialogo.current?.close(); ponerRetirada(null); }}>{t('actividad.cancelar')}</Boton><Boton principal disabled={guardando || !retirada} accion={() => { if (retirada) void escribir('DELETE', `/${retirada.id}?version_esperada=${retirada.version}`); }}>{t('actividad.confirmar_retirada')}</Boton></div></dialog>
    <dialog ref={ayuda} className="actividad-dialogo" aria-labelledby="actividad-ayuda-titulo"><h2 id="actividad-ayuda-titulo">{t('actividad.titulo')}</h2><p>{t('actividad.ayuda')}</p><Boton accion={() => ayuda.current?.close()}>{t('actividad.cerrar_ayuda')}</Boton></dialog>
  </section>;
}
