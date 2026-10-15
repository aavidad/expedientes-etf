import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import type { PropsPantalla } from '../registro-pantallas';
import type { Acceso } from '../armazon';
import { useCambios } from '../cambios';
import { consulta, cifrasDeInstantanea, descargar, ejecutarActo, FalloMemoria, pedir, type EstadoMemoria, type Instantanea, type ActoPendiente, type Apartado, type Equipo, type Informe, type Narrativas, type Personal } from './api';
import { etiquetaCifra } from './presentacion';
import { ConsultaMemoria } from './ConsultaMemoria';
import './memoria.css';

function Boton({ children, onPress, disabled = false, secundario = false }: { children: React.ReactNode; onPress: () => void; disabled?: boolean; secundario?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress, isDisabled: disabled }, ref);
  return <button {...buttonProps} ref={ref} className={secundario ? 'boton-secundario' : undefined}>{children}</button>;
}
const limites = [[1, 6], [7, 13], [14, 21], [22, 28]] as const;
const personas = ['psicologia', 'trabajo_social', 'educacion_social'] as const;
const sexos = ['hombres', 'mujeres'] as const;
const textos = ['d', 'e', 'f', 'g'] as const;

export function Memoria({ t, idioma, csrf, navegarRuta }: PropsPantalla) {
  const [anio, ponerAnio] = useState(Number(new URLSearchParams(location.search).get('anio')) || new Date().getFullYear());
  const [equipo, ponerEquipo] = useState(new URLSearchParams(location.search).get('etf') ?? '');
  const [equipos, ponerEquipos] = useState<Equipo[]>([]);
  const [acceso, ponerAcceso] = useState<Acceso>();
  const [informe, ponerInforme] = useState<Informe>();
  const [narrativa, ponerNarrativa] = useState<Narrativas>();
  const [original, ponerOriginal] = useState('');
  const [situacion, ponerSituacion] = useState<EstadoMemoria>();
  const [cerradaEn, ponerCerradaEn] = useState('');
  const [accion, ponerAccion] = useState<'cerrar' | 'reabrir'>('cerrar');
  const [motivo, ponerMotivo] = useState('');
  const dialogo = useRef<HTMLDialogElement>(null);
  const botonActo = useRef<HTMLButtonElement>(null);
  const actoPendiente = useRef<ActoPendiente | undefined>(undefined);
  const [estado, ponerEstado] = useState('cargando');
  const [vistaConsulta, ponerVistaConsulta] = useState(false);
  const [error, ponerError] = useState<FalloMemoria>();
  const [aviso, ponerAviso] = useState('');
  const [bloque, ponerBloque] = useState(() => { const ap = Number(new URLSearchParams(location.search).get('apartado')); return limites.findIndex(([desde, hasta]) => ap >= desde && ap < hasta + 1) < 0 ? 0 : limites.findIndex(([desde, hasta]) => ap >= desde && ap < hasta + 1); });
  const [ayuda, ponerAyuda] = useState(false);
  const carga = useRef<AbortController | undefined>(undefined);
  const mutacion = useRef<AbortController | undefined>(undefined);
  const incidencia = useRef<HTMLDivElement>(null);
  const claveWord = useRef('');
  const solicitudWord = useRef('');
  const accionError = useRef('cargar');
  const { ponerSucio, solicitarSalida } = useCambios();
  const edita = !!acceso?.perfiles?.includes('coordinacion') || (!!acceso?.perfiles?.includes('profesional_etf') && !!informe && informe.equipo_id === acceso.equipo_id);
  const cerrada = situacion?.estado === 'cerrada';
  const coordinacion = !!acceso?.perfiles?.includes('coordinacion');
  const todos = !!acceso?.perfiles?.some((p) => ['coordinacion', 'consulta'].includes(p));
  const sucio = !!narrativa && JSON.stringify(narrativa) !== original;
  const ocupado = ['cargando', 'guardando', 'descargando', 'cerrando', 'reabriendo'].includes(estado);
  const avisos = informe?.apartados.filter((a) => Object.values(a.avisos ?? {}).some((n) => n > 0) || a.celdas.some((c) => c.sin_dato > 0 || !!c.porcentaje?.aviso)) ?? [];
  const nombreBloques = t('memoria.bloques');
  const claseError = ['memoria-aviso', 'memoria-tarjeta'].join(' ');
  const numero = (valor: number) => new Intl.NumberFormat(idioma).format(valor);
  const fecha = (valor: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(valor));
  const etiqueta = (apartado: string, dimension: string, valor: string) => etiquetaCifra(t, apartado, dimension, valor);
  const fallo = (e: unknown, signal: AbortSignal) => {
    if (signal.aborted) return;
    ponerError(e instanceof FalloMemoria ? e : new FalloMemoria('estadistica.memoria_no_disponible')); ponerEstado('error');
  };
  const cargar = async (a = anio, e = equipo) => {
    carga.current?.abort(); const c = new AbortController(); carga.current = c;
    accionError.current = 'cargar';
    ponerEstado('cargando'); ponerError(undefined); ponerAviso(''); ponerInforme(undefined); ponerNarrativa(undefined); ponerSituacion(undefined); ponerCerradaEn('');
    try {
      const sesion = acceso ?? await pedir<Acceso>('/api/sesion', { signal: c.signal });
      if (c.signal.aborted) return;
      ponerAcceso(sesion);
      const profesional = sesion.perfiles?.includes('profesional_etf');
      const seleccion = !acceso && profesional && !sesion.perfiles?.includes('coordinacion') ? (sesion.equipo_id ?? '') : sesion.perfiles?.includes('coordinacion') || sesion.perfiles?.includes('consulta') ? e : (sesion.equipo_id ?? '');
      const soloCifras = !!sesion.perfiles?.includes('consulta') && !sesion.perfiles.includes('coordinacion') && (!profesional || seleccion !== sesion.equipo_id);
      ponerVistaConsulta(soloCifras);
      if (soloCifras) { ponerAnio(a); ponerEquipo(seleccion); ponerEstado('consulta'); return; }
      const puedeEditar = !!sesion.perfiles?.includes('coordinacion') || (!!sesion.perfiles?.includes('profesional_etf') && !!seleccion && seleccion === sesion.equipo_id);
      const [datosIniciales, lista, n, estadoMemoria] = await Promise.all([
        pedir<Informe>(`/api/estadistica/memoria?${consulta(a, seleccion)}`, { signal: c.signal }),
        pedir<Equipo[]>('/api/estadistica/memoria/equipos', { signal: c.signal }).then((lista) => { if (!c.signal.aborted) ponerEquipos(lista); return lista; }),
        puedeEditar ? pedir<Narrativas>(`/api/estadistica/memoria/narrativas?${consulta(a, seleccion)}`, { signal: c.signal }) : Promise.resolve(undefined),
        puedeEditar ? pedir<EstadoMemoria>(`/api/estadistica/memoria/estado?${consulta(a, seleccion)}`, { signal: c.signal }) : Promise.resolve(undefined),
      ]);
      let datos = datosIniciales;
      let textosActuales = n;
      if (estadoMemoria?.estado === 'cerrada') {
        const foto = await pedir<Instantanea>(`/api/estadistica/memoria/instantanea?${consulta(a, seleccion)}&revision=${estadoMemoria.revision_actual}`, { signal: c.signal });
        // Cifras y textos proceden de la misma revisión, aunque se reabra después.
        datos = cifrasDeInstantanea(foto);
        textosActuales = foto.informe.Narrativas;
        if (!c.signal.aborted) ponerCerradaEn(foto.cerrada_en);
      }
      if (c.signal.aborted) return;
      ponerSituacion(estadoMemoria);
      ponerAcceso(sesion); ponerEquipo(seleccion); ponerEquipos(lista); ponerInforme(datos); ponerNarrativa(textosActuales); ponerOriginal(JSON.stringify(textosActuales)); ponerEstado('lista'); ponerSucio(false);
    } catch (e) { fallo(e, c.signal); }
  };
  useEffect(() => { void cargar(); return () => { carga.current?.abort(); mutacion.current?.abort(); }; }, []);
  useEffect(() => { document.title = t('memoria.titulo'); }, [t]);
  useEffect(() => { ponerSucio(sucio || ['guardando', 'cerrando', 'reabriendo'].includes(estado)); }, [sucio, estado, ponerSucio]);
  useEffect(() => () => ponerSucio(false), [ponerSucio]);
  useEffect(() => { if (error) incidencia.current?.focus(); }, [error]);
  useEffect(() => {
    if (estado !== 'lista') return;
    const p = new URLSearchParams(location.search);
    if (p.has('apartado') && p.has('celda') && p.has('medida')) requestAnimationFrame(() => document.getElementById(`memoria-cifra-${p.get('apartado')}-${p.get('celda')}-${p.get('medida')}`)?.focus());
  }, [estado]);
  const cambiarFiltro = (a: number, e: string) => solicitarSalida(() => { ponerAnio(a); ponerEquipo(e); void cargar(a, e); });
  const guardar = async () => {
    if (!narrativa || !informe || ocupado || cerrada) return;
    const c = new AbortController(); mutacion.current = c; accionError.current = 'cargar'; ponerEstado('guardando'); ponerError(undefined); ponerAviso('');
    try {
      const n = await pedir<Narrativas>(`/api/estadistica/memoria/narrativas?${consulta(informe.anio, informe.equipo_id)}`, { method: 'PUT', signal: c.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify(narrativa) });
      if (c.signal.aborted) return;
      ponerNarrativa(n); ponerOriginal(JSON.stringify(n)); ponerSituacion((actual) => actual ? { ...actual, version: n.version } : actual); ponerSucio(false); ponerAviso('memoria.guardada'); ponerEstado('lista'); claveWord.current = '';
    } catch (e) { fallo(e, c.signal); }
  };
  const word = async () => {
    if (ocupado || sucio || !informe) return;
    const c = new AbortController(); mutacion.current = c; accionError.current = 'descargar'; ponerEstado('descargando'); ponerError(undefined); ponerAviso('');
    const firma = JSON.stringify({ anio: informe.anio, equipo_id: informe.equipo_id, idioma });
    if (solicitudWord.current !== firma) { claveWord.current = ''; solicitudWord.current = firma; }
    claveWord.current ||= crypto.randomUUID();
    try {
      const archivo = await descargar(informe.anio, informe.equipo_id, idioma, csrf, claveWord.current, c.signal);
      if (c.signal.aborted) return;
      const url = URL.createObjectURL(archivo); const enlace = document.createElement('a'); enlace.href = url; enlace.download = `memoria-${informe.anio}.docx`; enlace.click(); URL.revokeObjectURL(url);
      ponerEstado('lista'); ponerAviso('memoria.descargada'); claveWord.current = ''; solicitudWord.current = '';
    } catch (e) { fallo(e, c.signal); }
  };
  const abrirActo = (a: 'cerrar' | 'reabrir') => {
    if (ocupado || sucio || !situacion || !informe) return;
    ponerAccion(a); ponerMotivo(''); actoPendiente.current = undefined; dialogo.current?.showModal();
  };
  const terminarDialogo = () => { dialogo.current?.close(); botonActo.current?.focus(); };
  const confirmarActo = async () => {
    if (!informe || !situacion || ocupado || sucio) return;
    actoPendiente.current ??= { accion, anio: informe.anio, equipo: informe.equipo_id, version: situacion.version, motivo: motivo.trim(), clave: crypto.randomUUID() };
    const pendiente = actoPendiente.current;
    terminarDialogo(); const c = new AbortController(); mutacion.current = c; accionError.current = 'acto';
    ponerEstado(pendiente.accion === 'cerrar' ? 'cerrando' : 'reabriendo'); ponerError(undefined); ponerAviso('');
    try {
      await ejecutarActo(pendiente, csrf, c.signal);
      if (c.signal.aborted) return;
      actoPendiente.current = undefined; claveWord.current = ''; solicitudWord.current = '';
      await cargar(pendiente.anio, pendiente.equipo);
      if (!c.signal.aborted) ponerAviso(pendiente.accion === 'cerrar' ? 'memoria.cierre_guardado' : 'memoria.reapertura_guardada');
    } catch (e) { fallo(e, c.signal); }
  };
  const general = (campo: 'entidad_local' | 'denominacion' | 'observaciones', valor: string) => { if (narrativa) ponerNarrativa({ ...narrativa, a: { ...narrativa.a, [campo]: valor } }); };
  const personal = (profesion: typeof personas[number], sexo: keyof Personal, valor: string) => { if (narrativa) ponerNarrativa({ ...narrativa, a: { ...narrativa.a, [profesion]: { ...narrativa.a[profesion], [sexo]: valor === '' ? null : Number(valor) } } }); };
  const cifra = (a: Apartado, celda: number, medida: string, valor: number) => {
    const parametros = new URLSearchParams({ anio: String(informe!.anio), ...(informe!.equipo_id ? { etf: informe!.equipo_id } : {}), apartado: a.codigo, celda: String(celda), medida, esperado: String(valor), ...(celda >= 0 ? { clave: JSON.stringify({ Grupo: a.celdas[celda].grupo, Unidad: a.celdas[celda].unidad }) } : {}), ...(cerrada && situacion ? { revision: String(situacion.revision_actual) } : {}) });
    const ruta = `/memoria/desglose?${parametros}`;
    const grupoDesglose = medida === 'denominador' ? t(`memoria.denominador.${a.contrato.denominador.toLowerCase()}`) || t('memoria.denominador') : celda >= 0 ? Object.entries(a.celdas[celda].grupo).filter(([dimension, valor]) => !!valor && !(medida === 'sin_dato' && dimension === 'estado' && valor === 'no')).map(([dimension, valor]) => etiqueta(a.codigo, dimension, valor)).join(t('memoria.separador')) || t('memoria.sin_dato') : t('memoria.desglose.total');
    const unidadDesglose = celda >= 0 ? a.celdas[celda].unidad : medida;
    const etiquetaDesglose = t('memoria.desglose.ver').replace('{cantidad}', numero(valor)).replace('{apartado}', t(`memoria.apartado.${a.codigo}`)).replace('{medida}', t(`memoria.desglose.medida.${medida}`) || t(`memoria.unidad.${medida}`)).replace('{grupo}', grupoDesglose).replace('{unidad}', t(`memoria.unidad.${unidadDesglose}`) || t('memoria.unidad.familias'));
    return <a id={`memoria-cifra-${a.codigo}-${celda}-${medida}`} className="memoria-cifra" href={ruta} aria-label={etiquetaDesglose} onClick={(e) => { if (e.button === 0 && !e.ctrlKey && !e.metaKey && !e.shiftKey && !e.altKey) { e.preventDefault(); solicitarSalida(() => navegarRuta(ruta)); } }}>{numero(valor)}</a>;
  };
  const mostrarApartado = (a: Apartado) => <article className="memoria-apartado" key={a.codigo} tabIndex={-1} id={`memoria-apartado-${a.codigo}`}>
    <h3>{t(`memoria.apartado.${a.codigo}`)}</h3>
    <p>{Object.entries(a.total).filter(([unidad, cantidad]) => cantidad > 0 && unidad !== 'familias_sin_relacion_menores').map(([unidad, cantidad]) => <span key={unidad}>{t(`memoria.unidad.${unidad}`)} {cifra(a, -1, unidad, cantidad)} </span>)}</p>
    <p>{t('memoria.poblacion')} {t(a.codigo === '5' ? 'memoria.poblacion.no_aceptados' : `memoria.poblacion.${a.contrato.poblacion.toLowerCase()}`) || t('memoria.etiqueta_pendiente')}</p>
    <p>{t('memoria.denominador')} {t(`memoria.denominador.${a.contrato.denominador.toLowerCase()}`) || t('memoria.denominador_pendiente')}</p>
    <details><summary>{t('memoria.reglas')}</summary><p>{t('memoria.regla_fecha').replace('{fecha}', fecha(a.contrato.corte)).replace('{version}', a.contrato.version_regla)}</p></details>
    {Object.entries(a.avisos ?? {}).filter(([, cantidad]) => cantidad > 0).map(([clave, cantidad]) => <p className="memoria-aviso" key={clave}>{t(clave) || t('memoria.datos_faltantes')} {numero(cantidad)}</p>)}
    <div className="memoria-tabla" role="region" tabIndex={0} aria-label={t(`memoria.apartado.${a.codigo}`)}>
      <table><caption>{t(`memoria.apartado.${a.codigo}`)}</caption><thead><tr>{['grupo', 'unidad', 'cantidad', 'base', 'porcentaje', 'sin_dato'].map((col) => <th scope="col" key={col}>{t(`memoria.${col}`)}</th>)}</tr></thead>
        <tbody>{a.celdas.map((c, i) => <tr key={i}><th scope="row">{[...Object.entries(c.grupo).filter(([dimension, valor]) => !!valor && !(c.sin_dato > 0 && c.numerador === 0 && dimension === 'estado' && valor === 'no')).map(([dimension, valor]) => etiqueta(a.codigo, dimension, valor)), ...(c.sin_dato > 0 && c.numerador === 0 ? [t('memoria.sin_dato')] : [])].join(t('memoria.separador')) || t('memoria.sin_dato')}</th><td>{t(`memoria.unidad.${c.unidad}`)}</td><td>{cifra(a, i, 'numerador', c.numerador)}</td><td>{c.denominador.conocido ? cifra(a, i, 'denominador', c.denominador.valor) : t('memoria.sin_dato')}</td><td>{c.porcentaje && !['estadistica.denominador_pendiente'].includes(c.porcentaje.aviso) ? new Intl.NumberFormat(idioma, { style: 'percent', minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(c.porcentaje.centesimas / 10000) : t('memoria.sin_dato')}{c.porcentaje?.aviso && <span className="memoria-aviso">{t(c.porcentaje.aviso)}</span>}</td><td className={c.sin_dato ? 'memoria-aviso' : undefined}>{cifra(a, i, 'sin_dato', c.sin_dato)}</td></tr>)}</tbody>
      </table>
    </div>
    {!a.celdas.length && <p>{t('memoria.sin_poblacion')}</p>}
    <p>{t('memoria.relacion_desconocida')} {cifra(a, -1, 'familias_sin_relacion_menores', a.total.familias_sin_relacion_menores ?? 0)}</p>
  </article>;
  if (vistaConsulta) return <ConsultaMemoria t={t} idioma={idioma} csrf={csrf} anioInicial={anio} equipoInicial={equipo} equipoPropio={acceso?.perfiles?.includes('profesional_etf') ? acceso.equipo_id : undefined} alConsultarEquipoPropio={cambiarFiltro} />;
  return <section className="memoria" aria-labelledby="memoria-titulo">
    <div className="memoria-cabecera"><h1 id="memoria-titulo">{t('memoria.titulo')}</h1><Boton secundario onPress={() => ponerAyuda(!ayuda)}>{t('memoria.ayuda')}</Boton>{edita && <Boton disabled={ocupado || sucio || !informe} onPress={() => void word()}>{t('memoria.descargar')}</Boton>}</div>
    {ayuda && <p className="memoria-tarjeta">{t('memoria.ayuda_texto')}</p>}
    <form className="memoria-filtros" onSubmit={(e) => { e.preventDefault(); cambiarFiltro(anio, equipo); }}>
      <div><label htmlFor="memoria-anio">{t('memoria.anio')}</label><input id="memoria-anio" type="number" min={1} max={9998} required value={anio} disabled={ocupado} onChange={(e) => ponerAnio(Number(e.target.value))} /></div>
      {todos && <div><label htmlFor="memoria-equipo">{t('memoria.equipo')}</label><select id="memoria-equipo" value={equipo} disabled={ocupado} onChange={(e) => ponerEquipo(e.target.value)}><option value="">{t('memoria.provincial')}</option>{equipos.map((e) => <option value={e.id} key={e.id}>{t('memoria.equipo_opcion').replace('{codigo}', e.codigo)}</option>)}</select></div>}
      <button className="boton-secundario" type="submit" disabled={ocupado}>{t('memoria.consultar')}</button>
    </form>
    <p role="status">{ocupado ? t(`memoria.${estado}`) : aviso ? t(aviso) : ''}</p>
    {error && <div className={claseError} id="memoria-error" role="alert" ref={incidencia} tabIndex={-1}><p>{t(error.clave) || t('estadistica.memoria_no_disponible')}</p><Boton secundario disabled={ocupado} onPress={() => accionError.current === 'descargar' ? void word() : accionError.current === 'acto' && !error.conflicto ? void confirmarActo() : solicitarSalida(() => void cargar())}>{t(error.conflicto ? 'memoria.recuperar' : 'memoria.reintentar')}</Boton></div>}
    {informe && <>
      <p>{t('memoria.ambito').replace('{anio}', new Intl.NumberFormat(idioma, { useGrouping: false }).format(informe.anio)).replace('{equipo}', informe.equipo_codigo ? t('memoria.equipo_opcion').replace('{codigo}', informe.equipo_codigo) : t('memoria.provincial'))}</p>
      <p>{t('memoria.calculada_en').replace('{fecha}', fecha(informe.conocido_en))}</p>
      <p className="pildora">{t(cerrada ? 'memoria.estado_cerrada' : edita ? 'memoria.borrador' : 'memoria.cifras_publicadas')}</p>
      {cerradaEn && <p>{t('memoria.cerrada_fecha').replace('{fecha}', fecha(cerradaEn))}</p>}
      {edita && situacion && (!cerrada || coordinacion) && <button className="boton-secundario" ref={botonActo} disabled={ocupado || sucio} onClick={() => abrirActo(cerrada ? 'reabrir' : 'cerrar')}>{t(cerrada ? 'memoria.reabrir' : 'memoria.cerrar')}</button>}
      {cerrada && <p>{t('memoria.foto_conservada')}</p>}<p>{t('memoria.corte').replace('{fecha}', fecha(informe.corte))}</p>
      {informe.apartados.every((a) => a.celdas.length === 0) && <p>{t('memoria.vacio')}</p>}
      {avisos.length > 0 && <aside className="memoria-aviso" aria-labelledby="memoria-avisos"><h2 id="memoria-avisos">{t('memoria.revisar_avisos')}</h2><div className="memoria-bloques">{limites.map(([desde, hasta], i) => { const pendientes = avisos.filter((a) => Number(a.codigo) >= desde && Number(a.codigo) < hasta + 1); return pendientes.length > 0 && <Boton secundario key={i} onPress={() => { ponerBloque(i); requestAnimationFrame(() => { const elemento = document.getElementById(`memoria-apartado-${pendientes[0].codigo}`); elemento?.focus(); elemento?.scrollIntoView(); }); }}>{t('memoria.avisos_bloque').replace('{bloque}', t(`memoria.bloque.${i}`)).replace('{cantidad}', numero(pendientes.length))}</Boton>; })}</div></aside>}
      <nav className="memoria-bloques" aria-label={nombreBloques}>{limites.map((_, i) => <Boton secundario key={i} disabled={bloque === i} onPress={() => ponerBloque(i)}>{t(`memoria.bloque.${i}`)}</Boton>)}</nav>
      <h2>{t(`memoria.bloque.${bloque}`)}</h2>
      {informe.apartados.filter((a) => Number(a.codigo) >= limites[bloque][0] && Number(a.codigo) < limites[bloque][1] + 1).sort((a, b) => Number(a.codigo) - Number(b.codigo)).map(mostrarApartado)}
    </>}
    {edita && narrativa && <form className="memoria-tarjeta" onSubmit={(e) => { e.preventDefault(); void guardar(); }}>
      <h2>{t('memoria.textos')}</h2><p id="memoria-privacidad">{t('memoria.privacidad')}</p>
      <fieldset disabled={ocupado || cerrada} aria-describedby={error ? 'memoria-privacidad memoria-error' : 'memoria-privacidad'}><legend>{t('memoria.generales')}</legend>
        {(['entidad_local', 'denominacion'] as const).map((campo) => <div key={campo}><label htmlFor={`memoria-${campo}`}>{t(`memoria.${campo}`)}</label><input id={`memoria-${campo}`} value={narrativa.a[campo]} maxLength={1000} onChange={(e) => general(campo, e.target.value)} /></div>)}
        <label htmlFor="memoria-equipos-convenio">{t('memoria.equipos_convenio')}</label><input id="memoria-equipos-convenio" type="number" min={0} max={1000} value={narrativa.a.equipos_convenio ?? ''} onChange={(e) => ponerNarrativa({ ...narrativa, a: { ...narrativa.a, equipos_convenio: e.target.value === '' ? null : Number(e.target.value) } })} />
        {personas.map((p) => <fieldset key={p}><legend>{t(`memoria.${p}`)}</legend><div className="memoria-personal">{sexos.map((s) => <div key={s}><label htmlFor={`memoria-${p}-${s}`}>{t(`memoria.${s}`)}</label><input id={`memoria-${p}-${s}`} type="number" min={0} max={10000} value={narrativa.a[p][s] ?? ''} onChange={(e) => personal(p, s, e.target.value)} /></div>)}</div></fieldset>)}
        <label htmlFor="memoria-observaciones">{t('memoria.observaciones')}</label><textarea id="memoria-observaciones" value={narrativa.a.observaciones} onChange={(e) => general('observaciones', e.target.value)} />
      </fieldset>
      {textos.map((campo) => <div key={campo}><label htmlFor={`memoria-texto-${campo}`}>{t(`memoria.texto.${campo}`)}</label><p id={`memoria-limite-${campo}`}>{t(campo === 'g' ? 'memoria.limite30' : 'memoria.limite25')}</p><textarea id={`memoria-texto-${campo}`} disabled={ocupado || cerrada} aria-describedby={error ? `memoria-limite-${campo} memoria-error` : `memoria-limite-${campo}`} maxLength={campo === 'g' ? 24000 : 20000} onInput={(e) => e.currentTarget.setCustomValidity(e.currentTarget.value.replaceAll('\r\n', '\n').replaceAll('\r', '\n').split('\n').length > (campo === 'g' ? 30 : 25) ? t(campo === 'g' ? 'memoria.limite30' : 'memoria.limite25') : '')} value={narrativa[campo]} onChange={(e) => ponerNarrativa({ ...narrativa, [campo]: e.target.value })} /></div>)}
      {sucio && <p className="memoria-aviso">{t('memoria.sin_guardar')}</p>}
      <button className="boton-secundario" type="submit" disabled={ocupado || cerrada || !sucio || !!error?.conflicto}>{t('memoria.guardar')}</button>
    </form>}
    <dialog ref={dialogo} aria-labelledby="memoria-confirmacion" onClose={() => botonActo.current?.focus()}>
      <form onSubmit={(e) => { e.preventDefault(); void confirmarActo(); }}>
        <h2 id="memoria-confirmacion">{t(accion === 'cerrar' ? 'memoria.confirmar_cierre' : 'memoria.confirmar_reapertura')}</h2>
        <p>{t(accion === 'cerrar' ? 'memoria.consecuencia_cierre' : 'memoria.consecuencia_reapertura')}</p>
        {accion === 'reabrir' && <><label htmlFor="memoria-motivo">{t('memoria.motivo_reapertura')}</label><textarea id="memoria-motivo" value={motivo} required maxLength={4000} onChange={(e) => ponerMotivo(e.target.value)} /></>}
        <div className="memoria-cabecera"><Boton secundario onPress={terminarDialogo}>{t('memoria.cancelar')}</Boton><button type="submit" disabled={ocupado || (accion === 'reabrir' && !motivo.trim())}>{t(accion === 'cerrar' ? 'memoria.confirmar_cierre' : 'memoria.confirmar_reapertura')}</button></div>
      </form>
    </dialog>
  </section>;
}
