import { formulario, menor, adulto, type Formulario, type Opciones } from './modelo';

export type LecturaActa = { contenido: { tipo: string; plantilla_reconocida: boolean; campos: { codigo: string; valor: string; motivos?: string[] }[] }; antivirus: string };
const normalizar = (s: string) => s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLocaleLowerCase().replace(/[^a-z0-9]/g, '');
function unico(campos: LecturaActa['contenido']['campos'], codigo: string) {
  const valores = [...new Set(campos.filter(c => c.codigo === codigo).map(c => c.valor).filter(Boolean))];
  return valores.length === 1 ? valores[0] : '';
}
function opcion(valor: string, opciones: { id: string; clave_texto: string }[], textos: Record<string, string>) {
  const encontradas = opciones.filter(o => normalizar(textos[o.clave_texto] ?? '') === normalizar(valor));
  return valor && encontradas.length === 1 ? encontradas[0].id : '';
}
// Todo valor leído es una propuesta editable. No completa datos que faltan.
export function desdeActas(equipo: string, opciones: Opciones, aceptacion: LecturaActa, cierre?: LecturaActa): Formulario {
  const a = aceptacion.contenido.campos;
  const c = cierre?.contenido.campos ?? [];
  const numero = unico(a, 'numero');
  const numeroCierre = unico(c, 'numero');
  const codigoEquipo = opciones.equipos.find(e => e.id === equipo)?.codigo;
  for (const campos of [a,c]) {
    const equipoLeido = unico(campos,'equipo');
    const id = opcion(equipoLeido,opciones.equipos,opciones.textos);
    const codigo = equipoLeido.match(/(?:E\.?T\.?F\.?\s*[-:]?\s*)(\d{1,2})\b/i)?.[1]?.padStart(2,'0');
    if (id && id !== equipo || codigo && codigoEquipo && codigo !== codigoEquipo) throw new Error('carga.actas.equipo_distinto');
  }
  if (numero && codigoEquipo && numero.split('-')[0] !== codigoEquipo) throw new Error('carga.actas.equipo_distinto');
  if (numero && numeroCierre && numero !== numeroCierre) throw new Error('carga.actas.numero_distinto');
  if (numeroCierre && codigoEquipo && numeroCierre.split('-')[0] !== codigoEquipo) throw new Error('carga.actas.equipo_distinto');
  const f = { ...formulario(), equipo, numero, aceptacion: unico(a, 'fecha'), estado: cierre ? 'cerrada' : '', descripcion: unico(a, 'descripcion'), familiaNombre: unico(a, 'familia') };
  for (const [campo,papel] of [['madre','MADRE'],['padre','PADRE']]) {
    const nombre = unico(a,campo);
    if (nombre) f.adultos.push({ ...adulto(),nombre,papel,nombreCompleto:true });
  }
  f.municipio = opcion(unico(a, 'municipio'), opciones.municipios.filter(m => m.equipo_id === equipo && m.ambito_diputacion), opciones.textos);
  f.centro = opcion(unico(a, 'centro'), opciones.centros, opciones.textos);
  const subprograma = unico(a, 'subprograma');
  const sub = opciones.subprogramas.filter(o => normalizar(opciones.textos[o.clave_texto] ?? '') === normalizar(subprograma));
  f.subprograma = subprograma && sub.length === 1 ? sub[0].codigo : '';
  const menores = [];
  for (const campo of a) {
    if (campo.codigo === 'menor.nombre' && menores.length < 15) menores.push({ ...menor(), nombre: campo.valor, nombreCompleto: true });
    const actual = menores[menores.length - 1];
    if (actual && campo.codigo === 'menor.nacimiento') actual.fecha_nacimiento = campo.valor;
    if (actual && campo.codigo === 'menor.sexo') {
      const sexo = normalizar(campo.valor);
      if (['h', 'varon', 'masculino', 'xvaronfemenino'].includes(sexo) || /☒\s*Var[oó]n\s*☐\s*Femenino/i.test(campo.valor)) actual.sexo = 'H';
      if (['m', 'femenino', 'mujer', 'varonxfemenino'].includes(sexo) || /☐\s*Var[oó]n\s*☒\s*Femenino/i.test(campo.valor)) actual.sexo = 'M';
    }
  }
  if (menores.length) f.menores = menores;
  if (cierre) {
    f.episodios[0].cierre_fecha = unico(c, 'fecha');
    f.episodios[0].motivo_acta_texto = unico(c, 'motivo');
    const motivo = opcion(unico(c, 'motivo'), Object.keys(opciones.textos).filter(k => k.startsWith('motivo_finalizacion.')).map(k => ({ id: k.split('.')[1], clave_texto: k })), opciones.textos);
    f.episodios[0].motivo_cierre = motivo;
  }
  return f;
}
