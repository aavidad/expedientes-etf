import type { FichaBase } from '../fi1/modelo';
export type Familia = {
  direccion: string; telefono_contacto: string; email_contacto: string;
  composicion_familiar: string | null; composicion_otras_texto: string; familia_numerosa: boolean | null;
  vivienda: string | null; situacion_economica: string | null; ingresos_anuales_centimos: number | null; tramo_smi: string | null;
};
export type Adulto = {
  id: string; orden: number; nombre: string; apellidos: string; papel: string; sexo: string;
  fecha_nacimiento: string | null; edad_declarada: number | null; nivel_educativo: string | null;
  situacion_laboral: string | null; situacion_laboral_texto: string; es_tutor: boolean | null;
  telefono: string; email: string; otros_datos: string;
};
export type DatosFamilia = FichaBase & { familia: Familia; adultos: Adulto[] };
export function nuevoAdulto(): Adulto {
  return { id: '', orden: 1, nombre: '', apellidos: '', papel: '', sexo: '', fecha_nacimiento: null, edad_declarada: null,
    nivel_educativo: null, situacion_laboral: null, situacion_laboral_texto: '', es_tutor: null, telefono: '', email: '', otros_datos: '' };
}
export function centimos(valor: string): number | null {
  if (!valor.trim()) return null;
  if (!/^\d+(?:[.,]\d{1,2})?$/.test(valor.trim())) return NaN;
  const [entero, decimal = ''] = valor.trim().replace(',', '.').split('.');
  const cantidad = Number(entero) * 100 + Number(decimal.padEnd(2, '0'));
  return Number.isSafeInteger(cantidad) ? cantidad : NaN;
}
