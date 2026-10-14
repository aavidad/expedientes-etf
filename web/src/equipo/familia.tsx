import { useEffect, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useEtiquetas, useLectura, fecha, type Ficha } from './api';
import { EstadoLectura } from './equipo';
import { useFamiliaEdicion } from './familia-edicion';
import { centimos, nuevoAdulto, type Adulto, type DatosFamilia, type Familia } from './familia-modelo';
import { Campo, Errores, EstadoEdicion, enfocar, hoyCivil, type ErrorCampo } from '../fi1/comunes';
import { campoID, opciones, type Aviso } from '../fi1/modelo';

type Grupo = 'familia' | 'adultos';
const listasFamilia = { composicion_familiar: 'composicion_familiar', vivienda: 'vivienda', situacion_economica: 'situacion_economica', tramo_smi: 'tramo_ingresos_smi' };
const listasAdulto = { papel: 'papel_parental', sexo: 'sexo', nivel_educativo: 'nivel_educativo', situacion_laboral: 'situacion_laboral' };
const camposFamilia = ['adultos', 'composicion_familiar', 'familia_numerosa', 'vivienda', 'situacion_economica', 'ingresos_anuales_centimos', 'tramo_smi'];
const camposAdulto = ['papel', 'sexo', 'fecha_nacimiento', 'edad_declarada', 'nivel_educativo', 'situacion_laboral', 'es_tutor'];

