import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import type { ReactNode } from 'react';
import type { PropsPantalla } from './registro-pantallas';

function Button({ children, isDisabled, onPress, secundario }: { children: ReactNode; isDisabled?: boolean; onPress: () => void; secundario?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress, isDisabled }, ref);
  return <button {...buttonProps} ref={ref} className={secundario ? 'boton-secundario' : undefined}>{children}</button>;
}

type Fila = { expediente_id: string; numero: string; cierre: string; fin_conservacion: string; anios: number; version: number; estado: string; puede_confirmar: boolean };
export function Conservacion({ t, idioma, csrf }: PropsPantalla) {
  const [filas, cambiarFilas] = useState<Fila[]>([]);
  const [cargando, cambiarCarga] = useState(true);
  const [error, cambiarError] = useState('');
  const dialogo = useRef<HTMLDialogElement>(null);
  const resumenError = useRef<HTMLParagraphElement>(null);
  const resumenResultado = useRef<HTMLParagraphElement>(null);
  const [seleccion, cambiarSeleccion] = useState<Fila | null>(null);
  const [revisado, cambiarRevision] = useState(false);
  const [guardando, cambiarGuardando] = useState(false);
  const [resultado, cambiarResultado] = useState('');
  const controlador = useRef<AbortController | null>(null);
  const fecha = (valor: string) => new Intl.DateTimeFormat(idioma, { dateStyle: 'medium', timeZone: 'UTC' }).format(new Date(`${valor}T00:00:00Z`));
  const cargar = async (signal?: AbortSignal) => {
    cambiarCarga(true); cambiarError('');
    try {
      const r = await fetch('/api/conservacion/pendientes', { signal });
      if (!r.ok) { cambiarError(r.status === 401 || r.status === 403 ? 'conservacion.sin_permiso' : 'conservacion.error'); return; }
      const datos: unknown = await r.json();
      if (!Array.isArray(datos)) throw new Error('conservacion.error');
      if (!signal?.aborted) cambiarFilas(datos as Fila[]);
    } catch { if (!signal?.aborted) cambiarError('conservacion.error'); }
    finally { if (!signal?.aborted) cambiarCarga(false); }
  };
  useEffect(() => { const c = new AbortController(); controlador.current = c; void cargar(c.signal); return () => c.abort(); }, []);
  useEffect(() => { if (seleccion) dialogo.current?.showModal(); else dialogo.current?.close(); }, [seleccion]);
  useEffect(() => { if (error) resumenError.current?.focus(); else if (resultado) resumenResultado.current?.focus(); }, [error, resultado]);
  useEffect(() => { document.title = t('conservacion.titulo'); }, [t]);
  const confirmar = async () => {
    if (!seleccion || !revisado) return;
    const fila = seleccion;
    const accion = fila.puede_confirmar ? 'anonimizar' : 'proponer';
    cambiarGuardando(true); cambiarError(''); cambiarResultado('');
    try {
      const r = await fetch(`/api/conservacion/${fila.expediente_id}/${accion}`, { method: 'POST', signal: controlador.current?.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ version: fila.version }) });
      if (!r.ok) { cambiarError(r.status === 409 ? 'conservacion.conflicto' : r.status === 401 || r.status === 403 ? 'conservacion.sin_permiso' : 'conservacion.error'); cambiarSeleccion(null); return; }
      const datos = await r.json() as Fila;
      if (!datos.expediente_id || !datos.estado) throw new Error();
      cambiarFilas((actual) => actual.map((f) => f.expediente_id === datos.expediente_id ? datos : f));
      cambiarSeleccion(null); cambiarResultado(accion === 'anonimizar' ? 'conservacion.hecho' : 'conservacion.propuesta_hecha');
    } catch { if (!controlador.current?.signal.aborted) { cambiarSeleccion(null); cambiarError('conservacion.error'); } }
    finally { if (!controlador.current?.signal.aborted) cambiarGuardando(false); }
  };
  return <section className="contenido-admin" aria-labelledby="conservacion-titulo">
    <h1 id="conservacion-titulo">{t('conservacion.titulo')}</h1>
    <details className="ayuda-admin"><summary>{t('conservacion.ayuda')}</summary><p>{t('conservacion.ayuda_texto')}</p><a href="/admin/reglas">{t('conservacion.editar_plazo')}</a></details>
    {cargando && <p role="status">{t('conservacion.cargando')}</p>}
    {error && <p ref={resumenError} tabIndex={-1} role="alert">{t(error)}</p>}
    {resultado && <p ref={resumenResultado} tabIndex={-1} role="status">{t(resultado)}</p>}
    <Button secundario isDisabled={cargando || guardando} onPress={() => void cargar(controlador.current?.signal)}>{t('conservacion.actualizar')}</Button>
    {!cargando && !error && filas.length === 0 && <p>{t('conservacion.vacio')}</p>}
    <ul className="lista-admin">{filas.map((f) => <li className="fila-admin" key={f.expediente_id}>
      <h2>{f.numero}</h2>
      <dl><dt>{t('conservacion.cierre')}</dt><dd>{fecha(f.cierre)}</dd><dt>{t('conservacion.fin')}</dt><dd>{fecha(f.fin_conservacion)}</dd><dt>{t('conservacion.estado')}</dt><dd>{t(`conservacion.${f.estado}`)}</dd></dl>
      {(f.estado === 'pendiente_propuesta' || f.puede_confirmar) && <Button isDisabled={guardando || !!error} onPress={() => { cambiarSeleccion(f); cambiarRevision(false); cambiarResultado(''); }}>{t(f.puede_confirmar ? 'conservacion.revisar_anonimizacion' : 'conservacion.revisar_propuesta')}</Button>}
    </li>)}</ul>
    <dialog className="dialogo-admin" ref={dialogo} aria-labelledby="conservacion-revision" onCancel={(evento) => { evento.preventDefault(); if (!guardando) cambiarSeleccion(null); }}>
      <h2 id="conservacion-revision">{t('conservacion.revision')}</h2>
      <p>{seleccion?.numero}</p>
      {seleccion && <dl><dt>{t('conservacion.cierre')}</dt><dd>{fecha(seleccion.cierre)}</dd><dt>{t('conservacion.fin')}</dt><dd>{fecha(seleccion.fin_conservacion)}</dd></dl>}
      <p>{t(seleccion?.puede_confirmar ? 'conservacion.consecuencia_final' : 'conservacion.consecuencia_propuesta')}</p>
      <label className="casilla-admin"><input type="checkbox" checked={revisado} onChange={(e) => cambiarRevision(e.target.checked)} disabled={guardando} />{t('conservacion.revisado')}</label>
      <div className="acciones-admin"><Button isDisabled={!revisado || guardando} onPress={() => void confirmar()}>{t(guardando ? 'conservacion.guardando' : seleccion?.puede_confirmar ? 'conservacion.anonimizar' : 'conservacion.proponer')}</Button>
      <Button secundario isDisabled={guardando} onPress={() => cambiarSeleccion(null)}>{t('conservacion.cancelar')}</Button></div>
    </dialog>
  </section>;
}
