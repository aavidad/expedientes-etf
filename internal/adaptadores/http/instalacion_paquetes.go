package http

import (
	"context"
	"encoding/json"
	"errors"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
)

type ConfiguracionInstalacion struct {
	Instalacion entrada.Instalacion
}

func actorInstalacion(r *http.Request) entrada.Actor {
	a := identidad.AccesoActual(r.Context())
	actor := entrada.Actor{UsuarioID: a.Usuario.ID}
	for _, p := range a.Usuario.Perfiles {
		actor.Perfiles = append(actor.Perfiles, string(p))
	}
	return actor
}

func registrarInstalacion(mux *muxHTTP, c ConfiguracionInstalacion) {
	registrar := func(regla reglaRuta, handler http.HandlerFunc) {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// El actor procede exclusivamente de la sesión autorizada.
			if !actorInstalacion(r).PuedeAdministrar() {
				errorInstalacion(w, entrada.ErrPermiso)
				return
			}
			if c.Instalacion == nil {
				responderError(w, http.StatusServiceUnavailable, "paquete.no_disponible")
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				// El middleware de sesión valida CSRF y el origen configurado;
				// esta cabecera distingue las peticiones del instalador.
				if r.Header.Get("X-ETF-Paquetes") != "1" || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
					errorInstalacion(w, entrada.ErrPermiso)
					return
				}
			}
			handler(w, r)
		}), regla)
	}
	registrar(escritura("/api/admin/paquetes/preparaciones/{ticket}", []string{http.MethodDelete}, "administracion"), func(w http.ResponseWriter, r *http.Request) {
		if err := c.Instalacion.Descartar(r.Context(), actorInstalacion(r), r.PathValue("ticket")); err != nil {
			errorInstalacion(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	registrar(lectura("/api/admin/paquetes/{tipo}/{codigo}", "administracion"), func(w http.ResponseWriter, r *http.Request) {
		actual, err := c.Instalacion.Actual(r.Context(), actorInstalacion(r), r.PathValue("tipo"), r.PathValue("codigo"))
		if err != nil {
			errorInstalacion(w, err)
			return
		}
		responderJSON(w, http.StatusOK, actual)
	})
	registrar(escritura("/api/admin/paquetes/{tipo}/{codigo}/revisar", []string{http.MethodPost}, "administracion"), func(w http.ResponseWriter, r *http.Request) {
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/zip" {
			errorInstalacion(w, entrada.ErrPaqueteInvalido)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
		datos, err := io.ReadAll(r.Body)
		if err != nil {
			var limite *http.MaxBytesError
			if errors.As(err, &limite) {
				responderError(w, http.StatusRequestEntityTooLarge, "paquete.tamano_excedido")
			} else {
				errorInstalacion(w, entrada.ErrPaqueteInvalido)
			}
			return
		}
		ctx, cancelar := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancelar()
		preparacion, err := c.Instalacion.Preparar(ctx, actorInstalacion(r), r.PathValue("tipo"), r.PathValue("codigo"), datos)
		if err != nil {
			errorInstalacion(w, err)
			return
		}
		responderJSON(w, http.StatusCreated, preparacion)
	})
	registrar(escritura("/api/admin/paquetes/activar", []string{http.MethodPost}, "administracion"), func(w http.ResponseWriter, r *http.Request) {
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			errorInstalacion(w, entrada.ErrPaqueteInvalido)
			return
		}
		var peticion struct {
			Ticket  string `json:"ticket"`
			Version *int64 `json:"version"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if dec.Decode(&peticion) != nil || peticion.Version == nil || len(peticion.Ticket) != 64 || dec.Decode(&struct{}{}) != io.EOF {
			errorInstalacion(w, entrada.ErrPaqueteInvalido)
			return
		}
		actual, err := c.Instalacion.Activar(r.Context(), actorInstalacion(r), peticion.Ticket, *peticion.Version)
		if err != nil {
			errorInstalacion(w, err)
			return
		}
		responderJSON(w, http.StatusOK, actual)
	})
}

func errorInstalacion(w http.ResponseWriter, err error) {
	estado, clave := http.StatusServiceUnavailable, "paquete.no_disponible"
	switch {
	case errors.Is(err, entrada.ErrPermiso):
		estado, clave = http.StatusForbidden, "paquete.acceso_denegado"
	case errors.Is(err, entrada.ErrPaqueteInvalido):
		estado, clave = http.StatusBadRequest, "paquete.archivo_invalido"
	case errors.Is(err, entrada.ErrConflicto):
		estado, clave = http.StatusConflict, "paquete.conflicto"
	case errors.Is(err, entrada.ErrPreparacion):
		estado, clave = http.StatusGone, "paquete.preparacion_caducada"
	case errors.Is(err, entrada.ErrAntivirus):
		clave = "paquete.antivirus_no_disponible"
	case errors.Is(err, entrada.ErrInfectado):
		estado, clave = http.StatusUnprocessableEntity, "paquete.infectado"
	case errors.Is(err, entrada.ErrOcupado):
		estado, clave = http.StatusTooManyRequests, "paquete.ocupado"
	}
	responderError(w, estado, clave)
}
