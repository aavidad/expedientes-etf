package catalogo

import (
	"regexp"
	"strings"
	"time"
)

// Ubicacion cambia una entrada sin deducir el equipo del centro.
type Ubicacion struct {
	Tipo             string              `json:"tipo"`
	ID               string              `json:"id"`
	Codigo           string              `json:"codigo"`
	ActivoDesde      time.Time           `json:"activo_desde"`
	ActivoHasta      *time.Time          `json:"activo_hasta"`
	CentroID         *string             `json:"centro_ss_id"`
	EquipoID         *string             `json:"equipo_id"`
	AmbitoDiputacion bool                `json:"ambito_diputacion"`
	Variantes        map[string][]string `json:"variantes"`
}

func (u Ubicacion) ClaveTexto() string {
	if u.Tipo == "municipio" {
		return "municipio." + u.Codigo
	}
	if u.Tipo == "centro" {
		return "centro." + u.Codigo
	}
	return "equipo." + u.Codigo
}

var codigoEquipo = regexp.MustCompile(`^(0[1-9]|1[0-2])$`)

func (u Ubicacion) Validar() error {
	if u.ID != "" && !idFirmante.MatchString(u.ID) {
		return ErrEdicion
	}
	switch u.Tipo {
	case "equipo":
		if !codigoEquipo.MatchString(u.Codigo) {
			return ErrEdicion
		}
	case "centro":
		if !codigoOpcion.MatchString(u.Codigo) {
			return ErrEdicion
		}
	case "municipio":
		if u.ID == "" || u.Codigo == "" || len(u.Codigo) > 80 {
			return ErrEdicion
		}
		for _, id := range []*string{u.CentroID, u.EquipoID} {
			if id != nil && !idFirmante.MatchString(*id) {
				return ErrEdicion
			}
		}
	default:
		return ErrEdicion
	}
	if u.Tipo != "municipio" && (u.ActivoDesde.IsZero() || u.ActivoDesde.Year() < 1 || u.ActivoDesde.Year() > 9999 || u.ActivoHasta != nil && !FechaCivil(*u.ActivoHasta).After(FechaCivil(u.ActivoDesde))) {
		return ErrEdicion
	}
	if u.Tipo != "municipio" && len(u.Variantes) > 0 {
		return ErrEdicion
	}
	for idioma, variantes := range u.Variantes {
		if idioma != "es" && idioma != "en" || len(variantes) > 30 {
			return ErrEdicion
		}
		vistos := map[string]bool{}
		for _, texto := range variantes {
			if strings.TrimSpace(texto) == "" || len(texto) > 200 || vistos[texto] {
				return ErrEdicion
			}
			vistos[texto] = true
		}
	}
	return nil
}
