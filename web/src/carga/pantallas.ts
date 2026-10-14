import { CargaHoja, AltaHistorica } from './pantalla';
import type { Pantalla } from '../registro-pantallas';
export const pantallas: readonly Pantalla[] = [
  { ruta: '/coordinacion/carga-2026', perfiles: ['coordinacion', 'profesional_etf'], clave: 'carga.titulo', componente: CargaHoja },
  { ruta: '/equipo/expedientes/alta-historica', perfiles: ['coordinacion', 'profesional_etf'], clave: 'carga.manual_titulo', componente: AltaHistorica },
];
