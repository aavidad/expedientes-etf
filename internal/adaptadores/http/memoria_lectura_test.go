package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type lecturaMemoriaHTTPPrueba struct {
	llamadas int
	actor    entrada.ActorMemoria
	filtro   entrada.FiltroMemoria
	err      error
}

func (s *lecturaMemoriaHTTPPrueba) ConsultarMemoria(_ context.Context, a entrada.ActorMemoria, f entrada.FiltroMemoria) (entrada.CifrasMemoria, error) {
	s.llamadas++
	s.actor, s.filtro = a, f
	return entrada.CifrasMemoria{Anio: f.Anio, EquipoID: f.EquipoID, Apartados: []entrada.CifrasApartadoMemoria{}}, s.err
}

func (s *lecturaMemoriaHTTPPrueba) ConsultarEquiposMemoria(_ context.Context, a entrada.ActorMemoria) ([]entrada.EquipoCifrasMemoria, error) {
	s.llamadas++
	s.actor = a
	return []entrada.EquipoCifrasMemoria{{ID: a.EquipoID, Codigo: "02", ClaveTexto: "equipo.02"}}, s.err
}

func TestLecturaMemoriaHTTPPermisosSesionYParametros(t *testing.T) {
	const equipo = "22222222-2222-4222-8222-222222222222"
	for _, caso := range []struct {
		nombre, metodo, ruta string
		perfil               identidad.Perfil
		anonimo              bool
		err                  error
		estado, llamadas     int
	}{
		{"coordinación", "GET", "?anio=2026&etf=" + equipo, identidad.Coordinacion, false, nil, 200, 1},
		{"provincial", "GET", "?anio=2026", identidad.Coordinacion, false, nil, 200, 1},
		{"consulta", "GET", "?anio=2026", identidad.Consulta, false, nil, 200, 1},
		{"profesional", "GET", "?anio=2026", identidad.ProfesionalETF, false, nil, 200, 1},
		{"administración", "GET", "?anio=2026", identidad.Administracion, false, nil, 403, 0},
		{"anónimo", "GET", "?anio=2026", identidad.Coordinacion, true, nil, 401, 0},
		{"escritura", "POST", "?anio=2026", identidad.Coordinacion, false, nil, 405, 0},
		{"año duplicado", "GET", "?anio=2026&anio=2025", identidad.Coordinacion, false, nil, 400, 0},
		{"sin año", "GET", "", identidad.Coordinacion, false, nil, 400, 0},
		{"año mal formado", "GET", "?anio=abc", identidad.Coordinacion, false, nil, 400, 0},
		{"equipo vacío", "GET", "?anio=2026&etf=", identidad.Coordinacion, false, nil, 400, 0},
		{"equipo duplicado", "GET", "?anio=2026&etf=" + equipo + "&etf=" + equipo, identidad.Coordinacion, false, nil, 400, 0},
		{"actor cliente", "GET", "?anio=2026&usuario_id=otro", identidad.Coordinacion, false, nil, 400, 0},
		{"escape inválido", "GET", "?anio=2026&etf=%ZZ", identidad.Coordinacion, false, nil, 400, 0},
		{"denegado aplicación", "GET", "?anio=2026", identidad.Coordinacion, false, entrada.ErrPermisoMemoria, 403, 1},
		{"entrada aplicación", "GET", "?anio=2026", identidad.Coordinacion, false, entrada.ErrEntradaSPAF, 400, 1},
		{"fallo auditoría", "GET", "?anio=2026", identidad.Coordinacion, false, entrada.ErrDisponibleMemoria, 503, 1},
		{"foto ausente", "GET", "?anio=2026", identidad.Consulta, false, entrada.ErrCifrasNoPublicadasMemoria, 404, 1},
		{"head", "HEAD", "?anio=2026", identidad.Coordinacion, false, nil, 200, 1},
		{"equipos coordinación", "GET", "/equipos", identidad.Coordinacion, false, nil, 200, 1},
		{"equipos consulta", "GET", "/equipos", identidad.Consulta, false, nil, 200, 1},
		{"equipos profesional", "GET", "/equipos", identidad.ProfesionalETF, false, nil, 200, 1},
		{"equipos parámetros", "GET", "/equipos?perfil=coordinacion", identidad.Coordinacion, false, nil, 400, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			sesion, token := configuracionSesionSMI(caso.perfil)
			sesiones := sesion.servicio.(sesionesSMIPrueba)
			sesiones.acceso.Usuario.EquipoID = equipo
			if caso.anonimo {
				sesiones.acceso = identidadEntrada.Acceso{}
			}
			sesion.servicio = sesiones
			servicio := &lecturaMemoriaHTTPPrueba{err: caso.err}
			mux := nuevoMuxHTTP()
			registrarLecturaMemoria(mux, ConfiguracionLecturaMemoria{Servicio: servicio, Equipos: servicio})
			sesion.rutas = reglasDeRutas([]Modulo{ModuloLecturaMemoria{Configuracion: ConfiguracionLecturaMemoria{Servicio: servicio, Equipos: servicio}}})
			r := httptest.NewRequest(caso.metodo, "https://etf.test/api/estadistica/memoria"+caso.ruta, nil)
			if !caso.anonimo {
				r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			}
			w := httptest.NewRecorder()
			sesion.autorizar(mux).ServeHTTP(w, r)
			if w.Code != caso.estado || servicio.llamadas != caso.llamadas {
				t.Fatalf("estado=%d llamadas=%d cuerpo=%s", w.Code, servicio.llamadas, w.Body.String())
			}
			if servicio.llamadas > 0 && (servicio.actor.EquipoID != equipo || servicio.actor.UsuarioID != sesiones.acceso.Usuario.ID) {
				t.Fatal("actor ajeno a sesión", servicio.actor)
			}
			if w.Code == 200 {
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("cifras cacheables")
				}
				if caso.metodo == "HEAD" {
					if w.Body.Len() != 0 {
						t.Fatal("HEAD contiene datos")
					}
					return
				}
				var body any
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestLecturaMemoriaAliasSeleccionaTipoSegunPerfilYAmbito(t *testing.T) {
	const propio = "22222222-2222-4222-8222-222222222222"
	const ajeno = "33333333-3333-4333-8333-333333333333"
	for _, caso := range []struct {
		nombre   string
		perfiles []identidad.Perfil
		equipo   string
		numerica bool
	}{
		{"consulta provincial", []identidad.Perfil{identidad.Consulta}, "", true},
		{"consulta equipo", []identidad.Perfil{identidad.Consulta}, propio, true},
		{"mixto propio", []identidad.Perfil{identidad.Consulta, identidad.ProfesionalETF}, propio, false},
		{"mixto provincial", []identidad.Perfil{identidad.Consulta, identidad.ProfesionalETF}, "", true},
		{"mixto ajeno", []identidad.Perfil{identidad.Consulta, identidad.ProfesionalETF}, ajeno, true},
		{"coordinación consulta", []identidad.Perfil{identidad.Consulta, identidad.Coordinacion}, "", false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			sesion, token := configuracionSesionSMI(caso.perfiles[0])
			sesiones := sesion.servicio.(sesionesSMIPrueba)
			sesiones.acceso.Usuario.Perfiles = caso.perfiles
			sesiones.acceso.Usuario.EquipoID = propio
			sesion.servicio = sesiones
			completa, numerica := &lecturaMemoriaHTTPPrueba{}, &consultaHTTP{}
			config := ConfiguracionLecturaMemoria{Servicio: completa, Consulta: numerica, Equipos: completa}
			mux := nuevoMuxHTTP()
			registrarLecturaMemoria(mux, config)
			sesion.rutas = reglasDeRutas([]Modulo{ModuloLecturaMemoria{Configuracion: config}})
			ruta := "https://etf.test/api/estadistica/memoria?anio=2026"
			if caso.equipo != "" {
				ruta += "&etf=" + caso.equipo
			}
			r := httptest.NewRequest(http.MethodGet, ruta, nil)
			r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			w := httptest.NewRecorder()
			sesion.autorizar(mux).ServeHTTP(w, r)
			if w.Code != 200 {
				t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
			}
			if (numerica.llamadas == 1) != caso.numerica || completa.llamadas+numerica.llamadas != 1 {
				t.Fatalf("completa=%d numérica=%d", completa.llamadas, numerica.llamadas)
			}
			var campos map[string]json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &campos); err != nil {
				t.Fatal(err)
			}
			_, tieneFecha := campos["fecha_cifras"]
			if tieneFecha != caso.numerica {
				t.Fatal("respuesta con contrato incorrecto")
			}
		})
	}
}
