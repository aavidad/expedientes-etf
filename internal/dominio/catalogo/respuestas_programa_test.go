package catalogo_test

import (
	"encoding/json"
	"os"
	"testing"
)

func TestD9IlloraConservaCentroYEquipoIndependientes(t *testing.T) {
	b, err := os.ReadFile("../../../datos/catalogos/municipios.json")
	if err != nil {
		t.Fatal(err)
	}
	var datos struct {
		Municipios []struct {
			Codigo string
			Centro string `json:"centro_codigo"`
			Equipo string `json:"equipo_codigo"`
		}
	}
	if err = json.Unmarshal(b, &datos); err != nil {
		t.Fatal(err)
	}
	for _, m := range datos.Municipios {
		if m.Codigo == "18102" {
			if m.Centro != "PINOS_PUENTE_MOCLIN" || m.Equipo != "06" {
				t.Fatal(m)
			}
			return
		}
	}
	t.Fatal("Íllora no está en el catálogo inicial")
}
