package http

import (
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"regexp"
)

type ConfiguracionUsuarios struct{ Servicio entrada.AdministrarUsuarios }

func registrarUsuarios(mux *muxHTTP, c ConfiguracionUsuarios) {
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			responderError(w, 503, "identidad.persistencia_no_disponible")
			return
		}
		usuarios, err := c.Servicio.Listar(r.Context(), entrada.AccesoActual(r.Context()))
		if err != nil {
			errorUsuarios(w, err)
			return
		}
		responderJSON(w, 200, usuarios)
	}), lectura("/api/admin/usuarios", "administracion"))
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			responderError(w, 503, "identidad.persistencia_no_disponible")
			return
		}
		id := r.PathValue("id")
		if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(id) {
			responderError(w, 400, "identidad.peticion_no_valida")
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			responderError(w, 415, "identidad.peticion_no_valida")
			return
		}
		var cambio entrada.CambioPerfiles
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		dec.DisallowUnknownFields()
		if err = dec.Decode(&cambio); err != nil {
			responderError(w, 400, "identidad.peticion_no_valida")
			return
		}
		if err = dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			responderError(w, 400, "identidad.peticion_no_valida")
			return
		}
		usuario, err := c.Servicio.Cambiar(r.Context(), entrada.AccesoActual(r.Context()), id, cambio)
		if err != nil {
			if err.Error() == "identidad.conflicto" {
				actuales, fallo := c.Servicio.Listar(r.Context(), entrada.AccesoActual(r.Context()))
				if fallo != nil {
					errorUsuarios(w, fallo)
					return
				}
				for _, u := range actuales {
					if u.ID == id {
						responderJSON(w, 409, map[string]any{"error": map[string]any{"clave": "identidad.conflicto", "parametros": map[string]any{"version_actual": u.Version}}})
						return
					}
				}
			}
			errorUsuarios(w, err)
			return
		}
		responderJSON(w, 200, usuario)
	}), escritura("/api/admin/usuarios/{id}/perfiles", []string{http.MethodPut}, "administracion"))
}

func errorUsuarios(w http.ResponseWriter, err error) {
	estado, clave := 503, "identidad.persistencia_no_disponible"
	switch err.Error() {
	case "identidad.acceso_denegado", "identidad.cambio_propio_denegado":
		estado, clave = 403, err.Error()
	case "identidad.perfiles_invalidos":
		estado, clave = 400, err.Error()
	case "identidad.conflicto":
		estado, clave = 409, err.Error()
	case "identidad.usuario_no_disponible":
		estado, clave = 404, err.Error()
	}
	responderError(w, estado, clave)
}
