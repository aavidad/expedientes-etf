export type Campo = { codigo: string; valor: string; dudoso: boolean };
export type Derivacion = {
  id: string; estado: string; numero: string | null; version: number; municipio_id?: string | null; equipo_id?: string | null;
  lectura_anexo: { leida_en: string };
  derivacion_datos: { ubicacion_confirmada?: { municipio_id: string; centro_id: string; equipo_id: string }; contenido: { tipo: string; campos: Campo[] }; propuesta_municipio: { Candidatos: { Codigo: string; Fragmento: string; FueraAmbitoDiputacion: boolean }[] } };
  aceptacion?: Formulario;
  no_aceptacion?: { fecha: string; motivo: string; subprograma: string };
  fecha_lista_espera?: string;
};
export type Opcion = { id: string; clave_texto: string };
export type Municipio = Opcion & { ine: string; centro_ss_id: string | null; equipo_id: string | null; ambito_diputacion: boolean };
export type Opciones = { municipios: Municipio[]; centros: Opcion[]; equipos: Opcion[]; subprogramas: { codigo: string; clave_texto: string }[]; textos: Record<string, string> };
export type Registro = { version: number; estado: string; numero_registro?: string; fecha_registro?: string };
export type Emision = { id: string; tipo: string; generada_en: string; plantilla_version: string; registro?: Registro };
export type Formulario = {
  fecha_acta: string; municipio_id: string; centro_ss_id: string; equipo_id: string;
  subprograma: string; profesional_referencia_puesto: string; profesional_referencia_id: string | null;
  menores: { orden: number; sexo: string }[]; descripcion_situacion: string;
};
export class FalloAPI extends Error {
  constructor(public estado: number, public clave: string) { super(clave); }
}
export async function comprobar(respuesta: Response): Promise<void> {
  if (respuesta.ok) return;
  const cuerpo = await respuesta.json().catch(() => ({})) as { error?: string | { clave?: string } };
  throw new FalloAPI(respuesta.status, typeof cuerpo.error === 'string' ? cuerpo.error : cuerpo.error?.clave ?? 'decision.error');
}
export async function leer<T>(ruta: string, signal: AbortSignal): Promise<T> {
  const respuesta = await fetch(ruta, { signal }); await comprobar(respuesta); return respuesta.json() as Promise<T>;
}
export const tipos = {
  aceptada: ['entrada/aceptacion', 'entrada/aceptacion/oficio_sscc', 'entrada/aceptacion/oficio_spm'],
  no_aceptada: ['entrada/no_aceptacion'],
  lista_espera: ['entrada/no_aceptacion/lista_espera'],
};
export const claveDocumento: Record<string, string> = {
  'entrada/aceptacion': 'decision.acta_aceptacion',
  'entrada/aceptacion/oficio_sscc': 'decision.oficio_sscc',
  'entrada/aceptacion/oficio_spm': 'decision.oficio_spm',
  'entrada/no_aceptacion': 'decision.acta_no_aceptacion',
  'entrada/no_aceptacion/lista_espera': 'decision.oficio_espera',
};
export function preparar(d: Derivacion, o: Opciones): Formulario {
  if (d.aceptacion) return d.aceptacion;
  const candidatos = d.derivacion_datos.propuesta_municipio.Candidatos ?? [];
  const ubicacion = d.derivacion_datos.ubicacion_confirmada;
  const municipio = ubicacion ? o.municipios.find(m => m.id === ubicacion.municipio_id) : d.municipio_id ? o.municipios.find(m => m.id === d.municipio_id) : candidatos.length === 1 ? o.municipios.find(m => m.ine === candidatos[0].Codigo) : undefined;
  const campos = d.derivacion_datos.contenido.campos;
  return {
    fecha_acta: '', municipio_id: municipio?.id ?? '', centro_ss_id: ubicacion?.centro_id ?? municipio?.centro_ss_id ?? '', equipo_id: ubicacion?.equipo_id ?? d.equipo_id ?? municipio?.equipo_id ?? '',
    subprograma: d.no_aceptacion?.subprograma ?? (d.derivacion_datos.contenido.tipo === 'anexo3' ? 'RIESGO_SOCIAL' : ''),
    profesional_referencia_puesto: '', profesional_referencia_id: null,
    menores: campos.filter(c => c.codigo === 'menores.persona.nombre').map((_, i) => ({ orden: i + 1, sexo: '' })),
    descripcion_situacion: campos.find(c => c.codigo === 'valoracion.descripcion')?.valor ?? '',
  };
}
export function faltantes(f: Formulario): string[] {
  const campos = ['fecha_acta', 'municipio_id', 'centro_ss_id', 'equipo_id', 'subprograma', 'profesional_referencia_puesto', 'descripcion_situacion'] as const;
  return [...campos.filter(c => !f[c].trim()), ...f.menores.filter(m => !m.sexo).map(m => `sexo-${m.orden}`), ...(f.menores.length ? [] : ['menores'])];
}
