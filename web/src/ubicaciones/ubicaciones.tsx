import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import type { FormEvent, ReactNode } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useCambios } from '../cambios';
import '../administracion.css';
import './ubicaciones.css';

type Tipo = 'equipo' | 'centro' | 'municipio';
type Entrada = { id: string; codigo?: string; ine?: string; clave_texto: string; activo_desde?: string; activo_hasta?: string | null; centro_ss_id?: string | null; equipo_id?: string | null; ambito_diputacion?: boolean };
type Estado = { datos: { version: { numero: number; vigente_desde: string }; equipos: Entrada[]; centros: Entrada[]; municipios: Entrada[] }; referencias: Record<string, number>; textos: Record<string, Record<string, string>> };
type Campos = Record<string, string>;
const api = '/api/admin/ubicaciones';
const civil = (fecha: string) => `${fecha}T00:00:00Z`;
const fecha = (valor?: string | null) => valor?.slice(0, 10) ?? '';
const siguiente = (valor: string) => new Date(new Date(valor).getTime() + 86400000).toISOString().slice(0, 10);
function Boton({ children, onPress, disabled = false, type = 'button', secundario = false, etiqueta }: { children: ReactNode; onPress?: () => void; disabled?: boolean; type?: 'button' | 'submit'; secundario?: boolean; etiqueta?: string }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress, isDisabled: disabled, type }, ref);
  return <button {...buttonProps} ref={ref} className={secundario ? 'boton-secundario' : undefined} aria-label={etiqueta}>{children}</button>;
}
export function Ubicaciones({ t, idioma, csrf }: PropsPantalla) {
  const { ponerSucio } = useCambios();
  const [estado, ponerEstado] = useState<Estado>();
  const [tipo, ponerTipo] = useState<Tipo>('municipio');
  const [busqueda, ponerBusqueda] = useState('');
  const [campos, ponerCampos] = useState<Campos>();
  const [cargando, ponerCargando] = useState(true);
  const [guardando, ponerGuardando] = useState(false);
  const [error, ponerError] = useState('');
  const [conflicto, ponerConflicto] = useState(false);
  const [revisando, ponerRevisando] = useState(false);
  const [revisionAceptada, ponerRevisionAceptada] = useState(false);
  const titulo = useRef<HTMLHeadingElement>(null);
  const [guardado, ponerGuardado] = useState(false);
  const [intento, ponerIntento] = useState(0);
  const peticion = useRef<AbortController | undefined>(undefined);
  const tituloFormulario = useRef<HTMLHeadingElement>(null);
  const ayuda = useRef<HTMLDetailsElement>(null);
  const ayudaBoton = useRef<HTMLElement>(null);
  const claveError = (c: { error?: string | { clave?: string } }) => typeof c.error === 'string' ? c.error : c.error?.clave ?? 'catalogo.no_disponible';
  useEffect(() => { document.title = t('ubicaciones.titulo'); }, [t]);
  useEffect(() => {
    const controlador = new AbortController(); ponerCargando(true); ponerError('');
    void fetch(api, { signal: controlador.signal }).then(async (r) => {
      const c = await r.json() as Estado & { error?: string | { clave?: string } };
      if (!r.ok) throw new Error(claveError(c));
      if (!c.datos || !c.textos || !c.referencias) throw new Error('catalogo.no_disponible');
      if (!controlador.signal.aborted) { ponerEstado(c); ponerConflicto(false); }
    }).catch((e: unknown) => { if (!controlador.signal.aborted) ponerError(e instanceof Error && e.message.startsWith('catalogo.') ? e.message : 'catalogo.no_disponible'); })
      .finally(() => { if (!controlador.signal.aborted) ponerCargando(false); });
    return () => controlador.abort();
  }, [intento]);
  useEffect(() => () => { peticion.current?.abort(); ponerSucio(false); }, [ponerSucio]);
  useEffect(() => { if (campos) tituloFormulario.current?.focus(); }, [!!campos]);
  const texto = (clave: string, lengua = idioma) => estado?.textos[lengua]?.[clave] ?? estado?.textos.es?.[clave] ?? t('ubicaciones.sin_denominacion');
  const filas = estado ? tipo === 'equipo' ? estado.datos.equipos : tipo === 'centro' ? estado.datos.centros : estado.datos.municipios : [];
  const fechaVista = (valor?: string | null) => valor ? new Intl.DateTimeFormat(idioma, { timeZone: 'UTC', day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(valor)) : t('ubicaciones.sin_baja');
  const nombreAsignado = (id: string | null | undefined, entradas: Entrada[]) => { const entrada = entradas.find((e) => e.id === id); return entrada ? texto(entrada.clave_texto) : t('ubicaciones.sin_asignar'); };
  const editar = (fila?: Entrada) => {
    if (!estado) return;
    const grafias = (lengua: string) => Object.entries(estado.textos[lengua] ?? {}).filter(([k]) => k.startsWith(`${fila?.clave_texto}.variante.`)).map(([, v]) => v).join('\n');
    ponerCampos({ id: fila?.id ?? '', codigo: fila?.codigo ?? fila?.ine ?? (tipo === 'centro' ? `C_${crypto.randomUUID().replaceAll('-', '').toUpperCase()}` : (estado.datos.equipos.length < 12 ? Array.from({ length: 12 }, (_, i) => String(i + 1).padStart(2, '0')).find((codigo) => !estado.datos.equipos.some((e) => e.codigo === codigo)) ?? '' : '')), texto_es: fila ? texto(fila.clave_texto, 'es') : '', texto_en: fila ? texto(fila.clave_texto, 'en') : '', activo_desde: fecha(fila?.activo_desde), activo_hasta: fecha(fila?.activo_hasta), vigente_desde: siguiente(estado.datos.version.vigente_desde), centro_ss_id: fila?.centro_ss_id ?? '', equipo_id: fila?.equipo_id ?? '', ambito: String(fila?.ambito_diputacion ?? true), variantes_es: grafias('es'), variantes_en: grafias('en') });
    ponerError(''); ponerGuardado(false); ponerConflicto(false); ponerRevisando(false); ponerRevisionAceptada(false);
  };
  const cambiar = (clave: string, valor: string) => { ponerCampos((c) => ({ ...c, [clave]: valor })); ponerSucio(true); };
  const cancelar = () => { ponerCampos(undefined); ponerSucio(false); ponerError(''); titulo.current?.focus(); };
  async function guardar(e: FormEvent) {
    e.preventDefault(); if (!estado || !campos || guardando || cargando || revisando && !revisionAceptada) return;
    const controlador = new AbortController(); peticion.current = controlador;
    ponerGuardando(true); ponerError(''); ponerConflicto(false); ponerGuardado(false);
    const variantes = (valor: string) => valor.split('\n').map((v) => v.trim()).filter(Boolean);
    const ubicacion = { tipo, id: campos.id, codigo: campos.codigo, ...(tipo === 'municipio' ? { centro_ss_id: campos.centro_ss_id || null, equipo_id: campos.equipo_id || null, ambito_diputacion: campos.ambito === 'true', variantes: { es: variantes(campos.variantes_es), en: variantes(campos.variantes_en) } } : { activo_desde: civil(campos.activo_desde), activo_hasta: campos.activo_hasta ? civil(campos.activo_hasta) : null }) };
    try {
      const r = await fetch(api, { method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, signal: controlador.signal, body: JSON.stringify({ version_esperada: estado.datos.version.numero, vigente_desde: civil(campos.vigente_desde), referencias_esperadas: estado.referencias, textos: { es: campos.texto_es.trim(), en: campos.texto_en.trim() }, ubicacion }) });
      const c = await r.json() as Estado & { error?: string | { clave?: string } };
      if (!r.ok) { if (r.status === 409) { ponerConflicto(true); ponerRevisando(true); ponerRevisionAceptada(false); } throw new Error(claveError(c)); }
      if (!c.datos || !c.textos || !c.referencias) throw new Error('catalogo.no_disponible');
      if (!controlador.signal.aborted) { ponerEstado(c); ponerCampos(undefined); ponerSucio(false); ponerGuardado(true); titulo.current?.focus(); }
    } catch (e: unknown) { if (!controlador.signal.aborted) ponerError(e instanceof Error && e.message.startsWith('catalogo.') ? e.message : 'catalogo.no_disponible'); }
    finally { if (!controlador.signal.aborted) ponerGuardando(false); }
  }
  const campo = (clave: string, tipoCampo = 'text', requerido = true) => <div className="campo-admin"><label htmlFor={`ubicaciones-${clave}`}>{t(`ubicaciones.${clave}`)}</label><input id={`ubicaciones-${clave}`} type={tipoCampo} required={requerido} value={campos?.[clave] ?? ''} maxLength={clave.startsWith('texto_') ? 2000 : 64} readOnly={clave === 'codigo' && !!campos?.id} min={tipoCampo === 'date' ? clave === 'vigente_desde' && estado ? siguiente(estado.datos.version.vigente_desde) : clave === 'activo_hasta' && campos?.activo_desde ? siguiente(campos.activo_desde) : undefined : undefined} disabled={guardando} aria-describedby={error ? 'ubi-error' : undefined} onChange={(e) => cambiar(clave, e.target.value)} /></div>;
  const selector = (clave: string, entradas: Entrada[]) => <div className="campo-admin"><label htmlFor={`ubicaciones-${clave}`}>{t(`ubicaciones.${clave}`)}</label><select id={`ubicaciones-${clave}`} disabled={guardando} value={campos?.[clave] ?? ''} aria-describedby={error ? 'ubi-error' : undefined} onChange={(e) => cambiar(clave, e.target.value)}><option value="">{t('ubicaciones.sin_asignar')}</option>{entradas.map((e) => <option key={e.id} value={e.id}>{texto(e.clave_texto)}</option>)}</select><output>{nombreAsignado(campos?.[clave], entradas)}</output></div>;
  const nombreAyuda = t('ubicaciones.ayuda_etiqueta');
  return <section className={["contenido-admin","ubicaciones"].join(" ")} aria-labelledby={"ubicaciones-titulo"}>
    <div className="ubicaciones-cabecera"><h1 id="ubicaciones-titulo" ref={titulo} tabIndex={-1}>{t('ubicaciones.titulo')}</h1><details ref={ayuda} onKeyDown={(e) => { if (e.key === 'Escape' && ayuda.current) { ayuda.current.open = false; ayudaBoton.current?.focus(); } }}><summary ref={ayudaBoton} aria-label={nombreAyuda}>{t('ubicaciones.ayuda_boton')}</summary><p>{t('ubicaciones.ayuda')}</p></details></div>
    <p>{t('ubicaciones.consecuencia')}</p>
    {estado && <p className="ubicaciones-contexto">{t('ubicaciones.version').replace('{version}', new Intl.NumberFormat(idioma).format(estado.datos.version.numero)).replace('{fecha}', fechaVista(estado.datos.version.vigente_desde))}</p>}
    {cargando && <p role="status">{t('ubicaciones.cargando')}</p>}
    {guardado && <p role="status">{t('ubicaciones.guardado')}</p>}
    {error && <div id="ubi-error" role="alert"><p>{t(error)}</p>{conflicto && <p>{t('ubicaciones.conflicto')}</p>}<Boton secundario disabled={guardando} onPress={() => { ponerIntento((i) => i + 1); }}>{t('ubicaciones.actualizar')}</Boton></div>}
    {!cargando && estado && !campos && <>
      <div className="acciones-admin"><label htmlFor="ubicaciones-tipo">{t('ubicaciones.tipo')}</label><select id="ubicaciones-tipo" value={tipo} onChange={(e) => { ponerTipo(e.target.value as Tipo); ponerBusqueda(''); ponerGuardado(false); }}>{(['municipio', 'equipo', 'centro'] as Tipo[]).map((v) => <option key={v} value={v}>{t(`ubicaciones.tipo_${v}`)}</option>)}</select>{tipo !== 'municipio' && (tipo !== 'equipo' || estado.datos.equipos.length < 12) && <Boton onPress={() => editar()}>{t(`ubicaciones.alta_${tipo}`)}</Boton>}</div>
      <div className="campo-admin"><label htmlFor="ubicaciones-buscar">{t('ubicaciones.buscar')}</label><input id="ubicaciones-buscar" type="search" value={busqueda} onChange={(e) => ponerBusqueda(e.target.value)} /></div>
      {filas.filter((f) => texto(f.clave_texto).toLocaleLowerCase(idioma).includes(busqueda.toLocaleLowerCase(idioma))).length === 0 && <p role="status">{t('ubicaciones.vacio')}</p>}
      <ul className="lista-admin">{filas.filter((f) => texto(f.clave_texto).toLocaleLowerCase(idioma).includes(busqueda.toLocaleLowerCase(idioma))).map((f) => <li key={f.id} className="fila-admin"><div><h2>{texto(f.clave_texto)}</h2>{tipo === 'municipio' ? <><p>{t('ubicaciones.centro_ss_id')}{t('ubicaciones.separador')}{nombreAsignado(f.centro_ss_id, estado.datos.centros)}</p><p>{t('ubicaciones.equipo_id')}{t('ubicaciones.separador')}{nombreAsignado(f.equipo_id, estado.datos.equipos)}</p><p>{t(f.ambito_diputacion ? 'ubicaciones.dentro' : 'ubicaciones.fuera')}</p></> : <p>{t('ubicaciones.activo_desde')}{t('ubicaciones.separador')}{fechaVista(f.activo_desde)}{t('ubicaciones.separador_fecha')}{t('ubicaciones.activo_hasta')}{t('ubicaciones.separador')}{fechaVista(f.activo_hasta)}</p>}</div><Boton secundario etiqueta={t('ubicaciones.editar') + ' ' + texto(f.clave_texto)} onPress={() => editar(f)}>{t('ubicaciones.editar')}</Boton></li>)}</ul>
    </>}
    {campos && estado && <form className="formulario-admin" onSubmit={(e) => void guardar(e)}><h2 ref={tituloFormulario} tabIndex={-1}>{campos.id ? texto(`${tipo === 'municipio' ? 'municipio' : tipo}.${campos.codigo}`) : t(`ubicaciones.alta_${tipo}`)}</h2>
      {revisando && <section className="fila-admin" aria-labelledby="ubicaciones-actual"><div><h3 id="ubicaciones-actual">{t('ubicaciones.valores_actuales')}</h3>{(() => { const actual = filas.find((f) => f.id === campos.id); return actual ? <><p>{t('ubicaciones.texto_es')}{t('ubicaciones.separador')}{texto(actual.clave_texto, 'es')}</p><p>{t('ubicaciones.texto_en')}{t('ubicaciones.separador')}{texto(actual.clave_texto, 'en')}</p>{tipo === 'municipio' ? <><p>{t('ubicaciones.centro_ss_id')}{t('ubicaciones.separador')}{nombreAsignado(actual.centro_ss_id, estado.datos.centros)}</p><p>{t('ubicaciones.equipo_id')}{t('ubicaciones.separador')}{nombreAsignado(actual.equipo_id, estado.datos.equipos)}</p><p>{t(actual.ambito_diputacion ? 'ubicaciones.dentro' : 'ubicaciones.fuera')}</p>{(['es','en'] as const).map((v) => <p key={v}>{t(`ubicaciones.variantes_${v}`)}{t('ubicaciones.separador')}{Object.entries(estado.textos[v] ?? {}).filter(([k]) => k.startsWith(`${actual.clave_texto}.variante.`)).map(([,texto]) => texto).join('; ') || t('ubicaciones.sin_variantes')}</p>)}</> : <p>{t('ubicaciones.activo_desde')}{t('ubicaciones.separador')}{fechaVista(actual.activo_desde)}{t('ubicaciones.separador_fecha')}{t('ubicaciones.activo_hasta')}{t('ubicaciones.separador')}{fechaVista(actual.activo_hasta)}</p>}</> : <p>{t('ubicaciones.entrada_nueva')}</p>; })()}<label><input type="checkbox" checked={revisionAceptada} disabled={cargando || conflicto} onChange={(e) => ponerRevisionAceptada(e.target.checked)} />{t('ubicaciones.revisado')}</label></div></section>}
      <fieldset disabled={guardando}><legend>{t('ubicaciones.denominacion')}</legend>{!campos.id && tipo === 'equipo' && <div className="campo-admin"><label htmlFor="ubicaciones-codigo">{t('ubicaciones.numero_equipo')}</label><select id="ubicaciones-codigo" value={campos.codigo} onChange={(e) => cambiar('codigo', e.target.value)}>{Array.from({ length: 12 }, (_, i) => String(i + 1).padStart(2, '0')).filter((codigo) => !estado.datos.equipos.some((e) => e.codigo === codigo)).map((codigo) => <option key={codigo} value={codigo}>{t('ubicaciones.numero').replace('{numero}', new Intl.NumberFormat(idioma).format(Number(codigo)))}</option>)}</select></div>}{campo('texto_es')}{campo('texto_en')}</fieldset>
      {tipo === 'municipio' ? <fieldset disabled={guardando}><legend>{t('ubicaciones.asignacion')}</legend>{selector('centro_ss_id', estado.datos.centros)}{selector('equipo_id', estado.datos.equipos)}<div className="campo-admin"><label htmlFor="ubicaciones-ambito">{t('ubicaciones.ambito')}</label><select id="ubicaciones-ambito" value={campos.ambito} onChange={(e) => cambiar('ambito', e.target.value)}><option value="true">{t('ubicaciones.dentro')}</option><option value="false">{t('ubicaciones.fuera')}</option></select></div>{(['es', 'en'] as const).map((v) => <div className="campo-admin" key={v}><label htmlFor={`ubicaciones-variantes_${v}`}>{t(`ubicaciones.variantes_${v}`)}</label><textarea id={`ubicaciones-variantes_${v}`} rows={4} maxLength={6030} aria-describedby={"ubicaciones-variantes-ayuda"} value={campos[`variantes_${v}`]} onChange={(e) => cambiar(`variantes_${v}`, e.target.value)} /></div>)}<p id="ubicaciones-variantes-ayuda">{t('ubicaciones.una_por_linea')}</p></fieldset> : <fieldset disabled={guardando}><legend>{t('ubicaciones.vigencia')}</legend>{campo('activo_desde', 'date')}{campo('activo_hasta', 'date', false)}</fieldset>}
      <fieldset disabled={guardando}><legend>{t('ubicaciones.publicacion')}</legend>{campo('vigente_desde', 'date')}<p>{t('ubicaciones.publicacion_consecuencia')}</p></fieldset>
      <div className="acciones-admin"><Boton secundario disabled={guardando} onPress={cancelar}>{t('ubicaciones.cancelar')}</Boton><Boton type="submit" disabled={guardando || cargando || revisando && !revisionAceptada}>{t(guardando ? 'ubicaciones.guardando' : 'ubicaciones.publicar')}</Boton></div>
    </form>}
  </section>;
}
