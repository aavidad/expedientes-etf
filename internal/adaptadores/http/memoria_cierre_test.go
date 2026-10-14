package http

import (
	"bytes"
	"context"
	"encoding/base64"
	e "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

type cierreMemoriaPrueba struct {
	actor              e.ActorMemoria
	llamadas, rechazos int
}

func (s *cierreMemoriaPrueba) EstadoMemoria(_ context.Context, a e.ActorMemoria, f e.FiltroMemoria) (e.EstadoMemoria, error) {
	s.actor = a
	s.llamadas++
	return e.EstadoMemoria{FiltroMemoria: f, Estado: "borrador"}, nil
}
func (s *cierreMemoriaPrueba) CerrarMemoria(_ context.Context, a e.ActorMemoria, f e.FiltroMemoria, clave string, p e.ActoMemoria) (e.ResultadoActoMemoria, error) {
	s.actor = a
	s.llamadas++
	if err := p.Validar(clave, false); err != nil {
		return e.ResultadoActoMemoria{}, err
	}
	return e.ResultadoActoMemoria{EstadoMemoria: e.EstadoMemoria{FiltroMemoria: f, Estado: "cerrada", Version: p.Version + 1}}, nil
}
func (s *cierreMemoriaPrueba) ReabrirMemoria(_ context.Context, a e.ActorMemoria, f e.FiltroMemoria, clave string, p e.ActoMemoria) (e.ResultadoActoMemoria, error) {
	s.actor = a
	s.llamadas++
	if err := p.Validar(clave, true); err != nil {
		return e.ResultadoActoMemoria{}, err
	}
	return e.ResultadoActoMemoria{EstadoMemoria: e.EstadoMemoria{FiltroMemoria: f, Estado: "borrador", Version: p.Version + 1}}, nil
}
func (s *cierreMemoriaPrueba) InstantaneaMemoria(_ context.Context, a e.ActorMemoria, _ e.FiltroMemoria, _ int) (e.RevisionMemoria, error) {
	s.actor = a
	s.llamadas++
	return e.RevisionMemoria{}, e.ErrRevisionMemoria
}
func (s *cierreMemoriaPrueba) DiscrepanciasMemoria(_ context.Context, a e.ActorMemoria, _ e.FiltroMemoria) (e.DiscrepanciasMemoria, error) {
	s.actor = a
	s.llamadas++
	return e.DiscrepanciasMemoria{Avisos: []e.DiscrepanciaMemoria{}}, nil
}
func (s *cierreMemoriaPrueba) RechazarCierreMemoria(context.Context, e.ActorMemoria) error {
	s.rechazos++
	return e.ErrActoMemoria
}
func TestCierreMemoriaHTTPSesionCSRFYContrato(t *testing.T) {
	for _, caso := range []struct {
		nombre, metodo, ruta, cuerpo string
		perfil                       identidad.Perfil
		csrf                         bool
		estado                       int
	}{
		{"estado", "GET", "estado?anio=2026", "", identidad.ProfesionalETF, false, 200},
		{"cerrar", "POST", "cerrar?anio=2026", `{"version":0}`, identidad.ProfesionalETF, true, 200},
		{"reabrir", "POST", "reabrir?anio=2026", `{"version":2,"motivo":"Revisar la composición familiar."}`, identidad.Coordinacion, true, 200},
		{"equipo no reabre", "POST", "reabrir?anio=2026", `{"version":2,"motivo":"Revisar las cifras."}`, identidad.ProfesionalETF, true, 403},
		{"consulta sin foto", "GET", "instantanea?anio=2026", "", identidad.Consulta, false, 403},
		{"consulta sin revisión anterior", "GET", "instantanea?anio=2026&revision=1&etf=33333333-3333-4333-8333-333333333333&composicion=true&perfil=coordinacion", "", identidad.Consulta, false, 403},
		{"consulta sin estado", "GET", "estado?anio=2026", "", identidad.Consulta, false, 403},
		{"consulta sin discrepancias", "GET", "discrepancias?anio=2026&desglose=true", "", identidad.Consulta, false, 403},
		{"consulta no cierra", "POST", "cerrar?anio=2026", `{"version":0}`, identidad.Consulta, true, 403},
		{"consulta no reabre", "POST", "reabrir?anio=2026", `{"version":2,"motivo":"Revisar las cifras."}`, identidad.Consulta, true, 403},
		{"administración sin estado", "GET", "estado?anio=2026", "", identidad.Administracion, false, 403},
		{"discrepancias", "GET", "discrepancias?anio=2026", "", identidad.Coordinacion, false, 200},
		{"equipo sin discrepancias", "GET", "discrepancias?anio=2026", "", identidad.ProfesionalETF, false, 403},
		{"sin CSRF", "POST", "cerrar?anio=2026", `{"version":0}`, identidad.Coordinacion, false, 403},
		{"actor inyectado", "POST", "cerrar?anio=2026", `{"version":0,"usuario_id":"33333333-3333-4333-8333-333333333333"}`, identidad.Coordinacion, true, 400},
		{"sin versión", "POST", "cerrar?anio=2026", `{}`, identidad.Coordinacion, true, 400},
		{"año duplicado", "GET", "estado?anio=2026&anio=2025", "", identidad.Coordinacion, false, 400},
		{"json doble", "POST", "cerrar?anio=2026", `{"version":0} {}`, identidad.Coordinacion, true, 400},
		{"sin motivo", "POST", "reabrir?anio=2026", `{"version":2}`, identidad.Coordinacion, true, 400},
		{"foto inexistente", "GET", "instantanea?anio=2026&revision=4", "", identidad.Coordinacion, false, 404},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio := &cierreMemoriaPrueba{}
			session := &sesionAutorizacionPrueba{acceso: accesoPrueba(caso.perfil)}
			cfg := ConfiguracionSesion{Servicio: session, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
			handler, err := NuevoValidado(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Modulos: []Modulo{ModuloCierreMemoria{Servicio: servicio}}})
			if err != nil {
				t.Fatal(err)
			}
			token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
			r := httptest.NewRequest(caso.metodo, "https://etf.interno/api/estadistica/memoria/"+caso.ruta, strings.NewReader(caso.cuerpo))
			r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", "memoria-cierre")
			if caso.csrf {
				r.Header.Set("Origin", cfg.Origen)
				r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(cfg).testigo(token))
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != caso.estado {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
			if w.Code == 403 && servicio.llamadas+servicio.rechazos != 0 {
				t.Fatal("llamó sin permiso")
			}
			if w.Code == 200 && (servicio.actor.UsuarioID != session.acceso.Usuario.ID || w.Header().Get("Cache-Control") != "no-store") {
				t.Fatal("actor o caché")
			}
		})
	}
}
