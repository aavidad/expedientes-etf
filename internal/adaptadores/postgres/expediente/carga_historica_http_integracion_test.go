package expediente_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	httpETF "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	app "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestCargaHistoricaHTTPPostgresSesionCSRFYRecuperacion(t *testing.T) {
	b, ctx, deps, coord, p := prepararAceptacion(t)
	deps = dependenciasHistoricas(deps)
	s := app.Nuevo(deps)
	ss := &sesionFamilia{sesionAceptacion: sesionAceptacion{coord.UsuarioID}, equipo: p.EquipoID, perfil: identidad.ProfesionalETF}
	h := httpETF.Nuevo(nil, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, httpETF.Opciones{Sesion: httpETF.ConfiguracionSesion{Servicio: ss, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Expedientes: httpETF.ConfiguracionAceptaciones{Servicio: s, CargasHistoricas: s}})
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	pedir := func(metodo, ruta, csrf, clave string, cuerpo []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, bytes.NewReader(cuerpo)).WithContext(ctx)
		r.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: cookie})
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "https://etf.interno")
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
	var acceso struct {
		CSRF string `json:"csrf"`
	}
	respuesta := pedir("GET", "/api/sesion", "", "", nil)
	if respuesta.Code != 200 || json.Unmarshal(respuesta.Body.Bytes(), &acceso) != nil {
		t.Fatal(respuesta.Code)
	}
	caso := casoHistorico(p, "")
	carga := d.CargaHistorica{ConfirmarDuplicados: true, EquipoID: p.EquipoID, Casos: []d.CasoHistorico{caso}}
	cuerpo, _ := json.Marshal(carga)
	antes := contarCargas(t, b, ctx, coord)
	respuesta = pedir("POST", "/api/carga-2026", acceso.CSRF, "", cuerpo)
	var previa d.ValidacionCargaHistorica
	if respuesta.Code != 200 || json.Unmarshal(respuesta.Body.Bytes(), &previa) != nil || !d.IDValido(previa.LoteID) {
		t.Fatal(respuesta.Code, respuesta.Body.String())
	}
	if contarCargas(t, b, ctx, coord) != antes {
		t.Fatal("vista previa persiste")
	}
	ruta := "/api/carga-2026/" + previa.LoteID + "/confirmar"
	clave := "http-" + previa.LoteID
	for _, c := range []struct {
		csrf, clave string
		cuerpo      []byte
		estado      int
	}{{"", clave, cuerpo, 403}, {acceso.CSRF, "", cuerpo, 400}, {acceso.CSRF, clave, []byte(`{"usuario_id":"intruso"}`), 400}, {acceso.CSRF, clave, []byte(`{} {}`), 400}} {
		w := pedir("POST", ruta, c.csrf, c.clave, c.cuerpo)
		if w.Code != c.estado {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	respuesta = pedir("POST", ruta, acceso.CSRF, clave, cuerpo)
	if respuesta.Code != 200 {
		t.Fatal(respuesta.Code, respuesta.Body.String())
	}
	repetida := pedir("POST", ruta, acceso.CSRF, clave, cuerpo)
	if repetida.Code != 200 || repetida.Body.String() != respuesta.Body.String() {
		t.Fatal(repetida.Code, repetida.Body.String())
	}
	var recibo d.ResultadoCargaHistorica
	_ = json.Unmarshal(respuesta.Body.Bytes(), &recibo)
	vista := pedir("GET", "/api/expedientes/"+recibo.Expedientes[0], "", "", nil)
	var ficha d.Derivacion
	if vista.Code != 200 || json.Unmarshal(vista.Body.Bytes(), &ficha) != nil || !ficha.CargadoParaRevision || ficha.FechaDerivacion != "2026-01-15" || !ficha.Lectura.LeidaEn.IsZero() {
		t.Fatal(vista.Code, vista.Body.String())
	}
	cambiado := carga
	cambiado.Casos[0].FechaDerivacion = "2026-01-16"
	otro, _ := json.Marshal(cambiado)
	respuesta = pedir("POST", ruta, acceso.CSRF, clave, otro)
	if respuesta.Code != 422 {
		t.Fatal(respuesta.Code, respuesta.Body.String())
	}
	individual := pedir("POST", "/api/expedientes/historicos", acceso.CSRF, "manual-"+previa.LoteID, cuerpo)
	recuperada := pedir("POST", "/api/expedientes/historicos", acceso.CSRF, "manual-"+previa.LoteID, cuerpo)
	if individual.Code != 200 || recuperada.Body.String() != individual.Body.String() {
		t.Fatal(individual.Code, individual.Body.String(), recuperada.Code)
	}
	ss.perfil = identidad.Administracion
	respuesta = pedir("POST", "/api/carga-2026", acceso.CSRF, "", cuerpo)
	if respuesta.Code != 403 {
		t.Fatal(respuesta.Code)
	}
	t.Log("HTTP → sesión y CSRF → vista previa revertida → confirmación PostgreSQL → recibo idéntico → ficha histórica auditada")
}
