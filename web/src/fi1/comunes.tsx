import { useEffect, useRef } from 'react';
import { useButton } from 'react-aria';
import type { PropsPantalla } from '../registro-pantallas';
import { useLectura } from '../equipo/api';
import { campoID, type Aviso, type FichaBase, type Menor } from './modelo';
import type { useEdicion } from './edicion';

export type Textos = PropsPantalla['t'];
export type ErrorCampo = { id: string; clave: string };
export function hoyCivil() {
  const partes = new Intl.DateTimeFormat('en', { timeZone: 'Europe/Madrid', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date());
  return ['year', 'month', 'day'].map((tipo) => partes.find((p) => p.type === tipo)?.value).join('-');
}
export function enfocar(id: string) {
  const elemento = document.getElementById(id);
  let padre = elemento?.parentElement;
  while (padre) { if (padre instanceof HTMLDetailsElement) padre.open = true; padre = padre.parentElement; }
  const foco = elemento?.matches(':disabled') ? elemento.closest<HTMLElement>('.fi1-campo') : elemento;
  foco?.focus(); elemento?.scrollIntoView({ block: 'center' });
  if (foco && location.hash === '#' + id) history.replaceState(null, '', location.pathname + location.search);
}
export function BotonGuardar({ t, ocupado }: { t: Textos; ocupado: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ type: 'submit', isDisabled: ocupado }, ref);
  return <button {...buttonProps} ref={ref} className="principal">{t(ocupado ? 'fi1.guardando' : 'fi1.guardar')}</button>;
}
export function Campo({ id, clave, t, pendiente, error, children }: { id: string; clave: string; t: Textos; pendiente?: boolean; error?: string; children: React.ReactNode }) {
  return <div className="fi1-campo" tabIndex={-1}><label htmlFor={id}>{t(clave)}</label>{children}
    {(pendiente || error) && <small id={id + '-aviso'} className={error ? 'fi1-error' : 'fi1-pendiente'}>{t(error ?? 'fi1.falta')}</small>}
  </div>;
}
export function Errores({ errores, t }: { errores: ErrorCampo[]; t: Textos }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => { if (errores.length) ref.current?.focus(); }, [errores]);
  return errores.length ? <div className="fi1-resumen" ref={ref} tabIndex={-1} role="alert"><p>{t('fi1.revisar')}</p><ul>{errores.map((e) => <li key={e.id}><a href={'#' + e.id} onClick={(evento) => { evento.preventDefault(); enfocar(e.id); }}>{t(e.clave)}</a></li>)}</ul></div> : null;
}
export function EstadoEdicion<T extends FichaBase>({ edicion: e, t, comparacion }: { edicion: ReturnType<typeof useEdicion<T>>; t: Textos; comparacion: React.ReactNode }) {
  const textoError = t(e.error);
  return <>
    {e.cargando && <p role="status">{t('fi1.cargando')}</p>}
    {e.error && <div role="alert"><p>{textoError && textoError !== e.error ? textoError : t('fi1.error')}</p>{(!e.datos || e.recuperar || e.conflicto) && <button type="button" disabled={e.ocupado} onClick={() => void e.reintentar()}>{t('fi1.reintentar')}</button>}</div>}
    {e.recuperar && <p role="status">{t('fi1.recuperar')}</p>}
    {e.resultado && <p role="status">{t(e.resultado)}</p>}
    {e.sucio && <p role="status">{t('fi1.sin_guardar')}</p>}
    {e.conflicto && <section className="fi1-resumen" aria-labelledby="fi1-conflicto"><h3 id="fi1-conflicto">{t('fi1.conflicto')}</h3><p>{t('fi1.conflicto_detalle')}</p>
      <button type="button" disabled={e.ocupado} onClick={() => void e.reintentar()}>{t('fi1.ver_actuales')}</button>
      {e.actuales && <><h3>{t('fi1.actuales')}</h3>{comparacion}<p>{t('fi1.comparar')}</p><div className="fi1-acciones">
        <button type="button" onClick={e.mantener}>{t('fi1.mantener')}</button><button type="button" onClick={e.usarActuales}>{t('fi1.usar_actuales')}</button>
      </div></>}
    </section>}
  </>;
}
export function AvisosCampos({ id, campos, sujetos, t, revision }: { id: string; campos: readonly string[]; sujetos?: Menor[]; t: Textos; revision: number }) {
  const lectura = useLectura<{ avisos: Aviso[] }>(`/api/expedientes/${encodeURIComponent(id)}/avisos`);
  useEffect(() => { lectura.recargar(); }, [revision]);
  const avisos = lectura.datos?.avisos.filter((a) => campos.includes(a.campo) && (!a.sujeto_id || sujetos?.some((m) => m.id === a.sujeto_id))) ?? [];
  const unicos = avisos.filter((a, i) => avisos.findIndex((b) => b.campo === a.campo && b.sujeto_id === a.sujeto_id) === i);
  return <>
    {lectura.cargando && <p role="status">{t('fi1.avisos_cargando')}</p>}
    {lectura.error && <div role="alert"><p>{t('fi1.avisos_error')}</p><button type="button" onClick={lectura.recargar}>{t('fi1.reintentar')}</button></div>}
    {unicos.length > 0 && <aside className="fi1-resumen" aria-labelledby="fi1-avisos"><h3 id="fi1-avisos">{t('fi1.avisos')}</h3><ul>{unicos.map((a) => {
      const destino = campoID(a.campo, a.sujeto_id);
      const menor = sujetos?.find((m) => m.id === a.sujeto_id);
      const sujeto = menor ? t(menor.nombre ? 'fi1.aviso_menor_nombre' : 'fi1.aviso_menor').replace('{orden}', new Intl.NumberFormat(document.documentElement.lang).format(menor.orden)).replace('{nombre}', menor.nombre) : '';
      return <li key={destino}><a href={'#' + destino} onClick={(e) => { e.preventDefault(); enfocar(destino); }}>{t('fi1.campo.' + a.campo)}{sujeto}</a></li>;
    })}</ul></aside>}
  </>;
}
