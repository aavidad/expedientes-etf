package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type sesionAutorizacionPrueba struct {
	servicioSesionPrueba
	acceso       entrada.Acceso
	actorRechazo string
}

func (s *sesionAutorizacionPrueba) Consultar(context.Context, string) (entrada.Acceso, error) {
	s.consultas++
	return s.acceso, nil
}
func (s *sesionAutorizacionPrueba) RegistrarRechazo(ctx context.Context, accion, resultado string) error {
	s.actorRechazo = entrada.AccesoActual(ctx).Usuario.ID
	return s.servicioSesionPrueba.RegistrarRechazo(ctx, accion, resultado)
}

func accesoPrueba(perfiles ...dominio.Perfil) entrada.Acceso {
	return entrada.Acceso{Estado: dominio.Activo, Usuario: dominio.Usuario{ID: "00000000-0000-4000-8000-000000000010", Estado: dominio.Activo, Perfiles: perfiles, EquipoID: "00000000-0000-4000-8000-000000000001"}}
}
func sesionAdministracionPrueba() ConfiguracionSesion {
	return ConfiguracionSesion{Servicio: &sesionAutorizacionPrueba{acceso: accesoPrueba(dominio.Administracion)}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
}
func cookieAdministracionPrueba() *http.Cookie {
	return &http.Cookie{Name: cookieSesion, Value: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))}
}

type usuariosHTTPPrueba struct {
	llamadas int
	actor    entrada.Acceso
}

func (u *usuariosHTTPPrueba) Listar(_ context.Context, a entrada.Acceso) ([]entrada.UsuarioAdministrado, error) {
	u.llamadas++
	u.actor = a
	return []entrada.UsuarioAdministrado{}, nil
}
func (u *usuariosHTTPPrueba) Cambiar(_ context.Context, a entrada.Acceso, _ string, _ entrada.CambioPerfiles) (entrada.UsuarioAdministrado, error) {
	u.llamadas++
	u.actor = a
	return entrada.UsuarioAdministrado{}, nil
}
func TestAutorizacionRealPerfilesRetiradaYCSRF(t *testing.T) {
	u := &usuariosHTTPPrueba{}
	s := &sesionAutorizacionPrueba{acceso: accesoPrueba(dominio.Administracion)}
	cfg := sesionAdministracionPrueba()
	cfg.Servicio = s
	h := Nuevo(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Usuarios: ConfiguracionUsuarios{Servicio: u}})
	pedir := func(metodo, ruta, cuerpo, csrf string, cookie bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, strings.NewReader(cuerpo))
		if cookie {
			r.AddCookie(cookieAdministracionPrueba())
		}
		r.Header.Set("Origin", "https://etf.interno")
		r.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := pedir("GET", "/api/admin/usuarios", "", "", false); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, p := range []dominio.Perfil{dominio.Coordinacion, dominio.ProfesionalETF, dominio.Consulta} {
		s.acceso = accesoPrueba(p)
		if w := pedir("GET", "/api/admin/usuarios?perfil=administracion", "", "", true); w.Code != 403 {
			t.Fatal(p, w.Code)
		}
	}
	s.acceso = entrada.Acceso{Estado: dominio.PendienteAlta}
	if w := pedir("GET", "/api/admin/usuarios", "", "", true); w.Code != 403 {
		t.Fatal(w.Code)
	}
	s.acceso = accesoPrueba(dominio.Administracion, dominio.Consulta)
	if w := pedir("GET", "/api/admin/usuarios", "", "", true); w.Code != 200 || u.llamadas != 1 {
		t.Fatal(w.Code, u.llamadas)
	}
	for _, ruta := range []string{"/api/expedientes", "/api/expedientes/00000000-0000-4000-8000-000000000001", "/api/memoria", "/api/ruta-futura"} {
		if w := pedir("GET", ruta, "", "", true); w.Code != 403 {
			t.Fatal(ruta, w.Code)
		}
	}
	ruta := "/api/admin/usuarios/00000000-0000-4000-8000-000000000011/perfiles"
	if w := pedir("PUT", ruta, `{"version":1,"estado":"activo","perfiles":[],"equipo":null}`, "", true); w.Code != 403 || u.llamadas != 1 {
		t.Fatal("CSRF", w.Code)
	}
	token := nuevaSesionHTTP(cfg).testigo(cookieAdministracionPrueba().Value)
	if w := pedir("PUT", ruta, `{"version":1,"estado":"activo","perfiles":[],"equipo":null}`, token, true); w.Code != 200 || u.actor.Usuario.ID != s.acceso.Usuario.ID {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := pedir("PUT", ruta, `{"version":1,"estado":"activo","perfiles":[],"equipo":null,"actor":"otra.persona"}`, token, true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	s.acceso = accesoPrueba(dominio.Consulta)
	if w := pedir("GET", "/api/admin/usuarios", "", "", true); w.Code != 403 || u.llamadas != 2 {
		t.Fatal("retirada", w.Code, u.llamadas)
	}
}

func autorizarPeticionPrueba(r *http.Request) {
	// El proxy termina HTTPS; el transporte interno de la aplicación es HTTP.
	r.TLS = nil
	r.AddCookie(cookieAdministracionPrueba())
	r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(sesionAdministracionPrueba()).testigo(cookieAdministracionPrueba().Value))
	if r.Header.Get("Origin") == "" {
		r.Header.Set("Origin", "https://etf.interno")
	}
}

