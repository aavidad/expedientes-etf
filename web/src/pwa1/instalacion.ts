// Sólo metadatos públicos de idioma y tema; nunca sesión ni datos de expedientes.
export function actualizarInstalacion(idioma: string, tema: string): (() => void) | undefined {
  if (!tema) return;
  const preferencias = new URLSearchParams({ idioma, tema });
  const manifiesto = document.querySelector<HTMLLinkElement>('link[rel="manifest"]');
  if (manifiesto) manifiesto.href = `/manifest.webmanifest?${preferencias}`;
  if (!window.isSecureContext || !('serviceWorker' in navigator)) return;
  const datos = { tipo: 'preferencias', idioma, tema };
  const enviar = () => navigator.serviceWorker.controller?.postMessage(datos);
  const recibir = (evento: MessageEvent) => {
    if (evento.data?.tipo !== 'solicitar-preferencias') return;
    if (evento.ports[0]) evento.ports[0].postMessage(datos);
    else enviar();
  };
  navigator.serviceWorker.addEventListener('message', recibir);
  navigator.serviceWorker.addEventListener('controllerchange', enviar);
  let vigente = true;
  enviar();
  // La dirección del worker no depende de las preferencias: cambiarlas no instala otro worker,
  // que tardaría en relevar al activo mientras atiende navegaciones de otras ventanas.
  void navigator.serviceWorker.register('/service-worker.js', { scope: '/', updateViaCache: 'none' })
    .then(() => navigator.serviceWorker.ready).then(() => { if (vigente) enviar(); })
    .catch(() => { /* La aplicación sigue disponible si el navegador impide instalar el worker. */ });
  return () => {
    vigente = false;
    navigator.serviceWorker.removeEventListener('message', recibir);
    navigator.serviceWorker.removeEventListener('controllerchange', enviar);
  };
}
