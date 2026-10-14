package http

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"
)

func registrarPlantillas(mux *muxHTTP, c ConfiguracionDocumentos) {
	plazas := make(chan struct{}, 2)
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancelar()
		if c.Plantillas == nil {
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		acceso := identidad.AccesoActual(ctx)
		actor := entrada.Actor{UsuarioID: acceso.Usuario.ID}
		for _, p := range acceso.Usuario.Perfiles {
			actor.Perfiles = append(actor.Perfiles, string(p))
		}
		rechazar := func(err error) {
			if e := c.Plantillas.Rechazar(ctx, actor); e != nil {
				err = entrada.ErrNoDisponible
			}
			errorDocumento(w, err)
		}
		if err := actor.AutorizarPlantillas(); err != nil {
			rechazar(err)
			return
		}
		q, err := r.URL.Query(), error(nil)
		lectura := r.Method == http.MethodGet || r.Method == http.MethodHead
		if !lectura && r.Method != http.MethodPost {
			rechazar(entrada.ErrPeticion)
			return
		}
		for k, v := range q {
			if len(v) != 1 || (k != "tipo" && k != "idioma" && (r.Method != http.MethodPost || (k != "version" && k != "version_previa"))) {
				rechazar(entrada.ErrPeticion)
				return
			}
		}
		if lectura && len(q) == 0 && c.GestionPlantillas != nil {
			versiones, err := c.GestionPlantillas.Listar(ctx, actor)
			if err != nil {
				errorDocumento(w, err)
				return
			}
			responderJSON(w, 200, versiones)
			return
		}
		if lectura {
			actual, err := c.Plantillas.Actual(ctx, actor, q.Get("tipo"), q.Get("idioma"))
			if err != nil {
				errorDocumento(w, err)
				return
			}
			responderJSON(w, 200, actual)
			return
		}
		p := entrada.PublicarPlantilla{Tipo: q.Get("tipo"), Idioma: q.Get("idioma"), Version: q.Get("version")}
		p.VersionPrevia, err = strconv.ParseInt(q.Get("version_previa"), 10, 64)
		if err != nil || p.Validar() != nil {
			rechazar(entrada.ErrPeticion)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
			rechazar(entrada.ErrPeticion)
			return
		}
		select {
		case plazas <- struct{}{}:
			defer func() { <-plazas }()
		default:
			rechazar(entrada.ErrNoDisponible)
			return
		}
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, entrada.MaximoPlantilla))
		defer clear(b)
		if err != nil {
			var exceso *http.MaxBytesError
			if errors.As(err, &exceso) {
				rechazar(entrada.ErrTamanoPlantilla)
			} else {
				rechazar(entrada.ErrPeticion)
			}
			return
		}
		if c.GestionPlantillas == nil {
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		resultado, err := c.GestionPlantillas.Subir(ctx, actor, p, b)
		if err != nil {
			errorDocumento(w, err)
			return
		}
		responderJSON(w, 201, resultado)
	}), lectura("/api/admin/plantillas", "administracion"), escritura("/api/admin/plantillas", []string{http.MethodPost}, "administracion"))
}
