import type { Pantalla } from '../registro-pantallas';
import { DetalleDecision, ListaEspera } from './pantalla';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/coordinacion/lista-espera', perfiles: ['coordinacion'], clave: 'decision.lista_titulo', componente: ListaEspera },
  { ruta: '/coordinacion/derivaciones/{id}', perfiles: ['coordinacion'], clave: '', menuDe: '/coordinacion/lista-espera', componente: DetalleDecision },
];
