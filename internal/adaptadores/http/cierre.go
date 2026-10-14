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

func registrarCierres(mux *muxHTTP, s entrada.Cierres) {
	for _, accion := range []string{"cerrar", "reabrir"} {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
			defer cancelar()
			if s == nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			r = r.WithContext(ctx)
			a, err := actorDerivaciones(r)
			rechazar := func(err error) {
				if fallo := s.RechazarPropuesta(ctx, a, err.Error()); fallo != nil {
					err = fallo
				}
				errorAceptacion(w, err)
			}
			if err != nil {
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
			var p entrada.PeticionCierre
			var q entrada.PeticionReapertura
			var objetivo any = &p
			if accion == "reabrir" {
				objetivo = &q
			}
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
			dec.DisallowUnknownFields()
			if err := dec.Decode(objetivo); err != nil {
				rechazar(entrada.ErrPeticion)
				return
			}
			if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
				rechazar(entrada.ErrPeticion)
				return
			}
			var recibo entrada.ResultadoCierre
			if accion == "cerrar" {
				recibo, err = s.Cerrar(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), p)
			} else {
				recibo, err = s.Reabrir(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), q)
			}
			if err != nil {
				errorAceptacion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, recibo)
		}), escritura("/api/expedientes/{id}/"+accion, []string{http.MethodPost}, "coordinacion"))
	}
}
