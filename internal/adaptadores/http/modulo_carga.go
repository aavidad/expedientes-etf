package http

import (
	"context"
	catalogo "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	"net/http"
	"time"
)

type ModuloCarga struct {
	Opciones catalogo.ConsultaOpcionesCarga
}

func (m ModuloCarga) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
			defer cancelar()
			if m.Opciones == nil {
				responderError(w, 503, "catalogo.no_disponible")
				return
			}
			q := r.URL.Query()
			if len(q) != 1 || len(q["idioma"]) != 1 {
				responderError(w, 400, "catalogo.edicion_invalida")
				return
			}
			datos, err := m.Opciones.ConsultarOpcionesCarga(ctx, actorCatalogos(r), q.Get("idioma"))
			if err != nil {
				errorCatalogo(w, err)
				return
			}
			responderJSON(w, 200, datos)
		}), lectura("/api/carga-2026/opciones", "coordinacion", "profesional_etf"))
	}, pantalla("/coordinacion/carga-2026", "carga.titulo", "coordinacion", "profesional_etf"), pantalla("/equipo/expedientes/alta-historica", "carga.manual_titulo", "coordinacion", "profesional_etf"))
}
