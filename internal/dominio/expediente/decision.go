package expediente

import (
	"errors"
	"strings"
	"time"
)

const (
	NoAceptada  = "no_aceptada"
	ListaEspera = "lista_espera"
)

var (
	ErrMotivo       = errors.New("expediente.motivo_obligatorio")
	ErrSubprograma  = errors.New("expediente.subprograma_invalido")
	ErrOrigenEspera = errors.New("expediente.espera_solo_comunitarios")
	ErrFechaEspera  = errors.New("expediente.aceptacion_espera_fecha_real")
)

type NoAceptacion struct {
	Version     int    `json:"version"`
	Fecha       string `json:"fecha"`
	Motivo      string `json:"motivo"`
	Subprograma string `json:"subprograma"`
}

type Espera struct {
	Version int `json:"version"`
}

type ResultadoDecision struct {
	ID          string `json:"id"`
	Estado      string `json:"estado"`
	Version     int    `json:"version"`
	Fecha       string `json:"fecha"`
	Subprograma string `json:"subprograma,omitempty"`
}

// La causa es texto libre de negocio; no se copia a auditoría ni a listados.
func (p NoAceptacion) Validar(d Derivacion) (NoAceptacion, error) {
	if d.Estado != DerivacionRecibida && d.Estado != ListaEspera {
		return p, ErrEstado
	}
	f, err := time.Parse("2006-01-02", p.Fecha)
	if err != nil || f.Year() < 1 || p.Version < 1 {
		return p, ErrPeticion
	}
	if strings.TrimSpace(p.Motivo) == "" || len(p.Motivo) > 20000 {
		return p, ErrMotivo
	}
	if d.Datos.Extraccion.Tipo == "anexo3" {
		if p.Subprograma == "" {
			p.Subprograma = "RIESGO_SOCIAL"
		}
		if p.Subprograma != "RIESGO_SOCIAL" {
			return p, ErrSubprograma
		}
	} else if d.Datos.Extraccion.Tipo != "anexo4" {
		return p, ErrSubprograma
	}
	if p.Subprograma != "RIESGO_SOCIAL" && p.Subprograma != "REUNIFICACION" {
		return p, ErrSubprograma
	}
	return p, nil
}

func (p Espera) Validar(d Derivacion) error {
	if d.Estado != DerivacionRecibida {
		return ErrEstado
	}
	if p.Version < 1 {
		return ErrPeticion
	}
	if d.Datos.Extraccion.Tipo != "anexo3" {
		return ErrOrigenEspera
	}
	return nil
}

// FechaCivil usa la zona del reloj inyectado, también en el cambio de año.
func FechaCivil(t time.Time) string { return t.Format("2006-01-02") }
