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

func reglasMenores() []reglaRuta {
	perfiles := []string{"profesional_etf", "coordinacion"}
	return []reglaRuta{
		{Patron: "/api/expedientes/{id}/menores", Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: perfiles, ExigeAmbito: true},
		{Patron: "/api/expedientes/{id}/menores", Metodos: []string{http.MethodPut}, Perfiles: []string{"profesional_etf"}, ExigeAmbito: true, CSRF: true},
		{Patron: "/api/expedientes/{id}/menores/historia", Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: perfiles, ExigeAmbito: true},
	}
}
func registrarMenores(mux *muxHTTP, s entrada.Menores) {
	reglas := reglasMenores()
	mux.declarar(handlerMenores(s), reglas...)
}
func handlerMenores(s entrada.Menores) http.HandlerFunc {
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
			if e := s.RechazarMenores(ctx, actor); e != nil {
				err = e
			}
			errorMenores(w, err)
		}
		id := r.PathValue("id")
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			if r.URL.Path == "/api/expedientes/"+id+"/menores/historia" {
				q := r.URL.Query()
				if len(q) != 1 || len(q["corte"]) != 1 {
					rechazo(entrada.ErrMenores)
					return
				}
				resultado, err := s.ConsultarHistoriaMenores(ctx, actor, id, q.Get("corte"))
				if err != nil {
					errorMenores(w, err)
					return
				}
				responderJSON(w, http.StatusOK, resultado)
				return
			}
			if len(r.URL.Query()) != 0 {
				rechazo(entrada.ErrPeticion)
				return
			}
			resultado, err := s.ConsultarMenores(ctx, actor, id)
			if err != nil {
				errorMenores(w, err)
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
			rechazo(entrada.ErrMenores)
			return
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var p entrada.CambioMenores
		if err = decoder.Decode(&p); err != nil {
			rechazo(entrada.ErrMenores)
			return
		}
		if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			rechazo(entrada.ErrMenores)
			return
		}
		if p.Menores == nil {
			rechazo(entrada.ErrMenores)
			return
		}
		resultado, err := s.EditarMenores(ctx, actor, id, claves[0], p)
		if err != nil {
			errorMenores(w, err)
			return
		}
		responderJSON(w, http.StatusOK, resultado)
	}
}
func errorMenores(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrMenores) {
		responderError(w, http.StatusUnprocessableEntity, "expediente.menores_invalidos")
		return
	}
	errorAceptacion(w, err)
}
