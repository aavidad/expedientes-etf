import type { Tabla, VistaEmision } from './api';

type Solicitud = { tabla: VistaEmision; idioma: string; clave: string };
// Sólo metadatos de emisión en memoria de esta pestaña, separados por actor.
// Conserva la clave cuando el armazón desmonta la pantalla al cambiar idioma.
const pendientes = new Map<string, Solicitud>();
export const recuperar = (actor: string) => pendientes.get(actor);
export const olvidar = (actor: string) => { pendientes.delete(actor); };
export function preparar(actor: string, tabla: Tabla, idioma: string): Solicitud {
  const existente = recuperar(actor);
  if (existente) return existente;
  const solicitud = {
    tabla: { equipo_id: tabla.equipo_id, anio: tabla.anio, mes: tabla.mes, version_regla: tabla.version_regla,
      filas: (tabla.filas ?? []).map(f => ({ expediente_id: f.expediente_id, version_ficha: f.version_ficha })) },
    idioma, clave: crypto.randomUUID(),
  };
  pendientes.set(actor, solicitud); return solicitud;
}
