package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

type bandejaPrueba struct {
	llamadas int
	actor    entrada.Actor
	filtro   entrada.FiltroBandeja
}

func (b *bandejaPrueba) ConsultarBandeja(_ context.Context, a entrada.Actor, p entrada.FiltroBandeja) (entrada.ResultadoBandeja, error) {
	if err := p.Validar(); err != nil {
		return entrada.ResultadoBandeja{}, err
	}
	b.llamadas++
	b.actor = a
	b.filtro = p
	return entrada.ResultadoBandeja{}, nil
}
func TestBandejaHTTPAutorizacionValidacionYCSRF(t *testing.T) {
	b := &bandejaPrueba{}
	s := &sesionAutorizacionPrueba{acceso: accesoPrueba(identidad.Coordinacion)}
	cfg := sesionAdministracionPrueba()
	cfg.Servicio = s
	h := Nuevo(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Bandeja: b})
	pedir := func(metodo, ruta, cuerpo string, cookie bool, csrf string) int {
		r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, strings.NewReader(cuerpo))
		if cookie {
			r.AddCookie(cookieAdministracionPrueba())
		}
		r.Header.Set("Origin", "https://etf.interno")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Token", csrf)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if codigo := pedir("GET", "/api/coordinacion/bandeja", "", false, ""); codigo != 401 {
		t.Fatal(codigo)
	}
	for _, perfil := range []identidad.Perfil{identidad.Administracion, identidad.Consulta, identidad.ProfesionalETF} {
		s.acceso = accesoPrueba(perfil)
		if codigo := pedir("GET", "/api/coordinacion/bandeja?perfil=coordinacion", "", true, ""); codigo != 403 {
			t.Fatal(perfil, codigo)
		}
	}
	s.acceso = accesoPrueba(identidad.Coordinacion)
	if codigo := pedir("GET", "/api/coordinacion/bandeja?limite=20", "", true, ""); codigo != 200 || b.llamadas != 1 {
		t.Fatal(codigo, b.llamadas)
	}
	if b.actor.UsuarioID != s.acceso.Usuario.ID {
		t.Fatal("actor fuera de sesión")
	}
	for _, q := range []string{"limite=0", "limite=999", "estado=desconocido", "desde=2026-02-30", "limite=20&limite=30", "busqueda=Ortega", "equipo_id=ajeno", "desplazamiento=-1"} {
		if codigo := pedir("GET", "/api/coordinacion/bandeja?"+q, "", true, ""); codigo != 400 {
			t.Fatal(q, codigo)
		}
	}
	if codigo := pedir("POST", "/api/coordinacion/bandeja/buscar", `{"busqueda":"Ortega","limite":20}`, true, ""); codigo != 403 {
		t.Fatal("sin CSRF", codigo)
	}
	if codigo := pedir("DELETE", "/api/coordinacion/bandeja", "", true, ""); codigo != 405 {
		t.Fatal("método", codigo)
	}
	if b.llamadas != 1 {
		t.Fatal("se llamó servicio sin permiso", b.llamadas)
	}
}
