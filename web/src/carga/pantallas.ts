import { CargaHoja, AltaHistorica } from './pantalla';
import { PlazoCarga, LotesCarga } from './control';
import type { Pantalla } from '../registro-pantallas';
export const pantallas: readonly Pantalla[] = [
 { ruta:'/admin/carga-historica',perfiles:['administracion'],clave:'carga.control_titulo',componente:PlazoCarga },
 { ruta:'/coordinacion/carga-lotes',perfiles:['coordinacion'],clave:'carga.lotes_titulo',componente:LotesCarga },
  { ruta: '/coordinacion/carga-2026', perfiles: ['coordinacion', 'profesional_etf'], clave: 'carga.titulo', componente: CargaHoja },
  { ruta: '/equipo/expedientes/alta-historica', perfiles: ['coordinacion', 'profesional_etf'], clave: 'carga.manual_titulo', componente: AltaHistorica },
];
