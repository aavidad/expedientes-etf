package expediente_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestIntervencionHTTPPostgresVersionRecuperacionYFronteras(t *testing.T) {
	_, ctx, deps, a, id := prepararIntervencion(t)
	s := aplicacion.Nuevo(deps)
	ss := &sesionFamilia{sesionAceptacion: sesionAceptacion{a.UsuarioID}, equipo: a.EquipoID, perfil: identidad.ProfesionalETF}
	h := adaptadorHTTP.Nuevo(nil, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil,
		adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: ss, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Expedientes: adaptadorHTTP.ConfiguracionAceptaciones{Servicio: s, Intervencion: s}})
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	pedir := func(metodo, ruta, csrf, clave string, cuerpo []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, bytes.NewReader(cuerpo)).WithContext(ctx)
		r.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: cookie})
		r.Header.Set("Origin", "https://etf.interno")
		r.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		if clave != "" {
			r.Header.Set("Idempotency-Key", clave)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	sesion := pedir("GET", "/api/sesion", "", "", nil)
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	if json.Unmarshal(sesion.Body.Bytes(), &acceso) != nil || sesion.Code != 200 {
		t.Fatal(sesion.Code)
	}
	ms := datosIntervencion()
	p := dominio.CambioIntervencion{Version: 2, Desde: "2026-02-03", Intervencion: &ms}
	b, _ := json.Marshal(p)
	ruta := "/api/expedientes/" + id + "/intervencion"
	primera := pedir("PUT", ruta, acceso.CSRF, "guardar-http", b)
	if primera.Code != 200 {
		t.Fatal(primera.Code, primera.Body.String())
	}
	repetida := pedir("PUT", ruta, acceso.CSRF, "guardar-http", b)
	if repetida.Code != 200 || primera.Body.String() != repetida.Body.String() {
		t.Fatal(repetida.Code, repetida.Body.String())
	}
	conflicto := pedir("PUT", ruta, acceso.CSRF, "otra-edicion", b)
	if conflicto.Code != 409 || !strings.Contains(conflicto.Body.String(), `"version_actual":3`) {
		t.Fatal(conflicto.Code, conflicto.Body.String())
	}
	ficha := pedir("GET", ruta, "", "", nil)
	if ficha.Code != 200 || !strings.Contains(ficha.Body.String(), "ABSENTISMO") {
		t.Fatal(ficha.Code, ficha.Body.String())
	}
	for _, c := range []struct {
		csrf, clave string
		cuerpo      []byte
		estado      int
	}{{"", "c", b, 403}, {acceso.CSRF, "", b, 400}, {acceso.CSRF, "c", []byte(`{"version":3,"vigente_desde":"2026-03-01","intervencion":{"dni":"12345678Z"}}`), 422}, {acceso.CSRF, "c", []byte(`{} {}`), 422}, {acceso.CSRF, "c", []byte(`{"usuario_id":"intruso"}`), 422}} {
		w := pedir("PUT", ruta, c.csrf, c.clave, c.cuerpo)
		if w.Code != c.estado {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	for _, perfil := range []identidad.Perfil{identidad.Administracion, identidad.Consulta, identidad.Coordinacion} {
		ss.perfil = perfil
		w := pedir("PUT", ruta, acceso.CSRF, "perfil-ajeno", b)
		if w.Code != 403 {
			t.Fatal(perfil, w.Code)
		}
	}
	ss.perfil = identidad.ProfesionalETF
	for _, caso := range []struct {
		metodo, ruta string
		estado       int
	}{
		{"HEAD", ruta, 200},
		{"GET", ruta + "/historia?corte=2026-02-03", 200},
		{"GET", ruta + "/historia?corte=2026-02-03&extra=1", 422},
		{"DELETE", ruta, 405},
		{"GET", ruta + "/desconocida", 403},
	} {
		w := pedir(caso.metodo, caso.ruta, acceso.CSRF, "", nil)
		if w.Code != caso.estado {
			t.Fatal(caso, w.Code, w.Body.String())
		}
	}
	anonimo := httptest.NewRequest("GET", "https://etf.interno"+ruta, nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, anonimo)
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	ss.equipo = "88888888-8888-4888-8888-888888888888"
	if w := pedir("GET", ruta, "", "", nil); w.Code != 404 {
		t.Fatal(w.Code, w.Body.String())
	}
	ss.equipo = a.EquipoID
	ss.perfil = identidad.Coordinacion
	if w := pedir("GET", ruta, "", "", nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	t.Log("HTTP real: GET/HEAD/PUT e historia; anónimo 401; equipo ajeno 404; Coordinación sólo lectura; CSRF y perfiles 403; método 405; idempotencia y 409 con versión actual; JSON estricto")
}
