import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useButton } from 'react-aria';
import { Armazon, type Acceso } from './armazon';
import { useCambios } from './cambios';
import { menuPermitido, pantallaPermitida, type Pantalla } from './registro-pantallas';

type Textos = (clave: string) => string;
class FalloSesion extends Error {}
async function consultar(signal: AbortSignal): Promise<Acceso> {
  const respuesta = await fetch('/api/sesion', { signal, credentials: 'same-origin' });
  const datos = await respuesta.json();
  if (!respuesta.ok) throw new FalloSesion(datos.error?.clave ?? 'identidad.persistencia_no_disponible');
  if (!['sin_sesion', 'pendiente_alta', 'activo'].includes(datos.estado) || typeof datos.csrf !== 'string') {
    throw new FalloSesion('identidad.persistencia_no_disponible');
  }
  return datos as Acceso;
}
function ajustarRuta(acceso: Acceso, pantallas: readonly Pantalla[]) {
  const destino = acceso.estado === 'sin_sesion' ? '/entrar' : acceso.estado === 'pendiente_alta' ? '/pendiente' : menuPermitido(pantallas, acceso)[0]?.ruta ?? '/';
  if (acceso.estado === 'activo' && pantallaPermitida(pantallas, acceso, location.pathname)) return;
  if (location.pathname !== destino) history.replaceState(history.state, '', destino);
}
export function PantallaSesion({ t, idioma, preferencias, avisos, alActivar, pantallas }: {
  t: Textos; idioma: string; preferencias: ReactNode; avisos: ReactNode; alActivar: () => void; pantallas: readonly Pantalla[];
}) {
  const { solicitarSalida, sucio } = useCambios();
  const [acceso, ponerAcceso] = useState<Acceso>();
  const [rutaActual, ponerRuta] = useState(location.pathname + location.search);
  const [cuenta, ponerCuenta] = useState('');
  const [contrasena, ponerContrasena] = useState('');
  const [ocupado, ponerOcupado] = useState(true);
  const [error, ponerError] = useState('');
  const [intento, reintentar] = useState(0);
  const controlador = useRef<AbortController | null>(null);
  const enVuelo = useRef(false);
  const entrada = useRef<HTMLInputElement>(null);
  const botonEntrar = useRef<HTMLButtonElement>(null);
  const { buttonProps: propsEntrar } = useButton({ type: 'submit', isDisabled: ocupado }, botonEntrar);
  const posicion = useRef<number>(typeof history.state?.etfPosicion === 'number' ? history.state.etfPosicion : 0);
  const restaurando = useRef<(() => void) | null>(null);
  const permitirMovimiento = useRef(false);

  useEffect(() => {
    const abort = new AbortController(); controlador.current = abort;
    ponerOcupado(true); ponerError('');
    void consultar(abort.signal).catch(async (fallo: unknown) => {
      if (fallo instanceof FalloSesion && ['identidad.sesion_no_valida', 'identidad.usuario_no_disponible'].includes(fallo.message)) {
        ponerError(fallo.message); return consultar(abort.signal);
      }
      throw fallo;
    }).then((datos) => {
      if (!abort.signal.aborted) { ponerAcceso(datos); ajustarRuta(datos, pantallas); ponerRuta(location.pathname + location.search); }
    }).catch((fallo: unknown) => {
      if (!abort.signal.aborted) ponerError(fallo instanceof FalloSesion ? fallo.message : 'identidad.persistencia_no_disponible');
    }).finally(() => { if (!abort.signal.aborted) ponerOcupado(false); });
    return () => abort.abort();
  }, [intento]);

  async function cambiar(metodo: 'POST' | 'DELETE') {
    if (!acceso || enVuelo.current || !controlador.current) return;
    if (metodo === 'POST' && (!cuenta.trim() || !contrasena)) {
      ponerError('identidad.campos_obligatorios'); entrada.current?.focus(); return;
    }
    enVuelo.current = true; ponerOcupado(true); ponerError('');
    const signal = controlador.current.signal;
    try {
      const respuesta = await fetch('/api/sesion', {
        method: metodo, credentials: 'same-origin', signal,
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': acceso.csrf },
        ...(metodo === 'POST' ? { body: JSON.stringify({ cuenta, contrasena }) } : {}),
      });
      ponerContrasena('');
      if (!respuesta.ok) {
        const datos = await respuesta.json(); throw new FalloSesion(datos.error?.clave ?? 'identidad.persistencia_no_disponible');
      }
      const datos: Acceso = metodo === 'DELETE' ? await consultar(signal) : await respuesta.json();
      if (!signal.aborted) { ponerAcceso(datos); ajustarRuta(datos, pantallas); ponerRuta(location.pathname + location.search); ponerCuenta(''); }
    } catch (fallo) {
      if (!signal.aborted) ponerError(fallo instanceof FalloSesion ? fallo.message : 'identidad.persistencia_no_disponible');
    } finally {
      enVuelo.current = false;
      if (!signal.aborted) { ponerContrasena(''); ponerOcupado(false); }
    }
  }
  async function actualizar() {
    if (!controlador.current) return;
    try { const a = await consultar(controlador.current.signal); if (!controlador.current.signal.aborted) { ponerAcceso(a); ajustarRuta(a, pantallas); ponerRuta(location.pathname + location.search); } }
    catch { if (!controlador.current.signal.aborted) { ponerAcceso(undefined); reintentar((v) => v + 1); } }
  }
  function navegar(destino: string) {
    if (destino === rutaActual) return;
    solicitarSalida(() => {
      posicion.current += 1;
      history.pushState({ etfPosicion: posicion.current }, '', destino); ponerRuta(destino);
      requestAnimationFrame(() => document.getElementById('contenido')?.focus());
    });
  }
  useEffect(() => {
    if (typeof history.state?.etfPosicion !== 'number') history.replaceState({ etfPosicion: posicion.current }, '', location.href);
    function volver(evento: PopStateEvent) {
      if (restaurando.current) { const confirmar = restaurando.current; restaurando.current = null; confirmar(); return; }
      const destino = typeof evento.state?.etfPosicion === 'number' ? evento.state.etfPosicion as number : 0;
      const salto = posicion.current - destino;
      if (permitirMovimiento.current) permitirMovimiento.current = false;
      else if (sucio && salto !== 0) {
        // Restaura la entrada antes de ofrecer descartar para no perder el historial al cancelar.
        restaurando.current = () => solicitarSalida(() => { permitirMovimiento.current = true; history.go(-salto); });
        history.go(salto); return;
      }
      posicion.current = destino; ponerRuta(location.pathname + location.search);
    }
    window.addEventListener('popstate', volver);
    return () => window.removeEventListener('popstate', volver);
  }, [solicitarSalida, sucio]);

  const pathname = rutaActual.split('?')[0];
  const pantalla = pantallaPermitida(pantallas, acceso, pathname);
  const claveTitulo = (pantalla?.menuDe ? pantallas.find((p) => p.ruta === pantalla.menuDe)?.clave : pantalla?.clave) ?? (acceso?.estado === 'pendiente_alta' ? 'identidad.pendiente_titulo' : acceso?.estado === 'activo' ? 'identidad.acceso_titulo' : 'identidad.entrar_titulo');
  useEffect(() => { document.title = t(claveTitulo); }, [t, claveTitulo]);
  const Componente = pantalla?.componente;
  return <Armazon t={t} acceso={acceso} preferencias={preferencias} salir={() => solicitarSalida(() => { void cambiar('DELETE'); })} ocupado={ocupado} ruta={pathname} navegar={navegar}>
    {avisos}
    {error && <p id="error-sesion" role="alert">{t(error) || t('identidad.persistencia_no_disponible')}</p>}
    {pantalla && acceso && Componente ? <Componente key={pantalla.conservarAlNavegar ? undefined : rutaActual} t={t} idioma={idioma} csrf={acceso.csrf} actualizar={() => { void actualizar(); }} alActivar={alActivar} navegarRuta={navegar} />
      : <section className="sesion" aria-labelledby="titulo-sesion" aria-busy={ocupado}>
        <h1 id="titulo-sesion">{t(claveTitulo)}</h1>
        {ocupado && <p role="status">{t('identidad.validando')}</p>}
        {!acceso && !ocupado && <button type="button" onClick={() => reintentar((valor) => valor + 1)}>{t('comun.reintentar')}</button>}
        {acceso?.estado === 'sin_sesion' && <form noValidate onSubmit={(evento) => { evento.preventDefault(); void cambiar('POST'); }}>
          <p>{t('identidad.entrar_descripcion')}</p>
          <label htmlFor="cuenta-ad">{t('identidad.cuenta')}</label>
          <input ref={entrada} id="cuenta-ad" name="username" autoComplete="username" autoCapitalize="none" spellCheck={false} required maxLength={128} value={cuenta} disabled={ocupado} aria-describedby={error ? 'error-sesion' : undefined} onChange={(evento) => ponerCuenta(evento.target.value)} />
          <label htmlFor="contrasena-ad">{t('identidad.contrasena')}</label>
          <input id="contrasena-ad" name="password" type="password" autoComplete="current-password" required maxLength={2048} value={contrasena} disabled={ocupado} aria-describedby={error ? 'error-sesion' : undefined} onChange={(evento) => ponerContrasena(evento.target.value)} />
          <button {...propsEntrar} ref={botonEntrar}>{t('identidad.entrar')}</button>
        </form>}
        {acceso && acceso.estado !== 'sin_sesion' && <p className={acceso.estado === 'pendiente_alta' ? 'pendiente' : undefined}>{t(acceso.estado === 'pendiente_alta' ? 'identidad.pendiente_descripcion' : 'identidad.acceso_descripcion')}</p>}
      </section>}
  </Armazon>;
}
