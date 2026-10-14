package http

import (
	"bytes"
	"context"
	"encoding/base64"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	sesion "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

type actasHTTPPrueba struct {
	lecturas, rechazos int
	actor              entrada.Actor
	equipo             string
}
type sesionActasPrueba struct{ *sesionCoordinacion }

func (s *sesionActasPrueba) Consultar(ctx context.Context, token string) (sesion.Acceso, error) {
	a, err := s.sesionCoordinacion.Consultar(ctx, token)
	a.Usuario.EquipoID = idAceptacion
	return a, err
}

func (s *actasHTTPPrueba) LeerActa(_ context.Context, a entrada.Actor, equipo, tipo string, p entrada.Subida) (entrada.ResultadoActa, error) {
	s.lecturas++
	s.actor = a
	s.equipo = equipo
	return entrada.ResultadoActa{Antivirus: "limpio"}, nil
}
func (s *actasHTTPPrueba) RechazarActa(context.Context, entrada.Actor) error {
	s.rechazos++
	return nil
}

func TestActasHTTPRegistroPerfilesCSRFYLimites(t *testing.T) {
	for _, perfil := range []identidad.Perfil{identidad.Coordinacion, identidad.ProfesionalETF, identidad.Consulta, identidad.Administracion} {
		t.Run(string(perfil), func(t *testing.T) {
			c := ConfiguracionSesion{Servicio: &sesionActasPrueba{&sesionCoordinacion{perfiles: []identidad.Perfil{perfil}}}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
			s := &actasHTTPPrueba{}
			h := Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: c, Modulos: []Modulo{ModuloCarga{Actas: s}}})
			cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
			pedir := func(csrf bool, equipo string, n int, extra bool) *httptest.ResponseRecorder {
				var b bytes.Buffer
				form := multipart.NewWriter(&b)
				p, _ := form.CreateFormFile("anexo", "acta.pdf")
				_, _ = p.Write(bytes.Repeat([]byte{'a'}, n))
				if extra {
					_ = form.WriteField("actor", "inyectado")
				}
				_ = form.Close()
				r := httptest.NewRequest(http.MethodPost, "https://etf.interno/api/carga-2026/actas/leer/aceptacion", &b)
				r.Header.Set("Content-Type", form.FormDataContentType())
				r.Header.Set("X-Equipo-Carga", equipo)
				r.AddCookie(&http.Cookie{Name: cookieSesion, Value: cookie})
				if csrf {
					r.Header.Set("Origin", c.Origen)
					r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(c).testigo(cookie))
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				return w
			}
			if w := pedir(false, idAceptacion, 1, false); w.Code != 403 {
				t.Fatal("sin CSRF", w.Code)
			}
			permitido := perfil == identidad.Coordinacion || perfil == identidad.ProfesionalETF
			w := pedir(true, idAceptacion, 1, false)
			if !permitido {
				if w.Code != 403 || s.lecturas != 0 {
					t.Fatal("perfil indebido", w.Code)
				}
				return
			}
			if w.Code != 200 || s.lecturas != 1 || s.actor.UsuarioID == "" || s.equipo != idAceptacion {
				t.Fatal("montaje o actor", w.Code, w.Body.String(), s)
			}
			if w := pedir(true, idAceptacion, entrada.MaxBytes+1, false); w.Code != 413 {
				t.Fatal("límite", w.Code)
			}
			if w := pedir(true, idAceptacion, 1, true); w.Code != 400 {
				t.Fatal("parte extra", w.Code)
			}
		})
	}
}
