package estadistica_test

import (
	"encoding/json"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"strings"
	"testing"
)

func TestSupresionPrimariaSecundariaYMargenes(t *testing.T) {
	n, p := 37, 1234
	celdas := []d.CeldaConsulta{}
	// Matriz independiente 2x2: [1,8;9,19]. Una pequeña exige ocultar
	// la siguiente de fila y columna; el cierre debe ocultar las cuatro.
	for i, v := range []int{1, 8, 9, 19} {
		sexo := d.Sexo("h")
		if i%2 == 1 {
			sexo = d.Sexo("m")
		}
		origen := "SSSC"
		if i >= 2 {
			origen = "SPM"
		}
		celdas = append(celdas, d.CeldaConsulta{Grupo: d.GrupoConsulta{Sexo: sexo, Origen: origen}, Unidad: d.Unidad("familias"), Numerador: v, Denominador: &n, Porcentaje: &p, SinDato: 0})
	}
	original := d.CifrasConsulta{Apartados: []d.ApartadoConsulta{{Codigo: "1", Total: d.TotalConsulta{Familias: 37}, Celdas: celdas, Avisos: map[string]int{"estadistica.aceptacion_sin_dato": 1}}}}
	r := d.SuprimirCifras(original, 3)
	for _, c := range r.Apartados[0].Celdas {
		if c.Numerador != d.CifraSuprimida || c.Denominador != nil || c.Porcentaje != nil {
			t.Fatal("reconstrucción de celda", c)
		}
	}
	if original.Apartados[0].Celdas[0].Numerador != 1 || original.Apartados[0].Total.Familias != 37 {
		t.Fatal("modifica fuente")
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, dato := range []string{`"numerador":1,`, `"familias":37`, `"centesimas":1234`, `:-1`, `"estadistica.aceptacion_sin_dato":1`} {
		if strings.Contains(string(b), dato) {
			t.Fatal("filtración", dato, string(b))
		}
	}
	if !strings.Contains(string(b), `"numerador":null`) || !strings.Contains(string(b), `"familias":null`) {
		t.Fatal("ausencia ambigua", string(b))
	}
}
func TestUmbralCeroLimiteYEleccionMenor(t *testing.T) {
	for _, umbral := range []int{3, 5} {
		r := d.SuprimirCifras(d.CifrasConsulta{Apartados: []d.ApartadoConsulta{{Celdas: []d.CeldaConsulta{
			{Numerador: 0, Grupo: d.GrupoConsulta{Origen: "a"}},
			{Numerador: umbral, Grupo: d.GrupoConsulta{Origen: "b"}},
			{Numerador: umbral + 9, Grupo: d.GrupoConsulta{Origen: "c"}},
		}}}}, umbral)
		c := r.Apartados[0].Celdas
		if c[0].Numerador != -1 || c[1].Numerador != -1 || c[2].Numerador != umbral+9 {
			t.Fatal("secundaria no elige siguiente menor", c)
		}
	}
}

func TestCifrasSegurasConservanTotalesYPorcentajes(t *testing.T) {
	n, p := 12, 5000
	r := d.SuprimirCifras(d.CifrasConsulta{Apartados: []d.ApartadoConsulta{{Total: d.TotalConsulta{Familias: 12}, Celdas: []d.CeldaConsulta{{Numerador: 6, Denominador: &n, Porcentaje: &p, SinDato: 0}}}}}, 3)
	c := r.Apartados[0].Celdas[0]
	if c.Numerador != 6 || c.Denominador == nil || *c.Denominador != 12 || c.Porcentaje == nil || *c.Porcentaje != 5000 || r.Apartados[0].Total.Familias != 12 {
		t.Fatal("supresión innecesaria", r)
	}
	if c.SinDato != d.CifraSuprimida || r.Apartados[0].Total.Menores != d.CifraSuprimida {
		t.Fatal("cero expuesto")
	}
}
