import { useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useEtiquetas, useLectura, fecha, type Ficha } from './api';
import { EstadoLectura } from './equipo';
import { hoy, useEdicion } from './edicion';

type Adulto = { id: string; nombre: string; apellidos: string };
type Item = { codigo: string; tipo: string; minimo: number; maximo: number; paso: number; valores?: string[] };
type Escala = { id: string; instrumento: string; version: number; items: Item[] };
type Respuesta = { item: string; valor_numerico?: number; valor_codigo?: string };
type Aplicacion = { id: string; instrumento: string; momento: string; sujeto: { tipo: string; adulto_id?: string }; fecha?: string; version_escala_id: string; descripcion_cualitativa: string; version: number; puntuaciones: Respuesta[] };
type Instrumento = { id: string; codigo: string; fecha: string; texto_otros?: string; sujeto?: { adulto_id?: string }; momento?: string };
type Registro = { instrumentos: Instrumento[]; escalas: Aplicacion[]; definiciones?: { codigos: string[]; escalas: Escala[] } };

export function EvaluacionEquipo({ id, refrescando, ficha, t, idioma, csrf, alCambiar }: PropsPantalla & { id: string; refrescando: boolean; ficha: Ficha; alCambiar: () => void }) {
  const base = `/api/expedientes/${encodeURIComponent(id)}`;
  const lectura = useLectura<Registro>(`${base}/evaluaciones`);
  const familia = useLectura<{ equipo_id: string; adultos: Adulto[] }>(`${base}/familia`);
  const catalogo = useEtiquetas(idioma);
  const etiqueta = (clave: string) => catalogo.datos?.[clave] || t('equipo.sin_dato');
  const edicion = useEdicion(id, csrf);
  const [adulto, ponerAdulto] = useState('');
  const [momento, ponerMomento] = useState('inicial');
  const [instrumento, ponerInstrumento] = useState('');
  const [fechaAplicacion, ponerFecha] = useState('');
  const [otros, ponerOtros] = useState('');
  const [descripcion, ponerDescripcion] = useState('');
  const [respuestas, ponerRespuestas] = useState<Record<string, string>>({});
  const [aplicacionId, ponerAplicacionId] = useState('');
  const [nuevoId, ponerNuevoId] = useState(() => crypto.randomUUID());
  const definiciones = lectura.datos?.definiciones;
  const guardada = lectura.datos?.escalas.find((a) => a.id === aplicacionId);
  const escala = definiciones?.escalas.find((e) => guardada ? e.id === guardada.version_escala_id : e.instrumento === instrumento);
  const existente = lectura.datos?.escalas.find((a) => a.instrumento === instrumento && a.momento === momento && a.sujeto.adulto_id === adulto);
  const repetida = !!escala && !!existente && !guardada;
  const puedeEditar = ['aceptada', 'en_intervencion'].includes(ficha.estado);
  const bloquear = edicion.ocupado || refrescando || edicion.conflicto || lectura.cargando;
  const cambiar = () => edicion.editar();
  function limpiar() { ponerInstrumento(''); ponerFecha(''); ponerOtros(''); ponerDescripcion(''); ponerRespuestas({}); ponerAplicacionId(''); ponerNuevoId(crypto.randomUUID()); }
  function abrir(a: Aplicacion) {
    edicion.verCambios(() => {
      ponerAplicacionId(a.id); ponerAdulto(a.sujeto.adulto_id ?? ''); ponerMomento(a.momento); ponerInstrumento(a.instrumento); ponerFecha(a.fecha?.slice(0, 10) ?? ''); ponerDescripcion(a.descripcion_cualitativa);
      ponerRespuestas(Object.fromEntries(a.puntuaciones.map((p) => [p.item, String(p.valor_numerico ?? p.valor_codigo ?? '')])));
    });
  }
  async function guardar() {
    if (!familia.datos || !adulto || !instrumento) return;
    // Una combinación existente conserva su ID y exige su versión leída.
    if (repetida) return;
    const comun = { id: guardada?.id ?? nuevoId, expediente_id: id, equipo_id: familia.datos.equipo_id, sujeto: { tipo: 'figura_parental', adulto_id: adulto }, momento, fecha: `${fechaAplicacion}T00:00:00Z` };
    const cambio = escala ? { version_esperada: guardada?.version ?? 0, aplicacion: {
      ...comun, instrumento, version_escala_id: escala.id, descripcion_cualitativa: descripcion, version: (guardada?.version ?? 0) + 1,
      puntuaciones: escala.items.filter((i) => respuestas[i.codigo] !== undefined && respuestas[i.codigo] !== '').map((i) => i.tipo === 'numerico' ? { item: i.codigo, valor_numerico: Number(respuestas[i.codigo]) } : { item: i.codigo, valor_codigo: respuestas[i.codigo] }),
    } } : { version_esperada: 0, instrumento: { ...comun, codigo: instrumento, texto_otros: instrumento === 'OTROS' ? otros : '' } };
    const r = await edicion.enviar(`${base}/evaluaciones`, cambio);
    if (r) { limpiar(); lectura.recargar(); alCambiar(); }
  }
  const nombreAdulto = (adultoId?: string) => {
    const persona = familia.datos?.adultos.find((a) => a.id === adultoId);
    return persona ? `${persona.nombre} ${persona.apellidos}` : t('fi2.evaluacion.persona_no_vigente');
  };
  return <section aria-labelledby="fi2-evaluacion">
    <h2 id="fi2-evaluacion">{t('fi2.evaluacion.titulo')}</h2>
    <p className="equipo-aviso">{t('fi2.evaluacion.sin_baremo')}</p>
    <EstadoLectura {...lectura} t={t} /><EstadoLectura {...familia} t={t} /><EstadoLectura {...catalogo} t={t} />
    {lectura.datos && familia.datos && <>
      {!familia.datos.adultos.length && <p>{t('fi2.evaluacion.sin_adultos')}</p>}
      {puedeEditar && familia.datos.adultos.length > 0 && definiciones && <form onSubmit={(e) => { e.preventDefault(); void guardar(); }}>
        <fieldset disabled={bloquear} className="fi2-campos"><legend>{t(guardada ? 'fi2.evaluacion.editar' : 'fi2.evaluacion.registrar')}</legend>
          <label className="equipo-campo" htmlFor="fi2-adulto">{t('fi2.evaluacion.adulto')}<select id="fi2-adulto" required disabled={!!guardada} value={adulto} onChange={(e) => { ponerAdulto(e.target.value); cambiar(); }}><option value="">{t('fi2.evaluacion.elige_adulto')}</option>{familia.datos.adultos.map((a) => <option value={a.id} key={a.id}>{a.nombre} {a.apellidos}</option>)}</select></label>
          <label className="equipo-campo" htmlFor="fi2-momento">{t('fi2.evaluacion.momento')}<select id="fi2-momento" disabled={!!guardada} value={momento} onChange={(e) => { ponerMomento(e.target.value); cambiar(); }}>{['inicial', 'intermedia', 'final'].map((m) => <option value={m} key={m}>{t(`fi2.momento.${m}`)}</option>)}</select></label>
          <label className="equipo-campo" htmlFor="fi2-instrumento">{t('fi2.evaluacion.instrumento')}<select id="fi2-instrumento" required disabled={!!guardada} value={instrumento} onChange={(e) => { ponerInstrumento(e.target.value); ponerOtros(''); ponerRespuestas({}); cambiar(); }}><option value="">{t('fi2.evaluacion.elige_instrumento')}</option>{definiciones.codigos.map((c) => <option key={c} value={c}>{etiqueta(`evaluacion.instrumento.${c}`)}</option>)}</select></label>
          <label className="equipo-campo" htmlFor="fi2-fecha-aplicacion">{t('fi2.evaluacion.fecha')}<input id="fi2-fecha-aplicacion" type="date" required max={hoy()} value={fechaAplicacion} aria-describedby={edicion.error ? 'fi2-error-evaluacion' : undefined} onChange={(e) => { ponerFecha(e.target.value); cambiar(); }} /></label>
          {instrumento === 'OTROS' && <label className="equipo-campo" htmlFor="fi2-otros">{t('fi2.evaluacion.otros')}<input id="fi2-otros" required maxLength={500} value={otros} onChange={(e) => { ponerOtros(e.target.value); cambiar(); }} /></label>}
          {repetida && existente && <div role="status"><p>{t('fi2.evaluacion.duplicada')}</p><button type="button" onClick={() => abrir(existente)}>{t('fi2.evaluacion.ver_respuestas')}</button></div>}
          {instrumento && !escala && <p role="status">{t('fi2.evaluacion.items_pendientes')}</p>}
          {escala && <><fieldset><legend>{t('fi2.evaluacion.respuestas')}</legend>{escala.items.map((i, indice) => <label className="equipo-campo" key={i.codigo} htmlFor={`fi2-item-${indice}`}>{etiqueta(`evaluacion.item.${instrumento}.${i.codigo}`)}
            {i.tipo === 'numerico' ? <input id={`fi2-item-${indice}`} type="number" min={i.minimo} max={i.maximo} step={i.paso} value={respuestas[i.codigo] ?? ''} onChange={(e) => { ponerRespuestas({ ...respuestas, [i.codigo]: e.target.value }); cambiar(); }} /> : <select id={`fi2-item-${indice}`} value={respuestas[i.codigo] ?? ''} onChange={(e) => { ponerRespuestas({ ...respuestas, [i.codigo]: e.target.value }); cambiar(); }}><option value="">{t('fi2.evaluacion.sin_respuesta')}</option>{i.valores?.map((v) => <option value={v} key={v}>{etiqueta(`evaluacion.respuesta.${instrumento}.${v}`)}</option>)}</select>}
          </label>)}</fieldset><label className="equipo-campo" htmlFor="fi2-descripcion">{t('fi2.evaluacion.descripcion')}<textarea id="fi2-descripcion" maxLength={4000} value={descripcion} onChange={(e) => { ponerDescripcion(e.target.value); cambiar(); }} /></label></>}
          <button type="submit" className="principal" disabled={repetida}>{t(edicion.ocupado ? 'fi2.espera' : 'fi2.evaluacion.guardar')}</button>
          {guardada && <button type="button" onClick={() => edicion.verCambios(limpiar)}>{t('fi2.evaluacion.nuevo')}</button>}
        </fieldset>
      </form>}
      <h3>{t('fi2.evaluacion.registradas')}</h3>
      {!lectura.datos.instrumentos.length && !lectura.datos.escalas.length && <p>{t('fi2.evaluacion.vacio')}</p>}
      <ul className="equipo-sucesos">{lectura.datos.instrumentos.map((i) => <li key={i.id}><strong>{etiqueta(`evaluacion.instrumento.${i.codigo}`)}</strong><span>{i.sujeto ? nombreAdulto(i.sujeto.adulto_id) : t('fi2.evaluacion.familia')}</span>{i.momento && <span>{t(`fi2.momento.${i.momento}`)}</span>}<time dateTime={i.fecha}>{fecha(i.fecha, idioma)}</time>{i.texto_otros && <span>{i.texto_otros}</span>}</li>)}
        {lectura.datos.escalas.map((a) => <li key={a.id}><strong>{etiqueta(`evaluacion.instrumento.${a.instrumento}`)}</strong><span>{a.sujeto.tipo === 'figura_parental' ? nombreAdulto(a.sujeto.adulto_id) : t('fi2.evaluacion.familia')}</span><span>{t(`fi2.momento.${a.momento}`)}</span>{a.fecha && <time dateTime={a.fecha}>{fecha(a.fecha, idioma)}</time>}<details><summary>{t('fi2.evaluacion.respuestas')}</summary>{a.descripcion_cualitativa && <p>{a.descripcion_cualitativa}</p>}<dl className="equipo-detalle">{a.puntuaciones.map((p) => <div key={p.item}><dt>{etiqueta(`evaluacion.item.${a.instrumento}.${p.item}`)}</dt><dd>{p.valor_numerico !== undefined ? new Intl.NumberFormat(idioma).format(p.valor_numerico) : etiqueta(`evaluacion.respuesta.${a.instrumento}.${p.valor_codigo}`)}</dd></div>)}</dl>{!a.puntuaciones.length && <p>{t('fi2.evaluacion.sin_respuesta')}</p>}</details>{puedeEditar && a.sujeto.tipo === 'figura_parental' && a.momento !== 'derivacion' && <button type="button" disabled={bloquear} onClick={() => abrir(a)}>{t('fi2.evaluacion.ver_respuestas')}</button>}</li>)}
      </ul>
    </>}
    {edicion.error && <p id="fi2-error-evaluacion" role="alert">{t(edicion.error) || t('fi2.error')}</p>}
    {edicion.conflicto && <button type="button" onClick={() => edicion.verCambios(() => { limpiar(); lectura.recargar(); familia.recargar(); alCambiar(); })}>{t('fi2.ver_cambios')}</button>}
    <p role="status">{edicion.resultado && t(edicion.resultado)}</p>
  </section>;
}
