package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/aplicacion/paquete/puertos/entrada"
)

type paquetesPWAPrueba struct{ paquetesPrueba }

func (paquetesPWAPrueba) Idiomas(context.Context) ([]entrada.Idioma, error) {
	return []entrada.Idioma{{Codigo: "es"}, {Codigo: "en"}}, nil
}
func (paquetesPWAPrueba) Temas(context.Context) ([]entrada.Tema, error) {
	return []entrada.Tema{{Identificador: "expedientes-etf"}, {Identificador: "alto-contraste"}}, nil
}
func (paquetesPWAPrueba) RecursoTema(context.Context, string, string) (entrada.Recurso, error) {
	return entrada.Recurso{Tipo: "image/png", Contenido: []byte{}}, nil
}

func (paquetesPWAPrueba) Textos(_ context.Context, codigo string) (map[string]string, error) {
	return map[string]string{"pwa1.nombre": "Expedientes ETF", "pwa1.nombre_corto": "ETF", "pwa1.descripcion": "Descripción <segura>", "pwa1.sin_conexion": codigo + " <sin conexión>", "pwa1.sin_conexion_ayuda": "Conecta de nuevo", "comun.reintentar": "Reintentar"}, nil
}
func (paquetesPWAPrueba) Tema(_ context.Context, nombre string) (entrada.Tema, error) {
	return entrada.Tema{Identificador: nombre}, nil
}
func (paquetesPWAPrueba) Tokens(_ context.Context, nombre string) (map[string]string, error) {
	b, err := os.ReadFile("../../../temas/" + nombre + "/tokens.json")
	if err != nil {
		return nil, err
	}
	var tokens map[string]string
	err = json.Unmarshal(b, &tokens)
	return tokens, err
}
func TestPWAContratoYPreferencias(t *testing.T) {
	handler := Nuevo(paquetesPWAPrueba{}, os.DirFS("../../../web/public"), "es", "expedientes-etf", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{})
	for _, tema := range []string{"expedientes-etf", "alto-contraste"} {
		r := httptest.NewRecorder()
		handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/manifest.webmanifest?idioma=en&tema="+tema, nil))
		if r.Code != 200 || r.Header().Get("Content-Type") != "application/manifest+json" || r.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("manifiesto: %d %v", r.Code, r.Header())
		}
		var manifiesto struct {
			Name, Display, Scope string
			ThemeColor           string `json:"theme_color"`
			Icons                []struct{ Src, Sizes, Purpose string }
		}
		if err := json.Unmarshal(r.Body.Bytes(), &manifiesto); err != nil {
			t.Fatal(err)
		}
		tokens, _ := (paquetesPWAPrueba{}).Tokens(context.Background(), tema)
		if manifiesto.Name != "Expedientes ETF" || manifiesto.Display != "standalone" || manifiesto.Scope != "/" || manifiesto.ThemeColor != tokens["color.acento"] || len(manifiesto.Icons) != 4 {
			t.Fatalf("contrato: %+v", manifiesto)
		}
		for _, icono := range manifiesto.Icons {
			if !strings.Contains(icono.Src, "/"+tema+"/") || (icono.Purpose != "any" && icono.Purpose != "maskable") {
				t.Fatal(icono)
			}
		}
	}
	r := httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/service-worker.js?idioma=en", nil))
	if r.Code != 200 || !strings.Contains(r.Body.String(), "en \\u0026lt;sin conexión\\u0026gt;") {
		t.Fatalf("aviso traducido y escapado: %s", r.Body.String())
	}
	for _, prohibido := range []string{"ZgotmplZ", "caches.", "localStorage", "sessionStorage", "indexedDB"} {
		if strings.Contains(r.Body.String(), prohibido) {
			t.Fatalf("almacenamiento: %s", prohibido)
		}
	}
	if !strings.Contains(r.Body.String(), "u.pathname.startsWith('/api/')") {
		t.Fatal("API debe quedar fuera del manejador")
	}
	for _, directiva := range []string{"manifest-src 'self'", "worker-src 'self'"} {
		if !strings.Contains(r.Header().Get("Content-Security-Policy"), directiva) {
			t.Fatal(directiva)
		}
	}
}

// Un volumen anterior puede no contener los textos de la distribución nueva.
type paquetesPWAIncompletos struct {
	paquetesPWAPrueba
	ausente string
}

func (p paquetesPWAIncompletos) Textos(ctx context.Context, codigo string) (map[string]string, error) {
	textos, err := p.paquetesPWAPrueba.Textos(ctx, codigo)
	textos[p.ausente] = "  "
	return textos, err
}
func TestPWARechazaTextosIncompletos(t *testing.T) {
	for _, caso := range []struct{ ruta, clave string }{
		{"/manifest.webmanifest", "pwa1.nombre"},
		{"/manifest.webmanifest", "pwa1.nombre_corto"},
		{"/manifest.webmanifest", "pwa1.descripcion"},
		{"/service-worker.js", "pwa1.sin_conexion"},
		{"/service-worker.js", "pwa1.sin_conexion_ayuda"},
		{"/service-worker.js", "comun.reintentar"},
	} {
		t.Run(caso.clave, func(t *testing.T) {
			handler := Nuevo(paquetesPWAIncompletos{ausente: caso.clave}, os.DirFS("../../../web/public"), "es", "expedientes-etf", nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, Opciones{})
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, caso.ruta, nil))
			if respuesta.Code != http.StatusServiceUnavailable || !strings.Contains(respuesta.Body.String(), "paquete.no_disponible") {
				t.Fatalf("catálogo incompleto: %d %s", respuesta.Code, respuesta.Body.String())
			}
		})
	}
}
