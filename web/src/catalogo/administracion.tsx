import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import type { FormEvent, ReactNode } from 'react';
import '../administracion.css';
import './presentacion.css';
import { useCambios } from '../cambios';

type Opcion = { lista: string; codigo: string; orden: number; admite_texto: boolean; activa: boolean };
type Plazo = { clave: string; valor: number; unidad: string };
type Tramo = { tipo: string; orden: number; desde: number; hasta: number | null; unidad: string };
type Firmante = { cargo_textos: Record<string, string>; id: string; tipo: string; rol: string; nombre: string; cargo_codigo: string; organismo_codigo: string; direccion: string; vigente_desde: string; vigente_hasta: string | null; version: number };
type Version = { numero: number; vigente_desde: string };
type Estado = { datos: { version: Version; opciones: Opcion[]; plazos: Plazo[]; tramos: Tramo[] }; firmantes: Firmante[]; referencias: Record<string, number>; textos: Record<string, Record<string, string>> };
type Fallo = { error?: string | { clave?: string }; version_actual?: number };
const claveError = (cuerpo: Fallo) => typeof cuerpo.error === 'string' ? cuerpo.error : cuerpo.error?.clave ?? 'catalogo.no_disponible';
type Campos = Record<string, string>;
type Ambito = 'listas' | 'plazos' | 'tramos' | 'firmantes';
type Props = { t: (clave: string) => string; idioma: string; csrf: string; navegarRuta: (ruta: string) => void };
const rutas = { listas: '/admin/catalogos/listas', plazos: '/admin/reglas', firmantes: '/admin/firmantes' };
const api = (ambito: Ambito) => ambito === 'firmantes' ? '/api/admin/firmantes' : ambito === 'listas' ? '/api/admin/catalogos/listas' : `/api/admin/reglas?ambito=${ambito}`;
const ambitoInicial = (): Ambito => location.pathname === rutas.firmantes ? 'firmantes' : location.pathname === rutas.plazos ? new URLSearchParams(location.search).get('tipo') === 'tramos' ? 'tramos' : 'plazos' : 'listas';
const fecha = (valor: string | null | undefined) => valor?.slice(0, 10) ?? '';
const civil = (valor: string) => `${valor}T00:00:00Z`;

export function Boton({ children, onPress, disabled = false, type = 'button', secundario = false, etiqueta, actual }: { children: ReactNode; onPress?: () => void; disabled?: boolean; type?: 'button' | 'submit'; secundario?: boolean; etiqueta?: string; actual?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress, isDisabled: disabled, type }, ref);
  return <button {...buttonProps} ref={ref} className={secundario ? 'boton-secundario' : undefined} aria-label={etiqueta} aria-current={actual ? 'page' : undefined}>{children}</button>;
}

