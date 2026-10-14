package http

import (
	"bytes"
	"context"
	"encoding/base64"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	"github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

type plantillasHTTP struct {
	fallo         error
	actor         entrada.Actor
	llamadas      int
	publicaciones int
}

func (p *plantillasHTTP) Actual(_ context.Context, a entrada.Actor, _, _ string) (entrada.ReferenciaPlantilla, error) {
	p.actor = a
	p.llamadas++
	return entrada.ReferenciaPlantilla{Revision: 2}, p.fallo
}
func (p *plantillasHTTP) Publicar(_ context.Context, a entrada.Actor, _ entrada.PublicarPlantilla, _ []byte) (entrada.ReferenciaPlantilla, error) {
	p.actor = a
	p.llamadas++
	p.publicaciones++
	return entrada.ReferenciaPlantilla{Revision: 3}, p.fallo
}
func (*plantillasHTTP) Rechazar(context.Context, entrada.Actor) error { return nil }
func TestPlantillasHTTPAutorizacionCSRFYLimites(t *testing.T) {
	for _, caso := range []struct {
		nombre, metodo string
		perfil         identidad.Perfil
		sesion, csrf   bool
		fallo          error
		extra          string
		tamano, estado int
	}{
		{nombre: "publicar", metodo: "POST", perfil: identidad.Administracion, sesion: true, csrf: true, tamano: 2, estado: 201},
		{nombre: "actual", metodo: "GET", perfil: identidad.Administracion, sesion: true, estado: 200},
		{nombre: "anonimo", metodo: "POST", csrf: true, estado: 401},
		{nombre: "coordinacion", metodo: "POST", perfil: identidad.Coordinacion, sesion: true, csrf: true, estado: 403},
		{nombre: "consulta", metodo: "POST", perfil: identidad.Consulta, sesion: true, csrf: true, estado: 403},
		{nombre: "equipo", metodo: "POST", perfil: identidad.Perfil("equipo"), sesion: true, csrf: true, estado: 403},
		{nombre: "sin_csrf", metodo: "POST", perfil: identidad.Administracion, sesion: true, estado: 403},
		{nombre: "metodo", metodo: "PUT", perfil: identidad.Administracion, sesion: true, csrf: true, estado: 405},
		{nombre: "mayor_10_MB", metodo: "POST", perfil: identidad.Administracion, sesion: true, csrf: true, tamano: entrada.MaximoPlantilla + 1, estado: 413},
		{nombre: "exactamente_10_MB", metodo: "POST", perfil: identidad.Administracion, sesion: true, csrf: true, tamano: entrada.MaximoPlantilla, estado: 201},
		{nombre: "actor_cliente", metodo: "POST", perfil: identidad.Administracion, sesion: true, csrf: true, extra: "&usuario_id=otro", estado: 400},
		{nombre: "version_duplicada", metodo: "POST", perfil: identidad.Administracion, sesion: true, csrf: true, extra: "&version=9.0.0", estado: 400},
		{nombre: "conflicto", metodo: "POST", perfil: identidad.Administracion, sesion: true, csrf: true, fallo: &entrada.ConflictoPlantilla{VersionActual: 8}, estado: 409},
		{nombre: "virus", metodo: "POST", perfil: identidad.Administracion, sesion: true, csrf: true, fallo: entrada.ErrInfectada, estado: 422},
		{nombre: "clamd_caido", metodo: "POST", perfil: identidad.Administracion, sesion: true, csrf: true, fallo: entrada.ErrAntivirus, estado: 503},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p := &plantillasHTTP{fallo: caso.fallo}
			s := &sesionCoordinacion{perfiles: []identidad.Perfil{caso.perfil}}
			cfg := ConfiguracionSesion{Servicio: s, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
			h := Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Documentos: ConfiguracionDocumentos{Plantillas: p, GestionPlantillas: p}})
			q := "?tipo=entrada%2Faceptacion&idioma=es"
			if caso.metodo != "GET" {
				q += "&version=2.0.0&version_previa=2"
			}
			r := httptest.NewRequest(caso.metodo, "https://etf.interno/api/admin/plantillas"+q+caso.extra, bytes.NewReader(make([]byte, caso.tamano)))
			cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
			if caso.sesion {
				r.AddCookie(&http.Cookie{Name: cookieSesion, Value: cookie})
			}
			r.Header.Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
			if caso.csrf {
				r.Header.Set("Origin", cfg.Origen)
				r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(cfg).testigo(cookie))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if p.publicaciones != 0 {
				t.Fatal("POST activo plantilla al subir")
			}
			if w.Code != caso.estado {
				t.Fatal(w.Code, w.Body.String())
			}
			if caso.estado == 400 || caso.estado == 401 || caso.estado == 403 || caso.estado == 405 || caso.estado == 413 {
				if p.llamadas != 0 {
					t.Fatal("caso de uso invocado antes de validar")
				}
			}
			if p.llamadas > 0 && p.actor.UsuarioID != "ab60479d-b778-407f-a83e-012345678901" {
				t.Fatal("actor ajeno a sesión")
			}
			if caso.nombre == "conflicto" && !strings.Contains(w.Body.String(), `"version_actual":8`) {
				t.Fatal(w.Body.String())
			}
		})
	}
}

