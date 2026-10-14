package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
)

func registrarHitos(mux *muxHTTP, servicio entrada.Hitos, salidas ...entrada.SalidasServicio) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		if servicio == nil {
			errorAceptacion(w, entrada.ErrNoDisponible)
			return
		}
		acceso := identidad.AccesoActual(ctx)
		a := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
		for _, perfil := range acceso.Usuario.Perfiles {
			a.Perfiles = append(a.Perfiles, string(perfil))
		}
		rechazar := func(err error) {
			if fallo := servicio.RechazarHito(ctx, a, err.Error()); fallo != nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			errorHito(w, err)
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			hitos, err := servicio.ConsultarHitos(ctx, a, r.PathValue("id"))
			if err != nil {
				errorHito(w, err)
				return
			}
			responderJSON(w, http.StatusOK, hitos)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			rechazar(entrada.ErrPeticion)
			return
		}
		if len(r.Header.Values("Idempotency-Key")) != 1 {
			rechazar(entrada.ErrClave)
			return
		}
		var p entrada.ConfirmacionHito
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&p); err != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
			rechazar(entrada.ErrPeticion)
			return
		}
		resultado, err := servicio.ConfirmarHito(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), p)
		if err != nil {
			errorHito(w, err)
			return
		}
		responderJSON(w, http.StatusOK, resultado)
	})
	if len(salidas) > 0 && salidas[0] != nil {
		anterior := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))

				if err != nil {
					acceso := identidad.AccesoActual(r.Context())
					a := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
					for _, perfil := range acceso.Usuario.Perfiles {
						a.Perfiles = append(a.Perfiles, string(perfil))
					}
					if fallo := salidas[0].RechazarSalida(r.Context(), a, entrada.ErrPeticion.Error()); fallo != nil {
						errorAceptacion(w, entrada.ErrNoDisponible)
					} else {
						errorHito(w, entrada.ErrPeticion)
					}
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(cuerpo))
				var tipo struct {
					Tipo string `json:"tipo"`
				}
				if err == nil && json.Unmarshal(cuerpo, &tipo) == nil && tipo.Tipo == "derivacion_otro_servicio" {
					handlerSalidaServicio(salidas[0], "/api/expedientes/{id}/hitos").ServeHTTP(w, r)
					return
				}
			}
			anterior.ServeHTTP(w, r)
		})
	}

	mux.declarar(handler,
		lectura("/api/expedientes/{id}/hitos", "coordinacion", "profesional_etf"),
		escritura("/api/expedientes/{id}/hitos", []string{http.MethodPost}, "coordinacion", "profesional_etf"))
}

func errorHito(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrHitoTipo) {
		responderError(w, http.StatusUnprocessableEntity, "expediente.hito_tipo_invalido")
		return
	}
	if errors.Is(err, entrada.ErrHitoFecha) {
		responderError(w, http.StatusUnprocessableEntity, "expediente.hito_fecha_invalida")
		return
	}
	errorAceptacion(w, err)
}
