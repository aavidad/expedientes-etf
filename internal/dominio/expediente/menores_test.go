package expediente

import (
	"testing"
	"time"
)

func TestMenoresValidacionLimitesYDesconocidos(t *testing.T) {
	fecha := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	c := CatalogosMenores{Opciones: map[string][]string{"nivel_escolarizacion": {"PRIMARIA"}, "centro_proteccion": {"CP01"}}}
	menor := Menor{Orden: 1, Sexo: "M", Subprograma: "RIESGO_SOCIAL"}
	lista := []Menor{menor}
	p := CambioMenores{Version: 2, Desde: "2026-02-03", Menores: &lista}
	if err := p.Validar(c, fecha, fecha); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre  string
		cambiar func(*CambioMenores)
	}{
		{"dieciséis", func(p *CambioMenores) {
			for i := 2; i <= 16; i++ {
				m := menor
				m.Orden = i
				*p.Menores = append(*p.Menores, m)
			}
		}},
		{"orden duplicado", func(p *CambioMenores) { *p.Menores = append(*p.Menores, menor) }},
		{"id duplicado", func(p *CambioMenores) {
			m := menor
			m.ID = "00000000-0000-4000-8000-000000000001"
			n := m
			n.Orden = 2
			*p.Menores = []Menor{m, n}
		}},
		{"nacimiento futuro", func(p *CambioMenores) { n := "2026-02-04"; (*p.Menores)[0].Nacimiento = &n }},
		{"centro ambiguo", func(p *CambioMenores) {
			s := "CP01"
			(*p.Menores)[0].CentroProteccionCodigo = &s
			(*p.Menores)[0].CentroProteccionTexto = "Centro de acogida"
		}},
		{"catálogo desconocido", func(p *CambioMenores) { s := "INEXISTENTE"; (*p.Menores)[0].Escolarizacion = &s }},
		{"vigencia previa", func(p *CambioMenores) { p.Desde = "2026-02-02" }},
		{"colección ausente", func(p *CambioMenores) { p.Menores = nil }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ms := []Menor{menor}
			p := CambioMenores{Version: 2, Desde: "2026-02-03", Menores: &ms}
			caso.cambiar(&p)
			if err := p.Validar(c, fecha, fecha); err != ErrMenores {
				t.Fatal(err)
			}
		})
	}
	ms := make([]Menor, 15)
	for i := range ms {
		ms[i] = menor
		ms[i].Orden = i + 1
	}
	p.Menores = &ms
	if err := p.Validar(c, fecha, fecha); err != nil {
		t.Fatal(err)
	}
	vacia := []Menor{}
	p.Menores = &vacia
	if err := p.Validar(c, fecha, fecha); err != nil {
		t.Fatal(err)
	}
}
