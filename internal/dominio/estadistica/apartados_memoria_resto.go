package estadistica

import "time"

func calcularApartadosResto(m *Memoria, exp Expediente, menores Dato[[]Menor], corte, conocido time.Time, p Parametros) error {
	if !menores.Conocido {
		m.Apartados[CoordinacionRecursos].Total.FamiliasSinRelacionMenores++
		m.Apartados[AtencionTemprana].Total.FamiliasSinRelacionMenores++
		m.Apartados[FactoresMenores].Total.FamiliasSinRelacionMenores++
		m.Apartados[TipologiasDesproteccion].Total.FamiliasSinRelacionMenores++
		return nil
	}
	computables := menoresComputables(menores, m.Apartados[MenoresPorSexo].Contrato.FechaEdad)
	recursos, _, err := exp.RecursosCoordinacion.ValorEn(corte, conocido)
	if err != nil {
		return err
	}
	marcados := map[string]bool{}
	coordinacion, err := factoresEn(exp, "coordinacion", corte, conocido)
	if err != nil {
		return err
	}
	if recursos.Conocido {
		for _, c := range recursos.Valor {
			marcados[c] = true
		}
	}
	for _, men := range computables.Valor {
		g := Grupo{Subprograma: men.Subprograma, Sexo: men.Sexo}
		r := m.Apartados[CoordinacionRecursos]
		r.Total.Menores++
		for _, recurso := range p.RecursosCoordinacion {
			rg := g
			rg.Estado = recurso
			asegurarCelda(r, rg, Menores)
			if !recursos.Conocido {
				r.sumar(rg, Menores, true)
			} else if marcados[recurso] && !excluidoDeRecurso(coordinacion.Valor, recurso, men.ID) {
				r.sumar(rg, Menores, rg.Sexo == "" || rg.Subprograma == "")
			}
		}
		tr := m.Apartados[AtencionTemprana]
		tr.Total.Menores++
		at := men.PropuestaAtencionTemprana
		if !at.Conocido || g.Sexo == "" || g.Subprograma == "" {
			tr.sumar(g, Menores, true)
		} else if at.Valor {
			g.Estado = "si"
			tr.sumar(g, Menores, false)
		} else {
			g.Estado = "no"
			asegurarCelda(tr, g, Menores)
		}
	}
	if err := sumarFactoresMenores(m, exp, computables.Valor, corte, conocido, p.FactoresMenores, FactoresMenores); err != nil {
		return err
	}
	if err := sumarFactoresMenores(m, exp, computables.Valor, corte, conocido, p.TipologiasDesproteccion, TipologiasDesproteccion); err != nil {
		return err
	}
	return nil
}

func sumarFactoresMenores(m *Memoria, exp Expediente, menores []Menor, corte, conocido time.Time, codigos []string, a Apartado) error {
	lista := "factor_menor"
	if a == TipologiasDesproteccion {
		lista = "tipologia"
	}
	d, err := factoresEn(exp, lista, corte, conocido)
	if err != nil {
		return err
	}
	observados := map[string]Factor{}
	if d.Conocido {
		for _, f := range d.Valor {
			observados[f.Codigo] = f
		}
	}
	r := m.Apartados[a]
	for _, men := range menores {
		r.Total.Menores++
		for _, codigo := range codigos {
			g := Grupo{Subprograma: men.Subprograma, Sexo: men.Sexo, Estado: codigo}
			f, ok := observados[codigo]
			if !d.Conocido || (ok && !f.Presente.Conocido) {
				r.sumar(g, Menores, true)
				continue
			}
			if ok && f.Presente.Valor && !contiene(f.MenoresExcluidos, men.ID) {
				r.sumar(g, Menores, g.Sexo == "" || g.Subprograma == "")
			}
		}
	}
	return nil
}

// Cada lista conserva su valoración independiente. Un código puede aparecer
// en varios catálogos; las exclusiones pertenecen a su lista y revisión.
func factoresEn(exp Expediente, lista string, corte, conocido time.Time) (Dato[[]Factor], error) {
	d, _, err := exp.Factores.ValorEn(corte, conocido)
	if err != nil {
		return d, err
	}
	if len(exp.ListasConocidas) > 0 {
		listas, _, err := exp.ListasConocidas.ValorEn(corte, conocido)
		if err != nil {
			return Dato[[]Factor]{}, err
		}
		if !listas.Conocido || !contiene(listas.Valor, lista) {
			return Dato[[]Factor]{}, nil
		}
	}
	filtrados := []Factor{}
	for _, f := range d.Valor {
		if f.Lista == "" || f.Lista == lista {
			filtrados = append(filtrados, f)
		}
	}
	d.Valor = filtrados
	return d, nil
}

func excluidoDeRecurso(factores []Factor, codigo, menor string) bool {
	for _, f := range factores {
		if f.Codigo == codigo && contiene(f.MenoresExcluidos, menor) {
			return true
		}
	}
	return false
}

func contiene(v []string, buscado string) bool {
	for _, x := range v {
		if x == buscado {
			return true
		}
	}
	return false
}

func sumarResponsables(m *Memoria, exp Expediente, corte, conocido time.Time, codigos []string) error {
	d, _, err := exp.Responsables.ValorEn(corte, conocido)
	if err != nil {
		return err
	}
	r := m.Apartados[ResponsablesDesproteccion]
	r.Total.Familias++
	seleccionados := map[string]bool{}
	if d.Conocido {
		for _, c := range d.Valor {
			seleccionados[c] = true
		}
	}
	for _, codigo := range codigos {
		g := Grupo{Estado: codigo}
		if !d.Conocido {
			r.sumar(g, Familias, true)
		} else if seleccionados[codigo] {
			r.sumar(g, Familias, false)
		}
	}
	return nil
}
