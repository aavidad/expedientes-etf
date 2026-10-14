package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	estadistica "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"time"
)

type ConfiguracionMemoriaWord struct{ Servicio entrada.EmisionesMemoria }
type ModuloMemoriaWord struct{ Configuracion ConfiguracionMemoriaWord }

func (m ModuloMemoriaWord) Rutas() []Ruta {
	return rutasDeModulo(func(mux *muxHTTP) { registrarMemoriaWord(mux, m.Configuracion) })
}
func registrarMemoriaWord(mux *muxHTTP, c ConfiguracionMemoriaWord) {
	plazas := make(chan struct{}, 2)
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		if c.Servicio == nil {
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		actual := actorMemoria(r)
		a := entrada.Actor{UsuarioID: actual.UsuarioID, EquipoID: actual.EquipoID, Perfiles: actual.Perfiles}
		rechazar := func(err error) {
			if fallo := c.Servicio.RechazarMemoria(ctx, a); fallo != nil {
				errorDocumento(w, entrada.ErrNoDisponible)
				return
			}
			errorDocumento(w, err)
		}
		select {
		case plazas <- struct{}{}:
			defer func() { <-plazas }()
		default:
			rechazar(entrada.ErrNoDisponible)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			rechazar(entrada.ErrPeticion)
			return
		}
		var p entrada.SolicitudMemoria
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&p) != nil || decoder.Decode(new(any)) != io.EOF || len(r.URL.RawQuery) > 0 {
			rechazar(entrada.ErrPeticion)
			return
		}
		p.Clave = r.Header.Get("Idempotency-Key")
		result, err := c.Servicio.EmitirMemoria(ctx, a, p)
		if err != nil {
			if errors.Is(err, estadistica.ErrPermisoMemoria) {
				responderError(w, http.StatusForbidden, string(estadistica.ErrPermisoMemoria))
				return
			}
			errorDocumento(w, err)
			return
		}
		defer clear(result.Contenido)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.Header().Set("Content-Disposition", `attachment; filename="memoria.docx"`)
		w.Header().Set("X-Emision-ID", result.Emision.ID)
		w.Header().Set("X-Documento-SHA256", result.Emision.HuellaResultado)
		w.Header().Set("X-Datos-SHA256", result.Emision.HuellaDatos)
		w.Header().Set("X-Plantilla-Version", result.Emision.Plantilla.Version)
		if result.Repetida {
			w.Header().Set("X-Idempotencia-Repetida", "true")
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write(result.Contenido)
	}), escritura("/api/estadistica/memoria/word", []string{http.MethodPost}, "coordinacion", "profesional_etf"))
}
