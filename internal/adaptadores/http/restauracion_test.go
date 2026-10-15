package http

import (
	"bytes"
	"context"
	"encoding/base64"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"net/http"
	"net/http/httptest"
	"testing"
)

type sesionRestauracionPrueba struct{ servicioSesionPrueba }

func (*sesionRestauracionPrueba) Consultar(context.Context, string) (entrada.Acceso, error) {
	return entrada.Acceso{Estado: dominio.Activo, Usuario: dominio.Usuario{ID: "admin", Estado: dominio.Activo, Perfiles: []dominio.Perfil{dominio.Administracion, dominio.Coordinacion}, RevisionRestauracion: true}}, nil
}
func TestRestauracionRestringeAdministradorConVariosPerfiles(t *testing.T) {
	servicio := &sesionRestauracionPrueba{}
	s := nuevaSesionHTTP(ConfiguracionSesion{Servicio: servicio, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)})
	for _, ruta := range []string{"/api/admin/usuarios", "/api/coordinacion/derivaciones", "/admin/usuarios", "/admin/auditoria", "/api/admin/auditoria", "/api/admin/restauracion"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "https://etf.interno"+ruta, nil)
		r.AddCookie(&http.Cookie{Name: cookieSesion, Value: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))})
		llamado := false
		s.autorizarRegla(w, r, reglaRuta{Patron: ruta, Perfiles: []string{"administracion", "coordinacion"}}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { llamado = true; w.WriteHeader(204) }))
		if ruta == "/api/admin/restauracion" {
			if !llamado || w.Code != 204 {
				t.Fatal("revisión bloqueada")
			}
		} else if llamado || w.Code != 403 {
			t.Fatalf("ruta abierta %s: %d", ruta, w.Code)
		}
	}
	w := httptest.NewRecorder()
	a, _ := servicio.Consultar(context.Background(), "")
	s.responderAcceso(w, a, "opaco")
	datos := datosSesion(t, w)
	pantallas := datos["pantallas"].([]any)
	if len(pantallas) != 1 || pantallas[0].(map[string]any)["ruta"] != "/admin/restauracion" {
		t.Fatal("menú permite otras pantallas")
	}
}
