package expediente

import (
	"testing"
)

func TestAvisosFechasCivilesPlazoYFase(t *testing.T) {
	fase := "EVALUACION"
	desde := "2026-03-01"
	d := DatosAvisos{ID: "ab60479d-b778-407f-a83e-012345678911", Estado: Aceptada, Aceptacion: desde, Subprograma: "REUNIFICACION", Intervencion: &Intervencion{Fase: &fase}, FaseDesde: &desde}
	reglas := ReglasAvisos{ConsentimientoDias: 30, Duraciones: map[string]int{"REUNIFICACION:EVALUACION": 45}}
	for _, c := range []struct {
		hoy string
		n   int
	}{{"2026-03-31", 0}, {"2026-04-01", 1}, {"2026-04-15", 1}, {"2026-04-16", 2}} {
		r, err := CalcularAvisos(d, c.hoy, reglas, false)
		if err != nil || len(r) != c.n {
			t.Fatal(c, r, err)
		}
	}
	firmado := "2026-03-02"
	d.Consentimiento = &firmado
	r, err := CalcularAvisos(d, "2026-04-16", reglas, false)
	if err != nil || len(r) != 1 || r[0].Codigo != "fase" || r[0].FechaLimite != "2026-04-15" {
		t.Fatal(r, err)
	}
	reglas.Duraciones["REUNIFICACION:EVALUACION"] = 60
	r, err = CalcularAvisos(d, "2026-04-16", reglas, false)
	if err != nil || len(r) != 0 {
		t.Fatal(r, err)
	}
	d.Estado = "cerrado"
	d.Consentimiento = nil
	r, err = CalcularAvisos(d, "2026-12-01", reglas, false)
	if err != nil || len(r) != 0 {
		t.Fatal(r, err)
	}
}

func TestAvisosMemoriaDistingueDesconocidoNoYVacio(t *testing.T) {
	composicion := "MONOPARENTAL_MADRE"
	economia := "SIN_DIFICULTADES"
	tramo := "MENOS_50"
	no := false
	vacia := []MarcaIntervencion{}
	d := DatosAvisos{ID: "ab60479d-b778-407f-a83e-012345678911", Estado: "cerrado", AdultosConocidos: true, Familia: FamiliaEstadistica{Composicion: &composicion, Numerosa: &no, Economia: &economia, Tramo: &tramo}, Intervencion: &Intervencion{Listas: map[string]*[]MarcaIntervencion{"coordinacion": &vacia, "factor_familia": &vacia, "factor_menor": &vacia, "tipologia": &vacia}}}
	r, err := CalcularAvisos(d, "2026-12-31", ReglasAvisos{}, true)
	if err != nil || len(r) != 0 {
		t.Fatal(r, err)
	}
	d.Familia.Numerosa = nil
	d.Intervencion.Listas["factor_menor"] = nil
	d.Menores = []MenorEstadistico{{ID: "ab60479d-b778-407f-a83e-012345678913", Sexo: "M", Subprograma: "RIESGO_SOCIAL", AtencionTemprana: &no}}
	r, err = CalcularAvisos(d, "2026-12-31", ReglasAvisos{}, true)
	if err != nil || len(r) != 3 || r[0].Apartado != "18" || r[1].Apartado != "25" || r[2].Apartado != "26" {
		t.Fatal(r, err)
	}
	if r[1].Campo != "fecha_nacimiento" || r[1].SujetoID != d.Menores[0].ID || r[1].Enlace != "/expedientes/"+d.ID+"#fecha_nacimiento-"+d.Menores[0].ID {
		t.Fatal(r[1])
	}
}
