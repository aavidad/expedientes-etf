import { useCallback, useEffect, useState } from 'react';
import { Boton } from './componentes';
import { claveDocumento, type Emision, type Registro } from './api';

type Ejecutar = (ruta: string, datos: unknown, resultado: (r: Response) => Promise<void>) => void;
export function Documentos({ id, tipos, emisiones, ejecutar, recargar, t, idioma, ocupado, alEditar }: { id: string; tipos: string[]; emisiones: Emision[]; ejecutar: Ejecutar; recargar: () => Promise<void>; t: (clave: string) => string; idioma: string; ocupado: boolean; alEditar: (sucio: boolean) => void }) {
  const [aviso, ponerAviso] = useState('');
  const disponibles = [...tipos, ...emisiones.map(e => e.tipo).filter(tipo => !tipos.includes(tipo))];
  const [cambios, ponerCambios] = useState<Record<string, boolean>>({});
  useEffect(() => { alEditar(Object.values(cambios).some(Boolean)); }, [cambios, alEditar]);
  const registrarCambio = useCallback((id: string, valor: boolean) => { ponerCambios(c => c[id] === valor ? c : { ...c, [id]: valor }); }, []);
  return <section className="decision-tarjeta" aria-labelledby="decision-documentos">
    <h2 id="decision-documentos">{t('decision.documentos')}</h2>
    <p>{t('decision.documentos_estado')}</p>
    {aviso && <p role="status">{t(aviso)}</p>}
    {!disponibles.length && <p>{t('decision.documentos_vacio')}</p>}
    {disponibles.map(tipo => {
      const emision = emisiones.find(e => e.tipo === tipo);
      return <section className="decision-documento" key={tipo} aria-label={t(claveDocumento[tipo])}>
        <h3>{t(claveDocumento[tipo])}</h3>
        {emision && <dl><dt>{t('decision.generado')}</dt><dd>{new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(new Date(emision.generada_en))}</dd><dt>{t('decision.plantilla')}</dt><dd>{emision.plantilla_version}</dd></dl>}
        {!tipos.includes(tipo) && <p>{t('decision.documento_anterior')}</p>}
        {tipos.includes(tipo) && <Boton secundario disabled={ocupado || !!(emision && cambios[emision.id])} accion={() => ejecutar('/api/documentos/emisiones', { tipo, expediente_id: id, idioma }, async r => {
          const contenido = await r.blob();
          const enlace = document.createElement('a');
          const url = URL.createObjectURL(contenido); enlace.href = url; enlace.download = `${t(claveDocumento[tipo])}.docx`; enlace.click();
          window.setTimeout(() => URL.revokeObjectURL(url), 1000);
          ponerAviso(r.headers.get('X-Aviso-Clave') ?? 'decision.descargado'); await recargar();
        })}>{t('decision.descargar')} {t(claveDocumento[tipo])}</Boton>}
        {emision && cambios[emision.id] && <p>{t('decision.guardar_antes_descargar')}</p>}
        {emision?.registro && <RegistroOficio key={`${emision.id}-${emision.registro.version}`} emision={emision} ejecutar={ejecutar} recargar={recargar} t={t} idioma={idioma} ocupado={ocupado} alEditar={registrarCambio} />}
      </section>;
    })}
  </section>;
}
function RegistroOficio({ emision, ejecutar, recargar, t, idioma, ocupado, alEditar }: { emision: Emision; ejecutar: Ejecutar; recargar: () => Promise<void>; t: (clave: string) => string; idioma: string; ocupado: boolean; alEditar: (id: string, sucio: boolean) => void }) {
  const registro = emision.registro!;
  const [numero, ponerNumero] = useState(registro.numero_registro ?? '');
  const [fecha, ponerFecha] = useState(registro.fecha_registro ?? '');
  const [error, ponerError] = useState(false);
  const sucio = numero !== (registro.numero_registro ?? '') || fecha !== (registro.fecha_registro ?? '');
  useEffect(() => { alEditar(emision.id, sucio); return () => alEditar(emision.id, false); }, [sucio, emision.id, alEditar]);
  return <form onSubmit={e => {
    e.preventDefault(); if (!numero.trim() || !fecha) { ponerError(true); return; }
    ejecutar(`/api/documentos/emisiones/${emision.id}/registro-salida`, { version: registro.version, numero_registro: numero.trim(), fecha_registro: fecha }, async r => {
      const guardado = await r.json() as Registro; ponerNumero(guardado.numero_registro ?? ''); ponerFecha(guardado.fecha_registro ?? ''); ponerError(false); await recargar();
    });
  }}>
    <fieldset><legend>{t('decision.registro')}</legend>
      {registro.estado === 'anotado' && <p role="status">{t('decision.registro_guardado')} {registro.numero_registro} {new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(new Date(`${registro.fecha_registro}T12:00:00`))}</p>}
      <p>{t('decision.registro_fuera')}</p>
      <label htmlFor={`registro-numero-${emision.id}`}>{t('decision.registro_numero')}</label><input id={`registro-numero-${emision.id}`} maxLength={128} disabled={ocupado} value={numero} onChange={e => ponerNumero(e.target.value)} aria-invalid={error && !numero.trim()} aria-describedby={error ? `registro-error-${emision.id}` : undefined} />
      <label htmlFor={`registro-fecha-${emision.id}`}>{t('decision.registro_fecha')}</label><input id={`registro-fecha-${emision.id}`} type="date" disabled={ocupado} value={fecha} onChange={e => ponerFecha(e.target.value)} aria-invalid={error && !fecha} aria-describedby={error ? `registro-error-${emision.id}` : undefined} />
      {error && <p id={`registro-error-${emision.id}`} role="alert">{t('decision.registro_falta')}</p>}
      <button type="submit" className="boton-secundario" disabled={ocupado}>{t('decision.anotar_registro')}</button>
    </fieldset>
  </form>;
}
