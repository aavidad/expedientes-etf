import { Ayuda } from './pantalla';
import type { Pantalla } from '../registro-pantallas';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/ayuda', perfiles: ['administracion', 'coordinacion', 'profesional_etf', 'consulta'], clave: 'voz.titulo', componente: Ayuda },
];
