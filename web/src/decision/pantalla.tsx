import { useEffect, useRef, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useCambios } from '../cambios';
import { Ayuda, Boton, Confirmacion } from './componentes';
import { Documentos } from './documentos';
import { comprobar, FalloAPI, faltantes, leer, preparar, tipos, type Derivacion, type Emision, type Formulario, type Opciones, type Opcion } from './api';
import './decision.css';

export function ListaEspera({ t, idioma, navegarRuta }: PropsPantalla) {
  const nombreLista = t('decision.lista_titulo');
  const [filas, ponerFilas] = useState<{ id: string; fecha_recepcion: string; fecha_lista_espera: string }[]>();
  const [error, ponerError] = useState('');
  const [pagina, ponerPagina] = useState(0);
  const [intento, reintentar] = useState(0);
  useEffect(() => {
    const c = new AbortController(); ponerFilas(undefined); ponerError('');
    void leer<NonNullable<typeof filas>>(`/api/coordinacion/lista-espera?limite=20&desplazamiento=${pagina * 20}`, c.signal).then(f => { if (!c.signal.aborted) ponerFilas(f ?? []); }).catch(e => { if (!c.signal.aborted) ponerError(e instanceof FalloAPI && e.estado === 403 ? 'decision.sin_permiso' : 'decision.error'); });
    return () => c.abort();
  }, [pagina, intento]);
  useEffect(() => { document.title = t('decision.lista_titulo'); }, [t]);
  return <div className="decision"><h1>{t('decision.lista_titulo')}</h1>
    {error ? <div role="alert"><p>{t(error)}</p><Boton accion={() => reintentar(i => i + 1)}>{t('decision.reintentar')}</Boton></div> : !filas ? <p role="status">{t('decision.cargando')}</p> : <section className="decision-tarjeta" aria-label={nombreLista}>
      {!filas.length && <p>{t('decision.lista_vacia')}</p>}
      {filas.map((fila, i) => <div className="decision-fila-espera" key={fila.id}><span>{t('decision.derivacion')} {new Intl.NumberFormat(idioma).format(pagina * 20 + i + 1)}</span><dl><dt>{t('decision.recibida')}</dt><dd>{fecha(fila.fecha_recepcion, idioma)}</dd><dt>{t('decision.en_espera_desde')}</dt><dd>{fecha(fila.fecha_lista_espera, idioma)}</dd></dl><Boton secundario accion={() => navegarRuta(`/coordinacion/derivaciones/${fila.id}`)}>{t('decision.revisar')}</Boton></div>)}
      <div className="decision-acciones">{pagina > 0 && <Boton secundario accion={() => ponerPagina(p => p - 1)}>{t('decision.anterior')}</Boton>}{filas.length === 20 && <Boton secundario accion={() => ponerPagina(p => p + 1)}>{t('decision.siguiente')}</Boton>}</div>
    </section>}
  </div>;
}
const fecha = (valor: string, idioma: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(valor.length === 10 ? `${valor}T12:00:00` : valor));
type Operacion = 'aceptar' | 'no-aceptar' | 'lista-espera';
type Peticion = { ruta: string; cuerpo: string; clave: string; resultado: (r: Response) => Promise<void> };
export function DetalleDecision({ t, idioma, csrf, navegarRuta }: PropsPantalla) {
  const id = location.pathname.split('/').at(-1)!;
  const nombreRuta = t('decision.ruta');
  const [d, ponerD] = useState<Derivacion>();
  const [o, ponerO] = useState<Opciones>();
  const [f, ponerF] = useState<Formulario>();
  const [emisiones, ponerEmisiones] = useState<Emision[]>([]);
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState('');
  const [conflicto, ponerConflicto] = useState(false);
  const [ocupado, ponerOcupado] = useState(false);
  const [pendiente, ponerPendiente] = useState<Peticion>();
  const [confirmar, ponerConfirmar] = useState<Operacion>();
  const [rechazar, ponerRechazar] = useState(false);
  const [motivo, ponerMotivo] = useState('');
  const [fechaNo, ponerFechaNo] = useState('');
  const [errores, ponerErrores] = useState<string[]>([]);
  const [sucio, ponerEditado] = useState(false);
  const [sucioRegistro, ponerSucioRegistro] = useState(false);
  const [ayuda, ponerAyuda] = useState(false);
  const [lateralArriba, ponerLateralArriba] = useState(false);
  useEffect(() => {
    const ajustar = () => ponerLateralArriba(innerWidth < parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu')) * 5);
    ajustar(); window.addEventListener('resize', ajustar); return () => window.removeEventListener('resize', ajustar);
  }, []);
  const resumen = useRef<HTMLElement>(null);
  const controlador = useRef<AbortController>(new AbortController());
  const bloqueo = useRef(false);
  const { ponerSucio, solicitarSalida } = useCambios();
  const recargarEmisiones = async () => {
    const lista = await leer<Emision[]>(`/api/coordinacion/derivaciones/${id}/documentos`, controlador.current.signal);
    if (!controlador.current.signal.aborted) ponerEmisiones(lista);
  };
  const cargar = async () => {
    ponerCargando(true); ponerError('');
    try {
      const [derivacion, opciones] = await Promise.all([
        leer<Derivacion>(`/api/coordinacion/derivaciones/${id}`, controlador.current.signal),
        leer<Opciones>(`/api/coordinacion/decision/opciones?idioma=${encodeURIComponent(idioma)}`, controlador.current.signal),
      ]);
      if (controlador.current.signal.aborted) return;
      ponerD(derivacion); ponerO(opciones); ponerF(preparar(derivacion, opciones)); ponerMotivo(''); ponerFechaNo(''); ponerRechazar(false); ponerErrores([]); ponerEditado(false); ponerSucio(false); ponerConflicto(false);
      await recargarEmisiones();
    } catch (e) { if (!controlador.current.signal.aborted) { ponerD(undefined); ponerError(e instanceof FalloAPI && e.estado === 403 ? 'decision.sin_permiso' : e instanceof FalloAPI && e.estado === 404 ? 'decision.no_encontrada' : 'decision.error'); } }
    finally { if (!controlador.current.signal.aborted) ponerCargando(false); }
  };
  useEffect(() => { controlador.current = new AbortController(); void cargar(); return () => { controlador.current.abort(); ponerSucio(false); }; }, [id, idioma]);
  useEffect(() => { ponerSucio(sucio || sucioRegistro || !!pendiente || ocupado); }, [sucio, sucioRegistro, pendiente, ocupado, ponerSucio]);
  useEffect(() => { document.title = t('decision.titulo'); }, [t]);
  const editar = (cambio: Partial<Formulario>) => {
    if (!f) return;
    const nuevo = { ...f, ...cambio }; ponerF(nuevo); ponerEditado(true);
    const pendientes = faltantes(nuevo);
    ponerErrores(actuales => actuales.filter(c => pendientes.includes(c) || c === 'motivo' || c === 'fecha_no'));
  };
  const textoError = (e: unknown) => e instanceof FalloAPI && e.estado === 403 ? 'decision.sin_permiso' : e instanceof FalloAPI && t(e.clave) ? e.clave : 'decision.error';
  const enviar = async (p: Peticion) => {
    if (bloqueo.current) return;
    bloqueo.current = true; ponerOcupado(true); ponerError('');
    try {
      const r = await fetch(p.ruta, { method: 'POST', signal: controlador.current.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': p.clave }, body: p.cuerpo });
      await comprobar(r); if (controlador.current.signal.aborted) return;
      ponerPendiente(undefined); ponerConfirmar(undefined); await p.resultado(r);
    } catch (e) {
      if (controlador.current.signal.aborted) return;
      if (e instanceof FalloAPI) { ponerPendiente(undefined); ponerConfirmar(undefined); if (e.estado === 409) ponerConflicto(true); else ponerError(textoError(e)); }
      else { ponerPendiente(p); ponerConfirmar(undefined); ponerError('decision.resultado_incierto'); ponerSucio(true); }
    } finally { if (!controlador.current.signal.aborted) { bloqueo.current = false; ponerOcupado(false); } }
  };
  const ejecutar = (ruta: string, datos: unknown, resultado: Peticion['resultado']) => { if (!pendiente) void enviar({ ruta, cuerpo: JSON.stringify(datos), clave: crypto.randomUUID(), resultado }); };
  const validar = (accion: Operacion) => {
    const faltan = accion === 'aceptar' ? faltantes(f!) : accion === 'no-aceptar' ? [...(!motivo.trim() ? ['motivo'] : []), ...(!fechaNo ? ['fecha_no'] : []), ...(!f!.subprograma ? ['subprograma'] : [])] : [];
    ponerErrores(faltan); if (!faltan.length) ponerConfirmar(accion); else requestAnimationFrame(() => resumen.current?.focus());
  };
  if (cargando) return <div className="decision"><h1>{t('decision.titulo')}</h1><p role="status">{t('decision.cargando')}</p></div>;
  if (!d || !f || !o) return <div className="decision"><h1>{t('decision.titulo')}</h1><p role="alert">{t(error || 'decision.no_encontrada')}</p><Boton accion={() => void cargar()}>{t('decision.reintentar')}</Boton></div>;
  const campos = d.derivacion_datos.contenido.campos;
  const vacios = campos.filter(c => !c.valor.trim() && !c.codigo.endsWith('.orden'));
  const conValor = campos.filter(c => c.valor.trim());
  const menores = campos.filter(c => c.codigo === 'menores.persona.nombre');
  const editable = d.estado === 'derivacion_recibida' || d.estado === 'lista_espera';
  const faltan = faltantes(f);
  const etiqueta = (clave: string) => o.textos[clave] || t('decision.sin_indicar');
  const valorCampo = (valor: string) => {
    if (valor === 'SI' || valor === 'NO') return t(`decision.valor.${valor.toLowerCase()}`);
    const clave = Object.keys(o.textos).find(k => k.endsWith(`.${valor}`));
    return clave ? etiqueta(clave) : valor || t('decision.sin_indicar');
  };
  const tituloCampo = (campo: string) => t(`decision.campo.${campo}`) || t('decision.dato_anexo');
  const claveFalta = (campo: string) => campo.startsWith('sexo-') ? `${t('decision.sexo')} ${menores[Number(campo.split('-')[1]) - 1]?.valor ?? ''}` : t(`decision.${campo}`);
  const enfocar = (campo: string) => { const elemento = document.getElementById(`decision-${campo}`); elemento?.scrollIntoView({ block: 'center' }); elemento?.focus(); };
  const select = (campo: 'municipio_id' | 'centro_ss_id' | 'equipo_id', opciones: Opcion[]) => <div className="decision-campo">
    <label htmlFor={`decision-${campo}`}>{t(`decision.${campo}`)}</label>
    <select id={`decision-${campo}`} value={f[campo]} disabled={!editable || bloqueado} aria-invalid={errores.includes(campo)} aria-describedby={editable && !f[campo] ? 'decision-faltantes' : undefined} onChange={e => {
      const valor = e.target.value;
      if (campo === 'municipio_id') { const m = o.municipios.find(m => m.id === valor); editar({ municipio_id: valor, centro_ss_id: m?.centro_ss_id ?? '', equipo_id: m?.equipo_id ?? '' }); }
      else editar({ [campo]: valor });
    }}><option value="">{t('decision.elegir')}</option>{opciones.map(op => <option key={op.id} value={op.id}>{etiqueta(op.clave_texto)}</option>)}</select>
  </div>;
  const documentos = tipos[d.estado as keyof typeof tipos] ?? [];
  const visibles = documentos.filter(tipo => !tipo.endsWith('oficio_sscc') || d.derivacion_datos.contenido.tipo === 'anexo3').filter(tipo => !tipo.endsWith('oficio_spm') || d.derivacion_datos.contenido.tipo === 'anexo4');
  const bloqueado = ocupado || !!pendiente || conflicto;
  return <div className="decision" data-lateral-arriba={lateralArriba}>
    <nav aria-label={nombreRuta}><a href="/coordinacion/lista-espera" onClick={e => { e.preventDefault(); navegarRuta('/coordinacion/lista-espera'); }}>{t('decision.lista_titulo')}</a><span>{t('decision.titulo')}</span></nav>
    <div className="decision-cabecera"><div><h1>{t('decision.titulo')}</h1><p>{t(`decision.${d.derivacion_datos.contenido.tipo}`)} {fecha(d.lectura_anexo.leida_en, idioma)}</p></div><Boton secundario accion={() => ponerAyuda(v => !v)}>{t('decision.ayuda')}</Boton></div>
    {ayuda && <Ayuda titulo={t('decision.ayuda')} texto={t('decision.ayuda_texto')} cerrar={() => ponerAyuda(false)} t={t} />}
    {d.numero && <a href={`/coordinacion/expedientes/${id}/salidas`} onClick={e => { e.preventDefault(); solicitarSalida(() => navegarRuta(`/coordinacion/expedientes/${id}/salidas`)); }}>{t('salidas.titulo')}</a>}
    <p id="decision-estado" tabIndex={-1} className="decision-estado" role="status">{t(`decision.estado.${d.estado}`)} {d.numero && <strong>{d.numero}</strong>}</p>
    {error && <div role="alert"><p>{t(error)}</p>{pendiente ? <Boton accion={() => void enviar(pendiente)} disabled={ocupado}>{t('decision.reintentar_operacion')}</Boton> : <Boton secundario accion={() => solicitarSalida(() => { void cargar(); })}>{t('decision.reintentar')}</Boton>}</div>}
    {conflicto && <section className="pendiente" role="alert"><p>{t('decision.conflicto')}</p><Boton secundario accion={() => solicitarSalida(() => { void cargar(); })}>{t('decision.ver_cambios')}</Boton></section>}
    {ocupado && <p role="status">{t('decision.guardando')}</p>}
    {errores.length > 0 && <section className="pendiente" role="alert" ref={resumen} tabIndex={-1}><h2>{t('decision.revisar_campos')}</h2><ul>{errores.map(c => <li key={c}><a href={`#decision-${c}`} onClick={e => { e.preventDefault(); enfocar(c); }}>{claveFalta(c)}</a></li>)}</ul></section>}
    {editable && <p className="decision-info">{t('decision.leido')}</p>}
    <div className="decision-disposicion"><div className="decision-principal">
      <section className="decision-tarjeta" aria-labelledby="decision-ubicacion"><h2 id="decision-ubicacion">{t('decision.ubicacion')}</h2><div className="decision-campos">
        {select('municipio_id', o.municipios)}{select('centro_ss_id', o.centros)}{select('equipo_id', o.equipos)}
        <div className="decision-campo"><label htmlFor="decision-subprograma">{t('decision.subprograma')}</label><select id="decision-subprograma" value={f.subprograma} disabled={!editable || bloqueado || d.derivacion_datos.contenido.tipo === 'anexo3'} aria-invalid={errores.includes('subprograma')} onChange={e => editar({ subprograma: e.target.value })}><option value="">{t('decision.elegir')}</option>{o.subprogramas.map(op => <option key={op.codigo} value={op.codigo}>{etiqueta(op.clave_texto)}</option>)}</select>{d.derivacion_datos.contenido.tipo === 'anexo3' && <small>{t('decision.subprograma_anexo3')}</small>}</div>
        <div className="decision-campo"><label htmlFor="decision-profesional_referencia_puesto">{t('decision.profesional_referencia_puesto')}</label><select id="decision-profesional_referencia_puesto" value={f.profesional_referencia_puesto} disabled={!editable || bloqueado} aria-invalid={errores.includes('profesional_referencia_puesto')} onChange={e => editar({ profesional_referencia_puesto: e.target.value })}><option value="">{t('decision.elegir')}</option>{['TS', 'ES', 'PS'].map(p => <option key={p} value={p}>{t(`decision.puesto.${p.toLowerCase()}`)}</option>)}</select></div>
        <div className="decision-campo"><label htmlFor="decision-fecha_acta">{t('decision.fecha_acta')}</label><input id="decision-fecha_acta" type="date" value={f.fecha_acta} disabled={!editable || bloqueado} aria-invalid={errores.includes('fecha_acta')} onChange={e => editar({ fecha_acta: e.target.value })} /></div>
      </div>
        {(d.derivacion_datos.propuesta_municipio.Candidatos ?? []).map(c => <p className="decision-origen" key={c.Codigo}>{t('decision.origen_municipio')} {c.Fragmento}</p>)}
        {o.municipios.find(m => m.id === f.municipio_id)?.ambito_diputacion === false && <p className="pendiente">{t('decision.fuera_ambito')}</p>}
        <p>{d.numero ? t('decision.numero_asignado') : t('decision.numero_al_aceptar')} {d.numero}</p>
      </section>
      <section className="decision-tarjeta" aria-labelledby="decision-menores"><h2 id="decision-menores" tabIndex={-1}>{t('decision.menores')} {new Intl.NumberFormat(idioma).format(menores.length)}</h2>
        {!menores.length && <p className="pendiente">{t('decision.sin_menores')}</p>}
        {menores.map((m, i) => <section className="decision-persona" key={i}><h3>{m.valor}</h3><dl>{['nacimiento', 'escolarizacion'].map(tipo => { const valor = campos.filter(c => c.codigo === `menores.persona.${tipo}`)[i]?.valor; return valor?.trim() ? <div key={tipo}><dt>{t(`decision.${tipo}`)}</dt><dd>{tipo === 'nacimiento' && /^\d{4}-\d{2}-\d{2}$/.test(valor) ? fecha(valor, idioma) : valorCampo(valor)}</dd></div> : null; })}</dl>
          <fieldset className={editable && !f.menores[i]?.sexo ? 'decision-pendiente' : undefined}><legend>{t('decision.sexo')} {m.valor}</legend>{['M', 'H'].map(sexo => <label key={sexo} className="decision-radio"><input id={sexo === 'M' ? `decision-sexo-${i + 1}` : undefined} type="radio" name={`sexo-${i + 1}`} value={sexo} checked={f.menores[i]?.sexo === sexo} disabled={!editable || bloqueado} aria-describedby={editable && !f.menores[i]?.sexo ? 'decision-faltantes' : undefined} onChange={() => editar({ menores: f.menores.map((actual, j) => i === j ? { ...actual, sexo } : actual) })} />{t(`decision.sexo.${sexo.toLowerCase()}`)}</label>)}</fieldset>
        </section>)}
      </section>
      <section className="decision-tarjeta" aria-labelledby="decision-familia"><h2 id="decision-familia">{t('decision.familia')}</h2><dl>{conValor.filter(c => (c.codigo.startsWith('familia.') || c.codigo.startsWith('sociodemografia.')) && !c.codigo.endsWith('.orden')).map((c, i) => <div key={i}><dt>{tituloCampo(c.codigo)}</dt><dd>{valorCampo(c.valor)}</dd></div>)}</dl></section>
      <section className="decision-tarjeta" aria-labelledby="decision-situacion"><h2 id="decision-situacion">{t('decision.situacion')}</h2><label htmlFor="decision-descripcion_situacion">{t('decision.descripcion_situacion')}</label><textarea id="decision-descripcion_situacion" rows={5} maxLength={20000} value={f.descripcion_situacion} disabled={!editable || bloqueado} aria-invalid={errores.includes('descripcion_situacion')} onChange={e => editar({ descripcion_situacion: e.target.value })} /></section>
      {vacios.length > 0 && <details className="decision-tarjeta"><summary>{t('decision.datos_ausentes').replace('{cantidad}', new Intl.NumberFormat(idioma).format(vacios.length))}</summary><dl>{vacios.map((c, i) => <div key={i}><dt>{tituloCampo(c.codigo)}</dt><dd>{t('decision.sin_indicar')}</dd></div>)}</dl></details>}
      <details className="decision-tarjeta"><summary>{t('decision.resto_anexo')}</summary><dl>{conValor.filter(c => !c.codigo.startsWith('familia.') && !c.codigo.startsWith('sociodemografia.') && !c.codigo.startsWith('menores.persona.') && c.codigo !== 'valoracion.descripcion' && !c.codigo.endsWith('.orden')).map((c, i) => <div key={i}><dt>{tituloCampo(c.codigo)}</dt><dd>{valorCampo(c.valor)}{c.dudoso && <small>{t('decision.dato_dudoso')}</small>}</dd></div>)}</dl></details>
      {d.no_aceptacion && <section className="decision-tarjeta"><h2>{t('decision.no_aceptada')}</h2><p>{fecha(d.no_aceptacion.fecha, idioma)}</p><p>{d.no_aceptacion.motivo}</p></section>}
      <Documentos id={id} tipos={visibles} emisiones={emisiones} ejecutar={ejecutar} recargar={recargarEmisiones} t={t} idioma={idioma} ocupado={bloqueado} alEditar={ponerSucioRegistro} />
    </div><aside className="decision-lateral">
      {editable && <>
        {faltan.length > 0 && <section className="decision-pendiente" aria-labelledby="decision-faltantes"><h2 id="decision-faltantes">{t('decision.faltantes')}</h2><ul>{faltan.map(c => <li key={c}><a href={`#decision-${c}`} onClick={e => { e.preventDefault(); enfocar(c); }}>{claveFalta(c)}</a></li>)}</ul></section>}
        <section className="decision-tarjeta" aria-labelledby="decision-acciones-titulo"><h2 id="decision-acciones-titulo">{t('decision.decision')}</h2>
          <Boton disabled={bloqueado} accion={() => validar('aceptar')}>{t('decision.aceptar')}</Boton><p>{t('decision.aceptar_consecuencia')}</p>
          {d.estado === 'derivacion_recibida' && d.derivacion_datos.contenido.tipo === 'anexo3' && <><Boton secundario disabled={bloqueado} accion={() => validar('lista-espera')}>{t('decision.lista_espera')}</Boton><p>{t('decision.espera_consecuencia')}</p></>}
          <Boton secundario disabled={bloqueado} accion={() => ponerRechazar(true)}>{t('decision.no_aceptar')}</Boton><p>{t('decision.no_consecuencia')}</p>
          {rechazar && <form onSubmit={e => { e.preventDefault(); validar('no-aceptar'); }}><fieldset><legend>{t('decision.no_aceptar')}</legend><label htmlFor="decision-motivo">{t('decision.motivo')}</label><textarea id="decision-motivo" rows={4} maxLength={10000} value={motivo} disabled={bloqueado} onChange={e => { ponerMotivo(e.target.value); ponerEditado(true); if (e.target.value.trim()) ponerErrores(es => es.filter(c => c !== 'motivo')); }} aria-invalid={errores.includes('motivo')} /><label htmlFor="decision-fecha_no">{t('decision.fecha_no')}</label><input id="decision-fecha_no" type="date" value={fechaNo} disabled={bloqueado} onChange={e => { ponerFechaNo(e.target.value); ponerEditado(true); if (e.target.value) ponerErrores(es => es.filter(c => c !== 'fecha_no')); }} aria-invalid={errores.includes('fecha_no')} /><p>{t('decision.subprograma')} {etiqueta(o.subprogramas.find(op => op.codigo === f.subprograma)?.clave_texto ?? '')}</p><button type="submit" className="boton-secundario" disabled={bloqueado}>{t('decision.revisar_no')}</button></fieldset></form>}
        </section>
      </>}
      <section className="decision-tarjeta" aria-labelledby="decision-servicio"><h2 id="decision-servicio">{t('decision.servicio')}</h2><dl>{conValor.filter(c => c.codigo.startsWith('servicio.')).map((c, i) => <div key={i}><dt>{tituloCampo(c.codigo)}</dt><dd>{valorCampo(c.valor)}</dd></div>)}</dl></section>
    </aside></div>
    {confirmar && <Confirmacion titulo={t(`decision.confirmar.${confirmar.replaceAll('-', '_')}`)} texto={t(`decision.consecuencia.${confirmar.replaceAll('-', '_')}`)} t={t} ocupado={ocupado} volver={() => ponerConfirmar(undefined)} confirmar={() => {
      const datos = confirmar === 'aceptar' ? { ...f, version: d.version } : confirmar === 'lista-espera' ? { version: d.version } : { version: d.version, fecha: fechaNo, motivo: motivo.trim(), subprograma: f.subprograma };
      ejecutar(`/api/coordinacion/derivaciones/${id}/${confirmar}`, datos, async r => {
        const resultado = await r.json() as { id: string; estado: string; numero?: string };
        const esperado = confirmar === 'aceptar' ? 'aceptada' : confirmar === 'lista-espera' ? 'lista_espera' : 'no_aceptada';
        if (resultado.id !== id || resultado.estado !== esperado || (confirmar === 'aceptar' && !resultado.numero)) throw new FalloAPI(503, 'decision.error');
        ponerEditado(false); ponerSucio(false); await cargar();
      });
    }} />}
  </div>;
}
