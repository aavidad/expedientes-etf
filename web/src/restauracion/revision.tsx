import './revision.css';
import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import type { PropsPantalla } from '../registro-pantallas';
type Persona = { nombre: string; estado: string; perfiles: { perfil: string; desde: string; hasta: string | null }[]; equipo: { equipo_id: string; puesto: string; desde: string; hasta: string | null } | null };
type Cambio = { id: string; anterior: Persona | null; propuesto: Persona | null; decision: string | null };
type Revision = { id: string; copia_en: string; version: number; cambios: Cambio[]; equipos: Record<string,string> };
function Boton({ accion, ocupado, children, etiqueta, secundario = false, id }: { accion: () => void; ocupado: boolean; etiqueta?: string; secundario?: boolean; id?: string; children: React.ReactNode }) {
 const ref = useRef<HTMLButtonElement>(null); const { buttonProps } = useButton({ onPress: accion, isDisabled: ocupado, 'aria-label': etiqueta }, ref);
 return <button id={id} {...buttonProps} ref={ref} className={secundario ? 'boton-secundario' : undefined}>{children}</button>;
}
export function Restauracion({ t, idioma, csrf, actualizar }: PropsPantalla) {
 const [revision, ponerRevision] = useState<Revision>(); const [ocupado, ponerOcupado] = useState(true);
 const [error, ponerError] = useState(''); const [intento, reintentar] = useState(0);
 const [confirmacion, confirmar] = useState(false); const [aviso, ponerAviso] = useState('');
 const controlador = useRef<AbortController | null>(null); const enviando = useRef(false); const errorRef = useRef<HTMLParagraphElement>(null); const tituloConfirmacion = useRef<HTMLHeadingElement>(null);
 useEffect(() => { if (confirmacion) tituloConfirmacion.current?.focus(); }, [confirmacion]);
 async function leer(signal: AbortSignal) {
  const r = await fetch('/api/admin/restauracion', { signal, credentials: 'same-origin' }); const d = await r.json();
  if (!r.ok) throw new Error(d.error?.clave ?? 'identidad.persistencia_no_disponible');
  if (!signal.aborted) ponerRevision(d as Revision);
 }
 useEffect(() => { const c = new AbortController(); controlador.current = c; ponerOcupado(true); ponerError('');
  void leer(c.signal).catch((e: unknown) => { if (!c.signal.aborted) ponerError(e instanceof Error ? e.message : 'identidad.persistencia_no_disponible'); })
   .finally(() => { if (!c.signal.aborted) ponerOcupado(false); }); return () => c.abort();
 }, [intento]);
 async function decidir(cambio?: Cambio, decision?: string) {
  if (!revision || !controlador.current || enviando.current) return;
  const signal = controlador.current.signal; enviando.current = true; ponerOcupado(true); ponerError(''); ponerAviso('');
  try {
   const ruta = cambio ? `/api/admin/restauracion/${revision.id}/cambios/${cambio.id}` : `/api/admin/restauracion/${revision.id}/confirmar`;
   const r = await fetch(ruta, { method: 'POST', credentials: 'same-origin', signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ version: revision.version, ...(cambio ? { decision } : {}) }) });
   const d = await r.json(); if (!r.ok) throw new Error(d.error?.clave ?? 'identidad.persistencia_no_disponible');
   if (signal.aborted) return;
   if (!cambio) { actualizar(); return; }
   await leer(signal); ponerAviso('restauracion.decision_guardada');
  } catch (e) { if (!signal.aborted) { ponerError(e instanceof Error ? e.message : 'identidad.persistencia_no_disponible'); requestAnimationFrame(() => errorRef.current?.focus()); } }
  finally { enviando.current = false; if (!signal.aborted) ponerOcupado(false); }
 }
 function fecha(valor: string) { return new Intl.DateTimeFormat(idioma, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(valor)); }
 function datos(persona: Persona | null) {
  if (!persona) return <p>{t('restauracion.sin_persona')}</p>;
  return <><p>{t(`usuarios.estado_${persona.estado}`)}</p><ul>{persona.perfiles.map((p) => <li key={p.perfil}>{t(`usuarios.perfil_${p.perfil}`)}<p>{t('usuarios.desde')} {fecha(p.desde)}{p.hasta && <> {t('restauracion.separador')} {t('usuarios.hasta')} {fecha(p.hasta)}</>}</p></li>)}</ul>
   <p>{persona.equipo ? t('usuarios.equipo_opcion').replace('{codigo}', revision?.equipos[persona.equipo.equipo_id] ?? t('restauracion.equipo_pendiente')) : t('usuarios.sin_equipo')}</p>{persona.equipo && <p>{t(`usuarios.puesto_${persona.equipo.puesto}`)} {t('restauracion.separador')} {t('usuarios.desde')} {fecha(persona.equipo.desde)}{persona.equipo.hasta && <> {t('restauracion.separador')} {t('usuarios.hasta')} {fecha(persona.equipo.hasta)}</>}</p>}</>;
 }
 const pendientes = revision?.cambios.filter((c) => !c.decision).length ?? 0;
 return <section aria-labelledby="titulo-restauracion" aria-busy={ocupado}>
  <h1 id="titulo-restauracion">{t('restauracion.titulo')}</h1><p>{t('restauracion.descripcion')}</p>
  {revision && <p>{t('restauracion.fecha_copia')} <time>{new Intl.DateTimeFormat(idioma, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(revision.copia_en))}</time></p>}
  {ocupado && <p role="status">{t('comun.cargando')}</p>}
  {error && <p id="error-restauracion" ref={errorRef} tabIndex={-1} role="alert">{t(error) || t('identidad.persistencia_no_disponible')}</p>}
  {error && <Boton accion={() => reintentar((v) => v + 1)} ocupado={ocupado}>{t('restauracion.actualizar')}</Boton>}
  {aviso && <p role="status">{t(aviso)}</p>}
  {revision && <><p role="status">{t('restauracion.pendientes').replace('{cantidad}', new Intl.NumberFormat(idioma).format(pendientes))}</p>
   {revision.cambios.length === 0 && <p>{t('restauracion.vacio')}</p>}
   {revision.cambios.map((c) => <section key={c.id} className="revision-cambio" aria-labelledby={`persona-${c.id}`}>
    <h2 id={`persona-${c.id}`}>{c.propuesto?.nombre ?? c.anterior?.nombre}</h2>
    <p>{!c.anterior ? t('restauracion.alta') : !c.propuesto || c.propuesto.estado === 'baja' ? t('restauracion.baja') : t('restauracion.cambio')}</p>{c.anterior && c.propuesto && <ul>{JSON.stringify(c.anterior.perfiles) !== JSON.stringify(c.propuesto.perfiles) && <li>{t('restauracion.cambio_perfil')}</li>}{JSON.stringify(c.anterior.equipo) !== JSON.stringify(c.propuesto.equipo) && <li>{t('restauracion.cambio_equipo')}</li>}</ul>}
    <div className="revision-comparacion"><div><h3>{t('restauracion.en_copia')}</h3>{datos(c.anterior)}</div><div><h3>{t('restauracion.vigente')}</h3>{datos(c.propuesto)}</div></div>
    {c.decision ? <p>{t(`restauracion.${c.decision}_guardado`)}</p> : <><p>{t('restauracion.consecuencia')}</p>
     <div className="acciones-admin"><Boton accion={() => { void decidir(c, 'aplicar'); }} ocupado={ocupado} etiqueta={t('restauracion.aplicar_persona').replace('{nombre}', c.propuesto?.nombre ?? c.anterior?.nombre ?? '')}>{t('restauracion.aplicar')}</Boton>
     <Boton secundario accion={() => { void decidir(c, 'descartar'); }} ocupado={ocupado} etiqueta={t('restauracion.descartar_persona').replace('{nombre}', c.propuesto?.nombre ?? c.anterior?.nombre ?? '')}>{t('restauracion.descartar')}</Boton></div></>}
   </section>)}
   <p>{t('restauracion.abrir_consecuencia')}</p>
   {!confirmacion ? <Boton id="terminar-restauracion" accion={() => { if (pendientes) { ponerError('restauracion.cambios_pendientes'); requestAnimationFrame(() => errorRef.current?.focus()); } else confirmar(true); }} ocupado={ocupado}>{t('restauracion.revisar_confirmacion')}</Boton>
    : <section aria-labelledby="confirmar-restauracion"><h2 ref={tituloConfirmacion} tabIndex={-1} id="confirmar-restauracion">{t('restauracion.confirmacion')}</h2><p>{t('restauracion.confirmacion_detalle')}</p>
     <Boton accion={() => { void decidir(); }} ocupado={ocupado}>{t('restauracion.confirmar')}</Boton><Boton accion={() => { confirmar(false); requestAnimationFrame(() => document.getElementById('terminar-restauracion')?.focus()); }} ocupado={ocupado}>{t('restauracion.cancelar')}</Boton></section>}
  </>}
 </section>;
}
