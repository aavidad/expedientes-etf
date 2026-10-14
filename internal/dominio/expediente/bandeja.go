package expediente

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type FiltroBandeja struct {
	Estado         string `json:"estado"`
	EquipoID       string `json:"equipo_id"`
	MunicipioID    string `json:"municipio_id"`
	Desde          string `json:"desde"`
	Hasta          string `json:"hasta"`
	Busqueda       string `json:"busqueda"`
	Limite         int    `json:"limite"`
	Desplazamiento int    `json:"desplazamiento"`
}

func (p FiltroBandeja) Validar() error {
	if !slices.Contains([]string{"", DerivacionRecibida, "aceptada", NoAceptada, ListaEspera}, p.Estado) || p.Limite < 1 || p.Limite > 100 || p.Desplazamiento < 0 || p.Desplazamiento > 100000 || utf8.RuneCountInString(p.Busqueda) > 100 || strings.ContainsAny(p.Busqueda, "\n\r\x00") {
		return ErrPeticion
	}
	for _, id := range []string{p.EquipoID, p.MunicipioID} {
		if id != "" && !IDValido(id) {
			return ErrPeticion
		}
	}
	for _, fecha := range []string{p.Desde, p.Hasta} {
		if fecha != "" {
			if valor, err := time.Parse("2006-01-02", fecha); err != nil || valor.Year() < 1 {
				return ErrPeticion
			}
		}
	}
	if p.Desde != "" && p.Hasta != "" && p.Desde > p.Hasta {
		return ErrPeticion
	}
	return nil
}

type FilaBandeja struct {
	CargadoParaRevision bool      `json:"cargado_para_revision"`
	ID                  string    `json:"id"`
	Numero              string    `json:"numero"`
	Iniciales           string    `json:"iniciales"`
	Estado              string    `json:"estado"`
	FechaRecepcion      time.Time `json:"fecha_recepcion"`
	FechaAceptacion     string    `json:"fecha_aceptacion"`
	EquipoID            string    `json:"equipo_id"`
	MunicipioID         string    `json:"municipio_id"`
	MunicipioPropuesto  string    `json:"municipio_propuesto"`
	Subprograma         string    `json:"subprograma"`
}

type OpcionBandeja struct {
	ID       string `json:"id"`
	Etiqueta string `json:"etiqueta"`
	Codigo   string `json:"codigo"`
}
type ResultadoBandeja struct {
	Filas      []FilaBandeja   `json:"filas"`
	Total      int             `json:"total"`
	Resumen    map[string]int  `json:"resumen"`
	Equipos    []OpcionBandeja `json:"equipos"`
	Municipios []OpcionBandeja `json:"municipios"`
}
