import { useEffect, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useEtiquetas, useLectura, type Ficha } from '../equipo/api';
import { useEdicion } from './edicion';
import { campoID, nuevoMenor, opciones, type FichaMenores, type Menor } from './modelo';
import { AvisosCampos, BotonGuardar, Campo, Errores, EstadoEdicion, enfocar, hoyCivil, type ErrorCampo, type Textos } from './comunes';

const camposAviso = ['sexo', 'fecha_nacimiento', 'subprograma', 'propuesto_atencion_temprana'];
export function Menores({ id, ficha, alCambiar, ...props }: PropsPantalla & { id: string; ficha: Ficha; alCambiar: () => void }) {
  const { t, csrf, idioma } = props;
  const sesion = useLectura<{ perfiles: string[] }>('/api/sesion');
  const etiquetas = useEtiquetas(idioma);
  const edicion = useEdicion<FichaMenores>(`/api/expedientes/${encodeURIComponent(id)}/menores`, csrf, alCambiar);
  const [desde, ponerDesde] = useState(hoyCivil);
  const [errores, ponerErrores] = useState<ErrorCampo[]>([]);
  const datos = edicion.datos;
  useEffect(() => { if (datos && !sesion.cargando && !etiquetas.cargando && location.hash.startsWith('#fi1-')) enfocar(location.hash.slice(1)); }, [datos, sesion.cargando, etiquetas.cargando]);
  const [retirar, ponerRetirar] = useState<number>();
  const puede = ficha.estado !== 'cerrada' && datos?.estado !== 'cerrada' && !!sesion.datos?.perfiles.includes('profesional_etf');
  const bloqueado = !puede || edicion.ocupado || edicion.recuperar || edicion.conflicto || !etiquetas.datos;
  function cambiar(menores: Menor[]) { if (datos) { edicion.editar({ ...datos, menores }); ponerErrores([]); } }
  function validar() {
    if (!datos) return;
    const e: ErrorCampo[] = [];
    if (!desde || desde < datos.fecha_aceptacion.slice(0, 10) || desde > hoyCivil()) e.push({ id: 'fi1-vigencia', clave: 'fi1.vigencia_error' });
    datos.menores.forEach((m, indice) => {
      const sujeto = m.id || String(indice);
      if (!m.sexo) e.push({ id: campoID('sexo', sujeto), clave: 'fi1.sexo_error' });
      if (m.fecha_nacimiento && m.fecha_nacimiento > desde) e.push({ id: campoID('fecha_nacimiento', sujeto), clave: 'fi1.nacimiento_error' });
      if (!m.subprograma) e.push({ id: campoID('subprograma', sujeto), clave: 'fi1.subprograma_error' });
    });
    ponerErrores(e);
    if (!e.length) void edicion.guardar({ version: datos.version, vigente_desde: desde, menores: datos.menores.map((m, i) => ({ ...m, orden: i + 1 })) });
  }
  return <section className="fi1" aria-labelledby="fi1-menores-titulo"><h2 id="fi1-menores-titulo">{t('fi1.menores')}</h2>
    <details><summary>{t('fi1.ayuda')}</summary><p>{t('fi1.menores_ayuda')}</p></details>
    <AvisosCampos id={id} campos={camposAviso} sujetos={datos?.menores} t={t} revision={ficha.version} />
    <EstadoEdicion edicion={edicion} t={t} comparacion={edicion.actuales && <DatosMenores datos={edicion.actuales} etiquetas={etiquetas.datos ?? {}} t={t} />} />
    {etiquetas.cargando && <p role="status">{t('fi1.catalogo_cargando')}</p>}
    {etiquetas.error && <div role="alert"><p>{t('fi1.catalogo_error')}</p><button type="button" onClick={etiquetas.recargar}>{t('fi1.reintentar')}</button></div>}
    {datos && <form noValidate onSubmit={(e) => { e.preventDefault(); validar(); }}>
      {!puede && <p role="status">{t('fi1.solo_lectura')}</p>}
      <Errores errores={errores} t={t} />
      <fieldset className="fi1-grupo" disabled={bloqueado}><legend>{t('fi1.vigencia')}</legend><Campo id="fi1-vigencia" clave="fi1.vigente_desde" t={t} error={errores.find((e) => e.id === 'fi1-vigencia')?.clave}>
        <input id="fi1-vigencia" type="date" min={datos.fecha_aceptacion.slice(0, 10)} max={hoyCivil()} value={desde} aria-describedby={errores.some((e) => e.id === 'fi1-vigencia') ? 'fi1-vigencia-aviso' : undefined} onChange={(e) => { ponerDesde(e.target.value); edicion.editar(datos); }} />
      </Campo></fieldset>
      {!datos.menores.length && <p>{t(puede ? 'fi1.menores_vacio' : 'fi1.menores_vacio_lectura')}</p>}
      {datos.menores.map((m, indice) => {
        const sujeto = m.id || String(indice);
        const editar = (cambio: Partial<Menor>) => cambiar(datos.menores.map((actual, i) => i === indice ? { ...actual, ...cambio } : actual));
        const campo = (clave: keyof Menor, contenido: (id: string, aviso: string | undefined) => React.ReactNode, pendiente = false) => {
          const identificador = campoID(clave, sujeto); const error = errores.find((e) => e.id === identificador)?.clave;
          return <Campo id={identificador} clave={'fi1.campo.' + clave} t={t} pendiente={pendiente} error={error}>{contenido(identificador, pendiente || error ? identificador + '-aviso' : undefined)}</Campo>;
        };
        const titulo = [t('fi1.menor'), new Intl.NumberFormat(idioma).format(indice + 1), m.nombre, m.apellidos].filter(Boolean).join(' ');
        return <details key={sujeto} open={indice === 0 || !m.id}><summary>{titulo}</summary><fieldset className="fi1-grupo" disabled={bloqueado}><legend>{titulo}</legend><div className="fi1-rejilla">
          {(['nombre', 'apellidos', 'centro_educativo'] as const).map((clave) => <div key={clave}>{campo(clave, (id) => <input id={id} value={m[clave]} maxLength={254} onChange={(e) => editar({ [clave]: e.target.value })} />)}</div>)}
          {campo('sexo', (id, aviso) => <select id={id} aria-describedby={aviso} value={m.sexo} onChange={(e) => editar({ sexo: e.target.value })}><option value="">{t('fi1.sin_indicar')}</option>{['H', 'M'].map((s) => <option value={s} key={s}>{t('fi1.sexo.' + s.toLowerCase())}</option>)}</select>, !m.sexo)}
          {campo('fecha_nacimiento', (id, aviso) => <input id={id} type="date" max={desde} value={m.fecha_nacimiento ?? ''} aria-describedby={aviso} onChange={(e) => editar({ fecha_nacimiento: e.target.value || null })} />, !m.fecha_nacimiento)}
          {campo('nivel_escolarizacion', (id) => <select id={id} value={m.nivel_escolarizacion ?? ''} onChange={(e) => editar({ nivel_escolarizacion: e.target.value || null })}><option value="">{t('fi1.sin_indicar')}</option>{opciones(etiquetas.datos ?? {}, 'nivel_escolarizacion').map((o) => <option value={o.codigo} key={o.codigo}>{o.etiqueta}</option>)}</select>)}
          {campo('subprograma', (id, aviso) => <select id={id} aria-describedby={aviso} value={m.subprograma} onChange={(e) => editar({ subprograma: e.target.value })}><option value="">{t('fi1.sin_indicar')}</option>{opciones(etiquetas.datos ?? {}, 'subprograma').map((o) => <option value={o.codigo} key={o.codigo}>{o.etiqueta}</option>)}</select>, !m.subprograma)}
          {campo('propuesto_atencion_temprana', (id, aviso) => <select id={id} aria-describedby={aviso} value={m.propuesto_atencion_temprana === null ? '' : String(m.propuesto_atencion_temprana)} onChange={(e) => editar({ propuesto_atencion_temprana: e.target.value === '' ? null : e.target.value === 'true' })}><option value="">{t('fi1.sin_indicar')}</option><option value="true">{t('fi1.si')}</option><option value="false">{t('fi1.no')}</option></select>, m.propuesto_atencion_temprana === null)}
          {campo('centro_proteccion_codigo', (id) => <select id={id} value={m.centro_proteccion_codigo ?? ''} onChange={(e) => editar({ centro_proteccion_codigo: e.target.value || null, centro_proteccion_texto: '' })}><option value="">{t('fi1.centro_libre')}</option>{opciones(etiquetas.datos ?? {}, 'centro_proteccion').map((o) => <option value={o.codigo} key={o.codigo}>{o.etiqueta}</option>)}</select>)}
          {!m.centro_proteccion_codigo && campo('centro_proteccion_texto', (id) => <input id={id} value={m.centro_proteccion_texto} maxLength={254} onChange={(e) => editar({ centro_proteccion_texto: e.target.value })} />)}
        </div>{puede && <div className="fi1-acciones"><button type="button" onClick={() => ponerRetirar(indice)}>{t('fi1.retirar_menor')}</button></div>}
          {retirar === indice && <div role="alert"><p>{t('fi1.retirar_detalle')}</p><button type="button" onClick={() => { cambiar(datos.menores.filter((_, i) => i !== indice)); ponerRetirar(undefined); }}>{t('fi1.confirmar_retirar')}</button><button type="button" onClick={() => ponerRetirar(undefined)}>{t('fi1.cancelar')}</button></div>}
        </fieldset></details>;
      })}
      {puede && <div className="fi1-acciones">{datos.menores.length < 15 ? <button type="button" disabled={bloqueado} onClick={() => cambiar([...datos.menores, nuevoMenor(datos.menores.length + 1, datos.subprograma)])}>{t('fi1.anadir_menor')}</button> : <p role="status">{t('fi1.limite')}</p>}<BotonGuardar t={t} ocupado={bloqueado} /></div>}
    </form>}
  </section>;
}
function DatosMenores({ datos, etiquetas, t }: { datos: FichaMenores; etiquetas: Record<string, string>; t: Textos }) {
  return <div>{datos.menores.map((m) => <fieldset className="fi1-grupo" key={m.id}><legend>{m.nombre} {m.apellidos}</legend><dl>{Object.entries(m).filter(([k]) => !['id', 'orden'].includes(k)).map(([k, v]) => <div key={k}><dt>{t('fi1.campo.' + k)}</dt><dd>{v === null || v === '' ? t('fi1.sin_indicar') : typeof v === 'boolean' ? t(v ? 'fi1.si' : 'fi1.no') : k === 'sexo' ? t('fi1.sexo.' + String(v).toLowerCase()) : ['subprograma', 'nivel_escolarizacion', 'centro_proteccion_codigo'].includes(k) ? etiquetas[(k === 'centro_proteccion_codigo' ? 'centro_proteccion' : k) + '.' + String(v)] ?? t('fi1.sin_indicar') : k === 'fecha_nacimiento' ? new Intl.DateTimeFormat(document.documentElement.lang, { timeZone: 'UTC', day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(String(v))) : String(v)}</dd></div>)}</dl></fieldset>)}</div>;
}
