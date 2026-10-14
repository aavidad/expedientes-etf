package catalogo

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrEdicion = errors.New("catalogo.edicion_invalida")
var ErrPermiso = errors.New("catalogo.acceso_denegado")

type Conflicto struct {
	Actual  int    `json:"version_actual"`
	Recurso string `json:"recurso"`
}

func (*Conflicto) Error() string { return "catalogo.conflicto" }

type Plazo struct {
	Clave  string `json:"clave"`
	Valor  int    `json:"valor"`
	Unidad string `json:"unidad"`
}
type DuracionFase struct {
	Subprograma      string `json:"subprograma"`
	Fase             string `json:"fase"`
	DiasOrientativos int    `json:"dias_orientativos"`
	DiasAmpliable    *int   `json:"dias_ampliable"`
}
type Tramo struct {
	Tipo   string   `json:"tipo"`
	Orden  int      `json:"orden"`
	Desde  float64  `json:"desde"`
	Hasta  *float64 `json:"hasta"`
	Unidad string   `json:"unidad"`
}
type Firmante struct {
	ReferenciasTextos map[string]string `json:"-"`
	ID                string            `json:"id"`
	Tipo              string            `json:"tipo"`
	Rol               string            `json:"rol"`
	Nombre            string            `json:"nombre"`
	CargoCodigo       string            `json:"cargo_codigo"`
	OrganismoCodigo   string            `json:"organismo_codigo"`
	Direccion         string            `json:"direccion"`
	VigenteDesde      time.Time         `json:"vigente_desde"`
	VigenteHasta      *time.Time        `json:"vigente_hasta"`
	Version           int               `json:"version"`
}
type Cambio struct {
	VersionEsperada      int               `json:"version_esperada"`
	ReferenciasEsperadas map[string]int    `json:"referencias_esperadas"`
	VigenteDesde         time.Time         `json:"vigente_desde"`
	Opcion               *Opcion           `json:"opcion,omitempty"`
	Plazo                *Plazo            `json:"plazo,omitempty"`
	DuracionFase         *DuracionFase     `json:"duracion_fase,omitempty"`
	Tramo                *Tramo            `json:"tramo,omitempty"`
	Firmante             *Firmante         `json:"firmante,omitempty"`
	Textos               map[string]string `json:"textos"`
}

var codigoRegla = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
var codigoOpcion = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)
var idFirmante = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func Autorizar(actor Actor) error {
	if actor.UsuarioID != "" {
		for _, p := range actor.Perfiles {
			if p == "administracion" {
				return nil
			}
		}
	}
	return ErrPermiso
}
func (c Cambio) ClaveTexto() string {
	if c.Opcion != nil {
		return c.Opcion.Lista + "." + c.Opcion.Codigo
	}
	if c.Firmante != nil {
		return "cargo." + c.Firmante.CargoCodigo
	}
	return ""
}
func (c Cambio) Validar(ambito string) error {
	if c.VersionEsperada < 0 || c.VigenteDesde.IsZero() || c.VigenteDesde.Year() < 1 || c.VigenteDesde.Year() > 9999 {
		return ErrEdicion
	}
	n := 0
	if c.DuracionFase != nil {
		n++
	}
	if c.Opcion != nil {
		n++
	}
	if c.Plazo != nil {
		n++
	}
	if c.Tramo != nil {
		n++
	}
	if c.Firmante != nil {
		n++
	}
	if n != 1 {
		return ErrEdicion
	}
	switch ambito {
	case "listas":
		o := c.Opcion
		if o == nil || !codigoRegla.MatchString(o.Lista) || !codigoOpcion.MatchString(o.Codigo) || o.Orden < 0 {
			return ErrEdicion
		}
	case "plazos":
		if f := c.DuracionFase; f != nil {
			if (f.Subprograma != "RIESGO_SOCIAL" && f.Subprograma != "REUNIFICACION") || !codigoOpcion.MatchString(f.Fase) || f.DiasOrientativos < 0 || (f.DiasAmpliable != nil && *f.DiasAmpliable < f.DiasOrientativos) {
				return ErrEdicion
			}
			break
		}
		p := c.Plazo
		if p == nil || !codigoRegla.MatchString(p.Clave) || p.Valor < 0 || !codigoRegla.MatchString(p.Unidad) {
			return ErrEdicion
		}
	case "tramos":
		t := c.Tramo
		if t == nil || !codigoRegla.MatchString(t.Tipo) || t.Orden < 0 || t.Desde < 0 || t.Desde != t.Desde || (t.Hasta != nil && (*t.Hasta <= t.Desde || *t.Hasta != *t.Hasta)) || !codigoRegla.MatchString(t.Unidad) {
			return ErrEdicion
		}
	case "firmantes":
		f := c.Firmante
		if f == nil || (f.ID != "" && !idFirmante.MatchString(f.ID)) || (f.ID == "" && c.VersionEsperada != 0) || (f.Tipo != "firmante" && f.Tipo != "destinatario") || !codigoRegla.MatchString(f.Rol) || strings.TrimSpace(f.Nombre) == "" || len(f.Nombre) > 200 || !codigoOpcion.MatchString(f.CargoCodigo) || !codigoOpcion.MatchString(f.OrganismoCodigo) || len(f.Direccion) > 500 || (f.VigenteHasta != nil && !FechaCivil(*f.VigenteHasta).After(FechaCivil(c.VigenteDesde))) {
			return ErrEdicion
		}
	default:
		return ErrAmbito
	}
	if c.ClaveTexto() != "" {
		if _, ok := c.ReferenciasEsperadas["es"]; !ok {
			return ErrEdicion
		}
		if _, ok := c.ReferenciasEsperadas["en"]; !ok {
			return ErrEdicion
		}
		if len(c.Textos) != 2 || strings.TrimSpace(c.Textos["es"]) == "" || strings.TrimSpace(c.Textos["en"]) == "" || len(c.Textos["es"]) > 2000 || len(c.Textos["en"]) > 2000 || len(c.ReferenciasEsperadas) != 2 || c.ReferenciasEsperadas["es"] < 0 || c.ReferenciasEsperadas["en"] < 0 {
			return ErrEdicion
		}
	}
	return nil
}
