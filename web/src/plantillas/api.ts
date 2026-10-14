export type Plantilla = { id: string; tipo: string; idioma: string; version: string };
export type Version = {
  plantilla: Plantilla; activa: boolean; version_actual: number; subida_por: string; subida_en: string;
  antivirus: string; marcadores: string; activaciones: { por: string; en: string; revision: number }[];
};
export class FalloPlantilla extends Error {
  constructor(public clave: string, public faltan: string[] = [], public sobran: string[] = [], public antivirus = '') { super(clave); }
}
export async function peticion<T>(ruta: string, opciones: RequestInit): Promise<T> {
  const respuesta = await fetch(ruta, { ...opciones, credentials: 'same-origin' });
  const cuerpo = await respuesta.json();
  if (!respuesta.ok) {
    const clave = typeof cuerpo.error === 'string' ? cuerpo.error : cuerpo.error?.clave;
    const claves: Record<string, string> = { 'documento.no_disponible': 'plantillas.no_disponible', 'documento.peticion_invalida': 'plantillas.peticion_invalida', 'documento.acceso_denegado': 'plantillas.acceso_denegado', 'documento.plantilla_alterada': 'plantillas.alterada' };
    throw new FalloPlantilla(claves[clave] ?? clave ?? 'plantillas.no_disponible', cuerpo.marcadores?.faltan ?? [], cuerpo.marcadores?.sobran ?? [], cuerpo.antivirus ?? '');
  }
  return cuerpo as T;
}
export async function cargar(signal: AbortSignal) {
  const [versiones, usuarios] = await Promise.all([
    peticion<(Omit<Version, 'plantilla'> & { plantilla: { ID: string; Tipo: string; Idioma: string; Version: string } })[]>('/api/admin/plantillas', { signal }),
    peticion<{ id: string; nombre: string }[]>('/api/admin/usuarios', { signal }),
  ]);
  if (!Array.isArray(versiones) || !Array.isArray(usuarios)) throw new FalloPlantilla('plantillas.no_disponible');
  return { versiones: versiones.map((v) => ({ ...v, plantilla: { id: v.plantilla.ID, tipo: v.plantilla.Tipo, idioma: v.plantilla.Idioma, version: v.plantilla.Version } })), nombres: Object.fromEntries(usuarios.map((u) => [u.id, u.nombre])) };
}
export function subir(archivo: File, tipo: string, idioma: string, version: string, previa: number, csrf: string, signal: AbortSignal) {
  const consulta = new URLSearchParams({ tipo, idioma, version, version_previa: String(previa) });
  return peticion(`/api/admin/plantillas/versiones?${consulta}`, { method: 'POST', signal, headers: { 'Content-Type': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'X-CSRF-Token': csrf }, body: archivo });
}
export function activar(version: Version, csrf: string, signal: AbortSignal) {
  return peticion(`/api/admin/plantillas/${encodeURIComponent(version.plantilla.id)}/activar`, { method: 'POST', signal, headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ version_previa: version.version_actual, idioma:version.plantilla.idioma }) });
}
