import { Menores } from './menores';
import { Intervencion } from './intervencion';
import './fi1.css';

export const pestanas = [
  { codigo: 'menores', componente: Menores },
  { codigo: 'intervencion', componente: Intervencion },
] as const;
