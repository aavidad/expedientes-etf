package expediente

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const SPMAlaEspera = "a_la_espera"
const SPMInicioConocido = "inicio_desamparo_medida"
const SPMMedidaNoNecesaria = "medida_no_necesaria"

var ErrSPM = errors.New("expediente.spm_peticion_invalida")

type PeticionSPM struct {
	Version         int    `json:"version"`
	Motivo          string `json:"motivo,omitempty"`
	PropuestaMedida string `json:"propuesta_medida,omitempty"`
	Respuesta       string `json:"respuesta,omitempty"`
	Fecha           string `json:"fecha,omitempty"`
}
type SalidaSPM struct {
	ID              string `json:"id"`
	ExpedienteID    string `json:"expediente_id"`
	EquipoID        string `json:"equipo_id"`
	Version         int    `json:"version"`
	Motivo          string `json:"motivo"`
	PropuestaMedida string `json:"propuesta_medida"`
	FechaDerivacion string `json:"fecha_derivacion,omitempty"`
	Respuesta       string `json:"respuesta"`
	FechaRespuesta  string `json:"fecha_respuesta,omitempty"`
	MotivoForzado   string `json:"motivo_forzado,omitempty"`
	ForzadoID       string `json:"forzado_id,omitempty"`
}

// PermiteCierreOrdinarioSPM no toma el forzado como una respuesta habilitante.
// La ausencia de derivación al SPM permite el cierre ordinario.
func PermiteCierreOrdinarioSPM(s *SalidaSPM) bool {
	return s == nil || s.Respuesta == SPMMedidaNoNecesaria && fechaSPM(s.FechaRespuesta) || s.Respuesta == SPMInicioConocido && fechaSPM(s.FechaRespuesta)
}
func (a Actor) AutorizarSPM() error { return a.AutorizarConsulta() }
func (a Actor) AutorizarEquipoSPM(equipo string) error {
	if a.AutorizarSPM() != nil || !slices.Contains(a.Perfiles, "coordinacion") && a.EquipoID != equipo {
		return ErrAcceso
	}
	return nil
}
func textoSPM(s string) bool {
	if !utf8.ValidString(s) || strings.TrimSpace(s) != s || s == "" || len(s) > 4000 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}
func fechaSPM(s string) bool { t, e := time.Parse("2006-01-02", s); return e == nil && t.Year() > 0 }
func (p PeticionSPM) ValidarSPM(accion string, s *SalidaSPM) error {
	if p.Version < 1 {
		return ErrSPM
	}
	switch accion {
	case "crear":
		if !textoSPM(p.Motivo) || !textoSPM(p.PropuestaMedida) || p.Fecha != "" || p.Respuesta != "" {
			return ErrSPM
		}
	case "confirmar":
		if s == nil || s.FechaDerivacion != "" {
			return ErrEstado
		}
		if !fechaSPM(p.Fecha) || p.Respuesta != "" || p.Motivo != "" || p.PropuestaMedida != "" {
			return ErrSPM
		}
	case "responder":
		if s == nil || s.FechaDerivacion == "" {
			return ErrEstado
		}
		if p.Motivo != "" || p.PropuestaMedida != "" {
			return ErrSPM
		}
		if p.Respuesta == SPMAlaEspera {
			if p.Fecha != "" {
				return ErrSPM
			}
			return nil
		}
		if p.Respuesta != SPMInicioConocido && p.Respuesta != SPMMedidaNoNecesaria || !fechaSPM(p.Fecha) || p.Fecha < s.FechaDerivacion {
			return ErrSPM
		}
	case "forzar":
		if s == nil || PermiteCierreOrdinarioSPM(s) {
			return ErrEstado
		}
		if !textoSPM(p.Motivo) || p.PropuestaMedida != "" || p.Respuesta != "" || p.Fecha != "" {
			return ErrSPM
		}
	default:
		return ErrSPM
	}
	return nil
}
