package estadistica

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/estadistica"
	"time"
)

type Narrativas interface {
	LeerNarrativas(context.Context, d.FiltroMemoria) (d.Narrativas, error)
	GuardarNarrativas(context.Context, d.Narrativas, string, time.Time) (d.Narrativas, error)
}

type NarrativasHistoricas interface {
	NarrativasConocidas(context.Context, d.FiltroMemoria, time.Time) (d.Narrativas, error)
}
