import { useEffect, useRef } from 'react';
import { Boton } from '../salidas/componentes';

export function AyudaAnexo6({ t, cerrar }: { t: (clave: string) => string; cerrar: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const anterior = document.activeElement as HTMLElement | null;
    const dialogo = ref.current; dialogo?.showModal();
    return () => { dialogo?.close(); anterior?.focus(); };
  }, []);
  return <dialog ref={ref} className="salidas-dialogo" aria-labelledby="anexo6-ayuda" onCancel={cerrar}>
    <h2 id="anexo6-ayuda">{t('anexo6.ayuda_titulo')}</h2><p>{t('anexo6.ayuda_texto')}</p>
    <Boton accion={cerrar}>{t('salidas.cerrar_ayuda')}</Boton>
  </dialog>;
}
