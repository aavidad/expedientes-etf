package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type hitosPrueba struct {
	llamadas int
	actor    entrada.Actor
	fallo    error
}

func (s *hitosPrueba) ConfirmarHito(_ context.Context, a entrada.Actor, id, clave string, p entrada.ConfirmacionHito) (entrada.ResultadoHito, error) {
	s.llamadas++
	s.actor = a
	return entrada.ResultadoHito{ID: id, Tipo: p.Tipo, Fecha: p.Fecha, Estado: "en_intervencion", Version: 3}, s.fallo
}
func (s *hitosPrueba) ConsultarHitos(_ context.Context, a entrada.Actor, _ string) ([]entrada.Hito, error) {
	s.llamadas++
	s.actor = a
	return []entrada.Hito{}, s.fallo
}
func (*hitosPrueba) RechazarHito(context.Context, entrada.Actor, string) error { return nil }
func TestConsentimientosHTTPPermisosCSRFVersionYRechazoDNI(t *testing.T) {
	for _, c := range []struct {
		nombre, metodo, ruta, cuerpo string
		perfil                       identidad.Perfil
		csrf, anonimo                bool
		fallo                        error
		estado                       int
	}{
		{"word", "POST", "documentos/anexo1", `{"idioma":"es"}`, identidad.ProfesionalETF, true, false, nil, 201},
		{"word2", "POST", "documentos/anexo2", `{"idioma":"en"}`, identidad.ProfesionalETF, true, false, nil, 201},
		{"dni_word", "POST", "documentos/anexo1", `{"idioma":"es","dni":"12345678Z"}`, identidad.ProfesionalETF, true, false, nil, 400},
		{"dni_hito", "POST", "hitos", `{"version":2,"tipo":"consentimiento","fecha":"2026-10-13","dni":"12345678Z"}`, identidad.ProfesionalETF, true, false, nil, 400},
		{"firma", "POST", "hitos", `{"version":2,"tipo":"consentimiento","fecha":"2026-10-13"}`, identidad.ProfesionalETF, true, false, nil, 200},
		{"version", "POST", "hitos", `{"version":2,"tipo":"consentimiento","fecha":"2026-10-13"}`, identidad.ProfesionalETF, true, false, entrada.Conflicto{VersionActual: 3}, 409},
		{"consulta", "GET", "hitos", "", identidad.Consulta, false, false, nil, 403},
		{"admin", "POST", "documentos/anexo1", `{"idioma":"es"}`, identidad.Administracion, true, false, nil, 403},
		{"anonimo", "POST", "hitos", `{}`, identidad.ProfesionalETF, true, true, nil, 401},
		{"csrf", "POST", "hitos", `{}`, identidad.ProfesionalETF, false, false, nil, 403},
		{"metodo", "GET", "documentos/anexo1", "", identidad.ProfesionalETF, false, false, nil, 405},
		{"anexo7_no_generado", "POST", "documentos/anexo7", `{}`, identidad.ProfesionalETF, true, false, nil, 403},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			ss, token := configuracionSesionSMI(c.perfil)
			sess := ss.servicio.(sesionesSMIPrueba)
			sess.acceso.Usuario.EquipoID = "66666666-6666-4666-8666-666666666666"
			ss.servicio = sess
			mux := nuevoMuxHTTP()
			hitos := &hitosPrueba{fallo: c.fallo}
			docs := &emisionesPrueba{}
			registrarHitos(mux, hitos)
			registrarDocumentos(mux, ConfiguracionDocumentos{Servicio: docs})
			req := httptest.NewRequest(c.metodo, "https://etf.test/api/expedientes/"+idAceptacion+"/"+c.ruta, strings.NewReader(c.cuerpo))
			if !c.anonimo {
				req.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "consentimiento-1")
			if c.csrf {
				req.Header.Set("Origin", ss.origen)
				req.Header.Set("X-CSRF-Token", ss.testigo(token))
			}
			w := httptest.NewRecorder()
			ss.autorizar(mux).ServeHTTP(w, req)
			if w.Code != c.estado {
				t.Fatal(w.Code, w.Body.String())
			}
			if c.estado == 200 && hitos.actor.EquipoID != sess.acceso.Usuario.EquipoID {
				t.Fatal("actor ajeno a sesión")
			}
			if c.estado == 201 && docs.actor.EquipoID != sess.acceso.Usuario.EquipoID {
				t.Fatal("actor ajeno a sesión")
			}
			if c.estado == 409 && !strings.Contains(w.Body.String(), `"version_actual":3`) {
				t.Fatal(w.Body.String())
			}
			if c.estado >= 400 && c.estado != 409 && (docs.llamadas+hitos.llamadas) != 0 {
				t.Fatal("servicio invocado")
			}
		})
	}
}

func TestConsentimientoLecturaVersionDelExpedientePorPerfil(t *testing.T) {
	for _, caso := range []struct {
		perfil identidad.Perfil
		equipo string
		estado int
	}{
		{identidad.ProfesionalETF, "66666666-6666-4666-8666-666666666666", 200},
		{identidad.ProfesionalETF, "", 403},
		{identidad.Coordinacion, "", 200},
		{identidad.Administracion, "", 403},
		{identidad.Consulta, "", 403},
	} {
		ss, token := configuracionSesionSMI(caso.perfil)
		sesion := ss.servicio.(sesionesSMIPrueba)
		sesion.acceso.Usuario.EquipoID = caso.equipo
		ss.servicio = sesion
		servicio := &aceptacionesPrueba{}
		mux := nuevoMuxHTTP()
		registrarAceptaciones(mux, ConfiguracionAceptaciones{Servicio: servicio})
		r := httptest.NewRequest(http.MethodGet, "https://etf.test/api/expedientes/"+idAceptacion, nil)
		r.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
		w := httptest.NewRecorder()
		ss.autorizar(mux).ServeHTTP(w, r)
		if w.Code != caso.estado {
			t.Fatalf("%s %s: %d %s", caso.perfil, caso.equipo, w.Code, w.Body.String())
		}
		if caso.estado == 200 && (servicio.lecturas != 1 || servicio.actor.EquipoID != caso.equipo) {
			t.Fatal("lectura sin identidad de la sesión", servicio.actor)
		}
		if caso.estado == 403 && servicio.lecturas != 0 {
			t.Fatal("lectura sin permiso")
		}
	}
}
