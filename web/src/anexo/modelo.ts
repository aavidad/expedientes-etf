export type Campo = { codigo: string; valor: string; pagina: number; dudoso: boolean; motivos: string[] | null };
export type Opcion = { id: string; clave_texto: string; centro_id?: string; equipo_id?: string; fuera_ambito?: boolean };
export type Ubicacion = { municipio_id: string; centro_id: string; equipo_id: string };
export type Analisis = {
  lectura_anexo: { id: string; antivirus: string; antivirus_detalle?: string; formato: string };
  derivacion_datos: {
    contenido: { tipo: string; campos: Campo[]; avisos: string[] };
    opciones_ubicacion: { municipios: Opcion[]; centros: Opcion[]; equipos: Opcion[] };
    propuesta_municipio: { Estado: number; Candidatos: { MunicipioID: string; CentroID?: string; EquipoID?: string; Fragmento: string; Campo: number }[] };
  };
};
export const apartados = ['servicio', 'familia', 'menores', 'sociodemografia', 'valoracion', 'intervenciones', 'observaciones', 'desproteccion', 'resoluciones'] as const;
export const manuales = ['servicio.nombre', 'servicio.profesional', 'servicio.telefono', 'servicio.correo', 'familia.direccion', 'familia.contacto.nombre', 'familia.contacto.apellidos', 'familia.contacto.telefono', 'menores.persona.nombre', 'menores.persona.nacimiento', 'menores.persona.escolarizacion', 'sociodemografia.adulto.nombre', 'sociodemografia.adulto.papel', 'sociodemografia.adulto.edad', 'valoracion.descripcion', 'intervenciones.descripcion', 'observaciones.texto'];
export function camposIniciales(analisis: Analisis): Campo[] {
  const campos = analisis.derivacion_datos.contenido.campos ?? [];
  return campos.length ? campos : manuales.map((codigo) => ({ codigo, valor: '', pagina: 0, dudoso: true, motivos: [] }));
}
export function propuestaInicial(analisis: Analisis): Ubicacion {
  const candidatos = analisis.derivacion_datos.propuesta_municipio.Candidatos ?? [];
  const unico = candidatos.length === 1 ? candidatos[0] : undefined;
  const opciones = analisis.derivacion_datos.opciones_ubicacion;
  return { municipio_id: unico?.MunicipioID ?? '', centro_id: opciones?.centros.some((o) => o.id === unico?.CentroID) ? unico?.CentroID ?? '' : '', equipo_id: opciones?.equipos.some((o) => o.id === unico?.EquipoID) ? unico?.EquipoID ?? '' : '' };
}
export function claveError(cuerpo: unknown, respaldo = 'anexo.error'): string {
  if (typeof cuerpo === 'object' && cuerpo !== null && 'error' in cuerpo) {
    const error = cuerpo.error;
    if (typeof error === 'object' && error !== null && 'clave' in error && typeof error.clave === 'string') return error.clave;
  }
  return respaldo;
}
