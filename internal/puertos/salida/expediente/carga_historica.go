package expediente

import (
	"context"
	d "github.com/aavidad/expedientes-etf/internal/dominio/expediente"
	"time"
)

type CargasHistoricas interface {
	CrearLoteHojaHistorico(context.Context, string, string, string, time.Time, int, d.FuenteHoja) error
	CrearLoteHistorico(context.Context, string, string, string, time.Time, int) error
	CrearCasoHistorico(context.Context, string, string, string, d.CasoHistorico, string, time.Time) error
	EstadoHistorico(context.Context, string, string, string, time.Time) error
	RepetirCargaHistorica(context.Context, string, string, string, string, time.Time) (d.ResultadoCargaHistorica, bool, error)
	RecordarCargaHistorica(context.Context, string, string, string, string, d.ResultadoCargaHistorica, time.Time) error
}
