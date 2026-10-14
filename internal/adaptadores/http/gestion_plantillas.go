package http

import (
	"context"
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"
)

type ModuloPlantillas struct{}

func (ModuloPlantillas) Rutas() []Ruta {
	return []Ruta{pantalla("/admin/plantillas", "plantillas.titulo", "administracion")}
}
func actorPlantillas(r *http.Request) entrada.Actor {
	acceso := identidad.AccesoActual(r.Context())
	a := entrada.Actor{UsuarioID: acceso.Usuario.ID}
	for _, p := range acceso.Usuario.Perfiles {
		a.Perfiles = append(a.Perfiles, string(p))
	}
	return a
}
func registrarGestionPlantillas(mux *muxHTTP, s entrada.GestionPlantillas) {
	plazas := make(chan struct{}, 2)
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancelar()
		if s == nil {
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		a := actorPlantillas(r)
		rechazar := func(err error) {
			if e := s.Rechazar(ctx, a); e != nil {
				err = entrada.ErrNoDisponible
			}
			errorDocumento(w, err)
		}
		if err := a.AutorizarPlantillas(); err != nil {
			rechazar(err)
			return
		}
		q := r.URL.Query()
		for k, v := range q {
			if len(v) != 1 || (k != "tipo" && k != "idioma" && k != "version" && k != "version_previa") {
				rechazar(entrada.ErrPeticion)
				return
			}
		}
		previa, err := strconv.ParseInt(q.Get("version_previa"), 10, 64)
		p := entrada.PublicarPlantilla{Tipo: q.Get("tipo"), Idioma: q.Get("idioma"), Version: q.Get("version"), VersionPrevia: previa}
		tipo, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || e != nil || tipo != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" || p.Validar() != nil {
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
		resultado, err := s.Subir(ctx, a, p, b)
		if err != nil {
			errorDocumento(w, err)
			return
		}
		responderJSON(w, 201, resultado)
	}), escritura("/api/admin/plantillas/versiones", []string{http.MethodPost}, "administracion"))
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancelar()
		if s == nil {
			errorDocumento(w, entrada.ErrNoDisponible)
			return
		}
		a := actorPlantillas(r)
		rechazar := func(err error) {
			if e := s.Rechazar(ctx, a); e != nil {
				err = entrada.ErrNoDisponible
			}
			errorDocumento(w, err)
		}
		if err := a.AutorizarPlantillas(); err != nil {
			rechazar(err)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" || len(r.URL.Query()) != 0 {
			rechazar(entrada.ErrPeticion)
			return
		}
		var cuerpo struct {
			VersionPrevia int64  `json:"version_previa"`
			Idioma        string `json:"idioma"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
		dec.DisallowUnknownFields()
		if dec.Decode(&cuerpo) != nil || dec.Decode(new(any)) != io.EOF {
			rechazar(entrada.ErrPeticion)
			return
		}
		resultado, err := s.Activar(ctx, a, entrada.ActivarPlantilla{ID: r.PathValue("id"), Idioma: cuerpo.Idioma, VersionPrevia: cuerpo.VersionPrevia})
		if err != nil {
			errorDocumento(w, err)
			return
		}
		responderJSON(w, 200, resultado)
	}), escritura("/api/admin/plantillas/{id}/activar", []string{http.MethodPost}, "administracion"))
}
