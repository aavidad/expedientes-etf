package http

import (
	"encoding/json"
	"errors"
	catalogo "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"regexp"
	"time"
)

type ModuloRestauracion struct {
	Servicio  entrada.RevisarRestauracion
	Catalogos catalogo.Catalogos
}

func (m ModuloRestauracion) Rutas() []Ruta {
	return asociarSPA(rutasDeModulo(func(mux *muxHTTP) {
		mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.Servicio == nil || m.Catalogos == nil {
				responderError(w, 503, "identidad.persistencia_no_disponible")
				return
			}
			revision, err := m.Servicio.Leer(r.Context(), entrada.AccesoActual(r.Context()))
			if err != nil {
				errorRestauracionHTTP(w, err)
				return
			}
			datos, err := m.Catalogos.Datos(r.Context(), actorCatalogos(r), "asignacion_municipios", time.Now())
			if err != nil {
				errorCatalogo(w, err)
				return
			}
			for _, equipo := range datos.Equipos {
				revision.Equipos[equipo.ID] = equipo.Codigo
			}
			responderJSON(w, 200, revision)
		}), lectura("/api/admin/restauracion", "administracion"))
		cambiar := func(confirmar bool) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if m.Servicio == nil {
					responderError(w, 503, "identidad.persistencia_no_disponible")
					return
				}
				uuid := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
				if !uuid.MatchString(r.PathValue("id")) || (!confirmar && !uuid.MatchString(r.PathValue("cambio"))) {
					responderError(w, 400, "identidad.peticion_no_valida")
					return
				}
				tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil || tipo != "application/json" {
					responderError(w, 415, "identidad.peticion_no_valida")
					return
				}
				var peticion struct {
					Version  int64  `json:"version"`
					Decision string `json:"decision"`
				}
				dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
				dec.DisallowUnknownFields()
				if err = dec.Decode(&peticion); err != nil {
					responderError(w, 400, "identidad.peticion_no_valida")
					return
				}
				if err = dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
					responderError(w, 400, "identidad.peticion_no_valida")
					return
				}
				if confirmar {
					err = m.Servicio.Confirmar(r.Context(), entrada.AccesoActual(r.Context()), r.PathValue("id"), peticion.Version)
				} else {
					err = m.Servicio.Resolver(r.Context(), entrada.AccesoActual(r.Context()), r.PathValue("id"), r.PathValue("cambio"), peticion.Version, peticion.Decision)
				}
				if err != nil {
					errorRestauracionHTTP(w, err)
					return
				}
				responderJSON(w, 200, map[string]any{"estado": "ok"})
			}
		}
		mux.declarar(cambiar(false), escritura("/api/admin/restauracion/{id}/cambios/{cambio}", []string{http.MethodPost}, "administracion"))
		mux.declarar(cambiar(true), escritura("/api/admin/restauracion/{id}/confirmar", []string{http.MethodPost}, "administracion"))
	}, pantalla("/admin/restauracion", "restauracion.titulo", "administracion")), "/admin/restauracion")
}
func errorRestauracionHTTP(w http.ResponseWriter, err error) {
	switch err.Error() {
	case "restauracion.no_pendiente", "restauracion.equipo_no_disponible", "restauracion.cambios_pendientes", "restauracion.revision_propia":
		responderError(w, 409, err.Error())
	case "restauracion.acceso_bloqueado":
		responderError(w, 403, err.Error())
	default:
		errorUsuarios(w, err)
	}
}
