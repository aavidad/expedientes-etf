package documento

import "maps"

const TablaSPAF = "spaf"

// VersionesSPAF coincide con las referencias entregadas por la vista previa.
type VersionesSPAF struct {
	Regla  string            `json:"regla"`
	Fichas map[string]string `json:"fichas"`
}
type SolicitudSPAF struct {
	EquipoID  string        `json:"-"`
	Anio      int           `json:"-"`
	Mes       int           `json:"-"`
	Idioma    string        `json:"idioma"`
	Versiones VersionesSPAF `json:"versiones"`
	Clave     string        `json:"-"`
}

func (s SolicitudSPAF) Validar() error {
	if !IDValido(s.EquipoID) || s.Anio < 1 || s.Anio > 9998 || s.Mes < 1 || s.Mes > 12 || len(s.Idioma) > 63 || !idioma.MatchString(s.Idioma) || !clave.MatchString(s.Clave) || s.Versiones.Regla == "" || len(s.Versiones.Regla) > 128 || s.Versiones.Fichas == nil || len(s.Versiones.Fichas) > 1000 {
		return ErrPeticion
	}
	for id, v := range s.Versiones.Fichas {
		if !IDValido(id) || len(v) > 128 {
			return ErrPeticion
		}
	}
	return nil
}
func (v VersionesSPAF) Igual(o VersionesSPAF) bool {
	return v.Regla == o.Regla && maps.Equal(v.Fichas, o.Fichas)
}

type ConflictoSPAF struct{ Actual VersionesSPAF }

func (*ConflictoSPAF) Error() string { return "error.conflicto_version" }
