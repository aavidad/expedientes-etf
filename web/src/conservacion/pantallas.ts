import { Conservacion } from '../conservacion';
import type { Pantalla } from '../registro-pantallas';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/admin/conservacion', perfiles: ['administracion'], clave: 'conservacion.titulo', componente: Conservacion },
];
