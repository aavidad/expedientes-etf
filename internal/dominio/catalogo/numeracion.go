package catalogo

import (
	"errors"
	"strconv"
	"strings"
)

var ErrNumeracion = errors.New("catalogo.numeracion_invalida")

// ReglaNumeracion conserva la versión utilizada y el formato publicado.
type ReglaNumeracion struct {
	Version           Version  `json:"version"`
	Componentes       []string `json:"componentes"`
	CifrasEquipo      int      `json:"cifras_equipo"`
	CifrasCorrelativo int      `json:"cifras_correlativo"`
	CifrasAnio        int      `json:"cifras_anio"`
	Separador         string   `json:"separador"`
}

func (r ReglaNumeracion) Limite() int {
	if r.CifrasCorrelativo < 1 || r.CifrasCorrelativo > 9 {
		return 0
	}
	limite := 1
	for i := 0; i < r.CifrasCorrelativo; i++ {
		limite *= 10
	}
	return limite - 1
}

func (r ReglaNumeracion) Formatear(equipo string, correlativo, anio int) (string, error) {
	if r.CifrasEquipo < 1 || r.CifrasEquipo > 4 || r.Limite() == 0 || correlativo < 1 || correlativo > r.Limite() || anio < 1 || anio > 9999 || (r.CifrasAnio != 2 && r.CifrasAnio != 4) || (r.Separador != "-" && r.Separador != "/" && r.Separador != ".") || len(r.Componentes) != 3 {
		return "", ErrNumeracion
	}
	n, err := strconv.Atoi(equipo)
	if err != nil || n < 1 {
		return "", ErrNumeracion
	}
	rellenar := func(n, cifras int) (string, error) {
		s := strconv.Itoa(n)
		if len(s) > cifras {
			return "", ErrNumeracion
		}
		return strings.Repeat("0", cifras-len(s)) + s, nil
	}
	if r.CifrasAnio == 2 {
		anio %= 100
	}
	valores := map[string]int{"equipo": n, "correlativo": correlativo, "anio": anio}
	cifras := map[string]int{"equipo": r.CifrasEquipo, "correlativo": r.CifrasCorrelativo, "anio": r.CifrasAnio}
	partes := make([]string, 0, 3)
	vistos := map[string]bool{}
	for _, c := range r.Componentes {
		if _, ok := valores[c]; !ok || vistos[c] {
			return "", ErrNumeracion
		}
		vistos[c] = true
		p, err := rellenar(valores[c], cifras[c])
		if err != nil {
			return "", err
		}
		partes = append(partes, p)
	}
	return strings.Join(partes, r.Separador), nil
}