export function FamiliaEquipo({ t, idioma, csrf, id, ficha, alCambiar }: PropsPantalla & { id: string; ficha: Ficha; alCambiar: () => void }) {
  const base = `/api/expedientes/${encodeURIComponent(id)}`;
  const sesion = useLectura<{ perfiles: string[] }>('/api/sesion');
  const etiquetas = useEtiquetas(idioma);
  const avisos = useLectura<{ avisos: Aviso[] }>(base + '/avisos');
  const edicion = useFamiliaEdicion<DatosFamilia>(base + '/familia', csrf, () => { alCambiar(); avisos.recargar(); });
  const [grupo, ponerGrupo] = useState<Grupo>();
  const [desde, ponerDesde] = useState(hoyCivil);
  const [errores, ponerErrores] = useState<ErrorCampo[]>([]);
  const [ingresos, ponerIngresos] = useState('');
  const [retirar, ponerRetirar] = useState<number>();
  const datos = edicion.datos;
  useEffect(() => { if (!edicion.sucio && datos) { ponerGrupo(undefined); ponerIngresos(datos.familia.ingresos_anuales_centimos === null ? '' : String(datos.familia.ingresos_anuales_centimos / 100).replace('.', ',')); } }, [datos, edicion.sucio]);
  useEffect(() => { if (datos && !sesion.cargando && !etiquetas.cargando && location.hash.startsWith('#fi1-')) enfocar(location.hash.slice(1)); }, [datos, sesion.cargando, etiquetas.cargando]);
  const puede = !!sesion.datos?.perfiles.includes('profesional_etf') && ['aceptada', 'en_intervencion'].includes(ficha.estado) && ['aceptada', 'en_intervencion'].includes(datos?.estado ?? '');
  const bloqueado = !puede || edicion.ocupado || edicion.conflicto || edicion.recuperar || !etiquetas.datos;
  const bloquear = (g: Grupo) => bloqueado || !!grupo && grupo !== g;
  function cambiar(g: Grupo, d: DatosFamilia) { ponerGrupo(g); edicion.editar(d); ponerErrores([]); }
  const cambiarFamilia = (c: Partial<Familia>) => { if (datos) cambiar('familia', { ...datos, familia: { ...datos.familia, ...c } }); };
  const cambiarAdulto = (i: number, c: Partial<Adulto>) => { if (datos) cambiar('adultos', { ...datos, adultos: datos.adultos.map((a, n) => n === i ? { ...a, ...c } : a) }); };
  function validar(g: Grupo) {
    if (!datos || bloquear(g)) return;
    const e: ErrorCampo[] = [];
    if (!desde || desde < datos.fecha_aceptacion.slice(0, 10) || desde > hoyCivil()) e.push({ id: 'gap1-vigencia', clave: 'fi1.vigencia_error' });
    const cantidad = centimos(ingresos);
    if (g === 'familia' && !Number.isFinite(cantidad ?? 0)) e.push({ id: campoID('ingresos_anuales_centimos'), clave: 'gap1.ingresos_error' });
    if (g === 'adultos') datos.adultos.forEach((a, i) => {
      const sujeto = a.id || `nuevo-${i}`;
      if (!a.papel) e.push({ id: campoID('papel', sujeto), clave: 'gap1.papel_error' });
      if (!a.sexo) e.push({ id: campoID('sexo', sujeto), clave: 'fi1.sexo_error' });
      if (a.fecha_nacimiento && a.fecha_nacimiento > desde) e.push({ id: campoID('fecha_nacimiento', sujeto), clave: 'fi1.nacimiento_error' });
      if (a.edad_declarada !== null && (!Number.isInteger(a.edad_declarada) || a.edad_declarada < 18 || a.edad_declarada > 130)) e.push({ id: campoID('edad_declarada', sujeto), clave: 'gap1.edad_error' });
    });
    ponerErrores(e);
    if (!e.length) void edicion.guardar({ version: datos.version, vigente_desde: desde, ...(g === 'familia' ? { familia: { ...datos.familia, ingresos_anuales_centimos: cantidad, tramo_smi: cantidad === null ? datos.familia.tramo_smi : null } } : { adultos: datos.adultos.map((a, i) => ({ ...a, orden: i + 1 })) }) }, g === 'familia' ? base : base + '/adultos', g === 'familia' ? 'PATCH' : 'PUT');
  }
  const campo = (clave: string, contenido: (id: string, aviso?: string) => React.ReactNode, sujeto?: string) => {
    const identificador = campoID(clave, sujeto);
    const error = errores.find((e) => e.id === identificador)?.clave;
    const pendiente = avisos.datos?.avisos.some((a) => a.campo === clave && (a.sujeto_id ?? '') === (sujeto ?? ''));
    return <Campo id={identificador} clave={'gap1.campo.' + clave} t={t} pendiente={pendiente} error={error}>{contenido(identificador, pendiente || error ? identificador + '-aviso' : undefined)}</Campo>;
  };
  const seleccion = (clave: string, lista: string, valor: string | null, editar: (v: string | null) => void, sujeto?: string) => campo(clave, (id, aviso) => <select id={id} value={valor ?? ''} aria-describedby={aviso} onChange={(e) => editar(e.target.value || null)}><option value="">{t('equipo.sin_dato')}</option>{opciones(etiquetas.datos ?? {}, lista).map((o) => <option value={o.codigo} key={o.codigo}>{o.etiqueta}</option>)}</select>, sujeto);
  const booleano = (clave: string, valor: boolean | null, editar: (v: boolean | null) => void, sujeto?: string) => campo(clave, (id, aviso) => <select id={id} value={valor === null ? '' : String(valor)} aria-describedby={aviso} onChange={(e) => editar(e.target.value === '' ? null : e.target.value === 'true')}><option value="">{t('equipo.sin_dato')}</option><option value="true">{t('fi1.si')}</option><option value="false">{t('fi1.no')}</option></select>, sujeto);
  const pendientes = avisos.datos?.avisos.filter((a, i, todos) => (camposFamilia.includes(a.campo) && !a.sujeto_id || camposAdulto.includes(a.campo) && datos?.adultos.some((p) => p.id === a.sujeto_id)) && todos.findIndex((b) => b.campo === a.campo && b.sujeto_id === a.sujeto_id) === i) ?? [];
  return <section className="fi1" aria-labelledby="equipo-familia-titulo"><h2 id="equipo-familia-titulo">{t('equipo.familia.datos')}</h2>
    <details><summary>{t('fi1.ayuda')}</summary><p>{t('gap1.ayuda')}</p></details>
    <EstadoLectura {...sesion} t={t} /><EstadoLectura {...etiquetas} t={t} /><EstadoLectura {...avisos} t={t} />
    {pendientes.length > 0 && <aside className="fi1-resumen" aria-labelledby="gap1-avisos"><h3 id="gap1-avisos">{t('fi1.avisos')}</h3><ul>{pendientes.map((a) => {
      const destino = campoID(a.campo, a.sujeto_id); const adulto = datos?.adultos.find((p) => p.id === a.sujeto_id);
      return <li key={destino}><a href={'#' + destino} onClick={(e) => { e.preventDefault(); enfocar(destino); }}>{t('gap1.campo.' + a.campo)}{adulto ? t('gap1.aviso_adulto').replace('{orden}', new Intl.NumberFormat(idioma).format(adulto.orden)).replace('{nombre}', adulto.nombre) : ''}</a></li>;
    })}</ul></aside>}
    <EstadoEdicion edicion={edicion} t={t} comparacion={edicion.actuales && <Comparacion datos={edicion.actuales} etiquetas={etiquetas.datos ?? {}} t={t} idioma={idioma} />} />
    {datos && <>
      {!puede && <p role="status">{t('fi1.solo_lectura')}</p>}
      <Errores errores={errores} t={t} />
      <Campo id="gap1-vigencia" clave="fi1.vigente_desde" t={t} error={errores.find((e) => e.id === 'gap1-vigencia')?.clave}><input id="gap1-vigencia" type="date" disabled={bloqueado} min={datos.fecha_aceptacion.slice(0, 10)} max={hoyCivil()} value={desde} aria-describedby={errores.some((e) => e.id === 'gap1-vigencia') ? 'gap1-vigencia-aviso' : undefined} onChange={(e) => { ponerDesde(e.target.value); edicion.editar(datos); }} /></Campo>
      {grupo && <p role="status">{t('gap1.guardar_apartado')}</p>}
      <form noValidate onSubmit={(e) => { e.preventDefault(); validar('familia'); }}><fieldset className="fi1-grupo" disabled={bloquear('familia')}><legend>{t('equipo.familia.datos')}</legend><div className="fi1-rejilla">
        {(['direccion', 'telefono_contacto', 'email_contacto'] as const).map((c) => <div key={c}>{campo(c, (id) => <input id={id} type={c === 'email_contacto' ? 'email' : 'text'} maxLength={c === 'direccion' ? 2000 : c === 'telefono_contacto' ? 100 : 254} value={datos.familia[c] ?? ''} onChange={(e) => cambiarFamilia({ [c]: e.target.value })} />)}</div>)}
        {(['composicion_familiar', 'vivienda', 'situacion_economica'] as const).map((c) => <div key={c}>{seleccion(c, listasFamilia[c], datos.familia[c], (v) => cambiarFamilia(c === 'composicion_familiar' ? { [c]: v, composicion_otras_texto: '' } : { [c]: v }))}</div>)}
        {datos.familia.composicion_familiar === 'OTRAS' && campo('composicion_otras_texto', (id) => <textarea id={id} maxLength={4000} value={datos.familia.composicion_otras_texto ?? ''} onChange={(e) => cambiarFamilia({ composicion_otras_texto: e.target.value })} />)}
        {booleano('familia_numerosa', datos.familia.familia_numerosa, (v) => cambiarFamilia({ familia_numerosa: v }))}
        {campo('ingresos_anuales_centimos', (id, aviso) => <input id={id} inputMode="decimal" value={ingresos} aria-describedby={aviso} onChange={(e) => { ponerIngresos(e.target.value); cambiarFamilia({ ingresos_anuales_centimos: centimos(e.target.value), tramo_smi: e.target.value.trim() ? null : datos.familia.tramo_smi }); }} />)}
        {seleccion('tramo_smi', listasFamilia.tramo_smi, datos.familia.tramo_smi, (v) => { ponerIngresos(''); cambiarFamilia({ tramo_smi: v, ingresos_anuales_centimos: null }); })}
      </div><p>{t('gap1.smi_aviso')}</p>{puede && <button type="submit" className={grupo === 'adultos' ? undefined : 'principal'}>{t(edicion.ocupado ? 'fi1.guardando' : 'gap1.guardar_familia')}</button>}</fieldset></form>
      <form noValidate onSubmit={(e) => { e.preventDefault(); validar('adultos'); }}><fieldset className="fi1-grupo" disabled={bloquear('adultos')}><legend id={campoID('adultos')} tabIndex={-1}>{t('equipo.familia.adultos')}</legend>
        {!datos.adultos.length && <p>{t('equipo.familia.sin_adultos')}</p>}
        {datos.adultos.map((a, i) => {
          const sujeto = a.id || `nuevo-${i}`; const titulo = [t('gap1.persona_adulta'), new Intl.NumberFormat(idioma).format(i + 1), a.nombre, a.apellidos].filter(Boolean).join(' ');
          return <details key={sujeto} open={i === 0 || !a.id}><summary>{titulo}</summary><div className="fi1-rejilla">
            {(['nombre', 'apellidos', 'telefono', 'email'] as const).map((c) => <div key={c}>{campo(c, (id) => <input id={id} type={c === 'email' ? 'email' : 'text'} maxLength={254} value={a[c] ?? ''} onChange={(e) => cambiarAdulto(i, { [c]: e.target.value })} />, sujeto)}</div>)}
            {(Object.keys(listasAdulto) as (keyof typeof listasAdulto)[]).map((c) => <div key={c}>{seleccion(c, listasAdulto[c], a[c], (v) => cambiarAdulto(i, c === 'situacion_laboral' ? { [c]: v, situacion_laboral_texto: '' } : c === 'nivel_educativo' ? { [c]: v } : { [c]: v ?? '' }), sujeto)}</div>)}
            {campo('fecha_nacimiento', (id, aviso) => <input id={id} type="date" max={desde} value={a.fecha_nacimiento ?? ''} aria-describedby={aviso} onChange={(e) => cambiarAdulto(i, { fecha_nacimiento: e.target.value || null, edad_declarada: e.target.value ? null : a.edad_declarada })} />, sujeto)}
            {campo('edad_declarada', (id, aviso) => <input id={id} type="number" min={18} max={130} step={1} value={a.edad_declarada ?? ''} aria-describedby={aviso} onChange={(e) => cambiarAdulto(i, { edad_declarada: e.target.value ? Number(e.target.value) : null, fecha_nacimiento: e.target.value ? null : a.fecha_nacimiento })} />, sujeto)}
            {booleano('es_tutor', a.es_tutor, (v) => cambiarAdulto(i, { es_tutor: v }), sujeto)}
            {a.situacion_laboral === 'OTRAS' && campo('situacion_laboral_texto', (id) => <textarea id={id} maxLength={4000} value={a.situacion_laboral_texto ?? ''} onChange={(e) => cambiarAdulto(i, { situacion_laboral_texto: e.target.value })} />, sujeto)}
            {campo('otros_datos', (id) => <textarea id={id} maxLength={4000} value={a.otros_datos ?? ''} onChange={(e) => cambiarAdulto(i, { otros_datos: e.target.value })} />, sujeto)}
          </div>{puede && <div className="fi1-acciones"><button type="button" onClick={() => ponerRetirar(i)}>{t('gap1.retirar')}</button></div>}
            {retirar === i && <div role="alert"><p>{t('gap1.retirar_aviso')}</p><button type="button" onClick={() => { cambiar('adultos', { ...datos, adultos: datos.adultos.filter((_, n) => n !== i) }); ponerRetirar(undefined); }}>{t('fi1.confirmar_retirar')}</button><button type="button" onClick={() => ponerRetirar(undefined)}>{t('fi1.cancelar')}</button></div>}
          </details>;
        })}
        {puede && <div className="fi1-acciones"><button type="button" disabled={datos.adultos.length >= 100} onClick={() => cambiar('adultos', { ...datos, adultos: [...datos.adultos, nuevoAdulto()] })}>{t('gap1.anadir')}</button><button type="submit" className={grupo === 'adultos' ? 'principal' : undefined}>{t(edicion.ocupado ? 'fi1.guardando' : 'gap1.guardar_adultos')}</button></div>}
      </fieldset></form>
    </>}
  </section>;
}
function Comparacion({ datos, etiquetas, t, idioma }: { datos: DatosFamilia; etiquetas: Record<string, string>; t: PropsPantalla['t']; idioma: string }) {
  const valor = (c: string, v: unknown) => v === null || v === '' ? t('equipo.sin_dato') : typeof v === 'boolean' ? t(v ? 'fi1.si' : 'fi1.no') : c === 'fecha_nacimiento' ? fecha(String(v), idioma) : c === 'ingresos_anuales_centimos' ? new Intl.NumberFormat(idioma, { style: 'currency', currency: 'EUR' }).format(Number(v) / 100) : c in listasFamilia || c in listasAdulto ? etiquetas[({ ...listasFamilia, ...listasAdulto } as Record<string, string>)[c] + '.' + String(v)] ?? t('equipo.sin_dato') : String(v);
  const lista = (objeto: object) => <dl className="equipo-detalle">{Object.entries(objeto).filter(([c]) => !['id', 'orden'].includes(c)).map(([c, v]) => <div key={c}><dt>{t('gap1.campo.' + c)}</dt><dd>{valor(c, v)}</dd></div>)}</dl>;
  return <div><h4>{t('equipo.familia.datos')}</h4>{lista(datos.familia)}<h4>{t('equipo.familia.adultos')}</h4>{datos.adultos.map((a) => <section key={a.id}>{lista(a)}</section>)}</div>;
}
