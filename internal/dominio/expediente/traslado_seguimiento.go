package expediente

import (
	"errors"
	"time"
)

var ErrTraslado = errors.New("expediente.traslado_invalido")
var ErrSeguimiento = errors.New("expediente.seguimiento_invalido")

const SeguimientoTrasCese = "seguimiento_tras_cese"

type PeticionTraslado struct {
	PeticionCierre
	EquipoDestinoID string `json:"equipo_destino_id"`
}
type ResultadoTraslado struct {
	Cierre          ResultadoCierre `json:"cierre"`
	DestinoID       string          `json:"destino_id"`
	EquipoDestinoID string          `json:"equipo_destino_id"`
	VersionDestino  int             `json:"version_destino"`
}
type EnlaceTraslado struct {
	OrigenID        string `json:"origen_id"`
	DestinoID       string `json:"destino_id"`
	EquipoOrigenID  string `json:"equipo_origen_id"`
	EquipoDestinoID string `json:"equipo_destino_id"`
}
type CopiaTraslado struct {
	Datos   DatosDerivacion `json:"datos"`
	Familia Familia         `json:"familia"`
	Adultos []Adulto        `json:"adultos"`
	Menores []Menor         `json:"menores"`
}

func (p PeticionTraslado) ValidarDestino(s SituacionCierre) error {
	if !IDValido(p.EquipoDestinoID) || p.EquipoDestinoID == s.EquipoID || s.Motivo != "TRASLADO" {
		return ErrTraslado
	}
	return nil
}

type PeticionSeguimiento struct {
	Version int    `json:"version"`
	Fecha   string `json:"fecha"`
	Motivo  string `json:"motivo,omitempty"`
}
type Seguimiento struct {
	ID               string `json:"id"`
	ExpedienteID     string `json:"expediente_id"`
	EquipoID         string `json:"equipo_id"`
	EpisodioID       string `json:"episodio_id"`
	Version          int    `json:"version"`
	Estado           string `json:"estado"`
	FechaCese        string `json:"fecha_cese_medida"`
	TramoDesde       string `json:"tramo_desde"`
	FechaFin         string `json:"fecha_fin_prevista"`
	FechaTerminacion string `json:"fecha_terminacion,omitempty"`
	Motivo           string `json:"motivo,omitempty"`
	MotivoReapertura string `json:"motivo_reapertura,omitempty"`
	ReglaID          string `json:"regla_id"`
}
type ReglaSeguimiento struct {
	ID             string
	Meses          int
	AntelacionDias int
}
type AvisoSeguimiento struct {
	ExpedienteID  string `json:"expediente_id"`
	SeguimientoID string `json:"seguimiento_id"`
	Clave         string `json:"clave"`
	Desde         string `json:"vigente_desde"`
	Hasta         string `json:"vigente_hasta"`
}

// FinSeguimiento conserva el día del mes cuando existe; el último día de enero
// vence el último día de febrero, sin saltar a marzo.
func FinSeguimiento(fecha string, meses int) (string, error) {
	t, err := time.Parse("2006-01-02", fecha)
	if err != nil || t.Year() < 1 || meses < 1 || meses > 1200 {
		return "", ErrSeguimiento
	}
	inicio := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, meses, 0)
	ultimo := inicio.AddDate(0, 1, -1).Day()
	dia := min(t.Day(), ultimo)
	fin := inicio.AddDate(0, 0, dia-1)
	if fin.Year() > 9999 {
		return "", ErrSeguimiento
	}
	return fin.Format("2006-01-02"), nil
}
func (p PeticionSeguimiento) Validar(accion string, s *Seguimiento, ahora time.Time) error {
	if p.Version < 1 {
		return ErrSeguimiento
	}
	switch accion {
	case "iniciar":
		if s != nil || p.Motivo != "" || !fechaCierre(p.Fecha, "0001-01-01", ahora) {
			return ErrSeguimiento
		}
	case "terminar":
		if s == nil || s.Estado != "activo" || FechaCivil(ahora) >= s.FechaFin || !fechaCierre(p.Fecha, max(s.FechaCese, s.TramoDesde), ahora) || p.Fecha >= s.FechaFin || p.Motivo == "" {
			return ErrSeguimiento
		}
	case "reabrir":
		if s == nil || s.Estado != "terminado" || p.Fecha != "" || !textoSPM(p.Motivo) {
			return ErrSeguimiento
		}
	default:
		return ErrSeguimiento
	}
	return nil
}
