package catalogo

import (
	"errors"
	"time"
)

var (
	ErrVigencia     = errors.New("catalogo.vigencia_invalida")
	ErrAmbito       = errors.New("catalogo.ambito_invalido")
	ErrNoDisponible = errors.New("catalogo.version_no_disponible")
	ErrPersistencia = errors.New("catalogo.no_disponible")
)

type Actor struct {
	UsuarioID string
	Perfiles  []string
	EquipoID  string
}

type Version struct {
	ID           string     `json:"id"`
	Ambito       string     `json:"ambito"`
	Numero       int        `json:"numero"`
	VigenteDesde time.Time  `json:"vigente_desde"`
	VigenteHasta *time.Time `json:"vigente_hasta"`
	Estado       string     `json:"estado"`
	Motivo       string     `json:"motivo"`
}

func AmbitoValido(ambito string) bool {
	switch ambito {
	case "listas", "equivalencias", "plazos", "tramos", "numeracion", "escalas", "asignacion_municipios", "tipos_documento":
		return true
	}
	return false
}

// FechaCivil conserva el día civil recibido y elimina hora y zona horaria.
func FechaCivil(fecha time.Time) time.Time {
	return time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 0, 0, 0, 0, time.UTC)
}

func (v Version) Validar() error {
	if !AmbitoValido(v.Ambito) {
		return ErrAmbito
	}
	if v.VigenteDesde.IsZero() || (v.VigenteHasta != nil && !FechaCivil(*v.VigenteHasta).After(FechaCivil(v.VigenteDesde))) {
		return ErrVigencia
	}
	return nil
}

// Rige usa intervalos [desde, hasta): el día del cambio pertenece a la nueva versión.
func (v Version) Rige(fecha time.Time) bool {
	if v.Validar() != nil || fecha.IsZero() || v.Estado != "publicada" {
		return false
	}
	fecha = FechaCivil(fecha)
	return !fecha.Before(FechaCivil(v.VigenteDesde)) && (v.VigenteHasta == nil || fecha.Before(FechaCivil(*v.VigenteHasta)))
}

func FechaMemoria(anio int) (time.Time, error) {
	if anio < 1 || anio > 9999 {
		return time.Time{}, ErrVigencia
	}
	return time.Date(anio, time.December, 31, 0, 0, 0, 0, time.UTC), nil
}
