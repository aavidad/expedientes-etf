package estadistica

import (
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	ErrPermisoMemoria    Error = "estadistica.memoria_acceso_denegado"
	ErrNarrativas        Error = "estadistica.narrativas_invalidas"
	ErrDisponibleMemoria Error = "estadistica.memoria_no_disponible"
	ErrCerradaMemoria    Error = "estadistica.memoria_cerrada"
)

type ActorMemoria = ActorSPAF
type FiltroMemoria struct {
	Anio     int    `json:"anio"`
	EquipoID string `json:"equipo_id"`
}

func AutorizarMemoria(a ActorMemoria, f FiltroMemoria) (FiltroMemoria, error) {
	if !UUIDValidoSPAF(a.UsuarioID) {
		return f, ErrPermisoMemoria
	}
	if !slices.Contains(a.Perfiles, "coordinacion") {
		if !slices.Contains(a.Perfiles, "profesional_etf") || !UUIDValidoSPAF(a.EquipoID) || (f.EquipoID != "" && f.EquipoID != a.EquipoID) {
			return f, ErrPermisoMemoria
		}
		f.EquipoID = a.EquipoID
	}
	if f.Anio < 1 || f.Anio > 9998 || (f.EquipoID != "" && !UUIDValidoSPAF(f.EquipoID)) {
		return f, ErrEntrada
	}
	return f, nil
}

type PersonalMemoria struct {
	Hombres *int `json:"hombres"`
	Mujeres *int `json:"mujeres"`
}
type GeneralesMemoria struct {
	EntidadLocal    string          `json:"entidad_local"`
	EquiposConvenio *int            `json:"equipos_convenio"`
	Denominacion    string          `json:"denominacion"`
	Psicologia      PersonalMemoria `json:"psicologia"`
	TrabajoSocial   PersonalMemoria `json:"trabajo_social"`
	EducacionSocial PersonalMemoria `json:"educacion_social"`
	Observaciones   string          `json:"observaciones"`
}
type Narrativas struct {
	ID string `json:"id"`
	FiltroMemoria
	Version int              `json:"version"`
	A       GeneralesMemoria `json:"a"`
	D       string           `json:"d"`
	E       string           `json:"e"`
	F       string           `json:"f"`
	G       string           `json:"g"`
}

func textoNarrativo(s string, maxBytes, maxLineas int) bool {
	if !utf8.ValidString(s) || len(s) > maxBytes || strings.Count(strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n"), "\n")+1 > maxLineas {
		return false
	}
	for _, r := range s {
		if r != '\t' && r != '\n' && r != '\r' && (r < 32 || r >= 0xd800 && r <= 0xdfff || r == 0xfffe || r == 0xffff) {
			return false
		}
	}
	return true
}
func (n Narrativas) Validar() error {
	if n.Version < 0 || (n.A.EquiposConvenio != nil && (*n.A.EquiposConvenio < 0 || *n.A.EquiposConvenio > 1000)) || !textoNarrativo(n.A.EntidadLocal, 1000, 1) || !textoNarrativo(n.A.Denominacion, 1000, 1) || !textoNarrativo(n.A.Observaciones, 20000, 100) {
		return ErrNarrativas
	}
	for _, p := range []PersonalMemoria{n.A.Psicologia, n.A.TrabajoSocial, n.A.EducacionSocial} {
		if (p.Hombres != nil && (*p.Hombres < 0 || *p.Hombres > 10000)) || (p.Mujeres != nil && (*p.Mujeres < 0 || *p.Mujeres > 10000)) {
			return ErrNarrativas
		}
	}
	for _, s := range []string{n.D, n.E, n.F} {
		if !textoNarrativo(s, 20000, 25) {
			return ErrNarrativas
		}
	}
	if !textoNarrativo(n.G, 24000, 30) {
		return ErrNarrativas
	}
	return nil
}

type ConflictoMemoria struct {
	Actual int `json:"version_actual"`
}

func (*ConflictoMemoria) Error() string { return "estadistica.memoria_conflicto" }
