import { useEffect, useState } from 'react';
import type { Acceso } from '../armazon';

export function seccionCoordinacion(ruta: string): string {
  if (ruta === '/coordinacion' || ruta.includes('/anexos') || ruta.includes('/derivaciones') || ruta.includes('/lista-espera')) return 'derivaciones';
  if (ruta.includes('/comisiones') || ruta.includes('/tabla-spaf') || ruta.includes('/memoria')) return 'informes';
  if (ruta.includes('/smi') || ruta.startsWith('/admin/')) return 'configuracion';
  return 'expedientes';
}

export function useEquipoPersona(acceso: Acceso | undefined, idioma: string) {
  const [nombre, ponerNombre] = useState('');
  const [error, ponerError] = useState(false);
  const [intento, ponerIntento] = useState(0);
  useEffect(() => {
    const controlador = new AbortController();
    ponerNombre(''); ponerError(false);
    if (acceso?.estado !== 'activo' || !acceso.equipo_id || !acceso.perfiles?.some((p) => p === 'profesional_etf' || p === 'coordinacion')) return () => controlador.abort();
    async function leer() {
      const respuestas = await Promise.all([
        fetch('/api/equipo/expedientes?limite=1', { signal: controlador.signal, cache: 'no-store' }),
        fetch(`/api/equipo/etiquetas/${encodeURIComponent(idioma)}`, { signal: controlador.signal, cache: 'no-store' }),
      ]);
      if (respuestas.some((r) => !r.ok)) throw new Error();
      const [equipo, etiquetas] = await Promise.all(respuestas.map((r) => r.json())) as [{ equipo_clave: string }, Record<string, string>];
      const etiqueta = etiquetas[equipo.equipo_clave];
      if (!etiqueta) throw new Error();
      if (!controlador.signal.aborted) ponerNombre(etiqueta);
    }
    void leer().catch(() => { if (!controlador.signal.aborted) ponerError(true); });
    return () => controlador.abort();
  }, [acceso?.usuario?.id, acceso?.equipo_id, acceso?.estado, idioma, intento]);
  return { nombre, error, recargar: () => ponerIntento((n) => n + 1) };
}
