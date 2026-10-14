import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import { useCambios } from './cambios';

const perfiles = ['coordinacion', 'profesional_etf', 'consulta', 'administracion'] as const;
type Perfil = typeof perfiles[number];
type Concesion = { perfil: Perfil; desde: string; hasta: string | null };
type Equipo = { equipo_id: string; puesto: string; desde: string; hasta: string | null };
type Usuario = { id: string; cuenta: string; nombre: string; iniciales: string; estado: string; version: number; perfiles: Concesion[]; equipo: Equipo | null };
type OpcionEquipo = { id: string; codigo: string };
type Textos = (clave: string) => string;

function Boton({ children, accion, ocupado = false, etiqueta }: { children: React.ReactNode; accion: () => void; ocupado?: boolean; etiqueta?: string }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion, isDisabled: ocupado, 'aria-label': etiqueta }, ref);
  return <button {...buttonProps} ref={ref}>{children}</button>;
}

async function obtener<T>(ruta: string, signal: AbortSignal): Promise<T> {
  const respuesta = await fetch(ruta, { signal, credentials: 'same-origin' });
  const datos = await respuesta.json();
  if (!respuesta.ok) throw new Error(datos.error?.clave ?? 'identidad.persistencia_no_disponible');
  return datos as T;
}
function fechaLocal(iso: string | null): string {
  if (!iso) return '';
  const d = new Date(iso);
  return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
}
const iso = (valor: string) => new Date(valor).toISOString();

