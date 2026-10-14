package expediente

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"
)

const Aceptada = "aceptada"

var (
	ErrObligatorios = errors.New("expediente.aceptacion_incompleta")
	ErrProfesional  = errors.New("expediente.profesional_invalido")
	ErrSexo         = errors.New("expediente.sexo_obligatorio")
	ErrEstado       = errors.New("error.estado_no_permite")
	ErrIdempotencia = errors.New("error.idempotencia_reutilizada")
	ErrClave        = errors.New("expediente.idempotencia_obligatoria")
	ErrNumeracion   = errors.New("expediente.numeracion_agotada")
)

// Conflicto informa de la versión vigente sin exponer datos de la ficha.
type Conflicto struct{ VersionActual int }

func (Conflicto) Error() string { return "error.conflicto_version" }

type SexoMenor struct {
	Orden int    `json:"orden"`
	Sexo  string `json:"sexo"`
}

type Aceptacion struct {
	Version           int         `json:"version"`
	FechaActa         string      `json:"fecha_acta"`
	MunicipioID       string      `json:"municipio_id"`
	CentroID          string      `json:"centro_ss_id"`
	EquipoID          string      `json:"equipo_id"`
	Subprograma       string      `json:"subprograma"`
	ProfesionalPuesto string      `json:"profesional_referencia_puesto"`
	ProfesionalID     *string     `json:"profesional_referencia_id"`
	Menores           []SexoMenor `json:"menores"`
	Descripcion       string      `json:"descripcion_situacion"`
}

type ResultadoAceptacion struct {
	ID          string `json:"id"`
	Numero      string `json:"numero"`
	Estado      string `json:"estado"`
	Version     int    `json:"version"`
	EquipoID    string `json:"equipo_id"`
	Anio        int    `json:"anio_numero"`
	Correlativo int    `json:"correlativo"`
	ReglaID     string `json:"regla_numeracion_id"`
	FechaActa   string `json:"fecha_acta"`
}

type DatosAceptados struct {
	Aceptacion
	Anio        int    `json:"anio_numero"`
	Correlativo int    `json:"correlativo"`
	ReglaID     string `json:"regla_numeracion_id"`
	Fase        string `json:"fase_intervencion"`
}

func ClaveValida(clave string) bool {
	return regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`).MatchString(clave)
}

func (a Aceptacion) Fecha() (time.Time, error) {
	t, err := time.Parse("2006-01-02", a.FechaActa)
	if err != nil || t.Year() < 1 {
		return time.Time{}, ErrObligatorios
	}
	return t, nil
}

func (a Aceptacion) Validar(d Derivacion) error {
	if _, err := a.Fecha(); err != nil {
		return err
	}
	if a.Version < 1 || !IDValido(a.MunicipioID) || !IDValido(a.CentroID) || !IDValido(a.EquipoID) || strings.TrimSpace(a.Descripcion) == "" || len(a.Descripcion) > 20000 || (a.Subprograma != "RIESGO_SOCIAL" && a.Subprograma != "REUNIFICACION") {
		return ErrObligatorios
	}
	if a.ProfesionalPuesto != "TS" && a.ProfesionalPuesto != "ES" && a.ProfesionalPuesto != "PS" {
		return ErrProfesional
	}
	if a.ProfesionalID != nil && !IDValido(*a.ProfesionalID) {
		return ErrProfesional
	}
	if d.Datos.Extraccion.Tipo == "anexo3" && a.Subprograma != "RIESGO_SOCIAL" {
		return ErrObligatorios
	}
	total := 0
	for _, c := range d.Datos.Extraccion.Campos {
		if c.Codigo == "menores.persona.nombre" {
			total++
		}
	}
	if total < 1 || total > 15 || len(a.Menores) != total {
		return ErrSexo
	}
	vistos := map[int]bool{}
	for _, m := range a.Menores {
		if m.Orden < 1 || m.Orden > total || vistos[m.Orden] || (m.Sexo != "H" && m.Sexo != "M") {
			return ErrSexo
		}
		vistos[m.Orden] = true
	}
	return nil
}

// AutorizarConsulta permite abrir la ficha propia sin permitir aceptaciones.
func (a Actor) AutorizarConsulta() error {
	if a.Autorizar() == nil {
		return nil
	}
	if IDValido(a.UsuarioID) && IDValido(a.EquipoID) && slices.Contains(a.Perfiles, "profesional_etf") {
		return nil
	}
	return ErrAcceso
}
