package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"time"
)

func registrarRegistroSalida(mux *muxHTTP, c ConfiguracionDocumentos) {
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		r = r.WithContext(ctx)
		if c.Registros == nil {
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		var actor entrada.Actor
		rechazar := func(err error) {
			if fallo := c.Registros.Rechazar(ctx, actor, err.Error()); fallo != nil {
				errorDocumento(w, entrada.ErrNoDisponible)
				return
			}
			errorDocumento(w, err)
		}
		a, err := actorDerivaciones(r)
		if err != nil {
			rechazar(entrada.ErrAcceso)
			return
		}
		actor = entrada.Actor{UsuarioID: a.UsuarioID, Perfiles: a.Perfiles, EquipoID: a.EquipoID}
		if err := actor.AutorizarEntrada(); err != nil {
			rechazar(err)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			rechazar(entrada.ErrPeticion)
			return
		}
		if len(r.Header.Values("Idempotency-Key")) != 1 {
			rechazar(entrada.ErrPeticion)
			return
		}
		var p entrada.SolicitudRegistroSalida
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&p); err != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			rechazar(entrada.ErrPeticion)
			return
		}
		p.EmisionID = r.PathValue("id")
		p.Clave = r.Header.Get("Idempotency-Key")
		if err := p.Validar(); err != nil {
			rechazar(err)
			return
		}
		resultado, err := c.Registros.AnotarRegistroSalida(ctx, actor, p)
		if err != nil {
			errorDocumento(w, err)
			return
		}
		responderJSON(w, http.StatusOK, resultado)
	}), escritura("/api/documentos/emisiones/{id}/registro-salida", []string{http.MethodPost}, "coordinacion"))
}
