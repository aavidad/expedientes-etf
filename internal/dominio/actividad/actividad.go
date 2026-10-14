package actividad

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrDatos        = errors.New("actividad.datos_invalidos")
	ErrOtros        = errors.New("actividad.otros_obligatorio")
	ErrPermiso      = errors.New("actividad.acceso_denegado")
	ErrNoEncontrado = errors.New("actividad.no_encontrado")
	ErrPersistencia = errors.New("actividad.no_disponible")
)

type Tipo string

const (
	Comision Tipo = "comision"
	Informe  Tipo = "informe"
)

type Actor struct {
	UsuarioID string
	Perfiles  []string
	EquipoID  string
}

func UUIDValido(s string) bool {
	return regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`).MatchString(s)
}

// Autorizar mantiene la consulta provincial separada de la escritura del equipo.
func (a Actor) Autorizar(escritura bool) error {
	if !UUIDValido(a.UsuarioID) {
		return ErrPermiso
	}
	if !escritura && slices.Contains(a.Perfiles, "coordinacion") {
		return nil
	}
	if slices.Contains(a.Perfiles, "profesional_etf") && UUIDValido(a.EquipoID) {
		return nil
	}
	return ErrPermiso
}

type Registro struct {
	ID             string `json:"id"`
	EquipoID       string `json:"equipo_id"`
	Fecha          string `json:"fecha"`
	Observaciones  string `json:"observaciones,omitempty"`
	Organismo      string `json:"organismo,omitempty"`
	OrganismoOtros string `json:"organismo_otros,omitempty"`
	Version        int    `json:"version"`
}

type Cambio struct {
	Fecha           string `json:"fecha"`
	Observaciones   string `json:"observaciones,omitempty"`
	Organismo       string `json:"organismo,omitempty"`
	OrganismoOtros  string `json:"organismo_otros,omitempty"`
	VersionEsperada int    `json:"version_esperada"`
}

func (c Cambio) Validar(tipo Tipo, nuevo bool) error {
	fecha, err := time.Parse("2006-01-02", c.Fecha)
	if err != nil || fecha.Year() < 1 || fecha.Year() > 9998 || c.VersionEsperada < 0 ||
		(nuevo && c.VersionEsperada != 0) || (!nuevo && c.VersionEsperada < 1) ||
		utf8.RuneCountInString(c.Observaciones) > 4000 || utf8.RuneCountInString(c.OrganismoOtros) > 500 ||
		strings.ContainsRune(c.Observaciones+c.OrganismoOtros, 0) {
		return ErrDatos
	}
	switch tipo {
	case Comision:
		if c.Organismo != "" || c.OrganismoOtros != "" {
			return ErrDatos
		}
	case Informe:
		if c.Organismo == "" || c.Observaciones != "" {
			return ErrDatos
		}
		if c.Organismo == "OTROS" && strings.TrimSpace(c.OrganismoOtros) == "" {
			return ErrOtros
		}
		if c.Organismo != "OTROS" && c.OrganismoOtros != "" {
			return ErrDatos
		}
	default:
		return ErrDatos
	}
	return nil
}

type Filtro struct {
	Anio     int    `json:"anio"`
	EquipoID string `json:"equipo_id"`
}

func (f Filtro) Validar() error {
	if f.Anio < 1 || f.Anio > 9998 || (f.EquipoID != "" && !UUIDValido(f.EquipoID)) {
		return ErrDatos
	}
	return nil
}

type Organismo struct {
	Codigo      string `json:"codigo"`
	ClaveTexto  string `json:"clave_texto"`
	AdmiteTexto bool   `json:"admite_texto"`
	Version     int    `json:"version"`
}

// Recuento no contiene identificadores de registros ni textos libres.
type Recuento struct {
	Anio         int              `json:"anio"`
	EquipoID     string           `json:"equipo_id,omitempty"`
	Comisiones   int64            `json:"comisiones"`
	Informes     int64            `json:"informes"`
	PorOrganismo map[string]int64 `json:"por_organismo"`
}

type Conflicto struct{ Actual int }

func (*Conflicto) Error() string { return "actividad.conflicto" }

// Equipo identifica una etiqueta de catálogo, sin datos del personal.
type Equipo struct {
	ID         string `json:"id"`
	ClaveTexto string `json:"clave_texto"`
}
