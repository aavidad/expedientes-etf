package http

import (
	"bytes"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfirmacionHTTPJSONCSRFYUbicacionDeConsulta(t *testing.T) {
	s := &sesionCoordinacion{perfiles: []dominio.Perfil{dominio.Coordinacion}}
	d := &derivacionesPrueba{}
	h, ss, cookie := servidorDerivacionPrueba(s, d)
	for _, caso := range []struct {
		nombre, cuerpo string
		csrf           bool
		estado         int
	}{
		{"correcta", `{"lectura_id":"ab60479d-b778-407f-a83e-012345678902","tipo":"anexo3","campos":[],"ubicacion":{}}`, true, 201},
		{"sin_csrf", `{}`, false, 403},
		{"actor_ajeno", `{"usuario_id":"ab60479d-b778-407f-a83e-012345678903"}`, true, 400},
		{"json_sobrante", `{} {}`, true, 400},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := httptest.NewRequest("POST", "https://etf.interno/api/coordinacion/derivaciones/confirmar", bytes.NewBufferString(caso.cuerpo))
			r.AddCookie(&http.Cookie{Name: cookieSesion, Value: cookie})
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", strings.Repeat("c", 32))
			if caso.csrf {
				r.Header.Set("Origin", "https://etf.interno")
				r.Header.Set("X-CSRF-Token", ss.testigo(cookie))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado {
				t.Fatal(w.Code, w.Body.String())
			}
			if w.Code == 201 && (w.Header().Get("Location") == "" || w.Header().Get("Cache-Control") != "no-store") {
				t.Fatal(w.Header())
			}
		})
	}
}