func (p *plantillasHTTP) Listar(_ context.Context, a entrada.Actor) ([]entrada.VersionPlantilla, error) {
	p.actor = a
	p.llamadas++
	return []entrada.VersionPlantilla{}, p.fallo
}
func (p *plantillasHTTP) Subir(ctx context.Context, a entrada.Actor, v entrada.PublicarPlantilla, b []byte) (entrada.ReferenciaPlantilla, error) {
	p.actor = a
	p.llamadas++
	return entrada.ReferenciaPlantilla{Revision: 2}, p.fallo
}
func (p *plantillasHTTP) Activar(_ context.Context, a entrada.Actor, _ entrada.ActivarPlantilla) (entrada.ReferenciaPlantilla, error) {
	p.actor = a
	p.llamadas++
	return entrada.ReferenciaPlantilla{Revision: 4}, p.fallo
}
func TestGestionPlantillasHTTP(t *testing.T) {
	for _, ruta := range []string{"/api/admin/plantillas", "/api/admin/plantillas/versiones?tipo=entrada%2Faceptacion&idioma=es&version=2.0.0&version_previa=2", "/api/admin/plantillas/11111111-1111-4111-8111-111111111111/activar"} {
		for _, caso := range []struct {
			nombre       string
			perfil       identidad.Perfil
			sesion, csrf bool
			extra, body  string
			error        error
			estado       int
		}{
			{nombre: "administracion", perfil: identidad.Administracion, sesion: true, csrf: true, estado: 200},
			{nombre: "anonimo", csrf: true, estado: 401},
			{nombre: "coordinacion", perfil: identidad.Coordinacion, sesion: true, csrf: true, estado: 403},
			{nombre: "consulta", perfil: identidad.Consulta, sesion: true, csrf: true, estado: 403},
			{nombre: "profesional_etf", perfil: identidad.ProfesionalETF, sesion: true, csrf: true, estado: 403},
			{nombre: "sin_csrf", perfil: identidad.Administracion, sesion: true, estado: 403},
			{nombre: "marcadores", perfil: identidad.Administracion, sesion: true, csrf: true, error: &entrada.DetalleMarcadores{Faltan: []string{":numero"}, Sobran: []string{":intruso"}}, estado: 422},
			{nombre: "conflicto", perfil: identidad.Administracion, sesion: true, csrf: true, error: &entrada.ConflictoPlantilla{VersionActual: 9}, estado: 409},
		} {
			t.Run(ruta+caso.nombre, func(t *testing.T) {
				p := &plantillasHTTP{fallo: caso.error}
				cfg := ConfiguracionSesion{Servicio: &sesionCoordinacion{perfiles: []identidad.Perfil{caso.perfil}}, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
				h := Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: cfg, Documentos: ConfiguracionDocumentos{Plantillas: p, GestionPlantillas: p}})
				metodo, tipo, body := "POST", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "PK"
				estado := caso.estado
				if strings.HasSuffix(ruta, "activar") {
					tipo = "application/json"
					body = `{"version_previa":2}`
				} else if ruta == "/api/admin/plantillas" {
					metodo = "GET"
					if caso.nombre == "sin_csrf" {
						estado = 200
					}
				} else if estado == 200 {
					estado = 201
				}
				r := httptest.NewRequest(metodo, "https://etf.interno"+ruta, strings.NewReader(body))
				r.Header.Set("Content-Type", tipo)
				cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
				if caso.sesion {
					r.AddCookie(&http.Cookie{Name: cookieSesion, Value: cookie})
				}
				if caso.csrf {
					r.Header.Set("Origin", cfg.Origen)
					r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(cfg).testigo(cookie))
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != estado {
					t.Fatal(w.Code, w.Body.String())
				}
				if estado == 401 || estado == 403 {
					if p.llamadas != 0 {
						t.Fatal("caso de uso sin permiso")
					}
				}
				if p.llamadas > 0 && p.actor.UsuarioID != "ab60479d-b778-407f-a83e-012345678901" {
					t.Fatal("actor ajeno a sesion")
				}
				if caso.nombre == "marcadores" && (!strings.Contains(w.Body.String(), "intruso") || !strings.Contains(w.Body.String(), `"antivirus":"limpio"`)) {
					t.Fatal(w.Body.String())
				}
			})
		}
	}
}
