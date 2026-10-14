// Sólo una respuesta propia confirmada permite retirar la concesión del vídeo.
export function puedeRestaurar(actual, asignado, objetivo, equipo) {
  return Boolean(asignado && asignado.id === objetivo && actual?.id === objetivo
    && Number.isInteger(asignado.version) && actual.version === asignado.version
    && Array.isArray(actual.perfiles) && actual.perfiles.length === 1
    && actual.perfiles[0].perfil === 'profesional_etf' && actual.equipo?.equipo_id === equipo);
}
