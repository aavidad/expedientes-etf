package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	"net/http"
	"time"
)

type ModuloAnexo6 struct {
	Opciones entrada.ConsultaDestinosAnexo6
}

func (m ModuloAnexo6) Rutas() []Ruta {
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
			datos, err := m.Opciones.ConsultarDestinosAnexo6(ctx, actorCatalogos(r), q.Get("idioma"))
			if err != nil {
				errorCatalogo(w, err)
				return
			}
			responderJSON(w, 200, datos)
		}), lectura("/api/anexo6/opciones", "profesional_etf", "coordinacion"))
	}, pantalla("/equipo/anexo6", "", "profesional_etf", "coordinacion"))
}
