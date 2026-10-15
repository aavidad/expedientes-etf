import './sesion.css';
import { useEffect, useRef, useState } from 'react';

type Plazos = { caduca: number; maxima: number };
const eventoPlazos = 'etf-sesion-plazos';
const eventoCaducada = 'etf-sesion-caducada';

// Las cabeceras no contienen identidad. El reloj se corrige con la hora del
// servidor; ninguna comprobación periódica prolonga la sesión.
export function observarSesion() {
  const original = window.fetch;
  window.fetch = async (...args: Parameters<typeof fetch>) => {
    const respuesta = await original(...args);
    const url = new URL(args[0] instanceof Request ? args[0].url : String(args[0]), location.href);
    if (url.origin === location.origin && url.pathname.startsWith('/api/')) {
      const ahora = Date.parse(respuesta.headers.get('X-ETF-Sesion-Ahora') ?? '');
      const caduca = Date.parse(respuesta.headers.get('X-ETF-Sesion-Caduca') ?? '');
      const maxima = Date.parse(respuesta.headers.get('X-ETF-Sesion-Maxima') ?? '');
      if ([ahora, caduca, maxima].every(Number.isFinite)) {
        window.dispatchEvent(new CustomEvent(eventoPlazos, { detail: { caduca: Date.now() + caduca - ahora, maxima: Date.now() + maxima - ahora } }));
      }
      if (respuesta.status === 401 && url.pathname !== '/api/sesion') window.dispatchEvent(new Event(eventoCaducada));
    }
    return respuesta;
  };
  return () => { window.fetch = original; };
}

export function AvisoSesion({ t, csrf, activa, alCaducar }: { t: (clave: string) => string; csrf: string; activa: boolean; alCaducar: () => void }) {
  const [plazos, ponerPlazos] = useState<Plazos>();
  const [ahora, ponerAhora] = useState(Date.now());
  const [error, ponerError] = useState(false);
  const [ocupada, ponerOcupada] = useState(false);
  const ultimaRenovacion = useRef(0);
  const comprobando = useRef(false);
  const callback = useRef(alCaducar); callback.current = alCaducar;
  const csrfActual = useRef(csrf); csrfActual.current = csrf;
  const activaActual = useRef(activa); activaActual.current = activa;
  const plazosActuales = useRef(plazos); plazosActuales.current = plazos;
  const aviso = !!plazos && ahora >= plazos.caduca - 120000;
  const absoluta = !!plazos && plazos.maxima <= plazos.caduca;

  async function volverEntrar() {
    if (comprobando.current) return;
    comprobando.current = true; ponerOcupada(true); ponerError(false);
    try {
      const r = await fetch('/api/sesion', { method: 'DELETE', credentials: 'same-origin', headers: { 'X-CSRF-Token': csrfActual.current } });
      if (r.ok || r.status === 401) callback.current(); else ponerError(true);
    } catch { ponerError(true); }
    finally { comprobando.current = false; ponerOcupada(false); }
  }
  async function renovar() {
    if (comprobando.current || !activaActual.current) return;
    comprobando.current = true; ponerOcupada(true); ponerError(false);
    try {
      const r = await fetch('/api/sesion', { method: 'PUT', credentials: 'same-origin', headers: { 'X-CSRF-Token': csrfActual.current } });
      if (r.status === 401 || r.status === 403) callback.current();
      else if (!r.ok) ponerError(true);
      else ultimaRenovacion.current = Date.now();
    } catch { ponerError(true); }
    finally { comprobando.current = false; ponerOcupada(false); }
  }
  useEffect(() => {
    const tiempos = (e: Event) => { ponerPlazos((e as CustomEvent<Plazos>).detail); ponerAhora(Date.now()); };
    const caducada = () => { if (activaActual.current) callback.current(); };
    const actividad = (e: Event) => {
      const p = plazosActuales.current;
      if (!e.isTrusted || !activaActual.current || !p || Date.now() >= p.caduca - 120000) return;
      if (Date.now() - ultimaRenovacion.current > 30000) { ultimaRenovacion.current = Date.now(); void renovar(); }
    };
    window.addEventListener(eventoPlazos, tiempos);
    window.addEventListener(eventoCaducada, caducada);
    window.addEventListener('keydown', actividad);
    window.addEventListener('pointerdown', actividad);
    window.addEventListener('pointermove', actividad, { passive: true });
    window.addEventListener('wheel', actividad, { passive: true });
    const dejarObservar = observarSesion();
    const temporizador = window.setInterval(() => ponerAhora(Date.now()), 1000);
    return () => {
      window.clearInterval(temporizador); dejarObservar();
      window.removeEventListener(eventoPlazos, tiempos); window.removeEventListener(eventoCaducada, caducada);
      window.removeEventListener('keydown', actividad); window.removeEventListener('pointerdown', actividad);
      window.removeEventListener('pointermove', actividad); window.removeEventListener('wheel', actividad);
    };
  }, []);
  useEffect(() => {
    if (!activa || !plazos || ahora < plazos.caduca || comprobando.current) return;
    // La lectura pasiva detecta actividad legítima de otra pestaña sin renovar.
    comprobando.current = true;
    void fetch('/api/sesion', { credentials: 'same-origin' }).then(async r => {
      if (r.status === 401 || r.status === 403 || (r.ok && (await r.json()).estado === 'sin_sesion')) callback.current();
      else if (!r.ok) callback.current();
    }).catch(() => callback.current()).finally(() => { comprobando.current = false; });
  }, [activa, plazos, ahora]);
  if (!activa || !aviso) return null;
  return <section className={['sesion', 'seg1-aviso'].join(' ')} aria-labelledby="seg1-aviso-titulo">
    <div role="alert"><h2 id="seg1-aviso-titulo">{t('seg1.sesion_aviso')}</h2><p>{t(absoluta ? 'seg1.sesion_maxima' : 'seg1.sesion_inactividad')}</p><p>{t('seg1.datos_guardados')}</p></div>
    <button type="button" disabled={ocupada} onClick={() => { if (absoluta) void volverEntrar(); else void renovar(); }}>{t(absoluta ? 'seg1.volver_entrar' : 'seg1.continuar')}</button>
    {ocupada && <p role="status">{t('seg1.continuando')}</p>}
    {error && <p role="alert">{t('seg1.continuar_error')}</p>}
  </section>;
}
