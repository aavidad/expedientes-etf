package expediente

import (
	"errors"
	"slices"
	"time"
)

var ErrIntervencion = errors.New("expediente.intervencion_invalida")

// MarcaIntervencion se recoge por familia y se hereda salvo exclusión expresa.
type MarcaIntervencion struct {
	Codigo           string   `json:"codigo"`
	TextoOtros       string   `json:"texto_otros,omitempty"`
	MenoresExcluidos []string `json:"menores_excluidos"`
}
type Intervencion struct {
	Fase *string `json:"fase"`
	// Una lista nula es desconocida; una lista vacía confirma que no hay marcas.
	Listas map[string]*[]MarcaIntervencion `json:"listas"`
}
type CambioIntervencion struct {
	Version      int           `json:"version"`
	Desde        string        `json:"vigente_desde"`
	Intervencion *Intervencion `json:"intervencion"`
}
type ResultadoIntervencion = ResultadoFamilia
type FichaIntervencion struct {
	ID           string        `json:"id"`
	EquipoID     string        `json:"equipo_id"`
	Estado       string        `json:"estado"`
	Version      int           `json:"version"`
	Aceptacion   time.Time     `json:"fecha_aceptacion"`
	Intervencion *Intervencion `json:"intervencion"`
}
type IntervencionMenor struct {
	MenorID string               `json:"menor_id"`
	Listas  map[string]*[]string `json:"listas"`
}

// HistoriaIntervencion sólo contiene códigos y vínculos, sin textos libres.
type HistoriaIntervencion struct {
	FaseDesde    *string             `json:"fase_desde"`
	Intervencion *Intervencion       `json:"intervencion"`
	Vigencia     *Vigencia           `json:"vigencia"`
	Menores      []IntervencionMenor `json:"menores"`
}
type CatalogosIntervencion struct {
	Opciones map[string]map[string]bool
}

func (a Actor) AutorizarIntervencion(escritura bool) error { return a.AutorizarFamilia(escritura) }
func ListasIntervencion() []string {
	return []string{"coordinacion", "factor_familia", "factor_menor", "tipologia"}
}
func (p CambioIntervencion) Validar(c CatalogosIntervencion, aceptacion, ahora time.Time) error {
	fecha, err := time.Parse("2006-01-02", p.Desde)
	if err != nil || fecha.Year() < 1 || fecha.Before(aceptacion) || p.Desde > FechaCivil(ahora) || p.Version < 1 || p.Intervencion == nil {
		return ErrIntervencion
	}
	i := p.Intervencion
	if i.Fase != nil {
		if _, ok := c.Opciones["fase_intervencion"][*i.Fase]; !ok {
			return ErrIntervencion
		}
	}
	if len(i.Listas) != len(ListasIntervencion()) {
		return ErrIntervencion
	}
	for _, lista := range ListasIntervencion() {
		marcas, ok := i.Listas[lista]
		if !ok {
			return ErrIntervencion
		}
		if marcas == nil {
			continue
		}
		codigos := map[string]bool{}
		for _, m := range *marcas {
			texto, ok := c.Opciones[lista][m.Codigo]
			if !ok || codigos[m.Codigo] || !textoFamilia(m.TextoOtros, 4000) || !texto && m.TextoOtros != "" || len(m.MenoresExcluidos) > 15 {
				return ErrIntervencion
			}
			codigos[m.Codigo] = true
			ids := map[string]bool{}
			for _, id := range m.MenoresExcluidos {
				if !IDValido(id) || ids[id] {
					return ErrIntervencion
				}
				ids[id] = true
			}
		}
	}
	return nil
}

// Normalizada copia las colecciones antes de ordenarlas para la idempotencia.
func (i Intervencion) Normalizada() Intervencion {
	r := Intervencion{Fase: i.Fase, Listas: map[string]*[]MarcaIntervencion{}}
	for lista, marcas := range i.Listas {
		if marcas == nil {
			r.Listas[lista] = nil
			continue
		}
		copia := slices.Clone(*marcas)
		if copia == nil {
			copia = []MarcaIntervencion{}
		}
		for n := range copia {
			copia[n].MenoresExcluidos = slices.Clone(copia[n].MenoresExcluidos)
			if copia[n].MenoresExcluidos == nil {
				copia[n].MenoresExcluidos = []string{}
			}
			slices.Sort(copia[n].MenoresExcluidos)
		}
		slices.SortFunc(copia, func(a, b MarcaIntervencion) int {
			if a.Codigo < b.Codigo {
				return -1
			}
			if a.Codigo > b.Codigo {
				return 1
			}
			return 0
		})
		r.Listas[lista] = &copia
	}
	return r
}
func (i Intervencion) Exclusiones() []string {
	ids := []string{}
	for _, marcas := range i.Listas {
		if marcas != nil {
			for _, m := range *marcas {
				ids = append(ids, m.MenoresExcluidos...)
			}
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}
func (i Intervencion) Aplicar(menorID string) IntervencionMenor {
	r := IntervencionMenor{MenorID: menorID, Listas: map[string]*[]string{}}
	for lista, marcas := range i.Listas {
		if marcas == nil {
			r.Listas[lista] = nil
			continue
		}
		codigos := []string{}
		for _, m := range *marcas {
			if !slices.Contains(m.MenoresExcluidos, menorID) {
				codigos = append(codigos, m.Codigo)
			}
		}
		r.Listas[lista] = &codigos
	}
	return r
}
