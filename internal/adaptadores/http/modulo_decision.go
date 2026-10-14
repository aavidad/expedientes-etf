package http

import (
	"context"
	catalogo "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	documento "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	"net/http"
	"time"
)

type ModuloDecision struct {
	Opciones  catalogo.ConsultaOpcionesDecision
	Emisiones documento.ConsultaEmisionesDecision
}

func (m ModuloDecision) Rutas() []Ruta {
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
			datos, err := m.Opciones.ConsultarOpcionesDecision(ctx, actorCatalogos(r), q.Get("idioma"))
			if err != nil {
				errorCatalogo(w, err)
				return
			}
			responderJSON(w, 200, datos)
		}), lectura("/api/coordinacion/decision/opciones", "coordinacion"))
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
			defer cancelar()
			if m.Emisiones == nil {
				responderError(w, 503, "documento.no_disponible")
				return
			}
			a, err := actorDerivaciones(r)
			if err != nil {
				errorDerivacion(w, err)
				return
			}
			datos, err := m.Emisiones.ConsultarEmisionesDecision(ctx, documento.Actor{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}, r.PathValue("id"))
			if err != nil {
				errorDocumento(w, err)
				return
			}
			responderJSON(w, 200, datos)
		}), lectura("/api/coordinacion/derivaciones/{id}/documentos", "coordinacion"))
	}, pantalla("/coordinacion/lista-espera", "decision.lista_titulo", "coordinacion"), pantalla("/coordinacion/derivaciones/{id}", "", "coordinacion"))
}
