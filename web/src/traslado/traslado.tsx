import { useEffect, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import type { Opciones as OpcionesDestino } from '../decision/api';
import { buscarPropuesta, type Propuesta, type SPM, permiteCierre } from '../salidas/api';
import { useCambios } from '../cambios';
import { leer, fallo, fechaValida, hoy, useOperacion } from './api';
import { Boton, Dialogo, Errores, ErrorLectura, ErrorOperacion } from './componentes';
import './traslado.css';
type Ficha = { id: string; numero: string; version: number; estado: string; equipo_id: string; aceptacion?: { fecha_acta: string }; traslado?: { destino_id: string; origen_id: string } };
type Datos = { ficha: Ficha; opciones: OpcionesDestino; propuesta: Propuesta | null; final: { motivo: string }; spm: SPM | null; usuario: string };
type Recibo = { destino_id: string; equipo_destino_id: string; version_destino: number; cierre: { version: number; aviso?: string } };
export function Traslado({ t, idioma, csrf, navegarRuta }: PropsPantalla) {
  const id = location.pathname.split('/')[3];
  const [datos, ponerDatos] = useState<Datos>(); const [cargando, ponerCargando] = useState(true); const [error, ponerError] = useState(''); const [intento, ponerIntento] = useState(0);
  const [destino, ponerDestino] = useState(''); const [motivo, ponerMotivo] = useState(''); const [fecha, ponerFecha] = useState(''); const [errores, ponerErrores] = useState<string[]>([]);
  const [confirmar, ponerConfirmar] = useState(false); const [ayuda, ponerAyuda] = useState(false); const [recibo, ponerRecibo] = useState<Recibo>();
  const { ponerSucio, solicitarSalida } = useCambios();
  useEffect(() => { ponerDestino(''); ponerMotivo(''); ponerFecha(''); ponerRecibo(undefined); ponerErrores([]); }, [id]);
  useEffect(() => {
    const c = new AbortController(); ponerCargando(true); ponerDatos(undefined); ponerError('');
    void Promise.all([leer<Ficha>(`/api/expedientes/${id}`, c.signal), leer<OpcionesDestino>(`/api/coordinacion/decision/opciones?idioma=${encodeURIComponent(idioma)}`, c.signal), buscarPropuesta(id, c.signal), leer<{ motivo: string }>(`/api/expedientes/${id}/finalizacion`, c.signal), leer<SPM | null>(`/api/expedientes/${id}/salida-spm`, c.signal), leer<{ usuario: { id: string } }>('/api/sesion', c.signal)])
      .then(([ficha, opciones, propuesta, final, spm, sesion]) => { if (!c.signal.aborted) ponerDatos({ ficha, opciones, propuesta, final, spm, usuario: sesion.usuario.id }); })
      .catch(e => { if (!c.signal.aborted) ponerError(fallo(e)); }).finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    return () => c.abort();
  }, [id, idioma, intento]);
  useEffect(() => { ponerSucio(!recibo && !!(destino || motivo || fecha)); }, [destino, motivo, fecha, recibo, ponerSucio]);
  useEffect(() => () => ponerSucio(false), [ponerSucio]);
  useEffect(() => { document.title = `${t('traslado.titulo')} ${datos?.ficha.numero ?? ''}`; }, [t, datos]);
  const op = useOperacion<Recibo>(id, csrf, r => !!r.destino_id && r.version_destino > 0 && r.cierre?.version > 0, r => { ponerRecibo(r); ponerSucio(false); });
  useEffect(() => { if (op.error === 'traslado.fecha_cierre_revisar') ponerErrores(['fecha_traslado']); }, [op.error]);
  const etiqueta = (clave: string) => datos?.opciones.textos[clave] || t('traslado.etiqueta_pendiente');
  const destinos = datos?.opciones.equipos.filter(e => e.id !== datos.ficha.equipo_id) ?? [];
  const faltaPropuesta = !datos?.propuesta || datos?.final?.motivo !== 'TRASLADO';
  const mismaPersona = datos?.propuesta?.propuesta_por === datos?.usuario;
  const spmBloqueado = !permiteCierre(datos?.spm ?? null) && !datos?.spm?.forzado_id;
  const bloqueado = cargando || op.bloqueado || !!error || !!recibo;
  const posible = !!datos && ['aceptada', 'en_intervencion', 'derivada_spm'].includes(datos.ficha.estado);
  function solicitar() {
    const campos = [...(!destinos.some(e => e.id === destino) ? ['destino'] : []), ...(!motivo.trim() ? ['motivo_traslado'] : []), ...(!fechaValida(fecha, datos?.ficha.aceptacion?.fecha_acta) ? ['fecha_traslado'] : [])];
    ponerErrores(campos); if (!campos.length) ponerConfirmar(true);
  }
  function enviar() { if (!datos || bloqueado) return; ponerConfirmar(false); void op.enviar({ ruta: `/api/expedientes/${id}/trasladar`, clave: crypto.randomUUID(), cuerpo: JSON.stringify({ version: datos.ficha.version, fecha, equipo_destino_id: destino, motivo_acta_texto: motivo.trim(), forzado_id: datos.spm?.forzado_id }) }); }
  function recargar() { solicitarSalida(() => { ponerDestino(''); ponerMotivo(''); ponerFecha(''); ponerErrores([]); op.ponerError(''); ponerIntento(n => n + 1); }); }
  const enlace = (ruta: string, clave: string) => <a href={ruta} onClick={e => { e.preventDefault(); navegarRuta(ruta); }}>{t(clave)}</a>;
  const invalido = (nombre: string) => ({ 'aria-invalid': errores.includes(nombre), 'aria-describedby': errores.includes(nombre) ? `traslado-error-${nombre}` : undefined });
  const avisoCampo = (nombre: string) => errores.includes(nombre) && <p className="pendiente" id={`traslado-error-${nombre}`}>{t(nombre === 'fecha_traslado' ? 'traslado.fecha_invalida' : 'traslado.obligatorio')}</p>;
  const etiquetaRuta = t('traslado.ruta');
  return <section className="traslado"><nav aria-label={etiquetaRuta}>{enlace(`/coordinacion/expedientes/${id}/salidas`, 'traslado.volver_salidas')}</nav>
    <header className="traslado-acciones"><h1>{t('traslado.titulo')} {datos?.ficha.numero}</h1><Boton etiqueta={t('traslado.ayuda_titulo')} accion={() => ponerAyuda(true)}>{t('traslado.ayuda')}</Boton></header>
    <p role="status">{cargando ? t('traslado.cargando') : op.ocupado ? t('traslado.guardando') : ''}</p><ErrorLectura error={error} recargar={recargar} navegarRuta={navegarRuta} t={t} /><ErrorOperacion {...op} recargar={recargar} navegarRuta={navegarRuta} t={t} />
    {recibo ? <section className="traslado-tarjeta" aria-labelledby="traslado-hecho"><h2 id="traslado-hecho">{t('traslado.hecho')}</h2><p role="status">{t('traslado.destino_pendiente')}</p>{recibo.cierre.aviso && <p className="pendiente">{t('traslado.sin_anexo')}</p>}<div className="traslado-acciones">{enlace(`/coordinacion/derivaciones/${recibo.destino_id}`, 'traslado.aceptar_destino')}{enlace(`/coordinacion/expedientes/${id}/salidas`, 'traslado.acta_origen')}</div></section> : !cargando && datos && <section className="traslado-tarjeta" aria-labelledby="traslado-preparar"><h2 id="traslado-preparar">{t('traslado.preparar')}</h2>
      {!posible ? <><p>{t('traslado.estado_invalido')}</p>{datos.ficha.traslado?.origen_id === id && enlace(`/coordinacion/derivaciones/${datos.ficha.traslado.destino_id}`, 'traslado.ver_destino')}</> : <><p>{t('traslado.consecuencia')}</p>
        {faltaPropuesta && <div className="pendiente"><p>{t('traslado.falta_propuesta')}</p>{enlace(`/equipo/expediente?id=${id}&pestana=cierre`, 'traslado.ver_propuesta')}</div>}
        {!faltaPropuesta && <p>{t('traslado.motivo_propuesto')} {etiqueta('motivo_finalizacion.TRASLADO')}</p>}
        {mismaPersona && <p className="pendiente">{t('traslado.segunda_persona')}</p>}
        {spmBloqueado && <div className="pendiente"><p>{t('traslado.bloqueo_spm')}</p>{enlace(`/coordinacion/expedientes/${id}/salidas`, 'traslado.revisar_spm')}</div>}
        {!destinos.length && <p>{t('traslado.sin_destinos')}</p>}
        <Errores campos={errores} t={t} /><form onSubmit={e => { e.preventDefault(); solicitar(); }}>
          <div className="traslado-campo"><label htmlFor="traslado-destino">{t('traslado.destino')}</label><select id="traslado-destino" {...invalido('destino')} value={destino} disabled={bloqueado} onChange={e => { ponerDestino(e.target.value); ponerErrores(cs => cs.filter(c => c !== 'destino')); }}><option value="">{t('traslado.elegir')}</option>{destinos.map(d => <option key={d.id} value={d.id}>{etiqueta(d.clave_texto)}</option>)}</select>{avisoCampo('destino')}</div>
          <div className="traslado-campo"><label htmlFor="traslado-motivo_traslado">{t('traslado.motivo_traslado')}</label><textarea id="traslado-motivo_traslado" {...invalido('motivo_traslado')} rows={3} maxLength={1000} value={motivo} disabled={bloqueado} onChange={e => { ponerMotivo(e.target.value); ponerErrores(cs => cs.filter(c => c !== 'motivo_traslado')); }} />{avisoCampo('motivo_traslado')}</div>
          <div className="traslado-campo"><label htmlFor="traslado-fecha_traslado">{t('traslado.fecha_traslado')}</label><input id="traslado-fecha_traslado" {...invalido('fecha_traslado')} type="date" min={datos.ficha.aceptacion?.fecha_acta} max={hoy()} value={fecha} disabled={bloqueado} onChange={e => { ponerFecha(e.target.value); if (op.error === 'traslado.fecha_cierre_revisar') op.ponerError(''); ponerErrores(cs => cs.filter(c => c !== 'fecha_traslado')); }} />{avisoCampo('fecha_traslado')}</div>
          <Boton principal disabled={bloqueado || faltaPropuesta || mismaPersona || spmBloqueado || !destinos.length} accion={solicitar}>{t('traslado.trasladar')}</Boton>
        </form></>}
    </section>}
    {confirmar && <Dialogo titulo={t('traslado.trasladar')} cerrar={() => ponerConfirmar(false)}><p>{t('traslado.consecuencia')}</p><p>{etiqueta(destinos.find(d => d.id === destino)?.clave_texto ?? '')}</p><div className="traslado-acciones"><Boton accion={() => ponerConfirmar(false)}>{t('traslado.cancelar')}</Boton><Boton principal accion={enviar}>{t('traslado.trasladar')}</Boton></div></Dialogo>}
    {ayuda && <Dialogo titulo={t('traslado.ayuda_titulo')} cerrar={() => ponerAyuda(false)}><p>{t('traslado.ayuda_texto')}</p><Boton accion={() => ponerAyuda(false)}>{t('traslado.cerrar_ayuda')}</Boton></Dialogo>}
  </section>;
}
