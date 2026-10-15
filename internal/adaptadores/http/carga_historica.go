package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"time"
)

func registrarCargasHistoricas(mux *muxHTTP, s entrada.CargasHistoricas) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		if s == nil {
			errorAceptacion(w, entrada.ErrNoDisponible)
			return
		}
		acceso := actorActividad(r)
		a := entrada.Actor{UsuarioID: acceso.UsuarioID, EquipoID: acceso.EquipoID, Perfiles: acceso.Perfiles}
		rechazar := func(err error) {
			if fallo := s.RechazarCargaHistorica(ctx, a); fallo != nil {
				err = fallo
			}
			errorCargaHistorica(w, err)
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" || len(r.URL.Query()) != 0 {
			rechazar(entrada.ErrPeticion)
			return
		}
		var p entrada.CargaHistorica
		cuerpo, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
		if err != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		// Se cuentan los elementos antes de materializarlos: como mucho 100
		// casos por lote y colecciones pequeñas en cada uno.
		if (&presupuestoJSON{porArray: 500, total: 20000}).revisar(cuerpo) != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		dec := json.NewDecoder(bytes.NewReader(cuerpo))
		dec.DisallowUnknownFields()
		if err = dec.Decode(&p); err != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		if err = dec.Decode(new(any)); !errors.Is(err, io.EOF) {
			rechazar(entrada.ErrPeticion)
			return
		}
		if r.URL.Path == "/api/carga-2026" {
			v, err := s.ValidarCargaHistorica(ctx, a, p)
			if err != nil {
				errorCargaHistorica(w, err)
				return
			}
			responderJSON(w, http.StatusOK, v)
			return
		}
		if len(r.Header.Values("Idempotency-Key")) != 1 {
			rechazar(entrada.ErrClave)
			return
		}
		if r.URL.Path == "/api/expedientes/historicos" && len(p.Casos) != 1 {
			rechazar(entrada.ErrCargaHistorica)
			return
		}
		v, err := s.CargarHistoricos(ctx, a, r.PathValue("lote"), r.Header.Get("Idempotency-Key"), p)
		if err != nil {
			errorCargaHistorica(w, err)
			return
		}
		responderJSON(w, http.StatusOK, v)
	})
	mux.declarar(handler, escritura("/api/expedientes/historicos", []string{http.MethodPost}, "coordinacion", "profesional_etf"), escritura("/api/carga-2026", []string{http.MethodPost}, "coordinacion", "profesional_etf"), escritura("/api/carga-2026/{lote}/confirmar", []string{http.MethodPost}, "coordinacion", "profesional_etf"))
}
func errorCargaHistorica(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entrada.ErrCargaCerrada), errors.Is(err, entrada.ErrDuplicadosSinConfirmar), errors.Is(err, entrada.ErrCargaAgotaNumeracion), errors.Is(err, entrada.ErrLoteConActividad):
		responderError(w, http.StatusConflict, err.Error())
	case errors.Is(err, entrada.ErrControlCarga):
		responderError(w, http.StatusUnprocessableEntity, "carga.control_invalido")
	case errors.Is(err, entrada.ErrCargaHistorica):
		responderError(w, http.StatusUnprocessableEntity, "expediente.carga_historica_invalida")
	case errors.Is(err, entrada.ErrNumeroHistorico):
		responderError(w, http.StatusUnprocessableEntity, "expediente.numero_historico_invalido")
	case errors.Is(err, entrada.ErrNumeroDuplicado):
		responderError(w, http.StatusConflict, "expediente.numero_historico_duplicado")
	default:
		errorMenores(w, err)
	}
}
