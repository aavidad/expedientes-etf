package entrada

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
)

type Conservacion = d.Conservacion

var ErrConservacion = d.ErrConservacion

type Conservaciones interface {
	PendientesConservacion(context.Context, Actor) ([]Conservacion, error)
	ConfirmarConservacion(context.Context, Actor, string, int, bool) (Conservacion, error)
}
