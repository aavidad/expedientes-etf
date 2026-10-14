import { Actividad } from './Actividad';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/equipo/comisiones', perfiles: ['profesional_etf', 'coordinacion'], clave: 'actividad.titulo', componente: Actividad },
];
