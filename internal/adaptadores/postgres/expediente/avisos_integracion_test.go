package expediente_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
	adaptador "github.com/aavidad/expedientes-etf/internal/adaptadores/postgres/expediente"
	aplicacion "github.com/aavidad/expedientes-etf/internal/aplicacion/expediente"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"github.com/aavidad/expedientes-etf/internal/puertos/salida/plataforma"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

type relojAvisos struct{ fecha time.Time }

func (r relojAvisos) Ahora() time.Time { return r.fecha }

func TestAvisosHTTPReglasConsentimientoRLSYAuditoria(t *testing.T) {
	b, ctx, d, a, id := prepararIntervencion(t)
	d.Avisos = &adaptador.Repositorio{}
	d.Hitos = &adaptador.Repositorio{}
	d.Reloj = relojAvisos{time.Date(2026, 3, 6, 10, 0, 0, 0, time.UTC)}
	s := aplicacion.Nuevo(d)
	r, err := s.ConsultarAvisos(ctx, a, id)
	if err != nil {
		t.Fatal(err)
	}
	hay := func(r dominio.AvisosExpediente, codigo string) bool {
		for _, v := range r.Avisos {
			if v.Codigo == codigo {
				return true
			}
		}
		return false
	}
	if !hay(r, "consentimiento") || !hay(r, "memoria_dato_faltante") || r.VersionPlazos < 1 {
		t.Fatal(r)
	}
	if _, err := s.ConfirmarHito(ctx, a, id, "consentimiento-avisos", dominio.ConfirmacionHito{Version: 2, Tipo: "consentimiento", Fecha: "2026-03-01"}); err != nil {
		t.Fatal(err)
	}
	r, err = s.ConsultarAvisos(ctx, a, id)
	if err != nil || hay(r, "consentimiento") {
		t.Fatal(r, err)
	}
	ajeno := a
	ajeno.EquipoID = "00000000-0000-4000-8000-000000000011"
	if r, err := s.ConsultarAvisos(ctx, ajeno, id); !errors.Is(err, dominio.ErrNoEncontrada) || r.ID != "" {
		t.Fatal(r, err)
	}
	for _, actor := range []dominio.Actor{ajeno, {UsuarioID: a.UsuarioID, Perfiles: []string{"consulta"}}, {UsuarioID: a.UsuarioID, Perfiles: []string{"administracion"}}, {}} {
		err := b.Ejecutar(ctx, plataforma.Identidad{UsuarioID: actor.UsuarioID, Perfiles: actor.Perfiles, EquipoID: actor.EquipoID}, func(ctx context.Context) error {
			datos, err := (&adaptador.Repositorio{}).DatosAvisos(ctx, "", a.EquipoID, "2026-03-06")
			if err != nil {
				return err
			}
			if len(datos) != 0 {
				t.Fatal("RLS entrega datos", actor.Perfiles)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	ss := &sesionFamilia{sesionAceptacion: sesionAceptacion{a.UsuarioID}, equipo: a.EquipoID, perfil: identidad.ProfesionalETF}
	h := adaptadorHTTP.Nuevo(nil, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil,
		adaptadorHTTP.Opciones{Sesion: adaptadorHTTP.ConfiguracionSesion{Servicio: ss, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}, Expedientes: adaptadorHTTP.ConfiguracionAceptaciones{Avisos: s}})
	cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	pedir := func(metodo, ruta string, conSesion bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(metodo, "https://etf.interno"+ruta, nil).WithContext(ctx)
		if conSesion {
			req.AddCookie(&http.Cookie{Name: "__Host-etf-sesion", Value: cookie})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	ruta := "/api/expedientes/" + id + "/avisos"
	w := pedir("GET", ruta, true)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &r) != nil || r.ID != id {
		t.Fatal(w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "nombre") || strings.Contains(w.Body.String(), "telefono") || strings.Contains(w.Body.String(), "texto_otros") {
		t.Fatal("datos personales en avisos")
	}
	if w := pedir("GET", "/api/equipos/"+a.EquipoID+"/avisos/recuento", true); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := pedir("GET", "/api/equipos/"+ajeno.EquipoID+"/avisos/recuento", true); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := pedir("GET", ruta, false); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := pedir("POST", ruta, true); w.Code != 405 && w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := pedir("GET", ruta+"?equipo="+ajeno.EquipoID, true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	ss.perfil = identidad.Consulta
	if w := pedir("GET", ruta, true); w.Code != 403 {
		t.Fatal(w.Code)
	}
	ss.perfil = identidad.Administracion
	if w := pedir("GET", ruta, true); w.Code != 403 {
		t.Fatal(w.Code)
	}
	ss.perfil = identidad.Coordinacion
	if w := pedir("GET", ruta, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	d.RegistroTransaccional = registroFallido{}
	if r, err := aplicacion.Nuevo(d).ConsultarAvisos(ctx, a, id); err == nil || r.ID != "" {
		t.Fatal("lectura sin auditoría", r, err)
	}
	t.Log("AVISOS-OK HTTP, consentimiento confirmado, equipo ajeno, Consulta, Administración, anónimo y auditoría fallida")
}
