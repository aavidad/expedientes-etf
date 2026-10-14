export type Menor = {
  id: string; orden: number; nombre: string; apellidos: string; sexo: string;
  fecha_nacimiento: string | null; nivel_escolarizacion: string | null; centro_educativo: string;
  subprograma: string; centro_proteccion_codigo: string | null; centro_proteccion_texto: string;
  propuesto_atencion_temprana: boolean | null;
};
export type Marca = { codigo: string; texto_otros?: string; menores_excluidos: string[] };
export const listas = ['coordinacion', 'factor_familia', 'factor_menor', 'tipologia'] as const;
export type Lista = typeof listas[number];
export type Intervencion = { fase: string | null; listas: Record<Lista, Marca[] | null> };
export type FichaBase = { id: string; estado: string; version: number; fecha_aceptacion: string };
export type FichaMenores = FichaBase & { subprograma: string; menores: Menor[] };
export type FichaIntervencion = FichaBase & { intervencion: Intervencion | null };
export type Aviso = { clave_texto: string; campo: string; sujeto_id?: string; apartado?: string };
export function opciones(etiquetas: Record<string, string>, lista: string) {
  return Object.entries(etiquetas).filter(([clave]) => clave.startsWith(lista + '.')).map(([clave, etiqueta]) => ({ codigo: clave.slice(lista.length + 1), etiqueta }));
}
export function nuevaIntervencion(): Intervencion {
  return { fase: null, listas: { coordinacion: null, factor_familia: null, factor_menor: null, tipologia: null } };
}
export function nuevoMenor(orden: number, subprograma: string): Menor {
  return { id: '', orden, nombre: '', apellidos: '', sexo: '', fecha_nacimiento: null, nivel_escolarizacion: null, centro_educativo: '', subprograma,
    centro_proteccion_codigo: null, centro_proteccion_texto: '', propuesto_atencion_temprana: null };
}
export const campoID = (campo: string, sujeto?: string) => `fi1-${campo}${sujeto ? '-' + sujeto : ''}`;
