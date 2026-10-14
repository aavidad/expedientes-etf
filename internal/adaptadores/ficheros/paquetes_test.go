package ficheros

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const textosValidos = `{"comun.titulo":"Expedientes ETF","comun.error_carga":"No se pudo cargar","comun.reintentar":"Reintentar"}`

func TestPaquetesIncluidos(t *testing.T) {
	p := NuevosPaquetes("../../../idiomas", "../../../temas")
	textos, err := p.Textos(context.Background(), "es")
	if err != nil || textos["comun.titulo"] == "" {
		t.Fatalf("textos: %v", err)
	}
	tokens, err := p.Tokens(context.Background(), "expedientes-etf")
	if err != nil || tokens["color.borde-control"] != "#8a8577" {
		t.Fatalf("tokens: %v", err)
	}
}

func TestRutasYEnlacesNoEscapanDeLaRaiz(t *testing.T) {
	base := t.TempDir()
	exterior := t.TempDir()
	if err := os.MkdirAll(filepath.Join(exterior, "textos"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exterior, "textos", "comun.json"), []byte(textosValidos), 0600); err != nil {
		t.Fatal(err)
	}
	tokens, err := os.ReadFile("../../../temas/expedientes-etf/tokens.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exterior, "tokens.json"), tokens, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exterior, filepath.Join(base, "enlace")); err != nil {
		t.Fatal(err)
	}
	p := NuevosPaquetes(base, base)
	for _, nombre := range []string{"../", "..", "/etc", "es/../../", "es%2f..", "enlace"} {
		if _, err := p.Textos(context.Background(), nombre); err == nil {
			t.Errorf("ruta aceptada: %q", nombre)
		}
		if _, err := p.Tokens(context.Background(), nombre); err == nil {
			t.Errorf("tema aceptado: %q", nombre)
		}
	}
}

func TestNoSirveEnlaceComoJSON(t *testing.T) {
	base := t.TempDir()
	prepararIdioma(t, base)
	dir := filepath.Join(base, "es", "textos")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "privado.json"), []byte(textosValidos), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../privado.json", filepath.Join(dir, "comun.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevosPaquetes(base, base).Textos(context.Background(), "es"); err == nil {
		t.Fatal("enlace aceptado")
	}
}

func TestTokensRechazanCSSActivo(t *testing.T) {
	for _, caso := range [][2]string{
		{"color.fondo", "red; background:url(https://fuera.invalid)"},
		{"fuente.texto", "url(https://fuera.invalid)"},
		{"color.fondo}body{", "#ffffff"},
		{"espacio.1", "calc(1px + 2px)"},
		{"desconocido.token", "1px"},
		{"texto.normal", "1000000px"},
	} {
		if tokenValido(caso[0], caso[1]) {
			t.Errorf("CSS aceptado: %v", caso)
		}
	}
}

func TestRechazaJSONInvalidoYClavesDuplicadas(t *testing.T) {
	base := t.TempDir()
	prepararIdioma(t, base)
	dir := filepath.Join(base, "es", "textos")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	p := NuevosPaquetes(base, base)
	for _, dato := range []string{`no es JSON`, `null`, `{"comun.titulo":25}`, `{"comun.titulo":"sin textos de recuperación"}`} {
		if err := os.WriteFile(filepath.Join(dir, "comun.json"), []byte(dato), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := p.Textos(context.Background(), "es"); err == nil {
			t.Errorf("JSON aceptado: %s", dato)
		}
	}
	for _, nombre := range []string{"comun.json", "duplicado.json"} {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(textosValidos), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := p.Textos(context.Background(), "es"); err == nil {
		t.Fatal("clave repetida aceptada")
	}
}

func prepararIdioma(t *testing.T, base string) {
	t.Helper()
	for _, nombre := range []string{"textos", "catalogos", "plantillas"} {
		if err := os.MkdirAll(filepath.Join(base, "es", nombre), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "es", "idioma.json"), []byte(`{"codigo":"es","nombre":"Castellano","version":"0.1.0"}`), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestDetectaYRecargaIdiomasSinReiniciar(t *testing.T) {
	base := t.TempDir()
	prepararIdioma(t, base)
	fichero := filepath.Join(base, "es", "textos", "comun.json")
	if err := os.WriteFile(fichero, []byte(textosValidos), 0600); err != nil {
		t.Fatal(err)
	}
	p := NuevosPaquetes(base, base)
	lista, err := p.Idiomas(context.Background())
	if err != nil || len(lista) != 1 || lista[0].Codigo != "es" {
		t.Fatalf("lista: %v %v", lista, err)
	}
	if err := os.WriteFile(fichero, []byte(`{"comun.titulo":"Gestión familiar","comun.error_carga":"No se pudo cargar","comun.reintentar":"Reintentar"}`), 0600); err != nil {
		t.Fatal(err)
	}
	textos, err := p.Textos(context.Background(), "es")
	if err != nil || textos["comun.titulo"] != "Gestión familiar" {
		t.Fatalf("recarga: %v %v", textos, err)
	}
	if err := os.WriteFile(filepath.Join(base, "es", "idioma.json"), []byte(`{"codigo":"en","nombre":"English","version":"0.1.0"}`), 0600); err != nil {
		t.Fatal(err)
	}
	lista, err = p.Idiomas(context.Background())
	if err != nil || len(lista) != 0 {
		t.Fatalf("manifiesto inválido: %v %v", lista, err)
	}
}
