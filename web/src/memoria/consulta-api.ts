import { FalloMemoria, pedir } from './api';

export type CeldaConsulta = { grupo: Record<string, string>; unidad: string; numerador: number; denominador: number | null; porcentaje_centesimas: number | null; aviso_porcentaje: string; sin_dato: number };
export type ApartadoConsulta = { codigo: string; total: Record<string, number>; celdas: CeldaConsulta[]; avisos: Record<string, number> };
export type CifrasConsulta = { anio: number; equipo_id: string; equipo_codigo: string; fecha_cifras: string; corte: string; definitivas: boolean; apartados: ApartadoConsulta[] };

export function rutaCifras(anio: number, equipo: string, word = false) {
  const parametros = new URLSearchParams(equipo ? { equipo } : {});
  return `/api/consulta/memoria/${anio}${word ? '/word' : ''}${parametros.size ? `?${parametros}` : ''}`;
}
export function leerCifras(anio: number, equipo: string, signal: AbortSignal) {
  return pedir<CifrasConsulta>(rutaCifras(anio, equipo), { signal });
}
export async function descargarCifras(anio: number, equipo: string, idioma: string, csrf: string, signal: AbortSignal) {
  const respuesta = await fetch(rutaCifras(anio, equipo, true), { method: 'POST', credentials: 'same-origin', signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ idioma }) });
  if (!respuesta.ok) {
    const datos = await respuesta.json();
    throw new FalloMemoria(datos.error?.clave ?? 'estadistica.memoria_no_disponible');
  }
  return respuesta.blob();
}
