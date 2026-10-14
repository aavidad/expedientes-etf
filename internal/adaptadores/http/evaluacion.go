package http

import (
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/evaluacion/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
)

type ModuloEvaluacion struct{ Servicio entrada.Gestor }

func (m ModuloEvaluacion) Rutas() []Ruta {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.Servicio == nil {
			responderError(w, 503, "evaluacion.no_disponible")
			return
		}
		acceso := identidad.AccesoActual(r.Context())
		a := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
		for _, p := range acceso.Usuario.Perfiles {
			a.Perfiles = append(a.Perfiles, string(p))
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			datos, e := m.Servicio.Leer(r.Context(), a, r.PathValue("id"))
			if e != nil {
				errorEvaluacion(w, e)
				return
			}
			responderJSON(w, 200, datos)
			return
		}
		tipo, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || tipo != "application/json" {
			responderError(w, 415, "evaluacion.peticion_invalida")
			return
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
		dec.DisallowUnknownFields()
		var c entrada.Cambio
		if dec.Decode(&c) != nil || dec.Decode(&struct{}{}) != io.EOF {
			responderError(w, 400, "evaluacion.peticion_invalida")
			return
		}
		datos, e := m.Servicio.Guardar(r.Context(), a, r.PathValue("id"), c)
		if e != nil {
			errorEvaluacion(w, e)
			return
		}
		responderJSON(w, 200, datos)
	})
	return []Ruta{{ReglaRuta: lectura("/api/expedientes/{id}/evaluaciones", "profesional_etf", "coordinacion"), Handler: h}, {ReglaRuta: escritura("/api/expedientes/{id}/evaluaciones", []string{http.MethodPost}, "profesional_etf", "coordinacion"), Handler: h}}
}
func errorEvaluacion(w http.ResponseWriter, e error) {
	var c entrada.Conflicto
	if errors.As(e, &c) {
		responderJSON(w, 409, struct {
			Clave   string `json:"clave"`
			Version int    `json:"version_actual"`
		}{"evaluacion.conflicto", c.VersionActual})
		return
	}
	estado := 500
	clave := "evaluacion.no_disponible"
	switch {
	case errors.Is(e, entrada.ErrAcceso):
		estado = 403
		clave = "evaluacion.acceso_denegado"
	case errors.Is(e, entrada.ErrNoEncontrada):
		estado = 404
		clave = "evaluacion.no_encontrada"
	case errors.Is(e, entrada.ErrPeticion):
		estado = 400
		clave = "evaluacion.peticion_invalida"
	case errors.Is(e, entrada.ErrSujeto):
		estado = 400
		clave = "evaluacion.sujeto_invalido"
	case errors.Is(e, entrada.ErrPuntuacion):
		estado = 400
		clave = "evaluacion.puntuacion_invalida"
	case errors.Is(e, entrada.ErrVersionEscala):
		estado = 400
		clave = "evaluacion.version_escala_invalida"
	case errors.Is(e, entrada.ErrIdempotencia):
		estado = 409
		clave = "evaluacion.idempotencia_incompatible"
	case errors.Is(e, entrada.ErrNoDisponible):
		estado = 503
	}
	responderError(w, estado, clave)
}
