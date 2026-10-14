import { useEffect, useRef, type ReactNode } from 'react';
import { useButton } from 'react-aria';
import type { Operacion } from './api';
export function Boton({ accion, children, disabled = false, principal = false, etiqueta }: { accion: () => void; children: ReactNode; disabled?: boolean; principal?: boolean; etiqueta?: string }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion, isDisabled: disabled }, ref);
  return <button {...buttonProps} ref={ref} aria-label={etiqueta} className={principal ? ['boton-primario', 'principal'].join(' ') : 'boton-secundario'}>{children}</button>;
}
export function Dialogo({ titulo, children, cerrar }: { titulo: string; children: ReactNode; cerrar: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { const anterior = document.activeElement as HTMLElement | null; const d = ref.current; d?.showModal(); return () => { d?.close(); anterior?.focus(); }; }, []);
  return <dialog className="traslado-dialogo" ref={ref} aria-labelledby="traslado-dialogo-titulo" onCancel={cerrar}><h2 id="traslado-dialogo-titulo">{titulo}</h2>{children}</dialog>;
}
export function Errores({ campos, t }: { campos: string[]; t: (k: string) => string }) {
  const ref = useRef<HTMLElement>(null);
  useEffect(() => { if (campos.length) ref.current?.focus(); }, [campos]);
  return campos.length > 0 && <section ref={ref} tabIndex={-1} role="alert" className="pendiente"><h3>{t('traslado.faltan_datos')}</h3><ul>{campos.map(c => <li key={c}><a href={`#traslado-${c}`} onClick={e => { e.preventDefault(); document.getElementById(`traslado-${c}`)?.focus(); }}>{t(`traslado.${c}`)}</a></li>)}</ul></section>;
}
export function ErrorLectura({ error, recargar, navegarRuta, t }: { error: string; navegarRuta: (ruta: string) => void; recargar: () => void; t: (k: string) => string }) { return error && <div role="alert" className="pendiente"><p>{t(error)}</p><Boton accion={recargar}>{t('traslado.reintentar')}</Boton>{['traslado.sin_permiso', 'traslado.no_encontrado'].includes(error) && <a href="/equipo" onClick={e => { e.preventDefault(); navegarRuta('/equipo'); }}>{t('traslado.volver_bandeja')}</a>}</div>; }
export function ErrorOperacion({ error, pendiente, ocupado, enviar, recargar, navegarRuta, t }: { error: string; navegarRuta: (ruta: string) => void; pendiente?: Operacion; ocupado: boolean; enviar: (op: Operacion) => Promise<void>; recargar: () => void; t: (k: string) => string }) {
  return error && <div role="alert" className="pendiente"><p>{t(error)}</p>{pendiente ? <Boton disabled={ocupado} accion={() => void enviar(pendiente)}>{t('traslado.recuperar')}</Boton> : error === 'traslado.conflicto' && <Boton accion={recargar}>{t('traslado.ver_cambios')}</Boton>}{['traslado.sin_permiso', 'traslado.no_encontrado'].includes(error) && <a href="/equipo" onClick={e => { e.preventDefault(); navegarRuta('/equipo'); }}>{t('traslado.volver_bandeja')}</a>}</div>;
}
