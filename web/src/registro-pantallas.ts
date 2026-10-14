import type { ComponentType } from 'react';
import type { Acceso } from './armazon';

export type PropsPantalla = {
  t: (clave: string) => string;
  idioma: string;
  csrf: string;
  actualizar: () => void;
  alActivar: () => void;
  navegarRuta: (ruta: string) => void;
};
export type Pantalla = {
  ruta: string;
  perfiles: readonly string[];
  clave: string;
  componente: ComponentType<PropsPantalla>;
  menuDe?: string;
  conservarAlNavegar?: boolean;
};

export function validarPantallas(pantallas: readonly Pantalla[]): void {
  if (!pantallas.length) throw new Error('http.registro_invalido');
  const rutas = new Set<string>();
  for (const pantalla of pantallas) {
    if (!pantalla.ruta.startsWith('/') || rutas.has(pantalla.ruta) || !pantalla.perfiles.length || !pantalla.componente) throw new Error('http.registro_invalido');
    rutas.add(pantalla.ruta);
  }
  for (const pantalla of pantallas) {
    if (pantalla.menuDe && !pantallas.some((p) => p.ruta === pantalla.menuDe && p.clave)) throw new Error('http.registro_invalido');
  }
}

export function pantallaPermitida(pantallas: readonly Pantalla[], acceso: Acceso | undefined, ruta: string): Pantalla | undefined {
  if (acceso?.estado !== 'activo') return;
  return pantallas.find((p) => coincideRuta(p.ruta, ruta) && p.perfiles.some((perfil) => acceso.perfiles?.includes(perfil)) && acceso.pantallas?.some((servidor) => servidor.ruta === (p.menuDe ?? p.ruta) && servidor.perfiles.some((perfil) => p.perfiles.includes(perfil) && acceso.perfiles?.includes(perfil))));
}

export function menuPermitido(pantallas: readonly Pantalla[], acceso?: Acceso): Pantalla[] {
  return pantallas.filter((p) => p.clave && !p.menuDe && pantallaPermitida(pantallas, acceso, p.ruta));
}

// Un parámetro ocupa un solo segmento y nunca convierte una ruta ajena en autorizada.
export function coincideRuta(patron: string, ruta: string): boolean {
  const partes = patron.split('/');
  const actuales = ruta.split('/');
  return partes.length === actuales.length && partes.every((parte, i) => /^\{[a-z]+\}$/.test(parte) ? /^[a-zA-Z0-9-]+$/.test(actuales[i]) : parte === actuales[i]);
}