func TestINT3AutorizacionModulosYDenegacionPorDefecto(t *testing.T) {
	t.Setenv("ETF_MODO", "desarrollo")
	rutas := []struct{ metodo, ruta string }{
		{"GET", "/api/admin/usuarios"}, {"PUT", "/api/admin/usuarios/00000000-0000-4000-8000-000000000011/perfiles"},
		{"GET", "/api/admin/catalogos/listas"}, {"POST", "/api/admin/catalogos/listas"},
		{"GET", "/api/admin/catalogos/listas/versiones"}, {"PUT", "/api/admin/catalogos/plazos"},
		{"GET", "/api/admin/reglas?ambito=tramos"}, {"GET", "/api/admin/firmantes"}, {"PUT", "/api/admin/firmantes"},
		{"GET", "/api/admin/paquetes/idioma/es"}, {"POST", "/api/admin/paquetes/idioma/es/revisar"},
		{"POST", "/api/admin/paquetes/activar"}, {"DELETE", "/api/admin/paquetes/preparaciones/abc"},
		{"GET", "/api/coordinacion/derivaciones/abc"}, {"POST", "/api/coordinacion/derivaciones"},
		{"POST", "/api/admin/futura"}, {"PUT", "/api/coordinacion/futura"},
	}
	for _, perfil := range []dominio.Perfil{dominio.ProfesionalETF, dominio.Consulta, dominio.Administracion, dominio.Coordinacion} {
		t.Run(string(perfil), func(t *testing.T) {
			cfg := sesionAdministracionPrueba()
			cfg.Servicio = &sesionAutorizacionPrueba{acceso: accesoPrueba(perfil)}
			h := Nuevo(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg})
			for _, ruta := range rutas {
				r := httptest.NewRequest(ruta.metodo, "https://etf.interno"+ruta.ruta, nil)
				autorizarPeticionPrueba(r)
				r.Header.Set("X-Perfil", "administracion")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				autorizado := strings.HasPrefix(ruta.ruta, "/api/admin/") && !strings.Contains(ruta.ruta, "futura") && perfil == dominio.Administracion || strings.HasPrefix(ruta.ruta, "/api/coordinacion/derivaciones") && perfil == dominio.Coordinacion
				if autorizado && w.Code != 503 {
					t.Errorf("%s %s: %d", ruta.metodo, ruta.ruta, w.Code)
				}
				if !autorizado && w.Code != 403 {
					t.Errorf("%s %s: %d", ruta.metodo, ruta.ruta, w.Code)
				}
			}
		})
	}
}
