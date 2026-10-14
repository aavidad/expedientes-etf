package entrada

import "context"

type ConsultaOpcionesCarga interface {
	ConsultarOpcionesCarga(context.Context, Actor, string) (OpcionesDecision, error)
}
