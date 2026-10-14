import { TablaSPAF } from './TablaSPAF';
import { DetalleExpediente } from './DetalleExpediente';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/equipo/tabla-spaf', perfiles: ['profesional_etf', 'coordinacion'], clave: 'spaf.titulo', componente: TablaSPAF },
  { ruta: '/equipo/tabla-spaf/expedientes/{id}', perfiles: ['profesional_etf', 'coordinacion'], clave: '', menuDe: '/equipo/tabla-spaf', componente: DetalleExpediente },
];
