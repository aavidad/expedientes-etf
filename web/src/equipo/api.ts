import { useEffect, useRef, useState } from 'react';

export type Resumen = {
  id: string; numero: string; familia: string; estado: string; subprograma: string; fase: string;
  fecha_recepcion: string; fecha_aceptacion: string; menores: number; version: number; avisos: string[];
};
export type Suceso = { id: string; tipo: string; codigo: string; fecha: string; autor: string; version_plantilla?: string; resultado?: string };
export type Ficha = Resumen & { hitos: Suceso[]; emisiones: Suceso[]; historial: Suceso[]; salidas_servicio: { id: string; servicio_destino: string; motivo: string; estado: string; fecha: string }[]; capacidades: string[]; puede_proponer: boolean; motivos_cierre: { codigo: string; admite_texto: boolean; sin_consentimiento: boolean }[] };

export async function consultar<T>(ruta: string, signal: AbortSignal): Promise<T> {
  const respuesta = await fetch(ruta, { signal, credentials: 'same-origin', cache: 'no-store' });
  const datos = await respuesta.json() as T & { error?: { clave?: string } };
  if (!respuesta.ok) throw new Error(datos.error?.clave ?? 'equipo.error');
  return datos;
}

export function useLectura<T>(ruta: string, conservarAlRecargar = false) {
  const [datos, ponerDatos] = useState<T>();
  const [error, ponerError] = useState('');
  const [cargando, ponerCargando] = useState(true);
  const [intento, ponerIntento] = useState(0);
  const ultimaRuta = useRef(ruta);
  useEffect(() => {
    const abort = new AbortController();
    if (!conservarAlRecargar || ultimaRuta.current !== ruta) ponerDatos(undefined);
    ultimaRuta.current = ruta; ponerError(''); ponerCargando(true);
    void consultar<T>(ruta, abort.signal).then((d) => { if (!abort.signal.aborted) ponerDatos(d); })
      .catch((e: unknown) => { if (!abort.signal.aborted) { ponerDatos(undefined); ponerError(e instanceof Error ? e.message : 'equipo.error'); } })
      .finally(() => { if (!abort.signal.aborted) ponerCargando(false); });
    return () => abort.abort();
  }, [ruta, intento, conservarAlRecargar]);
  return { datos, error, cargando, recargar: () => ponerIntento((n) => n + 1) };
}

export function fecha(valor: string, idioma: string, hora = false): string {
  return new Intl.DateTimeFormat(idioma, { day: '2-digit', month: '2-digit', year: 'numeric', timeZone: valor.length === 10 ? 'UTC' : 'Europe/Madrid', ...(hora ? { hour: '2-digit', minute: '2-digit', hour12: false } : {}) }).format(new Date(valor));
}

export function useEtiquetas(idioma: string) { return useLectura<Record<string, string>>(`/api/equipo/etiquetas/${encodeURIComponent(idioma)}`); }
