import { Ubicaciones } from './ubicaciones';
import type { Pantalla } from '../registro-pantallas';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/admin/ubicaciones', perfiles: ['administracion'], clave: 'ubicaciones.titulo', conservarAlNavegar: true, componente: Ubicaciones },
];
