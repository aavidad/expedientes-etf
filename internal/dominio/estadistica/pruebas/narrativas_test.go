package estadistica_test

import (
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"strings"
	"testing"
)

func TestNarrativasLimitesPorApartado(t *testing.T) {
	n := d.Narrativas{D: strings.Repeat("línea\n", 24) + "final", G: strings.Repeat("línea\n", 29) + "final"}
	if err := n.Validar(); err != nil {
		t.Fatal(err)
	}
	n.D += "\nlínea extra"
	if err := n.Validar(); err != d.ErrNarrativas {
		t.Fatal("25 líneas", err)
	}
	n.D = ""
	n.G += "\nextra"
	if err := n.Validar(); err != d.ErrNarrativas {
		t.Fatal("30 líneas", err)
	}
	n.G = ""
	negativo := -1
	n.A.Psicologia.Hombres = &negativo
	if err := n.Validar(); err != d.ErrNarrativas {
		t.Fatal("recuento negativo", err)
	}
}
