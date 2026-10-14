import type { Pantalla } from '../registro-pantallas';
import { FichaEquipo } from '../equipo/equipo';
import { Traslado } from './traslado';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/coordinacion/expedientes/{id}/traslado', perfiles: ['coordinacion'], clave: '', menuDe: '/coordinacion/cierres', componente: Traslado },
  { ruta: '/equipo/seguimiento', perfiles: ['profesional_etf', 'coordinacion'], clave: '', menuDe: '/equipo', componente: FichaEquipo },
];
