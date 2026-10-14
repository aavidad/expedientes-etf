// Sólo avisos públicos incorporados por Go. Sin CacheStorage ni datos personales.
const configuracion = __CONFIGURACION__;
const preferencias = new Map();
self.addEventListener('install', event => { event.waitUntil(self.skipWaiting()); });
self.addEventListener('activate', event => {
 event.waitUntil((async () => {
  await self.clients.claim();
  for (const cliente of await self.clients.matchAll()) cliente.postMessage({ tipo: 'solicitar-preferencias' });
 })());
});
self.addEventListener('message', event => {
 const datos = event.data;
 if (datos?.tipo !== 'preferencias' || !event.source?.id) return;
 const clave = `${datos.idioma}/${datos.tema}`;
 if (Object.hasOwn(configuracion.avisos, clave)) preferencias.set(event.source.id, clave);
});
async function preferenciaCliente(event) {
 if (preferencias.has(event.clientId)) return preferencias.get(event.clientId);
 const cliente = await self.clients.get(event.clientId);
 if (!cliente) return configuracion.predeterminado;
 // Tras una suspensión del worker, pregunta a la ventana que sigue abierta.
 return new Promise(resolve => {
  const canal = new MessageChannel();
  const completar = clave => { clearTimeout(limite); canal.port1.close(); resolve(clave); };
  const limite = setTimeout(() => completar(configuracion.predeterminado), 500);
  canal.port1.onmessage = ({ data }) => {
   const clave = `${data?.idioma}/${data?.tema}`;
   completar(Object.hasOwn(configuracion.avisos, clave) ? clave : configuracion.predeterminado);
  };
  cliente.postMessage({ tipo: 'solicitar-preferencias' }, [canal.port2]);
 });
}
self.addEventListener('fetch', event => {
 const u = new URL(event.request.url);
 if (u.origin !== self.location.origin || u.pathname === '/api' || u.pathname.startsWith('/api/') || event.request.method !== 'GET' || event.request.mode !== 'navigate') return;
 event.respondWith(fetch(event.request).catch(async () => {
  const clave = await preferenciaCliente(event);
  if (event.resultingClientId) preferencias.set(event.resultingClientId, clave);
  return new Response(configuracion.avisos[clave], { status: 503, headers: { 'Content-Type': 'text/html; charset=utf-8', 'Cache-Control': 'no-store', 'Content-Security-Policy': configuracion.politica } });
 }));
});
