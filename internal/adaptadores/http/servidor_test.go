package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
	"html"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

type paquetesPrueba struct{}

func (paquetesPrueba) Tema(context.Context, string) (entrada.Tema, error) {
	return entrada.Tema{Identificador: "expedientes-etf"}, nil
}
func (paquetesPrueba) Temas(context.Context) ([]entrada.Tema, error) {
	return []entrada.Tema{{Identificador: "expedientes-etf", Nombre: map[string]string{"es": "Expedientes ETF"}}}, nil
}
func (paquetesPrueba) RecursoTema(context.Context, string, string) (entrada.Recurso, error) {
	return entrada.Recurso{}, fs.ErrNotExist
}

func (paquetesPrueba) Idiomas(context.Context) ([]entrada.Idioma, error) {
	return []entrada.Idioma{{Codigo: "es", Nombre: "Castellano", Version: "0.1.0"}}, nil
}

func (paquetesPrueba) Textos(_ context.Context, codigo string) (map[string]string, error) {
	if codigo != "es" {
		return nil, fs.ErrNotExist
	}
	return map[string]string{"comun.titulo": "Expedientes <ETF>", "comun.error_carga": "No se ha podido cargar", "comun.reintentar": "Reintentar"}, nil
}

func (paquetesPrueba) Tokens(_ context.Context, nombre string) (map[string]string, error) {
	if nombre != "expedientes-etf" {
		return nil, fs.ErrNotExist
	}
	return map[string]string{"color.fondo": "#f4f2ec"}, nil
}

func servidorPrueba(log *bytes.Buffer, avisos []string, comprobar ...func(context.Context) error) http.Handler {
	web := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte(`<html lang="{{.Idioma}}"><title>{{.Titulo}}</title><link rel="stylesheet" href="/api/temas/{{.Tema}}/tokens"></html>`)},
		"assets/app.js": &fstest.MapFile{Data: []byte(`export {};`)},
		".gitkeep":      &fstest.MapFile{},
	}
	var base func(context.Context) error
	if len(comprobar) > 0 {
		base = comprobar[0]
	}
	return Nuevo(paquetesPrueba{}, web, "es", "expedientes-etf", avisos, slog.New(slog.NewJSONHandler(log, nil)), base, Opciones{})
}

func TestSaludDetectaCaidaYRecuperacionDeLaBase(t *testing.T) {
	var log bytes.Buffer
	disponible := false
	servidor := servidorPrueba(&log, []string{"ETF_IDIOMA"}, func(ctx context.Context) error {
		if _, limitado := ctx.Deadline(); !limitado {
			t.Fatal("sin plazo")
		}
		if !disponible {
			return errors.New("credencial-reservada direccion-interna")
		}
		return nil
	})
	for _, estado := range []int{503, 200} {
		r := httptest.NewRecorder()
		servidor.ServeHTTP(r, httptest.NewRequest("GET", "/salud", nil))
		var datos map[string]any
		if err := json.Unmarshal(r.Body.Bytes(), &datos); err != nil {
			t.Fatal(err)
		}
		base, salud := "no_disponible", "degradado"
		if disponible {
			base, salud = "ok", "ok"
		}
		if r.Code != estado || datos["base"] != base || datos["estado"] != salud || datos["avisos"] == nil {
			t.Fatalf("salud: %d %s", r.Code, r.Body.String())
		}
		if strings.Contains(r.Body.String()+log.String(), "reservada") || strings.Contains(r.Body.String()+log.String(), "interna") {
			t.Fatal("se ha expuesto el diagnóstico de la base")
		}
		disponible = true
	}
}

