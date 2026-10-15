package catalogo

import (
	"testing"
	"time"
)

func TestUbicacionValidaIdentidadVigenciaYVariantes(t *testing.T) {
	fecha := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, codigo := range []string{"00", "13", "1", "xx", "0A"} {
		if (Ubicacion{Tipo: "equipo", Codigo: codigo, ActivoDesde: fecha}).Validar() == nil {
			t.Fatal(codigo)
		}
	}
	if (Ubicacion{Tipo: "equipo", Codigo: "12", ActivoDesde: fecha}).Validar() != nil {
		t.Fatal("equipo 12")
	}
	if (Ubicacion{Tipo: "centro", Codigo: "VEGA", ActivoDesde: fecha, ActivoHasta: &fecha}).Validar() == nil {
		t.Fatal("intervalo vacío")
	}
	if (Ubicacion{Tipo: "municipio", ID: "00000000-0000-4000-8000-000000000001", Codigo: "18102", Variantes: map[string][]string{"es": {"Íllora", "Íllora"}}}).Validar() == nil {
		t.Fatal("grafías duplicadas")
	}
}
