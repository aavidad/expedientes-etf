package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	identidadEntrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/catalogo"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

type servicioSMIPrueba struct {
	actor     entrada.Actor
	conflicto bool
	llamadas  int
}

func (s *servicioSMIPrueba) Consultar(_ context.Context, actor entrada.Actor) ([]entrada.SMI, error) {
	s.actor = actor
	if !tieneCoordinacion(actor) {
		return nil, catalogo.ErrPermiso
	}
	return []entrada.SMI{{Anio: 2028, Version: 0}}, nil
}
func (s *servicioSMIPrueba) Guardar(_ context.Context, actor entrada.Actor, c entrada.CambioSMI) (entrada.SMI, error) {
	s.actor = actor
	s.llamadas++
	if !tieneCoordinacion(actor) {
		return entrada.SMI{}, catalogo.ErrPermiso
	}
	if s.conflicto {
		return entrada.SMI{}, &entrada.Conflicto{Actual: 1, Recurso: "smi"}
	}
	return entrada.SMI{}, nil
}
func tieneCoordinacion(actor entrada.Actor) bool {
	for _, perfil := range actor.Perfiles {
		if perfil == "coordinacion" {
			return true
		}
	}
	return false
}

type sesionesSMIPrueba struct{ acceso identidadEntrada.Acceso }

func (s sesionesSMIPrueba) AdmitirPeticion(context.Context) error { return nil }
func (s sesionesSMIPrueba) Entrar(context.Context, identidadEntrada.Inicio) (identidadEntrada.SesionAbierta, error) {
	return identidadEntrada.SesionAbierta{}, errors.New("identidad.no_disponible")
}
func (s sesionesSMIPrueba) Consultar(context.Context, string) (identidadEntrada.Acceso, error) {
	return s.acceso, nil
}
func (sesionesSMIPrueba) RegistrarRechazo(context.Context, string, string) error { return nil }
func (sesionesSMIPrueba) Salir(context.Context, string) error                    { return nil }

func configuracionSesionSMI(perfiles ...identidad.Perfil) (*sesionHTTP, string) {
	valor := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	usuario := identidad.Usuario{ID: "00000000-0000-4000-8000-000000000001", Estado: identidad.Activo, Perfiles: perfiles}
	servicio := sesionesSMIPrueba{acceso: identidadEntrada.Acceso{Usuario: usuario, Estado: identidad.Activo}}
	return &sesionHTTP{servicio: servicio, origen: "https://etf.test", clave: bytes.Repeat([]byte{2}, 32)}, valor
}
func peticionSMI(t *testing.T, metodo, cuerpo string, sesion *sesionHTTP, token string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(metodo, "https://etf.test/api/coordinacion/smi", strings.NewReader(cuerpo))
	req.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
	if metodo == http.MethodPut {
		req.Header.Set("Origin", sesion.origen)
		req.Header.Set("X-CSRF-Token", sesion.testigo(token))
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestSMIRechazaImportesInvalidos(t *testing.T) {
	for _, importe := range []string{"950.001", "-950.00"} {
		servicio := &servicioSMIPrueba{}
		mux := nuevoMuxHTTP()
		sesion, token := configuracionSesionSMI(identidad.Coordinacion)
		registrarSMI(mux, ConfiguracionSMI{Servicio: servicio})
		req := peticionSMI(t, http.MethodPut, `{"anio":2028,"importe_mensual":"`+importe+`","version_esperada":0}`, sesion, token)
		res := httptest.NewRecorder()
		sesion.autorizar(mux).ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest || servicio.llamadas != 0 {
			t.Fatalf("importe=%s status=%d llamadas=%d cuerpo=%s", importe, res.Code, servicio.llamadas, res.Body.String())
		}
	}
}

func TestSMIConflicto409(t *testing.T) {
	servicio := &servicioSMIPrueba{conflicto: true}
	mux := nuevoMuxHTTP()
	sesion, token := configuracionSesionSMI(identidad.Coordinacion)
	registrarSMI(mux, ConfiguracionSMI{Servicio: servicio})
	req := peticionSMI(t, http.MethodPut, `{"anio":2028,"importe_mensual":"950.00","version_esperada":0}`, sesion, token)
	res := httptest.NewRecorder()
	sesion.autorizar(mux).ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d cuerpo=%s", res.Code, res.Body.String())
	}
}

func TestSMINiegaProfesionalETF(t *testing.T) {
	servicio := &servicioSMIPrueba{}
	mux := nuevoMuxHTTP()
	sesion, token := configuracionSesionSMI(identidad.ProfesionalETF)
	registrarSMI(mux, ConfiguracionSMI{Servicio: servicio})
	req := httptest.NewRequest(http.MethodGet, "https://etf.test/api/coordinacion/smi", nil)
	req.AddCookie(&http.Cookie{Name: cookieSesion, Value: token})
	res := httptest.NewRecorder()
	sesion.autorizar(mux).ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d cuerpo=%s", res.Code, res.Body.String())
	}
}
