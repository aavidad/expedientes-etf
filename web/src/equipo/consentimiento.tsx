import { useState } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import { useLectura, fecha, type Ficha } from './api';
import { EstadoLectura } from './equipo';
import { hoy, useEdicion } from './edicion';

type Avisos = { avisos: { codigo: string; clave_texto: string; fecha_limite?: string }[] };

export function ConsentimientoEquipo({ id, refrescando, ficha, alCambiar, t, csrf, idioma }: PropsPantalla & { id: string; refrescando: boolean; ficha: Ficha; alCambiar: () => void }) {
  const base = `/api/expedientes/${encodeURIComponent(id)}`;
  const avisos = useLectura<Avisos>(`${base}/avisos`);
  const edicion = useEdicion(id, csrf);
  const [fechas, ponerFechas] = useState<Record<string, string>>({});
  const anexo = ficha.subprograma === 'REUNIFICACION' ? 'anexo2' : 'anexo1';
  const proyecto = ficha.subprograma === 'REUNIFICACION' ? 'anexo8' : 'anexo7';
  const puedeEditar = ['aceptada', 'en_intervencion'].includes(ficha.estado);
  const aviso = avisos.datos?.avisos.find((a) => a.codigo === 'consentimiento');
  const confirmar = async (tipo: string) => {
    const r = await edicion.enviar(`${base}/hitos`, { tipo, fecha: fechas[tipo], version: ficha.version });
    if (r) { ponerFechas({}); alCambiar(); avisos.recargar(); }
  };
  return <section aria-labelledby="fi2-consentimiento">
    <h2 id="fi2-consentimiento">{t('fi2.consentimiento.titulo')}</h2>
    <EstadoLectura {...avisos} t={t} />
    {aviso && <div className={['equipo-tarjeta', 'equipo-aviso'].join(' ')} role="status"><p>{t('fi2.consentimiento.plazo')}</p>{aviso.fecha_limite && <p>{t('fi2.consentimiento.limite')} {fecha(aviso.fecha_limite, idioma)}</p>}<a href="#fi2-fecha-consentimiento">{t('fi2.consentimiento.ir_firma')}</a></div>}
    <p>{t('fi2.consentimiento.separado')}</p>
    <section className="equipo-tarjeta"><h3>{t(`fi2.${anexo}`)}</h3>
      <p>{t('fi2.consentimiento.word')}</p>
      {ficha.emisiones.filter((e) => e.codigo === `consentimiento/${anexo}`).slice(0, 1).map((e) => <p key={e.id}>{t('fi2.generado')} {fecha(e.fecha, idioma, true)}</p>)}
      {puedeEditar && <button type="button" disabled={edicion.ocupado || refrescando || edicion.conflicto || Object.values(fechas).some(Boolean)} onClick={async () => { if (await edicion.enviar(`${base}/documentos/${anexo}`, { idioma }, true)) alCambiar(); }}>{t(edicion.ocupado ? 'fi2.espera' : 'fi2.consentimiento.descargar')}</button>}
    </section>
    {[{ tipo: 'consentimiento', titulo: 'fi2.consentimiento.firma' }, { tipo: proyecto, titulo: `fi2.${proyecto}` }].map(({ tipo, titulo }) => {
      const hito = ficha.hitos.find((h) => h.codigo === tipo);
      return <section className="equipo-tarjeta" key={tipo}><h3>{t(titulo)}</h3>
        {tipo === proyecto && <p>{t('fi2.consentimiento.proyecto_fuera')}</p>}
        {hito ? <p>{t('fi2.confirmado')} <time dateTime={hito.fecha}>{fecha(hito.fecha, idioma)}</time></p> : <p className="equipo-aviso">{t('fi2.consentimiento.pendiente')}</p>}
        {puedeEditar && (!hito || tipo !== 'consentimiento') && <form onSubmit={(e) => { e.preventDefault(); void confirmar(tipo); }}>
          <label className="equipo-campo" htmlFor={`fi2-fecha-${tipo}`}>{t(tipo === 'consentimiento' ? 'fi2.consentimiento.firmado_el' : 'fi2.consentimiento.fecha_proyecto')}
            <input id={`fi2-fecha-${tipo}`} type="date" required min={ficha.fecha_aceptacion.slice(0, 10)} max={hoy()} value={fechas[tipo] ?? ''} disabled={edicion.ocupado || refrescando || edicion.conflicto || Object.entries(fechas).some(([otra, valor]) => otra !== tipo && !!valor)} aria-describedby={edicion.error ? 'fi2-error' : undefined} onChange={(e) => { ponerFechas({ [tipo]: e.target.value }); edicion.editar(); }} />
          </label>
          <button type="submit" className="principal" disabled={edicion.ocupado || refrescando || edicion.conflicto || !fechas[tipo]}>{t(edicion.ocupado ? 'fi2.espera' : 'fi2.consentimiento.confirmar')}</button>
        </form>}
      </section>;
    })}
    {edicion.error && <p id="fi2-error" role="alert">{t(edicion.error) || t('fi2.error')}</p>}
    {edicion.conflicto && <button type="button" onClick={() => edicion.verCambios(() => { ponerFechas({}); alCambiar(); avisos.recargar(); })}>{t('fi2.ver_cambios')}</button>}
    <p role="status">{edicion.resultado && t(edicion.resultado)}</p>
  </section>;
}
