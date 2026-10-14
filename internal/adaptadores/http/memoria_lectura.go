package http

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
)

type ConfiguracionLecturaMemoria struct {
	Consulta entrada.ConsultaCifrasMemoria
	Servicio entrada.ConsultaMemoria
	Equipos  entrada.ConsultaEquiposMemoria
}
type ModuloLecturaMemoria struct{ Configuracion ConfiguracionLecturaMemoria }

func (m ModuloLecturaMemoria) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) { registrarLecturaMemoria(mux, m.Configuracion) }, pantalla("/memoria", "memoria.titulo", "coordinacion", "profesional_etf", "consulta")), "/memoria")
}

func registrarLecturaMemoria(mux *muxHTTP, c ConfiguracionLecturaMemoria) {
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.RawQuery != "" {
			responderError(w, http.StatusBadRequest, string(entrada.ErrEntradaSPAF))
			return
		}
		if c.Equipos == nil {
			errorMemoria(w, entrada.ErrDisponibleMemoria)
			return
		}
		equipos, err := c.Equipos.ConsultarEquiposMemoria(r.Context(), actorMemoria(r))
		if err != nil {
			errorMemoria(w, err)
			return
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		responderJSON(w, http.StatusOK, equipos)
	}), lectura("/api/estadistica/memoria/equipos", "coordinacion", "profesional_etf", "consulta"))
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		q, err := url.ParseQuery(r.URL.RawQuery)
		valida := err == nil && len(q["anio"]) == 1 && len(q["etf"]) <= 1 && (!q.Has("etf") || q.Get("etf") != "")
		for clave := range q {
			if clave != "anio" && clave != "etf" {
				valida = false
			}
		}
		anio, err := strconv.Atoi(q.Get("anio"))
		if !valida || err != nil {
			responderError(w, http.StatusBadRequest, string(entrada.ErrEntradaSPAF))
			return
		}
		if c.Servicio == nil {
			responderError(w, http.StatusServiceUnavailable, string(entrada.ErrDisponibleMemoria))
			return
		}
		actor := actorMemoria(r)
		if c.Consulta != nil && slices.Contains(actor.Perfiles, "consulta") && !slices.Contains(actor.Perfiles, "coordinacion") && (!slices.Contains(actor.Perfiles, "profesional_etf") || q.Get("etf") != actor.EquipoID) {
			cifras, err := c.Consulta.ConsultarCifras(r.Context(), actor, entrada.FiltroMemoria{Anio: anio, EquipoID: q.Get("etf")})
			if err != nil {
				errorNavegacionMemoria(w, err)
				return
			}
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			responderJSON(w, http.StatusOK, cifras)
			return
		}
		cifras, err := c.Servicio.ConsultarMemoria(r.Context(), actorMemoria(r), entrada.FiltroMemoria{Anio: anio, EquipoID: q.Get("etf")})
		if err != nil {
			if errors.Is(err, entrada.ErrCifrasNoPublicadasMemoria) {
				responderError(w, http.StatusNotFound, string(entrada.ErrCifrasNoPublicadasMemoria))
				return
			}
			if errors.Is(err, entrada.ErrEntradaSPAF) {
				responderError(w, http.StatusBadRequest, string(entrada.ErrEntradaSPAF))
			} else {
				errorMemoria(w, err)
			}
			return
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		responderJSON(w, http.StatusOK, cifras)
	}), lectura("/api/estadistica/memoria", "coordinacion", "profesional_etf", "consulta"))
}
