package http

import (
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"net/http"
	"net/url"
	"strconv"
)

type ConfiguracionSPAF struct {
	Servicio entrada.ConsultaSPAF
	Opciones entrada.ConsultaOpcionesSPAF
}

func registrarSPAF(mux *muxHTTP, c ConfiguracionSPAF) {
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.RawQuery != "" {
			responderError(w, 400, string(entrada.ErrEntradaSPAF))
			return
		}
		if c.Opciones == nil {
			responderError(w, 503, string(entrada.ErrDisponibleSPAF))
			return
		}
		acceso := identidad.AccesoActual(r.Context())
		a := entrada.ActorSPAF{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
		for _, p := range acceso.Usuario.Perfiles {
			a.Perfiles = append(a.Perfiles, string(p))
		}
		opciones, err := c.Opciones.ConsultarOpcionesSPAF(r.Context(), a)
		if err != nil {
			if errors.Is(err, entrada.ErrPermisoSPAF) {
				responderError(w, 403, string(entrada.ErrPermisoSPAF))
			} else {
				responderError(w, 503, string(entrada.ErrDisponibleSPAF))
			}
			return
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(200)
			return
		}
		responderJSON(w, 200, opciones)
	}), lectura("/api/estadistica/spaf/opciones", "coordinacion", "profesional_etf"))
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			responderError(w, http.StatusMethodNotAllowed, "http.metodo_no_permitido")
			return
		}
		q, err := url.ParseQuery(r.URL.RawQuery)
		parametroAjeno := false
		for clave := range q {
			if clave != "anio" && clave != "mes" && clave != "etf" {
				parametroAjeno = true
			}
		}
		if err != nil || parametroAjeno || len(q["anio"]) != 1 || len(q["mes"]) != 1 || len(q["etf"]) > 1 || (q.Has("etf") && q.Get("etf") == "") || len(q) > 3 {
			responderError(w, http.StatusBadRequest, string(entrada.ErrEntradaSPAF))
			return
		}
		anio, errAnio := strconv.Atoi(q.Get("anio"))
		mes, errMes := strconv.Atoi(q.Get("mes"))
		if errAnio != nil || errMes != nil {
			responderError(w, http.StatusBadRequest, string(entrada.ErrEntradaSPAF))
			return
		}
		if c.Servicio == nil {
			responderError(w, http.StatusServiceUnavailable, string(entrada.ErrDisponibleSPAF))
			return
		}
		acceso := identidad.AccesoActual(r.Context())
		a := entrada.ActorSPAF{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
		for _, p := range acceso.Usuario.Perfiles {
			a.Perfiles = append(a.Perfiles, string(p))
		}
		tabla, err := c.Servicio.ConsultarSPAF(r.Context(), a, entrada.FiltroSPAF{Anio: anio, Mes: mes, EquipoID: q.Get("etf")})
		if err != nil {
			estado, clave := http.StatusServiceUnavailable, string(entrada.ErrDisponibleSPAF)
			if errors.Is(err, entrada.ErrEntradaSPAF) {
				estado, clave = http.StatusBadRequest, string(entrada.ErrEntradaSPAF)
			}
			if errors.Is(err, entrada.ErrPermisoSPAF) {
				estado, clave = http.StatusForbidden, string(entrada.ErrPermisoSPAF)
			}
			responderError(w, estado, clave)
			return
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		responderJSON(w, http.StatusOK, tabla)
	}), lectura("/api/estadistica/spaf", "coordinacion", "profesional_etf"))
}
