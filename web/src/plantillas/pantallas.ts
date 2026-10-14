import { Plantillas } from './Plantillas';
import type { Pantalla } from '../registro-pantallas';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/admin/plantillas', perfiles: ['administracion'], clave: 'plantillas.titulo', conservarAlNavegar: true, componente: Plantillas },
];
