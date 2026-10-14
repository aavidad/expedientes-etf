package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	actividadApp "github.com/aavidad/expedientes-etf/internal/aplicacion/actividad"
	dominioAuditoria "github.com/aavidad/expedientes-etf/internal/dominio/auditoria"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	salidaActividad "github.com/aavidad/expedientes-etf/internal/puertos/salida/actividad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
)

type moduloPrueba []Ruta

func (m moduloPrueba) Rutas() []Ruta { return m }

func TestRegistroRechazaDeclaracionesInvalidas(t *testing.T) {
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	valida := Ruta{ReglaRuta: lectura("/api/capacidad", "administracion"), Handler: handler}
	for _, caso := range []struct {
		nombre string
		rutas  []Ruta
	}{
		{"vacío", nil},
		{"duplicado", []Ruta{valida, valida}},
		{"sin métodos", []Ruta{{ReglaRuta: reglaRuta{Patron: "/api/capacidad", Perfiles: []string{"administracion"}}, Handler: handler}}},
		{"método vacío", []Ruta{{ReglaRuta: reglaRuta{Patron: "/api/capacidad", Metodos: []string{""}, Publica: true}, Handler: handler}}},
		{"sin permiso", []Ruta{{ReglaRuta: reglaRuta{Patron: "/api/capacidad", Metodos: []string{"GET"}}, Handler: handler}}},
		{"permiso ambiguo", []Ruta{{ReglaRuta: reglaRuta{Patron: "/api/capacidad", Metodos: []string{"GET"}, Publica: true, Perfiles: []string{"consulta"}}, Handler: handler}}},
		{"sin handler", []Ruta{{ReglaRuta: lectura("/api/capacidad", "administracion")}}},
		{"sin csrf", []Ruta{{ReglaRuta: reglaRuta{Patron: "/api/capacidad", Metodos: []string{"POST"}, Perfiles: []string{"administracion"}}, Handler: handler}}},
		{"perfil inexistente", []Ruta{{ReglaRuta: lectura("/api/capacidad", "equipo"), Handler: handler}}},
		{"patrones equivalentes", []Ruta{valida, {ReglaRuta: lectura("/api/{id}", "administracion"), Handler: handler}, {ReglaRuta: lectura("/api/{otro}", "administracion"), Handler: handler}}},
		{"spa ausente", []Ruta{{ReglaRuta: reglaRuta{Patron: "/api/capacidad", Metodos: []string{"GET"}, Publica: true, SPA: "/capacidad"}, Handler: handler}}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := ValidarRutas(caso.rutas); err == nil {
				t.Fatal("registro inválido aceptado")
			}
			if _, err := NuevoValidado(nil, nil, "", "", nil, nil, nil, Opciones{Modulos: []Modulo{moduloPrueba(caso.rutas)}}); err == nil {
				t.Fatal("arranque inválido aceptado")
			}
		})
	}
	if err := ValidarRutas([]Ruta{valida, {ReglaRuta: escritura("/api/capacidad", []string{"POST"}, "administracion"), Handler: handler}}); err != nil {
		t.Fatal(err)
	}
}

func TestModuloNuevoSeComponeSinModificarServidor(t *testing.T) {
	llamadas := 0
	ruta := Ruta{ReglaRuta: lectura("/api/capacidad", "administracion"), Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { llamadas++; w.WriteHeader(204) })}
	h, err := NuevoValidado(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: sesionAdministracionPrueba(), Modulos: []Modulo{moduloPrueba{ruta, pantalla("/capacidad", "usuarios.titulo", "administracion")}}})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/api/capacidad", nil)
	r.AddCookie(cookieAdministracionPrueba())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 || llamadas != 1 {
		t.Fatalf("estado=%d llamadas=%d", w.Code, llamadas)
	}
	r = httptest.NewRequest("GET", "/api/sesion", nil)
	r.AddCookie(cookieAdministracionPrueba())
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"ruta":"/capacidad"`) || strings.Contains(w.Body.String(), `/admin/usuarios`) {
		t.Fatalf("pantallas ajenas al registro: %s", w.Body.String())
	}
}

// El frontend consume este contrato del registro real, sin copiar la tabla de rutas.
func TestContratoSPA(t *testing.T) {
	var pantallas []reglaRuta
	for _, ruta := range tablaRutas() {
		if ruta.Pantalla {
			pantallas = append(pantallas, ruta)
		}
	}
	datos, err := json.Marshal(pantallas)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("INT5_SPA=%s", datos)
}

type transaccionesRegistroPrueba struct{}

func (transaccionesRegistroPrueba) Ejecutar(ctx context.Context, _ plataforma.Identidad, fn func(context.Context) error) error {
	return fn(ctx)
}

type auditoriaRegistroPrueba struct{}

func (auditoriaRegistroPrueba) Confirmar(context.Context, dominioAuditoria.Datos) (dominioAuditoria.Evento, error) {
	return dominioAuditoria.Evento{}, nil
}

func TestRegistroNegativosIndependientes(t *testing.T) {
	for _, caso := range []struct {
		nombre, metodo, ruta string
		perfil               identidad.Perfil
		cookie, csrf         bool
		estado               int
	}{
		{"anónimo", "GET", "/api/admin/usuarios", identidad.Administracion, false, false, 401},
		{"perfil incorrecto", "GET", "/api/admin/usuarios", identidad.Consulta, true, false, 403},
		{"equipo ajeno", "GET", "/api/informes?anio=2026&etf=44444444-4444-4444-8444-444444444444", identidad.ProfesionalETF, true, false, 403},
		{"método prohibido", "PATCH", "/api/admin/usuarios", identidad.Administracion, true, true, 405},
		{"sin csrf", "PUT", "/api/admin/usuarios/33333333-3333-4333-8333-333333333333/perfiles", identidad.Administracion, true, false, 403},
		{"ruta desconocida", "GET", "/api/capacidad-desconocida", identidad.Administracion, true, true, 403},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			cfg := sesionAdministracionPrueba()
			acceso := accesoPrueba(caso.perfil)
			acceso.Usuario.EquipoID = "22222222-2222-4222-8222-222222222222"
			cfg.Servicio = &sesionAutorizacionPrueba{acceso: acceso}
			usuarios := &usuariosHTTPPrueba{}
			// Repositorio sin implementación: una consulta a datos ajenos haría fallar
			// la prueba. El servicio real debe denegar antes de consultar persistencia.
			repo := struct{ salidaActividad.Repositorio }{}
			actividad := actividadApp.Nuevo(repo, transaccionesRegistroPrueba{}, auditoriaRegistroPrueba{}, time.Now)
			h := Nuevo(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Usuarios: ConfiguracionUsuarios{Servicio: usuarios}, Actividad: ConfiguracionActividad{Servicio: actividad}})
			r := httptest.NewRequest(caso.metodo, "https://etf.interno"+caso.ruta, nil)
			if caso.cookie {
				r.AddCookie(cookieAdministracionPrueba())
			}
			if caso.csrf {
				r.Header.Set("Origin", cfg.Origen)
				r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(cfg).testigo(cookieAdministracionPrueba().Value))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || usuarios.llamadas != 0 {
				t.Fatalf("estado=%d llamadas=%d respuesta=%s", w.Code, usuarios.llamadas, w.Body.String())
			}
		})
	}
}
