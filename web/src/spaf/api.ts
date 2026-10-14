export type Dato<T> = { Valor: T; Conocido: boolean };
export type Anexo = { anexo: number; fecha: string };
export type Fila = {
  expediente_id: string; numero: string; version_ficha: string; alta: boolean; baja: boolean;
  numero_menores: Dato<number>; menores: Dato<{ nombre: string; apellidos: string; nacimiento: Dato<string>; centro_proteccion: Dato<string> }[]>;
  guardadores: Dato<{ nombre: string; parentesco: string }[]>; municipio: Dato<string>; zts: Dato<string>;
  caracteristicas_familia: Dato<string[]>; derivacion: Dato<Anexo>; servicio_derivante: Dato<string>; coordinador_caso: Dato<string>;
  aceptacion: Dato<string>; fechas_anexos: Anexo[] | null; fase_intervencion: Dato<string>; temporalizacion_meses: Dato<number>;
  finalidad: Dato<string>; pronostico: Dato<string>; instrumentos: Dato<string[]>;
  cierres: { fecha: string; motivo: Dato<string> }[] | null;
};
export type Tabla = {
  anio: number; mes: number; equipo_id: string; version_regla: string; filas: Fila[] | null;
  resumen: { altas: number; bajas: number; casos: number; menores: number; casos_sin_dato_menores: number };
  avisos: Record<string, number> | null;
};
export type Opciones = { equipos: { id: string; clave_texto: string }[]; textos: Record<string, Record<string, string>> };
export type VistaEmision = Pick<Tabla, 'equipo_id' | 'anio' | 'mes' | 'version_regla'> & { filas: Pick<Fila, 'expediente_id' | 'version_ficha'>[] | null };
export class Fallo extends Error { constructor(public clave: string, public estado = 0) { super(clave); } }
const uuid = (v: unknown): v is string => typeof v === 'string' && /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i.test(v);
const dato = (v: unknown): v is Dato<unknown> => !!v && typeof v === 'object' && typeof (v as Dato<unknown>).Conocido === 'boolean' && 'Valor' in v;
const campos = ['numero_menores', 'menores', 'guardadores', 'municipio', 'zts', 'caracteristicas_familia', 'derivacion', 'servicio_derivante', 'coordinador_caso', 'aceptacion', 'fase_intervencion', 'temporalizacion_meses', 'finalidad', 'pronostico', 'instrumentos'] as const;
export function validarTabla(v: Tabla, equipo: string, mes: string): boolean {
  return !!v && v.equipo_id === equipo && `${v.anio}-${String(v.mes).padStart(2, '0')}` === mes && typeof v.version_regla === 'string' && !!v.version_regla &&
    !!v.resumen && (['altas', 'bajas', 'casos', 'menores', 'casos_sin_dato_menores'] as const).every(k => Number.isSafeInteger(v.resumen[k]) && v.resumen[k] >= 0) &&
    (v.filas === null || Array.isArray(v.filas)) && (v.filas ?? []).every(f => uuid(f.expediente_id) && typeof f.numero === 'string' && typeof f.version_ficha === 'string' && typeof f.alta === 'boolean' && typeof f.baja === 'boolean' && campos.every(c => dato(f[c]))) &&
    v.resumen.casos === (v.filas ?? []).length;
}
async function comprobar(respuesta: Response): Promise<void> {
  if (respuesta.ok) return;
  throw new Fallo(respuesta.status === 409 ? 'spaf.conflicto' : respuesta.status === 401 || respuesta.status === 403 ? 'spaf.sin_permiso' : respuesta.status === 400 || respuesta.status === 422 ? 'spaf.peticion_invalida' : 'spaf.error', respuesta.status);
}
export async function leer<T>(ruta: string, signal: AbortSignal): Promise<T> {
  const respuesta = await fetch(ruta, { signal, credentials: 'same-origin', cache: 'no-store' });
  await comprobar(respuesta); return respuesta.json() as Promise<T>;
}
export function solicitudWord(tabla: VistaEmision, idioma: string) {
  return { idioma, versiones: { regla: tabla.version_regla, fichas: Object.fromEntries((tabla.filas ?? []).map(f => [f.expediente_id, f.version_ficha])) } };
}
export async function emitir(tabla: VistaEmision, idioma: string, csrf: string, clave: string, signal: AbortSignal) {
  const respuesta = await fetch(`/api/equipos/${encodeURIComponent(tabla.equipo_id)}/tabla-spaf/${tabla.anio}/${tabla.mes}.docx`, {
    method: 'POST', credentials: 'same-origin', signal,
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf, 'Idempotency-Key': clave }, body: JSON.stringify(solicitudWord(tabla, idioma)),
  });
  if (respuesta.status >= 500) throw new Fallo('spaf.descarga_desconocida');
  await comprobar(respuesta);
  if (!uuid(respuesta.headers.get('X-Emision-ID')) || respuesta.headers.get('Content-Type')?.split(';')[0] !== 'application/vnd.openxmlformats-officedocument.wordprocessingml.document') throw new Fallo('spaf.descarga_desconocida');
  const contenido = await respuesta.blob();
  if (!contenido.size) throw new Fallo('spaf.descarga_desconocida');
  return { contenido, cambiado: !!respuesta.headers.get('X-Aviso-Clave'), repetida: respuesta.headers.get('X-Idempotencia-Repetida') === 'true' };
}
export function mesAnterior(ahora = new Date()): string {
  const mes = new Date(ahora.getFullYear(), ahora.getMonth() - 1, 1);
  return `${mes.getFullYear()}-${String(mes.getMonth() + 1).padStart(2, '0')}`;
}
export const mesValido = (mes: string): boolean => /^\d{4}-\d{2}$/.test(mes) && Number(mes.slice(0, 4)) >= 1 && Number(mes.slice(0, 4)) <= 9998 && Number(mes.slice(5)) >= 1 && Number(mes.slice(5)) <= 12;
export function iniciales(nombre: string): string {
  return nombre.trim().split(/\s+/u).filter(Boolean).map(p => `${Array.from(p)[0]}.`).join(' ');
}
