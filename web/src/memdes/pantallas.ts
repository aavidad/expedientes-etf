import type { Pantalla } from '../registro-pantallas';
import { DesgloseMemoria } from './DesgloseMemoria';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/memoria/desglose', perfiles: ['coordinacion', 'profesional_etf'], clave: '', menuDe: '/memoria', componente: DesgloseMemoria },
];
