import { useEffect, useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useEtiquetas, useLectura, type Ficha } from '../equipo/api';
import { useEdicion } from './edicion';
import { campoID, listas, nuevaIntervencion, opciones, type FichaIntervencion, type FichaMenores, type Intervencion as Datos, type Lista, type Marca } from './modelo';
import { AvisosCampos, BotonGuardar, Campo, Errores, EstadoEdicion, enfocar, hoyCivil, type ErrorCampo, type Textos } from './comunes';

export function Intervencion({ id, ficha, alCambiar, ...props }: PropsPantalla & { id: string; ficha: Ficha; alCambiar: () => void }) {
  const { t, csrf, idioma } = props;
  const sesion = useLectura<{ perfiles: string[] }>('/api/sesion');
  const etiquetas = useEtiquetas(idioma);
  const menores = useLectura<FichaMenores>(`/api/expedientes/${encodeURIComponent(id)}/menores`);
  const edicion = useEdicion<FichaIntervencion>(`/api/expedientes/${encodeURIComponent(id)}/intervencion`, csrf, alCambiar);
  const [desde, ponerDesde] = useState(hoyCivil);
  const [errores, ponerErrores] = useState<ErrorCampo[]>([]);
  const datos = edicion.datos;
  useEffect(() => { if (datos && !sesion.cargando && !etiquetas.cargando && !menores.cargando && location.hash.startsWith('#fi1-')) enfocar(location.hash.slice(1)); }, [datos, sesion.cargando, etiquetas.cargando, menores.cargando]);
  const intervencion = datos?.intervencion ?? nuevaIntervencion();
  const puede = ficha.estado !== 'cerrada' && datos?.estado !== 'cerrada' && !!sesion.datos?.perfiles.includes('profesional_etf');
  const bloqueado = !puede || edicion.ocupado || edicion.recuperar || edicion.conflicto || !etiquetas.datos || !menores.datos;
  function editar(cambio: Partial<Datos>) { if (datos) { edicion.editar({ ...datos, intervencion: { ...intervencion, ...cambio } }); ponerErrores([]); } }
  function cambiarLista(lista: Lista, marcas: Marca[] | null) { editar({ listas: { ...intervencion.listas, [lista]: marcas } }); }
  function validar() {
    if (!datos) return;
    const e: ErrorCampo[] = [];
    if (!desde || desde < datos.fecha_aceptacion.slice(0, 10) || desde > hoyCivil()) e.push({ id: 'fi1-vigencia', clave: 'fi1.vigencia_error' });
    ponerErrores(e);
    if (!e.length) void edicion.guardar({ version: datos.version, vigente_desde: desde, intervencion });
  }
  return <section className="fi1" aria-labelledby="fi1-intervencion-titulo"><h2 id="fi1-intervencion-titulo">{t('fi1.intervencion')}</h2>
    <details><summary>{t('fi1.ayuda')}</summary><p>{t('fi1.intervencion_ayuda')}</p></details>
    <AvisosCampos id={id} campos={['fase', ...listas]} t={t} revision={ficha.version} />
    <EstadoEdicion edicion={edicion} t={t} comparacion={edicion.actuales && <ResumenIntervencion datos={edicion.actuales.intervencion ?? nuevaIntervencion()} etiquetas={etiquetas.datos ?? {}} menores={menores.datos} t={t} />} />
    {(etiquetas.cargando || menores.cargando || sesion.cargando) && <p role="status">{t('fi1.catalogo_cargando')}</p>}
    {(etiquetas.error || menores.error || sesion.error) && <div role="alert"><p>{t('fi1.catalogo_error')}</p><button type="button" onClick={() => { etiquetas.recargar(); menores.recargar(); sesion.recargar(); }}>{t('fi1.reintentar')}</button></div>}
    {datos && <form noValidate onSubmit={(e) => { e.preventDefault(); validar(); }}>
      {!puede && !sesion.cargando && <p role="status">{t('fi1.solo_lectura')}</p>}
      {!datos.intervencion && <p>{t('fi1.intervencion_vacia')}</p>}
      <Errores errores={errores} t={t} />
      <fieldset className="fi1-grupo" disabled={bloqueado}><legend>{t('fi1.vigencia')}</legend><div className="fi1-rejilla">
        <Campo id="fi1-vigencia" clave="fi1.vigente_desde" t={t} error={errores.find((e) => e.id === 'fi1-vigencia')?.clave}><input id="fi1-vigencia" type="date" min={datos.fecha_aceptacion.slice(0, 10)} max={hoyCivil()} value={desde} aria-describedby={errores.length ? 'fi1-vigencia-aviso' : undefined} onChange={(e) => { ponerDesde(e.target.value); editar({}); }} /></Campo>
        <Campo id={campoID('fase')} clave="fi1.campo.fase" t={t} pendiente={!intervencion.fase}><select id={campoID('fase')} value={intervencion.fase ?? ''} aria-describedby={!intervencion.fase ? campoID('fase') + '-aviso' : undefined} onChange={(e) => editar({ fase: e.target.value || null })}><option value="">{t('fi1.sin_indicar')}</option>{opciones(etiquetas.datos ?? {}, 'fase_intervencion').map((o) => <option value={o.codigo} key={o.codigo}>{o.etiqueta}</option>)}</select></Campo>
      </div></fieldset>
      {listas.map((lista) => {
        const marcas = intervencion.listas[lista];
        return <details key={lista}><summary>{t('fi1.campo.' + lista)}<span className="fi1-separacion">{marcas === null ? t('fi1.sin_indicar') : t('fi1.opciones_marcadas').replace('{cantidad}', new Intl.NumberFormat(idioma).format(marcas.length))}</span></summary><fieldset className="fi1-grupo" disabled={bloqueado}><legend>{t('fi1.campo.' + lista)}</legend>
          <Campo id={campoID(lista)} clave="fi1.datos_lista" t={t} pendiente={marcas === null}><select id={campoID(lista)} value={marcas === null ? '' : 'conocido'} aria-describedby={marcas === null ? campoID(lista) + '-aviso' : undefined} onChange={(e) => cambiarLista(lista, e.target.value ? [] : null)}><option value="">{t('fi1.sin_indicar')}</option><option value="conocido">{t('fi1.lista_conocida')}</option></select></Campo>
          {marcas !== null && <>{!marcas.length && <p>{t('fi1.ninguna')}</p>}
            <div className="fi1-rejilla">{opciones(etiquetas.datos ?? {}, lista).map((o) => {
              const marca = marcas.find((m) => m.codigo === o.codigo);
              const cambiar = (nueva: Marca) => cambiarLista(lista, marcas.map((m) => m.codigo === o.codigo ? nueva : m));
              return <div className="fi1-opcion" key={o.codigo}><label className="fi1-casilla"><input type="checkbox" checked={!!marca} onChange={(e) => cambiarLista(lista, e.target.checked ? [...marcas, { codigo: o.codigo, menores_excluidos: [] }] : marcas.filter((m) => m.codigo !== o.codigo))} /><span>{o.etiqueta}</span></label>
                {marca && <>
                  {o.codigo === 'OTROS' && <Campo id={`fi1-${lista}-${o.codigo}-texto`} clave="fi1.otros_texto" t={t}><textarea id={`fi1-${lista}-${o.codigo}-texto`} maxLength={4000} value={marca.texto_otros ?? ''} onChange={(e) => cambiar({ ...marca, texto_otros: e.target.value })} /></Campo>}
                  <details><summary>{t('fi1.aplicacion')}</summary><p>{t('fi1.aplicacion_detalle')}</p>
                    {menores.datos?.menores.length ? menores.datos.menores.map((m) => <label className="fi1-casilla" key={m.id}><input type="checkbox" checked={!marca.menores_excluidos.includes(m.id)} onChange={(e) => cambiar({ ...marca, menores_excluidos: e.target.checked ? marca.menores_excluidos.filter((i) => i !== m.id) : [...marca.menores_excluidos, m.id] })} /><span>{m.nombre || t('fi1.menor')} {m.apellidos} {!m.nombre && new Intl.NumberFormat(idioma).format(m.orden)}</span></label>) : <p>{t('fi1.excepciones_vacias')}</p>}
                  </details>
                </>}
              </div>;
            })}</div>
          </>}
        </fieldset></details>;
      })}
      {puede && <BotonGuardar t={t} ocupado={bloqueado} />}
    </form>}
  </section>;
}
function ResumenIntervencion({ datos, etiquetas, menores, t }: { datos: Datos; etiquetas: Record<string, string>; menores?: FichaMenores; t: Textos }) {
  return <dl><dt>{t('fi1.campo.fase')}</dt><dd>{etiquetas['fase_intervencion.' + datos.fase] ?? t('fi1.sin_indicar')}</dd>{listas.map((lista) => <div key={lista}><dt>{t('fi1.campo.' + lista)}</dt><dd>{datos.listas[lista] === null ? t('fi1.sin_indicar') : !datos.listas[lista]?.length ? t('fi1.ninguna') : <ul>{datos.listas[lista]?.map((m) => <li key={m.codigo}>{etiquetas[lista + '.' + m.codigo] ?? t('fi1.sin_indicar')}{m.texto_otros && <p>{m.texto_otros}</p>}{m.menores_excluidos.length > 0 && <p>{t('fi1.excluidos')} {m.menores_excluidos.map((id) => { const menor = menores?.menores.find((n) => n.id === id); return menor ? `${menor.nombre} ${menor.apellidos}` : t('fi1.menor'); }).join(', ')}</p>}</li>)}</ul>}</dd></div>)}</dl>;
}