func TestRutasPublicasDelEsqueleto(t *testing.T) {
	var log bytes.Buffer
	servidor := servidorPrueba(&log, nil)
	for _, caso := range []struct {
		metodo, ruta    string
		estado          int
		tipo, contenido string
	}{
		{"GET", "/salud", 200, "application/json", `{"estado":"ok"}`},
		{"GET", "/favicon.ico", 204, "", ""},
		{"GET", "/api/idiomas/es/textos", 200, "application/json", `"comun.titulo"`},
		{"GET", "/api/temas/expedientes-etf/tokens", 200, "text/css", "--color-fondo:#f4f2ec;"},
		{"GET", "/", 200, "text/html", "Expedientes &lt;ETF&gt;"},
		{"GET", "/assets/app.js", 200, "text/javascript", "export {};"},
		{"GET", "/api/no-existe", 403, "application/json", "identidad.acceso_denegado"},
		{"GET", "/api/idiomas/inexistente/textos", 404, "application/json", "paquete.no_disponible"},
		{"GET", "/assets/", 404, "application/json", "http.no_encontrado"},
		{"GET", "/.gitkeep", 403, "application/json", "identidad.acceso_denegado"},
		{"GET", "/go.mod", 403, "application/json", "identidad.acceso_denegado"},
		{"POST", "/salud", 405, "application/json", "http.metodo_no_permitido"},
		{"POST", "/api/idiomas/es/textos", 405, "application/json", "http.metodo_no_permitido"},
	} {
		t.Run(caso.metodo+caso.ruta, func(t *testing.T) {
			r := httptest.NewRecorder()
			servidor.ServeHTTP(r, httptest.NewRequest(caso.metodo, caso.ruta, nil))
			if r.Code != caso.estado || !strings.HasPrefix(r.Header().Get("Content-Type"), caso.tipo) || !strings.Contains(r.Body.String(), caso.contenido) {
				t.Fatalf("respuesta: %d %v %s", r.Code, r.Header(), r.Body.String())
			}
			if r.Header().Get("Cache-Control") != "no-store" || r.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(r.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
				t.Fatal("faltan cabeceras")
			}
		})
	}
}

func TestRegistroNoContieneEntradasDeLaPeticion(t *testing.T) {
	var log bytes.Buffer
	servidor := servidorPrueba(&log, nil)
	peticion := httptest.NewRequest("GET", "/api/idiomas/dato-reservado/textos?nombre=persona-privada", nil)
	peticion.Header.Set("Authorization", "Bearer secreto-prueba")
	peticion.Header.Set("User-Agent", "agente-privado")
	servidor.ServeHTTP(httptest.NewRecorder(), peticion)
	for _, prohibido := range []string{"dato-reservado", "persona-privada", "secreto-prueba", "agente-privado", peticion.RemoteAddr} {
		if strings.Contains(log.String(), prohibido) {
			t.Fatalf("registro contiene %s", prohibido)
		}
	}
	var evento map[string]any
	if err := json.Unmarshal(log.Bytes(), &evento); err != nil {
		t.Fatal(err)
	}
	if evento["ruta"] != "/api/idiomas/{codigo}/textos" || evento["estado"] != float64(404) {
		t.Fatalf("evento: %v", evento)
	}
}

func TestSaludConPaquetesAusentesSigueRespondiendo(t *testing.T) {
	var log bytes.Buffer
	r := httptest.NewRecorder()
	servidorPrueba(&log, []string{"idiomas.no_disponibles"}).ServeHTTP(r, httptest.NewRequest("GET", "/salud", nil))
	if r.Code != 200 || !strings.Contains(r.Body.String(), "idiomas.no_disponibles") {
		t.Fatalf("salud: %d %s", r.Code, r.Body.String())
	}
}

func TestNonceDeEstilosEsDistintoYNoPermiteScriptsInline(t *testing.T) {
	var log bytes.Buffer
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte(`<html><head><title>{{.Titulo}}</title></head></html>`)}}
	handler := Nuevo(paquetesPrueba{}, web, "es", "expedientes-etf", nil, slog.New(slog.NewJSONHandler(&log, nil)), nil, Opciones{})
	var anterior string
	for range 2 {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest("GET", "/", nil))
		politica := r.Header().Get("Content-Security-Policy")
		if r.Code != 200 || !strings.Contains(politica, "style-src 'self' 'nonce-") || strings.Contains(politica, "unsafe-inline") || !strings.Contains(politica, "script-src 'self';") {
			t.Fatalf("CSP: %s", politica)
		}
		if politica == anterior {
			t.Fatal("nonce reutilizado")
		}
		anterior = politica
		inicio := strings.Index(politica, "'nonce-") + len("'nonce-")
		nonce := strings.SplitN(politica[inicio:], "'", 2)[0]
		if len(nonce) < 32 || !strings.Contains(html.UnescapeString(r.Body.String()), `nonce="`+nonce+`"`) {
			t.Fatal("nonce no coincide con la cabecera")
		}
	}
}
