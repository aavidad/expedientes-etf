package http

import (
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
)

type ConfiguracionNarrativas struct{ Servicio entrada.Narrativas }
type ModuloNarrativas struct{ Configuracion ConfiguracionNarrativas }

func (m ModuloNarrativas) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) { registrarNarrativas(mux, m.Configuracion) })
}
func actorMemoria(r *http.Request) entrada.ActorMemoria {
	acceso := identidad.AccesoActual(r.Context())
	a := entrada.ActorMemoria{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
	for _, p := range acceso.Usuario.Perfiles {
		a.Perfiles = append(a.Perfiles, string(p))
	}
	return a
}
func errorMemoria(w http.ResponseWriter, err error) {
	estado, clave := http.StatusServiceUnavailable, string(entrada.ErrDisponibleMemoria)
	var conflicto *entrada.ConflictoMemoria
	switch {
	case errors.As(err, &conflicto):
		responderJSON(w, http.StatusConflict, map[string]any{"clave": conflicto.Error(), "version_actual": conflicto.Actual})
		return
	case errors.Is(err, entrada.ErrPermisoMemoria):
		estado, clave = http.StatusForbidden, string(entrada.ErrPermisoMemoria)
	case errors.Is(err, entrada.ErrRevisionMemoria):
		estado, clave = http.StatusNotFound, string(entrada.ErrRevisionMemoria)
	case errors.Is(err, entrada.ErrIdempotenciaMemoria), errors.Is(err, entrada.ErrBorradorMemoria):
		estado, clave = http.StatusConflict, err.Error()
	case errors.Is(err, entrada.ErrActoMemoria):
		estado, clave = http.StatusBadRequest, string(entrada.ErrActoMemoria)
	case errors.Is(err, entrada.ErrCerradaMemoria):
		estado, clave = http.StatusConflict, string(entrada.ErrCerradaMemoria)
	case errors.Is(err, entrada.ErrNarrativasMemoria), errors.Is(err, entrada.ErrEntradaSPAF):
		estado, clave = http.StatusBadRequest, string(entrada.ErrNarrativasMemoria)
	}
	responderError(w, estado, clave)
}
func registrarNarrativas(mux *muxHTTP, c ConfiguracionNarrativas) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if c.Servicio == nil {
			errorMemoria(w, entrada.ErrDisponibleMemoria)
			return
		}
		q, errConsulta := url.ParseQuery(r.URL.RawQuery)
		anio, err := strconv.Atoi(q.Get("anio"))
		valido := err == nil && errConsulta == nil && len(q["anio"]) == 1 && len(q["etf"]) <= 1 && (!q.Has("etf") || q.Get("etf") != "")
		for k := range q {
			if k != "anio" && k != "etf" {
				valido = false
			}
		}
		f := entrada.FiltroMemoria{Anio: anio, EquipoID: q.Get("etf")}
		a := actorMemoria(r)
		if !valido {
			_, err := c.Servicio.GuardarNarrativas(r.Context(), a, entrada.NarrativaMemoria{FiltroMemoria: f, Version: -1})
			errorMemoria(w, err)
			return
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			n, err := c.Servicio.LeerNarrativas(r.Context(), a, f)
			if err != nil {
				errorMemoria(w, err)
				return
			}
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			responderJSON(w, http.StatusOK, n)
			return
		}
		var n entrada.NarrativaMemoria
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		leer := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
		leer.DisallowUnknownFields()
		if err != nil || tipo != "application/json" || leer.Decode(&n) != nil || leer.Decode(new(any)) != io.EOF || (n.Anio != 0 && n.Anio != f.Anio) || (n.EquipoID != "" && n.EquipoID != f.EquipoID) {
			n = entrada.NarrativaMemoria{Version: -1}
		}
		n.FiltroMemoria = f
		result, err := c.Servicio.GuardarNarrativas(r.Context(), a, n)
		if err != nil {
			errorMemoria(w, err)
			return
		}
		responderJSON(w, http.StatusOK, result)
	})
	mux.declarar(handler, lectura("/api/estadistica/memoria/narrativas", "coordinacion", "profesional_etf"), escritura("/api/estadistica/memoria/narrativas", []string{http.MethodPut}, "coordinacion", "profesional_etf"))
}
