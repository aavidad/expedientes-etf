package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type Cierres interface {
	BloquearCierre(context.Context, string) (d.SituacionCierre, error)
	ConfirmarCierre(context.Context, string, d.SituacionCierre, d.PeticionCierre, string, time.Time) (d.ResultadoCierre, error)
	Reabrir(context.Context, string, d.SituacionCierre, d.PeticionReapertura, string, string, time.Time) (d.ResultadoCierre, error)
	RepetirActoCierre(context.Context, string, string, string, string, time.Time) (d.ResultadoCierre, bool, error)
	RecordarActoCierre(context.Context, string, string, string, string, d.ResultadoCierre, time.Time) error
}
