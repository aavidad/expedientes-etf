package estadistica

import (
	"encoding/json"
	"slices"
	"strconv"
)

// CifraSuprimida sólo existe en la proyección de Consulta. En JSON se entrega
// null y en Word se resuelve mediante el texto de privacidad del idioma.
const CifraSuprimida = -1

func cifraPublica(n int) *int {
	if n < 0 {
		return nil
	}
	return &n
}

func (c CeldaConsulta) MarshalJSON() ([]byte, error) {
	type alias CeldaConsulta
	return json.Marshal(struct {
		alias
		Numerador *int `json:"numerador"`
		SinDato   *int `json:"sin_dato"`
	}{alias(c), cifraPublica(c.Numerador), cifraPublica(c.SinDato)})
}
func (t TotalConsulta) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]*int{
		"familias": cifraPublica(t.Familias), "menores": cifraPublica(t.Menores),
		"adultos": cifraPublica(t.Adultos), "comisiones": cifraPublica(t.Comisiones),
		"informes": cifraPublica(t.Informes), "familias_sin_relacion_menores": cifraPublica(t.FamiliasSinRelacionMenores),
	})
}

// SuprimirCifras no modifica la foto cerrada ni los cálculos de otros perfiles.
// Los márgenes se retiran de toda la respuesta: un total o porcentaje de otro
// apartado puede permitir recuperar el valor de una celda suprimida.
func SuprimirCifras(r CifrasConsulta, umbral int) CifrasConsulta {
	r.Umbral = umbral
	r.Apartados = slices.Clone(r.Apartados)
	hayOcultas := false
	for i := range r.Apartados {
		a := &r.Apartados[i]
		a.Celdas = slices.Clone(a.Celdas)
		avisos := a.Avisos
		a.Avisos = map[string]int{}
		for clave, n := range avisos {
			if n >= umbral {
				a.Avisos[clave] = n
			} else if n > 0 {
				hayOcultas = true
			}
		}
		for _, c := range a.Celdas {
			if c.Numerador < umbral || (c.SinDato > 0 && c.SinDato < umbral) {
				hayOcultas = true
			}
		}
		grupos := map[string][]int{}
		claves := []string{}
		for j, c := range a.Celdas {
			g := c.Grupo
			dimensiones := []string{string(g.Subprograma), g.Origen, string(g.Sexo), g.Tramo, string(g.Motivo), g.Papel, g.Estado, g.Causa, g.Organismo}
			for eje := range dimensiones {
				copia := slices.Clone(dimensiones)
				copia[eje] = "*"
				b, _ := json.Marshal(copia)
				clave := string(c.Unidad) + ":" + strconv.Itoa(eje) + ":" + string(b)
				if _, ok := grupos[clave]; !ok {
					claves = append(claves, clave)
				}
				grupos[clave] = append(grupos[clave], j)
			}
		}
		ocultas := make([]bool, len(a.Celdas))
		for j, c := range a.Celdas {
			ocultas[j] = c.Numerador < umbral
		}
		// Se repite hasta que ocultar una columna no deje una única oculta en otra.
		for cambio := true; cambio; {
			cambio = false
			for _, clave := range claves {
				indices := grupos[clave]
				cantidad, siguiente := 0, -1
				for _, j := range indices {
					if ocultas[j] {
						cantidad++
						continue
					}
					if siguiente < 0 || a.Celdas[j].Numerador < a.Celdas[siguiente].Numerador {
						siguiente = j
					}
				}
				if cantidad == 1 && siguiente >= 0 {
					ocultas[siguiente] = true
					cambio = true
				}
			}
		}
		for j := range a.Celdas {
			c := &a.Celdas[j]
			if ocultas[j] {
				c.Numerador = CifraSuprimida
				hayOcultas = true
			}
			if c.SinDato < umbral {
				c.SinDato = CifraSuprimida
			}
			if c.Denominador != nil && *c.Denominador < umbral {
				c.Denominador = nil
				hayOcultas = true
			}
		}
		for _, n := range []*int{&a.Total.Familias, &a.Total.Menores, &a.Total.Adultos, &a.Total.Comisiones, &a.Total.Informes, &a.Total.FamiliasSinRelacionMenores} {
			if *n < umbral {
				if *n > 0 {
					hayOcultas = true
				}
				*n = CifraSuprimida
			}
		}
	}
	if hayOcultas {
		r.MargenesOcultos = true
		for i := range r.Apartados {
			a := &r.Apartados[i]
			a.Avisos = map[string]int{}
			a.Total = TotalConsulta{CifraSuprimida, CifraSuprimida, CifraSuprimida, CifraSuprimida, CifraSuprimida, CifraSuprimida}
			for j := range a.Celdas {
				a.Celdas[j].Denominador = nil
				a.Celdas[j].Porcentaje = nil
				a.Celdas[j].SinDato = CifraSuprimida
			}
		}
	}
	return r
}
