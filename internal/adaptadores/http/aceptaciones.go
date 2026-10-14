package http

import (
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

type ConfiguracionAceptaciones struct {
	Hoja                ConfiguracionHoja
	Avisos              entrada.Avisos
	CargasHistoricas    entrada.CargasHistoricas
	TrasladoSeguimiento entrada.TrasladoSeguimiento
	Intervencion        entrada.Intervencion
	Menores             entrada.Menores
	SalidasServicio     entrada.SalidasServicio
	Cierres             entrada.Cierres
	SPM                 entrada.SalidasSPM
	Propuestas          entrada.PropuestasCierre
	Servicio            entrada.Aceptaciones
	Decisiones          entrada.Decisiones
	Familia             entrada.Familias
	Hitos               entrada.Hitos
}

func registrarAceptaciones(mux *muxHTTP, c ConfiguracionAceptaciones) {
	registrarAvisos(mux, c.Avisos)
	registrarCargasHistoricas(mux, c.CargasHistoricas)
	registrarTrasladoSeguimiento(mux, c.TrasladoSeguimiento)
	registrarHoja(mux, c.Hoja)
	registrarCierres(mux, c.Cierres)
	registrarSalidasServicio(mux, c.SalidasServicio)
	registrarDecisiones(mux, c.Decisiones)
	registrarFamilia(mux, c.Familia)
	registrarMenores(mux, c.Menores)
	registrarIntervencion(mux, c.Intervencion)

	registrarSPM(mux, c.SPM)
	registrarHitos(mux, c.Hitos, c.SalidasServicio)

	registrarPropuestasCierre(mux, c.Propuestas)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			handlerFamilia(c.Familia)(w, r)
			return
		}
		ctx, cancelar := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancelar()
		r = r.WithContext(ctx)
		if c.Servicio == nil {
			errorAceptacion(w, entrada.ErrNoDisponible)
			return
		}
		actor, err := actorDerivaciones(r)
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			acceso := identidad.AccesoActual(r.Context())
			actor = entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
			for _, perfil := range acceso.Usuario.Perfiles {
				actor.Perfiles = append(actor.Perfiles, string(perfil))
			}
			err = actor.AutorizarConsulta()
		}
		rechazar := func(actor entrada.Actor, err error) {
			if fallo := c.Servicio.RechazarAceptacion(r.Context(), actor, err.Error()); fallo != nil {
				errorAceptacion(w, fallo)
				return
			}
			errorAceptacion(w, err)
		}
		if err != nil {
			rechazar(entrada.Actor{}, err)
			return
		}
		id := r.PathValue("id")
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			d, err := c.Servicio.Consultar(r.Context(), actor, id)
			if err != nil {
				errorAceptacion(w, err)
				return
			}
			responderJSON(w, http.StatusOK, d)
			return
		}
		if err := actor.Autorizar(); err != nil {
			rechazar(actor, err)
			return
		}
		claves := r.Header.Values("Idempotency-Key")
		if len(claves) != 1 {
			rechazar(actor, entrada.ErrClave)
			return
		}
		tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || tipo != "application/json" {
			rechazar(actor, entrada.ErrPeticion)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var p entrada.Aceptacion
		if err := decoder.Decode(&p); err != nil {
			rechazar(actor, entrada.ErrPeticion)
			return
		}
		if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			rechazar(actor, entrada.ErrPeticion)
			return
		}
		resultado, err := c.Servicio.Aceptar(r.Context(), actor, id, claves[0], p)
		if err != nil {
			errorAceptacion(w, err)
			return
		}
		w.Header().Set("Location", "/api/expedientes/"+id)
		responderJSON(w, http.StatusOK, resultado)
	})
	mux.declarar(handler, escritura("/api/coordinacion/derivaciones/{id}/aceptar", []string{http.MethodPost}, "coordinacion"))
	mux.declarar(handler, lectura("/api/expedientes/{id}", "coordinacion", "profesional_etf"), reglasFamilia()[0])
	mux.declarar(handler, lectura("/api/coordinacion/expedientes/{id}", "coordinacion"))
}

func errorAceptacion(w http.ResponseWriter, err error) {
	var conflicto entrada.Conflicto
	if errors.As(err, &conflicto) {
		responderJSON(w, http.StatusConflict, map[string]any{"error": map[string]any{"clave": "error.conflicto_version", "parametros": map[string]any{"version_actual": conflicto.VersionActual}}})
		return
	}
	for _, clave := range []error{entrada.ErrTraslado, entrada.ErrSeguimiento, entrada.ErrCierre, entrada.ErrDobleIntervencion, entrada.ErrCierreSPM, entrada.ErrFinalizacion, entrada.ErrConsentimientoCierre, entrada.ErrMotivo, entrada.ErrSubprograma, entrada.ErrOrigenEspera, entrada.ErrFechaEspera} {
		if errors.Is(err, clave) {
			responderError(w, http.StatusUnprocessableEntity, clave.Error())
			return
		}
	}
	switch {
	case errors.Is(err, entrada.ErrPeticion):
		responderError(w, http.StatusBadRequest, "expediente.aceptacion_peticion_invalida")
	case errors.Is(err, entrada.ErrEstado):
		responderError(w, http.StatusConflict, "error.estado_no_permite")
	case errors.Is(err, entrada.ErrIdempotencia):
		responderError(w, http.StatusUnprocessableEntity, "error.idempotencia_reutilizada")
	case errors.Is(err, entrada.ErrObligatorios):
		responderError(w, http.StatusUnprocessableEntity, "expediente.aceptacion_incompleta")
	case errors.Is(err, entrada.ErrProfesional):
		responderError(w, http.StatusUnprocessableEntity, "expediente.profesional_invalido")
	case errors.Is(err, entrada.ErrSexo):
		responderError(w, http.StatusUnprocessableEntity, "expediente.sexo_obligatorio")
	case errors.Is(err, entrada.ErrClave):
		responderError(w, http.StatusBadRequest, "expediente.idempotencia_obligatoria")
	case errors.Is(err, entrada.ErrNumeracion):
		responderError(w, http.StatusConflict, "expediente.numeracion_agotada")
	default:
		errorDerivacion(w, err)
	}
}
