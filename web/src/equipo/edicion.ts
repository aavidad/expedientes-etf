import { useEffect, useRef, useState } from 'react';
import { useCambios } from '../cambios';

export function hoy() {
  const partes = new Intl.DateTimeFormat(undefined, { timeZone: 'Europe/Madrid', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date());
  return ['year', 'month', 'day'].map((tipo) => partes.find((p) => p.type === tipo)?.value).join('-');
}

// Una petición conserva su identidad cuando se reintenta tras un fallo de red.
export function useEdicion(id: string, csrf: string) {
  const { ponerSucio, solicitarSalida } = useCambios();
  const [ocupado, ponerOcupado] = useState(false);
  const [error, ponerError] = useState('');
  const [resultado, ponerResultado] = useState('');
  const [conflicto, ponerConflicto] = useState(false);
  const abort = useRef<AbortController | null>(null);
  const peticiones = useRef(new Map<string, string>());
  const bloqueo = useRef(false);
  useEffect(() => { abort.current = new AbortController(); return () => { abort.current?.abort(); ponerSucio(false); }; }, [id, ponerSucio]);
  const editar = () => { ponerSucio(true); ponerResultado(''); ponerError(''); };
  async function enviar(ruta: string, datos: object, word = false) {
    if (bloqueo.current || conflicto || !abort.current) return;
    bloqueo.current = true; ponerOcupado(true); ponerError(''); ponerResultado('');
    const cuerpoBase = JSON.stringify(datos);
    const identidad = `${ruta}:${cuerpoBase}`;
    const clave = peticiones.current.get(identidad) ?? crypto.randomUUID();
    peticiones.current.set(identidad, clave);
    const cuerpo = ruta.endsWith('/evaluaciones') ? JSON.stringify({ ...datos, clave_idempotencia: clave }) : cuerpoBase;
    try {
      const r = await fetch(ruta, { method: 'POST', signal: abort.current.signal, credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': clave }, body: cuerpo });
      if (!r.ok) {
        const d = await r.json() as { error?: { clave?: string }; clave?: string };
        ponerConflicto(r.status === 409);
        throw new Error(d.error?.clave ?? d.clave ?? 'fi2.error');
      }
      const datosRespuesta: unknown = word ? await r.blob() : await r.json();
      if (abort.current.signal.aborted) return;
      if (word) {
        const url = URL.createObjectURL(datosRespuesta as Blob);
        const enlace = document.createElement('a'); enlace.href = url; enlace.download = `${ruta.split('/').at(-1)}.docx`; enlace.click();
        setTimeout(() => URL.revokeObjectURL(url), 1000);
      } else ponerSucio(false);
      peticiones.current.delete(identidad);
      ponerResultado(word ? 'fi2.descargado' : 'fi2.guardado');
      return datosRespuesta;
    } catch (e: unknown) {
      if (!abort.current.signal.aborted) ponerError(e instanceof Error ? e.message : 'fi2.error');
    } finally {
      bloqueo.current = false;
      if (!abort.current.signal.aborted) ponerOcupado(false);
    }
  }
  return { ocupado, error, resultado, conflicto, editar, enviar, verCambios: (accion: () => void) => solicitarSalida(() => { ponerConflicto(false); ponerError(''); accion(); }) };
}
