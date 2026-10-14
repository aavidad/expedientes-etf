import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import type { PropsPantalla } from '../registro-pantallas';
import { useCambios } from '../cambios';
import { useLectura, useEtiquetas, type Ficha } from './api';
import { EstadoLectura } from './equipo';
import { useEdicion } from './edicion';

type Finalizacion = {
  version: number; motivo: string; motivo_texto: string; sin_consentimiento: boolean;
  derivacion_tras_cierre: string; coordinacion_recursos: string; objetivos_abordados: string; objetivos_pendientes: string;
};
const campos = ['motivo_texto', 'derivacion_tras_cierre', 'coordinacion_recursos', 'objetivos_abordados', 'objetivos_pendientes'] as const;

// F3.02a valida, versiona y audita los cambios. Esta pantalla usa sus API.
export function CierreEquipo({ id, ficha, t, csrf, idioma, alCambiar }: PropsPantalla & { id: string; ficha: Ficha; alCambiar: () => void }) {
  const catalogo = useEtiquetas(idioma);
  const documentos = useEdicion(id, csrf);
  const sesion = useLectura<{ perfiles: string[] }>('/api/sesion');
  const etiqueta = (clave: string) => catalogo.datos?.[clave] || t('equipo.sin_dato');
  const lectura = useLectura<Finalizacion>(`/api/expedientes/${encodeURIComponent(id)}/finalizacion`);
  const [borrador, ponerBorrador] = useState<Finalizacion>();
  const [sucio, ponerEditado] = useState(false);
  const [ocupado, ponerOcupado] = useState(false);
  const [esperandoVersion, ponerEsperandoVersion] = useState(false);
  const [error, ponerError] = useState('');
  const [resultado, ponerResultado] = useState('');
  const { ponerSucio, solicitarSalida } = useCambios();
  const peticiones = useRef(new Map<string, string>());
  const abort = useRef<AbortController | null>(null);
  const guardarRef = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ type: 'submit', isDisabled: ocupado || documentos.ocupado || esperandoVersion || lectura.cargando }, guardarRef);
  const pendiente = ficha.avisos.includes('equipo.aviso.cierre_pendiente');
  useEffect(() => { ponerBorrador(lectura.datos); ponerEditado(false); ponerSucio(false); if (lectura.datos) ponerEsperandoVersion(false); }, [lectura.datos, ponerSucio]);
  useEffect(() => { abort.current = new AbortController(); return () => { abort.current?.abort(); ponerSucio(false); }; }, [ponerSucio]);
  const editar = (cambio: Partial<Finalizacion>) => { ponerBorrador((b) => b ? { ...b, ...cambio } : b); ponerEditado(true); ponerSucio(true); ponerResultado(''); };
  async function cambiar(operacion: 'finalizacion' | 'proponer-cierre' | 'retirar-propuesta-cierre') {
    if (!borrador || ocupado || documentos.ocupado || esperandoVersion || lectura.cargando || !abort.current) return;
    ponerError(''); ponerResultado('');
    if (operacion === 'proponer-cierre' && (sucio || !borrador.motivo)) { ponerError('equipo.cierre.guarda_antes'); return; }
    ponerOcupado(true);
    const cuerpo = JSON.stringify(operacion === 'finalizacion' ? {
      version: borrador.version, motivo: borrador.motivo, motivo_texto: borrador.motivo_texto ?? '', sin_consentimiento: borrador.sin_consentimiento,
      derivacion_tras_cierre: borrador.derivacion_tras_cierre ?? '', coordinacion_recursos: borrador.coordinacion_recursos ?? '', objetivos_abordados: borrador.objetivos_abordados ?? '', objetivos_pendientes: borrador.objetivos_pendientes ?? '',
    } : { version: borrador.version });
    const huella = operacion + cuerpo;
    const clave = peticiones.current.get(huella) ?? crypto.randomUUID(); peticiones.current.set(huella, clave);
    try {
      const respuesta = await fetch(`/api/expedientes/${encodeURIComponent(id)}/${operacion}`, { method: operacion === 'finalizacion' ? 'PATCH' : 'POST', signal: abort.current.signal,
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': clave }, body: cuerpo });
      const datos = await respuesta.json() as { version?: number; error?: { clave?: string } };
      if (!respuesta.ok) throw new Error(respuesta.status === 409 ? 'equipo.cierre.conflicto' : datos.error?.clave ?? 'equipo.error');
      if (!Number.isInteger(datos.version) || (datos.version ?? 0) < 1) throw new Error('equipo.error');
      ponerBorrador((b) => b ? { ...b, version: datos.version! } : b); ponerEsperandoVersion(true);
      peticiones.current.delete(huella); ponerEditado(false); ponerSucio(false); ponerResultado('equipo.cierre.guardado'); lectura.recargar(); alCambiar();
    } catch (e) { if (!abort.current.signal.aborted) ponerError(e instanceof Error ? e.message : 'equipo.error'); }
    finally { if (!abort.current.signal.aborted) ponerOcupado(false); }
  }
  async function descargar(anexo: 'anexo11' | 'anexo12') {
    if (!borrador || ocupado || documentos.ocupado || esperandoVersion || lectura.cargando || sucio) return;
    if (await documentos.enviar(`/api/expedientes/${encodeURIComponent(id)}/documentos/${anexo}`, { idioma, version: borrador.version }, true)) alCambiar();
  }
  return <section aria-labelledby="equipo-cierre-titulo">
    <h2 id="equipo-cierre-titulo">{t('equipo.pestana.cierre')}</h2>
    <EstadoLectura {...lectura} t={t} /><EstadoLectura {...catalogo} t={t} />
    {error && <div role="alert"><p>{t(error) || t('equipo.error')}</p>{error === 'equipo.cierre.conflicto' && <button type="button" onClick={() => solicitarSalida(() => { ponerError(''); lectura.recargar(); })}>{t('equipo.cierre.ver_cambios')}</button>}</div>}
    {resultado && <p role="status">{t(resultado)}</p>}
    {borrador && <>
      <p role="status">{t(pendiente ? 'equipo.cierre.propuesto' : 'equipo.cierre.sin_propuesta')}</p>
      {ficha.puede_proponer && !pendiente ? <form onSubmit={(e) => { e.preventDefault(); void cambiar('finalizacion'); }}>
        <label className="equipo-campo" htmlFor="equipo-motivo">{t('equipo.cierre.motivo')}<select id="equipo-motivo" required value={borrador.motivo ?? ''} onChange={(e) => editar({ motivo: e.target.value, motivo_texto: '', sin_consentimiento: false })}>
          <option value="">{t('equipo.cierre.elige_motivo')}</option>{ficha.motivos_cierre.filter((m) => ficha.estado !== 'aceptada' || m.sin_consentimiento).map((m) => <option key={m.codigo} value={m.codigo}>{etiqueta(`motivo_finalizacion.${m.codigo}`)}</option>)}
        </select></label>
        {ficha.motivos_cierre.find((m) => m.codigo === borrador.motivo)?.sin_consentimiento && <label className="equipo-casilla"><input type="checkbox" checked={borrador.sin_consentimiento} onChange={(e) => editar({ sin_consentimiento: e.target.checked })} />{t('equipo.cierre.sin_consentimiento')}</label>}
        {campos.filter((c) => c !== 'motivo_texto' || ficha.motivos_cierre.find((m) => m.codigo === borrador.motivo)?.admite_texto).map((c) => <label className="equipo-campo" htmlFor={`equipo-cierre-${c}`} key={c}>{t(`equipo.cierre.${c}`)}<textarea id={`equipo-cierre-${c}`} maxLength={20000} value={borrador[c] ?? ''} onChange={(e) => editar({ [c]: e.target.value })} /></label>)}
        <div className="equipo-resumen"><button {...buttonProps} ref={guardarRef} className="principal">{t(ocupado ? 'equipo.cierre.guardando' : 'equipo.cierre.guardar')}</button><button type="button" disabled={ocupado || documentos.ocupado || esperandoVersion || lectura.cargando} onClick={() => void cambiar('proponer-cierre')}>{t('equipo.cierre.proponer')}</button></div>
      </form> : <dl className="equipo-detalle">
        <dt>{t('equipo.cierre.motivo')}</dt><dd>{borrador.motivo ? etiqueta(`motivo_finalizacion.${borrador.motivo}`) : t('equipo.sin_dato')}</dd>
        {campos.map((c) => <div key={c}><dt>{t(`equipo.cierre.${c}`)}</dt><dd>{borrador[c] || t('equipo.sin_dato')}</dd></div>)}
      </dl>}
      {sesion.datos?.perfiles.includes('profesional_etf') && ['aceptada', 'en_intervencion', 'derivada_spm'].includes(ficha.estado) && <section className="equipo-tarjeta" aria-labelledby="gap1-anexos"><h3 id="gap1-anexos">{t('gap1.anexos')}</h3><p>{t('gap1.anexos_aviso')}</p>
        {sucio && <p role="status">{t('gap1.descarga_guardar')}</p>}
        <div className="fi1-acciones">{(ficha.subprograma === 'REUNIFICACION' ? ['anexo12'] as const : ficha.subprograma === 'RIESGO_SOCIAL' ? ['anexo11'] as const : []).map((anexo) => <button key={anexo} type="button" disabled={ocupado || documentos.ocupado || documentos.conflicto || esperandoVersion || lectura.cargando || sucio} onClick={() => void descargar(anexo)}>{t(documentos.ocupado ? 'fi2.espera' : 'gap1.' + anexo)}</button>)}</div>
        {documentos.error && <p role="alert">{t(documentos.error) !== documentos.error ? t(documentos.error) : t('gap1.descarga_error')}</p>}
        {documentos.conflicto && <button type="button" onClick={() => documentos.verCambios(() => { lectura.recargar(); alCambiar(); })}>{t('fi2.ver_cambios')}</button>}
        {documentos.resultado && <p role="status">{t(documentos.resultado)}</p>}
      </section>}
      {ficha.puede_proponer && pendiente && <button type="button" disabled={ocupado || documentos.ocupado || esperandoVersion || lectura.cargando} onClick={() => void cambiar('retirar-propuesta-cierre')}>{t('equipo.cierre.retirar')}</button>}
    </>}
  </section>;
}
