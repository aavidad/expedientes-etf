import type { PropsPantalla } from '../registro-pantallas';
import { useLectura } from './api';
import { campoID, listas, type Aviso, type FichaMenores } from '../fi1/modelo';
import type { DatosFamilia } from './familia-modelo';
import { enfocar } from '../fi1/comunes';

export function AvisosFichaEquipo({ id, t, abrir }: { id: string; t: PropsPantalla['t']; abrir: (pestana: string, foco: () => void) => void }) {
  const lectura = useLectura<{ avisos: Aviso[] }>(`/api/expedientes/${encodeURIComponent(id)}/avisos`);
  const menores = useLectura<FichaMenores>(`/api/expedientes/${encodeURIComponent(id)}/menores`);
  const familia = useLectura<DatosFamilia>(`/api/expedientes/${encodeURIComponent(id)}/familia`);
  const unicos = lectura.datos?.avisos.filter((a, i, todos) => todos.findIndex((b) => a.campo === b.campo && a.sujeto_id === b.sujeto_id) === i) ?? [];
  return <>
    {lectura.cargando && <p role="status">{t('fi1.avisos_cargando')}</p>}
    {lectura.error && <div role="alert"><p>{t('fi1.avisos_error')}</p><button type="button" onClick={lectura.recargar}>{t('fi1.reintentar')}</button></div>}
    {unicos.length > 0 && <aside className="fi1-resumen" aria-labelledby="fi1-ficha-avisos"><details><summary id="fi1-ficha-avisos">{t('fi1.ver_avisos')}<span className="fi1-separacion">{new Intl.NumberFormat(document.documentElement.lang).format(unicos.length)}</span></summary><ul>{unicos.map((a) => {
      const indice = menores.datos?.menores.findIndex((m) => m.id === a.sujeto_id) ?? -1;
      const pestana = a.campo === 'fase' || listas.some((l) => l === a.campo) ? 'intervencion' : indice >= 0 || (a.sujeto_id && ['25', '14.1'].includes(a.apartado ?? '')) ? 'menores' : ['adultos', 'composicion_familiar', 'familia_numerosa', 'situacion_economica', 'ingresos_anuales_centimos', 'sexo', 'fecha_nacimiento', 'nivel_educativo', 'situacion_laboral'].includes(a.campo) ? 'familia' : undefined;
      const destino = campoID(a.campo, a.sujeto_id);
      const adulto = familia.datos?.adultos.find(p => p.id === a.sujeto_id);
      const textoCampo = t((pestana === 'familia' ? 'gap1.campo.' : 'fi1.campo.') + a.campo) + (adulto ? t('gap1.aviso_adulto').replace('{orden}', new Intl.NumberFormat(document.documentElement.lang).format(adulto.orden)).replace('{nombre}', adulto.nombre) : '');
      return <li key={destino}>{pestana ? <a href={'#' + destino} onClick={(e) => { e.preventDefault(); abrir(pestana, () => { history.replaceState(null, '', location.pathname + location.search + '#' + destino); enfocar(destino); }); }}>{indice >= 0 ? t('fi1.aviso_menor').replace('{campo}', textoCampo).replace('{numero}', new Intl.NumberFormat(document.documentElement.lang).format(indice + 1)) : textoCampo}</a> : <span>{textoCampo}</span>}</li>;
    })}</ul></details></aside>}
  </>;
}
