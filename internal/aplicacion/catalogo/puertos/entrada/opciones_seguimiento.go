package entrada

import "context"

// OpcionesSeguimiento contiene los motivos activos y las etiquetas de los motivos vigentes e históricos.
type OpcionesSeguimiento struct {
	Motivos []OpcionDecision  `json:"motivos"`
	Textos  map[string]string `json:"textos"`
}
type ConsultaOpcionesSeguimiento interface {
	ConsultarOpcionesSeguimiento(context.Context, Actor, string) (OpcionesSeguimiento, error)
}
