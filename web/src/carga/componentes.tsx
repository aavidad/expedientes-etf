import { useEffect, useRef, type ReactNode } from 'react';
import { useButton } from 'react-aria';
export function Boton({ children, accion, principal = false, disabled = false }: { children: ReactNode; accion: () => void; principal?: boolean; disabled?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion, isDisabled: disabled }, ref);
  return <button {...buttonProps} ref={ref} className={principal ? 'carga-principal' : undefined}>{children}</button>;
}
export function Ayuda({ t, cerrar }: { t: (k: string) => string; cerrar: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => { const antes = document.activeElement as HTMLElement; ref.current?.showModal(); return () => { ref.current?.close(); antes?.focus(); }; }, []);
  return <dialog className="carga" ref={ref} aria-labelledby="carga-ayuda-titulo" onCancel={e => { e.preventDefault(); cerrar(); }}>
    <h2 id="carga-ayuda-titulo">{t('carga.ayuda_titulo')}</h2><p>{t('carga.ayuda')}</p><h3>{t('carga.formato_titulo')}</h3><p>{t('carga.formato')}</p><Boton accion={cerrar}>{t('carga.cerrar_ayuda')}</Boton>
  </dialog>;
}
export function Campo({ id, etiqueta, valor, cambiar, tipo = 'text', obligatorio = false, error, area = false, disabled = false, opciones, t, max = 4000 }: {
  id: string; etiqueta: string; valor: string; cambiar: (v: string) => void; tipo?: string; obligatorio?: boolean; error?: string; area?: boolean; disabled?: boolean;
  opciones?: { valor: string; etiqueta: string }[]; t: (k: string) => string; max?: number;
}) {
  const props = { id, value: valor, disabled, required: obligatorio, 'aria-invalid': !!error, 'aria-describedby': error ? `${id}-error` : obligatorio && !valor ? `${id}-pendiente` : undefined, onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => cambiar(e.target.value) };
  return <div className={`carga-campo ${error || obligatorio && !valor ? 'carga-falta' : ''}`}>
    <label htmlFor={id}>{etiqueta}</label>
    {opciones ? <select {...props}><option value="">{t('carga.elegir')}</option>{opciones.map(o => <option key={o.valor} value={o.valor}>{o.etiqueta}</option>)}</select> : area ? <textarea {...props} maxLength={max} rows={3} /> : <input {...props} type={tipo} maxLength={max} />}
    {error ? <p id={`${id}-error`}>{error}</p> : obligatorio && !valor ? <p id={`${id}-pendiente`}>{t('carga.obligatorio')}</p> : null}
  </div>;
}
