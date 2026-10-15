package http

import (
	"bytes"
	"context"
	"encoding/base64"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/estadistica/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

type desgloseHTTP struct{ llamadas int }

func (s *desgloseHTTP) DesglosarMemoria(_ context.Context, a d.ActorMemoria, f d.FiltroMemoria, ap d.Apartado, _ entrada.SeleccionDesglose) (d.DetalleMemoria, error) {
	s.llamadas++
	_, err := d.AutorizarMemoria(a, f)
	return d.DetalleMemoria{Apartado: ap, Cifra: 2}, err
}
func (*desgloseHTTP) RechazarPeticionMemoria(context.Context, d.ActorMemoria) error { return nil }
func TestDesgloseHTTPConsulta403YParametrosEstrictos(t *testing.T) {
	for _, c := range []struct {
		perfil   identidad.Perfil
		consulta string
		estado   int
	}{
		{identidad.Coordinacion, "celda=0&medida=numerador", 200},
		{identidad.Consulta, "celda=0&medida=numerador", 403},
		{identidad.Administracion, "celda=0&medida=numerador", 403},
		{identidad.ProfesionalETF, "celda=0&medida=numerador&etf=33333333-3333-4333-8333-333333333333", 403},
		{identidad.Coordinacion, "celda=0&celda=1&medida=numerador", 400},
		{identidad.Coordinacion, "celda=0&medida=numerador&perfil=coordinacion", 400},
		{identidad.Coordinacion, "celda=no&medida=numerador", 400},
	} {
		s := &desgloseHTTP{}
		cfg := ConfiguracionSesion{Servicio: &sesionCoordinacion{perfiles: []identidad.Perfil{c.perfil}}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
		h, err := NuevoValidado(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Modulos: []Modulo{ModuloDesgloseMemoria{s}}})
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest("GET", "https://etf.interno/api/memoria/2026/desglose/1?"+c.consulta, nil)
		r.AddCookie(&http.Cookie{Name: cookieSesion, Value: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.estado {
			t.Fatalf("perfil %s: %d esperado %d: %s", c.perfil, w.Code, c.estado, w.Body.String())
		}
		if c.perfil == identidad.Consulta && s.llamadas != 0 {
			t.Fatal("Consulta llegó al desglose")
		}
	}
}
