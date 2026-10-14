export type Expediente = { id: string; numero: string | null; estado: string; version: number };
export type SPM = { version: number; respuesta: string; fecha_derivacion?: string; fecha_respuesta?: string; motivo: string; propuesta_medida: string; forzado_id?: string; motivo_forzado?: string };
export type Propuesta = { expediente_id: string; numero: string; version: number; motivo: string; propuesta_por: string; propuesta_en: string };
export type Opciones = { textos: Record<string, string> };
export class Fallo extends Error {
  constructor(public estado: number, public clave: string) { super(clave); }
}
export async function comprobar(r: Response) {
  if (r.ok) return;
  const datos = await r.json().catch(() => ({})) as { error?: { clave?: string } };
  throw new Fallo(r.status, datos.error?.clave ?? 'salidas.error');
}
export async function leer<T>(ruta: string, signal: AbortSignal): Promise<T> {
  const r = await fetch(ruta, { signal }); await comprobar(r); return r.json() as Promise<T>;
}
export function permiteCierre(s: SPM | null) {
  return !s || !!s.fecha_respuesta && ['inicio_desamparo_medida', 'medida_no_necesaria'].includes(s.respuesta);
}
export function claveFallo(f: unknown): string {
  if (!(f instanceof Fallo)) return 'salidas.error';
  if (f.estado === 409) return 'salidas.conflicto';
  if (f.estado === 403 || f.estado === 401) return 'salidas.sin_permiso';
  if (f.clave === 'expediente.cierre_segunda_persona') return 'salidas.segunda_persona';
  if (f.clave === 'expediente.cierre_bloqueado_anexo5') return 'salidas.bloqueo_spm';
  if (f.estado === 404) return 'salidas.no_encontrado';
  if (f.estado === 422 || f.estado === 400) return 'salidas.revisar_datos';
  return 'salidas.error';
}

export async function buscarPropuesta(id: string, signal: AbortSignal): Promise<Propuesta | null> {
  for (let desplazamiento = 0; desplazamiento <= 100000; desplazamiento += 100) {
    const filas = await leer<Propuesta[] | null>(`/api/coordinacion/cierres?limite=100&desplazamiento=${desplazamiento}`, signal);
    const propuesta = filas?.find(f => f.expediente_id === id);
    if (propuesta) return propuesta;
    if (!filas || filas.length < 100) return null;
  }
  throw new Error('salidas.error');
}
