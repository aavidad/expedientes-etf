package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"time"
)

func registrarSalidasServicio(mux *muxHTTP, servicio entrada.SalidasServicio) {
	mux.declarar(handlerSalidaServicio(servicio, "/api/expedientes/{id}/derivaciones-salida"), escritura("/api/expedientes/{id}/derivaciones-salida", []string{http.MethodPost}, "coordinacion", "profesional_etf"))
}

func handlerSalidaServicio(servicio entrada.SalidasServicio, ruta string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		if servicio == nil {
			errorAceptacion(w, entrada.ErrNoDisponible)
			return
		}
		sesion := identidad.AccesoActual(ctx)
		a := entrada.Actor{UsuarioID: sesion.Usuario.ID, EquipoID: sesion.Usuario.EquipoID}
		for _, p := range sesion.Usuario.Perfiles {
			a.Perfiles = append(a.Perfiles, string(p))
		}
		rechazar := func(err error) {
			if fallo := servicio.RechazarSalida(ctx, a, err.Error()); fallo != nil {
				errorAceptacion(w, entrada.ErrNoDisponible)
				return
			}
			errorSalidaServicio(w, err)
		}
		if err := a.AutorizarSalida(""); err != nil {
			rechazar(err)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" || len(r.Header.Values("Idempotency-Key")) != 1 {
			rechazar(entrada.ErrPeticion)
			return
		}
		var p entrada.NuevaSalidaServicio
		var h entrada.ConfirmarSalidaServicio
		var destino any = &p
		if ruta == "/api/expedientes/{id}/hitos" {
			destino = &h
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 24<<10))
		dec.DisallowUnknownFields()
		if dec.Decode(destino) != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
			rechazar(entrada.ErrPeticion)
			return
		}
		var resultado entrada.SalidaServicio
		if destino == &h {
			resultado, err = servicio.ConfirmarSalidaServicio(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), h)
		} else {
			resultado, err = servicio.CrearSalidaServicio(ctx, a, r.PathValue("id"), r.Header.Get("Idempotency-Key"), p)
		}
		if err != nil {
			errorSalidaServicio(w, err)
			return
		}
		responderJSON(w, http.StatusOK, resultado)
	})
}

func errorSalidaServicio(w http.ResponseWriter, err error) {
	if errors.Is(err, entrada.ErrPeticion) {
		responderError(w, http.StatusBadRequest, "expediente.salida_servicio.peticion_invalida")
		return
	}
	errorAceptacion(w, err)
}
