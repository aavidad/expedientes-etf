package http

import (
	"context"
	"encoding/json"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"net/http"
	"net/http/httptest"
	"testing"
)

type spafPrueba struct {
	llamadas    int
	actor       entrada.ActorSPAF
	filtro      entrada.FiltroSPAF
	errorFuente bool
}

func (s *spafPrueba) ConsultarOpcionesSPAF(_ context.Context, a entrada.ActorSPAF) (entrada.OpcionesSPAF, error) {
	s.llamadas++
	s.actor = a
	return entrada.OpcionesSPAF{}, nil
}

func TestSPAFHTTPProtegeOpciones(t *testing.T) {
	for _, c := range []struct {
		perfil identidad.Perfil
		estado int
	}{{identidad.Coordinacion, 200}, {identidad.ProfesionalETF, 200}, {identidad.Consulta, 403}, {identidad.Administracion, 403}} {
		sesion, token := configuracionSesionSMI(c.perfil)
		sesiones := sesion.servicio.(sesionesSMIPrueba)
		sesiones.acceso.Usuario.EquipoID = "22222222-2222-4222-8222-222222222222"
		sesion.servicio = sesiones
		s := &spafPrueba{}
		mux := nuevoMuxHTTP()
		registrarSPAF(mux, ConfiguracionSPAF{Opciones: s})
		r := httptest.NewRequest("GET", "https://etf.test/api/estadistica/spaf/opciones", nil)
		r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
		w := httptest.NewRecorder()
		sesion.autorizar(mux).ServeHTTP(w, r)
		if w.Code != c.estado {
			t.Fatal(c.perfil, w.Code, w.Body.String())
		}
		if c.estado == 200 && (s.llamadas != 1 || s.actor.UsuarioID == "" || s.actor.EquipoID == "") {
			t.Fatal("actor fuera de sesión")
		}
		if c.estado == 403 && s.llamadas != 0 {
			t.Fatal("consulta no autorizada")
		}
	}
}

func (s *spafPrueba) ConsultarSPAF(_ context.Context, a entrada.ActorSPAF, f entrada.FiltroSPAF) (entrada.TablaSPAF, error) {
	s.llamadas++
	s.actor = a
	f, err := a.AutorizarSPAF(f)
	if err != nil {
		return entrada.TablaSPAF{}, err
	}
	if s.errorFuente {
		return entrada.TablaSPAF{}, entrada.ErrDisponibleSPAF
	}
	s.filtro = f
	return entrada.TablaSPAF{FiltroSPAF: f}, nil
}

func TestSPAFHTTPPermisosValidacionYSesion(t *testing.T) {
	const propio = "22222222-2222-4222-8222-222222222222"
	const ajeno = "33333333-3333-4333-8333-333333333333"
	for _, caso := range []struct {
		nombre, metodo, consulta string
		perfil                   identidad.Perfil
		equipo                   string
		anonimo, noDisponible    bool
		estado, llamadas         int
	}{
		{"propio implícito", "GET", "anio=2026&mes=1", identidad.ProfesionalETF, propio, false, false, 200, 1},
		{"propio explícito", "GET", "anio=2026&mes=1&etf=" + propio, identidad.ProfesionalETF, propio, false, false, 200, 1},
		{"ajeno", "GET", "anio=2026&mes=1&etf=" + ajeno, identidad.ProfesionalETF, propio, false, false, 403, 1},
		{"coordinación", "GET", "anio=2026&mes=1&etf=" + ajeno, identidad.Coordinacion, "", false, false, 200, 1},
		{"sin equipo profesional", "GET", "anio=2026&mes=1", identidad.ProfesionalETF, "", false, false, 403, 0},
		{"provincial no admitida", "GET", "anio=2026&mes=1", identidad.Coordinacion, "", false, false, 400, 1},
		{"consulta", "GET", "anio=2026&mes=1&etf=" + propio, identidad.Consulta, "", false, false, 403, 0},
		{"administración", "GET", "anio=2026&mes=1&etf=" + propio, identidad.Administracion, "", false, false, 403, 0},
		{"anónimo", "GET", "anio=2026&mes=1&etf=" + propio, identidad.Coordinacion, "", true, false, 401, 0},
		{"escritura", "POST", "anio=2026&mes=1&etf=" + propio, identidad.Coordinacion, "", false, false, 405, 0},
		{"año duplicado", "GET", "anio=2026&anio=2025&mes=1", identidad.ProfesionalETF, propio, false, false, 400, 0},
		{"mes duplicado", "GET", "anio=2026&mes=1&mes=2", identidad.ProfesionalETF, propio, false, false, 400, 0},
		{"equipo duplicado", "GET", "anio=2026&mes=1&etf=" + propio + "&etf=" + ajeno, identidad.ProfesionalETF, propio, false, false, 400, 0},
		{"parámetro ajeno", "GET", "anio=2026&mes=1&usuario_id=otro", identidad.ProfesionalETF, propio, false, false, 400, 0},
		{"escape inválido", "GET", "anio=2026&mes=1&etf=%ZZ", identidad.ProfesionalETF, propio, false, false, 400, 0},
		{"equipo vacío", "GET", "anio=2026&mes=1&etf=", identidad.ProfesionalETF, propio, false, false, 400, 0},
		{"mes inválido", "GET", "anio=2026&mes=13", identidad.ProfesionalETF, propio, false, false, 400, 1},
		{"mes ausente", "GET", "anio=2026", identidad.ProfesionalETF, propio, false, false, 400, 0},
		{"adaptador pendiente", "GET", "anio=2026&mes=1", identidad.ProfesionalETF, propio, false, true, 503, 1},
		{"head", "HEAD", "anio=2026&mes=1", identidad.ProfesionalETF, propio, false, false, 200, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			sesion, token := configuracionSesionSMI(caso.perfil)
			sesiones := sesion.servicio.(sesionesSMIPrueba)
			sesiones.acceso.Usuario.EquipoID = caso.equipo
			if caso.anonimo {
				sesiones.acceso = identidadEntrada.Acceso{}
			}
			sesion.servicio = sesiones
			servicio := &spafPrueba{errorFuente: caso.noDisponible}
			mux := nuevoMuxHTTP()
			registrarSPAF(mux, ConfiguracionSPAF{Servicio: servicio})
			r := httptest.NewRequest(caso.metodo, "https://etf.test/api/estadistica/spaf?"+caso.consulta, nil)
			if !caso.anonimo {
				r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			}
			w := httptest.NewRecorder()
			sesion.autorizar(mux).ServeHTTP(w, r)
			if w.Code != caso.estado || servicio.llamadas != caso.llamadas {
				t.Fatalf("estado=%d llamadas=%d cuerpo=%s", w.Code, servicio.llamadas, w.Body.String())
			}
			if caso.llamadas > 0 && servicio.actor.EquipoID != caso.equipo {
				t.Fatal("actor ajeno a sesión", servicio.actor)
			}
			if w.Code == 200 {
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("respuesta con datos personales cacheable")
				}
				if caso.metodo == "HEAD" {
					if w.Body.Len() != 0 {
						t.Fatal("HEAD contiene datos")
					}
					return
				}
				var tabla entrada.TablaSPAF
				if err := json.Unmarshal(w.Body.Bytes(), &tabla); err != nil || tabla.EquipoID != servicio.filtro.EquipoID {
					t.Fatal(tabla, err)
				}
			}
		})
	}
}
