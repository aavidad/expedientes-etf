package expediente

import (
	"errors"
	"time"
)

const EnIntervencion = "en_intervencion"

var (
	ErrHitoTipo  = errors.New("expediente.hito_tipo_invalido")
	ErrHitoFecha = errors.New("expediente.hito_fecha_invalida")
)

type ConfirmacionHito struct {
	Tipo    string `json:"tipo"`
	Fecha   string `json:"fecha"`
	Version int    `json:"version"`
}

type ResultadoHito struct {
	ID      string `json:"id"`
	Tipo    string `json:"tipo"`
	Fecha   string `json:"fecha"`
	Estado  string `json:"estado"`
	Version int    `json:"version"`
}

type Hito struct {
	ID            string    `json:"id"`
	Tipo          string    `json:"tipo"`
	Fecha         string    `json:"fecha"`
	Origen        string    `json:"origen"`
	ConfirmadoPor string    `json:"confirmado_por"`
	ConfirmadoEn  time.Time `json:"confirmado_en"`
}

// SituacionHitos contiene sólo los datos necesarios para confirmar fechas.
type SituacionHitos struct {
	ID, Estado, EquipoID, Subprograma, FechaAceptacion string
	Version                                            int
}

func (a Actor) AutorizarHito() error { return a.AutorizarConsulta() }

func (p ConfirmacionHito) Validar(d SituacionHitos, hoy time.Time) error {
	if p.Version < 1 {
		return ErrPeticion
	}
	if p.Tipo != "consentimiento" && p.Tipo != "anexo7" && p.Tipo != "anexo8" {
		return ErrHitoTipo
	}
	if d.Estado != Aceptada && d.Estado != EnIntervencion {
		return ErrEstado
	}
	if p.Tipo == "consentimiento" && d.Estado != Aceptada {
		return ErrEstado
	}
	if (p.Tipo == "anexo7" && d.Subprograma != "RIESGO_SOCIAL") || (p.Tipo == "anexo8" && d.Subprograma != "REUNIFICACION") {
		return ErrSubprograma
	}
	fecha, err := time.Parse("2006-01-02", p.Fecha)
	if err != nil || fecha.Year() < 1 || p.Fecha < d.FechaAceptacion || p.Fecha > FechaCivil(hoy) {
		return ErrHitoFecha
	}
	return nil
}
