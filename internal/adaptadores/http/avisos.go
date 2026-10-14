package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"net/http"
	"time"
)

func registrarAvisos(mux *muxHTTP, s entrada.Avisos) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		if s == nil {
			errorFamilia(w, entrada.ErrNoDisponible)
			return
		}
		a := actorActividad(r)
		actor := entrada.Actor{UsuarioID: a.UsuarioID, EquipoID: a.EquipoID, Perfiles: a.Perfiles}
		if len(r.URL.Query()) != 0 {
			err := entrada.ErrPeticion
			if e := s.RechazarAvisos(ctx, actor); e != nil {
				err = e
			}
			errorFamilia(w, err)
			return
		}
		if r.PathValue("equipo") != "" {
			resultado, err := s.ContarAvisos(ctx, actor, r.PathValue("equipo"))
			if err != nil {
				errorFamilia(w, err)
				return
			}
			responderJSON(w, http.StatusOK, resultado)
			return
		}
		resultado, err := s.ConsultarAvisos(ctx, actor, r.PathValue("id"))
		if err != nil {
			errorFamilia(w, err)
			return
		}
		responderJSON(w, http.StatusOK, resultado)
	})
	mux.declarar(h,
		reglaRuta{Patron: "/api/expedientes/{id}/avisos", Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: []string{"profesional_etf", "coordinacion"}, ExigeAmbito: true},
		reglaRuta{Patron: "/api/equipos/{equipo}/avisos/recuento", Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: []string{"profesional_etf", "coordinacion"}, ExigeAmbito: true})
}
