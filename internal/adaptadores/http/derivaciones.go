package http

import (
	"context"
	"errors"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"net/http"
	"strings"
	"time"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
)

type ConfiguracionDerivaciones struct{ Servicio entrada.Derivaciones }

// actorDerivaciones es el único punto de obtención de identidad. Consulta la
// sesión vigente; no admite actores ni permisos en cabeceras o multipart.
func actorDerivaciones(r *http.Request) (entrada.Actor, error) {
	acceso := identidad.AccesoActual(r.Context())
	if !acceso.Permite("coordinacion") {
		return entrada.Actor{}, entrada.ErrAcceso
	}
	actor := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
	for _, p := range acceso.Usuario.Perfiles {
		actor.Perfiles = append(actor.Perfiles, string(p))
	}
	return actor, nil
}

func registrarDerivaciones(mux *muxHTTP, c ConfiguracionDerivaciones) {
	registrarConfirmacion(mux, c)
	plazas := make(chan struct{}, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		r = r.WithContext(ctx)
		if c.Servicio == nil {
			errorDerivacion(w, entrada.ErrNoDisponible)
			return
		}
		actor, err := actorDerivaciones(r)
		if err != nil {
			if fallo := c.Servicio.Rechazar(r.Context(), entrada.Actor{}, err.Error()); fallo != nil {
				errorDerivacion(w, entrada.ErrNoDisponible)
				return
			}
			errorDerivacion(w, err)
			return
		}
		rechazar := func(err error) {
			if fallo := c.Servicio.Rechazar(r.Context(), actor, err.Error()); fallo != nil {
				errorDerivacion(w, fallo)
				return
			}
			errorDerivacion(w, err)
		}
		if err := actor.Autorizar(); err != nil {
			rechazar(err)
			return
		}
		if r.Method == http.MethodPost {
			select {
			case plazas <- struct{}{}:
				defer func() { <-plazas }()
			case <-r.Context().Done():
				errorDerivacion(w, entrada.ErrNoDisponible)
				return
			}
			nombre, contenido, err := leerAnexo(w, r)
			if err != nil {
				rechazar(err)
				return
			}
			defer clear(contenido)
			d, err := c.Servicio.Subir(r.Context(), actor, entrada.Subida{Nombre: nombre, Contenido: contenido})
			if err != nil {
				errorDerivacion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, d)
			return
		}
		{
			id := r.PathValue("id")
			d, err := c.Servicio.Consultar(r.Context(), actor, id)
			if err != nil {
				errorDerivacion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, d)
			return
		}
	})
	mux.declarar(handler, escritura("/api/coordinacion/derivaciones", []string{http.MethodPost}, "coordinacion"))
	mux.declarar(handler, lectura("/api/coordinacion/derivaciones/{id}", "coordinacion"))
}

func leerAnexo(w http.ResponseWriter, r *http.Request) (string, []byte, error) {
	// El margen permite las cabeceras multipart; el fichero tiene su propia cota.
	r.Body = http.MaxBytesReader(w, r.Body, entrada.MaxBytes+(64<<10))
	m, err := r.MultipartReader()
	if err != nil {
		return "", nil, entrada.ErrPeticion
	}
	p, err := m.NextPart()
	if err != nil {
		return "", nil, entrada.ErrPeticion
	}
	defer p.Close()
	if p.FormName() != "anexo" || p.FileName() == "" {
		return "", nil, entrada.ErrPeticion
	}
	b, err := io.ReadAll(io.LimitReader(p, entrada.MaxBytes+1))
	var demasiado *http.MaxBytesError
	if errors.As(err, &demasiado) || len(b) > entrada.MaxBytes {
		clear(b)
		return "", nil, entrada.ErrTamano
	}
	if err != nil {
		clear(b)
		return "", nil, entrada.ErrPeticion
	}
	if _, err := m.NextPart(); err != io.EOF {
		clear(b)
		if errors.As(err, &demasiado) {
			return "", nil, entrada.ErrTamano
		}
		return "", nil, entrada.ErrPeticion
	}
	return p.FileName(), b, nil
}

func errorDerivacion(w http.ResponseWriter, err error) {
	estado, clave := http.StatusServiceUnavailable, "expediente.no_disponible"
	switch {
	case errors.Is(err, entrada.ErrAcceso):
		estado, clave = http.StatusForbidden, "expediente.acceso_denegado"
	case errors.Is(err, entrada.ErrTamano):
		estado, clave = http.StatusRequestEntityTooLarge, "expediente.anexo_tamano"
	case errors.Is(err, entrada.ErrInfectado):
		estado, clave = http.StatusUnprocessableEntity, "expediente.anexo_infectado"
	case errors.Is(err, entrada.ErrNoEncontrada):
		estado, clave = http.StatusNotFound, "expediente.derivacion_no_encontrada"
	case errors.Is(err, entrada.ErrPeticion):
		estado, clave = http.StatusBadRequest, "expediente.peticion_invalida"
	case errors.Is(err, entrada.ErrFormato):
		estado, clave = http.StatusUnprocessableEntity, "expediente.anexo_formato"
	case err.Error() == "identidad.csrf_no_valido":
		estado, clave = http.StatusForbidden, "identidad.csrf_no_valido"
	case strings.HasPrefix(err.Error(), "anexo."):
		estado, clave = http.StatusUnprocessableEntity, err.Error()
	case err.Error() == "expediente.anexo_demasiados_menores":
		estado, clave = http.StatusUnprocessableEntity, "expediente.anexo_demasiados_menores"
	}
	responderError(w, estado, clave)
}
