export type Referencia = { tipo: string; codigo: string; version_contenido: string; huella: string; version: number };
export type Preparacion = Omit<Referencia, 'version'> & { ticket: string };
export type Estado = 'inicio' | 'consultando' | 'subiendo' | 'antivirus' | 'validado' | 'activando' | 'activado' | 'error';

export function claveError(datos: unknown): string {
  if (typeof datos === 'object' && datos !== null && 'error' in datos && typeof datos.error === 'object' && datos.error !== null && 'clave' in datos.error && typeof datos.error.clave === 'string' && /^paquete\.[a-z_]+$/.test(datos.error.clave)) return datos.error.clave;
  return 'paquete.no_disponible';
}

export async function consultar(tipo: string, codigo: string, signal: AbortSignal): Promise<Referencia> {
  const respuesta = await fetch(`/api/admin/paquetes/${encodeURIComponent(tipo)}/${encodeURIComponent(codigo)}`, { signal });
  const datos: unknown = await respuesta.json();
  if (!respuesta.ok) throw new Error(claveError(datos));
  if (!referenciaValida(datos)) throw new Error('paquete.no_disponible');
  return datos;
}

export function referenciaValida(datos: unknown): datos is Referencia {
  return typeof datos === 'object' && datos !== null && 'version' in datos && Number.isSafeInteger(datos.version) && Number(datos.version) >= 0
    && 'codigo' in datos && typeof datos.codigo === 'string' && 'tipo' in datos && typeof datos.tipo === 'string'
    && 'version_contenido' in datos && typeof datos.version_contenido === 'string' && 'huella' in datos && typeof datos.huella === 'string';
}

export function subir(tipo: string, codigo: string, archivo: File, signal: AbortSignal, estado: (valor: Estado) => void, csrf: string): Promise<Preparacion> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    const abortar = () => xhr.abort();
    const terminar = () => signal.removeEventListener('abort', abortar);
    signal.addEventListener('abort', abortar, { once: true });
    xhr.open('POST', `/api/admin/paquetes/${encodeURIComponent(tipo)}/${encodeURIComponent(codigo)}/revisar`);
    xhr.setRequestHeader('Content-Type', 'application/zip');
    xhr.setRequestHeader('X-ETF-Paquetes', '1');
    xhr.setRequestHeader('X-CSRF-Token', csrf);
    xhr.responseType = 'json';
    xhr.timeout = 90000;
    xhr.upload.onload = () => { if (!signal.aborted) estado('antivirus'); };
    xhr.onload = () => {
      terminar();
      const datos: unknown = xhr.response;
      if (xhr.status !== 201) { reject(new Error(claveError(datos))); return; }
      if (typeof datos !== 'object' || datos === null || !('ticket' in datos) || typeof datos.ticket !== 'string' || !/^[a-f0-9]{64}$/.test(datos.ticket)
        || !('version_contenido' in datos) || typeof datos.version_contenido !== 'string' || !('codigo' in datos) || datos.codigo !== codigo || !('tipo' in datos) || datos.tipo !== tipo
        || !('huella' in datos) || typeof datos.huella !== 'string') { reject(new Error('paquete.no_disponible')); return; }
      resolve(datos as Preparacion);
    };
    xhr.onerror = xhr.ontimeout = () => { terminar(); reject(new Error('paquete.no_disponible')); };
    xhr.onabort = () => { terminar(); reject(new DOMException('', 'AbortError')); };
    if (signal.aborted) { terminar(); reject(new DOMException('', 'AbortError')); return; }
    estado('subiendo');
    xhr.send(archivo);
  });
}

export async function activar(preparacion: Preparacion, version: number, signal: AbortSignal, csrf: string): Promise<Referencia> {
  const respuesta = await fetch('/api/admin/paquetes/activar', {
    method: 'POST', signal, headers: { 'Content-Type': 'application/json', 'X-ETF-Paquetes': '1', 'X-CSRF-Token': csrf },
    body: JSON.stringify({ ticket: preparacion.ticket, version }),
  });
  const datos: unknown = await respuesta.json();
  if (!respuesta.ok) throw new Error(claveError(datos));
  if (!referenciaValida(datos) || datos.codigo !== preparacion.codigo || datos.tipo !== preparacion.tipo || datos.version_contenido !== preparacion.version_contenido || datos.huella !== preparacion.huella || datos.version !== version + 1) throw new Error('paquete.no_disponible');
  return datos;
}

export function descartar(ticket: string, csrf: string) {
  void fetch(`/api/admin/paquetes/preparaciones/${encodeURIComponent(ticket)}`, { method: 'DELETE', headers: { 'X-ETF-Paquetes': '1', 'X-CSRF-Token': csrf }, keepalive: true }).catch(() => undefined);
}
