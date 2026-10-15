import { useEffect, useRef, useState } from 'react';
import { Boton } from './componentes';
import { desdeActas, type LecturaActa } from './actas-modelo';
import type { Formulario, Opciones } from './modelo';

export function Actas({ equipo, opciones, csrf, t, disabled, aplicar, ocupado, hayDatos }: {
  equipo: string; opciones: Opciones; csrf: string; t: (k: string) => string; disabled: boolean;
  aplicar: (f: Formulario) => void; ocupado: (v: boolean) => void;
  hayDatos: boolean;
}) {
  const [aceptacion, ponerAceptacion] = useState<File>();
  const [cierre, ponerCierre] = useState<File>();
  const [leyendo, ponerLeyendo] = useState(false);
  const [mensaje, ponerMensaje] = useState('');
  const [error, ponerError] = useState('');
  const [fechaFirma, ponerFechaFirma] = useState(false);
  const [confirmacion, ponerConfirmacion] = useState(false);
  const peticion = useRef<AbortController | undefined>(undefined);
  const resumen = useRef<HTMLDivElement>(null);
  const entradaAceptacion = useRef<HTMLInputElement>(null);
  const entradaCierre = useRef<HTMLInputElement>(null);
  useEffect(() => () => { peticion.current?.abort(); ocupado(false); }, [ocupado]);
  useEffect(() => { peticion.current?.abort(); ponerLeyendo(false); ocupado(false); ponerMensaje(''); ponerError(''); ponerFechaFirma(false); }, [equipo, ocupado]);
  useEffect(() => { if (error) resumen.current?.focus(); }, [error]);
  async function leer() {
    if (!equipo || !aceptacion) { ponerError(!equipo ? 'carga.actas.elegir_equipo' : 'carga.actas.elegir_aceptacion'); return; }
    const controller = new AbortController(); peticion.current = controller;
    ponerLeyendo(true); ocupado(true); ponerError(''); ponerFechaFirma(false); ponerMensaje('carga.actas.antivirus');
    try {
      async function subir(file: File, tipo: string): Promise<LecturaActa> {
        if (!/\.(pdf|docx|odt)$/i.test(file.name) || file.size > 10000000 || !file.size) throw new Error('carga.actas.fichero_invalido');
        const datos = new FormData(); datos.append('anexo', file);
        const r = await fetch(`/api/carga-2026/actas/leer/${tipo}`, { method: 'POST', body: datos, headers: { 'X-CSRF-Token': csrf, 'X-Equipo-Carga': equipo }, signal: controller.signal });
        const resultado = await r.json();
        if (!r.ok) throw new Error(typeof resultado.error === 'string' ? resultado.error : resultado.error?.clave ?? 'carga.error');
        return resultado as LecturaActa;
      }
      const a = await subir(aceptacion, 'aceptacion');
      const c = cierre ? await subir(cierre, 'cierre') : undefined;
      if (controller.signal.aborted) return;
      aplicar(desdeActas(equipo, opciones, a, c));
      ponerFechaFirma([a,c].some(x => x?.contenido.campos.some(c => c.motivos?.includes('carga.actas.fecha_firma'))));
      ponerMensaje([a, c].some(x => x?.antivirus === 'no_revisado') ? 'carga.actas.no_revisado' : !a.contenido.plantilla_reconocida || c && !c.contenido.plantilla_reconocida ? 'carga.actas.sin_lectura' : 'carga.actas.revisar');
      ponerAceptacion(undefined); ponerCierre(undefined);
      if (entradaAceptacion.current) entradaAceptacion.current.value = '';
      if (entradaCierre.current) entradaCierre.current.value = '';
    } catch (e) {
      if (!controller.signal.aborted) { ponerMensaje(''); ponerError(e instanceof Error ? e.message : 'carga.error'); }
    } finally { if (!controller.signal.aborted) { ponerLeyendo(false); ocupado(false); } }
  }
  const textoError = t(error) === error ? t('carga.error') : t(error);
  return <section className="carga-tarjeta" aria-labelledby="carga-actas-titulo">
    <h2 id="carga-actas-titulo">{t('carga.actas.titulo')}</h2>
    <p>{t('carga.actas.contexto')}</p>
    <div className="carga-rejilla">
      <div className="carga-campo"><label htmlFor="carga-acta-aceptacion">{t('carga.actas.aceptacion')}</label><input ref={entradaAceptacion} id="carga-acta-aceptacion" type="file" accept=".pdf,.docx,.odt,application/vnd.oasis.opendocument.text" disabled={disabled || leyendo} aria-describedby="carga-actas-limite" onChange={e => { ponerAceptacion(e.target.files?.[0]); ponerError(''); ponerMensaje(''); }} /></div>
      <div className="carga-campo"><label htmlFor="carga-acta-cierre">{t('carga.actas.cierre')}</label><input ref={entradaCierre} id="carga-acta-cierre" type="file" accept=".pdf,.docx,.odt,application/vnd.oasis.opendocument.text" disabled={disabled || leyendo} aria-describedby="carga-actas-limite" onChange={e => { ponerCierre(e.target.files?.[0]); ponerError(''); ponerMensaje(''); }} /></div>
    </div>
    <p id="carga-actas-limite">{t('carga.actas.limite')}</p>
    {error && <div ref={resumen} tabIndex={-1} className="carga-aviso" role="alert"><p>{textoError}</p></div>}
    {mensaje && <p className="carga-info" role="status">{t(mensaje)}</p>}
    {fechaFirma && <p className="carga-aviso">{t('carga.actas.fecha_firma')}</p>}
    <Boton disabled={disabled || leyendo} accion={() => { if (hayDatos && aceptacion && equipo) ponerConfirmacion(true); else void leer(); }}>{t('carga.actas.leer')}</Boton>
    {confirmacion && <ConfirmarLectura t={t} cancelar={() => ponerConfirmacion(false)} aceptar={() => { ponerConfirmacion(false); void leer(); }} />}
  </section>;
}

function ConfirmarLectura({t,cancelar,aceptar}:{t:(k:string)=>string;cancelar:()=>void;aceptar:()=>void}) {
  const dialogo = useRef<HTMLDialogElement>(null);
  const foco = useRef<HTMLButtonElement>(null);
  useEffect(() => { const antes=document.activeElement as HTMLElement|null; dialogo.current?.showModal(); foco.current?.focus(); return () => { dialogo.current?.close(); antes?.focus(); }; },[]);
  return <dialog ref={dialogo} className="carga" aria-labelledby="carga-actas-confirmar" onCancel={e => { e.preventDefault(); cancelar(); }}>
    <h2 id="carga-actas-confirmar">{t('carga.actas.sustituir_titulo')}</h2><p>{t('carga.actas.sustituir_consecuencia')}</p>
    <div className="carga-acciones"><button ref={foco} type="button" onClick={cancelar}>{t('carga.actas.conservar')}</button><Boton principal accion={aceptar}>{t('carga.actas.sustituir')}</Boton></div>
  </dialog>;
}
