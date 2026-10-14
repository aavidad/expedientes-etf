package paquete

import (
	"math"
	"testing"
)

func TestContrasteSinRedondeo(t *testing.T) {
	for _, caso := range []struct {
		a, b     string
		esperado float64
	}{
		{"#000000", "#ffffff", 21}, {"#ffffff", "#000000", 21}, {"#ffffff", "#ffffff", 1},
	} {
		if got := Contraste(caso.a, caso.b); math.Abs(got-caso.esperado) > 0.000001 {
			t.Fatalf("contraste: %v", got)
		}
	}
	// Es el par que el documento prohíbe para texto apagado sobre fondo crema.
	if Contraste("#6b7075", "#f4f2ec") >= 4.5 {
		t.Fatal("se redondeó un contraste insuficiente")
	}
}
func TestManifiestoRechazaVersionFuturaYURLDeFuente(t *testing.T) {
	tema := Tema{Identificador: "costa", Nombre: map[string]string{"es": "Costa"}, Version: "0.1.0", Autor: "Responsable del producto", Licencia: "MIT", VersionMinimaApp: "0.1.0", Tipo: "claro"}
	if !tema.Valido() {
		t.Fatal("manifiesto válido rechazado")
	}
	tema.VersionMinimaApp = "0.2.0"
	if tema.Valido() {
		t.Fatal("versión futura aceptada")
	}
	tema.VersionMinimaApp = "0.1.0"
	tema.LicenciasFuentes = map[string]string{"texto": "OFL-1.1"}
	tema.Fuentes = map[string]string{"texto": "fuentes/a');background.woff2"}
	if tema.Valido() {
		t.Fatal("ruta CSS activa aceptada")
	}
}
