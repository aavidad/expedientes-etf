package http

import (
	"context"
	"errors"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type catalogosPrueba struct {
	llamadas int
	actor    entrada.Actor
	anio     int
	fallo    error
}

func (c *catalogosPrueba) Vigente(_ context.Context, a entrada.Actor, ambito string, _ time.Time) (entrada.Version, error) {
	c.llamadas++
	c.actor = a
	return entrada.Version{ID: "v1", Ambito: ambito}, c.fallo
}
func (c *catalogosPrueba) Memoria(ctx context.Context, a entrada.Actor, ambito string, anio int) (entrada.Version, error) {
	c.anio = anio
	return c.Vigente(ctx, a, ambito, time.Time{})
}
func (c *catalogosPrueba) Versiones(ctx context.Context, a entrada.Actor, ambito string) ([]entrada.Version, error) {
	v, err := c.Vigente(ctx, a, ambito, time.Time{})
	return []entrada.Version{v}, err
}
func (c *catalogosPrueba) Datos(ctx context.Context, a entrada.Actor, ambito string, f time.Time) (entrada.Datos, error) {
	v, err := c.Vigente(ctx, a, ambito, f)
	return entrada.Datos{Version: v}, err
}
func TestCatalogosMontadosDenieganPorDefecto(t *testing.T) {
	c := &catalogosPrueba{}
	for _, cfg := range []ConfiguracionCatalogos{{}, {Consulta: c}} {
		handler := Nuevo(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Catalogos: cfg})
		r := httptest.NewRequest("GET", "/api/admin/catalogos/listas/vigente?fecha=2026-01-01&desarrollo=1&perfil=administracion", nil)
		r.Header.Set("X-ETF-MODO", "desarrollo")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 403 || !strings.Contains(w.Body.String(), "identidad.acceso_denegado") || c.llamadas != 0 {
			t.Fatal(w.Code, w.Body.String(), c.llamadas)
		}
	}
}
func TestCatalogosSesionConsultaYValida(t *testing.T) {
	c := &catalogosPrueba{}
	handler := Nuevo(nil, nil, "", "", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Catalogos: ConfiguracionCatalogos{Consulta: c}, Sesion: sesionAdministracionPrueba()})
	for _, caso := range []struct {
		ruta   string
		estado int
	}{
		{"listas/versiones", 200}, {"listas/vigente?fecha=2026-06-30", 200}, {"listas/vigente?anio=2026", 200}, {"asignacion_municipios/datos?fecha=2026-01-01", 200},
		{"listas/vigente", 400}, {"listas/vigente?fecha=2026-02-30", 400}, {"listas/vigente?fecha=2026-01-01&anio=2026", 400}, {"listas/vigente?fecha=2026-01-01&fecha=2027-01-01", 400}, {"listas/vigente?anio=x", 400}, {"listas/vigente?anio=&fecha=2026-01-01", 400}, {"listas/datos?fecha=2026-01-01&anio=2026", 400},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/admin/catalogos/"+caso.ruta, nil)
		r.AddCookie(cookieAdministracionPrueba())
		handler.ServeHTTP(w, r)
		if w.Code != caso.estado {
			t.Fatal(caso.ruta, w.Code, w.Body.String())
		}
	}
	if c.actor.UsuarioID != "00000000-0000-4000-8000-000000000010" || c.anio != 2026 {
		t.Fatal(c.actor, c.anio)
	}
	for _, caso := range []struct {
		err    error
		estado int
		clave  string
	}{{entrada.ErrNoDisponible, 404, "catalogo.version_no_disponible"}, {entrada.ErrAmbito, 400, "catalogo.ambito_invalido"}, {errors.New("fallo.privado"), 503, "catalogo.no_disponible"}} {
		c.fallo = caso.err
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/admin/catalogos/listas/vigente?fecha=2026-01-01", nil)
		r.AddCookie(cookieAdministracionPrueba())
		handler.ServeHTTP(w, r)
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), caso.clave) || strings.Contains(w.Body.String(), "fallo.privado") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
