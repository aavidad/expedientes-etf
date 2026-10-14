package http

import (
	"encoding/json"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/actividad/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"io"
	"mime"
	"net/http"
	"strconv"
)

type ConfiguracionActividad struct{ Servicio entrada.Gestor }

func actorActividad(r *http.Request) entrada.Actor {
	acceso := identidad.AccesoActual(r.Context())
	a := entrada.Actor{UsuarioID: acceso.Usuario.ID, EquipoID: acceso.Usuario.EquipoID}
	for _, p := range acceso.Usuario.Perfiles {
		a.Perfiles = append(a.Perfiles, string(p))
	}
	return a
}

func registrarActividad(mux *muxHTTP, c ConfiguracionActividad) {
	for _, ruta := range []struct {
		patron string
		tipo   entrada.Tipo
	}{{"/api/comisiones", entrada.Comision}, {"/api/informes", entrada.Informe}} {
		servir := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c.Servicio == nil {
				responderError(w, http.StatusServiceUnavailable, "actividad.no_disponible")
				return
			}
			a := actorActividad(r)
			switch r.Method {
			case http.MethodGet, http.MethodHead:
				q := r.URL.Query()
				if len(q["anio"]) != 1 || len(q["etf"]) > 1 || (q.Has("etf") && q.Get("etf") == "") || len(q) > 2 {
					responderError(w, http.StatusBadRequest, "actividad.datos_invalidos")
					return
				}
				anio, err := strconv.Atoi(q.Get("anio"))
				if err != nil {
					errorActividad(w, entrada.ErrDatos)
					return
				}
				filas, err := c.Servicio.Consultar(r.Context(), a, ruta.tipo, entrada.Filtro{Anio: anio, EquipoID: q.Get("etf")})
				if err != nil {
					errorActividad(w, err)
					return
				}
				responderJSON(w, http.StatusOK, filas)
			case http.MethodPost, http.MethodPut:
				var cambio entrada.Cambio
				if !leerCambioActividad(w, r, &cambio) {
					return
				}
				var fila entrada.Registro
				var err error
				if r.Method == http.MethodPost {
					fila, err = c.Servicio.Registrar(r.Context(), a, ruta.tipo, cambio)
				} else {
					if !entrada.UUIDValido(r.PathValue("id")) {
						errorActividad(w, entrada.ErrDatos)
						return
					}
					fila, err = c.Servicio.Corregir(r.Context(), a, ruta.tipo, r.PathValue("id"), cambio)
				}
				if err != nil {
					errorActividad(w, err)
					return
				}
				estado := http.StatusOK
				if r.Method == http.MethodPost {
					estado = http.StatusCreated
				}
				responderJSON(w, estado, fila)
			case http.MethodDelete:
				q := r.URL.Query()
				version, err := strconv.Atoi(q.Get("version_esperada"))
				if err != nil || len(q) != 1 || len(q["version_esperada"]) != 1 || !entrada.UUIDValido(r.PathValue("id")) {
					errorActividad(w, entrada.ErrDatos)
					return
				}
				fila, err := c.Servicio.Retirar(r.Context(), a, ruta.tipo, r.PathValue("id"), version)
				if err != nil {
					errorActividad(w, err)
					return
				}
				responderJSON(w, http.StatusOK, fila)
			default:
				responderError(w, http.StatusMethodNotAllowed, "http.metodo_no_permitido")
			}
		})
		mux.declarar(servir, lectura(ruta.patron, "profesional_etf", "coordinacion"), reglaRuta{Patron: ruta.patron, Metodos: []string{http.MethodPost}, Perfiles: []string{"profesional_etf"}, ExigeAmbito: true, CSRF: true})
		mux.declarar(servir, reglaRuta{Patron: ruta.patron + "/{id}", Metodos: []string{http.MethodPut, http.MethodDelete}, Perfiles: []string{"profesional_etf"}, ExigeAmbito: true, CSRF: true})
	}
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			responderError(w, http.StatusServiceUnavailable, "actividad.no_disponible")
			return
		}
		filas, err := c.Servicio.Organismos(r.Context(), actorActividad(r))
		if err != nil {
			errorActividad(w, err)
			return
		}
		responderJSON(w, http.StatusOK, filas)
	}), lectura("/api/actividad/organismos", "profesional_etf", "coordinacion"))
	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			responderError(w, http.StatusServiceUnavailable, "actividad.no_disponible")
			return
		}
		filas, err := c.Servicio.Equipos(r.Context(), actorActividad(r))
		if err != nil {
			errorActividad(w, err)
			return
		}
		responderJSON(w, http.StatusOK, filas)
	}), lectura("/api/actividad/equipos", "profesional_etf", "coordinacion"))

	mux.declarar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.Servicio == nil {
			responderError(w, http.StatusServiceUnavailable, "actividad.no_disponible")
			return
		}
		textos, err := c.Servicio.Etiquetas(r.Context(), actorActividad(r))
		if err != nil {
			errorActividad(w, err)
			return
		}
		responderJSON(w, http.StatusOK, textos)
	}), lectura("/api/actividad/etiquetas", "profesional_etf", "coordinacion"))

}

func leerCambioActividad(w http.ResponseWriter, r *http.Request, c *entrada.Cambio) bool {
	tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" {
		responderError(w, http.StatusUnsupportedMediaType, "actividad.datos_invalidos")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(c); err != nil {
		var limite *http.MaxBytesError
		if errors.As(err, &limite) {
			responderError(w, http.StatusRequestEntityTooLarge, "actividad.datos_invalidos")
		} else {
			errorActividad(w, entrada.ErrDatos)
		}
		return false
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		errorActividad(w, entrada.ErrDatos)
		return false
	}
	return true
}

func errorActividad(w http.ResponseWriter, err error) {
	var conflicto *entrada.Conflicto
	if errors.As(err, &conflicto) {
		responderJSON(w, http.StatusConflict, map[string]any{"error": "actividad.conflicto", "version_actual": conflicto.Actual})
		return
	}
	switch {
	case errors.Is(err, entrada.ErrPermiso):
		responderError(w, http.StatusForbidden, "actividad.acceso_denegado")
	case errors.Is(err, entrada.ErrOtros):
		responderError(w, http.StatusUnprocessableEntity, "actividad.otros_obligatorio")
	case errors.Is(err, entrada.ErrDatos):
		responderError(w, http.StatusBadRequest, "actividad.datos_invalidos")
	case errors.Is(err, entrada.ErrNoEncontrado):
		responderError(w, http.StatusNotFound, "actividad.no_encontrado")
	default:
		responderError(w, http.StatusServiceUnavailable, "actividad.no_disponible")
	}
}
