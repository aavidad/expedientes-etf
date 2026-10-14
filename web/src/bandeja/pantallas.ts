import { Bandeja } from './bandeja';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/coordinacion', perfiles: ['coordinacion'], clave: 'bandeja.titulo', componente: Bandeja },
];
