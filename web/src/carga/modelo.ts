export type Opcion = { id: string; clave_texto: string; codigo?: string };
export type Opciones = {
  equipos: Opcion[]; centros: Opcion[];
  municipios: (Opcion & { centro_ss_id: string | null; equipo_id: string | null; ambito_diputacion: boolean })[];
  subprogramas: { codigo: string; clave_texto: string }[]; textos: Record<string, string>;
};
export type Episodio = { apertura_fecha: string; cierre_fecha: string; motivo_cierre: string; motivo_acta_texto: string; motivo_reapertura: string };
export type Menor = { id: string; nombre: string; apellidos: string; sexo: string; fecha_nacimiento: string; nombreCompleto?: boolean; nombreLeido?: string };
export type Adulto = Menor & { papel: string };
export type Formulario = {
  equipo: string; numero: string; estado: string; tipo: string; derivacion: string; aceptacion: string;
  municipio: string; centro: string; subprograma: string; puesto: string; descripcion: string;
  noAceptacion: string; causa: string; direccion: string; telefono: string; email: string;
  familiaNombre: string; adultos: Adulto[];
  menores: Menor[]; episodios: Episodio[];
};
export type Resumen = { fila: number; numero: string; estado: string; iniciales: string };
export type Informe = { avisos?: { fila: number; clave: string }[]; valida: boolean; casos: number; lote_id?: string; huella?: string; filas?: Resumen[]; incidencias: { fila: number; columna: string; clave: string; mensaje: string }[] };
export type Recibo = { referencia?: number; lote_id: string; expedientes: string[]; marca: string };
export const estados = ['aceptada', 'en_intervencion', 'cerrada', 'no_aceptada'];
export const columnas = ['numero', 'fecha_derivacion', 'estado', 'tipo_anexo', 'fecha_aceptacion', 'municipio_id', 'centro_ss_id', 'subprograma', 'profesional_referencia_puesto', 'profesional_referencia_id', 'descripcion_situacion', 'sexos_menores', 'fecha_no_aceptacion', 'causa_no_aceptacion', 'campos_derivacion', 'episodios', 'familia', 'menores'];
export function episodio(): Episodio { return { apertura_fecha: '', cierre_fecha: '', motivo_cierre: '', motivo_acta_texto: '', motivo_reapertura: '' }; }
export function menor(): Menor { return { id: crypto.randomUUID(), nombre: '', apellidos: '', sexo: '', fecha_nacimiento: '' }; }
export function adulto(): Adulto { return { ...menor(), papel: '' }; }
export function formulario(): Formulario { return { equipo: '', numero: '', estado: '', tipo: '', derivacion: '', aceptacion: '', municipio: '', centro: '', subprograma: '', puesto: '', descripcion: '', noAceptacion: '', causa: '', direccion: '', telefono: '', email: '', familiaNombre: '', adultos: [], menores: [menor()], episodios: [episodio()] }; }
export function iniciales(nombre: string): string { return nombre.split(/\s+/).filter(Boolean).map(p => `${p[0]}.`).join(' '); }
export function tieneDatos(f: Formulario): boolean {
  return Object.entries(f).some(([k,v]) => k !== 'equipo' && typeof v === 'string' && !!v) || f.adultos.length > 0 || f.menores.some(m => Object.entries(m).some(([k,v]) => k !== 'id' && !!v)) || f.episodios.some(e => Object.values(e).some(Boolean));
}
export function separarNombre<T extends Menor>(persona:T):T { return { ...persona, nombreLeido:persona.nombre, nombreCompleto:false, nombre:'', apellidos:'' }; }
export function preparar(f: Formulario) {
  const campos = f.menores.filter(m => m.nombre.trim()).map(m => ({ codigo: 'menores.persona.nombre', valor: [m.nombre, m.apellidos].filter(Boolean).join(' '), pagina: 1, dudoso: false, motivos: [] }));
  if (f.familiaNombre.trim()) campos.push({ codigo: 'familia.nombre', valor: f.familiaNombre, pagina: 0, dudoso: false, motivos: [] });
  const caso = {
    numero: f.estado === 'no_aceptada' ? '' : f.numero, fecha_derivacion: f.derivacion, estado: f.estado,
    derivacion_datos: { contenido: { tipo: f.tipo, campos, avisos: [], version: '1', plantilla_reconocida: false }, esquema_version: '1' },
    ...(f.estado === 'no_aceptada' ? { no_aceptacion: { version: 1, fecha: f.noAceptacion, motivo: f.causa, subprograma: f.subprograma } } : {
      aceptacion: { version: 1, fecha_acta: f.aceptacion, municipio_id: f.municipio, centro_ss_id: f.centro, equipo_id: f.equipo, subprograma: f.subprograma, profesional_referencia_puesto: f.puesto, profesional_referencia_id: null, menores: f.menores.map((m, i) => ({ orden: i + 1, sexo: m.sexo })), descripcion_situacion: f.descripcion },
    }),
    episodios: f.estado === 'no_aceptada' ? [] : f.episodios.map((e, i) => ({ ...e, apertura_fecha: i === 0 ? f.aceptacion : e.apertura_fecha, ...(e.cierre_fecha ? {} : { motivo_cierre: '', motivo_acta_texto: '' }) })),
    familia: f.estado === 'no_aceptada' ? [] : [
      ...(f.direccion || f.telefono || f.email ? [{ version: 1, vigente_desde: f.aceptacion, familia: { direccion: f.direccion, telefono_contacto: f.telefono, email_contacto: f.email } }] : []),
      ...(f.adultos.length ? [{ version: 1, vigente_desde: f.aceptacion, adultos: f.adultos.map((a, i) => ({ id: a.id, nombre: a.nombre, apellidos: a.apellidos, sexo: a.sexo, papel: a.papel, orden: i + 1, fecha_nacimiento: a.fecha_nacimiento || null })) }] : []),
    ],
    menores: f.estado === 'no_aceptada' ? [] : [{ version: 1, vigente_desde: f.aceptacion, menores: f.menores.map((m, i) => ({ id: m.id, nombre: m.nombre, apellidos: m.apellidos, sexo: m.sexo, orden: i + 1, fecha_nacimiento: m.fecha_nacimiento || null, subprograma: f.subprograma })) }],
  };
  return { equipo_id: f.equipo, casos: [caso] };
}
export function hojaInventada(t: (k: string) => string, opciones?: Opciones, equipo?: string): string {
  const datos: Record<string, string>[] = [];
  const municipio = opciones?.municipios.find(m => m.ambito_diputacion && m.equipo_id === equipo && m.centro_ss_id);
  const codigo = opciones?.equipos.find(e => e.id === equipo)?.codigo;
  if (municipio && codigo) {
    for (const [i, estado] of ['en_intervencion', 'cerrada'].entries()) {
      const cierre = estado === 'cerrada';
      datos.push({ numero: `${codigo}-${91 + i}-26`, fecha_derivacion: '2026-01-08', estado, tipo_anexo: 'anexo3', fecha_aceptacion: '2026-01-15', municipio_id: municipio.id, centro_ss_id: municipio.centro_ss_id ?? '', subprograma: 'RIESGO_SOCIAL', profesional_referencia_puesto: 'TS', descripcion_situacion: t('carga.hoja_descripcion'), sexos_menores: JSON.stringify([{ orden: 1, sexo: 'M' }]), campos_derivacion: JSON.stringify([{ codigo: 'menores.persona.nombre', valor: t(i ? 'carga.hoja_nombre_cerrado' : 'carga.hoja_nombre'), pagina: 1, dudoso: false, motivos: [] }]), episodios: JSON.stringify(cierre ? [{ apertura_fecha: '2026-01-15', cierre_fecha: '2026-04-20', motivo_cierre: 'OBJ_TOTAL', motivo_acta_texto: '' }] : [{ apertura_fecha: '2026-01-15', cierre_fecha: '2026-03-10', motivo_cierre: 'TRASLADO', motivo_acta_texto: '' }, { apertura_fecha: '2026-05-04', motivo_reapertura: t('carga.hoja_reapertura') }]) });
    }
  }
  datos.push({ fecha_derivacion: '2026-02-04', estado: 'no_aceptada', tipo_anexo: 'anexo3', subprograma: 'RIESGO_SOCIAL', fecha_no_aceptacion: '2026-02-12', causa_no_aceptacion: t('carga.hoja_causa'), campos_derivacion: JSON.stringify([{ codigo: 'menores.persona.nombre', valor: t('carga.hoja_nombre_no_aceptado'), pagina: 1, dudoso: false, motivos: [] }]) });
  const celda = (s: string) => `"${s.replaceAll('"', '""')}"`;
  return '\uFEFF' + columnas.join(';') + '\r\n' + datos.map(dato => columnas.map(c => celda(dato[c] ?? '')).join(';')).join('\r\n') + '\r\n';
}
