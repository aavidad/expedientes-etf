import { useEffect, useRef, useState } from 'react';
import { useCambios } from '../cambios';
import type { FichaBase } from '../fi1/modelo';

// Conserva el borrador y la clave de un reintento hasta confirmar el resultado.
export function useFamiliaEdicion<T extends FichaBase>(ruta: string, csrf: string, alCambiar: () => void) {
  const [datos, ponerDatos] = useState<T>();
  const [actuales, ponerActuales] = useState<T>();
  const [cargando, ponerCargando] = useState(true);
  const [ocupado, ponerOcupado] = useState(false);
  const [error, ponerError] = useState('');
  const [conflicto, ponerConflicto] = useState(false);
  const [recuperar, ponerRecuperar] = useState(false);
  const [resultado, ponerResultado] = useState('');
  const [sucio, ponerEditado] = useState(false);
  const { ponerSucio } = useCambios();
  const abort = useRef<AbortController | null>(null);
  const bloqueo = useRef(false);
  const claves = useRef(new Map<string, string>());
  async function leer(signal: AbortSignal): Promise<T> {
    const r = await fetch(ruta, { signal, cache: 'no-store', credentials: 'same-origin' });
    const d = await r.json() as T & { error?: { clave?: string } };
    if (!r.ok) throw new Error(d.error?.clave ?? 'fi1.error');
    if (!Number.isInteger(d.version) || d.version < 1) throw new Error('fi1.error');
    return d;
  }
  useEffect(() => {
    const control = new AbortController(); abort.current = control;
    setInicial();
    void leer(control.signal).then((d) => { if (!control.signal.aborted) ponerDatos(d); })
      .catch((e: unknown) => { if (!control.signal.aborted) ponerError(e instanceof Error ? e.message : 'fi1.error'); })
      .finally(() => { if (!control.signal.aborted) ponerCargando(false); });
    return () => { control.abort(); ponerSucio(false); };
    // La lectura sólo se reinicia al cambiar de expediente o pestaña.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ruta]);
  function setInicial() { ponerDatos(undefined); ponerCargando(true); ponerError(''); ponerEditado(false); ponerSucio(false); claves.current.clear(); }
  function editar(d: T) { ponerDatos(d); ponerEditado(true); ponerSucio(true); ponerResultado(''); }
  async function reintentar() {
    if (!abort.current || ocupado) return;
    ponerOcupado(true); ponerError('');
    try {
      const d = await leer(abort.current.signal);
      if (abort.current.signal.aborted) return;
      if (conflicto) ponerActuales(d);
      else { ponerDatos(d); ponerRecuperar(false); ponerEditado(false); ponerSucio(false); if (recuperar) { ponerResultado('fi1.guardado'); alCambiar(); } }
    } catch (e) { if (!abort.current.signal.aborted) ponerError(e instanceof Error ? e.message : 'fi1.error'); }
    finally { if (!abort.current.signal.aborted) ponerOcupado(false); }
  }
  function mantener() {
    if (!actuales || !datos) return;
    editar({ ...datos, version: actuales.version }); ponerActuales(undefined); ponerConflicto(false); ponerError('');
  }
  function usarActuales() {
    if (!actuales) return;
    ponerDatos(actuales); ponerActuales(undefined); ponerConflicto(false); ponerError(''); ponerEditado(false); ponerSucio(false);
  }
  async function guardar(cuerpo: object, destino = ruta, metodo: 'PATCH' | 'PUT' = 'PUT') {
    if (bloqueo.current || ocupado || conflicto || recuperar || !abort.current) return;
    bloqueo.current = true;
    ponerOcupado(true); ponerError(''); ponerResultado('');
    const body = JSON.stringify(cuerpo);
    const identidad = destino + body;
    const clave = claves.current.get(identidad) ?? crypto.randomUUID(); claves.current.set(identidad, clave);
    try {
      const r = await fetch(destino, { method: metodo, signal: abort.current.signal, credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': clave }, body });
      const d = await r.json() as { version?: number; error?: { clave?: string } };
      if (abort.current.signal.aborted) return;
      if (r.status === 409) { ponerConflicto(true); return; }
      if (!r.ok) throw new Error(d.error?.clave ?? 'fi1.error');
      if (!Number.isInteger(d.version) || (d.version ?? 0) < 1) throw new Error('fi1.error');
      ponerRecuperar(true);
      // Recupera identificadores asignados por el servidor antes de otra edición.
      const canonicos = await leer(abort.current.signal);
      if (abort.current.signal.aborted) return;
      ponerDatos(canonicos); ponerRecuperar(false); claves.current.delete(identidad); ponerEditado(false); ponerSucio(false);
      ponerResultado('fi1.guardado'); alCambiar();
    } catch (e) { if (!abort.current.signal.aborted) ponerError(e instanceof Error ? e.message : 'fi1.error'); }
    finally { bloqueo.current = false; if (!abort.current.signal.aborted) ponerOcupado(false); }
  }
  return { datos, actuales, cargando, ocupado, error, conflicto, recuperar, resultado, sucio, editar, guardar, reintentar, mantener, usarActuales };
}
