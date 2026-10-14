package http

import (
	"bytes"
	"context"
	"encoding/base64"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"testing/fstest"
)

type decisionesPrueba struct {
	llamadas int
	actor    entrada.Actor
}

func (s *decisionesPrueba) NoAceptar(_ context.Context, a entrada.Actor, id, clave string, p entrada.NoAceptacion) (entrada.ResultadoDecision, error) {
	s.llamadas++
	s.actor = a
	return entrada.ResultadoDecision{ID: id, Estado: "no_aceptada", Version: 2, Fecha: p.Fecha}, nil
}
func (s *decisionesPrueba) DejarEnEspera(_ context.Context, a entrada.Actor, id, clave string, p entrada.Espera) (entrada.ResultadoDecision, error) {
	s.llamadas++
	s.actor = a
	return entrada.ResultadoDecision{ID: id, Estado: "lista_espera", Version: 2, Fecha: "2027-01-01"}, nil
}
func (s *decisionesPrueba) ListarEspera(_ context.Context, a entrada.Actor, p entrada.FiltroEspera) ([]entrada.FilaEspera, error) {
	s.llamadas++
	s.actor = a
	return []entrada.FilaEspera{}, nil
}
func (*decisionesPrueba) RechazarDecision(context.Context, entrada.Actor, string) error { return nil }
func servidorDecisiones(perfil identidad.Perfil) (http.Handler, *sesionHTTP, string, *decisionesPrueba) {
	c := ConfiguracionSesion{Servicio: &sesionCoordinacion{perfiles: []identidad.Perfil{perfil}}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
	servicio := &decisionesPrueba{}
	h := Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: c, Expedientes: ConfiguracionAceptaciones{Decisiones: servicio}})
	return h, nuevaSesionHTTP(c), base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32)), servicio
}
func TestDecisionHTTPRutasSesionCSRFYCuerpos(t *testing.T) {
	for _, c := range []struct{ operacion, cuerpo string }{
		{"no-aceptar", `{"version":1,"fecha":"2027-01-01","motivo":"La familia recibe atención de otro recurso."}`},
		{"lista-espera", `{"version":1}`},
	} {
		h, s, cookie, servicio := servidorDecisiones(identidad.Coordinacion)
		ruta := "/api/coordinacion/derivaciones/" + idAceptacion + "/" + c.operacion
		w := peticionAceptacion(h, s, cookie, "POST", ruta, c.cuerpo, "decision", true)
		if w.Code != 200 || servicio.llamadas != 1 || servicio.actor.UsuarioID == "" {
			t.Fatal(w.Code, w.Body.String())
		}
		w = peticionAceptacion(h, s, cookie, "POST", ruta, c.cuerpo, "decision", false)
		if w.Code != 403 || servicio.llamadas != 1 {
			t.Fatal(w.Code, w.Body.String())
		}
		for _, cuerpo := range []string{`{"version":1,"usuario_id":"ajeno"}`, `{"version":1} {}`, `{"version":1,"motivo":"texto"}`} {
			// La no aceptación admite motivo; la espera sólo admite versión.
			if c.operacion == "no-aceptar" && cuerpo == `{"version":1,"motivo":"texto"}` {
				continue
			}
			w = peticionAceptacion(h, s, cookie, "POST", ruta, cuerpo, "decision", true)
			if w.Code != 400 || servicio.llamadas != 1 {
				t.Fatal(c.operacion, cuerpo, w.Code)
			}
		}
		for _, clave := range []string{"", "con espacio"} {
			w = peticionAceptacion(h, s, cookie, "POST", ruta, c.cuerpo, clave, true)
			if w.Code != 400 || servicio.llamadas != 1 {
				t.Fatal(w.Code)
			}
		}
		h, s, cookie, servicio = servidorDecisiones(identidad.Administracion)
		w = peticionAceptacion(h, s, cookie, "POST", ruta, c.cuerpo, "decision", true)
		if w.Code != 403 || servicio.llamadas != 0 {
			t.Fatal(w.Code)
		}
	}
	h, s, cookie, servicio := servidorDecisiones(identidad.Coordinacion)
	w := peticionAceptacion(h, s, cookie, "GET", "/api/coordinacion/lista-espera?limite=20&desplazamiento=1", "", "", false)
	if w.Code != 200 || servicio.llamadas != 1 {
		t.Fatal(w.Code)
	}
	for _, query := range []string{"limite=dos", "limite=1&limite=2", "perfil=coordinacion"} {
		w = peticionAceptacion(h, s, cookie, "GET", "/api/coordinacion/lista-espera?"+query, "", "", false)
		if w.Code != 400 || servicio.llamadas != 1 {
			t.Fatal(w.Code)
		}
	}
	w = peticionAceptacion(h, s, "", "GET", "/api/coordinacion/lista-espera", "", "", false)
	if w.Code != 401 || servicio.llamadas != 1 {
		t.Fatal(w.Code)
	}
}
