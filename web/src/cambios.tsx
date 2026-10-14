import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';

type Guardia = { sucio: boolean; ponerSucio: (valor: boolean) => void; solicitarSalida: (accion: () => void) => void };
const ContextoCambios = createContext<Guardia | null>(null);
export function useCambios(): Guardia {
  const guardia = useContext(ContextoCambios);
  if (!guardia) throw new Error('identidad.configuracion_invalida');
  return guardia;
}
function Boton({ accion, children }: { accion: () => void; children: React.ReactNode }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion }, ref);
  return <button {...buttonProps} ref={ref}>{children}</button>;
}
export function GuardiaCambios({ t, children }: { t: (clave: string) => string; children: React.ReactNode }) {
  const [sucio, ponerSucio] = useState(false);
  const [pendiente, ponerPendiente] = useState<(() => void)>();
  const aviso = useRef<HTMLParagraphElement>(null);
  const focoAnterior = useRef<HTMLElement | null>(null);
  const solicitarSalida = useCallback((accion: () => void) => {
    if (!sucio) { accion(); return; }
    focoAnterior.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    ponerPendiente(() => accion);
  }, [sucio]);
  useEffect(() => { if (pendiente) aviso.current?.focus(); }, [pendiente]);
  useEffect(() => {
    const avisar = (evento: BeforeUnloadEvent) => { if (sucio) evento.preventDefault(); };
    window.addEventListener('beforeunload', avisar);
    return () => window.removeEventListener('beforeunload', avisar);
  }, [sucio]);
  function continuar() {
    ponerPendiente(undefined);
    let ancestro = focoAnterior.current?.parentElement;
    while (ancestro) { if (ancestro instanceof HTMLDetailsElement) ancestro.open = true; ancestro = ancestro.parentElement; }
    focoAnterior.current?.focus();
  }
  function descartar() { const accion = pendiente; ponerPendiente(undefined); ponerSucio(false); accion?.(); }
  return <ContextoCambios.Provider value={{ sucio, ponerSucio, solicitarSalida }}>
    {pendiente && <section className="pendiente" aria-labelledby="aviso-cambios">
      <p id="aviso-cambios" ref={aviso} tabIndex={-1} role="alert">{t('usuarios.sin_guardar')}</p>
      <Boton accion={continuar}>{t('usuarios.seguir_editando')}</Boton>
      <Boton accion={descartar}>{t('usuarios.descartar')}</Boton>
    </section>}
    {children}
  </ContextoCambios.Provider>;
}
