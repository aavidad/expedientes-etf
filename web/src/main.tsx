import { useEffect, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { I18nProvider, useButton } from 'react-aria';
import './estructura.css';
import { actualizarInstalacion } from './pwa1/instalacion';
import { pantallas } from './pantallas';
import { PantallaSesion } from './sesion';
import { Armazon, Preferencias } from './armazon';
import { GuardiaCambios, useCambios } from './cambios';

type Idioma = { codigo: string; nombre: string; version: string; predeterminado: boolean };
type Tema = { identificador: string; nombre: Record<string, string> };
type Textos = Record<string, string>;
const textoInicial = (nombre: string) =>
  document.querySelector<HTMLMetaElement>(`meta[name="${nombre}"]`)?.content ?? '';

async function obtener<T>(ruta: string, signal: AbortSignal): Promise<T> {
  const respuesta = await fetch(ruta, { signal });
  if (!respuesta.ok) throw new Error('paquete.no_disponible');
  return respuesta.json() as Promise<T>;
}

function PantallaAplicacion({ actualizarTextosGuardia }: { actualizarTextosGuardia: (textos: Textos) => void }) {
  const { solicitarSalida } = useCambios();
  const [idioma, cambiarIdioma] = useState(document.documentElement.lang);
  const [idiomas, ponerIdiomas] = useState<Idioma[]>([]);
  const [temas, ponerTemas] = useState<Tema[]>([]);
  const [tema, cambiarTema] = useState(() => document.querySelector<HTMLLinkElement>('link[href^="/api/temas/"]')?.getAttribute('href')?.split('/')[3] ?? '');
  const [errorTema, ponerErrorTema] = useState(false);
  const [textos, ponerTextos] = useState<Textos>({});
  const [cargando, ponerCargando] = useState(true);
  const [error, ponerError] = useState(false);
  const [intento, reintentar] = useState(0);
  const referencia = useRef<HTMLButtonElement>(null);
  const focoTrasIdioma = useRef<'persona' | 'preferencias' | null>(null);
  const { buttonProps } = useButton({ onPress: () => reintentar((valor) => valor + 1) }, referencia);
  const t = (clave: string) => (textos[clave] ?? '').replaceAll('{entidad.nombre}', textos['entidad.nombre'] ?? '');

  useEffect(() => actualizarInstalacion(idioma, tema), [idioma, tema]);

  useEffect(() => {
    const controlador = new AbortController();
    ponerCargando(true);
    ponerError(false);
    void Promise.all([
      obtener<Idioma[]>('/api/idiomas', controlador.signal),
      obtener<Tema[]>('/api/temas', controlador.signal),
      obtener<Textos>(`/api/idiomas/${encodeURIComponent(idioma)}/textos`, controlador.signal),
    ]).then(([lista, listaTemas, traduccion]) => {
      if (controlador.signal.aborted) return;
      if (!traduccion['comun.titulo'] || !lista.length) throw new Error('paquete.no_disponible');
      const seleccionado = lista.find((opcion) => opcion.codigo === idioma) ?? lista.find((opcion) => opcion.predeterminado);
      if (!seleccionado) throw new Error('paquete.no_disponible');
      if (seleccionado.codigo !== idioma) { cambiarIdioma(seleccionado.codigo); return; }
      ponerIdiomas(lista);
      ponerTemas(listaTemas);
      ponerTextos(traduccion); actualizarTextosGuardia(traduccion);
      document.documentElement.lang = idioma;
      document.title = traduccion['comun.titulo'];
    }).catch(() => { if (!controlador.signal.aborted) ponerError(true); })
      .finally(() => { if (!controlador.signal.aborted) ponerCargando(false); });
    return () => controlador.abort();
  }, [idioma, intento]);

  useEffect(() => {
    if (!tema) return;
    const anterior = document.querySelector<HTMLLinkElement>('link[href^="/api/temas/"]');
    if (intento === 0 && anterior?.getAttribute('href') === `/api/temas/${encodeURIComponent(tema)}/tokens`) return;
    const enlace = document.createElement('link');
    enlace.rel = 'stylesheet';
    enlace.href = `/api/temas/${encodeURIComponent(tema)}/tokens`;
    enlace.onload = () => { anterior?.remove(); ponerErrorTema(false); };
    enlace.onerror = () => { enlace.remove(); ponerErrorTema(true); };
    document.head.append(enlace);
    return () => { enlace.onload = null; enlace.onerror = null; if (anterior?.isConnected) enlace.remove(); };
  }, [tema, intento]);

  useEffect(() => {
    if (cargando || document.documentElement.lang !== idioma || !focoTrasIdioma.current) return;
    const restaurar = () => {
      const selector = focoTrasIdioma.current === 'persona' ? '.persona-conectada > summary' : '.preferencias > summary';
      const destino = document.querySelector<HTMLElement>(selector);
      if (!destino || !destino.getClientRects().length) return;
      destino.focus();
      focoTrasIdioma.current = null;
      observar.disconnect();
    };
    const observar = new MutationObserver(restaurar);
    observar.observe(document.body, { childList: true, subtree: true });
    restaurar();
    return () => observar.disconnect();
  }, [cargando, idioma]);

  function elegirIdioma(valor: string) {
    if (valor === idioma) return;
    solicitarSalida(() => {
      focoTrasIdioma.current = document.querySelector('.persona-conectada') ? 'persona' : 'preferencias';
      cambiarIdioma(valor);
    });
  }

  const preferencias = idiomas.length > 0 && <Preferencias t={t} idioma={idioma} idiomas={idiomas} tema={tema} temas={temas} cambiarIdioma={elegirIdioma} cambiarTema={cambiarTema} />;
  const avisos = <>
    {cargando && <p role="status">{t('comun.cargando') || textoInicial('cargando')}</p>}
    {(error || errorTema) && <>
      <p role="alert">{t('comun.error_carga') || textoInicial('error-carga')}</p>
      <button {...buttonProps} ref={referencia}>{t('comun.reintentar') || textoInicial('reintentar')}</button>
    </>}
  </>;
  return <I18nProvider locale={idioma}>
    {!cargando && !error ? <PantallaSesion pantallas={pantallas} t={t} idioma={idioma} preferencias={preferencias} avisos={avisos} alActivar={() => reintentar((valor) => valor + 1)} />
      : <Armazon t={t} preferencias={preferencias}>{avisos}</Armazon>}
  </I18nProvider>;
}

function Aplicacion() {
  const [textos, ponerTextos] = useState<Textos>({});
  return <GuardiaCambios t={(clave) => textos[clave] ?? ''}><PantallaAplicacion actualizarTextosGuardia={ponerTextos} /></GuardiaCambios>;
}

createRoot(document.getElementById('root')!).render(<Aplicacion />);
