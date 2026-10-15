// Reproducción local y voluntaria; ninguna petición sale del curso.
(() => {
  const pararOtros = audio => document.querySelectorAll('audio,video').forEach(otro => { if (otro !== audio) otro.pause(); });
  document.querySelectorAll('[data-voz-pagina],[data-voz-seccion]').forEach(boton => {
    const audio = boton.nextElementSibling;
    const lista = boton.dataset.vozPagina ? JSON.parse(boton.dataset.vozPagina) : [boton.dataset.vozSeccion];
    let indice = 0;
    const iniciar = () => {
      audio.hidden = false;
      audio.src = lista[indice];
      void audio.play().catch(() => { audio.hidden = true; boton.hidden = true; });
    };
    boton.addEventListener('click', () => {
      if (!audio.paused) { audio.pause(); return; }
      if (audio.src && !audio.ended) void audio.play().catch(() => { boton.hidden = true; });
      else { indice = 0; iniciar(); }
    });
    audio.addEventListener('play', () => {
      pararOtros(audio);
      if (boton.dataset.vozPagina) boton.textContent = boton.dataset.vozPausar;
    });
    audio.addEventListener('pause', () => { if (boton.dataset.vozPagina) boton.textContent = boton.dataset.vozEscuchar; });
    audio.addEventListener('ended', () => {
      indice += 1;
      if (indice < lista.length) iniciar();
      else if (boton.dataset.vozPagina) boton.textContent = boton.dataset.vozEscuchar;
    });
    audio.addEventListener('error', () => { audio.hidden = true; boton.hidden = true; });
  });
})();
