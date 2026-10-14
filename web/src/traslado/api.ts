import { useEffect, useRef, useState } from 'react';
import { comprobar, Fallo, leer } from '../salidas/api';
export { leer };
export type Seguimiento = { id: string; version: number; estado: string; fecha_cese_medida: string; tramo_desde: string; fecha_fin_prevista: string; fecha_terminacion?: string; motivo?: string; motivo_reapertura?: string };
export type OpcionesSeguimiento = { motivos: { codigo: string; clave_texto: string }[]; textos: Record<string, string> };
export type Operacion = { ruta: string; cuerpo: string; clave: string };
export function fecha(valor: string, idioma: string) { return new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${valor}T12:00:00Z`)); }
export function hoy() { const p = new Intl.DateTimeFormat(undefined, { timeZone: 'Europe/Madrid', year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date()); return ['year', 'month', 'day'].map(k => p.find(v => v.type === k)?.value).join('-'); }
export function fechaValida(valor: string, desde = '', hasta = hoy()) { return /^\d{4}-\d{2}-\d{2}$/.test(valor) && Number.isFinite(Date.parse(valor)) && new Date(`${valor}T12:00:00Z`).toISOString().slice(0, 10) === valor && valor >= desde && valor <= hasta; }
export function fallo(e: unknown) {
  if (!(e instanceof Fallo)) return 'traslado.error';
  if (e.estado === 409) return 'traslado.conflicto';
  if (e.estado === 401 || e.estado === 403) return 'traslado.sin_permiso';
  if (e.estado === 404) return 'traslado.no_encontrado';
  if (e.clave === 'expediente.cierre_segunda_persona') return 'traslado.segunda_persona';
  if (e.clave === 'expediente.cierre_bloqueado_anexo5') return 'traslado.bloqueo_spm';
  if (e.clave === 'expediente.cierre_invalido') return 'traslado.fecha_cierre_revisar';
  if (e.estado === 400 || e.estado === 422) return 'traslado.revisar_datos';
  return 'traslado.error';
}
// Un resultado incierto conserva exactamente el cuerpo y la clave enviados.
export function useOperacion<T>(id: string, csrf: string, validar: (datos: T) => boolean, aceptar: (datos: T) => void) {
  const [error, ponerError] = useState(''); const [ocupado, ponerOcupado] = useState(false); const [pendiente, ponerPendiente] = useState<Operacion>();
  const abort = useRef<AbortController | null>(null); const enviando = useRef(false);
  useEffect(() => { const c = new AbortController(); abort.current = c; enviando.current = false; ponerPendiente(undefined); ponerError(''); ponerOcupado(false); return () => c.abort(); }, [id]);
  async function enviar(op: Operacion) {
    const signal = abort.current?.signal; if (!signal || enviando.current || signal.aborted) return;
    enviando.current = true; ponerOcupado(true); ponerError('');
    try {
      const r = await fetch(op.ruta, { method: 'POST', signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': op.clave }, body: op.cuerpo });
      await comprobar(r); const datos = await r.json() as T;
      if (!validar(datos)) throw new Error('traslado.error');
      if (!signal.aborted) { ponerPendiente(undefined); aceptar(datos); }
    } catch (e) { if (!signal.aborted) { ponerError(fallo(e)); ponerPendiente(!(e instanceof Fallo) || e.estado >= 500 ? op : undefined); } }
    finally { if (abort.current?.signal === signal) enviando.current = false; if (!signal.aborted) ponerOcupado(false); }
  }
  return { error, ponerError, ocupado, pendiente, enviar, bloqueado: ocupado || !!pendiente || error === 'traslado.conflicto' || error === 'traslado.sin_permiso' };
}
