import type { Pantalla } from '../registro-pantallas';
import { CierresPropuestos } from './lista';
import { Salidas } from './salidas';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/coordinacion/cierres', perfiles: ['coordinacion'], clave: 'salidas.cierres', componente: CierresPropuestos },
  { ruta: '/coordinacion/expedientes/{id}/salidas', perfiles: ['coordinacion'], clave: '', menuDe: '/coordinacion/cierres', componente: Salidas },
];
