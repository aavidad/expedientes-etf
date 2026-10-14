package expediente_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entradaIdentidad "github.com/aavidad/expedientes-etf/internal/aplicacion/identidad/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
)

// Sólo se sustituye la sesión: HTTP, aplicación, catálogo y PostgreSQL son reales.
type sesionAceptacion struct{ usuario string }

func (sesionAceptacion) AdmitirPeticion(context.Context) error { return nil }
func (sesionAceptacion) Entrar(context.Context, entradaIdentidad.Inicio) (entradaIdentidad.SesionAbierta, error) {
	return entradaIdentidad.SesionAbierta{}, identidad.ErrCredencialesInvalidas
}
func (sesionAceptacion) RegistrarRechazo(context.Context, string, string) error { return nil }
func (sesionAceptacion) Salir(context.Context, string) error                    { return nil }
func (s sesionAceptacion) Consultar(context.Context, string) (entradaIdentidad.Acceso, error) {
	return entradaIdentidad.Acceso{Estado: identidad.Activo, Usuario: identidad.Usuario{ID: s.usuario, Estado: identidad.Activo, Perfiles: []identidad.Perfil{identidad.Coordinacion}}}, nil
}

func TestAceptacionHTTPConPostgresYRecuperacion(t *testing.T) {
	_, ctx, deps, a, p := prepararAceptacion(t)
	d := subir(t, ctx, deps, a)
	p = conMenores(p, d)
	servicio := aplicacion.Nuevo(deps)
	h := adaptadorHTTP.Nuevo(nil, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil,
		adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: sesionAceptacion{a.UsuarioID}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Expedientes: adaptadorHTTP.ConfiguracionAceptaciones{Servicio: servicio}})
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	pedir := func(metodo, ruta, csrf string, cuerpo []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, bytes.NewReader(cuerpo)).WithContext(ctx)
		r.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: cookie})
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "https://etf.interno")
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		r.Header.Set("Idempotency-Key", "aceptacion-http")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	sesion := pedir("GET", "/api/sesion", "", nil)
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(sesion.Body.Bytes(), &acceso); err != nil || sesion.Code != 200 || acceso.CSRF == "" {
		t.Fatal(sesion.Code, err)
	}
	cuerpo, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	ruta := "/api/coordinacion/derivaciones/" + d.ID + "/aceptar"
	primera := pedir("POST", ruta, acceso.CSRF, cuerpo)
	if primera.Code != 200 {
		t.Fatal(primera.Code, primera.Body.String())
	}
	segunda := pedir("POST", ruta, acceso.CSRF, cuerpo)
	if segunda.Code != 200 || segunda.Body.String() != primera.Body.String() {
		t.Fatal("repetición HTTP distinta", segunda.Code)
	}
	ficha := pedir("GET", "/api/expedientes/"+d.ID, "", nil)
	var recuperada dominio.Derivacion
	if err := json.Unmarshal(ficha.Body.Bytes(), &recuperada); err != nil || ficha.Code != 200 || recuperada.Estado != dominio.Aceptada || recuperada.Aceptacion == nil || recuperada.Aceptacion.Descripcion != p.Descripcion || recuperada.Aceptacion.FechaActa != p.FechaActa {
		t.Fatal(ficha.Code, err)
	}
	t.Log("HTTP → sesión y CSRF → aceptación PostgreSQL → repetición idéntica → GET de ficha auditada")
}
