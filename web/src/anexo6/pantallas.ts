import { createElement } from 'react';
import type { Pantalla, PropsPantalla } from '../registro-pantallas';
import { Anexo6, AccesoAnexo6 } from './anexo6';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/equipo/anexo6', perfiles: ['profesional_etf', 'coordinacion'], clave: '', menuDe: '/equipo', componente: Anexo6 },
];

// Conserva los registros INT5 y añade la navegación desde las pantallas consumidoras.
export function enlazarAnexo6(pantallas: readonly Pantalla[]): Pantalla[] {
  return pantallas.map(p => !['/equipo/expediente', '/coordinacion/expedientes/{id}/salidas'].includes(p.ruta) ? p : {
    ...p, componente: (props: PropsPantalla) => createElement(AccesoAnexo6, { ...props, componente: p.componente }),
  });
}
