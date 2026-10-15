import { createElement } from 'react';
import { Restauracion } from './revision';
import type { Pantalla } from '../registro-pantallas';
export const pantallas: readonly Pantalla[] = [
 { ruta: '/admin/restauracion', perfiles: ['administracion'], clave: 'restauracion.titulo', componente: (props) => createElement(Restauracion, props) },
];
