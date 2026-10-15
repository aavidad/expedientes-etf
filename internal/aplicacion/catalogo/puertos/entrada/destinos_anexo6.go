package entrada

import "context"

// DestinosAnexo6 separa las opciones activas de las etiquetas históricas.
type DestinosAnexo6 struct {
	Servicios []OpcionDecision  `json:"servicios"`
	Textos    map[string]string `json:"textos"`
}

type ConsultaDestinosAnexo6 interface {
	ConsultarDestinosAnexo6(context.Context, Actor, string) (DestinosAnexo6, error)
}
