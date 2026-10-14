export type Celda = { grupo: Record<string, string>; unidad: string; numerador: number; denominador: { conocido: boolean; valor: number }; porcentaje: { centesimas: number; aviso: string } | null; sin_dato: number };
export type Apartado = { codigo: string; contrato: { poblacion: string; denominador: string; corte: string; version_regla: string }; total: Record<string, number>; celdas: Celda[]; avisos: Record<string, number> };
export type Informe = { anio: number; equipo_id: string; equipo_codigo: string; corte: string; conocido_en: string; apartados: Apartado[] };
export type Personal = { hombres: number | null; mujeres: number | null };
export type Generales = { entidad_local: string; denominacion: string; equipos_convenio: number | null; psicologia: Personal; trabajo_social: Personal; educacion_social: Personal; observaciones: string };
export type Narrativas = { anio: number; equipo_id: string; version: number; a: Generales; d: string; e: string; f: string; g: string };
export type Equipo = { id: string; codigo: string };
export class FalloMemoria extends Error {
  constructor(public clave: string, public conflicto = false) { super(clave); }
}
export async function pedir<T>(ruta: string, opciones: RequestInit = {}): Promise<T> {
  const respuesta = await fetch(ruta, { credentials: 'same-origin', ...opciones });
  const datos = await respuesta.json();
  if (!respuesta.ok) throw new FalloMemoria(datos.error?.clave ?? datos.clave ?? 'estadistica.memoria_no_disponible', respuesta.status === 409);
  return datos as T;
}
export const consulta = (anio: number, equipo: string) => new URLSearchParams({ anio: String(anio), ...(equipo ? { etf: equipo } : {}) }).toString();
export async function descargar(anio: number, equipo: string, idioma: string, csrf: string, clave: string, signal: AbortSignal): Promise<Blob> {
  const respuesta = await fetch('/api/estadistica/memoria/word', { method: 'POST', signal, credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': clave }, body: JSON.stringify({ anio, equipo_id: equipo, idioma }) });
  if (!respuesta.ok) {
    const datos = await respuesta.json();
    throw new FalloMemoria(datos.error?.clave ?? datos.clave ?? 'estadistica.memoria_no_disponible');
  }
  return respuesta.blob();
}
export type EstadoMemoria = { anio: number; equipo_id: string; estado: 'borrador' | 'cerrada'; version: number; revision_actual: number };
type CeldaCerrada = { Grupo: Record<string, string>; Unidad: string; Celda: { Numerador: number; Denominador: { Conocido: boolean; Valor: number }; Porcentaje: { Centesimas: number; Aviso: string } | null; SinDato: number } };
type ApartadoCerrado = { Codigo: string; Contrato: Record<string, string>; Total: Record<string, number>; Celdas: CeldaCerrada[]; Avisos: Record<string, number> };
export type Instantanea = { cerrada_en: string; informe: { anio: number; equipo_id: string; EquipoCodigo: string; Corte: string; ConocidoEn: string; Narrativas: Narrativas; Apartados: ApartadoCerrado[] } };
// Cambia la representación del contrato HTTP; conserva las cifras de la revisión.
export function cifrasDeInstantanea(foto: Instantanea): Informe {
  const f = foto.informe;
  return { anio: f.anio, equipo_id: f.equipo_id, equipo_codigo: f.EquipoCodigo, corte: f.Corte, conocido_en: f.ConocidoEn,
    apartados: f.Apartados.map((a) => ({ codigo: a.Codigo, contrato: { poblacion: a.Contrato.Poblacion, denominador: a.Contrato.Denominador, corte: a.Contrato.Corte, version_regla: a.Contrato.VersionRegla },
      total: { familias: a.Total.Familias, menores: a.Total.Menores, adultos: a.Total.Adultos, comisiones: a.Total.Comisiones, informes: a.Total.Informes, familias_sin_relacion_menores: a.Total.FamiliasSinRelacionMenores }, avisos: a.Avisos,
      celdas: a.Celdas.map(({ Grupo, Unidad, Celda: c }) => ({ grupo: Object.fromEntries(Object.entries(Grupo).map(([k, v]) => [k.toLowerCase(), v])), unidad: Unidad, numerador: c.Numerador, denominador: { conocido: c.Denominador.Conocido, valor: c.Denominador.Valor }, porcentaje: c.Porcentaje ? { centesimas: c.Porcentaje.Centesimas, aviso: c.Porcentaje.Aviso } : null, sin_dato: c.SinDato })) })) };
}
export type ActoPendiente = { accion: 'cerrar' | 'reabrir'; anio: number; equipo: string; version: number; motivo: string; clave: string };
export function ejecutarActo(p: ActoPendiente, csrf: string, signal: AbortSignal) {
  return pedir<EstadoMemoria>(`/api/estadistica/memoria/${p.accion}?${consulta(p.anio, p.equipo)}`, { method: 'POST', signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': p.clave }, body: JSON.stringify({ version: p.version, ...(p.accion === 'reabrir' ? { motivo: p.motivo } : {}) }) });
}
