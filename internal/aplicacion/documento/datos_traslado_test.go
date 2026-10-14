package documento

import (
	d "github.com/aavidad/expedientes-etf/internal/dominio/documento"
	e "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"testing"
)

func TestActaTrasladoUsaFamiliaYAdultosCopiados(t *testing.T) {
	fecha := "1985-04-12"
	a := &d.Aceptacion{Direccion: "Calle del Álamo 3", Familia: "Contacto anterior", Padre: d.Persona{Nombre: "Padre anterior"}, Adultos: []d.Adulto{{Persona: d.Persona{Nombre: "Adulto anterior"}}}}
	copia := e.CopiaTraslado{Familia: e.Familia{Direccion: "Calle Olivo 12"}, Adultos: []e.Adulto{
		{Nombre: "Teresa", Apellidos: "López Rivas", Papel: "MADRE", Nacimiento: &fecha},
		{Nombre: "Rafael", Apellidos: "López", Papel: "ABUELO"},
		{Papel: "PADRE"},
	}}
	if err := rellenarCopiaTraslado(a, copia, map[string]string{"papel_parental.ABUELO": "Abuelo"}); err != nil {
		t.Fatal(err)
	}
	if a.Direccion != "Calle Olivo 12" || a.Familia != "Teresa López Rivas; Rafael López" || a.Madre.Nombre != "Teresa López Rivas" || a.Madre.Nacimiento != fecha || a.Padre.Nombre != "" || len(a.Adultos) != 1 || a.Adultos[0].Nombre != "Rafael López" || a.Adultos[0].Parentesco != "Abuelo" {
		t.Fatal(a)
	}
	if err := rellenarCopiaTraslado(a, copia, nil); err != d.ErrNoDisponible {
		t.Fatal("parentesco sin catálogo", err)
	}
}
