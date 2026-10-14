package http

import (
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"io"
	"mime"
	"net/http"
)

// ModuloAnexo registra la pantalla; el módulo expediente conserva las API.
type ModuloAnexo struct{}

func (ModuloAnexo) Rutas() []Ruta {
	return []Ruta{pantalla("/coordinacion/anexos/nuevo", "anexo.titulo", "coordinacion")}
}

func registrarConfirmacion(mux *muxHTTP, c ConfiguracionDerivaciones) {
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			errorDerivacion(w, entrada.ErrNoDisponible)
			return
		}
		actor, err := actorDerivaciones(r)
		if err != nil {
			errorDerivacion(w, err)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		claves := r.Header.Values("Idempotency-Key")
		if err != nil || tipo != "application/json" || len(claves) != 1 {
			rechazarConfirmacion(w, r, c, actor, entrada.ErrPeticion)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if !acotarJSON(r) {
			rechazarConfirmacion(w, r, c, actor, entrada.ErrPeticion)
			return
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var p entrada.Confirmacion
		if decoder.Decode(&p) != nil || decoder.Decode(new(any)) != io.EOF {
			rechazarConfirmacion(w, r, c, actor, entrada.ErrPeticion)
			return
		}
		recibo, err := c.Servicio.ConfirmarDerivacion(r.Context(), actor, claves[0], p)
		if err != nil {
			errorConfirmacion(w, err)
			return
		}
		w.Header().Set("Location", "/api/coordinacion/derivaciones/"+recibo.ID)
		responderJSON(w, http.StatusCreated, recibo)
	}), escritura("/api/coordinacion/derivaciones/confirmar", []string{http.MethodPost}, "coordinacion"))
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			errorDerivacion(w, entrada.ErrNoDisponible)
			return
		}
		actor, err := actorDerivaciones(r)
		if err != nil {
			errorDerivacion(w, err)
			return
		}
		if err := c.Servicio.Descartar(r.Context(), actor, r.PathValue("id")); err != nil {
			errorDerivacion(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}), escritura("/api/coordinacion/anexos/lecturas/{id}", []string{http.MethodDelete}, "coordinacion"))
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			errorDerivacion(w, entrada.ErrNoDisponible)
			return
		}
		actor, err := actorDerivaciones(r)
		if err != nil {
			errorDerivacion(w, err)
			return
		}
		etiquetas, err := c.Servicio.Etiquetas(r.Context(), actor, r.URL.Query().Get("idioma"))
		if err != nil {
			errorDerivacion(w, err)
			return
		}
		responderJSON(w, http.StatusOK, etiquetas)
	}), lectura("/api/coordinacion/anexos/etiquetas", "coordinacion"))
}
func rechazarConfirmacion(w http.ResponseWriter, r *http.Request, c ConfiguracionDerivaciones, actor entrada.Actor, err error) {
	if fallo := c.Servicio.Rechazar(r.Context(), actor, err.Error()); fallo != nil {
		errorDerivacion(w, fallo)
		return
	}
	errorDerivacion(w, err)
}

// La revisión previa no solicita datos exclusivos del Acta de Aceptación.
func errorConfirmacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entrada.ErrObligatorios):
		responderError(w, http.StatusUnprocessableEntity, "anexo.ubicacion_invalida")
	case errors.Is(err, entrada.ErrPeticion):
		responderError(w, http.StatusBadRequest, "anexo.confirmacion_invalida")
	default:
		errorAceptacion(w, err)
	}
}
