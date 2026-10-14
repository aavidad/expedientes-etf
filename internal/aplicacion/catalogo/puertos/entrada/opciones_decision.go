package entrada

import "context"

// OpcionesDecision limita la lectura de Coordinación a los datos de la decisión.
type OpcionesDecision struct {
	Equipos      []EquipoDecision    `json:"equipos"`
	Centros      []EquipoDecision    `json:"centros"`
	Municipios   []MunicipioDecision `json:"municipios"`
	Subprogramas []OpcionDecision    `json:"subprogramas"`
	Textos       map[string]string   `json:"textos"`
}
type EquipoDecision struct {
	Codigo string `json:"codigo,omitempty"`
	ID     string `json:"id"`
	Clave  string `json:"clave_texto"`
}
type MunicipioDecision struct {
	ID       string  `json:"id"`
	INE      string  `json:"ine"`
	Clave    string  `json:"clave_texto"`
	CentroID *string `json:"centro_ss_id"`
	EquipoID *string `json:"equipo_id"`
	Ambito   bool    `json:"ambito_diputacion"`
}
type OpcionDecision struct {
	Codigo string `json:"codigo"`
	Clave  string `json:"clave_texto"`
}
type ConsultaOpcionesDecision interface {
	ConsultarOpcionesDecision(context.Context, Actor, string) (OpcionesDecision, error)
}
