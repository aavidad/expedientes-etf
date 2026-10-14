package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"time"
)

type ModuloConservacion struct{ Servicio entrada.Conservaciones }

func (m ModuloConservacion) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
			defer cancelar()
			if m.Servicio == nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			if r.URL.RawQuery != "" {
				errorAceptacion(w, entrada.ErrPeticion)
				return
			}
			datos, err := m.Servicio.PendientesConservacion(ctx, actorSeguimiento(r))
			if err != nil {
				errorConservacionHTTP(w, err)
				return
			}
			responderJSON(w, 200, datos)
		}), lectura("/api/conservacion/pendientes", "administracion"))
		for _, accion := range []string{"proponer", "anonimizar"} {
			mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
				defer cancelar()
				if m.Servicio == nil {
					errorAceptacion(w, entrada.ErrNoDisponible)
					return
				}
				tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil || tipo != "application/json" || r.URL.RawQuery != "" {
					errorAceptacion(w, entrada.ErrPeticion)
					return
				}
				var p struct {
					Version int `json:"version"`
				}
				dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
				dec.DisallowUnknownFields()
				if dec.Decode(&p) != nil || !errors.Is(dec.Decode(new(any)), io.EOF) {
					errorAceptacion(w, entrada.ErrPeticion)
					return
				}
				datos, err := m.Servicio.ConfirmarConservacion(ctx, actorSeguimiento(r), r.PathValue("id"), p.Version, accion == "anonimizar")
				if err != nil {
					errorConservacionHTTP(w, err)
					return
				}
				responderJSON(w, 200, datos)
			}), escritura("/api/conservacion/{id}/"+accion, []string{http.MethodPost}, "administracion"))
		}
	}, pantalla("/admin/conservacion", "conservacion.titulo", "administracion")), "/admin/conservacion")
}
func errorConservacionHTTP(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrConservacion) {
		responderError(w, 409, "conservacion.no_permitida")
		return
	}
	errorAceptacion(w, err)
}
