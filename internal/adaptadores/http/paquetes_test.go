package http

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete"
)

func TestIdiomasYTemasIntegrados(t *testing.T) {
	p := ficheros.NuevosPaquetes("../../../idiomas", "../../../temas")
	var log bytes.Buffer
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte(`<html lang="{{.Idioma}}"><title>{{.Titulo}}</title></html>`)}}
	handler := Nuevo(paquete.NuevaConsulta(p, p), web, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(&log, nil)), nil, Opciones{})
	for _, caso := range []struct {
		ruta, contenido string
		estado          int
	}{
		{"/api/idiomas", `"codigo":"en"`, 200},
		{"/api/idiomas/en/textos", "ETF case files", 200},
		{"/api/idiomas/fr/textos", "Expedientes ETF", 200},
		{"/api/temas", `"identificador":"alto-contraste"`, 200},
		{"/api/temas/alto-contraste/tokens", "--color-texto:#000000", 200},
		{"/api/temas/inexistente/tokens", "--color-fondo:#f4f2ec", 200},
		{"/api/temas/expedientes-etf/recursos/tokens.json", "paquete.no_disponible", 404},
		{"/api/temas/expedientes-etf/recursos/%2e%2e/alto-contraste/tokens.json", "http.no_encontrado", 404},
	} {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest("GET", caso.ruta, nil))
		if r.Code != caso.estado || !strings.Contains(r.Body.String(), caso.contenido) {
			t.Fatalf("%s: %d %s", caso.ruta, r.Code, r.Body.String())
		}
	}
}
