package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type aceptacionesPrueba struct {
	derivacionesPrueba
	aceptaciones int
	clave, id    string
	peticion     entrada.Aceptacion
}

func (s *aceptacionesPrueba) Aceptar(_ context.Context, a entrada.Actor, id, clave string, p entrada.Aceptacion) (entrada.ResultadoAceptacion, error) {
	s.aceptaciones++
	s.actor = a
	s.clave = clave
	s.id = id
	s.peticion = p
	return entrada.ResultadoAceptacion{ID: id, Numero: "08-01-26", Estado: "aceptada", Version: 2}, s.fallo
}
func servidorAceptaciones(t *testing.T, perfil identidad.Perfil) (http.Handler, *sesionHTTP, string, *aceptacionesPrueba) {
	t.Helper()
	ss := &sesionCoordinacion{perfiles: []identidad.Perfil{perfil}}
	c := ConfiguracionSesion{Servicio: ss, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
	servicio := &aceptacionesPrueba{}
	h := Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: c, Expedientes: ConfiguracionAceptaciones{Servicio: servicio}})
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	return h, nuevaSesionHTTP(c), cookie, servicio
}
func peticionAceptacion(h http.Handler, ss *sesionHTTP, cookie, metodo, ruta, cuerpo, clave string, csrf bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, bytes.NewBufferString(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	if clave != "" {
		r.Header.Set("Idempotency-Key", clave)
	}
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: cookieSesion, Value: cookie})
	}
	if csrf {
		r.Header.Set("Origin", "https://etf.interno")
		r.Header.Set("X-CSRF-Token", ss.testigo(cookie))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

const idAceptacion = "ab60479d-b778-407f-a83e-012345678902"

func TestAceptacionHTTPMontadaSesionCSRFYGET(t *testing.T) {
	h, s, cookie, servicio := servidorAceptaciones(t, identidad.Coordinacion)
	cuerpo := `{"version":1,"fecha_acta":"2026-02-03","profesional_referencia_puesto":"TS"}`
	w := peticionAceptacion(h, s, cookie, "POST", "/api/coordinacion/derivaciones/"+idAceptacion+"/aceptar", cuerpo, "confirmar", true)
	if w.Code != 200 || servicio.aceptaciones != 1 || servicio.clave != "confirmar" || servicio.id != idAceptacion || servicio.actor.UsuarioID != "ab60479d-b778-407f-a83e-012345678901" || servicio.peticion.FechaActa != "2026-02-03" || w.Header().Get("Location") != "/api/expedientes/"+idAceptacion || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String(), servicio)
	}
	for _, ruta := range []string{"/api/expedientes/", "/api/coordinacion/expedientes/"} {
		w = peticionAceptacion(h, s, cookie, "GET", ruta+idAceptacion, "", "", false)
		if w.Code != 200 || servicio.id != idAceptacion {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestAceptacionHTTPRechazaSuplantacionCSRFYCuerposInvalidos(t *testing.T) {
	for _, c := range []struct {
		nombre, perfil, cookie, cuerpo, clave string
		csrf                                  bool
		estado                                int
	}{
		{"sin sesión", "coordinacion", "", `{}`, "clave", true, 401},
		{"administración", "administracion", "cookie", `{}`, "clave", true, 403},
		{"consulta", "consulta", "cookie", `{}`, "clave", true, 403},
		{"profesional", "profesional_etf", "cookie", `{}`, "clave", true, 403},
		{"sin CSRF", "coordinacion", "cookie", `{}`, "clave", false, 403},
		{"sin clave", "coordinacion", "cookie", `{}`, "", true, 400},
		{"suplantar actor", "coordinacion", "cookie", `{"usuario_id":"suplantado"}`, "clave", true, 400},
		{"JSON adicional", "coordinacion", "cookie", `{} {}`, "clave", true, 400},
		{"JSON truncado", "coordinacion", "cookie", `{`, "clave", true, 400},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			h, s, cookie, servicio := servidorAceptaciones(t, identidad.Perfil(c.perfil))
			if c.cookie == "" {
				cookie = ""
			}
			w := peticionAceptacion(h, s, cookie, "POST", "/api/coordinacion/derivaciones/"+idAceptacion+"/aceptar", c.cuerpo, c.clave, c.csrf)
			if w.Code != c.estado || servicio.aceptaciones != 0 {
				t.Fatal(w.Code, w.Body.String(), servicio)
			}
		})
	}
}
func TestAceptacionHTTPConflictoVersionYErrores(t *testing.T) {
	for _, c := range []struct {
		err    error
		estado int
	}{
		{entrada.Conflicto{VersionActual: 3}, 409}, {entrada.ErrEstado, 409}, {entrada.ErrIdempotencia, 422},
		{entrada.ErrSexo, 422}, {entrada.ErrProfesional, 422}, {entrada.ErrObligatorios, 422}, {entrada.ErrNumeracion, 409},
	} {
		h, s, cookie, servicio := servidorAceptaciones(t, identidad.Coordinacion)
		servicio.fallo = c.err
		w := peticionAceptacion(h, s, cookie, "POST", "/api/coordinacion/derivaciones/"+idAceptacion+"/aceptar", `{}`, "clave", true)
		if w.Code != c.estado {
			t.Fatal(w.Code, w.Body.String())
		}
		if _, ok := c.err.(entrada.Conflicto); ok {
			var d struct {
				Parametros struct {
					Version int `json:"version_actual"`
				} `json:"parametros"`
			}
			var envoltura struct {
				Error json.RawMessage `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &envoltura); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(envoltura.Error, &d); err != nil || d.Parametros.Version != 3 {
				t.Fatal(err, d)
			}
		}
	}
}

func (s *aceptacionesPrueba) RechazarAceptacion(context.Context, entrada.Actor, string) error {
	s.rechazos++
	return nil
}

// Los errores conocidos envueltos sólo entregan la clave pública del dominio.
func TestErrorAceptacionNoExponeContenidoEnvuelto(t *testing.T) {
	for _, err := range []error{entrada.ErrTraslado, entrada.ErrSeguimiento, entrada.ErrCierre} {
		t.Run(err.Error(), func(t *testing.T) {
			w := httptest.NewRecorder()
			errorAceptacion(w, fmt.Errorf("diagnostico_interno: %w", err))
			if w.Code != http.StatusUnprocessableEntity || bytes.Contains(w.Body.Bytes(), []byte("diagnostico_interno")) {
				t.Fatal(w.Code, w.Body.String())
			}
			var respuesta struct {
				Error struct {
					Clave string `json:"clave"`
				} `json:"error"`
			}
			if fallo := json.Unmarshal(w.Body.Bytes(), &respuesta); fallo != nil || respuesta.Error.Clave != err.Error() {
				t.Fatal(w.Body.String(), fallo)
			}
		})
	}
}
