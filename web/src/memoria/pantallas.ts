import type { Pantalla } from '../registro-pantallas';
import { Memoria } from './Memoria';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/memoria', perfiles: ['coordinacion', 'profesional_etf', 'consulta'], clave: 'memoria.titulo', componente: Memoria },
];
