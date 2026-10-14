package expediente

import (
	"errors"
	"time"
)

var ErrCierre = errors.New("expediente.cierre_invalido")
var ErrDobleIntervencion = errors.New("expediente.cierre_segunda_persona")
var ErrCierreSPM = errors.New("expediente.cierre_bloqueado_anexo5")

const Cerrada = "cerrada"
const DerivadaSPM = "derivada_spm"

type PeticionCierre struct {
	Version         int    `json:"version"`
	Fecha           string `json:"fecha"`
	MotivoActaTexto string `json:"motivo_acta_texto,omitempty"`
	ForzadoID       string `json:"forzado_id,omitempty"`
}
type PeticionReapertura struct {
	Version int    `json:"version"`
	Fecha   string `json:"fecha"`
	Motivo  string `json:"motivo"`
}
type ResultadoCierre struct {
	ExpedienteID string `json:"expediente_id"`
	EpisodioID   string `json:"episodio_id"`
	Estado       string `json:"estado"`
	Fecha        string `json:"fecha"`
	Version      int    `json:"version"`
	Aviso        string `json:"aviso,omitempty"`
}

// SituacionCierre se lee bajo el bloqueo del expediente, en la transacción del acto.
type SituacionCierre struct {
	Estado         string
	Version        int
	EquipoID       string
	EpisodioID     string
	AperturaFecha  string
	UltimoCierre   string
	PropuestaID    string
	PropuestaPor   string
	Motivo         string
	MotivoActa     string
	MotivoSPAF     string
	Consentimiento bool
}
type ActaCierre struct {
	PropuestaID     string
	PropuestaPor    string
	MotivoActa      string
	MotivoActaTexto string
	Fecha           string
}

func fechaCierre(fecha, desde string, ahora time.Time) bool {
	t, err := time.Parse("2006-01-02", fecha)
	return err == nil && t.Year() > 0 && fecha >= desde && fecha <= ahora.Format("2006-01-02")
}
func (p PeticionCierre) Validar(s SituacionCierre, usuario string, ahora time.Time) error {
	if s.Estado != Aceptada && s.Estado != EnIntervencion && s.Estado != DerivadaSPM {
		return ErrEstado
	}
	if s.PropuestaID == "" || s.Motivo == "" {
		return ErrFinalizacion
	}
	if s.PropuestaPor == usuario {
		return ErrDobleIntervencion
	}
	if p.Version < 1 || !fechaCierre(p.Fecha, s.AperturaFecha, ahora) || p.MotivoActaTexto != "" && !textoSPM(p.MotivoActaTexto) || p.ForzadoID != "" && !IDValido(p.ForzadoID) {
		return ErrCierre
	}
	return nil
}
func (p PeticionReapertura) Validar(s SituacionCierre, ahora time.Time) error {
	if s.Estado != Cerrada {
		return ErrEstado
	}
	if p.Version < 1 || !fechaCierre(p.Fecha, s.UltimoCierre, ahora) || !textoSPM(p.Motivo) {
		return ErrCierre
	}
	return nil
}
func EstadoReapertura(consentimiento, permiteSPM bool) string {
	if !permiteSPM {
		return DerivadaSPM
	}
	if consentimiento {
		return EnIntervencion
	}
	return Aceptada
}
