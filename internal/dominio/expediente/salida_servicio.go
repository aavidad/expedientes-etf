package expediente

import (
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// SalidaServicio conserva el destino de catálogo. La fecha sólo aparece al
// confirmar el hito; emitir el Anexo 6 no modifica este registro.
type SalidaServicio struct {
	ID                string `json:"id"`
	ExpedienteID      string `json:"expediente_id"`
	EquipoID          string `json:"equipo_id"`
	Tipo              string `json:"tipo"`
	Servicio          string `json:"servicio_destino"`
	CatalogoID        string `json:"catalogo_id"`
	Motivo            string `json:"motivo"`
	Fecha             string `json:"fecha,omitempty"`
	Estado            string `json:"estado"`
	Version           int    `json:"version"`
	VersionExpediente int    `json:"version_expediente"`
}

type NuevaSalidaServicio struct {
	Tipo     string `json:"tipo"`
	Version  int    `json:"version"`
	Servicio string `json:"servicio_destino"`
	Motivo   string `json:"motivo"`
}

type ConfirmarSalidaServicio struct {
	Tipo          string `json:"tipo"`
	SalidaID      string `json:"salida_id"`
	Version       int    `json:"version"`
	VersionSalida int    `json:"version_salida"`
	Fecha         string `json:"fecha"`
}

func (a Actor) AutorizarSalida(equipo string) error {
	if !IDValido(a.UsuarioID) {
		return ErrAcceso
	}
	if slices.Contains(a.Perfiles, "coordinacion") {
		return nil
	}
	if slices.Contains(a.Perfiles, "profesional_etf") && IDValido(a.EquipoID) && (equipo == "" || equipo == a.EquipoID) {
		return nil
	}
	return ErrAcceso
}

func (p NuevaSalidaServicio) Validar() error {
	if p.Tipo != "otro_servicio" || p.Version < 1 || !regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`).MatchString(p.Servicio) || strings.TrimSpace(p.Motivo) == "" || !utf8.ValidString(p.Motivo) || len(p.Motivo) > 20000 {
		return ErrPeticion
	}
	return nil
}

func (p ConfirmarSalidaServicio) Validar() error {
	f, err := time.Parse("2006-01-02", p.Fecha)
	if p.Tipo != "derivacion_otro_servicio" || !IDValido(p.SalidaID) || p.Version < 1 || p.VersionSalida < 1 || err != nil || f.Year() < 1 {
		return ErrPeticion
	}
	return nil
}

func SalidaPermitida(estado string) bool {
	return slices.Contains([]string{"aceptada", "intervencion", "en_intervencion", "propuesta_cierre"}, estado)
}
