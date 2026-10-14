package expediente_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente/puertos/entrada"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestDecisionHTTPConPostgresEsperaNoAceptacionYRecuperacion(t *testing.T) {
	_, ctx, deps, a, _ := prepararDecision(t)
	d := subir(t, ctx, deps, a)
	servicio := aplicacion.Nuevo(deps)
	h := adaptadorHTTP.Nuevo(nil, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil,
		adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: sesionAceptacion{a.UsuarioID}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Derivaciones: adaptadorHTTP.ConfiguracionDerivaciones{Servicio: servicio}, Expedientes: adaptadorHTTP.ConfiguracionAceptaciones{Servicio: servicio, Decisiones: servicio}})
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	pedir := func(metodo, ruta, csrf, clave string, cuerpo []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, bytes.NewReader(cuerpo)).WithContext(ctx)
		r.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: cookie})
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "https://etf.interno")
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("Idempotency-Key", clave)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	sesion := pedir("GET", "/api/sesion", "", "", nil)
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(sesion.Body.Bytes(), &acceso); err != nil || acceso.CSRF == "" {
		t.Fatal(err)
	}
	ruta := "/api/coordinacion/derivaciones/" + d.ID
	espera := pedir("POST", ruta+"/lista-espera", acceso.CSRF, "http-espera", []byte(`{"version":1}`))
	if espera.Code != 200 {
		t.Fatal(espera.Code, espera.Body.String())
	}
	repetida := pedir("POST", ruta+"/lista-espera", acceso.CSRF, "http-espera", []byte(`{"version":1}`))
	if repetida.Code != 200 || repetida.Body.String() != espera.Body.String() {
		t.Fatal(repetida.Code)
	}
	listado := pedir("GET", "/api/coordinacion/lista-espera?limite=200", "", "", nil)
	var filas []entrada.FilaEspera
	if err := json.Unmarshal(listado.Body.Bytes(), &filas); err != nil || listado.Code != 200 {
		t.Fatal(listado.Code, err)
	}
	encontrada := false
	for _, f := range filas {
		if f.ID == d.ID {
			encontrada = true
		}
	}
	if !encontrada {
		t.Fatal("espera ausente en HTTP")
	}
	cuerpo := []byte(`{"version":2,"fecha":"2027-01-01","motivo":"El centro confirma atención por otro recurso."}`)
	rechazo := pedir("POST", ruta+"/no-aceptar", acceso.CSRF, "http-rechazo", cuerpo)
	if rechazo.Code != 200 {
		t.Fatal(rechazo.Code, rechazo.Body.String())
	}
	repetida = pedir("POST", ruta+"/no-aceptar", acceso.CSRF, "http-rechazo", cuerpo)
	if repetida.Code != 200 || repetida.Body.String() != rechazo.Body.String() {
		t.Fatal(repetida.Code)
	}
	ficha := pedir("GET", "/api/coordinacion/derivaciones/"+d.ID, "", "", nil)
	var recuperada dominio.Derivacion
	if err := json.Unmarshal(ficha.Body.Bytes(), &recuperada); err != nil || ficha.Code != 200 || recuperada.NoAceptacion == nil || recuperada.NoAceptacion.Subprograma != "RIESGO_SOCIAL" || recuperada.Version != 3 {
		t.Fatal(ficha.Code, err)
	}
	listado = pedir("GET", "/api/coordinacion/lista-espera?limite=200", "", "", nil)
	if err := json.Unmarshal(listado.Body.Bytes(), &filas); err != nil || listado.Code != 200 {
		t.Fatal(listado.Code, err)
	}
	for _, f := range filas {
		if f.ID == d.ID {
			t.Fatal("no aceptada permanece en lista")
		}
	}
	directa, err := servicio.Consultar(ctx, a, d.ID)
	if err != nil || !reflect.DeepEqual(directa, recuperada) {
		t.Fatal("HTTP no recupera decisión", err)
	}
	t.Log("HTTP real → sesión/CSRF → espera → lista auditada → no aceptación → repetición idéntica → GET recupera motivo y desaparece de espera")
}