export function Usuarios({ t, csrf, actualizar }: { t: Textos; csrf: string; actualizar: () => void }) {
  const [lista, ponerLista] = useState<Usuario[]>([]);
  const [equipos, ponerEquipos] = useState<OpcionEquipo[]>([]);
  const [seleccion, seleccionar] = useState<Usuario>();
  const [configuracion, configurar] = useState<Concesion[]>([]);
  const [equipoID, cambiarEquipo] = useState('');
  const [puesto, cambiarPuesto] = useState('ts');
  const [baja, cambiarBaja] = useState(false);
  const [ocupado, cambiarOcupado] = useState(true);
  const [error, cambiarError] = useState('');
  const [guardado, cambiarGuardado] = useState(false);
  const [conflicto, cambiarConflicto] = useState(false);
  const [intento, reintentar] = useState(0);
  const { ponerSucio: cambiarSucio, solicitarSalida } = useCambios();
  const enVuelo = useRef(false);
  const controlador = useRef<AbortController | null>(null);
  const tituloEditor = useRef<HTMLHeadingElement>(null);
  const errorFormulario = useRef<HTMLParagraphElement>(null);

  function editar(u: Usuario) {
    seleccionar(u); configurar(u.perfiles); cambiarEquipo(u.equipo?.equipo_id ?? ''); cambiarPuesto(u.equipo?.puesto ?? 'ts');
    cambiarBaja(false); cambiarError(''); cambiarGuardado(false); cambiarConflicto(false); cambiarSucio(false);
    requestAnimationFrame(() => tituloEditor.current?.focus());
  }
  useEffect(() => {
    const c = new AbortController(); controlador.current = c; cambiarOcupado(true); cambiarError('');
    const hoy = new Date().toISOString().slice(0, 10);
    void Promise.all([obtener<Usuario[]>('/api/admin/usuarios', c.signal), obtener<{ equipos: OpcionEquipo[] }>(`/api/admin/catalogos/asignacion_municipios/datos?fecha=${hoy}`, c.signal)])
      .then(([l, d]) => { if (!c.signal.aborted) { ponerLista(l); ponerEquipos(d.equipos); } })
      .catch((e: unknown) => { if (!c.signal.aborted) cambiarError(e instanceof Error ? e.message : 'identidad.persistencia_no_disponible'); })
      .finally(() => { if (!c.signal.aborted) cambiarOcupado(false); });
    return () => c.abort();
  }, [intento]);
  useEffect(() => () => cambiarSucio(false), [cambiarSucio]);

  function marcar(p: Perfil, marcado: boolean) {
    configurar((previa) => marcado ? [...previa, { perfil: p, desde: new Date().toISOString(), hasta: null }] : previa.filter((v) => v.perfil !== p));
    cambiarSucio(true); cambiarGuardado(false);
  }
  function fecha(p: Perfil, campo: 'desde' | 'hasta', valor: string) {
    configurar((previa) => previa.map((v) => v.perfil === p ? { ...v, [campo]: valor ? iso(valor) : null } as Concesion : v)); cambiarSucio(true); cambiarGuardado(false);
  }
  async function guardar() {
    if (!seleccion || enVuelo.current || !controlador.current) return;
    const profesional = configuracion.find((p) => p.perfil === 'profesional_etf');
    if (!baja && profesional && !equipoID) { cambiarError('usuarios.equipo_obligatorio'); requestAnimationFrame(() => document.getElementById('equipo-usuario')?.focus()); return; }
    if (!baja && configuracion.some((p) => !p.desde || (p.hasta && new Date(p.hasta) <= new Date(p.desde)))) {
      cambiarError('identidad.perfiles_invalidos'); requestAnimationFrame(() => errorFormulario.current?.focus()); return;
    }
    enVuelo.current = true; cambiarOcupado(true); cambiarError(''); cambiarGuardado(false);
    const signal = controlador.current.signal;
    try {
      const respuesta = await fetch(`/api/admin/usuarios/${encodeURIComponent(seleccion.id)}/perfiles`, {
        method: 'PUT', signal, credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf },
        body: JSON.stringify({ version: seleccion.version, estado: baja ? 'baja' : 'activo', perfiles: baja ? [] : configuracion,
          equipo: !baja && profesional ? { equipo_id: equipoID, puesto, desde: profesional.desde, hasta: profesional.hasta } : null }),
      });
      const datos = await respuesta.json();
      if (!respuesta.ok) { if (respuesta.status === 409) cambiarConflicto(true); throw new Error(datos.error?.clave ?? 'identidad.persistencia_no_disponible'); }
      if (signal.aborted) return;
      const u = datos as Usuario;
      ponerLista((l) => l.map((v) => v.id === u.id ? u : v).sort((a, b) => Number(b.estado === 'pendiente_alta') - Number(a.estado === 'pendiente_alta') || a.cuenta.localeCompare(b.cuenta)));
      editar(u); cambiarGuardado(true); actualizar();
    } catch (e) { if (!signal.aborted) cambiarError(e instanceof Error ? e.message : 'identidad.persistencia_no_disponible'); }
    finally { enVuelo.current = false; if (!signal.aborted) cambiarOcupado(false); }
  }
  async function verCambios() {
    if (!seleccion || !controlador.current) return;
    cambiarOcupado(true);
    try { const l = await obtener<Usuario[]>('/api/admin/usuarios', controlador.current.signal); ponerLista(l); const u = l.find((v) => v.id === seleccion.id); if (u) editar(u); }
    catch (e) { if (!controlador.current.signal.aborted) cambiarError(e instanceof Error ? e.message : 'identidad.persistencia_no_disponible'); }
    finally { if (!controlador.current.signal.aborted) cambiarOcupado(false); }
  }
  return <section className="usuarios" aria-labelledby="titulo-usuarios" aria-busy={ocupado}>
    <h1 id="titulo-usuarios">{t('usuarios.titulo')}</h1>
    <p>{t('usuarios.descripcion')}</p>
    {ocupado && <p role="status">{t('comun.cargando')}</p>}
    {error && <p ref={errorFormulario} tabIndex={-1} id="error-usuarios" role="alert">{t(error) || t('identidad.persistencia_no_disponible')}</p>}
    {guardado && <p role="status">{t('usuarios.guardado')}</p>}
    {conflicto && <Boton accion={() => { void verCambios(); }} ocupado={ocupado}>{t('usuarios.ver_cambios')}</Boton>}
    {error && !seleccion && <Boton accion={() => reintentar((v) => v + 1)} ocupado={ocupado}>{t('comun.reintentar')}</Boton>}
    {!ocupado && !error && lista.length === 0 && <p>{t('usuarios.vacio')}</p>}
    <table className="tabla-usuarios" aria-labelledby="titulo-usuarios">
      <thead><tr>
        <th scope="col">{t('usuarios.col_persona')}</th><th scope="col">{t('usuarios.col_estado')}</th>
        <th scope="col">{t('usuarios.col_perfiles')}</th><th scope="col">{t('usuarios.equipo')}</th><th scope="col">{t('usuarios.col_accion')}</th>
      </tr></thead>
      <tbody>{lista.map((u) => <tr key={u.id}>
        <th scope="row"><strong>{u.nombre}</strong><small>{u.cuenta}</small></th>
        <td><span className="etiqueta-celda" aria-hidden="true">{t('usuarios.col_estado')}</span><span className={`pildora estado-${u.estado}`}>{t(`usuarios.estado_${u.estado}`)}</span></td>
        <td><span className="etiqueta-celda" aria-hidden="true">{t('usuarios.col_perfiles')}</span>{u.perfiles.length ? <ul className="perfiles-tabla">{u.perfiles.map((p) => <li key={p.perfil}>{t(`usuarios.perfil_${p.perfil}`)}</li>)}</ul> : t('usuarios.sin_perfiles')}</td>
        <td><span className="etiqueta-celda" aria-hidden="true">{t('usuarios.equipo')}</span>{u.equipo ? equipos.some((e) => e.id === u.equipo?.equipo_id) ? t('usuarios.equipo_opcion').replace('{codigo}', equipos.find((e) => e.id === u.equipo?.equipo_id)!.codigo) : t('comun.equipo_asignado') : t('usuarios.sin_equipo')}</td>
        <td><Boton ocupado={ocupado} etiqueta={t('usuarios.editar_persona').replace('{nombre}', u.nombre).replace('{cuenta}', u.cuenta)} accion={() => solicitarSalida(() => editar(u))}>{t('usuarios.accion_editar')}</Boton></td>
      </tr>)}</tbody>
    </table>

    {seleccion && <form noValidate onSubmit={(evento) => { evento.preventDefault(); void guardar(); }}>
      <h2 ref={tituloEditor} tabIndex={-1}>{t('usuarios.editar')} {seleccion.nombre}</h2>
      <fieldset disabled={ocupado} aria-describedby={error ? 'error-usuarios' : undefined}>
        <legend>{t('usuarios.perfiles')}</legend>
        {perfiles.map((p) => {
          const c = configuracion.find((v) => v.perfil === p);
          return <div key={p} className="concesion">
            <label><input type="checkbox" checked={!!c} disabled={baja} onChange={(e) => marcar(p, e.target.checked)} />{t(`usuarios.perfil_${p}`)}</label>
            {c && !baja && <div className="vigencias">
              <label htmlFor={`desde-${p}`}>{t('usuarios.desde')}</label><input id={`desde-${p}`} type="datetime-local" required value={fechaLocal(c.desde)} onChange={(e) => fecha(p, 'desde', e.target.value)} />
              <label htmlFor={`hasta-${p}`}>{t('usuarios.hasta')}</label><input id={`hasta-${p}`} type="datetime-local" value={fechaLocal(c.hasta)} onChange={(e) => fecha(p, 'hasta', e.target.value)} />
            </div>}
          </div>;
        })}
      </fieldset>
      {!baja && configuracion.some((p) => p.perfil === 'profesional_etf') && <fieldset disabled={ocupado} aria-describedby={error ? 'error-usuarios' : undefined}>
        <legend>{t('usuarios.adscripcion')}</legend>
        <label htmlFor="equipo-usuario">{t('usuarios.equipo')}</label>
        <select id="equipo-usuario" value={equipoID} onChange={(e) => { cambiarEquipo(e.target.value); cambiarSucio(true); cambiarGuardado(false); }} required>
          <option value="">{t('usuarios.elegir_equipo')}</option>
          {equipos.map((e) => <option key={e.id} value={e.id}>{t('usuarios.equipo_opcion').replace('{codigo}', e.codigo)}</option>)}
        </select>
        <label htmlFor="puesto-usuario">{t('usuarios.puesto')}</label>
        <select id="puesto-usuario" value={puesto} onChange={(e) => { cambiarPuesto(e.target.value); cambiarSucio(true); cambiarGuardado(false); }}>
          {['ts', 'es', 'ps'].map((p) => <option key={p} value={p}>{t(`usuarios.puesto_${p}`)}</option>)}
        </select>
      </fieldset>}
      <label className="control-baja"><input type="checkbox" checked={baja} disabled={ocupado} onChange={(e) => { cambiarBaja(e.target.checked); cambiarSucio(true); cambiarGuardado(false); }} />{t('usuarios.baja')}</label>
      {baja && <p className="pendiente">{t('usuarios.baja_consecuencia')}</p>}
      <Boton accion={() => { void guardar(); }} ocupado={ocupado}>{t('usuarios.guardar')}</Boton>
    </form>}
  </section>;
}
