package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
)

type instalacionHTTP struct {
	fallo    error
	llamadas int
	actor    entrada.Actor
}

func (i *instalacionHTTP) Actual(context.Context, entrada.Actor, string, string) (entrada.Referencia, error) {
	i.llamadas++
	return entrada.Referencia{}, i.fallo
}
func (i *instalacionHTTP) Preparar(context.Context, entrada.Actor, string, string, []byte) (entrada.Preparacion, error) {
	i.llamadas++
	return entrada.Preparacion{}, i.fallo
}
func (i *instalacionHTTP) Activar(_ context.Context, actor entrada.Actor, _ string, _ int64) (entrada.Referencia, error) {
	i.llamadas++
	i.actor = actor
	return entrada.Referencia{}, i.fallo
}
func (i *instalacionHTTP) Descartar(context.Context, entrada.Actor, string) error {
	i.llamadas++
	return i.fallo
}

func TestInstalacionHTTPBarreraCSRFYErrores(t *testing.T) {
	for _, caso := range []struct {
		nombre                         string
		sesion                         bool
		origen, cabecera, tipo, cuerpo string
		fallo                          error
		esperado, llamadas             int
	}{
		{"sin_sesion", false, "", "1", "application/json", `{}`, nil, 401, 0},
		{"sin_cabecera", true, "", "", "application/json", `{}`, nil, 403, 0},
		{"origen_externo", true, "https://fuera.invalid", "1", "application/json", `{}`, nil, 403, 0},
		{"tipo_formulario", true, "", "1", "application/x-www-form-urlencoded", `{}`, nil, 400, 0},
		{"actor_cliente", true, "", "1", "application/json", `{"ticket":"` + strings.Repeat("a", 64) + `","version":0,"usuario_id":"otro"}`, nil, 400, 0},
		{"sin_revision", true, "", "1", "application/json", `{"ticket":"` + strings.Repeat("a", 64) + `"}`, nil, 400, 0},
		{"conflicto", true, "https://etf.interno", "1", "application/json", `{"ticket":"` + strings.Repeat("a", 64) + `","version":0}`, entrada.ErrConflicto, 409, 1},
		{"infectado", true, "", "1", "application/json", `{"ticket":"` + strings.Repeat("a", 64) + `","version":0}`, entrada.ErrInfectado, 422, 1},
		{"antivirus_caido", true, "", "1", "application/json", `{"ticket":"` + strings.Repeat("a", 64) + `","version":0}`, entrada.ErrAntivirus, 503, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			i := &instalacionHTTP{fallo: caso.fallo}
			h := Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: sesionAdministracionPrueba(), Instalacion: ConfiguracionInstalacion{Instalacion: i}})
			r := httptest.NewRequest("POST", "https://etf.interno/api/admin/paquetes/activar", strings.NewReader(caso.cuerpo))
			r.Header.Set("Origin", caso.origen)
			r.Header.Set("X-ETF-Paquetes", caso.cabecera)
			r.Header.Set("Content-Type", caso.tipo)
			if caso.sesion {
				autorizarPeticionPrueba(r)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if i.llamadas > 0 && (i.actor.UsuarioID != accesoPrueba().Usuario.ID || !i.actor.PuedeAdministrar()) {
				t.Fatal("actor ajeno", i.actor)
			}
			if w.Code != caso.esperado || i.llamadas != caso.llamadas {
				t.Fatal(w.Code, i.llamadas, w.Body.String())
			}
		})
	}
}

func TestInstalacionHTTPLimiteAntesCasoUso(t *testing.T) {
	i := &instalacionHTTP{}
	h := Nuevo(nil, nil, "es", "expedientes-etf", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: sesionAdministracionPrueba(), Instalacion: ConfiguracionInstalacion{Instalacion: i}})
	r := httptest.NewRequest(http.MethodPost, "https://etf.interno/api/admin/paquetes/idioma/fr/revisar", strings.NewReader(strings.Repeat("a", (20<<20)+1)))
	r.Header.Set("Content-Type", "application/zip")
	r.Header.Set("X-ETF-Paquetes", "1")
	autorizarPeticionPrueba(r)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 413 || i.llamadas != 0 {
		t.Fatal(w.Code, i.llamadas)
	}
}
