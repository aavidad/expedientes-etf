package http

import (
	"context"
	entrada "github.com/aavidad/expedientes-etf/internal/aplicacion/catalogo/puertos/entrada"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

type editorPrueba struct {
	llamadas int
	actor    entrada.Actor
	err      error
}

func (e *editorPrueba) Consultar(context.Context, entrada.Actor, string) (entrada.Administracion, error) {
	return entrada.Administracion{}, e.err
}
func (e *editorPrueba) Guardar(_ context.Context, a entrada.Actor, _ string, _ entrada.Cambio) (entrada.Administracion, error) {
	e.llamadas++
	e.actor = a
	return entrada.Administracion{}, e.err
}
func TestEscrituraCatalogoBarreraOrigenYValidacion(t *testing.T) {
	for _, caso := range []struct {
		nombre             string
		sesion             bool
		origen, body, tipo string
		estado             int
	}{
		{"cerrado", false, "", `{}`, "application/json", 401},
		{"origen ajeno", true, "http://ajeno.invalid", `{}`, "application/json", 403},
		{"formulario", true, "", `{}`, "text/plain", 415},
		{"actor de cliente", true, "", `{"usuario_id":"intruso"}`, "application/json", 400},
		{"documento adicional", true, "", `{} {}`, "application/json", 400},
		{"válido", true, "https://etf.interno", `{}`, "application/json", 201},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := &editorPrueba{}
			mux := Nuevo(paquetesPrueba{}, nil, "es", "expedientes-etf", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{Sesion: sesionAdministracionPrueba(), Catalogos: ConfiguracionCatalogos{Edicion: e}})
			r := httptest.NewRequest("POST", "https://etf.interno/api/admin/firmantes", strings.NewReader(caso.body))
			r.Header.Set("Content-Type", caso.tipo)
			if caso.origen != "" {
				r.Header.Set("Origin", caso.origen)
			}
			if caso.sesion {
				autorizarPeticionPrueba(r)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != caso.estado {
				t.Fatal(w.Code, w.Body.String())
			}
			if e.llamadas > 0 && e.actor.UsuarioID != accesoPrueba().Usuario.ID {
				t.Fatal("actor ajeno")
			}
			if caso.estado != 201 && e.llamadas > 0 {
				t.Fatal("guardado tras rechazo")
			}
		})
	}
}
func TestConflictoIncluyeVersionActual(t *testing.T) {
	w := httptest.NewRecorder()
	errorCatalogo(w, &entrada.Conflicto{Actual: 7, Recurso: "listas"})
	if w.Code != 409 || !strings.Contains(w.Body.String(), `"version_actual":7`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
