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
	"strconv"
	"time"
)

func registrarPropuestasCierre(mux *muxHTTP, s entrada.PropuestasCierre) {
	for _, operacion := range []string{"finalizacion", "proponer-cierre", "retirar-propuesta-cierre"} {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
			defer cancelar()
			if s == nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			acceso := identidad.AccesoActual(ctx)
			a := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
			for _, p := range acceso.Usuario.Perfiles {
				a.Perfiles = append(a.Perfiles, string(p))
			}
			rechazar := func(err error) {
				if fallo := s.RechazarPropuesta(ctx, a, err.Error()); fallo != nil {
					errorAceptacion(w, fallo)
				} else {
					errorAceptacion(w, err)
				}
			}
			if operacion == "finalizacion" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				final, err := s.ConsultarFinalizacion(ctx, a, r.PathValue("id"))
				if err != nil {
					errorAceptacion(w, err)
					return
				}
				responderJSON(w, http.StatusOK, final)
				return
			}
			if err := a.AutorizarPropuesta(); err != nil {
				rechazar(err)
				return
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
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 100<<10))
			dec.DisallowUnknownFields()
			var p entrada.Finalizacion
			var v entrada.VersionCierre
			var objetivo any = &v
			if operacion == "finalizacion" {
				objetivo = &p
			}
			if err := dec.Decode(objetivo); err != nil {
				rechazar(entrada.ErrPeticion)
				return
			}
			if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
				rechazar(entrada.ErrPeticion)
				return
			}
			var recibo entrada.PropuestaCierre
			switch operacion {
			case "finalizacion":
				recibo, err = s.GuardarFinalizacion(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), p)
			case "proponer-cierre":
				recibo, err = s.ProponerCierre(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), v)
			default:
				recibo, err = s.RetirarPropuestaCierre(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), v)
			}
			if err != nil {
				errorAceptacion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, recibo)
		})
		patron := "/api/expedientes/{id}/" + operacion
		metodo := http.MethodPost
		if operacion == "finalizacion" {
			metodo = http.MethodPatch
		}
		regla := escritura(patron, []string{metodo}, "profesional_etf")
		regla.ExigeAmbito = true
		if operacion == "finalizacion" {
			consulta := lectura(patron, "coordinacion", "profesional_etf")
			consulta.ExigeAmbito = true
			mux.declarar(handler, consulta, regla)
		} else {
			mux.declarar(handler, regla)
		}
	}
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s == nil {
			errorAceptacion(w, entrada.ErrNoDisponible)
			return
		}
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		a, err := actorDerivaciones(r)
		if err != nil {
			if s.RechazarPropuesta(ctx, a, err.Error()) != nil {
				err = entrada.ErrNoDisponible
			}
			errorAceptacion(w, err)
			return
		}
		l, o := 100, 0
		for k, destino := range map[string]*int{"limite": &l, "desplazamiento": &o} {
			if texto := r.URL.Query().Get(k); texto != "" {
				*destino, err = strconv.Atoi(texto)
				if err != nil {
					if s.RechazarPropuesta(ctx, a, entrada.ErrPeticion.Error()) != nil {
						errorAceptacion(w, entrada.ErrNoDisponible)
					} else {
						errorAceptacion(w, entrada.ErrPeticion)
					}
					return
				}
			}
		}
		filas, err := s.ListarPropuestasCierre(ctx, a, l, o)
		if err != nil {
			errorAceptacion(w, err)
			return
		}
		responderJSON(w, http.StatusOK, filas)
	}), lectura("/api/coordinacion/cierres", "coordinacion"))
}
