package expediente

import "testing"

func TestFiltroBandejaRechazaEntradasInvalidas(t *testing.T) {
	for _, p := range []FiltroBandeja{{Limite: 0}, {Limite: 101}, {Limite: 20, Desplazamiento: -1}, {Limite: 20, Estado: "cerrada"}, {Limite: 20, EquipoID: "ajeno"}, {Limite: 20, Desde: "2026-02-30"}, {Limite: 20, Desde: "2026-10-20", Hasta: "2026-10-01"}, {Limite: 20, Busqueda: "familia\n"}} {
		if p.Validar() == nil {
			t.Fatalf("aceptado: %+v", p)
		}
	}
	if err := (FiltroBandeja{Limite: 20, Desde: "2026-01-01", Hasta: "2026-12-31", Estado: ListaEspera}).Validar(); err != nil {
		t.Fatal(err)
	}
}
