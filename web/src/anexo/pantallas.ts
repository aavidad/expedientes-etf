import { NuevoAnexo } from './nuevo';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/coordinacion/anexos/nuevo', perfiles: ['coordinacion'], clave: 'anexo.titulo', componente: NuevoAnexo },
];
