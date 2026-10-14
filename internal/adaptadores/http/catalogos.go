package http

import (
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"net/http"
	"strconv"
	"time"
)

type ConfiguracionCatalogos struct {
	Edicion  entrada.Editor
	Consulta entrada.Catalogos
}

func actorCatalogos(r *http.Request) entrada.Actor {
	a := identidad.AccesoActual(r.Context())
	actor := entrada.Actor{UsuarioID: a.Usuario.ID, EquipoID: a.Usuario.EquipoID}
	for _, p := range a.Usuario.Perfiles {
		actor.Perfiles = append(actor.Perfiles, string(p))
	}
	return actor
}

func registrarCatalogos(mux *muxHTTP, c ConfiguracionCatalogos) {
	registrarConsulta := func(patron string, handler http.HandlerFunc) {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c.Consulta == nil {
				responderError(w, http.StatusServiceUnavailable, "catalogo.no_disponible")
				return
			}
			handler(w, r)
		}), lectura(patron, "administracion"))
	}
	registrarConsulta("/api/admin/catalogos/{ambito}/versiones", func(w http.ResponseWriter, r *http.Request) {
		versiones, err := c.Consulta.Versiones(r.Context(), actorCatalogos(r), r.PathValue("ambito"))
		if err != nil {
			errorCatalogo(w, err)
			return
		}
		responderJSON(w, http.StatusOK, versiones)
	})
	registrarConsulta("/api/admin/catalogos/{ambito}/vigente", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		ambito := r.PathValue("ambito")
		if len(q["fecha"]) > 1 || len(q["anio"]) > 1 {
			errorCatalogo(w, entrada.ErrVigencia)
			return
		}
		if anio := q.Get("anio"); q.Has("anio") {
			if q.Has("fecha") {
				errorCatalogo(w, entrada.ErrVigencia)
				return
			}
			n, err := strconv.Atoi(anio)
			if err != nil {
				errorCatalogo(w, entrada.ErrVigencia)
				return
			}
			v, err := c.Consulta.Memoria(r.Context(), actorCatalogos(r), ambito, n)
			if err != nil {
				errorCatalogo(w, err)
				return
			}
			responderJSON(w, http.StatusOK, v)
			return
		}
		fecha, err := time.Parse(time.DateOnly, q.Get("fecha"))
		if err != nil {
			errorCatalogo(w, entrada.ErrVigencia)
			return
		}
		v, err := c.Consulta.Vigente(r.Context(), actorCatalogos(r), ambito, fecha)
		if err != nil {
			errorCatalogo(w, err)
			return
		}
		responderJSON(w, http.StatusOK, v)
	})
	registrarConsulta("/api/admin/catalogos/{ambito}/datos", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if len(q["fecha"]) != 1 || q.Has("anio") {
			errorCatalogo(w, entrada.ErrVigencia)
			return
		}
		fecha, err := time.Parse(time.DateOnly, q.Get("fecha"))
		if err != nil {
			errorCatalogo(w, entrada.ErrVigencia)
			return
		}
		datos, err := c.Consulta.Datos(r.Context(), actorCatalogos(r), r.PathValue("ambito"), fecha)
		if err != nil {
			errorCatalogo(w, err)
			return
		}
		responderJSON(w, http.StatusOK, datos)
	})
	for _, ambito := range []string{"listas", "plazos", "tramos"} {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atenderEdicion(w, r, c, ambito) }), lectura("/api/admin/catalogos/"+ambito, "administracion"), escritura("/api/admin/catalogos/"+ambito, []string{http.MethodPost, http.MethodPut}, "administracion"))
	}
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atenderEdicion(w, r, c, "firmantes") }), lectura("/api/admin/firmantes", "administracion"), escritura("/api/admin/firmantes", []string{http.MethodPost, http.MethodPut}, "administracion"))
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ambito := r.URL.Query().Get("ambito")
		if ambito != "plazos" && ambito != "tramos" {
			errorCatalogo(w, entrada.ErrAmbito)
			return
		}
		atenderEdicion(w, r, c, ambito)
	}), lectura("/api/admin/reglas", "administracion"), escritura("/api/admin/reglas", []string{http.MethodPost, http.MethodPut}, "administracion"))
}

func errorCatalogo(w http.ResponseWriter, err error) {
	if conflictoCatalogo(w, err) {
		return
	}
	estado, clave := http.StatusServiceUnavailable, "catalogo.no_disponible"
	switch {
	case errors.Is(err, entrada.ErrPermiso):
		estado, clave = http.StatusForbidden, "catalogo.acceso_denegado"
	case errors.Is(err, entrada.ErrEdicion):
		estado, clave = http.StatusBadRequest, "catalogo.edicion_invalida"
	case errors.Is(err, entrada.ErrVigencia):
		estado, clave = http.StatusBadRequest, "catalogo.vigencia_invalida"
	case errors.Is(err, entrada.ErrAmbito):
		estado, clave = http.StatusBadRequest, "catalogo.ambito_invalido"
	case errors.Is(err, entrada.ErrNoDisponible):
		estado, clave = http.StatusNotFound, "catalogo.version_no_disponible"
	}
	responderError(w, estado, clave)
}
