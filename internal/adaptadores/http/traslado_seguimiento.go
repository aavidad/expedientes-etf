package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"time"
)

func registrarTrasladoSeguimiento(mux *muxHTTP, servicio entrada.TrasladoSeguimiento) {
	type operacion struct {
		ruta, accion string
		perfiles     []string
	}
	for _, op := range []operacion{
		{"/api/expedientes/{id}/trasladar", "trasladar", []string{"coordinacion"}},
		{"/api/expedientes/{id}/seguimiento", "iniciar", []string{"coordinacion", "profesional_etf"}},
		{"/api/expedientes/{id}/seguimiento/terminar", "terminar", []string{"coordinacion", "profesional_etf"}},
		{"/api/expedientes/{id}/seguimiento/reabrir", "reabrir", []string{"coordinacion"}},
	} {
		rules := []reglaRuta{escritura(op.ruta, []string{http.MethodPost}, op.perfiles...)}
		if op.accion == "iniciar" {
			rules = append(rules, lectura(op.ruta, op.perfiles...))
		}
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, fin := context.WithTimeout(r.Context(), 25*time.Second)
			defer fin()
			if servicio == nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			r = r.WithContext(ctx)
			a := actorSeguimiento(r)
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				if r.URL.RawQuery != "" {
					errorAceptacion(w, entrada.ErrPeticion)
					return
				}
				resultado, err := servicio.ConsultarSeguimiento(ctx, a, r.PathValue("id"))
				if err != nil {
					errorAceptacion(w, err)
					return
				}
				responderJSON(w, http.StatusOK, resultado)
				return
			}
			rechazar := func(err error) {
				if fallo := servicio.RechazarPropuesta(ctx, a, err.Error()); fallo != nil {
					err = fallo
				}
				errorAceptacion(w, err)
			}
			tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || tipo != "application/json" {
				rechazar(entrada.ErrPeticion)
				return
			}
			if len(r.Header.Values("Idempotency-Key")) != 1 {
				rechazar(entrada.ErrClave)
				return
			}
			var p entrada.PeticionTraslado
			var q entrada.PeticionSeguimiento
			var destino any = &q
			if op.accion == "trasladar" {
				destino = &p
			}
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
			dec.DisallowUnknownFields()
			if err = dec.Decode(destino); err != nil {
				rechazar(entrada.ErrPeticion)
				return
			}
			if err = dec.Decode(new(any)); !errors.Is(err, io.EOF) {
				rechazar(entrada.ErrPeticion)
				return
			}
			var recibo any
			if op.accion == "trasladar" {
				recibo, err = servicio.Trasladar(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), p)
			} else {
				recibo, err = servicio.Seguimiento(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), op.accion, q)
			}
			if err != nil {
				errorAceptacion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, recibo)
		}), rules...)
	}
	for _, ruta := range []string{"/api/avisos"} {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, fin := context.WithTimeout(r.Context(), 25*time.Second)
			defer fin()
			if servicio == nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			if r.URL.RawQuery != "" {
				errorAceptacion(w, entrada.ErrPeticion)
				return
			}
			a := actorSeguimiento(r)
			var resultado any
			var err error
			if ruta == "/api/avisos" {
				resultado, err = servicio.ConsultarAvisosSeguimiento(ctx, a)
			} else {
				resultado, err = servicio.ConsultarSeguimiento(ctx, a, r.PathValue("id"))
			}
			if err != nil {
				errorAceptacion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, resultado)
		}), lectura(ruta, "coordinacion", "profesional_etf"))
	}
}
func actorSeguimiento(r *http.Request) entrada.Actor {
	acceso := identidad.AccesoActual(r.Context())
	a := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
	for _, p := range acceso.Usuario.Perfiles {
		a.Perfiles = append(a.Perfiles, string(p))
	}
	return a
}
