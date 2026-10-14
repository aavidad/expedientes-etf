package paquete

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aavidad/expedientes-etf/internal/adaptadores/ficheros"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
)

func TestTemaInvalidoVuelveAlPredeterminadoYNoSeLista(t *testing.T) {
	base := t.TempDir()
	for _, nombre := range []string{"expedientes-etf", "costa"} {
		if err := os.MkdirAll(filepath.Join(base, nombre, "fuentes"), 0700); err != nil {
			t.Fatal(err)
		}
		for _, file := range []string{"tema.json", "tokens.json", "fuentes/public-sans.woff2", "fuentes/source-serif-4.woff2"} {
			b, err := os.ReadFile("../../../temas/expedientes-etf/" + file)
			if err != nil {
				t.Fatal(err)
			}
			if file == "tema.json" {
				var datos map[string]any
				if err := json.Unmarshal(b, &datos); err != nil {
					t.Fatal(err)
				}
				datos["identificador"] = nombre
				b, err = json.Marshal(datos)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(base, nombre, file), b, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	p := ficheros.NuevosPaquetes("../../../idiomas", base)
	c := NuevaConsulta(p, p)
	// Un tema parcial hereda los tokens ausentes.
	file := filepath.Join(base, "costa", "tokens.json")
	if err := os.WriteFile(file, []byte(`{"color.fondo":"#ffffff"}`), 0600); err != nil {
		t.Fatal(err)
	}
	temas, err := c.Temas(context.Background())
	if err != nil || len(temas) != 2 {
		t.Fatalf("parcial: %v %v", temas, err)
	}
	if err := os.WriteFile(file, []byte(`{"color.fondo":"#ffffff","color.texto":"#ffffff"}`), 0600); err != nil {
		t.Fatal(err)
	}
	tokens, err := c.Tokens(context.Background(), "costa")
	if err != nil || tokens["color.fondo"] != "#f4f2ec" {
		t.Fatalf("respaldo: %v %v", tokens, err)
	}
	temas, err = c.Temas(context.Background())
	if err != nil || len(temas) != 1 || temas[0].Identificador != dominio.TemaPredeterminado {
		t.Fatalf("lista inválida: %v %v", temas, err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := c.Tokens(ctx, "costa"); err != context.Canceled {
		t.Fatalf("cancelación: %v", err)
	}
}
