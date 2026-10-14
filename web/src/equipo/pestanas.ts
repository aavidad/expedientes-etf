import type { ComponentType } from 'react';
import type { PropsPantalla } from '../registro-pantallas';
import type { Ficha } from './api';
import { ConsentimientoEquipo } from './consentimiento';
import { EvaluacionEquipo } from './evaluacion';

export type PropsPestanaFicha = PropsPantalla & { id: string; ficha: Ficha; alCambiar: () => void; refrescando: boolean };
export type PestanaFicha = { codigo: string; clave: string; componente: ComponentType<PropsPestanaFicha> };

export const pestanas: readonly PestanaFicha[] = [
  { codigo: 'consentimiento', clave: 'equipo.pestana.consentimiento', componente: ConsentimientoEquipo },
  { codigo: 'evaluacion', clave: 'equipo.pestana.evaluacion', componente: EvaluacionEquipo },
];
