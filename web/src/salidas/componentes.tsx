import { useEffect, useRef, type ReactNode } from 'react';
import { useButton } from 'react-aria';
export function Boton({ accion, children, disabled = false, principal = false }: { accion: () => void; children: ReactNode; disabled?: boolean; principal?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion, isDisabled: disabled }, ref);
  return <button {...buttonProps} ref={ref} className={principal ? 'boton-primario' : 'boton-secundario'}>{children}</button>;
}
export function Confirmar({ titulo, texto, aceptar, cancelar, t }: { titulo: string; texto: string; aceptar: () => void; cancelar: () => void; t: (k: string) => string }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { const anterior = document.activeElement as HTMLElement | null; const d = ref.current; d?.showModal(); return () => { d?.close(); anterior?.focus(); }; }, []);
  return <dialog className="salidas-dialogo" ref={ref} aria-labelledby="salidas-confirmar" onCancel={cancelar}><h2 id="salidas-confirmar">{titulo}</h2><p>{texto}</p><div className="salidas-acciones"><Boton accion={cancelar}>{t('salidas.cancelar')}</Boton><Boton principal accion={aceptar}>{titulo}</Boton></div></dialog>;
}

export function Ayuda({ t, cerrar }: { t: (k: string) => string; cerrar: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { const anterior = document.activeElement as HTMLElement | null; const d = ref.current; d?.showModal(); return () => { d?.close(); anterior?.focus(); }; }, []);
  return <dialog className="salidas-dialogo" ref={ref} aria-labelledby="salidas-ayuda" onCancel={cerrar}><h2 id="salidas-ayuda">{t('salidas.ayuda_titulo')}</h2><p>{t('salidas.ayuda_texto')}</p><Boton accion={cerrar}>{t('salidas.cerrar_ayuda')}</Boton></dialog>;
}
