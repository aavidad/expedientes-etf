import { useEffect, useRef, type ReactNode } from 'react';
import { useButton } from 'react-aria';

export function Boton({ accion, children, secundario = false, disabled = false }: { accion: () => void; children: ReactNode; secundario?: boolean; disabled?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion, isDisabled: disabled }, ref);
  return <button {...buttonProps} ref={ref} className={secundario ? 'boton-secundario' : undefined}>{children}</button>;
}
export function Confirmacion({ titulo, texto, confirmar, volver, ocupado, t }: { titulo: string; texto: string; confirmar: () => void; volver: () => void; ocupado: boolean; t: (clave: string) => string }) {
  const dialogo = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const anterior = document.activeElement as HTMLElement | null;
    dialogo.current?.showModal();
    return () => { dialogo.current?.close(); anterior?.focus(); };
  }, []);
  return <dialog ref={dialogo} aria-labelledby="decision-confirmar-titulo" aria-describedby="decision-confirmar-texto" onCancel={e => { e.preventDefault(); if (!ocupado) volver(); }}>
    <h2 id="decision-confirmar-titulo">{titulo}</h2>
    <p id="decision-confirmar-texto">{texto}</p>
    <div className="decision-acciones"><Boton secundario accion={volver} disabled={ocupado}>{t('decision.volver')}</Boton><Boton accion={confirmar} disabled={ocupado}>{ocupado ? t('decision.guardando') : titulo}</Boton></div>
  </dialog>;
}

export function Ayuda({ titulo, texto, cerrar, t }: { titulo: string; texto: string; cerrar: () => void; t: (clave: string) => string }) {
  const dialogo = useRef<HTMLDialogElement>(null);
  useEffect(() => { const anterior = document.activeElement as HTMLElement | null; dialogo.current?.showModal(); return () => { dialogo.current?.close(); anterior?.focus(); }; }, []);
  return <dialog ref={dialogo} aria-labelledby="decision-ayuda-titulo" onCancel={e => { e.preventDefault(); cerrar(); }}><h2 id="decision-ayuda-titulo">{titulo}</h2><p>{texto}</p><Boton secundario accion={cerrar}>{t('decision.cerrar_ayuda')}</Boton></dialog>;
}
