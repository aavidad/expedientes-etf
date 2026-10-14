package expediente

import (
	"reflect"
	"testing"
	"time"
)

func TestIntervencionValidacionHerenciaYDesconocidos(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	fase := "EVALUACION"
	c := CatalogosIntervencion{Opciones: map[string]map[string]bool{"fase_intervencion": {"EVALUACION": false}, "factor_menor": {"ABSENTISMO": false}, "coordinacion": {"OTROS": true}}}
	nueva := func() CambioIntervencion {
		m := []MarcaIntervencion{{Codigo: "ABSENTISMO", MenoresExcluidos: []string{id}}}
		vacia := []MarcaIntervencion{}
		return CambioIntervencion{Version: 2, Desde: "2026-02-03", Intervencion: &Intervencion{Fase: &fase, Listas: map[string]*[]MarcaIntervencion{"factor_familia": nil, "factor_menor": &m, "tipologia": &vacia, "coordinacion": nil}}}
	}
	fecha := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	p := nueva()
	if err := p.Validar(c, fecha, fecha); err != nil {
		t.Fatal(err)
	}
	r := p.Intervencion.Aplicar(id)
	if r.Listas["factor_familia"] != nil || r.Listas["tipologia"] == nil || len(*r.Listas["factor_menor"]) != 0 {
		t.Fatal(r)
	}
	r = p.Intervencion.Aplicar("00000000-0000-4000-8000-000000000002")
	if !reflect.DeepEqual(*r.Listas["factor_menor"], []string{"ABSENTISMO"}) {
		t.Fatal(r)
	}
	for nombre, cambiar := range map[string]func(*CambioIntervencion){
		"lista desconocida":  func(p *CambioIntervencion) { p.Intervencion.Listas["desconocida"] = nil },
		"lista ausente":      func(p *CambioIntervencion) { delete(p.Intervencion.Listas, "tipologia") },
		"fase inválida":      func(p *CambioIntervencion) { s := "OTRA"; p.Intervencion.Fase = &s },
		"marca repetida":     func(p *CambioIntervencion) { m := p.Intervencion.Listas["factor_menor"]; *m = append(*m, (*m)[0]) },
		"texto no permitido": func(p *CambioIntervencion) { (*p.Intervencion.Listas["factor_menor"])[0].TextoOtros = "Observación" },
		"id inválido": func(p *CambioIntervencion) {
			(*p.Intervencion.Listas["factor_menor"])[0].MenoresExcluidos = []string{"ajeno"}
		},
		"exclusión repetida": func(p *CambioIntervencion) {
			(*p.Intervencion.Listas["factor_menor"])[0].MenoresExcluidos = []string{id, id}
		},
		"fecha anterior":   func(p *CambioIntervencion) { p.Desde = "2026-02-02" },
		"fecha futura":     func(p *CambioIntervencion) { p.Desde = "2026-02-04" },
		"versión inválida": func(p *CambioIntervencion) { p.Version = 0 },
		"sin intervención": func(p *CambioIntervencion) { p.Intervencion = nil },
	} {
		t.Run(nombre, func(t *testing.T) {
			p := nueva()
			cambiar(&p)
			if err := p.Validar(c, fecha, fecha); err != ErrIntervencion {
				t.Fatal(err)
			}
		})
	}
}
