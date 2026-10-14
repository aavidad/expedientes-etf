import { useEffect, useRef, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useCambios } from '../cambios';
import { buscarPropuesta, claveFallo, comprobar, Fallo, leer, permiteCierre, type Expediente, type Opciones, type SPM, type Propuesta } from './api';
import { Ayuda, Boton, Confirmar } from './componentes';
import './salidas.css';
type Finalizacion = { motivo: string; motivo_texto: string; motivo_acta: string; derivacion_tras_cierre: string; coordinacion_recursos: string };
type Operacion = { accion: string; ruta: string; metodo: string; cuerpo: string; clave: string };
export function Salidas({ t, idioma, csrf, navegarRuta }: PropsPantalla) {
  const id = location.pathname.split('/')[3];
  const [datos, ponerDatos] = useState<{ ficha: Expediente; spm: SPM | null; final: Finalizacion | null; opciones: Opciones; propuesta: Propuesta | null; usuario: string }>();
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState('');
  const [aviso, ponerAviso] = useState('');
  const [confirmar, ponerConfirmar] = useState('');
  const [ocupado, ponerOcupado] = useState(false);
  const [pendiente, ponerPendiente] = useState<Operacion>();
  const [intento, ponerIntento] = useState(0);
  const [respuesta, ponerRespuesta] = useState('');
  const [fechaRespuesta, ponerFechaRespuesta] = useState('');
  const [motivoForzado, ponerMotivoForzado] = useState('');
  const [fechaCierre, ponerFechaCierre] = useState('');
  const [motivoActa, ponerMotivoActa] = useState('');
  const [fechaReapertura, ponerFechaReapertura] = useState('');
  const [motivoReapertura, ponerMotivoReapertura] = useState('');
  const [errores, ponerErrores] = useState<string[]>([]);
  const resumen = useRef<HTMLElement>(null);
  const [ayuda, ponerAyuda] = useState(false);
  const controlador = useRef<AbortController | null>(null);
  const enviando = useRef(false);
  const { ponerSucio, solicitarSalida } = useCambios();
  useEffect(() => {
    const c = new AbortController(); controlador.current = c; ponerCargando(true); ponerDatos(undefined); ponerError('');
    void Promise.all([
      leer<Expediente>(`/api/expedientes/${id}`, c.signal), leer<SPM | null>(`/api/expedientes/${id}/salida-spm`, c.signal),
      leer<Finalizacion | null>(`/api/expedientes/${id}/finalizacion`, c.signal), leer<Opciones>(`/api/coordinacion/decision/opciones?idioma=${encodeURIComponent(idioma)}`, c.signal),
    ]).then(async ([ficha, spm, final, opciones]) => {
      const [propuesta, sesion] = await Promise.all([ficha.estado === 'cerrada' ? Promise.resolve(null) : buscarPropuesta(id, c.signal), leer<{ usuario: { id: string } }>('/api/sesion', c.signal)]);
      if (!c.signal.aborted) { ponerDatos({ ficha, spm, final: final?.motivo ? final : null, opciones, propuesta, usuario: sesion.usuario.id }); }
    }).catch((e: unknown) => { if (!c.signal.aborted) ponerError(claveFallo(e)); })
      .finally(() => { if (!c.signal.aborted) ponerCargando(false); });
    return () => { c.abort(); };
  }, [id, idioma, intento, ponerSucio]);
  useEffect(() => { ponerSucio(!!(respuesta || fechaRespuesta || motivoForzado || fechaCierre || motivoActa || fechaReapertura || motivoReapertura)); }, [respuesta, fechaRespuesta, motivoForzado, fechaCierre, motivoActa, fechaReapertura, motivoReapertura, ponerSucio]);
  useEffect(() => () => ponerSucio(false), [ponerSucio]);
  useEffect(() => { if (errores.length) resumen.current?.focus(); }, [errores]);
  function limpiar() { ponerRespuesta(''); ponerFechaRespuesta(''); ponerMotivoForzado(''); ponerFechaCierre(''); ponerMotivoActa(''); ponerFechaReapertura(''); ponerMotivoReapertura(''); ponerErrores([]); }
  const partesHoy = new Intl.DateTimeFormat(idioma, { year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date());
  const hoy = ['year', 'month', 'day'].map(tipo => partesHoy.find(p => p.type === tipo)?.value).join('-');
  function solicitar(accion: string) {
    const validos = (fecha: string, minimo = '') => /^\d{4}-\d{2}-\d{2}$/.test(fecha) && Number.isFinite(Date.parse(fecha)) && fecha <= hoy && fecha >= minimo;
    const campos = accion === 'respuesta' ? [...(!respuesta ? ['respuesta'] : []), ...(!validos(fechaRespuesta, datos?.spm?.fecha_derivacion) ? ['fecha_respuesta'] : [])]
      : accion === 'forzado' ? (!motivoForzado.trim() ? ['motivo_forzado'] : [])
      : accion === 'cerrar' ? (!validos(fechaCierre) ? ['fecha_cierre'] : [])
      : [...(!validos(fechaReapertura) ? ['fecha_reapertura'] : []), ...(!motivoReapertura.trim() ? ['motivo_reapertura'] : [])];
    ponerErrores(campos); if (!campos.length) ponerConfirmar(accion);
  }
  const editar = (accion: () => void) => { accion(); ponerSucio(true); };
  const formatoFecha = (valor: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${valor}T12:00:00Z`));
  const etiqueta = (clave: string) => datos?.opciones.textos[clave] || t('salidas.etiqueta_pendiente');
  const bloqueadoSPM = !permiteCierre(datos?.spm ?? null);
  const cerrada = datos?.ficha.estado === 'cerrada';
  const mismaPersona = !!datos?.propuesta && datos.propuesta.propuesta_por === datos.usuario;
  const bloqueado = cargando || ocupado || !!pendiente || error === 'salidas.conflicto' || error === 'salidas.sin_permiso';
  async function enviar(op: Operacion) {
    if (enviando.current) return;
    enviando.current = true; ponerOcupado(true); ponerError(''); ponerAviso('');
    const signal = controlador.current?.signal;
    try {
      const r = await fetch(op.ruta, { method: op.metodo, signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': op.clave }, body: op.cuerpo });
      await comprobar(r);
      if (op.accion === 'acta') {
        const contenido = await r.blob();
        if (!contenido.size) throw new Error('salidas.error');
        const url = URL.createObjectURL(contenido); const enlace = document.createElement('a'); enlace.href = url; enlace.download = `${t('salidas.acta_nombre')}.docx`; enlace.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
        ponerAviso('salidas.descargada');
      } else {
        const recibo = await r.json() as { aviso?: string; version?: number };
        if (!Number.isSafeInteger(recibo.version) || (recibo.version ?? 0) < 1) throw new Error('salidas.error');
        ponerAviso(recibo.aviso ? 'salidas.cierre_sin_anexo' : `salidas.hecho.${op.accion}`); if (op.accion === 'respuesta') { ponerRespuesta(''); ponerFechaRespuesta(''); }
        if (op.accion === 'forzado') ponerMotivoForzado('');
        if (op.accion === 'cerrar') { ponerFechaCierre(''); ponerMotivoActa(''); }
        if (op.accion === 'reabrir') { ponerFechaReapertura(''); ponerMotivoReapertura(''); }
        ponerErrores([]); ponerIntento(n => n + 1);
      }
      ponerPendiente(undefined);
    } catch (e: unknown) {
      if (signal?.aborted) return;
      ponerError(claveFallo(e));
      // Ante un resultado incierto, sólo se repite la misma petición y clave.
      if (!(e instanceof Fallo) || e.estado >= 500) ponerPendiente(op); else ponerPendiente(undefined);
    } finally { enviando.current = false; if (!signal?.aborted) ponerOcupado(false); }
  }
  function ejecutar(accion: string) {
    if (!datos || bloqueado) return;
    const base = `/api/expedientes/${id}`;
    const ruta = accion === 'respuesta' ? `${base}/salida-spm/respuesta` : accion === 'forzado' ? `${base}/salida-spm/forzado` : accion === 'acta' ? `${base}/documentos/acta_cierre` : `${base}/${accion}`;
    const cuerpo = accion === 'respuesta' ? { version: datos.spm?.version, respuesta, fecha: fechaRespuesta }
      : accion === 'forzado' ? { version: datos.spm?.version, motivo: motivoForzado.trim() }
      : accion === 'cerrar' ? { version: datos.ficha.version, fecha: fechaCierre, motivo_acta_texto: motivoActa.trim(), forzado_id: bloqueadoSPM ? datos.spm?.forzado_id : undefined }
      : accion === 'reabrir' ? { version: datos.ficha.version, fecha: fechaReapertura, motivo: motivoReapertura.trim() }
      : { idioma, version: datos.ficha.version };
    ponerConfirmar(''); void enviar({ accion, ruta, metodo: accion === 'respuesta' ? 'PATCH' : 'POST', cuerpo: JSON.stringify(cuerpo), clave: crypto.randomUUID() });
  }
  const campo = (nombre: string, valor: string, cambiar: (v: string) => void, tipo = 'texto', minimo?: string) => <div className="salidas-campo"><label htmlFor={`salidas-${nombre}`}>{t(`salidas.${nombre}`)}</label>{tipo === 'fecha' ? <input id={`salidas-${nombre}`} type="date" value={valor} min={minimo} max={hoy} aria-invalid={errores.includes(nombre)} aria-describedby={errores.includes(nombre) ? `salidas-error-${nombre}` : undefined} disabled={bloqueado} onChange={e => editar(() => cambiar(e.target.value))} /> : <textarea id={`salidas-${nombre}`} aria-invalid={errores.includes(nombre)} aria-describedby={errores.includes(nombre) ? `salidas-error-${nombre}` : undefined} maxLength={1000} value={valor} disabled={bloqueado} onChange={e => editar(() => cambiar(e.target.value))} />}{errores.includes(nombre) && <p id={`salidas-error-${nombre}`} className="pendiente">{t(tipo === 'fecha' ? 'salidas.fecha_invalida' : 'salidas.campo_obligatorio')}</p>}</div>;
  const formulario = (accion: string, children: React.ReactNode) => <form onSubmit={e => { e.preventDefault(); solicitar(accion); }}>{children}</form>;
  const etiquetaRutaTraslado = t('traslado.ruta');
  return <section className="salidas"><nav className="salidas-acciones"><a href="/coordinacion/cierres" onClick={e => { e.preventDefault(); navegarRuta('/coordinacion/cierres'); }}>{t('salidas.cierres')}</a><a href={`/coordinacion/derivaciones/${id}`} onClick={e => { e.preventDefault(); navegarRuta(`/coordinacion/derivaciones/${id}`); }}>{t('salidas.detalle')}</a></nav>
    <nav className="salidas-acciones" aria-label={etiquetaRutaTraslado}><a href={`/coordinacion/expedientes/${id}/traslado`} onClick={e => { e.preventDefault(); navegarRuta(`/coordinacion/expedientes/${id}/traslado`); }}>{t('traslado.titulo')}</a><a href={`/equipo/seguimiento?id=${id}`} onClick={e => { e.preventDefault(); navegarRuta(`/equipo/seguimiento?id=${id}`); }}>{t('traslado.seguimiento')}</a></nav>
    <header className="salidas-acciones"><h1>{t('salidas.titulo')} {datos?.ficha.numero}</h1><Boton accion={() => ponerAyuda(v => !v)}>{t('salidas.ayuda')}</Boton></header>
    {ayuda && <Ayuda t={t} cerrar={() => ponerAyuda(false)} />}
    <p role="status">{cargando ? t('salidas.cargando') : ocupado ? t('salidas.guardando') : aviso ? t(aviso) : ''}</p>
    {error && <div role="alert" className="pendiente"><p>{t(error)}</p>{pendiente ? <Boton disabled={ocupado} accion={() => void enviar(pendiente)}>{t('salidas.reintentar_operacion')}</Boton> : <Boton disabled={ocupado} accion={() => solicitarSalida(() => { limpiar(); ponerIntento(n => n + 1); })}>{t(error === 'salidas.conflicto' ? 'salidas.ver_cambios' : 'salidas.reintentar')}</Boton>}</div>}
    {errores.length > 0 && <section ref={resumen} tabIndex={-1} className="pendiente" role="alert"><h2>{t('salidas.faltan_datos')}</h2><ul>{errores.map(c => <li key={c}><a href={`#salidas-${c}`} onClick={e => { e.preventDefault(); document.getElementById(`salidas-${c}`)?.focus(); }}>{t(c === 'respuesta' ? 'salidas.respuesta_nueva' : `salidas.${c}`)}</a></li>)}</ul></section>}
    {!cargando && datos && <><p className="pildora">{t(`salidas.estado.${datos.ficha.estado}`) || t('salidas.estado_otro')}</p>
      <section className="salidas-tarjeta"><h2>{t('salidas.spm')}</h2>{!datos.spm ? <p>{t('salidas.sin_salida')}</p> : <>
        <dl><dt>{t('salidas.motivo_salida')}</dt><dd>{datos.spm.motivo}</dd><dt>{t('salidas.propuesta_medida')}</dt><dd>{datos.spm.propuesta_medida}</dd><dt>{t('salidas.derivada_el')}</dt><dd>{datos.spm.fecha_derivacion ? formatoFecha(datos.spm.fecha_derivacion) : t('salidas.fecha_pendiente')}</dd><dt>{t('salidas.respuesta')}</dt><dd>{t(`salidas.respuesta.${datos.spm.respuesta}`)}{datos.spm.fecha_respuesta && <> {formatoFecha(datos.spm.fecha_respuesta)}</>}</dd></dl>
        {datos.spm.fecha_derivacion && !cerrada && formulario('respuesta', <><div className="salidas-campo"><label htmlFor="salidas-respuesta">{t('salidas.respuesta_nueva')}</label><select id="salidas-respuesta" aria-invalid={errores.includes('respuesta')} aria-describedby={errores.includes('respuesta') ? 'salidas-error-respuesta' : undefined} value={respuesta} disabled={bloqueado} onChange={e => editar(() => ponerRespuesta(e.target.value))}><option value="">{t('salidas.elegir')}</option>{['inicio_desamparo_medida', 'medida_no_necesaria'].map(r => <option key={r} value={r}>{t(`salidas.respuesta.${r}`)}</option>)}</select>{errores.includes('respuesta') && <p id="salidas-error-respuesta" className="pendiente">{t('salidas.campo_obligatorio')}</p>}</div>{campo('fecha_respuesta', fechaRespuesta, ponerFechaRespuesta, 'fecha', datos.spm.fecha_derivacion)}<Boton disabled={bloqueado} accion={() => solicitar('respuesta')}>{t('salidas.accion.respuesta')}</Boton></>)}
        {bloqueadoSPM && !cerrada && <div className="pendiente"><p>{t('salidas.bloqueo_spm')}</p>{datos.spm.forzado_id ? <><p>{t('salidas.forzado_anotado')}</p><p>{datos.spm.motivo_forzado}</p></> : formulario('forzado', <>{campo('motivo_forzado', motivoForzado, ponerMotivoForzado)}<Boton disabled={bloqueado} accion={() => solicitar('forzado')}>{t('salidas.accion.forzado')}</Boton></>)}</div>}
      </>}</section>
      <section className="salidas-tarjeta"><h2>{t(cerrada ? 'salidas.cierre_confirmado' : 'salidas.revisar_cierre')}</h2>
        {datos.final ? <dl><dt>{t('salidas.motivo_equipo')}</dt><dd>{etiqueta(`motivo_finalizacion.${datos.final.motivo}`)}</dd>{datos.final.motivo_texto && <><dt>{t('salidas.detalle_motivo')}</dt><dd>{datos.final.motivo_texto}</dd></>}<dt>{t('salidas.motivo_acta')}</dt><dd>{etiqueta(`motivo_acta_cierre.${datos.final.motivo_acta}`)}</dd><dt>{t('salidas.derivacion_tras_cierre')}</dt><dd>{datos.final.derivacion_tras_cierre || t('salidas.sin_indicar')}</dd><dt>{t('salidas.coordinacion_recursos')}</dt><dd>{datos.final.coordinacion_recursos || t('salidas.sin_indicar')}</dd></dl> : <p>{t('salidas.sin_finalizacion')}</p>}
        {!cerrada && datos.propuesta && formulario('cerrar', <><p>{t('salidas.segunda_persona')}</p><p>{t('salidas.aviso_anexo')}</p>{campo('fecha_cierre', fechaCierre, ponerFechaCierre, 'fecha')}{campo('motivo_acta_nuevo', motivoActa, ponerMotivoActa)}<Boton principal disabled={bloqueado || mismaPersona || (bloqueadoSPM && !datos.spm?.forzado_id)} accion={() => solicitar('cerrar')}>{t('salidas.accion.cerrar')}</Boton></>)}
      </section>
      {!cerrada && !datos.propuesta && <p className="pendiente">{t('salidas.sin_propuesta')}</p>}
      {datos.final && (cerrada || !!datos.propuesta) && <section className="salidas-tarjeta"><h2>{t('salidas.acta_nombre')}</h2><p>{t('salidas.acta_consecuencia')}</p><Boton disabled={bloqueado || mismaPersona} accion={() => ejecutar('acta')}>{t('salidas.accion.acta')}</Boton></section>}
      {cerrada && <section className="salidas-tarjeta"><h2>{t('salidas.accion.reabrir')}</h2>{formulario('reabrir', <>{campo('fecha_reapertura', fechaReapertura, ponerFechaReapertura, 'fecha')}{campo('motivo_reapertura', motivoReapertura, ponerMotivoReapertura)}<p>{t('salidas.reapertura_consecuencia')}</p><Boton principal disabled={bloqueado} accion={() => solicitar('reabrir')}>{t('salidas.accion.reabrir')}</Boton></>)}</section>}
    </>}
    {confirmar && <Confirmar t={t} titulo={t(`salidas.accion.${confirmar}`)} texto={t(`salidas.confirmar.${confirmar}`)} cancelar={() => ponerConfirmar('')} aceptar={() => ejecutar(confirmar)} />}
  </section>;
}
