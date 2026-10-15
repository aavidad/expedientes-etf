package paquete

import (
	"slices"
	"testing"
)

func TestCapasTextosOrdenSinRepetir(t *testing.T) {
	casos := map[string][]string{
		"es":                     {"es"},
		"en":                     {"en", "es"},
		".versiones/en/1/h/n/en": {".versiones/en/1/h/n/en", "en", "es"},
		".versiones/es/1/h/n/es": {".versiones/es/1/h/n/es", "es"},
	}
	for activa, esperado := range casos {
		if obtenido := CapasTextos(activa); !slices.Equal(obtenido, esperado) {
			t.Fatal(activa, obtenido)
		}
	}
	textos := map[string]string{"a": "activa"}
	CompletarTextos(textos, map[string]string{"a": "base", "b": "base"})
	if textos["a"] != "activa" || textos["b"] != "base" {
		t.Fatal(textos)
	}
}
