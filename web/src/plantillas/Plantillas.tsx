import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import type { PropsPantalla } from '../registro-pantallas';
import { useCambios } from '../cambios';
import { activar, cargar, FalloPlantilla, subir, type Version } from './api';
import './plantillas.css';

function Boton({ children, onPress, disabled = false, secundario = false, id, etiqueta, expandido }: { id?: string; etiqueta?: string; expandido?: boolean; children: React.ReactNode; onPress: () => void; disabled?: boolean; secundario?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress, isDisabled: disabled, 'aria-label': etiqueta, 'aria-expanded': expandido }, ref);
  return <button {...buttonProps} id={id} ref={ref} className={secundario ? 'boton-secundario' : undefined}>{children}</button>;
}
export function Plantillas({ t, idioma, csrf }: PropsPantalla) {
  const [versiones, ponerVersiones] = useState<Version[]>([]);
  const [nombres, ponerNombres] = useState<Record<string, string>>({});
  const [estado, ponerEstado] = useState('cargando');
  const [error, ponerError] = useState<FalloPlantilla>();
  const [aviso, ponerAviso] = useState('');
  const [ayuda, ponerAyuda] = useState(false);
  const [formulario, ponerFormulario] = useState(false);
  const [tipo, ponerTipo] = useState('');
  const [lengua, ponerLengua] = useState('');
  const [numero, ponerNumero] = useState('');
  const [archivo, ponerArchivo] = useState<File>();
  const [elegida, ponerElegida] = useState<Version>();
  const peticion = useRef<AbortController | undefined>(undefined);
  const dialogo = useRef<HTMLDialogElement>(null);
  const resumen = useRef<HTMLDivElement>(null);
  const entradaArchivo = useRef<HTMLInputElement>(null);
  const abrir = useRef<HTMLButtonElement>(null);
  const { ponerSucio } = useCambios();
  const ocupado = ['cargando', 'revisando', 'activando'].includes(estado);
  const tipos = [...new Set(versiones.map((v) => v.plantilla.tipo))];
  const lenguas = [...new Set(versiones.filter((v) => v.plantilla.tipo === tipo).map((v) => v.plantilla.idioma))];
    const etiqueta = (valor: string) => t(`plantillas.tipo.${valor.replaceAll('/', '.')}`) || t('plantillas.tipo_desconocido');
  const grupos = [...new Set(versiones.map((v) => `${v.plantilla.tipo}|${v.plantilla.idioma}`))].sort((a,b) => {
    const [tipoA,idiomaA] = a.split('|'); const [tipoB,idiomaB] = b.split('|');
    return etiqueta(tipoA).localeCompare(etiqueta(tipoB),idioma) || (idiomaA === idioma ? -1 : idiomaB === idioma ? 1 : idiomaA.localeCompare(idiomaB));
  });
  const idiomaDocumento = (valor: string) => new Intl.DisplayNames([idioma], { type: 'language' }).of(valor) ?? t('plantillas.idioma_desconocido');
  const fecha = (valor: string) => new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(new Date(valor));
  const persona = (id: string) => nombres[id] || t('plantillas.autor_no_consta');
  const texto = (clave: string, valor: string) => t(clave).replace('{valor}', valor);

  const actualizarLista = async (signal: AbortSignal) => {
    const datos = await cargar(signal);
    if (signal.aborted) return;
    ponerVersiones(datos.versiones); ponerNombres(datos.nombres);
    if (!tipo && datos.versiones.length) {
      ponerTipo(datos.versiones[0].plantilla.tipo); ponerLengua(datos.versiones[0].plantilla.idioma);
    }
  };
  const fallo = (e: unknown, signal: AbortSignal) => {
    if (signal.aborted) return;
    ponerError(e instanceof FalloPlantilla ? e : new FalloPlantilla('plantillas.no_disponible'));
    ponerEstado('error');
  };
  const consultar = async () => {
    peticion.current?.abort(); const c = new AbortController(); peticion.current = c;
    ponerEstado('cargando'); ponerError(undefined);
    try { await actualizarLista(c.signal); if (!c.signal.aborted) ponerEstado('lista'); } catch (e) { fallo(e, c.signal); }
  };
  useEffect(() => { void consultar(); return () => peticion.current?.abort(); }, []);
  useEffect(() => { document.title = t('plantillas.titulo'); }, [t]);
  useEffect(() => { ponerSucio(formulario && (!!archivo || !!numero)); }, [formulario, archivo, numero, ponerSucio]);
  useEffect(() => () => ponerSucio(false), [ponerSucio]);
  useEffect(() => { if (error) resumen.current?.focus(); }, [error]);
  useEffect(() => { if (formulario) document.getElementById('plantillas-tipo')?.focus(); }, [formulario]);
  useEffect(() => {
    if (!ayuda) return;
    const cerrar = (evento: KeyboardEvent) => { if (evento.key === 'Escape') { ponerAyuda(false); document.getElementById('plantillas-boton-ayuda')?.focus(); } };
    window.addEventListener('keydown', cerrar); return () => window.removeEventListener('keydown', cerrar);
  }, [ayuda]);
  const cerrarFormulario = () => {
    ponerFormulario(false); ponerArchivo(undefined); ponerNumero(''); ponerError(undefined); abrir.current?.focus();
  };
  const subirVersion = async (evento: React.FormEvent) => {
    evento.preventDefault(); if (ocupado) return;
    ponerError(undefined); ponerAviso('');
    const actual = versiones.find((v) => v.activa && v.plantilla.tipo === tipo && v.plantilla.idioma === lengua);
    if (!archivo || !actual) { ponerError(new FalloPlantilla('plantillas.falta_archivo')); return; }
    if (archivo.size > 10 * 1024 * 1024 || archivo.size === 0 || !archivo.name.toLowerCase().endsWith('.docx')) { ponerError(new FalloPlantilla('plantillas.archivo_invalido')); return; }
    if (!/^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/.test(numero) || numero.length > 32) { ponerError(new FalloPlantilla('plantillas.version_invalida')); return; }
    const c = new AbortController(); peticion.current = c; ponerEstado('revisando');
    try {
      await subir(archivo, tipo, lengua, numero, actual.version_actual, csrf, c.signal);
      if (c.signal.aborted) return;
      ponerArchivo(undefined); ponerNumero(''); if (entradaArchivo.current) entradaArchivo.current.value = '';
      ponerAviso('plantillas.subida'); ponerSucio(false);
      await actualizarLista(c.signal); if (!c.signal.aborted) ponerEstado('lista');
    } catch (e) { fallo(e, c.signal); }
  };
  const confirmar = async () => {
    if (!elegida || ocupado) return;
    dialogo.current?.close(); const c = new AbortController(); peticion.current = c;
    ponerEstado('activando'); ponerError(undefined); ponerAviso('');
    try {
      await activar(elegida, csrf, c.signal);
      if (c.signal.aborted) return;
      ponerAviso('plantillas.activada'); ponerElegida(undefined);
      await actualizarLista(c.signal); if (!c.signal.aborted) ponerEstado('lista');
    } catch (e) { fallo(e, c.signal); }
  };
  const mostrarDialogo = (v: Version) => { ponerElegida(v); dialogo.current?.showModal(); };
  const autoria = (por: string, en: string) => t('plantillas.autoria_fecha').replace('{persona}', persona(por)).replace('{fecha}', fecha(en));
  const marcador = (valor: string) => valor.startsWith(':') ? valor.slice(1) : valor.replace(':', ' / ');

  return <section className="plantillas" aria-labelledby="plantillas-titulo">
    <div className="plantillas-cabecera"><h1 id="plantillas-titulo">{t('plantillas.titulo')}</h1>
      <Boton id="plantillas-boton-ayuda" etiqueta={t('plantillas.ayuda')} expandido={ayuda} secundario onPress={() => ponerAyuda(!ayuda)}>{t('plantillas.ayuda_simbolo')}</Boton>
      {!formulario && <button ref={abrir} disabled={ocupado || !grupos.length} onClick={() => { ponerFormulario(true); ponerAviso(''); }}>{t('plantillas.subir')}</button>}
    </div>
    {ayuda && <div className="plantillas-ayuda"><p>{t('plantillas.ayuda_texto')}</p></div>}
    <p role="status">{ocupado ? t(`plantillas.${estado}`) : aviso ? t(aviso) : ''}</p>
    {error && <div role="alert" ref={resumen} tabIndex={-1} className="plantillas-error" id="plantillas-error">
      <p>{t(error.clave) || t('plantillas.no_disponible')}</p>
      {error.antivirus === 'limpio' && <p>{t('plantillas.antivirus_limpio')}</p>}
      {error.faltan.map((m) => <p key={`f-${m}`}>{texto('plantillas.marcador_falta', marcador(m))}</p>)}
      {error.sobran.map((m) => <p key={`s-${m}`}>{texto('plantillas.marcador_sobra', marcador(m))}</p>)}
      <Boton secundario onPress={() => void consultar()} disabled={ocupado}>{t('plantillas.actualizar')}</Boton>
    </div>}
    {formulario && <form onSubmit={(e) => void subirVersion(e)} aria-busy={estado === 'revisando'} className="plantillas-formulario">
      <h2>{t('plantillas.nueva_version')}</h2>
      <fieldset disabled={ocupado}>
        <legend>{t('plantillas.documento_destino')}</legend>
        <label htmlFor="plantillas-tipo">{t('plantillas.tipo')}</label>
        <select id="plantillas-tipo" value={tipo} onChange={(e) => { ponerTipo(e.target.value); ponerLengua(versiones.find((v) => v.plantilla.tipo === e.target.value)?.plantilla.idioma ?? ''); }}>
          {tipos.map((v) => <option key={v} value={v}>{etiqueta(v)}</option>)}
        </select>
        <label htmlFor="plantillas-idioma">{t('plantillas.idioma')}</label>
        <select id="plantillas-idioma" value={lengua} onChange={(e) => ponerLengua(e.target.value)}>{lenguas.map((v) => <option key={v} value={v}>{idiomaDocumento(v)}</option>)}</select>
        <label htmlFor="plantillas-version">{t('plantillas.version')}</label>
        <input id="plantillas-version" value={numero} required maxLength={32} aria-describedby={error ? 'plantillas-formato plantillas-error' : 'plantillas-formato'} onChange={(e) => ponerNumero(e.target.value)} />
        <span id="plantillas-formato">{t('plantillas.formato_version')}</span>
        <label htmlFor="plantillas-archivo">{t('plantillas.archivo')}</label>
        <p id="plantillas-politica">{t('plantillas.politica')}</p>
        <input id="plantillas-archivo" ref={entradaArchivo} type="file" accept=".docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document" required aria-describedby={error ? 'plantillas-politica plantillas-error' : 'plantillas-politica'} onChange={(e) => ponerArchivo(e.target.files?.[0])} />
      </fieldset>
      <div className="plantillas-acciones"><button type="submit" disabled={ocupado}>{t('plantillas.subir_revisar')}</button><Boton secundario disabled={ocupado} onPress={cerrarFormulario}>{t('plantillas.cerrar')}</Boton></div>
    </form>}
    {!ocupado && !error && !versiones.length && <p>{t('plantillas.vacio')}</p>}
    <div className="plantillas-lista">{grupos.map((grupo) => {
      const historia = versiones.filter((v) => `${v.plantilla.tipo}|${v.plantilla.idioma}` === grupo);
      const actual = historia.find((v) => v.activa);
      return <article key={grupo} className="plantillas-tarjeta">
        <h2>{etiqueta(historia[0].plantilla.tipo)}</h2><p>{idiomaDocumento(historia[0].plantilla.idioma)}</p>
        <p className="plantillas-activa">{t('plantillas.version_activa')} <strong>{actual?.plantilla.version ?? t('plantillas.sin_activa')}</strong></p>
        {actual && <p>{actual.activaciones.length ? autoria(actual.activaciones[0].por, actual.activaciones[0].en) : t('plantillas.activacion_inicial')}</p>}
        <details><summary>{texto('plantillas.historial', String(historia.length))}</summary>
          <ol className="plantillas-historial">{historia.map((v) => <li key={v.plantilla.id}>
            <h3>{t('plantillas.version')} {v.plantilla.version} <span className="pildora">{t(v.activa ? 'plantillas.activa' : v.activaciones.length ? 'plantillas.anterior' : !v.subida_por ? 'plantillas.inicial' : 'plantillas.pendiente')}</span></h3>
            <dl><dt>{t('plantillas.antivirus')}</dt><dd>{t(v.antivirus === 'limpio' ? 'plantillas.antivirus_limpio' : 'plantillas.antivirus_no_consta')}</dd>
              <dt>{t('plantillas.marcadores')}</dt><dd>{t('plantillas.marcadores_validos')}</dd>
              <dt>{t('plantillas.subida_por')}</dt><dd>{autoria(v.subida_por, v.subida_en)}</dd></dl>
            {v.activaciones.map((a, i) => <p key={i}>{t('plantillas.activada_por')} {autoria(a.por, a.en)}</p>)}
            {!v.activa && <Boton secundario disabled={ocupado || (!!archivo || !!numero)} onPress={() => mostrarDialogo(v)}>{texto('plantillas.activar_version', v.plantilla.version)}</Boton>}
          </li>)}</ol>
        </details>
      </article>;
    })}</div>
    <dialog ref={dialogo} aria-labelledby="plantillas-confirmar" onCancel={() => ponerElegida(undefined)}>
      <h2 id="plantillas-confirmar">{t('plantillas.confirmar_titulo')}</h2>
      {elegida && <p>{t('plantillas.resumen_version').replace('{documento}', etiqueta(elegida.plantilla.tipo)).replace('{idioma}', idiomaDocumento(elegida.plantilla.idioma)).replace('{version}', elegida.plantilla.version)}</p>}
      <p>{t('plantillas.confirmar_consecuencia')}</p>
      <div className="plantillas-acciones"><Boton secundario onPress={() => { dialogo.current?.close(); ponerElegida(undefined); }}>{t('plantillas.cancelar')}</Boton><Boton onPress={() => void confirmar()}>{t('plantillas.confirmar')}</Boton></div>
    </dialog>
  </section>;
}