export function Administracion({ t, idioma, csrf, navegarRuta }: Props) {
  const { ponerSucio } = useCambios();
  const [ambito] = useState<Ambito>(ambitoInicial);
  const [datos, ponerDatos] = useState<Estado>();
  const [campos, ponerCampos] = useState<Campos>({});
  const [editando, ponerEditando] = useState(false);
  const [modificado, ponerModificadoEstado] = useState(false);
  const ponerModificado = (valor: boolean) => { ponerModificadoEstado(valor); ponerSucio(valor); };
  const [cargando, ponerCargando] = useState(false);
  const [guardando, ponerGuardando] = useState(false);
  const [error, ponerError] = useState('');
  const [conflicto, ponerConflicto] = useState<number>();
  const [guardado, ponerGuardado] = useState(false);
  const [intento, reintentar] = useState(0);
  const peticion = useRef<AbortController | null>(null);
  const formulario = useRef<HTMLFormElement>(null);
  const titulo = t(`catalogo.pantalla_${ambito}`);

  useEffect(() => {
    document.title = titulo || t('comun.titulo');
  }, [titulo, t]);
  useEffect(() => {
    const controlador = new AbortController();
    ponerCargando(true); ponerError(''); ponerDatos(undefined);
    void fetch(api(ambito), { signal: controlador.signal }).then(async (r) => {
      const cuerpo = await r.json() as Estado & Fallo;
      if (!r.ok) throw new Error(claveError(cuerpo));
      if (!cuerpo.referencias || !cuerpo.textos || !cuerpo.datos) throw new Error('catalogo.no_disponible');
      if (!controlador.signal.aborted) ponerDatos(cuerpo);
    }).catch((e: unknown) => { if (!controlador.signal.aborted) ponerError(e instanceof Error ? e.message : 'catalogo.no_disponible'); })
      .finally(() => { if (!controlador.signal.aborted) ponerCargando(false); });
    return () => controlador.abort();
  }, [ambito, intento]);
  useEffect(() => () => peticion.current?.abort(), []);
  useEffect(() => {
    const advertir = (e: BeforeUnloadEvent) => { if (modificado) e.preventDefault(); };
    window.addEventListener('beforeunload', advertir);
    return () => window.removeEventListener('beforeunload', advertir);
  }, [modificado]);
  useEffect(() => {
    if (editando) Array.from(formulario.current?.querySelectorAll<HTMLInputElement>('input') ?? []).find((entrada) => !entrada.closest('details:not([open])'))?.focus();
  }, [editando]);
  useEffect(() => () => ponerSucio(false), [ponerSucio]);
  const puedeSalir = () => !modificado || window.confirm(t('catalogo.cambios_sin_guardar'));
  const navegar = (nuevo: Ambito) => {
    if (guardando) return;
    navegarRuta(nuevo === 'tramos' ? `${rutas.plazos}?tipo=tramos` : rutas[nuevo]);
  };
  const etiqueta = (clave: string) => datos?.textos[idioma]?.[clave] ?? datos?.textos[document.documentElement.dataset.idiomaReserva ?? 'es']?.[clave] ?? t('catalogo.etiqueta_pendiente');
  const texto = (codigo: string, clave: string) => datos?.textos[codigo]?.[clave] ?? '';
  const seleccionar = (fila?: Opcion | Plazo | Tramo | Firmante) => {
    if (!puedeSalir()) return;
    const c: Campos = { vigente_desde: '', vigente_hasta: '', texto_es: '', texto_en: '', tipo: 'firmante', activa: 'true', admite_texto: 'false', orden: '0', unidad: ambito === 'plazos' ? 'dias_naturales' : 'meses' };
    if (fila) {
      c._edicion = 'true';
      if ('desde' in fila) c.tipo_tramo = fila.tipo;
      if ('vigente_desde' in fila) c._desde = fila.vigente_desde;
      for (const [k, v] of Object.entries(fila)) c[k] = v == null ? '' : String(v);
      c.vigente_desde = ''; c.vigente_hasta = '';
      const clave = 'lista' in fila ? `${fila.lista}.${fila.codigo}` : 'cargo_codigo' in fila ? `cargo.${fila.cargo_codigo}` : '';
      c.texto_es = texto('es', clave); c.texto_en = texto('en', clave);
    }
    ponerCampos(c); ponerEditando(true); ponerModificado(false); ponerGuardado(false); ponerError(''); ponerConflicto(undefined);
  };
  const editar = (clave: string, valor: string) => { ponerCampos((c) => ({ ...c, [clave]: valor })); ponerModificado(true); };
  const formatearFecha = (valor: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: 'UTC' }).format(new Date(valor));
  const numero = (valor: number) => new Intl.NumberFormat(idioma).format(valor);
  const vigencia = (version: number, desde: string) => t('catalogo.version_vigente').replace('{version}', numero(version)).replace('{fecha}', formatearFecha(desde));
  async function guardar(e: FormEvent) {
    e.preventDefault(); if (!datos || guardando) return;
    const controlador = new AbortController(); peticion.current = controlador;
    ponerGuardando(true); ponerError(''); ponerGuardado(false); ponerConflicto(undefined);
    const cambio: Record<string, unknown> = { version_esperada: ambito === 'firmantes' ? Number(campos.version ?? 0) : datos.datos.version.numero, referencias_esperadas: datos.referencias, vigente_desde: civil(campos.vigente_desde), textos: { es: campos.texto_es, en: campos.texto_en } };
    if (ambito === 'listas') cambio.opcion = { lista: campos.lista, codigo: campos.codigo, orden: Number(campos.orden), activa: campos.activa === 'true', admite_texto: campos.admite_texto === 'true' };
    if (ambito === 'plazos') cambio.plazo = { clave: campos.clave, valor: Number(campos.valor), unidad: campos.unidad };
    if (ambito === 'tramos') cambio.tramo = { tipo: campos.tipo_tramo, orden: Number(campos.orden), desde: Number(campos.desde), hasta: campos.hasta ? Number(campos.hasta) : null, unidad: campos.unidad };
    if (ambito === 'firmantes') cambio.firmante = { id: campos.id ?? '', tipo: campos.tipo, rol: campos.rol, nombre: campos.nombre, cargo_codigo: campos.cargo_codigo, organismo_codigo: campos.organismo_codigo, direccion: campos.direccion ?? '', vigente_hasta: campos.vigente_hasta ? civil(campos.vigente_hasta) : null };
    try {
      const r = await fetch(api(ambito), { method: campos._edicion ? 'PUT' : 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify(cambio), signal: controlador.signal });
      const cuerpo = await r.json() as Estado & Fallo;
      if (!r.ok) { if (r.status === 409) ponerConflicto(cuerpo.version_actual); throw new Error(claveError(cuerpo)); }
      if (!cuerpo.referencias || !cuerpo.textos || !cuerpo.datos) throw new Error('catalogo.no_disponible');
      if (!controlador.signal.aborted) { ponerDatos(cuerpo); ponerEditando(false); ponerModificado(false); ponerGuardado(true); }
    } catch (e: unknown) { if (!controlador.signal.aborted) ponerError(e instanceof Error ? e.message : 'catalogo.no_disponible'); }
    finally { if (!controlador.signal.aborted) ponerGuardando(false); }
  }
  const campo = (clave: string, tipo = 'text', requerido = true, bloqueado = false) => <div className="campo-admin" key={clave}>
    <label htmlFor={`campo-${clave}`}>{t(`catalogo.campo_${clave}`)}</label>
    <input id={`campo-${clave}`} type={tipo} value={campos[clave] ?? ''} required={requerido} readOnly={bloqueado} disabled={guardando} min={tipo === 'number' ? 0 : tipo === 'date' && (clave === 'vigente_hasta' ? campos.vigente_desde : campos._desde || (datos?.datos.version.numero ? datos.datos.version.vigente_desde : '')) ? fecha(new Date(new Date(clave === 'vigente_hasta' ? campos.vigente_desde : campos._desde || datos!.datos.version.vigente_desde).getTime() + 86400000).toISOString()) : undefined} step={tipo === 'number' && ['desde', 'hasta'].includes(clave) ? 'any' : undefined} maxLength={clave === 'direccion' ? 500 : clave.startsWith('texto_') ? 2000 : 200} aria-describedby={error ? 'error-admin' : undefined} onInvalid={(e) => { const detalle = e.currentTarget.closest('details'); if (detalle) detalle.open = true; }} onChange={(e) => editar(clave, e.target.value)} />
  </div>;
  const elegir = (clave: string, opciones: string[]) => <div className="campo-admin" key={clave}>
    <label htmlFor={`campo-${clave}`}>{t(`catalogo.campo_${clave}`)}</label>
    <select id={`campo-${clave}`} aria-describedby={error ? 'error-admin' : undefined} value={campos[clave]} disabled={guardando || (clave === 'tipo' && !!campos.id)} onChange={(e) => editar(clave, e.target.value)}>{opciones.map((v) => <option key={v} value={v}>{t(`catalogo.valor_${v}`)}</option>)}</select>
  </div>;
  const identificadores = (contenido: ReactNode) => <details className="identificadores-catalogo" open={!campos._edicion}><summary>{t('catalogo.identificadores')}</summary>{contenido}</details>;
  const filas = ambito === 'listas' ? datos?.datos.opciones ?? [] : ambito === 'plazos' ? datos?.datos.plazos ?? [] : ambito === 'tramos' ? datos?.datos.tramos ?? [] : datos?.firmantes ?? [];
  const nombreReglas = t('catalogo.tipo_regla');
  const nombreAyuda = t('catalogo.ayuda_etiqueta');
  const nombreAyudaPublicacion = t('catalogo.ayuda_publicacion_etiqueta');
  return <div className="administracion">
    <section className="contenido-admin" aria-labelledby="titulo-admin" aria-busy={cargando || guardando}>
      <div className="encabezado-catalogo"><h1 id="titulo-admin">{titulo}</h1>
      <details className="ayuda-catalogo" onKeyDown={(evento) => { if (evento.key === 'Escape') { evento.currentTarget.open = false; evento.currentTarget.querySelector('summary')?.focus(); } }}><summary aria-label={nombreAyuda}>{t('catalogo.ayuda')}</summary><p>{t('catalogo.presentacion')}</p></details></div>
      {['plazos', 'tramos'].includes(ambito) && <nav aria-label={nombreReglas} className="acciones-admin">
        <Boton secundario actual={ambito === 'plazos'} onPress={() => navegar('plazos')} disabled={guardando}>{t('catalogo.pantalla_plazos')}</Boton>
        <Boton secundario actual={ambito === 'tramos'} onPress={() => navegar('tramos')} disabled={guardando}>{t('catalogo.pantalla_tramos')}</Boton>
      </nav>}
      <p role="status">{cargando ? t('comun.cargando') : guardando ? t('catalogo.publicando') : guardado ? t('catalogo.publicado') : ''}</p>
      {error && <div role="alert" id="error-admin"><p>{t(error) || t('catalogo.no_disponible')}</p>{conflicto !== undefined && <p>{t('catalogo.version_actual')} {numero(conflicto)}</p>}
        <Boton secundario disabled={guardando} onPress={() => { if (puedeSalir()) { ponerEditando(false); ponerModificado(false); ponerConflicto(undefined); reintentar((n) => n + 1); } }}>{t(conflicto !== undefined ? 'catalogo.ver_cambios' : 'comun.reintentar')}</Boton>
      </div>}
      {datos && !editando && <>
        {datos.datos.version.numero > 0 && <p>{vigencia(datos.datos.version.numero, datos.datos.version.vigente_desde)}</p>}
        <Boton onPress={() => seleccionar()}>{t(`catalogo.nuevo_${ambito}`)}</Boton>
        {!filas.length && <p>{t('catalogo.vacio')}</p>}
        <ul className="lista-admin">{filas.map((f, i) => {
          const esFirmante = 'nombre' in f;
          const anterior = esFirmante && (datos.firmantes ?? []).some((otra) => otra.id === f.id && otra.version > f.version);
          const nombre = 'lista' in f ? etiqueta(`${f.lista}.${f.codigo}`) : 'clave' in f ? t(`catalogo.plazo_${f.clave}`) || t('catalogo.plazo_sin_etiqueta') : 'nombre' in f ? f.nombre : t(`catalogo.tramo_${f.tipo}`) || t('catalogo.tramo_sin_etiqueta');
          return <li key={i} className="fila-admin"><div><strong>{nombre}</strong>
            {'lista' in f && <><p className="contexto-catalogo">{t(`catalogo.lista_${f.lista}`) || t('catalogo.lista')}</p><p><span className={`pildora estado-${f.activa ? 'activo' : 'baja'}`}>{t(f.activa ? 'catalogo.valor_activa' : 'catalogo.valor_inactiva')}</span></p></>}
            {'valor' in f && <p>{numero(f.valor)} {t(`catalogo.unidad_${f.unidad}`) || t('catalogo.unidad_sin_etiqueta')}</p>}
            {'desde' in f && <p>{t('catalogo.intervalo').replace('{desde}', numero(f.desde)).replace('{hasta}', f.hasta !== null ? numero(f.hasta) : t('catalogo.sin_limite'))} {t(`catalogo.unidad_${f.unidad}`) || t('catalogo.unidad_sin_etiqueta')}</p>}
            {esFirmante && <><p>{t('catalogo.funcion_cargo').replace('{tipo}', t(`catalogo.valor_${f.tipo}`)).replace('{cargo}', f.cargo_textos[idioma] ?? f.cargo_textos[document.documentElement.dataset.idiomaReserva ?? 'es'] ?? t('catalogo.etiqueta_pendiente'))}</p><p className="contexto-catalogo">{vigencia(f.version, f.vigente_desde)}</p>{f.vigente_hasta && <p className="contexto-catalogo">{t('catalogo.hasta')} {formatearFecha(f.vigente_hasta)}</p>}</>}
          </div>{!anterior && <Boton secundario etiqueta={t('catalogo.editar_entrada').replace('{nombre}', nombre)} onPress={() => { seleccionar(f); }}>{t('catalogo.editar')}</Boton>}</li>;
        })}</ul>
      </>}
      {datos && editando && <form ref={formulario} onSubmit={(e) => { void guardar(e); }} className="formulario-admin">
        <h2>{t('catalogo.nueva_version')}</h2><details className="ayuda-catalogo"><summary aria-label={nombreAyudaPublicacion}>{t('catalogo.ayuda')}</summary><p id="ayuda-admin">{t('catalogo.ayuda_publicacion')}</p></details>
        <fieldset disabled={guardando}><legend>{t('catalogo.datos')}</legend>
          {ambito === 'listas' && <>{identificadores(<>{campo('lista', 'text', true, !!campos._edicion)}{campo('codigo', 'text', true, !!campos._edicion)}</>)}{campo('orden', 'number')}{elegir('activa', ['true', 'false'])}{elegir('admite_texto', ['true', 'false'])}</>}
          {ambito === 'plazos' && <>{campo('valor', 'number')}{identificadores(<>{campo('clave', 'text', true, !!campos._edicion)}{campo('unidad')}</>)}</>}
          {ambito === 'tramos' && <>{campo('orden', 'number')}{campo('desde', 'number')}{campo('hasta', 'number', false)}{identificadores(<>{campo('tipo_tramo', 'text', true, !!campos._edicion)}{campo('unidad')}</>)}</>}
          {ambito === 'firmantes' && <>{elegir('tipo', ['firmante', 'destinatario'])}{campo('nombre')}{identificadores(<>{campo('rol', 'text', true, !!campos.id)}{campo('cargo_codigo')}{campo('organismo_codigo')}</>)}{campo('direccion', 'text', false)}</>}
        </fieldset>
        <fieldset disabled={guardando}><legend>{t('catalogo.vigencia')}</legend>{campo('vigente_desde', 'date')}{ambito === 'firmantes' && campo('vigente_hasta', 'date', false)}</fieldset>
        {['listas', 'firmantes'].includes(ambito) && <fieldset disabled={guardando}><legend>{t(ambito === 'firmantes' ? 'catalogo.textos_cargo' : 'catalogo.textos_opcion')}</legend>{campo('texto_es')}{campo('texto_en')}</fieldset>}
        <div className="acciones-admin"><Boton type="submit" disabled={guardando || conflicto !== undefined}>{t('catalogo.publicar')}</Boton><Boton secundario disabled={guardando} onPress={() => { if (puedeSalir()) { ponerEditando(false); ponerModificado(false); ponerError(''); } }}>{t('catalogo.cancelar')}</Boton></div>
      </form>}
    </section>
  </div>;
}
