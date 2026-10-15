package http

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/auditoria/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type exportadorHTTPPrueba struct {
	llamadas int
	rechazos int
	actor    entrada.ActorExportacion
}

func (e *exportadorHTTPPrueba) Exportar(_ context.Context, a entrada.ActorExportacion, _ entrada.FiltroExportacion, _, _ string) (entrada.DescargaAuditoria, error) {
	e.llamadas++
	e.actor = a
	return entrada.DescargaAuditoria{Contenido: io.NopCloser(strings.NewReader("ZIP")), Tamano: 3, Filas: 1}, nil
}

func (e *exportadorHTTPPrueba) Rechazar(context.Context, entrada.ActorExportacion) error {
	e.rechazos++
	return nil
}

type textosAuditoriaPrueba struct{}

func (textosAuditoriaPrueba) Textos(context.Context, string) (map[string]string, error) {
	return map[string]string{"auditoria.instrucciones_verificacion": "auditoria.instrucciones_verificacion"}, nil
}
func TestExportacionAuditoriaPermisosCSRFYDescarga(t *testing.T) {
	for _, p := range []identidad.Perfil{identidad.Coordinacion, identidad.ProfesionalETF, identidad.Consulta, identidad.Administracion} {
		t.Run(string(p), func(t *testing.T) {
			servicio := &exportadorHTTPPrueba{}
			mux := nuevoMuxHTTP()
			sesion, token := configuracionSesionSMI(p)
			registrarExportacionAuditoria(mux, ConfiguracionAuditoria{Servicio: servicio, Textos: textosAuditoriaPrueba{}})
			req := httptest.NewRequest("POST", "https://etf.test/api/admin/auditoria/exportaciones", strings.NewReader(`{"desde":"2026-10-14","hasta":"2026-10-14","formato":"json","idioma":"es"}`))
			req.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
			req.Header.Set("Origin", sesion.origen)
			req.Header.Set("X-CSRF-Token", sesion.testigo(token))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			sesion.autorizar(mux).ServeHTTP(res, req)
			esperado := 403
			if p == identidad.Administracion {
				esperado = 200
			}
			if res.Code != esperado {
				t.Fatalf("status=%d cuerpo=%s", res.Code, res.Body.String())
			}
			if p != identidad.Administracion && servicio.llamadas != 0 {
				t.Fatal("ejecuta sin permiso")
			}
			if p == identidad.Administracion {
				if servicio.actor.UsuarioID == "" || res.Header().Get("Content-Type") != "application/zip" {
					t.Fatal("sin sesión o descarga")
				}
				req = httptest.NewRequest("POST", "https://etf.test/api/admin/auditoria/exportaciones", nil)
				req.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
				res = httptest.NewRecorder()
				sesion.autorizar(mux).ServeHTTP(res, req)
				if res.Code != 403 || servicio.llamadas != 1 {
					t.Fatal("sin CSRF")
				}
			}
		})
	}
}

type falloEscrituraAuditoria struct{ *httptest.ResponseRecorder }

func (f falloEscrituraAuditoria) Write([]byte) (int, error) {
	return 0, errors.New("auditoria.entrega_fallida")
}
func (f falloEscrituraAuditoria) WriteString(string) (int, error) {
	return 0, errors.New("auditoria.entrega_fallida")
}
func TestExportacionAuditoriaAnotaDescargaInterrumpida(t *testing.T) {
	servicio := &exportadorHTTPPrueba{}
	mux := nuevoMuxHTTP()
	sesion, token := configuracionSesionSMI(identidad.Administracion)
	registrarExportacionAuditoria(mux, ConfiguracionAuditoria{Servicio: servicio, Textos: textosAuditoriaPrueba{}})
	req := httptest.NewRequest("POST", "https://etf.test/api/admin/auditoria/exportaciones", strings.NewReader(`{"desde":"2026-10-14","hasta":"2026-10-14","formato":"json","idioma":"es"}`))
	req.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
	req.Header.Set("Origin", sesion.origen)
	req.Header.Set("X-CSRF-Token", sesion.testigo(token))
	req.Header.Set("Content-Type", "application/json")
	sesion.autorizar(mux).ServeHTTP(falloEscrituraAuditoria{httptest.NewRecorder()}, req)
	if servicio.llamadas != 1 || servicio.rechazos != 1 {
		t.Fatalf("exportaciones=%d fallos=%d", servicio.llamadas, servicio.rechazos)
	}
}
