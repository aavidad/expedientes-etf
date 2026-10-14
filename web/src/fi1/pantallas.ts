import { FichaEquipo } from '../equipo/equipo';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/equipo/menores', perfiles: ['profesional_etf', 'coordinacion'], clave: '', menuDe: '/equipo', componente: FichaEquipo },
  { ruta: '/equipo/intervencion', perfiles: ['profesional_etf', 'coordinacion'], clave: '', menuDe: '/equipo', componente: FichaEquipo },
];
