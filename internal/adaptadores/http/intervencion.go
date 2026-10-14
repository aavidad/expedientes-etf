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

func reglasIntervencion() []reglaRuta {
	perfiles := []string{"profesional_etf", "coordinacion"}
	return []reglaRuta{
		{Patron: "/api/expedientes/{id}/intervencion", Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: perfiles, ExigeAmbito: true},
		{Patron: "/api/expedientes/{id}/intervencion", Metodos: []string{http.MethodPut}, Perfiles: []string{"profesional_etf"}, ExigeAmbito: true, CSRF: true},
		{Patron: "/api/expedientes/{id}/intervencion/historia", Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: perfiles, ExigeAmbito: true},
	}
}
func registrarIntervencion(mux *muxHTTP, s entrada.Intervencion) {
	reglas := reglasIntervencion()
	mux.declarar(handlerIntervencion(s), reglas...)
}
func handlerIntervencion(s entrada.Intervencion) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		r = r.WithContext(ctx)
		if s == nil {
			errorAceptacion(w, entrada.ErrNoDisponible)
			return
		}
		a := actorActividad(r)
		actor := entrada.Actor{UsuarioID: a.UsuarioID, EquipoID: a.EquipoID, Perfiles: a.Perfiles}
		rechazo := func(err error) {
			if e := s.RechazarIntervencion(ctx, actor); e != nil {
				err = e
			}
			errorIntervencion(w, err)
		}
		id := r.PathValue("id")
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			if r.URL.Path == "/api/expedientes/"+id+"/intervencion/historia" {
				q := r.URL.Query()
				if len(q) != 1 || len(q["corte"]) != 1 {
					rechazo(entrada.ErrIntervencion)
					return
				}
				resultado, err := s.ConsultarHistoriaIntervencion(ctx, actor, id, q.Get("corte"))
				if err != nil {
					errorIntervencion(w, err)
					return
				}
				responderJSON(w, http.StatusOK, resultado)
				return
			}
			if len(r.URL.Query()) != 0 {
				rechazo(entrada.ErrPeticion)
				return
			}
			resultado, err := s.ConsultarIntervencion(ctx, actor, id)
			if err != nil {
				errorIntervencion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, resultado)
			return
		}
		claves := r.Header.Values("Idempotency-Key")
		if len(claves) != 1 {
			rechazo(entrada.ErrClave)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			rechazo(entrada.ErrPeticion)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 512<<10)
		if !acotarJSON(r) {
			rechazo(entrada.ErrIntervencion)
			return
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var p entrada.CambioIntervencion
		if err = decoder.Decode(&p); err != nil {
			rechazo(entrada.ErrIntervencion)
			return
		}
		if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			rechazo(entrada.ErrIntervencion)
			return
		}
		if p.Intervencion == nil {
			rechazo(entrada.ErrIntervencion)
			return
		}
		resultado, err := s.EditarIntervencion(ctx, actor, id, claves[0], p)
		if err != nil {
			errorIntervencion(w, err)
			return
		}
		responderJSON(w, http.StatusOK, resultado)
	}
}
func errorIntervencion(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrIntervencion) {
		responderError(w, http.StatusUnprocessableEntity, "expediente.intervencion_invalida")
		return
	}
	errorAceptacion(w, err)
}
