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

func registrarSPM(mux *muxHTTP, s entrada.SalidasSPM) {
	handler := func(accion string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
			defer cancelar()
			r = r.WithContext(ctx)
			if s == nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			a, err := actorSalidaSPM(r)
			rechazar := func(err error) {
				if s.RechazarSPM(ctx, a, err.Error()) != nil {
					err = entrada.ErrNoDisponible
				}
				errorSPM(w, err)
			}
			if err != nil {
				rechazar(err)
				return
			}
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				salida, err := s.ConsultarSalidaSPM(ctx, a, r.PathValue("id"))
				if err != nil {
					errorSPM(w, err)
					return
				}
				responderJSON(w, http.StatusOK, salida)
				return
			}
			if err := a.AutorizarSPM(); err != nil {
				rechazar(err)
				return
			}
			tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || tipo != "application/json" || len(r.Header.Values("Idempotency-Key")) != 1 {
				rechazar(entrada.ErrSPM)
				return
			}
			var p entrada.PeticionSPM
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
			dec.DisallowUnknownFields()
			if dec.Decode(&p) != nil {
				rechazar(entrada.ErrSPM)
				return
			}
			if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
				rechazar(entrada.ErrSPM)
				return
			}
			resultado, err := s.OperarSPM(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), accion, p)
			if err != nil {
				errorSPM(w, err)
				return
			}
			responderJSON(w, http.StatusOK, resultado)
		}
	}
	lecturaSPM := lectura("/api/expedientes/{id}/salida-spm", "coordinacion", "profesional_etf")
	lecturaSPM.ExigeAmbito = true
	crear := escritura("/api/expedientes/{id}/salida-spm", []string{http.MethodPost}, "coordinacion", "profesional_etf")
	crear.ExigeAmbito = true
	fecha := escritura("/api/expedientes/{id}/salida-spm/fecha", []string{http.MethodPost}, "coordinacion", "profesional_etf")
	fecha.ExigeAmbito = true
	respuesta := escritura("/api/expedientes/{id}/salida-spm/respuesta", []string{http.MethodPatch}, "coordinacion", "profesional_etf")
	respuesta.ExigeAmbito = true
	mux.declarar(handler("crear"), lecturaSPM, crear)
	mux.declarar(handler("confirmar"), fecha)
	mux.declarar(handler("responder"), respuesta)
	mux.declarar(handler("forzar"), escritura("/api/expedientes/{id}/salida-spm/forzado", []string{http.MethodPost}, "coordinacion"))
}
func errorSPM(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrSPM) {
		responderError(w, http.StatusUnprocessableEntity, entrada.ErrSPM.Error())
		return
	}
	errorAceptacion(w, err)
}

func actorSalidaSPM(r *http.Request) (entrada.Actor, error) {
	acceso := identidad.AccesoActual(r.Context())
	if !acceso.Permite("coordinacion") && !acceso.Permite("profesional_etf") {
		return entrada.Actor{}, entrada.ErrAcceso
	}
	a := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
	for _, p := range acceso.Usuario.Perfiles {
		a.Perfiles = append(a.Perfiles, string(p))
	}
	return a, a.AutorizarSPM()
}
