import { useEffect, useRef, useState } from 'react';
import { useButton } from 'react-aria';
import { activar, consultar, descartar, subir, type Estado, type Preparacion, type Referencia } from './paquetes-api';

type Props = { t: (clave: string) => string; csrf: string; alActivar: () => void };

function Boton({ children, onPress, disabled = false }: { children: React.ReactNode; onPress: () => void; disabled?: boolean }) {
  const ref = useRef<HTMLButtonElement>(null);
  const { buttonProps } = useButton({ onPress, isDisabled: disabled }, ref);
  return <button {...buttonProps} ref={ref}>{children}</button>;
}

export function AdministracionPaquetes({ t, alActivar, csrf }: Props) {
  const [tipo, setTipo] = useState(location.pathname === '/admin/temas' ? 'tema' : 'idioma');
  const [codigo, setCodigo] = useState('');
  const [archivo, setArchivo] = useState<File>();
  const [estado, setEstado] = useState<Estado>('inicio');
  const [actual, setActual] = useState<Referencia>();
  const [preparacion, setPreparacion] = useState<Preparacion>();
  const [error, setError] = useState('');
  const [ayuda, setAyuda] = useState(false);
  const controlador = useRef<AbortController | undefined>(undefined);
  const pendiente = useRef<string | undefined>(undefined);
  const dialogo = useRef<HTMLDialogElement>(null);
  const ocupado = ['subiendo', 'antivirus', 'activando', 'consultando'].includes(estado);

  useEffect(() => () => { controlador.current?.abort(); if (pendiente.current) descartar(pendiente.current, csrf); }, [csrf]);

  const reiniciar = () => {
    controlador.current?.abort();
    if (pendiente.current) descartar(pendiente.current, csrf);
    pendiente.current = undefined;
    setPreparacion(undefined); setActual(undefined); setError(''); setEstado('inicio');
  };

  const fallo = (err: unknown, signal: AbortSignal) => {
    if (signal.aborted) return;
    setError(err instanceof Error ? err.message : 'paquete.no_disponible'); setEstado('error');
  };

  const revisar = async () => {
    if (!archivo || !codigo.trim() || ocupado) return;
    reiniciar();
    if (archivo.size > 20 * 1024 * 1024) { setError('paquete.tamano_excedido'); setEstado('error'); return; }
    const c = new AbortController(); controlador.current = c; setEstado('consultando');
    try {
      const referencia = await consultar(tipo, codigo.trim(), c.signal);
      if (c.signal.aborted) return;
      setActual(referencia);
      const p = await subir(tipo, codigo.trim(), archivo, c.signal, setEstado, csrf);
      if (c.signal.aborted) { descartar(p.ticket, csrf); return; }
      pendiente.current = p.ticket; setPreparacion(p); setEstado('validado');
    } catch (err) { fallo(err, c.signal); }
  };

  const confirmar = async () => {
    if (!preparacion || !actual || ocupado) return;
    dialogo.current?.close();
    const c = new AbortController(); controlador.current = c; setEstado('activando'); setError('');
    try {
      const ref = await activar(preparacion, actual.version, c.signal, csrf);
      if (c.signal.aborted) return;
      pendiente.current = undefined; setPreparacion(undefined); setActual(ref); setEstado('activado'); alActivar();
    } catch (err) {
      pendiente.current = undefined; setPreparacion(undefined);
      fallo(err, c.signal);
    }
  };

  const refrescar = async () => {
    const c = new AbortController(); controlador.current = c; setEstado('consultando'); setError('');
    try {
      const ref = await consultar(tipo, codigo.trim(), c.signal);
      if (c.signal.aborted) return;
      setActual(ref); setEstado('inicio');
    } catch (err) { fallo(err, c.signal); }
  };

  const mensajes: Record<Estado, string> = {
    inicio: '', consultando: 'paquetes.consultando', subiendo: 'paquetes.subiendo', antivirus: 'paquetes.antivirus',
    validado: 'paquetes.validado', activando: 'paquetes.activando', activado: 'paquetes.activado', error: '',
  };

  return <section className="paquetes" aria-labelledby="paquetes-titulo">
    <h1 id="paquetes-titulo">{t('paquetes.titulo')}</h1>
    <Boton onPress={() => setAyuda(!ayuda)}>{t('paquetes.ayuda')}</Boton>
    {ayuda && <p>{t('paquetes.ayuda_contenido')}</p>}
    <form onSubmit={(e) => { e.preventDefault(); void revisar(); }} aria-busy={ocupado}>
      <fieldset disabled={ocupado}>
        <legend>{t('paquetes.preparar')}</legend>
        <label htmlFor="paquete-tipo">{t('paquetes.tipo')}</label>
        <select id="paquete-tipo" value={tipo} onChange={(e) => { reiniciar(); setTipo(e.target.value); }}>
          <option value="idioma">{t('paquetes.idioma')}</option>
          <option value="tema">{t('paquetes.tema')}</option>
        </select>
        <label htmlFor="paquete-codigo">{t('paquetes.codigo')}</label>
        <input id="paquete-codigo" required maxLength={64} value={codigo} onChange={(e) => { reiniciar(); setCodigo(e.target.value); }} aria-describedby="paquete-ayuda" />
        <p id="paquete-ayuda">{t('paquetes.codigo_ayuda')}</p>
        <label htmlFor="paquete-archivo">{t('paquetes.archivo')}</label>
        <input id="paquete-archivo" type="file" accept=".zip,application/zip" required onChange={(e) => { reiniciar(); setArchivo(e.target.files?.[0]); }} aria-describedby="paquete-limite paquete-error" />
        <p id="paquete-limite">{t('paquetes.limite')}</p>
        <Boton onPress={() => void revisar()} disabled={!archivo || !codigo.trim() || ocupado}>{t('paquetes.revisar')}</Boton>
      </fieldset>
    </form>
    <p role="status" aria-live="polite">{mensajes[estado] ? t(mensajes[estado]) : ''}</p>
    <p id="paquete-error" role="alert">{error ? (t(error) || t('paquete.no_disponible')) : ''}</p>
    {actual && <p>{t('paquetes.version_activa')} {actual.version_contenido || t('paquetes.version_inicial')}</p>}
    {preparacion && <>
      <p>{t('paquetes.version_revisada')} {preparacion.version_contenido}</p>
      <Boton onPress={() => dialogo.current?.showModal()} disabled={ocupado}>{t('paquetes.activar')}</Boton>
      <dialog ref={dialogo} aria-labelledby="confirmar-titulo" aria-describedby="confirmar-texto">
        <h2 id="confirmar-titulo">{t('paquetes.confirmar')}</h2>
        <p id="confirmar-texto">{t('paquetes.confirmar_contenido')} {preparacion.codigo} {preparacion.version_contenido}</p>
        <Boton onPress={() => dialogo.current?.close()}>{t('paquetes.cancelar')}</Boton>
        <Boton onPress={() => void confirmar()}>{t('paquetes.activar')}</Boton>
      </dialog>
    </>}
    {estado === 'error' && <Boton onPress={() => void refrescar()}>{t('paquetes.ver_actual')}</Boton>}
  </section>;
}
