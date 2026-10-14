package ficheros

import (
	"context"
	"testing"
)

func TestNombresDelCatalogoReal(t *testing.T) {
	nombres, err := NuevosMunicipios("../../../../idiomas", "es").Nombres(context.Background())
	if err != nil || nombres["municipio.18057"] != "Cúllar Vega" || nombres["municipio.18056"] != "Cúllar" {
		t.Fatal(err, nombres)
	}
	if nombres["municipio.18099.variante.huetor_santillan"] != "Huétor Santillán" || nombres["municipio.18013.variante.alhama"] != "Alhama" {
		t.Fatal("faltan variantes municipales")
	}
	if _, err := NuevosMunicipios("../../../../idiomas", "../es").Nombres(context.Background()); err == nil {
		t.Fatal("salto de directorio admitido")
	}
}
