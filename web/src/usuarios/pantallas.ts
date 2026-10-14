import { createElement } from 'react';
import { Usuarios } from '../usuarios';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/admin/usuarios', perfiles: ['administracion'], clave: 'usuarios.titulo', conservarAlNavegar: true, componente: ({ t, csrf, actualizar }) => createElement(Usuarios, { t, csrf, actualizar }) },
];
