package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"
)

func reglasFamilia() []reglaRuta {
	perfiles := []string{"profesional_etf", "coordinacion"}
	return []reglaRuta{
		{Patron: "/api/expedientes/{id}", Metodos: []string{http.MethodPatch}, Perfiles: []string{"profesional_etf"}, ExigeAmbito: true, CSRF: true},
		{Patron: "/api/expedientes/{id}/familia", Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: perfiles, ExigeAmbito: true},
		{Patron: "/api/expedientes/{id}/familia/historia", Metodos: []string{http.MethodGet, http.MethodHead}, Perfiles: perfiles, ExigeAmbito: true},
		{Patron: "/api/expedientes/{id}/adultos", Metodos: []string{http.MethodPut}, Perfiles: []string{"profesional_etf"}, ExigeAmbito: true, CSRF: true},
	}
}
func registrarFamilia(mux *muxHTTP, s entrada.Familias) {
	reglas := reglasFamilia()
	mux.declarar(handlerFamilia(s), reglas[1:]...)
}
func handlerFamilia(s entrada.Familias) http.HandlerFunc {
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
			if e := s.RechazarFamilia(ctx, actor); e != nil {
				err = e
			}
			errorFamilia(w, err)
		}
		id := r.PathValue("id")
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			if r.URL.Path == "/api/expedientes/"+id+"/familia/historia" {
				q := r.URL.Query()
				if len(q) != 2 || len(q["corte"]) != 1 || len(q["anio"]) != 1 {
					rechazo(entrada.ErrFamilia)
					return
				}
				anio, err := strconv.Atoi(q.Get("anio"))
				if err != nil {
					rechazo(entrada.ErrFamilia)
					return
				}
				resultado, err := s.ConsultarHistoriaFamilia(ctx, actor, id, q.Get("corte"), anio)
				if err != nil {
					errorFamilia(w, err)
					return
				}
				responderJSON(w, http.StatusOK, resultado)
				return
			}
			if len(r.URL.Query()) != 0 {
				rechazo(entrada.ErrPeticion)
				return
			}
			resultado, err := s.ConsultarFamilia(ctx, actor, id)
			if err != nil {
				errorFamilia(w, err)
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
			rechazo(entrada.ErrFamilia)
			return
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var p entrada.CambioFamilia
		if err = decoder.Decode(&p); err != nil {
			rechazo(entrada.ErrFamilia)
			return
		}
		if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			rechazo(entrada.ErrFamilia)
			return
		}
		if r.Method == http.MethodPatch && (p.Familia == nil || p.Adultos != nil) || r.Method == http.MethodPut && (p.Adultos == nil || p.Familia != nil) {
			rechazo(entrada.ErrFamilia)
			return
		}
		resultado, err := s.EditarFamilia(ctx, actor, id, claves[0], p)
		if err != nil {
			errorFamilia(w, err)
			return
		}
		responderJSON(w, http.StatusOK, resultado)
	}
}
func errorFamilia(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrFamilia) {
		responderError(w, http.StatusUnprocessableEntity, "expediente.familia_invalida")
		return
	}
	errorAceptacion(w, err)
}
