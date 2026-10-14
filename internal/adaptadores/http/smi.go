package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
)

type ServicioSMI interface {
	Consultar(ctx context.Context, actor entrada.Actor) ([]entrada.SMI, error)
	Guardar(ctx context.Context, actor entrada.Actor, cambio entrada.CambioSMI) (entrada.SMI, error)
}

type ConfiguracionSMI struct {
	Servicio ServicioSMI
}

func registrarSMI(mux *muxHTTP, c ConfiguracionSMI) {
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			responderError(w, http.StatusServiceUnavailable, "catalogo.no_disponible")
			return
		}
		actor, err := actorDerivaciones(r)
		if err != nil {
			errorSMI(w, err)
			return
		}
		actorSMI := entrada.Actor{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			filas, err := c.Servicio.Consultar(r.Context(), actorSMI)
			if err != nil {
				errorSMI(w, err)
				return
			}
			if r.URL.Query().Has("anio") {
				if len(r.URL.Query()["anio"]) != 1 {
					responderError(w, http.StatusBadRequest, "catalogo.smi_invalido")
					return
				}
				anio, err := strconv.Atoi(r.URL.Query().Get("anio"))
				if err != nil {
					responderError(w, http.StatusBadRequest, "catalogo.smi_invalido")
					return
				}
				for _, fila := range filas {
					if fila.Anio == anio {
						responderJSON(w, http.StatusOK, fila)
						return
					}
				}
				responderError(w, http.StatusNotFound, "catalogo.smi_no_disponible")
				return
			}
			responderJSON(w, http.StatusOK, filas)
		case http.MethodPut:
			if r.Header.Get("Content-Type") != "application/json" {
				responderError(w, http.StatusUnsupportedMediaType, "catalogo.smi_invalido")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
			defer r.Body.Close()
			var peticion struct {
				Anio    int    `json:"anio"`
				Importe string `json:"importe_mensual"`
				Version int    `json:"version_esperada"`
			}
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&peticion); err != nil {
				responderError(w, http.StatusBadRequest, "catalogo.smi_invalido")
				return
			}
			if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
				responderError(w, http.StatusBadRequest, "catalogo.smi_invalido")
				return
			}
			centimos, err := entrada.ParsearImporteSMI(peticion.Importe)
			if err != nil {
				responderError(w, http.StatusBadRequest, "catalogo.smi_invalido")
				return
			}
			fila, err := c.Servicio.Guardar(r.Context(), actorSMI, entrada.CambioSMI{Anio: peticion.Anio, ImporteMensualCentimos: centimos, VersionEsperada: peticion.Version})
			if err != nil {
				errorSMI(w, err)
				return
			}
			responderJSON(w, http.StatusOK, fila)
		default:
			w.Header().Set("Allow", "GET, HEAD, PUT")
			responderError(w, http.StatusMethodNotAllowed, "http.metodo_no_permitido")
		}
	}), lectura("/api/coordinacion/smi", "coordinacion"), escritura("/api/coordinacion/smi", []string{http.MethodPut}, "coordinacion"))
}

func errorSMI(w http.ResponseWriter, err error) {
	var conflicto *entrada.Conflicto
	if errors.As(err, &conflicto) {
		responderJSON(w, http.StatusConflict, map[string]any{"error": "catalogo.conflicto", "version_actual": conflicto.Actual, "recurso": conflicto.Recurso})
		return
	}
	switch {
	case errors.Is(err, entrada.ErrPermiso):
		responderError(w, http.StatusForbidden, "catalogo.acceso_denegado")
	case errors.Is(err, entrada.ErrSMI):
		responderError(w, http.StatusBadRequest, "catalogo.smi_invalido")
	case err.Error() == "expediente.acceso_denegado" || err.Error() == "identidad.sesion_no_valida" || err.Error() == "identidad.usuario_no_disponible":
		responderError(w, http.StatusForbidden, "catalogo.acceso_denegado")
	default:
		responderError(w, http.StatusServiceUnavailable, "catalogo.no_disponible")
	}
}
