package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	expediente "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type sesionCoordinacion struct {
	servicioSesionPrueba
	perfiles []dominio.Perfil
	caducada bool
}

func (s *sesionCoordinacion) Consultar(context.Context, string) (identidad.Acceso, error) {
	if s.caducada {
		return identidad.Acceso{}, dominio.ErrSesionNoValida
	}
	return identidad.Acceso{Estado: dominio.Activo, Usuario: dominio.Usuario{ID: "ab60479d-b778-407f-a83e-012345678901", Estado: dominio.Activo, Perfiles: s.perfiles}}, nil
}

type derivacionesPrueba struct {
	subidas, lecturas, rechazos int
	fallo                       error
	actor                       expediente.Actor
	b                           []byte
}

func (s *derivacionesPrueba) Subir(_ context.Context, a expediente.Actor, f expediente.Subida) (expediente.Derivacion, error) {
	s.subidas++
	s.actor = a
	s.b = f.Contenido
	return expediente.Derivacion{ID: "ab60479d-b778-407f-a83e-012345678902"}, s.fallo
}
func (s *derivacionesPrueba) Consultar(_ context.Context, a expediente.Actor, id string) (expediente.Derivacion, error) {
	s.lecturas++
	s.actor = a
	return expediente.Derivacion{ID: id}, s.fallo
}
func (s *derivacionesPrueba) Rechazar(context.Context, expediente.Actor, string) error {
	s.rechazos++
	return s.fallo
}

func servidorDerivacionPrueba(s *sesionCoordinacion, d *derivacionesPrueba) (http.Handler, *sesionHTTP, string) {
	c := ConfiguracionSesion{Servicio: s, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
	h := Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: c, Derivaciones: ConfiguracionDerivaciones{Servicio: d}})
	return h, nuevaSesionHTTP(c), base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
}

func pedirDerivacion(t *testing.T, h http.Handler, s *sesionHTTP, cookie, metodo, ruta string, b []byte, partes int, csrf bool) *httptest.ResponseRecorder {
	t.Helper()
	var cuerpo bytes.Buffer
	m := multipart.NewWriter(&cuerpo)
	for i := 0; i < partes; i++ {
		p, err := m.CreateFormFile("anexo", "Derivacion.docx")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, &cuerpo)
	r.Header.Set("Content-Type", m.FormDataContentType())
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: cookieSesion, Value: cookie})
	}
	if csrf {
		r.Header.Set("Origin", "https://etf.interno")
		r.Header.Set("X-CSRF-Token", s.testigo(cookie))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestDerivacionesHTTPSubirConsultarYDescartar(t *testing.T) {
	s := &sesionCoordinacion{perfiles: []dominio.Perfil{dominio.Coordinacion}}
	d := &derivacionesPrueba{}
	h, ss, cookie := servidorDerivacionPrueba(s, d)
	w := pedirDerivacion(t, h, ss, cookie, "POST", "/api/coordinacion/derivaciones", []byte("contenido"), 1, true)
	if w.Code != 200 || d.subidas != 1 || d.actor.UsuarioID != "ab60479d-b778-407f-a83e-012345678901" || w.Header().Get("Location") != "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String(), d)
	}
	if !bytes.Equal(d.b, make([]byte, len(d.b))) {
		t.Fatal("fichero retenido tras respuesta")
	}
	w = pedirDerivacion(t, h, ss, cookie, "GET", "/api/coordinacion/derivaciones/ab60479d-b778-407f-a83e-012345678902", nil, 0, false)
	if w.Code != 200 || d.lecturas != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestDerivacionesHTTPDeniegaSinSesionCSRFYPerfil(t *testing.T) {
	for _, caso := range []struct {
		nombre                 string
		perfil                 dominio.Perfil
		cookie, csrf, caducada bool
	}{
		{"sin_sesion", dominio.Coordinacion, false, true, false},
		{"caducada", dominio.Coordinacion, true, true, true},
		{"sin_csrf", dominio.Coordinacion, true, false, false},
		{"administracion", dominio.Administracion, true, true, false},
		{"consulta", dominio.Consulta, true, true, false},
		{"profesional_etf", dominio.ProfesionalETF, true, true, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s := &sesionCoordinacion{perfiles: []dominio.Perfil{caso.perfil}, caducada: caso.caducada}
			d := &derivacionesPrueba{}
			h, ss, cookie := servidorDerivacionPrueba(s, d)
			if !caso.cookie {
				cookie = ""
			}
			w := pedirDerivacion(t, h, ss, cookie, "POST", "/api/coordinacion/derivaciones", nil, 1, caso.csrf)
			esperado := 403
			if !caso.cookie || caso.caducada {
				esperado = 401
			}
			rechazosSesion := 1
			if caso.caducada {
				rechazosSesion = 0
			}
			if w.Code != esperado || d.subidas != 0 || d.rechazos != 0 || s.rechazos != rechazosSesion {
				t.Fatal(w.Code, w.Body.String(), d)
			}
		})
	}
}

func TestDerivacionesHTTPMultipartYTamano(t *testing.T) {
	for _, caso := range []struct {
		nombre                 string
		partes, tamano, estado int
	}{
		{"sin_fichero", 0, 0, 400}, {"dos_ficheros", 2, 1, 400}, {"demasiado", 1, expediente.MaxBytes + 1, 413},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s := &sesionCoordinacion{perfiles: []dominio.Perfil{dominio.Coordinacion}}
			d := &derivacionesPrueba{}
			h, ss, cookie := servidorDerivacionPrueba(s, d)
			w := pedirDerivacion(t, h, ss, cookie, "POST", "/api/coordinacion/derivaciones", make([]byte, caso.tamano), caso.partes, true)
			if w.Code != caso.estado || d.subidas != 0 || d.rechazos != 1 {
				t.Fatal(w.Code, w.Body.String(), d)
			}
		})
	}
}

func (s *derivacionesPrueba) ConfirmarDerivacion(context.Context, expediente.Actor, string, expediente.Confirmacion) (expediente.ReciboDerivacion, error) {
	return expediente.ReciboDerivacion{ID: "ab60479d-b778-407f-a83e-012345678902"}, s.fallo
}
func (s *derivacionesPrueba) Descartar(context.Context, expediente.Actor, string) error {
	return s.fallo
}
func (s *derivacionesPrueba) Etiquetas(context.Context, expediente.Actor, string) (map[string]string, error) {
	return map[string]string{}, s.fallo
}
