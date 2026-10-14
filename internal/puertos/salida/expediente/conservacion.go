package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Conservaciones interface {
	MarcarConservacion(context.Context, string, int, string, time.Time) error
	ListarConservacion(context.Context, time.Time) ([]d.Conservacion, error)
	ConfirmarConservacion(context.Context, string, int, bool, time.Time) (d.Conservacion, error)
}

type PlazoConservacion interface {
	Anios(context.Context) (int, error)
}
