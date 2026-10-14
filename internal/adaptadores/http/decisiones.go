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

func registrarDecisiones(mux *muxHTTP, servicio entrada.Decisiones) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		r = r.WithContext(ctx)
		if servicio == nil {
			errorAceptacion(w, entrada.ErrNoDisponible)
			return
		}
		rechazar := func(a entrada.Actor, err error) {
			if fallo := servicio.RechazarDecision(ctx, a, err.Error()); fallo != nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			errorDecision(w, err)
		}
		a, err := actorDerivaciones(r)
		if err != nil {
			rechazar(entrada.Actor{}, err)
			return
		}
		if err := a.Autorizar(); err != nil {
			rechazar(a, err)
			return
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			q := r.URL.Query()
			for k, v := range q {
				if (k != "limite" && k != "desplazamiento") || len(v) != 1 {
					rechazar(a, entrada.ErrPeticion)
					return
				}
			}
			p := entrada.FiltroEspera{Limite: 50}
			if q.Has("limite") {
				p.Limite, err = strconv.Atoi(q.Get("limite"))
				if err != nil {
					rechazar(a, entrada.ErrPeticion)
					return
				}
			}
			if q.Has("desplazamiento") {
				p.Desplazamiento, err = strconv.Atoi(q.Get("desplazamiento"))
				if err != nil {
					rechazar(a, entrada.ErrPeticion)
					return
				}
			}
			filas, err := servicio.ListarEspera(ctx, a, p)
			if err != nil {
				errorDecision(w, err)
				return
			}
			responderJSON(w, http.StatusOK, filas)
			return
		}
		claves := r.Header.Values("Idempotency-Key")
		if len(claves) != 1 || !claveDecisionValida(claves[0]) {
			rechazar(a, entrada.ErrClave)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			rechazar(a, entrada.ErrPeticion)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var p entrada.NoAceptacion
		var espera entrada.Espera
		var destino any = &p
		if r.PathValue("decision") == "lista-espera" {
			destino = &espera
		}
		if decoder.Decode(destino) != nil {
			rechazar(a, entrada.ErrPeticion)
			return
		}
		if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			rechazar(a, entrada.ErrPeticion)
			return
		}
		var resultado entrada.ResultadoDecision
		if r.PathValue("decision") == "lista-espera" {
			resultado, err = servicio.DejarEnEspera(ctx, a, r.PathValue("id"), claves[0], espera)
		} else {
			resultado, err = servicio.NoAceptar(ctx, a, r.PathValue("id"), claves[0], p)
		}
		if err != nil {
			errorDecision(w, err)
			return
		}
		w.Header().Set("Location", "/api/coordinacion/derivaciones/"+resultado.ID)
		responderJSON(w, http.StatusOK, resultado)
	})
	// Rutas exactas: una operación desconocida nunca llega al caso de uso.
	for _, decision := range []string{"no-aceptar", "lista-espera"} {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.SetPathValue("decision", decision)
			h.ServeHTTP(w, r)
		}), escritura("/api/coordinacion/derivaciones/{id}/"+decision, []string{http.MethodPost}, "coordinacion"))
	}
	mux.declarar(h, lectura("/api/coordinacion/lista-espera", "coordinacion"))
}

func claveDecisionValida(clave string) bool {
	if len(clave) < 1 || len(clave) > 128 {
		return false
	}
	for _, c := range clave {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func errorDecision(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrPeticion) {
		responderError(w, http.StatusBadRequest, "expediente.decision_peticion_invalida")
		return
	}
	errorAceptacion(w, err)
}
