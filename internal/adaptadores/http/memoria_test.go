package http

import (
	"bytes"
	"context"
	"encoding/base64"
	documento "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	estadistica "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

type memoriaWordPrueba struct {
	actor              documento.Actor
	llamadas, rechazos int
}

func (s *memoriaWordPrueba) EmitirMemoria(_ context.Context, a documento.Actor, p documento.SolicitudMemoria) (documento.ResultadoMemoria, error) {
	s.actor = a
	s.llamadas++
	if err := p.Validar(); err != nil {
		return documento.ResultadoMemoria{}, err
	}
	return documento.ResultadoMemoria{Contenido: []byte("PK-memoria"), Emision: d.EmisionMemoria{ID: "33333333-3333-4333-8333-333333333333", Plantilla: d.Plantilla{Version: "1.0.0"}}}, nil
}
func (s *memoriaWordPrueba) RechazarMemoria(context.Context, documento.Actor) error {
	s.rechazos++
	return nil
}

// Los caminos se fijan desde el requisito de privacidad, no desde sus perfiles
// declarados: ampliar por error un permiso debe romper esta prueba.
func TestConsultaNoAccedeADetallesNiDesglose(t *testing.T) {
	const id = "33333333-3333-4333-8333-333333333333"
	for _, caso := range []struct {
		ruta   string
		estado int
	}{
		{"/api/expedientes/" + id, 403},
		{"/api/expedientes/" + id + "/familia", 403},
		{"/api/expedientes/" + id + "/familia/historia", 403},
		{"/api/expedientes/" + id + "/menores", 403},
		{"/api/expedientes/" + id + "/menores/historia", 403},
		{"/api/expedientes/" + id + "/intervencion", 403},
		{"/api/estadistica/memoria/composicion", 403},
		{"/api/estadistica/memoria/desglose", 403},
	} {
		t.Run(caso.ruta, func(t *testing.T) {
			cfg := sesionAdministracionPrueba()
			cfg.Servicio = &sesionAutorizacionPrueba{acceso: accesoPrueba(identidad.Consulta)}
			llamadas := 0
			h := nuevaSesionHTTP(cfg).autorizar(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				llamadas++
				w.WriteHeader(http.StatusNoContent)
			}))
			r := httptest.NewRequest("GET", "https://etf.interno"+caso.ruta+"?anio=2026&perfil=coordinacion&etf="+id+"&composicion=true", nil)
			r.AddCookie(cookieAdministracionPrueba())
			r.Header.Set("X-Perfil", "coordinacion")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || llamadas != 0 {
				t.Fatalf("estado=%d; llamadas=%d; cuerpo=%s", w.Code, llamadas, w.Body.String())
			}
		})
	}
}

type narrativasPrueba struct {
	llamadas int
	actor    estadistica.ActorMemoria
}

func (s *narrativasPrueba) LeerNarrativas(_ context.Context, a estadistica.ActorMemoria, f estadistica.FiltroMemoria) (estadistica.NarrativaMemoria, error) {
	s.llamadas++
	s.actor = a
	return estadistica.NarrativaMemoria{FiltroMemoria: f, Version: 1}, nil
}
func (s *narrativasPrueba) GuardarNarrativas(_ context.Context, a estadistica.ActorMemoria, n estadistica.NarrativaMemoria) (estadistica.NarrativaMemoria, error) {
	s.llamadas++
	s.actor = a
	if err := n.Validar(); err != nil {
		return estadistica.NarrativaMemoria{}, err
	}
	n.Version++
	return n, nil
}
func TestMemoriaHTTPSesionCSRFPermisosYDescarga(t *testing.T) {
	for _, caso := range []struct {
		nombre, metodo, ruta, cuerpo string
		perfil                       identidad.Perfil
		csrf                         bool
		estado                       int
	}{
		{"word", "POST", "/api/estadistica/memoria/word", `{"anio":2026,"idioma":"es"}`, identidad.Coordinacion, true, 201},
		{"word por GET", "GET", "/api/estadistica/memoria/word", "", identidad.Coordinacion, false, 405},
		{"word sin CSRF", "POST", "/api/estadistica/memoria/word", `{}`, identidad.Coordinacion, false, 403},
		{"consulta sin texto", "GET", "/api/estadistica/memoria/narrativas?anio=2026", "", identidad.Consulta, false, 403},
		{"consulta inyecta equipo y perfil", "GET", "/api/estadistica/memoria/narrativas?anio=2026&etf=33333333-3333-4333-8333-333333333333&perfil=coordinacion&composicion=true", "", identidad.Consulta, false, 403},
		{"consulta inyecta año repetido", "GET", "/api/estadistica/memoria/narrativas?anio=2026&anio=2025", "", identidad.Consulta, false, 403},
		{"consulta no emite Word", "POST", "/api/estadistica/memoria/word", `{"anio":2026,"idioma":"es","equipo_id":"33333333-3333-4333-8333-333333333333"}`, identidad.Consulta, true, 403},
		{"consulta no edita narrativas", "PUT", "/api/estadistica/memoria/narrativas?anio=2026", `{"version":0,"d":"Se acordaron visitas con las familias."}`, identidad.Consulta, true, 403},
		{"administración sin texto", "GET", "/api/estadistica/memoria/narrativas?anio=2026", "", identidad.Administracion, false, 403},
		{"guardar", "PUT", "/api/estadistica/memoria/narrativas?anio=2026", `{"version":1,"d":"Se acordaron visitas con las familias."}`, identidad.Coordinacion, true, 200},
		{"guardar sin CSRF", "PUT", "/api/estadistica/memoria/narrativas?anio=2026", `{}`, identidad.Coordinacion, false, 403},
		{"atribuir actor", "POST", "/api/estadistica/memoria/word", `{"anio":2026,"idioma":"es","usuario_id":"77777777-7777-4777-8777-777777777777"}`, identidad.Coordinacion, true, 400},
		{"25 líneas", "PUT", "/api/estadistica/memoria/narrativas?anio=2026", `{"d":"` + strings.Repeat(`línea\n`, 25) + `final"}`, identidad.Coordinacion, true, 400},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			word := &memoriaWordPrueba{}
			narrativas := &narrativasPrueba{}
			session := &sesionCoordinacion{perfiles: []identidad.Perfil{caso.perfil}}
			cfg := ConfiguracionSesion{Servicio: session, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
			opts := Opciones{Sesion: cfg, Modulos: []Modulo{ModuloMemoriaWord{ConfiguracionMemoriaWord{word}}, ModuloNarrativas{ConfiguracionNarrativas{narrativas}}}}
			handler, err := NuevoValidado(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, opts)
			if err != nil {
				t.Fatal(err)
			}
			token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
			r := httptest.NewRequest(caso.metodo, "https://etf.interno"+caso.ruta, strings.NewReader(caso.cuerpo))
			r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", "memoria-prueba")
			if caso.csrf {
				r.Header.Set("Origin", cfg.Origen)
				r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(cfg).testigo(token))
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != caso.estado {
				t.Fatalf("estado %d cuerpo %s", w.Code, w.Body.String())
			}
			if w.Code == 201 && (w.Body.String() != "PK-memoria" || w.Header().Get("Cache-Control") != "no-store" || word.actor.UsuarioID != "ab60479d-b778-407f-a83e-012345678901") {
				t.Fatal("descarga o actor")
			}
			if w.Code == 403 && (word.llamadas+narrativas.llamadas) != 0 {
				t.Fatal("llamada sin permiso")
			}
		})
	}
}
