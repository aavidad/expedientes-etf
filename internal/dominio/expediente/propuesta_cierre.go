package expediente

import (
	"errors"
	"slices"
	"strings"
	"time"
)

var ErrFinalizacion = errors.New("expediente.finalizacion_invalida")
var ErrConsentimientoCierre = errors.New("expediente.cierre_requiere_falta_consentimiento")

type Finalizacion struct {
	Version              int    `json:"version"`
	Motivo               string `json:"motivo"`
	MotivoTexto          string `json:"motivo_texto"`
	SinConsentimiento    bool   `json:"sin_consentimiento"`
	DerivacionTrasCierre string `json:"derivacion_tras_cierre"`
	CoordinacionRecursos string `json:"coordinacion_recursos"`
	ObjetivosAbordados   string `json:"objetivos_abordados"`
	ObjetivosPendientes  string `json:"objetivos_pendientes"`
}

type FinalizacionGuardada struct {
	Finalizacion
	CatalogoVersionID string `json:"catalogo_version_id"`
	MotivoActa        string `json:"motivo_acta"`
	MotivoSPAF        string `json:"motivo_spaf"`
}

type PropuestaCierre struct {
	ID           string    `json:"id"`
	ExpedienteID string    `json:"expediente_id"`
	EquipoID     string    `json:"equipo_id"`
	Numero       string    `json:"numero"`
	PropuestaPor string    `json:"propuesta_por"`
	PropuestaEn  time.Time `json:"propuesta_en"`
	Estado       string    `json:"estado"`
	Version      int       `json:"version"`
	Motivo       string    `json:"motivo"`
}

type VersionCierre struct {
	Version int `json:"version"`
}

func (a Actor) AutorizarPropuesta() error {
	if IDValido(a.UsuarioID) && IDValido(a.EquipoID) && slices.Contains(a.Perfiles, "profesional_etf") {
		return nil
	}
	return ErrAcceso
}
func (f Finalizacion) Validar(d Derivacion, admiteTexto, permiteSinConsentimiento bool) error {
	if f.Version < 1 || f.Motivo == "" {
		return ErrFinalizacion
	}
	for _, texto := range []string{f.MotivoTexto, f.DerivacionTrasCierre, f.CoordinacionRecursos, f.ObjetivosAbordados, f.ObjetivosPendientes} {
		if len(texto) > 20000 || strings.ContainsRune(texto, 0) {
			return ErrFinalizacion
		}
	}
	if (admiteTexto && strings.TrimSpace(f.MotivoTexto) == "") || (!admiteTexto && f.MotivoTexto != "") {
		return ErrFinalizacion
	}
	if d.Estado != Aceptada && d.Estado != "en_intervencion" {
		return ErrEstado
	}
	if d.Estado == Aceptada && (!f.SinConsentimiento || !permiteSinConsentimiento) {
		return ErrConsentimientoCierre
	}
	if d.Estado == "en_intervencion" && f.SinConsentimiento {
		return ErrConsentimientoCierre
	}
	return nil
}
