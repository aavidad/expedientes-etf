import { BandejaEquipo, FichaEquipo } from './equipo';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/equipo', perfiles: ['profesional_etf', 'coordinacion'], clave: 'equipo.titulo', componente: BandejaEquipo },
  { ruta: '/equipo/expediente', perfiles: ['profesional_etf', 'coordinacion'], clave: '', menuDe: '/equipo', componente: FichaEquipo },
];
