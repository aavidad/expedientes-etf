import { useEffect, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import type { Ficha } from '../equipo/api';
import { useCambios } from '../cambios';
import { fallo, fecha, fechaValida, hoy, leer, useOperacion, type Seguimiento as DatosSeguimiento, type OpcionesSeguimiento } from './api';
import { Boton, Dialogo, Errores, ErrorLectura, ErrorOperacion } from './componentes';
import './traslado.css';
type Datos = { seguimiento: DatosSeguimiento | null; opciones: OpcionesSeguimiento; coordinacion: boolean };
export function Seguimiento({ id, ficha, alCambiar, t, idioma, csrf, navegarRuta }: PropsPantalla & { id: string; ficha: Ficha; alCambiar: () => void }) {
  const [avisos, ponerAvisos] = useState<{ expediente_id: string; vigente_desde: string; vigente_hasta: string }[]>([]); const [errorAvisos, ponerErrorAvisos] = useState(false);
  const [datos, ponerDatos] = useState<Datos>(); const [cargando, ponerCargando] = useState(true); const [error, ponerError] = useState(''); const [intento, ponerIntento] = useState(0);
  const [dia, ponerDia] = useState(''); const [motivo, ponerMotivo] = useState(''); const [errores, ponerErrores] = useState<string[]>([]); const [confirmar, ponerConfirmar] = useState(false); const [ayuda, ponerAyuda] = useState(false); const [resultado, ponerResultado] = useState('');
  const { ponerSucio, solicitarSalida } = useCambios();
  useEffect(() => { if (location.pathname === '/equipo/seguimiento') { document.getElementById('equipo-panel-seguimiento')?.scrollIntoView({ block: 'start' }); document.getElementById('traslado-seguimiento-titulo')?.focus({ preventScroll: true }); } }, [id]);
  useEffect(() => { ponerDia(''); ponerMotivo(''); ponerErrores([]); ponerResultado(''); }, [id]);
  useEffect(() => {
    const c = new AbortController(); ponerCargando(true); ponerDatos(undefined); ponerError('');
    void Promise.all([leer<DatosSeguimiento | null>(`/api/expedientes/${id}/seguimiento`, c.signal), leer<OpcionesSeguimiento>(`/api/seguimiento/opciones?idioma=${encodeURIComponent(idioma)}`, c.signal), leer<{ perfiles: string[] }>('/api/sesion', c.signal)])
      .then(([seguimiento, opciones, sesion]) => { if (!c.signal.aborted) ponerDatos({ seguimiento, opciones, coordinacion: sesion.perfiles.includes('coordinacion') }); })
      .catch(e => { if (!c.signal.aborted) ponerError(fallo(e)); }).finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    ponerErrorAvisos(false); ponerAvisos([]);
    void leer<typeof avisos>('/api/avisos', c.signal).then(a => { if (!c.signal.aborted) ponerAvisos(a); }).catch(() => { if (!c.signal.aborted) ponerErrorAvisos(true); });
    return () => c.abort();
  }, [id, idioma, intento]);
  useEffect(() => { ponerSucio(!!(dia || motivo)); }, [dia, motivo, ponerSucio]);
  useEffect(() => () => ponerSucio(false), [ponerSucio]);
  const op = useOperacion<DatosSeguimiento>(id, csrf, r => !!r.id && Number.isSafeInteger(r.version) && r.version > 0, r => { ponerDatos(d => d ? { ...d, seguimiento: r } : d); ponerDia(''); ponerMotivo(''); ponerErrores([]); ponerSucio(false); ponerResultado('traslado.seguimiento_guardado'); alCambiar(); });
  const s = datos?.seguimiento;
  const accion = !s ? 'iniciar' : s.estado === 'activo' ? 'terminar' : 'reabrir';
  const campoFecha = accion === 'iniciar' ? 'fecha_cese' : 'fecha_terminacion';
  const campoMotivo = accion === 'reabrir' ? 'motivo_reapertura' : 'motivo_fin';
  const bloqueado = cargando || !!error || op.bloqueado;
  const vencido = !!s && s.estado === 'activo' && s.fecha_fin_prevista <= hoy();
  const puede = !!datos && (!s ? ficha.estado === 'cerrada' && ficha.subprograma === 'REUNIFICACION' : s.estado === 'activo' ? !vencido : datos.coordinacion);
  const etiqueta = (clave: string) => datos?.opciones.textos[clave] || t('traslado.etiqueta_pendiente');
  const aviso = avisos.find(a => a.expediente_id === id && s?.estado === 'activo' && a.vigente_desde <= hoy() && hoy() < a.vigente_hasta);
  function solicitar() {
    const campos = [...(accion !== 'reabrir' && (!fechaValida(dia, accion === 'terminar' ? s?.tramo_desde : '') || (accion === 'terminar' && dia >= (s?.fecha_fin_prevista ?? ''))) ? [campoFecha] : []), ...(accion === 'terminar' && !datos?.opciones.motivos.some(m => m.codigo === motivo) || accion === 'reabrir' && !motivo.trim() ? [campoMotivo] : [])];
    ponerErrores(campos); if (!campos.length) ponerConfirmar(true);
  }
  function enviar() {
    if (!datos || bloqueado || !puede) return; ponerConfirmar(false);
    const cuerpo = { version: s?.version ?? ficha.version, fecha: accion === 'reabrir' ? '' : dia, ...(accion !== 'iniciar' ? { motivo: motivo.trim() } : {}) };
    void op.enviar({ ruta: `/api/expedientes/${id}/seguimiento${accion === 'iniciar' ? '' : `/${accion}`}`, cuerpo: JSON.stringify(cuerpo), clave: crypto.randomUUID() });
  }
  function recargar() { solicitarSalida(() => { ponerDia(''); ponerMotivo(''); ponerErrores([]); op.ponerError(''); ponerIntento(n => n + 1); }); }
  const invalido = (nombre: string) => ({ 'aria-invalid': errores.includes(nombre), 'aria-describedby': errores.includes(nombre) ? `traslado-error-${nombre}` : undefined });
  const avisoCampo = (nombre: string) => errores.includes(nombre) && <p className="pendiente" id={`traslado-error-${nombre}`}>{t(nombre === campoFecha ? 'traslado.fecha_invalida' : 'traslado.obligatorio')}</p>;
  return <section className="traslado" aria-labelledby="traslado-seguimiento-titulo"><header className="traslado-acciones"><h2 id="traslado-seguimiento-titulo" tabIndex={-1}>{t('traslado.seguimiento')}</h2><Boton etiqueta={t('traslado.ayuda_seguimiento_titulo')} accion={() => ponerAyuda(true)}>{t('traslado.ayuda')}</Boton></header>
    <p>{t('equipo.expediente')} {ficha.numero}</p>
    <p role="status">{cargando ? t('traslado.cargando') : op.ocupado ? t('traslado.guardando') : resultado ? t(resultado) : ''}</p><ErrorLectura error={error} recargar={recargar} navegarRuta={navegarRuta} t={t} /><ErrorOperacion {...op} recargar={recargar} navegarRuta={navegarRuta} t={t} />
    {errorAvisos && <p role="status">{t('traslado.avisos_no_disponibles')} <Boton accion={recargar}>{t('traslado.actualizar')}</Boton></p>}
    {!cargando && datos && <>{s ? <><p className="pildora">{t(s.estado === 'activo' ? 'traslado.seguimiento_activo' : 'traslado.seguimiento_terminado')}</p><dl><dt>{t('traslado.fecha_cese')}</dt><dd>{fecha(s.fecha_cese_medida, idioma)}</dd><dt>{t('traslado.plazo_desde')}</dt><dd>{fecha(s.tramo_desde, idioma)}</dd><dt>{t('traslado.fin_previsto')}</dt><dd>{fecha(s.fecha_fin_prevista, idioma)}</dd>{s.fecha_terminacion && <><dt>{t('traslado.fecha_terminacion')}</dt><dd>{fecha(s.fecha_terminacion, idioma)}</dd></>}{s.motivo && <><dt>{t('traslado.motivo_fin')}</dt><dd>{etiqueta(`fin_seguimiento.${s.motivo}`)}</dd></>}{s.motivo_reapertura && <><dt>{t('traslado.motivo_reapertura')}</dt><dd>{s.motivo_reapertura}</dd></>}</dl>{aviso && <p className="pendiente" role="status">{t('traslado.fin_proximo')} {fecha(s.fecha_fin_prevista, idioma)}</p>}{vencido && <p role="status">{t('traslado.vencimiento_pendiente')} <Boton accion={recargar}>{t('traslado.actualizar')}</Boton></p>}</> : <p>{t('traslado.sin_seguimiento')}</p>}
      {!s && !puede && <p>{t('traslado.solo_reunificacion')}</p>}
      {s?.estado === 'terminado' && !datos.coordinacion && <p>{t('traslado.reabre_coordinacion')}</p>}
      {puede && <><Errores campos={errores} t={t} /><form onSubmit={e => { e.preventDefault(); solicitar(); }}>
        <h3>{t(`traslado.accion.${accion}`)}</h3>
        {accion !== 'reabrir' && <div className="traslado-campo"><label htmlFor={`traslado-${campoFecha}`}>{t(`traslado.${campoFecha}`)}</label><input id={`traslado-${campoFecha}`} {...invalido(campoFecha)} type="date" min={accion === 'terminar' ? s?.tramo_desde : undefined} max={hoy()} disabled={bloqueado} value={dia} onChange={e => { ponerDia(e.target.value); ponerResultado(''); ponerErrores(cs => cs.filter(c => c !== campoFecha)); }} />{avisoCampo(campoFecha)}</div>}
        {accion === 'terminar' && <div className="traslado-campo"><label htmlFor="traslado-motivo_fin">{t('traslado.motivo_fin')}</label><select id="traslado-motivo_fin" {...invalido(campoMotivo)} value={motivo} disabled={bloqueado} onChange={e => { ponerMotivo(e.target.value); ponerResultado(''); ponerErrores(cs => cs.filter(c => c !== campoMotivo)); }}><option value="">{t('traslado.elegir')}</option>{datos.opciones.motivos.map(m => <option key={m.codigo} value={m.codigo}>{etiqueta(m.clave_texto)}</option>)}</select>{avisoCampo(campoMotivo)}{!datos.opciones.motivos.length && <p>{t('traslado.sin_motivos')}</p>}</div>}
        {accion === 'reabrir' && <div className="traslado-campo"><label htmlFor="traslado-motivo_reapertura">{t('traslado.motivo_reapertura')}</label><textarea id="traslado-motivo_reapertura" {...invalido(campoMotivo)} rows={3} value={motivo} maxLength={1000} disabled={bloqueado} onChange={e => { ponerMotivo(e.target.value); ponerResultado(''); ponerErrores(cs => cs.filter(c => c !== campoMotivo)); }} />{avisoCampo(campoMotivo)}</div>}
        <p>{t(`traslado.consecuencia.${accion}`)}</p><Boton principal disabled={bloqueado || accion === 'terminar' && !datos.opciones.motivos.length} accion={solicitar}>{t(`traslado.accion.${accion}`)}</Boton>
      </form></>}
      <a href={`/equipo/expediente?id=${id}&pestana=historial`} onClick={e => { e.preventDefault(); navegarRuta(`/equipo/expediente?id=${id}&pestana=historial`); }}>{t('traslado.ver_historia')}</a>
    </>}
    {confirmar && <Dialogo titulo={t(`traslado.accion.${accion}`)} cerrar={() => ponerConfirmar(false)}><p>{t(`traslado.consecuencia.${accion}`)}</p><div className="traslado-acciones"><Boton accion={() => ponerConfirmar(false)}>{t('traslado.cancelar')}</Boton><Boton principal accion={enviar}>{t(`traslado.accion.${accion}`)}</Boton></div></Dialogo>}
    {ayuda && <Dialogo titulo={t('traslado.ayuda_seguimiento_titulo')} cerrar={() => ponerAyuda(false)}><p>{t('traslado.ayuda_seguimiento')}</p><Boton accion={() => ponerAyuda(false)}>{t('traslado.cerrar_ayuda')}</Boton></Dialogo>}
  </section>;
}
