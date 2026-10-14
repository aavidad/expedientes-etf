import { Administracion } from './administracion';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/admin/catalogos', perfiles: ['administracion'], clave: 'catalogo.menu_listas', componente: Administracion },
  { ruta: '/admin/catalogos/listas', perfiles: ['administracion'], clave: '', menuDe: '/admin/catalogos', componente: Administracion },
  { ruta: '/admin/reglas', perfiles: ['administracion'], clave: 'catalogo.menu_plazos', componente: Administracion },
  { ruta: '/admin/firmantes', perfiles: ['administracion'], clave: 'catalogo.menu_firmantes', componente: Administracion },
];
