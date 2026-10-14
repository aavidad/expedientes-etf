import { AdministracionSMI } from '../smi';
import type { Pantalla } from '../registro-pantallas';

export const pantallas: readonly Pantalla[] = [
  { ruta: '/coordinacion/smi', perfiles: ['coordinacion'], clave: 'smi.titulo', conservarAlNavegar: true, componente: AdministracionSMI },
];
