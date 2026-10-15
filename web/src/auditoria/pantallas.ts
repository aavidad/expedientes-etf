import { Auditoria } from './Auditoria';
import type { Pantalla } from '../registro-pantallas';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/admin/auditoria', perfiles: ['administracion'], clave: 'auditoria.titulo', componente: Auditoria },
];
