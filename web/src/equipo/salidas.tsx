import { useEffect, useRef, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useCambios } from '../cambios';
import { useLectura, useEtiquetas, fecha, type Ficha } from './api';
import { EstadoLectura } from './equipo';

type Salida = { version: number; motivo: string; propuesta_medida: string; fecha_derivacion?: string; respuesta: string; fecha_respuesta?: string };

// F3.01a conserva la derivación, sus fechas y respuesta; aquí se consumen sus API.
export function SalidasEquipo({ t, idioma, csrf, id, ficha, alCambiar }: PropsPantalla & { id: string; ficha: Ficha; alCambiar: () => void }) {
  const catalogo = useEtiquetas(idioma);
  const etiqueta = (codigo: string) => catalogo.datos?.[`servicio_destino.${codigo}`] || t('equipo.sin_dato');
  const lectura = useLectura<Salida | null>(`/api/expedientes/${encodeURIComponent(id)}/salida-spm`);
  const [motivo, ponerMotivo] = useState('');
  const [medida, ponerMedida] = useState('');
  const [dia, ponerDia] = useState('');
  const [respuesta, ponerRespuesta] = useState('a_la_espera');
  const [error, ponerError] = useState('');
  const [ocupado, ponerOcupado] = useState(false);
  const [esperandoVersion, ponerEsperandoVersion] = useState(false);
  const [mensaje, ponerMensaje] = useState('');
  const { ponerSucio, solicitarSalida } = useCambios();
  const abort = useRef<AbortController | null>(null);
  const claves = useRef(new Map<string, string>());
  useEffect(() => { abort.current = new AbortController(); return () => { abort.current?.abort(); ponerSucio(false); }; }, [ponerSucio]);
  const salida = lectura.datos;
  useEffect(() => { if (lectura.datos !== undefined) { ponerEsperandoVersion(false); ponerDia(''); if (lectura.datos) ponerRespuesta(lectura.datos.respuesta); } }, [lectura.datos]);
  async function guardar(accion: 'crear' | 'confirmar' | 'responder') {
    if (ocupado || esperandoVersion || lectura.cargando || !abort.current) return;
    ponerError(''); ponerOcupado(true);
    const cuerpo = JSON.stringify(accion === 'crear' ? { version: ficha.version, motivo: motivo.trim(), propuesta_medida: medida.trim() } : { version: salida?.version, fecha: accion === 'responder' && respuesta === 'a_la_espera' ? '' : dia, ...(accion === 'responder' ? { respuesta } : {}) });
    const ruta = `/api/expedientes/${encodeURIComponent(id)}/salida-spm${accion === 'confirmar' ? '/fecha' : accion === 'responder' ? '/respuesta' : ''}`;
    const huella = accion + cuerpo;
    const clave = claves.current.get(huella) ?? crypto.randomUUID(); claves.current.set(huella, clave);
    try {
      const r = await fetch(ruta, { method: accion === 'responder' ? 'PATCH' : 'POST', signal: abort.current.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': clave }, body: cuerpo });
      const datos = await r.json() as { version?: number; error?: { clave?: string } };
      if (!r.ok) throw new Error(r.status === 409 ? 'equipo.cierre.conflicto' : datos.error?.clave ?? 'equipo.error');
      if (!Number.isInteger(datos.version) || (datos.version ?? 0) < 1) throw new Error('equipo.error');
      ponerEsperandoVersion(true); ponerDia(''); claves.current.delete(huella); ponerSucio(false); lectura.recargar(); alCambiar();
    } catch (e) { if (!abort.current.signal.aborted) ponerError(e instanceof Error ? e.message : 'equipo.error'); }
    finally { if (!abort.current.signal.aborted) ponerOcupado(false); }
  }
  async function descargar() {
    if (!salida || ocupado || esperandoVersion || !abort.current) return;
    ponerError(''); ponerMensaje(''); ponerOcupado(true);
    const cuerpo = JSON.stringify({ tipo: 'salida/spm/anexo5', expediente_id: id, idioma, version: salida.version });
    const clave = claves.current.get(cuerpo) ?? crypto.randomUUID(); claves.current.set(cuerpo, clave);
    try {
      const r = await fetch('/api/documentos/emisiones', { method: 'POST', signal: abort.current.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': clave }, body: cuerpo });
      if (!r.ok) { const datos = await r.json() as { error?: { clave?: string } }; throw new Error(datos.error?.clave ?? 'equipo.error'); }
      if (!r.headers.get('content-type')?.includes('wordprocessingml') || !r.headers.get('X-Emision-ID')) throw new Error('equipo.error');
      const documento = await r.blob();
      const enlace = document.createElement('a'); const direccion = URL.createObjectURL(documento);
      enlace.href = direccion; enlace.download = `${ficha.numero}-${t('equipo.spm.archivo')}`; document.body.append(enlace); enlace.click(); enlace.remove();
      setTimeout(() => URL.revokeObjectURL(direccion), 1000);
      claves.current.delete(cuerpo); ponerMensaje('equipo.spm.descargado');
    } catch (e) { if (!abort.current.signal.aborted) ponerError(e instanceof Error ? e.message : 'equipo.error'); }
    finally { if (!abort.current.signal.aborted) ponerOcupado(false); }
  }
  const editarFecha = (valor: string) => { ponerDia(valor); ponerSucio(true); };
  return <section aria-labelledby="equipo-salida-titulo">
    <section aria-labelledby="equipo-otras-salidas"><h2 id="equipo-otras-salidas">{t('equipo.otras_salidas')}</h2><EstadoLectura {...catalogo} t={t} />
      {ficha.salidas_servicio.length ? <ul className="equipo-sucesos">{ficha.salidas_servicio.map((s) => <li key={s.id}><strong>{etiqueta(s.servicio_destino)}</strong><span>{s.motivo}</span><span>{s.fecha ? fecha(s.fecha, idioma) : t('equipo.spm.sin_fecha')}</span></li>)}</ul> : <p>{t('equipo.sin_otras_salidas')}</p>}
    </section>
    <h2 id="equipo-salida-titulo">{t('equipo.spm.titulo')}</h2>
    <EstadoLectura {...lectura} t={t} />{mensaje && <p role="status">{t(mensaje)}</p>}
    {error && <div role="alert"><p>{t(error) || t('equipo.error')}</p>{error === 'equipo.cierre.conflicto' && <button type="button" onClick={() => solicitarSalida(() => { ponerError(''); ponerSucio(false); lectura.recargar(); })}>{t('equipo.cierre.ver_cambios')}</button>}</div>}
    {!lectura.cargando && !lectura.error && salida === null && <>
      <p>{t('equipo.spm.sin_salida')}</p>
      {ficha.estado !== 'cerrada' && <form onSubmit={(e) => { e.preventDefault(); void guardar('crear'); }}>
        <label className="equipo-campo" htmlFor="equipo-spm-motivo">{t('equipo.spm.motivo')}<textarea id="equipo-spm-motivo" required maxLength={4000} value={motivo} onChange={(e) => { ponerMotivo(e.target.value); ponerSucio(true); }} /></label>
        <label className="equipo-campo" htmlFor="equipo-spm-medida">{t('equipo.spm.medida')}<textarea id="equipo-spm-medida" required maxLength={4000} value={medida} onChange={(e) => { ponerMedida(e.target.value); ponerSucio(true); }} /></label>
        <button className="principal" type="submit" disabled={ocupado || esperandoVersion || lectura.cargando}>{t('equipo.spm.preparar')}</button>
      </form>}
    </>}
    {salida && <>
      <button type="button" disabled={ocupado || esperandoVersion || lectura.cargando} onClick={() => void descargar()}>{t(ocupado ? 'equipo.spm.generando' : 'equipo.spm.generar')}</button>
      <dl className="equipo-detalle"><dt>{t('equipo.spm.motivo')}</dt><dd>{salida.motivo}</dd><dt>{t('equipo.spm.medida')}</dt><dd>{salida.propuesta_medida}</dd>
        <dt>{t('equipo.spm.fecha_derivacion')}</dt><dd>{salida.fecha_derivacion ? fecha(salida.fecha_derivacion, idioma) : t('equipo.spm.sin_fecha')}</dd>
        <dt>{t('equipo.spm.respuesta')}</dt><dd>{t(`equipo.spm.respuesta.${salida.respuesta}`) || t('equipo.sin_dato')}</dd>
        {salida.fecha_respuesta && <><dt>{t('equipo.spm.fecha_respuesta')}</dt><dd>{fecha(salida.fecha_respuesta, idioma)}</dd></>}
      </dl>
      {ficha.estado !== 'cerrada' && <form onSubmit={(e) => { e.preventDefault(); void guardar(salida.fecha_derivacion ? 'responder' : 'confirmar'); }}>
        {salida.fecha_derivacion && <label className="equipo-campo" htmlFor="equipo-spm-respuesta">{t('equipo.spm.respuesta')}<select id="equipo-spm-respuesta" value={respuesta} onChange={(e) => { ponerRespuesta(e.target.value); ponerDia(''); ponerSucio(true); }}>{['a_la_espera', 'inicio_desamparo_medida', 'medida_no_necesaria'].map((c) => <option key={c} value={c}>{t(`equipo.spm.respuesta.${c}`)}</option>)}</select></label>}
        {(!salida.fecha_derivacion || respuesta !== 'a_la_espera') && <label className="equipo-campo" htmlFor="equipo-spm-fecha">{t(salida.fecha_derivacion ? 'equipo.spm.fecha_respuesta' : 'equipo.spm.fecha_derivacion')}<input id="equipo-spm-fecha" type="date" required value={dia} onChange={(e) => editarFecha(e.target.value)} /></label>}
        <button className="principal" type="submit" disabled={ocupado || esperandoVersion || lectura.cargando}>{t(salida.fecha_derivacion ? 'equipo.spm.guardar_respuesta' : 'equipo.spm.confirmar_fecha')}</button>
      </form>}
    </>}
  </section>;
}
