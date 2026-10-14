package paquete

import (
	"context"
	dominio "github.com/aavidad/expedientes-etf/internal/dominio/paquete"
	"io/fs"
	"testing"
)

type textosPrueba struct{}

func (textosPrueba) Idiomas(context.Context) ([]dominio.Idioma, error) { return nil, nil }
func (textosPrueba) Textos(_ context.Context, codigo string) (map[string]string, error) {
	switch codigo {
	case "es":
		return map[string]string{"comun.titulo": "Expedientes ETF", "comun.reintentar": "Reintentar"}, nil
	case "en":
		return map[string]string{"comun.titulo": "ETF case files"}, nil
	default:
		return nil, fs.ErrNotExist
	}
}
func TestIdiomaRespaldoYClaveAusente(t *testing.T) {
	c := NuevaConsulta(textosPrueba{}, nil)
	for _, codigo := range []string{"en", "fr"} {
		textos, err := c.Textos(context.Background(), codigo)
		if err != nil || textos["comun.reintentar"] != "Reintentar" {
			t.Fatalf("respaldo: %v %v", textos, err)
		}
		if codigo == "en" && textos["comun.titulo"] != "ETF case files" {
			t.Fatal("traducción perdida")
		}
	}
	if _, err := c.Textos(context.Background(), "../es"); err == nil {
		t.Fatal("ruta aceptada")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := c.Textos(ctx, "en"); err != context.Canceled {
		t.Fatalf("cancelación: %v", err)
	}
}
