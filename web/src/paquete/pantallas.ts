import { AdministracionPaquetes } from '../AdministracionPaquetes';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/admin/paquetes', perfiles: ['administracion'], clave: 'paquetes.titulo', componente: AdministracionPaquetes },
  { ruta: '/admin/idiomas', perfiles: ['administracion'], clave: 'paquetes.idiomas', componente: AdministracionPaquetes },
  { ruta: '/admin/temas', perfiles: ['administracion'], clave: 'paquetes.temas', componente: AdministracionPaquetes },
];
