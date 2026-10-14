import { useEffect, useState } from 'react';
import { useCambios } from './cambios';

type Fila = { anio: number; importe_mensual_centimos: number | null; importe_anual_centimos: number | null; version: number };
type Props = { t: (clave: string) => string; idioma: string; csrf: string };

const euros = (centimos: number | null, idioma: string) => centimos === null ? '' : new Intl.NumberFormat(idioma, { style: 'currency', currency: 'EUR' }).format(centimos / 100);

export function AdministracionSMI({ t, idioma, csrf }: Props) {
  const { ponerSucio } = useCambios();
  const [filas, ponerFilas] = useState<Fila[]>([]);
  const [edicion, ponerEdicion] = useState<Record<number, string>>({});
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState('');
  const [conflicto, ponerConflicto] = useState(false);
  const [guardando, ponerGuardando] = useState<number | null>(null);

  const cargar = async (signal?: AbortSignal) => {
    ponerCargando(true); ponerError('');
    try {
      const respuesta = await fetch('/api/coordinacion/smi', { signal });
      const cuerpo = await respuesta.json() as Fila[];
      if (!respuesta.ok || !Array.isArray(cuerpo)) throw new Error('catalogo.no_disponible');
      ponerFilas(cuerpo);
      ponerEdicion(Object.fromEntries(cuerpo.map((fila) => [fila.anio, fila.importe_mensual_centimos === null ? '' : (fila.importe_mensual_centimos / 100).toFixed(2)])));
    } catch (e) { if (!signal?.aborted) ponerError(e instanceof Error ? e.message : 'catalogo.no_disponible'); }
    finally { if (!signal?.aborted) ponerCargando(false); }
  };

  useEffect(() => { const controlador = new AbortController(); void cargar(controlador.signal); return () => controlador.abort(); }, []);
  useEffect(() => { ponerSucio(filas.some((fila) => edicion[fila.anio] !== (fila.importe_mensual_centimos === null ? '' : (fila.importe_mensual_centimos / 100).toFixed(2)))); }, [filas, edicion, ponerSucio]);
  useEffect(() => () => ponerSucio(false), [ponerSucio]);
  useEffect(() => { document.title = t('smi.titulo') || t('comun.titulo'); }, [t]);

  const guardar = async (fila: Fila) => {
    ponerGuardando(fila.anio); ponerError(''); ponerConflicto(false);
    try {
      const respuesta = await fetch('/api/coordinacion/smi', { method: 'PUT', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ anio: fila.anio, importe_mensual: edicion[fila.anio], version_esperada: fila.version }) });
      if (respuesta.status === 409) { ponerConflicto(true); await cargar(); return; }
      const cuerpo = await respuesta.json() as Fila & { error?: { clave?: string } };
      if (!respuesta.ok) throw new Error(cuerpo.error?.clave ?? 'catalogo.no_disponible');
      ponerFilas((actuales) => actuales.map((actual) => actual.anio === fila.anio ? cuerpo : actual));
      ponerEdicion((actual) => ({ ...actual, [fila.anio]: (cuerpo.importe_mensual_centimos! / 100).toFixed(2) }));
    } catch (e) { ponerError(e instanceof Error ? e.message : 'catalogo.no_disponible'); }
    finally { ponerGuardando(null); }
  };

  return <section className="contenido-admin" aria-labelledby="smi-titulo">
    <h1 id="smi-titulo">{t('smi.titulo')}</h1>
    <p>{t('smi.presentacion')}</p>
    {cargando && <p role="status">{t('smi.cargando')}</p>}
    {error && <p role="alert">{t(error)}</p>}
    {conflicto && <p role="alert">{t('smi.conflicto')}</p>}
    {!cargando && <div className="smi-tabla-contenedor"><table className="smi-tabla">
      <caption>{t('smi.tabla')}</caption>
      <thead><tr><th scope="col">{t('smi.anio')}</th><th scope="col">{t('smi.mensual')}</th><th scope="col">{t('smi.anual')}</th><th scope="col">{t('smi.acciones')}</th></tr></thead>
      <tbody>{filas.map((fila) => <tr key={fila.anio}>
        <th scope="row">{fila.anio}</th>
        <td><label className="smi-etiqueta" htmlFor={`smi-${fila.anio}`}>{t('smi.mensual')} {fila.anio}</label><input id={`smi-${fila.anio}`} type="number" min="0.01" step="0.01" inputMode="decimal" required aria-describedby={`smi-ayuda-${fila.anio}`} value={edicion[fila.anio] ?? ''} onChange={(evento) => ponerEdicion((actual) => ({ ...actual, [fila.anio]: evento.target.value }))} /><span id={`smi-ayuda-${fila.anio}`} className="smi-ayuda">{t('smi.dos_decimales')}</span></td>
        <td>{fila.importe_anual_centimos === null ? t('smi.pendiente') : euros(fila.importe_anual_centimos, idioma)}</td>
        <td><button type="button" disabled={guardando !== null || (fila.importe_mensual_centimos !== null && edicion[fila.anio] === (fila.importe_mensual_centimos / 100).toFixed(2))} onClick={() => void guardar(fila)}>{guardando === fila.anio ? t('smi.guardando') : t('smi.guardar')}</button></td>
      </tr>)}</tbody>
    </table></div>}
  </section>;
}
