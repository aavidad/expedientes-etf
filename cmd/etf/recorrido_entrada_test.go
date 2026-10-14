package main

import (
	"encoding/json"
	"os"
	"testing"

	adaptadorHTTP "github.com/aavidad/expedientes-etf/internal/adaptadores/http"
)

// Exporta únicamente reglas del mismo montaje que utiliza el proceso real.
// La salida opcional sirve al recorrido; no añade un endpoint a la aplicación.
func TestRegistroRecorridoEntrada(t *testing.T) {
	modulos := append([]adaptadorHTTP.Modulo{adaptadorHTTP.ModuloPlataforma{}}, componerModulos(dependenciasModulos{})...)
	var rutas []adaptadorHTTP.Ruta
	var reglas []adaptadorHTTP.ReglaRuta
	for _, modulo := range modulos {
		for _, ruta := range modulo.Rutas() {
			rutas = append(rutas, ruta)
			reglas = append(reglas, ruta.ReglaRuta)
		}
	}
	if err := adaptadorHTTP.ValidarRutas(rutas); err != nil {
		t.Fatal(err)
	}
	if destino := os.Getenv("ETF_RECORRIDO_REGISTRO"); destino != "" {
		datos, err := json.Marshal(reglas)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destino, datos, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("INT5: %d declaraciones exportadas desde composición real", len(reglas))
}
