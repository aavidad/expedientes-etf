package http

import (
	"bytes"
	"context"
	"encoding/base64"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/documento/puertos/entrada"
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	identidad "github.com/aavidad/expedientes-etf/internal/dominio/identidad"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

type emisionesPrueba struct {
	actor              entrada.Actor
	solicitud          entrada.Solicitud
	llamadas, rechazos int
	fallo              error
	contenido          []byte
}

func (e *emisionesPrueba) Emitir(_ context.Context, a entrada.Actor, p entrada.Solicitud) (entrada.Resultado, error) {
	e.actor = a
	e.solicitud = p
	e.llamadas++
	e.contenido = []byte("PK-documento")
	return entrada.Resultado{Contenido: e.contenido, Emision: d.Emision{ID: "33333333-3333-4333-8333-333333333333", Plantilla: d.Plantilla{Version: "1.0.0"}, Aviso: "documento.regenerado", HuellaResultado: d.Huella(e.contenido)}}, e.fallo
}
func (e *emisionesPrueba) Rechazar(context.Context, entrada.Actor, string) error {
	e.rechazos++
	return nil
}
func TestDocumentosHTTPContratoSesionCSRFYErrores(t *testing.T) {
	for _, caso := range []struct {
		nombre, metodo, cuerpo, clave string
		perfil                        identidad.Perfil
		csrf, caducada                bool
		fallo                         error
		estado                        int
	}{
		{"descarga", "POST", `{"tipo":"entrada/aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es"}`, "solicitud-1", identidad.Coordinacion, true, false, nil, 201},
		{"no_aceptacion", "POST", `{"tipo":"entrada/no_aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es"}`, "no-aceptacion", identidad.Coordinacion, true, false, nil, 201},
		{"lista_espera", "POST", `{"tipo":"entrada/no_aceptacion/lista_espera","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"en"}`, "lista-espera", identidad.Coordinacion, true, false, nil, 201},
		{"estado_no_aceptacion", "POST", `{"tipo":"entrada/no_aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es"}`, "estado-no", identidad.Coordinacion, true, false, entrada.ErrNoAceptada, 409},
		{"estado_lista_espera", "POST", `{"tipo":"entrada/no_aceptacion/lista_espera","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"en"}`, "estado-espera", identidad.Coordinacion, true, false, entrada.ErrListaEspera, 409},
		{"profesional_etf", "POST", `{"tipo":"entrada/no_aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es"}`, "perfil-equipo", identidad.ProfesionalETF, true, false, nil, 403},
		{"sin_csrf", "POST", `{}`, "solicitud-1", identidad.Coordinacion, false, false, nil, 403},
		{"caducada", "POST", `{}`, "solicitud-1", identidad.Coordinacion, true, true, nil, 401},
		{"administracion", "POST", `{}`, "solicitud-1", identidad.Administracion, true, false, nil, 403},
		{"consulta", "POST", `{}`, "solicitud-1", identidad.Consulta, true, false, nil, 403},
		{"get", "GET", `{}`, "solicitud-1", identidad.Coordinacion, true, false, nil, 405},
		{"datos_del_cliente", "POST", `{"tipo":"entrada/aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es","datos":{"familia":"Ortega"}}`, "solicitud-1", identidad.Coordinacion, true, false, nil, 400},
		{"sin_clave", "POST", `{"tipo":"entrada/aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es"}`, "", identidad.Coordinacion, true, false, nil, 400},
		{"idempotencia", "POST", `{"tipo":"entrada/aceptacion","expediente_id":"22222222-2222-4222-8222-222222222222","idioma":"es"}`, "solicitud-1", identidad.Coordinacion, true, false, entrada.ErrIdempotencia, 422},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := &emisionesPrueba{fallo: caso.fallo}
			s := &sesionCoordinacion{perfiles: []identidad.Perfil{caso.perfil}, caducada: caso.caducada}
			c := ConfiguracionSesion{Servicio: s, Origen: "https://etf.interno", ClaveCSRF: bytes.Repeat([]byte{9}, 32)}
			h := Nuevo(paquetesPrueba{}, fstest.MapFS{}, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), nil, Opciones{Sesion: c, Documentos: ConfiguracionDocumentos{Servicio: e}})
			cookie := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
			r := httptest.NewRequest(caso.metodo, "https://etf.interno/api/documentos/emisiones", bytes.NewBufferString(caso.cuerpo))
			r.AddCookie(&http.Cookie{Name: cookieSesion, Value: cookie})
			r.Header.Set("Content-Type", "application/json")
			if caso.clave != "" {
				r.Header.Set("Idempotency-Key", caso.clave)
			}
			if caso.csrf {
				r.Header.Set("Origin", c.Origen)
				r.Header.Set("X-CSRF-Token", nuevaSesionHTTP(c).testigo(cookie))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado {
				t.Fatal(w.Code, w.Body.String())
			}
			if caso.estado == 201 {
				if w.Header().Get("X-Aviso-Clave") != "documento.regenerado" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Plantilla-Version") != "1.0.0" || e.actor.UsuarioID != "ab60479d-b778-407f-a83e-012345678901" || e.solicitud.Clave != caso.clave || w.Body.String() != "PK-documento" || !bytes.Equal(e.contenido, make([]byte, len(e.contenido))) {
					t.Fatal("contrato de descarga incorrecto")
				}
			}
			if caso.estado == 400 || caso.estado == 403 || caso.estado == 405 {
				if e.llamadas != 0 {
					t.Fatal("emisión invocada en petición rechazada")
				}
			}
		})
	}
}
