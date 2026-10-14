package expediente

import (
	"math"
	"testing"
	"time"
)

func TestFamiliaTramosExactos(t *testing.T) {
	a, b, c, d := int64(50), int64(100), int64(150), int64(200)
	catalogo := CatalogosFamilia{Tramos: []TramoFamilia{{"A", 0, &a}, {"B", 50, &b}, {"C", 100, &c}, {"D", 150, &d}, {"E", 200, nil}}}
	for _, p := range []struct {
		valor    int64
		esperado string
	}{{0, "A"}, {4999, "A"}, {5000, "B"}, {9999, "B"}, {10000, "C"}, {15000, "D"}, {20000, "E"}, {math.MaxInt64, "E"}} {
		r := catalogo.CalcularTramo(p.valor, 10000)
		if r == nil || *r != p.esperado {
			t.Fatal(p, r)
		}
	}
	if catalogo.CalcularTramo(1, 0) != nil {
		t.Fatal("SMI desconocido")
	}
	t.Log("Tramos: límites exactos, cero conocido y máximo int64 sin desbordamiento")
}
func TestFamiliaVigenciaYValidacion(t *testing.T) {
	cat := CatalogosFamilia{Opciones: map[string][]string{"papel_parental": {"MADRE"}, "sexo": {"M"}}}
	fecha := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	p := CambioFamilia{Version: 2, Desde: "2026-02-03", Familia: &Familia{}}
	if err := p.Validar(cat, fecha, fecha); err != nil {
		t.Fatal(err)
	}
	for _, desde := range []string{"2026-02-02", "2026-02-04", "2026-02-30", "0000-01-01"} {
		p.Desde = desde
		if p.Validar(cat, fecha, fecha) == nil {
			t.Fatal(desde)
		}
	}
	p.Desde = "2026-02-03"
	ingresos := int64(0)
	tramo := "DESCONOCIDO"
	p.Familia = &Familia{Ingresos: &ingresos, Tramo: &tramo}
	if p.Validar(cat, fecha, fecha) == nil {
		t.Fatal("cantidad y tramo simultáneos")
	}
	adultos := []Adulto{{Orden: 1, Papel: "MADRE", Sexo: "M"}, {Orden: 2, Papel: "MADRE", Sexo: "M"}}
	p.Familia = nil
	p.Adultos = &adultos
	if err := p.Validar(cat, fecha, fecha); err != nil {
		t.Fatal("dos adultos nuevos", err)
	}
	adultos[1].Orden = 1
	if p.Validar(cat, fecha, fecha) == nil {
		t.Fatal("orden duplicado")
	}
}
