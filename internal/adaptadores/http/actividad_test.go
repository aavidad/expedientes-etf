package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/actividad/puertos/entrada"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type actividadPrueba struct {
	entrada.Gestor
	llamadas int
	actor    entrada.Actor
	filtro   entrada.Filtro
}

func (s *actividadPrueba) Registrar(_ context.Context, a entrada.Actor, _ entrada.Tipo, c entrada.Cambio) (entrada.Registro, error) {
	s.llamadas++
	s.actor = a
	if err := c.Validar(entrada.Informe, true); err != nil {
		return entrada.Registro{}, err
	}
	return entrada.Registro{ID: "33333333-3333-4333-8333-333333333333", EquipoID: a.EquipoID, Version: 1}, nil
}
func (s *actividadPrueba) Consultar(_ context.Context, a entrada.Actor, _ entrada.Tipo, f entrada.Filtro) ([]entrada.Registro, error) {
	s.llamadas++
	s.actor = a
	s.filtro = f
	return []entrada.Registro{}, nil
}

func (s *actividadPrueba) Equipos(_ context.Context, a entrada.Actor) ([]entrada.Equipo, error) {
	s.llamadas++
	s.actor = a
	return []entrada.Equipo{}, nil
}

func (s *actividadPrueba) Etiquetas(_ context.Context, a entrada.Actor) (map[string]map[string]string, error) {
	s.llamadas++
	s.actor = a
	return map[string]map[string]string{}, nil
}

func TestActividadAutorizaDesdeSesionYExigeCSRF(t *testing.T) {
	for _, caso := range []struct {
		nombre                       string
		perfil                       identidad.Perfil
		equipo, metodo, ruta, cuerpo string
		csrf, anonimo                bool
		estado, llamadas             int
	}{
		{"alta propia", identidad.ProfesionalETF, "22222222-2222-4222-8222-222222222222", "POST", "/api/informes", `{"fecha":"2026-04-01","organismo":"SPAF"}`, true, false, 201, 1},
		{"otros sin texto", identidad.ProfesionalETF, "22222222-2222-4222-8222-222222222222", "POST", "/api/informes", `{"fecha":"2026-04-01","organismo":"OTROS"}`, true, false, 422, 1},
		{"equipo inyectado", identidad.ProfesionalETF, "22222222-2222-4222-8222-222222222222", "POST", "/api/informes", `{"fecha":"2026-04-01","organismo":"SPAF","equipo_id":"44444444-4444-4444-8444-444444444444"}`, true, false, 400, 0},
		{"json concatenado", identidad.ProfesionalETF, "22222222-2222-4222-8222-222222222222", "POST", "/api/informes", `{} {}`, true, false, 400, 0},
		{"sin csrf", identidad.ProfesionalETF, "22222222-2222-4222-8222-222222222222", "POST", "/api/informes", `{}`, false, false, 403, 0},
		{"sin equipo", identidad.ProfesionalETF, "", "POST", "/api/informes", `{}`, true, false, 403, 0},
		{"coordinación no escribe", identidad.Coordinacion, "", "POST", "/api/informes", `{}`, true, false, 403, 0},
		{"consulta no lee", identidad.Consulta, "", "GET", "/api/comisiones?anio=2026", "", false, false, 403, 0},
		{"administración no lee", identidad.Administracion, "", "GET", "/api/informes?anio=2026", "", false, false, 403, 0},
		{"anónimo", identidad.Coordinacion, "", "GET", "/api/informes?anio=2026", "", false, true, 401, 0},
		{"método prohibido", identidad.Coordinacion, "", "PATCH", "/api/informes", "", false, false, 405, 0},
		{"ruta desconocida", identidad.Coordinacion, "", "GET", "/api/informes/33333333-3333-4333-8333-333333333333/extra", "", false, false, 403, 0},
		{"año repetido", identidad.Coordinacion, "", "GET", "/api/informes?anio=2026&anio=2027", "", false, false, 400, 0},
		{"etiquetas coordinación", identidad.Coordinacion, "", "GET", "/api/actividad/etiquetas", "", false, false, 200, 1},
		{"etiquetas consulta", identidad.Consulta, "", "GET", "/api/actividad/etiquetas", "", false, false, 403, 0},
		{"equipos coordinación", identidad.Coordinacion, "", "GET", "/api/actividad/equipos", "", false, false, 200, 1},
		{"equipos consulta", identidad.Consulta, "", "GET", "/api/actividad/equipos", "", false, false, 403, 0},
		{"equipos administración", identidad.Administracion, "", "GET", "/api/actividad/equipos", "", false, false, 403, 0},
		{"provincial", identidad.Coordinacion, "", "GET", "/api/comisiones?anio=2026", "", false, false, 200, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			sesion, token := configuracionSesionSMI(caso.perfil)
			servicioSesion := sesion.servicio.(sesionesSMIPrueba)
			servicioSesion.acceso.Usuario.EquipoID = caso.equipo
			if caso.anonimo {
				servicioSesion.acceso = identidadEntrada.Acceso{}
			}
			sesion.servicio = servicioSesion
			s := &actividadPrueba{}
			mux := nuevoMuxHTTP()
			registrarActividad(mux, ConfiguracionActividad{Servicio: s})
			r := httptest.NewRequest(caso.metodo, "https://etf.test"+caso.ruta, strings.NewReader(caso.cuerpo))
			if !caso.anonimo {
				r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			}
			r.Header.Set("Content-Type", "application/json")
			if caso.csrf {
				r.Header.Set("Origin", sesion.origen)
				r.Header.Set("X-CSRF-Token", sesion.testigo(token))
			}
			w := httptest.NewRecorder()
			sesion.autorizar(mux).ServeHTTP(w, r)
			if w.Code != caso.estado || s.llamadas != caso.llamadas {
				t.Fatalf("status=%d llamadas=%d cuerpo=%s", w.Code, s.llamadas, w.Body.String())
			}
			if caso.llamadas > 0 && s.actor.EquipoID != caso.equipo {
				t.Fatal("equipo no procede de sesión", s.actor)
			}
		})
	}
}
