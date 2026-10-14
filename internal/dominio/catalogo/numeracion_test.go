package catalogo

import "testing"

func TestNumeracionD1YLimites(t *testing.T) {
	r := ReglaNumeracion{Componentes: []string{"equipo", "correlativo", "anio"}, CifrasEquipo: 2, CifrasCorrelativo: 2, CifrasAnio: 2, Separador: "-"}
	for _, c := range []struct {
		n, anio int
		numero  string
	}{{1, 2026, "08-01-26"}, {99, 2026, "08-99-26"}, {1, 2100, "08-01-00"}} {
		n, err := r.Formatear("08", c.n, c.anio)
		if err != nil || n != c.numero {
			t.Fatal(n, err)
		}
	}
	for _, n := range []int{0, 100} {
		if _, err := r.Formatear("08", n, 2026); err == nil {
			t.Fatal(n)
		}
	}
	r.Componentes = []string{"anio", "equipo", "correlativo"}
	r.Separador = "/"
	r.CifrasCorrelativo = 3
	r.CifrasAnio = 4
	if n, err := r.Formatear("08", 1, 2027); err != nil || n != "2027/08/001" {
		t.Fatal(n, err)
	}
}
