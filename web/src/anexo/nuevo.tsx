import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import { useCambios } from '../cambios';
import type { PropsPantalla } from '../registro-pantallas';
import { apartados, camposIniciales, claveError, propuestaInicial, type Analisis, type Campo, type Opcion, type Ubicacion } from './modelo';
import './anexo.css';

function Boton({ children, accion, ocupado = false, principal = false }: { children: React.ReactNode; accion: () => void; ocupado?: boolean; principal?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress: accion, isDisabled: ocupado }, ref);
  return <button {...buttonProps} ref={ref} className={principal ? 'anexo-principal' : undefined}>{children}</button>;
}

export function NuevoAnexo({ t, idioma, csrf, navegarRuta }: PropsPantalla) {
  const { sucio, ponerSucio } = useCambios();
  const [amplio, ponerAmplio] = useState(false);
  const [analisis, ponerAnalisis] = useState<Analisis>();
  const [campos, ponerCampos] = useState<Campo[]>([]);
  const [ubicacion, ponerUbicacion] = useState<Ubicacion>({ municipio_id: '', centro_id: '', equipo_id: '' });
  const [tipo, ponerTipo] = useState('');
  const [estado, ponerEstado] = useState<'elegir' | 'subiendo' | 'revisando' | 'listo' | 'guardando'>('elegir');
  const [progreso, ponerProgreso] = useState(0);
  const [error, ponerError] = useState('');
  const [etiquetas, ponerEtiquetas] = useState<Record<string, string>>({});
  const [cargaEtiquetas, ponerCargaEtiquetas] = useState(true);
  const [intento, ponerIntento] = useState(0);
  const [faltan, ponerFaltan] = useState(false);
  const [incierto, ponerIncierto] = useState(false);
  const fichero = useRef<HTMLInputElement>(null);
  const peticion = useRef<XMLHttpRequest | undefined>(undefined);
  const controlador = useRef<AbortController | undefined>(undefined);
  const actual = useRef<Analisis | undefined>(undefined);
  const resumen = useRef<HTMLDivElement>(null);
  const encabezado = useRef<HTMLHeadingElement>(null);
  const ayuda = useRef<HTMLDialogElement>(null);
  const envio = useRef<{ clave: string; cuerpo: string } | undefined>(undefined);
  const completado = useRef(false);
  const ultimoCSRF = useRef(csrf); ultimoCSRF.current = csrf;
  const ocupado = estado === 'subiendo' || estado === 'revisando' || estado === 'guardando';
  const numero = (valor: number) => new Intl.NumberFormat(idioma).format(valor);
  const textoError = t(error) || t('anexo.error');
  const textoSubiendo = t('anexo.subiendo');
  const textoRevisando = t('anexo.revisando_fichero');

  useEffect(() => { const observar = new ResizeObserver(() => { const menu = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu')); ponerAmplio(document.documentElement.clientWidth >= menu * 5); }); observar.observe(document.documentElement); return () => observar.disconnect(); }, []);
  useEffect(() => {
    const abortar = new AbortController(); ponerCargaEtiquetas(true);
    void fetch(`/api/coordinacion/anexos/etiquetas?idioma=${encodeURIComponent(idioma)}`, { signal: abortar.signal })
      .then(async (r) => { const datos = await r.json(); if (!r.ok) throw new Error(claveError(datos)); return datos as Record<string, string>; })
      .then((datos) => { if (!abortar.signal.aborted) { ponerEtiquetas(datos); ponerError(''); } })
      .catch(() => { if (!abortar.signal.aborted) ponerError('anexo.error_catalogo'); })
      .finally(() => { if (!abortar.signal.aborted) ponerCargaEtiquetas(false); });
    return () => abortar.abort();
  }, [idioma, intento]);
  useEffect(() => { ponerSucio(ocupado || Boolean(analisis) || incierto); }, [ocupado, analisis, incierto, ponerSucio]);
  useEffect(() => { if (faltan) resumen.current?.focus(); }, [faltan]);
  useEffect(() => {
    if (analisis) { requestAnimationFrame(() => encabezado.current?.focus()); }
  }, [analisis]);
  useEffect(() => () => {
    peticion.current?.abort(); controlador.current?.abort(); ponerSucio(false);
    const id = actual.current?.lectura_anexo.id;
    if (id && !completado.current) void fetch(`/api/coordinacion/anexos/lecturas/${encodeURIComponent(id)}`, { method: 'DELETE', headers: { 'X-CSRF-Token': ultimoCSRF.current }, keepalive: true });
  }, [ponerSucio]);

  const subir = () => {
    const archivo = fichero.current?.files?.[0]; ponerError('');
    if (!archivo) { ponerError('anexo.falta_fichero'); fichero.current?.focus(); return; }
    if (!/\.(docx|pdf)$/i.test(archivo.name)) { ponerError('expediente.anexo_formato'); fichero.current?.focus(); return; }
    if (archivo.size > 10_000_000 || archivo.size === 0) { ponerError('expediente.anexo_tamano'); fichero.current?.focus(); return; }
    const datos = new FormData(); datos.append('anexo', archivo);
    const xhr = new XMLHttpRequest(); peticion.current = xhr;
    xhr.open('POST', '/api/coordinacion/derivaciones'); xhr.setRequestHeader('X-CSRF-Token', csrf);
    xhr.responseType = 'json'; xhr.timeout = 60000;
    xhr.upload.onprogress = (evento) => { if (evento.lengthComputable) ponerProgreso(Math.round(evento.loaded / evento.total * 100)); };
    xhr.upload.onload = () => ponerEstado('revisando');
    xhr.onload = () => {
      peticion.current = undefined;
      if (xhr.status !== 200 || !xhr.response?.lectura_anexo?.id) { ponerError(claveError(xhr.response)); ponerEstado('elegir'); return; }
      const datos = xhr.response as Analisis; actual.current = datos;
      ponerAnalisis(datos); ponerCampos(camposIniciales(datos)); ponerUbicacion(propuestaInicial(datos)); ponerTipo(datos.derivacion_datos.contenido.tipo || ''); ponerEstado('listo');
      if (fichero.current) fichero.current.value = '';
    };
    xhr.onerror = xhr.ontimeout = () => { peticion.current = undefined; ponerError('anexo.error'); ponerEstado('elegir'); };
    xhr.onabort = () => { peticion.current = undefined; ponerEstado('elegir'); ponerProgreso(0); };
    ponerProgreso(0); ponerEstado('subiendo'); xhr.send(datos);
  };
  const descartar = async () => {
    if (!analisis) return;
    ponerError(''); ponerEstado('guardando');
    try {
      const r = await fetch(`/api/coordinacion/anexos/lecturas/${encodeURIComponent(analisis.lectura_anexo.id)}`, { method: 'DELETE', headers: { 'X-CSRF-Token': csrf } });
      if (!r.ok) throw new Error(claveError(await r.json()));
      actual.current = undefined; ponerAnalisis(undefined); ponerCampos([]); envio.current = undefined; ponerIncierto(false); ponerFaltan(false); ponerEstado('elegir'); ponerSucio(false);
    } catch { ponerError('anexo.error'); ponerEstado('listo'); }
  };
  const confirmar = async () => {
    if (!analisis) return;
    if (!envio.current && (!tipo || Object.values(ubicacion).some((v) => !v))) { ponerFaltan(false); requestAnimationFrame(() => ponerFaltan(true)); return; }
    if (!envio.current) envio.current = { clave: crypto.randomUUID(), cuerpo: JSON.stringify({ lectura_id: analisis.lectura_anexo.id, tipo, campos, ubicacion }) };
    ponerEstado('guardando'); ponerError(''); ponerFaltan(false);
    const abortar = new AbortController(); controlador.current = abortar;
    try {
      const r = await fetch('/api/coordinacion/derivaciones/confirmar', { method: 'POST', signal: abortar.signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': envio.current.clave }, body: envio.current.cuerpo });
      const datos = await r.json();
      if (!r.ok) {
        if (r.status >= 400 && r.status < 500) { envio.current = undefined; ponerIncierto(false); }
        else ponerIncierto(true);
        throw new Error(claveError(datos));
      }
      if (!datos.id || datos.estado !== 'derivacion_recibida') throw new Error('anexo.error');
      completado.current = true; ponerIncierto(false); ponerSucio(false); ponerAnalisis(undefined); actual.current = undefined;
      // El armazón aplica la guardia con el estado del render siguiente.
      ponerEstado('elegir'); destino.current = `/coordinacion/derivaciones/${encodeURIComponent(datos.id)}`;
    } catch (fallo) {
      if (!abortar.signal.aborted) { if (envio.current) ponerIncierto(true); ponerError(fallo instanceof Error ? fallo.message : 'anexo.error'); ponerEstado('listo'); }
    }
  };
  const destino = useRef<string | undefined>(undefined);
  useEffect(() => { if (completado.current && !sucio && !analisis && estado === 'elegir' && destino.current) { const ruta = destino.current; destino.current = undefined; navegarRuta(ruta); } }, [analisis, estado, sucio, navegarRuta]);
  const enfocar = (id: string) => { const campo = document.getElementById(id); let padre = campo?.parentElement; while (padre) { if (padre instanceof HTMLDetailsElement) padre.open = true; padre = padre.parentElement; } campo?.focus(); };
  const nombreOpcion = (o: Opcion) => etiquetas[o.clave_texto] || t('anexo.sin_etiqueta');
  const elegirMunicipio = (id: string) => {
    const municipio = analisis?.derivacion_datos.opciones_ubicacion.municipios.find((o) => o.id === id);
    const opciones = analisis?.derivacion_datos.opciones_ubicacion;
    ponerUbicacion({ municipio_id: id, centro_id: opciones?.centros.some((o) => o.id === municipio?.centro_id) ? municipio?.centro_id ?? '' : '', equipo_id: opciones?.equipos.some((o) => o.id === municipio?.equipo_id) ? municipio?.equipo_id ?? '' : '' });
  };
  const pendientes = [!tipo && 'tipo', ...Object.entries(ubicacion).filter(([, v]) => !v).map(([k]) => k)].filter(Boolean) as string[];
  const vacios = campos.filter((c) => !c.valor.trim()).length;
  const dudosos = campos.filter((c) => c.dudoso && c.valor.trim()).length;
  const opciones = analisis?.derivacion_datos.opciones_ubicacion;
  const municipio = opciones?.municipios.find((o) => o.id === ubicacion.municipio_id);
  const candidato = analisis?.derivacion_datos.propuesta_municipio.Candidatos?.find((c) => c.MunicipioID === ubicacion.municipio_id);
  const bloqueado = ocupado || incierto;

  return <section className="anexo" data-amplio={amplio} aria-labelledby="anexo-titulo">
    <div className="anexo-encabezado"><h1 id="anexo-titulo" ref={encabezado} tabIndex={-1}>{t(analisis ? 'anexo.revisar' : 'anexo.titulo')}</h1><Boton accion={() => ayuda.current?.showModal()}>{t('anexo.ayuda_boton')}</Boton></div>
    <dialog ref={ayuda} aria-labelledby="anexo-ayuda"><h2 id="anexo-ayuda">{t('anexo.ayuda_titulo')}</h2><p>{t('anexo.ayuda')}</p><Boton accion={() => ayuda.current?.close()}>{t('anexo.cerrar_ayuda')}</Boton></dialog>
    {error && <div id="anexo-error" className="anexo-error" role="alert"><p>{textoError}</p>{error === 'anexo.error_catalogo' && <Boton accion={() => ponerIntento((v) => v + 1)}>{t('anexo.reintentar_catalogo')}</Boton>}</div>}
    {cargaEtiquetas && <p role="status">{t('anexo.cargando_catalogo')}</p>}
    {!analisis ? <section className="anexo-tarjeta" aria-labelledby="anexo-elegir">
      <h2 id="anexo-elegir">{t('anexo.elegir_titulo')}</h2><p id="anexo-formatos">{t('anexo.formatos')}</p><p>{t('anexo.descarte')}</p>
      <label htmlFor="anexo-fichero">{t('anexo.elegir_fichero')}</label><input ref={fichero} id="anexo-fichero" type="file" accept=".docx,.pdf" aria-invalid={['anexo.falta_fichero', 'expediente.anexo_formato', 'expediente.anexo_tamano'].includes(error)} aria-describedby={error ? "anexo-formatos anexo-error" : "anexo-formatos"} disabled={ocupado} />
      <div className="anexo-acciones"><Boton accion={subir} ocupado={ocupado || cargaEtiquetas || !Object.keys(etiquetas).length} principal>{t('anexo.subir')}</Boton>{ocupado && <Boton accion={() => peticion.current?.abort()}>{t('anexo.cancelar')}</Boton>}</div>
      <div role="status" aria-live="polite">{estado === 'subiendo' && <><p>{t('anexo.subiendo')} {numero(progreso)}{t('anexo.porcentaje')}</p><progress max={100} value={progreso} aria-label={textoSubiendo} /></>}{estado === 'revisando' && <><p>{t('anexo.revisando_fichero')}</p><progress aria-label={textoRevisando} /></>}</div>
    </section> : <>
      <p className="anexo-info" role="status">{t('anexo.leido')}</p>
      {analisis.lectura_anexo.antivirus !== 'limpio' ? <p className="anexo-aviso" role="status">{t('expediente.antivirus_no_revisado')}</p> : <p>{t('anexo.antivirus_limpio')}</p>}
      {analisis.derivacion_datos.contenido.avisos?.some((a) => a === 'anexo.pdf_sin_texto' || a === 'anexo.pdf_ilegible') && <p className="anexo-aviso" role="status">{t('anexo.lectura_manual')}</p>}
      {incierto && <p className="anexo-aviso" role="status">{t('anexo.confirmacion_incierta')}</p>}
      <form className="anexo-revision" onSubmit={(e) => { e.preventDefault(); void confirmar(); }} noValidate>
        <aside className="anexo-decision" aria-labelledby="anexo-decision-titulo">
          <section className="anexo-tarjeta"><h2 id="anexo-decision-titulo">{t('anexo.confirmar_titulo')}</h2>
            <p>{t('anexo.consecuencia')}</p>
            <div ref={resumen} tabIndex={-1} className={pendientes.length ? 'anexo-aviso' : undefined} role={faltan ? 'alert' : 'status'}>
              <h3>{t(pendientes.length ? 'anexo.faltan' : 'anexo.ubicacion_completa')}</h3>
              {pendientes.length > 0 && <ul>{pendientes.map((clave) => <li key={clave}><a href={`#anexo-${clave}`} onClick={(e) => { e.preventDefault(); enfocar(`anexo-${clave}`); }}>{t(`anexo.${clave}`)}</a></li>)}</ul>}
            </div><p>{t('anexo.vacios')} {numero(vacios)}{t('anexo.separador')}{t('anexo.dudosos')} {numero(dudosos)}</p><p>{t('anexo.completar_despues')}</p>
            <div className="anexo-acciones"><Boton accion={() => void confirmar()} ocupado={ocupado || cargaEtiquetas || !Object.keys(etiquetas).length} principal>{t(estado === 'guardando' ? 'anexo.guardando' : incierto ? 'anexo.recuperar' : 'anexo.confirmar')}</Boton><Boton accion={() => void descartar()} ocupado={bloqueado}>{t('anexo.descartar')}</Boton></div>
          </section>
        </aside>
        <div className="anexo-bloques"><fieldset disabled={bloqueado} className="anexo-tarjeta"><legend>{t('anexo.ubicacion_titulo')}</legend>
          <label htmlFor="anexo-tipo">{t('anexo.tipo')}</label><select id="anexo-tipo" value={tipo} onChange={(e) => ponerTipo(e.target.value)} required disabled={bloqueado || Boolean(analisis.derivacion_datos.contenido.tipo)}><option value="">{t('anexo.elegir')}</option><option value="anexo3">{t('anexo.anexo3')}</option><option value="anexo4">{t('anexo.anexo4')}</option></select>
          {(['municipio_id', 'centro_id', 'equipo_id'] as const).map((clave) => {
            const lista = clave === 'municipio_id' ? opciones?.municipios : clave === 'centro_id' ? opciones?.centros : opciones?.equipos;
            return <div className={`anexo-campo ${!ubicacion[clave] ? 'anexo-campo-pendiente' : ''}`} key={clave}><label htmlFor={`anexo-${clave}`}>{t(`anexo.${clave}`)}</label><select id={`anexo-${clave}`} required aria-describedby={`anexo-origen-${clave}`} value={ubicacion[clave]} onChange={(e) => clave === 'municipio_id' ? elegirMunicipio(e.target.value) : ponerUbicacion((v) => ({ ...v, [clave]: e.target.value }))}><option value="">{t('anexo.elegir')}</option>{lista?.map((o) => <option key={o.id} value={o.id}>{nombreOpcion(o)}</option>)}</select><p id={`anexo-origen-${clave}`} className="anexo-meta">{ubicacion[clave] && t(clave === 'municipio_id' ? (candidato ? 'anexo.origen_municipio' : 'anexo.origen_elegido') : ((clave === 'centro_id' ? municipio?.centro_id : municipio?.equipo_id) === ubicacion[clave] && ubicacion[clave] ? 'anexo.origen_asignacion' : 'anexo.origen_elegido'))}{clave === 'municipio_id' && candidato?.Fragmento && <q>{candidato.Fragmento}</q>}{!ubicacion[clave] && <span>{t('anexo.obligatorio')}</span>}</p></div>;
          })}
          {municipio?.fuera_ambito && <p className="anexo-aviso">{t('anexo.fuera_ambito')}</p>}
        </fieldset>
        {apartados.map((apartado) => {
          const indices = campos.flatMap((c, i) => c.codigo.startsWith(`${apartado}.`) ? [i] : []);
          if (!indices.length) return null;
          return <details className="anexo-tarjeta" key={apartado} open={apartado === 'menores' || apartado === 'familia'}><summary>{t(`anexo.apartado.${apartado}`)}</summary><fieldset disabled={bloqueado}><legend>{t(`anexo.apartado.${apartado}`)}</legend>
            {indices.map((i) => { const campo = campos[i]; const pendiente = !campo.valor.trim() || campo.dudoso; const original = analisis.derivacion_datos.contenido.campos?.[i]; const etiquetaValor = (valor: string) => Object.entries(etiquetas).find(([clave]) => clave.endsWith(`.${valor}`))?.[1] || valor;
              return <div className={`anexo-campo ${pendiente ? 'anexo-campo-pendiente' : ''}`} key={i}><label htmlFor={`anexo-campo-${i}`}>{t(`anexo.campo.${campo.codigo}`) || t('anexo.campo_generico')}{apartado === 'menores' && <span>{t('anexo.separador')}{t('anexo.menor')} {numero(campos.slice(0,i+1).filter((c) => c.codigo === 'menores.persona.nombre').length)}</span>}</label><textarea id={`anexo-campo-${i}`} rows={campo.codigo.startsWith('valoracion.') || campo.codigo.startsWith('observaciones.') ? 3 : 1} maxLength={50000} aria-describedby={`anexo-nota-${i}`} value={etiquetaValor(campo.valor)} onChange={(e) => ponerCampos((v) => v.map((c, indice) => indice === i ? { ...c, valor: e.target.value, dudoso: false, motivos: [] } : c))} /><p className="anexo-meta" id={`anexo-nota-${i}`}>{t(!campo.valor.trim() ? 'anexo.vacio' : campo.dudoso ? 'anexo.dudoso' : 'anexo.revisado')}{original?.dudoso && original.valor && <><span>{t('anexo.original')}</span><q>{etiquetaValor(original.valor)}</q></>}</p>{campo.dudoso && campo.valor.trim() && <Boton ocupado={bloqueado} accion={() => ponerCampos((v) => v.map((c, indice) => indice === i ? { ...c, dudoso: false, motivos: [] } : c))}>{t('anexo.confirmar_dato')}</Boton>}</div>;
            })}
          </fieldset>{apartado === 'menores' && <Boton ocupado={bloqueado || campos.filter((c) => c.codigo === 'menores.persona.nombre').length >= 15} accion={() => { const indice = campos.length; ponerCampos((v) => [...v, ...['nombre', 'nacimiento', 'escolarizacion', 'centro'].map((nombre) => ({ codigo: `menores.persona.${nombre}`, valor: '', pagina: 0, dudoso: true, motivos: [] }))]); requestAnimationFrame(() => enfocar(`anexo-campo-${indice}`)); }}>{t('anexo.anadir_menor')}</Boton>}</details>;
        })}
        </div>
      </form>
      {estado === 'guardando' && <p role="status">{t('anexo.guardando')}</p>}
    </>}
  </section>;
}
