import { useEffect, useRef, useState } from 'react';

export type AudioEditorial = { fichero: string; duracion: number; sha: string; original_sha?: string };
export type Manifiesto = Record<string, AudioEditorial>;
export const valido = (audio: AudioEditorial | undefined): audio is AudioEditorial => !!audio && /^[a-f0-9]{64}\.mp3$/.test(audio.fichero);
export async function manifiesto(grupo: string, signal: AbortSignal): Promise<Manifiesto> {
  try {
    const r = await fetch(`/voz/${grupo}/manifiesto.json`, { signal });
    if (!r.ok) return {};
    const datos = await r.json() as Manifiesto;
    return Object.fromEntries(Object.entries(datos).filter(([, audio]) => valido(audio)));
  } catch { return {}; }
}

export function Escuchar({ audio, grupo, t, titulo, seccion = false }: {
  audio?: AudioEditorial; grupo: string; t: (clave: string) => string; titulo?: string; seccion?: boolean;
}) {
  const ref = useRef<HTMLAudioElement>(null);
  const [abierto, abrir] = useState(false);
  const [ausente, ocultar] = useState(false);
  useEffect(() => {
    const reproductor = ref.current;
    return () => reproductor?.pause();
  }, [audio?.fichero]);
  const etiqueta = titulo ? t('voz.escuchar_seccion_aria').replace('{titulo}', titulo) : t('voz.escuchar_aria');
  const reproductor = t('voz.reproductor');
  if (!valido(audio) || ausente) return null;
  return <div className="voz-control">
    {!abierto && <button type="button" aria-label={etiqueta} onClick={() => {
      abrir(true);
      if (ref.current) void ref.current.play().catch(() => ocultar(true));
    }}>{t(seccion ? 'voz.escuchar_seccion' : 'voz.escuchar')}</button>}
    <audio ref={ref} hidden={!abierto} controls preload="none" src={`/voz/${grupo}/${audio.fichero}`} aria-label={reproductor} onError={() => ocultar(true)} onPlay={(e) => {
      document.querySelectorAll('audio').forEach(otro => { if (otro !== e.currentTarget) otro.pause(); });
    }} />
  </div>;
}
