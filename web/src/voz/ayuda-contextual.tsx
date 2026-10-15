import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { Escuchar, manifiesto, type Manifiesto } from './audio';
import './voz.css';

type Destino = { clave: string; elemento: HTMLElement };
const normalizar = (texto: string) => texto.replace(/\s+/g, ' ').trim();

// Amplía los diálogos existentes sin cambiar las pantallas de otros módulos.
// El contenido sólo se compara con el catálogo; nunca se envía ni se sintetiza.
export function AyudasVoz({ idioma, t }: { idioma: string; t: (clave: string) => string }) {
  const [audios, ponerAudios] = useState<Manifiesto>({});
  const [destinos, ponerDestinos] = useState<Destino[]>([]);
  useEffect(() => {
    const controlador = new AbortController();
    ponerAudios({});
    if (idioma === 'es') void manifiesto('ayuda', controlador.signal).then(async datos => {
      const actuales: Manifiesto = {};
      for (const [clave, audio] of Object.entries(datos)) {
        if (!clave.includes('ayuda') || !t(clave)) continue;
        const hash = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(t(clave)));
        const sha = Array.from(new Uint8Array(hash), n => n.toString(16).padStart(2, '0')).join('');
        if (sha === audio.original_sha) actuales[clave] = audio;
      }
      if (!controlador.signal.aborted) ponerAudios(actuales);
    });
    return () => controlador.abort();
  }, [idioma, t]);
  useEffect(() => {
    if (idioma !== 'es' || !Object.keys(audios).length) { ponerDestinos([]); return; }
    const creados = new Map<HTMLElement, Destino>();
    const revisar = () => {
      let cambio = false;
      for (const [texto, destino] of creados) {
        if (!texto.isConnected || !texto.getClientRects().length || normalizar(t(destino.clave)) !== normalizar(texto.textContent ?? '')) {
          destino.elemento.querySelector('audio')?.pause();
          destino.elemento.remove(); creados.delete(texto); cambio = true;
        }
      }
      document.querySelectorAll<HTMLElement>(':is(p,li)').forEach(texto => {
        if (texto.closest('.voz-ayuda, .voz-control') || creados.has(texto) || !texto.getClientRects().length) return;
        const clave = Object.keys(audios).find(k => normalizar(t(k)) === normalizar(texto.textContent ?? ''));
        if (!clave) return;
        const elemento = document.createElement('div');
        elemento.className = 'voz-contextual';
        texto.after(elemento); creados.set(texto, { clave, elemento }); cambio = true;
      });
      if (cambio) ponerDestinos([...creados.values()]);
    };
    const observador = new MutationObserver(revisar);
    observador.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['open', 'hidden', 'aria-expanded'] });
    revisar();
    return () => { observador.disconnect(); creados.forEach(d => { d.elemento.querySelector('audio')?.pause(); d.elemento.remove(); }); };
  }, [audios, idioma, t]);
  return destinos.map(({ clave, elemento }) => createPortal(<Escuchar audio={audios[clave]} grupo="ayuda" t={t} />, elemento));
}
